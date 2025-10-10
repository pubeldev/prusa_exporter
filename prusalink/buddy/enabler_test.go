package prusalink

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pstrobl96/prusa_exporter/config"
)

func TestGetLocalIP(t *testing.T) {
	// Save original configuration for cleanup
	originalConfig := configuration

	// Test case 1: When IP override is set
	t.Run("WithIPOverride", func(t *testing.T) {
		configuration = config.Config{}
		configuration.Exporter.IpOverride = "192.168.1.100"

		ip, err := getLocalIP()
		if err != nil {
			t.Fatalf("getLocalIP() returned error: %v", err)
		}

		if ip != "192.168.1.100" {
			t.Errorf("getLocalIP() = %v, want %v", ip, "192.168.1.100")
		}
	})

	// Test case 2: When IP override is empty (should find actual IP)
	t.Run("WithoutIPOverride", func(t *testing.T) {
		configuration = config.Config{}
		configuration.Exporter.IpOverride = ""

		ip, err := getLocalIP()

		// This test is environment-dependent, so we just check that it returns a valid IP format
		if err != nil {
			// It's acceptable if no IP is found in some test environments
			t.Logf("getLocalIP() returned error (may be expected in test env): %v", err)
			return
		}

		// Check if returned IP is valid IPv4
		if parsedIP := net.ParseIP(ip); parsedIP == nil || parsedIP.To4() == nil {
			t.Errorf("getLocalIP() returned invalid IPv4 address: %v", ip)
		}

		// Should not be loopback
		if strings.HasPrefix(ip, "127.") {
			t.Errorf("getLocalIP() returned loopback address: %v", ip)
		}
	})

	// Restore original configuration
	configuration = originalConfig
}

func TestGcodeInit(t *testing.T) {
	// Save original configuration for cleanup
	originalConfig := configuration

	testCases := []struct {
		name            string
		ipOverride      string
		expectedIP      string
		expectError     bool
		expectedMetrics []string
	}{
		{
			name:       "WithIPOverride",
			ipOverride: "10.0.0.1",
			expectedIP: "10.0.0.1",
			expectedMetrics: []string{
				"temp_noz", "ttemp_noz", "temp_bed", "ttemp_bed",
				"chamber_temp", "temp_mcu", "temp_hbr", "loadcell_value",
				"curr_inp", "volt_bed", "eth_in", "eth_out",
			},
		},
		{
			name:       "WithDifferentIP",
			ipOverride: "192.168.100.50",
			expectedIP: "192.168.100.50",
			expectedMetrics: []string{
				"temp_noz", "ttemp_noz", "temp_bed", "ttemp_bed",
				"chamber_temp", "temp_mcu", "temp_hbr", "loadcell_value",
				"curr_inp", "volt_bed", "eth_in", "eth_out",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup configuration
			configuration = config.Config{}
			configuration.Exporter.IpOverride = tc.ipOverride

			gcode, err := gcodeInit()

			if tc.expectError && err == nil {
				t.Errorf("gcodeInit() expected error but got none")
				return
			}

			if !tc.expectError && err != nil {
				t.Errorf("gcodeInit() unexpected error: %v", err)
				return
			}

			if err != nil {
				return // Expected error case
			}

			// Check if gcode contains expected IP
			expectedSyslogLine := fmt.Sprintf("M334 %s 8514", tc.expectedIP)
			if !strings.Contains(gcode, expectedSyslogLine) {
				t.Errorf("gcodeInit() missing expected syslog line: %v", expectedSyslogLine)
			}

			// Check if gcode starts with M330 SYSLOG
			if !strings.HasPrefix(gcode, "M330 SYSLOG") {
				t.Errorf("gcodeInit() should start with 'M330 SYSLOG', got: %v", gcode)
			}

			// Check if all expected metrics are present
			for _, metric := range tc.expectedMetrics {
				expectedMetricLine := fmt.Sprintf("M331 %s", metric)
				if !strings.Contains(gcode, expectedMetricLine) {
					t.Errorf("gcodeInit() missing expected metric line: %v", expectedMetricLine)
				}
			}

			// Count the number of M331 lines (should match number of metrics)
			m331Count := strings.Count(gcode, "M331")
			if m331Count != len(tc.expectedMetrics) {
				t.Errorf("gcodeInit() expected %d M331 lines, got %d", len(tc.expectedMetrics), m331Count)
			}
		})
	}

	// Restore original configuration
	configuration = originalConfig
}

func TestSendGcode(t *testing.T) {
	// Save original configuration for cleanup
	originalConfig := configuration

	// Create a test server that handles both DELETE and PUT requests
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v1/files/usb//test_file.gcode"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		// Handle DELETE request (deleteGcode called first in sendGcode)
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"deleted": true}`))
			return
		}

		// Handle PUT request (actual sendGcode)
		if r.Method == http.MethodPut {
			// Check headers for PUT request
			if r.Header.Get("Content-Type") != "text/x.gcode" {
				t.Errorf("Expected Content-Type text/x.gcode, got %s", r.Header.Get("Content-Type"))
			}

			if r.Header.Get("Overwrite") != "?1" {
				t.Errorf("Expected Overwrite ?1, got %s", r.Header.Get("Overwrite"))
			}

			// Return success response
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "success"}`))
			return
		}

		t.Errorf("Unexpected request method: %s", r.Method)
	}))
	defer testServer.Close()

	// Setup configuration
	configuration = config.Config{}
	configuration.Exporter.ScrapeTimeout = 10
	configuration.Exporter.IpOverride = "10.0.0.1"

	// Extract host from test server URL (remove http://)
	serverHost := strings.TrimPrefix(testServer.URL, "http://")

	printer := config.Printers{
		Address:  serverHost,
		Username: "test_user",
		Password: "test_pass",
	}

	result, err := sendGcode("test_file.gcode", printer)
	if err != nil {
		t.Errorf("sendGcode() unexpected error: %v", err)
	}

	expectedResult := `{"status": "success"}`
	if string(result) != expectedResult {
		t.Errorf("sendGcode() = %v, want %v", string(result), expectedResult)
	}

	// Restore original configuration
	configuration = originalConfig
}

func TestDeleteGcode(t *testing.T) {
	// Save original configuration for cleanup
	originalConfig := configuration

	// Create a test server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check method
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE request, got %s", r.Method)
		}

		// Check URL path
		expectedPath := "/api/v1/files/usb//test_file.gcode"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		// Return success response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"deleted": true}`))
	}))
	defer testServer.Close()

	// Setup configuration
	configuration = config.Config{}
	configuration.Exporter.ScrapeTimeout = 10

	// Extract host from test server URL (remove http://)
	serverHost := strings.TrimPrefix(testServer.URL, "http://")

	printer := config.Printers{
		Address:  serverHost,
		Username: "test_user",
		Password: "test_pass",
	}

	result, err := deleteGcode("test_file.gcode", printer)
	if err != nil {
		t.Errorf("deleteGcode() unexpected error: %v", err)
	}

	expectedResult := `{"deleted": true}`
	if string(result) != expectedResult {
		t.Errorf("deleteGcode() = %v, want %v", string(result), expectedResult)
	}

	// Restore original configuration
	configuration = originalConfig
}

func TestStartGcode(t *testing.T) {
	// Save original configuration for cleanup
	originalConfig := configuration

	testCases := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectedError bool
		errorContains string
	}{
		{
			name:          "Success",
			statusCode:    http.StatusNoContent,
			responseBody:  "",
			expectedError: false,
		},
		{
			name:          "BadRequest",
			statusCode:    http.StatusBadRequest,
			responseBody:  `{"error": "bad request"}`,
			expectedError: true,
			errorContains: "failed to start gcode file, status code: 400",
		},
		{
			name:          "ServerError",
			statusCode:    http.StatusInternalServerError,
			responseBody:  `{"error": "internal server error"}`,
			expectedError: true,
			errorContains: "failed to start gcode file, status code: 500",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test server
			testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check method
				if r.Method != http.MethodPost {
					t.Errorf("Expected POST request, got %s", r.Method)
				}

				// Check URL path
				expectedPath := "/api/v1/files/usb//test_file.gcode"
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				// Return test response
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(tc.responseBody))
			}))
			defer testServer.Close()

			// Setup configuration
			configuration = config.Config{}
			configuration.Exporter.ScrapeTimeout = 10

			// Extract host from test server URL (remove http://)
			serverHost := strings.TrimPrefix(testServer.URL, "http://")

			printer := config.Printers{
				Address:  serverHost,
				Username: "test_user",
				Password: "test_pass",
			}

			result, err := startGcode("test_file.gcode", printer)

			if tc.expectedError {
				if err == nil {
					t.Errorf("startGcode() expected error but got none")
					return
				}
				if tc.errorContains != "" && !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("startGcode() error = %v, expected to contain %v", err, tc.errorContains)
				}
				return
			}

			if err != nil {
				t.Errorf("startGcode() unexpected error: %v", err)
				return
			}

			if string(result) != tc.responseBody {
				t.Errorf("startGcode() = %v, want %v", string(result), tc.responseBody)
			}
		})
	}

	// Restore original configuration
	configuration = originalConfig
}

func TestEnableUDPmetrics(t *testing.T) {
	// Save original configuration for cleanup
	originalConfig := configuration

	// Track the requests made to the test server
	var requests []string
	requestCount := 0

	// Create a test server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		requests = append(requests, fmt.Sprintf("%s %s", r.Method, r.URL.Path))

		// Handle PUT request (sendGcode)
		if r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "enable_udp_metrics.gcode") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"uploaded": true}`))
			return
		}

		// Handle POST request (startGcode)
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "enable_udp_metrics.gcode") {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Handle DELETE request (deleteGcode - called before sendGcode)
		if r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "enable_udp_metrics.gcode") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"deleted": true}`))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer testServer.Close()

	// Setup configuration
	configuration = config.Config{}
	configuration.Exporter.ScrapeTimeout = 10
	configuration.Exporter.IpOverride = "10.0.0.1"

	// Extract host from test server URL (remove http://)
	serverHost := strings.TrimPrefix(testServer.URL, "http://")

	// Create test printers
	printers := []config.Printers{
		{
			Address:           serverHost,
			Username:          "test_user1",
			Password:          "test_pass1",
			Name:              "Printer1",
			UDPMetricsEnabled: false,
		},
		{
			Address:           serverHost,
			Username:          "test_user2",
			Password:          "test_pass2",
			Name:              "Printer2",
			UDPMetricsEnabled: false,
		},
	}

	// Set up configuration.Printers to match the input
	configuration.Printers = make([]config.Printers, len(printers))
	copy(configuration.Printers, printers)

	// Call the function
	EnableUDPmetrics(printers)

	// Verify that UDP metrics were enabled for all printers
	for i, printer := range configuration.Printers {
		if !printer.UDPMetricsEnabled {
			t.Errorf("Printer %d (%s) UDPMetricsEnabled should be true", i, printer.Name)
		}
	}

	// Verify that the correct number of requests were made
	// Each printer should make: DELETE, PUT, POST = 3 requests per printer
	expectedRequests := len(printers) * 3
	if requestCount != expectedRequests {
		t.Errorf("Expected %d requests, got %d", expectedRequests, requestCount)
	}

	// Verify the request patterns
	expectedRequestPatterns := []string{
		"DELETE", "PUT", "POST", // First printer
		"DELETE", "PUT", "POST", // Second printer
	}

	if len(requests) >= len(expectedRequestPatterns) {
		for i, expectedPattern := range expectedRequestPatterns {
			if !strings.Contains(requests[i], expectedPattern) {
				t.Errorf("Request %d should contain %s, got %s", i, expectedPattern, requests[i])
			}
		}
	}

	// Restore original configuration
	configuration = originalConfig
}

func TestListOfMetrics(t *testing.T) {
	expectedMetrics := []string{
		"temp_noz",
		"ttemp_noz",
		"temp_bed",
		"ttemp_bed",
		"chamber_temp",
		"temp_mcu",
		"temp_hbr",
		"loadcell_value",
		"curr_inp",
		"volt_bed",
		"eth_in",
		"eth_out",
	}

	if len(listOfMetrics) != len(expectedMetrics) {
		t.Errorf("listOfMetrics length = %d, want %d", len(listOfMetrics), len(expectedMetrics))
	}

	for i, metric := range expectedMetrics {
		if i >= len(listOfMetrics) || listOfMetrics[i] != metric {
			t.Errorf("listOfMetrics[%d] = %v, want %v", i, listOfMetrics[i], metric)
		}
	}
}

// TestErrorCases tests various error scenarios
func TestErrorCases(t *testing.T) {
	// Save original configuration for cleanup
	originalConfig := configuration

	t.Run("SendGcodeWithInvalidServer", func(t *testing.T) {
		configuration = config.Config{}
		configuration.Exporter.ScrapeTimeout = 1 // Short timeout for quick failure
		configuration.Exporter.IpOverride = "10.0.0.1"

		printer := config.Printers{
			Address:  "invalid-server:9999",
			Username: "test_user",
			Password: "test_pass",
		}

		_, err := sendGcode("test_file.gcode", printer)
		if err == nil {
			t.Errorf("sendGcode() with invalid server should return error")
		}
	})

	t.Run("DeleteGcodeWithInvalidServer", func(t *testing.T) {
		configuration = config.Config{}
		configuration.Exporter.ScrapeTimeout = 1 // Short timeout for quick failure

		printer := config.Printers{
			Address:  "invalid-server:9999",
			Username: "test_user",
			Password: "test_pass",
		}

		_, err := deleteGcode("test_file.gcode", printer)
		if err == nil {
			t.Errorf("deleteGcode() with invalid server should return error")
		}
	})

	t.Run("StartGcodeWithInvalidServer", func(t *testing.T) {
		configuration = config.Config{}
		configuration.Exporter.ScrapeTimeout = 1 // Short timeout for quick failure

		printer := config.Printers{
			Address:  "invalid-server:9999",
			Username: "test_user",
			Password: "test_pass",
		}

		_, err := startGcode("test_file.gcode", printer)
		if err == nil {
			t.Errorf("startGcode() with invalid server should return error")
		}
	})

	// Restore original configuration
	configuration = originalConfig
}

// Benchmark tests
func BenchmarkGcodeInit(b *testing.B) {
	// Save original configuration for cleanup
	originalConfig := configuration

	// Setup configuration
	configuration = config.Config{}
	configuration.Exporter.IpOverride = "10.0.0.1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gcodeInit()
		if err != nil {
			b.Errorf("gcodeInit() error: %v", err)
		}
	}

	// Restore original configuration
	configuration = originalConfig
}

func BenchmarkGetLocalIP(b *testing.B) {
	// Save original configuration for cleanup
	originalConfig := configuration

	// Setup configuration with IP override for consistent performance
	configuration = config.Config{}
	configuration.Exporter.IpOverride = "192.168.1.100"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := getLocalIP()
		if err != nil {
			b.Errorf("getLocalIP() error: %v", err)
		}
	}

	// Restore original configuration
	configuration = originalConfig
}
