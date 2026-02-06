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
	"net"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	handler := NewMemoryHandler(65536, 65536)
	server := NewServer(handler)

	if server == nil {
		t.Fatal("NewServer returned nil")
	}
}

func TestMemoryHandler_ReadWriteCoils(t *testing.T) {
	handler := NewMemoryHandler(65536, 65536)
	unitID := UnitID(1)

	// Write coil
	if err := handler.WriteSingleCoil(unitID, 10, true); err != nil {
		t.Fatalf("WriteSingleCoil failed: %v", err)
	}

	// Read coil
	coils, err := handler.ReadCoils(unitID, 10, 1)
	if err != nil {
		t.Fatalf("ReadCoils failed: %v", err)
	}
	if !coils[0] {
		t.Error("Coil should be true")
	}
}

func TestMemoryHandler_ReadWriteRegisters(t *testing.T) {
	handler := NewMemoryHandler(65536, 65536)
	unitID := UnitID(1)

	// Write register
	if err := handler.WriteSingleRegister(unitID, 100, 12345); err != nil {
		t.Fatalf("WriteSingleRegister failed: %v", err)
	}

	// Read register
	regs, err := handler.ReadHoldingRegisters(unitID, 100, 1)
	if err != nil {
		t.Fatalf("ReadHoldingRegisters failed: %v", err)
	}
	if regs[0] != 12345 {
		t.Errorf("Register: expected 12345, got %d", regs[0])
	}
}

func TestMemoryHandler_WriteMultipleCoils(t *testing.T) {
	handler := NewMemoryHandler(65536, 65536)
	unitID := UnitID(1)

	values := []bool{true, false, true, true, false}
	if err := handler.WriteMultipleCoils(unitID, 20, values); err != nil {
		t.Fatalf("WriteMultipleCoils failed: %v", err)
	}

	coils, err := handler.ReadCoils(unitID, 20, 5)
	if err != nil {
		t.Fatalf("ReadCoils failed: %v", err)
	}

	for i, v := range values {
		if coils[i] != v {
			t.Errorf("Coil[%d]: expected %v, got %v", i, v, coils[i])
		}
	}
}

func TestMemoryHandler_WriteMultipleRegisters(t *testing.T) {
	handler := NewMemoryHandler(65536, 65536)
	unitID := UnitID(1)

	values := []uint16{1111, 2222, 3333}
	if err := handler.WriteMultipleRegisters(unitID, 200, values); err != nil {
		t.Fatalf("WriteMultipleRegisters failed: %v", err)
	}

	regs, err := handler.ReadHoldingRegisters(unitID, 200, 3)
	if err != nil {
		t.Fatalf("ReadHoldingRegisters failed: %v", err)
	}

	for i, v := range values {
		if regs[i] != v {
			t.Errorf("Register[%d]: expected %d, got %d", i, v, regs[i])
		}
	}
}

func TestMemoryHandler_DiscreteInputs(t *testing.T) {
	handler := NewMemoryHandler(65536, 65536)
	unitID := UnitID(1)

	handler.SetDiscreteInput(unitID, 5, true)
	handler.SetDiscreteInput(unitID, 6, true)

	inputs, err := handler.ReadDiscreteInputs(unitID, 5, 3)
	if err != nil {
		t.Fatalf("ReadDiscreteInputs failed: %v", err)
	}

	if !inputs[0] {
		t.Error("Input[5] should be true")
	}
	if !inputs[1] {
		t.Error("Input[6] should be true")
	}
	if inputs[2] {
		t.Error("Input[7] should be false")
	}
}

func TestMemoryHandler_InputRegisters(t *testing.T) {
	handler := NewMemoryHandler(65536, 65536)
	unitID := UnitID(1)

	handler.SetInputRegister(unitID, 10, 500)
	handler.SetInputRegister(unitID, 11, 600)

	regs, err := handler.ReadInputRegisters(unitID, 10, 2)
	if err != nil {
		t.Fatalf("ReadInputRegisters failed: %v", err)
	}

	if regs[0] != 500 {
		t.Errorf("InputRegister[10]: expected 500, got %d", regs[0])
	}
	if regs[1] != 600 {
		t.Errorf("InputRegister[11]: expected 600, got %d", regs[1])
	}
}

func TestMemoryHandler_Diagnostics(t *testing.T) {
	handler := NewMemoryHandler(65536, 65536)
	unitID := UnitID(1)

	// Test echo
	data := []byte{0xAB, 0xCD}
	resp, err := handler.Diagnostics(unitID, DiagReturnQueryData, data)
	if err != nil {
		t.Fatalf("Diagnostics failed: %v", err)
	}

	if len(resp) != len(data) {
		t.Errorf("Expected %d bytes, got %d", len(data), len(resp))
	}
	for i, b := range data {
		if resp[i] != b {
			t.Errorf("Byte[%d]: expected 0x%02X, got 0x%02X", i, b, resp[i])
		}
	}
}

func TestMemoryHandler_ServerID(t *testing.T) {
	handler := NewMemoryHandler(65536, 65536)
	unitID := UnitID(1)

	handler.SetServerID([]byte("Test Server"))

	id, err := handler.ReportServerID(unitID)
	if err != nil {
		t.Fatalf("ReportServerID failed: %v", err)
	}

	if string(id) != "Test Server" {
		t.Errorf("Server ID: expected 'Test Server', got '%s'", string(id))
	}
}

func TestMemoryHandler_MultipleUnits(t *testing.T) {
	handler := NewMemoryHandler(65536, 65536)

	// Write to unit 1
	if err := handler.WriteSingleRegister(1, 0, 1111); err != nil {
		t.Fatalf("WriteSingleRegister unit 1 failed: %v", err)
	}

	// Write to unit 2
	if err := handler.WriteSingleRegister(2, 0, 2222); err != nil {
		t.Fatalf("WriteSingleRegister unit 2 failed: %v", err)
	}

	// Read from unit 1
	regs1, err := handler.ReadHoldingRegisters(1, 0, 1)
	if err != nil {
		t.Fatalf("ReadHoldingRegisters unit 1 failed: %v", err)
	}
	if regs1[0] != 1111 {
		t.Errorf("Unit 1 register: expected 1111, got %d", regs1[0])
	}

	// Read from unit 2
	regs2, err := handler.ReadHoldingRegisters(2, 0, 1)
	if err != nil {
		t.Fatalf("ReadHoldingRegisters unit 2 failed: %v", err)
	}
	if regs2[0] != 2222 {
		t.Errorf("Unit 2 register: expected 2222, got %d", regs2[0])
	}
}

func TestServerAddr(t *testing.T) {
	handler := NewMemoryHandler(65536, 65536)
	server := NewServer(handler)

	// Before listening, Addr should be nil
	if server.Addr() != nil {
		t.Error("Addr should be nil before listening")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}

	// Save the address before starting serve (since listener is set during Serve)
	expectedAddr := listener.Addr()

	go server.Serve(listener)
	defer server.Close()

	// Give server time to set up
	time.Sleep(10 * time.Millisecond)

	addr := server.Addr()
	if addr == nil {
		t.Error("Addr should not be nil after listening")
	} else if addr.String() != expectedAddr.String() {
		t.Errorf("Addr mismatch: expected %s, got %s", expectedAddr, addr)
	}
}
