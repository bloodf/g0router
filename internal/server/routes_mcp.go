package server

import (
	"github.com/bloodf/g0router/internal/admin"
	"github.com/fasthttp/router"
)

// RegisterMCPRoutes adds the MCP server-mode surface (bf-mcp-1):
//   - POST /mcp  — JSON-RPC 2.0 request/response over the global tool catalog.
//   - GET  /mcp  — SSE stream (": ping" heartbeat + deferred frames).
//
// Both are RAW JSON-RPC / SSE — NOT the {data,error} admin envelope (D1). /mcp
// is a NEW static path (fasthttp's router auto-orders static-before-{param});
// it is the public VK-gated MCP-server surface and is deliberately NOT added to
// guard.go LOCAL_ONLY_PATHS (D2). It also registers the session-gated
// complete-oauth route on the existing /api/mcp surface (D7).
func RegisterMCPRoutes(r *router.Router, h *admin.Handlers) {
	r.POST("/mcp", h.MCPServerPost)
	r.GET("/mcp", h.MCPServerSSE)
	r.POST("/api/mcp/instances/{id}/auth/complete", h.RequireSession(h.CompleteInstanceAuth))

	// bf-mcp-2: additive VK↔MCP assignment CRUD (session-gated, {data,error}
	// envelope). The create/update handlers run the subset validation (D5/049)
	// and compute the drift-detection config_hash (D8/079).
	r.GET("/api/mcp/vk-configs", h.RequireSession(h.ListVKMCPConfigs))
	r.POST("/api/mcp/vk-configs", h.RequireSession(h.CreateVKMCPConfig))
	r.GET("/api/mcp/vk-configs/{id}", h.RequireSession(h.GetVKMCPConfig))
	r.PUT("/api/mcp/vk-configs/{id}", h.RequireSession(h.UpdateVKMCPConfig))
	r.DELETE("/api/mcp/vk-configs/{id}", h.RequireSession(h.DeleteVKMCPConfig))
}
