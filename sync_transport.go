package rollbar

type SyncTransport struct {
	Token    string
	Endpoint string
}

func NewSyncTransport(token, endpoint string) *SyncTransport {
	return &SyncTransport{
		Token:    token,
		Endpoint: endpoint,
	}
}

func (t *SyncTransport) Send(body map[string]interface{}) error {
	return clientPost(t.Token, t.Endpoint, body)
}

func (t *SyncTransport) Wait() {}

func (t *SyncTransport) Close() error {
	return nil
}

func (t *SyncTransport) SetToken(token string) {
	t.Token = token
}

func (t *SyncTransport) SetEndpoint(endpoint string) {
	t.Endpoint = endpoint
}
