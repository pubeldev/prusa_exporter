package prusalink

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pstrobl96/prusa_exporter/config"
)

func TestSLBasicStructures(t *testing.T) {
	// Test basic JSON structures for SL printers
	// Since the SL package is currently just a dummy, we test basic JSON handling

	// Test basic version structure
	versionJSON := `{
		"api": "0.1",
		"server": "1.0.0",
		"text": "SL1S Firmware"
	}`

	var version map[string]interface{}
	err := json.Unmarshal([]byte(versionJSON), &version)
	if err != nil {
		t.Errorf("Failed to unmarshal version JSON: %v", err)
	}

	if version["api"] != "0.1" {
		t.Errorf("Version API = %v, expected 0.1", version["api"])
	}

	// Test basic job structure
	jobJSON := `{
		"state": "Printing",
		"progress": {
			"completion": 50.5,
			"printTime": 1800
		}
	}`

	var job map[string]interface{}
	err = json.Unmarshal([]byte(jobJSON), &job)
	if err != nil {
		t.Errorf("Failed to unmarshal job JSON: %v", err)
	}

	if job["state"] != "Printing" {
		t.Errorf("Job state = %v, expected Printing", job["state"])
	}
}

func TestSLAPIEndpoints(t *testing.T) {
	// Create a mock SL server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate SL printer API responses
		switch r.URL.Path {
		case "/api/version":
			w.Header().Set("Content-Type", "application/json")
			response := `{
				"api": "0.1",
				"server": "1.0.0",
				"text": "SL1S Firmware",
				"firmware": "4.3.2"
			}`
			w.Write([]byte(response))

		case "/api/job":
			w.Header().Set("Content-Type", "application/json")
			response := `{
				"state": "Operational",
				"job": {
					"estimatedPrintTime": null,
					"file": {
						"name": null,
						"path": null,
						"size": null
					}
				},
				"progress": {
					"completion": null,
					"printTime": null,
					"printTimeLeft": null
				}
			}`
			w.Write([]byte(response))

		case "/api/printer":
			w.Header().Set("Content-Type", "application/json")
			response := `{
				"telemetry": {
					"temp-bed": 25.0,
					"material": "---",
					"coverClosed": true,
					"fanBlower": 0.0,
					"fanRear": 50.0,
					"tempAmbient": 23.5,
					"tempCpu": 45.2,
					"tempUvLed": 26.8
				},
				"state": {
					"text": "Operational",
					"flags": {
						"operational": true,
						"printing": false,
						"paused": false,
						"error": false,
						"ready": true
					}
				}
			}`
			w.Write([]byte(response))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer testServer.Close()

	// Test version endpoint
	resp, err := http.Get(testServer.URL + "/api/version")
	if err != nil {
		t.Errorf("Failed to get version: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Version endpoint returned status %d, expected %d", resp.StatusCode, http.StatusOK)
	}

	var version map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&version)
	if err != nil {
		t.Errorf("Failed to decode version response: %v", err)
	}

	if version["api"] != "0.1" {
		t.Errorf("Version API = %v, expected 0.1", version["api"])
	}

	// Test job endpoint
	resp, err = http.Get(testServer.URL + "/api/job")
	if err != nil {
		t.Errorf("Failed to get job: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Job endpoint returned status %d, expected %d", resp.StatusCode, http.StatusOK)
	}

	// Test printer endpoint
	resp, err = http.Get(testServer.URL + "/api/printer")
	if err != nil {
		t.Errorf("Failed to get printer: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Printer endpoint returned status %d, expected %d", resp.StatusCode, http.StatusOK)
	}
}

func TestSLPrinterCollector(t *testing.T) {
	// Test that SL printer collector can be created
	// This is a basic test since we can't easily test the full collector without more setup

	config := config.Config{
		Printers: []config.Printers{
			{
				Address:  "192.168.1.100:80",
				Username: "maker",
				Password: "maker",
				Name:     "SL1S_Test",
				Type:     "SL1S",
			},
		},
	}

	// Test that we can create a collector (this would normally be in the actual collector file)
	if len(config.Printers) == 0 {
		t.Error("Should have at least one printer configured")
	}

	printer := config.Printers[0]
	if printer.Type != "SL1S" {
		t.Errorf("Printer type = %s, expected SL1S", printer.Type)
	}

	if printer.Name != "SL1S_Test" {
		t.Errorf("Printer name = %s, expected SL1S_Test", printer.Name)
	}
}

func TestSLSpecificFields(t *testing.T) {
	// Test SL-specific telemetry fields that are different from FDM printers

	telemetryJSON := `{
		"temp-bed": 25.0,
		"temp-nozzle": 0.0,
		"material": "Tough Resin",
		"coverClosed": false,
		"fanBlower": 100.0,
		"fanRear": 80.0,
		"fanUvLed": 0.0,
		"tempAmbient": 24.5,
		"tempCpu": 52.1,
		"tempUvLed": 28.3
	}`

	// Define the telemetry struct inline for testing
	var telemetry struct {
		TempBed     float64 `json:"temp-bed"`
		TempNozzle  float64 `json:"temp-nozzle"`
		Material    string  `json:"material"`
		CoverClosed bool    `json:"coverClosed"`
		FanBlower   float64 `json:"fanBlower"`
		FanRear     float64 `json:"fanRear"`
		FanUvLed    float64 `json:"fanUvLed"`
		TempAmbient float64 `json:"tempAmbient"`
		TempCPU     float64 `json:"tempCpu"`
		TempUvLed   float64 `json:"tempUvLed"`
	}

	err := json.Unmarshal([]byte(telemetryJSON), &telemetry)
	if err != nil {
		t.Errorf("Failed to unmarshal telemetry: %v", err)
	}

	// Test SL-specific values
	if telemetry.TempNozzle != 0.0 {
		t.Errorf("SL printer should have temp-nozzle = 0.0, got %f", telemetry.TempNozzle)
	}

	if telemetry.Material != "Tough Resin" {
		t.Errorf("Material = %s, expected Tough Resin", telemetry.Material)
	}

	if telemetry.CoverClosed {
		t.Error("Cover should be open (false)")
	}

	if telemetry.FanUvLed != 0.0 {
		t.Errorf("UV LED fan should be 0.0, got %f", telemetry.FanUvLed)
	}

	if telemetry.TempUvLed < 20.0 || telemetry.TempUvLed > 40.0 {
		t.Errorf("UV LED temperature %f seems out of reasonable range", telemetry.TempUvLed)
	}
}

func TestSLPrintStates(t *testing.T) {
	// Test SL-specific print states
	stateTests := []struct {
		name          string
		stateJSON     string
		expectedText  string
		expectedReady bool
	}{
		{
			name: "Ready state",
			stateJSON: `{
				"text": "Ready",
				"flags": {
					"operational": true,
					"ready": true,
					"printing": false,
					"paused": false,
					"error": false
				}
			}`,
			expectedText:  "Ready",
			expectedReady: true,
		},
		{
			name: "Printing state",
			stateJSON: `{
				"text": "Printing",
				"flags": {
					"operational": true,
					"ready": false,
					"printing": true,
					"paused": false,
					"error": false
				}
			}`,
			expectedText:  "Printing",
			expectedReady: false,
		},
		{
			name: "Error state",
			stateJSON: `{
				"text": "Error",
				"flags": {
					"operational": false,
					"ready": false,
					"printing": false,
					"paused": false,
					"error": true
				}
			}`,
			expectedText:  "Error",
			expectedReady: false,
		},
	}

	for _, tt := range stateTests {
		t.Run(tt.name, func(t *testing.T) {
			// Define state struct inline
			var state struct {
				Text  string `json:"text"`
				Flags struct {
					Operational bool `json:"operational"`
					Ready       bool `json:"ready"`
					Printing    bool `json:"printing"`
					Paused      bool `json:"paused"`
					Error       bool `json:"error"`
				} `json:"flags"`
			}

			err := json.Unmarshal([]byte(tt.stateJSON), &state)
			if err != nil {
				t.Errorf("Failed to unmarshal state: %v", err)
			}

			if state.Text != tt.expectedText {
				t.Errorf("State text = %s, expected %s", state.Text, tt.expectedText)
			}

			if state.Flags.Ready != tt.expectedReady {
				t.Errorf("State ready = %t, expected %t", state.Flags.Ready, tt.expectedReady)
			}
		})
	}
}

func BenchmarkSLJSONParsing(b *testing.B) {
	jobJSON := `{
		"state": "Printing",
		"job": {
			"estimatedPrintTime": 3600,
			"file": {
				"name": "test.sl1",
				"path": "/usb/test.sl1",
				"size": 1024000
			}
		},
		"progress": {
			"completion": 50.5,
			"printTime": 1800,
			"printTimeLeft": 1800
		}
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var job map[string]interface{}
		err := json.Unmarshal([]byte(jobJSON), &job)
		if err != nil {
			b.Errorf("Failed to unmarshal: %v", err)
		}
	}
}
