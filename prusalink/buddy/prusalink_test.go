package prusalink

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pstrobl96/prusa_exporter/config"
)

func TestBoolToFloat(t *testing.T) {
	tests := []struct {
		input    bool
		expected float64
	}{
		{true, 1.0},
		{false, 0.0},
	}

	for _, tt := range tests {
		result := BoolToFloat(tt.input)
		if result != tt.expected {
			t.Errorf("BoolToFloat(%t) = %f, expected %f", tt.input, result, tt.expected)
		}
	}
}

func TestGetStateFlag(t *testing.T) {
	tests := []struct {
		name     string
		printer  Printer
		expected float64
	}{
		{
			name: "Operational",
			printer: Printer{
				State: struct {
					Text  string `json:"text"`
					Flags struct {
						LinkState     string `json:"link_state"`
						Operational   bool   `json:"operational"`
						Paused        bool   `json:"paused"`
						Printing      bool   `json:"printing"`
						Cancelling    bool   `json:"cancelling"`
						Pausing       bool   `json:"pausing"`
						Error         bool   `json:"error"`
						SdReady       bool   `json:"sdReady"`
						ClosedOnError bool   `json:"closedOnError"`
						Ready         bool   `json:"ready"`
						Busy          bool   `json:"busy"`
						ClosedOrError bool   `json:"closedOrError"`
						Finished      bool   `json:"finished"`
						Prepared      bool   `json:"prepared"`
					} `json:"flags"`
				}{
					Flags: struct {
						LinkState     string `json:"link_state"`
						Operational   bool   `json:"operational"`
						Paused        bool   `json:"paused"`
						Printing      bool   `json:"printing"`
						Cancelling    bool   `json:"cancelling"`
						Pausing       bool   `json:"pausing"`
						Error         bool   `json:"error"`
						SdReady       bool   `json:"sdReady"`
						ClosedOnError bool   `json:"closedOnError"`
						Ready         bool   `json:"ready"`
						Busy          bool   `json:"busy"`
						ClosedOrError bool   `json:"closedOrError"`
						Finished      bool   `json:"finished"`
						Prepared      bool   `json:"prepared"`
					}{Operational: true},
				},
			},
			expected: 1,
		},
		{
			name: "Prepared",
			printer: Printer{
				State: struct {
					Text  string `json:"text"`
					Flags struct {
						LinkState     string `json:"link_state"`
						Operational   bool   `json:"operational"`
						Paused        bool   `json:"paused"`
						Printing      bool   `json:"printing"`
						Cancelling    bool   `json:"cancelling"`
						Pausing       bool   `json:"pausing"`
						Error         bool   `json:"error"`
						SdReady       bool   `json:"sdReady"`
						ClosedOnError bool   `json:"closedOnError"`
						Ready         bool   `json:"ready"`
						Busy          bool   `json:"busy"`
						ClosedOrError bool   `json:"closedOrError"`
						Finished      bool   `json:"finished"`
						Prepared      bool   `json:"prepared"`
					} `json:"flags"`
				}{
					Flags: struct {
						LinkState     string `json:"link_state"`
						Operational   bool   `json:"operational"`
						Paused        bool   `json:"paused"`
						Printing      bool   `json:"printing"`
						Cancelling    bool   `json:"cancelling"`
						Pausing       bool   `json:"pausing"`
						Error         bool   `json:"error"`
						SdReady       bool   `json:"sdReady"`
						ClosedOnError bool   `json:"closedOnError"`
						Ready         bool   `json:"ready"`
						Busy          bool   `json:"busy"`
						ClosedOrError bool   `json:"closedOrError"`
						Finished      bool   `json:"finished"`
						Prepared      bool   `json:"prepared"`
					}{Prepared: true},
				},
			},
			expected: 2,
		},
		{
			name: "Printing",
			printer: Printer{
				State: struct {
					Text  string `json:"text"`
					Flags struct {
						LinkState     string `json:"link_state"`
						Operational   bool   `json:"operational"`
						Paused        bool   `json:"paused"`
						Printing      bool   `json:"printing"`
						Cancelling    bool   `json:"cancelling"`
						Pausing       bool   `json:"pausing"`
						Error         bool   `json:"error"`
						SdReady       bool   `json:"sdReady"`
						ClosedOnError bool   `json:"closedOnError"`
						Ready         bool   `json:"ready"`
						Busy          bool   `json:"busy"`
						ClosedOrError bool   `json:"closedOrError"`
						Finished      bool   `json:"finished"`
						Prepared      bool   `json:"prepared"`
					} `json:"flags"`
				}{
					Flags: struct {
						LinkState     string `json:"link_state"`
						Operational   bool   `json:"operational"`
						Paused        bool   `json:"paused"`
						Printing      bool   `json:"printing"`
						Cancelling    bool   `json:"cancelling"`
						Pausing       bool   `json:"pausing"`
						Error         bool   `json:"error"`
						SdReady       bool   `json:"sdReady"`
						ClosedOnError bool   `json:"closedOnError"`
						Ready         bool   `json:"ready"`
						Busy          bool   `json:"busy"`
						ClosedOrError bool   `json:"closedOrError"`
						Finished      bool   `json:"finished"`
						Prepared      bool   `json:"prepared"`
					}{Printing: true},
				},
			},
			expected: 4,
		},
		{
			name: "No flags set",
			printer: Printer{
				State: struct {
					Text  string `json:"text"`
					Flags struct {
						LinkState     string `json:"link_state"`
						Operational   bool   `json:"operational"`
						Paused        bool   `json:"paused"`
						Printing      bool   `json:"printing"`
						Cancelling    bool   `json:"cancelling"`
						Pausing       bool   `json:"pausing"`
						Error         bool   `json:"error"`
						SdReady       bool   `json:"sdReady"`
						ClosedOnError bool   `json:"closedOnError"`
						Ready         bool   `json:"ready"`
						Busy          bool   `json:"busy"`
						ClosedOrError bool   `json:"closedOrError"`
						Finished      bool   `json:"finished"`
						Prepared      bool   `json:"prepared"`
					} `json:"flags"`
				}{},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStateFlag(tt.printer)
			if result != tt.expected {
				t.Errorf("getStateFlag() = %f, expected %f", result, tt.expected)
			}
		})
	}
}

func TestAccessPrinterEndpoint(t *testing.T) {
	// Create a test server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for API key first
		if apiKey := r.Header.Get("X-Api-Key"); apiKey != "" {
			// API key authentication
			if apiKey != "test_api_key" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		} else {
			// For digest auth, respond with 401 and digest challenge on first request
			auth := r.Header.Get("Authorization")
			if auth == "" {
				w.Header().Set("WWW-Authenticate", `Digest realm="Test", qop="auth", nonce="dcd98b7102dd2f0e8b11d0f600bfb0c093", opaque="5ccc069c403ebaf9f0171e9517f40e41"`)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			// For test purposes, check if this contains wrong credentials
			if strings.Contains(auth, "wrong_user") || strings.Contains(auth, "wrong_pass") {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			// Accept any other digest auth response with valid format
			if !strings.Contains(auth, "Digest") {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}

		// Return test data based on endpoint
		switch r.URL.Path {
		case "/api/v1/status":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"printer":{"state":"Operational"}}`))
		case "/api/v1/job":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"job":{"file":{"name":"test.gcode"}}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer testServer.Close()

	// Save original configuration and initialize properly
	originalConfig := configuration
	defer func() { configuration = originalConfig }()

	configuration = config.Config{
		Exporter: struct {
			ScrapeTimeout int    `yaml:"scrape_timeout"`
			LogLevel      string `yaml:"log_level"`
			IPOverride    string
			AllMetricsUDP bool
			ExtraMetrics  []string
			LokiPushURL   string
		}{
			ScrapeTimeout: 5,
		},
	}

	// Extract host from test server URL
	serverHost := strings.TrimPrefix(testServer.URL, "http://")

	tests := []struct {
		name         string
		path         string
		printer      config.Printers
		expectError  bool
		expectedData string
	}{
		{
			name: "Basic auth success",
			path: "/api/v1/status",
			printer: config.Printers{
				Address:  serverHost,
				Username: "test_user",
				Password: "test_pass",
			},
			expectError:  false,
			expectedData: `{"printer":{"state":"Operational"}}`,
		},
		{
			name: "API key auth success",
			path: "/api/v1/job",
			printer: config.Printers{
				Address: serverHost,
				Apikey:  "test_api_key",
			},
			expectError:  false,
			expectedData: `{"job":{"file":{"name":"test.gcode"}}}`,
		},
		{
			name: "Invalid credentials",
			path: "/api/v1/status",
			printer: config.Printers{
				Address:  serverHost,
				Username: "wrong_user",
				Password: "wrong_pass",
			},
			expectError: true,
		},
		{
			name: "Invalid endpoint",
			path: "/api/v1/invalid",
			printer: config.Printers{
				Address:  serverHost,
				Username: "test_user",
				Password: "test_pass",
			},
			expectError: true,
		},
		{
			name: "Invalid server",
			path: "/api/v1/status",
			printer: config.Printers{
				Address:  "invalid-server:9999",
				Username: "test_user",
				Password: "test_pass",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := accessPrinterEndpoint(tt.path, tt.printer)

			if tt.expectError {
				if err == nil {
					t.Errorf("accessPrinterEndpoint() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("accessPrinterEndpoint() unexpected error: %v", err)
				return
			}

			if string(result) != tt.expectedData {
				t.Errorf("accessPrinterEndpoint() = '%s' (len=%d), expected '%s' (len=%d)", string(result), len(result), tt.expectedData, len(tt.expectedData))
			}
		})
	}

	// Restore original configuration
	configuration = originalConfig
}

func TestPrinterTypes(t *testing.T) {
	expectedTypes := map[string]string{
		"PrusaMINI":         "MINI",
		"PrusaMK4":          "MK4",
		"PrusaXL":           "XL",
		"PrusaLink I3MK3S":  "I3MK3S",
		"PrusaLink I3MK3":   "I3MK3",
		"PrusaLink I3MK25S": "I3MK25S",
		"PrusaLink I3MK25":  "I3MK25",
		"prusa-sl1":         "SL1",
		"prusa-sl1s":        "SL1S",
		"Prusa_iX":          "IX",
	}

	for hostname, expectedType := range expectedTypes {
		if actualType, exists := printerTypes[hostname]; !exists || actualType != expectedType {
			t.Errorf("printerTypes[%s] = %s, expected %s", hostname, actualType, expectedType)
		}
	}
}

func TestHTTPTimeouts(t *testing.T) {
	// Create a test server that delays responses
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(6 * time.Second) // Delay longer than timeout (5 * 1 = 5 seconds)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer testServer.Close()

	// Save original configuration and initialize properly
	originalConfig := configuration
	defer func() { configuration = originalConfig }()

	configuration = config.Config{
		Exporter: struct {
			ScrapeTimeout int    `yaml:"scrape_timeout"`
			LogLevel      string `yaml:"log_level"`
			IPOverride    string
			AllMetricsUDP bool
			ExtraMetrics  []string
			LokiPushURL   string
		}{
			ScrapeTimeout: 1, // 1 second timeout
		},
	}

	serverHost := strings.TrimPrefix(testServer.URL, "http://")

	printer := config.Printers{
		Address:  serverHost,
		Username: "test_user",
		Password: "test_pass",
	}

	_, err := accessPrinterEndpoint("/api/v1/status", printer)
	if err == nil {
		t.Error("accessPrinterEndpoint() should timeout but didn't")
		return
	}

	// Check that error is timeout-related
	errMsg := err.Error()
	if !strings.Contains(errMsg, "timeout") && !strings.Contains(errMsg, "context deadline exceeded") && !strings.Contains(errMsg, "Timeout exceeded") {
		t.Errorf("Expected timeout error, got: %v", err)
	}

	// Restore original configuration
	configuration = originalConfig
}

func TestJSONStructures(t *testing.T) {
	// Test that our JSON structures can be marshaled and unmarshaled correctly

	// Test Version struct
	version := Version{
		API:      "1.0",
		Server:   "test",
		Hostname: "testhost",
	}

	jsonBytes, err := json.Marshal(version)
	if err != nil {
		t.Errorf("json.Marshal(Version) error: %v", err)
	}

	var unmarshaledVersion Version
	err = json.Unmarshal(jsonBytes, &unmarshaledVersion)
	if err != nil {
		t.Errorf("json.Unmarshal(Version) error: %v", err)
	}

	if unmarshaledVersion.API != version.API {
		t.Errorf("Unmarshaled API = %s, expected %s", unmarshaledVersion.API, version.API)
	}
}

func BenchmarkBoolToFloat(b *testing.B) {
	for i := 0; i < b.N; i++ {
		BoolToFloat(i%2 == 0)
	}
}

func BenchmarkGetStateFlag(b *testing.B) {
	printer := Printer{
		State: struct {
			Text  string `json:"text"`
			Flags struct {
				LinkState     string `json:"link_state"`
				Operational   bool   `json:"operational"`
				Paused        bool   `json:"paused"`
				Printing      bool   `json:"printing"`
				Cancelling    bool   `json:"cancelling"`
				Pausing       bool   `json:"pausing"`
				Error         bool   `json:"error"`
				SdReady       bool   `json:"sdReady"`
				ClosedOnError bool   `json:"closedOnError"`
				Ready         bool   `json:"ready"`
				Busy          bool   `json:"busy"`
				ClosedOrError bool   `json:"closedOrError"`
				Finished      bool   `json:"finished"`
				Prepared      bool   `json:"prepared"`
			} `json:"flags"`
		}{
			Flags: struct {
				LinkState     string `json:"link_state"`
				Operational   bool   `json:"operational"`
				Paused        bool   `json:"paused"`
				Printing      bool   `json:"printing"`
				Cancelling    bool   `json:"cancelling"`
				Pausing       bool   `json:"pausing"`
				Error         bool   `json:"error"`
				SdReady       bool   `json:"sdReady"`
				ClosedOnError bool   `json:"closedOnError"`
				Ready         bool   `json:"ready"`
				Busy          bool   `json:"busy"`
				ClosedOrError bool   `json:"closedOrError"`
				Finished      bool   `json:"finished"`
				Prepared      bool   `json:"prepared"`
			}{Printing: true},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getStateFlag(printer)
	}
}
