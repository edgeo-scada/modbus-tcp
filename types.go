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

// Package modbus provides a Modbus TCP client and server implementation.
package modbus

import (
	"context"
	"time"
)

// UnitID represents the Modbus unit identifier (slave address).
type UnitID uint8

// FunctionCode represents a Modbus function code.
type FunctionCode uint8

// Standard Modbus function codes.
const (
	FuncReadCoils              FunctionCode = 0x01
	FuncReadDiscreteInputs     FunctionCode = 0x02
	FuncReadHoldingRegisters   FunctionCode = 0x03
	FuncReadInputRegisters     FunctionCode = 0x04
	FuncWriteSingleCoil        FunctionCode = 0x05
	FuncWriteSingleRegister    FunctionCode = 0x06
	FuncReadExceptionStatus    FunctionCode = 0x07
	FuncDiagnostics            FunctionCode = 0x08
	FuncGetCommEventCounter    FunctionCode = 0x0B
	FuncWriteMultipleCoils     FunctionCode = 0x0F
	FuncWriteMultipleRegisters FunctionCode = 0x10
	FuncReportServerID         FunctionCode = 0x11
)

// Diagnostic sub-function codes (FC08).
const (
	DiagReturnQueryData                     uint16 = 0x00
	DiagRestartCommunications               uint16 = 0x01
	DiagReturnDiagnosticRegister            uint16 = 0x02
	DiagChangeASCIIInputDelimiter           uint16 = 0x03
	DiagForceListenOnlyMode                 uint16 = 0x04
	DiagClearCountersAndDiagnosticRegister  uint16 = 0x0A
	DiagReturnBusMessageCount               uint16 = 0x0B
	DiagReturnBusCommunicationErrorCount    uint16 = 0x0C
	DiagReturnBusExceptionErrorCount        uint16 = 0x0D
	DiagReturnServerMessageCount            uint16 = 0x0E
	DiagReturnServerNoResponseCount         uint16 = 0x0F
	DiagReturnServerNAKCount                uint16 = 0x10
	DiagReturnServerBusyCount               uint16 = 0x11
	DiagReturnBusCharacterOverrunCount      uint16 = 0x12
	DiagClearOverrunCounterAndFlag          uint16 = 0x14
)

// Protocol constants.
const (
	// MaxQuantityCoils is the maximum number of coils that can be read/written.
	MaxQuantityCoils = 2000

	// MaxQuantityDiscreteInputs is the maximum number of discrete inputs that can be read.
	MaxQuantityDiscreteInputs = 2000

	// MaxQuantityRegisters is the maximum number of registers that can be read.
	MaxQuantityRegisters = 125

	// MaxQuantityWriteRegisters is the maximum number of registers that can be written.
	MaxQuantityWriteRegisters = 123

	// MBAPHeaderSize is the size of the MBAP header in bytes.
	MBAPHeaderSize = 7

	// ProtocolID is the Modbus protocol identifier (always 0 for Modbus TCP).
	ProtocolID = 0

	// DefaultTimeout is the default timeout for Modbus operations.
	DefaultTimeout = 5 * time.Second

	// DefaultPort is the default Modbus TCP port.
	DefaultPort = 502
)

// Coil values for write operations.
const (
	CoilOn  uint16 = 0xFF00
	CoilOff uint16 = 0x0000
)

// Request represents a Modbus request that can be encoded.
type Request interface {
	FunctionCode() FunctionCode
	Encode() ([]byte, error)
}

// Response represents a Modbus response that can be decoded.
type Response interface {
	FunctionCode() FunctionCode
	Decode(data []byte) error
}

// Transporter defines the interface for sending and receiving Modbus frames.
type Transporter interface {
	Send(ctx context.Context, unitID UnitID, pdu []byte) ([]byte, error)
	Close() error
}

// Handler defines the interface for handling Modbus requests on the server side.
type Handler interface {
	// Coil operations
	ReadCoils(unitID UnitID, addr, qty uint16) ([]bool, error)
	ReadDiscreteInputs(unitID UnitID, addr, qty uint16) ([]bool, error)
	WriteSingleCoil(unitID UnitID, addr uint16, value bool) error
	WriteMultipleCoils(unitID UnitID, addr uint16, values []bool) error

	// Register operations
	ReadHoldingRegisters(unitID UnitID, addr, qty uint16) ([]uint16, error)
	ReadInputRegisters(unitID UnitID, addr, qty uint16) ([]uint16, error)
	WriteSingleRegister(unitID UnitID, addr, value uint16) error
	WriteMultipleRegisters(unitID UnitID, addr uint16, values []uint16) error

	// Diagnostic operations
	ReadExceptionStatus(unitID UnitID) (uint8, error)
	Diagnostics(unitID UnitID, subFunc uint16, data []byte) ([]byte, error)
	GetCommEventCounter(unitID UnitID) (status uint16, eventCount uint16, err error)
	ReportServerID(unitID UnitID) ([]byte, error)
}

// ConnectionState represents the state of a client connection.
type ConnectionState int

const (
	StateDisconnected ConnectionState = iota
	StateConnecting
	StateConnected
)

// String returns the string representation of the connection state.
func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "disconnected"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	default:
		return "unknown"
	}
}
