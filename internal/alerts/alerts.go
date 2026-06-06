package alerts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// EventType is a known alert event.
type EventType string

const (
	EventQuotaDepleted   EventType = "quota_depleted"
	EventConnectionStale EventType = "connection_stale"
	EventRateLimit       EventType = "rate_limit"
	EventBudgetExhausted EventType = "budget_exhausted"
)

// ChannelType identifies the delivery mechanism.
type ChannelType string

const (
	ChannelWebhook  ChannelType = "webhook"
	ChannelDiscord  ChannelType = "discord"
	ChannelTelegram ChannelType = "telegram"
)

// ChannelConfig holds the runtime configuration for a channel.
// No store imports — callers load and decrypt config before passing it in.
type ChannelConfig struct {
	Type     ChannelType
	URL      string
	Token    string
	ChatID   string
	Events   []EventType
	IsActive bool
}

// DispatchResult is the outcome of a single dispatch attempt.
type DispatchResult struct {
	Success bool
	Error   error
}

// Dispatch sends the event payload to the channel if the channel is active and
// subscribes to the event. It retries up to 3 times with exponential backoff.
func Dispatch(cfg ChannelConfig, event EventType, payload map[string]any) DispatchResult {
	if !cfg.IsActive {
		return DispatchResult{Error: fmt.Errorf("channel is inactive")}
	}
	if !subscribes(cfg.Events, event) {
		return DispatchResult{Error: fmt.Errorf("channel not subscribed to event %q", event)}
	}

	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		err = send(cfg, event, payload)
		if err == nil {
			return DispatchResult{Success: true}
		}
		if attempt < 3 {
			time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
		}
	}
	return DispatchResult{Error: fmt.Errorf("dispatch failed after 3 attempts: %w", err)}
}

func subscribes(events []EventType, target EventType) bool {
	for _, e := range events {
		if e == target {
			return true
		}
	}
	return false
}

func send(cfg ChannelConfig, event EventType, payload map[string]any) error {
	body, err := buildBody(cfg, event, payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, cfg.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func buildBody(cfg ChannelConfig, event EventType, payload map[string]any) ([]byte, error) {
	switch cfg.Type {
	case ChannelWebhook:
		msg := map[string]any{
			"event":   string(event),
			"message": payload["message"],
		}
		return json.Marshal(msg)
	case ChannelDiscord:
		msg := map[string]any{
			"content": payload["message"],
		}
		return json.Marshal(msg)
	case ChannelTelegram:
		url := fmt.Sprintf("%s/bot%s/sendMessage", cfg.URL, cfg.Token)
		cfg.URL = url
		msg := map[string]any{
			"chat_id": cfg.ChatID,
			"text":    payload["message"],
		}
		return json.Marshal(msg)
	default:
		return nil, fmt.Errorf("unknown channel type %q", cfg.Type)
	}
}
