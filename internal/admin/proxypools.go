package admin

import "github.com/valyala/fasthttp"

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

// ListProxyPools handles GET /api/proxy-pools. Stub: impl lands in T-proxypools STEP(b).
func (h *Handlers) ListProxyPools(ctx *fasthttp.RequestCtx) {
	writeData(ctx, fasthttp.StatusOK, []proxyPoolDTO{})
}

// CreateProxyPool handles POST /api/proxy-pools. Stub.
func (h *Handlers) CreateProxyPool(ctx *fasthttp.RequestCtx) {
	writeData(ctx, fasthttp.StatusCreated, proxyPoolDTO{})
}

// BatchProxyPools handles POST /api/proxy-pools/batch. Stub.
func (h *Handlers) BatchProxyPools(ctx *fasthttp.RequestCtx) {
	writeData(ctx, fasthttp.StatusOK, map[string]any{"created": 0})
}

// GetProxyPool handles GET /api/proxy-pools/{id}. Stub.
func (h *Handlers) GetProxyPool(ctx *fasthttp.RequestCtx) {
	writeData(ctx, fasthttp.StatusOK, proxyPoolDTO{})
}

// UpdateProxyPool handles PUT /api/proxy-pools/{id}. Stub.
func (h *Handlers) UpdateProxyPool(ctx *fasthttp.RequestCtx) {
	writeData(ctx, fasthttp.StatusOK, proxyPoolDTO{})
}

// DeleteProxyPool handles DELETE /api/proxy-pools/{id}. Stub.
func (h *Handlers) DeleteProxyPool(ctx *fasthttp.RequestCtx) {
	writeData(ctx, fasthttp.StatusOK, map[string]any{"message": "Proxy pool deleted successfully"})
}

// TestProxyPool handles POST /api/proxy-pools/{id}/test. Stub.
func (h *Handlers) TestProxyPool(ctx *fasthttp.RequestCtx) {
	writeData(ctx, fasthttp.StatusOK, map[string]any{"ok": false, "latency_ms": 0, "status": ""})
}
