package rollbar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"sync"
	"time"
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
	// ItemsPerMinute has the max number of items to send in a given minute
	ItemsPerMinute int
	// ErrorLevelFilters are the filtered error types and their corresponding levels
	ErrorLevelFilters map[reflect.Type]string
	// LoggerLevel is the logger level set globally
	LoggerLevel string
	// custom http client (http.DefaultClient used by default)
	httpClient *http.Client

	perMinCounter int
	startTime     time.Time
	lock          sync.RWMutex
}

// SetToken updates the token to use for future API requests.
func (t *baseTransport) SetToken(token string) {
	t.Token = token
}

// SetEndpoint updates the API endpoint to send items to.
func (t *baseTransport) SetEndpoint(endpoint string) {
	t.Endpoint = endpoint
}

// SetItemsPerMinute sets the max number of items to send in a given minute
func (t *baseTransport) SetItemsPerMinute(itemsPerMinute int) {
	t.lock.Lock()
	t.ItemsPerMinute = itemsPerMinute
	t.lock.Unlock()
}

// SetLogger updates the logger that this transport uses for reporting errors that occur while
// processing items.
func (t *baseTransport) SetLogger(logger ClientLogger) {
	t.lock.Lock()
	defer t.lock.Unlock()
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

func (t *baseTransport) clientPost(body io.Reader) (resp *http.Response, err error) {
	req, err := http.NewRequest("POST", t.Endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Rollbar-Access-Token", t.Token)
	return t.getHTTPClient().Do(req)
}

// post returns an error which indicates the type of error that occurred while attempting to
// send the body input to the endpoint given, or nil if no error occurred. If error is not nil, the
// boolean return parameter indicates whether the error is temporary or not. If this boolean return
// value is true then the caller could call this function again with the same input and possibly
// see a non-error response.
func (t *baseTransport) post(body map[string]interface{}) (bool, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()
	if len(t.Token) == 0 {
		rollbarError(t.Logger, "empty token")
		return false, nil
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		rollbarError(t.Logger, "failed to encode payload: %s", err.Error())
		return false, err
	}

	resp, err := t.clientPost(bytes.NewReader(jsonBody))
	if err != nil {
		rollbarError(t.Logger, "POST failed: %s", err.Error())
		return isTemporary(err), err
	}

	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()

	if resp.StatusCode != 200 {
		err = ErrHTTPError(resp.StatusCode)
		if !t.IsMessageFiltered(err, ERR) {
			rollbarError(t.Logger, "received response: %s", resp.Status)
		}
		// http.StatusTooManyRequests is only defined in Go 1.6+ so we use 429 directly
		isRateLimit := resp.StatusCode == 429
		return isRateLimit, err
	}

	return false, nil
}

func (t *baseTransport) shouldSend() bool {
	t.lock.RLock()
	defer t.lock.RUnlock()
	if t.ItemsPerMinute > 0 && t.perMinCounter >= t.ItemsPerMinute {
		rollbarError(t.Logger, fmt.Sprintf("item per minute limit reached: %d occurences, "+
			"ignoring errors until timeout", t.perMinCounter))
		return false
	}
	return true
}

// SetLoggerLevel sets the logger level globally
func (t *baseTransport) SetLoggerLevel(loggerLevel string) {
	t.LoggerLevel = loggerLevel
}

// SetErrorLevelFilters sets error level filters
func (t *baseTransport) SetErrorLevelFilters(errLevelFilters map[reflect.Type]string) {
	t.ErrorLevelFilters = errLevelFilters
}

// IsMessageFiltered determines if the message should be filtered or not
func (t *baseTransport) IsMessageFiltered(err interface{}, level string) bool {
	if LogLevelMap[t.LoggerLevel] >= LogLevelMap[level] {
		return true
	}

	for eType, lvl := range t.ErrorLevelFilters {
		if eType == reflect.TypeOf(err) && LogLevelMap[lvl] >= LogLevelMap[level] {
			return true
		}
	}

	return false
}
