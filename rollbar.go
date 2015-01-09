package rollbar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/adler32"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
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

type Rollbar struct {
	// Rollbar access token. If this is blank, no errors will be reported to
	// Rollbar.
	Token string
	// All errors and messages will be submitted under this environment.
	Environment string
	// API endpoint for Rollbar.
	Endpoint string
	// Maximum number of errors allowed in the sending queue before we start
	// dropping new errors on the floor.
	Buffer int
	// Filter HTTP Headers parameters from being sent to Rollbar.
	FilterHeaders *regexp.Regexp
	// Filter GET and POST parameters from being sent to Rollbar.
	FilterFields *regexp.Regexp
	// String describing the running code version on the server
	CodeVersion string
	// host: The server hostname. Will be indexed.
	ServerHost string
	// root: Path to the application code root, not including the final slash.
	// Used to collapse non-project code when displaying tracebacks.
	ServerRoot string
	// Queue of messages to be sent.
	bodyChannel chan map[string]interface{}
	waitGroup   sync.WaitGroup
}

type Client interface {
	Error(level string, err error)
	ErrorWithExtras(level string, err error, extras map[string]interface{})
	ErrorWithStackSkip(level string, err error, skip int)
	ErrorWithStackSkipWithExtras(level string, err error, skip int, extras map[string]interface{})

	RequestError(level string, r *http.Request, err error)
	RequestErrorWithExtras(level string, r *http.Request, err error, extras map[string]interface{})
	RequestErrorWithStackSkip(level string, r *http.Request, err error, skip int)
	RequestErrorWithStackSkipWithExtras(level string, r *http.Request, err error, skip int, extras map[string]interface{})

	Message(level string, msg string)
	MessageWithExtras(level string, msg string, extras map[string]interface{})

	Wait()
}

func New(token, environment, codeVersion, serverHost, serverRoot string) Client {
	return newRollbar(token, environment, codeVersion, serverHost, serverRoot)
}

func newRollbar(token, environment, codeVersion, serverHost, serverRoot string) *Rollbar {
	buffer := 1000
	client := &Rollbar{
		Token:         token,
		Environment:   environment,
		Endpoint:      "https://api.rollbar.com/api/1/item/",
		Buffer:        1000,
		FilterHeaders: regexp.MustCompile("Authorization"),
		FilterFields:  regexp.MustCompile("password|secret|token"),
		CodeVersion:   codeVersion,
		ServerHost:    serverHost,
		ServerRoot:    serverRoot,
		bodyChannel:   make(chan map[string]interface{}, buffer),
	}

	go func() {
		for body := range client.bodyChannel {
			client.post(body)
			client.waitGroup.Done()
		}
	}()
	return client
}

var (
	hostname, _ = os.Hostname()
	Std         = newRollbar("", "development", "", hostname, "")
)

func SetToken(token string) {
	Std.Token = token
}

func SetEnvironment(environment string) {
	Std.Environment = environment
}

func SetCodeVersion(codeVersion string) {
	Std.CodeVersion = codeVersion
}

func SetServerHost(serverHost string) {
	Std.ServerHost = serverHost
}

func SetServerRoot(serverRoot string) {
	Std.ServerRoot = serverRoot
}

// -- Error reporting

var noExtras map[string]interface{}

// Error asynchronously sends an error to Rollbar with the given severity level.
func Error(level string, err error) {
	Std.Error(level, err)
}

// Error asynchronously sends an error to Rollbar with the given severity level.
func (c *Rollbar) Error(level string, err error) {
	c.ErrorWithExtras(level, err, noExtras)
}

// Error asynchronously sends an error to Rollbar with the given severity level with extra custom data.
func ErrorWithExtras(level string, err error, extras map[string]interface{}) {
	Std.ErrorWithExtras(level, err, extras)
}

// Error asynchronously sends an error to Rollbar with the given severity level with extra custom data.
func (c *Rollbar) ErrorWithExtras(level string, err error, extras map[string]interface{}) {
	c.ErrorWithStackSkipWithExtras(level, err, 1, extras)
}

// RequestError asynchronously sends an error to Rollbar with the given
// severity level and request-specific information.
func RequestError(level string, r *http.Request, err error) {
	Std.RequestError(level, r, err)
}

// RequestError asynchronously sends an error to Rollbar with the given
// severity level and request-specific information.
func (c *Rollbar) RequestError(level string, r *http.Request, err error) {
	c.RequestErrorWithExtras(level, r, err, noExtras)
}

// RequestError asynchronously sends an error to Rollbar with the given
// severity level and request-specific information with extra custom data.
func RequestErrorWithExtras(level string, r *http.Request, err error, extras map[string]interface{}) {
	Std.RequestErrorWithExtras(level, r, err, extras)
}

// RequestError asynchronously sends an error to Rollbar with the given
// severity level and request-specific information with extra custom data.
func (c *Rollbar) RequestErrorWithExtras(level string, r *http.Request, err error, extras map[string]interface{}) {
	c.RequestErrorWithStackSkipWithExtras(level, r, err, 1, extras)
}

// ErrorWithStackSkip asynchronously sends an error to Rollbar with the given
// severity level and a given number of stack trace frames skipped.
func ErrorWithStackSkip(level string, err error, skip int) {
	Std.ErrorWithStackSkip(level, err, skip)
}

// ErrorWithStackSkip asynchronously sends an error to Rollbar with the given
// severity level and a given number of stack trace frames skipped.
func (c *Rollbar) ErrorWithStackSkip(level string, err error, skip int) {
	c.ErrorWithStackSkipWithExtras(level, err, skip, noExtras)
}

// ErrorWithStackSkip asynchronously sends an error to Rollbar with the given
// severity level and a given number of stack trace frames skipped with extra custom data.
func ErrorWithStackSkipWithExtras(level string, err error, skip int, extras map[string]interface{}) {
	Std.ErrorWithStackSkipWithExtras(level, err, skip, extras)
}

// ErrorWithStackSkip asynchronously sends an error to Rollbar with the given
// severity level and a given number of stack trace frames skipped with extra custom data.
func (c *Rollbar) ErrorWithStackSkipWithExtras(level string, err error, skip int, extras map[string]interface{}) {
	body := c.buildBody(level, err.Error(), extras)
	data := body["data"].(map[string]interface{})
	errBody, fingerprint := errorBody(err, skip)
	data["body"] = errBody
	data["fingerprint"] = fingerprint

	c.push(body)
}

// RequestErrorWithStackSkip asynchronously sends an error to Rollbar with the
// given severity level and a given number of stack trace frames skipped, in
// addition to extra request-specific information.
func RequestErrorWithStackSkip(level string, r *http.Request, err error, skip int) {
	Std.RequestErrorWithStackSkip(level, r, err, skip)
}

// RequestErrorWithStackSkip asynchronously sends an error to Rollbar with the
// given severity level and a given number of stack trace frames skipped, in
// addition to extra request-specific information.
func (c *Rollbar) RequestErrorWithStackSkip(level string, r *http.Request, err error, skip int) {
	c.RequestErrorWithStackSkipWithExtras(level, r, err, skip, noExtras)
}

// RequestErrorWithStackSkip asynchronously sends an error to Rollbar with the
// given severity level and a given number of stack trace frames skipped, in
// addition to extra request-specific information and extra custom data.
func RequestErrorWithStackSkipWithExtras(level string, r *http.Request, err error, skip int, extras map[string]interface{}) {
	Std.RequestErrorWithStackSkipWithExtras(level, r, err, skip, extras)
}

// RequestErrorWithStackSkip asynchronously sends an error to Rollbar with the
// given severity level and a given number of stack trace frames skipped, in
// addition to extra request-specific information and extra custom data.
func (c *Rollbar) RequestErrorWithStackSkipWithExtras(level string, r *http.Request, err error, skip int, extras map[string]interface{}) {
	body := c.buildBody(level, err.Error(), extras)
	data := body["data"].(map[string]interface{})

	errBody, fingerprint := errorBody(err, skip)
	data["body"] = errBody
	data["fingerprint"] = fingerprint

	data["request"] = c.errorRequest(r)

	c.push(body)
}

// -- Message reporting

// Message asynchronously sends a message to Rollbar with the given severity
// level. Rollbar request is asynchronous.
func Message(level string, msg string) {
	Std.Message(level, msg)
}

// Message asynchronously sends a message to Rollbar with the given severity
// level. Rollbar request is asynchronous.
func (c *Rollbar) Message(level string, msg string) {
	c.MessageWithExtras(level, msg, noExtras)
}

// Message asynchronously sends a message to Rollbar with the given severity
// level with extra custom data. Rollbar request is asynchronous.
func MessageWithExtras(level string, msg string, extras map[string]interface{}) {
	Std.MessageWithExtras(level, msg, extras)
}

// Message asynchronously sends a message to Rollbar with the given severity
// level with extra custom data. Rollbar request is asynchronous.
func (c *Rollbar) MessageWithExtras(level string, msg string, extras map[string]interface{}) {
	body := c.buildBody(level, msg, extras)
	data := body["data"].(map[string]interface{})
	data["body"] = messageBody(msg)

	c.push(body)
}

// -- Misc.

// Wait will block until the queue of errors / messages is empty.
func Wait() {
	Std.Wait()
}

// Wait will block until the queue of errors / messages is empty.
func (c *Rollbar) Wait() {
	c.waitGroup.Wait()
}

// Build the main JSON structure that will be sent to Rollbar with the
// appropriate metadata.
func (c *Rollbar) buildBody(level, title string, extras map[string]interface{}) map[string]interface{} {
	timestamp := time.Now().Unix()
	data := map[string]interface{}{
		"environment":  c.Environment,
		"title":        title,
		"level":        level,
		"timestamp":    timestamp,
		"platform":     runtime.GOOS,
		"language":     "go",
		"code_version": c.CodeVersion,
		"server": map[string]interface{}{
			"host": c.ServerHost,
			"root": c.ServerRoot,
		},
		"notifier": map[string]interface{}{
			"name":    NAME,
			"version": VERSION,
		},
	}

	for k, v := range extras {
		data[k] = v
	}

	return map[string]interface{}{
		"access_token": c.Token,
		"data":         data,
	}
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

// Extract error details from a Request to a format that Rollbar accepts.
func (c *Rollbar) errorRequest(r *http.Request) map[string]interface{} {
	cleanQuery := filterParams(c.FilterFields, r.URL.Query())

	return map[string]interface{}{
		"url":     r.URL.String(),
		"method":  r.Method,
		"headers": flattenValues(filterParams(c.FilterHeaders, r.Header)),

		// GET params
		"query_string": url.Values(cleanQuery).Encode(),
		"GET":          flattenValues(cleanQuery),

		// POST / PUT params
		"POST": flattenValues(filterParams(c.FilterFields, r.Form)),
	}
}

// filterParams filters sensitive information like passwords from being sent to
// Rollbar.
func filterParams(pattern *regexp.Regexp, values map[string][]string) map[string][]string {
	for key, _ := range values {
		if pattern.Match([]byte(key)) {
			values[key] = []string{FILTERED}
		}
	}

	return values
}

func flattenValues(values map[string][]string) map[string]interface{} {
	result := make(map[string]interface{})

	for k, v := range values {
		if len(v) == 1 {
			result[k] = v[0]
		} else {
			result[k] = v
		}
	}

	return result
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

// Queue the given JSON body to be POSTed to Rollbar.
func (c *Rollbar) push(body map[string]interface{}) {
	if len(c.bodyChannel) < c.Buffer {
		c.waitGroup.Add(1)
		c.bodyChannel <- body
	} else {
		rollbarError("buffer full, dropping error on the floor")
	}
}

// POST the given JSON body to Rollbar synchronously.
func (c *Rollbar) post(body map[string]interface{}) {
	if len(c.Token) == 0 {
		rollbarError("empty token")
		return
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		rollbarError("failed to encode payload: %s", err.Error())
		return
	}

	resp, err := http.Post(c.Endpoint, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		rollbarError("POST failed: %s", err.Error())
	} else if resp.StatusCode != 200 {
		rollbarError("received response: %s", resp.Status)
	}
	if resp != nil {
		resp.Body.Close()
	}
}

// -- rollbarError

func rollbarError(format string, args ...interface{}) {
	format = "Rollbar error: " + format + "\n"
	log.Printf(format, args...)
}
