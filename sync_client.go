package rollbar

import (
	"net/http"
	"net/url"
	"regexp"
	"runtime"
	"time"
)

// SyncClient is an alternate concrete implementation of the Client interface
// which sends all data to Rollbar synchronously.
type SyncClient struct {
	configuration configuration
}

// NewSync builds a synchronous implementation of the Client interface
func NewSync(token, environment, codeVersion, serverHost, serverRoot string) *SyncClient {
	configuration := configuration{
		token:         token,
		environment:   environment,
		endpoint:      "https://api.rollbar.com/api/1/item",
		filterHeaders: regexp.MustCompile("Authorization"),
		filterFields:  regexp.MustCompile("password|secret|token"),
		codeVersion:   codeVersion,
		serverHost:    serverHost,
		serverRoot:    serverRoot,
	}
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
	timestamp := time.Now().Unix()

	custom := c.configuration.custom
	for k, v := range extras {
		custom[k] = v
	}

	data := map[string]interface{}{
		"environment":  c.configuration.environment,
		"title":        title,
		"level":        level,
		"timestamp":    timestamp,
		"platform":     runtime.GOOS,
		"language":     "go",
		"code_version": c.configuration.codeVersion,
		"server": map[string]interface{}{
			"host": c.configuration.serverHost,
			"root": c.configuration.serverRoot,
		},
		"notifier": map[string]interface{}{
			"name":    NAME,
			"version": VERSION,
		},
		"custom": custom,
	}

	return map[string]interface{}{
		"access_token": c.configuration.token,
		"data":         data,
	}
}

// Extract error details from a Request to a format that Rollbar accepts.
func (c *SyncClient) errorRequest(r *http.Request) map[string]interface{} {
	cleanQuery := filterParams(c.configuration.filterFields, r.URL.Query())

	return map[string]interface{}{
		"url":     r.URL.String(),
		"method":  r.Method,
		"headers": flattenValues(filterParams(c.configuration.filterHeaders, r.Header)),

		// GET params
		"query_string": url.Values(cleanQuery).Encode(),
		"GET":          flattenValues(cleanQuery),

		// POST / PUT params
		"POST": flattenValues(filterParams(c.configuration.filterFields, r.Form)),
	}
}

// -- POST handling

// POST the given JSON body to Rollbar synchronously.
func (c *SyncClient) post(body map[string]interface{}) {
	clientPost(c.configuration.token, c.configuration.endpoint, body)
}
