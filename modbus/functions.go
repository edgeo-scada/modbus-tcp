package modbus

import (
	"encoding/binary"
	"fmt"
	"math"
)

// ReadCoilsRequest represents a request to read coils (FC01).
type ReadCoilsRequest struct {
	Address  uint16
	Quantity uint16
}

func (r *ReadCoilsRequest) FunctionCode() FunctionCode {
	return FuncReadCoils
}

func (r *ReadCoilsRequest) Encode() ([]byte, error) {
	return BuildReadCoilsPDU(r.Address, r.Quantity)
}

// ReadCoilsResponse represents a response to read coils.
type ReadCoilsResponse struct {
	Values []bool
}

func (r *ReadCoilsResponse) FunctionCode() FunctionCode {
	return FuncReadCoils
}

func (r *ReadCoilsResponse) Decode(data []byte) error {
	if len(data) < 2 {
		return fmt.Errorf("%w: response too short", ErrInvalidResponse)
	}
	byteCount := int(data[1])
	if len(data) < 2+byteCount {
		return fmt.Errorf("%w: incomplete response", ErrInvalidResponse)
	}
	// Calculate quantity from byte count (max possible)
	qty := byteCount * 8
	r.Values = make([]bool, qty)
	for i := 0; i < qty; i++ {
		r.Values[i] = (data[2+i/8] & (1 << (i % 8))) != 0
	}
	return nil
}

// ReadDiscreteInputsRequest represents a request to read discrete inputs (FC02).
type ReadDiscreteInputsRequest struct {
	Address  uint16
	Quantity uint16
}

func (r *ReadDiscreteInputsRequest) FunctionCode() FunctionCode {
	return FuncReadDiscreteInputs
}

func (r *ReadDiscreteInputsRequest) Encode() ([]byte, error) {
	return BuildReadDiscreteInputsPDU(r.Address, r.Quantity)
}

// ReadHoldingRegistersRequest represents a request to read holding registers (FC03).
type ReadHoldingRegistersRequest struct {
	Address  uint16
	Quantity uint16
}

func (r *ReadHoldingRegistersRequest) FunctionCode() FunctionCode {
	return FuncReadHoldingRegisters
}

func (r *ReadHoldingRegistersRequest) Encode() ([]byte, error) {
	return BuildReadHoldingRegistersPDU(r.Address, r.Quantity)
}

// ReadHoldingRegistersResponse represents a response to read holding registers.
type ReadHoldingRegistersResponse struct {
	Values []uint16
}

func (r *ReadHoldingRegistersResponse) FunctionCode() FunctionCode {
	return FuncReadHoldingRegisters
}

func (r *ReadHoldingRegistersResponse) Decode(data []byte) error {
	if len(data) < 2 {
		return fmt.Errorf("%w: response too short", ErrInvalidResponse)
	}
	byteCount := int(data[1])
	if byteCount%2 != 0 || len(data) < 2+byteCount {
		return fmt.Errorf("%w: invalid byte count", ErrInvalidResponse)
	}
	qty := byteCount / 2
	r.Values = make([]uint16, qty)
	for i := 0; i < qty; i++ {
		r.Values[i] = binary.BigEndian.Uint16(data[2+i*2:])
	}
	return nil
}

// ReadInputRegistersRequest represents a request to read input registers (FC04).
type ReadInputRegistersRequest struct {
	Address  uint16
	Quantity uint16
}

func (r *ReadInputRegistersRequest) FunctionCode() FunctionCode {
	return FuncReadInputRegisters
}

func (r *ReadInputRegistersRequest) Encode() ([]byte, error) {
	return BuildReadInputRegistersPDU(r.Address, r.Quantity)
}

// WriteSingleCoilRequest represents a request to write a single coil (FC05).
type WriteSingleCoilRequest struct {
	Address uint16
	Value   bool
}

func (r *WriteSingleCoilRequest) FunctionCode() FunctionCode {
	return FuncWriteSingleCoil
}

func (r *WriteSingleCoilRequest) Encode() ([]byte, error) {
	return BuildWriteSingleCoilPDU(r.Address, r.Value), nil
}

// WriteSingleRegisterRequest represents a request to write a single register (FC06).
type WriteSingleRegisterRequest struct {
	Address uint16
	Value   uint16
}

func (r *WriteSingleRegisterRequest) FunctionCode() FunctionCode {
	return FuncWriteSingleRegister
}

func (r *WriteSingleRegisterRequest) Encode() ([]byte, error) {
	return BuildWriteSingleRegisterPDU(r.Address, r.Value), nil
}

// ReadExceptionStatusRequest represents a request to read exception status (FC07).
type ReadExceptionStatusRequest struct{}

func (r *ReadExceptionStatusRequest) FunctionCode() FunctionCode {
	return FuncReadExceptionStatus
}

func (r *ReadExceptionStatusRequest) Encode() ([]byte, error) {
	return BuildReadExceptionStatusPDU(), nil
}

// ReadExceptionStatusResponse represents a response to read exception status.
type ReadExceptionStatusResponse struct {
	Status uint8
}

func (r *ReadExceptionStatusResponse) FunctionCode() FunctionCode {
	return FuncReadExceptionStatus
}

func (r *ReadExceptionStatusResponse) Decode(data []byte) error {
	status, err := ParseExceptionStatusResponse(data)
	if err != nil {
		return err
	}
	r.Status = status
	return nil
}

// DiagnosticsRequest represents a diagnostics request (FC08).
type DiagnosticsRequest struct {
	SubFunction uint16
	Data        []byte
}

func (r *DiagnosticsRequest) FunctionCode() FunctionCode {
	return FuncDiagnostics
}

func (r *DiagnosticsRequest) Encode() ([]byte, error) {
	return BuildDiagnosticsPDU(r.SubFunction, r.Data), nil
}

// DiagnosticsResponse represents a diagnostics response.
type DiagnosticsResponse struct {
	SubFunction uint16
	Data        []byte
}

func (r *DiagnosticsResponse) FunctionCode() FunctionCode {
	return FuncDiagnostics
}

func (r *DiagnosticsResponse) Decode(data []byte) error {
	subFunc, respData, err := ParseDiagnosticsResponse(data)
	if err != nil {
		return err
	}
	r.SubFunction = subFunc
	r.Data = respData
	return nil
}

// GetCommEventCounterRequest represents a get comm event counter request (FC11).
type GetCommEventCounterRequest struct{}

func (r *GetCommEventCounterRequest) FunctionCode() FunctionCode {
	return FuncGetCommEventCounter
}

func (r *GetCommEventCounterRequest) Encode() ([]byte, error) {
	return BuildGetCommEventCounterPDU(), nil
}

// GetCommEventCounterResponse represents a get comm event counter response.
type GetCommEventCounterResponse struct {
	Status     uint16
	EventCount uint16
}

func (r *GetCommEventCounterResponse) FunctionCode() FunctionCode {
	return FuncGetCommEventCounter
}

func (r *GetCommEventCounterResponse) Decode(data []byte) error {
	status, eventCount, err := ParseGetCommEventCounterResponse(data)
	if err != nil {
		return err
	}
	r.Status = status
	r.EventCount = eventCount
	return nil
}

// WriteMultipleCoilsRequest represents a request to write multiple coils (FC15).
type WriteMultipleCoilsRequest struct {
	Address uint16
	Values  []bool
}

func (r *WriteMultipleCoilsRequest) FunctionCode() FunctionCode {
	return FuncWriteMultipleCoils
}

func (r *WriteMultipleCoilsRequest) Encode() ([]byte, error) {
	return BuildWriteMultipleCoilsPDU(r.Address, r.Values)
}

// WriteMultipleRegistersRequest represents a request to write multiple registers (FC16).
type WriteMultipleRegistersRequest struct {
	Address uint16
	Values  []uint16
}

func (r *WriteMultipleRegistersRequest) FunctionCode() FunctionCode {
	return FuncWriteMultipleRegisters
}

func (r *WriteMultipleRegistersRequest) Encode() ([]byte, error) {
	return BuildWriteMultipleRegistersPDU(r.Address, r.Values)
}

// ReportServerIDRequest represents a report server ID request (FC17).
type ReportServerIDRequest struct{}

func (r *ReportServerIDRequest) FunctionCode() FunctionCode {
	return FuncReportServerID
}

func (r *ReportServerIDRequest) Encode() ([]byte, error) {
	return BuildReportServerIDPDU(), nil
}

// ReportServerIDResponse represents a report server ID response.
type ReportServerIDResponse struct {
	Data []byte
}

func (r *ReportServerIDResponse) FunctionCode() FunctionCode {
	return FuncReportServerID
}

func (r *ReportServerIDResponse) Decode(data []byte) error {
	respData, err := ParseReportServerIDResponse(data)
	if err != nil {
		return err
	}
	r.Data = respData
	return nil
}

// Helper functions for data conversion

// BoolsToBytes converts a slice of bools to a byte slice (packed).
func BoolsToBytes(values []bool) []byte {
	byteCount := (len(values) + 7) / 8
	result := make([]byte, byteCount)
	for i, v := range values {
		if v {
			result[i/8] |= 1 << (i % 8)
		}
	}
	return result
}

// BytesToBools converts a byte slice to a slice of bools (unpacked).
func BytesToBools(data []byte, count int) []bool {
	result := make([]bool, count)
	for i := 0; i < count; i++ {
		result[i] = (data[i/8] & (1 << (i % 8))) != 0
	}
	return result
}

// Uint16sToBytes converts a slice of uint16 to a byte slice (big endian).
func Uint16sToBytes(values []uint16) []byte {
	result := make([]byte, len(values)*2)
	for i, v := range values {
		binary.BigEndian.PutUint16(result[i*2:], v)
	}
	return result
}

// BytesToUint16s converts a byte slice to a slice of uint16 (big endian).
func BytesToUint16s(data []byte) []uint16 {
	count := len(data) / 2
	result := make([]uint16, count)
	for i := 0; i < count; i++ {
		result[i] = binary.BigEndian.Uint16(data[i*2:])
	}
	return result
}

// Float32ToRegisters converts a float32 to two uint16 registers (big endian).
func Float32ToRegisters(f float32) [2]uint16 {
	bits := math.Float32bits(f)
	return [2]uint16{
		uint16(bits >> 16),
		uint16(bits & 0xFFFF),
	}
}

// RegistersToFloat32 converts two uint16 registers to a float32 (big endian).
func RegistersToFloat32(regs [2]uint16) float32 {
	bits := uint32(regs[0])<<16 | uint32(regs[1])
	return math.Float32frombits(bits)
}

// Int32ToRegisters converts an int32 to two uint16 registers (big endian).
func Int32ToRegisters(i int32) [2]uint16 {
	return [2]uint16{
		uint16(i >> 16),
		uint16(i & 0xFFFF),
	}
}

// RegistersToInt32 converts two uint16 registers to an int32 (big endian).
func RegistersToInt32(regs [2]uint16) int32 {
	return int32(regs[0])<<16 | int32(regs[1])
}

// Uint32ToRegisters converts a uint32 to two uint16 registers (big endian).
func Uint32ToRegisters(u uint32) [2]uint16 {
	return [2]uint16{
		uint16(u >> 16),
		uint16(u & 0xFFFF),
	}
}

// RegistersToUint32 converts two uint16 registers to a uint32 (big endian).
func RegistersToUint32(regs [2]uint16) uint32 {
	return uint32(regs[0])<<16 | uint32(regs[1])
}
