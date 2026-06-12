package server

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/bloodf/g0router/internal/admin"
	"github.com/bloodf/g0router/internal/api"
	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/translation"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

// Compile-time proof that the production credential resolver satisfies the
// handler dependency (w5-pre Finding-3).
var _ api.CredentialRefresher = (*auth.CredentialResolver)(nil)

// fakeResolver is a test seam for KeyResolver.
type fakeResolver struct{ key schemas.Key }

func (f *fakeResolver) ResolveKey(providerID string) (schemas.Key, map[string]string, error) {
	return f.key, nil, nil
}

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

func TestServerGuardWired(t *testing.T) {
	st := newTestStore(t)
	sessions := auth.NewSessions(st, time.Hour)
	if _, err := sessions.SeedAdmin("admin", "123456"); err != nil {
		t.Fatalf("SeedAdmin: %v", err)
	}

	client := startServer(t, New(testUIFS(), st, nil))

	// Remote /api/settings without a session is rejected by the central guard.
	req, err := http.NewRequest("GET", "http://server/api/settings", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Host", "remote.example")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("settings: %v", err)
	}
	body := readBody(t, resp)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("remote /api/settings status = %d body = %s", resp.StatusCode, body)
	}
	if !strings.Contains(body, `"error"`) {
		t.Fatalf("remote /api/settings body = %s, want envelope", body)
	}

	// Remote /v1 without an API key is blocked by the central guard.
	resp2, err := client.Post("http://server/v1/chat/completions", "application/json",
		strings.NewReader(`{"model":"gpt-4"}`))
	if err != nil {
		t.Fatalf("/v1 chat: %v", err)
	}
	body2 := readBody(t, resp2)
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("remote /v1 without key status = %d, want 401: %s", resp2.StatusCode, body2)
	}
	if !strings.Contains(body2, `"error"`) || !strings.Contains(body2, "API key required for remote API access") {
		t.Fatalf("remote /v1 without key body = %s, want API key error", body2)
	}

	// Remote /v1 with a valid API key reaches the LLM handler (no management envelope).
	rec, err := st.CreateAPIKey("server-test")
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	apiKey := rec.Key
	req2, err := http.NewRequest("POST", "http://server/v1/chat/completions", strings.NewReader(`{"model":"gpt-4"}`))
	if err != nil {
		t.Fatalf("new /v1 request: %v", err)
	}
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+apiKey)
	resp3, err := client.Do(req2)
	if err != nil {
		t.Fatalf("/v1 chat with key: %v", err)
	}
	body3 := readBody(t, resp3)
	if resp3.StatusCode == http.StatusNotFound {
		t.Fatalf("/v1/chat/completions returned 404: %s", body3)
	}
	// The guard must not produce the management API envelope; the LLM handler does its own auth.
	if strings.Contains(body3, `"data"`) && strings.Contains(body3, `"error"`) {
		t.Fatalf("/v1/chat/completions with key was blocked by guard: %s", body3)
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

// TestServerWiresKeyResolver verifies that SetKeyResolver feeds keys through
// the router. This test passes only when server.New wires the credential
// resolver into the inference router.
func TestServerWiresKeyResolver(t *testing.T) {
	router := inference.NewRouter(translation.NewRegistry())
	resolver := &fakeResolver{key: schemas.Key{Value: "wired-key"}}
	router.SetKeyResolver(resolver)

	_, key, err := router.Resolve("gpt-4")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if key.Value != "wired-key" {
		t.Errorf("key.Value = %q, want wired-key", key.Value)
	}
}

// TestServerFlowsIncludeGeminiXai verifies that the flows map built by
// server.New contains gemini and xai OAuth flows, so OAuthStart returns an
// auth URL instead of a "provider not supported" error.
func TestServerFlowsIncludeGeminiXai(t *testing.T) {
	st := newTestStore(t)
	sessions := auth.NewSessions(st, time.Hour)
	flows := map[string]*auth.OAuthFlow{
		"gemini": auth.NewOAuthFlow(auth.GeminiOAuth(), st, nil),
		"xai":    auth.NewOAuthFlow(auth.XaiOAuth(), st, nil),
	}
	h := admin.New(st, sessions, flows)

	for _, provider := range []string{"gemini", "xai"} {
		var ctx fasthttp.RequestCtx
		ctx.SetUserValue("provider", provider)
		h.OAuthStart(&ctx)

		if ctx.Response.StatusCode() == fasthttp.StatusInternalServerError {
			t.Errorf("%s: status = 500, want non-500", provider)
		}
		body := string(ctx.Response.Body())
		if strings.Contains(body, "provider not supported") {
			t.Errorf("%s: body contains 'provider not supported'", provider)
		}
	}
}

// TestProductionComboDispatcherBridges builds the exact construction chain
// from server.New and asserts that newComboDispatcher correctly adapts the
// inference.ComboEngine to the api.ComboDispatcher interface.
func TestProductionComboDispatcherBridges(t *testing.T) {
	st := newTestStore(t)

	// The account runner resolves "m1"/"m2" to the "openai" provider, so the
	// seeded connection must carry that provider id. Seed the provider directly
	// so its id is deterministic.
	now := time.Now().Unix()
	if _, err := st.DB().Exec(
		"INSERT INTO providers (id, name, type, base_url, enabled, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"openai", "OpenAI", "openai", "", 1, now, now,
	); err != nil {
		t.Fatalf("insert provider: %v", err)
	}
	if err := st.SetSetting("providerStrategies", `{"openai":{"fallbackStrategy":"fill-first"}}`); err != nil {
		t.Fatalf("SetSetting providerStrategies: %v", err)
	}

	conn := &store.Connection{
		ProviderID: "openai",
		Name:       "main",
		Kind:       "api_key",
		Secret:     "sk-secret",
	}
	if err := st.CreateConnection(conn); err != nil {
		t.Fatalf("CreateConnection: %v", err)
	}

	if err := st.CreateCombo(&store.Combo{Name: "best", Models: []string{"m1", "m2"}}); err != nil {
		t.Fatalf("CreateCombo: %v", err)
	}

	// Exact chain constructed by server.New.
	cd := inference.NewCooldownEngine(st, time.Now)
	sel := inference.NewSelectionEngine(st, st, cd, time.Now)
	runner := inference.NewAccountRunner(sel)
	comboEngine := inference.NewComboEngine(st, st, runner, time.Now, func(time.Duration) {})
	disp := newComboDispatcher(st, comboEngine)

	// (a) Combo identity.
	if !disp.IsCombo("best") {
		t.Error("IsCombo(best) = false, want true")
	}
	if disp.IsCombo("m1") {
		t.Error("IsCombo(m1) = true, want false")
	}

	// (b) ExecuteCombo invokes fn with the seeded connection's ID and credential.
	var gotConnID, gotCredential string
	if err := disp.ExecuteCombo("best", func(model, connID, credential string) (inference.Verdict, error) {
		gotConnID = connID
		gotCredential = credential
		return inference.VerdictUnknown, nil
	}); err != nil {
		t.Fatalf("ExecuteCombo best: %v", err)
	}
	if gotConnID != conn.ID {
		t.Errorf("fn connID = %q, want %q", gotConnID, conn.ID)
	}
	if gotCredential == "" {
		t.Error("fn credential empty, want non-empty")
	}

	// (c) A quota-style verdict error for m1 falls through to m2.
	var seenModels []string
	err := disp.ExecuteCombo("best", func(model, connID, credential string) (inference.Verdict, error) {
		seenModels = append(seenModels, model)
		if model == "m1" {
			return inference.VerdictRateLimit, errors.New("rate limited")
		}
		return inference.VerdictUnknown, nil
	})
	if err != nil {
		t.Fatalf("ExecuteCombo fallback: %v", err)
	}
	if len(seenModels) != 2 || seenModels[0] != "m1" || seenModels[1] != "m2" {
		t.Errorf("seenModels = %v, want [m1 m2]", seenModels)
	}
}
