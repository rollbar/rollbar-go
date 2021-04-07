package rollbar

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// Telemetry struct contains writer (for logs) and round tripper (for http client) and enables to queue the events
type Telemetry struct {
	Writer  io.Writer
	Proxied http.RoundTripper
	Queue   *Queue
}

// Write is the writer for telemetry logs
func (t *Telemetry) Write(p []byte) (int, error) {
	telemetryData := t.populateLoggerBody(p)
	t.Queue.Push(telemetryData)
	return t.Writer.Write(p)
}

// RoundTrip implements RoundTrip in http.RoundTripper
func (t *Telemetry) RoundTrip(req *http.Request) (res *http.Response, e error) {

	// Send the request, get the response (or the error)
	res, e = t.Proxied.RoundTrip(req)
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
	var dataHeaders = map[string][]string{}
	dataBody["status_code"] = nil
	data["level"] = "info"
	if res != nil {
		dataBody["status_code"] = res.StatusCode
		if res.StatusCode >= http.StatusInternalServerError {
			data["level"] = "critical"
		} else if res.StatusCode >= http.StatusBadRequest {
			data["level"] = "error"
		}
	}
	dataBody["url"] = req.URL.Scheme + "://" + req.Host + req.URL.Path
	dataBody["method"] = req.Method
	dataBody["subtype"] = "http"

	for k, v := range req.Header {
		dataHeaders[k] = v
	}
	dataBody["request_headers"] = dataHeaders

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

// WithCustomTransporter sets the custom transporter
func WithCustomTransporter(t http.RoundTripper) OptionFunc {
	return func(f *Telemetry) {
		f.Proxied = t
	}
}

// WithCustomQueueSize initializes the queue with a custom size
func WithCustomQueueSize(size int) OptionFunc {
	return func(f *Telemetry) {
		f.Queue = NewQueue(size)
	}
}

// NewTelemetry initializes telemetry object
func NewTelemetry(options ...OptionFunc) *Telemetry {
	res := &Telemetry{
		Proxied: http.DefaultTransport,
		Queue:   NewQueue(50),
		Writer:  os.Stdout,
	}
	for _, opt := range options {
		opt(res)
	}

	log.SetOutput(res)
	http.DefaultClient = &http.Client{Transport: res}
	return res
}
