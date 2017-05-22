package rollbar

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"runtime"
	"time"
)

type Client interface {
	io.Closer

	// Rollbar access token.
	SetToken(token string)
	// All errors and messages will be submitted under this environment.
	SetEnvironment(environment string)
	// Set the Platform to be reported for all items
	SetPlatform(platform string)
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
	Token() string
	// All errors and messages will be submitted under this environment.
	Environment() string
	// Platform is the platform reported for all Rollbar items. The default is
	// the running operating system (darwin, freebsd, linux, etc.) but it can
	// also be application specific (client, heroku, etc.).
	Platform() string
	// String describing the running code version on the server
	CodeVersion() string
	// host: The server hostname. Will be indexed.
	ServerHost() string
	// root: Path to the application code root, not including the final slash.
	// Used to collapse non-project code when displaying tracebacks.
	ServerRoot() string
	// custom: Any arbitrary metadata you want to send.
	Custom() map[string]interface{}

	// Error sends an error to Rollbar with the given severity level.
	Error(level string, err error)
	// Errorf sends an error to Rollbar with the given format string and arguments.
	Errorf(level string, format string, args ...interface{})
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

type configuration struct {
	token         string
	environment   string
	platform      string
	codeVersion   string
	serverHost    string
	serverRoot    string
	endpoint      string
	custom        map[string]interface{}
	filterHeaders *regexp.Regexp
	filterFields  *regexp.Regexp
}

func createConfiguration(token, environment, codeVersion, serverHost, serverRoot string) configuration {
	return configuration{
		token:         token,
		environment:   environment,
		platform:      runtime.GOOS,
		endpoint:      "https://api.rollbar.com/api/1/item",
		filterHeaders: regexp.MustCompile("Authorization"),
		filterFields:  regexp.MustCompile("password|secret|token"),
		codeVersion:   codeVersion,
		serverHost:    serverHost,
		serverRoot:    serverRoot,
	}
}

// Build the main JSON structure that will be sent to Rollbar with the
// appropriate metadata.
func buildBody(configuration configuration, level, title string, extras map[string]interface{}) map[string]interface{} {
	timestamp := time.Now().Unix()

	custom := configuration.custom
	for k, v := range extras {
		custom[k] = v
	}

	data := map[string]interface{}{
		"environment":  configuration.environment,
		"title":        title,
		"level":        level,
		"timestamp":    timestamp,
		"platform":     configuration.platform,
		"language":     "go",
		"code_version": configuration.codeVersion,
		"server": map[string]interface{}{
			"host": configuration.serverHost,
			"root": configuration.serverRoot,
		},
		"notifier": map[string]interface{}{
			"name":    NAME,
			"version": VERSION,
		},
		"custom": custom,
	}

	return map[string]interface{}{
		"access_token": configuration.token,
		"data":         data,
	}
}

// Extract error details from a Request to a format that Rollbar accepts.
func errorRequest(configuration configuration, r *http.Request) map[string]interface{} {
	cleanQuery := filterParams(configuration.filterFields, r.URL.Query())

	return map[string]interface{}{
		"url":     r.URL.String(),
		"method":  r.Method,
		"headers": flattenValues(filterParams(configuration.filterHeaders, r.Header)),

		// GET params
		"query_string": url.Values(cleanQuery).Encode(),
		"GET":          flattenValues(cleanQuery),

		// POST / PUT params
		"POST": flattenValues(filterParams(configuration.filterFields, r.Form)),
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

func clientPost(token, endpoint string, body map[string]interface{}) {
	if len(token) == 0 {
		rollbarError("empty token")
		return
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		rollbarError("failed to encode payload: %s", err.Error())
		return
	}

	resp, err := http.Post(endpoint, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		rollbarError("POST failed: %s", err.Error())
	} else if resp.StatusCode != 200 {
		rollbarError("received response: %s", resp.Status)
	}
	if resp != nil {
		resp.Body.Close()
	}
}
