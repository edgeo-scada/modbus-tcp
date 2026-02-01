package modbus

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// Pool manages a pool of Modbus client connections.
type Pool struct {
	addr string
	opts *poolOptions

	mu       sync.Mutex
	conns    chan *pooledClient
	factory  func() (*Client, error)
	closed   int32
	size     int
	created  int
	metrics  *PoolMetrics
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

type pooledClient struct {
	client   *Client
	lastUsed time.Time
}

// PoolMetrics holds pool-specific metrics.
type PoolMetrics struct {
	Gets      Counter
	Puts      Counter
	Hits      Counter
	Misses    Counter
	Timeouts  Counter
	Created   Counter
	Closed    Counter
	Available Counter
}

// NewPool creates a new connection pool.
func NewPool(addr string, opts ...PoolOption) (*Pool, error) {
	if addr == "" {
		return nil, errors.New("modbus: pool address cannot be empty")
	}

	options := defaultPoolOptions()
	for _, opt := range opts {
		opt(options)
	}

	if options.size < 1 {
		options.size = 1
	}

	p := &Pool{
		addr:    addr,
		opts:    options,
		conns:   make(chan *pooledClient, options.size),
		size:    options.size,
		metrics: &PoolMetrics{},
		stopCh:  make(chan struct{}),
	}

	p.factory = func() (*Client, error) {
		return NewClient(addr, options.clientOpts...)
	}

	// Start health checker if enabled
	if options.healthCheckFreq > 0 {
		p.wg.Add(1)
		go p.healthChecker()
	}

	return p, nil
}

// Get retrieves a client from the pool, creating one if necessary.
func (p *Pool) Get(ctx context.Context) (*Client, error) {
	if atomic.LoadInt32(&p.closed) == 1 {
		return nil, ErrPoolClosed
	}

	p.metrics.Gets.Add(1)

	// Try to get an existing connection
	select {
	case pc := <-p.conns:
		p.metrics.Hits.Add(1)
		p.metrics.Available.Add(-1)

		// Check if connection is still valid
		if pc.client.State() != StateConnected {
			pc.client.Close()
			p.decrementCreated()
			return p.createAndConnect(ctx)
		}

		// Check if connection is too old
		if p.opts.maxIdleTime > 0 && time.Since(pc.lastUsed) > p.opts.maxIdleTime {
			pc.client.Close()
			p.decrementCreated()
			return p.createAndConnect(ctx)
		}

		return pc.client, nil

	default:
		p.metrics.Misses.Add(1)
	}

	// No available connection, create a new one if under limit
	p.mu.Lock()
	if p.created < p.size {
		p.created++
		p.mu.Unlock()
		return p.createAndConnect(ctx)
	}
	p.mu.Unlock()

	// Wait for a connection to become available
	select {
	case pc := <-p.conns:
		p.metrics.Available.Add(-1)
		if pc.client.State() != StateConnected {
			pc.client.Close()
			p.decrementCreated()
			return p.createAndConnect(ctx)
		}
		return pc.client, nil

	case <-ctx.Done():
		p.metrics.Timeouts.Add(1)
		return nil, ctx.Err()

	case <-p.stopCh:
		return nil, ErrPoolClosed
	}
}

func (p *Pool) decrementCreated() {
	p.mu.Lock()
	p.created--
	p.mu.Unlock()
}

func (p *Pool) createAndConnect(ctx context.Context) (*Client, error) {
	client, err := p.factory()
	if err != nil {
		p.decrementCreated()
		return nil, err
	}

	if err := client.Connect(ctx); err != nil {
		client.Close()
		p.decrementCreated()
		return nil, err
	}

	p.metrics.Created.Add(1)
	return client, nil
}

// Put returns a client to the pool.
func (p *Pool) Put(client *Client) {
	if client == nil {
		return
	}

	if atomic.LoadInt32(&p.closed) == 1 {
		client.Close()
		p.decrementCreated()
		p.metrics.Closed.Add(1)
		return
	}

	p.metrics.Puts.Add(1)

	// Don't return disconnected clients
	if client.State() != StateConnected {
		client.Close()
		p.decrementCreated()
		p.metrics.Closed.Add(1)
		return
	}

	pc := &pooledClient{
		client:   client,
		lastUsed: time.Now(),
	}

	select {
	case p.conns <- pc:
		p.metrics.Available.Add(1)
	default:
		// Pool is full, close the client
		client.Close()
		p.decrementCreated()
		p.metrics.Closed.Add(1)
	}
}

// Close closes all connections in the pool.
func (p *Pool) Close() error {
	if !atomic.CompareAndSwapInt32(&p.closed, 0, 1) {
		return nil
	}

	// Signal health checker to stop
	close(p.stopCh)

	// Close the channel to unblock any waiting Get calls
	close(p.conns)

	// Drain and close all connections
	for pc := range p.conns {
		pc.client.Close()
		p.metrics.Closed.Add(1)
	}

	// Wait for health checker to finish
	p.wg.Wait()

	return nil
}

// Stats returns pool statistics.
func (p *Pool) Stats() PoolStats {
	p.mu.Lock()
	created := p.created
	p.mu.Unlock()

	return PoolStats{
		Size:      p.size,
		Created:   created,
		Available: len(p.conns),
		Gets:      p.metrics.Gets.Value(),
		Puts:      p.metrics.Puts.Value(),
		Hits:      p.metrics.Hits.Value(),
		Misses:    p.metrics.Misses.Value(),
		Timeouts:  p.metrics.Timeouts.Value(),
	}
}

// Metrics returns the pool metrics.
func (p *Pool) Metrics() *PoolMetrics {
	return p.metrics
}

// PoolStats holds pool statistics.
type PoolStats struct {
	Size      int
	Created   int
	Available int
	Gets      int64
	Puts      int64
	Hits      int64
	Misses    int64
	Timeouts  int64
}

func (p *Pool) healthChecker() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.opts.healthCheckFreq)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if atomic.LoadInt32(&p.closed) == 1 {
				return
			}
			p.checkHealth()
		case <-p.stopCh:
			return
		}
	}
}

func (p *Pool) checkHealth() {
	// Drain and check all connections
	checked := make([]*pooledClient, 0, p.size)

	// Use a timeout to prevent blocking forever
	timeout := time.NewTimer(100 * time.Millisecond)
	defer timeout.Stop()

	for {
		select {
		case pc, ok := <-p.conns:
			if !ok {
				// Channel closed
				return
			}
			p.metrics.Available.Add(-1)
			if pc.client.State() == StateConnected {
				if p.opts.maxIdleTime == 0 || time.Since(pc.lastUsed) <= p.opts.maxIdleTime {
					checked = append(checked, pc)
					continue
				}
			}
			// Close unhealthy or idle connection
			pc.client.Close()
			p.decrementCreated()
			p.metrics.Closed.Add(1)

		case <-timeout.C:
			// Put healthy connections back
			for _, pc := range checked {
				select {
				case p.conns <- pc:
					p.metrics.Available.Add(1)
				default:
					pc.client.Close()
					p.decrementCreated()
					p.metrics.Closed.Add(1)
				}
			}
			return

		default:
			// No more connections to check
			// Put healthy connections back
			for _, pc := range checked {
				select {
				case p.conns <- pc:
					p.metrics.Available.Add(1)
				default:
					pc.client.Close()
					p.decrementCreated()
					p.metrics.Closed.Add(1)
				}
			}
			return
		}
	}
}

// PooledClient wraps a client with automatic return to pool.
type PooledClient struct {
	*Client
	pool     *Pool
	mu       sync.Mutex
	returned bool
}

// GetPooled retrieves a client that automatically returns to the pool when closed.
func (p *Pool) GetPooled(ctx context.Context) (*PooledClient, error) {
	client, err := p.Get(ctx)
	if err != nil {
		return nil, err
	}

	return &PooledClient{
		Client: client,
		pool:   p,
	}, nil
}

// Close returns the client to the pool instead of closing it.
func (pc *PooledClient) Close() error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.returned {
		return nil
	}
	pc.returned = true
	pc.pool.Put(pc.Client)
	return nil
}

// ForceClose actually closes the underlying connection without returning to pool.
func (pc *PooledClient) ForceClose() error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.returned {
		return nil
	}
	pc.returned = true

	// Decrement created count since we're not returning to pool
	pc.pool.decrementCreated()
	pc.pool.metrics.Closed.Add(1)

	return pc.Client.Close()
}

// Discard marks the client as bad and closes it without returning to pool.
// Use this when the connection is known to be in a bad state.
func (pc *PooledClient) Discard() error {
	return pc.ForceClose()
}
