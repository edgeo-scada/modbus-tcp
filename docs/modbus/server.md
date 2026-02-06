# Modbus TCP Server

The Modbus TCP server allows you to simulate a Modbus device or create a gateway.

## Creation

```go
server := modbus.NewServer(handler Handler, opts ...ServerOption) *Server
```

**Parameters:**
- `handler`: Implementation of the `Handler` interface
- `opts`: Configuration options

## Handler Interface

To create a server, you must implement the `Handler` interface:

```go
type Handler interface {
    // Coil operations
    ReadCoils(unitID UnitID, addr, qty uint16) ([]bool, error)
    ReadDiscreteInputs(unitID UnitID, addr, qty uint16) ([]bool, error)
    WriteSingleCoil(unitID UnitID, addr uint16, value bool) error
    WriteMultipleCoils(unitID UnitID, addr uint16, values []bool) error

    // Register operations
    ReadHoldingRegisters(unitID UnitID, addr, qty uint16) ([]uint16, error)
    ReadInputRegisters(unitID UnitID, addr, qty uint16) ([]uint16, error)
    WriteSingleRegister(unitID UnitID, addr, value uint16) error
    WriteMultipleRegisters(unitID UnitID, addr uint16, values []uint16) error

    // Diagnostic operations
    ReadExceptionStatus(unitID UnitID) (uint8, error)
    Diagnostics(unitID UnitID, subFunc uint16, data []byte) ([]byte, error)
    GetCommEventCounter(unitID UnitID) (status uint16, eventCount uint16, err error)
    ReportServerID(unitID UnitID) ([]byte, error)
}
```

## MemoryHandler

A memory handler is provided for testing and simulations:

```go
handler := modbus.NewMemoryHandler(coilSize, registerSize int)
```

### Data Initialization

```go
handler := modbus.NewMemoryHandler(65536, 65536)
unitID := modbus.UnitID(1)

// Coils
handler.SetCoil(unitID, 0, true)
handler.SetCoil(unitID, 1, false)

// Discrete inputs
handler.SetDiscreteInput(unitID, 0, true)

// Holding registers
handler.SetHoldingRegister(unitID, 0, 1234)
handler.SetHoldingRegister(unitID, 1, 5678)

// Input registers
handler.SetInputRegister(unitID, 0, 100)

// Server ID
handler.SetServerID([]byte("My Modbus Server v1.0"))
```

## Starting the Server

### ListenAndServe

```go
func (s *Server) ListenAndServe(addr string) error
```

```go
server := modbus.NewServer(handler)
if err := server.ListenAndServe(":502"); err != nil {
    log.Fatal(err)
}
```

### ListenAndServeContext

With context cancellation support:

```go
func (s *Server) ListenAndServeContext(ctx context.Context, addr string) error
```

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Graceful shutdown on signal
go func() {
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh
    cancel()
}()

server.ListenAndServeContext(ctx, ":502")
```

### Serve

With a custom listener:

```go
func (s *Server) Serve(listener net.Listener) error
```

```go
listener, err := net.Listen("tcp", ":502")
if err != nil {
    log.Fatal(err)
}
server.Serve(listener)
```

## Stopping the Server

```go
func (s *Server) Close() error
```

Properly closes all active connections.

## Server Information

```go
// Server address
addr := server.Addr()

// Number of active connections
count := server.ActiveConnections()
```

## Server Options

```go
server := modbus.NewServer(handler,
    modbus.WithServerLogger(logger),      // Custom logger
    modbus.WithMaxConnections(100),       // Max simultaneous connections
    modbus.WithReadTimeout(30*time.Second), // Read timeout
)
```

See [Options](./options.md) for the complete list.

## Server Metrics

```go
type ServerMetrics struct {
    RequestsTotal   Counter  // Total requests received
    RequestsSuccess Counter  // Successfully processed requests
    RequestsErrors  Counter  // Failed requests
    ActiveConns     Counter  // Active connections
    TotalConns      Counter  // Total connections received
}
```

```go
metrics := server.Metrics()
fmt.Printf("Active connections: %d\n", metrics.ActiveConns.Value())
fmt.Printf("Total requests: %d\n", metrics.RequestsTotal.Value())
```

## Custom Handler

Example of a custom handler that connects to a database:

```go
type DBHandler struct {
    db *sql.DB
}

func (h *DBHandler) ReadHoldingRegisters(unitID modbus.UnitID, addr, qty uint16) ([]uint16, error) {
    // Read from database
    rows, err := h.db.Query("SELECT value FROM registers WHERE unit_id = ? AND addr >= ? AND addr < ?",
        unitID, addr, addr+qty)
    if err != nil {
        return nil, modbus.NewModbusError(modbus.FuncReadHoldingRegisters, modbus.ExceptionServerDeviceFailure)
    }
    defer rows.Close()

    values := make([]uint16, qty)
    // ... populate values
    return values, nil
}

func (h *DBHandler) WriteSingleRegister(unitID modbus.UnitID, addr, value uint16) error {
    _, err := h.db.Exec("UPDATE registers SET value = ? WHERE unit_id = ? AND addr = ?",
        value, unitID, addr)
    if err != nil {
        return modbus.NewModbusError(modbus.FuncWriteSingleRegister, modbus.ExceptionServerDeviceFailure)
    }
    return nil
}

// Implement other methods...
```

## Returning Modbus Errors

Use `NewModbusError` to return standard Modbus exceptions:

```go
func (h *MyHandler) ReadHoldingRegisters(unitID modbus.UnitID, addr, qty uint16) ([]uint16, error) {
    if addr > 1000 {
        return nil, modbus.NewModbusError(
            modbus.FuncReadHoldingRegisters,
            modbus.ExceptionIllegalDataAddress,
        )
    }
    // ...
}
```

Available exceptions:
- `ExceptionIllegalFunction` (0x01)
- `ExceptionIllegalDataAddress` (0x02)
- `ExceptionIllegalDataValue` (0x03)
- `ExceptionServerDeviceFailure` (0x04)
- `ExceptionServerDeviceBusy` (0x06)
