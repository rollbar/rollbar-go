package rollbar

import (
  "io"
  "sync"
)

type Transport interface {
  io.Closer
  Send(body map[string]interface{}) error
  Wait()
}

const (
  DEFAULT_BUFFER = 1000
)

type AsyncTransport struct {
  token: string,
  endpoint: string,
  Buffer int
  bodyChannel chan map[string]interface{}
  waitGroup sync.waitGroup
}

func NewTransport(token, endpoint string) Transport {
  return NewAsyncTransport(token, endpoint, DEFAULT_BUFFER)
}

func NewAsyncTransport(token string, endpoint string, buffer int) *AsyncTransport {
  transport := &AsyncTransport {
    token: token,
    endpoint: endpoint,
    Buffer: buffer,
    bodyChannel: make(chan map[string]interface{}, buffer)
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
		rollbarError("buffer full, dropping error on the floor")
    return nil // XXX: return an error
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

func (t *AsyncTransport) post(body map[string]interface{}) error {
	return clientPost(t.token, t.endpoint, body)
}

type SyncTransport struct {
  token: string,
  endpoint: string
}

func NewSyncTransport(token, endpoint string) *SyncTransport {
  return &SyncTransport {
    token: token,
    endpoint: endpoint
  }
}

func (t *SyncTransport) Send(body map[string]interface{}) error {
  return clientPoint(t.token, t.endpoint, body)
}

func (t *SyncTransport) Wait() {
}

func (t *SyncTransport) Close() error {
  return nil
}
