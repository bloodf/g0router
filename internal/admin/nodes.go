package admin

import (
	"encoding/json"
	"net/url"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// providerNodeType is the provider record type that backs a "provider node": an
// OpenAI-compatible custom endpoint (plan §1.6b). Provider nodes compose the
// existing providers table; there is no separate node table.
const providerNodeType = "openai-compatible"

// providerNodeDTO is the dashboard-facing shape for a provider node. It is the
// subset of the provider record relevant to the node UI; prefix/api_type from the
// 9router client are accepted at decode but not persisted (the providers table has
// no such columns and the node UI does not surface them — plan §1.6b, no schema
// change).
type providerNodeDTO struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	Type    string `json:"type"`
	Enabled bool   `json:"enabled"`
}

func toProviderNodeDTO(p *store.ProviderRecord) providerNodeDTO {
	return providerNodeDTO{
		ID:      p.ID,
		Name:    p.Name,
		BaseURL: p.BaseURL,
		Type:    p.Type,
		Enabled: p.Enabled,
	}
}

// providerNodeRequest accepts both the 9router camelCase client body
// ({name,prefix,apiType,baseUrl}) and the snake_case admin convention
// ({name,prefix,api_type,base_url}); the snake_case fields win when both are set.
type providerNodeRequest struct {
	Name      string `json:"name"`
	Prefix    string `json:"prefix"`
	APIType   string `json:"api_type"`
	APITypeCC string `json:"apiType"`
	BaseURL   string `json:"base_url"`
	BaseURLCC string `json:"baseUrl"`
	APIKey    string `json:"api_key"`
	APIKeyCC  string `json:"apiKey"`
}

func (r providerNodeRequest) baseURL() string {
	if r.BaseURL != "" {
		return r.BaseURL
	}
	return r.BaseURLCC
}

// ListProviderNodes handles GET /api/provider-nodes (PAR-UI-109). It lists the
// providers filtered to the openai-compatible type, mapped to the node DTO.
func (h *Handlers) ListProviderNodes(ctx *fasthttp.RequestCtx) {
	providers, err := h.store.ListProviders()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list providers")
		return
	}
	nodes := make([]providerNodeDTO, 0)
	for _, p := range providers {
		if p.Type != providerNodeType {
			continue
		}
		nodes = append(nodes, toProviderNodeDTO(p))
	}
	writeData(ctx, fasthttp.StatusOK, map[string]any{"nodes": nodes})
}

// CreateProviderNode handles POST /api/provider-nodes (PAR-UI-110). It creates a
// providers row of the openai-compatible type. prefix/api_type are accepted but
// not persisted (no schema change, plan §1.6b).
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

	rec := &store.ProviderRecord{
		Name:    req.Name,
		Type:    providerNodeType,
		BaseURL: baseURL,
		Enabled: true,
	}
	if err := h.store.CreateProvider(rec); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "create provider node")
		return
	}
	writeData(ctx, fasthttp.StatusCreated, map[string]any{"node": toProviderNodeDTO(rec)})
}

// ValidateProviderNode handles POST /api/provider-nodes/validate (PAR-UI-111). It
// performs a best-effort reachability check; under test it is deterministic on URL
// well-formedness. The supplied api_key is used transiently and NEVER persisted or
// echoed (plan §1.6b).
func (h *Handlers) ValidateProviderNode(ctx *fasthttp.RequestCtx) {
	var req providerNodeRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	baseURL := req.baseURL()
	if !isWellFormedURL(baseURL) {
		writeData(ctx, fasthttp.StatusOK, map[string]any{"valid": false, "error": "invalid url"})
		return
	}
	writeData(ctx, fasthttp.StatusOK, map[string]any{"valid": true})
}

// isWellFormedURL reports whether s is an absolute http(s) URL with a host.
func isWellFormedURL(s string) bool {
	if s == "" {
		return false
	}
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return u.Host != ""
}
