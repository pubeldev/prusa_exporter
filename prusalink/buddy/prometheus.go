package prusalink

import (
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/pstrobl96/prusa_exporter/config"
	"github.com/rs/zerolog/log"
)

// Collector is a struct of all printer metrics
type Collector struct {
	metricDesc     map[MetricName]*prometheus.Desc
	metricDisabled map[MetricName]bool

	configuration config.Config
	commonLabels  []string
}

type MetricName string

const (
	MetricPrinterTemp               MetricName = "prusa_temperature_celsius"
	MetricPrinterTempTarget                    = "prusa_temperature_target_celsius"
	MetricPrinterPrintTimeRemaining            = "prusa_printing_time_remaining_seconds"
	MetricPrinterPrintProgressRatio            = "prusa_printing_progress_ratio"
	MetricPrinterFiles                         = "prusa_files_count"
	MetricPrinterMaterial                      = "prusa_material_info"
	MetricPrinterPrintTime                     = "prusa_print_time_seconds"
	MetricPrinterUp                            = "prusa_up"
	MetricPrinterNozzleSize                    = "prusa_nozzle_size_meters"
	MetricPrinterStatus                        = "prusa_status_info"
	MetricPrinterAxis                          = "prusa_axis"
	MetricPrinterFlow                          = "prusa_print_flow_ratio"
	MetricPrinterInfo                          = "prusa_info"
	MetricPrinterMMU                           = "prusa_mmu"
	MetricPrinterFanSpeedRpm                   = "prusa_fan_speed_rpm"
	MetricPrinterPrintSpeedRatio               = "prusa_print_speed_ratio"
	MetricPrinterJobImage                      = "prusa_job_image"
	MetricPrinterCurrentJob                    = "prusa_job"
	MetricPrinterUDPMetricsEnabled             = "prusa_udp_metrics_enabled"
)

type metricDesc struct {
	Name        MetricName
	Description string
	Labels      []string
}

var metrics = []metricDesc{
	{MetricPrinterTemp, "Current temp of printer in Celsius", []string{"printer_heated_element"}},
	{MetricPrinterTempTarget, "Target temp of printer in Celsius", []string{"printer_heated_element"}},
	{MetricPrinterPrintTimeRemaining, "Returns time that remains for completion of current print", nil},
	{MetricPrinterPrintProgressRatio, "Returns information about completion of current print in ratio (0.0-1.0)", nil},
	{MetricPrinterFiles, "Number of files in storage", []string{"printer_storage"}},
	{MetricPrinterMaterial, "Returns information about loaded filament. Returns 0 if there is no loaded filament", []string{"printer_filament"}},
	{MetricPrinterPrintTime, "Returns information about current print time.", nil},
	{MetricPrinterNozzleSize, "Returns information about selected nozzle size.", nil},
	{MetricPrinterStatus, "Returns information status of printer.", []string{"printer_state"}},
	{MetricPrinterAxis, "Returns information about position of axis.", []string{"printer_axis"}},
	{MetricPrinterFlow, "Returns information about of filament flow in ratio (0.0 - 1.0).", nil},
	{MetricPrinterInfo, "Returns information about printer.", []string{"api_version", "server_version", "version_text", "prusalink_name", "printer_location", "serial_number", "printer_hostname"}},
	{MetricPrinterMMU, "Returns information if MMU is enabled.", nil},
	{MetricPrinterFanSpeedRpm, "Returns information about speed of hotend fan in rpm.", []string{"fan"}},
	{MetricPrinterPrintSpeedRatio, "Current setting of printer speed in values from 0.0 - 1.0", nil},
	{MetricPrinterJobImage, "Returns information about image of current print job.", []string{"printer_job_image"}},
}

// Unlike `metrics`, these ignore common labels.
var specialMetrics = []metricDesc{
	{MetricPrinterUp, "Return information about online printers. If printer is registered as offline then returned value is 0.", []string{"printer_address", "printer_model", "printer_name"}},
	{MetricPrinterUDPMetricsEnabled, "Return information if the UDP metrics were enabled successfully.", []string{"printer_address", "printer_model", "printer_name"}},

	{MetricPrinterCurrentJob, "Returns information about the current print job.", []string{"printer_address", "printer_model", "printer_name", "printer_job_name", "printer_job_path"}},
}

func (c *Collector) metricEnabled(m MetricName) bool {
	// Zero value is `false`, so if not set - the metric is enabled.
	return !c.metricDisabled[m]
}

// NewCollector returns a new Collector for printer metrics
func NewCollector(config config.Config) *Collector {
	configuration = config
	commonLabels := config.PrusaLink.CommonLabels
	if len(commonLabels) == 0 {
		commonLabels = []string{"printer_address", "printer_model", "printer_name", "printer_job_name", "printer_job_path"}
	}
	c := &Collector{
		configuration:  config,
		commonLabels:   commonLabels,
		metricDesc:     map[MetricName]*prometheus.Desc{},
		metricDisabled: map[MetricName]bool{},
	}

	for _, m := range metrics {
		c.metricDesc[m.Name] = prometheus.NewDesc(string(m.Name), m.Description, append(commonLabels, m.Labels...), nil)
	}
	for _, m := range specialMetrics {
		c.metricDesc[m.Name] = prometheus.NewDesc(string(m.Name), m.Description, m.Labels, nil)
	}

	for _, m := range config.PrusaLink.DisableMetrics {
		c.metricDisabled[MetricName(m)] = true
	}
	return c
}

// Describe implements prometheus.Collector
func (collector *Collector) Describe(ch chan<- *prometheus.Desc) {
	// Iterating over metrics instead of collector.metricDesc just to
	// preserve ordering. Not that it matters, but still.
	for _, m := range metrics {
		ch <- collector.metricDesc[m.Name]
	}
}

// Collect implements prometheus.Collector
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	var wg sync.WaitGroup
	for _, s := range c.configuration.Printers {
		wg.Add(1)
		go func(s config.Printers) {
			defer wg.Done()

			log.Debug().Msg("Printer scraping at " + s.Address)
			printerUp := prometheus.MustNewConstMetric(c.metricDesc[MetricPrinterUp], prometheus.GaugeValue,
				0, s.Address, s.Type, s.Name)

			job, err := GetJob(s)
			if err != nil {
				log.Error().Msg("Error while scraping job endpoint at " + s.Address + " - " + err.Error())
				ch <- printerUp
				return
			}

			printer, err := GetPrinter(s)
			if err != nil {
				log.Error().Msg("Error while scraping printer endpoint at " + s.Address + " - " + err.Error())
				ch <- printerUp
				return
			}

			version, err := GetVersion(s)
			if err != nil {
				log.Error().Msg("Error while scraping version endpoint at " + s.Address + " - " + err.Error())
				ch <- printerUp
				return
			}

			status, err := GetStatus(s)

			if err != nil {
				log.Error().Msg("Error while scraping status endpoint at " + s.Address + " - " + err.Error())
			}

			info, err := GetInfo(s)

			if err != nil {
				log.Error().Msg("Error while scraping info endpoint at " + s.Address + " - " + err.Error())
			}

			if c.metricEnabled(MetricPrinterInfo) {
				printerInfo := prometheus.MustNewConstMetric(
					c.metricDesc[MetricPrinterInfo], prometheus.GaugeValue,
					1,
					c.GetLabels(s, job, version.API, version.Server, version.Text, info.Name, info.Location, info.Serial, info.Hostname)...)

				ch <- printerInfo
			}

			if c.metricEnabled(MetricPrinterCurrentJob) {
				value := float64(1)
				if job.Job.File.Name == "" {
					value = 0
				}
				jobInfo := prometheus.MustNewConstMetric(c.metricDesc[MetricPrinterCurrentJob], prometheus.GaugeValue,
					value,
					s.Address, s.Type, s.Name, job.Job.File.Name, job.Job.File.Path)

				ch <- jobInfo
			}

			if c.metricEnabled(MetricPrinterFanSpeedRpm) {
				printerFanHotend := prometheus.MustNewConstMetric(c.metricDesc[MetricPrinterFanSpeedRpm], prometheus.GaugeValue,
					status.Printer.FanHotend, c.GetLabels(s, job, "hotend")...)

				ch <- printerFanHotend

				printerFanPrint := prometheus.MustNewConstMetric(c.metricDesc[MetricPrinterFanSpeedRpm], prometheus.GaugeValue,
					status.Printer.FanPrint, c.GetLabels(s, job, "print")...)

				ch <- printerFanPrint
			}

			if c.metricEnabled(MetricPrinterNozzleSize) {
				printerNozzleSize := prometheus.MustNewConstMetric(c.metricDesc[MetricPrinterNozzleSize], prometheus.GaugeValue,
					info.NozzleDiameter, c.GetLabels(s, job)...)

				ch <- printerNozzleSize
			}

			if c.metricEnabled(MetricPrinterPrintSpeedRatio) {
				printSpeed := prometheus.MustNewConstMetric(
					c.metricDesc[MetricPrinterPrintSpeedRatio], prometheus.GaugeValue,
					printer.Telemetry.PrintSpeed/100,
					c.GetLabels(s, job)...)

				ch <- printSpeed
			}

			if c.metricEnabled(MetricPrinterPrintTime) {
				printTime := prometheus.MustNewConstMetric(
					c.metricDesc[MetricPrinterPrintTime], prometheus.GaugeValue,
					job.Progress.PrintTime,
					c.GetLabels(s, job)...)

				ch <- printTime
			}

			if c.metricEnabled(MetricPrinterPrintTimeRemaining) {
				printTimeRemaining := prometheus.MustNewConstMetric(
					c.metricDesc[MetricPrinterPrintTimeRemaining], prometheus.GaugeValue,
					job.Progress.PrintTimeLeft,
					c.GetLabels(s, job)...)

				ch <- printTimeRemaining
			}

			if c.metricEnabled(MetricPrinterPrintProgressRatio) {
				printProgress := prometheus.MustNewConstMetric(
					c.metricDesc[MetricPrinterPrintProgressRatio], prometheus.GaugeValue,
					job.Progress.Completion,
					c.GetLabels(s, job)...)

				ch <- printProgress
			}

			if c.metricEnabled(MetricPrinterMaterial) {
				material := prometheus.MustNewConstMetric(
					c.metricDesc[MetricPrinterMaterial], prometheus.GaugeValue,
					BoolToFloat(!(strings.Contains(printer.Telemetry.Material, "-"))),
					c.GetLabels(s, job, printer.Telemetry.Material)...)

				ch <- material
			}

			if c.metricEnabled(MetricPrinterAxis) {
				printerAxisX := prometheus.MustNewConstMetric(
					c.metricDesc[MetricPrinterAxis], prometheus.GaugeValue,
					printer.Telemetry.AxisX,
					c.GetLabels(s, job, "x")...)

				ch <- printerAxisX

				printerAxisY := prometheus.MustNewConstMetric(
					c.metricDesc[MetricPrinterAxis], prometheus.GaugeValue,
					printer.Telemetry.AxisY,
					c.GetLabels(s, job, "y")...)

				ch <- printerAxisY

				printerAxisZ := prometheus.MustNewConstMetric(
					c.metricDesc[MetricPrinterAxis], prometheus.GaugeValue,
					printer.Telemetry.AxisZ,
					c.GetLabels(s, job, "z")...)

				ch <- printerAxisZ
			}

			if c.metricEnabled(MetricPrinterFlow) {
				printerFlow := prometheus.MustNewConstMetric(c.metricDesc[MetricPrinterFlow], prometheus.GaugeValue,
					status.Printer.Flow/100, c.GetLabels(s, job)...)

				ch <- printerFlow
			}

			if c.metricEnabled(MetricPrinterMMU) {
				printerMMU := prometheus.MustNewConstMetric(c.metricDesc[MetricPrinterMMU], prometheus.GaugeValue,
					BoolToFloat(info.Mmu), c.GetLabels(s, job)...)
				ch <- printerMMU
			}

			if c.metricEnabled(MetricPrinterTemp) {
				printerBedTemp := prometheus.MustNewConstMetric(c.metricDesc[MetricPrinterTemp], prometheus.GaugeValue,
					printer.Temperature.Bed.Actual, c.GetLabels(s, job, "bed")...)

				ch <- printerBedTemp

				printerToolTemp := prometheus.MustNewConstMetric(c.metricDesc[MetricPrinterTemp], prometheus.GaugeValue,
					printer.Temperature.Tool0.Actual, c.GetLabels(s, job, "tool0")...)

				ch <- printerToolTemp
			}

			if c.metricEnabled(MetricPrinterTempTarget) {
				printerBedTempTarget := prometheus.MustNewConstMetric(c.metricDesc[MetricPrinterTempTarget], prometheus.GaugeValue,
					printer.Temperature.Bed.Target, c.GetLabels(s, job, "bed")...)

				ch <- printerBedTempTarget

				printerToolTempTarget := prometheus.MustNewConstMetric(c.metricDesc[MetricPrinterTempTarget], prometheus.GaugeValue,
					printer.Temperature.Tool0.Target, c.GetLabels(s, job, "tool0")...)

				ch <- printerToolTempTarget
			}

			if c.metricEnabled(MetricPrinterStatus) {
				printerStatus := prometheus.MustNewConstMetric(
					c.metricDesc[MetricPrinterStatus], prometheus.GaugeValue,
					getStateFlag(printer),
					c.GetLabels(s, job, printer.State.Text)...)

				ch <- printerStatus
			}

			if c.metricEnabled(MetricPrinterJobImage) && getStateFlag(printer) == 4 {
				image, err := GetJobImage(s, job.Job.File.Path)

				if err != nil {
					log.Error().Msg("Error while scraping image endpoint at " + s.Address + " - " + err.Error())
				} else {
					printerJobImage := prometheus.MustNewConstMetric(c.metricDesc[MetricPrinterJobImage], prometheus.GaugeValue,
						1, c.GetLabels(s, job, image)...)

					ch <- printerJobImage
				}

			}

			printerUp = prometheus.MustNewConstMetric(c.metricDesc[MetricPrinterUp], prometheus.GaugeValue,
				1, s.Address, s.Type, s.Name)

			ch <- printerUp

			log.Debug().Msg("Scraping done at " + s.Address)
		}(s)
	}
	wg.Wait()
}

// GetLabels is used to get the labels for the given printer and job
func (c *Collector) GetLabels(printer config.Printers, job Job, labelValues ...string) []string {
	commonValues := make([]string, len(c.commonLabels), len(c.commonLabels)+len(labelValues))

	for i, l := range c.commonLabels {
		switch l {
		case "printer_address":
			commonValues[i] = printer.Address
		case "printer_model":
			commonValues[i] = printer.Type
		case "printer_name":
			commonValues[i] = printer.Name

		// job is passed by value, and none of the fields are pointers,
		// so we don't need to worry about nil dereferences.
		case "printer_job_name":
			commonValues[i] = job.Job.File.Name
		case "printer_job_path":
			commonValues[i] = job.Job.File.Path
		}
	}
	return append(commonValues, labelValues...)
}
