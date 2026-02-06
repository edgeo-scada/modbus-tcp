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
	"bytes"
	"testing"
)

func TestMBAPHeader_Encode(t *testing.T) {
	header := MBAPHeader{
		TransactionID: 0x0001,
		ProtocolID:    0x0000,
		Length:        0x0006,
		UnitID:        0x01,
	}

	expected := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x01}
	result := header.Encode()

	if !bytes.Equal(result, expected) {
		t.Errorf("Expected %x, got %x", expected, result)
	}
}

func TestMBAPHeader_Decode(t *testing.T) {
	data := []byte{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x01}

	var header MBAPHeader
	if err := header.Decode(data); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if header.TransactionID != 0x0001 {
		t.Errorf("TransactionID: expected 0x0001, got 0x%04X", header.TransactionID)
	}
	if header.ProtocolID != 0x0000 {
		t.Errorf("ProtocolID: expected 0x0000, got 0x%04X", header.ProtocolID)
	}
	if header.Length != 0x0006 {
		t.Errorf("Length: expected 0x0006, got 0x%04X", header.Length)
	}
	if header.UnitID != 0x01 {
		t.Errorf("UnitID: expected 0x01, got 0x%02X", header.UnitID)
	}
}

func TestMBAPHeader_Decode_TooShort(t *testing.T) {
	data := []byte{0x00, 0x01, 0x00}

	var header MBAPHeader
	err := header.Decode(data)
	if err == nil {
		t.Error("Expected error for short data")
	}
}

func TestFrame_Encode(t *testing.T) {
	frame := Frame{
		Header: MBAPHeader{
			TransactionID: 0x0001,
			ProtocolID:    0x0000,
			UnitID:        0x01,
		},
		PDU: []byte{0x03, 0x00, 0x00, 0x00, 0x0A}, // Read holding registers
	}

	result := frame.Encode()

	// Header should have Length = PDU length + 1 (for UnitID)
	expectedLength := len(frame.PDU) + 1
	actualLength := int(result[4])<<8 | int(result[5])
	if actualLength != expectedLength {
		t.Errorf("Length: expected %d, got %d", expectedLength, actualLength)
	}

	// Check PDU is appended correctly
	if !bytes.Equal(result[7:], frame.PDU) {
		t.Errorf("PDU mismatch: expected %x, got %x", frame.PDU, result[7:])
	}
}

func TestFrame_Decode(t *testing.T) {
	data := []byte{
		0x00, 0x01, // Transaction ID
		0x00, 0x00, // Protocol ID
		0x00, 0x06, // Length
		0x01,                               // Unit ID
		0x03, 0x00, 0x00, 0x00, 0x0A, // PDU
	}

	var frame Frame
	if err := frame.Decode(data); err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if frame.Header.TransactionID != 0x0001 {
		t.Errorf("TransactionID: expected 0x0001, got 0x%04X", frame.Header.TransactionID)
	}
	expectedPDU := []byte{0x03, 0x00, 0x00, 0x00, 0x0A}
	if !bytes.Equal(frame.PDU, expectedPDU) {
		t.Errorf("PDU: expected %x, got %x", expectedPDU, frame.PDU)
	}
}

func TestBuildReadCoilsPDU(t *testing.T) {
	pdu, err := BuildReadCoilsPDU(0x0013, 0x0025)
	if err != nil {
		t.Fatalf("BuildReadCoilsPDU failed: %v", err)
	}

	expected := []byte{0x01, 0x00, 0x13, 0x00, 0x25}
	if !bytes.Equal(pdu, expected) {
		t.Errorf("Expected %x, got %x", expected, pdu)
	}
}

func TestBuildReadCoilsPDU_InvalidQuantity(t *testing.T) {
	_, err := BuildReadCoilsPDU(0, 0)
	if err == nil {
		t.Error("Expected error for quantity 0")
	}

	_, err = BuildReadCoilsPDU(0, MaxQuantityCoils+1)
	if err == nil {
		t.Error("Expected error for quantity > max")
	}
}

func TestBuildReadHoldingRegistersPDU(t *testing.T) {
	pdu, err := BuildReadHoldingRegistersPDU(0x006B, 0x0003)
	if err != nil {
		t.Fatalf("BuildReadHoldingRegistersPDU failed: %v", err)
	}

	expected := []byte{0x03, 0x00, 0x6B, 0x00, 0x03}
	if !bytes.Equal(pdu, expected) {
		t.Errorf("Expected %x, got %x", expected, pdu)
	}
}

func TestBuildWriteSingleCoilPDU(t *testing.T) {
	// Test ON
	pduOn := BuildWriteSingleCoilPDU(0x00AC, true)
	expectedOn := []byte{0x05, 0x00, 0xAC, 0xFF, 0x00}
	if !bytes.Equal(pduOn, expectedOn) {
		t.Errorf("ON: expected %x, got %x", expectedOn, pduOn)
	}

	// Test OFF
	pduOff := BuildWriteSingleCoilPDU(0x00AC, false)
	expectedOff := []byte{0x05, 0x00, 0xAC, 0x00, 0x00}
	if !bytes.Equal(pduOff, expectedOff) {
		t.Errorf("OFF: expected %x, got %x", expectedOff, pduOff)
	}
}

func TestBuildWriteSingleRegisterPDU(t *testing.T) {
	pdu := BuildWriteSingleRegisterPDU(0x0001, 0x0003)
	expected := []byte{0x06, 0x00, 0x01, 0x00, 0x03}
	if !bytes.Equal(pdu, expected) {
		t.Errorf("Expected %x, got %x", expected, pdu)
	}
}

func TestBuildWriteMultipleCoilsPDU(t *testing.T) {
	values := []bool{true, false, true, true, false, false, true, true, true, false}
	pdu, err := BuildWriteMultipleCoilsPDU(0x0013, values)
	if err != nil {
		t.Fatalf("BuildWriteMultipleCoilsPDU failed: %v", err)
	}

	expected := []byte{0x0F, 0x00, 0x13, 0x00, 0x0A, 0x02, 0xCD, 0x01}
	if !bytes.Equal(pdu, expected) {
		t.Errorf("Expected %x, got %x", expected, pdu)
	}
}

func TestBuildWriteMultipleRegistersPDU(t *testing.T) {
	values := []uint16{0x000A, 0x0102}
	pdu, err := BuildWriteMultipleRegistersPDU(0x0001, values)
	if err != nil {
		t.Fatalf("BuildWriteMultipleRegistersPDU failed: %v", err)
	}

	expected := []byte{0x10, 0x00, 0x01, 0x00, 0x02, 0x04, 0x00, 0x0A, 0x01, 0x02}
	if !bytes.Equal(pdu, expected) {
		t.Errorf("Expected %x, got %x", expected, pdu)
	}
}

func TestParseCoilsResponse(t *testing.T) {
	// Response for reading 19 coils
	pdu := []byte{0x01, 0x03, 0xCD, 0x6B, 0x05}
	values, err := ParseCoilsResponse(pdu, 19)
	if err != nil {
		t.Fatalf("ParseCoilsResponse failed: %v", err)
	}

	if len(values) != 19 {
		t.Errorf("Expected 19 values, got %d", len(values))
	}

	// Check first byte: 0xCD = 11001101
	expectedFirst := []bool{true, false, true, true, false, false, true, true}
	for i, v := range expectedFirst {
		if values[i] != v {
			t.Errorf("values[%d]: expected %v, got %v", i, v, values[i])
		}
	}
}

func TestParseRegistersResponse(t *testing.T) {
	pdu := []byte{0x03, 0x06, 0x00, 0x6B, 0x00, 0x02, 0x00, 0x64}
	values, err := ParseRegistersResponse(pdu, 3)
	if err != nil {
		t.Fatalf("ParseRegistersResponse failed: %v", err)
	}

	expected := []uint16{0x006B, 0x0002, 0x0064}
	for i, v := range expected {
		if values[i] != v {
			t.Errorf("values[%d]: expected 0x%04X, got 0x%04X", i, v, values[i])
		}
	}
}

func TestIsExceptionResponse(t *testing.T) {
	// Normal response
	normalPDU := []byte{0x03, 0x02, 0x00, 0x01}
	if IsExceptionResponse(normalPDU) {
		t.Error("Normal response should not be exception")
	}

	// Exception response (FC 0x83 = 0x03 | 0x80)
	exceptionPDU := []byte{0x83, 0x02}
	if !IsExceptionResponse(exceptionPDU) {
		t.Error("Exception response should be detected")
	}
}

func TestParseExceptionResponse(t *testing.T) {
	pdu := []byte{0x83, 0x02}
	err := ParseExceptionResponse(pdu)

	if err == nil {
		t.Fatal("Expected error")
	}
	if err.FunctionCode != FuncReadHoldingRegisters {
		t.Errorf("FunctionCode: expected %d, got %d", FuncReadHoldingRegisters, err.FunctionCode)
	}
	if err.ExceptionCode != ExceptionIllegalDataAddress {
		t.Errorf("ExceptionCode: expected %d, got %d", ExceptionIllegalDataAddress, err.ExceptionCode)
	}
}

func TestTransactionIDGenerator(t *testing.T) {
	var gen TransactionIDGenerator

	id1 := gen.Next()
	id2 := gen.Next()
	id3 := gen.Next()

	if id1 != 1 {
		t.Errorf("First ID should be 1, got %d", id1)
	}
	if id2 != 2 {
		t.Errorf("Second ID should be 2, got %d", id2)
	}
	if id3 != 3 {
		t.Errorf("Third ID should be 3, got %d", id3)
	}
}

func TestReadFrame(t *testing.T) {
	data := []byte{
		0x00, 0x01, // Transaction ID
		0x00, 0x00, // Protocol ID
		0x00, 0x05, // Length
		0x01,                         // Unit ID
		0x03, 0x02, 0x00, 0x0A, // PDU
	}

	r := bytes.NewReader(data)
	frame, err := ReadFrame(r)
	if err != nil {
		t.Fatalf("ReadFrame failed: %v", err)
	}

	if frame.Header.TransactionID != 0x0001 {
		t.Errorf("TransactionID: expected 0x0001, got 0x%04X", frame.Header.TransactionID)
	}
	if frame.Header.UnitID != 0x01 {
		t.Errorf("UnitID: expected 0x01, got 0x%02X", frame.Header.UnitID)
	}

	expectedPDU := []byte{0x03, 0x02, 0x00, 0x0A}
	if !bytes.Equal(frame.PDU, expectedPDU) {
		t.Errorf("PDU: expected %x, got %x", expectedPDU, frame.PDU)
	}
}
