package rollbar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	NAME         = "go-rollbar"
	VERSION      = "0.0.1"
	CHANNEL_SIZE = 100
)

var (
	Token       = ""
	Environment = "development"
	Endpoint    = "https://api.rollbar.com/api/1/item/"

	bodyChannel chan map[string]interface{}
	once        sync.Once
)

func Error(level string, err error) {
	once.Do(initChannel)

	body := buildBody(level, err.Error())
	data := body["data"].(map[string]interface{})
	data["body"] = errorBody(err)

	push(body)
}

func Message(level string, msg string) {
	once.Do(initChannel)

	body := buildBody(level, msg)
	data := body["data"].(map[string]interface{})
	data["body"] = messageBody(msg)

	push(body)
}

func initChannel() {
	bodyChannel = make(chan map[string]interface{}, CHANNEL_SIZE)

	go func() {
		for body := range bodyChannel {
			post(body)
		}
	}()
}

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

func errorBody(err error) map[string]interface{} {
	errorClass := reflect.TypeOf(err).String()
	if errorClass == "" {
		errorClass = "panic"
	} else {
		errorClass = strings.TrimPrefix(errorClass, "*")
	}

	return map[string]interface{}{
		"trace": map[string]interface{}{
			"frames": stacktraceFrames(3),
			"exception": map[string]interface{}{
				"class":   errorClass,
				"message": err.Error(),
			},
		},
	}
}

func messageBody(s string) map[string]interface{} {
	return map[string]interface{}{
		"message": map[string]interface{}{
			"body": s,
		},
	}
}

func push(body map[string]interface{}) {
	if len(bodyChannel) < CHANNEL_SIZE {
		bodyChannel <- body
	}
}

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
