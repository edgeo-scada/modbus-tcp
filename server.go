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
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

// Server is a Modbus TCP server.
type Server struct {
	handler Handler
	opts    *serverOptions

	mu       sync.Mutex
	listener net.Listener
	conns    map[net.Conn]struct{}
	closed   int32
	wg       sync.WaitGroup
	metrics  *ServerMetrics
}

// ServerMetrics holds server-side metrics.
type ServerMetrics struct {
	RequestsTotal   Counter
	RequestsSuccess Counter
	RequestsErrors  Counter
	ActiveConns     Counter
	TotalConns      Counter
}

// NewServer creates a new Modbus TCP server.
func NewServer(handler Handler, opts ...ServerOption) *Server {
	options := defaultServerOptions()
	for _, opt := range opts {
		opt(options)
	}

	return &Server{
		handler: handler,
		opts:    options,
		conns:   make(map[net.Conn]struct{}),
		metrics: &ServerMetrics{},
	}
}

// Metrics returns the server metrics.
func (s *Server) Metrics() *ServerMetrics {
	return s.metrics
}

// ListenAndServe starts the server on the given address.
func (s *Server) ListenAndServe(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.Serve(listener)
}

// Serve starts serving connections on the given listener.
func (s *Server) Serve(listener net.Listener) error {
	s.mu.Lock()
	s.listener = listener
	s.mu.Unlock()
	s.opts.logger.Info("server started", slog.String("addr", listener.Addr().String()))

	for {
		conn, err := listener.Accept()
		if err != nil {
			if atomic.LoadInt32(&s.closed) == 1 {
				return nil
			}
			s.opts.logger.Error("accept error", slog.String("error", err.Error()))
			continue
		}

		s.mu.Lock()
		if len(s.conns) >= s.opts.maxConns {
			s.mu.Unlock()
			s.opts.logger.Warn("max connections reached, rejecting",
				slog.String("remote", conn.RemoteAddr().String()))
			conn.Close()
			continue
		}
		s.conns[conn] = struct{}{}
		s.metrics.ActiveConns.Add(1)
		s.metrics.TotalConns.Add(1)
		s.mu.Unlock()

		// Configure TCP options
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			tcpConn.SetKeepAlive(true)
			tcpConn.SetKeepAlivePeriod(30 * time.Second)
			tcpConn.SetNoDelay(true)
		}

		s.wg.Add(1)
		go s.handleConn(conn)
	}
}

// Close shuts down the server gracefully.
func (s *Server) Close() error {
	if !atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		return nil
	}

	s.mu.Lock()
	var err error
	if s.listener != nil {
		err = s.listener.Close()
	}
	for conn := range s.conns {
		conn.Close()
	}
	s.mu.Unlock()

	s.wg.Wait()
	s.opts.logger.Info("server stopped")
	return err
}

// Addr returns the server's address.
func (s *Server) Addr() net.Addr {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		return s.listener.Addr()
	}
	return nil
}

// ActiveConnections returns the number of active connections.
func (s *Server) ActiveConnections() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.conns)
}

func (s *Server) handleConn(conn net.Conn) {
	defer func() {
		// Recover from panic to prevent server crash
		if r := recover(); r != nil {
			s.opts.logger.Error("panic in connection handler",
				slog.String("remote", conn.RemoteAddr().String()),
				slog.Any("panic", r),
				slog.String("stack", string(debug.Stack())))
		}

		s.wg.Done()
		conn.Close()
		s.mu.Lock()
		delete(s.conns, conn)
		s.metrics.ActiveConns.Add(-1)
		s.mu.Unlock()
	}()

	s.opts.logger.Debug("connection accepted",
		slog.String("remote", conn.RemoteAddr().String()))

	for {
		if atomic.LoadInt32(&s.closed) == 1 {
			return
		}

		if s.opts.readTimeout > 0 {
			conn.SetReadDeadline(timeNow().Add(s.opts.readTimeout))
		}

		frame, err := ReadFrame(conn)
		if err != nil {
			if err != io.EOF && atomic.LoadInt32(&s.closed) == 0 {
				// Don't log timeout errors as they're expected for idle connections
				if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
					s.opts.logger.Debug("read error",
						slog.String("remote", conn.RemoteAddr().String()),
						slog.String("error", err.Error()))
				}
			}
			return
		}

		s.metrics.RequestsTotal.Add(1)
		response := s.processRequest(frame)

		// Set write deadline
		if s.opts.readTimeout > 0 {
			conn.SetWriteDeadline(timeNow().Add(s.opts.readTimeout))
		}

		if _, err := conn.Write(response.Encode()); err != nil {
			s.metrics.RequestsErrors.Add(1)
			s.opts.logger.Debug("write error",
				slog.String("remote", conn.RemoteAddr().String()),
				slog.String("error", err.Error()))
			return
		}

		s.metrics.RequestsSuccess.Add(1)
	}
}

func (s *Server) processRequest(req *Frame) *Frame {
	resp := &Frame{
		Header: MBAPHeader{
			TransactionID: req.Header.TransactionID,
			ProtocolID:    ProtocolID,
			UnitID:        req.Header.UnitID,
		},
	}

	if len(req.PDU) < 1 {
		resp.PDU = s.buildException(0, ExceptionIllegalFunction)
		return resp
	}

	fc := FunctionCode(req.PDU[0])
	unitID := req.Header.UnitID

	s.opts.logger.Debug("processing request",
		slog.Uint64("tx_id", uint64(req.Header.TransactionID)),
		slog.Uint64("unit_id", uint64(unitID)),
		slog.String("func", fc.String()))

	var pdu []byte
	var err error

	switch fc {
	case FuncReadCoils:
		pdu, err = s.handleReadCoils(unitID, req.PDU)
	case FuncReadDiscreteInputs:
		pdu, err = s.handleReadDiscreteInputs(unitID, req.PDU)
	case FuncReadHoldingRegisters:
		pdu, err = s.handleReadHoldingRegisters(unitID, req.PDU)
	case FuncReadInputRegisters:
		pdu, err = s.handleReadInputRegisters(unitID, req.PDU)
	case FuncWriteSingleCoil:
		pdu, err = s.handleWriteSingleCoil(unitID, req.PDU)
	case FuncWriteSingleRegister:
		pdu, err = s.handleWriteSingleRegister(unitID, req.PDU)
	case FuncReadExceptionStatus:
		pdu, err = s.handleReadExceptionStatus(unitID, req.PDU)
	case FuncDiagnostics:
		pdu, err = s.handleDiagnostics(unitID, req.PDU)
	case FuncGetCommEventCounter:
		pdu, err = s.handleGetCommEventCounter(unitID, req.PDU)
	case FuncWriteMultipleCoils:
		pdu, err = s.handleWriteMultipleCoils(unitID, req.PDU)
	case FuncWriteMultipleRegisters:
		pdu, err = s.handleWriteMultipleRegisters(unitID, req.PDU)
	case FuncReportServerID:
		pdu, err = s.handleReportServerID(unitID, req.PDU)
	default:
		pdu = s.buildException(fc, ExceptionIllegalFunction)
	}

	if err != nil {
		pdu = s.handleError(fc, err)
	}

	resp.PDU = pdu
	return resp
}

func (s *Server) buildException(fc FunctionCode, ec ExceptionCode) []byte {
	return []byte{byte(fc) | 0x80, byte(ec)}
}

func (s *Server) handleError(fc FunctionCode, err error) []byte {
	if modbusErr, ok := err.(*ModbusError); ok {
		return s.buildException(fc, modbusErr.ExceptionCode)
	}
	s.opts.logger.Error("handler error",
		slog.String("func", fc.String()),
		slog.String("error", err.Error()))
	return s.buildException(fc, ExceptionServerDeviceFailure)
}

func (s *Server) handleReadCoils(unitID UnitID, pdu []byte) ([]byte, error) {
	if len(pdu) < 5 {
		return s.buildException(FuncReadCoils, ExceptionIllegalDataValue), nil
	}
	addr := binary.BigEndian.Uint16(pdu[1:3])
	qty := binary.BigEndian.Uint16(pdu[3:5])

	if qty < 1 || qty > MaxQuantityCoils {
		return s.buildException(FuncReadCoils, ExceptionIllegalDataValue), nil
	}

	// Check for address overflow
	if uint32(addr)+uint32(qty) > 65536 {
		return s.buildException(FuncReadCoils, ExceptionIllegalDataAddress), nil
	}

	values, err := s.handler.ReadCoils(unitID, addr, qty)
	if err != nil {
		return nil, err
	}

	// Validate handler returned correct number of values
	if uint16(len(values)) != qty {
		return s.buildException(FuncReadCoils, ExceptionServerDeviceFailure), nil
	}

	byteCount := (qty + 7) / 8
	resp := make([]byte, 2+byteCount)
	resp[0] = byte(FuncReadCoils)
	resp[1] = byte(byteCount)
	for i, v := range values {
		if v {
			resp[2+i/8] |= 1 << (i % 8)
		}
	}
	return resp, nil
}

func (s *Server) handleReadDiscreteInputs(unitID UnitID, pdu []byte) ([]byte, error) {
	if len(pdu) < 5 {
		return s.buildException(FuncReadDiscreteInputs, ExceptionIllegalDataValue), nil
	}
	addr := binary.BigEndian.Uint16(pdu[1:3])
	qty := binary.BigEndian.Uint16(pdu[3:5])

	if qty < 1 || qty > MaxQuantityDiscreteInputs {
		return s.buildException(FuncReadDiscreteInputs, ExceptionIllegalDataValue), nil
	}

	if uint32(addr)+uint32(qty) > 65536 {
		return s.buildException(FuncReadDiscreteInputs, ExceptionIllegalDataAddress), nil
	}

	values, err := s.handler.ReadDiscreteInputs(unitID, addr, qty)
	if err != nil {
		return nil, err
	}

	if uint16(len(values)) != qty {
		return s.buildException(FuncReadDiscreteInputs, ExceptionServerDeviceFailure), nil
	}

	byteCount := (qty + 7) / 8
	resp := make([]byte, 2+byteCount)
	resp[0] = byte(FuncReadDiscreteInputs)
	resp[1] = byte(byteCount)
	for i, v := range values {
		if v {
			resp[2+i/8] |= 1 << (i % 8)
		}
	}
	return resp, nil
}

func (s *Server) handleReadHoldingRegisters(unitID UnitID, pdu []byte) ([]byte, error) {
	if len(pdu) < 5 {
		return s.buildException(FuncReadHoldingRegisters, ExceptionIllegalDataValue), nil
	}
	addr := binary.BigEndian.Uint16(pdu[1:3])
	qty := binary.BigEndian.Uint16(pdu[3:5])

	if qty < 1 || qty > MaxQuantityRegisters {
		return s.buildException(FuncReadHoldingRegisters, ExceptionIllegalDataValue), nil
	}

	if uint32(addr)+uint32(qty) > 65536 {
		return s.buildException(FuncReadHoldingRegisters, ExceptionIllegalDataAddress), nil
	}

	values, err := s.handler.ReadHoldingRegisters(unitID, addr, qty)
	if err != nil {
		return nil, err
	}

	if uint16(len(values)) != qty {
		return s.buildException(FuncReadHoldingRegisters, ExceptionServerDeviceFailure), nil
	}

	byteCount := qty * 2
	resp := make([]byte, 2+byteCount)
	resp[0] = byte(FuncReadHoldingRegisters)
	resp[1] = byte(byteCount)
	for i, v := range values {
		binary.BigEndian.PutUint16(resp[2+i*2:], v)
	}
	return resp, nil
}

func (s *Server) handleReadInputRegisters(unitID UnitID, pdu []byte) ([]byte, error) {
	if len(pdu) < 5 {
		return s.buildException(FuncReadInputRegisters, ExceptionIllegalDataValue), nil
	}
	addr := binary.BigEndian.Uint16(pdu[1:3])
	qty := binary.BigEndian.Uint16(pdu[3:5])

	if qty < 1 || qty > MaxQuantityRegisters {
		return s.buildException(FuncReadInputRegisters, ExceptionIllegalDataValue), nil
	}

	if uint32(addr)+uint32(qty) > 65536 {
		return s.buildException(FuncReadInputRegisters, ExceptionIllegalDataAddress), nil
	}

	values, err := s.handler.ReadInputRegisters(unitID, addr, qty)
	if err != nil {
		return nil, err
	}

	if uint16(len(values)) != qty {
		return s.buildException(FuncReadInputRegisters, ExceptionServerDeviceFailure), nil
	}

	byteCount := qty * 2
	resp := make([]byte, 2+byteCount)
	resp[0] = byte(FuncReadInputRegisters)
	resp[1] = byte(byteCount)
	for i, v := range values {
		binary.BigEndian.PutUint16(resp[2+i*2:], v)
	}
	return resp, nil
}

func (s *Server) handleWriteSingleCoil(unitID UnitID, pdu []byte) ([]byte, error) {
	if len(pdu) < 5 {
		return s.buildException(FuncWriteSingleCoil, ExceptionIllegalDataValue), nil
	}
	addr := binary.BigEndian.Uint16(pdu[1:3])
	value := binary.BigEndian.Uint16(pdu[3:5])

	var boolValue bool
	if value == CoilOn {
		boolValue = true
	} else if value != CoilOff {
		return s.buildException(FuncWriteSingleCoil, ExceptionIllegalDataValue), nil
	}

	if err := s.handler.WriteSingleCoil(unitID, addr, boolValue); err != nil {
		return nil, err
	}

	// Echo request as response (copy to avoid sharing slice)
	resp := make([]byte, 5)
	copy(resp, pdu[:5])
	return resp, nil
}

func (s *Server) handleWriteSingleRegister(unitID UnitID, pdu []byte) ([]byte, error) {
	if len(pdu) < 5 {
		return s.buildException(FuncWriteSingleRegister, ExceptionIllegalDataValue), nil
	}
	addr := binary.BigEndian.Uint16(pdu[1:3])
	value := binary.BigEndian.Uint16(pdu[3:5])

	if err := s.handler.WriteSingleRegister(unitID, addr, value); err != nil {
		return nil, err
	}

	// Echo request as response (copy to avoid sharing slice)
	resp := make([]byte, 5)
	copy(resp, pdu[:5])
	return resp, nil
}

func (s *Server) handleReadExceptionStatus(unitID UnitID, pdu []byte) ([]byte, error) {
	status, err := s.handler.ReadExceptionStatus(unitID)
	if err != nil {
		return nil, err
	}

	return []byte{byte(FuncReadExceptionStatus), status}, nil
}

func (s *Server) handleDiagnostics(unitID UnitID, pdu []byte) ([]byte, error) {
	if len(pdu) < 3 {
		return s.buildException(FuncDiagnostics, ExceptionIllegalDataValue), nil
	}
	subFunc := binary.BigEndian.Uint16(pdu[1:3])
	data := pdu[3:]

	respData, err := s.handler.Diagnostics(unitID, subFunc, data)
	if err != nil {
		return nil, err
	}

	resp := make([]byte, 3+len(respData))
	resp[0] = byte(FuncDiagnostics)
	binary.BigEndian.PutUint16(resp[1:3], subFunc)
	copy(resp[3:], respData)
	return resp, nil
}

func (s *Server) handleGetCommEventCounter(unitID UnitID, pdu []byte) ([]byte, error) {
	status, eventCount, err := s.handler.GetCommEventCounter(unitID)
	if err != nil {
		return nil, err
	}

	resp := make([]byte, 5)
	resp[0] = byte(FuncGetCommEventCounter)
	binary.BigEndian.PutUint16(resp[1:3], status)
	binary.BigEndian.PutUint16(resp[3:5], eventCount)
	return resp, nil
}

func (s *Server) handleWriteMultipleCoils(unitID UnitID, pdu []byte) ([]byte, error) {
	if len(pdu) < 6 {
		return s.buildException(FuncWriteMultipleCoils, ExceptionIllegalDataValue), nil
	}
	addr := binary.BigEndian.Uint16(pdu[1:3])
	qty := binary.BigEndian.Uint16(pdu[3:5])
	byteCount := int(pdu[5])

	if qty < 1 || qty > MaxQuantityCoils {
		return s.buildException(FuncWriteMultipleCoils, ExceptionIllegalDataValue), nil
	}

	if uint32(addr)+uint32(qty) > 65536 {
		return s.buildException(FuncWriteMultipleCoils, ExceptionIllegalDataAddress), nil
	}

	expectedBytes := int((qty + 7) / 8)
	if byteCount != expectedBytes || len(pdu) < 6+byteCount {
		return s.buildException(FuncWriteMultipleCoils, ExceptionIllegalDataValue), nil
	}

	values := make([]bool, qty)
	for i := uint16(0); i < qty; i++ {
		values[i] = (pdu[6+i/8] & (1 << (i % 8))) != 0
	}

	if err := s.handler.WriteMultipleCoils(unitID, addr, values); err != nil {
		return nil, err
	}

	resp := make([]byte, 5)
	resp[0] = byte(FuncWriteMultipleCoils)
	binary.BigEndian.PutUint16(resp[1:3], addr)
	binary.BigEndian.PutUint16(resp[3:5], qty)
	return resp, nil
}

func (s *Server) handleWriteMultipleRegisters(unitID UnitID, pdu []byte) ([]byte, error) {
	if len(pdu) < 6 {
		return s.buildException(FuncWriteMultipleRegisters, ExceptionIllegalDataValue), nil
	}
	addr := binary.BigEndian.Uint16(pdu[1:3])
	qty := binary.BigEndian.Uint16(pdu[3:5])
	byteCount := int(pdu[5])

	if qty < 1 || qty > MaxQuantityWriteRegisters {
		return s.buildException(FuncWriteMultipleRegisters, ExceptionIllegalDataValue), nil
	}

	if uint32(addr)+uint32(qty) > 65536 {
		return s.buildException(FuncWriteMultipleRegisters, ExceptionIllegalDataAddress), nil
	}

	expectedBytes := int(qty * 2)
	if byteCount != expectedBytes || len(pdu) < 6+byteCount {
		return s.buildException(FuncWriteMultipleRegisters, ExceptionIllegalDataValue), nil
	}

	values := make([]uint16, qty)
	for i := uint16(0); i < qty; i++ {
		values[i] = binary.BigEndian.Uint16(pdu[6+i*2:])
	}

	if err := s.handler.WriteMultipleRegisters(unitID, addr, values); err != nil {
		return nil, err
	}

	resp := make([]byte, 5)
	resp[0] = byte(FuncWriteMultipleRegisters)
	binary.BigEndian.PutUint16(resp[1:3], addr)
	binary.BigEndian.PutUint16(resp[3:5], qty)
	return resp, nil
}

func (s *Server) handleReportServerID(unitID UnitID, pdu []byte) ([]byte, error) {
	data, err := s.handler.ReportServerID(unitID)
	if err != nil {
		return nil, err
	}

	// Limit server ID length
	if len(data) > 251 {
		data = data[:251]
	}

	resp := make([]byte, 2+len(data))
	resp[0] = byte(FuncReportServerID)
	resp[1] = byte(len(data))
	copy(resp[2:], data)
	return resp, nil
}

// timeNow is a variable for testing
var timeNow = time.Now

// MemoryHandler is a simple in-memory implementation of Handler.
// It is thread-safe and suitable for testing and simulation.
type MemoryHandler struct {
	mu             sync.RWMutex
	coils          map[UnitID][]bool
	discreteInputs map[UnitID][]bool
	holdingRegs    map[UnitID][]uint16
	inputRegs      map[UnitID][]uint16
	serverID       []byte
	eventCounter   uint16
	initialized    map[UnitID]bool
}

// NewMemoryHandler creates a new MemoryHandler.
func NewMemoryHandler(coilSize, registerSize int) *MemoryHandler {
	return &MemoryHandler{
		coils:          make(map[UnitID][]bool),
		discreteInputs: make(map[UnitID][]bool),
		holdingRegs:    make(map[UnitID][]uint16),
		inputRegs:      make(map[UnitID][]uint16),
		serverID:       []byte("Modbus Server"),
		initialized:    make(map[UnitID]bool),
	}
}

// ensureUnitLocked initializes data for a unit if not already done.
// Must be called with write lock held.
func (h *MemoryHandler) ensureUnitLocked(unitID UnitID) {
	if !h.initialized[unitID] {
		h.coils[unitID] = make([]bool, 65536)
		h.discreteInputs[unitID] = make([]bool, 65536)
		h.holdingRegs[unitID] = make([]uint16, 65536)
		h.inputRegs[unitID] = make([]uint16, 65536)
		h.initialized[unitID] = true
	}
}

// getOrInitUnit gets unit data, initializing if needed.
func (h *MemoryHandler) getOrInitUnit(unitID UnitID) {
	h.mu.RLock()
	if h.initialized[unitID] {
		h.mu.RUnlock()
		return
	}
	h.mu.RUnlock()

	h.mu.Lock()
	h.ensureUnitLocked(unitID)
	h.mu.Unlock()
}

func (h *MemoryHandler) ReadCoils(unitID UnitID, addr, qty uint16) ([]bool, error) {
	h.getOrInitUnit(unitID)

	h.mu.RLock()
	defer h.mu.RUnlock()

	if int(addr)+int(qty) > len(h.coils[unitID]) {
		return nil, NewModbusError(FuncReadCoils, ExceptionIllegalDataAddress)
	}

	result := make([]bool, qty)
	copy(result, h.coils[unitID][addr:addr+qty])
	return result, nil
}

func (h *MemoryHandler) ReadDiscreteInputs(unitID UnitID, addr, qty uint16) ([]bool, error) {
	h.getOrInitUnit(unitID)

	h.mu.RLock()
	defer h.mu.RUnlock()

	if int(addr)+int(qty) > len(h.discreteInputs[unitID]) {
		return nil, NewModbusError(FuncReadDiscreteInputs, ExceptionIllegalDataAddress)
	}

	result := make([]bool, qty)
	copy(result, h.discreteInputs[unitID][addr:addr+qty])
	return result, nil
}

func (h *MemoryHandler) WriteSingleCoil(unitID UnitID, addr uint16, value bool) error {
	h.getOrInitUnit(unitID)

	h.mu.Lock()
	defer h.mu.Unlock()

	if int(addr) >= len(h.coils[unitID]) {
		return NewModbusError(FuncWriteSingleCoil, ExceptionIllegalDataAddress)
	}

	h.coils[unitID][addr] = value
	return nil
}

func (h *MemoryHandler) WriteMultipleCoils(unitID UnitID, addr uint16, values []bool) error {
	h.getOrInitUnit(unitID)

	h.mu.Lock()
	defer h.mu.Unlock()

	if int(addr)+len(values) > len(h.coils[unitID]) {
		return NewModbusError(FuncWriteMultipleCoils, ExceptionIllegalDataAddress)
	}

	copy(h.coils[unitID][addr:], values)
	return nil
}

func (h *MemoryHandler) ReadHoldingRegisters(unitID UnitID, addr, qty uint16) ([]uint16, error) {
	h.getOrInitUnit(unitID)

	h.mu.RLock()
	defer h.mu.RUnlock()

	if int(addr)+int(qty) > len(h.holdingRegs[unitID]) {
		return nil, NewModbusError(FuncReadHoldingRegisters, ExceptionIllegalDataAddress)
	}

	result := make([]uint16, qty)
	copy(result, h.holdingRegs[unitID][addr:addr+qty])
	return result, nil
}

func (h *MemoryHandler) ReadInputRegisters(unitID UnitID, addr, qty uint16) ([]uint16, error) {
	h.getOrInitUnit(unitID)

	h.mu.RLock()
	defer h.mu.RUnlock()

	if int(addr)+int(qty) > len(h.inputRegs[unitID]) {
		return nil, NewModbusError(FuncReadInputRegisters, ExceptionIllegalDataAddress)
	}

	result := make([]uint16, qty)
	copy(result, h.inputRegs[unitID][addr:addr+qty])
	return result, nil
}

func (h *MemoryHandler) WriteSingleRegister(unitID UnitID, addr, value uint16) error {
	h.getOrInitUnit(unitID)

	h.mu.Lock()
	defer h.mu.Unlock()

	if int(addr) >= len(h.holdingRegs[unitID]) {
		return NewModbusError(FuncWriteSingleRegister, ExceptionIllegalDataAddress)
	}

	h.holdingRegs[unitID][addr] = value
	return nil
}

func (h *MemoryHandler) WriteMultipleRegisters(unitID UnitID, addr uint16, values []uint16) error {
	h.getOrInitUnit(unitID)

	h.mu.Lock()
	defer h.mu.Unlock()

	if int(addr)+len(values) > len(h.holdingRegs[unitID]) {
		return NewModbusError(FuncWriteMultipleRegisters, ExceptionIllegalDataAddress)
	}

	copy(h.holdingRegs[unitID][addr:], values)
	return nil
}

func (h *MemoryHandler) ReadExceptionStatus(unitID UnitID) (uint8, error) {
	return 0, nil
}

func (h *MemoryHandler) Diagnostics(unitID UnitID, subFunc uint16, data []byte) ([]byte, error) {
	switch subFunc {
	case DiagReturnQueryData:
		// Echo data back (make a copy)
		result := make([]byte, len(data))
		copy(result, data)
		return result, nil
	default:
		return nil, NewModbusError(FuncDiagnostics, ExceptionIllegalFunction)
	}
}

func (h *MemoryHandler) GetCommEventCounter(unitID UnitID) (status uint16, eventCount uint16, err error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return 0xFFFF, h.eventCounter, nil
}

func (h *MemoryHandler) ReportServerID(unitID UnitID) ([]byte, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	// Return a copy
	result := make([]byte, len(h.serverID))
	copy(result, h.serverID)
	return result, nil
}

// SetServerID sets the server ID returned by ReportServerID.
func (h *MemoryHandler) SetServerID(id []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.serverID = make([]byte, len(id))
	copy(h.serverID, id)
}

// SetCoil sets a coil value directly.
func (h *MemoryHandler) SetCoil(unitID UnitID, addr uint16, value bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ensureUnitLocked(unitID)
	if int(addr) < len(h.coils[unitID]) {
		h.coils[unitID][addr] = value
	}
}

// SetDiscreteInput sets a discrete input value directly.
func (h *MemoryHandler) SetDiscreteInput(unitID UnitID, addr uint16, value bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ensureUnitLocked(unitID)
	if int(addr) < len(h.discreteInputs[unitID]) {
		h.discreteInputs[unitID][addr] = value
	}
}

// SetHoldingRegister sets a holding register value directly.
func (h *MemoryHandler) SetHoldingRegister(unitID UnitID, addr, value uint16) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ensureUnitLocked(unitID)
	if int(addr) < len(h.holdingRegs[unitID]) {
		h.holdingRegs[unitID][addr] = value
	}
}

// SetInputRegister sets an input register value directly.
func (h *MemoryHandler) SetInputRegister(unitID UnitID, addr, value uint16) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.ensureUnitLocked(unitID)
	if int(addr) < len(h.inputRegs[unitID]) {
		h.inputRegs[unitID][addr] = value
	}
}

// ListenAndServeContext starts the server with context support.
func (s *Server) ListenAndServeContext(ctx context.Context, addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	go func() {
		<-ctx.Done()
		s.Close()
	}()

	return s.Serve(listener)
}
