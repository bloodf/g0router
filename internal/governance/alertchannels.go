package governance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

// Sender delivers a test notification to a channel's configured target. The
// real implementation performs a best-effort HTTP POST; tests inject a fake.
type Sender interface {
	Send(ctx context.Context, channelType string, config map[string]any) error
}

// AlertDispatcher sends test notifications through an injectable Sender.
type AlertDispatcher struct {
	sender Sender
}

// NewAlertDispatcher creates a dispatcher with the given sender.
func NewAlertDispatcher(sender Sender) *AlertDispatcher {
	return &AlertDispatcher{sender: sender}
}

// Dispatch sends a test notification for the channel. It returns a result
// suitable for the API response; the message never carries the secret config.
func (d *AlertDispatcher) Dispatch(ctx context.Context, ch *store.AlertChannel) (ok bool, message string) {
	if err := d.sender.Send(ctx, ch.ChannelType, ch.Config); err != nil {
		return false, "Test notification failed"
	}
	return true, "Test notification sent"
}

// httpSender posts a small JSON payload to the channel's configured URL.
type httpSender struct {
	client *http.Client
}

// NewHTTPSender returns a Sender backed by an HTTP client with a short timeout.
func NewHTTPSender() Sender {
	return &httpSender{client: &http.Client{Timeout: 5 * time.Second}}
}

func (s *httpSender) Send(ctx context.Context, channelType string, config map[string]any) error {
	target := configString(config, "url")
	if target == "" {
		target = configString(config, "webhook_url")
	}
	if target == "" {
		return fmt.Errorf("alert channel %s has no configured url", channelType)
	}

	payload, err := json.Marshal(map[string]any{
		"text":  "g0router test notification",
		"event": "test",
	})
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(payload))
	if err != nil {
		// The error from net/http may embed the URL; wrap with a generic message
		// so callers never surface the secret target.
		return fmt.Errorf("build alert request for %s", channelType)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("deliver alert to %s channel", channelType)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("alert %s channel returned status %d", channelType, resp.StatusCode)
	}
	return nil
}

func configString(config map[string]any, key string) string {
	if config == nil {
		return ""
	}
	if v, ok := config[key].(string); ok {
		return v
	}
	return ""
}
