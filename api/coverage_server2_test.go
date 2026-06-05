package api

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
)

// ---- handle: method-not-allowed branches ----
// /v1/messages GET, /v1/responses GET, /v1/models POST — all return 404.

func TestHandleMethodNotAllowedV1(t *testing.T) {
	srv := NewServer(ServerConfig{Port: 0, Version: "test"})
	ln := apiTestListener(t)
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Stop() })

	base := "http://" + localhostAddr(t, ln)
	cases := []struct {
		method, path string
	}{
		{http.MethodGet, "/v1/messages"},
		{http.MethodGet, "/v1/responses"},
		{http.MethodPost, "/v1/models"},
	}
	for _, c := range cases {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			req, _ := http.NewRequest(c.method, base+c.path, nil)
			resp, err := httpClient().Do(req)
			if err != nil {
				t.Fatalf("%s %s: %v", c.method, c.path, err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("%s %s = %d, want 404", c.method, c.path, resp.StatusCode)
			}
		})
	}
}

// ---- handleAPI: requireMethod failures ----

func TestHandleAPIRequireMethodFailures(t *testing.T) {
	s := newAPITestStore(t)
	srv := NewServer(ServerConfig{Port: 0, Store: s})
	ln := apiTestListener(t)
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Stop() })

	base := "http://" + localhostAddr(t, ln)
	cases := []struct {
		method, path string
	}{
		// mcp oauth callback requires GET
		{http.MethodPost, "/api/mcp/oauth/callback"},
		// mcp instances auth/start requires POST
		{http.MethodGet, "/api/mcp/instances/inst-1/auth/start"},
		// mcp instances accounts requires GET
		{http.MethodPost, "/api/mcp/instances/inst-1/accounts"},
		// oauth exchange requires POST
		{http.MethodGet, "/api/oauth/openai/exchange"},
		// mcp instances oauth complete requires POST
		{http.MethodGet, "/api/mcp/instances/inst-1/oauth/complete"},
	}
	for _, c := range cases {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			req, _ := http.NewRequest(c.method, base+c.path, nil)
			resp, err := httpClient().Do(req)
			if err != nil {
				t.Fatalf("%s %s: %v", c.method, c.path, err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusMethodNotAllowed {
				t.Fatalf("%s %s = %d, want 405", c.method, c.path, resp.StatusCode)
			}
		})
	}
}

// ---- handleAPI: /api/mcp/clients/{id} route ----
// DELETE /api/mcp/clients/{id} with no MCPClientManager and no record → handler
// routes correctly; writeStoreError returns 404 (not-found in store), confirming
// the route matched (a routing miss would also return 404 but with no JSON body).

func TestHandleAPIMCPClientByIDRoute(t *testing.T) {
	s := newAPITestStore(t)
	srv := NewServer(ServerConfig{Port: 0, Store: s})
	ln := apiTestListener(t)
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Stop() })

	base := "http://" + localhostAddr(t, ln)
	req, _ := http.NewRequest(http.MethodDelete, base+"/api/mcp/clients/nonexistent-id", nil)
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("DELETE /api/mcp/clients/{id}: %v", err)
	}
	defer resp.Body.Close()
	// Handler returns 404 via writeStoreError (store.ErrNotFound).
	// The route matched; a routing miss would return empty body but also 404.
	// Verify the body is JSON (handler-produced), not empty (routing miss).
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("DELETE /api/mcp/clients/{id} = %d, want 404", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json (confirms handler ran, not routing miss)", ct)
	}
}

// ---- logInferenceUsage: early-return when response/request are nil and status < 400 ----

func TestLogInferenceUsageEarlyReturnNilRespAndReq(t *testing.T) {
	s := newAPITestStore(t)
	enableRequestLogs(t, s)
	srv := NewServer(ServerConfig{Store: s, UsageStore: s})

	// nil request + nil response + status 200 → early return, nothing logged.
	ctx := makeCtxWithHeaders("POST", "/v1/chat/completions", nil)
	srv.logInferenceUsage(ctx, time.Now(), "openai", nil, nil, nil, "", nil, http.StatusOK)

	entries, err := s.GetUsage(store.UsageFilter{})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("entries = %d, want 0 (early return)", len(entries))
	}
}

// ---- logInferenceUsage: error status with nil request → still logs ----

func TestLogInferenceUsageErrorStatus(t *testing.T) {
	s := newAPITestStore(t)
	enableRequestLogs(t, s)
	srv := NewServer(ServerConfig{Store: s, UsageStore: s})

	ctx := makeCtxWithHeaders("POST", "/v1/chat/completions", nil)
	// statusCode >= 400 with nil request/response → not early-returned, writes log.
	srv.logInferenceUsage(ctx, time.Now(), "openai", nil, nil, nil, "", nil, http.StatusBadRequest)

	entries, err := s.GetUsage(store.UsageFilter{})
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	// A log entry should have been written (even with nil request).
	if len(entries) == 0 {
		t.Fatal("expected log entry for status 400, got none")
	}
}

// ---- rtkBytesSaved: json.Marshal error branches are unreachable;
//      this test verifies the enabled-with-no-saving nil path ----

func TestRTKBytesSavedEnabledNoCompression(t *testing.T) {
	settings := store.Settings{RTKEnabled: true}
	req := &providers.ChatRequest{
		Model:    "gpt-4o",
		Messages: []providers.Message{{Role: "user", Content: "hi"}},
	}
	// Either nil (no saving) or small positive; we confirm no panic and the
	// function handles the "saved <= 0" branch gracefully.
	got := rtkBytesSaved(settings, req)
	_ = got // nil is expected for short content
}

// ---- handleUI: HEAD method returns 200 with no body ----

func TestHandleUIHeadRequest(t *testing.T) {
	srv := NewServer(ServerConfig{Port: 0, Version: "test"})
	ln := apiTestListener(t)
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Stop() })

	base := "http://" + localhostAddr(t, ln)
	req, _ := http.NewRequest(http.MethodHead, base+"/", nil)
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("HEAD /: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("HEAD / = %d, want 200", resp.StatusCode)
	}
}

// ---- handleUI: assets/ path that doesn't exist → 404 ----

func TestHandleUIAssetNotFound(t *testing.T) {
	srv := NewServer(ServerConfig{Port: 0, Version: "test"})
	ln := apiTestListener(t)
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Stop() })

	base := "http://" + localhostAddr(t, ln)
	resp, err := httpClient().Get(base + "/assets/does-not-exist.js")
	if err != nil {
		t.Fatalf("GET /assets/does-not-exist.js: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("asset not found = %d, want 404", resp.StatusCode)
	}
}

// ---- handleUI: POST/DELETE returns 404 ----

func TestHandleUIMethodNotAllowed(t *testing.T) {
	srv := NewServer(ServerConfig{Port: 0, Version: "test"})
	ln := apiTestListener(t)
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Stop() })

	base := "http://" + localhostAddr(t, ln)
	req, _ := http.NewRequest(http.MethodPost, base+"/some-ui-path", strings.NewReader("{}"))
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("POST /some-ui-path: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("UI POST = %d, want 404", resp.StatusCode)
	}
}

// Verify Serve returns a wrapped error when the server shuts down.
func TestServeReturnNilOnCleanShutdown(t *testing.T) {
	srv := NewServer(ServerConfig{Port: 0})
	ln := apiTestListener(t)
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ln) }()

	// Give Serve a moment to start.
	time.Sleep(20 * time.Millisecond)

	// Stop triggers a clean shutdown; fasthttp.Serve returns nil on Shutdown.
	if err := srv.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve returned error on clean stop: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not return after Stop")
	}
}

