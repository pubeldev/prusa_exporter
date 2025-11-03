package prusalink

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/icholy/digest"
	"github.com/pstrobl96/prusa_exporter/config"
	"github.com/rs/zerolog/log"
)

var (
	listOfMetrics = []string{ // default metrics to enable - contains all metrics for Mini / MK4 / Core One and XL
		"adj_z",
		"temp_ambient",
		"temp_bed",
		"temp_brd",
		"temp_chamber",
		"temp_mcu",
		"temp_noz",
		"temp_hbr",
		"temp_psu",
		"temp_sandwich",
		"temp_splitter",
		"dwarf_mcu_temp",
		"dwarf_board_temp",
		"buddy_temp",
		"bedlet_temp",
		"bed_mcu_temp",
		"chamber_temp",
		"ttemp_noz",
		"ttemp_bed",
		"chamber_ttemp",
		"cur_mmu_imp",
		"curr_inp",
		"Sandwitch5VCurrent",
		"splitter_5V_current",
		"bed_curr",
		"bedlet_curr",
		"curr_nozz",
		"dwarf_heat_curr",
		"xlbuddy5VCurrent",
		"eth_in",
		"eth_out",
		"esp_in",
		"esp_out",
		"volt_bed",
		"volt_nozz",
		"24VVoltage",
		"5VVoltage",
		"loadcell_value",
		"fan",
		"fan_hbr_speed",
		"fan_speed",
		"xbe_fan",
		"print_fan_act",
		"hbr_fan_act",
		"hbr_fan_enc",
		"cpu_usage",
		"heap",
		"heap_free",
		"heap_total",
		"fsensor",
		"door_sensor",
		"fw_version",
		"buddy_revision",
		"buddy_bom",
	}
)

// getLocalIP finds and returns the first ethernet or WiFi IP address, avoiding Docker interfaces.
func getLocalIP() (string, error) {

	if configuration.Exporter.IPOverride != "" {
		return configuration.Exporter.IPOverride, nil
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	// Helper function to check if interface name suggests it's ethernet or WiFi
	isPreferredInterface := func(name string) bool {
		name = strings.ToLower(name)
		// Common ethernet interface names
		if strings.HasPrefix(name, "eth") || strings.HasPrefix(name, "en") ||
			strings.HasPrefix(name, "eno") || strings.HasPrefix(name, "enp") ||
			strings.HasPrefix(name, "ens") || strings.HasPrefix(name, "enx") {
			return true
		}
		// Common WiFi interface names
		if strings.HasPrefix(name, "wlan") || strings.HasPrefix(name, "wlp") ||
			strings.HasPrefix(name, "wl") || strings.HasPrefix(name, "wifi") {
			return true
		}
		return false
	}

	// Helper function to check if interface should be avoided (Docker, etc.)
	isAvoidedInterface := func(name string) bool {
		name = strings.ToLower(name)
		return strings.HasPrefix(name, "docker") || strings.HasPrefix(name, "br-") ||
			strings.HasPrefix(name, "veth") || name == "bridge0"
	}

	var fallbackIP string

	// First pass: look for preferred interfaces (ethernet/WiFi)
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		if isAvoidedInterface(iface.Name) {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip == nil || ip.IsLoopback() {
				continue
			}

			ip = ip.To4()
			if ip != nil {
				ipStr := ip.String()
				if isPreferredInterface(iface.Name) {
					// Found preferred interface, return immediately
					return ipStr, nil
				}
				// Keep as fallback if no preferred interface found
				if fallbackIP == "" {
					fallbackIP = ipStr
				}
			}
		}
	}

	// If we found a fallback IP (non-Docker, non-preferred), use it
	if fallbackIP != "" {
		return fallbackIP, nil
	}

	return "", fmt.Errorf("could not find a valid local IP address")
}

func gcodeInit() (init string, err error) {
	var builder strings.Builder

	ip, err := getLocalIP()
	if err != nil {
		return "", fmt.Errorf("failed to get local IP address: %v", err)
	}

	// Write the initial lines
	builder.WriteString(fmt.Sprintf("M330 SYSLOG\nM334 %s 8514", ip))

	if configuration.Exporter.AllMetricsUDP {
		for _, metric := range allMetricsList {
			builder.WriteString(fmt.Sprintf("\nM331 %s", metric))
		}
		return builder.String(), nil
	}

	for _, metric := range allMetricsList {
		builder.WriteString(fmt.Sprintf("\nM332 %s", metric)) // disable all metrics first for ease the life of the MCU
	}

	if len(configuration.Exporter.ExtraMetrics) > 0 {
		log.Info().Msgf("Adding extra UDP metrics: %v", configuration.Exporter.ExtraMetrics)
		listOfMetrics = append(listOfMetrics, configuration.Exporter.ExtraMetrics...)
	}

	// Loop through the list of metrics and append each line
	for _, metric := range listOfMetrics {
		builder.WriteString(fmt.Sprintf("\nM331 %s", metric))
	}

	return builder.String(), nil

}

func sendGcode(filename string, printer config.Printers) ([]byte, error) {

	deleteGcode(filename, printer) // ignore error, file might not exist

	gcode, err := gcodeInit()
	if err != nil {
		return nil, fmt.Errorf("error creating gcode init: %w", err)
	}

	payload := strings.NewReader(gcode)

	url := fmt.Sprintf("http://%s/api/v1/files/usb//%s", printer.Address, filename)

	client := &http.Client{
		Transport: &digest.Transport{
			Username: printer.Username,
			Password: printer.Password,
		},
		Timeout: time.Duration(configuration.Exporter.ScrapeTimeout) * time.Second,
	}

	// Create a new PUT request
	req, err := http.NewRequest(http.MethodPut, url, payload)
	if err != nil {
		return nil, fmt.Errorf("error creating PUT request: %w", err)
	}

	// Set a Content-Type header if needed
	req.Header.Set("Content-Type", "text/x.gcode")
	req.Header.Set("Overwrite", "?1")

	// Send the request
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending PUT request: %w", err)
	}
	defer res.Body.Close()

	// Read the response body
	result, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	return result, nil
}

func deleteGcode(filename string, printer config.Printers) ([]byte, error) {

	url := fmt.Sprintf("http://%s/api/v1/files/usb//%s", printer.Address, filename)

	client := &http.Client{
		Transport: &digest.Transport{
			Username: printer.Username,
			Password: printer.Password,
		},
		Timeout: time.Duration(configuration.Exporter.ScrapeTimeout) * time.Second,
	}

	// Create a new DELETE request. The third argument is nil as DELETE requests do not have a body.
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating DELETE request: %w", err)
	}

	// Send the request.
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending DELETE request: %w", err)
	}
	defer res.Body.Close()

	// Read the response body.
	result, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// Return the response body from the server.
	return result, nil
}

func startGcode(filename string, printer config.Printers) ([]byte, error) {
	url := fmt.Sprintf("http://%s/api/v1/files/usb//%s", printer.Address, filename)
	var (
		res    *http.Response
		result []byte
		err    error
	)

	client := &http.Client{
		Transport: &digest.Transport{
			Username: printer.Username,
			Password: printer.Password,
		},
		Timeout: time.Duration(configuration.Exporter.ScrapeTimeout) * time.Second,
	}
	res, err = client.Post(url, "application/json", nil)

	if err != nil {
		return result, err
	}

	if res.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("failed to start gcode file, status code: %d", res.StatusCode)
	}
	result, err = io.ReadAll(res.Body)
	res.Body.Close()

	if err != nil {
		log.Error().Msg(err.Error())
	}

	return result, nil
}

// EnableUDPmetrics enables UDP metrics on all printers concurrently
func EnableUDPmetrics(printers []config.Printers) {
	var wg sync.WaitGroup

	for i, s := range printers {
		wg.Add(1)
		go func(i int, s config.Printers) {
			defer wg.Done()
			log.Debug().Msg("Enabling UDP metrics at " + s.Address)

			send, err := sendGcode("enable_udp_metrics.gcode", s)

			if err != nil {
				log.Error().Msg("Failed to send gcode to " + s.Address + ": " + err.Error())
				configuration.Printers[i].UDPMetricsEnabled = false
				return
			}
			log.Debug().Msg("Gcode sent to " + s.Address + ": " + string(send))

			start, err := startGcode("enable_udp_metrics.gcode", s)

			if err != nil {
				log.Error().Msg("Failed to start gcode at " + s.Address + ": " + err.Error())
				configuration.Printers[i].UDPMetricsEnabled = false
				return
			}
			log.Debug().Msg("Gcode started at " + s.Address + ": " + string(start))

			configuration.Printers[i].UDPMetricsEnabled = true
			log.Info().Msgf("UDP metrics gcode for printer %s (%s) sent and started", s.Name, s.Address)
		}(i, s)
	}
	wg.Wait()
}
