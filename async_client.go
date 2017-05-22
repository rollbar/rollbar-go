package rollbar

import (
	"net/http"
	"sync"
)

// AsyncClient is the default concrete implementation of the Client interface
// which sends all data to Rollbar asynchronously.
type AsyncClient struct {
	// Maximum number of errors allowed in the sending queue before we start
	// dropping new errors on the floor.
	Buffer int

	configuration configuration
	bodyChannel   chan map[string]interface{}
	waitGroup     sync.WaitGroup
}

// New returns the default implementation of a Client
func New(token, environment, codeVersion, serverHost, serverRoot string) Client {
	return NewAsync(token, environment, codeVersion, serverHost, serverRoot)
}

// NewAsync builds an asynchronous implementation of the Client interface
func NewAsync(token, environment, codeVersion, serverHost, serverRoot string) *AsyncClient {
	buffer := 1000
	configuration := createConfiguration(token, environment, codeVersion, serverHost, serverRoot)
	client := &AsyncClient{
		Buffer:        buffer,
		configuration: configuration,
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
	c.configuration.token = token
}

func (c *AsyncClient) SetEnvironment(environment string) {
	c.configuration.environment = environment
}

func (c *AsyncClient) SetPlatform(platform string) {
	c.configuration.platform = platform
}

func (c *AsyncClient) SetCodeVersion(codeVersion string) {
	c.configuration.codeVersion = codeVersion
}

func (c *AsyncClient) SetServerHost(serverHost string) {
	c.configuration.serverHost = serverHost
}

func (c *AsyncClient) SetServerRoot(serverRoot string) {
	c.configuration.serverRoot = serverRoot
}

func (c *AsyncClient) SetCustom(custom map[string]interface{}) {
	c.configuration.custom = custom
}

// -- Getters

func (c *AsyncClient) Token() string {
	return c.configuration.token
}

func (c *AsyncClient) Environment() string {
	return c.configuration.environment
}

func (c *AsyncClient) Platform() string {
	return c.configuration.platform
}

func (c *AsyncClient) CodeVersion() string {
	return c.configuration.codeVersion
}

func (c *AsyncClient) ServerHost() string {
	return c.configuration.serverHost
}

func (c *AsyncClient) ServerRoot() string {
	return c.configuration.serverRoot
}

func (c *AsyncClient) Custom() map[string]interface{} {
	return c.configuration.custom
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
	return buildBody(c.configuration, level, title, extras)
}

// Extract error details from a Request to a format that Rollbar accepts.
func (c *AsyncClient) errorRequest(r *http.Request) map[string]interface{} {
	return errorRequest(c.configuration, r)
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
	clientPost(c.configuration.token, c.configuration.endpoint, body)
}
