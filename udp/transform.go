package udp

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"gopkg.in/mcuadros/go-syslog.v2/format"
)

// AddrResolver maps syslog source IPs to canonical printer_address label
// values. Configured addresses (from prusa.yml) take priority over reverse DNS.
// Populate with SetAddress before starting any goroutine that calls Resolve.
type AddrResolver struct {
	// addresses is written only at startup (before concurrent Resolve calls).
	addresses map[string]string
	// cache stores DNS-fallback results for IPs not in addresses.
	cache sync.Map
}

// NewAddrResolver returns a ready-to-use AddrResolver.
func NewAddrResolver() *AddrResolver {
	return &AddrResolver{addresses: make(map[string]string)}
}

// SetAddress registers the canonical label for a printer IP.
// Must be called before any concurrent Resolve calls.
func (r *AddrResolver) SetAddress(ip, address string) {
	r.addresses[ip] = address
}

// Resolve returns the canonical printer_address label for a syslog source IP.
// Priority: configured address > reverse DNS short hostname > raw IP.
// DNS-fallback results are cached after the first lookup.
func (r *AddrResolver) Resolve(ip string) string {
	// Configured address always wins — not cached, so a new SetAddress call is
	// always visible without needing to invalidate any cache.
	if addr, ok := r.addresses[ip]; ok {
		return addr
	}
	// Check cache for previously resolved DNS-fallback result.
	if v, ok := r.cache.Load(ip); ok {
		return v.(string)
	}
	// Fall back to reverse DNS short hostname.
	result := ip
	names, err := net.LookupAddr(ip)
	if err == nil && len(names) > 0 {
		host := strings.TrimSuffix(names[0], ".")
		if i := strings.IndexByte(host, '.'); i != -1 {
			host = host[:i]
		}
		if host != "" {
			result = host
		}
	}
	r.cache.Store(ip, result)
	return result
}

type point struct {
	Measurement string
	Tags        map[string]string
	Fields      map[string]interface{} // Use interface{} to handle different field types
}

func process(data format.LogParts, prefix string, resolver *AddrResolver) {
	mac, ip, err := processIdentifiers(data)
	if err != nil {
		log.Error().Msg(fmt.Sprintf("Error processing identifiers: %v", err))
		return
	}
	// Strip port once here; handles both IPv4 (1.2.3.4:port) and IPv6 ([::1]:port).
	host, _, err := net.SplitHostPort(ip)
	if err != nil {
		host = ip // already a bare address
	}
	addr := resolver.Resolve(host)
	lastPush.WithLabelValues(mac, addr).Set(float64(time.Now().Unix())) // Set the last push timestamp

	log.Debug().Msg(fmt.Sprintf("Processing data for printer %s", mac))
	metrics, err := processMessage(data["message"].(string), mac, prefix, addr)
	if err != nil {
		log.Error().Msg(fmt.Sprintf("Error processing message: %v", err))
		return
	}

	for _, line := range metrics {
		point, err := parseLineProtocol(line)
		if err != nil {
			log.Debug().Msgf("Error parsing line '%s': %v", line, err) // printer sends error with several measurements - tmc_read returns "value_too_long" as well as some raw output data
			continue
		}

		registerMetric(*point) // Register the metric with the udp registry

	}
}

// processIdentifiers returns the MAC address and ip from the ingested data
func processIdentifiers(data format.LogParts) (string, string, error) {
	mac, ok := data["hostname"].(string)
	if !ok {
		return "", "", fmt.Errorf("mac is not an string")
	}

	ip, ok := data["client"].(string)
	if !ok {
		return "", "", fmt.Errorf("ip is not an string")
	}

	return mac, ip, nil
}

func processMessage(message string, mac string, prefix string, addr string) ([]string, error) {
	messageSplit := strings.Split(message, "\n")

	if len(messageSplit) == 0 {
		return nil, fmt.Errorf("message is empty")
	}

	firstMessage, err := parseFirstMessage(messageSplit[0])

	if err != nil {
		return nil, fmt.Errorf("error parsing first message: %v", err)
	}

	messageSplit = append(messageSplit[1:], firstMessage)

	for i, line := range messageSplit {
		splitted := strings.Split(line, " ")
		splitted, err = updateMetric(splitted, prefix, mac, addr)
		if err != nil {
			log.Error().Msg("Expected error while adding mac label for metric: " + splitted[0] + " error:" + err.Error())
			continue
		}
		messageSplit[i] = strings.Join(splitted, " ")
	}
	return messageSplit, nil
}

func parseFirstMessage(message string) (string, error) {
	splitted := strings.Split(message, " ")
	if len(splitted) == 0 {
		return "", fmt.Errorf("splitted message is empty")
	}
	firstMsg := splitted[1:]
	return strings.Join(firstMsg, " "), nil
}

func updateMetric(splitted []string, prefix string, mac string, addr string) ([]string, error) {
	if len(splitted) == 0 {
		return nil, fmt.Errorf("splitted message is empty")
	}

	splitted[0] = fmt.Sprintf("%s%s,printer_mac=%s,printer_address=%s", prefix, splitted[0], mac, addr)
	return splitted, nil
}

func newPoint() *point {
	return &point{
		Tags:   make(map[string]string),
		Fields: make(map[string]interface{}),
	}
}

func parseLineProtocol(line string) (*point, error) {
	p := newPoint()

	parts := splitLine(line)
	if len(parts) < 2 || len(parts) > 3 {
		return nil, fmt.Errorf("invalid udp format: %s", line) // this happens when printer sends error message
	}

	measurementTags := parts[0]
	measurementTagParts := strings.Split(measurementTags, ",")
	p.Measurement = measurementTagParts[0]

	for i := 1; i < len(measurementTagParts); i++ {
		tag := measurementTagParts[i]
		tagParts := strings.SplitN(tag, "=", 2)
		if len(tagParts) != 2 {
			return nil, fmt.Errorf("invalid tag format: %s", tag)
		}
		p.Tags[tagParts[0]] = tagParts[1]
	}

	fieldStr := parts[1]
	fieldParts := strings.Split(fieldStr, ",")
	for _, field := range fieldParts {
		kv := strings.SplitN(field, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid field format: %s", field)
		}
		key := kv[0]
		val := kv[1]

		// parsing metrics as different data types

		if strings.HasSuffix(val, "i") { // Integer
			if iVal, err := strconv.ParseInt(val[:len(val)-1], 10, 64); err == nil {
				p.Fields[key] = iVal
				continue
			}
		}

		if bVal, err := strconv.ParseBool(val); err == nil { // boolean
			p.Fields[key] = bVal
			continue
		}

		if fVal, err := strconv.ParseFloat(val, 64); err == nil { // float
			p.Fields[key] = fVal
			continue
		}

		if strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"") { // string
			p.Fields[key] = val[1 : len(val)-1]
			continue
		}

		// fallback
		p.Fields[key] = val
	}

	return p, nil
}

func splitLine(s string) []string {
	r := []string{}

	// Run a simple finite state machine to handle quoted strings and escaped characters.
	type State int
	const (
		Normal = iota
		QuotedString
	)

	start := 0
	state := Normal
	for i := 0; i < len(s); i++ {
		switch state {
		case Normal:
			switch s[i] {
			case ' ':
				if start < i {
					r = append(r, s[start:i])
				}
				start = i + 1

			case '\\':
				i++ // just skip the next character

			case '"':
				state = QuotedString
			}

		case QuotedString:
			switch s[i] {
			case '\\':
				i++ // just skip the next character

			case '"':
				state = Normal
			}
		}
	}
	if start < len(s) {
		r = append(r, s[start:])
	}

	return r
}
