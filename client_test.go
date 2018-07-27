package rollbar_test

import (
	"errors"
	"github.com/rollbar/rollbar-go"
	"regexp"
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

func (t *TestTransport) SetToken(_t string)                {}
func (t *TestTransport) SetEndpoint(_e string)             {}
func (t *TestTransport) SetLogger(_l rollbar.ClientLogger) {}
func (t *TestTransport) SetRetryAttempts(_r int)           {}
func (t *TestTransport) SetPrintPayloadOnError(_p bool)    {}
func (t *TestTransport) Send(body map[string]interface{}) error {
	t.Body = body
	return nil
}

func testClient() *rollbar.Client {
	c := rollbar.New("", "test", "", "", "")
	c.Transport = &TestTransport{}
	return c
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

func TestGettersAndSetters_Default(t *testing.T) {
	c := testClient()
	testGettersAndSetters(c, t)
}

func TestGettersAndSetters_Async(t *testing.T) {
	c := rollbar.NewAsync("", "", "", "", "")
	testGettersAndSetters(c, t)
}

func TestGettersAndSetters_Sync(t *testing.T) {
	c := rollbar.NewSync("", "", "", "", "")
	testGettersAndSetters(c, t)
}

func testGettersAndSetters(client *rollbar.Client, t *testing.T) {
	token := "abc123"
	environment := "TestEnvironment"
	endpoint := "SomeEndpoint"
	platform := "ThePlatform"
	codeVersion := "CodeVersion"
	host := "SomeHost"
	root := "////"
	scrubHeaders := regexp.MustCompile("Foo")
	scrubFields := regexp.MustCompile("squirrel|doggo")

	errorIfEqual(token, client.Token(), t)
	errorIfEqual(environment, client.Environment(), t)
	errorIfEqual(endpoint, client.Endpoint(), t)
	errorIfEqual(platform, client.Platform(), t)
	errorIfEqual(codeVersion, client.CodeVersion(), t)
	errorIfEqual(host, client.ServerHost(), t)
	errorIfEqual(root, client.ServerRoot(), t)

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
	client.SetFingerprint(true)
	client.SetLogger(&rollbar.SilentClientLogger{})
	client.SetScrubHeaders(scrubHeaders)
	client.SetScrubFields(scrubFields)

	client.SetEnabled(true)

	errorIfNotEqual(token, client.Token(), t)
	errorIfNotEqual(environment, client.Environment(), t)
	errorIfNotEqual(endpoint, client.Endpoint(), t)
	errorIfNotEqual(platform, client.Platform(), t)
	errorIfNotEqual(codeVersion, client.CodeVersion(), t)
	errorIfNotEqual(host, client.ServerHost(), t)
	errorIfNotEqual(root, client.ServerRoot(), t)

	if !client.Fingerprint() {
		t.Error("expected fingerprint to default to false")
	}

	if !client.ScrubHeaders().MatchString("Foo") {
		t.Error("expected matching scrub header")
	}

	if !client.ScrubFields().MatchString("squirrel") {
		t.Error("expected matching scrub field")
	}
}

func errorIfEqual(a, b string, t *testing.T) {
	if a == b {
		t.Error("Expected", a, " != ", b)
	}
}

func errorIfNotEqual(a, b string, t *testing.T) {
	if a != b {
		t.Error("Expected", a, " == ", b)
	}
}

func TestSetPerson(t *testing.T) {
	client := testClient()
	id, username, email := "42", "bork", "bork@foobar.com"

	client.SetPerson(id, username, email)
	client.ErrorWithLevel(rollbar.ERR, errors.New("Person Bork"))

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
	} else {
		t.Fail()
	}
}

func TestClearPerson(t *testing.T) {
	client := testClient()
	id, username, email := "42", "bork", "bork@foobar.com"

	client.SetPerson(id, username, email)
	client.ClearPerson()
	client.ErrorWithLevel(rollbar.ERR, errors.New("Person Bork"))

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

	client.ErrorWithLevel(rollbar.ERR, errors.New("Bork"))

	if transport, ok := client.Transport.(*TestTransport); ok {
		body := transport.Body
		if body["data"] == nil {
			t.Error("body should have data")
		}
		data := body["data"].(map[string]interface{})
		if data["some_custom_field"] != "hello_world" {
			t.Error("data should have field set by transform")
		}
	} else {
		t.Fail()
	}
}

func TestEnabled(t *testing.T) {
	client := testClient()
	client.SetEnabled(false)

	client.ErrorWithLevel(rollbar.ERR, errors.New("Bork"))

	if transport, ok := client.Transport.(*TestTransport); ok {
		body := transport.Body
		if body != nil {
			t.Error("Transport should not be called when enabled is false")
		}
	} else {
		t.Fail()
	}

	client.SetEnabled(true)
	client.ErrorWithLevel(rollbar.ERR, errors.New("Bork"))

	if transport, ok := client.Transport.(*TestTransport); ok {
		body := transport.Body
		if body == nil {
			t.Error("Transport should be called when enabled is true")
		}
	} else {
		t.Fail()
	}
}
