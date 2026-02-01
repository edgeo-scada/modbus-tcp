# Configuration

La configuration utilise le pattern des options fonctionnelles (functional options).

## Options Client

### WithUnitID

Définit l'Unit ID par défaut pour les requêtes.

```go
modbus.WithUnitID(id UnitID)
```

```go
client, _ := modbus.NewClient("localhost:502",
    modbus.WithUnitID(1),
)
```

L'Unit ID identifie l'équipement cible sur le réseau Modbus. Valeurs: 1-247.

### WithTimeout

Définit le timeout pour les opérations.

```go
modbus.WithTimeout(d time.Duration)
```

```go
client, _ := modbus.NewClient("localhost:502",
    modbus.WithTimeout(5*time.Second),
)
```

**Défaut:** 5 secondes

### WithAutoReconnect

Active la reconnexion automatique en cas de perte de connexion.

```go
modbus.WithAutoReconnect(enable bool)
```

```go
client, _ := modbus.NewClient("localhost:502",
    modbus.WithAutoReconnect(true),
)
```

**Défaut:** false

### WithReconnectBackoff

Définit le délai initial entre les tentatives de reconnexion.

```go
modbus.WithReconnectBackoff(d time.Duration)
```

Le backoff augmente exponentiellement jusqu'à `MaxReconnectTime`.

**Défaut:** 1 seconde

### WithMaxReconnectTime

Définit le délai maximum entre les tentatives de reconnexion.

```go
modbus.WithMaxReconnectTime(d time.Duration)
```

**Défaut:** 30 secondes

### WithMaxRetries

Définit le nombre maximum de tentatives pour une requête.

```go
modbus.WithMaxRetries(n int)
```

```go
client, _ := modbus.NewClient("localhost:502",
    modbus.WithAutoReconnect(true),
    modbus.WithMaxRetries(5),
)
```

**Défaut:** 3

### WithOnConnect

Définit un callback appelé lors de la connexion.

```go
modbus.WithOnConnect(fn func())
```

```go
client, _ := modbus.NewClient("localhost:502",
    modbus.WithOnConnect(func() {
        log.Println("Connecté!")
    }),
)
```

### WithOnDisconnect

Définit un callback appelé lors de la déconnexion.

```go
modbus.WithOnDisconnect(fn func(error))
```

```go
client, _ := modbus.NewClient("localhost:502",
    modbus.WithOnDisconnect(func(err error) {
        log.Printf("Déconnecté: %v\n", err)
    }),
)
```

### WithLogger

Définit le logger pour le client.

```go
modbus.WithLogger(logger *slog.Logger)
```

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

client, _ := modbus.NewClient("localhost:502",
    modbus.WithLogger(logger),
)
```

**Défaut:** `slog.Default()`

## Options Serveur

### WithServerLogger

Définit le logger pour le serveur.

```go
modbus.WithServerLogger(logger *slog.Logger)
```

### WithMaxConnections

Définit le nombre maximum de connexions simultanées.

```go
modbus.WithMaxConnections(n int)
```

```go
server := modbus.NewServer(handler,
    modbus.WithMaxConnections(100),
)
```

**Défaut:** 100

### WithReadTimeout

Définit le timeout de lecture pour les connexions client.

```go
modbus.WithReadTimeout(d time.Duration)
```

Les connexions inactives plus longtemps que ce timeout sont fermées.

**Défaut:** 30 secondes

## Options Pool

### WithSize

Définit la taille maximale du pool.

```go
modbus.WithSize(size int)
```

```go
pool, _ := modbus.NewPool("localhost:502",
    modbus.WithSize(20),
)
```

**Défaut:** 5

### WithMaxIdleTime

Définit la durée maximale d'inactivité avant fermeture d'une connexion.

```go
modbus.WithMaxIdleTime(d time.Duration)
```

```go
pool, _ := modbus.NewPool("localhost:502",
    modbus.WithMaxIdleTime(10*time.Minute),
)
```

**Défaut:** 5 minutes

### WithHealthCheckFrequency

Définit la fréquence des vérifications de santé des connexions.

```go
modbus.WithHealthCheckFrequency(d time.Duration)
```

```go
pool, _ := modbus.NewPool("localhost:502",
    modbus.WithHealthCheckFrequency(30*time.Second),
)
```

**Défaut:** 1 minute. Mettre à 0 pour désactiver.

### WithClientOptions

Définit les options à utiliser pour créer les clients du pool.

```go
modbus.WithClientOptions(opts ...Option)
```

```go
pool, _ := modbus.NewPool("localhost:502",
    modbus.WithSize(10),
    modbus.WithClientOptions(
        modbus.WithTimeout(3*time.Second),
        modbus.WithUnitID(1),
    ),
)
```

## Exemple complet

```go
package main

import (
    "log/slog"
    "os"
    "time"

    "github.com/edgeo/drivers/modbus"
)

func main() {
    // Logger personnalisé
    logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))

    // Client avec toutes les options
    client, _ := modbus.NewClient("192.168.1.100:502",
        // Identification
        modbus.WithUnitID(1),

        // Timeouts
        modbus.WithTimeout(5*time.Second),

        // Reconnexion
        modbus.WithAutoReconnect(true),
        modbus.WithReconnectBackoff(500*time.Millisecond),
        modbus.WithMaxReconnectTime(30*time.Second),
        modbus.WithMaxRetries(5),

        // Callbacks
        modbus.WithOnConnect(func() {
            logger.Info("connected")
        }),
        modbus.WithOnDisconnect(func(err error) {
            logger.Warn("disconnected", slog.String("error", err.Error()))
        }),

        // Logging
        modbus.WithLogger(logger),
    )
    defer client.Close()
}
```
