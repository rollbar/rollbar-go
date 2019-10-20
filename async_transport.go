package rollbar

import (
	"sync"
)

// AsyncTransport is a concrete implementation of the Transport type which communicates with the
// Rollbar API asynchronously using a buffered channel.
type AsyncTransport struct {
	baseTransport
	// Buffer is the size of the channel used for queueing asynchronous payloads for sending to
	// Rollbar.
	Buffer      int
	bodyChannel chan payload
	waitGroup   sync.WaitGroup
}

type payload struct {
	body        map[string]interface{}
	retriesLeft int
}

// NewAsyncTransport builds an asynchronous transport which sends data to the Rollbar API at the
// specified endpoint using the given access token. The channel is limited to the size of the input
// buffer argument.
func NewAsyncTransport(token string, endpoint string, buffer int) *AsyncTransport {
	transport := &AsyncTransport{
		baseTransport: baseTransport{
			Token:               token,
			Endpoint:            endpoint,
			RetryAttempts:       DefaultRetryAttempts,
			PrintPayloadOnError: true,
		},
		bodyChannel: make(chan payload, buffer),
		Buffer:      buffer,
	}

	go func() {
		for p := range transport.bodyChannel {
			canRetry, err := transport.post(p.body)
			if err != nil {
				if canRetry && p.retriesLeft > 0 {
					p.retriesLeft -= 1
					select {
					case transport.bodyChannel <- p:
					default:
						// This can happen if the bodyChannel had an item added to it from another
						// thread while we are processing such that the channel is now full. If we try
						// to send the payload back to the channel without this select statement we
						// could deadlock. Instead we consider this a retry failure.
						if transport.PrintPayloadOnError {
							writePayloadToStderr(transport.Logger, p.body)
						}
						transport.waitGroup.Done()
					}
				} else {
					if transport.PrintPayloadOnError {
						writePayloadToStderr(transport.Logger, p.body)
					}
					transport.waitGroup.Done()
				}
			} else {
				transport.waitGroup.Done()
			}
		}
	}()
	return transport
}

// Send the body to Rollbar if the channel is not currently full.
// Returns ErrBufferFull if the underlying channel is full.
func (t *AsyncTransport) Send(body map[string]interface{}) error {
	if len(t.bodyChannel) < t.Buffer {
		t.waitGroup.Add(1)
		p := payload{
			body:        body,
			retriesLeft: t.RetryAttempts,
		}
		t.bodyChannel <- p
	} else {
		err := ErrBufferFull{}
		rollbarError(t.Logger, err.Error())
		if t.PrintPayloadOnError {
			writePayloadToStderr(t.Logger, body)
		}
		return err
	}
	return nil
}

// Wait blocks until all of the items currently in the queue have been sent.
func (t *AsyncTransport) Wait() {
	t.waitGroup.Wait()
}

// Close is an alias for Wait for the asynchronous transport
func (t *AsyncTransport) Close() error {
	close(t.bodyChannel)
	t.Wait()
	return nil
}
