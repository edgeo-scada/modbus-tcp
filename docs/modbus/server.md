# Serveur Modbus TCP

Le serveur Modbus TCP permet de simuler un équipement Modbus ou de créer une passerelle.

## Création

```go
server := modbus.NewServer(handler Handler, opts ...ServerOption) *Server
```

**Paramètres:**
- `handler`: Implémentation de l'interface `Handler`
- `opts`: Options de configuration

## Interface Handler

Pour créer un serveur, vous devez implémenter l'interface `Handler`:

```go
type Handler interface {
    // Opérations sur les coils
    ReadCoils(unitID UnitID, addr, qty uint16) ([]bool, error)
    ReadDiscreteInputs(unitID UnitID, addr, qty uint16) ([]bool, error)
    WriteSingleCoil(unitID UnitID, addr uint16, value bool) error
    WriteMultipleCoils(unitID UnitID, addr uint16, values []bool) error

    // Opérations sur les registres
    ReadHoldingRegisters(unitID UnitID, addr, qty uint16) ([]uint16, error)
    ReadInputRegisters(unitID UnitID, addr, qty uint16) ([]uint16, error)
    WriteSingleRegister(unitID UnitID, addr, value uint16) error
    WriteMultipleRegisters(unitID UnitID, addr uint16, values []uint16) error

    // Opérations de diagnostic
    ReadExceptionStatus(unitID UnitID) (uint8, error)
    Diagnostics(unitID UnitID, subFunc uint16, data []byte) ([]byte, error)
    GetCommEventCounter(unitID UnitID) (status uint16, eventCount uint16, err error)
    ReportServerID(unitID UnitID) ([]byte, error)
}
```

## MemoryHandler

Un handler en mémoire est fourni pour les tests et simulations:

```go
handler := modbus.NewMemoryHandler(coilSize, registerSize int)
```

### Initialisation des données

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
handler.SetServerID([]byte("Mon Serveur Modbus v1.0"))
```

## Démarrage du serveur

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

Avec support d'annulation via context:

```go
func (s *Server) ListenAndServeContext(ctx context.Context, addr string) error
```

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Arrêt gracieux sur signal
go func() {
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh
    cancel()
}()

server.ListenAndServeContext(ctx, ":502")
```

### Serve

Avec un listener personnalisé:

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

## Arrêt du serveur

```go
func (s *Server) Close() error
```

Ferme proprement toutes les connexions actives.

## Informations sur le serveur

```go
// Adresse du serveur
addr := server.Addr()

// Nombre de connexions actives
count := server.ActiveConnections()
```

## Options du serveur

```go
server := modbus.NewServer(handler,
    modbus.WithServerLogger(logger),      // Logger personnalisé
    modbus.WithMaxConnections(100),       // Max connexions simultanées
    modbus.WithReadTimeout(30*time.Second), // Timeout de lecture
)
```

Voir [Options](./options.md) pour la liste complète.

## Métriques serveur

```go
type ServerMetrics struct {
    RequestsTotal   Counter  // Total des requêtes reçues
    RequestsSuccess Counter  // Requêtes traitées avec succès
    RequestsErrors  Counter  // Requêtes en erreur
    ActiveConns     Counter  // Connexions actives
    TotalConns      Counter  // Total des connexions reçues
}
```

```go
metrics := server.Metrics()
fmt.Printf("Connexions actives: %d\n", metrics.ActiveConns.Value())
fmt.Printf("Total requêtes: %d\n", metrics.RequestsTotal.Value())
```

## Handler personnalisé

Exemple d'un handler personnalisé qui connecte à une base de données:

```go
type DBHandler struct {
    db *sql.DB
}

func (h *DBHandler) ReadHoldingRegisters(unitID modbus.UnitID, addr, qty uint16) ([]uint16, error) {
    // Lire depuis la base de données
    rows, err := h.db.Query("SELECT value FROM registers WHERE unit_id = ? AND addr >= ? AND addr < ?",
        unitID, addr, addr+qty)
    if err != nil {
        return nil, modbus.NewModbusError(modbus.FuncReadHoldingRegisters, modbus.ExceptionServerDeviceFailure)
    }
    defer rows.Close()

    values := make([]uint16, qty)
    // ... remplir values
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

// Implémenter les autres méthodes...
```

## Retourner des erreurs Modbus

Utilisez `NewModbusError` pour retourner des exceptions Modbus standard:

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

Exceptions disponibles:
- `ExceptionIllegalFunction` (0x01)
- `ExceptionIllegalDataAddress` (0x02)
- `ExceptionIllegalDataValue` (0x03)
- `ExceptionServerDeviceFailure` (0x04)
- `ExceptionServerDeviceBusy` (0x06)
