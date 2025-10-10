package prusalink

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"net/http"
	"time"

	"github.com/icholy/digest"
	"github.com/pstrobl96/prusa_exporter/config"
	"github.com/rs/zerolog/log"
)

var (
	/*printerBoards = map[string]string{
		"MINI":    "buddy",
		"MK35":    "buddy",
		"MK39":    "buddy",
		"MK4":     "buddy",
		"XL":      "buddy",
		"IX":      "buddy",
		"I3MK3S":  "einsy",
		"I3MK3":   "einsy",
		"I3MK25S": "einsy",
		"I3MK25":  "einsy",
		"SL1":     "sl",
		"SL1S":    "sl",
	}*/

	// used for autodetection - does not work with changed hostname :sad:
	printerTypes = map[string]string{
		"PrusaMINI":         "MINI",
		"PrusaMK4":          "MK4", // unfortunately MK3.5 is also detected as MK4
		"PrusaXL":           "XL",
		"PrusaLink I3MK3S":  "I3MK3S",
		"PrusaLink I3MK3":   "I3MK3",
		"PrusaLink I3MK25S": "I3MK25S",
		"PrusaLink I3MK25":  "I3MK25",
		"prusa-sl1":         "SL1",
		"prusa-sl1s":        "SL1S",
		"Prusa_iX":          "IX", // can be found in src/common/config.h in firmware source code
	}

	configuration config.Config
)

// BoolToFloat is used for basic parsing boolean to float64
// 0.0 for false, 1.0 for true
func BoolToFloat(boolean bool) float64 {
	if !boolean {
		return 0.0
	}

	return 1.0
}

// getStateFlag returns the state flag for the given printer.
// The state flag is a float64 value representing the current state of the printer.
// It is used for tracking the printer's status and progress.
func getStateFlag(printer Printer) float64 {
	if printer.State.Flags.Operational {
		return 1
	} else if printer.State.Flags.Prepared {
		return 2
	} else if printer.State.Flags.Paused {
		return 3
	} else if printer.State.Flags.Printing {
		return 4
	} else if printer.State.Flags.Cancelling {
		return 5
	} else if printer.State.Flags.Pausing {
		return 6
	} else if printer.State.Flags.Error {
		return 7
	} else if printer.State.Flags.SdReady {
		return 8
	} else if printer.State.Flags.ClosedOrError || printer.State.Flags.ClosedOnError {
		return 9
	} else if printer.State.Flags.Ready {
		return 10
	} else if printer.State.Flags.Busy {
		return 11
	} else if printer.State.Flags.Finished {
		return 12
	} else {
		return 0
	}
}

// accessPrinterEndpoint is used to access the printer's API endpoint
func accessPrinterEndpoint(path string, printer config.Printers) ([]byte, error) {
	url := string("http://" + printer.Address + path)
	var (
		res    *http.Response
		result []byte
		err    error
	)

	if printer.Apikey == "" {
		client := &http.Client{
			Transport: &digest.Transport{
				Username: printer.Username,
				Password: printer.Password,
			},
			Timeout: 5 * time.Duration(configuration.Exporter.ScrapeTimeout) * time.Second,
		}
		res, err = client.Get(url)

		if err != nil {
			return result, err
		}
	} else {
		req, err := http.NewRequest("GET", url, nil)
		client := &http.Client{
			Timeout: 5 * time.Duration(configuration.Exporter.ScrapeTimeout) * time.Second,
		}

		if err != nil {
			return result, err
		}

		req.Header.Add("X-Api-Key", printer.Apikey)
		res, err = client.Do(req)
		if err != nil {
			return result, err
		}
	}
	result, err = io.ReadAll(res.Body)
	res.Body.Close()

	if err != nil {
		log.Error().Msg(err.Error())
	}

	return result, nil
}

// GetVersion is used to get the printer's version API endpoint
func GetVersion(printer config.Printers) (Version, error) {
	var version Version
	response, err := accessPrinterEndpoint("/api/version", printer)

	if err != nil {
		return version, err
	}

	err = json.Unmarshal(response, &version)

	return version, err
}

// GetJob is used to get the printer's job API endpoint
func GetJob(printer config.Printers) (Job, error) {
	var job Job
	response, err := accessPrinterEndpoint("/api/job", printer)

	if err != nil {
		return job, err
	}

	err = json.Unmarshal(response, &job)

	return job, err
}

// GetPrinter is used to get the printer's printer API endpoint
func GetPrinter(printer config.Printers) (Printer, error) {
	var printerData Printer
	response, err := accessPrinterEndpoint("/api/printer", printer)

	if err != nil {
		return printerData, err
	}

	err = json.Unmarshal(response, &printerData)

	return printerData, err
}

// GetFiles is used to get the printer's files API endpoint
func GetFiles(printer config.Printers) (Files, error) {
	var files Files
	response, err := accessPrinterEndpoint("/api/files?recursive=true", printer)

	if err != nil {
		return files, err
	}

	err = json.Unmarshal(response, &files)

	return files, err
}

// GetJobV1 is used to get the printer's job v1 API endpoint
func GetJobV1(printer config.Printers) (JobV1, error) {
	var job JobV1
	response, err := accessPrinterEndpoint("/api/v1/job", printer)

	if err != nil {
		return job, err
	}

	err = json.Unmarshal(response, &job)

	return job, err
}

// GetStatus is used to get Buddy status endpoint
func GetStatus(printer config.Printers) (Status, error) {
	var status Status
	response, err := accessPrinterEndpoint("/api/v1/status", printer)

	if err != nil {
		return status, err
	}

	err = json.Unmarshal(response, &status)

	return status, err
}

// GetStorageV1 is used to get the printer's storage v1 API endpoint
func GetStorageV1(printer config.Printers) (StorageV1, error) {
	var storage StorageV1
	response, err := accessPrinterEndpoint("/api/v1/storage", printer)

	if err != nil {
		return storage, err
	}

	err = json.Unmarshal(response, &storage)

	return storage, err
}

// GetInfo is used to get the printer's info API endpoint
func GetInfo(printer config.Printers) (Info, error) {
	var info Info
	response, err := accessPrinterEndpoint("/api/v1/info", printer)

	if err != nil {
		return info, err
	}

	err = json.Unmarshal(response, &info)

	return info, err
}

// GetSettings is used to get the printer's settings API endpoint
func GetSettings(printer config.Printers) (Settings, error) {
	var settings Settings
	response, err := accessPrinterEndpoint("/api/settings", printer)

	if err != nil {
		return settings, err
	}

	err = json.Unmarshal(response, &settings)

	return settings, err
}

// GetCameras is used to get the printer's cameras API endpoint
func GetCameras(printer config.Printers) (Cameras, error) {
	var cameras Cameras
	response, err := accessPrinterEndpoint("/api/v1/cameras", printer)

	if err != nil {
		return cameras, err
	}

	err = json.Unmarshal(response, &cameras)

	return cameras, err
}

// GetPrinterProfiles is used to get the printer's printerprofiles API endpoint
func GetPrinterProfiles(printer config.Printers) (PrinterProfiles, error) {
	var profiles PrinterProfiles
	response, err := accessPrinterEndpoint("/api/v1/printerprofiles", printer)

	if err != nil {
		return profiles, err
	}

	err = json.Unmarshal(response, &profiles)

	return profiles, err
}

// GetJobImage is used to get the printer's job image from API
func GetJobImage(printer config.Printers, imagePath string) (string, error) { // returns base64 encoded image
	//http://192.168.20.50/thumb/l/usb/PYTHON~1.BGC
	response, err := accessPrinterEndpoint("/thumb/l"+imagePath, printer)
	if err != nil {
		return "", err
	}

	image, err := compressPNG(response, png.BestCompression)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(image), nil
}

func compressPNG(input []byte, compressionLevel png.CompressionLevel) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(input))

	if err != nil {
		return nil, err
	}

	var compressedBuffer bytes.Buffer
	encoder := png.Encoder{
		CompressionLevel: compressionLevel,
	}

	err = encoder.Encode(&compressedBuffer, img)

	if err != nil {
		return nil, err
	}

	return compressedBuffer.Bytes(), nil
}

// GetPrinterType returns the printer type of the given printer - e.g. "MINI", "MK4", "XL", "I3MK3S", "I3MK3", "I3MK25S",
func GetPrinterType(printer config.Printers) (string, error) {
	version, err := GetVersion(printer)
	if err != nil {
		return "unknown", err
	}

	printerType := version.Hostname

	if version.Hostname == "" {
		if version.Original == "" {
			info, err := GetInfo(printer)
			if err != nil {
				return "unknown", err
			}
			printerType = info.Hostname
		} else {
			printerType = version.Original
		}
	} else if version.Original != "" {
		printerType = version.Original
	}

	if printerTypes[printerType] != "" {
		printerType = printerTypes[printerType]
	}

	if printerType == "" {
		printerType = "unknown"
	}

	log.Trace().Msg(printerType + " detected for " + printer.Address + " (" + printer.Name + ")")

	return printerType, nil
}

// ProbePrinter is used to probe the printer - just testing the connection
func ProbePrinter(printer config.Printers) (bool, error) {
	req, _ := http.NewRequest("GET", "http://"+printer.Address+"/", nil)
	client := &http.Client{Timeout: time.Duration(configuration.Exporter.ScrapeTimeout) * time.Millisecond}
	r, e := client.Do(req)

	if e != nil {
		return false, e
	}

	if r.StatusCode == 401 {
		log.Debug().Msg("401 Unauthorized, trying to access with API key - " + printer.Address)
		req, _ := http.NewRequest("GET", "http://"+printer.Address+"/api/v1/status", nil)
		req.Header.Add("X-Api-Key", printer.Apikey)
		r, e = client.Do(req)
		if e != nil {
			return false, e
		}
	}

	return r.StatusCode == 200, nil
}
