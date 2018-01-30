package rollbar

import (
	"io"
	"log"
)

const (
	// We use a buffered channel for queueing items to send to Rollbar in the asynchronous
	// implementation of Transport. By default the channel has a capacity of this size.
	DEFAULT_BUFFER = 1000
)

// Transport represents an object used for communicating with the Rollbar API.
type Transport interface {
	io.Closer
	// Send the body to the API, returning an error if the send fails. If the implementation to
	// asyncronous, then a failure can still occur when this method returns no error. In that case
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

/// NewTransport creates a transport that sends items to the Rollbar API asyncronously.
func NewTransport(token, endpoint string) Transport {
	return NewAsyncTransport(token, endpoint, DEFAULT_BUFFER)
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
