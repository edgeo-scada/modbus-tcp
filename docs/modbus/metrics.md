# Metrics and Observability

The package includes built-in metrics for monitoring.

## Client Metrics

### Structure

```go
type Metrics struct {
    RequestsTotal   Counter          // Total requests sent
    RequestsSuccess Counter          // Successful requests
    RequestsErrors  Counter          // Failed requests
    Reconnections   Counter          // Number of reconnections
    ActiveConns     Counter          // Active connections (0 or 1)
    Latency         *LatencyHistogram // Latency histogram
}
```

### Accessing Metrics

```go
client, _ := modbus.NewClient("localhost:502")

// After some operations...
metrics := client.Metrics().Collect()

fmt.Printf("Total requests: %v\n", metrics["requests_total"])
fmt.Printf("Successful requests: %v\n", metrics["requests_success"])
fmt.Printf("Errors: %v\n", metrics["requests_errors"])
fmt.Printf("Reconnections: %v\n", metrics["reconnections"])
fmt.Printf("Active connections: %v\n", metrics["active_conns"])
```

### Latency

```go
metrics := client.Metrics().Collect()
latency := metrics["latency"].(modbus.LatencyStats)

fmt.Printf("Average latency: %.2f ms\n", latency.Avg)
fmt.Printf("Min latency: %.2f ms\n", latency.Min)
fmt.Printf("Max latency: %.2f ms\n", latency.Max)
fmt.Printf("Number of measurements: %d\n", latency.Count)

// Distribution by bucket
for bucket, count := range latency.Buckets {
    fmt.Printf("  %s: %d\n", bucket, count)
}
```

Latency buckets:
- 1ms, 5ms, 10ms, 25ms, 50ms, 100ms, 250ms, 500ms, 1s, 5s+

### Metrics by Function

```go
// Metrics for a specific function
fcMetrics := client.Metrics().ForFunction(modbus.FuncReadHoldingRegisters)
fmt.Printf("ReadHoldingRegisters - Requests: %d\n", fcMetrics.Requests.Value())
fmt.Printf("ReadHoldingRegisters - Errors: %d\n", fcMetrics.Errors.Value())
```

### Resetting Metrics

```go
client.Metrics().Reset()
```

## Server Metrics

### Structure

```go
type ServerMetrics struct {
    RequestsTotal   Counter  // Total requests received
    RequestsSuccess Counter  // Successfully processed requests
    RequestsErrors  Counter  // Failed requests
    ActiveConns     Counter  // Active connections
    TotalConns      Counter  // Total connections received
}
```

### Access

```go
server := modbus.NewServer(handler)

// ...

metrics := server.Metrics()
fmt.Printf("Active connections: %d\n", metrics.ActiveConns.Value())
fmt.Printf("Total connections: %d\n", metrics.TotalConns.Value())
fmt.Printf("Total requests: %d\n", metrics.RequestsTotal.Value())
fmt.Printf("Successful requests: %d\n", metrics.RequestsSuccess.Value())
fmt.Printf("Failed requests: %d\n", metrics.RequestsErrors.Value())
```

## Pool Metrics

### Structure

```go
type PoolMetrics struct {
    Gets      Counter  // Get calls
    Puts      Counter  // Put calls
    Hits      Counter  // Reused connections
    Misses    Counter  // New connections created
    Timeouts  Counter  // Timeouts during Get
    Created   Counter  // Total connections created
    Closed    Counter  // Total connections closed
    Available Counter  // Currently available connections
}
```

### Statistics

```go
pool, _ := modbus.NewPool("localhost:502", modbus.WithSize(10))

// ...

stats := pool.Stats()
fmt.Printf("Size: %d\n", stats.Size)
fmt.Printf("Created: %d\n", stats.Created)
fmt.Printf("Available: %d\n", stats.Available)
fmt.Printf("Gets: %d\n", stats.Gets)
fmt.Printf("Puts: %d\n", stats.Puts)
fmt.Printf("Hits: %d\n", stats.Hits)
fmt.Printf("Misses: %d\n", stats.Misses)
fmt.Printf("Timeouts: %d\n", stats.Timeouts)

// Reuse rate
if stats.Gets > 0 {
    hitRate := float64(stats.Hits) / float64(stats.Gets) * 100
    fmt.Printf("Reuse rate: %.1f%%\n", hitRate)
}
```

## Prometheus Integration

Example of exposing metrics for Prometheus:

```go
package main

import (
    "net/http"

    "github.com/edgeo-scada/modbus-tcp/modbus"
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

## expvar Integration

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

Access via `http://localhost:8080/debug/vars`.

## Counter

The `Counter` type is thread-safe:

```go
type Counter struct {
    value int64
}

func (c *Counter) Add(delta int64)  // Add (or subtract if negative)
func (c *Counter) Value() int64     // Read the value
func (c *Counter) Reset()           // Reset to zero
```

## LatencyHistogram

```go
type LatencyHistogram struct {
    // ...
}

func (h *LatencyHistogram) Observe(d time.Duration)  // Record a measurement
func (h *LatencyHistogram) Stats() LatencyStats      // Get statistics
func (h *LatencyHistogram) Reset()                   // Reset to zero
```

```go
type LatencyStats struct {
    Count   int64              // Number of measurements
    Sum     float64            // Total sum (ms)
    Avg     float64            // Average (ms)
    Min     float64            // Minimum (ms)
    Max     float64            // Maximum (ms)
    Buckets map[string]int64   // Distribution by bucket
}
```
