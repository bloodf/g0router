package admin

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// proxyPoolDTO is the canonical snake_case proxy-pool shape. It NEVER carries
// the cleartext password; password_set reports whether a password is configured.
type proxyPoolDTO struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Protocol        string `json:"protocol"`
	Host            string `json:"host"`
	Port            int    `json:"port"`
	Username        string `json:"username"`
	PasswordSet     bool   `json:"password_set"`
	IsActive        bool   `json:"is_active"`
	LastCheckStatus string `json:"last_check_status"`
	LastCheckAt     string `json:"last_check_at"`
}

func toProxyPoolDTO(p *store.ProxyPool) proxyPoolDTO {
	return proxyPoolDTO{
		ID:              p.ID,
		Name:            p.Name,
		Protocol:        p.Protocol,
		Host:            p.Host,
		Port:            p.Port,
		Username:        p.Username,
		PasswordSet:     p.Password != "",
		IsActive:        p.IsActive,
		LastCheckStatus: p.LastCheckStatus,
		LastCheckAt:     p.LastCheckAt,
	}
}

type proxyPoolRequest struct {
	Name     string  `json:"name"`
	Protocol string  `json:"protocol"`
	Host     string  `json:"host"`
	Port     int     `json:"port"`
	Username string  `json:"username"`
	Password *string `json:"password"`
	IsActive *bool   `json:"is_active"`
}

func validateProxyPoolRequest(req *proxyPoolRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if req.Host == "" {
		return fmt.Errorf("host is required")
	}
	if req.Port < 0 || req.Port > 65535 {
		return fmt.Errorf("port must be between 0 and 65535")
	}
	return nil
}

// ListProxyPools handles GET /api/proxy-pools[?isActive=true]. The response data
// is a bare array of proxyPoolDTO under {data}, mirroring the UI mock.
func (h *Handlers) ListProxyPools(ctx *fasthttp.RequestCtx) {
	var filterActive *bool
	if raw := string(ctx.QueryArgs().Peek("isActive")); raw != "" {
		v := raw == "true" || raw == "1"
		filterActive = &v
	}
	pools, err := h.proxyPools.List(filterActive)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list proxy pools")
		return
	}
	out := make([]proxyPoolDTO, 0, len(pools))
	for _, p := range pools {
		out = append(out, toProxyPoolDTO(p))
	}
	writeData(ctx, fasthttp.StatusOK, out)
}

// CreateProxyPool handles POST /api/proxy-pools.
func (h *Handlers) CreateProxyPool(ctx *fasthttp.RequestCtx) {
	var req proxyPoolRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := validateProxyPoolRequest(&req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, err.Error())
		return
	}
	created, err := h.proxyPools.Create(proxyPoolFromRequest(&req, "", true))
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "create proxy pool")
		return
	}
	h.recordAudit(ctx, "proxy_pool.create", created.Name, "Created proxy pool "+created.Name)
	writeData(ctx, fasthttp.StatusCreated, toProxyPoolDTO(created))
}

// BatchProxyPools handles POST /api/proxy-pools/batch — bulk import via a thin
// loop over Create. Returns {data:{created:N}}.
func (h *Handlers) BatchProxyPools(ctx *fasthttp.RequestCtx) {
	var body struct {
		Items []proxyPoolRequest `json:"items"`
	}
	if err := json.Unmarshal(ctx.PostBody(), &body); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	created := 0
	for i := range body.Items {
		req := body.Items[i]
		if err := validateProxyPoolRequest(&req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, err.Error())
			return
		}
		if _, err := h.proxyPools.Create(proxyPoolFromRequest(&req, "", true)); err != nil {
			writeError(ctx, fasthttp.StatusInternalServerError, "create proxy pool")
			return
		}
		created++
	}
	h.recordAudit(ctx, "proxy_pool.batch", "", fmt.Sprintf("Imported %d proxy pools", created))
	writeData(ctx, fasthttp.StatusOK, map[string]any{"created": created})
}

// GetProxyPool handles GET /api/proxy-pools/{id}.
func (h *Handlers) GetProxyPool(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	p, err := h.proxyPools.Get(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "proxy pool not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load proxy pool")
		return
	}
	writeData(ctx, fasthttp.StatusOK, toProxyPoolDTO(p))
}

// UpdateProxyPool handles PUT /api/proxy-pools/{id}. The password is left
// unchanged when the body omits it (mirrors the connection update pattern).
func (h *Handlers) UpdateProxyPool(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	var req proxyPoolRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := validateProxyPoolRequest(&req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, err.Error())
		return
	}

	existing, err := h.proxyPools.Get(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "proxy pool not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load proxy pool")
		return
	}

	pool := proxyPoolFromRequest(&req, id, existing.IsActive)
	// Preserve existing fields the update path must not clobber.
	pool.LastCheckStatus = existing.LastCheckStatus
	pool.LastCheckAt = existing.LastCheckAt
	if req.Password == nil {
		pool.Password = existing.Password // password unchanged when omitted
	}
	if err := h.proxyPools.Update(pool); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "update proxy pool")
		return
	}
	updated, err := h.proxyPools.Get(id)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load proxy pool")
		return
	}
	h.recordAudit(ctx, "proxy_pool.update", updated.Name, "Updated proxy pool "+updated.Name)
	writeData(ctx, fasthttp.StatusOK, toProxyPoolDTO(updated))
}

// DeleteProxyPool handles DELETE /api/proxy-pools/{id}. It returns 409 when a
// connection still references the pool (bound-connection guard).
func (h *Handlers) DeleteProxyPool(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	existing, err := h.proxyPools.Get(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "proxy pool not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load proxy pool")
		return
	}

	bound, err := h.proxyPools.CountBoundConnections(id)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "check bound connections")
		return
	}
	if bound > 0 {
		writeError(ctx, fasthttp.StatusConflict, fmt.Sprintf("Proxy pool is in use by %d connection(s)", bound))
		return
	}

	if err := h.proxyPools.Delete(id); errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "proxy pool not found")
		return
	} else if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "delete proxy pool")
		return
	}
	h.recordAudit(ctx, "proxy_pool.delete", existing.Name, "Deleted proxy pool "+existing.Name)
	writeData(ctx, fasthttp.StatusOK, map[string]any{"message": "Proxy pool deleted successfully"})
}

// TestProxyPool handles POST /api/proxy-pools/{id}/test. It runs a connectivity
// probe through the pool's proxy (SSRF-guarded), persists the result, and returns
// {ok, latency_ms, status}. The body never echoes secrets.
func (h *Handlers) TestProxyPool(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	result, err := h.proxyPools.TestConnectivity(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "proxy pool not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "test proxy pool")
		return
	}
	h.recordAudit(ctx, "proxy_pool.test", id, "Tested proxy pool "+id)
	writeData(ctx, fasthttp.StatusOK, map[string]any{
		"ok":         result.OK,
		"latency_ms": result.LatencyMs,
		"status":     result.Status,
	})
}

// proxyPoolFromRequest maps a request to a store.ProxyPool. id is empty on
// create. defaultActive is applied when the request omits is_active.
func proxyPoolFromRequest(req *proxyPoolRequest, id string, defaultActive bool) *store.ProxyPool {
	protocol := req.Protocol
	if protocol == "" {
		protocol = "http"
	}
	isActive := defaultActive
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	password := ""
	if req.Password != nil {
		password = *req.Password
	}
	return &store.ProxyPool{
		ID:       id,
		Name:     req.Name,
		Protocol: protocol,
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		Password: password,
		IsActive: isActive,
	}
}
