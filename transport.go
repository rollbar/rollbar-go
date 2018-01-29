package rollbar

import (
	"io"
)

const (
	DEFAULT_BUFFER = 1000
)

type Transport interface {
	io.Closer
	Send(body map[string]interface{}) error
	Wait()
	SetToken(token string)
	SetEndpoint(endpoint string)
}

func NewTransport(token, endpoint string) Transport {
	return NewAsyncTransport(token, endpoint, DEFAULT_BUFFER)
}
