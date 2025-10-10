package config

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// Config struct for the configuration file prusa.yml
type Config struct {
	Exporter struct {
		ScrapeTimeout int    `yaml:"scrape_timeout"`
		LogLevel      string `yaml:"log_level"`
		IpOverride    string
		AllMetricsUDP bool
		ExtraMetrics  []string
		LokiPushURL   string
	} `yaml:"exporter"`
	Printers  []Printers `yaml:"printers"`
	PrusaLink struct {
		CommonLabels   []string `yaml:"common_labels"`
		DisableMetrics []string `yaml:"disable_metrics"`
	} `yaml:"prusalink"`
}

// Printers struct containing the printer configuration
type Printers struct {
	Address           string `yaml:"address"`
	Username          string `yaml:"username,omitempty"`
	Password          string `yaml:"password,omitempty"`
	Apikey            string `yaml:"apikey,omitempty"`
	Name              string `yaml:"name,omitempty"`
	Type              string `yaml:"type,omitempty"`
	Reachable         bool
	UDPMetricsEnabled bool
}

// LoadConfig function to load and parse the configuration file
func LoadConfig(path string, prusaLinkScrapeTimeout int, udpIpOverride string, udpAllMetrics bool, udpExtraMetrics string, lokiPushUrl string) (Config, error) {
	var config Config
	file, err := os.ReadFile(path)

	if err != nil {
		return config, err
	}

	if err := yaml.Unmarshal(file, &config); err != nil {
		return config, err
	}
	config.Exporter.ScrapeTimeout = prusaLinkScrapeTimeout
	if udpIpOverride != "" {
		config.Exporter.IpOverride = udpIpOverride
		log.Info().Msgf("Overriding IP address for UDP metrics: %s", udpIpOverride)
	}

	config.Exporter.AllMetricsUDP = udpAllMetrics
	if udpAllMetrics {
		log.Warn().Msg("Exposing all UDP metrics. This will severely impact CPU capabilities of the printer!")
	}

	if udpExtraMetrics != "" {
		splitMetrics := strings.Split(udpExtraMetrics, ",")
		config.Exporter.ExtraMetrics = splitMetrics
		log.Info().Msgf("Adding extra UDP metrics: %v", splitMetrics)
	}

	config.Exporter.LokiPushURL = lokiPushUrl
	if lokiPushUrl == "" {
		log.Debug().Msgf("Loki push URL not set, image will not be pushed to Loki")
	}

	return config, err
}

// GetLogLevel function to parse the log level for zerolog
func GetLogLevel(level string) zerolog.Level {
	switch level {
	case "info":
		return zerolog.InfoLevel
	case "debug":
		return zerolog.DebugLevel
	case "trace":
		return zerolog.TraceLevel
	case "error":
		return zerolog.ErrorLevel
	case "panic":
		return zerolog.PanicLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}
