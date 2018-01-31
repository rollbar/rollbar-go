package rollbar_test

import (
	"github.com/rollbar/rollbar-go"
	"testing"
)

func TestAsyncTransportSend(t *testing.T) {
	transport := rollbar.NewAsyncTransport("", "", 1)
	transport.SetLogger(&rollbar.SilentClientLogger{})
	body := map[string]interface{}{
		"hello": "world",
	}
	result := transport.Send(body)
	if result != nil {
		t.Error("Send returned an unexpected error:", result)
	}
	transport.Wait()
}

func TestAsyncTransportSendFull(t *testing.T) {
	transport := rollbar.NewAsyncTransport("", "", 1)
	transport.SetLogger(&rollbar.SilentClientLogger{})
	body := map[string]interface{}{
		"hello": "world",
	}

	transport.Send(body)
	result := transport.Send(body)
	if result == nil {
		t.Error("Expected to receive ErrBufferFull")
	}
	transport.Wait()
}

func TestAsyncTransportClose(t *testing.T) {
	transport := rollbar.NewAsyncTransport("", "", 1)
	transport.SetLogger(&rollbar.SilentClientLogger{})
	result := transport.Close()
	if result != nil {
		t.Error("Close returned an unexpected error:", result)
	}
}

func TestAsyncTransportSetToken(t *testing.T) {
	transport := rollbar.NewAsyncTransport("", "", 1)
	transport.SetLogger(&rollbar.SilentClientLogger{})
	token := "abc"
	transport.SetToken(token)
	if transport.Token != token {
		t.Error("SetToken failed")
	}
}

func TestAsyncTransportSetEndpoint(t *testing.T) {
	transport := rollbar.NewAsyncTransport("", "", 1)
	transport.SetLogger(&rollbar.SilentClientLogger{})
	endpoint := "https://fake.com"
	transport.SetEndpoint(endpoint)
	if transport.Endpoint != endpoint {
		t.Error("SetEndpoint failed")
	}
}
