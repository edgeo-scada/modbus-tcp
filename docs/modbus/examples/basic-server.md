# Exemple: Serveur Modbus

Un exemple complet de serveur Modbus TCP avec le handler en mémoire.

## Code source

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log/slog"
    "os"
    "os/signal"
    "syscall"

    "github.com/edgeo/drivers/modbus"
)

func main() {
    addr := flag.String("addr", ":502", "Server address")
    flag.Parse()

    // Setup logging
    logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    }))

    // Create memory handler with some initial data
    handler := modbus.NewMemoryHandler(65536, 65536)

    // Set some initial values for testing
    unitID := modbus.UnitID(1)

    // Set coils
    handler.SetCoil(unitID, 0, true)
    handler.SetCoil(unitID, 1, false)
    handler.SetCoil(unitID, 2, true)

    // Set discrete inputs
    handler.SetDiscreteInput(unitID, 0, true)
    handler.SetDiscreteInput(unitID, 1, true)

    // Set holding registers
    handler.SetHoldingRegister(unitID, 0, 1234)
    handler.SetHoldingRegister(unitID, 1, 5678)
    handler.SetHoldingRegister(unitID, 2, 9012)

    // Set input registers
    handler.SetInputRegister(unitID, 0, 100)
    handler.SetInputRegister(unitID, 1, 200)

    // Set server ID
    handler.SetServerID([]byte("Edgeo Modbus Server v1.0"))

    // Create server
    server := modbus.NewServer(handler,
        modbus.WithServerLogger(logger),
        modbus.WithMaxConnections(10),
    )

    // Handle shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-sigCh
        fmt.Println("\nShutting down...")
        cancel()
        server.Close()
    }()

    // Start server
    fmt.Printf("Starting Modbus TCP server on %s\n", *addr)
    fmt.Println("Press Ctrl+C to stop")
    fmt.Println("\nInitial data:")
    fmt.Printf("  Coils[0:3]: true, false, true\n")
    fmt.Printf("  Discrete Inputs[0:2]: true, true\n")
    fmt.Printf("  Holding Registers[0:3]: 1234, 5678, 9012\n")
    fmt.Printf("  Input Registers[0:2]: 100, 200\n")
    fmt.Println()

    if err := server.ListenAndServeContext(ctx, *addr); err != nil {
        logger.Error("server error", slog.String("error", err.Error()))
        os.Exit(1)
    }
}
```

## Exécution

```bash
# Port par défaut (502) - nécessite root/admin
sudo go run main.go

# Port alternatif (ne nécessite pas root)
go run main.go -addr :1502
```

## Handler personnalisé

Exemple d'un handler qui simule un capteur de température:

```go
package main

import (
    "math/rand"
    "sync"
    "time"

    "github.com/edgeo/drivers/modbus"
)

type TemperatureSensorHandler struct {
    mu          sync.RWMutex
    temperature float64
    humidity    float64
    lastUpdate  time.Time
}

func NewTemperatureSensorHandler() *TemperatureSensorHandler {
    h := &TemperatureSensorHandler{
        temperature: 20.0,
        humidity:    50.0,
    }

    // Simuler des variations de température
    go func() {
        for {
            time.Sleep(1 * time.Second)
            h.mu.Lock()
            h.temperature += (rand.Float64() - 0.5) * 0.5
            h.humidity += (rand.Float64() - 0.5) * 1.0
            if h.humidity < 0 {
                h.humidity = 0
            }
            if h.humidity > 100 {
                h.humidity = 100
            }
            h.lastUpdate = time.Now()
            h.mu.Unlock()
        }
    }()

    return h
}

// ReadInputRegisters retourne la température et l'humidité
// Registre 0: Température * 100 (ex: 2050 = 20.50°C)
// Registre 1: Humidité * 100 (ex: 5025 = 50.25%)
func (h *TemperatureSensorHandler) ReadInputRegisters(unitID modbus.UnitID, addr, qty uint16) ([]uint16, error) {
    if addr > 1 || addr+qty > 2 {
        return nil, modbus.NewModbusError(
            modbus.FuncReadInputRegisters,
            modbus.ExceptionIllegalDataAddress,
        )
    }

    h.mu.RLock()
    defer h.mu.RUnlock()

    values := make([]uint16, qty)
    for i := uint16(0); i < qty; i++ {
        switch addr + i {
        case 0:
            values[i] = uint16(h.temperature * 100)
        case 1:
            values[i] = uint16(h.humidity * 100)
        }
    }
    return values, nil
}

// ReadHoldingRegisters - Configuration (lecture/écriture)
// Registre 0: Intervalle de mise à jour (secondes)
func (h *TemperatureSensorHandler) ReadHoldingRegisters(unitID modbus.UnitID, addr, qty uint16) ([]uint16, error) {
    if addr > 0 || addr+qty > 1 {
        return nil, modbus.NewModbusError(
            modbus.FuncReadHoldingRegisters,
            modbus.ExceptionIllegalDataAddress,
        )
    }
    return []uint16{1}, nil // Intervalle fixe de 1 seconde
}

// Les autres méthodes retournent des erreurs "fonction non supportée"
func (h *TemperatureSensorHandler) ReadCoils(unitID modbus.UnitID, addr, qty uint16) ([]bool, error) {
    return nil, modbus.NewModbusError(modbus.FuncReadCoils, modbus.ExceptionIllegalFunction)
}

func (h *TemperatureSensorHandler) ReadDiscreteInputs(unitID modbus.UnitID, addr, qty uint16) ([]bool, error) {
    return nil, modbus.NewModbusError(modbus.FuncReadDiscreteInputs, modbus.ExceptionIllegalFunction)
}

func (h *TemperatureSensorHandler) WriteSingleCoil(unitID modbus.UnitID, addr uint16, value bool) error {
    return modbus.NewModbusError(modbus.FuncWriteSingleCoil, modbus.ExceptionIllegalFunction)
}

func (h *TemperatureSensorHandler) WriteMultipleCoils(unitID modbus.UnitID, addr uint16, values []bool) error {
    return modbus.NewModbusError(modbus.FuncWriteMultipleCoils, modbus.ExceptionIllegalFunction)
}

func (h *TemperatureSensorHandler) WriteSingleRegister(unitID modbus.UnitID, addr, value uint16) error {
    return modbus.NewModbusError(modbus.FuncWriteSingleRegister, modbus.ExceptionIllegalFunction)
}

func (h *TemperatureSensorHandler) WriteMultipleRegisters(unitID modbus.UnitID, addr uint16, values []uint16) error {
    return modbus.NewModbusError(modbus.FuncWriteMultipleRegisters, modbus.ExceptionIllegalFunction)
}

func (h *TemperatureSensorHandler) ReadExceptionStatus(unitID modbus.UnitID) (uint8, error) {
    return 0, nil
}

func (h *TemperatureSensorHandler) Diagnostics(unitID modbus.UnitID, subFunc uint16, data []byte) ([]byte, error) {
    if subFunc == modbus.DiagReturnQueryData {
        result := make([]byte, len(data))
        copy(result, data)
        return result, nil
    }
    return nil, modbus.NewModbusError(modbus.FuncDiagnostics, modbus.ExceptionIllegalFunction)
}

func (h *TemperatureSensorHandler) GetCommEventCounter(unitID modbus.UnitID) (uint16, uint16, error) {
    return 0xFFFF, 0, nil
}

func (h *TemperatureSensorHandler) ReportServerID(unitID modbus.UnitID) ([]byte, error) {
    return []byte("Temperature Sensor v1.0"), nil
}

func main() {
    handler := NewTemperatureSensorHandler()
    server := modbus.NewServer(handler)
    server.ListenAndServe(":502")
}
```

## Test avec le client

```bash
# Dans un terminal, lancer le serveur
go run server/main.go -addr :1502

# Dans un autre terminal, lancer le client
go run client/main.go -addr localhost:1502
```
