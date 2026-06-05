// Package notify provides a small, dependency-free webhook notifier used to
// alert operators when a connection goes stale and needs re-authentication.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Event describes a single notification payload.
type Event struct {
	Title   string
	Message string
	Level   string
}

// Notifier delivers events to an external destination.
type Notifier interface {
	Notify(ctx context.Context, event Event) error
}

// NewNotifier returns a Notifier for the given webhook URL. When url is empty a
// no-op notifier is returned. When client is nil http.DefaultClient is used.
func NewNotifier(url string, client *http.Client) Notifier {
	if strings.TrimSpace(url) == "" {
		return noopNotifier{}
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &webhookNotifier{url: url, client: client}
}

type noopNotifier struct{}

func (noopNotifier) Notify(ctx context.Context, event Event) error { return nil }

// webhookNotifier POSTs a JSON body to a configured URL. It detects Discord and
// Telegram webhook URLs and shapes the payload accordingly; otherwise it sends a
// generic {title,message,level} body.
type webhookNotifier struct {
	url    string
	client *http.Client
}

func (w *webhookNotifier) Notify(ctx context.Context, event Event) error {
	body, err := json.Marshal(payloadFor(w.url, event))
	if err != nil {
		return fmt.Errorf("encode notification: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build notification request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("send notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("notification webhook returned status %d", resp.StatusCode)
	}
	return nil
}

// payloadFor returns the destination-specific JSON shape for the event.
func payloadFor(url string, event Event) any {
	text := event.Title
	if event.Message != "" {
		if text != "" {
			text += ": "
		}
		text += event.Message
	}
	switch {
	case strings.Contains(url, "discord.com/api/webhooks"):
		return map[string]string{"content": text}
	case strings.Contains(url, "api.telegram.org"):
		return map[string]string{"text": text}
	default:
		return map[string]string{
			"title":   event.Title,
			"message": event.Message,
			"level":   event.Level,
		}
	}
}
