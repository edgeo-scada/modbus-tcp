# Getting Started

## Prerequisites

- Go 1.21 or higher

## Installation

```bash
go get github.com/edgeo-scada/modbus-tcp/modbus
```

## Modbus Client

### Basic Connection

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/edgeo-scada/modbus-tcp/modbus"
)

func main() {
    // Create the client
    client, err := modbus.NewClient("192.168.1.100:502",
        modbus.WithUnitID(1),
        modbus.WithTimeout(5*time.Second),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Connect
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }

    fmt.Println("Connected!")
}
```

### Reading Registers

```go
// Read 10 holding registers starting from address 0
regs, err := client.ReadHoldingRegisters(ctx, 0, 10)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Registers: %v\n", regs)

// Read 8 coils starting from address 0
coils, err := client.ReadCoils(ctx, 0, 8)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Coils: %v\n", coils)
```

### Writing Registers

```go
// Write a single register
err := client.WriteSingleRegister(ctx, 100, 1234)
if err != nil {
    log.Fatal(err)
}

// Write multiple registers
err = client.WriteMultipleRegisters(ctx, 100, []uint16{1111, 2222, 3333})
if err != nil {
    log.Fatal(err)
}

// Write a coil
err = client.WriteSingleCoil(ctx, 0, true)
if err != nil {
    log.Fatal(err)
}
```

## Modbus Server

### Server with MemoryHandler

```go
package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"

    "github.com/edgeo-scada/modbus-tcp/modbus"
)

func main() {
    // Create a memory handler
    handler := modbus.NewMemoryHandler(65536, 65536)

    // Initialize some data
    handler.SetHoldingRegister(1, 0, 1234)
    handler.SetCoil(1, 0, true)

    // Create the server
    server := modbus.NewServer(handler,
        modbus.WithMaxConnections(100),
    )

    // Graceful shutdown handling
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-sigCh
        fmt.Println("Shutting down...")
        server.Close()
    }()

    // Start the server
    fmt.Println("Modbus server on :502")
    if err := server.ListenAndServeContext(ctx, ":502"); err != nil {
        fmt.Printf("Error: %v\n", err)
    }
}
```

## Connection Pool

For high-performance applications:

```go
// Create a pool
pool, err := modbus.NewPool("192.168.1.100:502",
    modbus.WithSize(10),
    modbus.WithMaxIdleTime(5*time.Minute),
    modbus.WithClientOptions(
        modbus.WithTimeout(5*time.Second),
    ),
)
if err != nil {
    log.Fatal(err)
}
defer pool.Close()

// Get a connection from the pool
client, err := pool.Get(ctx)
if err != nil {
    log.Fatal(err)
}

regs, err := client.ReadHoldingRegisters(ctx, 0, 10)
// ...

// Return the connection to the pool
pool.Put(client)
```

Or with automatic return:

```go
pc, err := pool.GetPooled(ctx)
if err != nil {
    log.Fatal(err)
}
defer pc.Close() // Automatically returns to the pool

regs, err := pc.ReadHoldingRegisters(ctx, 0, 10)
```
