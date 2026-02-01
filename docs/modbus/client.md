# Client Modbus TCP

Le client Modbus TCP permet de communiquer avec des équipements Modbus (automates, capteurs, etc.).

## Création

```go
client, err := modbus.NewClient(addr string, opts ...Option) (*Client, error)
```

**Paramètres:**
- `addr`: Adresse du serveur Modbus (ex: `"192.168.1.100:502"`)
- `opts`: Options de configuration (voir [Options](./options.md))

**Exemple:**
```go
client, err := modbus.NewClient("192.168.1.100:502",
    modbus.WithUnitID(1),
    modbus.WithTimeout(5*time.Second),
    modbus.WithAutoReconnect(true),
)
```

## Connexion et déconnexion

### Connect

```go
func (c *Client) Connect(ctx context.Context) error
```

Établit la connexion TCP avec le serveur.

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

Ferme la connexion proprement.

```go
defer client.Close()
```

### État de connexion

```go
func (c *Client) IsConnected() bool
func (c *Client) State() ConnectionState
```

États possibles: `StateDisconnected`, `StateConnecting`, `StateConnected`

## Lecture de données

### ReadCoils (FC01)

Lit les coils (bits de sortie).

```go
func (c *Client) ReadCoils(ctx context.Context, addr, qty uint16) ([]bool, error)
```

| Paramètre | Description |
|-----------|-------------|
| `addr` | Adresse de départ (0-65535) |
| `qty` | Nombre de coils (1-2000) |

```go
coils, err := client.ReadCoils(ctx, 0, 16)
// coils = []bool{true, false, true, ...}
```

### ReadDiscreteInputs (FC02)

Lit les entrées discrètes (bits d'entrée).

```go
func (c *Client) ReadDiscreteInputs(ctx context.Context, addr, qty uint16) ([]bool, error)
```

```go
inputs, err := client.ReadDiscreteInputs(ctx, 0, 8)
```

### ReadHoldingRegisters (FC03)

Lit les registres de maintien (lecture/écriture).

```go
func (c *Client) ReadHoldingRegisters(ctx context.Context, addr, qty uint16) ([]uint16, error)
```

| Paramètre | Description |
|-----------|-------------|
| `addr` | Adresse de départ (0-65535) |
| `qty` | Nombre de registres (1-125) |

```go
regs, err := client.ReadHoldingRegisters(ctx, 100, 10)
// regs = []uint16{1234, 5678, ...}
```

### ReadInputRegisters (FC04)

Lit les registres d'entrée (lecture seule).

```go
func (c *Client) ReadInputRegisters(ctx context.Context, addr, qty uint16) ([]uint16, error)
```

```go
inputs, err := client.ReadInputRegisters(ctx, 0, 5)
```

## Écriture de données

### WriteSingleCoil (FC05)

Écrit un seul coil.

```go
func (c *Client) WriteSingleCoil(ctx context.Context, addr uint16, value bool) error
```

```go
err := client.WriteSingleCoil(ctx, 10, true)  // Active le coil 10
err = client.WriteSingleCoil(ctx, 10, false)  // Désactive le coil 10
```

### WriteSingleRegister (FC06)

Écrit un seul registre.

```go
func (c *Client) WriteSingleRegister(ctx context.Context, addr, value uint16) error
```

```go
err := client.WriteSingleRegister(ctx, 100, 1234)
```

### WriteMultipleCoils (FC15)

Écrit plusieurs coils.

```go
func (c *Client) WriteMultipleCoils(ctx context.Context, addr uint16, values []bool) error
```

```go
err := client.WriteMultipleCoils(ctx, 0, []bool{true, false, true, true})
```

### WriteMultipleRegisters (FC16)

Écrit plusieurs registres.

```go
func (c *Client) WriteMultipleRegisters(ctx context.Context, addr uint16, values []uint16) error
```

```go
err := client.WriteMultipleRegisters(ctx, 100, []uint16{1111, 2222, 3333})
```

## Fonctions de diagnostic

### ReadExceptionStatus (FC07)

```go
func (c *Client) ReadExceptionStatus(ctx context.Context) (uint8, error)
```

### Diagnostics (FC08)

```go
func (c *Client) Diagnostics(ctx context.Context, subFunc uint16, data []byte) ([]byte, error)
```

Sous-fonctions disponibles:
- `DiagReturnQueryData` (0x00): Echo
- `DiagRestartCommunications` (0x01)
- `DiagReturnDiagnosticRegister` (0x02)
- etc.

```go
// Test d'écho
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

## Opérations avec Unit ID spécifique

Toutes les méthodes ont une variante `WithUnit` pour spécifier un Unit ID différent:

```go
// Utilise le Unit ID par défaut
regs, _ := client.ReadHoldingRegisters(ctx, 0, 10)

// Utilise un Unit ID spécifique
regs, _ := client.ReadHoldingRegistersWithUnit(ctx, modbus.UnitID(2), 0, 10)
```

Méthodes disponibles:
- `ReadCoilsWithUnit`
- `ReadDiscreteInputsWithUnit`
- `ReadHoldingRegistersWithUnit`
- `ReadInputRegistersWithUnit`
- `WriteSingleCoilWithUnit`
- `WriteSingleRegisterWithUnit`
- `WriteMultipleCoilsWithUnit`
- `WriteMultipleRegistersWithUnit`

## Gestion du Unit ID

```go
// Définir le Unit ID par défaut
client.SetUnitID(modbus.UnitID(2))

// Récupérer le Unit ID actuel
unitID := client.UnitID()
```

## Métriques

```go
metrics := client.Metrics().Collect()
fmt.Printf("Requêtes totales: %v\n", metrics["requests_total"])
fmt.Printf("Requêtes réussies: %v\n", metrics["requests_success"])
fmt.Printf("Erreurs: %v\n", metrics["requests_errors"])
```

Voir [Métriques](./metrics.md) pour plus de détails.
