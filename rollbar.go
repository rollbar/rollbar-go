package rollbar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/adler32"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	NAME    = "go-rollbar"
	VERSION = "0.0.1"
)

var (
	// Rollbar access token. This must be set in order to report errors
	// successfully.
	Token = ""

	// All errors and messages will be submitted under this environment.
	Environment = "development"

	// API endpoint for Rollbar.
	Endpoint = "https://api.rollbar.com/api/1/item/"

	// Number of requests to queue up for sending before discarding new requests.
	Buffer = 100

	bodyChannel chan map[string]interface{}
	once        sync.Once
)

// -- Error reporting

// Error sends an error to Rollbar with the given severity level. The Rollbar
// request is asynchronous.
func Error(level string, err error) {
	ErrorWithStackSkip(level, err, 1)
}

// Error sends an error to Rollbar with the given severity level and a given
// number of stack trace frames skipped. The Rollbar request is asynchronous.
func ErrorWithStackSkip(level string, err error, skip int) {
	once.Do(initChannel)

	body := buildBody(level, err.Error())
	data := body["data"].(map[string]interface{})
	data["body"] = errorBody(err, skip)

	push(body)
}

// -- Message reporting

// Message sends a message to Rollbar with the given severity level. The
// Rollbar request is asynchronous.
func Message(level string, msg string) {
	once.Do(initChannel)

	body := buildBody(level, msg)
	data := body["data"].(map[string]interface{})
	data["body"] = messageBody(msg)

	push(body)
}

// -- Misc.

// Build the main JSON structure that will be sent to Rollbar with the
// appropriate metadata.
func buildBody(level, title string) map[string]interface{} {
	timestamp := time.Now().Unix()
	hostname, _ := os.Hostname()
	cwd, _ := os.Getwd()

	return map[string]interface{}{
		"access_token": Token,
		"data": map[string]interface{}{
			"environment": Environment,
			"title":       title,
			"level":       level,
			"timestamp":   timestamp,
			"platform":    runtime.GOOS,
			"language":    "go",
			"server": map[string]interface{}{
				"host": hostname,
				"root": cwd,
			},
			"notifier": map[string]interface{}{
				"name":    NAME,
				"version": VERSION,
			},
		},
	}
}

// Build an error inner-body for the given error. If skip is provided, that
// number of stack trace frames will be skipped.
func errorBody(err error, skip int) map[string]interface{} {
	return map[string]interface{}{
		"trace": map[string]interface{}{
			"frames": stacktraceFrames(3 + skip),
			"exception": map[string]interface{}{
				"class":   errorClass(err),
				"message": err.Error(),
			},
		},
	}
}

// Build a message inner-body for the given message string.
func messageBody(s string) map[string]interface{} {
	return map[string]interface{}{
		"message": map[string]interface{}{
			"body": s,
		},
	}
}

func errorClass(err error) string {
	class := reflect.TypeOf(err).String()
	if class == "" {
		return "panic"
	} else if class == "*errors.errorString" {
		checksum := adler32.Checksum([]byte(err.Error()))
		return fmt.Sprintf("errors.errorString{%x}", checksum)
	} else {
		return strings.TrimPrefix(class, "*")
	}
}

// -- POST handling

// Starts a goroutine that handles the sending of all JSON bodies sent on the
// bodyChannel.
func initChannel() {
	bodyChannel = make(chan map[string]interface{}, Buffer)

	go func() {
		for body := range bodyChannel {
			post(body)
		}
	}()
}

// Queues the given JSON body for POSTing to Rollbar.
func push(body map[string]interface{}) {
	if len(bodyChannel) < Buffer {
		bodyChannel <- body
	}
}

// POSTS the given JSON body to Rollbar synchronously.
func post(body map[string]interface{}) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		stderr(fmt.Sprintf("Payload couldn't be encoded: %s", err.Error()))
		return
	}
	bodyReader := bytes.NewReader(jsonBody)
	resp, err := http.Post(Endpoint, "application/json", bodyReader)
	if err != nil {
		stderr(fmt.Sprintf("POST failed: %s", err.Error()))
	} else if resp.StatusCode != 200 {
		stderr(fmt.Sprintf("Rollbar response: %s", resp.Status))
	}
}
