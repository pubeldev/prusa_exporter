[![docker](https://img.shields.io/github/actions/workflow/status/pstrobl96/prusa_exporter/docker.yml)](https://github.com/pstrobl96/prusa_exporter/actions/workflows/docker.yml) 
![issues](https://img.shields.io/github/issues/pstrobl96/prusa_exporter) 
![go](https://img.shields.io/github/go-mod/go-version/pstrobl96/prusa_exporter) 
![tag](https://img.shields.io/github/v/tag/pstrobl96/prusa_exporter) 
![license](https://img.shields.io/github/license/pstrobl96/prusa_exporter)

# prusa_exporter

Prusa Exporter or more known as prusa_exporter is a tool that allows users to expose metrics from the Prusa Research 3D printers. Its approach is to scrape metrics from [Prusa Link](https://help.prusa3d.com/article/prusa-connect-and-prusalink-explained_302608) REST API and also from [UDP](https://github.com/prusa3d/Prusa-Firmware-Buddy/blob/master/doc/metrics.md) type of metrics. After gettng data it's simply exposes the metrics at `/metrics/prusalink` and `/metrics/udp` endpoints. You can also access `http://localhost:10009`.

**I strongly recommend to connect printers via Ethernet as WiFi is not considered stable**

**UDP** is configured in printer - Settings -> Network -> Metrics & Log

**BEWARE** - Altrough Prusa Mini sends some metrics via UDP as well, it's board does not contain needed sensors. So that means you are basically unable to get anything meaningful from those metrics. 

- Host => address where prusa_exporter is running aka your computer / server
- Metrics Port => default 8514 same as prusa_exporter but you can change it
- Enable Metrics => enable
- Metrics List => list of enabled metrics
  - You can select all but it has actual impact on performance so choose wisely

List of metrics needed for dashboard (values differs between printers)
- ttemp_noz
- temp_noz
- ttemp_bed
- temp_bed
- chamber_temp
- temp_mcu
- temp_hbr
- loadcell_value
- curr_inp
- volt_bed
- eth_out
- eth_in

Of course you can configure metrics with gcode as well - that gcode can be found [here](docs/examples/syslog/config_full.gcode) as well

```
M330 SYSLOG
M334 192.168.20.20 8514
M331 ttemp_noz
M331 temp_noz
M331 ttemp_bed
M331 temp_bed
M331 chamber_temp
M331 temp_mcu
M331 temp_hbr
M331 loadcell_value
M331 curr_inp
M331 volt_bed
M331 eth_out
M331 eth_in
```

**Prusa Link** is configured with [prusa.yml](docs/config/prusa.yml) where you need to fill - Settings -> Network -> PrusaLink

- `address` of the printer
- `username` => default `maker`
- `password` for Prusa Link
- `name` of the printer
  - your chosen name => just use basic name non standard - type
- `type` - model of the printer
  - MK3.9 / MK4 / MK4S / XL / Core One ...

### Dashboard

Pretty basic but nice and cozy [dashboard](docs/Prusa_Metrics_MK4_C1.json) for TV.

![dashboard](docs/dashboard.png)

# Roadmap

omega2
- [x] working udp metrics with influx2cortex proxy
- [x] working PrusaLink metrics
- [x] development restarted 🎉

alpha1
- [x] transfering prusa_metrics_handler codebase into prusa_exporter
- [x] working UDP metrics via influxdb_exporter
- [x] Core One / MK4S dashboard

alpha2
- [x] working UDP metrics without any external tool
- [x] split UDP and PrusaLink metrics
- [x] update Go to 1.24
- [x] drop Einsy support
- [x] overall optimization
- [x] update dashboard for Core One / MK4S

alpha3
- [x] auto enable syslog metrics
- [x] create FAQ
- [x] ~~check if the address from the udp and prusalink metrics are the same~~ - there is an issue in the firmware. Even though printer should sent metrics via selected network, it can sent them via ESP32 if it's connected. And vice versa - it can sent metrics via Ethernet if the ESP32 is selected as network adapter.
- [ ] ~~compress image of print~~ - expose link to image instead
- [x] ~~rename udp metrics~~ - keeping old names for compatibility with metrics_handler
- [x] check PrusaLink metrics - done by ([imax9000](https://github.com/imax9000)) 
- [ ] XL dashboard

alpha4
- [ ] ~~PoC controlling printer via Grafana~~ - PoC work but it's flawed - scrapping
- [ ] Mini dashboard

beta1
- [ ] ~~start testing at Raspberry Pi 4 (if not feasible then 5)~~ - not going to build Raspberry Pi image
- [ ] create tests
- [ ] reenable tests in pipeline

beta2
- [ ] improve stability and optimize code
- [ ] ~~finalize controlling printer via Grafana~~

rc1
- [ ] create overview dashboard for all printers in system
- [ ] further testing

final
- [ ] 🎉

# FAQ

### My printer is correctly connected via Prusa Link API but I got no UDP metrics

After start of the exporter G-Code containing configuration is sent to the printer but it was not probably loaded properly. You can trigger reload by either restart of the exporter or you can run it manually on the printer.

### After expoterer started my printers returned Warning - The G-code isn't fully compatible 

This is correct - just click on PRINT and that's it. It's possible to technically avoid this but even then you will be informed that G-Code is changing metrics configuration so it's pointless.

