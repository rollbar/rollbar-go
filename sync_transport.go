package rollbar

import (
	"context"
	"time"
)

// SyncTransport is a concrete implementation of the Transport type which communicates with the
// Rollbar API synchronously.
type SyncTransport struct {
	baseTransport
}

// NewSyncTransport builds a synchronous transport which sends data to the Rollbar API at the
// specified endpoint using the given access token.
func NewSyncTransport(token, endpoint string) *SyncTransport {
	return &SyncTransport{
		baseTransport{
			Token:               token,
			Endpoint:            endpoint,
			RetryAttempts:       DefaultRetryAttempts,
			PrintPayloadOnError: true,
			ItemsPerMinute:      0,
			perMinCounter:       0,
			startTime:           time.Now(),
		},
	}
}

// Send the body to Rollbar.
// Returns errors associated with the http request if any.
// If the access token has not been set or is empty then this will
// not send anything and will return nil.
func (t *SyncTransport) Send(body map[string]interface{}) error {
	return t.doSend(body, t.RetryAttempts)
}

func (t *SyncTransport) doSend(body map[string]interface{}, retriesLeft int) error {
	elapsedTime := time.Now().Sub(t.startTime).Seconds()
	if elapsedTime < 0 || elapsedTime >= 60 {
		t.startTime = time.Now()
		t.perMinCounter = 0
	}
	if t.shouldSend() {
		canRetry, err := t.post(body)
		if err != nil {
			if !canRetry || retriesLeft <= 0 {
				if t.PrintPayloadOnError {
					writePayloadToStderr(t.Logger, body)
				}
				return err
			}
			return t.doSend(body, retriesLeft-1)
		} else {
			t.perMinCounter++
		}
	}
	return nil
}

// Wait is a no-op for the synchronous transport.
func (t *SyncTransport) Wait() {}

// Close is a no-op for the synchronous transport.
func (t *SyncTransport) Close() error {
	return nil
}
func (t *SyncTransport) SetContext(ctx context.Context) {
}
