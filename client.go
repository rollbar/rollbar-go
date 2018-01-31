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

// An instance of Client can be used to interact with Rollbar via the configured Transport.
// The functions at the root of the `rollbar` package are the recommend way of using a Client. One
// should not need to manage instances of the Client type manually in most normal scenarios.
// However, if you want to customize the underlying transport layer, or you need to have
// independent instances of a Client, then you can use the constructors provided for this
// type.
type Client struct {
	io.Closer
	// Transport used to send data to the Rollbar API. By default an asyncronous
	// implementation of the Transport interface is used.
	Transport     Transport
	configuration configuration
}

// New returns the default implementation of a Client.
// This uses the AsyncTransport.
func New(token, environment, codeVersion, serverHost, serverRoot string) *Client {
	return NewAsync(token, environment, codeVersion, serverHost, serverRoot)
}

// NewAsync builds a Client with the asynchronous implementation of the transport interface.
func NewAsync(token, environment, codeVersion, serverHost, serverRoot string) *Client {
	configuration := createConfiguration(token, environment, codeVersion, serverHost, serverRoot)
	transport := NewTransport(token, configuration.endpoint)
	return &Client{
		Transport:     transport,
		configuration: configuration,
	}
}

// NewSync builds a Client with the synchronous implementation of the transport interface.
func NewSync(token, environment, codeVersion, serverHost, serverRoot string) *Client {
	configuration := createConfiguration(token, environment, codeVersion, serverHost, serverRoot)
	transport := NewSyncTransport(token, configuration.endpoint)
	return &Client{
		Transport:     transport,
		configuration: configuration,
	}
}

// A Rollbar access token with scope "post_server_item"
// It is required to set this value before any of the other functions herein will be able to work
// properly. This also configures the underlying Transport.
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

// Person information for identifying a user associated with
// any subsequent errors or messages. Only id is required to be
// non-empty.
func (c *Client) SetPerson(id, username, email string) {
	c.configuration.person = person{
		id:       id,
		username: username,
		email:    email,
	}
}

// ClearPerson clears any previously set person information. See `SetPerson` for more
// information.
func (c *Client) ClearPerson() {
	c.configuration.person = person{}
}

// Whether or not to use custom client-side fingerprint
func (c *Client) SetFingerprint(fingerprint bool) {
	c.configuration.fingerprint = fingerprint
}

// Set the logger on the underlying transport
func (c *Client) SetLogger(logger ClientLogger) {
	c.Transport.SetLogger(logger)
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

// CheckIgnore is called during the recovery process of a panic that
// occurred inside a function wrapped by Wrap or WrapAndWait
// Return true if you wish to ignore this panic, false if you wish to
// report it to Rollbar. If an error is the argument to the panic, then
// this function is called with the result of calling Error(), otherwise
// the string representation of the value is passed to this function.
func (c *Client) SetCheckIgnore(checkIgnore func(string) bool) {
	c.configuration.checkIgnore = checkIgnore
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

// The server hostname. Will be indexed.
func (c *Client) ServerHost() string {
	return c.configuration.serverHost
}

// Path to the application code root, not including the final slash.
// Used to collapse non-project code when displaying tracebacks.
func (c *Client) ServerRoot() string {
	return c.configuration.serverRoot
}

// Any arbitrary metadata you want to send with every subsequently sent item.
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

// ErrorWithLevel sends an error to Rollbar with the given severity level.
func (c *Client) ErrorWithLevel(level string, err error) {
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

// -- Panics

// Wrap calls f and then recovers and reports a panic to Rollbar if it occurs.
// If an error is captured it is subsequently returned
func (c *Client) Wrap(f func()) (err interface{}) {
	defer func() {
		err = recover()
		switch val := err.(type) {
		case nil:
			return
		case error:
			if c.configuration.checkIgnore(val.Error()) {
				return
			}
			c.ErrorWithStackSkip(CRIT, val, 2)
		default:
			str := fmt.Sprint(val)
			if c.configuration.checkIgnore(str) {
				return
			}
			c.Message(CRIT, str)
		}
	}()

	f()
	return
}

// WrapAndWait calls f, and recovers and reports a panic to Rollbar if it occurs.
// This also waits before returning to ensure the message was reported
// If an error is captured it is subsequently returned.
func (c *Client) WrapAndWait(f func()) (err interface{}) {
	defer func() {
		err = recover()
		switch val := err.(type) {
		case nil:
			return
		case error:
			if c.configuration.checkIgnore(val.Error()) {
				return
			}
			c.ErrorWithStackSkip(CRIT, val, 2)
		default:
			str := fmt.Sprint(val)
			if c.configuration.checkIgnore(str) {
				return
			}
			c.Message(CRIT, str)
		}
		c.Wait()
	}()

	f()
	return
}

// Wait will call the Wait method of the Transport. If using an asyncronous
// transport then this will block until the queue of
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

func (c *Client) buildBody(level, title string, extras map[string]interface{}) map[string]interface{} {
	return buildBody(c.configuration, level, title, extras)
}

func (c *Client) requestDetails(r *http.Request) map[string]interface{} {
	return requestDetails(c.configuration, r)
}

func (c *Client) push(body map[string]interface{}) error {
	return c.Transport.Send(body)
}

type person struct {
	id       string
	username string
	email    string
}

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
	checkIgnore  func(string) bool
	person       person
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
		checkIgnore:  func(_s string) bool { return false },
		person:       person{},
	}
}

func clientPost(token, endpoint string, body map[string]interface{}, logger ClientLogger) error {
	if len(token) == 0 {
		rollbarError(logger, "empty token")
		return nil
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		rollbarError(logger, "failed to encode payload: %s", err.Error())
		return err
	}

	resp, err := http.Post(endpoint, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		rollbarError(logger, "POST failed: %s", err.Error())
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		rollbarError(logger, "received response: %s", resp.Status)
		return ErrHTTPError(resp.StatusCode)
	}

	return nil
}
