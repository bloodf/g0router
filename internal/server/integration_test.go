package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/admin"
	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/inference"
	"github.com/bloodf/g0router/internal/store"
	"github.com/bloodf/g0router/internal/translation"
	httprouter "github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

// integrationEnv serves the full route surface (OpenAI + admin) through the
// real middleware chain over an in-memory listener, with the OAuth flow
// pointed at a local fake token endpoint.
type integrationEnv struct {
	client *http.Client
	token  string
}

func newIntegrationEnv(t *testing.T) (*integrationEnv, *store.Store) {
	t.Helper()
	st := newTestStore(t)
	sessions := auth.NewSessions(st, time.Hour)
	if _, err := sessions.SeedAdmin("admin", "123456"); err != nil {
		t.Fatalf("SeedAdmin: %v", err)
	}

	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		switch r.PostForm.Get("grant_type") {
		case "authorization_code":
			json.NewEncoder(w).Encode(map[string]any{"access_token": "at-int", "refresh_token": "rt-int", "expires_in": 3600})
		case "refresh_token":
			json.NewEncoder(w).Encode(map[string]any{"access_token": "at-int-2", "refresh_token": "rt-int", "expires_in": 3600})
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(tokenSrv.Close)

	flows := map[string]*auth.OAuthFlow{
		"anthropic": auth.NewOAuthFlow(auth.OAuthConfig{
			Provider:     "anthropic",
			ClientID:     "int-client",
			AuthorizeURL: "https://example.com/authorize",
			TokenURL:     tokenSrv.URL,
			RedirectURI:  "http://localhost/cb",
		}, st, tokenSrv.Client()),
	}

	r := httprouter.New()
	r.NotFound = uiHandler(testUIFS())
	r.GET("/api/health", healthHandler())
	RegisterOpenAIRoutes(r, inference.NewRouter(translation.NewRegistry()), nil, nil, nil, nil, nil, nil)
	RegisterAdminRoutes(r, admin.New(st, sessions, flows))

	srv := &fasthttp.Server{Handler: Chain(r.Handler, RequestIDMiddleware, CORSMiddleware(nil))}
	client := startServer(t, srv)
	return &integrationEnv{client: client}, st
}

func (e *integrationEnv) do(t *testing.T, method, path, body string) (int, map[string]json.RawMessage) {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, "http://server"+path, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if e.token != "" {
		req.Header.Set("Authorization", "Bearer "+e.token)
	}
	resp, err := e.client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	envelope := map[string]json.RawMessage{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &envelope); err != nil {
			t.Fatalf("%s %s: not a JSON envelope: %v\nbody: %s", method, path, err, raw)
		}
	}
	return resp.StatusCode, envelope
}

func decodeData[T any](t *testing.T, envelope map[string]json.RawMessage) T {
	t.Helper()
	var out T
	raw, ok := envelope["data"]
	if !ok {
		t.Fatalf("envelope missing data: %v", envelope)
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode data: %v\nraw: %s", err, raw)
	}
	return out
}

func TestManagementAPIFullFlow(t *testing.T) {
	env, st := newIntegrationEnv(t)

	// 1. Unauthenticated request is rejected.
	status, _ := env.do(t, "GET", "/api/providers", "")
	if status != http.StatusUnauthorized {
		t.Fatalf("unauthenticated providers status = %d", status)
	}

	// 2. Login.
	status, envl := env.do(t, "POST", "/api/auth/login", `{"username":"admin","password":"123456"}`)
	if status != http.StatusOK {
		t.Fatalf("login status = %d", status)
	}
	login := decodeData[map[string]any](t, envl)
	env.token = login["token"].(string)

	// 3. /api/auth/me identifies the user.
	status, envl = env.do(t, "GET", "/api/auth/me", "")
	if status != http.StatusOK {
		t.Fatalf("me status = %d", status)
	}
	if me := decodeData[map[string]any](t, envl); me["username"] != "admin" {
		t.Fatalf("me = %v", me)
	}

	// 4. Settings round trip.
	status, _ = env.do(t, "PUT", "/api/settings", `{"default_model":"gpt-5"}`)
	if status != http.StatusOK {
		t.Fatalf("put settings status = %d", status)
	}
	status, envl = env.do(t, "GET", "/api/settings", "")
	if status != http.StatusOK {
		t.Fatalf("get settings status = %d", status)
	}
	if settings := decodeData[map[string]string](t, envl); settings["default_model"] != "gpt-5" {
		t.Fatalf("settings = %v", settings)
	}

	// 5. Provider CRUD.
	status, envl = env.do(t, "POST", "/api/providers", `{"name":"Anthropic","type":"anthropic","enabled":true}`)
	if status != http.StatusCreated {
		t.Fatalf("create provider status = %d", status)
	}
	provider := decodeData[map[string]any](t, envl)
	providerID := provider["id"].(string)

	status, envl = env.do(t, "GET", "/api/providers", "")
	if status != http.StatusOK {
		t.Fatalf("list providers status = %d", status)
	}
	if list := decodeData[[]map[string]any](t, envl); len(list) != 1 {
		t.Fatalf("providers = %v", list)
	}

	// 6. Connection with an API key: created via HTTP, encrypted at rest,
	// decrypts correctly through the store.
	connBody := fmt.Sprintf(`{"provider_id":%q,"name":"main","kind":"api_key","secret":"sk-int-secret"}`, providerID)
	status, envl = env.do(t, "POST", "/api/connections", connBody)
	if status != http.StatusCreated {
		t.Fatalf("create connection status = %d", status)
	}
	conn := decodeData[map[string]any](t, envl)
	connID := conn["id"].(string)
	if conn["secret_set"] != true {
		t.Fatalf("conn = %v", conn)
	}

	var rawSecret string
	if err := st.DB().QueryRow("SELECT secret_enc FROM connections WHERE id = ?", connID).Scan(&rawSecret); err != nil {
		t.Fatalf("scan secret_enc: %v", err)
	}
	if strings.Contains(rawSecret, "sk-int-secret") {
		t.Fatalf("secret stored in plaintext: %q", rawSecret)
	}
	stored, err := st.GetConnection(connID)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if stored.Secret != "sk-int-secret" {
		t.Fatalf("decrypted secret = %q", stored.Secret)
	}

	// 7. OAuth start → callback → refresh.
	status, envl = env.do(t, "GET", "/api/oauth/anthropic/start", "")
	if status != http.StatusOK {
		t.Fatalf("oauth start status = %d", status)
	}
	start := decodeData[map[string]any](t, envl)
	state := start["state"].(string)
	if !strings.Contains(start["auth_url"].(string), "code_challenge") {
		t.Fatalf("auth_url = %v", start["auth_url"])
	}

	cbBody := fmt.Sprintf(`{"state":%q,"code":"any","provider_id":%q}`, state, providerID)
	status, envl = env.do(t, "POST", "/api/oauth/anthropic/callback", cbBody)
	if status != http.StatusCreated {
		t.Fatalf("oauth callback status = %d", status)
	}
	oauthConn := decodeData[map[string]any](t, envl)
	oauthConnID := oauthConn["id"].(string)

	storedOAuth, err := st.GetConnection(oauthConnID)
	if err != nil {
		t.Fatalf("GetConnection oauth: %v", err)
	}
	if storedOAuth.AccessToken != "at-int" || storedOAuth.RefreshToken != "rt-int" {
		t.Fatalf("oauth tokens = %+v", storedOAuth)
	}

	status, _ = env.do(t, "POST", "/api/connections/"+oauthConnID+"/refresh", "")
	if status != http.StatusOK {
		t.Fatalf("refresh status = %d", status)
	}
	storedOAuth, err = st.GetConnection(oauthConnID)
	if err != nil {
		t.Fatalf("GetConnection after refresh: %v", err)
	}
	if storedOAuth.AccessToken != "at-int-2" {
		t.Fatalf("refreshed token = %q", storedOAuth.AccessToken)
	}

	// 8. Cleanup CRUD: delete connection and provider.
	status, _ = env.do(t, "DELETE", "/api/connections/"+connID, "")
	if status != http.StatusOK {
		t.Fatalf("delete connection status = %d", status)
	}
	status, _ = env.do(t, "DELETE", "/api/providers/"+providerID, "")
	if status != http.StatusOK {
		t.Fatalf("delete provider status = %d", status)
	}

	// 9. Logout revokes the token.
	status, _ = env.do(t, "POST", "/api/auth/logout", "")
	if status != http.StatusOK {
		t.Fatalf("logout status = %d", status)
	}
	status, _ = env.do(t, "GET", "/api/providers", "")
	if status != http.StatusUnauthorized {
		t.Fatalf("providers after logout status = %d", status)
	}
}
