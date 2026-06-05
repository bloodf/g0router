package notify

import (
	"context"
	"net/http"
	"testing"
)

// TestNewNotifierNilClientUsesDefault exercises the client==nil branch in NewNotifier.
func TestNewNotifierNilClientUsesDefault(t *testing.T) {
	// Passing nil client: NewNotifier must not panic and must return a webhookNotifier.
	// We pass a whitespace-only URL as well to verify TrimSpace branch.
	n := NewNotifier("   ", nil)
	if _, ok := n.(noopNotifier); !ok {
		t.Fatalf("whitespace URL should return noopNotifier")
	}

	// Now pass a real URL with nil client — exercises the `client = http.DefaultClient` branch.
	// We don't actually call Notify (would hit the network); just confirm no panic on construction.
	n2 := NewNotifier("https://example.com/hook", nil)
	if n2 == nil {
		t.Fatal("NewNotifier with nil client returned nil")
	}
	w, ok := n2.(*webhookNotifier)
	if !ok {
		t.Fatalf("NewNotifier with nil client returned %T, want *webhookNotifier", n2)
	}
	if w.client != http.DefaultClient {
		t.Error("NewNotifier with nil client should assign http.DefaultClient")
	}
}

// TestWebhookNotifierBadURLRequestBuildError exercises the http.NewRequestWithContext
// error branch inside Notify (line 56-58).
func TestWebhookNotifierBadURLRequestBuildError(t *testing.T) {
	// An invalid URL causes http.NewRequestWithContext to return an error.
	n := &webhookNotifier{url: "://bad-url", client: http.DefaultClient}
	err := n.Notify(context.Background(), Event{Title: "T", Message: "M"})
	if err == nil {
		t.Fatal("Notify with bad URL should return error")
	}
}

// TestWebhookNotifierTransportError exercises the client.Do error branch (line 63-65).
func TestWebhookNotifierTransportError(t *testing.T) {
	// A client with an always-failing transport.
	client := &http.Client{
		Transport: alwaysErrorTransport{},
	}
	n := &webhookNotifier{url: "http://localhost:0/hook", client: client}
	err := n.Notify(context.Background(), Event{Title: "T", Message: "M"})
	if err == nil {
		t.Fatal("Notify with failing transport should return error")
	}
}

// alwaysErrorTransport rejects every request.
type alwaysErrorTransport struct{}

func (alwaysErrorTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, &alwaysError{}
}

type alwaysError struct{}

func (*alwaysError) Error() string { return "transport error injected by test" }
