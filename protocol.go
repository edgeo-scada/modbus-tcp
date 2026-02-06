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
	"encoding/binary"
	"fmt"
	"io"
	"sync/atomic"
)

// MBAPHeader represents the Modbus Application Protocol header for TCP.
type MBAPHeader struct {
	TransactionID uint16 // Transaction identifier
	ProtocolID    uint16 // Protocol identifier (always 0 for Modbus)
	Length        uint16 // Number of following bytes (Unit ID + PDU)
	UnitID        UnitID // Unit identifier (slave address)
}

// Encode encodes the MBAP header to bytes.
func (h *MBAPHeader) Encode() []byte {
	buf := make([]byte, MBAPHeaderSize)
	binary.BigEndian.PutUint16(buf[0:2], h.TransactionID)
	binary.BigEndian.PutUint16(buf[2:4], h.ProtocolID)
	binary.BigEndian.PutUint16(buf[4:6], h.Length)
	buf[6] = byte(h.UnitID)
	return buf
}

// Decode decodes the MBAP header from bytes.
func (h *MBAPHeader) Decode(data []byte) error {
	if len(data) < MBAPHeaderSize {
		return fmt.Errorf("%w: MBAP header too short", ErrInvalidFrame)
	}
	h.TransactionID = binary.BigEndian.Uint16(data[0:2])
	h.ProtocolID = binary.BigEndian.Uint16(data[2:4])
	h.Length = binary.BigEndian.Uint16(data[4:6])
	h.UnitID = UnitID(data[6])
	return nil
}

// TransactionIDGenerator generates unique transaction IDs.
type TransactionIDGenerator struct {
	counter uint32
}

// Next returns the next transaction ID.
func (g *TransactionIDGenerator) Next() uint16 {
	return uint16(atomic.AddUint32(&g.counter, 1))
}

// Frame represents a complete Modbus TCP frame (MBAP header + PDU).
type Frame struct {
	Header MBAPHeader
	PDU    []byte
}

// Encode encodes the frame to bytes.
func (f *Frame) Encode() []byte {
	f.Header.Length = uint16(len(f.PDU) + 1) // PDU length + Unit ID
	header := f.Header.Encode()
	buf := make([]byte, MBAPHeaderSize+len(f.PDU))
	copy(buf, header)
	copy(buf[MBAPHeaderSize:], f.PDU)
	return buf
}

// Decode decodes a frame from bytes.
func (f *Frame) Decode(data []byte) error {
	if len(data) < MBAPHeaderSize {
		return fmt.Errorf("%w: frame too short", ErrInvalidFrame)
	}
	if err := f.Header.Decode(data[:MBAPHeaderSize]); err != nil {
		return err
	}
	pduLen := int(f.Header.Length) - 1 // Length includes Unit ID
	if pduLen < 0 {
		return fmt.Errorf("%w: invalid length field", ErrInvalidFrame)
	}
	if len(data) < MBAPHeaderSize+pduLen {
		return fmt.Errorf("%w: incomplete frame", ErrInvalidFrame)
	}
	f.PDU = make([]byte, pduLen)
	copy(f.PDU, data[MBAPHeaderSize:MBAPHeaderSize+pduLen])
	return nil
}

// ReadFrame reads a complete Modbus TCP frame from a reader.
func ReadFrame(r io.Reader) (*Frame, error) {
	header := make([]byte, MBAPHeaderSize)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}

	var f Frame
	if err := f.Header.Decode(header); err != nil {
		return nil, err
	}

	// Validate protocol ID
	if f.Header.ProtocolID != ProtocolID {
		return nil, fmt.Errorf("%w: invalid protocol ID %d", ErrInvalidFrame, f.Header.ProtocolID)
	}

	// Read PDU
	pduLen := int(f.Header.Length) - 1
	if pduLen < 0 || pduLen > 253 { // Max PDU size is 253 bytes
		return nil, fmt.Errorf("%w: invalid PDU length %d", ErrInvalidFrame, pduLen)
	}

	f.PDU = make([]byte, pduLen)
	if _, err := io.ReadFull(r, f.PDU); err != nil {
		return nil, err
	}

	return &f, nil
}

// PDU builders for various function codes

// BuildReadCoilsPDU builds a PDU for reading coils (FC01).
func BuildReadCoilsPDU(addr, qty uint16) ([]byte, error) {
	if qty < 1 || qty > MaxQuantityCoils {
		return nil, fmt.Errorf("%w: quantity must be 1-%d", ErrInvalidQuantity, MaxQuantityCoils)
	}
	// Check for address overflow
	if uint32(addr)+uint32(qty) > 65536 {
		return nil, fmt.Errorf("%w: address range exceeds 65535", ErrInvalidAddress)
	}
	pdu := make([]byte, 5)
	pdu[0] = byte(FuncReadCoils)
	binary.BigEndian.PutUint16(pdu[1:3], addr)
	binary.BigEndian.PutUint16(pdu[3:5], qty)
	return pdu, nil
}

// BuildReadDiscreteInputsPDU builds a PDU for reading discrete inputs (FC02).
func BuildReadDiscreteInputsPDU(addr, qty uint16) ([]byte, error) {
	if qty < 1 || qty > MaxQuantityDiscreteInputs {
		return nil, fmt.Errorf("%w: quantity must be 1-%d", ErrInvalidQuantity, MaxQuantityDiscreteInputs)
	}
	if uint32(addr)+uint32(qty) > 65536 {
		return nil, fmt.Errorf("%w: address range exceeds 65535", ErrInvalidAddress)
	}
	pdu := make([]byte, 5)
	pdu[0] = byte(FuncReadDiscreteInputs)
	binary.BigEndian.PutUint16(pdu[1:3], addr)
	binary.BigEndian.PutUint16(pdu[3:5], qty)
	return pdu, nil
}

// BuildReadHoldingRegistersPDU builds a PDU for reading holding registers (FC03).
func BuildReadHoldingRegistersPDU(addr, qty uint16) ([]byte, error) {
	if qty < 1 || qty > MaxQuantityRegisters {
		return nil, fmt.Errorf("%w: quantity must be 1-%d", ErrInvalidQuantity, MaxQuantityRegisters)
	}
	if uint32(addr)+uint32(qty) > 65536 {
		return nil, fmt.Errorf("%w: address range exceeds 65535", ErrInvalidAddress)
	}
	pdu := make([]byte, 5)
	pdu[0] = byte(FuncReadHoldingRegisters)
	binary.BigEndian.PutUint16(pdu[1:3], addr)
	binary.BigEndian.PutUint16(pdu[3:5], qty)
	return pdu, nil
}

// BuildReadInputRegistersPDU builds a PDU for reading input registers (FC04).
func BuildReadInputRegistersPDU(addr, qty uint16) ([]byte, error) {
	if qty < 1 || qty > MaxQuantityRegisters {
		return nil, fmt.Errorf("%w: quantity must be 1-%d", ErrInvalidQuantity, MaxQuantityRegisters)
	}
	if uint32(addr)+uint32(qty) > 65536 {
		return nil, fmt.Errorf("%w: address range exceeds 65535", ErrInvalidAddress)
	}
	pdu := make([]byte, 5)
	pdu[0] = byte(FuncReadInputRegisters)
	binary.BigEndian.PutUint16(pdu[1:3], addr)
	binary.BigEndian.PutUint16(pdu[3:5], qty)
	return pdu, nil
}

// BuildWriteSingleCoilPDU builds a PDU for writing a single coil (FC05).
func BuildWriteSingleCoilPDU(addr uint16, value bool) []byte {
	pdu := make([]byte, 5)
	pdu[0] = byte(FuncWriteSingleCoil)
	binary.BigEndian.PutUint16(pdu[1:3], addr)
	if value {
		binary.BigEndian.PutUint16(pdu[3:5], CoilOn)
	} else {
		binary.BigEndian.PutUint16(pdu[3:5], CoilOff)
	}
	return pdu
}

// BuildWriteSingleRegisterPDU builds a PDU for writing a single register (FC06).
func BuildWriteSingleRegisterPDU(addr, value uint16) []byte {
	pdu := make([]byte, 5)
	pdu[0] = byte(FuncWriteSingleRegister)
	binary.BigEndian.PutUint16(pdu[1:3], addr)
	binary.BigEndian.PutUint16(pdu[3:5], value)
	return pdu
}

// BuildReadExceptionStatusPDU builds a PDU for reading exception status (FC07).
func BuildReadExceptionStatusPDU() []byte {
	return []byte{byte(FuncReadExceptionStatus)}
}

// BuildDiagnosticsPDU builds a PDU for diagnostics (FC08).
func BuildDiagnosticsPDU(subFunc uint16, data []byte) []byte {
	pdu := make([]byte, 3+len(data))
	pdu[0] = byte(FuncDiagnostics)
	binary.BigEndian.PutUint16(pdu[1:3], subFunc)
	copy(pdu[3:], data)
	return pdu
}

// BuildGetCommEventCounterPDU builds a PDU for getting comm event counter (FC11).
func BuildGetCommEventCounterPDU() []byte {
	return []byte{byte(FuncGetCommEventCounter)}
}

// BuildWriteMultipleCoilsPDU builds a PDU for writing multiple coils (FC15).
func BuildWriteMultipleCoilsPDU(addr uint16, values []bool) ([]byte, error) {
	qty := uint16(len(values))
	if qty < 1 || qty > MaxQuantityCoils {
		return nil, fmt.Errorf("%w: quantity must be 1-%d", ErrInvalidQuantity, MaxQuantityCoils)
	}
	if uint32(addr)+uint32(qty) > 65536 {
		return nil, fmt.Errorf("%w: address range exceeds 65535", ErrInvalidAddress)
	}
	byteCount := (qty + 7) / 8
	pdu := make([]byte, 6+byteCount)
	pdu[0] = byte(FuncWriteMultipleCoils)
	binary.BigEndian.PutUint16(pdu[1:3], addr)
	binary.BigEndian.PutUint16(pdu[3:5], qty)
	pdu[5] = byte(byteCount)

	// Pack coils into bytes
	for i, v := range values {
		if v {
			pdu[6+i/8] |= 1 << (i % 8)
		}
	}
	return pdu, nil
}

// BuildWriteMultipleRegistersPDU builds a PDU for writing multiple registers (FC16).
func BuildWriteMultipleRegistersPDU(addr uint16, values []uint16) ([]byte, error) {
	qty := uint16(len(values))
	if qty < 1 || qty > MaxQuantityWriteRegisters {
		return nil, fmt.Errorf("%w: quantity must be 1-%d", ErrInvalidQuantity, MaxQuantityWriteRegisters)
	}
	if uint32(addr)+uint32(qty) > 65536 {
		return nil, fmt.Errorf("%w: address range exceeds 65535", ErrInvalidAddress)
	}
	byteCount := qty * 2
	pdu := make([]byte, 6+byteCount)
	pdu[0] = byte(FuncWriteMultipleRegisters)
	binary.BigEndian.PutUint16(pdu[1:3], addr)
	binary.BigEndian.PutUint16(pdu[3:5], qty)
	pdu[5] = byte(byteCount)

	// Pack registers
	for i, v := range values {
		binary.BigEndian.PutUint16(pdu[6+i*2:], v)
	}
	return pdu, nil
}

// BuildReportServerIDPDU builds a PDU for reporting server ID (FC17).
func BuildReportServerIDPDU() []byte {
	return []byte{byte(FuncReportServerID)}
}

// Response parsing helpers

// ParseCoilsResponse parses a coils response (FC01/FC02) and returns the values.
func ParseCoilsResponse(pdu []byte, qty uint16) ([]bool, error) {
	if len(pdu) < 2 {
		return nil, fmt.Errorf("%w: response too short", ErrInvalidResponse)
	}
	byteCount := int(pdu[1])
	expectedBytes := int((qty + 7) / 8)
	if byteCount != expectedBytes || len(pdu) < 2+byteCount {
		return nil, fmt.Errorf("%w: invalid byte count", ErrInvalidResponse)
	}

	values := make([]bool, qty)
	for i := uint16(0); i < qty; i++ {
		values[i] = (pdu[2+i/8] & (1 << (i % 8))) != 0
	}
	return values, nil
}

// ParseRegistersResponse parses a registers response (FC03/FC04) and returns the values.
func ParseRegistersResponse(pdu []byte, qty uint16) ([]uint16, error) {
	if len(pdu) < 2 {
		return nil, fmt.Errorf("%w: response too short", ErrInvalidResponse)
	}
	byteCount := int(pdu[1])
	expectedBytes := int(qty * 2)
	if byteCount != expectedBytes || len(pdu) < 2+byteCount {
		return nil, fmt.Errorf("%w: invalid byte count", ErrInvalidResponse)
	}

	values := make([]uint16, qty)
	for i := uint16(0); i < qty; i++ {
		values[i] = binary.BigEndian.Uint16(pdu[2+i*2:])
	}
	return values, nil
}

// ParseWriteResponse parses a write response (FC05/FC06) and validates it.
func ParseWriteResponse(pdu []byte, expectedAddr, expectedValue uint16) error {
	if len(pdu) < 5 {
		return fmt.Errorf("%w: response too short", ErrInvalidResponse)
	}
	addr := binary.BigEndian.Uint16(pdu[1:3])
	value := binary.BigEndian.Uint16(pdu[3:5])
	if addr != expectedAddr {
		return fmt.Errorf("%w: address mismatch", ErrInvalidResponse)
	}
	if value != expectedValue {
		return fmt.Errorf("%w: value mismatch", ErrInvalidResponse)
	}
	return nil
}

// ParseWriteMultipleResponse parses a write multiple response (FC15/FC16) and validates it.
func ParseWriteMultipleResponse(pdu []byte, expectedAddr, expectedQty uint16) error {
	if len(pdu) < 5 {
		return fmt.Errorf("%w: response too short", ErrInvalidResponse)
	}
	addr := binary.BigEndian.Uint16(pdu[1:3])
	qty := binary.BigEndian.Uint16(pdu[3:5])
	if addr != expectedAddr {
		return fmt.Errorf("%w: address mismatch", ErrInvalidResponse)
	}
	if qty != expectedQty {
		return fmt.Errorf("%w: quantity mismatch", ErrInvalidResponse)
	}
	return nil
}

// ParseExceptionStatusResponse parses an exception status response (FC07).
func ParseExceptionStatusResponse(pdu []byte) (uint8, error) {
	if len(pdu) < 2 {
		return 0, fmt.Errorf("%w: response too short", ErrInvalidResponse)
	}
	return pdu[1], nil
}

// ParseDiagnosticsResponse parses a diagnostics response (FC08).
func ParseDiagnosticsResponse(pdu []byte) (subFunc uint16, data []byte, err error) {
	if len(pdu) < 3 {
		return 0, nil, fmt.Errorf("%w: response too short", ErrInvalidResponse)
	}
	subFunc = binary.BigEndian.Uint16(pdu[1:3])
	if len(pdu) > 3 {
		data = make([]byte, len(pdu)-3)
		copy(data, pdu[3:])
	}
	return subFunc, data, nil
}

// ParseGetCommEventCounterResponse parses a get comm event counter response (FC11).
func ParseGetCommEventCounterResponse(pdu []byte) (status, eventCount uint16, err error) {
	if len(pdu) < 5 {
		return 0, 0, fmt.Errorf("%w: response too short", ErrInvalidResponse)
	}
	status = binary.BigEndian.Uint16(pdu[1:3])
	eventCount = binary.BigEndian.Uint16(pdu[3:5])
	return status, eventCount, nil
}

// ParseReportServerIDResponse parses a report server ID response (FC17).
func ParseReportServerIDResponse(pdu []byte) ([]byte, error) {
	if len(pdu) < 2 {
		return nil, fmt.Errorf("%w: response too short", ErrInvalidResponse)
	}
	byteCount := int(pdu[1])
	if len(pdu) < 2+byteCount {
		return nil, fmt.Errorf("%w: incomplete response", ErrInvalidResponse)
	}
	data := make([]byte, byteCount)
	copy(data, pdu[2:2+byteCount])
	return data, nil
}

// IsExceptionResponse checks if the PDU is an exception response.
func IsExceptionResponse(pdu []byte) bool {
	return len(pdu) > 0 && (pdu[0]&0x80) != 0
}

// ParseExceptionResponse parses an exception response.
func ParseExceptionResponse(pdu []byte) *ModbusError {
	if len(pdu) < 2 {
		return nil
	}
	return &ModbusError{
		FunctionCode:  FunctionCode(pdu[0] & 0x7F),
		ExceptionCode: ExceptionCode(pdu[1]),
	}
}
