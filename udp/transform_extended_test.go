package udp

import (
	"reflect"
	"testing"
)

func TestNewPoint(t *testing.T) {
	p := newPoint()

	if p == nil {
		t.Error("newPoint() returned nil")
	}

	if p.Tags == nil {
		t.Error("newPoint() Tags map is nil")
	}

	if p.Fields == nil {
		t.Error("newPoint() Fields map is nil")
	}

	if len(p.Tags) != 0 {
		t.Errorf("newPoint() Tags map should be empty, got %d items", len(p.Tags))
	}

	if len(p.Fields) != 0 {
		t.Errorf("newPoint() Fields map should be empty, got %d items", len(p.Fields))
	}
}

func TestParseLineProtocol(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    *point
		expectError bool
	}{
		{
			name:  "Simple metric with single field",
			input: "temperature,sensor=nozzle value=220.5 1234567890",
			expected: &point{
				Measurement: "temperature",
				Tags:        map[string]string{"sensor": "nozzle"},
				Fields:      map[string]interface{}{"value": 220.5},
			},
			expectError: false,
		},
		{
			name:  "Metric with multiple tags",
			input: "temp_noz,printer_mac=ABC123,printer_address=192.168.1.100 v=25.3 1234567890",
			expected: &point{
				Measurement: "temp_noz",
				Tags:        map[string]string{"printer_mac": "ABC123", "printer_address": "192.168.1.100"},
				Fields:      map[string]interface{}{"v": 25.3},
			},
			expectError: false,
		},
		{
			name:  "Metric with integer field",
			input: "fan_speed,printer_mac=ABC123 rpm=1500i 1234567890",
			expected: &point{
				Measurement: "fan_speed",
				Tags:        map[string]string{"printer_mac": "ABC123"},
				Fields:      map[string]interface{}{"rpm": int64(1500)},
			},
			expectError: false,
		},
		{
			name:  "Metric with boolean field",
			input: "door_sensor,printer_mac=ABC123 open=true 1234567890",
			expected: &point{
				Measurement: "door_sensor",
				Tags:        map[string]string{"printer_mac": "ABC123"},
				Fields:      map[string]interface{}{"open": true},
			},
			expectError: false,
		},
		{
			name:  "Metric with string field",
			input: `filament_type,printer_mac=ABC123 material="PLA" 1234567890`,
			expected: &point{
				Measurement: "filament_type",
				Tags:        map[string]string{"printer_mac": "ABC123"},
				Fields:      map[string]interface{}{"material": "PLA"},
			},
			expectError: false,
		},
		{
			name:  "Metric with multiple fields",
			input: "printer_stats,printer_mac=ABC123 temp=220.5,target=225.0,pwm=80i 1234567890",
			expected: &point{
				Measurement: "printer_stats",
				Tags:        map[string]string{"printer_mac": "ABC123"},
				Fields:      map[string]interface{}{"temp": 220.5, "target": 225.0, "pwm": int64(80)},
			},
			expectError: false,
		},
		{
			name:        "Invalid format - no fields",
			input:       "temperature,sensor=nozzle",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "Invalid format - too many parts",
			input:       "temperature,sensor=nozzle value=220.5 1234567890 extra",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "Invalid tag format",
			input:       "temperature,sensor value=220.5 1234567890",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "Invalid field format",
			input:       "temperature,sensor=nozzle value 1234567890",
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseLineProtocol(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("parseLineProtocol() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("parseLineProtocol() unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("parseLineProtocol() returned nil result")
				return
			}

			if result.Measurement != tt.expected.Measurement {
				t.Errorf("parseLineProtocol() Measurement = %v, expected %v", result.Measurement, tt.expected.Measurement)
			}

			if !reflect.DeepEqual(result.Tags, tt.expected.Tags) {
				t.Errorf("parseLineProtocol() Tags = %v, expected %v", result.Tags, tt.expected.Tags)
			}

			if !reflect.DeepEqual(result.Fields, tt.expected.Fields) {
				t.Errorf("parseLineProtocol() Fields = %v, expected %v", result.Fields, tt.expected.Fields)
			}
		})
	}
}

func TestProcessMessage(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		mac      string
		prefix   string
		ip       string
		expected int // expected number of metrics
	}{
		{
			name:     "Single metric message",
			message:  "12345 temp_noz v=220.5 1637000000",
			mac:      "ABC123",
			prefix:   "prusa_",
			ip:       "192.168.1.100:8514",
			expected: 1,
		},
		{
			name: "Multiple metrics message",
			message: `12345 temp_noz v=220.5 1637000000
temp_bed v=60.0 1637000000
fan_speed rpm=1500i 1637000000`,
			mac:      "ABC123",
			prefix:   "prusa_",
			ip:       "192.168.1.100:8514",
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processMessage(tt.message, tt.mac, tt.prefix, tt.ip)
			if err != nil {
				t.Errorf("processMessage() error = %v", err)
				return
			}

			if len(result) != tt.expected {
				t.Errorf("processMessage() returned %d metrics, expected %d", len(result), tt.expected)
			}

			// Check that each metric contains the expected prefix and labels
			for _, metric := range result {
				if !contains(metric, tt.prefix) {
					t.Errorf("processMessage() metric %s should contain prefix %s", metric, tt.prefix)
				}
				if !contains(metric, "printer_mac="+tt.mac) {
					t.Errorf("processMessage() metric %s should contain mac %s", metric, tt.mac)
				}
				if !contains(metric, "printer_address=192.168.1.100") {
					t.Errorf("processMessage() metric %s should contain IP address", metric)
				}
			}
		})
	}
}

func TestParseFirstMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{
			name:     "Valid first message",
			input:    "12345 temp_noz v=220.5 1637000000",
			expected: "temp_noz v=220.5 1637000000",
			hasError: false,
		},
		{
			name:     "Message with multiple parts",
			input:    "12345 temp_bed v=60.0 fan_speed rpm=1500i 1637000000",
			expected: "temp_bed v=60.0 fan_speed rpm=1500i 1637000000",
			hasError: false,
		},
		{
			name:     "Empty message",
			input:    "",
			expected: "",
			hasError: false, // Empty string splits to [""], so len > 0
		},
		{
			name:     "Single part message",
			input:    "12345",
			expected: "",
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseFirstMessage(tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("parseFirstMessage() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("parseFirstMessage() unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("parseFirstMessage() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestUpdateMetric(t *testing.T) {
	tests := []struct {
		name     string
		splitted []string
		prefix   string
		mac      string
		ip       string
		expected string
	}{
		{
			name:     "Basic metric update",
			splitted: []string{"temp_noz", "v=220.5", "1637000000"},
			prefix:   "prusa_",
			mac:      "ABC123",
			ip:       "192.168.1.100:8514",
			expected: "prusa_temp_noz,printer_mac=ABC123,printer_address=192.168.1.100",
		},
		{
			name:     "Metric with existing tags",
			splitted: []string{"fan,type=print", "rpm=1500i", "1637000000"},
			prefix:   "prusa_",
			mac:      "DEF456",
			ip:       "10.0.0.5:8514",
			expected: "prusa_fan,type=print,printer_mac=DEF456,printer_address=10.0.0.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := updateMetric(tt.splitted, tt.prefix, tt.mac, tt.ip)
			if err != nil {
				t.Errorf("updateMetric() error = %v", err)
				return
			}

			if len(result) == 0 {
				t.Error("updateMetric() returned empty result")
				return
			}

			if !contains(result[0], tt.expected) {
				t.Errorf("updateMetric() result[0] = %v, should contain %v", result[0], tt.expected)
			}
		})
	}

	// Test error case
	t.Run("Empty splitted message", func(t *testing.T) {
		_, err := updateMetric([]string{}, "prusa_", "ABC123", "192.168.1.100:8514")
		if err == nil {
			t.Error("updateMetric() expected error for empty input")
		}
	})
}

func TestProcessIdentifiers(t *testing.T) {
	tests := []struct {
		name        string
		data        map[string]interface{}
		expectedMAC string
		expectedIP  string
		expectError bool
	}{
		{
			name: "Valid identifiers",
			data: map[string]interface{}{
				"hostname": "ABC123DEF456",
				"client":   "192.168.1.100:54321",
			},
			expectedMAC: "ABC123DEF456",
			expectedIP:  "192.168.1.100:54321",
			expectError: false,
		},
		{
			name: "Missing hostname",
			data: map[string]interface{}{
				"client": "192.168.1.100:54321",
			},
			expectedMAC: "",
			expectedIP:  "",
			expectError: true,
		},
		{
			name: "Missing client",
			data: map[string]interface{}{
				"hostname": "ABC123DEF456",
			},
			expectedMAC: "",
			expectedIP:  "",
			expectError: true,
		},
		{
			name: "Invalid hostname type",
			data: map[string]interface{}{
				"hostname": 12345,
				"client":   "192.168.1.100:54321",
			},
			expectedMAC: "",
			expectedIP:  "",
			expectError: true,
		},
		{
			name: "Invalid client type",
			data: map[string]interface{}{
				"hostname": "ABC123DEF456",
				"client":   12345,
			},
			expectedMAC: "",
			expectedIP:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mac, ip, err := processIdentifiers(tt.data)

			if tt.expectError {
				if err == nil {
					t.Error("processIdentifiers() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("processIdentifiers() unexpected error: %v", err)
				return
			}

			if mac != tt.expectedMAC {
				t.Errorf("processIdentifiers() mac = %v, expected %v", mac, tt.expectedMAC)
			}

			if ip != tt.expectedIP {
				t.Errorf("processIdentifiers() ip = %v, expected %v", ip, tt.expectedIP)
			}
		})
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func BenchmarkParseLineProtocol(b *testing.B) {
	testLine := "temp_noz,printer_mac=ABC123,printer_address=192.168.1.100 v=220.5,target=225.0 1637000000"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parseLineProtocol(testLine)
		if err != nil {
			b.Errorf("parseLineProtocol() error: %v", err)
		}
	}
}

func BenchmarkSplitLine(b *testing.B) {
	testLine := `temp_noz,printer_mac=ABC123 v=220.5,target="PLA filament" 1637000000`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = splitLine(testLine)
	}
}
