package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file for testing
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_prusa.yml")

	validConfig := `
exporter:
  scrape_timeout: 5
  log_level: info

prusalink:
  common_labels: ["printer_name", "printer_type"]
  disable_metrics: ["some_metric"]

printers:
  - address: "192.168.1.100:80"
    username: "test_user"
    password: "test_pass"
    name: "TestPrinter1"
    type: "MK4"
  - address: "192.168.1.101:80"
    apikey: "test_api_key"
    name: "TestPrinter2"
    type: "XL"
`

	err := os.WriteFile(configPath, []byte(validConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	tests := []struct {
		name                   string
		prusaLinkScrapeTimeout int
		udpIPOverride          string
		udpAllMetrics          bool
		udpExtraMetrics        string
		lokiPushURL            string
		expectedScrapeTimeout  int
		expectedIPOverride     string
		expectedAllMetrics     bool
		expectedExtraMetrics   []string
		expectedLokiURL        string
		expectedPrinterCount   int
	}{
		{
			name:                   "Default values",
			prusaLinkScrapeTimeout: 10,
			udpIPOverride:          "",
			udpAllMetrics:          false,
			udpExtraMetrics:        "",
			lokiPushURL:            "",
			expectedScrapeTimeout:  10,
			expectedIPOverride:     "",
			expectedAllMetrics:     false,
			expectedExtraMetrics:   nil,
			expectedLokiURL:        "",
			expectedPrinterCount:   2,
		},
		{
			name:                   "With overrides",
			prusaLinkScrapeTimeout: 15,
			udpIPOverride:          "192.168.1.50",
			udpAllMetrics:          true,
			udpExtraMetrics:        "metric1,metric2,metric3",
			lokiPushURL:            "http://loki:3100/loki/api/v1/push",
			expectedScrapeTimeout:  15,
			expectedIPOverride:     "192.168.1.50",
			expectedAllMetrics:     true,
			expectedExtraMetrics:   []string{"metric1", "metric2", "metric3"},
			expectedLokiURL:        "http://loki:3100/loki/api/v1/push",
			expectedPrinterCount:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := LoadConfig(configPath, tt.prusaLinkScrapeTimeout, tt.udpIPOverride, tt.udpAllMetrics, tt.udpExtraMetrics, tt.lokiPushURL)
			if err != nil {
				t.Errorf("LoadConfig() error = %v", err)
				return
			}

			if config.Exporter.ScrapeTimeout != tt.expectedScrapeTimeout {
				t.Errorf("ScrapeTimeout = %d, expected %d", config.Exporter.ScrapeTimeout, tt.expectedScrapeTimeout)
			}

			if config.Exporter.IPOverride != tt.expectedIPOverride {
				t.Errorf("IPOverride = %s, expected %s", config.Exporter.IPOverride, tt.expectedIPOverride)
			}

			if config.Exporter.AllMetricsUDP != tt.expectedAllMetrics {
				t.Errorf("AllMetricsUDP = %t, expected %t", config.Exporter.AllMetricsUDP, tt.expectedAllMetrics)
			}

			if len(config.Exporter.ExtraMetrics) != len(tt.expectedExtraMetrics) {
				t.Errorf("ExtraMetrics length = %d, expected %d", len(config.Exporter.ExtraMetrics), len(tt.expectedExtraMetrics))
			} else {
				for i, metric := range tt.expectedExtraMetrics {
					if config.Exporter.ExtraMetrics[i] != metric {
						t.Errorf("ExtraMetrics[%d] = %s, expected %s", i, config.Exporter.ExtraMetrics[i], metric)
					}
				}
			}

			if config.Exporter.LokiPushURL != tt.expectedLokiURL {
				t.Errorf("LokiPushURL = %s, expected %s", config.Exporter.LokiPushURL, tt.expectedLokiURL)
			}

			if len(config.Printers) != tt.expectedPrinterCount {
				t.Errorf("Printers count = %d, expected %d", len(config.Printers), tt.expectedPrinterCount)
			}

			// Test first printer
			if len(config.Printers) > 0 {
				p1 := config.Printers[0]
				if p1.Address != "192.168.1.100:80" {
					t.Errorf("Printer 1 Address = %s, expected 192.168.1.100:80", p1.Address)
				}
				if p1.Username != "test_user" {
					t.Errorf("Printer 1 Username = %s, expected test_user", p1.Username)
				}
				if p1.Password != "test_pass" {
					t.Errorf("Printer 1 Password = %s, expected test_pass", p1.Password)
				}
				if p1.Name != "TestPrinter1" {
					t.Errorf("Printer 1 Name = %s, expected TestPrinter1", p1.Name)
				}
				if p1.Type != "MK4" {
					t.Errorf("Printer 1 Type = %s, expected MK4", p1.Type)
				}
			}

			// Test second printer
			if len(config.Printers) > 1 {
				p2 := config.Printers[1]
				if p2.Address != "192.168.1.101:80" {
					t.Errorf("Printer 2 Address = %s, expected 192.168.1.101:80", p2.Address)
				}
				if p2.Apikey != "test_api_key" {
					t.Errorf("Printer 2 Apikey = %s, expected test_api_key", p2.Apikey)
				}
				if p2.Name != "TestPrinter2" {
					t.Errorf("Printer 2 Name = %s, expected TestPrinter2", p2.Name)
				}
				if p2.Type != "XL" {
					t.Errorf("Printer 2 Type = %s, expected XL", p2.Type)
				}
			}
		})
	}
}

func TestLoadConfigErrors(t *testing.T) {
	t.Run("NonExistentFile", func(t *testing.T) {
		_, err := LoadConfig("nonexistent.yml", 10, "", false, "", "")
		if err == nil {
			t.Error("LoadConfig() expected error for non-existent file")
		}
	})

	t.Run("InvalidYAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "invalid.yml")

		invalidConfig := `
exporter:
  scrape_timeout: 5
  log_level: info
printers:
  - address: "192.168.1.100:80
    username: test_user  # missing quote
    password: "test_pass"
`

		err := os.WriteFile(configPath, []byte(invalidConfig), 0644)
		if err != nil {
			t.Fatalf("Failed to create test config file: %v", err)
		}

		_, err = LoadConfig(configPath, 10, "", false, "", "")
		if err == nil {
			t.Error("LoadConfig() expected error for invalid YAML")
		}
	})
}

func TestGetLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected zerolog.Level
	}{
		{"info", zerolog.InfoLevel},
		{"debug", zerolog.DebugLevel},
		{"trace", zerolog.TraceLevel},
		{"error", zerolog.ErrorLevel},
		{"panic", zerolog.PanicLevel},
		{"fatal", zerolog.FatalLevel},
		{"invalid", zerolog.InfoLevel}, // default
		{"", zerolog.InfoLevel},        // default
		{"INFO", zerolog.InfoLevel},    // case sensitive, should default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := GetLogLevel(tt.input)
			if result != tt.expected {
				t.Errorf("GetLogLevel(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPrintersStruct(t *testing.T) {
	// Test Printers struct initialization
	printer := Printers{
		Address:           "192.168.1.100:80",
		Username:          "user",
		Password:          "pass",
		Apikey:            "key",
		Name:              "TestPrinter",
		Type:              "MK4",
		Reachable:         true,
		UDPMetricsEnabled: false,
	}

	if printer.Address != "192.168.1.100:80" {
		t.Errorf("Address = %s, expected 192.168.1.100:80", printer.Address)
	}
	if printer.Username != "user" {
		t.Errorf("Username = %s, expected user", printer.Username)
	}
	if printer.Password != "pass" {
		t.Errorf("Password = %s, expected pass", printer.Password)
	}
	if printer.Apikey != "key" {
		t.Errorf("Apikey = %s, expected key", printer.Apikey)
	}
	if printer.Name != "TestPrinter" {
		t.Errorf("Name = %s, expected TestPrinter", printer.Name)
	}
	if printer.Type != "MK4" {
		t.Errorf("Type = %s, expected MK4", printer.Type)
	}
	if !printer.Reachable {
		t.Error("Reachable should be true")
	}
	if printer.UDPMetricsEnabled {
		t.Error("UDPMetricsEnabled should be false")
	}
}

func BenchmarkLoadConfig(b *testing.B) {
	tmpDir := b.TempDir()
	configPath := filepath.Join(tmpDir, "bench_prusa.yml")

	validConfig := `
exporter:
  scrape_timeout: 5
  log_level: info

printers:
  - address: "192.168.1.100:80"
    username: "test_user"
    password: "test_pass"
    name: "TestPrinter1"
    type: "MK4"
`

	err := os.WriteFile(configPath, []byte(validConfig), 0644)
	if err != nil {
		b.Fatalf("Failed to create test config file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := LoadConfig(configPath, 10, "", false, "", "")
		if err != nil {
			b.Errorf("LoadConfig() error: %v", err)
		}
	}
}
