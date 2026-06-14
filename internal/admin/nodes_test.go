package admin

import (
	"encoding/json"
	"testing"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

// seedProvider creates a provider of the given type and returns its id.
func seedProvider(t *testing.T, env *testEnv, name, typ, baseURL string) string {
	t.Helper()
	body := `{"name":"` + name + `","type":"` + typ + `","base_url":"` + baseURL + `","enabled":true}`
	status, envl := call(t, env.handlers.CreateProvider, "POST", "/api/providers", body, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("seed provider %q status = %d err = %q", name, status, errMessage(t, envl))
	}
	return dataField[map[string]any](t, envl)["id"].(string)
}

func TestListProviderNodesFiltersOpenAICompatible(t *testing.T) {
	env := newTestEnv(t)
	seedProvider(t, env, "OpenAI", "openai", "https://api.openai.com/v1")
	compatID := seedProvider(t, env, "Local LM", "openai-compatible", "http://localhost:1234/v1")

	status, envl := call(t, env.handlers.ListProviderNodes, "GET", "/api/provider-nodes", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("list status = %d err = %q", status, errMessage(t, envl))
	}
	data := dataField[map[string]json.RawMessage](t, envl)
	var nodes []map[string]any
	if err := json.Unmarshal(data["nodes"], &nodes); err != nil {
		t.Fatalf("decode nodes: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected exactly one openai-compatible node, got %d: %v", len(nodes), nodes)
	}
	n := nodes[0]
	if n["id"] != compatID {
		t.Fatalf("node id = %v, want %v", n["id"], compatID)
	}
	if n["type"] != "openai-compatible" {
		t.Fatalf("node type = %v, want openai-compatible", n["type"])
	}
	if n["base_url"] != "http://localhost:1234/v1" {
		t.Fatalf("node base_url = %v", n["base_url"])
	}
	if n["name"] != "Local LM" {
		t.Fatalf("node name = %v", n["name"])
	}
}

func TestCreateProviderNodePersists(t *testing.T) {
	env := newTestEnv(t)

	// Accept camelCase baseUrl from the 9router client.
	body := `{"name":"My Node","prefix":"mn","apiType":"openai","baseUrl":"https://node.example.com/v1"}`
	status, envl := call(t, env.handlers.CreateProviderNode, "POST", "/api/provider-nodes", body, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create status = %d err = %q", status, errMessage(t, envl))
	}
	data := dataField[map[string]json.RawMessage](t, envl)
	var node map[string]any
	if err := json.Unmarshal(data["node"], &node); err != nil {
		t.Fatalf("decode node: %v", err)
	}
	if node["id"] == "" || node["id"] == nil {
		t.Fatalf("node missing id: %v", node)
	}
	if node["name"] != "My Node" || node["base_url"] != "https://node.example.com/v1" {
		t.Fatalf("node = %v", node)
	}
	if node["type"] != "openai-compatible" {
		t.Fatalf("node type = %v, want openai-compatible", node["type"])
	}

	// The node is now listed.
	status, envl = call(t, env.handlers.ListProviderNodes, "GET", "/api/provider-nodes", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("list after create status = %d", status)
	}
	listData := dataField[map[string]json.RawMessage](t, envl)
	var nodes []map[string]any
	if err := json.Unmarshal(listData["nodes"], &nodes); err != nil {
		t.Fatalf("decode nodes: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected one node after create, got %d", len(nodes))
	}
}

func TestCreateProviderNodeSnakeCaseBaseURL(t *testing.T) {
	env := newTestEnv(t)
	body := `{"name":"Snake Node","base_url":"https://snake.example.com/v1"}`
	status, envl := call(t, env.handlers.CreateProviderNode, "POST", "/api/provider-nodes", body, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create status = %d err = %q", status, errMessage(t, envl))
	}
	node := dataField[map[string]json.RawMessage](t, envl)
	var n map[string]any
	if err := json.Unmarshal(node["node"], &n); err != nil {
		t.Fatalf("decode node: %v", err)
	}
	if n["base_url"] != "https://snake.example.com/v1" {
		t.Fatalf("base_url = %v", n["base_url"])
	}
}

func TestCreateProviderNodeValidation(t *testing.T) {
	env := newTestEnv(t)

	status, _ := call(t, env.handlers.CreateProviderNode, "POST", "/api/provider-nodes",
		`{"name":"","baseUrl":"https://x.example.com"}`, nil, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("missing name status = %d, want 400", status)
	}

	status, _ = call(t, env.handlers.CreateProviderNode, "POST", "/api/provider-nodes",
		`{"name":"No URL"}`, nil, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("missing base_url status = %d, want 400", status)
	}

	status, _ = call(t, env.handlers.CreateProviderNode, "POST", "/api/provider-nodes",
		`not-json`, nil, nil)
	if status != fasthttp.StatusBadRequest {
		t.Fatalf("malformed body status = %d, want 400", status)
	}
}

func TestValidateProviderNodeURL(t *testing.T) {
	env := newTestEnv(t)

	status, envl := call(t, env.handlers.ValidateProviderNode, "POST", "/api/provider-nodes/validate",
		`{"base_url":"https://good.example.com/v1","api_key":"sk-secret","type":"openai-compatible"}`, nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("validate status = %d err = %q", status, errMessage(t, envl))
	}
	res := dataField[map[string]any](t, envl)
	if res["valid"] != true {
		t.Fatalf("well-formed url valid = %v, want true", res["valid"])
	}

	// Bad URL → valid:false with an error string.
	status, envl = call(t, env.handlers.ValidateProviderNode, "POST", "/api/provider-nodes/validate",
		`{"base_url":"not a url","type":"openai-compatible"}`, nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("validate bad url status = %d", status)
	}
	res = dataField[map[string]any](t, envl)
	if res["valid"] != false {
		t.Fatalf("bad url valid = %v, want false", res["valid"])
	}
	if res["error"] == nil || res["error"] == "" {
		t.Fatalf("bad url should carry an error: %v", res)
	}

	// camelCase baseUrl is also accepted.
	status, envl = call(t, env.handlers.ValidateProviderNode, "POST", "/api/provider-nodes/validate",
		`{"baseUrl":"https://camel.example.com","type":"openai-compatible"}`, nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("validate camelCase status = %d", status)
	}
	if dataField[map[string]any](t, envl)["valid"] != true {
		t.Fatalf("camelCase baseUrl should be valid")
	}
}

func TestValidateProviderNodeNeverPersistsAPIKey(t *testing.T) {
	env := newTestEnv(t)

	status, _ := call(t, env.handlers.ValidateProviderNode, "POST", "/api/provider-nodes/validate",
		`{"base_url":"https://good.example.com/v1","api_key":"sk-supersecret","type":"openai-compatible"}`, nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("validate status = %d", status)
	}

	// Validate must NOT create any provider row.
	providers, err := env.store.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders: %v", err)
	}
	if len(providers) != 0 {
		t.Fatalf("validate persisted a provider row: %v", providers)
	}
}

// TestNodesRouteDisambiguation proves the fasthttp/router matcher resolves the
// static /api/provider-nodes/validate route distinctly from the bare collection
// /api/provider-nodes route (plan §1.6b / §8 ESC-4).
func TestNodesRouteDisambiguation(t *testing.T) {
	env := newTestEnv(t)
	r := router.New()
	r.GET("/api/provider-nodes", env.handlers.ListProviderNodes)
	r.POST("/api/provider-nodes", env.handlers.CreateProviderNode)
	r.POST("/api/provider-nodes/validate", env.handlers.ValidateProviderNode)

	// GET /api/provider-nodes resolves to the list handler (returns {nodes:[...]}).
	var listCtx fasthttp.RequestCtx
	listCtx.Request.Header.SetMethod("GET")
	listCtx.Request.SetRequestURI("/api/provider-nodes")
	r.Handler(&listCtx)
	if listCtx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("GET /api/provider-nodes status = %d", listCtx.Response.StatusCode())
	}
	var listEnv struct {
		Data struct {
			Nodes []map[string]any `json:"nodes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listCtx.Response.Body(), &listEnv); err != nil {
		t.Fatalf("list body is not a {nodes} envelope: %v\n%s", err, listCtx.Response.Body())
	}

	// POST /api/provider-nodes/validate resolves to the validate handler ({valid}).
	var valCtx fasthttp.RequestCtx
	valCtx.Request.Header.SetMethod("POST")
	valCtx.Request.SetRequestURI("/api/provider-nodes/validate")
	valCtx.Request.SetBody([]byte(`{"base_url":"https://ok.example.com","type":"openai-compatible"}`))
	r.Handler(&valCtx)
	if valCtx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("POST /api/provider-nodes/validate status = %d", valCtx.Response.StatusCode())
	}
	var valEnv struct {
		Data struct {
			Valid bool `json:"valid"`
		} `json:"data"`
	}
	if err := json.Unmarshal(valCtx.Response.Body(), &valEnv); err != nil {
		t.Fatalf("validate body is not a {valid} envelope: %v\n%s", err, valCtx.Response.Body())
	}
	if !valEnv.Data.Valid {
		t.Fatalf("validate of a well-formed url should be valid")
	}
}
