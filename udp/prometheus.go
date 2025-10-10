package udp

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

var (
	lastPush = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "prusa_last_push_timestamp",
			Help: "Last time the printer pushed metrics to the exporter.",
		},
		[]string{"printer_mac", "printer_address"},
	)
	udpRegistry *prometheus.Registry

	registryMetrics = safeRegistryMetrics{
		mu:      sync.Mutex{},
		metrics: make(map[string]*prometheus.GaugeVec),
	}
)

type safeRegistryMetrics struct {
	mu      sync.Mutex
	metrics map[string]*prometheus.GaugeVec
	labels  map[string][]string
}

// Init initializes the Prometheus udp registry.
func Init(udpMainRegistry *prometheus.Registry) {
	udpRegistry = udpMainRegistry

	udpRegistry.MustRegister(lastPush)
	registryMetrics.mu.Lock()
	registryMetrics.metrics = make(map[string]*prometheus.GaugeVec)
	registryMetrics.labels = make(map[string][]string)
	registryMetrics.metrics["last_push"] = lastPush
	registryMetrics.mu.Unlock()
}

func registerMetric(point point) {
	var metric *prometheus.GaugeVec

	for key, value := range point.Fields {
		metricName := point.Measurement
		tagLabels := getLabels(point.Tags)

		if key != "v" && key != "value" {
			metricName = metricName + "_" + key
		}

		registryMetrics.mu.Lock()
		if existingMetric, exists := registryMetrics.metrics[metricName]; exists {
			metric = existingMetric
		} else {
			// Create a new metric with the given point
			metric = prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: metricName,
					Help: "Metric for " + metricName + " from " + point.Measurement,
				},
				tagLabels,
			)
			if err := udpRegistry.Register(metric); err != nil {
				log.Trace().Msgf("Metric already registered %s: %v", metricName, err) // not a neccessary and error
			}
			registryMetrics.metrics[metricName] = metric
			registryMetrics.labels[metricName] = tagLabels
		}

		labels := []string{}

		for _, label := range registryMetrics.labels[metricName] {
			labels = append(labels, point.Tags[label])

		}

		registryMetrics.mu.Unlock()
		metric.WithLabelValues(labels...).Set(toFloat64(value))

	}
}

func getLabels(tags map[string]string) []string {
	labels := make([]string, 0, len(tags))
	for key := range tags {
		labels = append(labels, key)
	}
	return labels
}

func toFloat64(value interface{}) float64 {
	switch v := value.(type) {
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case float64:
		return v
	case bool:
		if v {
			return 1.0
		}
		return 0.0
	case nil:
		log.Warn().Msg("Received nil value, returning 0.0")
		return 0.0
	case string:
		if v == "PLA" {
			return 1.0
		} else if v == "PETG" {
			return 2.0
		} else if v == "ASA" {
			return 3.0
		} else if v == "PC" {
			return 4.0
		} else if v == "PVB" {
			return 5.0
		} else if v == "ABS" {
			return 6.0
		} else if v == "HIPS" {
			return 7.0
		} else if v == "PP" {
			return 8.0
		} else if v == "FLEX" {
			return 9.0
		} else if v == "PA" {
			return 10.0
		} else if v == "---" {
			return -1.0 // special case for "---" to indicate no loaded filament
		} else {
			return 0.0 // return for custom
		}
	default:
		log.Warn().Msgf("Unsupported type %T for value %v", value, value)
		return 0.0
	}
}
