package prusalink

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// PushImageToLoki pushes a base64-encoded job image to Grafana Loki as a log entry.
func PushImageToLoki(lokiURL, printerAddress, printerModel, printerName, printerJobName, printerJobPath, image string) error {
	// Prepare the log line with base64 image
	logLine := map[string]interface{}{
		"streams": []map[string]interface{}{
			{
				"stream": map[string]string{
					"job":              "prusa_job_image",
					"printer_ip":       printerAddress,
					"printer_model":    printerModel,
					"printer_name":     printerName,
					"printer_job_name": printerJobName,
					"printer_job_path": printerJobPath,
				},
				"values": [][]string{
					{
						fmt.Sprintf("%d000000000", time.Now().Unix()), // nanoseconds
						image,
					},
				},
			},
		},
	}

	payload, err := json.Marshal(logLine)
	if err != nil {
		return fmt.Errorf("failed to marshal log line: %w", err)
	}

	req, err := http.NewRequest("POST", lokiURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to Loki: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("loki returned status: %s", resp.Status)
	}

	return nil
}
