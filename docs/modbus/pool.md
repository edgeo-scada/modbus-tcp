# Pool de connexions

Le pool de connexions permet de réutiliser des connexions Modbus pour de meilleures performances.

## Création

```go
pool, err := modbus.NewPool(addr string, opts ...PoolOption) (*Pool, error)
```

**Paramètres:**
- `addr`: Adresse du serveur Modbus
- `opts`: Options de configuration du pool

```go
pool, err := modbus.NewPool("192.168.1.100:502",
    modbus.WithSize(10),
    modbus.WithMaxIdleTime(5*time.Minute),
    modbus.WithHealthCheckFrequency(1*time.Minute),
    modbus.WithClientOptions(
        modbus.WithTimeout(5*time.Second),
        modbus.WithUnitID(1),
    ),
)
if err != nil {
    log.Fatal(err)
}
defer pool.Close()
```

## Utilisation manuelle

### Get / Put

```go
// Obtenir une connexion
client, err := pool.Get(ctx)
if err != nil {
    log.Fatal(err)
}

// Utiliser la connexion
regs, err := client.ReadHoldingRegisters(ctx, 0, 10)

// IMPORTANT: Toujours remettre la connexion dans le pool
pool.Put(client)
```

### Pattern recommandé

```go
func readRegisters(ctx context.Context, pool *modbus.Pool) ([]uint16, error) {
    client, err := pool.Get(ctx)
    if err != nil {
        return nil, err
    }
    defer pool.Put(client)

    return client.ReadHoldingRegisters(ctx, 0, 10)
}
```

## Utilisation avec retour automatique

### GetPooled

La méthode `GetPooled` retourne un wrapper qui remet automatiquement la connexion dans le pool lors de l'appel à `Close()`:

```go
pc, err := pool.GetPooled(ctx)
if err != nil {
    log.Fatal(err)
}
defer pc.Close()  // Remet automatiquement dans le pool

regs, err := pc.ReadHoldingRegisters(ctx, 0, 10)
```

### Discard

Si la connexion est dans un état invalide, utilisez `Discard()` au lieu de `Close()`:

```go
pc, err := pool.GetPooled(ctx)
if err != nil {
    log.Fatal(err)
}

regs, err := pc.ReadHoldingRegisters(ctx, 0, 10)
if err != nil {
    pc.Discard()  // Ne remet pas dans le pool, ferme définitivement
    return nil, err
}

pc.Close()  // Remet dans le pool
return regs, nil
```

## Options du pool

| Option | Description | Défaut |
|--------|-------------|--------|
| `WithSize(n)` | Taille maximale du pool | 5 |
| `WithMaxIdleTime(d)` | Durée max d'inactivité avant fermeture | 5 min |
| `WithHealthCheckFrequency(d)` | Fréquence des vérifications de santé | 1 min |
| `WithClientOptions(opts...)` | Options pour les clients créés | - |

```go
pool, _ := modbus.NewPool("localhost:502",
    modbus.WithSize(20),                           // 20 connexions max
    modbus.WithMaxIdleTime(10*time.Minute),        // Ferme après 10min d'inactivité
    modbus.WithHealthCheckFrequency(30*time.Second), // Vérifie toutes les 30s
    modbus.WithClientOptions(
        modbus.WithTimeout(3*time.Second),
        modbus.WithAutoReconnect(false),  // Le pool gère la reconnexion
    ),
)
```

## Statistiques

```go
stats := pool.Stats()
fmt.Printf("Taille du pool: %d\n", stats.Size)
fmt.Printf("Connexions créées: %d\n", stats.Created)
fmt.Printf("Connexions disponibles: %d\n", stats.Available)
fmt.Printf("Get total: %d\n", stats.Gets)
fmt.Printf("Put total: %d\n", stats.Puts)
fmt.Printf("Hits (réutilisation): %d\n", stats.Hits)
fmt.Printf("Misses (nouvelle connexion): %d\n", stats.Misses)
fmt.Printf("Timeouts: %d\n", stats.Timeouts)
```

## Métriques

```go
type PoolMetrics struct {
    Gets      Counter  // Nombre d'appels à Get
    Puts      Counter  // Nombre d'appels à Put
    Hits      Counter  // Connexions réutilisées
    Misses    Counter  // Nouvelles connexions créées
    Timeouts  Counter  // Timeouts lors de Get
    Created   Counter  // Total connexions créées
    Closed    Counter  // Total connexions fermées
    Available Counter  // Connexions disponibles actuellement
}
```

```go
metrics := pool.Metrics()
hitRate := float64(metrics.Hits.Value()) / float64(metrics.Gets.Value()) * 100
fmt.Printf("Taux de réutilisation: %.1f%%\n", hitRate)
```

## Fermeture

```go
err := pool.Close()
```

La fermeture:
1. Arrête le health checker
2. Ferme toutes les connexions actives
3. Attend la fin des goroutines

## Comportement du pool

### Obtention d'une connexion

1. Tente d'obtenir une connexion disponible du pool
2. Vérifie que la connexion est valide (connectée, pas trop vieille)
3. Si aucune connexion disponible et `created < size`, crée une nouvelle connexion
4. Sinon, attend qu'une connexion se libère ou que le context expire

### Retour d'une connexion

1. Vérifie que la connexion est toujours connectée
2. Si connectée, la remet dans le pool
3. Si déconnectée ou pool plein, la ferme

### Health check

Le health checker vérifie périodiquement:
- Que les connexions sont toujours actives
- Que les connexions ne sont pas trop vieilles (idle time)

Les connexions invalides sont automatiquement fermées.
