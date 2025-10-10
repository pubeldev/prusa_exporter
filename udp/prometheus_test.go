package udp

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestInit(t *testing.T) {
	// Create a new test registry
	testRegistry := prometheus.NewRegistry()

	// Call Init
	Init(testRegistry)

	// Verify udpRegistry is set
	if udpRegistry == nil {
		t.Error("Init() did not set udpRegistry")
	}

	if udpRegistry != testRegistry {
		t.Error("Init() did not set udpRegistry to provided registry")
	}

	// Verify lastPush metric is registered
	if registryMetrics.metrics == nil {
		t.Error("Init() did not initialize registryMetrics.metrics")
	}

	if registryMetrics.labels == nil {
		t.Error("Init() did not initialize registryMetrics.labels")
	}

	if _, exists := registryMetrics.metrics["last_push"]; !exists {
		t.Error("Init() did not register last_push metric")
	}

	// Test that we can gather metrics from the registry
	metricFamilies, err := testRegistry.Gather()
	if err != nil {
		t.Errorf("Init() registry.Gather() error: %v", err)
	}

	// We may not have metrics immediately, so let's just check that gathering works
	// and we have the registry initialized properly
	if metricFamilies == nil {
		t.Error("Init() registry.Gather() should not return nil")
	}

	// The metric will appear only after it's actually used/set
	// For now, just verify the registry is functional
	t.Logf("Init() registry initialized with %d metric families", len(metricFamilies))
}

func TestRegisterMetric(t *testing.T) {
	// Initialize with a test registry
	testRegistry := prometheus.NewRegistry()
	Init(testRegistry)

	tests := []struct {
		name  string
		point point
	}{
		{
			name: "Simple metric",
			point: point{
				Measurement: "temperature",
				Tags:        map[string]string{"sensor": "nozzle", "printer_mac": "ABC123"},
				Fields:      map[string]interface{}{"value": 220.5},
			},
		},
		{
			name: "Metric with v field",
			point: point{
				Measurement: "temp_bed",
				Tags:        map[string]string{"printer_mac": "ABC123", "printer_address": "192.168.1.100"},
				Fields:      map[string]interface{}{"v": 60.0},
			},
		},
		{
			name: "Metric with multiple fields",
			point: point{
				Measurement: "fan",
				Tags:        map[string]string{"type": "print", "printer_mac": "ABC123"},
				Fields:      map[string]interface{}{"rpm": int64(1500), "pwm": int64(80)},
			},
		},
		{
			name: "Metric with boolean field",
			point: point{
				Measurement: "door_sensor",
				Tags:        map[string]string{"printer_mac": "ABC123"},
				Fields:      map[string]interface{}{"open": true},
			},
		},
		{
			name: "Metric with string field",
			point: point{
				Measurement: "filament",
				Tags:        map[string]string{"printer_mac": "ABC123"},
				Fields:      map[string]interface{}{"material": "PLA"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Register the metric
			registerMetric(tt.point)

			// Check that metrics were created
			registryMetrics.mu.Lock()
			defer registryMetrics.mu.Unlock()

			for fieldName := range tt.point.Fields {
				metricName := tt.point.Measurement
				if fieldName != "v" && fieldName != "value" {
					metricName = metricName + "_" + fieldName
				}

				if _, exists := registryMetrics.metrics[metricName]; !exists {
					t.Errorf("registerMetric() did not create metric %s", metricName)
				}

				if _, exists := registryMetrics.labels[metricName]; !exists {
					t.Errorf("registerMetric() did not store labels for metric %s", metricName)
				}
			}
		})
	}

	// Test that metrics can be gathered from registry
	metricFamilies, err := testRegistry.Gather()
	if err != nil {
		t.Errorf("registerMetric() registry.Gather() error: %v", err)
	}

	if len(metricFamilies) == 0 {
		t.Error("registerMetric() should create gatherable metrics")
	}
}

func TestGetLabels(t *testing.T) {
	tests := []struct {
		name     string
		tags     map[string]string
		expected int // expected number of labels
	}{
		{
			name:     "Empty tags",
			tags:     map[string]string{},
			expected: 0,
		},
		{
			name:     "Single tag",
			tags:     map[string]string{"printer_mac": "ABC123"},
			expected: 1,
		},
		{
			name: "Multiple tags",
			tags: map[string]string{
				"printer_mac":     "ABC123",
				"printer_address": "192.168.1.100",
				"sensor":          "nozzle",
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getLabels(tt.tags)

			if len(result) != tt.expected {
				t.Errorf("getLabels() returned %d labels, expected %d", len(result), tt.expected)
			}

			// Check that all tag keys are present in result
			tagKeys := make(map[string]bool)
			for key := range tt.tags {
				tagKeys[key] = true
			}

			for _, label := range result {
				if !tagKeys[label] {
					t.Errorf("getLabels() returned unexpected label: %s", label)
				}
				delete(tagKeys, label)
			}

			if len(tagKeys) > 0 {
				t.Errorf("getLabels() missing labels: %v", tagKeys)
			}
		})
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected float64
	}{
		{"int", 42, 42.0},
		{"int64", int64(123), 123.0},
		{"float64", 3.14, 3.14},
		{"bool true", true, 1.0},
		{"bool false", false, 0.0},
		{"nil", nil, 0.0},
		{"string PLA", "PLA", 1.0},
		{"string PETG", "PETG", 2.0},
		{"string ASA", "ASA", 3.0},
		{"string PC", "PC", 4.0},
		{"string PVB", "PVB", 5.0},
		{"string ABS", "ABS", 6.0},
		{"string HIPS", "HIPS", 7.0},
		{"string PP", "PP", 8.0},
		{"string FLEX", "FLEX", 9.0},
		{"string PA", "PA", 10.0},
		{"string ---", "---", -1.0},
		{"string unknown", "UNKNOWN", 0.0},
		{"unsupported type", []int{1, 2, 3}, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toFloat64(tt.input)
			if result != tt.expected {
				t.Errorf("toFloat64(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConcurrentMetricRegistration(t *testing.T) {
	// Test concurrent access to registryMetrics
	testRegistry := prometheus.NewRegistry()
	Init(testRegistry)

	// Create multiple goroutines that register metrics simultaneously
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			point := point{
				Measurement: "concurrent_test",
				Tags:        map[string]string{"printer_mac": "ABC123", "id": string(rune('A' + id))},
				Fields:      map[string]interface{}{"value": float64(id)},
			}

			registerMetric(point)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify that the metric was created and no race conditions occurred
	registryMetrics.mu.Lock()
	defer registryMetrics.mu.Unlock()

	if _, exists := registryMetrics.metrics["concurrent_test"]; !exists {
		t.Error("Concurrent metric registration failed to create metric")
	}
}

func BenchmarkRegisterMetric(b *testing.B) {
	testRegistry := prometheus.NewRegistry()
	Init(testRegistry)

	point := point{
		Measurement: "benchmark_metric",
		Tags:        map[string]string{"printer_mac": "ABC123", "printer_address": "192.168.1.100"},
		Fields:      map[string]interface{}{"value": 220.5, "target": 225.0},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registerMetric(point)
	}
}

func BenchmarkToFloat64(b *testing.B) {
	testValues := []interface{}{
		42,
		int64(123),
		3.14,
		true,
		"PLA",
		nil,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, val := range testValues {
			_ = toFloat64(val)
		}
	}
}
