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

func TestAdminRoutesRegistered(t *testing.T) {
	st := newTestStore(t)
	sessions := auth.NewSessions(st, time.Hour)
	if _, err := sessions.SeedAdmin("admin", "123456"); err != nil {
		t.Fatalf("SeedAdmin: %v", err)
	}

	client := startServer(t, New(testUIFS(), st))

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

func TestNewWithoutStoreSkipsAdminRoutes(t *testing.T) {
	client := startServer(t, New(testUIFS(), nil))

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
