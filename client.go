// Copyright 2025 Edgeo SCADA
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package modbus

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net"
	"sync"
	"time"

	"github.com/edgeo-scada/modbus/internal/transport"
)

// Client is a Modbus TCP client with support for automatic reconnection.
type Client struct {
	addr   string
	unitID UnitID
	opts   *clientOptions

	transport *transport.TCPTransport
	txIDGen   TransactionIDGenerator

	mu      sync.Mutex
	state   ConnectionState
	closed  bool
	closeCh chan struct{}
	metrics *Metrics
	logger  *slog.Logger
}

// NewClient creates a new Modbus TCP client.
func NewClient(addr string, opts ...Option) (*Client, error) {
	if addr == "" {
		return nil, errors.New("modbus: address cannot be empty")
	}

	options := defaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	c := &Client{
		addr:      addr,
		unitID:    options.unitID,
		opts:      options,
		transport: transport.NewTCPTransport(addr, options.timeout),
		state:     StateDisconnected,
		closeCh:   make(chan struct{}),
		metrics:   NewMetrics(),
		logger:    options.logger,
	}

	return c, nil
}

// Connect establishes a connection to the Modbus server.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return ErrConnectionClosed
	}
	if c.state == StateConnected {
		c.mu.Unlock()
		return nil
	}
	c.state = StateConnecting
	c.mu.Unlock()

	c.logger.Debug("connecting", slog.String("addr", c.addr))

	if err := c.transport.Connect(ctx); err != nil {
		c.mu.Lock()
		c.state = StateDisconnected
		c.mu.Unlock()
		return err
	}

	c.mu.Lock()
	c.state = StateConnected
	c.metrics.ActiveConns.Add(1)
	c.mu.Unlock()

	c.logger.Info("connected", slog.String("addr", c.addr))

	if c.opts.onConnect != nil {
		c.opts.onConnect()
	}

	return nil
}

// Close closes the client connection.
func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	close(c.closeCh)
	wasConnected := c.state == StateConnected
	c.state = StateDisconnected
	if wasConnected {
		c.metrics.ActiveConns.Add(-1)
	}
	c.mu.Unlock()

	c.logger.Debug("closing connection", slog.String("addr", c.addr))
	return c.transport.Close()
}

// State returns the current connection state.
func (c *Client) State() ConnectionState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// IsConnected returns true if the client is connected.
func (c *Client) IsConnected() bool {
	return c.State() == StateConnected
}

// Metrics returns the client metrics.
func (c *Client) Metrics() *Metrics {
	return c.metrics
}

// SetUnitID sets the default unit ID for subsequent requests.
func (c *Client) SetUnitID(id UnitID) {
	c.mu.Lock()
	c.unitID = id
	c.mu.Unlock()
}

// UnitID returns the current default unit ID.
func (c *Client) UnitID() UnitID {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.unitID
}

// Address returns the server address.
func (c *Client) Address() string {
	return c.addr
}

// send sends a PDU and receives the response with optional retry logic.
func (c *Client) send(ctx context.Context, pdu []byte) ([]byte, error) {
	c.mu.Lock()
	unitID := c.unitID
	c.mu.Unlock()

	return c.sendWithUnit(ctx, unitID, pdu)
}

func (c *Client) sendWithUnit(ctx context.Context, unitID UnitID, pdu []byte) ([]byte, error) {
	if len(pdu) == 0 {
		return nil, errors.New("modbus: empty PDU")
	}

	var lastErr error
	maxRetries := 1
	if c.opts.autoReconnect {
		maxRetries = c.opts.maxRetries
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			c.logger.Debug("retrying request",
				slog.Int("attempt", attempt+1),
				slog.Int("max", maxRetries))

			if err := c.reconnect(ctx); err != nil {
				lastErr = err
				continue
			}
		}

		resp, err := c.doSend(ctx, unitID, pdu)
		if err != nil {
			lastErr = err
			if !c.opts.autoReconnect || !isRetryableError(err) {
				return nil, err
			}
			c.handleDisconnect(err)
			continue
		}
		return resp, nil
	}

	return nil, fmt.Errorf("%w: %v", ErrMaxRetriesExceeded, lastErr)
}

func (c *Client) doSend(ctx context.Context, unitID UnitID, pdu []byte) ([]byte, error) {
	c.mu.Lock()
	if c.state != StateConnected {
		c.mu.Unlock()
		return nil, ErrNotConnected
	}
	c.mu.Unlock()

	start := time.Now()
	c.metrics.RequestsTotal.Add(1)

	// Build frame
	txID := c.txIDGen.Next()
	frame := Frame{
		Header: MBAPHeader{
			TransactionID: txID,
			ProtocolID:    ProtocolID,
			UnitID:        unitID,
		},
		PDU: pdu,
	}

	expectedFC := FunctionCode(pdu[0])

	c.logger.Debug("sending request",
		slog.Uint64("tx_id", uint64(txID)),
		slog.Uint64("unit_id", uint64(unitID)),
		slog.String("func", expectedFC.String()))

	// Send and receive
	respData, err := c.transport.Send(ctx, frame.Encode())
	if err != nil {
		c.metrics.RequestsErrors.Add(1)
		return nil, err
	}

	// Parse response frame
	var respFrame Frame
	if err := respFrame.Decode(respData); err != nil {
		c.metrics.RequestsErrors.Add(1)
		return nil, err
	}

	// Validate transaction ID
	if respFrame.Header.TransactionID != txID {
		c.metrics.RequestsErrors.Add(1)
		return nil, fmt.Errorf("%w: transaction ID mismatch (expected %d, got %d)",
			ErrInvalidResponse, txID, respFrame.Header.TransactionID)
	}

	// Validate unit ID
	if respFrame.Header.UnitID != unitID {
		c.metrics.RequestsErrors.Add(1)
		return nil, fmt.Errorf("%w: unit ID mismatch (expected %d, got %d)",
			ErrInvalidResponse, unitID, respFrame.Header.UnitID)
	}

	// Check for exception response
	if IsExceptionResponse(respFrame.PDU) {
		c.metrics.RequestsErrors.Add(1)
		return nil, ParseExceptionResponse(respFrame.PDU)
	}

	// Validate function code
	if len(respFrame.PDU) > 0 && FunctionCode(respFrame.PDU[0]) != expectedFC {
		c.metrics.RequestsErrors.Add(1)
		return nil, fmt.Errorf("%w: function code mismatch (expected %02X, got %02X)",
			ErrInvalidResponse, expectedFC, respFrame.PDU[0])
	}

	duration := time.Since(start)
	c.metrics.RequestsSuccess.Add(1)
	c.metrics.Latency.Observe(duration)

	c.logger.Debug("received response",
		slog.Uint64("tx_id", uint64(txID)),
		slog.Duration("duration", duration))

	return respFrame.PDU, nil
}

func (c *Client) handleDisconnect(err error) {
	c.mu.Lock()
	wasConnected := c.state == StateConnected
	c.state = StateDisconnected
	if wasConnected {
		c.metrics.ActiveConns.Add(-1)
	}
	c.mu.Unlock()

	c.transport.Close()

	c.logger.Warn("disconnected", slog.String("error", err.Error()))

	if c.opts.onDisconnect != nil {
		c.opts.onDisconnect(err)
	}
}

func (c *Client) reconnect(ctx context.Context) error {
	backoff := c.opts.reconnectBackoff

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.closeCh:
			return ErrConnectionClosed
		default:
		}

		c.logger.Info("attempting reconnection",
			slog.String("addr", c.addr),
			slog.Duration("backoff", backoff))

		c.metrics.Reconnections.Add(1)

		if err := c.Connect(ctx); err == nil {
			c.logger.Info("reconnected", slog.String("addr", c.addr))
			return nil
		}

		// Exponential backoff
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.closeCh:
			return ErrConnectionClosed
		case <-time.After(backoff):
		}

		backoff = time.Duration(math.Min(
			float64(backoff)*2,
			float64(c.opts.maxReconnectTime),
		))
	}
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	// Don't retry Modbus protocol errors
	var modbusErr *ModbusError
	if errors.As(err, &modbusErr) {
		return false
	}
	// Don't retry context errors
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	// Retry connection errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	return true
}

// ReadCoils reads coils from the server (FC01).
func (c *Client) ReadCoils(ctx context.Context, addr, qty uint16) ([]bool, error) {
	pdu, err := BuildReadCoilsPDU(addr, qty)
	if err != nil {
		return nil, err
	}
	resp, err := c.send(ctx, pdu)
	if err != nil {
		return nil, err
	}
	return ParseCoilsResponse(resp, qty)
}

// ReadDiscreteInputs reads discrete inputs from the server (FC02).
func (c *Client) ReadDiscreteInputs(ctx context.Context, addr, qty uint16) ([]bool, error) {
	pdu, err := BuildReadDiscreteInputsPDU(addr, qty)
	if err != nil {
		return nil, err
	}
	resp, err := c.send(ctx, pdu)
	if err != nil {
		return nil, err
	}
	return ParseCoilsResponse(resp, qty)
}

// ReadHoldingRegisters reads holding registers from the server (FC03).
func (c *Client) ReadHoldingRegisters(ctx context.Context, addr, qty uint16) ([]uint16, error) {
	pdu, err := BuildReadHoldingRegistersPDU(addr, qty)
	if err != nil {
		return nil, err
	}
	resp, err := c.send(ctx, pdu)
	if err != nil {
		return nil, err
	}
	return ParseRegistersResponse(resp, qty)
}

// ReadInputRegisters reads input registers from the server (FC04).
func (c *Client) ReadInputRegisters(ctx context.Context, addr, qty uint16) ([]uint16, error) {
	pdu, err := BuildReadInputRegistersPDU(addr, qty)
	if err != nil {
		return nil, err
	}
	resp, err := c.send(ctx, pdu)
	if err != nil {
		return nil, err
	}
	return ParseRegistersResponse(resp, qty)
}

// WriteSingleCoil writes a single coil (FC05).
func (c *Client) WriteSingleCoil(ctx context.Context, addr uint16, value bool) error {
	pdu := BuildWriteSingleCoilPDU(addr, value)
	resp, err := c.send(ctx, pdu)
	if err != nil {
		return err
	}
	expectedValue := CoilOff
	if value {
		expectedValue = CoilOn
	}
	return ParseWriteResponse(resp, addr, expectedValue)
}

// WriteSingleRegister writes a single register (FC06).
func (c *Client) WriteSingleRegister(ctx context.Context, addr, value uint16) error {
	pdu := BuildWriteSingleRegisterPDU(addr, value)
	resp, err := c.send(ctx, pdu)
	if err != nil {
		return err
	}
	return ParseWriteResponse(resp, addr, value)
}

// WriteMultipleCoils writes multiple coils (FC15).
func (c *Client) WriteMultipleCoils(ctx context.Context, addr uint16, values []bool) error {
	if len(values) == 0 {
		return ErrInvalidQuantity
	}
	pdu, err := BuildWriteMultipleCoilsPDU(addr, values)
	if err != nil {
		return err
	}
	resp, err := c.send(ctx, pdu)
	if err != nil {
		return err
	}
	return ParseWriteMultipleResponse(resp, addr, uint16(len(values)))
}

// WriteMultipleRegisters writes multiple registers (FC16).
func (c *Client) WriteMultipleRegisters(ctx context.Context, addr uint16, values []uint16) error {
	if len(values) == 0 {
		return ErrInvalidQuantity
	}
	pdu, err := BuildWriteMultipleRegistersPDU(addr, values)
	if err != nil {
		return err
	}
	resp, err := c.send(ctx, pdu)
	if err != nil {
		return err
	}
	return ParseWriteMultipleResponse(resp, addr, uint16(len(values)))
}

// ReadExceptionStatus reads the exception status (FC07).
func (c *Client) ReadExceptionStatus(ctx context.Context) (uint8, error) {
	pdu := BuildReadExceptionStatusPDU()
	resp, err := c.send(ctx, pdu)
	if err != nil {
		return 0, err
	}
	return ParseExceptionStatusResponse(resp)
}

// Diagnostics performs a diagnostic operation (FC08).
func (c *Client) Diagnostics(ctx context.Context, subFunc uint16, data []byte) ([]byte, error) {
	pdu := BuildDiagnosticsPDU(subFunc, data)
	resp, err := c.send(ctx, pdu)
	if err != nil {
		return nil, err
	}
	_, respData, err := ParseDiagnosticsResponse(resp)
	return respData, err
}

// GetCommEventCounter gets the communication event counter (FC11).
func (c *Client) GetCommEventCounter(ctx context.Context) (status, eventCount uint16, err error) {
	pdu := BuildGetCommEventCounterPDU()
	resp, err := c.send(ctx, pdu)
	if err != nil {
		return 0, 0, err
	}
	return ParseGetCommEventCounterResponse(resp)
}

// ReportServerID requests the server ID (FC17).
func (c *Client) ReportServerID(ctx context.Context) ([]byte, error) {
	pdu := BuildReportServerIDPDU()
	resp, err := c.send(ctx, pdu)
	if err != nil {
		return nil, err
	}
	return ParseReportServerIDResponse(resp)
}

// ReadCoilsWithUnit reads coils using a specific unit ID.
func (c *Client) ReadCoilsWithUnit(ctx context.Context, unitID UnitID, addr, qty uint16) ([]bool, error) {
	pdu, err := BuildReadCoilsPDU(addr, qty)
	if err != nil {
		return nil, err
	}
	resp, err := c.sendWithUnit(ctx, unitID, pdu)
	if err != nil {
		return nil, err
	}
	return ParseCoilsResponse(resp, qty)
}

// ReadDiscreteInputsWithUnit reads discrete inputs using a specific unit ID.
func (c *Client) ReadDiscreteInputsWithUnit(ctx context.Context, unitID UnitID, addr, qty uint16) ([]bool, error) {
	pdu, err := BuildReadDiscreteInputsPDU(addr, qty)
	if err != nil {
		return nil, err
	}
	resp, err := c.sendWithUnit(ctx, unitID, pdu)
	if err != nil {
		return nil, err
	}
	return ParseCoilsResponse(resp, qty)
}

// ReadHoldingRegistersWithUnit reads holding registers using a specific unit ID.
func (c *Client) ReadHoldingRegistersWithUnit(ctx context.Context, unitID UnitID, addr, qty uint16) ([]uint16, error) {
	pdu, err := BuildReadHoldingRegistersPDU(addr, qty)
	if err != nil {
		return nil, err
	}
	resp, err := c.sendWithUnit(ctx, unitID, pdu)
	if err != nil {
		return nil, err
	}
	return ParseRegistersResponse(resp, qty)
}

// ReadInputRegistersWithUnit reads input registers using a specific unit ID.
func (c *Client) ReadInputRegistersWithUnit(ctx context.Context, unitID UnitID, addr, qty uint16) ([]uint16, error) {
	pdu, err := BuildReadInputRegistersPDU(addr, qty)
	if err != nil {
		return nil, err
	}
	resp, err := c.sendWithUnit(ctx, unitID, pdu)
	if err != nil {
		return nil, err
	}
	return ParseRegistersResponse(resp, qty)
}

// WriteSingleCoilWithUnit writes a single coil using a specific unit ID.
func (c *Client) WriteSingleCoilWithUnit(ctx context.Context, unitID UnitID, addr uint16, value bool) error {
	pdu := BuildWriteSingleCoilPDU(addr, value)
	resp, err := c.sendWithUnit(ctx, unitID, pdu)
	if err != nil {
		return err
	}
	expectedValue := CoilOff
	if value {
		expectedValue = CoilOn
	}
	return ParseWriteResponse(resp, addr, expectedValue)
}

// WriteSingleRegisterWithUnit writes a single register using a specific unit ID.
func (c *Client) WriteSingleRegisterWithUnit(ctx context.Context, unitID UnitID, addr, value uint16) error {
	pdu := BuildWriteSingleRegisterPDU(addr, value)
	resp, err := c.sendWithUnit(ctx, unitID, pdu)
	if err != nil {
		return err
	}
	return ParseWriteResponse(resp, addr, value)
}

// WriteMultipleCoilsWithUnit writes multiple coils using a specific unit ID.
func (c *Client) WriteMultipleCoilsWithUnit(ctx context.Context, unitID UnitID, addr uint16, values []bool) error {
	if len(values) == 0 {
		return ErrInvalidQuantity
	}
	pdu, err := BuildWriteMultipleCoilsPDU(addr, values)
	if err != nil {
		return err
	}
	resp, err := c.sendWithUnit(ctx, unitID, pdu)
	if err != nil {
		return err
	}
	return ParseWriteMultipleResponse(resp, addr, uint16(len(values)))
}

// WriteMultipleRegistersWithUnit writes multiple registers using a specific unit ID.
func (c *Client) WriteMultipleRegistersWithUnit(ctx context.Context, unitID UnitID, addr uint16, values []uint16) error {
	if len(values) == 0 {
		return ErrInvalidQuantity
	}
	pdu, err := BuildWriteMultipleRegistersPDU(addr, values)
	if err != nil {
		return err
	}
	resp, err := c.sendWithUnit(ctx, unitID, pdu)
	if err != nil {
		return err
	}
	return ParseWriteMultipleResponse(resp, addr, uint16(len(values)))
}
