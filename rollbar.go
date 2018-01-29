package rollbar

import (
	"log"
	"net/http"
	"os"
	"regexp"
)

const (
	NAME    = "rollbar/rollbar-go"
	VERSION = "0.6.0"

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
	std         = NewAsync("", "development", "", hostname, "")
	nilErrTitle = "<nil>"
)

// Rollbar access token.
func SetToken(token string) {
	std.SetToken(token)
}

// All errors and messages will be submitted under this environment.
func SetEnvironment(environment string) {
	std.SetEnvironment(environment)
}

// String describing the running code version on the server
func SetCodeVersion(codeVersion string) {
	std.SetCodeVersion(codeVersion)
}

// host: The server hostname. Will be indexed.
func SetServerHost(serverHost string) {
	std.SetServerHost(serverHost)
}

// root: Path to the application code root, not including the final slash.
// Used to collapse non-project code when displaying tracebacks.
func SetServerRoot(serverRoot string) {
	std.SetServerRoot(serverRoot)
}

// custom: Any arbitrary metadata you want to send.
func SetCustom(custom map[string]interface{}) {
	std.SetCustom(custom)
}

// Regular expression used to match headers for scrubbing
// The default value is regexp.MustCompile("Authorization")
func SetScrubHeaders(headers *regexp.Regexp) {
	std.SetScrubHeaders(headers)
}

// Regular expression to match keys in the item payload for scrubbing
// The default vlaue is regexp.MustCompile("password|secret|token"),
func SetScrubFields(fields *regexp.Regexp) {
	std.SetScrubFields(fields)
}

// CheckIgnore is called during the recovery process of a panic that
// occurred inside a function wrapped by Wrap or WrapAndWait
// Return true if you wish to ignore this panic, false if you wish to
// report it to Rollbar. If an error is the argument to the panic, then
// this function is called with the result of calling Error(), otherwise
// the string representation of the value is passed to this function.
func SetCheckIgnore(checkIgnore func(string) bool) {
	std.SetCheckIgnore(checkIgnore)
}

// -- Getters

// Rollbar access token.
func Token() string {
	return std.Token()
}

// All errors and messages will be submitted under this environment.
func Environment() string {
	return std.Environment()
}

// String describing the running code version on the server
func CodeVersion() string {
	return std.CodeVersion()
}

// host: The server hostname. Will be indexed.
func ServerHost() string {
	return std.ServerHost()
}

// root: Path to the application code root, not including the final slash.
// Used to collapse non-project code when displaying tracebacks.
func ServerRoot() string {
	return std.ServerRoot()
}

// custom: Any arbitrary metadata you want to send.
func Custom() map[string]interface{} {
	return std.Custom()
}

// -- Error reporting

// Error asynchronously sends an error to Rollbar with the given severity level.
func Error(level string, err error) {
	std.Error(level, err)
}

// ErrorWithExtras asynchronously sends an error to Rollbar with the given
// severity level with extra custom data.
func ErrorWithExtras(level string, err error, extras map[string]interface{}) {
	std.ErrorWithExtras(level, err, extras)
}

// RequestError asynchronously sends an error to Rollbar with the given
// severity level and request-specific information.
func RequestError(level string, r *http.Request, err error) {
	std.RequestError(level, r, err)
}

// RequestErrorWithExtras asynchronously sends an error to Rollbar with the given
// severity level and request-specific information with extra custom data.
func RequestErrorWithExtras(level string, r *http.Request, err error, extras map[string]interface{}) {
	std.RequestErrorWithExtras(level, r, err, extras)
}

// ErrorWithStackSkip asynchronously sends an error to Rollbar with the given
// severity level and a given number of stack trace frames skipped.
func ErrorWithStackSkip(level string, err error, skip int) {
	std.ErrorWithStackSkip(level, err, skip)
}

// ErrorWithStackSkipWithExtras asynchronously sends an error to Rollbar with the given
// severity level and a given number of stack trace frames skipped with extra custom data.
func ErrorWithStackSkipWithExtras(level string, err error, skip int, extras map[string]interface{}) {
	std.ErrorWithStackSkipWithExtras(level, err, skip, extras)
}

// RequestErrorWithStackSkip asynchronously sends an error to Rollbar with the
// given severity level and a given number of stack trace frames skipped, in
// addition to extra request-specific information.
func RequestErrorWithStackSkip(level string, r *http.Request, err error, skip int) {
	std.RequestErrorWithStackSkip(level, r, err, skip)
}

// RequestErrorWithStackSkipWithExtras asynchronously sends an error to Rollbar
// with the given severity level and a given number of stack trace frames skipped,
// in addition to extra request-specific information and extra custom data.
func RequestErrorWithStackSkipWithExtras(level string, r *http.Request, err error, skip int, extras map[string]interface{}) {
	std.RequestErrorWithStackSkipWithExtras(level, r, err, skip, extras)
}

// -- Message reporting

// Message asynchronously sends a message to Rollbar with the given severity
// level. Rollbar request is asynchronous.
func Message(level string, msg string) {
	std.Message(level, msg)
}

// MessageWithExtras asynchronously sends a message to Rollbar with the given severity
// level with extra custom data. Rollbar request is asynchronous.
func MessageWithExtras(level string, msg string, extras map[string]interface{}) {
	std.MessageWithExtras(level, msg, extras)
}

// RequestMessage asynchronously sends a message to Rollbar with the given
// severity level and request-specific information.
func RequestMessage(level string, r *http.Request, msg string) {
	std.RequestMessage(level, r, msg)
}

// RequestMessageWithExtras asynchronously sends a message to Rollbar with the given severity
// level with extra custom data in addition to extra request-specific information.
// Rollbar request is asynchronous.
func RequestMessageWithExtras(level string, r *http.Request, msg string, extras map[string]interface{}) {
	std.RequestMessageWithExtras(level, r, msg, extras)
}

// Wait will block until the queue of errors / messages is empty.
func Wait() {
	std.Wait()
}

// Wrap calls f and then recovers and reports a panic to Rollbar if it occurs.
// If an error is captured it is subsequently returned.
func Wrap(f func()) interface{} {
	return std.Wrap(f)
}

// WrapAndWait calls f, and recovers and reports a panic to Rollbar if it occurs.
// This also waits before returning to ensure the message was reported
// If an error is captured it is subsequently returned.
func WrapAndWait(f func()) interface{} {
	return std.WrapAndWait(f)
}

// Errors can implement this interface to create a trace_chain
// Callers are required to call BuildStack on their own at the
// time the cause is wrapped.
type CauseStacker interface {
	error
	Cause() error
	Stack() Stack
}

// -- rollbarError

func rollbarError(format string, args ...interface{}) {
	format = "Rollbar error: " + format + "\n"
	log.Printf(format, args...)
}
