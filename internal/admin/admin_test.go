package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/auth"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type testEnv struct {
	store    *store.Store
	sessions *auth.Sessions
	handlers *Handlers
}

func newTestEnv(t *testing.T) *testEnv {
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

	sessions := auth.NewSessions(st, time.Hour)
	if _, err := sessions.SeedAdmin("admin", "123456"); err != nil {
		t.Fatalf("SeedAdmin: %v", err)
	}
	return &testEnv{store: st, sessions: sessions, handlers: New(st, sessions, nil)}
}

func (e *testEnv) withOAuth(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		switch r.PostForm.Get("grant_type") {
		case "authorization_code":
			json.NewEncoder(w).Encode(map[string]any{"access_token": "at-cb", "refresh_token": "rt-cb", "expires_in": 3600})
		case "refresh_token":
			json.NewEncoder(w).Encode(map[string]any{"access_token": "at-refreshed", "refresh_token": "rt-cb", "expires_in": 3600})
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(srv.Close)

	flow := auth.NewOAuthFlow(auth.OAuthConfig{
		Provider:     "anthropic",
		ClientID:     "client-x",
		AuthorizeURL: "https://example.com/authorize",
		TokenURL:     srv.URL,
		RedirectURI:  "http://localhost/cb",
	}, e.store, srv.Client())
	e.handlers = New(e.store, e.sessions, map[string]*auth.OAuthFlow{"anthropic": flow})
	return srv
}

// call invokes a handler with the given method, body, and user values,
// then decodes the {data, error} envelope.
func call(t *testing.T, h fasthttp.RequestHandler, method, uri, body string, userValues map[string]any, headers map[string]string) (int, map[string]json.RawMessage) {
	t.Helper()
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(method)
	ctx.Request.SetRequestURI(uri)
	if body != "" {
		ctx.Request.SetBody([]byte(body))
	}
	for k, v := range userValues {
		ctx.SetUserValue(k, v)
	}
	for k, v := range headers {
		ctx.Request.Header.Set(k, v)
	}
	h(&ctx)

	envelope := map[string]json.RawMessage{}
	if len(ctx.Response.Body()) > 0 {
		if err := json.Unmarshal(ctx.Response.Body(), &envelope); err != nil {
			t.Fatalf("response is not a JSON envelope: %v\nbody: %s", err, ctx.Response.Body())
		}
	}
	return ctx.Response.StatusCode(), envelope
}

func dataField[T any](t *testing.T, envelope map[string]json.RawMessage) T {
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

func errMessage(t *testing.T, envelope map[string]json.RawMessage) string {
	t.Helper()
	raw, ok := envelope["error"]
	if !ok || string(raw) == "null" {
		return ""
	}
	var e struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &e); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	return e.Message
}

func loginToken(t *testing.T, env *testEnv) string {
	t.Helper()
	status, envl := call(t, env.handlers.Login, "POST", "/api/auth/login", `{"username":"admin","password":"123456"}`, nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("login status = %d, err = %q", status, errMessage(t, envl))
	}
	data := dataField[map[string]any](t, envl)
	token, _ := data["token"].(string)
	if token == "" {
		t.Fatalf("login data = %v", data)
	}
	return token
}

func TestLoginSuccessAndEnvelope(t *testing.T) {
	env := newTestEnv(t)

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.SetRequestURI("/api/auth/login")
	ctx.Request.SetBody([]byte(`{"username":"admin","password":"123456"}`))
	env.handlers.Login(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d body = %s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	body := string(ctx.Response.Body())
	if !strings.Contains(body, `"data"`) || !strings.Contains(body, `"error":null`) {
		t.Fatalf("not a {data, error} envelope: %s", body)
	}
	if !strings.Contains(body, `"token"`) || !strings.Contains(body, `"username":"admin"`) {
		t.Fatalf("login body = %s", body)
	}
	if ct := string(ctx.Response.Header.ContentType()); ct != "application/json" {
		t.Fatalf("content type = %q", ct)
	}
	setCookie := string(ctx.Response.Header.Peek("Set-Cookie"))
	if !strings.Contains(setCookie, sessionCookieName+"=") {
		t.Fatalf("Set-Cookie = %q", setCookie)
	}
}

func TestLoginFailures(t *testing.T) {
	env := newTestEnv(t)

	status, envl := call(t, env.handlers.Login, "POST", "/api/auth/login", `{"username":"admin","password":"wrong"}`, nil, nil)
	if status != fasthttp.StatusUnauthorized {
		t.Fatalf("wrong password status = %d", status)
	}
	if errMessage(t, envl) == "" {
		t.Fatal("expected error message")
	}

	status, _ = call(t, env.handlers.Login, "POST", "/api/auth/login", `not-json`, nil, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("malformed body status = %d", status)
	}

	status, _ = call(t, env.handlers.Login, "POST", "/api/auth/login", `{"username":"","password":""}`, nil, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("empty credentials status = %d", status)
	}
}

func TestRequireSession(t *testing.T) {
	env := newTestEnv(t)
	token := loginToken(t, env)

	protected := env.handlers.RequireSession(func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.SetBodyString(`{"data":"ok","error":null}`)
	})

	// No token → 401.
	status, _ := call(t, protected, "GET", "/api/settings", "", nil, nil)
	if status != fasthttp.StatusUnauthorized {
		t.Fatalf("no token status = %d", status)
	}

	// Bearer token → 200.
	status, _ = call(t, protected, "GET", "/api/settings", "", nil, map[string]string{"Authorization": "Bearer " + token})
	if status != fasthttp.StatusOK {
		t.Fatalf("bearer status = %d", status)
	}

	// Cookie → 200.
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/api/settings")
	ctx.Request.Header.SetCookie(sessionCookieName, token)
	protected(&ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("cookie status = %d", ctx.Response.StatusCode())
	}

	// Garbage token → 401.
	status, _ = call(t, protected, "GET", "/api/settings", "", nil, map[string]string{"Authorization": "Bearer nope"})
	if status != fasthttp.StatusUnauthorized {
		t.Fatalf("garbage token status = %d", status)
	}
}

func TestMeAndLogout(t *testing.T) {
	env := newTestEnv(t)
	token := loginToken(t, env)
	authHeader := map[string]string{"Authorization": "Bearer " + token}

	status, envl := call(t, env.handlers.RequireSession(env.handlers.Me), "GET", "/api/auth/me", "", nil, authHeader)
	if status != fasthttp.StatusOK {
		t.Fatalf("me status = %d", status)
	}
	me := dataField[map[string]any](t, envl)
	if me["username"] != "admin" {
		t.Fatalf("me = %v", me)
	}

	status, _ = call(t, env.handlers.RequireSession(env.handlers.Logout), "POST", "/api/auth/logout", "", nil, authHeader)
	if status != fasthttp.StatusOK {
		t.Fatalf("logout status = %d", status)
	}

	status, _ = call(t, env.handlers.RequireSession(env.handlers.Me), "GET", "/api/auth/me", "", nil, authHeader)
	if status != fasthttp.StatusUnauthorized {
		t.Fatalf("me after logout status = %d", status)
	}
}

func TestSettingsGetPut(t *testing.T) {
	env := newTestEnv(t)

	status, envl := call(t, env.handlers.GetSettings, "GET", "/api/settings", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("get status = %d", status)
	}
	settings := dataField[map[string]string](t, envl)
	if len(settings) != 0 {
		t.Fatalf("initial settings = %v", settings)
	}

	status, _ = call(t, env.handlers.PutSettings, "PUT", "/api/settings", `{"theme":"dark","log_level":"debug"}`, nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("put status = %d", status)
	}

	status, envl = call(t, env.handlers.GetSettings, "GET", "/api/settings", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("get status = %d", status)
	}
	settings = dataField[map[string]string](t, envl)
	if settings["theme"] != "dark" || settings["log_level"] != "debug" {
		t.Fatalf("settings = %v", settings)
	}

	status, _ = call(t, env.handlers.PutSettings, "PUT", "/api/settings", `not-json`, nil, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("bad body status = %d", status)
	}
}

func TestProviderCRUDHandlers(t *testing.T) {
	env := newTestEnv(t)

	// Create.
	status, envl := call(t, env.handlers.CreateProvider, "POST", "/api/providers",
		`{"name":"OpenAI","type":"openai","base_url":"https://api.openai.com/v1","enabled":true}`, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create status = %d err = %q", status, errMessage(t, envl))
	}
	created := dataField[map[string]any](t, envl)
	id, _ := created["id"].(string)
	if id == "" || created["name"] != "OpenAI" || created["base_url"] != "https://api.openai.com/v1" {
		t.Fatalf("created = %v", created)
	}

	// Validation.
	status, _ = call(t, env.handlers.CreateProvider, "POST", "/api/providers", `{"name":"","type":""}`, nil, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("invalid create status = %d", status)
	}

	// List.
	status, envl = call(t, env.handlers.ListProviders, "GET", "/api/providers", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("list status = %d", status)
	}
	list := dataField[[]map[string]any](t, envl)
	if len(list) != 1 {
		t.Fatalf("list = %v", list)
	}

	// Update.
	status, envl = call(t, env.handlers.UpdateProvider, "PUT", "/api/providers/"+id,
		`{"name":"OpenAI EU","type":"openai","base_url":"https://eu.api.openai.com/v1","enabled":false}`,
		map[string]any{"id": id}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("update status = %d err = %q", status, errMessage(t, envl))
	}
	updated := dataField[map[string]any](t, envl)
	if updated["name"] != "OpenAI EU" || updated["enabled"] != false {
		t.Fatalf("updated = %v", updated)
	}

	// Update missing → 404.
	status, _ = call(t, env.handlers.UpdateProvider, "PUT", "/api/providers/missing",
		`{"name":"X","type":"openai"}`, map[string]any{"id": "missing"}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("update missing status = %d", status)
	}

	// Delete.
	status, _ = call(t, env.handlers.DeleteProvider, "DELETE", "/api/providers/"+id, "", map[string]any{"id": id}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("delete status = %d", status)
	}
	status, _ = call(t, env.handlers.DeleteProvider, "DELETE", "/api/providers/"+id, "", map[string]any{"id": id}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("delete missing status = %d", status)
	}
}

func TestConnectionCRUDHandlersMaskSecrets(t *testing.T) {
	env := newTestEnv(t)

	status, envl := call(t, env.handlers.CreateProvider, "POST", "/api/providers",
		`{"name":"Anthropic","type":"anthropic","enabled":true}`, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create provider status = %d", status)
	}
	providerID := dataField[map[string]any](t, envl)["id"].(string)

	// Create with unknown provider → 400.
	status, _ = call(t, env.handlers.CreateConnection, "POST", "/api/connections",
		`{"provider_id":"nope","name":"k","kind":"api_key","secret":"sk-x"}`, nil, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("unknown provider status = %d", status)
	}

	// Create.
	body := fmt.Sprintf(`{"provider_id":%q,"name":"main key","kind":"api_key","secret":"sk-ant-supersecret"}`, providerID)
	status, envl = call(t, env.handlers.CreateConnection, "POST", "/api/connections", body, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create status = %d err = %q", status, errMessage(t, envl))
	}
	created := dataField[map[string]any](t, envl)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatalf("created = %v", created)
	}
	if created["secret_set"] != true {
		t.Fatalf("secret_set = %v", created["secret_set"])
	}
	raw, _ := json.Marshal(created)
	if strings.Contains(string(raw), "supersecret") {
		t.Fatalf("response leaks secret: %s", raw)
	}

	// Store has the real secret.
	stored, err := env.store.GetConnection(id)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if stored.Secret != "sk-ant-supersecret" {
		t.Fatalf("stored secret = %q", stored.Secret)
	}

	// List masks secrets.
	status, envl = call(t, env.handlers.ListConnections, "GET", "/api/connections", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("list status = %d", status)
	}
	listRaw, _ := json.Marshal(dataField[[]map[string]any](t, envl))
	if strings.Contains(string(listRaw), "supersecret") {
		t.Fatalf("list leaks secret: %s", listRaw)
	}

	// Update with empty secret preserves the stored one.
	updateBody := fmt.Sprintf(`{"provider_id":%q,"name":"renamed","kind":"api_key","secret":""}`, providerID)
	status, _ = call(t, env.handlers.UpdateConnection, "PUT", "/api/connections/"+id, updateBody, map[string]any{"id": id}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("update status = %d", status)
	}
	stored, err = env.store.GetConnection(id)
	if err != nil {
		t.Fatalf("GetConnection after update: %v", err)
	}
	if stored.Secret != "sk-ant-supersecret" || stored.Name != "renamed" {
		t.Fatalf("after update = %+v", stored)
	}

	// Update with a new secret rotates it.
	rotateBody := fmt.Sprintf(`{"provider_id":%q,"name":"renamed","kind":"api_key","secret":"sk-rotated"}`, providerID)
	status, _ = call(t, env.handlers.UpdateConnection, "PUT", "/api/connections/"+id, rotateBody, map[string]any{"id": id}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("rotate status = %d", status)
	}
	stored, err = env.store.GetConnection(id)
	if err != nil {
		t.Fatalf("GetConnection after rotate: %v", err)
	}
	if stored.Secret != "sk-rotated" {
		t.Fatalf("rotated secret = %q", stored.Secret)
	}

	// Delete.
	status, _ = call(t, env.handlers.DeleteConnection, "DELETE", "/api/connections/"+id, "", map[string]any{"id": id}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("delete status = %d", status)
	}
	status, _ = call(t, env.handlers.DeleteConnection, "DELETE", "/api/connections/"+id, "", map[string]any{"id": id}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("delete missing status = %d", status)
	}
}

func TestOAuthStartCallbackRefresh(t *testing.T) {
	env := newTestEnv(t)
	env.withOAuth(t)

	status, envl := call(t, env.handlers.CreateProvider, "POST", "/api/providers",
		`{"name":"Anthropic","type":"anthropic","enabled":true}`, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create provider status = %d", status)
	}
	providerID := dataField[map[string]any](t, envl)["id"].(string)

	// Start.
	status, envl = call(t, env.handlers.OAuthStart, "GET", "/api/oauth/anthropic/start", "",
		map[string]any{"provider": "anthropic"}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("start status = %d err = %q", status, errMessage(t, envl))
	}
	startData := dataField[map[string]any](t, envl)
	state, _ := startData["state"].(string)
	authURL, _ := startData["auth_url"].(string)
	if state == "" || !strings.Contains(authURL, "code_challenge") {
		t.Fatalf("start data = %v", startData)
	}

	// Unknown provider → 404.
	status, _ = call(t, env.handlers.OAuthStart, "GET", "/api/oauth/nope/start", "",
		map[string]any{"provider": "nope"}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("unknown provider start status = %d", status)
	}

	// Callback creates an oauth connection with tokens.
	cbBody := fmt.Sprintf(`{"state":%q,"code":"any-code","provider_id":%q,"name":"claude oauth"}`, state, providerID)
	status, envl = call(t, env.handlers.OAuthCallback, "POST", "/api/oauth/anthropic/callback", cbBody,
		map[string]any{"provider": "anthropic"}, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("callback status = %d err = %q", status, errMessage(t, envl))
	}
	conn := dataField[map[string]any](t, envl)
	connID, _ := conn["id"].(string)
	if connID == "" || conn["kind"] != "oauth" || conn["access_token_set"] != true || conn["refresh_token_set"] != true {
		t.Fatalf("callback conn = %v", conn)
	}

	stored, err := env.store.GetConnection(connID)
	if err != nil {
		t.Fatalf("GetConnection: %v", err)
	}
	if stored.AccessToken != "at-cb" || stored.RefreshToken != "rt-cb" {
		t.Fatalf("stored tokens = %+v", stored)
	}

	// Reused state → error.
	status, _ = call(t, env.handlers.OAuthCallback, "POST", "/api/oauth/anthropic/callback", cbBody,
		map[string]any{"provider": "anthropic"}, nil)
	if status == fasthttp.StatusCreated {
		t.Fatal("state reuse accepted")
	}

	// Refresh rotates the access token.
	status, envl = call(t, env.handlers.RefreshConnection, "POST", "/api/connections/"+connID+"/refresh", "",
		map[string]any{"id": connID}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("refresh status = %d err = %q", status, errMessage(t, envl))
	}
	stored, err = env.store.GetConnection(connID)
	if err != nil {
		t.Fatalf("GetConnection after refresh: %v", err)
	}
	if stored.AccessToken != "at-refreshed" {
		t.Fatalf("refreshed access token = %q", stored.AccessToken)
	}
}

func TestPathIDRejectsNonString(t *testing.T) {
	env := newTestEnv(t)
	token := loginToken(t, env)
	authHeader := map[string]string{"Authorization": "Bearer " + token}

	// Pass an int as the {id} route param.
	status, _ := call(t, env.handlers.RequireSession(env.handlers.UpdateProvider), "PUT", "/api/providers/123",
		`{"name":"X","type":"openai"}`, map[string]any{"id": 123}, authHeader)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("non-string pathID status = %d, want 400", status)
	}
}

func TestUpdateConnectionRejectsUnknownProviderID(t *testing.T) {
	env := newTestEnv(t)
	token := loginToken(t, env)
	authHeader := map[string]string{"Authorization": "Bearer " + token}

	status, envl := call(t, env.handlers.CreateProvider, "POST", "/api/providers",
		`{"name":"Anthropic","type":"anthropic","enabled":true}`, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create provider status = %d", status)
	}
	providerID := dataField[map[string]any](t, envl)["id"].(string)

	body := fmt.Sprintf(`{"provider_id":%q,"name":"main","kind":"api_key"}`, providerID)
	status, envl = call(t, env.handlers.CreateConnection, "POST", "/api/connections", body, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create connection status = %d", status)
	}
	connID := dataField[map[string]any](t, envl)["id"].(string)

	// Update with unknown provider_id → 400 same shape as CreateConnection.
	status, envl = call(t, env.handlers.UpdateConnection, "PUT", "/api/connections/"+connID,
		`{"provider_id":"nope","name":"renamed","kind":"api_key"}`, map[string]any{"id": connID}, authHeader)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("unknown provider_id status = %d, want 400", status)
	}
	if msg := errMessage(t, envl); msg != "unknown provider_id" {
		t.Fatalf("error message = %q, want %q", msg, "unknown provider_id")
	}
}

func TestRefreshConnectionRequiresRefreshToken(t *testing.T) {
	env := newTestEnv(t)
	env.withOAuth(t)

	status, envl := call(t, env.handlers.CreateProvider, "POST", "/api/providers",
		`{"name":"Anthropic","type":"anthropic","enabled":true}`, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create provider status = %d", status)
	}
	providerID := dataField[map[string]any](t, envl)["id"].(string)

	body := fmt.Sprintf(`{"provider_id":%q,"name":"key","kind":"api_key","secret":"sk-1"}`, providerID)
	status, envl = call(t, env.handlers.CreateConnection, "POST", "/api/connections", body, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create connection status = %d", status)
	}
	id := dataField[map[string]any](t, envl)["id"].(string)

	status, _ = call(t, env.handlers.RefreshConnection, "POST", "/api/connections/"+id+"/refresh", "",
		map[string]any{"id": id}, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("refresh without refresh token status = %d", status)
	}
}

func TestOAuthStartGemini(t *testing.T) {
	env := newTestEnv(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.PostForm.Get("client_secret") != auth.GeminiOAuth().ClientSecret {
			t.Errorf("missing client_secret")
		}
		json.NewEncoder(w).Encode(map[string]any{"access_token": "at-gem", "refresh_token": "rt-gem", "expires_in": 3600})
	}))
	t.Cleanup(srv.Close)

	flow := auth.NewOAuthFlow(auth.GeminiOAuth(), env.store, srv.Client())
	env.handlers = New(env.store, env.sessions, map[string]*auth.OAuthFlow{"gemini": flow})

	status, envl := call(t, env.handlers.OAuthStart, "GET", "/api/oauth/gemini/start", "",
		map[string]any{"provider": "gemini"}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("start status = %d err = %q", status, errMessage(t, envl))
	}
	startData := dataField[map[string]any](t, envl)
	authURL := startData["auth_url"].(string)
	if !strings.Contains(authURL, "client_id="+auth.GeminiOAuth().ClientID) {
		t.Fatalf("auth_url missing gemini client_id: %s", authURL)
	}
}

func TestOAuthStartXai(t *testing.T) {
	env := newTestEnv(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"access_token": "at-xai", "refresh_token": "rt-xai", "expires_in": 3600})
	}))
	t.Cleanup(srv.Close)

	flow := auth.NewOAuthFlow(auth.XaiOAuth(), env.store, srv.Client())
	env.handlers = New(env.store, env.sessions, map[string]*auth.OAuthFlow{"xai": flow})

	status, envl := call(t, env.handlers.OAuthStart, "GET", "/api/oauth/xai/start", "",
		map[string]any{"provider": "xai"}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("start status = %d err = %q", status, errMessage(t, envl))
	}
	startData := dataField[map[string]any](t, envl)
	authURL := startData["auth_url"].(string)
	if !strings.Contains(authURL, "client_id=b1a00492-073a-47ea-816f-4c329264a828") {
		t.Fatalf("auth_url missing xai client_id: %s", authURL)
	}
	if strings.Contains(authURL, "client_secret") {
		t.Error("xai auth_url should not contain client_secret")
	}
}

func TestRedirectURIFromRequestOrigin(t *testing.T) {
	env := newTestEnv(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"access_token": "at", "refresh_token": "rt", "expires_in": 3600})
	}))
	t.Cleanup(srv.Close)

	cfg := auth.AnthropicOAuth()
	cfg.RedirectURI = ""
	flow := auth.NewOAuthFlow(cfg, env.store, srv.Client())
	env.handlers = New(env.store, env.sessions, map[string]*auth.OAuthFlow{"anthropic": flow})

	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/api/oauth/anthropic/start")
	ctx.SetUserValue("provider", "anthropic")
	ctx.Request.Header.Set("Origin", "https://remote.example.com")

	env.handlers.OAuthStart(&ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d body = %s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	envelope := map[string]json.RawMessage{}
	if err := json.Unmarshal(ctx.Response.Body(), &envelope); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	data := dataField[map[string]any](t, envelope)
	authURL := data["auth_url"].(string)
	if !strings.Contains(authURL, "redirect_uri=https%3A%2F%2Fremote.example.com%2Fapi%2Foauth%2Fanthropic%2Fcallback") {
		t.Fatalf("auth_url missing origin redirect_uri: %s", authURL)
	}
}

func TestRedirectURISettingsOverride(t *testing.T) {
	env := newTestEnv(t)
	if err := env.store.SetSettings(map[string]string{"oauth_redirect_uri": "https://dashboard.example.com/callback"}); err != nil {
		t.Fatalf("SetSettings: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"access_token": "at", "refresh_token": "rt", "expires_in": 3600})
	}))
	t.Cleanup(srv.Close)

	cfg := auth.AnthropicOAuth()
	cfg.RedirectURI = ""
	flow := auth.NewOAuthFlow(cfg, env.store, srv.Client())
	env.handlers = New(env.store, env.sessions, map[string]*auth.OAuthFlow{"anthropic": flow})

	status, envl := call(t, env.handlers.OAuthStart, "GET", "/api/oauth/anthropic/start", "",
		map[string]any{"provider": "anthropic"}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("start status = %d", status)
	}
	data := dataField[map[string]any](t, envl)
	authURL := data["auth_url"].(string)
	if !strings.Contains(authURL, "redirect_uri=https%3A%2F%2Fdashboard.example.com%2Fcallback") {
		t.Fatalf("auth_url missing override redirect_uri: %s", authURL)
	}
}

func TestLoginLockout429AndRetryAfter(t *testing.T) {
	env := newTestEnv(t)
	ip := "1.2.3.4"
	headers := map[string]string{"X-Forwarded-For": ip}

	// 4 failed attempts return 401.
	for i := 0; i < 4; i++ {
		status, _ := call(t, env.handlers.Login, "POST", "/api/auth/login",
			`{"username":"admin","password":"wrong"}`, nil, headers)
		if status != fasthttp.StatusUnauthorized {
			t.Fatalf("attempt %d status = %d, want 401", i+1, status)
		}
	}

	// 5th attempt triggers lock and returns 429.
	status, envl := call(t, env.handlers.Login, "POST", "/api/auth/login",
		`{"username":"admin","password":"wrong"}`, nil, headers)
	if status != fasthttp.StatusTooManyRequests {
		t.Fatalf("locked status = %d, want 429", status)
	}

	msg := errMessage(t, envl)
	if msg == "" {
		t.Fatal("expected error message")
	}
	if !strings.Contains(msg, "Too many failed attempts") {
		t.Fatalf("error message = %q", msg)
	}

	// Body should contain retry_after and reset_hint.
	var bodyMap map[string]json.RawMessage
	var lastBody []byte
	// Re-run the call to capture the raw body since call decodes envelope.
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.SetRequestURI("/api/auth/login")
	ctx.Request.SetBody([]byte(`{"username":"admin","password":"wrong"}`))
	ctx.Request.Header.Set("X-Forwarded-For", ip)
	env.handlers.Login(&ctx)
	lastBody = ctx.Response.Body()
	if err := json.Unmarshal(lastBody, &bodyMap); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	errObj := map[string]json.RawMessage{}
	if err := json.Unmarshal(bodyMap["error"], &errObj); err != nil {
		t.Fatalf("unmarshal error object: %v", err)
	}
	if _, ok := errObj["retry_after"]; !ok {
		t.Fatalf("missing retry_after in error: %s", bodyMap["error"])
	}
	if _, ok := errObj["reset_hint"]; !ok {
		t.Fatalf("missing reset_hint in error: %s", bodyMap["error"])
	}

	ra := string(ctx.Response.Header.Peek("Retry-After"))
	if ra == "" {
		t.Fatal("missing Retry-After header")
	}
}

func TestLoginDefaultPasswordWhenNoHash(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		env := newTestEnv(t)
		if err := env.store.SetUserPasswordHash("admin", ""); err != nil {
			t.Fatalf("SetUserPasswordHash: %v", err)
		}

		status, envl := call(t, env.handlers.Login, "POST", "/api/auth/login",
			`{"username":"admin","password":"123456"}`, nil, nil)
		if status != fasthttp.StatusOK {
			t.Fatalf("login status = %d, err = %q", status, errMessage(t, envl))
		}
	})

	t.Run("env-override", func(t *testing.T) {
		t.Setenv("INITIAL_PASSWORD", "custompw")
		env := newTestEnv(t)
		if err := env.store.SetUserPasswordHash("admin", ""); err != nil {
			t.Fatalf("SetUserPasswordHash: %v", err)
		}

		status, envl := call(t, env.handlers.Login, "POST", "/api/auth/login",
			`{"username":"admin","password":"custompw"}`, nil, nil)
		if status != fasthttp.StatusOK {
			t.Fatalf("login status = %d, err = %q", status, errMessage(t, envl))
		}
	})
}

func TestLoginOidcModeBlocksPassword(t *testing.T) {
	env := newTestEnv(t)

	// Without OIDC configured, mode "oidc" does NOT block (safety guard).
	if err := env.store.SetSettings(map[string]string{"auth_mode": "oidc"}); err != nil {
		t.Fatalf("SetSettings: %v", err)
	}
	status, envl := call(t, env.handlers.Login, "POST", "/api/auth/login",
		`{"username":"admin","password":"123456"}`, nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("oidc without config: status = %d, err = %q", status, errMessage(t, envl))
	}

	// With OIDC configured, mode "oidc" blocks password login.
	if err := env.store.SetSettings(map[string]string{
		"auth_mode":          "oidc",
		"oidc_issuer_url":    "https://example.com",
		"oidc_client_id":     "client",
		"oidc_client_secret": "secret",
	}); err != nil {
		t.Fatalf("SetSettings: %v", err)
	}
	status, envl = call(t, env.handlers.Login, "POST", "/api/auth/login",
		`{"username":"admin","password":"123456"}`, nil, nil)
	if status != fasthttp.StatusForbidden {
		t.Fatalf("oidc with config: status = %d, want 403", status)
	}
	msg := errMessage(t, envl)
	if !strings.Contains(msg, "Password login is disabled") {
		t.Fatalf("error = %q, want 'Password login is disabled'", msg)
	}
}

func TestStatusReportsAuthMode(t *testing.T) {
	env := newTestEnv(t)
	if err := env.store.SetSettings(map[string]string{"auth_mode": "oidc"}); err != nil {
		t.Fatalf("SetSettings: %v", err)
	}

	status, envl := call(t, env.handlers.Status, "GET", "/api/auth/status", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("status = %d", status)
	}
	data := dataField[map[string]any](t, envl)
	if data["auth_mode"] != "oidc" {
		t.Fatalf("auth_mode = %v, want oidc", data["auth_mode"])
	}
}
