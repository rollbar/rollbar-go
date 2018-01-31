package rollbar

import (
	"sync"
)

// AsyncTransport is a concrete implementation of the Transport type which communicates with the
// Rollbar API asynchronously using a buffered channel.
type AsyncTransport struct {
	// Rollbar access token used by this transport for communication with the Rollbar API.
	Token string
	// Endpoint to post items to.
	Endpoint string
	// Logger used to report errors when sending data to Rollbar, e.g.
	// when the Rollbar API returns 409 Too Many Requests response.
	// If not set, the client will use the standard log.Printf by default.
	Logger ClientLogger
	// Buffer is the size of the channel used for queueing asynchronous payloads for sending to
	// Rollbar.
	Buffer      int
	bodyChannel chan map[string]interface{}
	waitGroup   sync.WaitGroup
}

// NewAsyncTransport builds an asynchronous transport which sends data to the Rollbar API at the
// specified endpoint using the given access token. The channel is limited to the size of the input
// buffer argument.
func NewAsyncTransport(token string, endpoint string, buffer int) *AsyncTransport {
	transport := &AsyncTransport{
		Token:       token,
		Endpoint:    endpoint,
		Buffer:      buffer,
		bodyChannel: make(chan map[string]interface{}, buffer),
	}

	go func() {
		for body := range transport.bodyChannel {
			transport.post(body)
			transport.waitGroup.Done()
		}
	}()
	return transport
}

// Send the body to Rollbar if the channel is not currently full.
// Returns ErrBufferFull if the underlying channel is full.
func (t *AsyncTransport) Send(body map[string]interface{}) error {
	if len(t.bodyChannel) < t.Buffer {
		t.waitGroup.Add(1)
		t.bodyChannel <- body
	} else {
		err := ErrBufferFull{}
		rollbarError(t.Logger, err.Error())
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
	t.Wait()
	return nil
}

// SetToken updates the token to use for future API requests.
// Any request that is currently in the queue will use this
// updated token value. If you want to change the token without
// affecting the items currently in the queue, use Wait first
// to flush the queue.
func (t *AsyncTransport) SetToken(token string) {
	t.Token = token
}

// SetEndpoint updates the API endpoint to send items to.
// Any request that is currently in the queue will use this
// updated endpoint value. If you want to change the endpoint without
// affecting the items currently in the queue, use Wait first
// to flush the queue.
func (t *AsyncTransport) SetEndpoint(endpoint string) {
	t.Endpoint = endpoint
}

// SetLogger updates the logger that this transport uses for reporting errors that occur while
// processing items.
func (t *AsyncTransport) SetLogger(logger ClientLogger) {
	t.Logger = logger
}

func (t *AsyncTransport) post(body map[string]interface{}) error {
	return clientPost(t.Token, t.Endpoint, body, t.Logger)
}
