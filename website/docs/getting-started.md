# Démarrage rapide

## Prérequis

- Go 1.21 ou supérieur

## Installation

```bash
go get github.com/edgeo/drivers/modbus
```

## Client Modbus

### Connexion basique

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
    // Créer le client
    client, err := modbus.NewClient("192.168.1.100:502",
        modbus.WithUnitID(1),
        modbus.WithTimeout(5*time.Second),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Connexion
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := client.Connect(ctx); err != nil {
        log.Fatal(err)
    }

    fmt.Println("Connecté!")
}
```

### Lecture de registres

```go
// Lire 10 holding registers à partir de l'adresse 0
regs, err := client.ReadHoldingRegisters(ctx, 0, 10)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Registres: %v\n", regs)

// Lire 8 coils à partir de l'adresse 0
coils, err := client.ReadCoils(ctx, 0, 8)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Coils: %v\n", coils)
```

### Écriture de registres

```go
// Écrire un registre
err := client.WriteSingleRegister(ctx, 100, 1234)
if err != nil {
    log.Fatal(err)
}

// Écrire plusieurs registres
err = client.WriteMultipleRegisters(ctx, 100, []uint16{1111, 2222, 3333})
if err != nil {
    log.Fatal(err)
}

// Écrire un coil
err = client.WriteSingleCoil(ctx, 0, true)
if err != nil {
    log.Fatal(err)
}
```

## Serveur Modbus

### Serveur avec MemoryHandler

```go
package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"

    "github.com/edgeo/drivers/modbus"
)

func main() {
    // Créer un handler en mémoire
    handler := modbus.NewMemoryHandler(65536, 65536)

    // Initialiser des données
    handler.SetHoldingRegister(1, 0, 1234)
    handler.SetCoil(1, 0, true)

    // Créer le serveur
    server := modbus.NewServer(handler,
        modbus.WithMaxConnections(100),
    )

    // Gestion de l'arrêt gracieux
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-sigCh
        fmt.Println("Arrêt...")
        server.Close()
    }()

    // Démarrer le serveur
    fmt.Println("Serveur Modbus sur :502")
    if err := server.ListenAndServeContext(ctx, ":502"); err != nil {
        fmt.Printf("Erreur: %v\n", err)
    }
}
```

## Pool de connexions

Pour les applications à haute performance:

```go
// Créer un pool
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

// Utiliser une connexion du pool
client, err := pool.Get(ctx)
if err != nil {
    log.Fatal(err)
}

regs, err := client.ReadHoldingRegisters(ctx, 0, 10)
// ...

// Remettre la connexion dans le pool
pool.Put(client)
```

Ou avec retour automatique:

```go
pc, err := pool.GetPooled(ctx)
if err != nil {
    log.Fatal(err)
}
defer pc.Close() // Remet automatiquement dans le pool

regs, err := pc.ReadHoldingRegisters(ctx, 0, 10)
```
