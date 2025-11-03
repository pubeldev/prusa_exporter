[![docker](https://img.shields.io/github/actions/workflow/status/pstrobl96/prusa_exporter/docker.yml)](https://github.com/pstrobl96/prusa_exporter/actions/workflows/docker.yml) 
![issues](https://img.shields.io/github/issues/pstrobl96/prusa_exporter) 
![go](https://img.shields.io/github/go-mod/go-version/pstrobl96/prusa_exporter) 
![tag](https://img.shields.io/github/v/tag/pstrobl96/prusa_exporter) 
![license](https://img.shields.io/github/license/pstrobl96/prusa_exporter)

# prusa_exporter

Prusa Exporter or more known as prusa_exporter is a tool that allows users to expose metrics from the Prusa Research FDM 3D printers. Its approach is to scrape metrics from [Prusa Link](https://help.prusa3d.com/article/prusa-connect-and-prusalink-explained_302608) REST API and also getting [UDP](https://github.com/prusa3d/Prusa-Firmware-Buddy/blob/master/doc/metrics.md) type of metrics. After gettng data it's simply exposes the metrics at `/metrics/prusalink` and `/metrics/udp` endpoints. You can also access `http://localhost:10009`.

**I strongly recommend to connect printers via Ethernet as WiFi is not considered stable**

**UDP** is configured in printer - Settings -> Network -> Metrics & Log

By default at start of the exporter it will send the configuration gcode to the printers that are configured in prusa.yml. 

**BEWARE** - Altrough Prusa Mini sends some metrics via UDP as well, it's board does not contain needed sensors. So that means you are basically unable to get anything meaningful from those metrics. 

- Host => address where prusa_exporter is running aka your computer / server
- Metrics Port => default 8514 same as prusa_exporter but you can change it
- Enable Metrics => enable
- Metrics List => list of enabled metrics
  - You can select all but it has actual impact on performance so choose wisely


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

### Dashboards

I've prepared cozy [dashboards](docs/), but this being Prometheus, you can do whatever you want. Fun fact, Mini dashboard works for MKx and Core One and MKx dashboard works for Core One but not vice versa. XL dashboard is specific for XL.

#### Prusa MKx (MK3.5(S) / MK3.9(S) / MK4(S))
![mkxdashboard](docs/readme/mkx_dashboard.png)

#### Prusa Core One
![c1dashboard](docs/readme/c1_dashboard.png)

#### Prusa XL

PLACEHOLDER

#### Prusa Mini
![minidashboard](docs/readme/mini_dashboard.png)

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
- [x] ~~compress image of print~~ - just use Loki for image
- [x] ~~rename udp metrics~~ - keeping old names for compatibility with metrics_handler
- [x] check PrusaLink metrics - done by ([imax9000](https://github.com/imax9000)) 

alpha4
- [x] further optimization
- [x] ~~PoC controlling printer via Grafana~~ - PoC work but it's flawed - scrapping - [prusa_proxy](https://github.com/pubeldev/prusa_proxy) - I've used skeleton in sending metrics gcode
- [x] Mini dashboard
- [x] MKx dashboard
- [x] Core One dashboard
- [ ] XL dashboard

beta1
- [x] ~~start testing at Raspberry Pi 4 (if not feasible then 5)~~ - not going to build Raspberry Pi image
- [x] create tests
- [x] reenable tests in pipeline

beta2
- [ ] improve stability and optimize code
- [x] ~~finalize controlling printer via Grafana~~

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

### My printer has UDP metrics host set as 172.x.x.x

That is because you haven't stared the docker compose with `start_docker.sh` or `start_docker.bat`. This script will export HOST_IP address and `docker-compose` will pass it into exporter.

### Can I enable all UDP metrics?

Yes, but you may face the issue that MCU of the printer will not keep up and it can cause issues with printing.

### Can I get also logs from the printer?

Technically yes but that is not the scode of prusa_exporter. Use [prusa_log_processor](https://github.com/pubeldev/prusa_log_processor) instead.

### Why is there Loki in docker compose? Isn't this exporter of metrics?

Well, yes. But I wanted expose png of actual print file so I've ended with using Loki. It's not necessary and you can disable by just not setting loki.push-url or setting it to `""`.

### I don't have Prusa Link metrics. Why?

Double check the prusa.yml for typos.

### Why do I have to fill printer model in configuration file?

At this moment it's not possible to get information what printer is running at the defined endpoint. So I don't even use this information in the code. If in the future will the information be available I'll use it to generate compatible metrics gcode and the `prusa.yml` will be bit easier to configure.

### Is the docker compose production ready?

Absolutely not, you can take it as example but you should properly configure Loki, Grafana and Prometheus.

### What metrics are better? UDP or Prusa Link

UDP metrics have by default 1 second of resultion while Prusa Link are scraped every 60 seconds. So because of this I'd say UDP are better but they have issue with UDP being UDP and sometimes they will just not work. 

### Can I run the exporter on Raspberry Pi?

Yes, I build binaries for 

Linux
- amd64
- arm64
- rics64
- arm

macOS
- arm64
- amd64

Windows
- arm64
- amd64

And images for

- linux/amd64
- linux/arm64
- linux/arm/v7

### What UDP metrics are enabled by default?

- temp_ambient
- temp_bed
- temp_brd
- temp_chamber
- temp_mcu
- temp_noz
- temp_hbr
- temp_psu
- temp_sandwich
- temp_splitter
- dwarf_mcu_temp
- dwarf_board_temp
- buddy_temp
- bedlet_temp
- bed_mcu_temp
- chamber_temp
- ttemp_noz
- ttemp_bed
- chamber_ttemp
- curr_inp
- Sandwitch5VCurrent
- splitter_5V_current
- bed_curr
- bedlet_curr
- curr_nozz
- dwarf_heat_curr
- xlbuddy5VCurrent
- eth_in
- eth_out
- esp_in
- esp_out
- volt_bed
- volt_nozz
- 24VVoltage
- 5VVoltage
- loadcell_value
- fan
- fan_hbr_speed
- fan_speed
- xbe_fan
- print_fan_act
- hbr_fan_act
- hbr_fan_enc
- cpu_usage
- heap
- heap_free
- heap_total
- fsensor
- door_sensor