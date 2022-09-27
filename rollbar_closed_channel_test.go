package rollbar

import (
	"testing"
)

func TestDisableDefaultClient(t *testing.T) {
	defer func() {
		std = NewAsync("", "development", "", hostname, "")

	}()
	DisableDefaultClient(false)
	err := std.push(map[string]interface{}{
		"data": map[string]interface{}{},
	})
	if err == nil {
		t.Error("error should indicate that channel is closed")
	}
	DisableDefaultClient(true)
	if std != nil {
		t.Error("error should indicate that std is nil")
	}
}
