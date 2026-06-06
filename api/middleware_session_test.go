package api

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

func createTestSession(t *testing.T, s *store.Store) (rawToken string, userID int64) {
	t.Helper()
	user, err := s.CreateDashboardUser("admin", "password123", "Admin", "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	userID, _ = strconv.ParseInt(user.ID, 10, 64)
	rawToken = "testtoken123"
	if err := s.CreateDashboardSession(userID, rawToken, "test-agent", "127.0.0.1", time.Now().UTC().Add(7*24*time.Hour)); err != nil {
		t.Fatalf("create session: %v", err)
	}
	return rawToken, userID
}

func createExpiredTestSession(t *testing.T, s *store.Store) string {
	t.Helper()
	user, err := s.CreateDashboardUser("admin2", "password123", "Admin2", "admin")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	userID, _ := strconv.ParseInt(user.ID, 10, 64)
	rawToken := "expiredtoken123"
	if err := s.CreateDashboardSession(userID, rawToken, "test-agent", "127.0.0.1", time.Now().UTC().Add(-1*time.Hour)); err != nil {
		t.Fatalf("create expired session: %v", err)
	}
	return rawToken
}

func setRequireLogin(t *testing.T, srv *Server, s *store.Store, value bool) {
	t.Helper()
	settings := srv.runtimeSettings()
	settings.RequireLogin = value
	if value {
		users, err := s.ListDashboardUsers()
		if err != nil {
			t.Fatalf("ListDashboardUsers: %v", err)
		}
		if len(users) == 0 {
			if _, err := s.CreateDashboardUser("admin", "password123", "Admin", "admin"); err != nil {
				t.Fatalf("CreateDashboardUser: %v", err)
			}
		}
	}
	if err := srv.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
}

func setTrustProxyHeaders(t *testing.T, srv *Server, value bool) {
	t.Helper()
	settings := srv.runtimeSettings()
	settings.TrustProxyHeaders = value
	if err := srv.UpdateSettings(settings); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
}

func TestSessionCookieGrantsAPIAccessWhenRequireLoginTrue(t *testing.T) {
	s := newAPITestStore(t)
	rawToken, _ := createTestSession(t, s)
	srv, baseURL := startTestServer(t, ServerConfig{Port: 0, Version: "test", Store: s, RequireAPIKey: false})
	setRequireLogin(t, srv, s, true)

	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/settings", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Cookie", "g0router_session="+rawToken)
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("GET /api/settings: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestBearerKeyWorksOnAPIWithRequireLoginTrue(t *testing.T) {
	s := newAPITestStore(t)
	srv, baseURL := startTestServer(t, ServerConfig{
		Port:          0,
		Version:       "test",
		Store:         s,
		RequireAPIKey: true,
		APIKeySecret:  "test-secret",
		APIKeyValidator: fakeAPIKeyValidator{
			validKeys: map[string]bool{"g0r_valid": true},
		},
	})
	setRequireLogin(t, srv, s, true)

	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/settings", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer g0r_valid")
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("GET /api/settings: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestExemptRoutesReachableWithoutAuthWhenRequireLoginTrue(t *testing.T) {
	s := newAPITestStore(t)
	srv, baseURL := startTestServer(t, ServerConfig{Port: 0, Version: "test", Store: s, RequireAPIKey: false})
	setRequireLogin(t, srv, s, true)

	tests := []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodPost, path: "/api/auth/setup", body: `{"username":"admin","password":"password123","display_name":"Admin"}`},
		{method: http.MethodPost, path: "/api/auth/login", body: `{"username":"admin","password":"password123"}`},
		{method: http.MethodGet, path: "/api/auth/status", body: ""},
		{method: http.MethodGet, path: "/api/oauth/callback", body: ""},
		{method: http.MethodGet, path: "/api/mcp/oauth/callback", body: ""},
	}

	for _, tc := range tests {
		var body io.Reader
		if tc.body != "" {
			body = strings.NewReader(tc.body)
		}
		req, err := http.NewRequest(tc.method, baseURL+tc.path, body)
		if err != nil {
			t.Fatalf("NewRequest %s %s: %v", tc.method, tc.path, err)
		}
		if tc.body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := httpClient().Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", tc.method, tc.path, err)
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusUnauthorized {
			t.Fatalf("%s %s status = %d, want exempt from auth", tc.method, tc.path, resp.StatusCode)
		}
	}
}

func TestGarbageSessionCookieReturns401WhenRequireLoginTrue(t *testing.T) {
	s := newAPITestStore(t)
	srv, baseURL := startTestServer(t, ServerConfig{Port: 0, Version: "test", Store: s, RequireAPIKey: false})
	setRequireLogin(t, srv, s, true)

	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/settings", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Cookie", "g0router_session=notavalidtoken")
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("GET /api/settings: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestExpiredSessionCookieReturns401WhenRequireLoginTrue(t *testing.T) {
	s := newAPITestStore(t)
	rawToken := createExpiredTestSession(t, s)
	srv, baseURL := startTestServer(t, ServerConfig{Port: 0, Version: "test", Store: s, RequireAPIKey: false})
	setRequireLogin(t, srv, s, true)

	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/settings", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Cookie", "g0router_session="+rawToken)
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("GET /api/settings: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestCookieMutatingRequestMismatchedOriginReturns403(t *testing.T) {
	s := newAPITestStore(t)
	rawToken, _ := createTestSession(t, s)
	srv, baseURL := startTestServer(t, ServerConfig{Port: 0, Version: "test", Store: s, RequireAPIKey: false})
	setRequireLogin(t, srv, s, true)

	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/connections", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Cookie", "g0router_session="+rawToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://evil.example")
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("POST /api/connections: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", resp.StatusCode)
	}
}

func TestBearerMutatingRequestSkipsCSRFCheck(t *testing.T) {
	s := newAPITestStore(t)
	srv, baseURL := startTestServer(t, ServerConfig{
		Port:          0,
		Version:       "test",
		Store:         s,
		RequireAPIKey: true,
		APIKeySecret:  "test-secret",
		APIKeyValidator: fakeAPIKeyValidator{
			validKeys: map[string]bool{"g0r_valid": true},
		},
	})
	setRequireLogin(t, srv, s, true)

	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/connections", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer g0r_valid")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://evil.example")
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("POST /api/connections: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden {
		t.Fatal("bearer request should skip CSRF check")
	}
}

func TestLogoutInvalidatesSession(t *testing.T) {
	s := newAPITestStore(t)
	rawToken, _ := createTestSession(t, s)
	srv, baseURL := startTestServer(t, ServerConfig{Port: 0, Version: "test", Store: s, RequireAPIKey: false})
	setRequireLogin(t, srv, s, true)

	// Verify session works
	req, _ := http.NewRequest(http.MethodGet, baseURL+"/api/settings", nil)
	req.Header.Set("Cookie", "g0router_session="+rawToken)
	resp, _ := httpClient().Do(req)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("pre-logout status = %d, want 200", resp.StatusCode)
	}

	// Logout
	logoutReq, _ := http.NewRequest(http.MethodPost, baseURL+"/api/auth/logout", nil)
	logoutReq.Header.Set("Cookie", "g0router_session="+rawToken)
	logoutReq.Header.Set("Origin", baseURL)
	logoutResp, _ := httpClient().Do(logoutReq)
	logoutResp.Body.Close()
	if logoutResp.StatusCode != http.StatusNoContent {
		t.Fatalf("logout status = %d, want 204", logoutResp.StatusCode)
	}

	// Verify session no longer works
	req2, _ := http.NewRequest(http.MethodGet, baseURL+"/api/settings", nil)
	req2.Header.Set("Cookie", "g0router_session="+rawToken)
	resp2, _ := httpClient().Do(req2)
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("post-logout status = %d, want 401", resp2.StatusCode)
	}
}

func TestClientIPWithTrustProxyHeaders(t *testing.T) {
	s := newAPITestStore(t)
	srv := NewServer(ServerConfig{Store: s})

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")
	ctx.SetRemoteAddr(&net.TCPAddr{IP: net.ParseIP("127.0.0.1")})

	// Without trust_proxy_headers
	ip := srv.clientIP(ctx)
	if ip != "127.0.0.1" {
		t.Fatalf("ip = %q, want 127.0.0.1", ip)
	}

	// With trust_proxy_headers
	setTrustProxyHeaders(t, srv, true)
	ip = srv.clientIP(ctx)
	if ip != "1.2.3.4" {
		t.Fatalf("ip = %q, want 1.2.3.4", ip)
	}
}

func TestClientIPHonoredInAuthSetupSession(t *testing.T) {
	s := newAPITestStore(t)
	srv, baseURL := startTestServer(t, ServerConfig{Port: 0, Version: "test", Store: s, RequireAPIKey: false})
	setTrustProxyHeaders(t, srv, true)

	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/auth/setup",
		strings.NewReader(`{"username":"admin","password":"password123","display_name":"Admin"}`))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "9.8.7.6")
	resp, err := httpClient().Do(req)
	if err != nil {
		t.Fatalf("POST /api/auth/setup: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}

	// Find the session and verify IP
	var rawToken string
	for _, c := range resp.Cookies() {
		if c.Name == "g0router_session" {
			rawToken = c.Value
			break
		}
	}
	if rawToken == "" {
		t.Fatal("expected session cookie")
	}
	h := sha256.Sum256([]byte(rawToken))
	tokenHash := hex.EncodeToString(h[:])
	session, err := s.GetDashboardSessionByTokenHash(tokenHash)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if session.IP != "9.8.7.6" {
		t.Fatalf("session ip = %q, want 9.8.7.6", session.IP)
	}
}
