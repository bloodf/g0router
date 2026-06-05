package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/notify"
	"github.com/bloodf/g0router/internal/proxy"
)

// TestRunConnectionRefreshOncePanicsAreRecovered exercises the panic recovery
// inside runConnectionRefreshGuarded (lines 230-234).
func TestRunConnectionRefreshGuardedPanicRecovered(t *testing.T) {
	srv := NewServer(ServerConfig{})
	// Must not propagate the panic.
	srv.runConnectionRefreshGuarded(func(time.Time) {
		panic("injected panic for coverage")
	}, time.Now().UTC())
}

// TestRunConnectionRefreshOnceNilRefresher exercises the early return when
// connRefresher is nil (line 244-246).
func TestRunConnectionRefreshOnceNilRefresher(t *testing.T) {
	srv := NewServer(ServerConfig{})
	// connRefresher is nil → early return, no panic.
	srv.runConnectionRefreshOnce(time.Now().UTC())
}

// TestStartConnectionRefreshNilRefresherNoOp verifies StartConnectionRefresh
// returns immediately (no goroutine launched) when connRefresher is nil.
func TestStartConnectionRefreshNilRefresherNoOp(t *testing.T) {
	srv := NewServer(ServerConfig{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.StartConnectionRefresh(noopContext())
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("StartConnectionRefresh with nil refresher did not return quickly")
	}
}

// TestStartConnectionRefreshZeroIntervalFallsBackToDefault exercises the
// interval<=0 fallback branch in StartConnectionRefresh (lines 205-207).
func TestStartConnectionRefreshZeroIntervalFallsBack(t *testing.T) {
	ref := &fakeConnRefresher{}
	srv := newRefreshTestServer(t, ref, &captureNotifier{})
	srv.connectionRefreshInterval = 0 // triggers fallback

	ref.set([]proxy.RefreshOutcome{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Must not panic; the fallback interval is 1 minute but we cancel quickly.
	srv.StartConnectionRefresh(ctx)
	// Give the startup goroutine a moment to enter the loop, then cancel.
	time.Sleep(20 * time.Millisecond)
	cancel()
}

// TestStartConnectionRefreshTickerFires exercises the ticker.C arm (line 221)
// in the StartConnectionRefresh goroutine.
func TestStartConnectionRefreshTickerFires(t *testing.T) {
	ref := &fakeConnRefresher{}
	notifier := &captureNotifier{}
	srv := newRefreshTestServer(t, ref, notifier)
	srv.connectionRefreshInterval = 20 * time.Millisecond

	// First failure: startup pass notifies once, then throttled on subsequent ticks.
	// Use a connection that recovers after first tick so the second tick can re-notify.
	// Strategy: first return a failure (notified), then a recovery (clears throttle),
	// then a failure again (second notification). We achieve this by counting calls.
	callCount := 0
	var countedRefresher countingFakeRefresher
	countedRefresher.outcomes = func() []proxy.RefreshOutcome {
		callCount++
		switch callCount {
		case 1: // startup pass: failure → notified
			return []proxy.RefreshOutcome{{ConnectionID: "c2", Provider: "openai", Name: "tick", Failed: true, Reason: "x"}}
		case 2: // first tick: recovery → clears throttle
			return []proxy.RefreshOutcome{{ConnectionID: "c2", Provider: "openai", Name: "tick", Refreshed: true}}
		default: // subsequent ticks: failure again → new notification
			return []proxy.RefreshOutcome{{ConnectionID: "c2", Provider: "openai", Name: "tick", Failed: true, Reason: "x"}}
		}
	}
	srv.connRefresher = &countedRefresher

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv.StartConnectionRefresh(ctx)

	// Wait until we see ≥2 notifications (startup failure + post-recovery failure).
	deadline := time.Now().Add(3 * time.Second)
	for notifier.count() < 2 {
		if time.Now().After(deadline) {
			t.Fatalf("ticker did not fire a second notification; got %d", notifier.count())
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
}

type countingFakeRefresher struct {
	outcomes func() []proxy.RefreshOutcome
}

func (c *countingFakeRefresher) RefreshExpiringConnections(ctx context.Context, now time.Time) []proxy.RefreshOutcome {
	return c.outcomes()
}

// TestNotifyStaleNilNotifierIsNoOp exercises the `notifier == nil` guard in
// notifyStale (line 277-279).
func TestNotifyStaleNilNotifierIsNoOp(t *testing.T) {
	srv := NewServer(ServerConfig{})
	// notifierFor returns nil → notifyStale must not panic.
	srv.notifierFor = func(url string) notify.Notifier { return nil }
	srv.notifyStale("https://example.com/hook", proxy.RefreshOutcome{
		ConnectionID: "c1",
		Provider:     "openai",
		Name:         "oauth",
		Reason:       "test",
	})
}

// TestRunConnectionRefreshOnceNoOutcomesIsNoOp verifies no panic when outcomes
// is empty (the for-loop body is never entered).
func TestRunConnectionRefreshOnceNoOutcomes(t *testing.T) {
	ref := &fakeConnRefresher{}
	ref.set(nil)
	srv := newRefreshTestServer(t, ref, &captureNotifier{})
	srv.runConnectionRefreshOnce(time.Now().UTC())
}

// TestRunConnectionRefreshOnceSkipsNeitherRefreshedNorFailed exercises the
// !outcome.Failed branch (line 260): outcome where both Refreshed and Failed
// are false → should be skipped without notifying.
func TestRunConnectionRefreshOnceSkipsNeutralOutcome(t *testing.T) {
	ref := &fakeConnRefresher{}
	notifier := &captureNotifier{}
	srv := newRefreshTestServer(t, ref, notifier)

	// Neither Refreshed nor Failed → the !outcome.Failed continue branch fires.
	ref.set([]proxy.RefreshOutcome{{ConnectionID: "c-neutral", Provider: "openai", Name: "oauth", Refreshed: false, Failed: false}})
	srv.runConnectionRefreshOnce(time.Now().UTC())
	if got := notifier.count(); got != 0 {
		t.Fatalf("neutral outcome should not notify, got %d", got)
	}
}

// TestNotifyStaleNotifyError exercises the error-log branch in notifyStale
// (line 285-287): when Notify returns an error it is logged (not returned).
func TestNotifyStaleNotifyError(t *testing.T) {
	srv := NewServer(ServerConfig{})
	// Inject a notifier that always errors.
	srv.notifierFor = func(url string) notify.Notifier {
		return &errorNotifier{}
	}
	// Must not panic even when Notify returns an error.
	srv.notifyStale("https://example.com/hook", proxy.RefreshOutcome{
		ConnectionID: "c1",
		Provider:     "openai",
		Name:         "oauth",
		Reason:       "test",
	})
}

type errorNotifier struct{}

func (e *errorNotifier) Notify(_ context.Context, _ notify.Event) error {
	return &notifyTestError{}
}

type notifyTestError struct{}

func (e *notifyTestError) Error() string { return "notify test error" }

// TestNotifyStaleDefaultNotifierForLambda exercises the default notifierFor
// lambda in NewServer (server.go line 123): `return notify.NewNotifier(url, nil)`.
// We call notifyStale without overriding notifierFor, with a real httptest server
// as the webhook URL.
func TestNotifyStaleDefaultNotifierForLambda(t *testing.T) {
	var called bool
	webhook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer webhook.Close()

	// Server with real store so runtimeSettings can return NotifyWebhookURL.
	s := newAPITestStore(t)
	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	settings.NotifyWebhookURL = webhook.URL
	settings.NotifyOnReauth = true
	if err := s.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	// NewServer sets the default notifierFor lambda — do NOT override it.
	srv := NewServer(ServerConfig{Store: s})
	srv.notifyStale(webhook.URL, proxy.RefreshOutcome{
		ConnectionID: "c-default",
		Provider:     "openai",
		Name:         "oauth",
		Reason:       "test",
	})
	if !called {
		t.Fatal("default notifierFor should have called the webhook")
	}
}
