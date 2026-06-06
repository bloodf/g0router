package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type tunnelResponse struct {
	Type      string `json:"type"`
	IsEnabled bool   `json:"is_enabled"`
	URL       string `json:"url,omitempty"`
	Status    string `json:"status"`
	LastError string `json:"last_error,omitempty"`
}

type proxyTestResponse struct {
	OK        bool   `json:"ok"`
	LatencyMS int    `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

type tunnelStore interface {
	ListTunnelConfigs() ([]store.TunnelConfig, error)
	UpsertTunnelConfig(cfg store.TunnelConfig) error
	UpdateTunnelStatus(tunnelType, status, lastError string) error
}

// TunnelManager orchestrates tunnel binaries.
type TunnelManager interface {
	StartCloudflare(port string) (string, error)
	StopCloudflare() error
	StartTailscale(port string) (string, error)
	StopTailscale() error
}

type createTunnelRequest struct {
	Port string `json:"port,omitempty"`
}

// TunnelList returns all tunnel configs (config field is never exposed).
func TunnelList(ctx *fasthttp.RequestCtx, s tunnelStore) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	configs, err := s.ListTunnelConfigs()
	if err != nil {
		log.Printf("list tunnel configs: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to list tunnels")
		return
	}
	views := make([]tunnelResponse, 0, len(configs))
	for _, cfg := range configs {
		views = append(views, tunnelResponse{
			Type:      cfg.Type,
			IsEnabled: cfg.IsEnabled,
			URL:       cfg.URL,
			Status:    cfg.Status,
			LastError: cfg.LastError,
		})
	}
	writeJSON(ctx, fasthttp.StatusOK, listResponse[tunnelResponse]{Data: views})
}

// TunnelCloudflareCreate starts a Cloudflare tunnel.
func TunnelCloudflareCreate(ctx *fasthttp.RequestCtx, s tunnelStore, mgr TunnelManager, audit auditWriter, defaultPort string) {
	if isStoreNil(s) || isStoreNil(audit) || isStoreNil(mgr) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	var req createTunnelRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}
	port := req.Port
	if port == "" {
		port = defaultPort
	}
	url, err := mgr.StartCloudflare(port)
	if err != nil {
		log.Printf("start cloudflare tunnel: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to start cloudflare tunnel")
		return
	}
	if err := audit.AppendAudit(store.AuditEntry{
		Action: "tunnel.cloudflare.create",
		Target: port,
	}); err != nil {
		log.Printf("append audit: %v", err)
	}
	writeJSON(ctx, fasthttp.StatusCreated, map[string]any{
		"url":    url,
		"status": "active",
	})
}

// TunnelCloudflareDelete stops the Cloudflare tunnel.
func TunnelCloudflareDelete(ctx *fasthttp.RequestCtx, s tunnelStore, mgr TunnelManager, audit auditWriter) {
	if isStoreNil(s) || isStoreNil(audit) || isStoreNil(mgr) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	if err := mgr.StopCloudflare(); err != nil {
		log.Printf("stop cloudflare tunnel: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to stop cloudflare tunnel")
		return
	}
	if err := audit.AppendAudit(store.AuditEntry{
		Action: "tunnel.cloudflare.delete",
	}); err != nil {
		log.Printf("append audit: %v", err)
	}
	ctx.SetStatusCode(fasthttp.StatusNoContent)
}

// TunnelTailscaleCreate starts a Tailscale funnel.
func TunnelTailscaleCreate(ctx *fasthttp.RequestCtx, s tunnelStore, mgr TunnelManager, audit auditWriter, defaultPort string) {
	if isStoreNil(s) || isStoreNil(audit) || isStoreNil(mgr) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	var req createTunnelRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}
	port := req.Port
	if port == "" {
		port = defaultPort
	}
	url, err := mgr.StartTailscale(port)
	if err != nil {
		if strings.Contains(err.Error(), "tailscale not found on PATH") {
			writeError(ctx, fasthttp.StatusConflict, "tailscale is not installed. Install it from https://tailscale.com/download")
			return
		}
		log.Printf("start tailscale tunnel: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to start tailscale tunnel")
		return
	}
	if err := audit.AppendAudit(store.AuditEntry{
		Action: "tunnel.tailscale.create",
		Target: port,
	}); err != nil {
		log.Printf("append audit: %v", err)
	}
	writeJSON(ctx, fasthttp.StatusCreated, map[string]any{
		"url":    url,
		"status": "active",
	})
}

// TunnelTailscaleDelete stops the Tailscale funnel.
func TunnelTailscaleDelete(ctx *fasthttp.RequestCtx, s tunnelStore, mgr TunnelManager, audit auditWriter) {
	if isStoreNil(s) || isStoreNil(audit) || isStoreNil(mgr) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	if err := mgr.StopTailscale(); err != nil {
		log.Printf("stop tailscale tunnel: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to stop tailscale tunnel")
		return
	}
	if err := audit.AppendAudit(store.AuditEntry{
		Action: "tunnel.tailscale.delete",
	}); err != nil {
		log.Printf("append audit: %v", err)
	}
	ctx.SetStatusCode(fasthttp.StatusNoContent)
}

// TunnelHealth checks reachability of each enabled tunnel.
func TunnelHealth(ctx *fasthttp.RequestCtx, s tunnelStore) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	configs, err := s.ListTunnelConfigs()
	if err != nil {
		log.Printf("list tunnel configs: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to list tunnels")
		return
	}

	type healthItem struct {
		Type      string `json:"type"`
		URL       string `json:"url"`
		Reachable bool   `json:"reachable"`
		LatencyMS int    `json:"latency_ms"`
	}

	results := make([]healthItem, 0, len(configs))
	client := &http.Client{Timeout: 5 * time.Second}
	for _, cfg := range configs {
		if !cfg.IsEnabled || cfg.URL == "" {
			continue
		}
		healthURL := strings.TrimRight(cfg.URL, "/") + "/healthz"
		start := time.Now()
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, healthURL, nil)
		resp, err := client.Do(req)
		latency := int(time.Since(start).Milliseconds())
		reachable := err == nil && resp != nil && resp.StatusCode == http.StatusOK
		if resp != nil {
			_ = resp.Body.Close()
		}
		results = append(results, healthItem{
			Type:      cfg.Type,
			URL:       cfg.URL,
			Reachable: reachable,
			LatencyMS: latency,
		})
	}
	writeJSON(ctx, fasthttp.StatusOK, listResponse[healthItem]{Data: results})
}

// ProxyTest tests connectivity through a proxy URL.
func ProxyTest(ctx *fasthttp.RequestCtx) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}
	if req.URL == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "url required")
		return
	}

	u, err := url.Parse(req.URL)
	if err != nil {
		writeJSON(ctx, fasthttp.StatusOK, proxyTestResponse{
			OK:    false,
			Error: "invalid proxy URL: " + err.Error(),
		})
		return
	}

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		switch u.Scheme {
		case "http":
			port = "80"
		case "https":
			port = "443"
		case "socks5":
			port = "1080"
		default:
			port = "80"
		}
	}

	start := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 5*time.Second)
	latency := int(time.Since(start).Milliseconds())
	if err != nil {
		writeJSON(ctx, fasthttp.StatusOK, proxyTestResponse{
			OK:        false,
			LatencyMS: latency,
			Error:     err.Error(),
		})
		return
	}
	_ = conn.Close()

	writeJSON(ctx, fasthttp.StatusOK, proxyTestResponse{
		OK:        true,
		LatencyMS: latency,
	})
}
