package rollbar

import (
	"fmt"
	"net/http"
)

// SyncClient is an alternate concrete implementation of the Client interface
// which sends all data to Rollbar synchronously.
type SyncClient struct {
	configuration configuration
}

// NewSync builds a synchronous implementation of the Client interface
func NewSync(token, environment, codeVersion, serverHost, serverRoot string) *SyncClient {
	configuration := createConfiguration(token, environment, codeVersion, serverHost, serverRoot)
	client := &SyncClient{
		configuration: configuration,
	}
	return client
}

func (c *SyncClient) SetToken(token string) {
	c.configuration.token = token
}

func (c *SyncClient) SetEnvironment(environment string) {
	c.configuration.environment = environment
}

func (c *SyncClient) SetPlatform(platform string) {
	c.configuration.platform = platform
}

func (c *SyncClient) SetCodeVersion(codeVersion string) {
	c.configuration.codeVersion = codeVersion
}

func (c *SyncClient) SetServerHost(serverHost string) {
	c.configuration.serverHost = serverHost
}

func (c *SyncClient) SetServerRoot(serverRoot string) {
	c.configuration.serverRoot = serverRoot
}

func (c *SyncClient) SetCustom(custom map[string]interface{}) {
	c.configuration.custom = custom
}

func (c *SyncClient) Token() string {
	return c.configuration.token
}

func (c *SyncClient) Environment() string {
	return c.configuration.environment
}

func (c *SyncClient) Platform() string {
	return c.configuration.platform
}

func (c *SyncClient) CodeVersion() string {
	return c.configuration.codeVersion
}

func (c *SyncClient) ServerHost() string {
	return c.configuration.serverHost
}

func (c *SyncClient) ServerRoot() string {
	return c.configuration.serverRoot
}

func (c *SyncClient) Custom() map[string]interface{} {
	return c.configuration.custom
}

// -- Error reporting

func (c *SyncClient) Error(level string, err error) {
	c.ErrorWithExtras(level, err, noExtras)
}

func (c *SyncClient) Errorf(level string, format string, args ...interface{}) {
	c.ErrorWithStackSkipWithExtras(level, fmt.Errorf(format, args...), 1, noExtras)
}

func (c *SyncClient) ErrorWithExtras(level string, err error, extras map[string]interface{}) {
	c.ErrorWithStackSkipWithExtras(level, err, 1, extras)
}

func (c *SyncClient) RequestError(level string, r *http.Request, err error) {
	c.RequestErrorWithExtras(level, r, err, noExtras)
}

func (c *SyncClient) RequestErrorWithExtras(level string, r *http.Request, err error, extras map[string]interface{}) {
	c.RequestErrorWithStackSkipWithExtras(level, r, err, 1, extras)
}

func (c *SyncClient) ErrorWithStackSkip(level string, err error, skip int) {
	c.ErrorWithStackSkipWithExtras(level, err, skip, noExtras)
}

func (c *SyncClient) ErrorWithStackSkipWithExtras(level string, err error, skip int, extras map[string]interface{}) {
	body := c.buildBody(level, err.Error(), extras)
	data := body["data"].(map[string]interface{})
	errBody, fingerprint := errorBody(err, skip)
	data["body"] = errBody
	data["fingerprint"] = fingerprint

	c.post(body)
}

func (c *SyncClient) RequestErrorWithStackSkip(level string, r *http.Request, err error, skip int) {
	c.RequestErrorWithStackSkipWithExtras(level, r, err, skip, noExtras)
}

func (c *SyncClient) RequestErrorWithStackSkipWithExtras(level string, r *http.Request, err error, skip int, extras map[string]interface{}) {
	body := c.buildBody(level, err.Error(), extras)
	data := body["data"].(map[string]interface{})

	errBody, fingerprint := errorBody(err, skip)
	data["body"] = errBody
	data["fingerprint"] = fingerprint

	data["request"] = c.errorRequest(r)

	c.post(body)
}

// -- Message reporting

func (c *SyncClient) Message(level string, msg string) {
	c.MessageWithExtras(level, msg, noExtras)
}

func (c *SyncClient) MessageWithExtras(level string, msg string, extras map[string]interface{}) {
	body := c.buildBody(level, msg, extras)
	data := body["data"].(map[string]interface{})
	data["body"] = messageBody(msg)

	c.post(body)
}

// -- Misc.

// Close on the synchronous Client does nothing
func (c *SyncClient) Close() error {
	return nil
}

// Build the main JSON structure that will be sent to Rollbar with the
// appropriate metadata.
func (c *SyncClient) buildBody(level, title string, extras map[string]interface{}) map[string]interface{} {
	return buildBody(c.configuration, level, title, extras)
}

// Extract error details from a Request to a format that Rollbar accepts.
func (c *SyncClient) errorRequest(r *http.Request) map[string]interface{} {
	return errorRequest(c.configuration, r)
}

// -- POST handling

// POST the given JSON body to Rollbar synchronously.
func (c *SyncClient) post(body map[string]interface{}) error {
	return clientPost(c.configuration.token, c.configuration.endpoint, body)
}
