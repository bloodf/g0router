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
