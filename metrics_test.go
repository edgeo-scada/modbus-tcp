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
	"testing"
	"time"
)

func TestCounter(t *testing.T) {
	var c Counter

	if c.Value() != 0 {
		t.Errorf("Initial value: expected 0, got %d", c.Value())
	}

	c.Add(5)
	if c.Value() != 5 {
		t.Errorf("After Add(5): expected 5, got %d", c.Value())
	}

	c.Add(-2)
	if c.Value() != 3 {
		t.Errorf("After Add(-2): expected 3, got %d", c.Value())
	}

	c.Reset()
	if c.Value() != 0 {
		t.Errorf("After Reset: expected 0, got %d", c.Value())
	}
}

func TestLatencyHistogram(t *testing.T) {
	h := NewLatencyHistogram()

	// Record some observations
	h.Observe(500 * time.Microsecond) // 0.5ms
	h.Observe(2 * time.Millisecond)   // 2ms
	h.Observe(10 * time.Millisecond)  // 10ms
	h.Observe(50 * time.Millisecond)  // 50ms
	h.Observe(100 * time.Millisecond) // 100ms

	stats := h.Stats()

	if stats.Count != 5 {
		t.Errorf("Count: expected 5, got %d", stats.Count)
	}

	if stats.Min < 0.4 || stats.Min > 0.6 {
		t.Errorf("Min: expected ~0.5, got %.2f", stats.Min)
	}

	if stats.Max < 99 || stats.Max > 101 {
		t.Errorf("Max: expected ~100, got %.2f", stats.Max)
	}

	// Check buckets
	if stats.Buckets["1ms"] != 1 {
		t.Errorf("Bucket 1ms: expected 1, got %d", stats.Buckets["1ms"])
	}
	if stats.Buckets["5ms"] != 1 {
		t.Errorf("Bucket 5ms: expected 1, got %d", stats.Buckets["5ms"])
	}
}

func TestLatencyHistogramReset(t *testing.T) {
	h := NewLatencyHistogram()

	h.Observe(5 * time.Millisecond)
	h.Observe(10 * time.Millisecond)

	h.Reset()

	stats := h.Stats()
	if stats.Count != 0 {
		t.Errorf("Count after reset: expected 0, got %d", stats.Count)
	}
	if stats.Sum != 0 {
		t.Errorf("Sum after reset: expected 0, got %.2f", stats.Sum)
	}
}

func TestMetrics(t *testing.T) {
	m := NewMetrics()

	m.RequestsTotal.Add(10)
	m.RequestsSuccess.Add(8)
	m.RequestsErrors.Add(2)
	m.Reconnections.Add(1)

	collected := m.Collect()

	if collected["requests_total"] != int64(10) {
		t.Errorf("requests_total: expected 10, got %v", collected["requests_total"])
	}
	if collected["requests_success"] != int64(8) {
		t.Errorf("requests_success: expected 8, got %v", collected["requests_success"])
	}
	if collected["requests_errors"] != int64(2) {
		t.Errorf("requests_errors: expected 2, got %v", collected["requests_errors"])
	}
	if collected["reconnections"] != int64(1) {
		t.Errorf("reconnections: expected 1, got %v", collected["reconnections"])
	}
}

func TestMetricsReset(t *testing.T) {
	m := NewMetrics()

	m.RequestsTotal.Add(10)
	m.Latency.Observe(5 * time.Millisecond)

	m.Reset()

	if m.RequestsTotal.Value() != 0 {
		t.Errorf("RequestsTotal after reset: expected 0, got %d", m.RequestsTotal.Value())
	}

	stats := m.Latency.Stats()
	if stats.Count != 0 {
		t.Errorf("Latency.Count after reset: expected 0, got %d", stats.Count)
	}
}

func TestFunctionMetrics(t *testing.T) {
	m := NewMetrics()

	// Get metrics for a function
	fm := m.ForFunction(FuncReadHoldingRegisters)
	fm.Requests.Add(5)
	fm.Errors.Add(1)

	// Get same function again - should be same instance
	fm2 := m.ForFunction(FuncReadHoldingRegisters)
	if fm2.Requests.Value() != 5 {
		t.Errorf("Requests: expected 5, got %d", fm2.Requests.Value())
	}

	// Different function should be different instance
	fm3 := m.ForFunction(FuncWriteSingleRegister)
	fm3.Requests.Add(3)

	if fm3.Requests.Value() != 3 {
		t.Errorf("WriteSingleRegister requests: expected 3, got %d", fm3.Requests.Value())
	}
	if fm.Requests.Value() != 5 {
		t.Errorf("ReadHoldingRegisters requests: expected 5, got %d", fm.Requests.Value())
	}
}

func TestFunctionCodeString(t *testing.T) {
	tests := []struct {
		fc     FunctionCode
		expect string
	}{
		{FuncReadCoils, "ReadCoils"},
		{FuncReadDiscreteInputs, "ReadDiscreteInputs"},
		{FuncReadHoldingRegisters, "ReadHoldingRegisters"},
		{FuncReadInputRegisters, "ReadInputRegisters"},
		{FuncWriteSingleCoil, "WriteSingleCoil"},
		{FuncWriteSingleRegister, "WriteSingleRegister"},
		{FuncWriteMultipleCoils, "WriteMultipleCoils"},
		{FuncWriteMultipleRegisters, "WriteMultipleRegisters"},
		{FunctionCode(0xFF), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expect, func(t *testing.T) {
			if tt.fc.String() != tt.expect {
				t.Errorf("FunctionCode %d: expected %s, got %s", tt.fc, tt.expect, tt.fc.String())
			}
		})
	}
}
