package rollbar

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"
)

type CustomError struct {
	s string
}

func (e *CustomError) Error() string {
	return e.s
}

func testErrorStack(s string) {
	testErrorStack2(s)
}

func testErrorStack2(s string) {
	ErrorWithLevel("error", errors.New(s))
}

func testErrorStackWithSkip(s string) {
	testErrorStackWithSkip2(s)
}

func testErrorStackWithSkip2(s string) {
	ErrorWithStackSkip("error", errors.New(s), 2)
}

func testErrorStackWithSkipGeneric(s string) {
	testErrorStackWithSkipGeneric2(s)
}

func testErrorStackWithSkipGeneric2(s string) {
	Warning(errors.New(s), 2)
}

func TestErrorClass(t *testing.T) {
	errors := map[string]error{
		"{4b81076c}":          fmt.Errorf("something is broken"),
		"rollbar.CustomError": &CustomError{"terrible mistakes were made"},
	}

	for expected, err := range errors {
		if errorClass(err) != expected {
			t.Error("Got:", errorClass(err), "Expected:", expected)
		}
	}
}

func TestEverything(t *testing.T) {
	SetToken(os.Getenv("TOKEN"))
	SetEnvironment("test")
	if Token() != os.Getenv("TOKEN") {
		t.Error("Token should be as set")
	}
	if Environment() != "test" {
		t.Error("Token should be as set")
	}

	ErrorWithLevel("critical", errors.New("Normal critical error"))
	ErrorWithLevel("error", &CustomError{"This is a custom error"})

	testErrorStack("This error should have a nice stacktrace")
	testErrorStackWithSkip("This error should have a skipped stacktrace")

	done := make(chan bool)
	go func() {
		testErrorStack("I'm in a goroutine")
		done <- true
	}()
	<-done

	Message("error", "This is an error message")
	Message("info", "And this is an info message")

	SetFingerprint(true)

	Errorf("error", "%s %s", "Some argument", "Another argument")

	r, _ := http.NewRequest("GET", "http://foo.com/somethere?param1=true", nil)
	r.RemoteAddr = "1.1.1.1:123"

	RequestMessage("debug", r, "This is a debug message with a request")
	SetCaptureIp(CaptureIpAnonymize)
	RequestError("info", r, errors.New("Some info error with a request"))
	r.RemoteAddr = "FE80::0202:B3FF:FE1E:8329"
	RequestErrorWithStackSkip("info", r, errors.New("Some info error with a request"), 2)

	Wait()
}

type someNonstandardTypeForLogFailing struct{}

func TestEverythingGeneric(t *testing.T) {
	SetToken(os.Getenv("TOKEN"))
	SetEnvironment("test")
	SetCaptureIp(CaptureIpAnonymize)
	if Token() != os.Getenv("TOKEN") {
		t.Error("Token should be as set")
	}
	if Environment() != "test" {
		t.Error("Token should be as set")
	}

	Critical(errors.New("Normal generic critical error"))
	Error(&CustomError{"This is a generic custom error"})

	testErrorStackWithSkipGeneric("This generic error should have a skipped stacktrace")

	done := make(chan bool)
	go func() {
		testErrorStack("I'm in a generic goroutine")
		done <- true
	}()
	<-done

	Error("This is a generic error message")
	Info("And this is a generic info message", map[string]interface{}{
		"hello": "rollbar",
	})

	SetLogger(&SilentClientLogger{})
	Info(someNonstandardTypeForLogFailing{}, "I am a string and I did not fail")
	SetLogger(nil)

	r, _ := http.NewRequest("GET", "http://foo.com/somethere?param1=true", nil)
	r.RemoteAddr = "1.1.1.1:123"

	Debug(r, "This is a message with a generic request")
	Warning(errors.New("Some generic error with a request"), r, map[string]interface{}{
		"hello": "request",
	})

	Wait()
}

func TestBuildBody(t *testing.T) {
	// custom provided at config time
	baseCustom := map[string]interface{}{
		"BASE_CUSTOM_KEY":       "BASE_CUSTOM_VALUE",
		"OVERRIDDEN_CUSTOM_KEY": "BASE",
	}
	SetCustom(baseCustom)

	// custom provided at call site
	extraCustom := map[string]interface{}{
		"EXTRA_CUSTOM_KEY":      "EXTRA_CUSTOM_VALUE",
		"OVERRIDDEN_CUSTOM_KEY": "EXTRA",
	}
	body := interface{}(std).(*Client).buildBody(context.TODO(), ERR, "test error", extraCustom)

	if body["data"] == nil {
		t.Error("body should have data")
	}
	data := body["data"].(map[string]interface{})
	if data["custom"] == nil {
		t.Error("data should have custom")
	}
	custom := data["custom"].(map[string]interface{})
	if custom["BASE_CUSTOM_KEY"] != "BASE_CUSTOM_VALUE" {
		t.Error("custom should have base")
	}
	if custom["EXTRA_CUSTOM_KEY"] != "EXTRA_CUSTOM_VALUE" {
		t.Error("custom should have extra")
	}
	if custom["OVERRIDDEN_CUSTOM_KEY"] != "EXTRA" {
		t.Error("extra custom should overwrite base custom where keys match")
	}
	if Custom()["EXTRA_CUSTOM_KEY"] != nil {
		t.Error("adding extra modified the client custom data config")
	}
}

func TestBuildBodyNoBaseCustom(t *testing.T) {
	extraCustom := map[string]interface{}{
		"EXTRA_CUSTOM_KEY":      "EXTRA_CUSTOM_VALUE",
		"OVERRIDDEN_CUSTOM_KEY": "EXTRA",
	}
	body := interface{}(std).(*Client).buildBody(context.TODO(), ERR, "test error", extraCustom)

	if body["data"] == nil {
		t.Error("body should have data")
	}
	data := body["data"].(map[string]interface{})
	if data["custom"] == nil {
		t.Error("data should have custom")
	}
	custom := data["custom"].(map[string]interface{})
	if custom["EXTRA_CUSTOM_KEY"] != "EXTRA_CUSTOM_VALUE" {
		t.Error("custom should have extra")
	}
	if custom["OVERRIDDEN_CUSTOM_KEY"] != "EXTRA" {
		t.Error("extra custom should also work")
	}
}

func TestErrorRequest(t *testing.T) {
	r, _ := http.NewRequest("GET", "http://foo.com/somethere?param1=true", nil)
	r.RemoteAddr = "1.1.1.1:123"

	object := std.requestDetails(r)

	if object["url"] != "http://foo.com/somethere?param1=true" {
		t.Errorf("wrong url, got %v", object["url"])
	}

	if object["method"] != "GET" {
		t.Errorf("wrong method, got %v", object["method"])
	}

	if object["query_string"] != "param1=true" {
		t.Errorf("wrong id, got %v", object["query_string"])
	}
}

func TestErrorRequestHeaders(t *testing.T) {
	r, _ := http.NewRequest("GET", "http://foo.com/somethere?param1=true", nil)
	r.RemoteAddr = "1.1.1.1:123"
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Content-Type", "application/x-json")
	r.Header.Add("X-Foo-Bar", "baz")
	r.Header.Add("X-Mult", "a")
	r.Header.Add("X-Mult", "b")

	object := std.requestDetails(r)

	if object["url"] != "http://foo.com/somethere?param1=true" {
		t.Errorf("wrong url, got %v", object["url"])
	}

	if object["method"] != "GET" {
		t.Errorf("wrong method, got %v", object["method"])
	}

	if object["query_string"] != "param1=true" {
		t.Errorf("wrong id, got %v", object["query_string"])
	}

	headers := object["headers"].(map[string]interface{})
	if headers["Content-Type"].(string) != "application/json" {
		t.Errorf("expected single string value for Content-Type, got %v", headers["Content-Type"])
	}
	if headers["X-Foo-Bar"].(string) != "baz" {
		t.Errorf("expected single string value for X-Foo-Bar, got %v", headers["X-Foo-Bar"])
	}
	if len(headers["X-Mult"].([]string)) != 2 {
		t.Errorf("expected X-Mult to have two string values, got %v", headers["X-Mult"])
	}

	multHeaders := headers["X-Mult"].([]string)
	if multHeaders[0] != "a" || multHeaders[1] != "b" {
		t.Errorf("expected multiple string values for X-Mult, got %v", headers["X-Mult"])
	}
}

func TestFilterParams(t *testing.T) {
	values := map[string][]string{
		"password":     {"one"},
		"ok":           {"one"},
		"access_token": {"one"},
	}

	clean := filterParams(std.configuration.scrubFields, values)
	if clean["password"][0] != FILTERED {
		t.Error("should filter password parameter")
	}

	if clean["ok"][0] == FILTERED {
		t.Error("should keep ok parameter")
	}

	if clean["access_token"][0] != FILTERED {
		t.Error("should filter access_token parameter")
	}
}

func TestFlattenValues(t *testing.T) {
	values := map[string][]string{
		"a": {"one"},
		"b": {"one", "two"},
	}

	flattened := flattenValues(values)
	if flattened["a"].(string) != "one" {
		t.Error("should flatten single parameter to string")
	}

	if len(flattened["b"].([]string)) != 2 {
		t.Error("should leave multiple parametres as []string")
	}
}

func TestFilterFlatten(t *testing.T) {
	values := map[string][]string{
		"password":     {"one"},
		"ok":           {"one", "two"},
		"access_token": {"one", "two"},
		"thing":        {"foo", "bar"},
		"a":            {"single"},
		"b":            {"more", "than", "one"},
	}

	clean := filterFlatten(std.configuration.scrubFields, values, nil)
	if clean["password"] != FILTERED {
		t.Error("should filter password parameter")
	}

	if clean["ok"] == FILTERED {
		t.Error("should keep ok parameter")
	}

	if len(clean["ok"].([]string)) != 2 {
		t.Error("should not flatten ok parameter")
	}

	if clean["access_token"] != FILTERED {
		t.Error("should filter access_token parameter")
	}

	special := map[string]struct{}{
		"thing": struct{}{},
	}

	clean2 := filterFlatten(std.configuration.scrubFields, values, special)
	if clean2["password"] != FILTERED {
		t.Error("should filter password parameter")
	}

	if clean2["ok"] == FILTERED {
		t.Error("should keep ok parameter")
	}

	if len(clean2["ok"].([]string)) != 2 {
		t.Error("should not flatten ok parameter")
	}

	if clean2["access_token"] != FILTERED {
		t.Error("should filter access_token parameter")
	}

	if clean2["thing"] != "foo" {
		t.Error("should force flatten a special key")
	}

	if clean2["a"].(string) != "single" {
		t.Error("should flatten single parameter to string")
	}

	if len(clean2["b"].([]string)) != 3 {
		t.Error("should leave multiple parametres as []string")
	}
}

type cs struct {
	error
	cause error
	stack Stack
}

func (cs cs) Cause() error {
	return cs.cause
}

func (cs cs) Stack() Stack {
	return cs.stack
}

func TestGetCauseOfStdErr(t *testing.T) {
	if nil != getCause(fmt.Errorf("")) {
		t.Error("cause should be nil for standard error")
	}
}

func TestGetCauseOfCauseStacker(t *testing.T) {
	cause := fmt.Errorf("cause")
	effect := cs{fmt.Errorf("effect"), cause, nil}
	if cause != getCause(effect) {
		t.Error("effect should return cause")
	}
}

func TestGetOrBuildStackOfStdErrWithoutParent(t *testing.T) {
	err := cs{fmt.Errorf(""), nil, BuildStack(0)}
	if nil == getOrBuildStack(err, nil, 0) {
		t.Error("should build stack if parent is not a CauseStacker")
	}
}

func TestGetOrBuildStackOfStdErrWithParent(t *testing.T) {
	cause := fmt.Errorf("cause")
	effect := cs{fmt.Errorf("effect"), cause, BuildStack(0)}
	if 0 != len(getOrBuildStack(cause, effect, 0)) {
		t.Error("should return empty stack of stadard error if parent is CauseStacker")
	}
}

func TestGetOrBuildStackOfCauseStackerWithoutParent(t *testing.T) {
	cause := fmt.Errorf("cause")
	effect := cs{fmt.Errorf("effect"), cause, BuildStack(0)}
	if effect.Stack()[0] != getOrBuildStack(effect, nil, 0)[0] {
		t.Error("should use stack from effect")
	}
}

func TestGetOrBuildStackOfCauseStackerWithParent(t *testing.T) {
	cause := fmt.Errorf("cause")
	effect := cs{fmt.Errorf("effect"), cause, BuildStack(0)}
	effect2 := cs{fmt.Errorf("effect2"), effect, BuildStack(0)}
	if effect.Stack()[0] != getOrBuildStack(effect2, effect, 0)[0] {
		t.Error("should use stack from effect2")
	}
}

func TestErrorBodyWithoutChain(t *testing.T) {
	err := fmt.Errorf("ERR")
	errorBody, fingerprint := errorBody(configuration{fingerprint: true}, err, 0)
	if nil != errorBody["trace"] {
		t.Error("should not have trace element")
	}
	if nil == errorBody["trace_chain"] {
		t.Error("should have trace_chain element")
	}
	traces := errorBody["trace_chain"].([]map[string]interface{})
	if 1 != len(traces) {
		t.Error("chain should contain 1 trace")
	}
	if "ERR" != traces[0]["exception"].(map[string]interface{})["message"] {
		t.Error("chain should contain err")
	}
	if "0" == fingerprint {
		t.Error("fingerprint should be auto-generated and non-zero. got: ", fingerprint)
	}
}

func TestErrorBodyWithChain(t *testing.T) {
	cause := fmt.Errorf("cause")
	effect := cs{fmt.Errorf("effect1"), cause, BuildStack(0)}
	effect2 := cs{fmt.Errorf("effect2"), effect, BuildStack(0)}
	errorBody, fingerprint := errorBody(configuration{fingerprint: true}, effect2, 0)
	if nil != errorBody["trace"] {
		t.Error("should not have trace element")
	}
	if nil == errorBody["trace_chain"] {
		t.Error("should have trace_chain element")
	}
	traces := errorBody["trace_chain"].([]map[string]interface{})
	if 3 != len(traces) {
		t.Error("chain should contain 3 traces")
	}
	if "effect2" != traces[0]["exception"].(map[string]interface{})["message"] {
		t.Error("chain should contain effect2 first")
	}
	if "effect1" != traces[1]["exception"].(map[string]interface{})["message"] {
		t.Error("chain should contain effect1 second")
	}
	if "cause" != traces[2]["exception"].(map[string]interface{})["message"] {
		t.Error("chain should contain cause third")
	}
	if effect2.Stack().Fingerprint()+effect.Stack().Fingerprint()+"0" != fingerprint {
		t.Error("fingerprint should be the fingerprints in chain concatenated together. got: ", fingerprint)
	}
}
