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
	"errors"
	"fmt"
)

// ExceptionCode represents a Modbus exception code.
type ExceptionCode uint8

// Modbus exception codes.
const (
	ExceptionIllegalFunction                    ExceptionCode = 0x01
	ExceptionIllegalDataAddress                 ExceptionCode = 0x02
	ExceptionIllegalDataValue                   ExceptionCode = 0x03
	ExceptionServerDeviceFailure                ExceptionCode = 0x04
	ExceptionAcknowledge                        ExceptionCode = 0x05
	ExceptionServerDeviceBusy                   ExceptionCode = 0x06
	ExceptionMemoryParityError                  ExceptionCode = 0x08
	ExceptionGatewayPathUnavailable             ExceptionCode = 0x0A
	ExceptionGatewayTargetDeviceFailedToRespond ExceptionCode = 0x0B
)

// String returns the string representation of the exception code.
func (e ExceptionCode) String() string {
	switch e {
	case ExceptionIllegalFunction:
		return "illegal function"
	case ExceptionIllegalDataAddress:
		return "illegal data address"
	case ExceptionIllegalDataValue:
		return "illegal data value"
	case ExceptionServerDeviceFailure:
		return "server device failure"
	case ExceptionAcknowledge:
		return "acknowledge"
	case ExceptionServerDeviceBusy:
		return "server device busy"
	case ExceptionMemoryParityError:
		return "memory parity error"
	case ExceptionGatewayPathUnavailable:
		return "gateway path unavailable"
	case ExceptionGatewayTargetDeviceFailedToRespond:
		return "gateway target device failed to respond"
	default:
		return fmt.Sprintf("unknown exception (0x%02X)", uint8(e))
	}
}

// ModbusError represents a Modbus protocol error (exception response).
type ModbusError struct {
	FunctionCode  FunctionCode
	ExceptionCode ExceptionCode
}

// Error implements the error interface.
func (e *ModbusError) Error() string {
	return fmt.Sprintf("modbus: exception %s (FC=%02X)", e.ExceptionCode, e.FunctionCode)
}

// Is checks if the error matches the target.
func (e *ModbusError) Is(target error) bool {
	t, ok := target.(*ModbusError)
	if !ok {
		return false
	}
	return e.ExceptionCode == t.ExceptionCode
}

// Common errors.
var (
	// ErrInvalidResponse indicates the response was malformed or unexpected.
	ErrInvalidResponse = errors.New("modbus: invalid response")

	// ErrInvalidCRC indicates a CRC validation failure (RTU mode).
	ErrInvalidCRC = errors.New("modbus: invalid CRC")

	// ErrInvalidFrame indicates a malformed frame.
	ErrInvalidFrame = errors.New("modbus: invalid frame")

	// ErrTimeout indicates a timeout occurred.
	ErrTimeout = errors.New("modbus: timeout")

	// ErrConnectionClosed indicates the connection was closed.
	ErrConnectionClosed = errors.New("modbus: connection closed")

	// ErrInvalidQuantity indicates an invalid quantity was specified.
	ErrInvalidQuantity = errors.New("modbus: invalid quantity")

	// ErrInvalidAddress indicates an invalid address was specified.
	ErrInvalidAddress = errors.New("modbus: invalid address")

	// ErrPoolExhausted indicates no connections are available in the pool.
	ErrPoolExhausted = errors.New("modbus: connection pool exhausted")

	// ErrPoolClosed indicates the pool has been closed.
	ErrPoolClosed = errors.New("modbus: connection pool closed")

	// ErrNotConnected indicates the client is not connected.
	ErrNotConnected = errors.New("modbus: not connected")

	// ErrMaxRetriesExceeded indicates the maximum number of retries was exceeded.
	ErrMaxRetriesExceeded = errors.New("modbus: max retries exceeded")
)

// NewModbusError creates a new Modbus exception error.
func NewModbusError(fc FunctionCode, ec ExceptionCode) *ModbusError {
	return &ModbusError{
		FunctionCode:  fc,
		ExceptionCode: ec,
	}
}

// IsException checks if an error is a specific Modbus exception.
func IsException(err error, code ExceptionCode) bool {
	var modbusErr *ModbusError
	if errors.As(err, &modbusErr) {
		return modbusErr.ExceptionCode == code
	}
	return false
}

// IsIllegalFunction checks if the error is an illegal function exception.
func IsIllegalFunction(err error) bool {
	return IsException(err, ExceptionIllegalFunction)
}

// IsIllegalDataAddress checks if the error is an illegal data address exception.
func IsIllegalDataAddress(err error) bool {
	return IsException(err, ExceptionIllegalDataAddress)
}

// IsIllegalDataValue checks if the error is an illegal data value exception.
func IsIllegalDataValue(err error) bool {
	return IsException(err, ExceptionIllegalDataValue)
}

// IsServerDeviceFailure checks if the error is a server device failure exception.
func IsServerDeviceFailure(err error) bool {
	return IsException(err, ExceptionServerDeviceFailure)
}
