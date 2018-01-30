package rollbar

import (
	"sync"
)

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

func (t *AsyncTransport) Wait() {
	t.waitGroup.Wait()
}

func (t *AsyncTransport) Close() error {
	t.Wait()
	return nil
}

func (t *AsyncTransport) SetToken(token string) {
	t.Token = token
}

func (t *AsyncTransport) SetEndpoint(endpoint string) {
	t.Endpoint = endpoint
}

func (t *AsyncTransport) SetLogger(logger ClientLogger) {
	t.Logger = logger
}

func (t *AsyncTransport) post(body map[string]interface{}) error {
	return clientPost(t.Token, t.Endpoint, body, t.Logger)
}
