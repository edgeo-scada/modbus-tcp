# Gestion des erreurs

## Erreurs standard

Le package définit plusieurs erreurs sentinelles:

| Erreur | Description |
|--------|-------------|
| `ErrInvalidResponse` | Réponse malformée ou inattendue |
| `ErrInvalidCRC` | Validation CRC échouée (mode RTU) |
| `ErrInvalidFrame` | Trame malformée |
| `ErrTimeout` | Timeout dépassé |
| `ErrConnectionClosed` | Connexion fermée |
| `ErrInvalidQuantity` | Quantité invalide (hors limites) |
| `ErrInvalidAddress` | Adresse invalide |
| `ErrPoolExhausted` | Pool de connexions épuisé |
| `ErrPoolClosed` | Pool de connexions fermé |
| `ErrNotConnected` | Client non connecté |
| `ErrMaxRetriesExceeded` | Nombre max de tentatives dépassé |

### Vérification des erreurs

```go
import "errors"

regs, err := client.ReadHoldingRegisters(ctx, 0, 10)
if err != nil {
    if errors.Is(err, modbus.ErrNotConnected) {
        // Reconnecter
        client.Connect(ctx)
    } else if errors.Is(err, modbus.ErrTimeout) {
        // Réessayer
    } else if errors.Is(err, modbus.ErrMaxRetriesExceeded) {
        // Abandon
    }
}
```

## Erreurs Modbus (Exceptions)

Les erreurs du protocole Modbus sont représentées par `ModbusError`:

```go
type ModbusError struct {
    FunctionCode  FunctionCode
    ExceptionCode ExceptionCode
}
```

### Codes d'exception

| Code | Constante | Description |
|------|-----------|-------------|
| 0x01 | `ExceptionIllegalFunction` | Fonction non supportée |
| 0x02 | `ExceptionIllegalDataAddress` | Adresse invalide |
| 0x03 | `ExceptionIllegalDataValue` | Valeur invalide |
| 0x04 | `ExceptionServerDeviceFailure` | Erreur interne serveur |
| 0x05 | `ExceptionAcknowledge` | Requête acceptée, traitement en cours |
| 0x06 | `ExceptionServerDeviceBusy` | Serveur occupé |
| 0x08 | `ExceptionMemoryParityError` | Erreur de parité mémoire |
| 0x0A | `ExceptionGatewayPathUnavailable` | Passerelle indisponible |
| 0x0B | `ExceptionGatewayTargetDeviceFailedToRespond` | Équipement cible ne répond pas |

### Vérification des exceptions

```go
regs, err := client.ReadHoldingRegisters(ctx, 1000, 10)
if err != nil {
    // Vérifier si c'est une exception Modbus
    var modbusErr *modbus.ModbusError
    if errors.As(err, &modbusErr) {
        switch modbusErr.ExceptionCode {
        case modbus.ExceptionIllegalDataAddress:
            log.Println("Adresse invalide")
        case modbus.ExceptionIllegalDataValue:
            log.Println("Valeur invalide")
        case modbus.ExceptionServerDeviceBusy:
            log.Println("Serveur occupé, réessayer plus tard")
        default:
            log.Printf("Exception Modbus: %v\n", modbusErr)
        }
    }
}
```

### Fonctions utilitaires

```go
// Vérifier une exception spécifique
if modbus.IsException(err, modbus.ExceptionIllegalDataAddress) {
    // Adresse invalide
}

// Raccourcis pour les exceptions courantes
if modbus.IsIllegalFunction(err) {
    // Fonction non supportée
}

if modbus.IsIllegalDataAddress(err) {
    // Adresse invalide
}

if modbus.IsIllegalDataValue(err) {
    // Valeur invalide
}

if modbus.IsServerDeviceFailure(err) {
    // Erreur serveur
}
```

## Créer des erreurs Modbus (côté serveur)

Dans votre Handler, retournez des erreurs Modbus:

```go
func (h *MyHandler) ReadHoldingRegisters(unitID modbus.UnitID, addr, qty uint16) ([]uint16, error) {
    // Vérifier l'adresse
    if addr >= 10000 {
        return nil, modbus.NewModbusError(
            modbus.FuncReadHoldingRegisters,
            modbus.ExceptionIllegalDataAddress,
        )
    }

    // Vérifier la quantité
    if qty > 125 {
        return nil, modbus.NewModbusError(
            modbus.FuncReadHoldingRegisters,
            modbus.ExceptionIllegalDataValue,
        )
    }

    // Erreur interne
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

## Erreurs de connexion

### Gestion avec reconnexion automatique

```go
client, _ := modbus.NewClient("localhost:502",
    modbus.WithAutoReconnect(true),
    modbus.WithMaxRetries(5),
    modbus.WithOnDisconnect(func(err error) {
        log.Printf("Déconnexion: %v\n", err)
    }),
)

// Les erreurs réseau sont automatiquement gérées avec retry
regs, err := client.ReadHoldingRegisters(ctx, 0, 10)
if err != nil {
    // Après 5 tentatives échouées
    if errors.Is(err, modbus.ErrMaxRetriesExceeded) {
        log.Fatal("Impossible de joindre le serveur")
    }
}
```

### Gestion manuelle

```go
client, _ := modbus.NewClient("localhost:502",
    modbus.WithAutoReconnect(false),
)

regs, err := client.ReadHoldingRegisters(ctx, 0, 10)
if err != nil {
    if errors.Is(err, modbus.ErrNotConnected) {
        // Reconnexion manuelle
        if err := client.Connect(ctx); err != nil {
            log.Fatal(err)
        }
        // Réessayer
        regs, err = client.ReadHoldingRegisters(ctx, 0, 10)
    }
}
```

## Bonnes pratiques

1. **Toujours vérifier les erreurs** - Ne jamais ignorer les erreurs retournées

2. **Distinguer les types d'erreurs** - Les erreurs Modbus (protocol) vs erreurs réseau

3. **Logger les erreurs** - Pour le debugging et monitoring

4. **Utiliser les timeouts** - Éviter les blocages indéfinis

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

regs, err := client.ReadHoldingRegisters(ctx, 0, 10)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        log.Println("Timeout - le serveur ne répond pas")
    }
}
```
