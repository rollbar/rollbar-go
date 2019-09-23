package rollbar_test

import (
	"runtime"

	"github.com/rollbar/rollbar-go"
)

type CustomTraceError struct {
	error
	trace []runtime.Frame
}

func (e CustomTraceError) GetTrace() []runtime.Frame {
	return e.trace
}

func ExampleSetStackTracer() {
	rollbar.SetStackTracer(func(err error) ([]runtime.Frame, bool) {
		// preserve the default behavior for other types of errors
		if trace, ok := rollbar.DefaultStackTracer(err); ok {
			return trace, ok
		}

		if cerr, ok := err.(CustomTraceError); ok {
			return cerr.GetTrace(), true
		}

		return nil, false
	})
}

