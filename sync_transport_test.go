package rollbar

import (
	"testing"
)

func TestSyncTransportSend(t *testing.T) {
	transport := NewSyncTransport("", "")
	transport.SetLogger(&SilentClientLogger{})
	body := map[string]interface{}{
		"hello": "world",
	}
	result := transport.Send(body)
	if result != nil {
		t.Error("Send returned an unexpected error:", result)
	}
}

func TestSyncTransportSendTwice(t *testing.T) {
	transport := NewSyncTransport("", "")
	transport.SetLogger(&SilentClientLogger{})
	body := map[string]interface{}{
		"hello": "world",
	}

	transport.Send(body)
	result := transport.Send(body)
	if result != nil {
		t.Error("Send returned an unexpected error:", result)
	}

	if transport.perMinCounter != 2 {
		t.Error("shouldSend check failed")
	}
}

func TestSyncTransportClose(t *testing.T) {
	transport := NewSyncTransport("", "")
	transport.SetLogger(&SilentClientLogger{})
	result := transport.Close()
	if result != nil {
		t.Error("Close returned an unexpected error:", result)
	}
}

func TestSyncTransportSetToken(t *testing.T) {
	transport := NewSyncTransport("", "")
	transport.SetLogger(&SilentClientLogger{})
	token := "abc"
	transport.SetToken(token)
	if transport.Token != token {
		t.Error("SetToken failed")
	}
}

func TestSyncTransportSetEndpoint(t *testing.T) {
	transport := NewSyncTransport("", "")
	transport.SetLogger(&SilentClientLogger{})
	endpoint := "https://fake.com"
	transport.SetEndpoint(endpoint)
	if transport.Endpoint != endpoint {
		t.Error("SetEndpoint failed")
	}
}

func TestSyncTransportNotSend(t *testing.T) {
	transport := NewSyncTransport("", "")
	transport.SetLogger(&SilentClientLogger{})
	transport.SetItemsPerMinute(1)
	if transport.ItemsPerMinute != 1 {
		t.Error("SetItemsPerMinute failed")
	}

	body := map[string]interface{}{
		"hello": "world",
	}

	transport.Send(body)
	result := transport.Send(body)
	if result != nil {
		t.Error("Send returned an unexpected error:", result)
	}
	if transport.perMinCounter != 1 {
		t.Error("shouldSend check failed")
	}
}
