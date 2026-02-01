# Métriques et observabilité

Le package inclut des métriques intégrées pour le monitoring.

## Métriques Client

### Structure

```go
type Metrics struct {
    RequestsTotal   Counter          // Total des requêtes envoyées
    RequestsSuccess Counter          // Requêtes réussies
    RequestsErrors  Counter          // Requêtes en erreur
    Reconnections   Counter          // Nombre de reconnexions
    ActiveConns     Counter          // Connexions actives (0 ou 1)
    Latency         *LatencyHistogram // Histogramme de latence
}
```

### Accès aux métriques

```go
client, _ := modbus.NewClient("localhost:502")

// Après quelques opérations...
metrics := client.Metrics().Collect()

fmt.Printf("Requêtes totales: %v\n", metrics["requests_total"])
fmt.Printf("Requêtes réussies: %v\n", metrics["requests_success"])
fmt.Printf("Erreurs: %v\n", metrics["requests_errors"])
fmt.Printf("Reconnexions: %v\n", metrics["reconnections"])
fmt.Printf("Connexions actives: %v\n", metrics["active_conns"])
```

### Latence

```go
metrics := client.Metrics().Collect()
latency := metrics["latency"].(modbus.LatencyStats)

fmt.Printf("Latence moyenne: %.2f ms\n", latency.Avg)
fmt.Printf("Latence min: %.2f ms\n", latency.Min)
fmt.Printf("Latence max: %.2f ms\n", latency.Max)
fmt.Printf("Nombre de mesures: %d\n", latency.Count)

// Distribution par bucket
for bucket, count := range latency.Buckets {
    fmt.Printf("  %s: %d\n", bucket, count)
}
```

Les buckets de latence:
- 1ms, 5ms, 10ms, 25ms, 50ms, 100ms, 250ms, 500ms, 1s, 5s+

### Métriques par fonction

```go
// Métriques pour une fonction spécifique
fcMetrics := client.Metrics().ForFunction(modbus.FuncReadHoldingRegisters)
fmt.Printf("ReadHoldingRegisters - Requêtes: %d\n", fcMetrics.Requests.Value())
fmt.Printf("ReadHoldingRegisters - Erreurs: %d\n", fcMetrics.Errors.Value())
```

### Reset des métriques

```go
client.Metrics().Reset()
```

## Métriques Serveur

### Structure

```go
type ServerMetrics struct {
    RequestsTotal   Counter  // Total des requêtes reçues
    RequestsSuccess Counter  // Requêtes traitées avec succès
    RequestsErrors  Counter  // Requêtes en erreur
    ActiveConns     Counter  // Connexions actives
    TotalConns      Counter  // Total des connexions reçues
}
```

### Accès

```go
server := modbus.NewServer(handler)

// ...

metrics := server.Metrics()
fmt.Printf("Connexions actives: %d\n", metrics.ActiveConns.Value())
fmt.Printf("Total connexions: %d\n", metrics.TotalConns.Value())
fmt.Printf("Requêtes totales: %d\n", metrics.RequestsTotal.Value())
fmt.Printf("Requêtes réussies: %d\n", metrics.RequestsSuccess.Value())
fmt.Printf("Requêtes en erreur: %d\n", metrics.RequestsErrors.Value())
```

## Métriques Pool

### Structure

```go
type PoolMetrics struct {
    Gets      Counter  // Appels à Get
    Puts      Counter  // Appels à Put
    Hits      Counter  // Connexions réutilisées
    Misses    Counter  // Nouvelles connexions créées
    Timeouts  Counter  // Timeouts lors de Get
    Created   Counter  // Total connexions créées
    Closed    Counter  // Total connexions fermées
    Available Counter  // Connexions disponibles
}
```

### Statistiques

```go
pool, _ := modbus.NewPool("localhost:502", modbus.WithSize(10))

// ...

stats := pool.Stats()
fmt.Printf("Taille: %d\n", stats.Size)
fmt.Printf("Créées: %d\n", stats.Created)
fmt.Printf("Disponibles: %d\n", stats.Available)
fmt.Printf("Gets: %d\n", stats.Gets)
fmt.Printf("Puts: %d\n", stats.Puts)
fmt.Printf("Hits: %d\n", stats.Hits)
fmt.Printf("Misses: %d\n", stats.Misses)
fmt.Printf("Timeouts: %d\n", stats.Timeouts)

// Taux de réutilisation
if stats.Gets > 0 {
    hitRate := float64(stats.Hits) / float64(stats.Gets) * 100
    fmt.Printf("Taux de réutilisation: %.1f%%\n", hitRate)
}
```

## Intégration Prometheus

Exemple d'exposition des métriques pour Prometheus:

```go
package main

import (
    "net/http"

    "github.com/edgeo/drivers/modbus"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

type ModbusCollector struct {
    client *modbus.Client

    requestsTotal   *prometheus.Desc
    requestsSuccess *prometheus.Desc
    requestsErrors  *prometheus.Desc
    latencyAvg      *prometheus.Desc
}

func NewModbusCollector(client *modbus.Client) *ModbusCollector {
    return &ModbusCollector{
        client: client,
        requestsTotal: prometheus.NewDesc(
            "modbus_requests_total",
            "Total number of Modbus requests",
            nil, nil,
        ),
        requestsSuccess: prometheus.NewDesc(
            "modbus_requests_success_total",
            "Number of successful Modbus requests",
            nil, nil,
        ),
        requestsErrors: prometheus.NewDesc(
            "modbus_requests_errors_total",
            "Number of failed Modbus requests",
            nil, nil,
        ),
        latencyAvg: prometheus.NewDesc(
            "modbus_latency_avg_ms",
            "Average request latency in milliseconds",
            nil, nil,
        ),
    }
}

func (c *ModbusCollector) Describe(ch chan<- *prometheus.Desc) {
    ch <- c.requestsTotal
    ch <- c.requestsSuccess
    ch <- c.requestsErrors
    ch <- c.latencyAvg
}

func (c *ModbusCollector) Collect(ch chan<- prometheus.Metric) {
    metrics := c.client.Metrics().Collect()

    ch <- prometheus.MustNewConstMetric(
        c.requestsTotal,
        prometheus.CounterValue,
        float64(metrics["requests_total"].(int64)),
    )
    ch <- prometheus.MustNewConstMetric(
        c.requestsSuccess,
        prometheus.CounterValue,
        float64(metrics["requests_success"].(int64)),
    )
    ch <- prometheus.MustNewConstMetric(
        c.requestsErrors,
        prometheus.CounterValue,
        float64(metrics["requests_errors"].(int64)),
    )

    if latency, ok := metrics["latency"].(modbus.LatencyStats); ok {
        ch <- prometheus.MustNewConstMetric(
            c.latencyAvg,
            prometheus.GaugeValue,
            latency.Avg,
        )
    }
}

func main() {
    client, _ := modbus.NewClient("localhost:502")

    collector := NewModbusCollector(client)
    prometheus.MustRegister(collector)

    http.Handle("/metrics", promhttp.Handler())
    http.ListenAndServe(":9090", nil)
}
```

## Intégration expvar

```go
import (
    "encoding/json"
    "expvar"
)

func init() {
    expvar.Publish("modbus", expvar.Func(func() interface{} {
        return client.Metrics().Collect()
    }))
}
```

Accès via `http://localhost:8080/debug/vars`.

## Counter

Le type `Counter` est thread-safe:

```go
type Counter struct {
    value int64
}

func (c *Counter) Add(delta int64)  // Ajouter (ou soustraire si négatif)
func (c *Counter) Value() int64     // Lire la valeur
func (c *Counter) Reset()           // Remettre à zéro
```

## LatencyHistogram

```go
type LatencyHistogram struct {
    // ...
}

func (h *LatencyHistogram) Observe(d time.Duration)  // Enregistrer une mesure
func (h *LatencyHistogram) Stats() LatencyStats      // Obtenir les statistiques
func (h *LatencyHistogram) Reset()                   // Remettre à zéro
```

```go
type LatencyStats struct {
    Count   int64              // Nombre de mesures
    Sum     float64            // Somme totale (ms)
    Avg     float64            // Moyenne (ms)
    Min     float64            // Minimum (ms)
    Max     float64            // Maximum (ms)
    Buckets map[string]int64   // Distribution par bucket
}
```
