package rollbar

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
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
		// generic error
		"errors.errorString": fmt.Errorf("something is broken"),
		// custom error
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
	SetPerson("1", "user", "email")
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

func TestSetContext(t *testing.T) {
	SetToken(os.Getenv("TOKEN"))
	SetEnvironment("test")
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	SetContext(ctx)
	if std.ctx != ctx {
		t.Error("Client ctx must be set")
	}
	tr := std.Transport.(*AsyncTransport)
	if tr.getContext() != ctx {
		t.Error("Transport ctx must be set")
	}
}

func TestEverythingGeneric(t *testing.T) {
	SetToken(os.Getenv("TOKEN"))
	SetEnvironment("test")
	SetPerson("1", "user", "email", WithPersonExtra(map[string]string{"v": "k"}))
	SetCaptureIp(CaptureIpAnonymize)
	if Token() != os.Getenv("TOKEN") {
		t.Error("Token should be as set")
	}
	if Environment() != "test" {
		t.Error("Token should be as set")
	}

	if std.ctx != context.Background() {
		t.Error("Client ctx must be set")
	}
	tr := std.Transport.(*AsyncTransport)
	if tr.getContext() != context.Background() {
		t.Error("Transport ctx must be set")
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

	SetItemsPerMinute(2000)
	SetRetryAttempts(123)
	SetLogger(&SilentClientLogger{})
	Info(someNonstandardTypeForLogFailing{}, "I am a string and I did not fail")
	SetLogger(nil)

	r, _ := http.NewRequest("GET", "http://foo.com/somethere?param1=true", nil)
	r.RemoteAddr = "1.1.1.1:123"

	Debug(r, "This is a message with a generic request")
	Warning(errors.New("Some generic error with a request"), r, map[string]interface{}{
		"hello": "request",
	})

	Close()
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
	SetCaptureIp(CaptureIpFull)
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
	if object["user_ip"] != "1.1.1.1" {
		t.Errorf("wrong user_ip, got %v", object["user_ip"])
	}
}

func TestRequestForwardedIP(t *testing.T) {
	SetCaptureIp(CaptureIpFull)
	r, _ := http.NewRequest("GET", "http://foo.com/somethere?param1=true", nil)
	r.RemoteAddr = "1.1.1.1:123"
	r.Header.Add("X-Forwarded-For", "1.2.3.4, 2.3.4.5, 3.4.5.6")

	object := std.requestDetails(r)

	if object["user_ip"] != "1.2.3.4" {
		t.Errorf("wrong user_ip, got %v", object["user_ip"])
	}
}

func TestRequestMutlipleIPHeaders(t *testing.T) {
	SetCaptureIp(CaptureIpFull)
	r, _ := http.NewRequest("GET", "http://foo.com/somethere?param1=true", nil)
	r.RemoteAddr = "1.1.1.1:123"
	r.Header.Add("X-Real-Ip", "8.9.10.11")
	r.Header.Add("X-Forwarded-For", "1.2.3.4, 2.3.4.5, 3.4.5.6")

	object := std.requestDetails(r)

	if object["user_ip"] != "8.9.10.11" {
		t.Errorf("wrong user_ip, got %v", object["user_ip"])
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
	stack []runtime.Frame
}

var _ Stacker = cs{}
var _ CauseStacker = cs{}

func (cs cs) Cause() error {
	return cs.cause
}

func (cs cs) Stack() []runtime.Frame {
	return cs.stack
}

type uw struct {
	error
	wrapped error
}

func (uw uw) Unwrap() error {
	return uw.wrapped
}

func TestDefaultUnwrapper(t *testing.T) {
	t.Run("standard error", func(t *testing.T) {
		if nil != DefaultUnwrapper(fmt.Errorf("")) {
			t.Error("unwrapping a standard error should get nil")
		}
	})
	t.Run("unwrap", func(t *testing.T) {
		wrapped := fmt.Errorf("wrapped")
		parent := uw{fmt.Errorf("parent"), wrapped}
		if wrapped != DefaultUnwrapper(parent) {
			t.Error("parent should return wrapped")
		}
	})
	t.Run("CauseStacker", func(t *testing.T) {
		cause := fmt.Errorf("cause")
		effect := cs{fmt.Errorf("effect"), cause, nil}
		if cause != DefaultUnwrapper(effect) {
			t.Error("effect should return cause")
		}
	})
}

func TestDefaultStackTracer(t *testing.T) {
	t.Run("standard error", func(t *testing.T) {
		trace, ok := DefaultStackTracer(fmt.Errorf("standard error"))
		if trace != nil {
			t.Error("standard errors should not return a trace")
		}
		if ok {
			t.Errorf("standard errors should not be handled")
		}
	})
	t.Run("Stacker", func(t *testing.T) {
		trace := getCallersFrames(0)
		err := cs{fmt.Errorf("cause"), nil, trace}
		extractedTrace, ok := DefaultStackTracer(err)
		if extractedTrace == nil {
			t.Error("Stackers should return a trace")
		} else if extractedTrace[0] != trace[0] {
			t.Error("the trace from the error must be extracted")
		}
		if !ok {
			t.Error("Stackers should be handled")
		}
	})
}

func TestGetOrBuildFrames(t *testing.T) {
	// These tests all use the default stack tracer. The logic this is testing doesn't really
	// depend on how the stack trace is extracted.

	t.Run("standard error without parent", func(t *testing.T) {
		err := fmt.Errorf("")
		trace := getOrBuildFrames(err, nil, 0, DefaultStackTracer)
		if nil == trace {
			t.Error("should build a new stack trace if error has no stack and parent is nil")
		}
	})
	t.Run("standard error with traceable parent", func(t *testing.T) {
		cause := fmt.Errorf("cause")
		effect := cs{fmt.Errorf("effect"), cause, getCallersFrames(0)}
		if nil != getOrBuildFrames(cause, effect, 0, DefaultStackTracer) {
			t.Error("should return nil if child is not traceable but parent is")
		}
	})
	t.Run("standard error with non-traceable parent", func(t *testing.T) {
		child := fmt.Errorf("child")
		parent := uw{fmt.Errorf("parent"), child}
		trace := getOrBuildFrames(child, parent, 0, DefaultStackTracer)
		if nil == trace {
			t.Error("should build a new stack trace if parent is not traceable")
		}
	})
	t.Run("traceable error without parent", func(t *testing.T) {
		cause := fmt.Errorf("cause")
		effect := cs{fmt.Errorf("effect"), cause, getCallersFrames(0)}
		if effect.Stack()[0] != getOrBuildFrames(effect, nil, 0, DefaultStackTracer)[0] {
			t.Error("should use stack trace from effect")
		}
	})
	t.Run("traceable error with traceable parent", func(t *testing.T) {
		cause := fmt.Errorf("cause")
		effect := cs{fmt.Errorf("effect"), cause, getCallersFrames(0)}
		effect2 := cs{fmt.Errorf("effect2"), effect, getCallersFrames(0)}
		if effect.Stack()[0] != getOrBuildFrames(effect, effect2, 0, DefaultStackTracer)[0] {
			t.Error("should use stack from child, not parent")
		}
	})
	t.Run("traceable error with non-traceable parent", func(t *testing.T) {
		cause := fmt.Errorf("cause")
		effect := cs{fmt.Errorf("effect"), cause, getCallersFrames(0)}
		effect2 := uw{fmt.Errorf("effect2"), effect}
		if effect.Stack()[0] != getOrBuildFrames(effect, effect2, 0, DefaultStackTracer)[0] {
			t.Error("should use stack from child")
		}
	})
}

func TestErrorBodyWithoutChain(t *testing.T) {
	err := fmt.Errorf("ERR")
	errorBody, fingerprint := errorBody(configuration{
		fingerprint: true,
		unwrapper:   DefaultUnwrapper,
		stackTracer: DefaultStackTracer,
	}, err, 0)
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
	effect := cs{fmt.Errorf("effect1"), cause, getCallersFrames(0)}
	effect2 := cs{fmt.Errorf("effect2"), effect, getCallersFrames(0)}
	errorBody, fingerprint := errorBody(configuration{
		fingerprint: true,
		unwrapper:   DefaultUnwrapper,
		stackTracer: DefaultStackTracer,
	}, effect2, 0)
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

	if buildStack(effect2.Stack()).Fingerprint()+buildStack(effect.Stack()).Fingerprint()+"0" != fingerprint {
		t.Error("fingerprint should be the fingerprints in chain concatenated together. got: ", fingerprint)
	}
}

func TestSetUnwrapper(t *testing.T) {
	type myCustomError struct {
		error
		wrapped error
	}

	client := NewAsync("example", "test", "0.0.0", "", "")
	child := fmt.Errorf("child")
	parent := myCustomError{fmt.Errorf("parent"), child}

	if client.configuration.unwrapper(parent) != nil {
		t.Fatal("bad test; default unwrapper must not recognize the custom error type")
	}

	client.SetUnwrapper(func(err error) error {
		if e, ok := err.(myCustomError); ok {
			return e.wrapped
		}

		return nil
	})

	if client.configuration.unwrapper(parent) != child {
		t.Error("error did not unwrap correctly")
	}
}

func TestSetStackTracer(t *testing.T) {
	type myCustomError struct {
		error
		trace []runtime.Frame
	}

	client := NewAsync("example", "test", "0.0.0", "", "")
	err := myCustomError{fmt.Errorf("some error"), getCallersFrames(0)}

	if trace, ok := client.configuration.stackTracer(err); ok || trace != nil {
		t.Fatal("bad test; default stack tracer must not recognize the custom error type")
	}

	client.SetStackTracer(func(err error) (frames []runtime.Frame, b bool) {
		if e, ok := err.(myCustomError); ok {
			return e.trace, true
		}

		return nil, false
	})

	trace, ok := client.configuration.stackTracer(err)
	if !ok {
		t.Error("error was not handled by custom stack tracer")
	}
	if trace == nil {
		t.Errorf("custom tracer failed to extract trace")
	} else if trace[0] != err.trace[0] {
		t.Errorf("custom tracer got the wrong trace")
	}
}

type roundTripFunc func(r *http.Request) (*http.Response, error)

func (s roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return s(r)
}

func TestSetHttpClient(t *testing.T) {
	used := false
	c := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			used = true
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader("")),
			}, nil
		}),
	}

	client := NewAsync("example", "test", "0.0.0", "", "")
	client.SetHTTPClient(c)

	err := client.Transport.Send(map[string]interface{}{})
	client.Wait()
	if err != nil {
		t.Fatal("failed to send body:", err.Error())
	}

	if !used {
		t.Fatal("custom http client had not been invoked")
	}

	used = false
	client = NewSync("example", "test", "0.0.0", "", "")
	client.SetHTTPClient(c)

	if err := client.Transport.Send(map[string]interface{}{}); err != nil {
		t.Fatal("failed to send body:", err.Error())
	}

	if !used {
		t.Fatal("custom http client had not been invoked")
	}
}
