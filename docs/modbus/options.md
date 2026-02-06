# Configuration

Configuration uses the functional options pattern.

## Client Options

### WithUnitID

Sets the default Unit ID for requests.

```go
modbus.WithUnitID(id UnitID)
```

```go
client, _ := modbus.NewClient("localhost:502",
    modbus.WithUnitID(1),
)
```

The Unit ID identifies the target device on the Modbus network. Values: 1-247.

### WithTimeout

Sets the timeout for operations.

```go
modbus.WithTimeout(d time.Duration)
```

```go
client, _ := modbus.NewClient("localhost:502",
    modbus.WithTimeout(5*time.Second),
)
```

**Default:** 5 seconds

### WithAutoReconnect

Enables automatic reconnection on connection loss.

```go
modbus.WithAutoReconnect(enable bool)
```

```go
client, _ := modbus.NewClient("localhost:502",
    modbus.WithAutoReconnect(true),
)
```

**Default:** false

### WithReconnectBackoff

Sets the initial delay between reconnection attempts.

```go
modbus.WithReconnectBackoff(d time.Duration)
```

The backoff increases exponentially up to `MaxReconnectTime`.

**Default:** 1 second

### WithMaxReconnectTime

Sets the maximum delay between reconnection attempts.

```go
modbus.WithMaxReconnectTime(d time.Duration)
```

**Default:** 30 seconds

### WithMaxRetries

Sets the maximum number of attempts for a request.

```go
modbus.WithMaxRetries(n int)
```

```go
client, _ := modbus.NewClient("localhost:502",
    modbus.WithAutoReconnect(true),
    modbus.WithMaxRetries(5),
)
```

**Default:** 3

### WithOnConnect

Sets a callback called on connection.

```go
modbus.WithOnConnect(fn func())
```

```go
client, _ := modbus.NewClient("localhost:502",
    modbus.WithOnConnect(func() {
        log.Println("Connected!")
    }),
)
```

### WithOnDisconnect

Sets a callback called on disconnection.

```go
modbus.WithOnDisconnect(fn func(error))
```

```go
client, _ := modbus.NewClient("localhost:502",
    modbus.WithOnDisconnect(func(err error) {
        log.Printf("Disconnected: %v\n", err)
    }),
)
```

### WithLogger

Sets the logger for the client.

```go
modbus.WithLogger(logger *slog.Logger)
```

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

client, _ := modbus.NewClient("localhost:502",
    modbus.WithLogger(logger),
)
```

**Default:** `slog.Default()`

## Server Options

### WithServerLogger

Sets the logger for the server.

```go
modbus.WithServerLogger(logger *slog.Logger)
```

### WithMaxConnections

Sets the maximum number of simultaneous connections.

```go
modbus.WithMaxConnections(n int)
```

```go
server := modbus.NewServer(handler,
    modbus.WithMaxConnections(100),
)
```

**Default:** 100

### WithReadTimeout

Sets the read timeout for client connections.

```go
modbus.WithReadTimeout(d time.Duration)
```

Connections inactive longer than this timeout are closed.

**Default:** 30 seconds

## Pool Options

### WithSize

Sets the maximum pool size.

```go
modbus.WithSize(size int)
```

```go
pool, _ := modbus.NewPool("localhost:502",
    modbus.WithSize(20),
)
```

**Default:** 5

### WithMaxIdleTime

Sets the maximum idle time before closing a connection.

```go
modbus.WithMaxIdleTime(d time.Duration)
```

```go
pool, _ := modbus.NewPool("localhost:502",
    modbus.WithMaxIdleTime(10*time.Minute),
)
```

**Default:** 5 minutes

### WithHealthCheckFrequency

Sets the frequency of connection health checks.

```go
modbus.WithHealthCheckFrequency(d time.Duration)
```

```go
pool, _ := modbus.NewPool("localhost:502",
    modbus.WithHealthCheckFrequency(30*time.Second),
)
```

**Default:** 1 minute. Set to 0 to disable.

### WithClientOptions

Sets options to use when creating pool clients.

```go
modbus.WithClientOptions(opts ...Option)
```

```go
pool, _ := modbus.NewPool("localhost:502",
    modbus.WithSize(10),
    modbus.WithClientOptions(
        modbus.WithTimeout(3*time.Second),
        modbus.WithUnitID(1),
    ),
)
```

## Complete Example

```go
package main

import (
    "log/slog"
    "os"
    "time"

    "github.com/edgeo-scada/modbus-tcp/modbus"
)

func main() {
    // Custom logger
    logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))

    // Client with all options
    client, _ := modbus.NewClient("192.168.1.100:502",
        // Identification
        modbus.WithUnitID(1),

        // Timeouts
        modbus.WithTimeout(5*time.Second),

        // Reconnection
        modbus.WithAutoReconnect(true),
        modbus.WithReconnectBackoff(500*time.Millisecond),
        modbus.WithMaxReconnectTime(30*time.Second),
        modbus.WithMaxRetries(5),

        // Callbacks
        modbus.WithOnConnect(func() {
            logger.Info("connected")
        }),
        modbus.WithOnDisconnect(func(err error) {
            logger.Warn("disconnected", slog.String("error", err.Error()))
        }),

        // Logging
        modbus.WithLogger(logger),
    )
    defer client.Close()
}
```
