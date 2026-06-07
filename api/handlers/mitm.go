package handlers

import (
	"encoding/json"
	"log"

	"github.com/bloodf/g0router/internal/mitm"
	"github.com/valyala/fasthttp"
)

type mitmProxy interface {
	IsRunning() bool
	Addr() string
	Start() error
	Stop() error
	CACertPEM() []byte
	ToolEnabled(name string) bool
	SetToolEnabled(name string, enabled bool)
}

type mitmStatusResponse struct {
	Running bool                     `json:"running"`
	Addr    string                   `json:"addr"`
	Tools   []map[string]interface{} `json:"tools"`
}

type mitmToggleRequest struct {
	Enabled bool `json:"enabled"`
}

type mitmToolRequest struct {
	Enabled bool `json:"enabled"`
}

func requireMITMFlag(ctx *fasthttp.RequestCtx, store featureFlagStore) bool {
	if isStoreNil(store) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return false
	}
	flag, err := store.GetFeatureFlagByKey("mitm_proxy")
	if err != nil {
		log.Printf("get feature flag: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to check feature flag")
		return false
	}
	if flag == nil || !flag.Enabled {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return false
	}
	return true
}

// MITMStatus returns the current proxy state and per-tool setup instructions.
func MITMStatus(ctx *fasthttp.RequestCtx, proxy mitmProxy, store featureFlagStore) {
	if !requireMITMFlag(ctx, store) {
		return
	}
	if proxy == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "proxy unavailable")
		return
	}

	addr := proxy.Addr()
	if addr == "" {
		addr = "not started"
	}

	tools := mitm.ToolInstructions(addr)
	for i := range tools {
		name := tools[i]["name"].(string)
		tools[i]["enabled"] = proxy.ToolEnabled(name)
	}

	writeJSON(ctx, fasthttp.StatusOK, mitmStatusResponse{
		Running: proxy.IsRunning(),
		Addr:    addr,
		Tools:   tools,
	})
}

// MITMToggle starts or stops the MITM proxy.
func MITMToggle(ctx *fasthttp.RequestCtx, proxy mitmProxy, store featureFlagStore) {
	if !requireMITMFlag(ctx, store) {
		return
	}
	if proxy == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "proxy unavailable")
		return
	}

	var req mitmToggleRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}

	var err error
	if req.Enabled {
		err = proxy.Start()
	} else {
		err = proxy.Stop()
	}
	if err != nil {
		log.Printf("mitm toggle: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to toggle proxy")
		return
	}

	writeJSON(ctx, fasthttp.StatusOK, map[string]interface{}{
		"running": proxy.IsRunning(),
		"addr":    proxy.Addr(),
	})
}

// MITMCACert returns the CA certificate in PEM format.
func MITMCACert(ctx *fasthttp.RequestCtx, proxy mitmProxy, store featureFlagStore) {
	if !requireMITMFlag(ctx, store) {
		return
	}
	if proxy == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "proxy unavailable")
		return
	}

	pem := proxy.CACertPEM()
	ctx.SetContentType("application/x-pem-file")
	ctx.SetBody(pem)
}

// MITMTools toggles interception for a specific tool.
func MITMTools(ctx *fasthttp.RequestCtx, proxy mitmProxy, store featureFlagStore, toolID string) {
	if !requireMITMFlag(ctx, store) {
		return
	}
	if proxy == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "proxy unavailable")
		return
	}

	if toolID == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "tool id required")
		return
	}

	var req mitmToolRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}

	proxy.SetToolEnabled(toolID, req.Enabled)

	writeJSON(ctx, fasthttp.StatusOK, map[string]interface{}{
		"tool":    toolID,
		"enabled": proxy.ToolEnabled(toolID),
	})
}
