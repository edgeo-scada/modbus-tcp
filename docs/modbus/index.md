---
slug: /
---

# Modbus TCP Driver

[![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)](./changelog)
[![Go](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](https://github.com/edgeo-scada/modbus-tcp/blob/main/LICENSE)

A complete Go implementation of the Modbus TCP protocol, with client, server, and connection pool.

## Installation

```bash
go get github.com/edgeo-scada/modbus-tcp/modbus@v1.0.0
```

To verify the installed version:

```go
import "github.com/edgeo-scada/modbus-tcp/modbus"

func main() {
    fmt.Printf("Modbus driver version: %s\n", modbus.Version)
    // Output: Modbus driver version: 1.0.0
}
```

## Features

- **Modbus TCP Client** with automatic reconnection
- **Modbus TCP Server** with multi-client support
- **Connection Pool** with health checks
- **Built-in Metrics** (latency, counters, histograms)
- **Structured Logging** via `slog`

## Supported Modbus Functions

| Code | Function | Description |
|------|----------|-------------|
| FC01 | Read Coils | Read output bits |
| FC02 | Read Discrete Inputs | Read discrete inputs |
| FC03 | Read Holding Registers | Read holding registers |
| FC04 | Read Input Registers | Read input registers |
| FC05 | Write Single Coil | Write a single output bit |
| FC06 | Write Single Register | Write a single register |
| FC07 | Read Exception Status | Read exception status |
| FC08 | Diagnostics | Diagnostic operations |
| FC11 | Get Comm Event Counter | Event counter |
| FC15 | Write Multiple Coils | Write multiple bits |
| FC16 | Write Multiple Registers | Write multiple registers |
| FC17 | Report Server ID | Server identification |

## Quick Example

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/edgeo-scada/modbus-tcp/modbus"
)

func main() {
    // Create a client
    client, err := modbus.NewClient("localhost:502",
        modbus.WithTimeout(5*time.Second),
        modbus.WithAutoReconnect(true),
    )
    if err != nil {
        panic(err)
    }
    defer client.Close()

    // Connect
    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        panic(err)
    }

    // Read registers
    regs, err := client.ReadHoldingRegisters(ctx, 0, 10)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Registers: %v\n", regs)
}
```

## Package Structure

```
modbus/
├── client.go      # Modbus TCP Client
├── server.go      # Modbus TCP Server
├── pool.go        # Connection Pool
├── options.go     # Functional Configuration
├── types.go       # Types and Constants
├── errors.go      # Error Handling
├── metrics.go     # Metrics and Observability
├── protocol.go    # Protocol Encoding/Decoding
├── functions.go   # Modbus Functions (PDU builders)
└── version.go     # Version Information
```

## Next Steps

- [Getting Started](./getting-started)
- [Client Documentation](./client)
- [Server Documentation](./server)
- [Connection Pool](./pool)
- [Configuration](./options)
- [Error Handling](./errors)
- [Metrics](./metrics)
- [Changelog](./changelog)
