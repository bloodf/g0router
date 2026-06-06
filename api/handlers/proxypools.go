package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

type proxyPoolResponse struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Protocol        string `json:"protocol"`
	Host            string `json:"host"`
	Port            int    `json:"port"`
	Username        string `json:"username,omitempty"`
	IsActive        bool   `json:"is_active"`
	LastCheckAt     string `json:"last_check_at,omitempty"`
	LastCheckStatus string `json:"last_check_status,omitempty"`
	CreatedAt       string `json:"created_at"`
}

func newProxyPoolResponse(pool store.ProxyPool) proxyPoolResponse {
	return proxyPoolResponse{
		ID:              pool.ID,
		Name:            pool.Name,
		Protocol:        pool.Protocol,
		Host:            pool.Host,
		Port:            pool.Port,
		Username:        pool.Username,
		IsActive:        pool.IsActive,
		LastCheckAt:     pool.LastCheckAt,
		LastCheckStatus: pool.LastCheckStatus,
		CreatedAt:       pool.CreatedAt,
	}
}

type createProxyPoolRequest struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type updateProxyPoolRequest struct {
	Name     *string `json:"name"`
	Protocol *string `json:"protocol"`
	Host     *string `json:"host"`
	Port     *int    `json:"port"`
	Username *string `json:"username"`
	Password *string `json:"password"`
}

type batchImportRequest struct {
	Lines []string `json:"lines"`
}

type batchImportError struct {
	Line  string `json:"line"`
	Error string `json:"error"`
}

type batchImportResponse struct {
	Created []proxyPoolResponse `json:"created"`
	Errors  []batchImportError  `json:"errors"`
}

type proxyPoolStore interface {
	ListProxyPools() ([]store.ProxyPool, error)
	GetProxyPool(id string) (*store.ProxyPool, error)
	CreateProxyPool(pool store.ProxyPool) (*store.ProxyPool, error)
	UpdateProxyPool(id string, pool store.ProxyPool) error
	DeleteProxyPool(id string) error
	TestProxyPool(id string) (bool, int, error)
}

// ProxyPoolList returns all proxy pools.
func ProxyPoolList(ctx *fasthttp.RequestCtx, s proxyPoolStore) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	pools, err := s.ListProxyPools()
	if err != nil {
		log.Printf("list proxy pools: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to list proxy pools")
		return
	}
	views := make([]proxyPoolResponse, 0, len(pools))
	for _, pool := range pools {
		views = append(views, newProxyPoolResponse(pool))
	}
	writeJSON(ctx, fasthttp.StatusOK, listResponse[proxyPoolResponse]{Data: views})
}

// ProxyPoolGet returns a single proxy pool by id.
func ProxyPoolGet(ctx *fasthttp.RequestCtx, s proxyPoolStore, id string) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	if id == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "proxy pool id required")
		return
	}
	pool, err := s.GetProxyPool(id)
	if err != nil {
		writeStoreError(ctx, "get proxy pool", err)
		return
	}
	writeJSON(ctx, fasthttp.StatusOK, map[string]any{"data": newProxyPoolResponse(*pool)})
}

// ProxyPoolCreate creates a new proxy pool.
func ProxyPoolCreate(ctx *fasthttp.RequestCtx, s proxyPoolStore, audit auditWriter) {
	if isStoreNil(s) || isStoreNil(audit) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	var req createProxyPoolRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}
	if req.Name == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "name required")
		return
	}
	if req.Protocol != "http" && req.Protocol != "https" && req.Protocol != "socks5" {
		writeError(ctx, fasthttp.StatusBadRequest, "protocol must be http, https, or socks5")
		return
	}
	if req.Host == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "host required")
		return
	}
	if req.Port < 1 || req.Port > 65535 {
		writeError(ctx, fasthttp.StatusBadRequest, "port must be between 1 and 65535")
		return
	}
	pool, err := s.CreateProxyPool(store.ProxyPool{
		Name:     req.Name,
		Protocol: req.Protocol,
		Host:     req.Host,
		Port:     req.Port,
		Username: req.Username,
		Password: req.Password,
		IsActive: true,
	})
	if err != nil {
		log.Printf("create proxy pool: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to create proxy pool")
		return
	}
	if err := audit.AppendAudit(store.AuditEntry{
		Action: "proxy_pool.create",
		Target: req.Name,
	}); err != nil {
		log.Printf("append audit: %v", err)
	}
	writeJSON(ctx, fasthttp.StatusCreated, map[string]any{"data": newProxyPoolResponse(*pool)})
}

// ProxyPoolUpdate updates an existing proxy pool.
func ProxyPoolUpdate(ctx *fasthttp.RequestCtx, s proxyPoolStore, audit auditWriter, id string) {
	if isStoreNil(s) || isStoreNil(audit) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	if id == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "proxy pool id required")
		return
	}
	var req updateProxyPoolRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}
	existing, err := s.GetProxyPool(id)
	if err != nil {
		writeStoreError(ctx, "get proxy pool", err)
		return
	}
	updated := *existing
	if req.Name != nil {
		updated.Name = *req.Name
	}
	if req.Protocol != nil {
		if *req.Protocol != "http" && *req.Protocol != "https" && *req.Protocol != "socks5" {
			writeError(ctx, fasthttp.StatusBadRequest, "protocol must be http, https, or socks5")
			return
		}
		updated.Protocol = *req.Protocol
	}
	if req.Host != nil {
		updated.Host = *req.Host
	}
	if req.Port != nil {
		if *req.Port < 1 || *req.Port > 65535 {
			writeError(ctx, fasthttp.StatusBadRequest, "port must be between 1 and 65535")
			return
		}
		updated.Port = *req.Port
	}
	if req.Username != nil {
		updated.Username = *req.Username
	}
	if req.Password != nil {
		updated.Password = *req.Password
	}
	if err := s.UpdateProxyPool(id, updated); err != nil {
		writeStoreError(ctx, "update proxy pool", err)
		return
	}
	got, err := s.GetProxyPool(id)
	if err != nil {
		writeStoreError(ctx, "get proxy pool", err)
		return
	}
	if err := audit.AppendAudit(store.AuditEntry{
		Action: "proxy_pool.update",
		Target: id,
	}); err != nil {
		log.Printf("append audit: %v", err)
	}
	writeJSON(ctx, fasthttp.StatusOK, map[string]any{"data": newProxyPoolResponse(*got)})
}

// ProxyPoolDelete deletes a proxy pool.
func ProxyPoolDelete(ctx *fasthttp.RequestCtx, s proxyPoolStore, audit auditWriter, id string) {
	if isStoreNil(s) || isStoreNil(audit) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	if id == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "proxy pool id required")
		return
	}
	if err := s.DeleteProxyPool(id); err != nil {
		writeStoreError(ctx, "delete proxy pool", err)
		return
	}
	if err := audit.AppendAudit(store.AuditEntry{
		Action: "proxy_pool.delete",
		Target: id,
	}); err != nil {
		log.Printf("append audit: %v", err)
	}
	ctx.SetStatusCode(fasthttp.StatusNoContent)
}

// ProxyPoolTest tests connectivity for a proxy pool.
func ProxyPoolTest(ctx *fasthttp.RequestCtx, s proxyPoolStore, id string) {
	if isStoreNil(s) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	if id == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "proxy pool id required")
		return
	}
	if _, err := s.GetProxyPool(id); err != nil {
		writeStoreError(ctx, "get proxy pool", err)
		return
	}
	ok, latencyMs, err := s.TestProxyPool(id)
	if err != nil {
		log.Printf("test proxy pool: %v", err)
		writeError(ctx, fasthttp.StatusInternalServerError, "failed to test proxy pool")
		return
	}
	errStr := ""
	if !ok {
		errStr = "proxy test failed"
	}
	writeJSON(ctx, fasthttp.StatusOK, map[string]any{
		"ok":         ok,
		"latency_ms": latencyMs,
		"error":      errStr,
	})
}

// ProxyPoolBatchImport imports multiple proxy pools from lines.
func ProxyPoolBatchImport(ctx *fasthttp.RequestCtx, s proxyPoolStore, audit auditWriter) {
	if isStoreNil(s) || isStoreNil(audit) {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "store unavailable")
		return
	}
	var req batchImportRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON")
		return
	}
	var created []proxyPoolResponse
	var errs []batchImportError
	for _, line := range req.Lines {
		pool, err := parseProxyLine(line)
		if err != nil {
			errs = append(errs, batchImportError{Line: line, Error: err.Error()})
			continue
		}
		newPool, err := s.CreateProxyPool(pool)
		if err != nil {
			errs = append(errs, batchImportError{Line: line, Error: err.Error()})
			continue
		}
		created = append(created, newProxyPoolResponse(*newPool))
	}
	if len(created) > 0 {
		if err := audit.AppendAudit(store.AuditEntry{
			Action: "proxy_pool.batch_create",
			Target: fmt.Sprintf("%d", len(created)),
		}); err != nil {
			log.Printf("append audit: %v", err)
		}
	}
	writeJSON(ctx, fasthttp.StatusOK, batchImportResponse{
		Created: created,
		Errors:  errs,
	})
}

func parseProxyLine(line string) (store.ProxyPool, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return store.ProxyPool{}, fmt.Errorf("empty line")
	}
	var protocol, host, username, password string
	var port int
	if strings.Contains(line, "://") {
		u, err := url.Parse(line)
		if err != nil {
			return store.ProxyPool{}, fmt.Errorf("invalid URL: %w", err)
		}
		protocol = u.Scheme
		host = u.Hostname()
		portStr := u.Port()
		if portStr == "" {
			return store.ProxyPool{}, fmt.Errorf("port required")
		}
		p, err := strconv.Atoi(portStr)
		if err != nil || p < 1 || p > 65535 {
			return store.ProxyPool{}, fmt.Errorf("invalid port")
		}
		port = p
		if u.User != nil {
			username = u.User.Username()
			if pw, ok := u.User.Password(); ok {
				password = pw
			}
		}
	} else {
		protocol = "http"
		h, portStr, err := net.SplitHostPort(line)
		if err != nil {
			return store.ProxyPool{}, fmt.Errorf("invalid host:port")
		}
		host = h
		p, err := strconv.Atoi(portStr)
		if err != nil || p < 1 || p > 65535 {
			return store.ProxyPool{}, fmt.Errorf("invalid port")
		}
		port = p
	}
	switch protocol {
	case "http", "https", "socks5":
	default:
		return store.ProxyPool{}, fmt.Errorf("invalid protocol %q", protocol)
	}
	name := fmt.Sprintf("proxy-%s-%d", host, port)
	return store.ProxyPool{
		Name:     name,
		Protocol: protocol,
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		IsActive: true,
	}, nil
}
