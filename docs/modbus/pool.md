# Connection Pool

The connection pool allows reusing Modbus connections for better performance.

## Creation

```go
pool, err := modbus.NewPool(addr string, opts ...PoolOption) (*Pool, error)
```

**Parameters:**
- `addr`: Modbus server address
- `opts`: Pool configuration options

```go
pool, err := modbus.NewPool("192.168.1.100:502",
    modbus.WithSize(10),
    modbus.WithMaxIdleTime(5*time.Minute),
    modbus.WithHealthCheckFrequency(1*time.Minute),
    modbus.WithClientOptions(
        modbus.WithTimeout(5*time.Second),
        modbus.WithUnitID(1),
    ),
)
if err != nil {
    log.Fatal(err)
}
defer pool.Close()
```

## Manual Usage

### Get / Put

```go
// Get a connection
client, err := pool.Get(ctx)
if err != nil {
    log.Fatal(err)
}

// Use the connection
regs, err := client.ReadHoldingRegisters(ctx, 0, 10)

// IMPORTANT: Always return the connection to the pool
pool.Put(client)
```

### Recommended Pattern

```go
func readRegisters(ctx context.Context, pool *modbus.Pool) ([]uint16, error) {
    client, err := pool.Get(ctx)
    if err != nil {
        return nil, err
    }
    defer pool.Put(client)

    return client.ReadHoldingRegisters(ctx, 0, 10)
}
```

## Automatic Return Usage

### GetPooled

The `GetPooled` method returns a wrapper that automatically returns the connection to the pool when `Close()` is called:

```go
pc, err := pool.GetPooled(ctx)
if err != nil {
    log.Fatal(err)
}
defer pc.Close()  // Automatically returns to the pool

regs, err := pc.ReadHoldingRegisters(ctx, 0, 10)
```

### Discard

If the connection is in an invalid state, use `Discard()` instead of `Close()`:

```go
pc, err := pool.GetPooled(ctx)
if err != nil {
    log.Fatal(err)
}

regs, err := pc.ReadHoldingRegisters(ctx, 0, 10)
if err != nil {
    pc.Discard()  // Does not return to the pool, closes permanently
    return nil, err
}

pc.Close()  // Returns to the pool
return regs, nil
```

## Pool Options

| Option | Description | Default |
|--------|-------------|---------|
| `WithSize(n)` | Maximum pool size | 5 |
| `WithMaxIdleTime(d)` | Max idle time before closing | 5 min |
| `WithHealthCheckFrequency(d)` | Health check frequency | 1 min |
| `WithClientOptions(opts...)` | Options for created clients | - |

```go
pool, _ := modbus.NewPool("localhost:502",
    modbus.WithSize(20),                           // 20 max connections
    modbus.WithMaxIdleTime(10*time.Minute),        // Close after 10min of inactivity
    modbus.WithHealthCheckFrequency(30*time.Second), // Check every 30s
    modbus.WithClientOptions(
        modbus.WithTimeout(3*time.Second),
        modbus.WithAutoReconnect(false),  // Pool handles reconnection
    ),
)
```

## Statistics

```go
stats := pool.Stats()
fmt.Printf("Pool size: %d\n", stats.Size)
fmt.Printf("Connections created: %d\n", stats.Created)
fmt.Printf("Available connections: %d\n", stats.Available)
fmt.Printf("Total gets: %d\n", stats.Gets)
fmt.Printf("Total puts: %d\n", stats.Puts)
fmt.Printf("Hits (reuse): %d\n", stats.Hits)
fmt.Printf("Misses (new connection): %d\n", stats.Misses)
fmt.Printf("Timeouts: %d\n", stats.Timeouts)
```

## Metrics

```go
type PoolMetrics struct {
    Gets      Counter  // Number of Get calls
    Puts      Counter  // Number of Put calls
    Hits      Counter  // Reused connections
    Misses    Counter  // New connections created
    Timeouts  Counter  // Timeouts during Get
    Created   Counter  // Total connections created
    Closed    Counter  // Total connections closed
    Available Counter  // Currently available connections
}
```

```go
metrics := pool.Metrics()
hitRate := float64(metrics.Hits.Value()) / float64(metrics.Gets.Value()) * 100
fmt.Printf("Reuse rate: %.1f%%\n", hitRate)
```

## Closing

```go
err := pool.Close()
```

Closing:
1. Stops the health checker
2. Closes all active connections
3. Waits for goroutines to finish

## Pool Behavior

### Getting a Connection

1. Attempts to get an available connection from the pool
2. Verifies the connection is valid (connected, not too old)
3. If no connection available and `created < size`, creates a new connection
4. Otherwise, waits for a connection to become available or for context to expire

### Returning a Connection

1. Verifies the connection is still connected
2. If connected, returns it to the pool
3. If disconnected or pool is full, closes it

### Health Check

The health checker periodically verifies:
- That connections are still active
- That connections are not too old (idle time)

Invalid connections are automatically closed.
