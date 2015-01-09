package rollbar

import (
	"fmt"
	"hash/adler32"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
)

const (
	NAME    = "heroku/rollbar"
	VERSION = "0.3.0"

	// Severity levels
	CRIT  = "critical"
	ERR   = "error"
	WARN  = "warning"
	INFO  = "info"
	DEBUG = "debug"

	FILTERED = "[FILTERED]"
)

var (
	hostname, _ = os.Hostname()
	Std         = New("", "development", "", hostname, "")
)

func SetToken(token string) {
	Std.SetToken(token)
}

func SetEnvironment(environment string) {
	Std.SetEnvironment(environment)
}

func SetCodeVersion(codeVersion string) {
	Std.SetCodeVersion(codeVersion)
}

func SetServerHost(serverHost string) {
	Std.SetServerHost(serverHost)
}

func SetServerRoot(serverRoot string) {
	Std.SetServerRoot(serverRoot)
}

// -- Error reporting

// Error asynchronously sends an error to Rollbar with the given severity level.
func Error(level string, err error) {
	Std.Error(level, err)
}

// Error asynchronously sends an error to Rollbar with the given severity level with extra custom data.
func ErrorWithExtras(level string, err error, extras map[string]interface{}) {
	Std.ErrorWithExtras(level, err, extras)
}

// RequestError asynchronously sends an error to Rollbar with the given
// severity level and request-specific information.
func RequestError(level string, r *http.Request, err error) {
	Std.RequestError(level, r, err)
}

// RequestError asynchronously sends an error to Rollbar with the given
// severity level and request-specific information with extra custom data.
func RequestErrorWithExtras(level string, r *http.Request, err error, extras map[string]interface{}) {
	Std.RequestErrorWithExtras(level, r, err, extras)
}

// ErrorWithStackSkip asynchronously sends an error to Rollbar with the given
// severity level and a given number of stack trace frames skipped.
func ErrorWithStackSkip(level string, err error, skip int) {
	Std.ErrorWithStackSkip(level, err, skip)
}

// ErrorWithStackSkip asynchronously sends an error to Rollbar with the given
// severity level and a given number of stack trace frames skipped with extra custom data.
func ErrorWithStackSkipWithExtras(level string, err error, skip int, extras map[string]interface{}) {
	Std.ErrorWithStackSkipWithExtras(level, err, skip, extras)
}

// RequestErrorWithStackSkip asynchronously sends an error to Rollbar with the
// given severity level and a given number of stack trace frames skipped, in
// addition to extra request-specific information.
func RequestErrorWithStackSkip(level string, r *http.Request, err error, skip int) {
	Std.RequestErrorWithStackSkip(level, r, err, skip)
}

// RequestErrorWithStackSkip asynchronously sends an error to Rollbar with the
// given severity level and a given number of stack trace frames skipped, in
// addition to extra request-specific information and extra custom data.
func RequestErrorWithStackSkipWithExtras(level string, r *http.Request, err error, skip int, extras map[string]interface{}) {
	Std.RequestErrorWithStackSkipWithExtras(level, r, err, skip, extras)
}

// -- Message reporting

// Message asynchronously sends a message to Rollbar with the given severity
// level. Rollbar request is asynchronous.
func Message(level string, msg string) {
	Std.Message(level, msg)
}

// Message asynchronously sends a message to Rollbar with the given severity
// level with extra custom data. Rollbar request is asynchronous.
func MessageWithExtras(level string, msg string, extras map[string]interface{}) {
	Std.MessageWithExtras(level, msg, extras)
}

// -- Misc.

// Wait will block until the queue of errors / messages is empty.
func Wait() {
	Std.Wait()
}

// Errors can implement this interface to create a trace_chain
// Callers are required to call BuildStack on their own at the
// time the cause is wrapped.
type CauseStacker interface {
	error
	Cause() error
	Stack() Stack
}

// Build an error inner-body for the given error. If skip is provided, that
// number of stack trace frames will be skipped. If the error has a Cause
// method, the causes will be traversed until nil.
func errorBody(err error, skip int) (map[string]interface{}, string) {
	var parent error
	traceChain := []map[string]interface{}{}
	fingerprint := ""
	for err != nil {
		stack := getOrBuildStack(err, parent, skip)
		traceChain = append(traceChain, buildTrace(err, stack))
		fingerprint = fingerprint + stack.Fingerprint()
		parent = err
		err = getCause(err)
	}
	errBody := map[string]interface{}{"trace_chain": traceChain}
	return errBody, fingerprint
}

// builds one trace element in trace_chain
func buildTrace(err error, stack Stack) map[string]interface{} {
	return map[string]interface{}{
		"frames": stack,
		"exception": map[string]interface{}{
			"class":   errorClass(err),
			"message": err.Error(),
		},
	}
}

func getCause(err error) error {
	if cs, ok := err.(CauseStacker); ok {
		return cs.Cause()
	} else {
		return nil
	}
}

// gets Stack from errors that provide one of their own
// otherwise, builds a new stack
func getOrBuildStack(err error, parent error, skip int) Stack {
	if cs, ok := err.(CauseStacker); ok {
		if s := cs.Stack(); s != nil {
			return s
		}
	} else {
		if _, ok := parent.(CauseStacker); !ok {
			return BuildStack(4 + skip)
		}
	}

	return make(Stack, 0)
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

// -- rollbarError

func rollbarError(format string, args ...interface{}) {
	format = "Rollbar error: " + format + "\n"
	log.Printf(format, args...)
}
