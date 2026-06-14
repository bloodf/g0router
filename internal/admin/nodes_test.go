package admin

import (
	"encoding/json"
	"net"
	"strings"
	"testing"

	"github.com/bloodf/g0router/internal/platform"
	"github.com/bloodf/g0router/internal/store"
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

func TestProviderNodesCreatePersists(t *testing.T) {
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

func TestProviderNodesCreateSnakeCaseBaseURL(t *testing.T) {
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

func TestProviderNodesCreateValidation(t *testing.T) {
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

func TestProviderNodesValidateURL(t *testing.T) {
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

func TestProviderNodesValidateNeverPersistsAPIKey(t *testing.T) {
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

// TestProviderNodesPersistsPrefixAPIType proves the EXTENDED create persists
// prefix/api_type and the list+DTO surface them (w7-platnodes, PAR-PLAT-010).
func TestProviderNodesPersistsPrefixAPIType(t *testing.T) {
	env := newTestEnv(t)

	body := `{"name":"My Node","prefix":"mn","apiType":"openai","baseUrl":"https://node.example.com/v1"}`
	status, envl := call(t, env.handlers.CreateProviderNode, "POST", "/api/provider-nodes", body, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create status = %d err = %q", status, errMessage(t, envl))
	}
	var node map[string]any
	if err := json.Unmarshal(dataField[map[string]json.RawMessage](t, envl)["node"], &node); err != nil {
		t.Fatalf("decode node: %v", err)
	}
	if node["prefix"] != "mn" || node["api_type"] != "openai" {
		t.Fatalf("create DTO missing prefix/api_type: %v", node)
	}
	id, _ := node["id"].(string)

	// Persisted on the providers row.
	rec, err := env.store.GetProvider(id)
	if err != nil {
		t.Fatalf("GetProvider: %v", err)
	}
	if rec.Prefix != "mn" || rec.APIType != "openai" {
		t.Fatalf("row not persisted: %+v", rec)
	}

	// Listed with prefix/api_type.
	status, envl = call(t, env.handlers.ListProviderNodes, "GET", "/api/provider-nodes", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("list status = %d", status)
	}
	var nodes []map[string]any
	if err := json.Unmarshal(dataField[map[string]json.RawMessage](t, envl)["nodes"], &nodes); err != nil {
		t.Fatalf("decode nodes: %v", err)
	}
	if len(nodes) != 1 || nodes[0]["prefix"] != "mn" || nodes[0]["api_type"] != "openai" {
		t.Fatalf("list missing prefix/api_type: %v", nodes)
	}
}

// TestProviderNodesListIncludesAllNodeTypes proves the list filter spans all
// three node types and excludes plain providers (w7-platnodes, PAR-PLAT-010).
func TestProviderNodesListIncludesAllNodeTypes(t *testing.T) {
	env := newTestEnv(t)
	seedProvider(t, env, "OpenAI", "openai", "https://api.openai.com/v1")
	seedProvider(t, env, "Compat", "openai-compatible", "https://compat.example.com/v1")
	seedProvider(t, env, "AnthroNode", "anthropic-compatible", "https://an.example.com")
	seedProvider(t, env, "Embed", "custom-embedding", "https://embed.example.com")

	status, envl := call(t, env.handlers.ListProviderNodes, "GET", "/api/provider-nodes", "", nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("list status = %d", status)
	}
	var nodes []map[string]any
	if err := json.Unmarshal(dataField[map[string]json.RawMessage](t, envl)["nodes"], &nodes); err != nil {
		t.Fatalf("decode nodes: %v", err)
	}
	if len(nodes) != 3 {
		t.Fatalf("expected 3 node-type rows, got %d: %v", len(nodes), nodes)
	}
}

// TestProviderNodesGetUpdateDelete proves the NEW {id} CRUD: get/update/delete +
// 404 (w7-platnodes, PAR-PLAT-010/012).
func TestProviderNodesGetUpdateDelete(t *testing.T) {
	env := newTestEnv(t)

	body := `{"name":"My Node","prefix":"mn","apiType":"openai","baseUrl":"https://node.example.com/v1"}`
	status, envl := call(t, env.handlers.CreateProviderNode, "POST", "/api/provider-nodes", body, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create status = %d", status)
	}
	var created map[string]any
	json.Unmarshal(dataField[map[string]json.RawMessage](t, envl)["node"], &created)
	id := created["id"].(string)

	// Get.
	status, envl = call(t, env.handlers.GetProviderNode, "GET", "/api/provider-nodes/"+id, "", map[string]any{"id": id}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("get status = %d err = %q", status, errMessage(t, envl))
	}
	var got map[string]any
	json.Unmarshal(dataField[map[string]json.RawMessage](t, envl)["node"], &got)
	if got["id"] != id || got["prefix"] != "mn" {
		t.Fatalf("get node = %v", got)
	}

	// Get missing → 404.
	status, _ = call(t, env.handlers.GetProviderNode, "GET", "/api/provider-nodes/missing", "", map[string]any{"id": "missing"}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("get missing status = %d, want 404", status)
	}

	// Update cascades the sanitized base_url + api_type onto the row.
	upd := `{"name":"My Node 2","prefix":"mn","apiType":"anthropic","type":"anthropic-compatible","baseUrl":"https://node2.example.com/v1/messages"}`
	status, envl = call(t, env.handlers.UpdateProviderNode, "PUT", "/api/provider-nodes/"+id, upd, map[string]any{"id": id}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("update status = %d err = %q", status, errMessage(t, envl))
	}
	var updated map[string]any
	json.Unmarshal(dataField[map[string]json.RawMessage](t, envl)["node"], &updated)
	if updated["base_url"] != "https://node2.example.com/v1" || updated["api_type"] != "anthropic" {
		t.Fatalf("update did not cascade/sanitize: %v", updated)
	}

	// Update missing → 404.
	status, _ = call(t, env.handlers.UpdateProviderNode, "PUT", "/api/provider-nodes/missing", upd, map[string]any{"id": "missing"}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("update missing status = %d, want 404", status)
	}

	// Delete + 404 on second delete.
	status, _ = call(t, env.handlers.DeleteProviderNode, "DELETE", "/api/provider-nodes/"+id, "", map[string]any{"id": id}, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("delete status = %d", status)
	}
	status, _ = call(t, env.handlers.DeleteProviderNode, "DELETE", "/api/provider-nodes/"+id, "", map[string]any{"id": id}, nil)
	if status != fasthttp.StatusNotFound {
		t.Fatalf("delete missing status = %d, want 404", status)
	}
}

// TestProviderNodesValidateHermeticProbe proves the real validate runs through the
// injectable prober (no network), preserving the well-formed-URL→valid invariant
// and the no-api-key-leak invariant (w7-platnodes, PAR-PLAT-013).
func TestProviderNodesValidateHermeticProbe(t *testing.T) {
	env := newTestEnv(t)
	env.handlers.SetNodeResolver(func(string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("93.184.216.34")}, nil
	})
	var probedKey string
	env.handlers.SetNodeProber(func(req platform.NodeProbeRequest) (platform.NodeProbeResult, error) {
		probedKey = req.APIKey
		return platform.NodeProbeResult{Valid: true}, nil
	})

	status, envl := call(t, env.handlers.ValidateProviderNode, "POST", "/api/provider-nodes/validate",
		`{"base_url":"https://good.example.com/v1","api_key":"sk-supersecret","type":"openai-compatible"}`, nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("validate status = %d", status)
	}
	res := dataField[map[string]any](t, envl)
	if res["valid"] != true {
		t.Fatalf("valid = %v, want true", res["valid"])
	}
	// Probe received the key transiently.
	if probedKey != "sk-supersecret" {
		t.Fatalf("prober key = %q", probedKey)
	}
	// Response never echoes the key.
	raw, _ := json.Marshal(res)
	if strings.Contains(string(raw), "supersecret") {
		t.Fatalf("validate response leaks api_key: %s", raw)
	}
}

// TestProviderNodesValidateSSRFBlocked proves a base URL resolving to a blocked
// IP is refused without probing (w7-platnodes, PAR-PLAT-013 SSRF).
func TestProviderNodesValidateSSRFBlocked(t *testing.T) {
	env := newTestEnv(t)
	env.handlers.SetNodeResolver(func(string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("127.0.0.1")}, nil
	})
	probed := false
	env.handlers.SetNodeProber(func(req platform.NodeProbeRequest) (platform.NodeProbeResult, error) {
		probed = true
		return platform.NodeProbeResult{Valid: true}, nil
	})

	status, envl := call(t, env.handlers.ValidateProviderNode, "POST", "/api/provider-nodes/validate",
		`{"base_url":"https://internal.example.com","type":"openai-compatible"}`, nil, nil)
	if status != fasthttp.StatusOK {
		t.Fatalf("validate status = %d", status)
	}
	if dataField[map[string]any](t, envl)["valid"] != false {
		t.Fatalf("ssrf-blocked should be valid:false")
	}
	if probed {
		t.Fatalf("prober called for an SSRF-blocked target")
	}
}

// TestProviderNodesCreateProvisionsConnection proves create-with-key provisions a
// bound api_key connection and never echoes the key (w7-platnodes, ESC-PROVISION).
func TestProviderNodesCreateProvisionsConnection(t *testing.T) {
	env := newTestEnv(t)

	body := `{"name":"Keyed","prefix":"kn","apiType":"openai","baseUrl":"https://keyed.example.com/v1","apiKey":"sk-node-secret"}`
	status, envl := call(t, env.handlers.CreateProviderNode, "POST", "/api/provider-nodes", body, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create status = %d err = %q", status, errMessage(t, envl))
	}
	var node map[string]any
	json.Unmarshal(dataField[map[string]json.RawMessage](t, envl)["node"], &node)
	id := node["id"].(string)

	// Response carries no secret.
	raw, _ := json.Marshal(node)
	if strings.Contains(string(raw), "sk-node-secret") {
		t.Fatalf("create response leaks api_key: %s", raw)
	}

	// A bound connection exists with the encrypted key.
	conns, err := env.store.ListConnections()
	if err != nil {
		t.Fatalf("ListConnections: %v", err)
	}
	var found *store.Connection
	for _, c := range conns {
		if c.ProviderID == id {
			found = c
		}
	}
	if found == nil || found.Secret != "sk-node-secret" {
		t.Fatalf("provisioned connection = %+v", found)
	}

	// Create WITHOUT a key provisions nothing.
	status, envl = call(t, env.handlers.CreateProviderNode, "POST", "/api/provider-nodes",
		`{"name":"NoKey","prefix":"nk","apiType":"openai","baseUrl":"https://nokey.example.com/v1"}`, nil, nil)
	if status != fasthttp.StatusCreated {
		t.Fatalf("create nokey status = %d", status)
	}
	var node2 map[string]any
	json.Unmarshal(dataField[map[string]json.RawMessage](t, envl)["node"], &node2)
	id2 := node2["id"].(string)
	conns, _ = env.store.ListConnections()
	for _, c := range conns {
		if c.ProviderID == id2 {
			t.Fatalf("create without key provisioned a connection: %+v", c)
		}
	}
}

// TestNodesRouteDisambiguation proves the fasthttp/router matcher resolves the
// static /api/provider-nodes/validate route distinctly from the bare collection
// /api/provider-nodes route AND from the {id} param route (plan §1.9 / §8 ESC-ROUTE).
func TestNodesRouteDisambiguation(t *testing.T) {
	env := newTestEnv(t)
	r := router.New()
	r.GET("/api/provider-nodes", env.handlers.ListProviderNodes)
	r.POST("/api/provider-nodes", env.handlers.CreateProviderNode)
	r.POST("/api/provider-nodes/validate", env.handlers.ValidateProviderNode)
	r.GET("/api/provider-nodes/{id}", env.handlers.GetProviderNode)
	r.PUT("/api/provider-nodes/{id}", env.handlers.UpdateProviderNode)
	r.DELETE("/api/provider-nodes/{id}", env.handlers.DeleteProviderNode)

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
	env.handlers.SetNodeResolver(func(string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("93.184.216.34")}, nil
	})
	env.handlers.SetNodeProber(func(req platform.NodeProbeRequest) (platform.NodeProbeResult, error) {
		return platform.NodeProbeResult{Valid: true}, nil
	})
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

	// GET /api/provider-nodes/{id} resolves to the get handler distinctly from
	// the static collection and the static /validate route.
	id := seedProvider(t, env, "Compat", "openai-compatible", "https://compat.example.com/v1")
	var getCtx fasthttp.RequestCtx
	getCtx.Request.Header.SetMethod("GET")
	getCtx.Request.SetRequestURI("/api/provider-nodes/" + id)
	r.Handler(&getCtx)
	if getCtx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("GET /api/provider-nodes/{id} status = %d body=%s", getCtx.Response.StatusCode(), getCtx.Response.Body())
	}
	var getEnv struct {
		Data struct {
			Node map[string]any `json:"node"`
		} `json:"data"`
	}
	if err := json.Unmarshal(getCtx.Response.Body(), &getEnv); err != nil {
		t.Fatalf("get body is not a {node} envelope: %v\n%s", err, getCtx.Response.Body())
	}
	if getEnv.Data.Node["id"] != id {
		t.Fatalf("get {id} resolved wrong node: %v", getEnv.Data.Node)
	}
}
