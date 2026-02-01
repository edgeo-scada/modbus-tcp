package modbus

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client, err := NewClient("localhost:502")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	if client.State() != StateDisconnected {
		t.Errorf("Initial state should be Disconnected, got %v", client.State())
	}
}

func TestClientWithOptions(t *testing.T) {
	client, err := NewClient("localhost:502",
		WithUnitID(5),
		WithTimeout(10*time.Second),
		WithAutoReconnect(true),
		WithMaxRetries(5),
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	if client.unitID != 5 {
		t.Errorf("UnitID: expected 5, got %d", client.unitID)
	}
	if client.opts.timeout != 10*time.Second {
		t.Errorf("Timeout: expected 10s, got %v", client.opts.timeout)
	}
	if !client.opts.autoReconnect {
		t.Error("AutoReconnect should be true")
	}
	if client.opts.maxRetries != 5 {
		t.Errorf("MaxRetries: expected 5, got %d", client.opts.maxRetries)
	}
}

func TestClientSetUnitID(t *testing.T) {
	client, err := NewClient("localhost:502")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	client.SetUnitID(10)
	if client.unitID != 10 {
		t.Errorf("UnitID: expected 10, got %d", client.unitID)
	}
}

func TestClientConnectNotRunning(t *testing.T) {
	client, err := NewClient("localhost:59999") // Non-existent server
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err == nil {
		t.Error("Expected connection error")
	}
}

func TestClientMetrics(t *testing.T) {
	client, err := NewClient("localhost:502")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	metrics := client.Metrics()
	if metrics == nil {
		t.Error("Metrics should not be nil")
	}

	collected := metrics.Collect()
	if collected["requests_total"] != int64(0) {
		t.Errorf("Initial requests_total should be 0, got %v", collected["requests_total"])
	}
}

// Integration test - requires running server
func TestClientIntegration(t *testing.T) {
	// Start a test server
	handler := NewMemoryHandler(65536, 65536)
	handler.SetHoldingRegister(1, 0, 1234)
	handler.SetHoldingRegister(1, 1, 5678)
	handler.SetCoil(1, 0, true)

	server := NewServer(handler)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}

	go server.Serve(listener)
	defer server.Close()

	addr := listener.Addr().String()

	// Create client
	client, err := NewClient(addr, WithUnitID(1))
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Connect
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Test ReadHoldingRegisters
	t.Run("ReadHoldingRegisters", func(t *testing.T) {
		regs, err := client.ReadHoldingRegisters(ctx, 0, 2)
		if err != nil {
			t.Fatalf("ReadHoldingRegisters failed: %v", err)
		}
		if len(regs) != 2 {
			t.Errorf("Expected 2 registers, got %d", len(regs))
		}
		if regs[0] != 1234 {
			t.Errorf("Register[0]: expected 1234, got %d", regs[0])
		}
		if regs[1] != 5678 {
			t.Errorf("Register[1]: expected 5678, got %d", regs[1])
		}
	})

	// Test ReadCoils
	t.Run("ReadCoils", func(t *testing.T) {
		coils, err := client.ReadCoils(ctx, 0, 8)
		if err != nil {
			t.Fatalf("ReadCoils failed: %v", err)
		}
		if len(coils) != 8 {
			t.Errorf("Expected 8 coils, got %d", len(coils))
		}
		if !coils[0] {
			t.Error("Coil[0] should be true")
		}
	})

	// Test WriteSingleRegister
	t.Run("WriteSingleRegister", func(t *testing.T) {
		if err := client.WriteSingleRegister(ctx, 10, 9999); err != nil {
			t.Fatalf("WriteSingleRegister failed: %v", err)
		}

		regs, err := client.ReadHoldingRegisters(ctx, 10, 1)
		if err != nil {
			t.Fatalf("ReadHoldingRegisters failed: %v", err)
		}
		if regs[0] != 9999 {
			t.Errorf("Register[10]: expected 9999, got %d", regs[0])
		}
	})

	// Test WriteSingleCoil
	t.Run("WriteSingleCoil", func(t *testing.T) {
		if err := client.WriteSingleCoil(ctx, 5, true); err != nil {
			t.Fatalf("WriteSingleCoil failed: %v", err)
		}

		coils, err := client.ReadCoils(ctx, 5, 1)
		if err != nil {
			t.Fatalf("ReadCoils failed: %v", err)
		}
		if !coils[0] {
			t.Error("Coil[5] should be true")
		}
	})

	// Test WriteMultipleRegisters
	t.Run("WriteMultipleRegisters", func(t *testing.T) {
		values := []uint16{111, 222, 333}
		if err := client.WriteMultipleRegisters(ctx, 100, values); err != nil {
			t.Fatalf("WriteMultipleRegisters failed: %v", err)
		}

		regs, err := client.ReadHoldingRegisters(ctx, 100, 3)
		if err != nil {
			t.Fatalf("ReadHoldingRegisters failed: %v", err)
		}
		for i, v := range values {
			if regs[i] != v {
				t.Errorf("Register[%d]: expected %d, got %d", 100+i, v, regs[i])
			}
		}
	})

	// Test WriteMultipleCoils
	t.Run("WriteMultipleCoils", func(t *testing.T) {
		values := []bool{true, false, true, false, true}
		if err := client.WriteMultipleCoils(ctx, 50, values); err != nil {
			t.Fatalf("WriteMultipleCoils failed: %v", err)
		}

		coils, err := client.ReadCoils(ctx, 50, 5)
		if err != nil {
			t.Fatalf("ReadCoils failed: %v", err)
		}
		for i, v := range values {
			if coils[i] != v {
				t.Errorf("Coil[%d]: expected %v, got %v", 50+i, v, coils[i])
			}
		}
	})

	// Test Diagnostics
	t.Run("Diagnostics", func(t *testing.T) {
		data := []byte{0x12, 0x34}
		resp, err := client.Diagnostics(ctx, DiagReturnQueryData, data)
		if err != nil {
			t.Fatalf("Diagnostics failed: %v", err)
		}
		if len(resp) != len(data) {
			t.Errorf("Expected echo of %d bytes, got %d", len(data), len(resp))
		}
	})

	// Test ReportServerID
	t.Run("ReportServerID", func(t *testing.T) {
		id, err := client.ReportServerID(ctx)
		if err != nil {
			t.Fatalf("ReportServerID failed: %v", err)
		}
		if len(id) == 0 {
			t.Error("Server ID should not be empty")
		}
	})
}
