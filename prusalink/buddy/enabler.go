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
	localIP       string
	listOfMetrics = []string{
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
)

// getLocalIP finds and returns the first non-loopback local IP address.
func getLocalIP() (string, error) {

	if localIP != "" {
		return localIP, nil
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
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
				return ip.String(), nil
			}
		}
	}
	return "", fmt.Errorf("could not find a valid local IP address")
}

func gcodeInit() (init string, err error) {
	ip, err := getLocalIP()
	if err != nil {
		return "", fmt.Errorf("failed to get local IP address: %v", err)
	}

	var builder strings.Builder

	// Write the initial lines
	builder.WriteString(fmt.Sprintf("M330 SYSLOG\nM334 %s 8514", ip))

	// Loop through the list of metrics and append each line
	for _, metric := range listOfMetrics {
		builder.WriteString(fmt.Sprintf("\nM331 %s", metric))
	}

	return builder.String(), nil

}

func sendGcode(filename string, printer config.Printers) ([]byte, error) {

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
		Timeout: 5 * time.Duration(configuration.Exporter.ScrapeTimeout) * time.Second,
	}

	// Create a new PUT request
	req, err := http.NewRequest(http.MethodPut, url, payload)
	if err != nil {
		return nil, fmt.Errorf("error creating PUT request: %w", err)
	}

	// Set a Content-Type header if needed
	req.Header.Set("Content-Type", "text/x.gcode")

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
		Timeout: 5 * time.Duration(configuration.Exporter.ScrapeTimeout) * time.Second,
	}
	res, err = client.Post(url, "application/json", nil)

	if err != nil {
		return result, err
	}

	result, err = io.ReadAll(res.Body)
	res.Body.Close()

	if err != nil {
		log.Error().Msg(err.Error())
	}

	return result, nil
}

func EnableUDPmetrics(printers []config.Printers) error {
	var wg sync.WaitGroup
	for _, s := range printers {
		wg.Add(1)
		go func(s config.Printers) {
			defer wg.Done()

			log.Debug().Msg("Enabling UDP metrics at " + s.Address)

			send, err := sendGcode("enable_udp_metrics.gcode", s)

			if err != nil {
				log.Error().Msg("Failed to send gcode to " + s.Address + ": " + err.Error())
				return
			}
			log.Debug().Msg("Gcode sent to " + s.Address + ": " + string(send))

			start, err := startGcode("enable_udp_metrics.gcode", s)

			if err != nil {
				log.Error().Msg("Failed to start gcode at " + s.Address + ": " + err.Error())
				return
			}
			log.Debug().Msg("Gcode started at " + s.Address + ": " + string(start))

			s.UDPMetricsEnabled = true
			log.Info().Msgf("UDP metrics enabled for printer %s (%s)", s.Name, s.Address)
		}(s)
	}
	wg.Wait()

	return nil
}
