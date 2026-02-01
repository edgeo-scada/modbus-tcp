# Modbus TCP Driver

A comprehensive Modbus TCP client library and CLI tool written in Go.

## Features

### Library (`modbus/`)

- Full Modbus TCP protocol implementation
- All standard function codes (FC01-FC17)
- Configurable timeouts, retries, and unit IDs
- Thread-safe client with automatic reconnection
- Clean API with context support

### CLI (`modbuscli`)

- Read/write coils and registers
- Multiple output formats (table, JSON, CSV, hex, raw)
- Device scanning and discovery
- Continuous monitoring (watch mode)
- Interactive REPL mode
- Register range dump with hexdump support
- Diagnostic functions
- Configuration file support

## Installation

### CLI Tool

```bash
go install github.com/edgeo/drivers/cmd/modbuscli@latest
```

### Library

```bash
go get github.com/edgeo/drivers/modbus
```

## Quick Start

### CLI Examples

```bash
# Read 10 holding registers from address 0
modbuscli read hr -a 0 -c 10 -H 192.168.1.100

# Read coils
modbuscli read coils -a 0 -c 16 -H 192.168.1.100

# Write a single register
modbuscli write register -a 100 -v 1234 -H 192.168.1.100

# Write a coil (ON)
modbuscli write coil -a 0 -v true -H 192.168.1.100

# Get device information
modbuscli info -H 192.168.1.100

# Scan for devices on a network
modbuscli scan -H 192.168.1.0/24

# Watch registers continuously (1 second interval)
modbuscli watch hr -a 0 -c 5 -i 1s -H 192.168.1.100

# Dump registers to CSV
modbuscli dump hr -a 0 -e 999 -f registers.csv -H 192.168.1.100

# Interactive mode
modbuscli interactive -H 192.168.1.100
```

### Library Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/edgeo/drivers/modbus"
)

func main() {
    // Create a new client
    client, err := modbus.NewClient(
        "192.168.1.100:502",
        modbus.WithUnitID(1),
        modbus.WithTimeout(5*time.Second),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    ctx := context.Background()

    // Connect to the device
    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }

    // Read holding registers
    values, err := client.ReadHoldingRegisters(ctx, 0, 10)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Registers: %v\n", values)

    // Write a single register
    if err := client.WriteSingleRegister(ctx, 100, 1234); err != nil {
        log.Fatal(err)
    }

    // Read coils
    coils, err := client.ReadCoils(ctx, 0, 16)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Coils: %v\n", coils)

    // Write a single coil
    if err := client.WriteSingleCoil(ctx, 0, true); err != nil {
        log.Fatal(err)
    }
}
```

## CLI Reference

### Global Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--host` | `-H` | Modbus server host | `localhost` |
| `--port` | `-p` | Modbus server port | `502` |
| `--unit` | `-u` | Modbus unit ID (1-247) | `1` |
| `--timeout` | `-t` | Operation timeout | `5s` |
| `--retries` | `-r` | Number of retries on failure | `3` |
| `--output` | `-o` | Output format: table, json, csv, hex, raw | `table` |
| `--verbose` | `-v` | Verbose output | `false` |
| `--no-color` | | Disable color output | `false` |
| `--byte-order` | | Byte order: big, little | `big` |
| `--word-order` | | Word order for 32-bit values | `big` |
| `--config` | | Config file path | `$HOME/.modbuscli.yaml` |

### Commands

#### Read Commands

```bash
# Read coils (FC01)
modbuscli read coils -a <address> -c <count>

# Read discrete inputs (FC02)
modbuscli read discrete-inputs -a <address> -c <count>

# Read holding registers (FC03)
modbuscli read holding-registers -a <address> -c <count>
# Aliases: hr, holding

# Read input registers (FC04)
modbuscli read input-registers -a <address> -c <count>
# Aliases: ir, input
```

#### Write Commands

```bash
# Write single coil (FC05)
modbuscli write coil -a <address> -v <true|false|1|0>

# Write multiple coils (FC15)
modbuscli write coils -a <address> -v <1,0,1,1,0>

# Write single register (FC06)
modbuscli write register -a <address> -v <value>

# Write multiple registers (FC16)
modbuscli write registers -a <address> -v <val1,val2,val3>
```

#### Scan Command

```bash
# Scan a single host for responding unit IDs
modbuscli scan -H 192.168.1.100

# Scan a network range
modbuscli scan -H 192.168.1.0/24

# Scan specific unit ID range
modbuscli scan -H 192.168.1.100 --unit-start 1 --unit-end 10

# Scan with custom concurrency
modbuscli scan -H 192.168.1.0/24 --concurrency 50
```

#### Watch Command

```bash
# Watch holding registers every second
modbuscli watch hr -a 0 -c 10 -i 1s -H 192.168.1.100

# Watch with change detection only
modbuscli watch hr -a 0 -c 10 --changes-only -H 192.168.1.100

# Watch with threshold alerts
modbuscli watch hr -a 0 -c 1 --alert-above 100 --alert-below 10 -H 192.168.1.100

# Watch coils
modbuscli watch coils -a 0 -c 16 -i 500ms -H 192.168.1.100
```

#### Dump Command

```bash
# Dump holding registers to stdout
modbuscli dump hr -a 0 -e 999 -H 192.168.1.100

# Dump to CSV file
modbuscli dump hr -a 0 -e 999 -f output.csv -H 192.168.1.100

# Dump as JSON
modbuscli dump hr -a 0 -e 999 -o json -H 192.168.1.100

# Dump as hexdump
modbuscli dump hr -a 0 -e 999 -o hex -H 192.168.1.100

# Dump coils
modbuscli dump coils -a 0 -e 1000 -H 192.168.1.100
```

#### Info Command

```bash
# Probe device and show capabilities
modbuscli info -H 192.168.1.100

# JSON output
modbuscli info -H 192.168.1.100 -o json
```

#### Diagnostic Commands

```bash
# Read exception status (FC07)
modbuscli diag exception -H 192.168.1.100

# Run diagnostics (FC08)
modbuscli diag run --sub 0 -H 192.168.1.100

# Get comm event counter (FC11)
modbuscli diag counter -H 192.168.1.100

# Report server ID (FC17)
modbuscli diag server-id -H 192.168.1.100
```

#### Interactive Mode

```bash
modbuscli interactive -H 192.168.1.100
```

Available commands in interactive mode:
- `read hr|ir|coils|di <addr> [count]` - Read registers/coils
- `write reg|coil <addr> <value>` - Write register/coil
- `info` - Show device information
- `scan [start] [end]` - Scan unit IDs
- `set unit|timeout <value>` - Change settings
- `help` - Show help
- `exit` / `quit` - Exit

## Configuration File

Create `~/.modbuscli.yaml`:

```yaml
host: 192.168.1.100
port: 502
unit: 1
timeout: 5s
retries: 3
output: table
verbose: false
byte-order: big
word-order: big
```

## Supported Function Codes

| Code | Function | Read | Write |
|------|----------|------|-------|
| FC01 | Read Coils | Yes | - |
| FC02 | Read Discrete Inputs | Yes | - |
| FC03 | Read Holding Registers | Yes | - |
| FC04 | Read Input Registers | Yes | - |
| FC05 | Write Single Coil | - | Yes |
| FC06 | Write Single Register | - | Yes |
| FC07 | Read Exception Status | Yes | - |
| FC08 | Diagnostics | Yes | - |
| FC11 | Get Comm Event Counter | Yes | - |
| FC15 | Write Multiple Coils | - | Yes |
| FC16 | Write Multiple Registers | - | Yes |
| FC17 | Report Server ID | Yes | - |

## Project Structure

```
.
├── cmd/
│   └── modbuscli/          # CLI application
│       ├── main.go
│       ├── root.go         # Root command and global flags
│       ├── read.go         # Read commands
│       ├── write.go        # Write commands
│       ├── scan.go         # Network/device scanning
│       ├── watch.go        # Continuous monitoring
│       ├── dump.go         # Register dump
│       ├── info.go         # Device information
│       ├── diag.go         # Diagnostic functions
│       ├── interactive.go  # REPL mode
│       └── output.go       # Output formatting
├── modbus/                 # Modbus library (importable)
│   ├── client.go           # Main client implementation
│   ├── functions.go        # Function code implementations
│   ├── protocol.go         # Protocol encoding/decoding
│   ├── types.go            # Type definitions
│   ├── options.go          # Client options
│   └── errors.go           # Error types
├── go.mod
├── go.sum
├── go.work                 # Go workspace for local development
└── README.md
```

## Building from Source

```bash
# Clone the repository
git clone https://github.com/edgeo/drivers.git
cd drivers

# Build the CLI
go build -o modbuscli ./cmd/modbuscli

# Run tests
go test ./...
```

## License

MIT License

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
