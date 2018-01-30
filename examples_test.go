package rollbar_test

import (
	"errors"
	"github.com/rollbar/rollbar-go"
)

func ExampleCritical_error() {
	rollbar.Critical(errors.New("bork"), map[string]interface{}{
		"hello": "world",
	})
}

func ExampleCritical_message() {
	rollbar.Critical("bork", map[string]interface{}{
		"hello": "world",
	})
}

func ExampleError_error() {
	rollbar.Error(errors.New("bork"), map[string]interface{}{
		"hello": "world",
	})
}

func ExampleError_message() {
	rollbar.Error("bork", map[string]interface{}{
		"hello": "world",
	})
}

func ExampleWarning_error() {
	rollbar.Warning(errors.New("bork"), map[string]interface{}{
		"hello": "world",
	})
}

func ExampleWarning_message() {
	rollbar.Warning("bork", map[string]interface{}{
		"hello": "world",
	})
}

func ExampleInfo_error() {
	rollbar.Info(errors.New("bork"), map[string]interface{}{
		"hello": "world",
	})
}

func ExampleInfo_message() {
	rollbar.Info("bork", map[string]interface{}{
		"hello": "world",
	})
}

func ExampleDebug_error() {
	rollbar.Debug(errors.New("bork"), map[string]interface{}{
		"hello": "world",
	})
}

func ExampleDebug_message() {
	rollbar.Debug("bork", map[string]interface{}{
		"hello": "world",
	})
}
