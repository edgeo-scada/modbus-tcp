# Error Handling

## Standard Errors

The package defines several sentinel errors:

| Error | Description |
|-------|-------------|
| `ErrInvalidResponse` | Malformed or unexpected response |
| `ErrInvalidCRC` | CRC validation failed (RTU mode) |
| `ErrInvalidFrame` | Malformed frame |
| `ErrTimeout` | Timeout exceeded |
| `ErrConnectionClosed` | Connection closed |
| `ErrInvalidQuantity` | Invalid quantity (out of bounds) |
| `ErrInvalidAddress` | Invalid address |
| `ErrPoolExhausted` | Connection pool exhausted |
| `ErrPoolClosed` | Connection pool closed |
| `ErrNotConnected` | Client not connected |
| `ErrMaxRetriesExceeded` | Maximum retry count exceeded |

### Error Checking

```go
import "errors"

regs, err := client.ReadHoldingRegisters(ctx, 0, 10)
if err != nil {
    if errors.Is(err, modbus.ErrNotConnected) {
        // Reconnect
        client.Connect(ctx)
    } else if errors.Is(err, modbus.ErrTimeout) {
        // Retry
    } else if errors.Is(err, modbus.ErrMaxRetriesExceeded) {
        // Give up
    }
}
```

## Modbus Errors (Exceptions)

Modbus protocol errors are represented by `ModbusError`:

```go
type ModbusError struct {
    FunctionCode  FunctionCode
    ExceptionCode ExceptionCode
}
```

### Exception Codes

| Code | Constant | Description |
|------|----------|-------------|
| 0x01 | `ExceptionIllegalFunction` | Function not supported |
| 0x02 | `ExceptionIllegalDataAddress` | Invalid address |
| 0x03 | `ExceptionIllegalDataValue` | Invalid value |
| 0x04 | `ExceptionServerDeviceFailure` | Internal server error |
| 0x05 | `ExceptionAcknowledge` | Request accepted, processing in progress |
| 0x06 | `ExceptionServerDeviceBusy` | Server busy |
| 0x08 | `ExceptionMemoryParityError` | Memory parity error |
| 0x0A | `ExceptionGatewayPathUnavailable` | Gateway unavailable |
| 0x0B | `ExceptionGatewayTargetDeviceFailedToRespond` | Target device not responding |

### Exception Checking

```go
regs, err := client.ReadHoldingRegisters(ctx, 1000, 10)
if err != nil {
    // Check if it's a Modbus exception
    var modbusErr *modbus.ModbusError
    if errors.As(err, &modbusErr) {
        switch modbusErr.ExceptionCode {
        case modbus.ExceptionIllegalDataAddress:
            log.Println("Invalid address")
        case modbus.ExceptionIllegalDataValue:
            log.Println("Invalid value")
        case modbus.ExceptionServerDeviceBusy:
            log.Println("Server busy, retry later")
        default:
            log.Printf("Modbus exception: %v\n", modbusErr)
        }
    }
}
```

### Utility Functions

```go
// Check for a specific exception
if modbus.IsException(err, modbus.ExceptionIllegalDataAddress) {
    // Invalid address
}

// Shortcuts for common exceptions
if modbus.IsIllegalFunction(err) {
    // Function not supported
}

if modbus.IsIllegalDataAddress(err) {
    // Invalid address
}

if modbus.IsIllegalDataValue(err) {
    // Invalid value
}

if modbus.IsServerDeviceFailure(err) {
    // Server error
}
```

## Creating Modbus Errors (Server-side)

In your Handler, return Modbus errors:

```go
func (h *MyHandler) ReadHoldingRegisters(unitID modbus.UnitID, addr, qty uint16) ([]uint16, error) {
    // Check address
    if addr >= 10000 {
        return nil, modbus.NewModbusError(
            modbus.FuncReadHoldingRegisters,
            modbus.ExceptionIllegalDataAddress,
        )
    }

    // Check quantity
    if qty > 125 {
        return nil, modbus.NewModbusError(
            modbus.FuncReadHoldingRegisters,
            modbus.ExceptionIllegalDataValue,
        )
    }

    // Internal error
    values, err := h.db.Query(...)
    if err != nil {
        return nil, modbus.NewModbusError(
            modbus.FuncReadHoldingRegisters,
            modbus.ExceptionServerDeviceFailure,
        )
    }

    return values, nil
}
```

## Connection Errors

### Handling with Automatic Reconnection

```go
client, _ := modbus.NewClient("localhost:502",
    modbus.WithAutoReconnect(true),
    modbus.WithMaxRetries(5),
    modbus.WithOnDisconnect(func(err error) {
        log.Printf("Disconnection: %v\n", err)
    }),
)

// Network errors are automatically handled with retry
regs, err := client.ReadHoldingRegisters(ctx, 0, 10)
if err != nil {
    // After 5 failed attempts
    if errors.Is(err, modbus.ErrMaxRetriesExceeded) {
        log.Fatal("Unable to reach the server")
    }
}
```

### Manual Handling

```go
client, _ := modbus.NewClient("localhost:502",
    modbus.WithAutoReconnect(false),
)

regs, err := client.ReadHoldingRegisters(ctx, 0, 10)
if err != nil {
    if errors.Is(err, modbus.ErrNotConnected) {
        // Manual reconnection
        if err := client.Connect(ctx); err != nil {
            log.Fatal(err)
        }
        // Retry
        regs, err = client.ReadHoldingRegisters(ctx, 0, 10)
    }
}
```

## Best Practices

1. **Always check errors** - Never ignore returned errors

2. **Distinguish error types** - Modbus errors (protocol) vs network errors

3. **Log errors** - For debugging and monitoring

4. **Use timeouts** - Avoid indefinite blocking

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

regs, err := client.ReadHoldingRegisters(ctx, 0, 10)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        log.Println("Timeout - the server is not responding")
    }
}
```
