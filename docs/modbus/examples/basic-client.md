# Example: Modbus Client

A complete example of a Modbus TCP client demonstrating the main features.

## Source Code

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log/slog"
    "os"
    "time"

    "github.com/edgeo-scada/modbus-tcp/modbus"
)

func main() {
    addr := flag.String("addr", "localhost:502", "Server address")
    unitID := flag.Uint("unit", 1, "Unit ID")
    flag.Parse()

    // Setup logging
    logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))

    // Create client
    client, err := modbus.NewClient(*addr,
        modbus.WithUnitID(modbus.UnitID(*unitID)),
        modbus.WithTimeout(5*time.Second),
        modbus.WithAutoReconnect(true),
        modbus.WithLogger(logger),
    )
    if err != nil {
        fmt.Printf("Failed to create client: %v\n", err)
        os.Exit(1)
    }
    defer client.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Connect
    fmt.Printf("Connecting to %s...\n", *addr)
    if err := client.Connect(ctx); err != nil {
        fmt.Printf("Failed to connect: %v\n", err)
        os.Exit(1)
    }
    fmt.Println("Connected!")
    fmt.Println()

    // Read coils (FC01)
    fmt.Println("=== Reading Coils (FC01) ===")
    coils, err := client.ReadCoils(ctx, 0, 8)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
    } else {
        fmt.Printf("Coils[0:8]: %v\n", coils)
    }
    fmt.Println()

    // Read discrete inputs (FC02)
    fmt.Println("=== Reading Discrete Inputs (FC02) ===")
    inputs, err := client.ReadDiscreteInputs(ctx, 0, 8)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
    } else {
        fmt.Printf("Discrete Inputs[0:8]: %v\n", inputs)
    }
    fmt.Println()

    // Read holding registers (FC03)
    fmt.Println("=== Reading Holding Registers (FC03) ===")
    holdingRegs, err := client.ReadHoldingRegisters(ctx, 0, 5)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
    } else {
        fmt.Printf("Holding Registers[0:5]: %v\n", holdingRegs)
    }
    fmt.Println()

    // Read input registers (FC04)
    fmt.Println("=== Reading Input Registers (FC04) ===")
    inputRegs, err := client.ReadInputRegisters(ctx, 0, 5)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
    } else {
        fmt.Printf("Input Registers[0:5]: %v\n", inputRegs)
    }
    fmt.Println()

    // Write single coil (FC05)
    fmt.Println("=== Writing Single Coil (FC05) ===")
    fmt.Println("Writing coil[5] = true")
    if err := client.WriteSingleCoil(ctx, 5, true); err != nil {
        fmt.Printf("Error: %v\n", err)
    } else {
        fmt.Println("Success!")
    }
    fmt.Println()

    // Write single register (FC06)
    fmt.Println("=== Writing Single Register (FC06) ===")
    fmt.Println("Writing register[10] = 42")
    if err := client.WriteSingleRegister(ctx, 10, 42); err != nil {
        fmt.Printf("Error: %v\n", err)
    } else {
        fmt.Println("Success!")
    }
    fmt.Println()

    // Write multiple coils (FC15)
    fmt.Println("=== Writing Multiple Coils (FC15) ===")
    fmt.Println("Writing coils[20:25] = [true, false, true, false, true]")
    if err := client.WriteMultipleCoils(ctx, 20, []bool{true, false, true, false, true}); err != nil {
        fmt.Printf("Error: %v\n", err)
    } else {
        fmt.Println("Success!")
    }
    fmt.Println()

    // Write multiple registers (FC16)
    fmt.Println("=== Writing Multiple Registers (FC16) ===")
    fmt.Println("Writing registers[100:103] = [1111, 2222, 3333]")
    if err := client.WriteMultipleRegisters(ctx, 100, []uint16{1111, 2222, 3333}); err != nil {
        fmt.Printf("Error: %v\n", err)
    } else {
        fmt.Println("Success!")
    }
    fmt.Println()

    // Diagnostics - Echo (FC08)
    fmt.Println("=== Diagnostics - Echo (FC08) ===")
    echoData := []byte{0x12, 0x34}
    respData, err := client.Diagnostics(ctx, modbus.DiagReturnQueryData, echoData)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
    } else {
        fmt.Printf("Echo response: %x\n", respData)
    }
    fmt.Println()

    // Report Server ID (FC17)
    fmt.Println("=== Report Server ID (FC17) ===")
    serverID, err := client.ReportServerID(ctx)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
    } else {
        fmt.Printf("Server ID: %s\n", string(serverID))
    }
    fmt.Println()

    // Print metrics
    fmt.Println("=== Client Metrics ===")
    metrics := client.Metrics().Collect()
    fmt.Printf("Total requests: %v\n", metrics["requests_total"])
    fmt.Printf("Successful requests: %v\n", metrics["requests_success"])
    fmt.Printf("Failed requests: %v\n", metrics["requests_errors"])
    if latency, ok := metrics["latency"].(modbus.LatencyStats); ok {
        fmt.Printf("Average latency: %.2f ms\n", latency.Avg)
        fmt.Printf("Min latency: %.2f ms\n", latency.Min)
        fmt.Printf("Max latency: %.2f ms\n", latency.Max)
    }

    fmt.Println("\nDone!")
}
```

## Running

```bash
# With the default server on localhost:502
go run main.go

# With a specific server
go run main.go -addr 192.168.1.100:502

# With a different Unit ID
go run main.go -addr 192.168.1.100:502 -unit 2
```

## Example Output

```
Connecting to localhost:502...
Connected!

=== Reading Coils (FC01) ===
Coils[0:8]: [true false true false false false false false]

=== Reading Discrete Inputs (FC02) ===
Discrete Inputs[0:8]: [true true false false false false false false]

=== Reading Holding Registers (FC03) ===
Holding Registers[0:5]: [1234 5678 9012 0 0]

=== Reading Input Registers (FC04) ===
Input Registers[0:5]: [100 200 0 0 0]

=== Writing Single Coil (FC05) ===
Writing coil[5] = true
Success!

=== Writing Single Register (FC06) ===
Writing register[10] = 42
Success!

=== Writing Multiple Coils (FC15) ===
Writing coils[20:25] = [true, false, true, false, true]
Success!

=== Writing Multiple Registers (FC16) ===
Writing registers[100:103] = [1111, 2222, 3333]
Success!

=== Diagnostics - Echo (FC08) ===
Echo response: 1234

=== Report Server ID (FC17) ===
Server ID: Edgeo Modbus Server v1.0

=== Client Metrics ===
Total requests: 12
Successful requests: 12
Failed requests: 0
Average latency: 0.45 ms
Min latency: 0.21 ms
Max latency: 1.23 ms

Done!
```
