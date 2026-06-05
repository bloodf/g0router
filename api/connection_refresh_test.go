package api

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/notify"
	"github.com/bloodf/g0router/internal/proxy"
)

type fakeConnRefresher struct {
	mu       sync.Mutex
	outcomes []proxy.RefreshOutcome
}

func (f *fakeConnRefresher) set(o []proxy.RefreshOutcome) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.outcomes = o
}

func (f *fakeConnRefresher) RefreshExpiringConnections(ctx context.Context, now time.Time) []proxy.RefreshOutcome {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.outcomes
}

type captureNotifier struct {
	mu     sync.Mutex
	events []notify.Event
}

func (c *captureNotifier) Notify(ctx context.Context, e notify.Event) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, e)
	return nil
}

func (c *captureNotifier) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.events)
}

func newRefreshTestServer(t *testing.T, ref *fakeConnRefresher, notifier notify.Notifier) *Server {
	t.Helper()
	s := newAPITestStore(t)
	settings, err := s.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	settings.NotifyWebhookURL = "https://example.com/hook"
	settings.NotifyOnReauth = true
	if err := s.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	srv := NewServer(ServerConfig{Store: s})
	srv.connRefresher = ref
	srv.notifierFor = func(url string) notify.Notifier { return notifier }
	return srv
}

func TestRunConnectionRefreshOnceNotifiesOnceThenClearsOnRecovery(t *testing.T) {
	ref := &fakeConnRefresher{}
	notifier := &captureNotifier{}
	srv := newRefreshTestServer(t, ref, notifier)

	now := time.Now().UTC()

	// First tick: connection newly fails -> one notification.
	ref.set([]proxy.RefreshOutcome{{ConnectionID: "c1", Provider: "openai", Name: "oauth", Failed: true, Reason: "invalid_grant"}})
	srv.runConnectionRefreshOnce(now)
	if got := notifier.count(); got != 1 {
		t.Fatalf("after first failure, notifications = %d, want 1", got)
	}

	// Second tick: still failing -> no new notification (throttled).
	srv.runConnectionRefreshOnce(now)
	if got := notifier.count(); got != 1 {
		t.Fatalf("repeat failure should not re-notify, notifications = %d, want 1", got)
	}

	// Third tick: connection recovers (refreshed) -> still no new notification,
	// throttle state cleared.
	ref.set([]proxy.RefreshOutcome{{ConnectionID: "c1", Provider: "openai", Name: "oauth", Refreshed: true}})
	srv.runConnectionRefreshOnce(now)
	if got := notifier.count(); got != 1 {
		t.Fatalf("recovery should not notify, notifications = %d, want 1", got)
	}

	// Fourth tick: fails again after recovery -> notifies again (new episode).
	ref.set([]proxy.RefreshOutcome{{ConnectionID: "c1", Provider: "openai", Name: "oauth", Failed: true, Reason: "invalid_grant"}})
	srv.runConnectionRefreshOnce(now)
	if got := notifier.count(); got != 2 {
		t.Fatalf("new stale episode should notify, notifications = %d, want 2", got)
	}
}

func TestRunConnectionRefreshOnceNoNotifyWhenDisabled(t *testing.T) {
	ref := &fakeConnRefresher{}
	notifier := &captureNotifier{}
	srv := newRefreshTestServer(t, ref, notifier)

	settings, _ := srv.config.Store.GetSettings()
	settings.NotifyOnReauth = false
	if err := srv.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	ref.set([]proxy.RefreshOutcome{{ConnectionID: "c1", Provider: "openai", Name: "oauth", Failed: true, Reason: "x"}})
	srv.runConnectionRefreshOnce(time.Now().UTC())
	if got := notifier.count(); got != 0 {
		t.Fatalf("NotifyOnReauth=false should suppress notifications, got %d", got)
	}
}

func TestStartConnectionRefreshRunsAtStartupAndStopsOnCancel(t *testing.T) {
	ref := &fakeConnRefresher{}
	notifier := &captureNotifier{}
	srv := newRefreshTestServer(t, ref, notifier)
	srv.connectionRefreshInterval = 5 * time.Millisecond
	ref.set([]proxy.RefreshOutcome{{ConnectionID: "c1", Provider: "openai", Name: "oauth", Failed: true, Reason: "x"}})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv.StartConnectionRefresh(ctx)

	deadline := time.Now().Add(2 * time.Second)
	for notifier.count() < 1 {
		if time.Now().After(deadline) {
			t.Fatal("startup refresh pass did not notify")
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
}
