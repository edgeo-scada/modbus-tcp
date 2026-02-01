# Changelog

Toutes les modifications notables de ce projet sont documentées dans ce fichier.

Le format est basé sur [Keep a Changelog](https://keepachangelog.com/fr/1.0.0/),
et ce projet adhère au [Semantic Versioning](https://semver.org/lang/fr/).

## [1.0.0] - 2024-02-01

### Ajouté

- **Client Modbus TCP**
  - Support de toutes les fonctions Modbus standard (FC01-FC17)
  - Reconnexion automatique avec backoff exponentiel
  - Configuration via options fonctionnelles
  - Métriques intégrées (latence, compteurs)
  - Logging structuré via `slog`
  - Variantes `WithUnit` pour toutes les opérations

- **Serveur Modbus TCP**
  - Support multi-clients concurrent
  - `MemoryHandler` pour tests et simulations
  - Interface `Handler` pour implémentations personnalisées
  - Limite de connexions configurable
  - Arrêt gracieux avec context

- **Pool de connexions**
  - Réutilisation des connexions
  - Health checks automatiques
  - Gestion du temps d'inactivité
  - `PooledClient` avec retour automatique

- **Métriques**
  - Compteurs atomiques thread-safe
  - Histogramme de latence avec buckets
  - Métriques par code fonction
  - Export compatible Prometheus/expvar

- **Gestion des erreurs**
  - Erreurs Modbus typées (`ModbusError`)
  - Tous les codes d'exception standard
  - Fonctions utilitaires (`IsException`, `IsIllegalDataAddress`, etc.)

### Sécurité

- Validation des entrées sur toutes les opérations
- Protection contre les dépassements d'adresse
- Timeouts configurables pour éviter les blocages

---

## Convention de versioning

Ce projet utilise le [Semantic Versioning](https://semver.org/lang/fr/):

- **MAJOR** (X.0.0): Changements incompatibles avec les versions précédentes
- **MINOR** (0.X.0): Nouvelles fonctionnalités rétrocompatibles
- **PATCH** (0.0.X): Corrections de bugs rétrocompatibles

### Accès à la version

```go
import "github.com/edgeo/drivers/modbus"

// Version string
fmt.Println(modbus.Version) // "1.0.0"

// Version détaillée
info := modbus.GetVersion()
fmt.Printf("v%d.%d.%d\n", info.Major, info.Minor, info.Patch)
```
