package rollbar

import (
	"errors"
)

func ExampleCritical_error() {
	Critical(errors.New("bork"), map[string]interface{}{
		"hello": "world",
	})
}

func ExampleCritical_message() {
	Critical("bork", map[string]interface{}{
		"hello": "world",
	})
}

func ExampleError_error() {
	Error(errors.New("bork"), map[string]interface{}{
		"hello": "world",
	})
}

func ExampleError_message() {
	Error("bork", map[string]interface{}{
		"hello": "world",
	})
}

func ExampleWarning_error() {
	Warning(errors.New("bork"), map[string]interface{}{
		"hello": "world",
	})
}

func ExampleWarning_message() {
	Warning("bork", map[string]interface{}{
		"hello": "world",
	})
}

func ExampleInfo_error() {
	Info(errors.New("bork"), map[string]interface{}{
		"hello": "world",
	})
}

func ExampleInfo_message() {
	Info("bork", map[string]interface{}{
		"hello": "world",
	})
}

func ExampleDebug_error() {
	Debug(errors.New("bork"), map[string]interface{}{
		"hello": "world",
	})
}

func ExampleDebug_message() {
	Debug("bork", map[string]interface{}{
		"hello": "world",
	})
}
