package rollbar

import (
	"errors"
	"os"
	"testing"
	"time"
)

type CustomError struct {
	s string
}

func (e *CustomError) Error() string {
	return e.s
}

func testErrorStack(s string) {
	testErrorStack2(s)
}

func testErrorStack2(s string) {
	Error("error", errors.New(s))
}

func testErrorStackWithSkip(s string) {
	testErrorStackWithSkip2(s)
}

func testErrorStackWithSkip2(s string) {
	ErrorWithStackSkip("error", errors.New(s), 2)
}

func TestEverything(t *testing.T) {
	Token = os.Getenv("TOKEN")
	Environment = "test"

	Error("critical", errors.New("Normal critical error"))
	Error("error", &CustomError{"This is a custom error"})

	testErrorStack("This error should have a nice stacktrace")
	testErrorStackWithSkip("This error should have a skipped stacktrace")

	done := make(chan bool)
	go func() {
		testErrorStack("I'm in a goroutine")
		done <- true
	}()
	<-done

	Message("error", "This is an error message")
	Message("info", "And this is an info message")

	// Wait for all messages to be sent
	for {
		if len(bodyChannel) == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
}
