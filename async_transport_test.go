package rollbar

import (
	"testing"
)

func TestAsyncTransportSend(t *testing.T) {
	transport := NewAsyncTransport("", "", 1)
	transport.SetLogger(&SilentClientLogger{})
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
	transport := NewAsyncTransport("", "", 0)
	transport.SetLogger(&SilentClientLogger{})
	body := map[string]interface{}{
		"hello": "world",
	}

	result := transport.Send(body)
	if result == nil {
		t.Error("Expected to receive ErrBufferFull")
	}
	transport.Wait()
	if transport.perMinCounter != 0 {
		t.Error("shouldSend check failed")
	}
}

func TestAsyncTransportSendRecover(t *testing.T) {
	transport := NewAsyncTransport("", "", 1)
	transport.SetLogger(&SilentClientLogger{})

	transport.Close()
	result := transport.Send(nil)
	if result == nil {
		t.Error("Expected to receive ErrChannelClosed")
	}
	transport.Wait()
}

func TestAsyncTransportClose(t *testing.T) {
	transport := NewAsyncTransport("", "", 1)
	transport.SetLogger(&SilentClientLogger{})
	result := transport.Close()
	if result != nil {
		t.Error("Close returned an unexpected error:", result)
	}
}

func TestAsyncTransportSetToken(t *testing.T) {
	transport := NewAsyncTransport("", "", 1)
	transport.SetLogger(&SilentClientLogger{})
	token := "abc"
	transport.SetToken(token)
	if transport.Token != token {
		t.Error("SetToken failed")
	}
}

func TestAsyncTransportSetEndpoint(t *testing.T) {
	transport := NewAsyncTransport("", "", 1)
	transport.SetLogger(&SilentClientLogger{})
	endpoint := "https://fake.com"
	transport.SetEndpoint(endpoint)
	if transport.Endpoint != endpoint {
		t.Error("SetEndpoint failed")
	}
}

func TestAsyncTransportNotSend(t *testing.T) {
	transport := NewAsyncTransport("", "", 2)
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
	transport.Wait()
	if transport.perMinCounter != 1 {
		t.Error("shouldSend check failed")
	}
}
