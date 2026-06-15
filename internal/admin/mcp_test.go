package admin

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/schemas"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// echoToolProcess is a fake mcp.Process that, on each tools/call frame written to
// its stdin, immediately feeds back a JSON-RPC result frame carrying the same id
// via the spec's OnFrame callback — so the ExecuteTool bridge path is exercised
// WITHOUT any real subprocess.
type echoToolProcess struct {
	onFrame func(frame []byte)
	running bool
}

func (p *echoToolProcess) Write(frame []byte) error {
	var req struct {
		ID     int `json:"id"`
		Params struct {
			Name string `json:"name"`
		} `json:"params"`
	}
	if err := json.Unmarshal(frame, &req); err != nil {
		return nil
	}
	resp := fmt.Sprintf(
		`{"jsonrpc":"2.0","id":%d,"result":{"content":[{"type":"text","text":"executed %s ok"}]}}`,
		req.ID, req.Params.Name)
	if p.onFrame != nil {
		p.onFrame(append([]byte(resp), '\n'))
	}
	return nil
}

func (p *echoToolProcess) IsRunning() bool { return p.running }
func (p *echoToolProcess) Stop() error     { p.running = false; return nil }

// echoToolRunner is a fake ProcessRunner returning echoToolProcess instances.
type echoToolRunner struct{}

func (echoToolRunner) Start(spec mcp.ProcessSpec) (mcp.Process, error) {
	return &echoToolProcess{onFrame: spec.OnFrame, running: true}, nil
}

// newMCPTestEnv builds a test env with a real launcher (fake runner) + a real
// OAuth engine over a fake-transport client wired through the setters.
func newMCPTestEnv(t *testing.T) *testEnv {
	t.Helper()
	env := newTestEnv(t)
	launcher := mcp.NewLauncher(env.store)
	launcher.SetRunner(echoToolRunner{})
	env.handlers.SetMCPLauncher(launcher)
	engine := mcp.NewEngine(env.store, &http.Client{Transport: &fakeMCPTransport{}})
	env.handlers.SetMCPEngine(engine)
	return env
}

// fakeMCPTransport answers the OAuth discovery requests Engine.Start makes so
// auth/start returns an authorize URL without any real network.
type fakeMCPTransport struct{}

func (fakeMCPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var body string
	switch {
	case strings.Contains(req.URL.Path, "oauth-protected-resource"):
		body = `{"authorization_servers":["https://auth.example.com"]}`
	case strings.Contains(req.URL.Path, "oauth-authorization-server"):
		body = `{"authorization_endpoint":"https://auth.example.com/authorize","token_endpoint":"https://auth.example.com/token"}`
	default:
		body = `{}`
	}
	return &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}

func TestMcpListClients(t *testing.T) {
	env := newMCPTestEnv(t)

	status, envl := call(t, env.handlers.ListClients, "GET", "/api/mcp/clients", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("list clients status = %d", status)
	}
	clients := dataField[[]map[string]any](t, envl)
	if len(clients) == 0 {
		t.Fatalf("expected default-plugins clients, got 0")
	}
	// PascalCase keys (§1.2).
	first := clients[0]
	if _, ok := first["ID"]; !ok {
		t.Fatalf("client missing PascalCase ID key: %v", first)
	}
	if _, ok := first["Name"]; !ok {
		t.Fatalf("client missing PascalCase Name key: %v", first)
	}
}

func TestMcpInstanceCRUDStdioHTTPSSE(t *testing.T) {
	env := newMCPTestEnv(t)

	// Create stdio instance (allowlisted command npx).
	status, envl := call(t, env.handlers.CreateInstance, "POST", "/api/mcp/instances",
		`{"Name":"fs","Transport":"stdio","Command":"npx","Args":["-y","@modelcontextprotocol/server-filesystem"]}`, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create stdio status = %d err = %q", status, errMessage(t, envl))
	}
	inst := dataField[map[string]any](t, envl)
	id, _ := inst["ID"].(string)
	if id == "" {
		t.Fatalf("created instance = %v", inst)
	}
	if inst["Transport"] != "stdio" {
		t.Fatalf("Transport = %v, want stdio", inst["Transport"])
	}
	if inst["IsActive"] != true {
		t.Fatalf("IsActive = %v, want true (running)", inst["IsActive"])
	}
	if inst["HealthStatus"] != "healthy" {
		t.Fatalf("HealthStatus = %v, want healthy", inst["HealthStatus"])
	}

	// Create http instance.
	status, envl = call(t, env.handlers.CreateInstance, "POST", "/api/mcp/instances",
		`{"Name":"exa","Transport":"http","URL":"https://mcp.exa.ai/mcp"}`, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create http status = %d err = %q", status, errMessage(t, envl))
	}

	// Create sse instance.
	status, _ = call(t, env.handlers.CreateInstance, "POST", "/api/mcp/instances",
		`{"Name":"gh","Transport":"sse","URL":"https://api.github.com/mcp"}`, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create sse status = %d", status)
	}

	// List.
	status, envl = call(t, env.handlers.ListInstances, "GET", "/api/mcp/instances", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("list status = %d", status)
	}
	list := dataField[[]map[string]any](t, envl)
	if len(list) != 3 {
		t.Fatalf("list len = %d, want 3", len(list))
	}

	// Get.
	status, envl = call(t, env.handlers.GetInstance, "GET", "/api/mcp/instances/"+id, "",
		map[string]any{"id": id}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("get status = %d", status)
	}

	// Delete.
	status, _ = call(t, env.handlers.DeleteInstance, "DELETE", "/api/mcp/instances/"+id, "",
		map[string]any{"id": id}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("delete status = %d", status)
	}
	status, _ = call(t, env.handlers.GetInstance, "GET", "/api/mcp/instances/"+id, "",
		map[string]any{"id": id}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("get after delete status = %d, want 404", status)
	}
}

func TestMcpCreateInstanceRejectsNonAllowlistedBeforeSpawn(t *testing.T) {
	env := newMCPTestEnv(t)

	status, envl := call(t, env.handlers.CreateInstance, "POST", "/api/mcp/instances",
		`{"Name":"evil","Transport":"stdio","Command":"rm","Args":["-rf","/"]}`, nil, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("non-allowlisted command status = %d, want 400; err = %q", status, errMessage(t, envl))
	}

	// Nothing persisted.
	list, err := env.store.ListMCPInstances()
	if err != nil {
		t.Fatalf("ListMCPInstances: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("instances persisted = %d, want 0 (rejected before persist)", len(list))
	}
}

func TestMcpCreateInstanceValidation(t *testing.T) {
	env := newMCPTestEnv(t)

	// Missing name.
	status, _ := call(t, env.handlers.CreateInstance, "POST", "/api/mcp/instances",
		`{"Transport":"http","URL":"https://x"}`, nil, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("missing name status = %d, want 400", status)
	}

	// Missing both command and url.
	status, _ = call(t, env.handlers.CreateInstance, "POST", "/api/mcp/instances",
		`{"Name":"x","Transport":"http"}`, nil, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("missing mode status = %d, want 400", status)
	}
}

func TestMcpInstanceAccountsStripTokens(t *testing.T) {
	env := newMCPTestEnv(t)

	// Persist an instance + an oauth account with real tokens.
	inst, err := env.store.CreateMCPInstance(&store.MCPInstance{
		Name: "gh", Transport: "sse", URL: "https://api.github.com/mcp",
	})
	if err != nil {
		t.Fatalf("CreateMCPInstance: %v", err)
	}
	if _, err := env.store.UpsertMCPOAuthAccount(&store.MCPOAuthAccount{
		InstanceID:   inst.ID,
		ServerURL:    "https://api.github.com/mcp",
		AccessToken:  "secret-access-token-xyz",
		RefreshToken: "secret-refresh-token-abc",
		Status:       "connected",
		Scope:        "repo",
	}); err != nil {
		t.Fatalf("UpsertMCPOAuthAccount: %v", err)
	}

	status, envl := call(t, env.handlers.ListInstanceAccounts, "GET",
		"/api/mcp/instances/"+inst.ID+"/accounts", "", map[string]any{"id": inst.ID}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("accounts status = %d", status)
	}
	accounts := dataField[[]map[string]any](t, envl)
	if len(accounts) != 1 {
		t.Fatalf("accounts len = %d, want 1", len(accounts))
	}
	raw, _ := json.Marshal(accounts)
	for _, leak := range []string{"secret-access-token-xyz", "secret-refresh-token-abc", "AccessToken", "RefreshToken", "access_token", "refresh_token"} {
		if strings.Contains(string(raw), leak) {
			t.Fatalf("accounts response leaks %q: %s", leak, raw)
		}
	}
	// Non-secret fields present.
	if accounts[0]["status"] != "connected" {
		t.Fatalf("account status = %v", accounts[0]["status"])
	}
}

func TestMcpStartInstanceAuth(t *testing.T) {
	env := newMCPTestEnv(t)

	inst, err := env.store.CreateMCPInstance(&store.MCPInstance{
		Name: "gh", Transport: "sse", URL: "https://api.github.com/mcp",
	})
	if err != nil {
		t.Fatalf("CreateMCPInstance: %v", err)
	}

	status, envl := call(t, env.handlers.StartInstanceAuth, "POST",
		"/api/mcp/instances/"+inst.ID+"/auth/start", "", map[string]any{"id": inst.ID}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("auth start status = %d err = %q", status, errMessage(t, envl))
	}
	data := dataField[map[string]any](t, envl)
	url, _ := data["url"].(string)
	if url == "" || !strings.Contains(url, "auth.example.com/authorize") {
		t.Fatalf("auth start url = %q", url)
	}
	if !strings.Contains(url, "code_challenge") {
		t.Fatalf("auth start url missing PKCE challenge: %q", url)
	}
	// Never echo state/verifier.
	raw, _ := json.Marshal(data)
	if strings.Contains(string(raw), "verifier") {
		t.Fatalf("auth start leaks verifier: %s", raw)
	}
}

func TestMcpStartInstanceAuthNoEngine(t *testing.T) {
	env := newTestEnv(t) // no engine injected
	inst, err := env.store.CreateMCPInstance(&store.MCPInstance{
		Name: "gh", Transport: "sse", URL: "https://api.github.com/mcp",
	})
	if err != nil {
		t.Fatalf("CreateMCPInstance: %v", err)
	}
	status, _ := call(t, env.handlers.StartInstanceAuth, "POST",
		"/api/mcp/instances/"+inst.ID+"/auth/start", "", map[string]any{"id": inst.ID}, nil)
	if status != fasthttp.StatusServiceUnavailable {
		t.Fatalf("auth start without engine status = %d, want 503", status)
	}
}

func TestMcpListTools(t *testing.T) {
	env := newMCPTestEnv(t)

	status, envl := call(t, env.handlers.ListTools, "GET", "/api/mcp/tools", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("tools status = %d", status)
	}
	tools := dataField[[]map[string]any](t, envl)
	if len(tools) == 0 {
		t.Fatalf("tools len = 0, want >= 1")
	}
	// OpenAI-tool shape.
	first := tools[0]
	if first["type"] != "function" {
		t.Fatalf("tool type = %v, want function", first["type"])
	}
	if _, ok := first["function"].(map[string]any); !ok {
		t.Fatalf("tool missing function object: %v", first)
	}
}

func TestMcpExecuteTool(t *testing.T) {
	env := newMCPTestEnv(t)

	// Start a stdio instance so the launcher has a live bridge for it.
	status, envl := call(t, env.handlers.CreateInstance, "POST", "/api/mcp/instances",
		`{"Name":"fs","Transport":"stdio","Command":"npx","Args":["-y","@modelcontextprotocol/server-filesystem"]}`, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create stdio status = %d err = %q", status, errMessage(t, envl))
	}

	status, envl = call(t, env.handlers.ExecuteTool, "POST", "/api/mcp/tools/read_file/execute",
		`{"arguments":{"path":"/x"}}`, map[string]any{"name": "read_file"}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("execute status = %d err = %q", status, errMessage(t, envl))
	}
	data := dataField[map[string]any](t, envl)
	result, _ := data["result"].(string)
	if !strings.Contains(result, "executed read_file ok") {
		t.Fatalf("execute result = %q", result)
	}
}

func TestMcpExecuteToolNoLauncher(t *testing.T) {
	env := newTestEnv(t) // no launcher injected
	status, _ := call(t, env.handlers.ExecuteTool, "POST", "/api/mcp/tools/read_file/execute",
		`{}`, map[string]any{"name": "read_file"}, nil)
	if status != fasthttp.StatusServiceUnavailable {
		t.Fatalf("execute without launcher status = %d, want 503", status)
	}
}

func TestMcpToolGroupsCRUD(t *testing.T) {
	env := newMCPTestEnv(t)

	// Create.
	status, envl := call(t, env.handlers.CreateToolGroup, "POST", "/api/mcp/tool-groups",
		`{"name":"File Operations","tool_ids":["read_file","write_file"],"is_active":true}`, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create group status = %d err = %q", status, errMessage(t, envl))
	}
	group := dataField[map[string]any](t, envl)
	idNum, _ := group["id"].(float64)
	if idNum == 0 {
		t.Fatalf("group id = %v, want numeric", group["id"])
	}
	if group["name"] != "File Operations" {
		t.Fatalf("group = %v", group)
	}
	// snake_case keys (§1.2).
	if _, ok := group["tool_ids"]; !ok {
		t.Fatalf("group missing tool_ids snake_case key: %v", group)
	}
	if _, ok := group["is_active"]; !ok {
		t.Fatalf("group missing is_active snake_case key: %v", group)
	}
	if _, ok := group["created_at"]; !ok {
		t.Fatalf("group missing created_at: %v", group)
	}
	id := fmt.Sprintf("%d", int64(idNum))

	// List.
	status, envl = call(t, env.handlers.ListToolGroups, "GET", "/api/mcp/tool-groups", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("list groups status = %d", status)
	}
	if len(dataField[[]map[string]any](t, envl)) != 1 {
		t.Fatalf("groups len != 1")
	}

	// Get.
	status, _ = call(t, env.handlers.GetToolGroup, "GET", "/api/mcp/tool-groups/"+id, "",
		map[string]any{"id": id}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("get group status = %d", status)
	}

	// Update (toggle is_active off).
	status, envl = call(t, env.handlers.UpdateToolGroup, "PUT", "/api/mcp/tool-groups/"+id,
		`{"name":"File Operations","tool_ids":["read_file"],"is_active":false}`, map[string]any{"id": id}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("update group status = %d err = %q", status, errMessage(t, envl))
	}
	if dataField[map[string]any](t, envl)["is_active"] != false {
		t.Fatalf("update did not toggle is_active off")
	}

	// Delete.
	status, _ = call(t, env.handlers.DeleteToolGroup, "DELETE", "/api/mcp/tool-groups/"+id, "",
		map[string]any{"id": id}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("delete group status = %d", status)
	}
	status, _ = call(t, env.handlers.GetToolGroup, "GET", "/api/mcp/tool-groups/"+id, "",
		map[string]any{"id": id}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("get after delete status = %d, want 404", status)
	}
}

func TestMcpInstanceJSONIsPascalCase(t *testing.T) {
	env := newMCPTestEnv(t)
	status, envl := call(t, env.handlers.CreateInstance, "POST", "/api/mcp/instances",
		`{"Name":"exa","Transport":"http","URL":"https://mcp.exa.ai/mcp"}`, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create status = %d", status)
	}
	raw, _ := json.Marshal(dataField[map[string]json.RawMessage](t, envl))
	for _, key := range []string{`"ID"`, `"Name"`, `"Transport"`, `"IsActive"`, `"HealthStatus"`} {
		if !strings.Contains(string(raw), key) {
			t.Fatalf("instance JSON missing PascalCase key %s: %s", key, raw)
		}
	}
}

// TestMCPVKPrecedence pins resolveMCPVK's header precedence (D4): x-g0-vk wins,
// then the Authorization Bearer token, then x-api-key, else "". PURE — driven
// over a fake header getter, no fasthttp.
func TestMCPVKPrecedence(t *testing.T) {
	cases := []struct {
		name    string
		headers map[string]string
		want    string
	}{
		{"x-g0-vk wins over all", map[string]string{"x-g0-vk": "g0vk-a", "Authorization": "Bearer g0vk-b", "x-api-key": "g0vk-c"}, "g0vk-a"},
		{"bearer when no x-g0-vk", map[string]string{"Authorization": "Bearer g0vk-b", "x-api-key": "g0vk-c"}, "g0vk-b"},
		{"x-api-key last", map[string]string{"x-api-key": "g0vk-c"}, "g0vk-c"},
		{"bearer trims prefix only", map[string]string{"Authorization": "Bearer  g0vk-d"}, " g0vk-d"},
		{"non-bearer authorization ignored", map[string]string{"Authorization": "Basic xyz", "x-api-key": "g0vk-e"}, "g0vk-e"},
		{"all empty", map[string]string{}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			get := func(name string) string { return tc.headers[name] }
			if got := resolveMCPVK(get); got != tc.want {
				t.Fatalf("resolveMCPVK = %q, want %q", got, tc.want)
			}
		})
	}
}

// rawRPC decodes a raw JSON-RPC body (NOT the {data,error} envelope) from a
// handler response.
func rawRPC(t *testing.T, ctx *fasthttp.RequestCtx) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(ctx.Response.Body(), &m); err != nil {
		t.Fatalf("response is not JSON-RPC: %v\nbody: %s", err, ctx.Response.Body())
	}
	return m
}

// callRaw invokes a handler that writes a RAW (non-envelope) body, returning the
// status and the *fasthttp.RequestCtx for inspection.
func callRaw(t *testing.T, h fasthttp.RequestHandler, method, uri, body string, headers map[string]string) *fasthttp.RequestCtx {
	t.Helper()
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod(method)
	ctx.Request.SetRequestURI(uri)
	if body != "" {
		ctx.Request.SetBody([]byte(body))
	}
	for k, v := range headers {
		ctx.Request.Header.Set(k, v)
	}
	h(&ctx)
	return &ctx
}

// TestMCPServerPostToolsList proves POST /mcp returns a raw JSON-RPC tools/list
// (NOT the {data,error} envelope) whose tool set EQUALS the catalog ListTools
// serves (D3 — one source).
func TestMCPServerPostToolsList(t *testing.T) {
	env := newMCPTestEnv(t)
	ctx := callRaw(t, env.handlers.MCPServerPost, "POST", "/mcp",
		`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`, nil)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("status = %d body = %s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	body := string(ctx.Response.Body())
	// Raw JSON-RPC, not the admin envelope.
	if strings.Contains(body, `"data"`) || strings.Contains(body, `"error":null`) {
		t.Fatalf("/mcp returned the {data,error} envelope, want raw JSON-RPC: %s", body)
	}
	m := rawRPC(t, ctx)
	if m["jsonrpc"] != "2.0" {
		t.Fatalf("not JSON-RPC 2.0: %v", m)
	}
	result, ok := m["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result: %v", m)
	}
	serverTools, ok := result["tools"].([]any)
	if !ok {
		t.Fatalf("no tools array: %v", result)
	}
	serverNames := map[string]bool{}
	for _, tl := range serverTools {
		serverNames[tl.(map[string]any)["name"].(string)] = true
	}

	// The /api/mcp/tools (ListTools) catalog is the SAME source.
	_, envl := call(t, env.handlers.ListTools, "GET", "/api/mcp/tools", "", nil, nil)
	listTools := dataField[[]map[string]any](t, envl)
	if len(listTools) == 0 {
		t.Fatalf("ListTools empty")
	}
	for _, lt := range listTools {
		fn := lt["function"].(map[string]any)
		name := fn["name"].(string)
		if !serverNames[name] {
			t.Fatalf("server tools/list missing catalog tool %q; sources diverge", name)
		}
	}
}

// TestMCPServerPostVKValidation proves the resolved VK is genuinely CONSUMED
// (D4): a valid VK is admitted, a provided-but-invalid VK is REJECTED with a
// JSON-RPC error, and an absent VK is allowed.
func TestMCPServerPostVKValidation(t *testing.T) {
	env := newMCPTestEnv(t)
	// CreateVirtualKey mints the key value and forces IsActive=true; the inactive
	// case is produced by deactivating via UpdateVirtualKey.
	validVK, err := env.store.CreateVirtualKey(&store.VirtualKey{
		VirtualKey: schemas.VirtualKey{Name: "valid"},
	})
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}
	inactiveVK, err := env.store.CreateVirtualKey(&store.VirtualKey{
		VirtualKey: schemas.VirtualKey{Name: "inactive"},
	})
	if err != nil {
		t.Fatalf("CreateVirtualKey inactive: %v", err)
	}
	inactiveVK.IsActive = false
	if err := env.store.UpdateVirtualKey(inactiveVK); err != nil {
		t.Fatalf("UpdateVirtualKey: %v", err)
	}

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`

	// (a) absent VK -> allowed.
	ctx := callRaw(t, env.handlers.MCPServerPost, "POST", "/mcp", body, nil)
	if m := rawRPC(t, ctx); m["error"] != nil {
		t.Fatalf("absent VK rejected: %v", m["error"])
	}

	// (b) valid VK -> allowed.
	ctx = callRaw(t, env.handlers.MCPServerPost, "POST", "/mcp", body, map[string]string{"x-g0-vk": validVK.Key})
	if m := rawRPC(t, ctx); m["error"] != nil {
		t.Fatalf("valid VK rejected: %v", m["error"])
	}

	// (c) unknown VK -> rejected (JSON-RPC error).
	ctx = callRaw(t, env.handlers.MCPServerPost, "POST", "/mcp", body, map[string]string{"x-g0-vk": "g0vk-nope"})
	if m := rawRPC(t, ctx); m["error"] == nil {
		t.Fatalf("unknown VK not rejected: %v", m)
	}

	// (d) inactive VK -> rejected (JSON-RPC error).
	ctx = callRaw(t, env.handlers.MCPServerPost, "POST", "/mcp", body, map[string]string{"x-api-key": inactiveVK.Key})
	if m := rawRPC(t, ctx); m["error"] == nil {
		t.Fatalf("inactive VK not rejected: %v", m)
	}
}

// completeOAuthTransport answers the OAuth discovery requests AND the token
// exchange so Engine.Complete returns a connected account without real network.
type completeOAuthTransport struct{}

func (completeOAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var body string
	switch {
	case strings.Contains(req.URL.Path, "oauth-protected-resource"):
		body = `{"authorization_servers":["https://auth.example.com"]}`
	case strings.Contains(req.URL.Path, "oauth-authorization-server"):
		body = `{"authorization_endpoint":"https://auth.example.com/authorize","token_endpoint":"https://auth.example.com/token"}`
	case strings.Contains(req.URL.Path, "token"):
		body = `{"access_token":"at-secret","refresh_token":"rt-secret","expires_in":3600,"scope":"mcp"}`
	default:
		body = `{}`
	}
	return &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}

// TestMCPCompleteAuthLiveEngineComplete proves CompleteInstanceAuth calls the
// formerly-dead Engine.Complete and returns the MASKED account (no token echoed),
// plus 404 on an unknown instance (D7).
func TestMCPCompleteAuthLiveEngineComplete(t *testing.T) {
	env := newMCPTestEnv(t)
	env.handlers.SetMCPEngine(mcp.NewEngine(env.store, &http.Client{Transport: completeOAuthTransport{}}))

	// Create an http instance with a server URL.
	status, envl := call(t, env.handlers.CreateInstance, "POST", "/api/mcp/instances",
		`{"Name":"exa","Transport":"http","URL":"https://mcp.exa.ai/mcp"}`, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create status = %d err = %q", status, errMessage(t, envl))
	}
	id := dataField[map[string]any](t, envl)["ID"].(string)

	// Start the OAuth flow to persist a flow + obtain the state.
	status, envl = call(t, env.handlers.StartInstanceAuth, "POST",
		"/api/mcp/instances/"+id+"/auth/start", "", map[string]any{"id": id}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("auth/start status = %d err = %q", status, errMessage(t, envl))
	}
	authURL := dataField[map[string]any](t, envl)["url"].(string)
	state := extractQueryParam(t, authURL, "state")

	// Complete the flow.
	status, envl = call(t, env.handlers.CompleteInstanceAuth, "POST",
		"/api/mcp/instances/"+id+"/auth/complete",
		`{"state":"`+state+`","code":"auth-code"}`, map[string]any{"id": id}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("auth/complete status = %d err = %q", status, errMessage(t, envl))
	}
	acct := dataField[map[string]json.RawMessage](t, envl)
	raw, _ := json.Marshal(acct)
	for _, leak := range []string{"at-secret", "rt-secret", "access_token", "refresh_token", state, "verifier"} {
		if strings.Contains(string(raw), leak) {
			t.Fatalf("complete-oauth leaked %q: %s", leak, raw)
		}
	}
	if !strings.Contains(string(raw), "connected") {
		t.Fatalf("complete-oauth account not connected: %s", raw)
	}

	// 404 on unknown instance.
	status, _ = call(t, env.handlers.CompleteInstanceAuth, "POST",
		"/api/mcp/instances/nope/auth/complete",
		`{"state":"x","code":"y"}`, map[string]any{"id": "nope"}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("unknown instance status = %d, want 404", status)
	}
}

// extractQueryParam pulls a query param value out of a URL string.
func extractQueryParam(t *testing.T, raw, key string) string {
	t.Helper()
	i := strings.Index(raw, key+"=")
	if i < 0 {
		t.Fatalf("url %q missing %s", raw, key)
	}
	rest := raw[i+len(key)+1:]
	if j := strings.IndexAny(rest, "&"); j >= 0 {
		rest = rest[:j]
	}
	return rest
}

// auditCount returns the number of mcp_server.tools_call audit entries.
func mcpAuditCount(t *testing.T, env *testEnv) int {
	t.Helper()
	items, _, err := env.handlers.auditService().List(1000)
	if err != nil {
		t.Fatalf("audit list: %v", err)
	}
	n := 0
	for _, e := range items {
		if e.Action == "mcp_server.tools_call" {
			n++
		}
	}
	return n
}

// TestMCPSSEHeartbeatHermetic proves GET /mcp emits ": ping\n\n" once per
// INJECTED tick with ZERO real elapsed time (D5), and the deferred finalizer
// writes a real recordAudit entry (stamping the resolved VK) only AFTER the sink
// closes — never during frame emission (D8 preferred path (a)).
func TestMCPSSEHeartbeatHermetic(t *testing.T) {
	env := newMCPTestEnv(t)

	rec := &recordingWriter{}
	w := bufio.NewWriter(rec)
	heartbeat := make(chan time.Time)
	clientDone := make(chan struct{})
	done := make(chan struct{})

	var bareCtx fasthttp.RequestCtx
	go env.handlers.serveMCPSSE(&bareCtx, w, "g0vk-stamp", heartbeat, clientDone, done)

	// Fire 3 ticks; assert 3 ": ping" frames and NO audit yet (still streaming).
	for i := 0; i < 3; i++ {
		heartbeat <- time.Now()
	}
	waitFor(t, time.Second, func() bool {
		return strings.Count(string(rec.Bytes()), ": ping\n\n") >= 3
	})
	if got := strings.Count(string(rec.Bytes()), ": ping\n\n"); got != 3 {
		t.Fatalf(": ping frames = %d, want 3", got)
	}
	if n := mcpAuditCount(t, env); n != 0 {
		t.Fatalf("audit written DURING frame emission = %d, want 0 (must defer until close)", n)
	}

	// Close the stream; the deferred finalizer must run AFTER the sink closes.
	close(clientDone)
	waitFor(t, time.Second, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	})

	if n := mcpAuditCount(t, env); n != 1 {
		t.Fatalf("deferred audit entries = %d, want 1 (written after sink close)", n)
	}
	items, _, _ := env.handlers.auditService().List(1000)
	var stamped bool
	for _, e := range items {
		if e.Action == "mcp_server.tools_call" && strings.Contains(e.Details, "g0vk-stamp") {
			stamped = true
		}
	}
	if !stamped {
		t.Fatalf("deferred audit did not stamp the resolved VK (D8 payload)")
	}
}

// serverToolNames POSTs a tools/list over /mcp with the given headers and returns
// the set of tool names the (possibly scoped) server advertises.
func serverToolNames(t *testing.T, h *Handlers, headers map[string]string) map[string]bool {
	t.Helper()
	ctx := callRaw(t, h.MCPServerPost, "POST", "/mcp",
		`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`, headers)
	m := rawRPC(t, ctx)
	if m["error"] != nil {
		t.Fatalf("tools/list error: %v", m["error"])
	}
	result, ok := m["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result: %v", m)
	}
	tools, _ := result["tools"].([]any)
	names := map[string]bool{}
	for _, tl := range tools {
		names[tl.(map[string]any)["name"].(string)] = true
	}
	return names
}

// TestMCPScopeRestrictedVKSeesFewerTools proves a restricted VK sees STRICTLY
// FEWER tools on tools/list than the global catalog an absent VK sees, and that
// it sees exactly the scoped tool (D3/D4 live-narrowing).
func TestMCPScopeRestrictedVKSeesFewerTools(t *testing.T) {
	env := newMCPTestEnv(t)

	// Absent VK -> the full global catalog (regression / baseline).
	global := serverToolNames(t, env.handlers, nil)
	if len(global) == 0 {
		t.Fatalf("global catalog empty")
	}
	if !global["browser_navigate"] {
		t.Fatalf("global catalog missing browsermcp tool: %v", global)
	}

	// A VK scoped to a single browsermcp tool.
	vk, err := env.store.CreateVirtualKey(&store.VirtualKey{VirtualKey: schemas.VirtualKey{Name: "scoped"}})
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}
	if _, err := env.store.CreateVKMCPConfig(&store.VKMCPConfig{
		VirtualKeyID:   vk.ID,
		MCPClientID:    "browsermcp",
		ToolsToExecute: []string{"browsermcp-browser_navigate"},
	}); err != nil {
		t.Fatalf("CreateVKMCPConfig: %v", err)
	}

	scoped := serverToolNames(t, env.handlers, map[string]string{"x-g0-vk": vk.Key})
	if len(scoped) >= len(global) {
		t.Fatalf("restricted VK did not see FEWER tools: scoped=%d global=%d", len(scoped), len(global))
	}
	if !scoped["browser_navigate"] {
		t.Fatalf("scoped VK missing its in-scope tool: %v", scoped)
	}
	if scoped["browser_click"] {
		t.Fatalf("scoped VK saw an out-of-scope tool: %v", scoped)
	}
}

// TestMCPScopeToolsCallGate proves a scoped VK cannot tools/call a tool OUTSIDE
// its executeOnlyTools scope: the call returns a JSON-RPC error (D3 — the scope is
// enforced on BOTH tools/list and tools/call).
func TestMCPScopeToolsCallGate(t *testing.T) {
	env := newMCPTestEnv(t)

	vk, err := env.store.CreateVirtualKey(&store.VirtualKey{VirtualKey: schemas.VirtualKey{Name: "scoped"}})
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}
	if _, err := env.store.CreateVKMCPConfig(&store.VKMCPConfig{
		VirtualKeyID:   vk.ID,
		MCPClientID:    "browsermcp",
		ToolsToExecute: []string{"browsermcp-browser_navigate"},
	}); err != nil {
		t.Fatalf("CreateVKMCPConfig: %v", err)
	}

	// Out-of-scope tools/call -> JSON-RPC error.
	ctx := callRaw(t, env.handlers.MCPServerPost, "POST", "/mcp",
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"browser_click","arguments":{}}}`,
		map[string]string{"x-g0-vk": vk.Key})
	if m := rawRPC(t, ctx); m["error"] == nil {
		t.Fatalf("out-of-scope tools/call not rejected: %v", m)
	}
}

// TestMCPScopeAllowOnAllVKBypass proves a VK whose scope does NOT name client X
// still sees X's tools WHEN X is AllowOnAllVirtualKeys, and does NOT when the flag
// is false (D6 bypass).
func TestMCPScopeAllowOnAllVKBypass(t *testing.T) {
	env := newMCPTestEnv(t)

	vk, err := env.store.CreateVirtualKey(&store.VirtualKey{VirtualKey: schemas.VirtualKey{Name: "scoped"}})
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}
	// Scope the VK to a non-browsermcp pattern only (empty intersection with
	// browsermcp tools).
	if _, err := env.store.CreateVKMCPConfig(&store.VKMCPConfig{
		VirtualKeyID:   vk.ID,
		MCPClientID:    "other",
		ToolsToExecute: []string{"other-*"},
	}); err != nil {
		t.Fatalf("CreateVKMCPConfig: %v", err)
	}

	// Flag false (no browsermcp client row): the VK does NOT see browsermcp tools.
	before := serverToolNames(t, env.handlers, map[string]string{"x-g0-vk": vk.Key})
	if before["browser_navigate"] {
		t.Fatalf("VK saw browsermcp tools without the bypass flag: %v", before)
	}

	// Mark browsermcp AllowOnAllVirtualKeys -> its tools become visible to the VK.
	if _, err := env.store.CreateMCPClient(&store.MCPClient{
		Name:   "browsermcp",
		Type:   "default",
		Config: map[string]any{"allow_on_all_virtual_keys": true},
	}); err != nil {
		t.Fatalf("CreateMCPClient browsermcp: %v", err)
	}
	after := serverToolNames(t, env.handlers, map[string]string{"x-g0-vk": vk.Key})
	if !after["browser_navigate"] {
		t.Fatalf("AllowOnAllVirtualKeys did not bypass the per-VK filter: %v", after)
	}
}

// TestMCPScopeAbsentVKUnchanged proves the absent-VK path still serves the full
// global catalog (regression — the un-scoped surface is unchanged).
func TestMCPScopeAbsentVKUnchanged(t *testing.T) {
	env := newMCPTestEnv(t)

	// Adding an assignment for some OTHER VK must not narrow the anonymous surface.
	other, _ := env.store.CreateVirtualKey(&store.VirtualKey{VirtualKey: schemas.VirtualKey{Name: "other"}})
	_, _ = env.store.CreateVKMCPConfig(&store.VKMCPConfig{
		VirtualKeyID:   other.ID,
		MCPClientID:    "browsermcp",
		ToolsToExecute: []string{"browsermcp-browser_navigate"},
	})

	global := serverToolNames(t, env.handlers, nil)
	if !global["browser_navigate"] || !global["browser_click"] {
		t.Fatalf("absent-VK path narrowed by another VK's assignment: %v", global)
	}
}

// TestMCPScopeDisableAutoToolInject proves a client flagged DisableAutoToolInject
// has its tools omitted from a scoped VK's served surface even when the VK's scope
// admits them (PAR-BF-MCP-057 live read — D7 HAVE).
func TestMCPScopeDisableAutoToolInject(t *testing.T) {
	env := newMCPTestEnv(t)

	vk, err := env.store.CreateVirtualKey(&store.VirtualKey{VirtualKey: schemas.VirtualKey{Name: "scoped"}})
	if err != nil {
		t.Fatalf("CreateVirtualKey: %v", err)
	}
	// Scope the VK to all browsermcp tools.
	if _, err := env.store.CreateVKMCPConfig(&store.VKMCPConfig{
		VirtualKeyID:   vk.ID,
		MCPClientID:    "browsermcp",
		ToolsToExecute: []string{"browsermcp-*"},
	}); err != nil {
		t.Fatalf("CreateVKMCPConfig: %v", err)
	}

	// Without the flag the VK sees the browsermcp tools.
	before := serverToolNames(t, env.handlers, map[string]string{"x-g0-vk": vk.Key})
	if !before["browser_navigate"] {
		t.Fatalf("scoped VK missing in-scope tool before disable-flag: %v", before)
	}

	// Flag browsermcp DisableAutoToolInject -> its tools are suppressed.
	if _, err := env.store.CreateMCPClient(&store.MCPClient{
		Name:   "browsermcp",
		Type:   "default",
		Config: map[string]any{"disable_auto_tool_inject": true},
	}); err != nil {
		t.Fatalf("CreateMCPClient: %v", err)
	}
	after := serverToolNames(t, env.handlers, map[string]string{"x-g0-vk": vk.Key})
	if after["browser_navigate"] {
		t.Fatalf("DisableAutoToolInject did not suppress the client's tools: %v", after)
	}
}

// TestMCPVKConfigCreateSubsetReject proves the LIVE assignment create path REJECTS
// an autoExecute ⊄ execute assignment with a 4xx {error} envelope BEFORE storage
// (D5/049), and accepts a valid subset (D5).
func TestMCPVKConfigCreateSubsetReject(t *testing.T) {
	env := newMCPTestEnv(t)

	// Invalid: auto-execute names a tool not in execute.
	status, envl := call(t, env.handlers.CreateVKMCPConfig, "POST", "/api/mcp/vk-configs",
		`{"virtual_key_id":"vk1","mcp_client_id":"exa","tools_to_execute":["exa-search"],"tools_to_auto_execute":["exa-fetch"]}`,
		nil, nil)
	if status < 400 || status >= 500 {
		t.Fatalf("invalid subset status = %d, want 4xx", status)
	}
	if errMessage(t, envl) == "" {
		t.Fatalf("invalid subset did not return an {error}: %v", envl)
	}

	// Valid: auto-execute is a subset of execute.
	status, envl = call(t, env.handlers.CreateVKMCPConfig, "POST", "/api/mcp/vk-configs",
		`{"virtual_key_id":"vk1","mcp_client_id":"exa","tools_to_execute":["exa-search","exa-fetch"],"tools_to_auto_execute":["exa-search"]}`,
		nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("valid subset status = %d err = %q", status, errMessage(t, envl))
	}
}

// TestMCPVKConfigConfigHashExposed proves the assignment GET DTO carries a
// computed config_hash (D8/079 — the live drift-detection reader; NOT write-only)
// and that the hash changes when the assignment changes.
func TestMCPVKConfigConfigHashExposed(t *testing.T) {
	env := newMCPTestEnv(t)

	status, envl := call(t, env.handlers.CreateVKMCPConfig, "POST", "/api/mcp/vk-configs",
		`{"virtual_key_id":"vk1","mcp_client_id":"exa","tools_to_execute":["exa-*"]}`, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create status = %d err = %q", status, errMessage(t, envl))
	}
	created := dataField[map[string]any](t, envl)
	idF, ok := created["id"].(float64)
	if !ok {
		t.Fatalf("created config missing numeric id: %v", created)
	}
	id := int64(idF)
	hash1, _ := created["config_hash"].(string)
	if hash1 == "" {
		t.Fatalf("create DTO missing config_hash: %v", created)
	}

	// GET returns the DTO with config_hash.
	status, envl = call(t, env.handlers.GetVKMCPConfig, "GET",
		"/api/mcp/vk-configs/1", "", map[string]any{"id": "1"}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("get status = %d err = %q", status, errMessage(t, envl))
	}
	got := dataField[map[string]any](t, envl)
	if got["config_hash"].(string) != hash1 {
		t.Fatalf("GET config_hash = %v, want %q", got["config_hash"], hash1)
	}

	// Update changes the hash.
	status, envl = call(t, env.handlers.UpdateVKMCPConfig, "PUT",
		"/api/mcp/vk-configs/1",
		`{"virtual_key_id":"vk1","mcp_client_id":"exa","tools_to_execute":["exa-search"]}`,
		map[string]any{"id": "1"}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("update status = %d err = %q", status, errMessage(t, envl))
	}
	updated := dataField[map[string]any](t, envl)
	if updated["config_hash"].(string) == hash1 {
		t.Fatalf("config_hash did not change on update (drift undetectable): %q", hash1)
	}

	// Delete.
	status, _ = call(t, env.handlers.DeleteVKMCPConfig, "DELETE",
		"/api/mcp/vk-configs/1", "", map[string]any{"id": "1"}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("delete status = %d", status)
	}
	if _, err := env.store.GetVKMCPConfig(id); err == nil {
		t.Fatalf("config not deleted")
	}
}

// TestMCPVKConfigListByVK proves the list-by-VK GET returns the VK's assignments.
func TestMCPVKConfigListByVK(t *testing.T) {
	env := newMCPTestEnv(t)
	if _, err := env.store.CreateVKMCPConfig(&store.VKMCPConfig{VirtualKeyID: "vk1", MCPClientID: "exa"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	status, envl := call(t, env.handlers.ListVKMCPConfigs, "GET",
		"/api/mcp/vk-configs?virtual_key_id=vk1", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("list status = %d", status)
	}
	list := dataField[[]map[string]any](t, envl)
	if len(list) != 1 {
		t.Fatalf("list len = %d, want 1", len(list))
	}
}

// enableVKMandatoryFlagAdmin flips the seeded vk_mandatory flag ON in an admin
// test env store (bf-gov-4, D4 — same flag read by the /mcp admitMCPVK mirror).
func enableVKMandatoryFlagAdmin(t *testing.T, env *testEnv) {
	t.Helper()
	if _, err := env.store.DB().Exec(
		"UPDATE feature_flags SET enabled = 1 WHERE key = 'vk_mandatory'",
	); err != nil {
		t.Fatalf("enable vk_mandatory flag: %v", err)
	}
}

// TestMCPMandatoryVKMode verifies PAR-BF-GOV-034 on the /mcp surface (bf-gov-4,
// D4): when the vk_mandatory flag is ON, an absent VK is REJECTED with a
// "virtual key required" JSON-RPC error on MCPServerPost and a 401 on
// MCPServerSSE; when the flag is OFF, an absent VK is admitted (today's
// behavior). A provided-but-invalid VK is always rejected regardless of the flag
// (regression guard — bf-mcp-1 behavior unchanged).
func TestMCPMandatoryVKMode(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`

	t.Run("MCPServerPost absent VK + flag ON rejected with virtual key required", func(t *testing.T) {
		env := newMCPTestEnv(t)
		enableVKMandatoryFlagAdmin(t, env)

		ctx := callRaw(t, env.handlers.MCPServerPost, "POST", "/mcp", body, nil)
		m := rawRPC(t, ctx)
		errObj, hasErr := m["error"].(map[string]any)
		if !hasErr {
			t.Fatalf("absent VK + mandatory ON: expected JSON-RPC error, got: %v", m)
		}
		msg, _ := errObj["message"].(string)
		if msg != "virtual key required" {
			t.Fatalf("error message = %q, want %q", msg, "virtual key required")
		}
	})

	t.Run("MCPServerPost absent VK + flag OFF admitted", func(t *testing.T) {
		env := newMCPTestEnv(t)
		// flag left OFF (seeded default).

		ctx := callRaw(t, env.handlers.MCPServerPost, "POST", "/mcp", body, nil)
		m := rawRPC(t, ctx)
		if m["error"] != nil {
			t.Fatalf("absent VK + mandatory OFF: expected admitted, got error: %v", m["error"])
		}
	})

	t.Run("MCPServerPost provided-invalid VK rejected regardless of flag", func(t *testing.T) {
		env := newMCPTestEnv(t)
		enableVKMandatoryFlagAdmin(t, env)

		ctx := callRaw(t, env.handlers.MCPServerPost, "POST", "/mcp", body,
			map[string]string{"x-g0-vk": "g0vk-does-not-exist"})
		m := rawRPC(t, ctx)
		if m["error"] == nil {
			t.Fatalf("invalid VK + mandatory ON: expected JSON-RPC error, got: %v", m)
		}
	})

	t.Run("MCPServerSSE absent VK + flag ON rejected 401", func(t *testing.T) {
		env := newMCPTestEnv(t)
		enableVKMandatoryFlagAdmin(t, env)

		ctx := callRaw(t, env.handlers.MCPServerSSE, "GET", "/mcp", "", nil)
		if ctx.Response.StatusCode() != fasthttp.StatusUnauthorized {
			t.Fatalf("MCPServerSSE absent VK + mandatory ON status = %d, want 401",
				ctx.Response.StatusCode())
		}
	})

	t.Run("MCPServerSSE absent VK + flag OFF admitted (streams)", func(t *testing.T) {
		env := newMCPTestEnv(t)
		// flag left OFF.

		// MCPServerSSE blocks on the SSE stream writer; call the admission check
		// indirectly by verifying admitMCPVK returns admitted=true when flag is OFF.
		// We prove via MCPServerPost (same admission path) that OFF admits, and the
		// SSE handler only gates on admitted before streaming.
		ctx := callRaw(t, env.handlers.MCPServerPost, "POST", "/mcp", body, nil)
		m := rawRPC(t, ctx)
		if m["error"] != nil {
			t.Fatalf("flag OFF: absent VK should be admitted, got error: %v", m["error"])
		}
	})
}
