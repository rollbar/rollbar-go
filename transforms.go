package rollbar

import (
	"fmt"
	"hash/adler32"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// Build the main JSON structure that will be sent to Rollbar with the
// appropriate metadata.
func buildBody(configuration configuration, level, title string, extras map[string]interface{}) map[string]interface{} {
	timestamp := time.Now().Unix()

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
	}

	custom := buildCustom(configuration.custom, extras)
	if custom != nil {
		data["custom"] = custom
	}

	person := configuration.person
	if person.id != "" {
		data["person"] = map[string]string{
			"id":       person.id,
			"username": person.username,
			"email":    person.email,
		}
	}

	return map[string]interface{}{
		"access_token": configuration.token,
		"data":         data,
	}
}

func buildCustom(custom map[string]interface{}, extras map[string]interface{}) map[string]interface{} {
	if custom == nil && extras == nil {
		return nil
	}
	m := map[string]interface{}{}
	for k, v := range custom {
		m[k] = v
	}
	for k, v := range extras {
		m[k] = v
	}
	return m
}

func addErrorToBody(configuration configuration, body map[string]interface{}, err error, skip int) map[string]interface{} {
	data := body["data"].(map[string]interface{})
	errBody, fingerprint := errorBody(configuration, err, skip)
	data["body"] = errBody
	if configuration.fingerprint {
		data["fingerprint"] = fingerprint
	}
	return data
}

func requestDetails(configuration configuration, r *http.Request) map[string]interface{} {
	cleanQuery := filterParams(configuration.scrubFields, r.URL.Query())

	return map[string]interface{}{
		"url":     r.URL.String(),
		"method":  r.Method,
		"headers": flattenValues(filterParams(configuration.scrubHeaders, r.Header)),

		// GET params
		"query_string": url.Values(cleanQuery).Encode(),
		"GET":          flattenValues(cleanQuery),

		// POST / PUT params
		"POST":    flattenValues(filterParams(configuration.scrubFields, r.Form)),
		"user_ip": filterIp(r.RemoteAddr, configuration.captureIp),
	}
}

// filterParams filters sensitive information like passwords from being sent to
// Rollbar.
func filterParams(pattern *regexp.Regexp, values map[string][]string) map[string][]string {
	for key := range values {
		if pattern.Match([]byte(key)) {
			values[key] = []string{FILTERED}
		}
	}

	return values
}

// flattenValues takes a map from strings to lists of strings and performs a lift
// on values which have length 1.
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

// filterIp takes an ip address string and a capture policy and returns a possibly
// transformed ip address string.
func filterIp(ip string, captureIp captureIp) string {
	switch captureIp {
	case CaptureIpFull:
		return ip
	case CaptureIpAnonymize:
		if strings.Contains(ip, ".") {
			parts := strings.Split(ip, ".")
			parts[len(parts)-1] = "0"
			return strings.Join(parts, ".")
		}
		if strings.Contains(ip, ":") {
			parts := strings.Split(ip, ":")
			if len(parts) > 2 {
				parts = parts[0:3]
				parts = append(parts, "0000:0000:0000:0000:0000")
				return strings.Join(parts, ":")
			}
			return ip
		}
		return ip
	case CaptureIpNone:
		return ""
	default:
		return ""
	}
}

// Build an error inner-body for the given error. If skip is provided, that
// number of stack trace frames will be skipped. If the error has a Cause
// method, the causes will be traversed until nil.
func errorBody(configuration configuration, err error, skip int) (map[string]interface{}, string) {
	var parent error
	traceChain := []map[string]interface{}{}
	fingerprint := ""
	for {
		stack := getOrBuildStack(err, parent, skip)
		traceChain = append(traceChain, buildTrace(err, stack))
		if configuration.fingerprint {
			fingerprint = fingerprint + stack.Fingerprint()
		}
		parent = err
		err = getCause(err)
		if err == nil {
			break
		}
	}
	errBody := map[string]interface{}{"trace_chain": traceChain}
	return errBody, fingerprint
}

// builds one trace element in trace_chain
func buildTrace(err error, stack Stack) map[string]interface{} {
	message := nilErrTitle
	if err != nil {
		message = err.Error()
	}
	return map[string]interface{}{
		"frames": stack,
		"exception": map[string]interface{}{
			"class":   errorClass(err),
			"message": message,
		},
	}
}

func getCause(err error) error {
	if cs, ok := err.(CauseStacker); ok {
		return cs.Cause()
	}
	return nil
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

// Build a message inner-body for the given message string.
func messageBody(s string) map[string]interface{} {
	return map[string]interface{}{
		"message": map[string]interface{}{
			"body": s,
		},
	}
}

func errorClass(err error) string {
	if err == nil {
		return nilErrTitle
	}

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
