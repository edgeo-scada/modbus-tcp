package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// TCPTransport implements a TCP transport for Modbus TCP.
type TCPTransport struct {
	addr    string
	timeout time.Duration

	mu   sync.Mutex
	conn net.Conn
}

// NewTCPTransport creates a new TCP transport.
func NewTCPTransport(addr string, timeout time.Duration) *TCPTransport {
	return &TCPTransport{
		addr:    addr,
		timeout: timeout,
	}
}

// Connect establishes a TCP connection.
func (t *TCPTransport) Connect(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.conn != nil {
		return nil // Already connected
	}

	dialer := &net.Dialer{
		Timeout:   t.timeout,
		KeepAlive: 30 * time.Second, // Enable TCP keep-alive for industrial reliability
	}

	conn, err := dialer.DialContext(ctx, "tcp", t.addr)
	if err != nil {
		return fmt.Errorf("tcp connect: %w", err)
	}

	// Configure TCP options for industrial use
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
		tcpConn.SetNoDelay(true) // Disable Nagle's algorithm for low latency
	}

	t.conn = conn
	return nil
}

// Close closes the TCP connection.
func (t *TCPTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.conn == nil {
		return nil
	}

	err := t.conn.Close()
	t.conn = nil
	return err
}

// IsConnected returns true if the transport is connected.
func (t *TCPTransport) IsConnected() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.conn != nil
}

// Send sends data and returns the response.
// This method is thread-safe and holds the lock during the entire transaction.
func (t *TCPTransport) Send(ctx context.Context, data []byte) ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.conn == nil {
		return nil, errors.New("not connected")
	}

	// Set deadline from context or use default timeout
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(t.timeout)
	}

	if err := t.conn.SetDeadline(deadline); err != nil {
		return nil, fmt.Errorf("set deadline: %w", err)
	}

	// Send request
	written := 0
	for written < len(data) {
		n, err := t.conn.Write(data[written:])
		if err != nil {
			t.closeConnLocked()
			return nil, fmt.Errorf("write: %w", err)
		}
		written += n
	}

	// Read MBAP header (7 bytes)
	header := make([]byte, 7)
	if err := t.readFullLocked(header); err != nil {
		t.closeConnLocked()
		return nil, fmt.Errorf("read header: %w", err)
	}

	// Validate protocol ID (bytes 2-3 must be 0x0000)
	protocolID := int(header[2])<<8 | int(header[3])
	if protocolID != 0 {
		t.closeConnLocked()
		return nil, fmt.Errorf("invalid protocol ID: %d", protocolID)
	}

	// Parse length from header (bytes 4-5)
	length := int(header[4])<<8 | int(header[5])
	if length < 1 || length > 254 {
		t.closeConnLocked()
		return nil, fmt.Errorf("invalid length: %d", length)
	}

	// Read PDU (length - 1 for unit ID which is in header)
	pduLen := length - 1
	response := make([]byte, 7+pduLen)
	copy(response, header)
	if pduLen > 0 {
		if err := t.readFullLocked(response[7:]); err != nil {
			t.closeConnLocked()
			return nil, fmt.Errorf("read pdu: %w", err)
		}
	}

	return response, nil
}

// Conn returns the underlying connection.
func (t *TCPTransport) Conn() net.Conn {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.conn
}

// SetConn sets the underlying connection (used for reconnection).
func (t *TCPTransport) SetConn(conn net.Conn) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.conn = conn
}

// closeConnLocked closes the connection without acquiring the lock.
// Must be called with mu held.
func (t *TCPTransport) closeConnLocked() {
	if t.conn != nil {
		t.conn.Close()
		t.conn = nil
	}
}

// readFullLocked reads exactly len(buf) bytes.
// Must be called with mu held.
func (t *TCPTransport) readFullLocked(buf []byte) error {
	total := 0
	for total < len(buf) {
		n, err := t.conn.Read(buf[total:])
		total += n
		if err != nil {
			if err == io.EOF && total == len(buf) {
				return nil
			}
			return err
		}
	}
	return nil
}
