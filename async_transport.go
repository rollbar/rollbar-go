package rollbar

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// AsyncTransport is a concrete implementation of the Transport type which communicates with the
// Rollbar API asynchronously using a buffered channel.
type AsyncTransport struct {
	ctx context.Context
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

func isClosed(ch chan payload) bool {
	if len(ch) == 0 {
		select {
		case _, ok := <-ch:
			return !ok
		}
	}
	return false
}

// NewAsyncTransport builds an asynchronous transport which sends data to the Rollbar API at the
// specified endpoint using the given access token. The channel is limited to the size of the input
// buffer argument.
func NewAsyncTransport(token string, endpoint string, buffer int, opts ...transportOption) *AsyncTransport {
	transport := &AsyncTransport{
		baseTransport: baseTransport{
			Token:               token,
			Endpoint:            endpoint,
			RetryAttempts:       DefaultRetryAttempts,
			PrintPayloadOnError: true,
			ItemsPerMinute:      0,
		},
		bodyChannel: make(chan payload, buffer),
		Buffer:      buffer,
	}
	for _, opt := range opts {
		// Call the option giving the instantiated
		// Transport as the argument
		opt(transport)
	}
	if transport.ctx == nil {
		transport.ctx = context.Background()
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				pc, _, _, _ := runtime.Caller(4)
				fnName := runtime.FuncForPC(pc).Name()
				if isClosed(transport.bodyChannel) {
					fmt.Println(fnName, "recover: channel is closed")
				} else {
					fmt.Println(fnName, "recovered:", r)
				}
			}
		}()

		for p := range transport.bodyChannel {
			elapsedTime := time.Now().Sub(transport.startTime).Seconds()
			if elapsedTime < 0 || elapsedTime >= 60 {
				transport.startTime = time.Now()
				transport.perMinCounter = 0
			}
			if transport.shouldSend() {
				canRetry, err := transport.post(p.body)
				if err != nil {
					if canRetry && p.retriesLeft > 0 {
						p.retriesLeft -= 1
						select {
						case <-transport.ctx.Done(): // check for early termination
							writePayloadToStderr(transport.Logger, p.body)
							transport.waitGroup.Done()
							return
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
					transport.perMinCounter++
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
func (t *AsyncTransport) Send(body map[string]interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			pc, _, _, _ := runtime.Caller(4)
			fnName := runtime.FuncForPC(pc).Name()
			if _, ok := err.(*ErrBufferFull); !ok && isClosed(t.bodyChannel) {
				fmt.Println(fnName, "recover: channel is closed")
				t.waitGroup.Done()
				err = ErrChannelClosed{}
			} else {
				fmt.Println(fnName, "recovered:", r)
			}
		}
	}()
	if len(t.bodyChannel) < t.Buffer {
		t.waitGroup.Add(1)
		p := payload{
			body:        body,
			retriesLeft: t.RetryAttempts,
		}
		select {
		case <-t.ctx.Done(): // check for early termination
			writePayloadToStderr(t.Logger, body)
			return t.ctx.Err()
		case t.bodyChannel <- p:
		default:
		}
	} else {
		err = ErrBufferFull{}
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

func (t *AsyncTransport) setContext(ctx context.Context) {
	t.ctx = ctx
}

func (t *AsyncTransport) getContext() context.Context {
	return t.ctx
}
