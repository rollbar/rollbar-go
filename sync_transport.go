package rollbar

type SyncTransport struct {
	// Rollbar access token used by this transport for communication with the Rollbar API.
	Token string
	// Endpoint to post items to.
	Endpoint string
	// Logger used to report errors when sending data to Rollbar, e.g.
	// when the Rollbar API returns 409 Too Many Requests response.
	// If not set, the client will use the standard log.Printf by default.
	Logger ClientLogger
}

func NewSyncTransport(token, endpoint string) *SyncTransport {
	return &SyncTransport{
		Token:    token,
		Endpoint: endpoint,
	}
}

func (t *SyncTransport) Send(body map[string]interface{}) error {
	return clientPost(t.Token, t.Endpoint, body, t.Logger)
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

func (t *SyncTransport) SetLogger(logger ClientLogger) {
	t.Logger = logger
}
