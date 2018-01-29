package rollbar

import (
	"sync"
)

type AsyncTransport struct {
	Token       string
	Endpoint    string
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
		var err = ErrBufferFull{}
		rollbarError(err.Error())
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

func (t *AsyncTransport) post(body map[string]interface{}) error {
	return clientPost(t.Token, t.Endpoint, body)
}
