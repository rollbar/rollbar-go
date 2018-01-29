package rollbar_test

import (
	"errors"
	"github.com/rollbar/rollbar-go"
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

func (t *TestTransport) SetToken(_t string)    {}
func (t *TestTransport) SetEndpoint(_e string) {}
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

	errorIfEqual(token, client.Token(), t)
	errorIfEqual(environment, client.Environment(), t)
	errorIfEqual(endpoint, client.Endpoint(), t)
	errorIfEqual(platform, client.Platform(), t)
	errorIfEqual(codeVersion, client.CodeVersion(), t)
	errorIfEqual(host, client.ServerHost(), t)
	errorIfEqual(root, client.ServerRoot(), t)

	client.SetToken(token)
	client.SetEnvironment(environment)
	client.SetEndpoint(endpoint)
	client.SetPlatform(platform)
	client.SetCodeVersion(codeVersion)
	client.SetServerHost(host)
	client.SetServerRoot(root)

	errorIfNotEqual(token, client.Token(), t)
	errorIfNotEqual(environment, client.Environment(), t)
	errorIfNotEqual(endpoint, client.Endpoint(), t)
	errorIfNotEqual(platform, client.Platform(), t)
	errorIfNotEqual(codeVersion, client.CodeVersion(), t)
	errorIfNotEqual(host, client.ServerHost(), t)
	errorIfNotEqual(root, client.ServerRoot(), t)
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
