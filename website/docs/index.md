---
slug: /
---

# Modbus TCP Driver

[![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)](./changelog)
[![Go](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](https://github.com/edgeo/drivers/blob/main/LICENSE)

Une implémentation Go complète du protocole Modbus TCP, avec client, serveur et pool de connexions.

## Installation

```bash
go get github.com/edgeo/drivers/modbus@v1.0.0
```

Pour vérifier la version installée:

```go
import "github.com/edgeo/drivers/modbus"

func main() {
    fmt.Printf("Modbus driver version: %s\n", modbus.Version)
    // Output: Modbus driver version: 1.0.0
}
```

## Fonctionnalités

- **Client Modbus TCP** avec reconnexion automatique
- **Serveur Modbus TCP** avec support multi-clients
- **Pool de connexions** avec health checks
- **Métriques** intégrées (latence, compteurs, histogrammes)
- **Logging** structuré via `slog`

## Fonctions Modbus supportées

| Code | Fonction | Description |
|------|----------|-------------|
| FC01 | Read Coils | Lecture de bits de sortie |
| FC02 | Read Discrete Inputs | Lecture d'entrées discrètes |
| FC03 | Read Holding Registers | Lecture de registres de maintien |
| FC04 | Read Input Registers | Lecture de registres d'entrée |
| FC05 | Write Single Coil | Écriture d'un bit de sortie |
| FC06 | Write Single Register | Écriture d'un registre |
| FC07 | Read Exception Status | Lecture du statut d'exception |
| FC08 | Diagnostics | Opérations de diagnostic |
| FC11 | Get Comm Event Counter | Compteur d'événements |
| FC15 | Write Multiple Coils | Écriture de plusieurs bits |
| FC16 | Write Multiple Registers | Écriture de plusieurs registres |
| FC17 | Report Server ID | Identification du serveur |

## Exemple rapide

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/edgeo/drivers/modbus"
)

func main() {
    // Créer un client
    client, err := modbus.NewClient("localhost:502",
        modbus.WithTimeout(5*time.Second),
        modbus.WithAutoReconnect(true),
    )
    if err != nil {
        panic(err)
    }
    defer client.Close()

    // Connexion
    ctx := context.Background()
    if err := client.Connect(ctx); err != nil {
        panic(err)
    }

    // Lire des registres
    regs, err := client.ReadHoldingRegisters(ctx, 0, 10)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Registres: %v\n", regs)
}
```

## Structure du package

```
modbus/
├── client.go      # Client Modbus TCP
├── server.go      # Serveur Modbus TCP
├── pool.go        # Pool de connexions
├── options.go     # Configuration fonctionnelle
├── types.go       # Types et constantes
├── errors.go      # Gestion des erreurs
├── metrics.go     # Métriques et observabilité
├── protocol.go    # Encodage/décodage du protocole
├── functions.go   # Fonctions Modbus (PDU builders)
└── version.go     # Informations de version
```

## Prochaines étapes

- [Démarrage rapide](./getting-started)
- [Documentation Client](./client)
- [Documentation Serveur](./server)
- [Pool de connexions](./pool)
- [Configuration](./options)
- [Gestion des erreurs](./errors)
- [Métriques](./metrics)
- [Changelog](./changelog)
