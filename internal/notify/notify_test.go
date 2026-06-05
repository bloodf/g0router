package notify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestWebhookNotifierGenericPost(t *testing.T) {
	var gotBody []byte
	var gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := NewNotifier(srv.URL, srv.Client())
	err := n.Notify(context.Background(), Event{Title: "Stale", Message: "conn down", Level: "warning"})
	if err != nil {
		t.Fatalf("Notify: %v", err)
	}
	if gotContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}

	var payload map[string]any
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if payload["title"] != "Stale" || payload["message"] != "conn down" || payload["level"] != "warning" {
		t.Errorf("generic payload = %v, want title/message/level fields", payload)
	}
}

func TestWebhookNotifierDiscordShape(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	// Force discord shape by passing a discord-looking URL but routing the
	// client transport to the test server.
	client := srv.Client()
	client.Transport = rewriteTransport{base: client.Transport, target: srv.URL}
	n := NewNotifier("https://discord.com/api/webhooks/123/abc", client)
	if err := n.Notify(context.Background(), Event{Title: "T", Message: "M", Level: "info"}); err != nil {
		t.Fatalf("Notify: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	content, ok := payload["content"].(string)
	if !ok {
		t.Fatalf("discord payload missing content: %v", payload)
	}
	if !strings.Contains(content, "T") || !strings.Contains(content, "M") {
		t.Errorf("discord content = %q, want title+message", content)
	}
}

func TestWebhookNotifierTelegramShape(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := srv.Client()
	client.Transport = rewriteTransport{base: client.Transport, target: srv.URL}
	n := NewNotifier("https://api.telegram.org/bot123/sendMessage?chat_id=42", client)
	if err := n.Notify(context.Background(), Event{Title: "T", Message: "M", Level: "info"}); err != nil {
		t.Fatalf("Notify: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := payload["text"]; !ok {
		t.Errorf("telegram payload missing text: %v", payload)
	}
}

func TestWebhookNotifierNon2xxError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	n := NewNotifier(srv.URL, srv.Client())
	if err := n.Notify(context.Background(), Event{Title: "T", Message: "M"}); err == nil {
		t.Fatal("Notify should return error on non-2xx response")
	}
}

func TestNewNotifierEmptyURLNoOp(t *testing.T) {
	var called int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&called, 1)
	}))
	defer srv.Close()

	n := NewNotifier("", srv.Client())
	if err := n.Notify(context.Background(), Event{Title: "T", Message: "M"}); err != nil {
		t.Fatalf("no-op Notify should not error: %v", err)
	}
	if atomic.LoadInt32(&called) != 0 {
		t.Error("no-op notifier should not make any HTTP request")
	}
}

// rewriteTransport redirects every request to target while preserving the
// original request body, so tests can use provider-specific URLs without real
// network access.
type rewriteTransport struct {
	base   http.RoundTripper
	target string
}

func (rt rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := rt.base
	if base == nil {
		base = http.DefaultTransport
	}
	u, err := req.URL.Parse(rt.target)
	if err != nil {
		return nil, err
	}
	req.URL.Scheme = u.Scheme
	req.URL.Host = u.Host
	req.Host = u.Host
	return base.RoundTrip(req)
}
