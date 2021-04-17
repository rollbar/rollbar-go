package rollbar

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTelemetryDefault(t *testing.T) {
	telemetry := NewTelemetry()
	assert.NotNil(t, telemetry)
	expectedTelemetry := &Telemetry{Queue: NewQueue(TelemetryQueueSize)}
	expectedTelemetry.Network.ScrubHeaders = regexp.MustCompile("Authorization")

	assert.Equal(t, expectedTelemetry, telemetry)

}

func TestNewTelemetryWithOptions(t *testing.T) {
	client := http.Client{}
	telemetry := NewTelemetry(SetCustomQueueSize(100), EnableNetworkTelemetry(&client),
		EnableNetworkTelemetryRequestHeaders(), EnableNetworkTelemetryResponseHeaders(), EnableLoggerTelemetry())
	expectedTelemetry := &Telemetry{Queue: telemetry.Queue}
	expectedTelemetry.Network.ScrubHeaders = regexp.MustCompile("Authorization")
	expectedTelemetry.Network.enableReqHeaders = true
	expectedTelemetry.Network.enableResHeaders = true
	expectedTelemetry.Network.Proxied = http.DefaultTransport
	expectedTelemetry.Logger.Writer = os.Stdout

	assert.Equal(t, expectedTelemetry, telemetry)
	assert.Equal(t, client.Transport, expectedTelemetry)
}
func TestPopulateBody(t *testing.T) {
	req := httptest.NewRequest("GET", "/some_url", nil)
	req.Header.Set("Some_name", "some_value")
	rec := httptest.NewRecorder()

	telemetry := NewTelemetry()
	EnableNetworkTelemetryRequestHeaders()(telemetry)
	EnableNetworkTelemetryResponseHeaders()(telemetry)
	data := telemetry.populateTransporterBody(req, rec.Result())
	assert.NotNil(t, data)
	assert.True(t, data["timestamp_ms"].(int64) > 0)
	delete(data, "timestamp_ms")

	expectedBodyData := map[string]interface{}{"method": "GET", "status_code": 200, "subtype": "http",
		"url": "://example.com/some_url", "request_headers": map[string]interface{}{"Some_name": "some_value"},
		"response": map[string]interface{}{"headers": map[string]interface{}{}}}

	expectedData := map[string]interface{}{"body": expectedBodyData, "level": "info", "source": "client", "type": "network"}

	assert.Equal(t, expectedData, data)
}

func TestPopulateLoggerBody(t *testing.T) {

	message := "some message"
	telemetry := NewTelemetry()

	data := telemetry.populateLoggerBody([]byte(message))

	assert.NotNil(t, data)
	assert.True(t, data["timestamp_ms"].(int64) > 0)
	delete(data, "timestamp_ms")
	expectedData := map[string]interface{}{"body": map[string]interface{}{"message": message}, "level": "log",
		"source": "client", "type": "log"}

	assert.Equal(t, expectedData, data)
}

func TestRoundTrip(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Print(">request: ", r)
		if r.URL.String() == "/good" {
			fmt.Fprintln(w, "Hello, client")
		}
	}))
	defer ts.Close()

	client := http.Client{}
	telemetry := NewTelemetry(EnableNetworkTelemetry(&client))

	req := httptest.NewRequest("GET", ts.URL+"/good", nil)
	res, err := telemetry.RoundTrip(req)
	assert.Nil(t, err)
	assert.NotNil(t, res)

	body, err := ioutil.ReadAll(res.Body)
	assert.Nil(t, err)
	assert.Equal(t, "Hello, client\n", string(body))

	items := telemetry.GetQueueItems()
	assert.NotNil(t, items)

	item := items[0]
	expectedData := telemetry.populateTransporterBody(req, res)
	assert.Equal(t, item, expectedData)
}
