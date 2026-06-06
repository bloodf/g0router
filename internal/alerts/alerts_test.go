package alerts

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEventTypes(t *testing.T) {
	if EventQuotaDepleted != "quota_depleted" {
		t.Fatalf("EventQuotaDepleted = %q", EventQuotaDepleted)
	}
	if EventConnectionStale != "connection_stale" {
		t.Fatalf("EventConnectionStale = %q", EventConnectionStale)
	}
	if EventRateLimit != "rate_limit" {
		t.Fatalf("EventRateLimit = %q", EventRateLimit)
	}
	if EventBudgetExhausted != "budget_exhausted" {
		t.Fatalf("EventBudgetExhausted = %q", EventBudgetExhausted)
	}
}

func TestDispatchWebhook(t *testing.T) {
	var receivedBody string
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		data, _ := io.ReadAll(r.Body)
		receivedBody = string(data)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := ChannelConfig{
		Type:     ChannelWebhook,
		URL:      server.URL + "/hook",
		Events:   []EventType{EventQuotaDepleted},
		IsActive: true,
	}

	res := Dispatch(cfg, EventQuotaDepleted, map[string]any{"message": "quota hit"})
	if !res.Success {
		t.Fatalf("expected success, got error: %v", res.Error)
	}

	if !strings.Contains(receivedBody, "quota hit") {
		t.Fatalf("body = %q", receivedBody)
	}
	if receivedHeaders.Get("Content-Type") != "application/json" {
		t.Fatalf("content-type = %q", receivedHeaders.Get("Content-Type"))
	}
}

func TestDispatchWebhookNotSubscribed(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := ChannelConfig{
		Type:     ChannelWebhook,
		URL:      server.URL + "/hook",
		Events:   []EventType{EventQuotaDepleted},
		IsActive: true,
	}

	res := Dispatch(cfg, EventRateLimit, map[string]any{"message": "rate limited"})
	if res.Success {
		t.Fatal("expected skipped (not success)")
	}
	if res.Error == nil || !strings.Contains(res.Error.Error(), "not subscribed") {
		t.Fatalf("expected not-subscribed error, got %v", res.Error)
	}
	if called {
		t.Fatal("server should not have been called")
	}
}

func TestDispatchInactiveChannel(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := ChannelConfig{
		Type:     ChannelWebhook,
		URL:      server.URL + "/hook",
		Events:   []EventType{EventQuotaDepleted},
		IsActive: false,
	}

	res := Dispatch(cfg, EventQuotaDepleted, map[string]any{"message": "quota hit"})
	if res.Success {
		t.Fatal("expected skipped")
	}
	if res.Error == nil || !strings.Contains(res.Error.Error(), "inactive") {
		t.Fatalf("expected inactive error, got %v", res.Error)
	}
	if called {
		t.Fatal("server should not have been called")
	}
}

func TestDispatchWebhookRetriesOnFailure(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	cfg := ChannelConfig{
		Type:     ChannelWebhook,
		URL:      server.URL + "/hook",
		Events:   []EventType{EventQuotaDepleted},
		IsActive: true,
	}

	res := Dispatch(cfg, EventQuotaDepleted, map[string]any{"message": "quota hit"})
	if res.Success {
		t.Fatal("expected failure")
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
	if res.Error == nil {
		t.Fatal("expected error")
	}
}

func TestDispatchDiscord(t *testing.T) {
	var receivedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		receivedBody = string(data)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cfg := ChannelConfig{
		Type:     ChannelDiscord,
		URL:      server.URL + "/discord",
		Events:   []EventType{EventBudgetExhausted},
		IsActive: true,
	}

	res := Dispatch(cfg, EventBudgetExhausted, map[string]any{"message": "out of money"})
	if !res.Success {
		t.Fatalf("expected success, got %v", res.Error)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(receivedBody), &payload); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if payload["content"] != "out of money" {
		t.Fatalf("content = %v", payload["content"])
	}
}

func TestDispatchTelegram(t *testing.T) {
	var receivedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		receivedBody = string(data)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	cfg := ChannelConfig{
		Type:     ChannelTelegram,
		URL:      server.URL + "/telegram",
		Token:    "bot-token",
		ChatID:   "12345",
		Events:   []EventType{EventConnectionStale},
		IsActive: true,
	}

	res := Dispatch(cfg, EventConnectionStale, map[string]any{"message": "stale conn"})
	if !res.Success {
		t.Fatalf("expected success, got %v", res.Error)
	}

	if !strings.Contains(receivedBody, "stale conn") {
		t.Fatalf("body = %q", receivedBody)
	}
}

func TestDispatchUnknownChannelType(t *testing.T) {
	cfg := ChannelConfig{
		Type:     "unknown",
		Events:   []EventType{EventQuotaDepleted},
		IsActive: true,
	}

	res := Dispatch(cfg, EventQuotaDepleted, map[string]any{})
	if res.Success {
		t.Fatal("expected failure")
	}
	if res.Error == nil || !strings.Contains(res.Error.Error(), "unknown channel type") {
		t.Fatalf("expected unknown type error, got %v", res.Error)
	}
}
