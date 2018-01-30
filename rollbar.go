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

// A Rollbar access token with scope "post_server_item"
// It is required to set this value before any of the other functions herein will be able to work
// properly.
func SetToken(token string) {
	std.SetToken(token)
}

// All errors and messages will be submitted under this environment.
func SetEnvironment(environment string) {
	std.SetEnvironment(environment)
}

// The endpoint to post items to.
// The default value is https://api.rollbar.com/api/1/item/
func SetEndpoint(endpoint string) {
	std.SetEndpoint(endpoint)
}

// Platform is the platform reported for all Rollbar items. The default is
// the running operating system (darwin, freebsd, linux, etc.) but it can
// also be application specific (client, heroku, etc.).
func SetPlatform(platform string) {
	std.SetPlatform(platform)
}

// String describing the running code version on the server
func SetCodeVersion(codeVersion string) {
	std.SetCodeVersion(codeVersion)
}

// The server hostname. Will be indexed.
func SetServerHost(serverHost string) {
	std.SetServerHost(serverHost)
}

// Path to the application code root, not including the final slash.
// Used to collapse non-project code when displaying tracebacks.
func SetServerRoot(serverRoot string) {
	std.SetServerRoot(serverRoot)
}

// Any arbitrary metadata you want to send with every subsequently sent item.
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

// SetPerson information for identifying a user associated with
// any subsequent errors or messages. Only id is required to be
// non-empty.
func SetPerson(id, username, email string) {
	std.SetPerson(id, username, email)
}

// ClearPerson clears any previously set person information. See `SetPerson` for more information.
func ClearPerson() {
	std.ClearPerson()
}

// Whether or not to use custom client-side fingerprint
// based on a CRC32 checksum. The alternative is to let the server compute a fingerprint for each
// item. The default is false.
func SetFingerprint(fingerprint bool) {
	std.SetFingerprint(fingerprint)
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

// Get the currently configured endpoint.
func Endpoint() string {
	return std.Endpoint()
}

// Platform is the platform reported for all Rollbar items. The default is
// the running operating system (darwin, freebsd, linux, etc.) but it can
// also be application specific (client, heroku, etc.).
func Platform() string {
	return std.Platform()
}

// String describing the running code version on the server.
func CodeVersion() string {
	return std.CodeVersion()
}

// The server hostname. Will be indexed.
func ServerHost() string {
	return std.ServerHost()
}

// Path to the application code root, not including the final slash.
// Used to collapse non-project code when displaying tracebacks.
func ServerRoot() string {
	return std.ServerRoot()
}

// Any arbitrary metadata you want to send with every subsequently sent item.
func Custom() map[string]interface{} {
	return std.Custom()
}

// Whether or not to use a custom client-side fingerprint.
func Fingerprint() bool {
	return std.Fingerprint()
}

// -- Reporting

// Report an item with level `critical`. This function recognizes arguments with the following types:
//    *http.Request
//    error
//    string
//    map[string]interface{}
//    int
// The string and error types are mutually exclusive.
// If an error is present then a stack trace is captured. If an int is also present then we skip
// that number of stack frames. If the map is present it is used as extra custom data in the
// item. If a string is present without an error, then we log a message without a stack
// trace. If a request is present we extract as much relevant information from it as we can.
func Critical(interfaces ...interface{}) {
	Log(CRIT, interfaces...)
}

// Report an item with level `error`. This function recognizes arguments with the following types:
//    *http.Request
//    error
//    string
//    map[string]interface{}
//    int
// The string and error types are mutually exclusive.
// If an error is present then a stack trace is captured. If an int is also present then we skip
// that number of stack frames. If the map is present it is used as extra custom data in the
// item. If a string is present without an error, then we log a message without a stack
// trace. If a request is present we extract as much relevant information from it as we can.
func Error(interfaces ...interface{}) {
	Log(ERR, interfaces...)
}

// Report an item with level `warning`. This function recognizes arguments with the following types:
//    *http.Request
//    error
//    string
//    map[string]interface{}
//    int
// The string and error types are mutually exclusive.
// If an error is present then a stack trace is captured. If an int is also present then we skip
// that number of stack frames. If the map is present it is used as extra custom data in the
// item. If a string is present without an error, then we log a message without a stack
// trace. If a request is present we extract as much relevant information from it as we can.
func Warning(interfaces ...interface{}) {
	Log(WARN, interfaces...)
}

// Report an item with level `info`. This function recognizes arguments with the following types:
//    *http.Request
//    error
//    string
//    map[string]interface{}
//    int
// The string and error types are mutually exclusive.
// If an error is present then a stack trace is captured. If an int is also present then we skip
// that number of stack frames. If the map is present it is used as extra custom data in the
// item. If a string is present without an error, then we log a message without a stack
// trace. If a request is present we extract as much relevant information from it as we can.
func Info(interfaces ...interface{}) {
	Log(INFO, interfaces...)
}

// Report an item with level `debug`. This function recognizes arguments with the following types:
//    *http.Request
//    error
//    string
//    map[string]interface{}
//    int
// The string and error types are mutually exclusive.
// If an error is present then a stack trace is captured. If an int is also present then we skip
// that number of stack frames. If the map is present it is used as extra custom data in the
// item. If a string is present without an error, then we log a message without a stack
// trace. If a request is present we extract as much relevant information from it as we can.
func Debug(interfaces ...interface{}) {
	Log(DEBUG, interfaces...)
}

// Report an item with the given level. This function recognizes arguments with the following types:
//    *http.Request
//    error
//    string
//    map[string]interface{}
//    int
// The string and error types are mutually exclusive.
// If an error is present then a stack trace is captured. If an int is also present then we skip
// that number of stack frames. If the map is present it is used as extra custom data in the
// item. If a string is present without an error, then we log a message without a stack
// trace. If a request is present we extract as much relevant information from it as we can.
func Log(level string, interfaces ...interface{}) {
	var r *http.Request
	var err error
	var skip int
	var extras map[string]interface{}
	var msg string
	for _, ival := range interfaces {
		switch val := ival.(type) {
		case *http.Request:
			r = val
		case error:
			err = val
		case int:
			skip = val
		case string:
			msg = val
		case map[string]interface{}:
			extras = val
		default:
			rollbarError("Unknown input type: %T", val)
		}
	}
	if err != nil {
		if r == nil {
			std.ErrorWithStackSkipWithExtras(level, err, skip, extras)
		} else {
			std.RequestErrorWithStackSkipWithExtras(level, r, err, skip, extras)
		}
	} else {
		if r == nil {
			std.MessageWithExtras(level, msg, extras)
		} else {
			std.RequestMessageWithExtras(level, r, msg, extras)
		}
	}
}

// -- Error reporting

// ErrorWithLevel asynchronously sends an error to Rollbar with the given severity level.
func ErrorWithLevel(level string, err error) {
	std.ErrorWithLevel(level, err)
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
