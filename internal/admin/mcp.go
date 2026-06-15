package admin

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/mcp"
	"github.com/bloodf/g0router/internal/store"
	"github.com/valyala/fasthttp"
)

// mcpSSEHeartbeatInterval is the GET /mcp SSE heartbeat interval (PAR-BF-MCP-053:
// ": ping\n\n" every 15s). It is constructed into a real ticker ONLY in
// production wiring (MCPServerSSE); every unit test drives serveMCPSSE's injected
// tick channel so go test opens no socket and sleeps for no real interval (D5).
const mcpSSEHeartbeatInterval = 15 * time.Second

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

// completeAuthRequest is the complete-oauth body: the callback state + code.
type completeAuthRequest struct {
	State string `json:"state"`
	Code  string `json:"code"`
}

// CompleteInstanceAuth handles POST /api/mcp/instances/{id}/auth/complete. It is
// the FIRST live caller of the shipped-but-dead Engine.Complete (oauth.go:88,
// D7): it consumes the persisted PKCE flow, exchanges {state, code} for tokens
// at the discovered token endpoint, and returns the MASKED account (tokens
// STRIPPED; state/verifier NEVER echoed). The create-vs-update distinction is
// ESC (g0router has a single create flow). Session-gated /api/mcp surface.
func (h *Handlers) CompleteInstanceAuth(ctx *fasthttp.RequestCtx) {
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
	var req completeAuthRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.State == "" || req.Code == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "state and code are required")
		return
	}
	redirectURI := h.mcpRedirectURI(ctx)
	acct, err := h.mcpEngine.Complete(context.Background(), in.URL, req.State, req.Code, redirectURI)
	if err != nil {
		writeError(ctx, fasthttp.StatusBadGateway, "complete mcp oauth")
		return
	}
	h.recordAudit(ctx, "mcp_instance.auth_complete", in.Name, "Completed MCP OAuth for "+in.Name)
	writeData(ctx, fasthttp.StatusOK, accountDTO{
		ID:         acct.ID,
		InstanceID: acct.InstanceID,
		ServerURL:  acct.ServerURL,
		Status:     acct.Status,
		Scope:      acct.Scope,
		ExpiresAt:  acct.ExpiresAt,
	})
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

// catalogEntry is the single source-of-truth tool record both the admin DTO
// surface (ListTools) and the /mcp server-mode surface (assembleServerCatalog,
// D3) map from — so the two catalogs never diverge.
type catalogEntry struct {
	Name        string
	Description string
	Unavailable bool
	// Client is the owning MCP client/plugin name for this tool (bf-mcp-2). It is
	// the join key the per-VK scope filter (D4/D6) matches "<client>-*" /
	// "<client>-<tool>" patterns and the AllowOnAllVirtualKeys bypass against. The
	// antigravity ride-along carries an empty Client (no owning client).
	Client string
}

// mcpToolCatalog aggregates the global tool surface ONCE (D3): discovered tools
// across instances (via the probe), the default-plugin baseline, and the
// antigravity ride-along unavailable tool (PAR-MCP-060). Tool names are
// prefix-stripped (PAR-MCP-046). This is the SHARED assembler.
func (h *Handlers) mcpToolCatalog() []catalogEntry {
	out := make([]catalogEntry, 0)
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
				out = append(out, catalogEntry{Name: name, Description: pt.Description, Client: in.Name})
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
			out = append(out, catalogEntry{Name: name, Client: p.Name})
		}
	}

	out = append(out, catalogEntry{
		Name:        unavailableAntigravityTool.Function.Name,
		Description: unavailableAntigravityTool.Function.Description,
		Unavailable: true,
	})
	return out
}

// ListTools handles GET /api/mcp/tools. It maps the SHARED catalog (D3) to the
// OpenAI-shaped DTO the w6-l page reads.
func (h *Handlers) ListTools(ctx *fasthttp.RequestCtx) {
	entries := h.mcpToolCatalog()
	out := make([]toolDTO, 0, len(entries))
	for _, e := range entries {
		out = append(out, toolDTO{
			Type:        "function",
			Function:    toolFunctionDTO{Name: e.Name, Description: e.Description},
			Unavailable: e.Unavailable,
		})
	}
	writeData(ctx, fasthttp.StatusOK, out)
}

// assembleServerCatalog maps the SHARED catalog (D3) to the MCP server-mode
// tools/list shape. The /mcp server-mode surface and the /api/mcp/tools admin
// surface therefore re-expose the SAME aggregated catalog — one source.
func (h *Handlers) assembleServerCatalog() []mcp.ServerTool {
	entries := h.mcpToolCatalog()
	out := make([]mcp.ServerTool, 0, len(entries))
	for _, e := range entries {
		out = append(out, mcp.ServerTool{Name: e.Name, Description: e.Description})
	}
	return out
}

// --- MCP server mode (POST/GET /mcp; D1-D8) ---

// serverCatalogSource adapts the Handlers' shared catalog assembler to the
// mcp.CatalogSource the server-mode dispatcher consumes (D3 — one source).
type serverCatalogSource struct {
	h *Handlers
}

func (s serverCatalogSource) ListServerTools() []mcp.ServerTool {
	return s.h.assembleServerCatalog()
}

// fixedCatalogSource is a CatalogSource over a pre-computed (already-scoped) tool
// slice — the request-time "lazy creation" output for a resolved VK (D1/D3).
type fixedCatalogSource struct {
	tools []mcp.ServerTool
}

func (s fixedCatalogSource) ListServerTools() []mcp.ServerTool { return s.tools }

// scopedDispatcher gates tools/call to a set of admitted tool names before
// delegating to the wrapped dispatcher (D3 — the scope is enforced on BOTH
// tools/list and tools/call so a VK cannot call a tool it cannot see). An
// out-of-scope tool returns an error, surfaced as a JSON-RPC error by the server.
type scopedDispatcher struct {
	allowed map[string]bool
	inner   mcp.ToolDispatcher
}

func (d scopedDispatcher) Execute(ctx context.Context, name string, args map[string]any) (string, error) {
	if !d.allowed[name] {
		return "", errors.New("tool out of scope for virtual key")
	}
	if d.inner == nil {
		return "", errors.New("no tool dispatcher")
	}
	return d.inner.Execute(ctx, name, args)
}

// scopedServerTools computes the request-time tool surface a resolved VK sees
// (D1/D3/D4/D6): it reads the global catalog ONCE, looks up the VK's assignment
// rows (ListVKMCPConfigsByVK), narrows the global slice to the union of the per-
// client executeOnlyTools patterns (scopeTools), and UNIONs every tool whose
// owning client is AllowOnAllVirtualKeys (the D6 bypass) and is not marked
// DisableAutoToolInject (D7). An empty VK or a VK with no assignment rows keeps
// the full global catalog (the un-scoped path is unchanged). The boolean reports
// whether scoping applied (so callers gate tools/call only for a scoped VK).
func (h *Handlers) scopedServerTools(vk string) (tools []mcp.ServerTool, scoped bool) {
	entries := h.mcpToolCatalog()
	global := make([]mcp.ServerTool, 0, len(entries))
	clientOf := make(map[string]string, len(entries))
	for _, e := range entries {
		global = append(global, mcp.ServerTool{Name: e.Name, Description: e.Description})
		clientOf[e.Name] = e.Client
	}

	if vk == "" {
		return global, false
	}
	rec, err := h.store.GetVirtualKeyByKey(vk)
	if err != nil || rec == nil {
		return global, false
	}
	rows, err := h.store.ListVKMCPConfigsByVK(rec.ID)
	if err != nil || len(rows) == 0 {
		return global, false // no assignment: un-scoped (full catalog).
	}

	// AllowOnAllVirtualKeys bypass set (D6) + DisableAutoToolInject suppression
	// set (D7), read from the per-client config blob.
	bypassClients, suppressedClients := h.mcpClientFlagSets()

	// Union the per-client executeOnlyTools patterns across the VK's rows.
	patterns := make([]string, 0)
	for _, r := range rows {
		patterns = append(patterns, r.ToolsToExecute...)
	}

	clientFn := func(tool string) string { return clientOf[tool] }
	admitted := mcp.ScopeTools(global, patterns, clientFn)
	admittedSet := map[string]bool{}
	for _, t := range admitted {
		admittedSet[t.Name] = true
	}

	out := make([]mcp.ServerTool, 0, len(global))
	for _, t := range global {
		owner := clientOf[t.Name]
		if suppressedClients[owner] {
			continue // D7: omit a disable-auto-inject client's tools.
		}
		if admittedSet[t.Name] || bypassClients[owner] {
			out = append(out, t)
		}
	}
	return out, true
}

// mcpClientFlagSets reads the per-client config-blob flags into the bypass
// (AllowOnAllVirtualKeys, D6) and suppression (DisableAutoToolInject, D7) sets,
// keyed by client name. A best-effort read: a store error yields empty sets.
func (h *Handlers) mcpClientFlagSets() (bypass, suppressed map[string]bool) {
	bypass = map[string]bool{}
	suppressed = map[string]bool{}
	clients, err := h.store.ListMCPClients()
	if err != nil {
		return bypass, suppressed
	}
	for _, c := range clients {
		if store.MCPClientAllowOnAllVKs(c) {
			bypass[c.Name] = true
		}
		if store.MCPClientDisableAutoToolInject(c) {
			suppressed[c.Name] = true
		}
	}
	return bypass, suppressed
}

// newMCPServer constructs a per-request server-mode dispatcher over the shared
// catalog (D3) and the running plugin bridge (the shipped dispatch path),
// SCOPED to the resolved VK (D1/D3): a VK with an assignment sees only its
// narrowed tool surface on tools/list AND can only tools/call a tool in that
// surface (the scopedDispatcher gate). An empty/un-assigned VK keeps the full
// global catalog. When no plugin bridge is running, a nil inner dispatcher makes
// an in-scope tools/call return a JSON-RPC internal error while
// initialize/tools/list still serve.
func (h *Handlers) newMCPServer(vk string) *mcp.Server {
	var disp mcp.ToolDispatcher
	if bridge, ok := h.resolveToolBridge(); ok {
		disp = mcp.NewBridgeDispatcher(bridge)
	}
	// Anonymous (absent-VK) path preserves the bf-mcp-1 one-source adapter exactly
	// — no assignment lookup, no extra catalog assembly.
	if vk == "" {
		return mcp.NewServer(serverCatalogSource{h: h}, disp)
	}
	tools, scoped := h.scopedServerTools(vk)
	if !scoped {
		// A provided-but-un-assigned VK keeps the full catalog (one-source adapter).
		return mcp.NewServer(serverCatalogSource{h: h}, disp)
	}
	allowed := make(map[string]bool, len(tools))
	for _, t := range tools {
		allowed[t.Name] = true
	}
	return mcp.NewServer(fixedCatalogSource{tools: tools}, scopedDispatcher{allowed: allowed, inner: disp})
}

// ctxHeaderGetter wraps a fasthttp request's header peek into a HeaderGetter so
// resolveMCPVK runs over the same precedence in production as in its unit test.
func ctxHeaderGetter(ctx *fasthttp.RequestCtx) HeaderGetter {
	return func(name string) string { return string(ctx.Request.Header.Peek(name)) }
}

// admitMCPVK resolves the /mcp virtual key (D4) and validates it: an ABSENT VK
// is allowed (optional global surface; per-VK scoping is bf-mcp-2), a PROVIDED
// VK is validated via the shipped store lookup and REJECTED when unknown or
// inactive. It returns the resolved key (for the deferred audit stamp, D8) and
// whether the request is admitted. This is the resolver's LIVE consumer — a
// provided-but-invalid VK changes the response, so the value is not merely
// attached to context.
func (h *Handlers) admitMCPVK(ctx *fasthttp.RequestCtx) (vk string, admitted bool) {
	key := resolveMCPVK(ctxHeaderGetter(ctx))
	if key == "" {
		return "", true // absent VK: allowed (optional surface).
	}
	rec, err := h.store.GetVirtualKeyByKey(key)
	if err != nil || rec == nil || !rec.IsActive {
		return key, false // provided-but-unknown/inactive: rejected.
	}
	return key, true
}

// writeRawJSONRPC writes a raw JSON-RPC body (NOT the {data,error} envelope) for
// the /mcp surface (D1).
func writeRawJSONRPC(ctx *fasthttp.RequestCtx, status int, body []byte) {
	ctx.SetStatusCode(status)
	ctx.SetContentType("application/json")
	ctx.SetBody(body)
}

// MCPServerPost handles POST /mcp: a JSON-RPC 2.0 request/response over the
// global un-scoped tool surface (D1/D3). It validates any supplied VK (D4),
// dispatches via the shared server-mode dispatcher, and best-effort audits a
// tools/call with the resolved VK stamped (D8). The body is RAW JSON-RPC.
func (h *Handlers) MCPServerPost(ctx *fasthttp.RequestCtx) {
	vk, admitted := h.admitMCPVK(ctx)
	if !admitted {
		writeRawJSONRPC(ctx, fasthttp.StatusOK,
			marshalMCPError("virtual key unknown or inactive"))
		return
	}

	body := ctx.PostBody()
	resp := h.newMCPServer(vk).Dispatch(context.Background(), body)

	// Best-effort audit per tools/call, stamping the resolved VK (D8 payload).
	if name, ok := mcpToolCallName(body); ok {
		h.recordAudit(ctx, "mcp_server.tools_call", name, mcpAuditDetails(name, vk))
	}
	writeRawJSONRPC(ctx, fasthttp.StatusOK, resp)
}

// marshalMCPError builds a minimal raw JSON-RPC 2.0 error frame for transport-
// level rejections (e.g. VK rejection) that occur before dispatch. Uses the
// JSON-RPC invalid-request code.
func marshalMCPError(message string) []byte {
	b, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      nil,
		"error":   map[string]any{"code": -32600, "message": message},
	})
	return b
}

// mcpToolCallName extracts the tools/call tool name from a raw JSON-RPC body,
// reporting false for any other method or a malformed body.
func mcpToolCallName(body []byte) (string, bool) {
	var req struct {
		Method string `json:"method"`
		Params struct {
			Name string `json:"name"`
		} `json:"params"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return "", false
	}
	if req.Method != "tools/call" {
		return "", false
	}
	return req.Params.Name, true
}

// mcpAuditDetails renders the audit detail string for an /mcp tools/call,
// recording the resolved VK (or "anonymous" when absent). NEVER echoes a token.
func mcpAuditDetails(tool, vk string) string {
	who := vk
	if who == "" {
		who = "anonymous"
	}
	return "MCP server tools/call " + tool + " (vk=" + who + ")"
}

// MCPServerSSE handles GET /mcp: the SSE stream (heartbeat + deferred frames).
// It validates any supplied VK (D4 — a provided-but-invalid VK is rejected by
// closing the connection before streaming) and then streams via fasthttp's
// SetBodyStreamWriter, driving the heartbeat from a real ticker. The
// trace/audit completion is DEFERRED until the stream sink closes (D8 — avoids
// the fasthttp body-materialization deadlock). The hermetic seam is serveMCPSSE,
// which the unit tests drive with an injected tick channel (no real timing).
func (h *Handlers) MCPServerSSE(ctx *fasthttp.RequestCtx) {
	vk, admitted := h.admitMCPVK(ctx)
	if !admitted {
		// Reject the connection before streaming (no SSE body).
		writeError(ctx, fasthttp.StatusUnauthorized, "virtual key unknown or inactive")
		return
	}

	interval := h.mcpSSEBeat
	if interval <= 0 {
		interval = mcpSSEHeartbeatInterval
	}

	ctx.SetContentType("text/event-stream")
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")

	clientDone := safeCtxDone(ctx)
	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		h.serveMCPSSE(ctx, w, vk, ticker.C, clientDone, nil)
	})
}

// safeCtxDone returns ctx.Done(), recovering from the panic a bare
// *fasthttp.RequestCtx (no live server) raises so the handler stays callable in
// unit tests (mirrors api/chat.go tryDeriveCancel). A nil channel never fires,
// so the SSE loop relies on the heartbeat-channel close to exit in that case.
func safeCtxDone(ctx *fasthttp.RequestCtx) (done <-chan struct{}) {
	defer func() { _ = recover() }()
	return ctx.Done()
}

// mcpSSEWriter is the flushable sink the SSE loop writes to. It is the same
// shape as the chat.go streamWriter seam (Write/WriteString) plus Flush so a
// test injects an in-memory recorder and asserts the bytes written — no real
// socket. *bufio.Writer satisfies it.
type mcpSSEWriter interface {
	WriteString(s string) (int, error)
	Flush() error
}

// serveMCPSSE runs the SSE loop against the injected sink and heartbeat channel.
// On each tick it emits the literal ": ping\n\n" SSE comment frame (D5). When the
// loop exits (client disconnect / channel close) it runs the DEFERRED finalizer
// (D8): a best-effort recordAudit("mcp_server.tools_call", …) stamping the
// resolved VK — written AFTER the sink closes, never during frame emission,
// avoiding the fasthttp body-materialization deadlock. If done is non-nil it is
// closed on exit so tests can observe termination. PURE of real timing — the
// heartbeat channel is injected.
func (h *Handlers) serveMCPSSE(ctx *fasthttp.RequestCtx, w mcpSSEWriter, vk string, heartbeat <-chan time.Time, clientDone <-chan struct{}, done chan<- struct{}) {
	if done != nil {
		defer close(done)
	}
	// DEFERRED trace/audit completion (D8): runs only after the loop returns,
	// i.e. after the sink closes — a REAL payload, not a no-op.
	defer h.recordAudit(ctx, "mcp_server.tools_call", "sse_session", mcpAuditDetails("sse_session", vk))

	for {
		select {
		case <-clientDone:
			return
		case _, ok := <-heartbeat:
			if !ok {
				return
			}
			if _, err := w.WriteString(": ping\n\n"); err != nil {
				return
			}
			if err := w.Flush(); err != nil {
				return
			}
		}
	}
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

// --- VK↔MCP assignment CRUD (bf-mcp-2 D-routes; serial-held route surface) ---

// vkMCPConfigDTO is the assignment shape with snake_case json tags. It carries
// only tool patterns + flags + the drift-detection config_hash — never a secret
// (the no-leak DTO discipline). config_hash is the live drift-detection reader
// (D8/079): it is RETURNED on read so an operator/client can detect config drift.
type vkMCPConfigDTO struct {
	ID                 int64    `json:"id"`
	VirtualKeyID       string   `json:"virtual_key_id"`
	MCPClientID        string   `json:"mcp_client_id"`
	ToolsToExecute     []string `json:"tools_to_execute"`
	ToolsToAutoExecute []string `json:"tools_to_auto_execute"`
	ConfigHash         string   `json:"config_hash"`
	CreatedAt          string   `json:"created_at,omitempty"`
	UpdatedAt          string   `json:"updated_at,omitempty"`
}

func toVKMCPConfigDTO(c *store.VKMCPConfig) vkMCPConfigDTO {
	exec := c.ToolsToExecute
	if exec == nil {
		exec = []string{}
	}
	auto := c.ToolsToAutoExecute
	if auto == nil {
		auto = []string{}
	}
	return vkMCPConfigDTO{
		ID:                 c.ID,
		VirtualKeyID:       c.VirtualKeyID,
		MCPClientID:        c.MCPClientID,
		ToolsToExecute:     exec,
		ToolsToAutoExecute: auto,
		ConfigHash:         c.ConfigHash,
		CreatedAt:          isoFromUnix(c.CreatedAt),
		UpdatedAt:          isoFromUnix(c.UpdatedAt),
	}
}

// vkMCPConfigRequest is the assignment create/update body (snake_case).
type vkMCPConfigRequest struct {
	VirtualKeyID       string   `json:"virtual_key_id"`
	MCPClientID        string   `json:"mcp_client_id"`
	ToolsToExecute     []string `json:"tools_to_execute"`
	ToolsToAutoExecute []string `json:"tools_to_auto_execute"`
}

// vkMCPConfigHash computes the deterministic drift-detection hash (D8/079): a
// SHA-256 over the canonicalized assignment fields (VK, client, both pattern
// lists). The hash changes iff the assignment changed, so a reader can detect
// drift. PURE over its inputs.
func vkMCPConfigHash(c *store.VKMCPConfig) string {
	payload, _ := json.Marshal([]any{c.VirtualKeyID, c.MCPClientID, c.ToolsToExecute, c.ToolsToAutoExecute})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// ListVKMCPConfigs handles GET /api/mcp/vk-configs?virtual_key_id=. It lists the
// assignments for a virtual key (the {data} array carries the DTO incl.
// config_hash).
func (h *Handlers) ListVKMCPConfigs(ctx *fasthttp.RequestCtx) {
	vk := string(ctx.QueryArgs().Peek("virtual_key_id"))
	rows, err := h.store.ListVKMCPConfigsByVK(vk)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "list vk mcp configs")
		return
	}
	out := make([]vkMCPConfigDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, toVKMCPConfigDTO(r))
	}
	writeData(ctx, fasthttp.StatusOK, out)
}

// CreateVKMCPConfig handles POST /api/mcp/vk-configs. It runs the D5 subset
// validation (rejecting an autoExecute ⊄ execute assignment with a 4xx {error}
// BEFORE storage) and computes the D8 config_hash on write.
func (h *Handlers) CreateVKMCPConfig(ctx *fasthttp.RequestCtx) {
	var req vkMCPConfigRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.VirtualKeyID == "" || req.MCPClientID == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "virtual_key_id and mcp_client_id are required")
		return
	}
	if err := mcp.ValidateAutoExecuteSubset(req.ToolsToExecute, req.ToolsToAutoExecute); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, err.Error())
		return
	}
	rec := &store.VKMCPConfig{
		VirtualKeyID:       req.VirtualKeyID,
		MCPClientID:        req.MCPClientID,
		ToolsToExecute:     req.ToolsToExecute,
		ToolsToAutoExecute: req.ToolsToAutoExecute,
	}
	rec.ConfigHash = vkMCPConfigHash(rec)
	created, err := h.store.CreateVKMCPConfig(rec)
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "create vk mcp config")
		return
	}
	h.recordAudit(ctx, "mcp_vk_config.create", created.VirtualKeyID, "Created VK↔MCP assignment for "+created.VirtualKeyID)
	writeData(ctx, fasthttp.StatusCreated, toVKMCPConfigDTO(created))
}

// GetVKMCPConfig handles GET /api/mcp/vk-configs/{id}. The DTO carries the
// config_hash for drift-detection (D8/079 live reader).
func (h *Handlers) GetVKMCPConfig(ctx *fasthttp.RequestCtx) {
	id, ok := flagID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	c, err := h.store.GetVKMCPConfig(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "vk mcp config not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load vk mcp config")
		return
	}
	writeData(ctx, fasthttp.StatusOK, toVKMCPConfigDTO(c))
}

// UpdateVKMCPConfig handles PUT /api/mcp/vk-configs/{id}. It re-runs the D5 subset
// validation and re-computes the D8 config_hash so a change is reflected in the
// drift-detection hash.
func (h *Handlers) UpdateVKMCPConfig(ctx *fasthttp.RequestCtx) {
	id, ok := flagID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	var req vkMCPConfigRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.VirtualKeyID == "" || req.MCPClientID == "" {
		writeError(ctx, fasthttp.StatusBadRequest, "virtual_key_id and mcp_client_id are required")
		return
	}
	if err := mcp.ValidateAutoExecuteSubset(req.ToolsToExecute, req.ToolsToAutoExecute); err != nil {
		writeError(ctx, fasthttp.StatusBadRequest, err.Error())
		return
	}
	rec := &store.VKMCPConfig{
		VirtualKeyID:       req.VirtualKeyID,
		MCPClientID:        req.MCPClientID,
		ToolsToExecute:     req.ToolsToExecute,
		ToolsToAutoExecute: req.ToolsToAutoExecute,
	}
	rec.ConfigHash = vkMCPConfigHash(rec)
	updated, err := h.store.UpdateVKMCPConfig(id, rec)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "vk mcp config not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "update vk mcp config")
		return
	}
	h.recordAudit(ctx, "mcp_vk_config.update", updated.VirtualKeyID, "Updated VK↔MCP assignment for "+updated.VirtualKeyID)
	writeData(ctx, fasthttp.StatusOK, toVKMCPConfigDTO(updated))
}

// DeleteVKMCPConfig handles DELETE /api/mcp/vk-configs/{id}.
func (h *Handlers) DeleteVKMCPConfig(ctx *fasthttp.RequestCtx) {
	id, ok := flagID(ctx.UserValue("id"))
	if !ok {
		writeError(ctx, fasthttp.StatusBadRequest, "invalid route parameter")
		return
	}
	existing, err := h.store.GetVKMCPConfig(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(ctx, fasthttp.StatusNotFound, "vk mcp config not found")
		return
	}
	if err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "load vk mcp config")
		return
	}
	if err := h.store.DeleteVKMCPConfig(id); err != nil {
		writeError(ctx, fasthttp.StatusInternalServerError, "delete vk mcp config")
		return
	}
	h.recordAudit(ctx, "mcp_vk_config.delete", existing.VirtualKeyID, "Deleted VK↔MCP assignment for "+existing.VirtualKeyID)
	writeData(ctx, fasthttp.StatusOK, map[string]any{})
}
