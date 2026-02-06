// Copyright 2025 Edgeo SCADA
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package modbus

import (
	"sync"
	"sync/atomic"
	"time"
)

// Counter is a simple atomic counter.
type Counter struct {
	value int64
}

// Add adds delta to the counter.
func (c *Counter) Add(delta int64) {
	atomic.AddInt64(&c.value, delta)
}

// Value returns the current counter value.
func (c *Counter) Value() int64 {
	return atomic.LoadInt64(&c.value)
}

// Reset resets the counter to zero.
func (c *Counter) Reset() {
	atomic.StoreInt64(&c.value, 0)
}

// LatencyHistogram tracks latency distribution.
type LatencyHistogram struct {
	mu      sync.Mutex
	buckets []int64    // count per bucket
	bounds  []float64  // upper bounds in ms
	sum     float64    // sum of all observations
	count   int64      // total count
	min     float64    // minimum observed value
	max     float64    // maximum observed value
}

// NewLatencyHistogram creates a new latency histogram with default buckets.
func NewLatencyHistogram() *LatencyHistogram {
	return &LatencyHistogram{
		buckets: make([]int64, 10),
		bounds:  []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 5000}, // ms
		min:     -1,
		max:     -1,
	}
}

// Observe records a latency observation.
func (h *LatencyHistogram) Observe(d time.Duration) {
	ms := float64(d.Microseconds()) / 1000.0

	h.mu.Lock()
	defer h.mu.Unlock()

	h.sum += ms
	h.count++

	if h.min < 0 || ms < h.min {
		h.min = ms
	}
	if ms > h.max {
		h.max = ms
	}

	for i, bound := range h.bounds {
		if ms <= bound {
			h.buckets[i]++
			return
		}
	}
	// Greater than all bounds
	h.buckets[len(h.buckets)-1]++
}

// Stats returns histogram statistics.
func (h *LatencyHistogram) Stats() LatencyStats {
	h.mu.Lock()
	defer h.mu.Unlock()

	stats := LatencyStats{
		Count:   h.count,
		Sum:     h.sum,
		Buckets: make(map[string]int64),
	}

	if h.count > 0 {
		stats.Avg = h.sum / float64(h.count)
		stats.Min = h.min
		stats.Max = h.max
	}

	// Copy bucket counts
	labels := []string{"1ms", "5ms", "10ms", "25ms", "50ms", "100ms", "250ms", "500ms", "1s", "5s+"}
	for i, count := range h.buckets {
		if i < len(labels) {
			stats.Buckets[labels[i]] = count
		}
	}

	return stats
}

// Reset resets the histogram.
func (h *LatencyHistogram) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for i := range h.buckets {
		h.buckets[i] = 0
	}
	h.sum = 0
	h.count = 0
	h.min = -1
	h.max = -1
}

// LatencyStats holds latency statistics.
type LatencyStats struct {
	Count   int64
	Sum     float64
	Avg     float64
	Min     float64
	Max     float64
	Buckets map[string]int64
}

// Metrics holds all client metrics.
type Metrics struct {
	RequestsTotal   Counter
	RequestsSuccess Counter
	RequestsErrors  Counter
	Reconnections   Counter
	ActiveConns     Counter
	Latency         *LatencyHistogram

	// Per-function code metrics
	funcMetrics sync.Map // FunctionCode -> *FunctionMetrics
}

// FunctionMetrics holds metrics for a specific function code.
type FunctionMetrics struct {
	Requests Counter
	Errors   Counter
	Latency  *LatencyHistogram
}

// NewMetrics creates a new Metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{
		Latency: NewLatencyHistogram(),
	}
}

// ForFunction returns metrics for a specific function code.
func (m *Metrics) ForFunction(fc FunctionCode) *FunctionMetrics {
	if val, ok := m.funcMetrics.Load(fc); ok {
		return val.(*FunctionMetrics)
	}

	fm := &FunctionMetrics{
		Latency: NewLatencyHistogram(),
	}
	actual, _ := m.funcMetrics.LoadOrStore(fc, fm)
	return actual.(*FunctionMetrics)
}

// Collect returns all metrics as a map (compatible with expvar/prometheus).
func (m *Metrics) Collect() map[string]interface{} {
	result := map[string]interface{}{
		"requests_total":   m.RequestsTotal.Value(),
		"requests_success": m.RequestsSuccess.Value(),
		"requests_errors":  m.RequestsErrors.Value(),
		"reconnections":    m.Reconnections.Value(),
		"active_conns":     m.ActiveConns.Value(),
		"latency":          m.Latency.Stats(),
	}

	// Collect per-function metrics
	funcStats := make(map[string]interface{})
	m.funcMetrics.Range(func(key, value interface{}) bool {
		fc := key.(FunctionCode)
		fm := value.(*FunctionMetrics)
		funcStats[fc.String()] = map[string]interface{}{
			"requests": fm.Requests.Value(),
			"errors":   fm.Errors.Value(),
			"latency":  fm.Latency.Stats(),
		}
		return true
	})
	if len(funcStats) > 0 {
		result["functions"] = funcStats
	}

	return result
}

// Reset resets all metrics.
func (m *Metrics) Reset() {
	m.RequestsTotal.Reset()
	m.RequestsSuccess.Reset()
	m.RequestsErrors.Reset()
	m.Reconnections.Reset()
	m.Latency.Reset()

	m.funcMetrics.Range(func(key, value interface{}) bool {
		fm := value.(*FunctionMetrics)
		fm.Requests.Reset()
		fm.Errors.Reset()
		fm.Latency.Reset()
		return true
	})
}

// String returns a string representation of FunctionCode.
func (fc FunctionCode) String() string {
	switch fc {
	case FuncReadCoils:
		return "ReadCoils"
	case FuncReadDiscreteInputs:
		return "ReadDiscreteInputs"
	case FuncReadHoldingRegisters:
		return "ReadHoldingRegisters"
	case FuncReadInputRegisters:
		return "ReadInputRegisters"
	case FuncWriteSingleCoil:
		return "WriteSingleCoil"
	case FuncWriteSingleRegister:
		return "WriteSingleRegister"
	case FuncReadExceptionStatus:
		return "ReadExceptionStatus"
	case FuncDiagnostics:
		return "Diagnostics"
	case FuncGetCommEventCounter:
		return "GetCommEventCounter"
	case FuncWriteMultipleCoils:
		return "WriteMultipleCoils"
	case FuncWriteMultipleRegisters:
		return "WriteMultipleRegisters"
	case FuncReportServerID:
		return "ReportServerID"
	default:
		return "Unknown"
	}
}
