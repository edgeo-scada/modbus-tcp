package modbus

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestNewPool(t *testing.T) {
	pool, err := NewPool("localhost:502", WithSize(5))
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}
	defer pool.Close()

	stats := pool.Stats()
	if stats.Size != 5 {
		t.Errorf("Size: expected 5, got %d", stats.Size)
	}
}

func TestPoolIntegration(t *testing.T) {
	// Start test server
	handler := NewMemoryHandler(65536, 65536)
	handler.SetHoldingRegister(1, 0, 1234)
	server := NewServer(handler)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	go server.Serve(listener)
	defer server.Close()

	addr := listener.Addr().String()

	// Create pool
	pool, err := NewPool(addr,
		WithSize(3),
		WithClientOptions(WithUnitID(1)),
	)
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// Get a client
	client, err := pool.Get(ctx)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Use client
	regs, err := client.ReadHoldingRegisters(ctx, 0, 1)
	if err != nil {
		t.Fatalf("ReadHoldingRegisters failed: %v", err)
	}
	if regs[0] != 1234 {
		t.Errorf("Register: expected 1234, got %d", regs[0])
	}

	// Return to pool
	pool.Put(client)

	// Check stats
	stats := pool.Stats()
	if stats.Gets != 1 {
		t.Errorf("Gets: expected 1, got %d", stats.Gets)
	}
	if stats.Puts != 1 {
		t.Errorf("Puts: expected 1, got %d", stats.Puts)
	}
}

func TestPoolGetMultiple(t *testing.T) {
	// Start test server
	handler := NewMemoryHandler(65536, 65536)
	server := NewServer(handler)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	go server.Serve(listener)
	defer server.Close()

	addr := listener.Addr().String()

	// Create pool with size 2
	pool, err := NewPool(addr, WithSize(2))
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// Get 2 clients
	client1, err := pool.Get(ctx)
	if err != nil {
		t.Fatalf("Get client1 failed: %v", err)
	}

	client2, err := pool.Get(ctx)
	if err != nil {
		t.Fatalf("Get client2 failed: %v", err)
	}

	// Third get should wait/block (we test with timeout)
	ctxTimeout, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	_, err = pool.Get(ctxTimeout)
	if err == nil {
		t.Error("Expected timeout error when pool exhausted")
	}

	// Return clients
	pool.Put(client1)
	pool.Put(client2)

	// Now we should be able to get again
	client3, err := pool.Get(ctx)
	if err != nil {
		t.Fatalf("Get client3 failed: %v", err)
	}
	pool.Put(client3)
}

func TestPooledClient(t *testing.T) {
	// Start test server
	handler := NewMemoryHandler(65536, 65536)
	handler.SetHoldingRegister(1, 0, 5555)
	server := NewServer(handler)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	go server.Serve(listener)
	defer server.Close()

	addr := listener.Addr().String()

	pool, err := NewPool(addr, WithSize(2), WithClientOptions(WithUnitID(1)))
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// Get pooled client
	pc, err := pool.GetPooled(ctx)
	if err != nil {
		t.Fatalf("GetPooled failed: %v", err)
	}

	// Use it
	regs, err := pc.ReadHoldingRegisters(ctx, 0, 1)
	if err != nil {
		t.Fatalf("ReadHoldingRegisters failed: %v", err)
	}
	if regs[0] != 5555 {
		t.Errorf("Register: expected 5555, got %d", regs[0])
	}

	// Close returns to pool
	pc.Close()

	// Multiple close is safe
	pc.Close()

	stats := pool.Stats()
	if stats.Available != 1 {
		t.Errorf("Available: expected 1, got %d", stats.Available)
	}
}

func TestPoolClose(t *testing.T) {
	pool, err := NewPool("localhost:502", WithSize(3))
	if err != nil {
		t.Fatalf("NewPool failed: %v", err)
	}

	// Close pool
	if err := pool.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Get should fail
	ctx := context.Background()
	_, err = pool.Get(ctx)
	if err != ErrPoolClosed {
		t.Errorf("Expected ErrPoolClosed, got %v", err)
	}

	// Double close is safe
	pool.Close()
}
