package handlers

import (
	"errors"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/provider/oauth"
	"github.com/bloodf/g0router/internal/providers"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// --- MCP unexported helpers ---

func TestFilterCompactTools(t *testing.T) {
	tools := []providers.Tool{
		{Function: providers.ToolFunction{Name: "search"}},
		{Function: providers.ToolFunction{Name: "fetch"}},
		{Function: providers.ToolFunction{Name: "write"}},
	}
	if got := filterCompactTools(tools, nil); len(got) != 3 {
		t.Fatalf("nil allowlist should return all, got %d", len(got))
	}
	got := filterCompactTools(tools, []string{"search", " ", "write"})
	if len(got) != 2 || got[0].Function.Name != "search" || got[1].Function.Name != "write" {
		t.Fatalf("filtered = %+v, want search+write", got)
	}
	if got := filterCompactTools(tools, []string{"missing"}); len(got) != 0 {
		t.Fatalf("non-matching allowlist should return empty, got %d", len(got))
	}
}

func TestAppendAllowedTools(t *testing.T) {
	out := appendAllowedTools(nil, "  ")
	if len(out) != 0 {
		t.Fatalf("blank should be dropped, got %d", len(out))
	}
	out = appendAllowedTools(out, " search ")
	if len(out) != 1 || out[0] != "search" {
		t.Fatalf("trimmed name = %+v, want [search]", out)
	}
}

func TestAllowedToolsFromRequest(t *testing.T) {
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/mcp/tools?allowed_tool=a&allowed_tools=b,c,%20&allowed_tool=")
	got := allowedToolsFromRequest(ctx)
	if strings.Join(got, ",") != "a,b,c" {
		t.Fatalf("allowed tools = %+v, want a,b,c", got)
	}
}

func TestStringValueHelper(t *testing.T) {
	if stringValue(nil) != "" {
		t.Fatal("nil pointer should be empty string")
	}
	v := "x"
	if stringValue(&v) != "x" {
		t.Fatal("pointer deref failed")
	}
}

func TestIsMCPSecretKey(t *testing.T) {
	for _, k := range []string{"TOKEN", "api_key", "Authorization", "secret", "PASSWORD"} {
		if !isMCPSecretKey(k) {
			t.Fatalf("%q should be secret", k)
		}
	}
	if isMCPSecretKey("region") {
		t.Fatal("region should not be secret")
	}
}

func TestRedactMCPSecretMap(t *testing.T) {
	if redactMCPSecretMap(nil) != nil {
		t.Fatal("nil map should stay nil")
	}
	out := redactMCPSecretMap(map[string]string{"TOKEN": "x", "REGION": "us"})
	if out["TOKEN"] != mcp.RedactedValue || out["REGION"] != "us" {
		t.Fatalf("redacted = %+v", out)
	}
}

func TestWriteMCPToolError(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{mcp.ErrToolNotFound, fasthttp.StatusNotFound},
		{mcp.ErrClientNotFound, fasthttp.StatusNotFound},
		{mcp.ErrInvalidToolArguments, fasthttp.StatusBadRequest},
		{errors.New("sqlite boom"), fasthttp.StatusBadGateway},
	}
	for _, c := range cases {
		ctx := &fasthttp.RequestCtx{}
		writeMCPToolError(ctx, c.err)
		if ctx.Response.StatusCode() != c.want {
			t.Fatalf("err %v status = %d, want %d", c.err, ctx.Response.StatusCode(), c.want)
		}
		if c.want == fasthttp.StatusBadGateway {
			assertNoInternalDetail(t, ctx.Response.Body())
		}
	}
}

func TestDecodeMCPInstanceRequestInvalidJSON(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetBodyString(`{`)
	if _, ok := decodeMCPInstanceRequest(ctx); ok {
		t.Fatal("invalid JSON should fail")
	}
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestDecodeMCPClientRequestInvalidJSON(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetBodyString(`{`)
	if _, ok := decodeMCPClientRequest(ctx); ok {
		t.Fatal("invalid JSON should fail")
	}
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestDecodeMCPToolExecuteRequest(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetBodyString(`{`)
	if _, ok := decodeMCPToolExecuteRequest(ctx); ok {
		t.Fatal("invalid JSON should fail")
	}
	// Empty arguments default to {}.
	ctx = &fasthttp.RequestCtx{}
	ctx.Request.SetBodyString(`{"allowed_tools":["a"]}`)
	req, ok := decodeMCPToolExecuteRequest(ctx)
	if !ok || string(req.Arguments) != `{}` {
		t.Fatalf("defaulted args = %s ok=%v, want {}", req.Arguments, ok)
	}
}

func TestMCPInstancesNilStoreAndMethod(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		MCPInstances(ctx, nil, nil, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("nil store = %d, want 503", ctx.Response.StatusCode())
	}
	s := newHandlerStore(t)
	ctx, _ = runHandler(t, fasthttp.MethodPatch, "", func(ctx *fasthttp.RequestCtx) {
		MCPInstances(ctx, s, nil, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("method = %d, want 405", ctx.Response.StatusCode())
	}
	// DELETE blank id.
	ctx, _ = runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		MCPInstances(ctx, s, nil, "   ")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("blank id = %d, want 400", ctx.Response.StatusCode())
	}
	// POST invalid JSON.
	ctx, _ = runHandler(t, fasthttp.MethodPost, `{`, func(ctx *fasthttp.RequestCtx) {
		MCPInstances(ctx, s, nil, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("invalid json = %d, want 400", ctx.Response.StatusCode())
	}
	// POST active instance with nil runtime -> 503 and rollback.
	ctx, _ = runHandler(t, fasthttp.MethodPost, `{"name":"i","server_key":"k","launch_type":"http","transport":"streamable-http","url":"https://mcp.example/mcp","is_active":true}`, func(ctx *fasthttp.RequestCtx) {
		MCPInstances(ctx, s, nil, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("active w/o runtime = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestMCPClientsNilStoreMethodAndPostNoRuntime(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		MCPClients(ctx, nil, nil, nil, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("nil store = %d, want 503", ctx.Response.StatusCode())
	}
	s := newHandlerStore(t)
	ctx, _ = runHandler(t, fasthttp.MethodPatch, "", func(ctx *fasthttp.RequestCtx) {
		MCPClients(ctx, s, nil, nil, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("method = %d, want 405", ctx.Response.StatusCode())
	}
	// POST without runtime -> 503.
	ctx, _ = runHandler(t, fasthttp.MethodPost, `{"name":"c"}`, func(ctx *fasthttp.RequestCtx) {
		MCPClients(ctx, s, nil, nil, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("POST no runtime = %d, want 503", ctx.Response.StatusCode())
	}
	// DELETE blank id.
	ctx, _ = runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		MCPClients(ctx, s, nil, nil, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("blank id = %d, want 400", ctx.Response.StatusCode())
	}
	// DELETE missing client -> 404.
	ctx, _ = runHandler(t, fasthttp.MethodDelete, "", func(ctx *fasthttp.RequestCtx) {
		MCPClients(ctx, s, nil, nil, "missing")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("missing client delete = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestMCPToolsMethodNotAllowedAndPostValidation(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPatch, "", func(ctx *fasthttp.RequestCtx) {
		MCPTools(ctx, nil, nil, "")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("method = %d, want 405", ctx.Response.StatusCode())
	}
	// POST with nil tools -> 503.
	ctx, _ = runHandler(t, fasthttp.MethodPost, `{}`, func(ctx *fasthttp.RequestCtx) {
		MCPTools(ctx, nil, nil, "tool")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("nil tools = %d, want 503", ctx.Response.StatusCode())
	}
}

func TestCompactToolListNilToolsNilStore(t *testing.T) {
	out, err := compactToolList(nil, nil, nil, "", "", nil)
	if err != nil || out != nil {
		t.Fatalf("nil tools+store = %v / %+v, want nil/nil", err, out)
	}
}

func TestMCPOAuthStartValidation(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		MCPOAuthStart(ctx, nil, "id")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("nil store = %d, want 503", ctx.Response.StatusCode())
	}
	s := newHandlerStore(t)
	ctx, _ = runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		MCPOAuthStart(ctx, s, "  ")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("blank id = %d, want 400", ctx.Response.StatusCode())
	}
	ctx, _ = runHandler(t, fasthttp.MethodPost, `{`, func(ctx *fasthttp.RequestCtx) {
		MCPOAuthStart(ctx, s, "id")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("invalid json = %d, want 400", ctx.Response.StatusCode())
	}
	ctx, _ = runHandler(t, fasthttp.MethodPost, `{"authorization_url":"https://a/x"}`, func(ctx *fasthttp.RequestCtx) {
		MCPOAuthStart(ctx, s, "id")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("missing resource_uri = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestMCPOAuthAccountsNilAndMethod(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodGet, "", func(ctx *fasthttp.RequestCtx) {
		MCPOAuthAccounts(ctx, nil, "id")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("nil store = %d, want 503", ctx.Response.StatusCode())
	}
	s := newHandlerStore(t)
	ctx, _ = runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		MCPOAuthAccounts(ctx, s, "id")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("method = %d, want 405", ctx.Response.StatusCode())
	}
}

// --- MCP OAuth completion helpers ---

func TestMCPOAuthCompleteValidation(t *testing.T) {
	ctx, _ := runHandler(t, fasthttp.MethodPost, "", func(ctx *fasthttp.RequestCtx) {
		MCPOAuthComplete(ctx, nil, nil, nil, "  ")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("blank id = %d, want 400", ctx.Response.StatusCode())
	}
	completer := &fakeMCPOAuthCompleter{}
	ctx, _ = runHandler(t, fasthttp.MethodPost, `{`, func(ctx *fasthttp.RequestCtx) {
		MCPOAuthComplete(ctx, completer, nil, nil, "id")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("invalid json = %d, want 400", ctx.Response.StatusCode())
	}
	// Nil completer -> 503.
	ctx, _ = runHandler(t, fasthttp.MethodPost, `{"callback_url":"https://x?code=c&state=s"}`, func(ctx *fasthttp.RequestCtx) {
		MCPOAuthComplete(ctx, nil, nil, nil, "id")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("nil completer = %d, want 503", ctx.Response.StatusCode())
	}
	// Completer not-found error -> 404.
	nf := &fakeMCPOAuthCompleter{err: mcp.ErrOAuthFlowNotFound}
	ctx, _ = runHandler(t, fasthttp.MethodPost, `{"callback_url":"https://x?code=c&state=s"}`, func(ctx *fasthttp.RequestCtx) {
		MCPOAuthComplete(ctx, nf, nil, nil, "id")
	})
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("flow not found = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestMCPOAuthCallbackMissingInstanceID(t *testing.T) {
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/mcp/oauth/callback")
	MCPOAuthCallback(ctx, &fakeMCPOAuthCompleter{}, nil, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("missing instance_id = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestValidateCallbackURL(t *testing.T) {
	if err := validateCallbackURL("https://x?code=c&state=s"); err != nil {
		t.Fatalf("valid url rejected: %v", err)
	}
	if err := validateCallbackURL("https://x?state=s"); err == nil {
		t.Fatal("missing code should error")
	}
	if err := validateCallbackURL("https://x?code=c"); err == nil {
		t.Fatal("missing state should error")
	}
	if err := validateCallbackURL("://bad-url"); err == nil {
		t.Fatal("unparseable url should error")
	}
}

func TestDecodeCallbackInstanceID(t *testing.T) {
	if got := decodeCallbackInstanceID("plain"); got != "plain" {
		t.Fatalf("plain = %q", got)
	}
	if got := decodeCallbackInstanceID("b64:" + "aW5zdGFuY2U"); got != "instance" {
		t.Fatalf("b64 decode = %q, want instance", got)
	}
	if got := decodeCallbackInstanceID("b64:!!!notbase64"); got != "" {
		t.Fatalf("bad b64 = %q, want empty", got)
	}
}

// --- OAuth unexported helpers ---

func TestOAuthProviderFromPath(t *testing.T) {
	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/oauth/Anthropic/start")
	if got := oauthProviderFromPath(ctx); string(got) != "anthropic" {
		t.Fatalf("provider = %q, want anthropic", got)
	}
	bad := newHandlerCtx(fasthttp.MethodPost, "/api/other")
	if got := oauthProviderFromPath(bad); got != "" {
		t.Fatalf("bad path = %q, want empty", got)
	}
}

func TestRedirectURIFromAuthURL(t *testing.T) {
	if got := redirectURIFromAuthURL(""); got != "" {
		t.Fatalf("empty = %q", got)
	}
	if got := redirectURIFromAuthURL("https://a/x?redirect_uri=https%3A%2F%2Fcb"); got != "https://cb" {
		t.Fatalf("redirect = %q, want https://cb", got)
	}
	if got := redirectURIFromAuthURL("://bad"); got != "" {
		t.Fatalf("bad url = %q, want empty", got)
	}
}

func TestSplitStoredOAuthSession(t *testing.T) {
	state, verifier := splitStoredOAuthSession("state.verifier")
	if state != "state" || verifier != "verifier" {
		t.Fatalf("split = %q/%q", state, verifier)
	}
	state, verifier = splitStoredOAuthSession("only-state")
	if state != "only-state" || verifier != "" {
		t.Fatalf("no-verifier split = %q/%q", state, verifier)
	}
}

func TestConsumeOAuthSessionNilStore(t *testing.T) {
	if _, err := consumeOAuthSession(nil, "s"); err == nil {
		t.Fatal("nil store should error")
	}
}

func TestGetOAuthSessionNilAndNotFound(t *testing.T) {
	if got, err := getOAuthSession(nil, "s"); got != nil || err != nil {
		t.Fatalf("nil store = %+v / %v, want nil/nil", got, err)
	}
	s := newHandlerStore(t)
	if got, err := getOAuthSession(s, "missing"); got != nil || err != nil {
		t.Fatalf("missing session = %+v / %v, want nil/nil", got, err)
	}
}

func TestPersistOAuthConnectionNilStore(t *testing.T) {
	if _, err := persistOAuthConnection(nil, oauth.TokenResult{}, "label", "openai"); err == nil {
		t.Fatal("nil store should error")
	}
}

func TestCreateOAuthSessionNilAndEmpty(t *testing.T) {
	if err := createOAuthSession(nil, &oauth.AuthSession{SessionID: "s"}, ""); err != nil {
		t.Fatalf("nil store should no-op: %v", err)
	}
	s := newHandlerStore(t)
	if err := createOAuthSession(s, &oauth.AuthSession{SessionID: ""}, ""); err != nil {
		t.Fatalf("empty session id should no-op: %v", err)
	}
}

func TestOAuthFlowForPathMissingProvider(t *testing.T) {
	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/other")
	_, _, ok := oauthFlowForPath(ctx, OAuthFlows{})
	if ok {
		t.Fatal("missing provider should fail")
	}
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestOAuthFlowNotFound(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	_, ok := oauthFlow(ctx, OAuthFlows{}, oauth.ProviderID("openai"))
	if ok {
		t.Fatal("unknown provider should fail")
	}
	if ctx.Response.StatusCode() != fasthttp.StatusNotFound {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}

func TestDecodeOAuthStartRequestVariants(t *testing.T) {
	// Empty body returns empty request, ok.
	empty := &fasthttp.RequestCtx{}
	if _, ok := decodeOAuthStartRequest(empty); !ok {
		t.Fatal("empty body should be ok")
	}
	// Invalid JSON -> 400.
	bad := &fasthttp.RequestCtx{}
	bad.Request.SetBodyString(`{`)
	if _, ok := decodeOAuthStartRequest(bad); ok {
		t.Fatal("invalid json should fail")
	}
	if bad.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("status = %d, want 400", bad.Response.StatusCode())
	}
	// Valid body.
	good := &fasthttp.RequestCtx{}
	good.Request.SetBodyString(`{"account_label":"work"}`)
	req, ok := decodeOAuthStartRequest(good)
	if !ok || req.AccountLabel != "work" {
		t.Fatalf("valid req = %+v ok=%v", req, ok)
	}
}

func TestOAuthCallbackValidation(t *testing.T) {
	flows := OAuthFlows{}
	// error param.
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/oauth/callback?error=denied")
	OAuthCallback(ctx, nil, flows)
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("error param = %d, want 400", ctx.Response.StatusCode())
	}
	// missing code.
	ctx = newHandlerCtx(fasthttp.MethodGet, "/api/oauth/callback?state=s")
	OAuthCallback(ctx, nil, flows)
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("missing code = %d, want 400", ctx.Response.StatusCode())
	}
	// missing state.
	ctx = newHandlerCtx(fasthttp.MethodGet, "/api/oauth/callback?code=c")
	OAuthCallback(ctx, nil, flows)
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("missing state = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestOAuthExchangeValidation(t *testing.T) {
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("anthropic")}
	flows := OAuthFlows{oauth.CanonicalFlowProviderID(flow.provider): flow}
	// Invalid JSON.
	ctx := newHandlerCtx(fasthttp.MethodPost, "/api/oauth/anthropic/exchange")
	ctx.Request.SetBodyString(`{`)
	OAuthExchange(ctx, nil, flows)
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("invalid json = %d, want 400", ctx.Response.StatusCode())
	}
	// Missing code.
	ctx = newHandlerCtx(fasthttp.MethodPost, "/api/oauth/anthropic/exchange")
	ctx.Request.SetBodyString(`{"state":"s"}`)
	OAuthExchange(ctx, nil, flows)
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("missing code = %d, want 400", ctx.Response.StatusCode())
	}
	// Missing state.
	ctx = newHandlerCtx(fasthttp.MethodPost, "/api/oauth/anthropic/exchange")
	ctx.Request.SetBodyString(`{"code":"c"}`)
	OAuthExchange(ctx, nil, flows)
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("missing state = %d, want 400", ctx.Response.StatusCode())
	}
}

func TestOAuthPollValidation(t *testing.T) {
	flow := &fakeOAuthFlow{provider: oauth.ProviderID("anthropic")}
	flows := OAuthFlows{oauth.CanonicalFlowProviderID(flow.provider): flow}
	// Missing session_id.
	ctx := newHandlerCtx(fasthttp.MethodGet, "/api/oauth/anthropic/poll")
	OAuthPoll(ctx, nil, flows)
	if ctx.Response.StatusCode() != fasthttp.StatusBadRequest {
		t.Fatalf("missing session_id = %d, want 400", ctx.Response.StatusCode())
	}
}

// --- usage quotaKeyFromConnection ---

func TestQuotaKeyFromConnection(t *testing.T) {
	if _, ok := quotaKeyFromConnection(providers.ProviderOpenAI, nil); ok {
		t.Fatal("nil connection should fail")
	}
	apiKey := "sk-test"
	key, ok := quotaKeyFromConnection(providers.ProviderOpenAI, &store.Connection{
		ID: "c1", AuthType: store.AuthTypeAPIKey, APIKey: &apiKey,
	})
	if !ok || key.Value != "sk-test" || key.ConnID != "c1" {
		t.Fatalf("api key path = %+v ok=%v", key, ok)
	}
	access := "tok"
	key, ok = quotaKeyFromConnection(providers.ProviderOpenAI, &store.Connection{
		ID: "c2", AuthType: store.AuthTypeOAuth, AccessToken: &access,
	})
	if !ok || key.Value != "tok" {
		t.Fatalf("oauth path = %+v ok=%v", key, ok)
	}
	// No usable credential.
	if _, ok := quotaKeyFromConnection(providers.ProviderOpenAI, &store.Connection{ID: "c3", AuthType: store.AuthTypeOAuth}); ok {
		t.Fatal("connection without credential should fail")
	}
}
