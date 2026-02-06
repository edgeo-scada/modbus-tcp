# Modbus TCP Client

The Modbus TCP client allows communication with Modbus devices (PLCs, sensors, etc.).

## Creation

```go
client, err := modbus.NewClient(addr string, opts ...Option) (*Client, error)
```

**Parameters:**
- `addr`: Modbus server address (e.g., `"192.168.1.100:502"`)
- `opts`: Configuration options (see [Options](./options.md))

**Example:**
```go
client, err := modbus.NewClient("192.168.1.100:502",
    modbus.WithUnitID(1),
    modbus.WithTimeout(5*time.Second),
    modbus.WithAutoReconnect(true),
)
```

## Connection and Disconnection

### Connect

```go
func (c *Client) Connect(ctx context.Context) error
```

Establishes the TCP connection with the server.

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := client.Connect(ctx); err != nil {
    log.Fatal(err)
}
```

### Close

```go
func (c *Client) Close() error
```

Closes the connection properly.

```go
defer client.Close()
```

### Connection State

```go
func (c *Client) IsConnected() bool
func (c *Client) State() ConnectionState
```

Possible states: `StateDisconnected`, `StateConnecting`, `StateConnected`

## Reading Data

### ReadCoils (FC01)

Reads coils (output bits).

```go
func (c *Client) ReadCoils(ctx context.Context, addr, qty uint16) ([]bool, error)
```

| Parameter | Description |
|-----------|-------------|
| `addr` | Starting address (0-65535) |
| `qty` | Number of coils (1-2000) |

```go
coils, err := client.ReadCoils(ctx, 0, 16)
// coils = []bool{true, false, true, ...}
```

### ReadDiscreteInputs (FC02)

Reads discrete inputs (input bits).

```go
func (c *Client) ReadDiscreteInputs(ctx context.Context, addr, qty uint16) ([]bool, error)
```

```go
inputs, err := client.ReadDiscreteInputs(ctx, 0, 8)
```

### ReadHoldingRegisters (FC03)

Reads holding registers (read/write).

```go
func (c *Client) ReadHoldingRegisters(ctx context.Context, addr, qty uint16) ([]uint16, error)
```

| Parameter | Description |
|-----------|-------------|
| `addr` | Starting address (0-65535) |
| `qty` | Number of registers (1-125) |

```go
regs, err := client.ReadHoldingRegisters(ctx, 100, 10)
// regs = []uint16{1234, 5678, ...}
```

### ReadInputRegisters (FC04)

Reads input registers (read-only).

```go
func (c *Client) ReadInputRegisters(ctx context.Context, addr, qty uint16) ([]uint16, error)
```

```go
inputs, err := client.ReadInputRegisters(ctx, 0, 5)
```

## Writing Data

### WriteSingleCoil (FC05)

Writes a single coil.

```go
func (c *Client) WriteSingleCoil(ctx context.Context, addr uint16, value bool) error
```

```go
err := client.WriteSingleCoil(ctx, 10, true)  // Enable coil 10
err = client.WriteSingleCoil(ctx, 10, false)  // Disable coil 10
```

### WriteSingleRegister (FC06)

Writes a single register.

```go
func (c *Client) WriteSingleRegister(ctx context.Context, addr, value uint16) error
```

```go
err := client.WriteSingleRegister(ctx, 100, 1234)
```

### WriteMultipleCoils (FC15)

Writes multiple coils.

```go
func (c *Client) WriteMultipleCoils(ctx context.Context, addr uint16, values []bool) error
```

```go
err := client.WriteMultipleCoils(ctx, 0, []bool{true, false, true, true})
```

### WriteMultipleRegisters (FC16)

Writes multiple registers.

```go
func (c *Client) WriteMultipleRegisters(ctx context.Context, addr uint16, values []uint16) error
```

```go
err := client.WriteMultipleRegisters(ctx, 100, []uint16{1111, 2222, 3333})
```

## Diagnostic Functions

### ReadExceptionStatus (FC07)

```go
func (c *Client) ReadExceptionStatus(ctx context.Context) (uint8, error)
```

### Diagnostics (FC08)

```go
func (c *Client) Diagnostics(ctx context.Context, subFunc uint16, data []byte) ([]byte, error)
```

Available sub-functions:
- `DiagReturnQueryData` (0x00): Echo
- `DiagRestartCommunications` (0x01)
- `DiagReturnDiagnosticRegister` (0x02)
- etc.

```go
// Echo test
resp, err := client.Diagnostics(ctx, modbus.DiagReturnQueryData, []byte{0x12, 0x34})
// resp = []byte{0x12, 0x34}
```

### GetCommEventCounter (FC11)

```go
func (c *Client) GetCommEventCounter(ctx context.Context) (status, eventCount uint16, err error)
```

### ReportServerID (FC17)

```go
func (c *Client) ReportServerID(ctx context.Context) ([]byte, error)
```

```go
serverID, err := client.ReportServerID(ctx)
fmt.Println(string(serverID))  // "Modbus Server v1.0"
```

## Operations with Specific Unit ID

All methods have a `WithUnit` variant to specify a different Unit ID:

```go
// Uses the default Unit ID
regs, _ := client.ReadHoldingRegisters(ctx, 0, 10)

// Uses a specific Unit ID
regs, _ := client.ReadHoldingRegistersWithUnit(ctx, modbus.UnitID(2), 0, 10)
```

Available methods:
- `ReadCoilsWithUnit`
- `ReadDiscreteInputsWithUnit`
- `ReadHoldingRegistersWithUnit`
- `ReadInputRegistersWithUnit`
- `WriteSingleCoilWithUnit`
- `WriteSingleRegisterWithUnit`
- `WriteMultipleCoilsWithUnit`
- `WriteMultipleRegistersWithUnit`

## Unit ID Management

```go
// Set the default Unit ID
client.SetUnitID(modbus.UnitID(2))

// Get the current Unit ID
unitID := client.UnitID()
```

## Metrics

```go
metrics := client.Metrics().Collect()
fmt.Printf("Total requests: %v\n", metrics["requests_total"])
fmt.Printf("Successful requests: %v\n", metrics["requests_success"])
fmt.Printf("Errors: %v\n", metrics["requests_errors"])
```

See [Metrics](./metrics.md) for more details.
