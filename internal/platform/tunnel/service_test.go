package tunnel

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/bloodf/g0router/internal/store"
)

// fakeRunner is a deterministic Runner used to unit-test the state machine
// WITHOUT spawning any process, downloading any binary, or touching the network.
// It mirrors the platform.Prober fake-injection pattern.
type fakeRunner struct {
	startURL string
	startErr error
	stopErr  error
	started  bool
	stopped  bool
	lastOpts StartOpts
}

func (f *fakeRunner) Start(opts StartOpts) (string, error) {
	f.lastOpts = opts
	if f.startErr != nil {
		return "", f.startErr
	}
	f.started = true
	return f.startURL, nil
}

func (f *fakeRunner) Stop() error {
	if f.stopErr != nil {
		return f.stopErr
	}
	f.stopped = true
	return nil
}

func (f *fakeRunner) Status() (RunnerStatus, error) {
	if f.started {
		return RunnerStatus{Running: true, URL: f.startURL, Status: StatusActive}, nil
	}
	return RunnerStatus{Status: StatusInactive}, nil
}

func newServiceTestStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	secret, err := store.LoadOrCreateSecret(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateSecret: %v", err)
	}
	st, err := store.Open(filepath.Join(dir, "test.db"), secret)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestServiceEnableActivatesAndPersists(t *testing.T) {
	st := newServiceTestStore(t)
	svc := NewService(st)
	fake := &fakeRunner{startURL: "https://brave-tree-1234.trycloudflare.com"}
	svc.SetRunner(TypeCloudflare, fake)

	tn, err := svc.Enable(TypeCloudflare, "", "quick")
	if err != nil {
		t.Fatalf("Enable: %v", err)
	}
	if !tn.IsEnabled || tn.Status != StatusActive {
		t.Fatalf("enable result = %+v", tn)
	}
	if tn.URL != "https://brave-tree-1234.trycloudflare.com" {
		t.Fatalf("url not set: %+v", tn)
	}
	if !fake.started {
		t.Fatalf("runner.Start was not invoked")
	}

	// Persisted.
	stored, err := st.GetTunnel(TypeCloudflare)
	if err != nil {
		t.Fatalf("GetTunnel: %v", err)
	}
	if !stored.IsEnabled || stored.Status != StatusActive || stored.URL != tn.URL {
		t.Fatalf("not persisted: %+v", stored)
	}
}

func TestServiceEnableErrorMarksErrorState(t *testing.T) {
	st := newServiceTestStore(t)
	svc := NewService(st)
	fake := &fakeRunner{startErr: errors.New("spawn failed")}
	svc.SetRunner(TypeCloudflare, fake)

	tn, err := svc.Enable(TypeCloudflare, "tok", "named")
	if err != nil {
		t.Fatalf("Enable should not return a hard error on runner failure: %v", err)
	}
	if tn.Status != StatusError {
		t.Fatalf("status = %q, want error", tn.Status)
	}
	if tn.LastError == "" {
		t.Fatalf("last_error empty on failure")
	}
	if !tn.IsEnabled {
		t.Fatalf("enabled-but-failing tunnel should stay enabled: %+v", tn)
	}

	stored, _ := st.GetTunnel(TypeCloudflare)
	if stored.Status != StatusError || stored.LastError == "" {
		t.Fatalf("error state not persisted: %+v", stored)
	}
}

func TestServiceDisableDeactivates(t *testing.T) {
	st := newServiceTestStore(t)
	svc := NewService(st)
	fake := &fakeRunner{startURL: "https://x.trycloudflare.com"}
	svc.SetRunner(TypeCloudflare, fake)

	if _, err := svc.Enable(TypeCloudflare, "", "quick"); err != nil {
		t.Fatalf("Enable: %v", err)
	}
	tn, err := svc.Disable(TypeCloudflare)
	if err != nil {
		t.Fatalf("Disable: %v", err)
	}
	if tn.IsEnabled || tn.Status != StatusInactive || tn.URL != "" {
		t.Fatalf("disable result = %+v", tn)
	}
	if !fake.stopped {
		t.Fatalf("runner.Stop was not invoked")
	}

	stored, _ := st.GetTunnel(TypeCloudflare)
	if stored.IsEnabled || stored.Status != StatusInactive {
		t.Fatalf("not persisted: %+v", stored)
	}
}

func TestServiceDisableIdempotent(t *testing.T) {
	st := newServiceTestStore(t)
	svc := NewService(st)
	svc.SetRunner(TypeTailscale, &fakeRunner{})

	// Disable a never-enabled tunnel → no error, inactive.
	tn, err := svc.Disable(TypeTailscale)
	if err != nil {
		t.Fatalf("idempotent Disable: %v", err)
	}
	if tn.IsEnabled || tn.Status != StatusInactive {
		t.Fatalf("idempotent disable = %+v", tn)
	}
}

func TestServiceUnknownType(t *testing.T) {
	st := newServiceTestStore(t)
	svc := NewService(st)

	if _, err := svc.Enable("ngrok", "", ""); !errors.Is(err, ErrUnknownType) {
		t.Fatalf("Enable(unknown) err = %v, want ErrUnknownType", err)
	}
	if _, err := svc.Disable("ngrok"); !errors.Is(err, ErrUnknownType) {
		t.Fatalf("Disable(unknown) err = %v, want ErrUnknownType", err)
	}
}

func TestServiceListAlwaysTwoEntries(t *testing.T) {
	st := newServiceTestStore(t)
	svc := NewService(st)
	svc.SetRunner(TypeCloudflare, &fakeRunner{startURL: "https://x.trycloudflare.com"})

	// Nothing enabled yet → still 2 known types.
	list, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 tunnels, got %d: %+v", len(list), list)
	}
	if list[0].Type != TypeCloudflare || list[1].Type != TypeTailscale {
		t.Fatalf("order = %s, %s", list[0].Type, list[1].Type)
	}
	for _, tn := range list {
		if tn.Status != StatusInactive {
			t.Fatalf("default status = %q, want inactive", tn.Status)
		}
	}

	// After enabling cloudflare, the cloudflare entry reflects active state.
	if _, err := svc.Enable(TypeCloudflare, "", "quick"); err != nil {
		t.Fatalf("Enable: %v", err)
	}
	list, _ = svc.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 tunnels after enable, got %d", len(list))
	}
	if list[0].Status != StatusActive || !list[0].IsEnabled {
		t.Fatalf("cloudflare not active: %+v", list[0])
	}
}

func TestServiceHealth(t *testing.T) {
	st := newServiceTestStore(t)
	svc := NewService(st)
	svc.SetRunner(TypeCloudflare, &fakeRunner{startURL: "https://x.trycloudflare.com"})
	svc.SetRunner(TypeTailscale, &fakeRunner{startErr: errors.New("down")})

	// All disabled → healthy.
	if h, err := svc.Health(); err != nil || !h {
		t.Fatalf("Health(all-disabled) = %v, %v; want true", h, err)
	}

	// Enabled + active → healthy.
	if _, err := svc.Enable(TypeCloudflare, "", "quick"); err != nil {
		t.Fatalf("Enable cloudflare: %v", err)
	}
	if h, err := svc.Health(); err != nil || !h {
		t.Fatalf("Health(active) = %v, %v; want true", h, err)
	}

	// Enabled + error → unhealthy.
	if _, err := svc.Enable(TypeTailscale, "", "funnel"); err != nil {
		t.Fatalf("Enable tailscale: %v", err)
	}
	if h, err := svc.Health(); err != nil || h {
		t.Fatalf("Health(error) = %v, %v; want false", h, err)
	}
}
