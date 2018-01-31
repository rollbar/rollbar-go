package rollbar

import (
	"io"
	"log"
)

const (
	// DefaultBuffer is the default size of the buffered channel used
	// for queueing items to send to Rollbar in the asynchronous
	// implementation of Transport.
	DefaultBuffer = 1000
)

// Transport represents an object used for communicating with the Rollbar API.
type Transport interface {
	io.Closer
	// Send the body to the API, returning an error if the send fails. If the implementation to
	// asynchronous, then a failure can still occur when this method returns no error. In that case
	// this error represents a failure (or not) of enqueuing the payload.
	Send(body map[string]interface{}) error
	// Wait blocks until all messages currently waiting to be processed have been sent.
	Wait()
	// Set the access token to use for sending items with this transport.
	SetToken(token string)
	// Set the endpoint to send items to.
	SetEndpoint(endpoint string)
	// Set the logger to use instead of the standard log.Printf
	SetLogger(logger ClientLogger)
}

// ClientLogger is the interface used by the rollbar Client/Transport to report problems.
type ClientLogger interface {
	Printf(format string, args ...interface{})
}

// NewTransport creates a transport that sends items to the Rollbar API asynchronously.
func NewTransport(token, endpoint string) Transport {
	return NewAsyncTransport(token, endpoint, DefaultBuffer)
}

// -- rollbarError

func rollbarError(logger ClientLogger, format string, args ...interface{}) {
	format = "Rollbar error: " + format + "\n"
	if logger != nil {
		logger.Printf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}
