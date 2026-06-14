package admin

import (
	"encoding/json"

	"github.com/bloodf/g0router/internal/platform"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// providerNodeType is the default provider-node type for a node created without an
// explicit type. Provider nodes compose the existing providers table; there is no
// separate node table (w7-platnodes, PAR-PLAT-014).
const providerNodeType = "openai-compatible"

// providerNodeDTO is the dashboard-facing shape for a provider node. It surfaces
// the routing prefix and api_type now persisted on the providers row
// (w7-platnodes). It NEVER carries an api_key.
type providerNodeDTO struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	Type    string `json:"type"`
	Enabled bool   `json:"enabled"`
	Prefix  string `json:"prefix"`
	APIType string `json:"api_type"`
}

func toProviderNodeDTO(p *store.ProviderRecord) providerNodeDTO {
	return providerNodeDTO{
		ID:      p.ID,
		Name:    p.Name,
		BaseURL: p.BaseURL,
		Type:    p.Type,
		Enabled: p.Enabled,
		Prefix:  p.Prefix,
		APIType: p.APIType,
	}
}

// providerNodeRequest accepts both the 9router camelCase client body
// ({name,prefix,apiType,baseUrl,apiKey}) and the snake_case admin convention
// ({name,prefix,api_type,base_url,api_key}); the snake_case fields win when both
// are set.
type providerNodeRequest struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Prefix    string `json:"prefix"`
	APIType   string `json:"api_type"`
	APITypeCC string `json:"apiType"`
	BaseURL   string `json:"base_url"`
	BaseURLCC string `json:"baseUrl"`
	APIKey    string `json:"api_key"`
	APIKeyCC  string `json:"apiKey"`
	ModelID   string `json:"model_id"`
	ModelIDCC string `json:"modelId"`
}

func (r providerNodeRequest) baseURL() string {
	if r.BaseURL != "" {
		return r.BaseURL
	}
	return r.BaseURLCC
}

func (r providerNodeRequest) apiType() string {
	if r.APIType != "" {
		return r.APIType
	}
	return r.APITypeCC
}

func (r providerNodeRequest) apiKey() string {
	if r.APIKey != "" {
		return r.APIKey
	}
	return r.APIKeyCC
}

func (r providerNodeRequest) modelID() string {
	if r.ModelID != "" {
		return r.ModelID
	}
	return r.ModelIDCC
}

// nodeType returns the requested node type, defaulting to openai-compatible.
func (r providerNodeRequest) nodeType() string {
	if r.Type != "" {
		return r.Type
	}
	return providerNodeType
}

// ListProviderNodes handles GET /api/provider-nodes (PAR-PLAT-010). It lists the
// providers filtered to the node-type set, mapped to the node DTO.
func (h *Handlers) ListProviderNodes(ctx *fasthttp.RequestCtx) {
	nodes, err := h.providerNodes.List()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list provider nodes")
		return
	}
	out := make([]providerNodeDTO, 0, len(nodes))
	for _, p := range nodes {
		out = append(out, toProviderNodeDTO(p))
	}
	writeData(ctx, fasthttp.StatusOK, map[string]any{"nodes": out})
}

// CreateProviderNode handles POST /api/provider-nodes (PAR-PLAT-010). It persists
// prefix/api_type, sanitizes the base URL, optionally provisions a bound api_key
// connection, and records an audit entry. The api_key is never echoed.
func (h *Handlers) CreateProviderNode(ctx *fasthttp.RequestCtx) {
	var req providerNodeRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	baseURL := req.baseURL()
	if req.Name == "" || baseURL == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "name and base_url are required")
		return
	}

	node, err := h.providerNodes.Create(platform.NodeCreate{
		Name:    req.Name,
		Type:    req.nodeType(),
		Prefix:  req.Prefix,
		APIType: req.apiType(),
		BaseURL: baseURL,
		APIKey:  req.apiKey(),
	})
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "create provider node")
		return
	}
	h.recordAudit(ctx, "provider_node.create", node.ID, node.Name)
	writeData(ctx, fasthttp.StatusCreated, map[string]any{"node": toProviderNodeDTO(node)})
}

// GetProviderNode handles GET /api/provider-nodes/{id} (PAR-PLAT-010).
func (h *Handlers) GetProviderNode(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid id")
		return
	}
	node, err := h.providerNodes.Get(id)
	if err != nil {
		writeError(ctx, fasthttp.StatusNotFound, "provider node not found")
		return
	}
	writeData(ctx, fasthttp.StatusOK, map[string]any{"node": toProviderNodeDTO(node)})
}

// UpdateProviderNode handles PUT /api/provider-nodes/{id} (PAR-PLAT-010/012). It
// re-sanitizes the base URL and cascades prefix/baseUrl/apiType onto the providers
// row, from which bound connections resolve transitively.
func (h *Handlers) UpdateProviderNode(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid id")
		return
	}
	var req providerNodeRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	baseURL := req.baseURL()
	if req.Name == "" || baseURL == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "name and base_url are required")
		return
	}
	node, err := h.providerNodes.Update(platform.NodeUpdate{
		ID:      id,
		Name:    req.Name,
		Type:    req.Type,
		Prefix:  req.Prefix,
		APIType: req.apiType(),
		BaseURL: baseURL,
	})
	if err != nil {
		writeError(ctx, fasthttp.StatusNotFound, "provider node not found")
		return
	}
	h.recordAudit(ctx, "provider_node.update", node.ID, node.Name)
	writeData(ctx, fasthttp.StatusOK, map[string]any{"node": toProviderNodeDTO(node)})
}

// DeleteProviderNode handles DELETE /api/provider-nodes/{id} (PAR-PLAT-010).
func (h *Handlers) DeleteProviderNode(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid id")
		return
	}
	if err := h.providerNodes.Delete(id); err != nil {
		writeError(ctx, fasthttp.StatusNotFound, "provider node not found")
		return
	}
	h.recordAudit(ctx, "provider_node.delete", id, "")
	writeData(ctx, fasthttp.StatusOK, map[string]any{"message": "Provider node deleted successfully"})
}

// ValidateProviderNode handles POST /api/provider-nodes/validate (PAR-PLAT-013).
// It runs the real reachability probe through an injectable seam (hermetic in
// tests), SSRF-guarded before dialing. The supplied api_key is used transiently
// and NEVER persisted or echoed.
func (h *Handlers) ValidateProviderNode(ctx *fasthttp.RequestCtx) {
	var req providerNodeRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	res, err := h.providerNodes.Validate(platform.NodeProbeRequest{
		APIType: req.nodeType(),
		BaseURL: req.baseURL(),
		APIKey:  req.apiKey(),
		ModelID: req.modelID(),
	})
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "validate provider node")
		return
	}
	out := map[string]any{"valid": res.Valid}
	if res.Error != "" {
		out["error"] = res.Error
	}
	writeData(ctx, fasthttp.StatusOK, out)
}
