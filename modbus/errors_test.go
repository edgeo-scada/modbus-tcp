package modbus

import (
	"errors"
	"testing"
)

func TestExceptionCode_String(t *testing.T) {
	tests := []struct {
		code     ExceptionCode
		expected string
	}{
		{ExceptionIllegalFunction, "illegal function"},
		{ExceptionIllegalDataAddress, "illegal data address"},
		{ExceptionIllegalDataValue, "illegal data value"},
		{ExceptionServerDeviceFailure, "server device failure"},
		{ExceptionAcknowledge, "acknowledge"},
		{ExceptionServerDeviceBusy, "server device busy"},
		{ExceptionMemoryParityError, "memory parity error"},
		{ExceptionGatewayPathUnavailable, "gateway path unavailable"},
		{ExceptionGatewayTargetDeviceFailedToRespond, "gateway target device failed to respond"},
		{ExceptionCode(0xFF), "unknown exception (0xFF)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.code.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, tt.code.String())
			}
		})
	}
}

func TestModbusError(t *testing.T) {
	err := NewModbusError(FuncReadHoldingRegisters, ExceptionIllegalDataAddress)

	if err.FunctionCode != FuncReadHoldingRegisters {
		t.Errorf("FunctionCode: expected %d, got %d", FuncReadHoldingRegisters, err.FunctionCode)
	}
	if err.ExceptionCode != ExceptionIllegalDataAddress {
		t.Errorf("ExceptionCode: expected %d, got %d", ExceptionIllegalDataAddress, err.ExceptionCode)
	}

	// Test Error() string
	errStr := err.Error()
	if errStr == "" {
		t.Error("Error string should not be empty")
	}
}

func TestIsException(t *testing.T) {
	err := NewModbusError(FuncReadCoils, ExceptionIllegalFunction)

	if !IsException(err, ExceptionIllegalFunction) {
		t.Error("IsException should return true for matching exception")
	}
	if IsException(err, ExceptionIllegalDataAddress) {
		t.Error("IsException should return false for non-matching exception")
	}
	if IsException(errors.New("other error"), ExceptionIllegalFunction) {
		t.Error("IsException should return false for non-Modbus error")
	}
}

func TestIsIllegalFunction(t *testing.T) {
	err := NewModbusError(FuncReadCoils, ExceptionIllegalFunction)
	if !IsIllegalFunction(err) {
		t.Error("IsIllegalFunction should return true")
	}

	err2 := NewModbusError(FuncReadCoils, ExceptionIllegalDataAddress)
	if IsIllegalFunction(err2) {
		t.Error("IsIllegalFunction should return false for other exception")
	}
}

func TestIsIllegalDataAddress(t *testing.T) {
	err := NewModbusError(FuncReadHoldingRegisters, ExceptionIllegalDataAddress)
	if !IsIllegalDataAddress(err) {
		t.Error("IsIllegalDataAddress should return true")
	}
}

func TestIsIllegalDataValue(t *testing.T) {
	err := NewModbusError(FuncWriteSingleRegister, ExceptionIllegalDataValue)
	if !IsIllegalDataValue(err) {
		t.Error("IsIllegalDataValue should return true")
	}
}

func TestIsServerDeviceFailure(t *testing.T) {
	err := NewModbusError(FuncReadCoils, ExceptionServerDeviceFailure)
	if !IsServerDeviceFailure(err) {
		t.Error("IsServerDeviceFailure should return true")
	}
}

func TestModbusError_Is(t *testing.T) {
	err1 := NewModbusError(FuncReadCoils, ExceptionIllegalFunction)
	err2 := NewModbusError(FuncWriteSingleCoil, ExceptionIllegalFunction)
	err3 := NewModbusError(FuncReadCoils, ExceptionIllegalDataAddress)

	// Same exception code, different function code
	if !errors.Is(err1, err2) {
		t.Error("Errors with same exception code should match")
	}

	// Different exception code
	if errors.Is(err1, err3) {
		t.Error("Errors with different exception codes should not match")
	}
}
