package admin

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// isoFromUnix renders a unix-second timestamp as an ISO-8601 string (mirroring
// the w6-l mock created_at), or "" for a zero timestamp.
func isoFromUnix(sec int64) string {
	if sec == 0 {
		return ""
	}
	return time.Unix(sec, 0).UTC().Format(time.RFC3339)
}

// firstNonEmpty returns the first non-empty string (defensive Pascal/lower key
// reads — ESC-CASING).
func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// trimRightSlash trims a single trailing slash for redirect-URI assembly.
func trimRightSlash(s string) string { return strings.TrimRight(s, "/") }

// HeaderGetter reads one request header by name. It exists so resolveMCPVK is a
// PURE function unit-tested over a fake getter (no fasthttp).
type HeaderGetter func(name string) string

// resolveMCPVK resolves the virtual key for the /mcp server-mode surface in the
// precedence x-g0-vk > Authorization Bearer <token> > x-api-key, returning ""
// when none is supplied (D4). The header names are g0router-variant of the
// matrix's x-bf-vk chain (PAR-BF-MCP-052, VAR). PURE.
func resolveMCPVK(get HeaderGetter) string {
	if v := get("x-g0-vk"); v != "" {
		return v
	}
	if authz := get("Authorization"); authz != "" {
		if tok, ok := strings.CutPrefix(authz, "Bearer "); ok && tok != "" {
			return tok
		}
	}
	return get("x-api-key")
}

// sanitizeName applies the MCP plugin-name sanitizer (PAR-MCP-048).
func sanitizeName(s string) string { return mcp.SanitizePluginName(s) }

// stripPrefix strips repeated "<server>-" prefixes from a tool name
// (PAR-MCP-046).
func stripPrefix(server, tool string) string { return mcp.StripServerPrefix(server, tool) }

// --- DTOs (key-casing per §1.2: PascalCase clients/instances; snake_case
// tool-groups; OpenAI shape tools; flat accounts) ---

// clientDTO is the marketplace client shape. PascalCase json tags mirror the
// frozen w6-l page's consumed contract (ESC-CASING).
type clientDTO struct {
	ID           string            `json:"ID"`
	Name         string            `json:"Name"`
	Transport    string            `json:"Transport"`
	Command      string            `json:"Command,omitempty"`
	Args         []string          `json:"Args,omitempty"`
	Env          map[string]string `json:"Env,omitempty"`
	URL          string            `json:"URL,omitempty"`
	IsActive     bool              `json:"IsActive"`
	HealthStatus string            `json:"HealthStatus"`
	CreatedAt    string            `json:"CreatedAt,omitempty"`
	UpdatedAt    string            `json:"UpdatedAt,omitempty"`
}

// instanceDTO is the MCP instance shape. PascalCase json tags (§1.2). IsActive /
// HealthStatus are derived from the stored Status via instanceHealth; the raw
// Status is not echoed under a name the page does not read.
type instanceDTO struct {
	ID           string            `json:"ID"`
	ClientID     string            `json:"ClientID,omitempty"`
	Name         string            `json:"Name"`
	Transport    string            `json:"Transport"`
	URL          string            `json:"URL,omitempty"`
	Command      string            `json:"Command,omitempty"`
	Args         []string          `json:"Args,omitempty"`
	Env          map[string]string `json:"Env,omitempty"`
	IsActive     bool              `json:"IsActive"`
	HealthStatus string            `json:"HealthStatus"`
	CreatedAt    string            `json:"CreatedAt,omitempty"`
	UpdatedAt    string            `json:"UpdatedAt,omitempty"`
}

// accountDTO is the masked OAuth account shape — it carries NO token fields and
// NO PKCE verifier (the no-leak discipline, §5).
type accountDTO struct {
	ID         string `json:"id"`
	InstanceID string `json:"instance_id"`
	ServerURL  string `json:"server_url"`
	Status     string `json:"status"`
	Scope      string `json:"scope,omitempty"`
	ExpiresAt  int64  `json:"expires_at,omitempty"`
}

// toolFunctionDTO / toolDTO mirror the OpenAI tool shape the w6-l page reads.
type toolFunctionDTO struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters,omitempty"`
}

type toolDTO struct {
	Type        string          `json:"type"`
	Function    toolFunctionDTO `json:"function"`
	Unavailable bool            `json:"unavailable,omitempty"`
}

// toolGroupDTO is the tool-group shape with snake_case json tags (§1.2).
type toolGroupDTO struct {
	ID        int64    `json:"id"`
	Name      string   `json:"name"`
	ToolIDs   []string `json:"tool_ids"`
	IsActive  bool     `json:"is_active"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at,omitempty"`
}

// unavailableAntigravityTool is the PAR-MCP-060 ride-along: a single tool
// definition hardcoded as unavailable (antigravity.js:433). The antigravity
// EXECUTOR itself is w7-prov-special; this is the tool-definition ride-along
// only (ESC-ANTIGRAVITY).
var unavailableAntigravityTool = toolDTO{
	Type: "function",
	Function: toolFunctionDTO{
		Name:        "mcp_sequential-thinking_sequentialthinking",
		Description: "This tool is currently unavailable.",
	},
	Unavailable: true,
}

// instanceHealth derives the page-consumed IsActive/HealthStatus from the stored
// instance Status. PURE (§1.2).
func instanceHealth(status string) (isActive bool, health string) {
	switch status {
	case "running":
		return true, "healthy"
	case "error":
		return false, "unhealthy"
	default:
		return false, "unknown"
	}
}

func toInstanceDTO(in *store.MCPInstance) instanceDTO {
	active, health := instanceHealth(in.Status)
	return instanceDTO{
		ID:           in.ID,
		ClientID:     in.ClientID,
		Name:         in.Name,
		Transport:    in.Transport,
		URL:          in.URL,
		Command:      in.Command,
		Args:         in.Args,
		Env:          in.Env,
		IsActive:     active,
		HealthStatus: health,
		CreatedAt:    isoFromUnix(in.CreatedAt),
		UpdatedAt:    isoFromUnix(in.UpdatedAt),
	}
}

// --- Clients (read-only marketplace source: store ∪ DefaultPlugins, ESC-CLIENT-SRC) ---

// ListClients handles GET /api/mcp/clients. The data is a bare array of the
// marketplace catalog: the union of stored clients and the default plugin
// definitions (so the marketplace always shows the default catalog).
func (h *Handlers) ListClients(ctx *fasthttp.RequestCtx) {
	out := make([]clientDTO, 0)
	seen := map[string]bool{}

	stored, err := h.store.ListMCPClients()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list mcp clients")
		return
	}
	for _, c := range stored {
		out = append(out, clientDTO{
			ID:           c.ID,
			Name:         c.Name,
			IsActive:     true,
			HealthStatus: "unknown",
			CreatedAt:    isoFromUnix(c.CreatedAt),
			UpdatedAt:    isoFromUnix(c.UpdatedAt),
		})
		seen[c.Name] = true
	}
	for _, p := range mcp.DefaultPlugins() {
		if seen[p.Name] {
			continue
		}
		out = append(out, clientDTO{
			ID:           "default:" + p.Name,
			Name:         p.Name,
			Transport:    p.Transport,
			Command:      p.Command,
			Args:         p.Args,
			URL:          p.URL,
			IsActive:     true,
			HealthStatus: "unknown",
		})
	}
	writeData(ctx, fasthttp.StatusOK, out)
}

// GetClient handles GET /api/mcp/clients/{id}.
func (h *Handlers) GetClient(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	c, err := h.store.GetMCPClient(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "mcp client not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load mcp client")
		return
	}
	writeData(ctx, fasthttp.StatusOK, clientDTO{
		ID:           c.ID,
		Name:         c.Name,
		IsActive:     true,
		HealthStatus: "unknown",
		CreatedAt:    isoFromUnix(c.CreatedAt),
		UpdatedAt:    isoFromUnix(c.UpdatedAt),
	})
}

// --- Instances ---

// instanceRequest reads both PascalCase (page/marketplace) and lower-case keys
// defensively (ESC-CASING note).
type instanceRequest struct {
	Name      string            `json:"Name"`
	NameLower string            `json:"name"`
	ClientID  string            `json:"ClientID"`
	Transport string            `json:"Transport"`
	TransLow  string            `json:"transport"`
	URL       string            `json:"URL"`
	URLLow    string            `json:"url"`
	Command   string            `json:"Command"`
	CmdLow    string            `json:"command"`
	Args      []string          `json:"Args"`
	ArgsLow   []string          `json:"args"`
	Env       map[string]string `json:"Env"`
	EnvLow    map[string]string `json:"env"`
}

func (r *instanceRequest) name() string      { return firstNonEmpty(r.Name, r.NameLower) }
func (r *instanceRequest) transport() string { return firstNonEmpty(r.Transport, r.TransLow) }
func (r *instanceRequest) url() string       { return firstNonEmpty(r.URL, r.URLLow) }
func (r *instanceRequest) command() string   { return firstNonEmpty(r.Command, r.CmdLow) }
func (r *instanceRequest) args() []string {
	if len(r.Args) > 0 {
		return r.Args
	}
	return r.ArgsLow
}
func (r *instanceRequest) env() map[string]string {
	if len(r.Env) > 0 {
		return r.Env
	}
	return r.EnvLow
}

// ListInstances handles GET /api/mcp/instances.
func (h *Handlers) ListInstances(ctx *fasthttp.RequestCtx) {
	instances, err := h.store.ListMCPInstances()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list mcp instances")
		return
	}
	out := make([]instanceDTO, 0, len(instances))
	for _, in := range instances {
		out = append(out, toInstanceDTO(in))
	}
	writeData(ctx, fasthttp.StatusOK, out)
}

// CreateInstance handles POST /api/mcp/instances. It branches on url-vs-command
// (PAR-MCP-022): a command path is allowlist-gated pre-spawn via the launcher
// (PAR-MCP-049, rejecting before any spawn AND before persist); a url path
// records the http/sse mode. The name is sanitized (PAR-MCP-048).
func (h *Handlers) CreateInstance(ctx *fasthttp.RequestCtx) {
	var req instanceRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	name := sanitizeName(req.name())
	if name == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "name is required")
		return
	}
	command := req.command()
	url := req.url()
	if command == "" && url == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "either command or url is required")
		return
	}

	transport := req.transport()
	status := "stopped"

	// Start the transport BEFORE persisting so an allowlist rejection (command
	// path) fails with 400 and nothing is written.
	if command != "" {
		if transport == "" {
			transport = "stdio"
		}
		if h.mcpLauncher == nil {
			writeError(ctx, fasthttp.StatusServiceUnavailable, "mcp launcher unavailable")
			return
		}
		if _, err := h.mcpLauncher.StartStdio(name, command, req.args(), req.env()); err != nil {
			if errors.Is(err, mcp.ErrCommandNotAllowed) {
				writeError(ctx, fasthttp.StatusBadRequest, "command not allowed")
				return
			}
			writeError(ctx, fasthttp.StatusInternalServerError, "start mcp plugin")
			return
		}
		status = "running"
	} else {
		if transport == "" {
			transport = "http"
		}
		if h.mcpLauncher != nil {
			switch transport {
			case "sse":
				_ = h.mcpLauncher.StartSSE(name, url)
			default:
				_ = h.mcpLauncher.StartHTTP(name, url)
			}
		}
	}

	created, err := h.store.CreateMCPInstance(&store.MCPInstance{
		ClientID:  req.ClientID,
		Name:      name,
		Transport: transport,
		URL:       url,
		Command:   command,
		Args:      req.args(),
		Env:       req.env(),
		Status:    status,
	})
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "create mcp instance")
		return
	}
	h.recordAudit(ctx, "mcp_instance.create", created.Name, "Created MCP instance "+created.Name)
	writeData(ctx, fasthttp.StatusCreated, toInstanceDTO(created))
}

// GetInstance handles GET /api/mcp/instances/{id}.
func (h *Handlers) GetInstance(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	in, err := h.store.GetMCPInstance(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "mcp instance not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load mcp instance")
		return
	}
	writeData(ctx, fasthttp.StatusOK, toInstanceDTO(in))
}

// DeleteInstance handles DELETE /api/mcp/instances/{id}. It removes the instance
// and best-effort stops the launcher bridge (PAR-MCP-050 subset — the Cowork
// 3p-config reset is N/A, ESC-COWORK-CONFIG).
func (h *Handlers) DeleteInstance(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	existing, err := h.store.GetMCPInstance(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "mcp instance not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load mcp instance")
		return
	}
	if err := h.store.DeleteMCPInstance(id); errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "mcp instance not found")
		return
	} else if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "delete mcp instance")
		return
	}
	if h.mcpLauncher != nil {
		_ = h.mcpLauncher.Stop(existing.Name)
	}
	h.recordAudit(ctx, "mcp_instance.delete", existing.Name, "Deleted MCP instance "+existing.Name)
	writeData(ctx, fasthttp.StatusOK, map[string]any{})
}

// ListInstanceAccounts handles GET /api/mcp/instances/{id}/accounts. The returned
// accounts are MASKED — they carry no OAuth tokens (§5 no-leak).
func (h *Handlers) ListInstanceAccounts(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	out := make([]accountDTO, 0, 1)
	acct, err := h.store.GetMCPOAuthAccountByInstance(id)
	if err == nil && acct != nil {
		out = append(out, accountDTO{
			ID:         acct.ID,
			InstanceID: acct.InstanceID,
			ServerURL:  acct.ServerURL,
			Status:     acct.Status,
			Scope:      acct.Scope,
			ExpiresAt:  acct.ExpiresAt,
		})
	} else if err != nil && !errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusInternalServerError, "load mcp accounts")
		return
	}
	writeData(ctx, fasthttp.StatusOK, out)
}

// StartInstanceAuth handles POST /api/mcp/instances/{id}/auth/start. It resolves
// the instance's server URL and starts the OAuth flow, returning {url}. The
// state/verifier are NEVER echoed.
func (h *Handlers) StartInstanceAuth(ctx *fasthttp.RequestCtx) {
	id, ok := pathID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	if h.mcpEngine == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "mcp oauth engine unavailable")
		return
	}
	in, err := h.store.GetMCPInstance(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "mcp instance not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load mcp instance")
		return
	}
	if in.URL == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "instance has no server url")
		return
	}
	redirectURI := h.mcpRedirectURI(ctx)
	result, err := h.mcpEngine.Start(context.Background(), in.URL, in.ID, redirectURI)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadGateway, "start mcp oauth")
		return
	}
	h.recordAudit(ctx, "mcp_instance.auth_start", in.Name, "Started MCP OAuth for "+in.Name)
	writeData(ctx, fasthttp.StatusOK, map[string]any{"url": result.AuthURL})
}

// mcpRedirectURI derives the OAuth callback the same way the provider flow does
// (settings override → request Origin → scheme+host), with the MCP callback path
// (ESC-OAUTH-REDIRECT).
func (h *Handlers) mcpRedirectURI(ctx *fasthttp.RequestCtx) string {
	if settings, err := h.store.GetSettings(); err == nil {
		if override, ok := settings["oauth_redirect_uri"]; ok && override != "" {
			return override
		}
	}
	origin := string(ctx.Request.Header.Peek("Origin"))
	if origin == "" {
		scheme := string(ctx.Request.URI().Scheme())
		if scheme == "" {
			scheme = "http"
		}
		host := string(ctx.Request.Host())
		if host == "" {
			host = string(ctx.Request.URI().Host())
		}
		if host != "" {
			origin = scheme + "://" + host
		}
	}
	if origin == "" {
		return ""
	}
	return trimRightSlash(origin) + "/api/mcp/auth/callback"
}

// --- Tools ---

// ListTools handles GET /api/mcp/tools. It aggregates tools across instances via
// the discovery probe (when injected) and includes the antigravity ride-along
// unavailable tool (PAR-MCP-060). Tool names are prefix-stripped (PAR-MCP-046).
func (h *Handlers) ListTools(ctx *fasthttp.RequestCtx) {
	out := make([]toolDTO, 0)
	seen := map[string]bool{}

	instances, err := h.store.ListMCPInstances()
	if err == nil {
		for _, in := range instances {
			if in.URL == "" || h.mcpProbe == nil {
				continue
			}
			res := h.mcpProbe.Run(context.Background(), in.URL)
			for _, pt := range res.Tools {
				name := stripPrefix(in.Name, pt.Name)
				if seen[name] {
					continue
				}
				seen[name] = true
				out = append(out, toolDTO{
					Type:     "function",
					Function: toolFunctionDTO{Name: name, Description: pt.Description},
				})
			}
		}
	}

	// Default-plugin tool names give the marketplace a baseline catalog so the
	// list is non-empty even before any live probe.
	for _, p := range mcp.DefaultPlugins() {
		for _, tn := range p.ToolNames {
			name := stripPrefix(p.Name, tn)
			if seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, toolDTO{
				Type:     "function",
				Function: toolFunctionDTO{Name: name},
			})
		}
	}

	out = append(out, unavailableAntigravityTool)
	writeData(ctx, fasthttp.StatusOK, out)
}

// executeRequest is the tool-execute body.
type executeRequest struct {
	Arguments map[string]any `json:"arguments"`
}

// ExecuteTool handles POST /api/mcp/tools/{name}/execute. It resolves a running
// stdio instance, drives the tool call through the shared bridge-backed
// ToolExecutor (Bridge.Send + capturing SessionSink + smartFilterText), and
// returns {result}. The agent loop uses the SAME executor.
func (h *Handlers) ExecuteTool(ctx *fasthttp.RequestCtx) {
	name, ok := pathID(ctx.UserValue("name"))
	if !ok || name == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	if h.mcpLauncher == nil {
		writeError(ctx, fasthttp.StatusServiceUnavailable, "mcp launcher unavailable")
		return
	}
	var req executeRequest
	if len(ctx.PostBody()) > 0 {
		if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
			return
		}
	}

	bridge, found := h.resolveToolBridge()
	if !found {
		writeError(ctx, fasthttp.StatusNotFound, "no running mcp plugin for tool")
		return
	}

	executor := mcp.NewBridgeToolExecutor(bridge)
	result, err := executor.Execute(context.Background(), name, req.Arguments)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadGateway, "execute mcp tool")
		return
	}
	h.recordAudit(ctx, "mcp_tool.execute", name, "Executed MCP tool "+name)
	writeData(ctx, fasthttp.StatusOK, map[string]any{"result": result})
}

// resolveToolBridge returns the first running plugin bridge. The single-bridge
// resolution is sufficient for the parity bar (the tool-execute path); a
// per-tool routing map is out of scope.
func (h *Handlers) resolveToolBridge() (*mcp.Bridge, bool) {
	instances, err := h.store.ListMCPInstances()
	if err != nil {
		return nil, false
	}
	for _, in := range instances {
		if b, ok := h.mcpLauncher.Bridge(in.Name); ok && b.IsRunning() {
			return b, true
		}
	}
	return nil, false
}

// --- Tool groups (CRUD over the additive numeric-id store table) ---

type toolGroupRequest struct {
	Name     string   `json:"name"`
	ToolIDs  []string `json:"tool_ids"`
	IsActive *bool    `json:"is_active"`
}

func (r *toolGroupRequest) isActive() bool {
	if r.IsActive == nil {
		return true
	}
	return *r.IsActive
}

func toToolGroupDTO(g *store.MCPToolGroup) toolGroupDTO {
	ids := g.ToolIDs
	if ids == nil {
		ids = []string{}
	}
	return toolGroupDTO{
		ID:        g.ID,
		Name:      g.Name,
		ToolIDs:   ids,
		IsActive:  g.IsActive,
		CreatedAt: g.CreatedAt,
		UpdatedAt: g.UpdatedAt,
	}
}

// ListToolGroups handles GET /api/mcp/tool-groups.
func (h *Handlers) ListToolGroups(ctx *fasthttp.RequestCtx) {
	groups, err := h.store.ListMCPToolGroups()
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list mcp tool groups")
		return
	}
	out := make([]toolGroupDTO, 0, len(groups))
	for _, g := range groups {
		out = append(out, toToolGroupDTO(g))
	}
	writeData(ctx, fasthttp.StatusOK, out)
}

// CreateToolGroup handles POST /api/mcp/tool-groups.
func (h *Handlers) CreateToolGroup(ctx *fasthttp.RequestCtx) {
	var req toolGroupRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "name is required")
		return
	}
	created, err := h.store.CreateMCPToolGroup(&store.MCPToolGroup{
		Name:     req.Name,
		ToolIDs:  req.ToolIDs,
		IsActive: req.isActive(),
	})
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "create mcp tool group")
		return
	}
	h.recordAudit(ctx, "mcp_tool_group.create", created.Name, "Created MCP tool group "+created.Name)
	writeData(ctx, fasthttp.StatusCreated, toToolGroupDTO(created))
}

// GetToolGroup handles GET /api/mcp/tool-groups/{id}.
func (h *Handlers) GetToolGroup(ctx *fasthttp.RequestCtx) {
	id, ok := flagID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	g, err := h.store.GetMCPToolGroup(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "mcp tool group not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load mcp tool group")
		return
	}
	writeData(ctx, fasthttp.StatusOK, toToolGroupDTO(g))
}

// UpdateToolGroup handles PUT /api/mcp/tool-groups/{id}.
func (h *Handlers) UpdateToolGroup(ctx *fasthttp.RequestCtx) {
	id, ok := flagID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	var req toolGroupRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "name is required")
		return
	}
	updated, err := h.store.UpdateMCPToolGroup(id, &store.MCPToolGroup{
		Name:     req.Name,
		ToolIDs:  req.ToolIDs,
		IsActive: req.isActive(),
	})
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "mcp tool group not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "update mcp tool group")
		return
	}
	h.recordAudit(ctx, "mcp_tool_group.update", updated.Name, "Updated MCP tool group "+updated.Name)
	writeData(ctx, fasthttp.StatusOK, toToolGroupDTO(updated))
}

// DeleteToolGroup handles DELETE /api/mcp/tool-groups/{id}.
func (h *Handlers) DeleteToolGroup(ctx *fasthttp.RequestCtx) {
	id, ok := flagID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	existing, err := h.store.GetMCPToolGroup(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "mcp tool group not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load mcp tool group")
		return
	}
	if err := h.store.DeleteMCPToolGroup(id); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "delete mcp tool group")
		return
	}
	h.recordAudit(ctx, "mcp_tool_group.delete", existing.Name, "Deleted MCP tool group "+existing.Name)
	writeData(ctx, fasthttp.StatusOK, map[string]any{})
}
