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
	VERSION = "0.0.4"
)

var (
	// Rollbar access token. If this is blank, no errors will be reported to
	// Rollbar.
	Token = ""

	// All errors and messages will be submitted under this environment.
	Environment = "development"

	// API endpoint for Rollbar.
	Endpoint = "https://api.rollbar.com/api/1/item/"

	// Maximum number of errors allowed in the sending queue before we start
	// dropping new errors on the floor.
	Buffer = 100

	// Queue of messages to be sent.
	bodyChannel chan map[string]interface{}
	once        sync.Once
	waitGroup   sync.WaitGroup
)

// -- Error reporting

// Error asynchronously sends an error to Rollbar with the given severity level.
func Error(level string, err error) {
	ErrorWithStackSkip(level, err, 1)
}

// ErrorWithStackSkip asynchronously sends an error to Rollbar with the given
// severity level and a given number of stack trace frames skipped.
func ErrorWithStackSkip(level string, err error, skip int) {
	once.Do(initChannel)

	body := buildBody(level, err.Error())
	data := body["data"].(map[string]interface{})
	data["body"] = errorBody(err, skip)

	push(body)
}

// -- Message reporting

// Message asynchronously sends a message to Rollbar with the given severity
// level. Rollbar request is asynchronous.
func Message(level string, msg string) {
	once.Do(initChannel)

	body := buildBody(level, msg)
	data := body["data"].(map[string]interface{})
	data["body"] = messageBody(msg)

	push(body)
}

// -- Misc.

// Wait will block until the queue of errors / messages is empty.
func Wait() {
	waitGroup.Wait()
}

// Build the main JSON structure that will be sent to Rollbar with the
// appropriate metadata.
func buildBody(level, title string) map[string]interface{} {
	timestamp := time.Now().Unix()
	hostname, _ := os.Hostname()

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
		return fmt.Sprintf("{%x}", checksum)
	} else {
		return strings.TrimPrefix(class, "*")
	}
}

// -- POST handling

// Start a goroutine that sends all errors and messages to Rollbar.
func initChannel() {
	bodyChannel = make(chan map[string]interface{}, Buffer)

	go func() {
		for body := range bodyChannel {
			post(body)
			waitGroup.Done()
		}
	}()
}

// Queue the given JSON body to be POSTed to Rollbar.
func push(body map[string]interface{}) {
	if len(bodyChannel) < Buffer {
		waitGroup.Add(1)
		bodyChannel <- body
	}
}

// POST the given JSON body to Rollbar synchronously.
func post(body map[string]interface{}) {
	if len(Token) == 0 {
		stderr("Token is empty")
		return
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		stderr(fmt.Sprintf("Rollbar payload couldn't be encoded: %s", err.Error()))
		return
	}

	resp, err := http.Post(Endpoint, "application/json", bytes.NewReader(jsonBody))
	defer resp.Body.Close()
	if err != nil {
		stderr(fmt.Sprintf("Rollbar POST failed: %s", err.Error()))
	} else if resp.StatusCode != 200 {
		stderr(fmt.Sprintf("Rollbar response: %s", resp.Status))
	}
}
