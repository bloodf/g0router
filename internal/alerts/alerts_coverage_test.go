package alerts

import (
	"strings"
	"testing"
)

func TestDispatchWebhookRequestError(t *testing.T) {
	cfg := ChannelConfig{
		Type:     ChannelWebhook,
		URL:      "://invalid-url",
		Events:   []EventType{EventQuotaDepleted},
		IsActive: true,
	}

	res := Dispatch(cfg, EventQuotaDepleted, map[string]any{"message": "test"})
	if res.Success {
		t.Fatal("expected failure")
	}
	if res.Error == nil || !strings.Contains(res.Error.Error(), "parse") {
		t.Fatalf("expected URL parse error, got %v", res.Error)
	}
}

func TestDispatchWebhookNetworkError(t *testing.T) {
	cfg := ChannelConfig{
		Type:     ChannelWebhook,
		URL:      "http://localhost:1/hook",
		Events:   []EventType{EventQuotaDepleted},
		IsActive: true,
	}

	res := Dispatch(cfg, EventQuotaDepleted, map[string]any{"message": "test"})
	if res.Success {
		t.Fatal("expected failure")
	}
	if res.Error == nil {
		t.Fatal("expected network error")
	}
}
