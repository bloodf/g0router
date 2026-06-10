package server

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

func newTestStore(t *testing.T) *store.Store {
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

// startServer runs the fasthttp server on an in-memory listener and
// returns an http.Client that talks to it.
func startServer(t *testing.T, srv *fasthttp.Server) *http.Client {
	t.Helper()
	ln := fasthttputil.NewInmemoryListener()
	t.Cleanup(func() { ln.Close() })
	go srv.Serve(ln)

	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return ln.Dial()
			},
		},
	}
}

func testUIFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<div id=\"root\"></div>")},
	}
}

func TestMessagesRouteRegistered(t *testing.T) {
	client := startServer(t, New(testUIFS(), nil, nil))
	resp, err := client.Post("http://server/v1/messages", "application/json", strings.NewReader(`{"model":"claude-opus-4"}`))
	if err != nil {
		t.Fatalf("post /v1/messages: %v", err)
	}
	body := readBody(t, resp)
	// With no store the request may fail upstream, but the route must exist (not 404).
	if resp.StatusCode == http.StatusNotFound {
		t.Fatalf("/v1/messages returned 404: %s", body)
	}
}

func TestAdminRoutesRegistered(t *testing.T) {
	st := newTestStore(t)
	sessions := auth.NewSessions(st, time.Hour)
	if _, err := sessions.SeedAdmin("admin", "123456"); err != nil {
		t.Fatalf("SeedAdmin: %v", err)
	}

	client := startServer(t, New(testUIFS(), st, nil))

	// Protected routes reject unauthenticated requests with the envelope, not the UI fallback.
	for _, route := range []struct{ method, path string }{
		{"GET", "/api/settings"},
		{"PUT", "/api/settings"},
		{"GET", "/api/auth/me"},
		{"GET", "/api/providers"},
		{"POST", "/api/providers"},
		{"GET", "/api/connections"},
		{"POST", "/api/connections"},
		{"GET", "/api/oauth/anthropic/start"},
	} {
		req, err := http.NewRequest(route.method, "http://server"+route.path, strings.NewReader("{}"))
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", route.method, route.path, err)
		}
		body := readBody(t, resp)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("%s %s status = %d body = %s", route.method, route.path, resp.StatusCode, body)
		}
		if !strings.Contains(body, `"error"`) {
			t.Fatalf("%s %s body = %s, want envelope", route.method, route.path, body)
		}
	}

	// Login is public and succeeds.
	resp, err := client.Post("http://server/api/auth/login", "application/json",
		strings.NewReader(`{"username":"admin","password":"123456"}`))
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	loginBody := readBody(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d body = %s", resp.StatusCode, loginBody)
	}
	var login struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(loginBody), &login); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	if login.Data.Token == "" {
		t.Fatalf("no token in %s", loginBody)
	}

	// Token unlocks a protected route.
	req, err := http.NewRequest("GET", "http://server/api/settings", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+login.Data.Token)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	if body := readBody(t, resp); resp.StatusCode != http.StatusOK {
		t.Fatalf("settings status = %d body = %s", resp.StatusCode, body)
	}

	// Health stays public; UI fallback still works.
	resp, err = client.Get("http://server/api/health")
	if err != nil {
		t.Fatalf("health: %v", err)
	}
	if readBody(t, resp); resp.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d", resp.StatusCode)
	}
	resp, err = client.Get("http://server/dashboard")
	if err != nil {
		t.Fatalf("ui fallback: %v", err)
	}
	if body := readBody(t, resp); resp.StatusCode != http.StatusOK || !strings.Contains(body, "root") {
		t.Fatalf("ui fallback status = %d body = %s", resp.StatusCode, body)
	}
}

func TestCORSMiddleware(t *testing.T) {
	st := newTestStore(t)
	sessions := auth.NewSessions(st, time.Hour)
	if _, err := sessions.SeedAdmin("admin", "123456"); err != nil {
		t.Fatalf("SeedAdmin: %v", err)
	}

	client := startServer(t, New(testUIFS(), st, []string{"http://localhost:5173"}))

	// (a) Evil origin receives no CORS headers.
	req, err := http.NewRequest("GET", "http://server/api/health", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Origin", "https://evil.example")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("evil origin: %v", err)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("evil origin got ACAO = %q", got)
	}
	if got := resp.Header.Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("evil origin got ACAC = %q", got)
	}

	// (b) Allowlisted origin receives both.
	req2, err := http.NewRequest("GET", "http://server/api/health", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req2.Header.Set("Origin", "http://localhost:5173")
	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("allowlisted origin: %v", err)
	}
	if got := resp2.Header.Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("allowlisted origin got ACAO = %q", got)
	}
	if got := resp2.Header.Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("allowlisted origin got ACAC = %q", got)
	}

	// Same-origin (no Origin header) still works without CORS headers.
	req3, err := http.NewRequest("GET", "http://server/api/health", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp3, err := client.Do(req3)
	if err != nil {
		t.Fatalf("no origin: %v", err)
	}
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("no origin status = %d", resp3.StatusCode)
	}
}

func TestNewWithoutStoreSkipsAdminRoutes(t *testing.T) {
	client := startServer(t, New(testUIFS(), nil, nil))

	resp, err := client.Post("http://server/api/auth/login", "application/json",
		strings.NewReader(`{"username":"admin","password":"123456"}`))
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	// Without a store the route doesn't exist; the SPA fallback answers.
	body := readBody(t, resp)
	if resp.StatusCode == http.StatusOK && strings.Contains(body, `"token"`) {
		t.Fatalf("admin routes active without store: %s", body)
	}
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		sb.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return sb.String()
}
