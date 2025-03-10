package rollbar

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

type TestTransport struct {
	Body       map[string]interface{}
	WaitCalled bool
}

func (t *TestTransport) Close() error {
	return nil
}

func (t *TestTransport) Wait() {
	t.WaitCalled = true
}
func (t *TestTransport) SetContext(ctx context.Context) {
}

func (t *TestTransport) SetToken(_t string)             {}
func (t *TestTransport) SetEndpoint(_e string)          {}
func (t *TestTransport) SetLogger(_l ClientLogger)      {}
func (t *TestTransport) SetRetryAttempts(_r int)        {}
func (t *TestTransport) SetPrintPayloadOnError(_p bool) {}
func (t *TestTransport) SetHTTPClient(_c *http.Client)  {}
func (t *TestTransport) SetItemsPerMinute(_r int)       {}
func (t *TestTransport) Send(body map[string]interface{}) error {
	t.Body = body
	return nil
}

func testClient() *Client {
	c := New("", "test", "", "", "")
	c.Transport = &TestTransport{}
	return c
}

func TestLogPanic(t *testing.T) {
	client := testClient()
	client.LogPanic(errors.New("logged error"), false)
	if transport, ok := client.Transport.(*TestTransport); ok {
		if transport.WaitCalled {
			t.Error("Wait called unexpectedly")
		}
		body := transport.Body
		if body["data"] == nil {
			t.Error("body should have data")
		}
		data := body["data"].(map[string]interface{})
		dataError := errorFromData(data)
		if dataError["message"] != "logged error" {
			t.Error("data should have correct error message")
		}
	} else {
		t.Fail()
	}
	client.Close()
}

func TestWrap(t *testing.T) {
	client := testClient()
	err := errors.New("bork")
	result := client.Wrap(func() {
		panic(err)
	})
	if err != result {
		t.Error("Got:", result, "Expected:", err)
	}
	if transport, ok := client.Transport.(*TestTransport); ok {
		if transport.WaitCalled {
			t.Error("Wait called unexpectedly")
		}
	} else {
		t.Fail()
	}
	client.Close()
}

func TestWrapWithArgs(t *testing.T) {
	client := testClient()
	result := client.Wrap(func(foo string, num int) (string, int) {
		panic(fmt.Errorf("%v-%v", foo, num))
	}, "foo", 42)
	if fmt.Sprintf("%T", result) != "*errors.errorString" {
		t.Error("Return value should be error type")
	}
	if transport, ok := client.Transport.(*TestTransport); ok {
		if transport.WaitCalled {
			t.Error("Wait called unexpectedly")
		}
		body := transport.Body
		if body["data"] == nil {
			t.Error("body should have data")
		}
		data := body["data"].(map[string]interface{})
		dataError := errorFromData(data)
		if dataError["message"] != "foo-42" {
			t.Error("data should have correct error message")
		}
	} else {
		t.Fail()
	}
	client.Close()
}

func TestWrapNonError(t *testing.T) {
	client := testClient()
	err := "hello rollbar"
	result := client.Wrap(func() {
		panic(err)
	})
	if err != result {
		t.Error("Got:", result, "Expected:", err)
	}
}

func TestWrapNoPanic(t *testing.T) {
	client := testClient()
	result := client.Wrap(func() {})
	if result != nil {
		t.Error("Got:", result, "Expected:", nil)
	}
}

func TestWrapIgnore(t *testing.T) {
	client := testClient()
	err := errors.New("bork 42")
	client.SetCheckIgnore(func(msg string) bool {
		if msg == "bork 42" {
			return true
		}
		return false
	})
	result := client.Wrap(func() {
		panic(err)
	})
	if err != result {
		t.Error("Got:", result, "Expected:", err)
	}
	client.Wait()
	if transport, ok := client.Transport.(*TestTransport); ok {
		if transport.Body != nil {
			t.Error("Expected Body to be nil, got:", transport.Body)
		}
	} else {
		t.Fail()
	}
}

func TestWrapNonErrorIgnore(t *testing.T) {
	client := testClient()
	err := "borkXXX"
	client.SetCheckIgnore(func(msg string) bool {
		if msg == "borkXXX" {
			return true
		}
		return false
	})
	result := client.Wrap(func() {
		panic(err)
	})
	if err != result {
		t.Error("Got:", result, "Expected:", err)
	}
	client.Wait()
	if transport, ok := client.Transport.(*TestTransport); ok {
		if transport.Body != nil {
			t.Error("Expected Body to be nil, got:", transport.Body)
		}
	} else {
		t.Fail()
	}
}

func TestWrapAndWait(t *testing.T) {
	client := testClient()
	err := errors.New("bork")
	result := client.WrapAndWait(func() {
		panic(err)
	})
	if err != result {
		t.Error("Got:", result, "Expected:", err)
	}
	if transport, ok := client.Transport.(*TestTransport); ok {
		if !transport.WaitCalled {
			t.Error("Expected wait to be called")
		}
	} else {
		t.Fail()
	}
}

func TestWrapAndWaitNonError(t *testing.T) {
	client := testClient()
	err := "hello rollbar"
	result := client.WrapAndWait(func() {
		panic(err)
	})
	if err != result {
		t.Error("Got:", result, "Expected:", err)
	}
}

func TestWrapAndWaitNoPanic(t *testing.T) {
	client := testClient()
	result := client.WrapAndWait(func() {})
	if result != nil {
		t.Error("Got:", result, "Expected:", nil)
	}
}

func TestWrapAndWaitIgnore(t *testing.T) {
	client := testClient()
	err := errors.New("bork 42")
	client.SetCheckIgnore(func(msg string) bool {
		if msg == "bork 42" {
			return true
		}
		return false
	})
	result := client.WrapAndWait(func() {
		panic(err)
	})
	if err != result {
		t.Error("Got:", result, "Expected:", err)
	}
	if transport, ok := client.Transport.(*TestTransport); ok {
		if transport.Body != nil {
			t.Error("Expected Body to be nil, got:", transport.Body)
		}
	} else {
		t.Fail()
	}
}

func TestWrapAndWaitNonErrorIgnore(t *testing.T) {
	client := testClient()
	err := "borkXXX"
	client.SetCheckIgnore(func(msg string) bool {
		if msg == "borkXXX" {
			return true
		}
		return false
	})
	result := client.WrapAndWait(func() {
		panic(err)
	})
	if err != result {
		t.Error("Got:", result, "Expected:", err)
	}
	if transport, ok := client.Transport.(*TestTransport); ok {
		if transport.Body != nil {
			t.Error("Expected Body to be nil, got:", transport.Body)
		}
	} else {
		t.Fail()
	}
}

func testCallLambdaHandler(handler interface{}) interface{} {
	fn := reflect.ValueOf(handler)
	var args []reflect.Value
	return fn.Call(args)
}

func testLambdaHandlerWithContext(ctx context.Context) (context.Context, error) {
	return ctx, errors.New("test")
}

func testLambdaHandlerWithMessage(message TestMessage) (TestMessage, error) {
	return message, errors.New("test")
}

type TestMessage struct {
	Name string
}

func TestLambdaWrapperWithError(t *testing.T) {
	client := testClient()
	err := errors.New("bork")

	defer func() {
		recoveredError := recover()

		if recoveredError != err {
			if recoveredError == nil {
				t.Error("Expected wrapper to bubble up the custom panic error")
			} else {
				t.Errorf("Unexpected panic %s", recoveredError)
			}
		}

		if transport, ok := client.Transport.(*TestTransport); ok {
			if transport.Body == nil {
				t.Error("Expected Body to be present")
			}
			if !transport.WaitCalled {
				t.Error("Expected wait to be called")
			}
		} else {
			t.Fail()
		}
	}()

	//ctx := context.TODO()
	handler := client.LambdaWrapper(func() {
		panic(err)
	})
	fn := reflect.ValueOf(handler)
	var args []reflect.Value
	fn.Call(args)
	//testCallLambdaHandler(handler)
}

func TestLambdaWrapperWithErrorAndMultipleReturnValues(t *testing.T) {
	client := testClient()
	err := errors.New("bork")

	defer func() {
		recoveredError := recover()

		if recoveredError != err {
			if recoveredError == nil {
				t.Error("Expected wrapper to bubble up the custom panic error")
			} else {
				t.Errorf("Unexpected panic %s", recoveredError)
			}
		}

		if transport, ok := client.Transport.(*TestTransport); ok {
			if transport.Body == nil {
				t.Error("Expected Body to be present")
			}
			if !transport.WaitCalled {
				t.Error("Expected wait to be called")
			}
		} else {
			t.Fail()
		}
	}()

	handler := client.LambdaWrapper(func() (string, error) {
		panic(err)
	})
	fn := reflect.ValueOf(handler)
	var args []reflect.Value
	fn.Call(args)
}

func TestLambdaWrapperWithContext(t *testing.T) {
	client := testClient()
	ctx := context.TODO()
	handler := client.LambdaWrapper(testLambdaHandlerWithContext)
	var args []reflect.Value
	args = append(args, reflect.ValueOf(ctx))
	resp := reflect.ValueOf(handler).Call(args)
	var outCtx context.Context
	outCtx = resp[0].Interface().(context.Context)
	var err error
	err = resp[1].Interface().(error)

	if outCtx != ctx {
		t.Error("Expected ctx to be present")
	}
	if err.Error() != "test" {
		t.Error("Expected error to be present")
	}
}

func TestLambdaWrapperWithMessage(t *testing.T) {
	client := testClient()
	message := TestMessage{Name: "foo"}
	handler := client.LambdaWrapper(testLambdaHandlerWithMessage)
	var args []reflect.Value
	args = append(args, reflect.ValueOf(message))
	resp := reflect.ValueOf(handler).Call(args)
	var outMessage TestMessage
	outMessage = resp[0].Interface().(TestMessage)
	var err error
	err = resp[1].Interface().(error)

	if outMessage != message {
		t.Error("Expected message to be present")
	}
	if err.Error() != "test" {
		t.Error("Expected error to be present")
	}
}

func TestGettersAndSetters_Default(t *testing.T) {
	c := testClient()
	c.Transport = &TestTransport{}
	testGettersAndSetters(c, t)
}

func TestGettersAndSetters_Async(t *testing.T) {
	c := NewAsync("", "", "", "", "")
	c.Transport = &TestTransport{}
	testGettersAndSetters(c, t)
}

func TestGettersAndSetters_Sync(t *testing.T) {
	c := NewSync("", "", "", "", "")
	c.Transport = &TestTransport{}
	testGettersAndSetters(c, t)
}

func testGettersAndSetters(client *Client, t *testing.T) {
	token := "abc123"
	environment := "TestEnvironment"
	endpoint := "SomeEndpoint"
	platform := "ThePlatform"
	codeVersion := "CodeVersion"
	host := "SomeHost"
	root := "////"
	fingerprint := true
	scrubHeaders := regexp.MustCompile("Foo")
	scrubFields := regexp.MustCompile("squirrel|doggo")
	captureIP := CaptureIpNone
	itemsPerMinute := 10

	errorIfEqual(token, client.Token(), t)
	errorIfEqual(environment, client.Environment(), t)
	errorIfEqual(endpoint, client.Endpoint(), t)
	errorIfEqual(platform, client.Platform(), t)
	errorIfEqual(codeVersion, client.CodeVersion(), t)
	errorIfEqual(host, client.ServerHost(), t)
	errorIfEqual(root, client.ServerRoot(), t)
	errorIfEqual(fingerprint, client.Fingerprint(), t)
	errorIfEqual(captureIP, client.CaptureIp(), t)
	errorIfEqual(scrubHeaders, client.ScrubHeaders(), t)
	errorIfEqual(scrubHeaders, client.Telemetry.Network.ScrubHeaders, t)
	errorIfEqual(scrubFields, client.ScrubFields(), t)
	errorIfEqual(itemsPerMinute, client.ItemsPerMinute(), t)

	if client.Fingerprint() {
		t.Error("expected fingerprint to default to false")
	}

	if client.ScrubHeaders().MatchString("Foo") {
		t.Error("unexpected matching scrub header")
	}

	if client.ScrubFields().MatchString("squirrel") {
		t.Error("unexpected matching scrub field")
	}

	client.SetEnabled(false)

	client.SetToken(token)
	client.SetEnvironment(environment)
	client.SetEndpoint(endpoint)
	client.SetPlatform(platform)
	client.SetCodeVersion(codeVersion)
	client.SetServerHost(host)
	client.SetServerRoot(root)
	client.SetFingerprint(fingerprint)
	client.SetLogger(&SilentClientLogger{})
	client.SetScrubHeaders(scrubHeaders)
	client.SetScrubFields(scrubFields)
	client.SetCaptureIp(captureIP)
	client.SetTelemetry()

	client.SetEnabled(true)
	client.SetItemsPerMinute(itemsPerMinute)

	errorIfNotEqual(token, client.Token(), t)
	errorIfNotEqual(environment, client.Environment(), t)
	errorIfNotEqual(endpoint, client.Endpoint(), t)
	errorIfNotEqual(platform, client.Platform(), t)
	errorIfNotEqual(codeVersion, client.CodeVersion(), t)
	errorIfNotEqual(host, client.ServerHost(), t)
	errorIfNotEqual(root, client.ServerRoot(), t)
	errorIfNotEqual(fingerprint, client.Fingerprint(), t)
	errorIfNotEqual(captureIP, client.CaptureIp(), t)
	errorIfNotEqual(scrubHeaders, client.ScrubHeaders(), t)
	errorIfNotEqual(scrubHeaders, client.Telemetry.Network.ScrubHeaders, t)
	errorIfNotEqual(scrubFields, client.ScrubFields(), t)
	errorIfNotEqual(itemsPerMinute, client.ItemsPerMinute(), t)

	if !client.Fingerprint() {
		t.Error("expected fingerprint to default to false")
	}

	if !client.ScrubHeaders().MatchString("Foo") {
		t.Error("expected matching scrub header")
	}

	if !client.ScrubFields().MatchString("squirrel") {
		t.Error("expected matching scrub field")
	}

	client.ErrorWithLevel(ERR, errors.New("Bork"))

	if transport, ok := client.Transport.(*TestTransport); ok {
		body := transport.Body
		if body["data"] == nil {
			t.Error("body should have data")
		}
		data := body["data"].(map[string]interface{})
		configuredOptions := configuredOptionsFromData(data)

		errorIfNotEqual(environment, configuredOptions["environment"].(string), t)
		errorIfNotEqual(endpoint, configuredOptions["endpoint"].(string), t)
		errorIfNotEqual(platform, configuredOptions["platform"].(string), t)
		errorIfNotEqual(codeVersion, configuredOptions["codeVersion"].(string), t)
		errorIfNotEqual(host, configuredOptions["serverHost"].(string), t)
		errorIfNotEqual(root, configuredOptions["serverRoot"].(string), t)
		errorIfNotEqual(fingerprint, configuredOptions["fingerprint"].(bool), t)
		errorIfNotEqual(scrubHeaders, configuredOptions["scrubHeaders"].(*regexp.Regexp), t)
		errorIfNotEqual(scrubFields, configuredOptions["scrubFields"].(*regexp.Regexp), t)
		errorIfNotEqual(itemsPerMinute, configuredOptions["itemsPerMinute"].(int), t)
	} else {
		t.Fail()
	}
}

func errorIfEqual(a, b interface{}, t *testing.T) {
	if a == b {
		t.Error("Expected", a, " != ", b)
	}
}

func errorIfNotEqual(a, b interface{}, t *testing.T) {
	if a != b {
		t.Error("Expected", a, " == ", b)
	}
}

func TestSetPerson(t *testing.T) {
	client := testClient()
	id, username, email := "42", "bork", "bork@foobar.com"

	client.SetPerson(id, username, email, WithPersonExtra(map[string]string{
		"person_extra1": "value1", "person_extra2": "value2", "id": "43"}))

	client.ErrorWithLevel(ERR, errors.New("Person Bork"))

	if transport, ok := client.Transport.(*TestTransport); ok {
		body := transport.Body
		if body["data"] == nil {
			t.Error("body should have data")
		}
		data := body["data"].(map[string]interface{})
		if data["person"] == nil {
			t.Error("data should have person")
		}
		person := data["person"].(map[string]string)
		errorIfNotEqual(id, person["id"], t)
		errorIfNotEqual(username, person["username"], t)
		errorIfNotEqual(email, person["email"], t)
		errorIfNotEqual("value1", person["person_extra1"], t)
		errorIfNotEqual("value2", person["person_extra2"], t)

		configuredOptions := configuredOptionsFromData(data)
		configuredPerson := configuredOptions["person"].(map[string]string)

		errorIfNotEqual(id, configuredPerson["Id"], t)
		errorIfNotEqual(username, configuredPerson["Username"], t)
		errorIfNotEqual(email, configuredPerson["Email"], t)
	} else {
		t.Fail()
	}
}

func TestClearPerson(t *testing.T) {
	client := testClient()
	id, username, email := "42", "bork", "bork@foobar.com"

	client.SetPerson(id, username, email)
	client.ClearPerson()
	client.ErrorWithLevel(ERR, errors.New("Person Bork"))

	if transport, ok := client.Transport.(*TestTransport); ok {
		body := transport.Body
		if body["data"] == nil {
			t.Error("body should have data")
		}
		data := body["data"].(map[string]interface{})
		if data["person"] != nil {
			t.Error("data should not have a person")
		}
	} else {
		t.Fail()
	}
}

func TestTransform(t *testing.T) {
	client := testClient()
	client.SetTransform(func(data map[string]interface{}) {
		data["some_custom_field"] = "hello_world"
	})

	client.ErrorWithLevel(ERR, errors.New("Bork"))

	if transport, ok := client.Transport.(*TestTransport); ok {
		body := transport.Body
		if body["data"] == nil {
			t.Error("body should have data")
		}
		data := body["data"].(map[string]interface{})
		if data["some_custom_field"] != "hello_world" {
			t.Error("data should have field set by transform")
		}
		configuredOptions := configuredOptionsFromData(data)
		if !strings.Contains(configuredOptions["transform"].(string), "TestTransform.func1") {
			t.Error("data should have transform in diagnostic object")
		}
	} else {
		t.Fail()
	}
}

func TestSetUnwrapperClient(t *testing.T) {
	client := testClient()
	client.SetUnwrapper(DefaultUnwrapper)

	client.ErrorWithLevel(ERR, errors.New("Bork"))

	if transport, ok := client.Transport.(*TestTransport); ok {
		body := transport.Body
		if body["data"] == nil {
			t.Error("body should have data")
		}
		data := body["data"].(map[string]interface{})
		configuredOptions := configuredOptionsFromData(data)
		if !strings.Contains(configuredOptions["unwrapper"].(string), "func1") {
			t.Error("data should have unwrapper in diagnostic object")
		}
	} else {
		t.Fail()
	}
}

func TestSetStackTracerClient(t *testing.T) {
	client := testClient()
	client.SetStackTracer(DefaultStackTracer)

	client.ErrorWithLevel(ERR, errors.New("Bork"))

	if transport, ok := client.Transport.(*TestTransport); ok {
		body := transport.Body
		if body["data"] == nil {
			t.Error("body should have data")
		}
		data := body["data"].(map[string]interface{})
		configuredOptions := configuredOptionsFromData(data)
		if !strings.Contains(configuredOptions["stackTracer"].(string), "func2") {
			t.Error("data should have stackTracer in diagnostic object")
		}
	} else {
		t.Fail()
	}
}

func TestEnabled(t *testing.T) {
	client := testClient()
	client.SetEnabled(false)

	client.ErrorWithLevel(ERR, errors.New("Bork"))

	if transport, ok := client.Transport.(*TestTransport); ok {
		body := transport.Body
		if body != nil {
			t.Error("Transport should not be called when enabled is false")
		}
	} else {
		t.Fail()
	}

	client.SetEnabled(true)
	client.ErrorWithLevel(ERR, errors.New("Bork"))

	if transport, ok := client.Transport.(*TestTransport); ok {
		body := transport.Body
		if body == nil {
			t.Error("Transport should be called when enabled is true")
		}
	} else {
		t.Fail()
	}
}

func TestCaptureTelemetryEvent(t *testing.T) {
	client := testClient()
	data := map[string]interface{}{"message": "some message"}
	client.CaptureTelemetryEvent("eventType", "eventLevel", data)
	items := client.Telemetry.GetQueueItems()
	if len(items) < 1 {
		t.Error("Queue should not be empty")
	}
	item := items[0].(map[string]interface{})
	delete(item, "timestamp_ms")
	expectedData := map[string]interface{}{"body": data, "type": "eventType", "level": "eventLevel", "source": "client"}
	eq := reflect.DeepEqual(item, expectedData)
	if !eq {
		t.Error("Maps are different")
	}
}

func configuredOptionsFromData(data map[string]interface{}) map[string]interface{} {
	notifier := data["notifier"].(map[string]interface{})
	diagnostic := notifier["diagnostic"].(map[string]interface{})
	configuredOptions := diagnostic["configuredOptions"].(map[string]interface{})
	return configuredOptions
}

func errorFromData(data map[string]interface{}) map[string]interface{} {
	body := data["body"].(map[string]interface{})
	traceChain := body["trace_chain"].([]map[string]interface{})
	return traceChain[0]["exception"].(map[string]interface{})
}
