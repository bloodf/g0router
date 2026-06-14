package admin

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/platform/tunnel"
	"github.com/valyala/fasthttp"
)

// fakeTunnelRunner is a deterministic tunnel.Runner for the admin handler tests:
// no process, no network, no download.
type fakeTunnelRunner struct {
	url string
}

func (f *fakeTunnelRunner) Start(opts tunnel.StartOpts) (string, error) {
	return f.url, nil
}
func (f *fakeTunnelRunner) Stop() error { return nil }
func (f *fakeTunnelRunner) Status() (tunnel.RunnerStatus, error) {
	return tunnel.RunnerStatus{Status: tunnel.StatusInactive}, nil
}

func TestListTunnelsReturnsTwoEntries(t *testing.T) {
	env := newTestEnv(t)
	env.handlers.SetTunnelRunner("cloudflare", &fakeTunnelRunner{})
	env.handlers.SetTunnelRunner("tailscale", &fakeTunnelRunner{})

	status, envl := call(t, env.handlers.ListTunnels, "GET", "/api/tunnels", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("list status = %d err = %q", status, errMessage(t, envl))
	}
	list := dataField[[]map[string]any](t, envl)
	if len(list) != 2 {
		t.Fatalf("expected 2 tunnels, got %d: %v", len(list), list)
	}
	if list[0]["type"] != "cloudflare" || list[1]["type"] != "tailscale" {
		t.Fatalf("order = %v, %v", list[0]["type"], list[1]["type"])
	}
	// Canonical 4-field DTO shape.
	for _, tn := range list {
		if _, ok := tn["is_enabled"]; !ok {
			t.Fatalf("missing is_enabled: %v", tn)
		}
		if _, ok := tn["url"]; !ok {
			t.Fatalf("missing url: %v", tn)
		}
		if _, ok := tn["status"]; !ok {
			t.Fatalf("missing status: %v", tn)
		}
	}
}

func TestEnableTunnelActivates(t *testing.T) {
	env := newTestEnv(t)
	env.handlers.SetTunnelRunner("cloudflare", &fakeTunnelRunner{url: "https://brave-tree-1234.trycloudflare.com"})

	status, envl := call(t, env.handlers.EnableTunnel, "POST", "/api/tunnels/cloudflare", "",
		map[string]any{"type": "cloudflare"}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("enable status = %d err = %q", status, errMessage(t, envl))
	}
	tn := dataField[map[string]any](t, envl)
	if tn["is_enabled"] != true || tn["status"] != "active" {
		t.Fatalf("enable result = %v", tn)
	}
	if tn["url"] != "https://brave-tree-1234.trycloudflare.com" {
		t.Fatalf("url = %v", tn["url"])
	}
}

func TestEnableTunnelWithTokenDoesNotLeakSecret(t *testing.T) {
	env := newTestEnv(t)
	env.handlers.SetTunnelRunner("cloudflare", &fakeTunnelRunner{url: "https://x.trycloudflare.com"})

	body := `{"token":"cf-named-token-supersecret","mode":"named"}`
	status, envl := call(t, env.handlers.EnableTunnel, "POST", "/api/tunnels/cloudflare", body,
		map[string]any{"type": "cloudflare"}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("enable status = %d err = %q", status, errMessage(t, envl))
	}
	tn := dataField[map[string]any](t, envl)
	raw, _ := json.Marshal(tn)
	if strings.Contains(string(raw), "supersecret") || strings.Contains(string(raw), "cf-named-token") {
		t.Fatalf("response leaks token: %s", raw)
	}
	if _, ok := tn["token"]; ok {
		t.Fatalf("response has a token field: %v", tn)
	}
	if _, ok := tn["token_enc"]; ok {
		t.Fatalf("response has a token_enc field: %v", tn)
	}

	// The store has the real token (encrypted at rest).
	stored, err := env.store.GetTunnel("cloudflare")
	if err != nil {
		t.Fatalf("GetTunnel: %v", err)
	}
	if stored.Token != "cf-named-token-supersecret" {
		t.Fatalf("stored token = %q", stored.Token)
	}
}

func TestDisableTunnelDeactivates(t *testing.T) {
	env := newTestEnv(t)
	env.handlers.SetTunnelRunner("cloudflare", &fakeTunnelRunner{url: "https://x.trycloudflare.com"})

	if status, envl := call(t, env.handlers.EnableTunnel, "POST", "/api/tunnels/cloudflare", "",
		map[string]any{"type": "cloudflare"}, nil); status != fasthttp.StatusOK {
		t.Fatalf("enable status = %d err = %q", status, errMessage(t, envl))
	}

	status, envl := call(t, env.handlers.DisableTunnel, "DELETE", "/api/tunnels/cloudflare", "",
		map[string]any{"type": "cloudflare"}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("disable status = %d err = %q", status, errMessage(t, envl))
	}
	tn := dataField[map[string]any](t, envl)
	if tn["is_enabled"] != false || tn["status"] != "inactive" {
		t.Fatalf("disable result = %v", tn)
	}
}

func TestEnableUnknownTunnelType400(t *testing.T) {
	env := newTestEnv(t)
	status, envl := call(t, env.handlers.EnableTunnel, "POST", "/api/tunnels/ngrok", "",
		map[string]any{"type": "ngrok"}, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("unknown type status = %d, want 400", status)
	}
	if msg := errMessage(t, envl); msg != "unknown tunnel type" {
		t.Fatalf("error = %q, want 'unknown tunnel type'", msg)
	}

	status, _ = call(t, env.handlers.DisableTunnel, "DELETE", "/api/tunnels/ngrok", "",
		map[string]any{"type": "ngrok"}, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("unknown type disable status = %d, want 400", status)
	}
}

func TestTunnelHealth(t *testing.T) {
	env := newTestEnv(t)
	env.handlers.SetTunnelRunner("cloudflare", &fakeTunnelRunner{url: "https://x.trycloudflare.com"})
	env.handlers.SetTunnelRunner("tailscale", &fakeTunnelRunner{})

	status, envl := call(t, env.handlers.TunnelHealth, "GET", "/api/tunnels/health", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("health status = %d", status)
	}
	data := dataField[map[string]any](t, envl)
	if data["healthy"] != true {
		t.Fatalf("healthy = %v, want true", data["healthy"])
	}
}

func TestEnableTunnelRecordsAudit(t *testing.T) {
	env := newTestEnv(t)
	env.handlers.SetTunnelRunner("cloudflare", &fakeTunnelRunner{url: "https://x.trycloudflare.com"})

	if status, _ := call(t, env.handlers.EnableTunnel, "POST", "/api/tunnels/cloudflare", "",
		map[string]any{"type": "cloudflare"}, nil); status != fasthttp.StatusOK {
		t.Fatalf("enable status = %d", status)
	}

	entries, err := env.store.ListAuditEntries(10)
	if err != nil {
		t.Fatalf("ListAuditEntries: %v", err)
	}
	found := false
	for _, e := range entries {
		if strings.Contains(e.Action, "tunnel") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("no tunnel audit entry recorded: %+v", entries)
	}
}
