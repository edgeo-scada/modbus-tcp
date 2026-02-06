# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2024-02-01

### Added

- **Modbus TCP Client**
  - Support for all standard Modbus functions (FC01-FC17)
  - Automatic reconnection with exponential backoff
  - Configuration via functional options
  - Built-in metrics (latency, counters)
  - Structured logging via `slog`
  - `WithUnit` variants for all operations

- **Modbus TCP Server**
  - Concurrent multi-client support
  - `MemoryHandler` for testing and simulations
  - `Handler` interface for custom implementations
  - Configurable connection limit
  - Graceful shutdown with context

- **Connection Pool**
  - Connection reuse
  - Automatic health checks
  - Idle time management
  - `PooledClient` with automatic return

- **Metrics**
  - Thread-safe atomic counters
  - Latency histogram with buckets
  - Metrics by function code
  - Prometheus/expvar compatible export

- **Error Handling**
  - Typed Modbus errors (`ModbusError`)
  - All standard exception codes
  - Utility functions (`IsException`, `IsIllegalDataAddress`, etc.)

### Security

- Input validation on all operations
- Protection against address overflows
- Configurable timeouts to prevent blocking

---

## Versioning Convention

This project uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html):

- **MAJOR** (X.0.0): Backward-incompatible changes
- **MINOR** (0.X.0): Backward-compatible new features
- **PATCH** (0.0.X): Backward-compatible bug fixes

### Accessing the Version

```go
import "github.com/edgeo-scada/modbus-tcp/modbus"

// Version string
fmt.Println(modbus.Version) // "1.0.0"

// Detailed version
info := modbus.GetVersion()
fmt.Printf("v%d.%d.%d\n", info.Major, info.Minor, info.Patch)
```
a/modbus-tcp/modbus"

// Version string
fmt.Println(modbus.Version) // "1.0.0"

// Detailed version
info := modbus.GetVersion()
fmt.Printf("v%d.%d.%d\n", info.Major, info.Minor, info.Patch)
```
