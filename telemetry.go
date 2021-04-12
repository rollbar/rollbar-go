package rollbar

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"
)

const TelemetryQueueSize = 50

// Telemetry struct contains writer (for logs) and round tripper (for http client) and enables to queue the events
type Telemetry struct {
	Logger struct {
		Writer io.Writer
	}

	Network struct {
		Proxied      http.RoundTripper
		ScrubHeaders *regexp.Regexp

		enbaleDefaultClient bool
		disableReqHeaders   bool
		disableResHeaders   bool
	}
	Queue *Queue
}

// Write is the writer for telemetry logs
func (t *Telemetry) Write(p []byte) (int, error) {
	telemetryData := t.populateLoggerBody(p)
	t.Queue.Push(telemetryData)
	return t.Logger.Writer.Write(p)
}

// RoundTrip implements RoundTrip in http.RoundTripper
func (t *Telemetry) RoundTrip(req *http.Request) (res *http.Response, e error) {

	// Send the request, get the response (or the error)
	res, e = t.Network.Proxied.RoundTrip(req)
	if e != nil {
		fmt.Printf("Error: %v", e)
	}
	telemetryData := t.populateTransporterBody(req, res)
	t.Queue.Push(telemetryData)
	return
}

func (t *Telemetry) populateLoggerBody(p []byte) map[string]interface{} {
	var data = map[string]interface{}{}
	message := map[string]interface{}{"message": string(p)}
	data["body"] = message
	data["source"] = "client"
	data["timestamp_ms"] = time.Now().UnixNano() / int64(time.Millisecond)
	data["type"] = "log"
	data["level"] = "log"
	return data
}

func (t *Telemetry) populateTransporterBody(req *http.Request, res *http.Response) map[string]interface{} {
	var data = map[string]interface{}{}
	var dataBody = map[string]interface{}{}
	dataBody["status_code"] = nil
	data["level"] = "info"
	if res != nil {
		dataBody["status_code"] = res.StatusCode
		if res.StatusCode >= http.StatusInternalServerError {
			data["level"] = "critical"
		} else if res.StatusCode >= http.StatusBadRequest {
			data["level"] = "error"
		}

		if !t.Network.disableResHeaders {
			var dataHeaders = map[string][]string{}
			for k, v := range res.Header {
				dataHeaders[k] = v
			}
			filteredDataHeaders := filterFlatten(t.Network.ScrubHeaders, dataHeaders, nil)
			response := map[string]interface{}{"headers": filteredDataHeaders}
			dataBody["response"] = response
		}

	}
	dataBody["url"] = req.URL.Scheme + "://" + req.Host + req.URL.Path
	dataBody["method"] = req.Method
	dataBody["subtype"] = "http"

	if !t.Network.disableReqHeaders {
		var dataHeaders = map[string][]string{}
		for k, v := range req.Header {
			dataHeaders[k] = v
		}
		filteredDataHeaders := filterFlatten(t.Network.ScrubHeaders, dataHeaders, nil)
		dataBody["request_headers"] = filteredDataHeaders
	}
	data["body"] = dataBody
	data["source"] = "client"
	data["timestamp_ms"] = time.Now().UnixNano() / int64(time.Millisecond)
	data["type"] = "network"
	return data
}

// GetQueueItems gets all the items from the queue
func (t *Telemetry) GetQueueItems() []interface{} {
	return t.Queue.Items()
}

// OptionFunc is the pointer to the optional parameter function
type OptionFunc func(*Telemetry)

// EnableNetworkTelemetry enables the network telemetry.
// if no custom http Transport is needed, then nil can be passed
func EnableNetworkTelemetry(t http.RoundTripper) OptionFunc {
	return func(f *Telemetry) {
		f.Network.Proxied = http.DefaultTransport
		if t != nil {
			f.Network.Proxied = t
		}
	}
}

// EnableNetworkTelemetryForDefaultClient sets the http.DefaultClient for telemetry
func EnableNetworkTelemetryForDefaultClient() OptionFunc {
	return func(f *Telemetry) {
		f.Network.enbaleDefaultClient = true
	}
}

// DisableNetworkTelemetryRequestHeaders disables telemetry request headers
func DisableNetworkTelemetryRequestHeaders() OptionFunc {
	return func(f *Telemetry) {
		f.Network.disableReqHeaders = true
	}
}

// DisableNetworkTelemetryResponseHeaders disables telemetry response headers
func DisableNetworkTelemetryResponseHeaders() OptionFunc {
	return func(f *Telemetry) {
		f.Network.disableResHeaders = true
	}
}

// SetCustomQueueSize initializes the queue with a custom size
func SetCustomQueueSize(size int) OptionFunc {
	return func(f *Telemetry) {
		f.Queue = NewQueue(size)
	}
}

// EnableLoggerTelemetry enables logger telemetry
func EnableLoggerTelemetry() OptionFunc {
	return func(f *Telemetry) {
		f.Logger.Writer = os.Stdout
		log.SetOutput(f)
	}
}

// NewTelemetry initializes telemetry object
func NewTelemetry(options ...OptionFunc) *Telemetry {
	res := &Telemetry{
		Queue: NewQueue(TelemetryQueueSize),
	}

	for _, opt := range options {
		opt(res)
	}

	if res.Network.ScrubHeaders == nil { // set/define only once
		res.Network.ScrubHeaders = regexp.MustCompile("Authorization")
	}
	if res.Network.enbaleDefaultClient {
		http.DefaultClient.Transport = res
	}
	return res
}
