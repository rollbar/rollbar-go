package rollbar

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
)

type baseTransport struct {
	// Rollbar access token used by this transport for communication with the Rollbar API.
	Token string
	// Endpoint to post items to.
	Endpoint string
	// Logger used to report errors when sending data to Rollbar, e.g.
	// when the Rollbar API returns 409 Too Many Requests response.
	// If not set, the client will use the standard log.Printf by default.
	Logger ClientLogger
	// RetryAttempts is how often to attempt to resend an item when a temporary network error occurs
	// This defaults to DefaultRetryAttempts
	// Set this value to 0 if you do not want retries to happen
	RetryAttempts int
	// PrintPayloadOnError is whether or not to output the payload to the set logger or to stderr if
	// an error occurs during transport to the Rollbar API.
	PrintPayloadOnError bool
	// custom http client (http.DefaultClient used by default)
	httpClient *http.Client
}

// SetToken updates the token to use for future API requests.
func (t *baseTransport) SetToken(token string) {
	t.Token = token
}

// SetEndpoint updates the API endpoint to send items to.
func (t *baseTransport) SetEndpoint(endpoint string) {
	t.Endpoint = endpoint
}

// SetLogger updates the logger that this transport uses for reporting errors that occur while
// processing items.
func (t *baseTransport) SetLogger(logger ClientLogger) {
	t.Logger = logger
}

// SetRetryAttempts is how often to attempt to resend an item when a temporary network error occurs
// This defaults to DefaultRetryAttempts
// Set this value to 0 if you do not want retries to happen
func (t *baseTransport) SetRetryAttempts(retryAttempts int) {
	t.RetryAttempts = retryAttempts
}

// SetPrintPayloadOnError is whether or not to output the payload to stderr if an error occurs during
// transport to the Rollbar API.
func (t *baseTransport) SetPrintPayloadOnError(printPayloadOnError bool) {
	t.PrintPayloadOnError = printPayloadOnError
}

// SetHTTPClient sets custom http client. http.DefaultClient is used by default
func (t *baseTransport) SetHTTPClient(c *http.Client) {
	t.httpClient = c
}

// getHTTPClient returns either custom client (if set) or http.DefaultClient
func (t *baseTransport) getHTTPClient() *http.Client {
	if t.httpClient != nil {
		return t.httpClient
	}

	return http.DefaultClient
}

// post returns an error which indicates the type of error that occurred while attempting to
// send the body input to the endpoint given, or nil if no error occurred. If error is not nil, the
// boolean return parameter indicates whether the error is temporary or not. If this boolean return
// value is true then the caller could call this function again with the same input and possibly
// see a non-error response.
func (t *baseTransport) post(body map[string]interface{}) (bool, error) {
	if len(t.Token) == 0 {
		rollbarError(t.Logger, "empty token")
		return false, nil
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		rollbarError(t.Logger, "failed to encode payload: %s", err.Error())
		return false, err
	}

	resp, err := t.getHTTPClient().Post(t.Endpoint, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		rollbarError(t.Logger, "POST failed: %s", err.Error())
		return isTemporary(err), err
	}

	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()

	if resp.StatusCode != 200 {
		rollbarError(t.Logger, "received response: %s", resp.Status)
		// http.StatusTooManyRequests is only defined in Go 1.6+ so we use 429 directly
		isRateLimit := resp.StatusCode == 429
		return isRateLimit, ErrHTTPError(resp.StatusCode)
	}

	return false, nil
}
