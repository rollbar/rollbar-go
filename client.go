package rollbar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"runtime"
)

// AsyncClient is the default concrete implementation of the Client interface
// which sends all data to Rollbar asynchronously.
type Client struct {
	io.Closer
	Transport     Transport
	configuration configuration
}

// New returns the default implementation of a Client
func New(token, environment, codeVersion, serverHost, serverRoot string) *Client {
	return NewAsync(token, environment, codeVersion, serverHost, serverRoot)
}

// NewAsync builds an asynchronous implementation of the Client interface
func NewAsync(token, environment, codeVersion, serverHost, serverRoot string) *Client {
	configuration := createConfiguration(token, environment, codeVersion, serverHost, serverRoot)
	transport := NewTransport(token, configuration.endpoint)
	return &Client{
		Transport:     transport,
		configuration: configuration,
	}
}

func NewSync(token, environment, codeVersion, serverHost, serverRoot string) *Client {
	configuration := createConfiguration(token, environment, codeVersion, serverHost, serverRoot)
	transport := NewSyncTransport(token, configuration.endpoint)
	return &Client{
		Transport:     transport,
		configuration: configuration,
	}
}

// Rollbar access token.
func (c *Client) SetToken(token string) {
	c.configuration.token = token
	c.Transport.SetToken(token)
}

// All errors and messages will be submitted under this environment.
func (c *Client) SetEnvironment(environment string) {
	c.configuration.environment = environment
}

// The endpoint to post items to
func (c *Client) SetEndpoint(endpoint string) {
	c.configuration.endpoint = endpoint
	c.Transport.SetEndpoint(endpoint)
}

// Set the Platform to be reported for all items
func (c *Client) SetPlatform(platform string) {
	c.configuration.platform = platform
}

// String describing the running code version on the server
func (c *Client) SetCodeVersion(codeVersion string) {
	c.configuration.codeVersion = codeVersion
}

// host: The server hostname. Will be indexed.
func (c *Client) SetServerHost(serverHost string) {
	c.configuration.serverHost = serverHost
}

// root: Path to the application code root, not including the final slash.
// Used to collapse non-project code when displaying tracebacks.
func (c *Client) SetServerRoot(serverRoot string) {
	c.configuration.serverRoot = serverRoot
}

// custom: Any arbitrary metadata you want to send.
func (c *Client) SetCustom(custom map[string]interface{}) {
	c.configuration.custom = custom
}

// Whether or not to use custom client-side fingerprint
func (c *Client) SetFingerprint(fingerprint bool) {
	c.configuration.fingerprint = fingerprint
}

// Regular expression used to match headers for scrubbing
// The default value is regexp.MustCompile("Authorization")
func (c *Client) SetScrubHeaders(headers *regexp.Regexp) {
	c.configuration.scrubHeaders = headers
}

// Regular expression to match keys in the item payload for scrubbing
// The default vlaue is regexp.MustCompile("password|secret|token"),
func (c *Client) SetScrubFields(fields *regexp.Regexp) {
	c.configuration.scrubFields = fields
}

// -- Getters

// Rollbar access token.
func (c *Client) Token() string {
	return c.configuration.token
}

// All errors and messages will be submitted under this environment.
func (c *Client) Environment() string {
	return c.configuration.environment
}

// The endpoint used for posting items
func (c *Client) Endpoint() string {
	return c.configuration.endpoint
}

// Platform is the platform reported for all Rollbar items. The default is
// the running operating system (darwin, freebsd, linux, etc.) but it can
// also be application specific (client, heroku, etc.).
func (c *Client) Platform() string {
	return c.configuration.platform
}

// String describing the running code version on the server
func (c *Client) CodeVersion() string {
	return c.configuration.codeVersion
}

// host: The server hostname. Will be indexed.
func (c *Client) ServerHost() string {
	return c.configuration.serverHost
}

// root: Path to the application code root, not including the final slash.
// Used to collapse non-project code when displaying tracebacks.
func (c *Client) ServerRoot() string {
	return c.configuration.serverRoot
}

// custom: Any arbitrary metadata you want to send.
func (c *Client) Custom() map[string]interface{} {
	return c.configuration.custom
}

// Whether or not to use custom client-side fingerprint
func (c *Client) Fingerprint() bool {
	return c.configuration.fingerprint
}

// Regular expression used to match headers for scrubbing
func (c *Client) ScrubHeaders() *regexp.Regexp {
	return c.configuration.scrubHeaders
}

// Regular expression to match keys in the item payload for scrubbing
func (c *Client) ScrubFields() *regexp.Regexp {
	return c.configuration.scrubFields
}

// -- Error reporting

var noExtras map[string]interface{}

// Error sends an error to Rollbar with the given severity level.
func (c *Client) Error(level string, err error) {
	c.ErrorWithExtras(level, err, noExtras)
}

// Errorf sends an error to Rollbar with the given format string and arguments.
func (c *Client) Errorf(level string, format string, args ...interface{}) {
	c.ErrorWithStackSkipWithExtras(level, fmt.Errorf(format, args...), 1, noExtras)
}

// ErrorWithExtras sends an error to Rollbar with the given severity
// level with extra custom data.
func (c *Client) ErrorWithExtras(level string, err error, extras map[string]interface{}) {
	c.ErrorWithStackSkipWithExtras(level, err, 1, extras)
}

// RequestError sends an error to Rollbar with the given severity level
// and request-specific information.
func (c *Client) RequestError(level string, r *http.Request, err error) {
	c.RequestErrorWithExtras(level, r, err, noExtras)
}

// RequestErrorWithExtras sends an error to Rollbar with the given
// severity level and request-specific information with extra custom data.
func (c *Client) RequestErrorWithExtras(level string, r *http.Request, err error, extras map[string]interface{}) {
	c.RequestErrorWithStackSkipWithExtras(level, r, err, 1, extras)
}

// ErrorWithStackSkip sends an error to Rollbar with the given severity
// level and a given number of stack trace frames skipped.
func (c *Client) ErrorWithStackSkip(level string, err error, skip int) {
	c.ErrorWithStackSkipWithExtras(level, err, skip, noExtras)
}

// ErrorWithStackSkipWithExtras sends an error to Rollbar with the given
// severity level and a given number of stack trace frames skipped with
// extra custom data.
func (c *Client) ErrorWithStackSkipWithExtras(level string, err error, skip int, extras map[string]interface{}) {
	body := c.buildBody(level, err.Error(), extras)
	addErrorToBody(c.configuration, body, err, skip)
	c.push(body)
}

// RequestErrorWithStackSkip sends an error to Rollbar with the given
// severity level and a given number of stack trace frames skipped, in
// addition to extra request-specific information.
func (c *Client) RequestErrorWithStackSkip(level string, r *http.Request, err error, skip int) {
	c.RequestErrorWithStackSkipWithExtras(level, r, err, skip, noExtras)
}

// RequestErrorWithStackSkipWithExtras sends an error to Rollbar with
// the given severity level and a given number of stack trace frames
// skipped, in addition to extra request-specific information and extra
// custom data.
func (c *Client) RequestErrorWithStackSkipWithExtras(level string, r *http.Request, err error, skip int, extras map[string]interface{}) {
	body := c.buildBody(level, err.Error(), extras)
	data := addErrorToBody(c.configuration, body, err, skip)
	data["request"] = c.requestDetails(r)
	c.push(body)
}

// -- Message reporting

// Message sends a message to Rollbar with the given severity level.
func (c *Client) Message(level string, msg string) {
	c.MessageWithExtras(level, msg, noExtras)
}

// MessageWithExtras sends a message to Rollbar with the given severity
// level with extra custom data.
func (c *Client) MessageWithExtras(level string, msg string, extras map[string]interface{}) {
	body := c.buildBody(level, msg, extras)
	data := body["data"].(map[string]interface{})
	data["body"] = messageBody(msg)
	c.push(body)
}

// RequestMessage sends a message to Rollbar with the given severity level
// and request-specific information.
func (c *Client) RequestMessage(level string, r *http.Request, msg string) {
	c.RequestMessageWithExtras(level, r, msg, noExtras)
}

// RequestMessageWithExtras sends a message to Rollbar with the given
// severity level and request-specific information with extra custom data.
func (c *Client) RequestMessageWithExtras(level string, r *http.Request, msg string, extras map[string]interface{}) {
	body := c.buildBody(level, msg, extras)
	data := body["data"].(map[string]interface{})
	data["body"] = messageBody(msg)
	data["request"] = c.requestDetails(r)
	c.push(body)
}

// -- Misc.

// Wait will call the Wait method of the Transport. If using an asyncronous
// transport then this will blow until until the queue of
// errors / messages is empty. If using a syncronous transport then there
// is no queue so this will be a no-op.
func (c *Client) Wait() {
	c.Transport.Wait()
}

// Close delegates to the Close method of the Transport. For the asyncronous
// transport this is an alias for Wait, and is a no-op for the synchronous
// transport.
func (c *Client) Close() error {
	return c.Transport.Close()
}

// Build the main JSON structure that will be sent to Rollbar with the
// appropriate metadata.
func (c *Client) buildBody(level, title string, extras map[string]interface{}) map[string]interface{} {
	return buildBody(c.configuration, level, title, extras)
}

// Extract error details from a Request to a format that Rollbar accepts.
func (c *Client) requestDetails(r *http.Request) map[string]interface{} {
	return requestDetails(c.configuration, r)
}

// -- POST handling

// Queue the given JSON body to be POSTed to Rollbar.
func (c *Client) push(body map[string]interface{}) error {
	return c.Transport.Send(body)
}

// -- Internal

type configuration struct {
	token        string
	environment  string
	platform     string
	codeVersion  string
	serverHost   string
	serverRoot   string
	endpoint     string
	custom       map[string]interface{}
	fingerprint  bool
	scrubHeaders *regexp.Regexp
	scrubFields  *regexp.Regexp
}

func createConfiguration(token, environment, codeVersion, serverHost, serverRoot string) configuration {
	hostname := serverHost
	if hostname == "" {
		hostname, _ = os.Hostname()
	}
	return configuration{
		token:        token,
		environment:  environment,
		platform:     runtime.GOOS,
		endpoint:     "https://api.rollbar.com/api/1/item/",
		scrubHeaders: regexp.MustCompile("Authorization"),
		scrubFields:  regexp.MustCompile("password|secret|token"),
		codeVersion:  codeVersion,
		serverHost:   hostname,
		serverRoot:   serverRoot,
		fingerprint:  false,
	}
}

// -- POST handling

func clientPost(token, endpoint string, body map[string]interface{}) error {
	if len(token) == 0 {
		rollbarError("empty token")
		return nil
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		rollbarError("failed to encode payload: %s", err.Error())
		return err
	}

	resp, err := http.Post(endpoint, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		rollbarError("POST failed: %s", err.Error())
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		rollbarError("received response: %s", resp.Status)
		return ErrHTTPError(resp.StatusCode)
	}

	return nil
}
