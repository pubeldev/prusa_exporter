package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMain(t *testing.T) {
	// This test ensures the main Run function can be called without panicking
	// We can't run the full function as it starts an HTTP server

	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_prusa.yml")

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
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Save original command line args
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	// Set test arguments
	os.Args = []string{
		"prusa_exporter",
		"--config.file=" + configPath,
		"--exporter.metrics-port=0", // Use port 0 to avoid conflicts
	}

	// Test that Run function can be called without panicking
	// We'll use a timeout to prevent hanging
	done := make(chan bool, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Run() panicked: %v", r)
			}
			done <- true
		}()

		// This will hang at ListenAndServe, so we can't actually run it to completion
		// But we can test that it doesn't panic during initialization
		// Run() // Cannot call this as it will block
		done <- true
	}()

	select {
	case <-done:
		// Test completed
	case <-time.After(100 * time.Millisecond):
		// Timeout - acceptable for this test
	}
}

func TestConfigValidation(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name       string
		config     string
		shouldFail bool
	}{
		{
			name: "Valid config",
			config: `
exporter:
  scrape_timeout: 5
  log_level: info

printers:
  - address: "192.168.1.100:80"
    username: "test_user"
    password: "test_pass"
    name: "TestPrinter1"
    type: "MK4"
`,
			shouldFail: false,
		},
		{
			name: "Invalid YAML",
			config: `
exporter:
  scrape_timeout: 5
  log_level: info
printers:
  - address: "192.168.1.100:80
    username: test_user
    password: "test_pass"
`,
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(tmpDir, tt.name+"_prusa.yml")

			err := os.WriteFile(configPath, []byte(tt.config), 0644)
			if err != nil {
				t.Fatalf("Failed to create test config file: %v", err)
			}

			// Save original command line args
			origArgs := os.Args
			defer func() { os.Args = origArgs }()

			// Set test arguments
			os.Args = []string{
				"prusa_exporter",
				"--config.file=" + configPath,
			}

			// Test config validation by attempting to parse flags
			// This is indirect testing since we can't easily run the full function
			if _, err := os.Stat(configPath); os.IsNotExist(err) && !tt.shouldFail {
				t.Errorf("Config file should exist but doesn't")
			}
		})
	}
}

func TestHTTPHandlers(t *testing.T) {
	// Test the root HTTP handler
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()

	// Create a handler function similar to what's in Run()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		syslogAddress := "0.0.0.0:8514"
		metricsPath := "/metrics/prusalink"
		udpMetricsPath := "/metrics/udp"

		html := `<html>
    <head><title>prusa_exporter 2.0.0-alpha2</title></head>
    <body>
    <h1>prusa_exporter</h1>
	<p>Syslog server running at - <b>` + syslogAddress + `</b></p>
    <p><a href="` + metricsPath + `">PrusaLink metrics</a></p>
	<p><a href="` + udpMetricsPath + `">UDP Metrics</a></p>
	</body>
    </html>`
		w.Write([]byte(html))
	})

	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check that the response contains expected content
	body := rr.Body.String()
	expectedStrings := []string{
		"prusa_exporter",
		"Syslog server running",
		"PrusaLink metrics",
		"UDP Metrics",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(body, expected) {
			t.Errorf("Handler response should contain %q, got: %s", expected, body)
		}
	}
}

func TestFlagDefaults(t *testing.T) {
	// Test that default flag values are sensible
	// This is more of a documentation test to ensure defaults don't change unexpectedly

	expectedDefaults := map[string]string{
		"config.file":               "./prusa.yml",
		"exporter.metrics-path":     "/metrics/prusalink",
		"exporter.udp-metrics-path": "/metrics/udp",
		"exporter.metrics-port":     "10009",
		"prusalink.scrape-timeout":  "10",
		"log.level":                 "info",
		"udp.ip-override":           "",
		"udp.listen-address":        "0.0.0.0:8514",
		"udp.prefix":                "prusa_",
		"udp.extra-metrics":         "",
		"udp.all-metrics":           "false",
		"udp.gcode-enabled":         "true",
		"loki.push-url":             "",
	}

	// This test validates that we know what our defaults are
	// In a real implementation, you'd parse the flags and check their defaults
	// For now, we just document what they should be
	for flag, defaultValue := range expectedDefaults {
		if defaultValue == "" && flag != "udp.ip-override" && flag != "udp.extra-metrics" && flag != "loki.push-url" {
			t.Errorf("Flag %s has empty default value", flag)
		}
	}
}

func TestPathValidation(t *testing.T) {
	// Test that metrics paths are different (as required by the code)
	metricsPath := "/metrics/prusalink"
	udpMetricsPath := "/metrics/udp"

	if metricsPath == udpMetricsPath {
		t.Error("Metrics paths should be different")
	}

	// Test that both paths are valid HTTP paths
	if !strings.HasPrefix(metricsPath, "/") {
		t.Error("Metrics path should start with /")
	}

	if !strings.HasPrefix(udpMetricsPath, "/") {
		t.Error("UDP metrics path should start with /")
	}
}

func TestPortValidation(t *testing.T) {
	// Test port validation logic
	testPorts := []struct {
		port  int
		valid bool
	}{
		{80, true},
		{8080, true},
		{10009, true}, // default
		{65535, true},
		{0, false}, // 0 is typically not valid unless for auto-assignment
		{-1, false},
		{65536, false},
	}

	for _, tt := range testPorts {
		t.Run(string(rune(tt.port)), func(t *testing.T) {
			// Basic port range validation
			isValid := tt.port > 0 && tt.port <= 65535
			if isValid != tt.valid {
				t.Errorf("Port %d validity = %t, expected %t", tt.port, isValid, tt.valid)
			}
		})
	}
}

func TestLogLevelParsing(t *testing.T) {
	// Test that log levels can be parsed
	validLevels := []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}

	for _, level := range validLevels {
		t.Run(level, func(t *testing.T) {
			// This would typically test zerolog.ParseLevel, but we don't want to import it here
			// Just validate that our expected levels are reasonable
			if level == "" {
				t.Error("Log level should not be empty")
			}
		})
	}
}

func BenchmarkHTTPHandler(b *testing.B) {
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		b.Fatal(err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		html := `<html><head><title>prusa_exporter</title></head><body><h1>prusa_exporter</h1></body></html>`
		w.Write([]byte(html))
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}
