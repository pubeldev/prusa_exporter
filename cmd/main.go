package cmd

import (
	"net/http"
	"os"
	"strconv"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/pstrobl96/prusa_exporter/config"
	prusalink "github.com/pstrobl96/prusa_exporter/prusalink/buddy"
	udp "github.com/pstrobl96/prusa_exporter/udp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	configFile             = kingpin.Flag("config.file", "Configuration file for prusa_exporter.").Default("./prusa.yml").ExistingFile()
	metricsPath            = kingpin.Flag("exporter.metrics-path", "Path where to expose Prusa Link metrics.").Default("/metrics/prusalink").String()
	udpMetricsPath         = kingpin.Flag("exporter.udp-metrics-path", "Path where to expose udp metrics.").Default("/metrics/udp").String()
	metricsPort            = kingpin.Flag("exporter.metrics-port", "Port where to expose metrics.").Default("10009").Int()
	ipOverride             = kingpin.Flag("exporter.ip-override", "Override the IP address of the server with this value.").Default("").String()
	prusaLinkScrapeTimeout = kingpin.Flag("prusalink.scrape-timeout", "Timeout in seconds to scrape prusalink metrics.").Default("10").Int()
	logLevel               = kingpin.Flag("log.level", "Log level for zerolog.").Default("info").String()
	syslogListenAddress    = kingpin.Flag("listen-address", "Address where to expose port for gathering metrics. - format <address>:<port>").Default("0.0.0.0:8514").String()
	udpPrefix              = kingpin.Flag("prefix", "Prefix for udp metrics").Default("prusa_").String()
	udpRegistry            = prometheus.NewRegistry()
)

// Run function to start the exporter
func Run() {
	kingpin.Parse()
	log.Info().Msg("Prusa exporter starting")

	if *udpMetricsPath == *metricsPath {
		log.Panic().Msg("udp_metrics_path must be different from metrics_path")
	}

	if _, err := os.Stat(*configFile); os.IsNotExist(err) {
		log.Panic().Msg("Configuration file does not exist: " + *configFile)
	}

	log.Info().Msg("Loading configuration file: " + *configFile)

	config, err := config.LoadConfig(*configFile, *prusaLinkScrapeTimeout, *ipOverride)

	if err != nil {
		log.Panic().Msg("Error loading configuration file " + err.Error())
	}

	logLevel, err := zerolog.ParseLevel(*logLevel)

	if err != nil {
		logLevel = zerolog.InfoLevel // default log level
	}
	zerolog.SetGlobalLevel(logLevel)

	var collectors []prometheus.Collector

	log.Info().Msg("PrusaLink metrics enabled!")
	collectors = append(collectors, prusalink.NewCollector(config))
	prusalink.EnableUDPmetrics(config.Printers)

	// starting syslog server

	log.Info().Msg("Syslog server starting at: " + *syslogListenAddress)
	go udp.MetricsListener(*syslogListenAddress, *udpPrefix)
	log.Info().Msg("Syslog server ready to receive metrics")

	// registering the prometheus metrics

	prometheus.MustRegister(collectors...)
	log.Info().Msg("Metrics registered")
	http.Handle(*metricsPath, promhttp.Handler())
	log.Info().Msg("PrusaLink metrics initialized")

	udp.Init(udpRegistry)

	http.Handle(*udpMetricsPath, promhttp.HandlerFor(udpRegistry, promhttp.HandlerOpts{
		Registry: udpRegistry,
	}))
	log.Info().Msg("UDP metrics initialized")

	log.Info().Msg("Listening at port: " + strconv.Itoa(*metricsPort))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
    <head><title>prusa_exporter 2.0.0-alpha2</title></head>
    <body>
    <h1>prusa_exporter</h1>
	<p>Syslog server running at - <b>` + *syslogListenAddress + `</b></p>
    <p><a href="` + *metricsPath + `">PrusaLink metrics</a></p>
	<p><a href="` + *udpMetricsPath + `">UDP Metrics</a></p>
	</body>
    </html>`))
	})

	log.Fatal().Msg(http.ListenAndServe(":"+strconv.Itoa(*metricsPort), nil).Error())

}
