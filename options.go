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
	"log/slog"
	"time"
)

// Option is a functional option for configuring the client.
type Option func(*clientOptions)

type clientOptions struct {
	// Connection settings
	unitID  UnitID
	timeout time.Duration

	// Reconnection settings
	autoReconnect    bool
	reconnectBackoff time.Duration
	maxReconnectTime time.Duration
	maxRetries       int

	// Callbacks
	onConnect    func()
	onDisconnect func(error)

	// Logging
	logger *slog.Logger

	// Pool settings (for pool creation)
	poolSize int
}

func defaultOptions() *clientOptions {
	return &clientOptions{
		unitID:           1,
		timeout:          DefaultTimeout,
		autoReconnect:    false,
		reconnectBackoff: 1 * time.Second,
		maxReconnectTime: 30 * time.Second,
		maxRetries:       3,
		logger:           slog.Default(),
		poolSize:         5,
	}
}

// WithUnitID sets the default unit ID for requests.
func WithUnitID(id UnitID) Option {
	return func(o *clientOptions) {
		o.unitID = id
	}
}

// WithTimeout sets the timeout for operations.
func WithTimeout(d time.Duration) Option {
	return func(o *clientOptions) {
		o.timeout = d
	}
}

// WithAutoReconnect enables automatic reconnection on connection loss.
func WithAutoReconnect(enable bool) Option {
	return func(o *clientOptions) {
		o.autoReconnect = enable
	}
}

// WithReconnectBackoff sets the initial backoff duration for reconnection attempts.
func WithReconnectBackoff(d time.Duration) Option {
	return func(o *clientOptions) {
		o.reconnectBackoff = d
	}
}

// WithMaxReconnectTime sets the maximum time between reconnection attempts.
func WithMaxReconnectTime(d time.Duration) Option {
	return func(o *clientOptions) {
		o.maxReconnectTime = d
	}
}

// WithMaxRetries sets the maximum number of retries for operations.
func WithMaxRetries(n int) Option {
	return func(o *clientOptions) {
		o.maxRetries = n
	}
}

// WithOnConnect sets a callback to be called when the connection is established.
func WithOnConnect(fn func()) Option {
	return func(o *clientOptions) {
		o.onConnect = fn
	}
}

// WithOnDisconnect sets a callback to be called when the connection is lost.
func WithOnDisconnect(fn func(error)) Option {
	return func(o *clientOptions) {
		o.onDisconnect = fn
	}
}

// WithLogger sets the logger for the client.
func WithLogger(logger *slog.Logger) Option {
	return func(o *clientOptions) {
		o.logger = logger
	}
}

// WithPoolSize sets the connection pool size.
func WithPoolSize(size int) Option {
	return func(o *clientOptions) {
		o.poolSize = size
	}
}

// ServerOption is a functional option for configuring the server.
type ServerOption func(*serverOptions)

type serverOptions struct {
	logger      *slog.Logger
	maxConns    int
	readTimeout time.Duration
}

func defaultServerOptions() *serverOptions {
	return &serverOptions{
		logger:      slog.Default(),
		maxConns:    100,
		readTimeout: 30 * time.Second,
	}
}

// WithServerLogger sets the logger for the server.
func WithServerLogger(logger *slog.Logger) ServerOption {
	return func(o *serverOptions) {
		o.logger = logger
	}
}

// WithMaxConnections sets the maximum number of concurrent connections.
func WithMaxConnections(n int) ServerOption {
	return func(o *serverOptions) {
		o.maxConns = n
	}
}

// WithReadTimeout sets the read timeout for client connections.
func WithReadTimeout(d time.Duration) ServerOption {
	return func(o *serverOptions) {
		o.readTimeout = d
	}
}

// PoolOption is a functional option for configuring the connection pool.
type PoolOption func(*poolOptions)

type poolOptions struct {
	size            int
	maxIdleTime     time.Duration
	healthCheckFreq time.Duration
	clientOpts      []Option
}

func defaultPoolOptions() *poolOptions {
	return &poolOptions{
		size:            5,
		maxIdleTime:     5 * time.Minute,
		healthCheckFreq: 1 * time.Minute,
	}
}

// WithSize sets the pool size.
func WithSize(size int) PoolOption {
	return func(o *poolOptions) {
		o.size = size
	}
}

// WithMaxIdleTime sets the maximum idle time before a connection is closed.
func WithMaxIdleTime(d time.Duration) PoolOption {
	return func(o *poolOptions) {
		o.maxIdleTime = d
	}
}

// WithHealthCheckFrequency sets how often to check connection health.
func WithHealthCheckFrequency(d time.Duration) PoolOption {
	return func(o *poolOptions) {
		o.healthCheckFreq = d
	}
}

// WithClientOptions sets the options to use when creating new client connections.
func WithClientOptions(opts ...Option) PoolOption {
	return func(o *poolOptions) {
		o.clientOpts = opts
	}
}
