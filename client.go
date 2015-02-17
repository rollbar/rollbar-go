package rollbar

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"runtime"
	"sync"
	"time"
)

type Client interface {
	io.Closer

	// Rollbar access token.
	SetToken(token string)
	// All errors and messages will be submitted under this environment.
	SetEnvironment(environment string)
	// String describing the running code version on the server
	SetCodeVersion(codeVersion string)
	// host: The server hostname. Will be indexed.
	SetServerHost(serverHost string)
	// root: Path to the application code root, not including the final slash.
	// Used to collapse non-project code when displaying tracebacks.
	SetServerRoot(serverRoot string)
	// custom: Any arbitrary metadata you want to send.
	SetCustom(custom map[string]interface{})

	// Rollbar access token.
	GetToken() string
	// All errors and messages will be submitted under this environment.
	GetEnvironment() string
	// String describing the running code version on the server
	GetCodeVersion() string
	// host: The server hostname. Will be indexed.
	GetServerHost() string
	// root: Path to the application code root, not including the final slash.
	// Used to collapse non-project code when displaying tracebacks.
	GetServerRoot() string
	// custom: Any arbitrary metadata you want to send.
	GetCustom() map[string]interface{}

	// Error sends an error to Rollbar with the given severity level.
	Error(level string, err error)
	// ErrorWithExtras sends an error to Rollbar with the given severity
	// level with extra custom data.
	ErrorWithExtras(level string, err error, extras map[string]interface{})
	// ErrorWithStackSkip sends an error to Rollbar with the given severity
	// level and a given number of stack trace frames skipped.
	ErrorWithStackSkip(level string, err error, skip int)
	// ErrorWithStackSkipWithExtras sends an error to Rollbar with the given
	// severity level and a given number of stack trace frames skipped with
	// extra custom data.
	ErrorWithStackSkipWithExtras(level string, err error, skip int, extras map[string]interface{})

	// RequestError sends an error to Rollbar with the given severity level
	// and request-specific information.
	RequestError(level string, r *http.Request, err error)
	// RequestErrorWithExtras sends an error to Rollbar with the given
	// severity level and request-specific information with extra custom data.
	RequestErrorWithExtras(level string, r *http.Request, err error, extras map[string]interface{})
	// RequestErrorWithStackSkip sends an error to Rollbar with the given
	// severity level and a given number of stack trace frames skipped, in
	// addition to extra request-specific information.
	RequestErrorWithStackSkip(level string, r *http.Request, err error, skip int)
	// RequestErrorWithStackSkipWithExtras sends an error to Rollbar with
	// the given severity level and a given number of stack trace frames
	// skipped, in addition to extra request-specific information and extra
	// custom data.
	RequestErrorWithStackSkipWithExtras(level string, r *http.Request, err error, skip int, extras map[string]interface{})

	// Message sends a message to Rollbar with the given severity level.
	Message(level string, msg string)
	// MessageWithExtras sends a message to Rollbar with the given severity
	// level with extra custom data.
	MessageWithExtras(level string, msg string, extras map[string]interface{})
}

// AsyncClient is the default concrete implementation of the Client interface
// which sends all data to Rollbar asynchronously.
type AsyncClient struct {
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
	// custom: Any arbitrary metadata you want to send.
	Custom map[string]interface{}
	// Queue of messages to be sent.
	bodyChannel chan map[string]interface{}
	waitGroup   sync.WaitGroup
}

// New returns the default implementation of a Client
func New(token, environment, codeVersion, serverHost, serverRoot string) Client {
	return NewAsync(token, environment, codeVersion, serverHost, serverRoot)
}

// NewAsync builds an asynchronous implementation of the Client interface
func NewAsync(token, environment, codeVersion, serverHost, serverRoot string) *AsyncClient {
	buffer := 1000
	client := &AsyncClient{
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

func (c *AsyncClient) SetToken(token string) {
	c.Token = token
}

func (c *AsyncClient) SetEnvironment(environment string) {
	c.Environment = environment
}

func (c *AsyncClient) SetCodeVersion(codeVersion string) {
	c.CodeVersion = codeVersion
}

func (c *AsyncClient) SetServerHost(serverHost string) {
	c.ServerHost = serverHost
}

func (c *AsyncClient) SetServerRoot(serverRoot string) {
	c.ServerRoot = serverRoot
}

func (c *AsyncClient) SetCustom(custom map[string]interface{}) {
	c.Custom = custom
}

// -- Getters

func (c *AsyncClient) GetToken() string {
	return c.Token
}

func (c *AsyncClient) GetEnvironment() string {
	return c.Environment
}

func (c *AsyncClient) GetCodeVersion() string {
	return c.CodeVersion
}

func (c *AsyncClient) GetServerHost() string {
	return c.ServerHost
}

func (c *AsyncClient) GetServerRoot() string {
	return c.ServerRoot
}

func (c *AsyncClient) GetCustom() map[string]interface{} {
	return c.Custom
}

// -- Error reporting

var noExtras map[string]interface{}

func (c *AsyncClient) Error(level string, err error) {
	c.ErrorWithExtras(level, err, noExtras)
}

func (c *AsyncClient) ErrorWithExtras(level string, err error, extras map[string]interface{}) {
	c.ErrorWithStackSkipWithExtras(level, err, 1, extras)
}

func (c *AsyncClient) RequestError(level string, r *http.Request, err error) {
	c.RequestErrorWithExtras(level, r, err, noExtras)
}

func (c *AsyncClient) RequestErrorWithExtras(level string, r *http.Request, err error, extras map[string]interface{}) {
	c.RequestErrorWithStackSkipWithExtras(level, r, err, 1, extras)
}

func (c *AsyncClient) ErrorWithStackSkip(level string, err error, skip int) {
	c.ErrorWithStackSkipWithExtras(level, err, skip, noExtras)
}

func (c *AsyncClient) ErrorWithStackSkipWithExtras(level string, err error, skip int, extras map[string]interface{}) {
	body := c.buildBody(level, err.Error(), extras)
	data := body["data"].(map[string]interface{})
	errBody, fingerprint := errorBody(err, skip)
	data["body"] = errBody
	data["fingerprint"] = fingerprint

	c.push(body)
}

func (c *AsyncClient) RequestErrorWithStackSkip(level string, r *http.Request, err error, skip int) {
	c.RequestErrorWithStackSkipWithExtras(level, r, err, skip, noExtras)
}

func (c *AsyncClient) RequestErrorWithStackSkipWithExtras(level string, r *http.Request, err error, skip int, extras map[string]interface{}) {
	body := c.buildBody(level, err.Error(), extras)
	data := body["data"].(map[string]interface{})

	errBody, fingerprint := errorBody(err, skip)
	data["body"] = errBody
	data["fingerprint"] = fingerprint

	data["request"] = c.errorRequest(r)

	c.push(body)
}

// -- Message reporting

func (c *AsyncClient) Message(level string, msg string) {
	c.MessageWithExtras(level, msg, noExtras)
}

func (c *AsyncClient) MessageWithExtras(level string, msg string, extras map[string]interface{}) {
	body := c.buildBody(level, msg, extras)
	data := body["data"].(map[string]interface{})
	data["body"] = messageBody(msg)

	c.push(body)
}

// -- Misc.

// wait will block until the queue of errors / messages is empty.
func (c *AsyncClient) Wait() {
	c.waitGroup.Wait()
}

// Close on the asynchronous Client is an alias to Wait
func (c *AsyncClient) Close() error {
	c.Wait()
	return nil
}

// Build the main JSON structure that will be sent to Rollbar with the
// appropriate metadata.
func (c *AsyncClient) buildBody(level, title string, extras map[string]interface{}) map[string]interface{} {
	timestamp := time.Now().Unix()

	custom := c.Custom
	for k, v := range extras {
		custom[k] = v
	}

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
		"custom": custom,
	}

	return map[string]interface{}{
		"access_token": c.Token,
		"data":         data,
	}
}

// Extract error details from a Request to a format that Rollbar accepts.
func (c *AsyncClient) errorRequest(r *http.Request) map[string]interface{} {
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

// -- POST handling

// Queue the given JSON body to be POSTed to Rollbar.
func (c *AsyncClient) push(body map[string]interface{}) {
	if len(c.bodyChannel) < c.Buffer {
		c.waitGroup.Add(1)
		c.bodyChannel <- body
	} else {
		rollbarError("buffer full, dropping error on the floor")
	}
}

// POST the given JSON body to Rollbar synchronously.
func (c *AsyncClient) post(body map[string]interface{}) {
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
