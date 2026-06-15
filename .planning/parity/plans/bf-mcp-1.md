# Micro-plan bf-mcp-1 — MCP server-mode foundation (Go)

```
program: bifrost-parity (Waves 6+; CLI_ORCHESTRATOR.md governs, HANDOFF.md retired)
plan: bf-mcp-1
status: READY (rev 1 — authored against the merged Waves 0–7 MCP CLIENT tree
  (LIVE in-tree @ internal/mcp/{runner,bridge,launcher,filter,allowlist,defaults,
  process,probe,registry,oauth,sse,healthmonitor,discovery,agent,toolpolicy}.go +
  internal/store/{mcpinstances,mcpoauth,mcptoolgroups}.go + internal/admin/mcp.go +
  the SHIPPED bf-gov chain). BIFROST-MAP.md ledger row bf-mcp-1 §303; bifrost-mcp
  disposition table §259-271; architectural decision #6 §121-138; freeze rules
  §384-399. PRIMARY structural template = the just-shipped bf-gov-1.md (rigor/
  section shape); PRIMARY technical template = the SHIPPED w7-mcp-3 admin/route
  pattern (writeData/writeError envelope, recordAudit, additive routes_admin block
  in ONE commit, the SetMCP* injection seam) + the w7-mcp-2 injected-*http.Client*
  hermetic seam + the w7-plat-2 injected-runner philosophy for the SSE heartbeat
  clock/ticker.)
runs: MCP track. bf-mcp-1 is the FIRST holder of the routes_admin.go MCP serial
  chain (BIFROST-MAP §343-346: bf-mcp-1 → bf-mcp-2 → bf-core-2). Reuses internal/mcp
  CLIENT infra (consume; do NOT rebuild). Runs ∥ openai + gov tracks (disjoint
  domain). RELEASES the routes_admin.go slot to bf-mcp-2 on close.
branch: main (direct commits, per AGENTS.md "No PR workflow")
commit-prefix: phase-1/bf-mcp-1:  (matches the shipped bifrost chain — verified in
  git log: `phase-1/bf-gov-3: close …`)
footer: Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>
ref-source: BLOCKED — ESC-REF-ABSENT (BIFROST-MAP §47-68). The frozen Bifrost ref
  @ ca21298 is ABSENT on this host. The ONLY ground truth is
  .planning/parity/matrix/bifrost-mcp.md. Build to documented matrix behavior +
  g0router's own conventions + the MCP base spec (JSON-RPC 2.0 + SSE) the matrix
  rows themselves name. NEVER build to a guessed Bifrost wire format. Any wire
  detail the matrix note does not capture → STOP-escalate the affected row (§3),
  do NOT invent Bifrost's framing.
go-serial-slot: internal/server/routes_admin.go — bf-mcp-1 TAKES the FIRST MCP-chain
  slot (BIFROST-MAP §343-346). Register the new /mcp routes here OR in a NEW
  internal/server/routes_mcp.go that is CALLED from RegisterAdminRoutes (D2). Either
  way the registration edit to routes_admin.go is the serial-held surface. RELEASE
  to bf-mcp-2 on close. NOT a routes_openai.go holder.
new-route: YES. NEW static endpoints POST /mcp + GET /mcp (D1). These are the ONLY
  new routes. NO new /api/* route, NO UI route file.
```

---

## 0. Objective + ground truth

### 0.1 Objective

Expose g0router **as** an MCP server — the SERVER-MODE direction g0router has never
had — over two new static endpoints `POST /mcp` (JSON-RPC 2.0 request/response) and
`GET /mcp` (SSE stream). The endpoint serves a **global, un-scoped** tool surface:
g0router's existing CLIENT-mode tool catalog (the tools it has discovered across its
running MCP instances, plus the antigravity ride-along) re-exposed to an MCP client
that connects TO g0router. The plan plumbs **VK resolution** for `/mcp`
(`x-g0-vk > Bearer > x-api-key`), a hermetic **SSE heartbeat** (`: ping` every 15s
via an injected ticker/clock — NO real timing in tests), **deferred trace/usage
completion** for the SSE path, aligns the **MCP client-config table** to the
server-mode needs, and wires the **complete-oauth** endpoint to the already-shipped
but currently-unconsumed `Engine.Complete`.

Additive-only: a NEW `internal/mcp/server.go` (the server-mode JSON-RPC dispatcher
re-exposing the existing catalog), a NEW `internal/server/routes_mcp.go` (or an
additive block in `routes_admin.go`) registering `/mcp`, an additive
`StartInstanceAuthComplete`-style handler + route for complete-oauth, and any
additive column the client-config-table-alignment row genuinely requires. The
server-mode dispatcher REUSES the SHIPPED CLIENT-mode primitives — it does NOT
duplicate JSON-RPC framing or SSE helpers:

- `internal/mcp/bridge.go:12 SessionSink` + `:91 broadcast` + `:136 splitFrames`
  (the newline-delimited JSON-RPC frame split — REUSE for outbound SSE frames).
- `internal/mcp/sse.go:112 parseSSEFrame` + `:127 parseSSEDataFrames`
  (SSE wire helpers — REUSE; the heartbeat reuses the same `: ping` SSE-comment
  convention).
- `internal/mcp/agent.go:127 NewBridgeToolExecutor` + `:163 tools/call` framing
  (the existing `tools/call` dispatch path — REUSE for server-mode `tools/call`).
- `internal/admin/mcp.go:501 ListTools` aggregation + `:592 resolveToolBridge`
  (the existing tool-catalog assembly + bridge resolution — the SOURCE of the
  "global tool surface" the server exposes — D3).

This is the bifrost-mcp analogue of BIFROST-MAP decision #6: *g0router's MCP is
CLIENT-mode + rich; bifrost-mcp's server-mode is genuinely NEW; the client-mode
infra is REUSABLE adjacent infrastructure.* Per-VK scoping, per-user OAuth/header
credential flows, and the sessions API are explicitly OUT (bf-mcp-2 / ESC — §3).

### 0.2 Rows this plan closes (matrix = ground truth)

| Row | Behavior (matrix) | Disposition | Flip target after bf-mcp-1 |
|---|---|---|---|
| PAR-BF-MCP-002 | MCP server mode: expose g0router as MCP server over HTTP JSON-RPC + SSE; routes `/mcp` POST+GET | **BUILD** | NEW `server.go` dispatcher + `POST/GET /mcp` (D1/D2). MISSING→**HAVE** (global un-scoped surface; per-VK = bf-mcp-2). |
| PAR-BF-MCP-003 | MCP server mode: global MCP server (un-scoped tools) | **BUILD** | A single global server instance serving the existing aggregated catalog (D3). MISSING→**HAVE**. |
| PAR-BF-MCP-032 | MCP client config table (`config_mcp_clients`) with encryption at rest | **BUILD (align)** | g0router already has `mcp_clients` + `mcp_oauth_accounts`/`mcp_oauth_flows` (`*_enc`) shipped in w7-mcp-1. bf-mcp-1 ALIGNS the existing table additively ONLY for the server-mode/complete-oauth fields it genuinely consumes (D6). MISSING→**PARTIAL** (g0router-variant table; the full Bifrost column set with per-user/code-mode/pricing columns = ESC). |
| PAR-BF-MCP-039 | POST `/api/mcp/client/{id}/complete-oauth` (distinguishes create vs update flow) | **BUILD** | Wire a complete-oauth handler over the already-shipped, currently-UNCONSUMED `Engine.Complete` (`internal/mcp/oauth.go:88`) — D7. g0router has no update-vs-create branch (single create flow) → the create branch is built; the update branch = ESC. MISSING→**PARTIAL** (create flow HAVE; update-flow distinction ESC). |
| PAR-BF-MCP-052 | VK resolution for MCP server: `x-bf-vk` > Authorization Bearer `vk_*` > `x-api-key vk_*` | **BUILD (VAR header)** | `x-g0-vk > Bearer > x-api-key` precedence resolver for `/mcp` (D4). The matrix's `x-bf-vk` maps to g0router's `x-g0-vk` (VAR). MISSING→**HAVE** (g0router header names). |
| PAR-BF-MCP-053 | SSE heartbeat: `: ping\n\n` every 15s to detect disconnect via reader.Send() | **BUILD** | Hermetic heartbeat via an injected ticker/clock (D5). MISSING→**HAVE**. |
| PAR-BF-MCP-054 | Trace completion deferred for SSE to avoid fasthttp body materialization deadlock | **BUILD** | Defer completion until the SSE stream closes, with a REAL deferred payload — a best-effort `recordAudit` per `tools/call` stamping the resolved VK (D8). MISSING→**HAVE** *if the audit payload is wired* (preferred path (a), `recordAudit` verified reachable on `*admin.Handlers`); **PARTIAL** only on the fallback (deferral mechanism present, no payload) — NOT HAVE on a no-op. The exact Bifrost `BifrostContextKeyDeferTraceCompletion` key is VAR. |
| PAR-BF-MCP-075 | `mcp-go` server integration for JSON-RPC message handling (`mcpServer.HandleMessage`) | **BUILD (VAR — own bridge)** | g0router rolls its OWN JSON-RPC bridge (no `mark3labs/mcp-go` dep) — `server.go` dispatches `initialize`/`tools/list`/`tools/call` over the SHIPPED framing (D1). MISSING→**HAVE-by-variant** (the BEHAVIOR — JSON-RPC message handling — is built; the `mcp-go` DEPENDENCY is VAR, recorded). |

**Honest scoping note:** 002/003/052/053 close fully (global surface). 054 closes
**HAVE** only on the preferred path (a) — the deferred finalizer carries a real
`recordAudit` payload (D8); on the fallback (no payload) it flips **PARTIAL**, never
HAVE on a no-op. 032/039 close to **PARTIAL** because the full Bifrost table
column-set and the create-vs-update complete-oauth distinction depend on the
per-user/enterprise surface that is ESC; the g0router-shaped subset (server-mode
columns + single create flow) is built and the residual is recorded in
`open-questions.md` (§7). 075 is HAVE-by-variant (own bridge, no `mcp-go`). No row is
closed by inventing un-evidenced Bifrost framing — see the STOP-condition in D1.

### 0.3 Preconditions already satisfied (evidence — read 3 files, AGENTS.md)

- **The JSON-RPC framing primitives EXIST (REUSE — do NOT rebuild).**
  `internal/mcp/bridge.go:136 splitFrames(buf) (frames [][]byte, rest []byte)` — the
  PURE newline-delimited JSON-RPC frame splitter; `:91 broadcast(frame)` — fan-out to
  every `SessionSink`; `:12 type SessionSink func(frame []byte) error`. The
  server-mode dispatcher's outbound SSE writes REUSE this `SessionSink` shape.
- **The SSE wire helpers EXIST (REUSE).** `internal/mcp/sse.go:112 parseSSEFrame` +
  `:127 parseSSEDataFrames`. The CLIENT `sseClient` (`sse.go:29`) connects OUT to
  remote servers; bf-mcp-1's SERVER path emits SSE inbound-to-g0router but reuses the
  same `event:`/`data:`/`: comment` line conventions (D5). `defaultHTTPClient`
  (`sse.go:16`) is the injectable-client precedent.
- **The `tools/call` dispatch path EXISTS (REUSE).**
  `internal/mcp/agent.go:127 NewBridgeToolExecutor(b *Bridge) ToolExecutor`;
  `:149 Execute(ctx, name, args)` builds a `tools/call` JSON-RPC frame
  (`:163-165 {"jsonrpc":"2.0", ..., "method":"tools/call"}`), `Bridge.Send`s it
  (`bridge.go:122`), and waits for the matching response. The server-mode
  `tools/call` handler DELEGATES to this — it does NOT re-implement tool execution.
- **The "global tool surface" SOURCE EXISTS (REUSE — D3).**
  `internal/admin/mcp.go:501 ListTools` aggregates `[]ProbeTool` across every running
  instance (`:505 ListMCPInstances` → `:511 mcpProbe.Run(ctx, in.URL)` →
  `:518 toolDTO`) and appends `:118 unavailableAntigravityTool`.
  `:592 resolveToolBridge()` returns the first running plugin `*mcp.Bridge`. This is
  the live catalog + dispatch the server re-exposes.
- **The OAuth engine + `Complete` EXIST but `Complete` has NO live caller (D7).**
  `internal/mcp/oauth.go:88 Engine.Complete(ctx, serverURL, state, code, redirectURI)
  (*store.MCPOAuthAccount, error)` — verified via grep: `\.Complete\(` has ZERO
  non-test production callers. bf-mcp-1's complete-oauth route is its FIRST live
  consumer (no-leftovers: the route makes a shipped-but-dead method live).
  `:39 StartResult{AuthURL,State}`; `:303 maskAccount` (tokens stripped on return).
- **The MCP store + tables EXIST (consume; align additively).**
  `internal/store/mcpinstances.go` (clients/instances CRUD),
  `internal/store/mcpoauth.go:44 UpsertMCPOAuthAccount` (tokens `*_enc`),
  `:139 CreateMCPOAuthFlow` (verifier `*_enc`), `:158 ConsumeMCPOAuthFlow`. The four
  `mcp_*` tables shipped in w7-mcp-1 (`mcp_clients`, `mcp_instances`,
  `mcp_oauth_accounts`, `mcp_oauth_flows`). Additive `ensureColumn`/`ensureTable`
  only (`migrate.go`).
- **The admin injection seam + handlers EXIST (consume).**
  `internal/server/routes_admin.go:100-102` already constructs + injects
  `h.SetMCPLauncher(mcp.NewLauncher(st))`, `h.SetMCPEngine(mcp.NewEngine(st, nil))`,
  `h.SetMCPProbe(mcp.NewProbe(nil))`. `internal/admin/mcp.go:434 StartInstanceAuth`
  (calls `Engine.Start`), `:407 ListInstanceAccounts` (tokens stripped via
  `maskAccount`). bf-mcp-1 ADDS a sibling complete-oauth handler the SAME way.
- **The VK resolver + header convention EXIST (extend additively — D4).**
  `internal/api/chat.go:365 ctx.Request.Header.Peek("x-g0-vk")` — the ONLY header
  read today; `internal/server/routes_openai.go:165 vkResolverAdapter.ResolveVK(key)`
  → `:166 GetVirtualKeyByKey(key)`. bf-mcp-1 adds the Bearer/x-api-key FALLBACK for
  the `/mcp` resolution (the precedence chain is NEW for the MCP endpoint).
- **The HTTP router auto-orders static-before-param (precondition for /mcp).**
  `internal/server/routes_admin.go:16 "github.com/fasthttp/router"` — fasthttp's
  router resolves static segments before `{param}` and panics on a true conflict.
  `/mcp` is a brand-new static path with NO `/api/*` or `/v1/*` conflict.
- **The hermetic SSE-stream seam EXISTS (the heartbeat-test precedent — D5).**
  `internal/api/chat.go:23-30 streamWriter interface {Write; WriteString}` — "exists
  so tests can inject write failures … fasthttp's in-memory response buffer never
  returns errors." bf-mcp-1's SSE writer accepts the SAME `streamWriter` (or an
  equivalent flushable sink) so the heartbeat is driven by a test, NOT a real socket;
  the 15s interval is an injected ticker/clock, NOT `time.Sleep`.
- **The guard local-only list (decision point — D2).**
  `internal/server/guard.go:45 LOCAL_ONLY_PATHS = ["/api/mcp/", …]`. `/mcp` (bare) is
  NOT under `/api/mcp/`; it is the public MCP-server surface gated by VK resolution,
  NOT local-only. Do NOT add `/mcp` to LOCAL_ONLY_PATHS (D2 / open-question).

---

## 1. Decisions made (and why) — binding

### D1 — The `/mcp` JSON-RPC 2.0 wire contract + served methods (own bridge — VAR vs mcp-go)

`/mcp` is a JSON-RPC 2.0 endpoint, NOT the `{data,error}` admin envelope and NOT a
`/v1/*` OpenAI surface. Its request/response bodies are raw JSON-RPC 2.0 objects
(`{"jsonrpc":"2.0","id":N,"method":...,"params":...}` →
`{"jsonrpc":"2.0","id":N,"result":...}` or `{"jsonrpc":"2.0","id":N,"error":{code,
message}}`). This contract is documented explicitly so no reviewer mistakes it for
the admin envelope.

**Decision — served methods (the matrix-evidenced minimum):** `server.go` dispatches
exactly the three methods the matrix names for server-mode and the existing client
primitives already support:
1. **`initialize`** — returns `{protocolVersion, capabilities:{tools:{}}, serverInfo:
   {name:"g0router", version:<build>}}`. The `protocolVersion` REUSES the shipped
   `mcpProtocolVersion = "2025-06-18"` constant (`probe.go:17`) so client and server
   agree on the version g0router already speaks.
2. **`tools/list`** — returns `{tools:[{name, description, inputSchema}]}` built from
   the global catalog (D3).
3. **`tools/call`** — `{name, arguments}` → delegates to the SHIPPED
   `NewBridgeToolExecutor(bridge).Execute` (`agent.go:127/149`) over the resolved
   running bridge (`resolveToolBridge`), applies the SHIPPED `smartFilterText`, and
   returns `{content:[{type:"text", text:<result>}]}`.

g0router rolls its OWN dispatcher (no `mark3labs/mcp-go` dependency) — PAR-BF-MCP-075
is **HAVE-by-variant**: the BEHAVIOR (JSON-RPC message handling) is built over the
shipped framing; the DEPENDENCY is a deliberate VAR (g0router has rolled its own
bridge since w7-mcp-1, AGENTS.md "No mocks; … test real behavior"; adding `mcp-go`
would duplicate shipped code).

**STOP-condition (ESC-REF-ABSENT, binding):** the matrix names the methods and the
POST/GET split but does NOT capture Bifrost's exact `initialize` capability object,
its error-code numbering, or any non-standard framing. bf-mcp-1 builds to the **base
MCP/JSON-RPC 2.0 spec** the matrix rows themselves cite (`jsonrpc:"2.0"`, the
`-32600`-family JSON-RPC error codes, the standard `initialize`/`tools/list`/
`tools/call` shapes g0router's own client `probe.go` already emits). If, at impl, a
reviewer requires a Bifrost-SPECIFIC field/code that is NOT in the matrix note and
NOT in the base spec g0router's client already uses, the affected behavior **STOPS
and escalates** rather than fabricating Bifrost's protocol. The methods above are
safe to build because g0router's own CLIENT already speaks them (`probe.go:61,76,80`
emits `initialize`/`notifications/initialized`/`tools/list`; `agent.go:165` emits
`tools/call`) — the server is the mirror of the client g0router already ships.

### D2 — Route registration: NEW `routes_mcp.go` called from `RegisterAdminRoutes`

**Decision:** add a NEW `internal/server/routes_mcp.go` exposing
`RegisterMCPRoutes(r *router.Router, h *admin.Handlers)` and CALL it from
`RegisterAdminRoutes` (`routes_admin.go:119`) with a one-line additive append. This
keeps the bulk of the server-mode wiring out of the serial hot file while still
taking the serial slot for the one-line registration edit (the registration is the
contended surface; isolating it minimizes the hot-file diff — the W3–W7 lesson). The
new routes:
```
r.POST("/mcp", h.MCPServerPost)     // JSON-RPC request/response
r.GET ("/mcp", h.MCPServerSSE)      // SSE stream (heartbeat + deferred frames)
```
plus the complete-oauth route (D7), registered alongside the existing `/api/mcp/*`
block:
```
r.POST("/api/mcp/instances/{id}/auth/complete", h.RequireSession(h.CompleteInstanceAuth))
```
`/mcp` is a NEW static path; fasthttp's router auto-orders static-before-`{param}`
and there is no conflict with `/api/*` or `/v1/*` (§0.3). **`/mcp` is NOT added to
`guard.go:45 LOCAL_ONLY_PATHS`** — it is the public MCP-server surface authenticated
by VK resolution (D4), not a local-only admin path. (Whether `/mcp` should ALSO be
local-only-gated in some deployments is an open question — §7 — default: VK-gated
public, mirroring the matrix's VK-resolution row 052.)

### D3 — The "global tool surface" source = the existing aggregated catalog

**Decision (matrix-confirmed):** PAR-BF-MCP-003 "global MCP server (un-scoped tools)"
= g0router's EXISTING client-mode catalog re-exposed. The source is the SAME
aggregation `internal/admin/mcp.go:501 ListTools` already computes: for each running
instance, the discovered `[]ProbeTool` (via `mcpProbe.Run`) mapped to `{name,
description, inputSchema}`, plus `unavailableAntigravityTool` (`:118`). `server.go`'s
`tools/list` calls a SHARED catalog assembler (extracted from / shared with
`ListTools` so the admin DTO surface and the server-mode surface return the SAME
catalog — no second source of truth). `tools/call` resolves the owning running bridge
via the SAME `resolveToolBridge` path and dispatches via `NewBridgeToolExecutor`.

**No-leftovers + ambiguity guard:** "global un-scoped" means NO per-VK filtering in
bf-mcp-1 (per-VK scoping is bf-mcp-2 — §3). The VK resolved in D4 is plumbed and
validated (a provided-but-invalid VK is rejected; an absent VK is allowed — D4) but
does NOT yet scope the tool list (a valid VK and an absent VK see the SAME catalog). If, at impl, the matrix's "global tool surface" turns out to
mean a source OTHER than the existing aggregated catalog (e.g. a separate Bifrost
registry the matrix references but g0router lacks), **STOP and escalate** rather than
inventing a new catalog source — the existing aggregation is the only g0router-grounded
source and the matrix note (row 003: "globalMCPServer … un-scoped tools") supports it.

### D4 — VK resolution for `/mcp`: `x-g0-vk > Bearer > x-api-key` (VAR header names)

PAR-BF-MCP-052 wants `x-bf-vk > Authorization Bearer vk_* > x-api-key vk_*`.
g0router's header is `x-g0-vk` (`chat.go:365`), and g0router VK values are `g0vk-`
prefixed (`virtualkeys.go:78`), NOT `vk_*`.

**Decision:** add a PURE resolver `resolveMCPVK(hdr HeaderGetter) (key string)` that
returns the first non-empty of, in order: (1) `x-g0-vk`, (2) the `Authorization:
Bearer <token>` token, (3) `x-api-key`. The resolved key is passed to the SHIPPED
`vkResolverAdapter.ResolveVK` / `store.GetVirtualKeyByKey`
(`routes_openai.go:165-166`) — bf-mcp-1 REUSES the existing VK resolution, it only
adds the multi-header PRECEDENCE wrapper for the `/mcp` endpoint. The `x-bf-vk →
x-g0-vk` and `vk_* → g0vk-` mappings are **VAR** (g0router naming, recorded). The
PURE precedence function is unit-tested over a fake header getter (every order case +
empty).

**The resolved VK is genuinely CONSUMED via VALIDATION (the live consumer — closes
the resolved-but-unread inertness trap, the bf-gov-1 "dead ValidateBudgetOwner"
lesson).** `MCPServerPost`/`MCPServerSSE`, when ANY of the three headers supplies a
VK, **validate it** through the shipped `vkResolverAdapter.ResolveVK` /
`store.GetVirtualKeyByKey` (and the existing active check the gate already applies —
`vk.go`): if the supplied VK is **unknown OR inactive**, the request is rejected — a
JSON-RPC error on `POST /mcp`, and a connection reject on `GET /mcp` (close the SSE
before streaming). An **ABSENT** VK (no header supplied) remains **allowed** (the
global un-scoped surface stays open by default — optional VK). This is a REAL,
scoping-independent consumer: a provided-but-invalid VK changes the response (reject),
so the resolved value is not merely "attached to context" — it gates admission. It
does **NOT** do per-VK tool scoping (still bf-mcp-2 — §3): a valid VK is admitted to
the SAME global catalog as an absent VK.

Whether VK resolution is **mandatory** (reject even when no VK is supplied) vs the
bf-mcp-1 default (**optional**: absent allowed, present-and-invalid rejected) is
recorded §7; mandatory-VK + per-VK scoping is bf-mcp-2.

**No-leftovers (binding):** the resolver's live consumer is *validation-and-rejection*
(a provided-but-unknown/inactive VK returns a JSON-RPC error / SSE reject), grep- and
behavior-proven in §5 — NOT a mere context attachment. If the validation path has no
live effect (i.e. an invalid VK is not actually rejected), STOP+escalate per §3.

### D5 — Hermetic SSE heartbeat (`: ping` every 15s via injected ticker/clock)

PAR-BF-MCP-053: `: ping\n\n` every 15s. The hard constraint: **NO real 15s sleep, NO
real socket timing in tests** (Wave-7 hermetic lesson; the w7-plat-2 injected-runner /
w7-mcp-2 injected-clock pattern).

**Decision:** the SSE writer is driven by an INJECTED heartbeat source. The SSE
handler signature accepts (a) a `streamWriter`-style flushable sink (REUSE the
`chat.go:23 streamWriter` interface so a test injects an in-memory recorder that
captures the bytes written — no real socket) and (b) an injected heartbeat tick
channel / clock. The production constructor wires a real `time.NewTicker(15 *
time.Second)`; the test injects a `chan time.Time` it fires manually (or a fake clock
the test advances). On each tick the writer emits the literal SSE comment frame
`: ping\n\n` (an SSE comment line — the same `:`-prefixed convention `parseSSEFrame`
already tolerates). The heartbeat-interval constant is
`mcpSSEHeartbeatInterval = 15 * time.Second` (matrix-evidenced literal). The unit
test fires the injected tick N times and asserts N `: ping\n\n` frames landed in the
recorder, with ZERO real elapsed time. **Binding:** the 15s ticker is constructed
ONLY in production wiring; every test drives the injected channel/clock — `go test
./...` opens no socket and sleeps for no real interval (§5 hermetic grep proof).

### D6 — Client-config table alignment: additive, server-mode subset ONLY

PAR-BF-MCP-032 wants `config_mcp_clients` with encryption at rest. g0router ALREADY
has `mcp_clients` + the `*_enc` OAuth tables (w7-mcp-1). The full Bifrost column-set
(`per_user_header_keys_json`, `is_code_mode_client`, `tool_pricing_json`,
`config_hash`, `allow_on_all_virtual_keys`, …) is the per-user/enterprise/code-mode
surface that is ESC (§3).

**Decision:** bf-mcp-1 adds an additive column to `mcp_clients` (via `ensureColumn`)
ONLY IF the server-mode dispatcher or complete-oauth flow genuinely consumes it. The
**default expectation is ZERO new columns** — the server re-exposes the existing
catalog (D3) and complete-oauth uses the existing `mcp_oauth_accounts`/`flows`
tables. **No-leftovers (binding):** if no server-mode behavior in this plan reads or
writes a new column, NO column is added — the row flips to PARTIAL on the strength of
the EXISTING table satisfying the server-mode/encryption-at-rest subset, with the
enterprise columns recorded as ESC in `open-questions.md`. Adding a dead column is an
automatic REJECT (§3 / Wave-5 dead-wiring lesson). If, at impl, complete-oauth needs
a server-mode field absent from the current schema, add ONLY that field additively
and prove its live consumer in §5.

### D7 — complete-oauth: wire the shipped-but-dead `Engine.Complete` (create flow only)

PAR-BF-MCP-039: `POST …/complete-oauth` distinguishing create vs update flow.
`Engine.Complete` (`oauth.go:88`) is shipped but has ZERO live callers (§0.3).

**Decision:** add an admin handler `CompleteInstanceAuth` (sibling of the shipped
`StartInstanceAuth` @ `admin/mcp.go:434`) on
`POST /api/mcp/instances/{id}/auth/complete` (g0router's instance-scoped path; the
Bifrost `/api/mcp/client/{id}/complete-oauth` maps to g0router's instance route —
VAR). It reads `{state, code}` from the request, resolves the instance's server URL +
the configured redirect URI (REUSE `admin/mcp.go:470 mcpRedirectURI`), calls
`Engine.Complete(ctx, serverURL, state, code, redirectURI)`, and returns the masked
account (`maskAccount` — tokens STRIPPED, never echoed) under the `{data}` envelope.
This gives the dead method its first live consumer (no-leftovers). The matrix's
**create-vs-update distinction is ESC**: g0router has a single create flow (no
in-place client update OAuth re-rotation — PAR-BF-MCP-051 OAuth-rotation-disabled is
ESC, §3); the create branch is built, the update branch is recorded ESC. The handler
is on the `/api/mcp/` (local-only, session-gated) surface, NOT the public `/mcp`
JSON-RPC surface. **No-leftovers:** `CompleteInstanceAuth` MUST call `Engine.Complete`
(grep-proven §5) or STOP+escalate.

### D8 — Deferred trace/usage completion for the SSE path (VAR key)

PAR-BF-MCP-054: defer trace completion for SSE to avoid fasthttp body-materialization
deadlock. Bifrost uses a context key `BifrostContextKeyDeferTraceCompletion`.

**Decision:** the SSE handler (`MCPServerSSE`) does NOT finalize trace/audit
recording inline (which would force fasthttp to materialize the streaming body and
deadlock — the same hazard `chat.go` streaming already navigates with
`SetBodyStreamWriter`). Instead it defers completion to AFTER the stream writer
returns (the SSE loop watches `ctx.Done()` and the writer closure runs the deferred
finalizer on exit — mirroring `chat.go`'s `writeSSEStream` returning before any
post-stream bookkeeping). The exact Bifrost context-key NAME
(`BifrostContextKeyDeferTraceCompletion`) is **VAR** (g0router does not need a magic
context key — it defers via the streaming writer's return ordering).

**The deferred finalizer has a REAL payload via shipped infra (closes the inert-no-op
trap — preferred path (a), verified reachable).** The finalizer runs a **best-effort
audit entry** per `/mcp tools/call` through the shipped `recordAudit` seam
(`internal/admin/audit.go:64 func (h *Handlers) recordAudit(ctx, action, target,
details)` — already called across w7/governance and inside this very file at
`internal/admin/mcp.go:585 h.recordAudit(ctx, "mcp_tool.execute", …)`). The MCP
server-mode finalizer stamps `recordAudit(ctx, "mcp_server.tools_call", toolName,
<details incl. the resolved VK from D4>)` — which ALSO reinforces D4's live VK
consumer (the resolved VK is written into a real audit record). The deferral ORDERING
(the finalizer's `recordAudit` write runs AFTER the SSE sink closes, never during
frame emission — avoiding the fasthttp materialization deadlock) is thereby exercised
over a REAL write, not a no-op. `recordAudit` is best-effort (never fails the parent —
`audit.go`), so it is deadlock-safe in the deferred slot.

The unit test asserts (1) the audit entry is written AFTER the injected stream sink
closes, NOT during frame emission (hermetic — an in-memory sink + the test store's
audit table, no real fasthttp body, no real socket), and (2) the audit `details`
carries the resolved VK. **This makes 054 → HAVE honestly** (deferral mechanism +
real deferred payload).

**FALLBACK (only if (a) proves unreachable at impl):** if `recordAudit` turns out NOT
cleanly callable from the MCP server-mode handler (it is on `*admin.Handlers` and the
`/mcp` handlers ARE methods on `*admin.Handlers` per §2, so this is expected to be
reachable — verify at impl), then implement the deferral-ordering mechanism with an
injected finalizer hook (the test flips a recorder flag asserting post-close
ordering) and flip **054 → PARTIAL** (deferral mechanism present; no audit/usage
payload wired yet), NOT HAVE, recording the residual in `open-questions.md`. **Do NOT
flip 054 HAVE on a pure no-op finalizer.** The specific usage/cost shape for an MCP
`tools/call` is NOT invented (ESC-REF-ABSENT) — the audit entry is the parity-
meaningful deferred payload; "what numeric usage is recorded for a /mcp tools/call"
stays an open question (§7).

---

## 2. Target files

### IN-SCOPE — CREATE (NEW)

| File | Contract |
|---|---|
| `internal/mcp/server.go` | `Server` — the server-mode JSON-RPC dispatcher. `NewServer(catalog CatalogSource, exec ToolDispatcher)` (additive constructor; injectable catalog + dispatch so tests feed canned tools + a fake executor). `Dispatch(req []byte) (resp []byte, err error)` routes `initialize`/`tools/list`/`tools/call` (D1) over the SHIPPED `splitFrames` framing + `NewBridgeToolExecutor` (REUSE — no duplicate framing). PURE-ish dispatch core (no I/O) so it is fully unit-tested with canned JSON-RPC requests. No `init()`; errors-as-values. |
| `internal/mcp/server_test.go` | RED first: `initialize`→version/capabilities; `tools/list`→canned catalog; `tools/call`→fake executor result + `smartFilterText`; unknown method→JSON-RPC error; malformed JSON→`-32700`. Hermetic — no network/process. |
| `internal/server/routes_mcp.go` | `RegisterMCPRoutes(r, h)` registering `POST /mcp`, `GET /mcp` (D2). Thin; the handlers live on `*admin.Handlers`. |
| `internal/server/routes_mcp_test.go` | RED first: routes registered; `POST /mcp` dispatches a canned `tools/list` and returns JSON-RPC (not the `{data,error}` envelope); VK precedence resolved (D4); `GET /mcp` heartbeat fires via injected tick (D5); `/mcp` not in LOCAL_ONLY. Hermetic. |

### IN-SCOPE — EXTEND (additive only)

| File | Change (additive ONLY) |
|---|---|
| `internal/admin/mcp.go` | ADD `MCPServerPost` + `MCPServerSSE` handlers (the `/mcp` JSON-RPC + SSE entry points: VK-resolve via D4, delegate to `mcp.Server.Dispatch`, SSE heartbeat via injected ticker D5, deferred finalizer D8). ADD `CompleteInstanceAuth` (D7, calls `Engine.Complete`). ADD the PURE `resolveMCPVK` precedence helper (D4). EXTRACT/SHARE the catalog assembler so `ListTools` + `Server.tools/list` use ONE source (D3). PRESERVE all existing handler bodies/signatures. |
| `internal/admin/mcp_test.go` | RED first: `MCPServerPost` JSON-RPC round-trips; `resolveMCPVK` precedence (every order + empty); `MCPServerSSE` heartbeat via injected tick (hermetic, NO real 15s); deferred-finalizer ordering (D8); `CompleteInstanceAuth` calls `Engine.Complete` + returns masked account (NO token leak). |
| `internal/admin/handlers.go` | ADD nil-able field(s) for the server-mode `*mcp.Server` + the injected heartbeat ticker source IF the handler needs them as fields; ADD `SetMCPServer`-style additive setter mirroring the shipped `SetMCPLauncher`/`SetMCPEngine`/`SetMCPProbe` (`routes_admin.go:100-102`). NO `New(...)` signature change. (DECIDE at impl whether the server is a field or constructed per-request from the existing launcher/probe — default: a nil-able `mcpServer` field + setter; degrade to 503 when unset, mirroring the shipped nil-able pattern.) |
| `internal/server/routes_admin.go` | ADD a one-line `RegisterMCPRoutes(r, h)` call + the `POST /api/mcp/instances/{id}/auth/complete` route line + (if D6 needs it, but default NOT) any wiring. SERIAL SLOT. Static-before-`{param}` is automatic (fasthttp router). NOTHING else. RELEASE to bf-mcp-2 on close. |
| `internal/store/migrate.go` | ADD an additive `ensureColumn` row to `mcp_clients` ONLY IF D6 proves a live server-mode consumer. DEFAULT: NO change. |
| `.planning/parity/matrix/bifrost-mcp.md` | Flip rows per §7 (at close). |
| `.planning/parity/plans/open-questions.md` | Append ESC + deferred items (§7). |
| `docs/WORKFLOW.md` | Add the bf-mcp-1 row (at close). |

### FORBIDDEN (automatic REJECT if touched)

- `internal/mcp/{bridge,sse,probe,agent,launcher,oauth,filter,allowlist,defaults,
  registry,healthmonitor,discovery,toolpolicy,process,runner}.go` — CONSUME the
  shipped CLIENT-mode primitives; do NOT edit them (BIFROST-MAP §397 "Wave-7 MCP
  client-mode is consume-only").
- `internal/server/routes_openai.go` — bf-mcp-1 is NOT a routes_openai holder; the
  `/mcp` VK resolver is a NEW MCP-only helper, NOT an edit to the OpenAI gate.
- `internal/server/guard.go` — do NOT add `/mcp` to `LOCAL_ONLY_PATHS` (D2); consume
  the existing `/api/mcp/` entry as-is.
- The `mark3labs/mcp-go` dependency — NOT added (D1 VAR; g0router rolls its own bridge).
- Any per-VK tool scoping / `executeOnlyTools` / `AllowOnAllVirtualKeys` / VK↔MCP
  assignment — that is **bf-mcp-2** (§3).
- Any per-user OAuth/header credential / sessions table / `/api/mcp/sessions*` /
  `/api/oauth/per-user/*` — **ESC** (§3).
- Any `init()`, any new free global state, any `New(...)`/`Register*(...)` signature
  change (use the additive-setter pattern), any GORM hook, any destructive DDL.
- Any UI file (`ui/**`) — bf-mcp-1 is Go-only; `/mcp` is an API surface with no UI
  page (BIFROST-MAP §378 "new bifrost surfaces with no UI page … ship a Go
  integration test and need no UI touch").

---

## 3. Scope / Non-goals — explicit ESC list

**bf-mcp-1 builds ONLY the global un-scoped server-mode foundation + VK-resolution
plumbing + complete-oauth create flow.** The following are **ESC** (recorded in
`open-questions.md` at close):

| ESC item | Matrix row(s) | Why ESC |
|---|---|---|
| **Per-VK scoped MCP server** (lazy creation, `executeOnlyTools`, `AllowOnAllVirtualKeys`, VK↔MCP assignment table) | 004, 019, 020, 033 | **bf-mcp-2** (BIFROST-MAP §266). bf-mcp-1 is the global surface; VK is resolved but does NOT yet scope tools (D3/D4). |
| **Per-user OAuth + per-user header credential flows + sessions API** | 012-016, 027-031, 040-047, 058-059, 064-065 | The bifrost-mcp enterprise multi-tenant core (BIFROST-MAP §269). g0router has server-level OAuth, no per-user identity model. Large; deferred. |
| **`mark3labs/mcp-go` dependency** | 075, 076 | g0router rolls its own bridge (D1 VAR). The behavior (JSON-RPC handling) is built; the dependency is NOT. |
| **Code-mode VFS / plugin-pipeline-for-nested-tools / two-phase create-update rollback / discovered-tool key migration / retry-on-in-flight-reconnect / header redaction-merge / OAuth-rotation-disabled guard / sentinel error types / tool-pricing catalog / flexible-duration quirks** | 022, 023, 050, 051, 055, 056, 060-063, 068-070, 072-074 | Implementation-quirk + plugin-pipeline-coupled rows that presuppose Bifrost's handler architecture + the absent ref (BIFROST-MAP §270). Defer; cherry-pick only if matrix-evidenced + cheap. |
| **complete-oauth create-vs-UPDATE distinction** | 039 (update branch) | g0router has a single create flow; in-place OAuth re-rotation (051) is ESC. Create branch built (D7); update branch ESC. |
| **Full Bifrost `config_mcp_clients` column-set** (per-user/code-mode/pricing/config-hash/allow-on-all-vk columns) | 032 (enterprise columns) | Per-user/enterprise/code-mode surface (D6 / above ESC rows). g0router aligns ONLY the server-mode subset it consumes. |
| **Mandatory-VK enforcement + per-VK tool filtering for `/mcp`** | (implied by 052 in the per-VK context) | bf-mcp-1 resolves the VK (D4) but treats it as optional + non-scoping; mandatory + scoping is bf-mcp-2. |

No-leftovers (binding, §3 CLI_ORCHESTRATOR): every NEW surface bf-mcp-1 adds — the
`/mcp` routes, `server.go` dispatch, `resolveMCPVK` (live consumer =
**validation-and-rejection** of a provided-but-invalid VK, NOT a context attachment —
D4), the SSE heartbeat, the deferred finalizer (live payload = a real `recordAudit`
write stamping the VK — D8, else 054→PARTIAL), `CompleteInstanceAuth` (live caller of
the formerly-dead `Engine.Complete` — D7), and any (default-zero) new column — MUST
have a live, reachable consumer with a grep AND behavior proof (§5) or the plan STOPS
and escalates. No dead handler, no dead column, no dead method, no resolved-but-unread
value, no no-op finalizer.

---

## 4. Task graph (TDD; `N. [step] -> verify: [check]`)

Cadence (AGENTS.md "TDD always"): no impl file/field lands before its covering
`_test.go` is committed RED. `go test ./... && go vet ./... && go build ./...` green,
**HERMETIC** (no real network, no subprocess spawn, no real SSE timing) at EVERY
commit. Serial: bf-mcp-1 holds routes_admin.go (the one-line registration).

1. **[VK precedence resolver, RED→GREEN]** Write `resolveMCPVK` cases in
   `internal/admin/mcp_test.go` (x-g0-vk wins; Bearer fallback; x-api-key fallback;
   all-empty → ""). Implement the PURE `resolveMCPVK(HeaderGetter) string` in
   `admin/mcp.go` (D4). -> verify: `go test ./internal/admin/ -run MCPVK` green;
   `go vet ./... && go build ./...` exit 0. Commit RED then GREEN:
   `phase-1/bf-mcp-1: failing /mcp VK precedence resolver (TDD red)` →
   `phase-1/bf-mcp-1: x-g0-vk > Bearer > x-api-key resolver for /mcp`.
   (The resolver's VALIDATION consumer lands in step 3 — the resolved key is verified
   against the shipped VK resolver and a provided-but-invalid VK is rejected.)

2. **[server-mode dispatcher, RED→GREEN]** Write `internal/mcp/server_test.go`
   (initialize/tools/list/tools/call over canned input + fake executor; unknown
   method → JSON-RPC error; malformed → -32700). Implement `internal/mcp/server.go`
   reusing `splitFrames` + `NewBridgeToolExecutor` + `smartFilterText` (D1/D3). ->
   verify: `go test ./internal/mcp/ -run Server` green; gates exit 0. Commit RED then
   GREEN: `phase-1/bf-mcp-1: failing server-mode JSON-RPC dispatcher (TDD red)` →
   `phase-1/bf-mcp-1: MCP server-mode dispatcher (initialize/tools-list/tools-call; own bridge)`.

3. **[shared catalog + /mcp POST handler + VK validation, RED→GREEN]** Extend
   `mcp_test.go`: `MCPServerPost` round-trips a `tools/list` returning the SAME
   catalog `ListTools` serves (D3), in JSON-RPC (NOT `{data,error}`); **VK validation
   cases (D4): valid VK → allowed; provided-but-unknown/inactive VK → JSON-RPC error
   (rejected); absent VK → allowed.** Extract the shared catalog assembler; implement
   `MCPServerPost` (calling `resolveMCPVK` then validating any supplied VK via the
   shipped `vkResolverAdapter.ResolveVK`/`GetVirtualKeyByKey` + active check, rejecting
   invalid) + `SetMCPServer` additive setter on `handlers.go`. -> verify: `go test
   ./internal/admin/ -run MCPServer` green; grep proof: `ListTools` + `Server.tools/list`
   share one source AND a provided-but-invalid VK is rejected (the resolver has a live
   validation consumer, not a mere context attachment). Commit RED then GREEN.

4. **[SSE heartbeat + deferred finalizer w/ real audit payload, RED→GREEN]** Extend
   `mcp_test.go`: `MCPServerSSE` emits `: ping\n\n` on each INJECTED tick (N ticks → N
   frames, ZERO real elapsed time, D5); **the deferred finalizer runs a best-effort
   `recordAudit("mcp_server.tools_call", …)` stamping the resolved VK, and that audit
   write lands AFTER the injected stream sink closes — NOT during frame emission (D8,
   preferred path (a)).** Implement `MCPServerSSE` with the injected ticker/clock +
   `streamWriter` sink + the deferred `recordAudit` finalizer. -> verify: `go test
   ./internal/admin/ -run MCPSSE` green; the audit entry exists in the test store's
   audit table with the VK in `details` AND is written post-close (ordering asserted);
   hermetic grep proof (no `time.Sleep`/`net.Dial` in the test). If `recordAudit` is
   unreachable from the handler (NOT expected — the handler is a `*admin.Handlers`
   method), FALL BACK to the injected-finalizer-hook ordering test and flip 054
   PARTIAL (§7) — do NOT ship a no-op. Commit RED then GREEN:
   `phase-1/bf-mcp-1: failing SSE heartbeat + deferred-audit tests (TDD red)` →
   `phase-1/bf-mcp-1: hermetic SSE heartbeat (: ping/15s injected) + deferred tools/call audit`.

5. **[complete-oauth handler, RED→GREEN]** Extend `mcp_test.go`:
   `CompleteInstanceAuth` calls `Engine.Complete` and returns the masked account (NO
   token in the JSON body). Implement the handler (D7). -> verify: `go test
   ./internal/admin/ -run CompleteAuth` green; grep proof: `Engine.Complete` now has a
   live caller. Commit RED then GREEN:
   `phase-1/bf-mcp-1: complete-oauth handler (live Engine.Complete; masked account)`.

6. **[route registration, RED→GREEN — SERIAL SLOT]** Write
   `internal/server/routes_mcp_test.go` (routes registered; `POST /mcp` → JSON-RPC;
   `GET /mcp` heartbeat via injected tick; `/mcp` not LOCAL_ONLY; complete-oauth route
   present). Create `internal/server/routes_mcp.go` + add the one-line
   `RegisterMCPRoutes(r,h)` + the complete-oauth route to `routes_admin.go`. -> verify:
   `go test ./internal/server/... && go test ./... && go vet ./... && go build ./...`
   exit 0; grep proof: `/mcp` registered, not in LOCAL_ONLY. Commit RED then GREEN:
   `phase-1/bf-mcp-1: register POST/GET /mcp + complete-oauth route (serial)`.

7. **[close]** Full validation (§6); flip matrix rows (§7); append `open-questions.md`
   (ESC list §3 + D-deferred items); update `docs/WORKFLOW.md`; RELEASE the
   routes_admin.go serial slot to bf-mcp-2. -> verify: §6 all green; matrix + WORKFLOW
   + open-questions committed. Commit:
   `phase-1/bf-mcp-1: close — MCP server-mode foundation; matrix flip; serial release`.

---

## 5. Acceptance criteria (binary; file:line / grep proofs)

**Test gates** (each yes/no, exit 0; HERMETIC — no network, no subprocess, no real SSE
timing):
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/mcp/ -run Server -v` → dispatcher cases pass (initialize/
  tools-list/tools-call/unknown-method/malformed).
- `go test ./internal/admin/ -run 'MCPVK|MCPServer|MCPSSE|CompleteAuth' -v` → all pass.
- `go test ./internal/server/ -run MCP -v` → route + JSON-RPC-not-envelope +
  heartbeat-via-injected-tick pass.

**TDD-order proof** — each impl's covering test is in an earlier-or-equal commit:
```bash
for pair in \
  "internal/mcp/server_test.go:internal/mcp/server.go" \
  "internal/server/routes_mcp_test.go:internal/server/routes_mcp.go" ; do
  tf=${pair%%:*}; im=${pair##*:}
  ct=$(git log --format=%ct --diff-filter=A -1 -- "$tf")
  cf=$(git log --format=%ct --diff-filter=A -1 -- "$im")
  [ "$ct" -le "$cf" ] || echo "TDD VIOLATION: $im"   # prints nothing
done
```

**Grep / live-endpoint proofs:**
```bash
# /mcp is a LIVE JSON-RPC endpoint (POST + GET registered)
grep -nE 'POST\("/mcp"|GET\("/mcp"|RegisterMCPRoutes' internal/server/routes_mcp.go internal/server/routes_admin.go
# served methods present in the dispatcher
grep -nE '"initialize"|"tools/list"|"tools/call"' internal/mcp/server.go            # all three
# REUSE not duplicate: server.go does NOT redefine framing; it consumes shipped helpers
grep -nE 'splitFrames|NewBridgeToolExecutor|smartFilterText' internal/mcp/server.go ; echo "^ reuses shipped framing/dispatch"
! grep -nE 'func splitFrames|func parseSSEFrame' internal/mcp/server.go && echo "no duplicated framing OK"
# global tool surface = the SAME catalog ListTools serves (one source — D3)
grep -nE 'tools/list' internal/mcp/server.go ; grep -nE 'ListTools|catalog' internal/admin/mcp.go | head
# VK precedence resolved + CONSUMED-BY-VALIDATION (no-leftovers D4 — NOT a bare attach)
grep -nE 'func resolveMCPVK' internal/admin/mcp.go
grep -nE 'resolveMCPVK\(' internal/admin/mcp.go | grep -v 'func '                   # ≥1 live caller
# the resolved VK is VALIDATED via the shipped resolver and a provided-but-invalid VK is rejected
grep -nE 'ResolveVK|GetVirtualKeyByKey' internal/admin/mcp.go                       # the validation call in MCPServerPost/SSE
# behavior proof: a provided-but-unknown/inactive VK returns a JSON-RPC error (test name asserts reject)
grep -nE 'invalid.*vk|vk.*invalid|reject|inactive' internal/admin/mcp_test.go | grep -iE 'mcp' # the reject test exists
# SSE heartbeat literal + hermetic (injected ticker, no real sleep — D5)
grep -nE ': ping|mcpSSEHeartbeatInterval|15 \* time.Second' internal/admin/mcp.go internal/mcp/server.go
! grep -nE 'time\.Sleep' internal/admin/mcp_test.go internal/server/routes_mcp_test.go && echo "no real sleep in SSE tests OK"
# deferred finalizer has a REAL payload (recordAudit per tools/call) — NOT a no-op (D8)
grep -nE 'recordAudit\(ctx, "mcp_server.tools_call"' internal/admin/mcp.go          # the deferred audit write
# (if this grep is EMPTY at impl → fallback path: 054 flips PARTIAL, never HAVE on a no-op)
# complete-oauth makes the shipped-dead Engine.Complete LIVE (no-leftovers D7)
grep -nE 'func .*CompleteInstanceAuth' internal/admin/mcp.go
grep -rnE '\.Complete\(' internal/admin/ --include='*.go' | grep -v _test.go        # ≥1 live caller
# /mcp is NOT local-only (D2) ; complete-oauth IS on /api/mcp (session-gated)
! grep -nE '"/mcp"' internal/server/guard.go && echo "/mcp not local-only OK"
# JSON-RPC body is NOT the {data,error} admin envelope on /mcp
! grep -nE 'writeData|writeError' internal/mcp/server.go && echo "/mcp dispatch is raw JSON-RPC OK"
# additive only: no New() sig change, no init, no mcp-go dep, no destructive DDL
! grep -rnE 'func init\(' internal/mcp/server.go internal/server/routes_mcp.go && echo "no init() OK"
! grep -rnE 'mark3labs/mcp-go' go.mod internal/ && echo "no mcp-go dep OK"
git diff <base>..HEAD -- internal/store/migrate.go | grep -E '^\+' | grep -iE 'DROP|RENAME' | wc -l   # = 0
# no token leak on complete-oauth / accounts
! grep -nE 'AccessToken|RefreshToken|access_token"|refresh_token"' internal/admin/mcp.go | grep -iE 'json:"' && echo "no token field echoed OK"
```

**Behavioral acceptance (binary):**
- `POST /mcp` with a `tools/list` JSON-RPC request returns a JSON-RPC `{"jsonrpc":
  "2.0","id":N,"result":{"tools":[…]}}` whose tool set EQUALS the catalog
  `GET /api/mcp/tools` (`ListTools`) serves (the same source — D3), NOT the
  `{data,error}` envelope.
- `POST /mcp` with `tools/call` dispatches through the SHIPPED bridge executor and
  returns the `smartFilterText`-filtered result; an unknown method returns a JSON-RPC
  error; malformed JSON returns `-32700`.
- `GET /mcp` SSE stream emits `: ping\n\n` once per injected heartbeat tick, proven
  with ZERO real elapsed time (injected ticker/clock); the deferred finalizer writes a
  real `recordAudit("mcp_server.tools_call", …)` entry (stamping the resolved VK) only
  AFTER the stream sink closes — never during frame emission (D8). On the fallback
  path (recordAudit unreachable), 054 is PARTIAL, not HAVE.
- `resolveMCPVK` returns x-g0-vk when present, else the Bearer token, else x-api-key,
  else "" — and the resolved key is genuinely CONSUMED: a valid VK is admitted, a
  provided-but-unknown/inactive VK is REJECTED (JSON-RPC error on POST, connection
  reject on SSE), an absent VK is allowed (live validation consumer — D4).
- `POST /api/mcp/instances/{id}/auth/complete` calls `Engine.Complete` and returns
  the masked account (no `access_token`/`refresh_token`/`state`/`verifier` in the
  body); 404 on unknown instance.
- No dead handler/column/method: every §5 grep "live caller" proof is non-empty.

---

## 6. Validation commands

```bash
go test ./... && go vet ./... && go build ./...                          # exit 0 (binding)
go test ./internal/mcp/ -run Server -v
go test ./internal/admin/ -run 'MCPVK|MCPServer|MCPSSE|CompleteAuth' -v
go test ./internal/server/ -run MCP -v
```
No UI build / Playwright needed — bf-mcp-1 ships NO UI touch and NO mock correction
(`/mcp` + complete-oauth have no UI page; BIFROST-MAP §378). **Hermetic only (D5/D8):**
no test may open a socket, spawn a subprocess, or sleep for a real SSE interval — the
heartbeat ticker and the SSE sink are injected.

---

## 7. Freeze rules + matrix-flip + open-questions + WORKFLOW + no-leftovers

**Freeze rules (binding):**
- `internal/server/routes_admin.go` — bf-mcp-1 is the FIRST holder of the MCP serial
  chain (BIFROST-MAP §343-346: bf-mcp-1 → bf-mcp-2 → bf-core-2). It holds the slot for
  the one-line `RegisterMCPRoutes` + complete-oauth route additive edit and RELEASES
  to bf-mcp-2 on the close commit (step 7). NOT a routes_openai.go holder.
- Wave-7 MCP CLIENT-mode (`internal/mcp/{bridge,sse,probe,agent,launcher,oauth,…}.go`)
  is **consume-only** (BIFROST-MAP §397) — REUSE the framing/dispatch/SSE helpers; do
  NOT edit them. The ONLY new `internal/mcp` file is `server.go`.
- Migrations: additive `ensureColumn` ONLY, and DEFAULT zero (D6); no destructive DDL.
- No `mark3labs/mcp-go` dependency (D1 VAR). No `init()`, no new free global state, no
  `New(...)`/`Register*(...)` signature change (additive-setter pattern, like the
  shipped `SetMCPLauncher`/`SetMCPEngine`/`SetMCPProbe`).
- No reverse-engineering of the absent Bifrost ref (ESC-REF-ABSENT) — build to the
  matrix + the base JSON-RPC/SSE spec g0router's own client already speaks; STOP-
  escalate any undocumented Bifrost-specific framing detail (D1).

**Matrix-flip (at close, in `.planning/parity/matrix/bifrost-mcp.md`):**
- PAR-BF-MCP-002 → **HAVE** (server-mode `/mcp` JSON-RPC+SSE; global surface). Cite
  bf-mcp-1 + D1/D2.
- PAR-BF-MCP-003 → **HAVE** (global un-scoped catalog re-exposed; per-VK = bf-mcp-2).
  Cite D3.
- PAR-BF-MCP-032 → **PARTIAL** (existing `mcp_clients` + `*_enc` tables satisfy the
  server-mode/encryption subset; full enterprise column-set ESC). Cite D6.
- PAR-BF-MCP-039 → **PARTIAL** (complete-oauth create flow HAVE via live
  `Engine.Complete`; create-vs-update distinction ESC). Cite D7.
- PAR-BF-MCP-052 → **HAVE** (`x-g0-vk > Bearer > x-api-key`; resolved VK validated +
  invalid rejected; header names VAR). Cite D4.
- PAR-BF-MCP-053 → **HAVE** (`: ping`/15s; hermetic injected ticker). Cite D5.
- PAR-BF-MCP-054 → **HAVE** *if the deferred finalizer carries a real `recordAudit`
  payload (preferred path (a), expected — recordAudit verified reachable)*; **PARTIAL**
  on the fallback (deferral mechanism present, no payload wired) — NEVER HAVE on a
  no-op finalizer. Context-key name VAR. Cite D8.
- PAR-BF-MCP-075 → **HAVE-by-variant** (JSON-RPC message handling over g0router's own
  bridge; `mcp-go` dependency NOT added — VAR). Cite D1.

**`open-questions.md` (append at close):**
```
## bf-mcp-1 — MCP server-mode foundation — 2026-06-15
- [ ] Per-VK tool scoping for /mcp — ESC; bf-mcp-2 (rows 004/019/020/033). bf-mcp-1 resolves the VK but does NOT scope tools. Why: global un-scoped surface is the foundation; scoping is the next plan.
- [ ] Mandatory-VK enforcement for /mcp (reject when NO VK is supplied) — bf-mcp-1 validates a PROVIDED VK and rejects an invalid/inactive one (D4), but an ABSENT VK is allowed (optional global surface); mandatory-when-absent + per-VK scoping is bf-mcp-2. Why: avoid breaking the global surface before per-VK lands.
- [ ] /mcp local-only vs public — bf-mcp-1 ships /mcp as VK-gated public (NOT in LOCAL_ONLY_PATHS, D2). Operator decision if some deployments need /mcp local-only. Why: it is the public MCP-server surface per matrix row 052.
- [ ] MCP server-mode numeric usage/cost shape (row 054) — bf-mcp-1's deferred finalizer writes a real best-effort `recordAudit` per /mcp tools/call (D8, the parity-meaningful deferred payload), but the exact NUMERIC usage/cost recorded for a /mcp tools/call is undocumented in the matrix (ESC-REF-ABSENT) and NOT invented. Why: the audit entry proves the deferral works over a real write; a usage-cost shape needs the ref or a g0router decision.
- [ ] complete-oauth create-vs-update distinction (row 039) — ESC; g0router has a single create flow (OAuth re-rotation 051 ESC). Why: update-flow presupposes the enterprise update path.
- [ ] config_mcp_clients enterprise columns (row 032: per_user/code_mode/tool_pricing/config_hash/allow_on_all_vk) — ESC; g0router aligns only the server-mode subset (D6). Why: per-user/enterprise/code-mode surface.
- [ ] mark3labs/mcp-go dependency (rows 075/076) — VAR; g0router rolls its own bridge (D1). Why: adding the dep would duplicate shipped framing.
- [ ] Per-user OAuth/header credential + sessions API (~22 rows) — ESC; the bifrost-mcp enterprise core (BIFROST-MAP §269).
- [ ] ESC-REF-ABSENT: the exact Bifrost initialize-capability object, JSON-RPC error-code numbering, and any non-standard framing are unverifiable; bf-mcp-1 built to the base MCP/JSON-RPC 2.0 spec g0router's own client already speaks (D1). Restore the ref to confirm wire fidelity.
```

**`docs/WORKFLOW.md` (update at close):** add a bf-mcp-1 row — MCP server-mode
foundation shipped (Go-only; NEW `POST/GET /mcp` JSON-RPC+SSE over `internal/mcp/
server.go` reusing the shipped CLIENT bridge/SSE/dispatch; VK precedence resolver;
hermetic `: ping`/15s heartbeat; deferred SSE trace completion; complete-oauth wires
the formerly-dead `Engine.Complete`); rows 002/003/052/053/054/075 → HAVE(-variant),
032/039 → PARTIAL; ESC items recorded in open-questions; routes_admin.go serial slot
released to bf-mcp-2; ESC-REF-ABSENT honored (built to matrix + base spec only).

**No-leftovers confirmation (binding):** bf-mcp-1 adds `POST/GET /mcp` (consumed by a
real JSON-RPC client; `tools/list` returns the real catalog, `tools/call` dispatches
through the real shipped bridge — no dead handler), `server.go` (consumed by the
`/mcp` handlers), `resolveMCPVK` (consumed by **validation-and-rejection** — a
provided-but-invalid VK is rejected, not merely attached to context; STOP if the
invalid-VK path has no live rejecting effect — D4), the SSE heartbeat (emits real
frames on the injected tick — STOP if unwired), the deferred finalizer (carries a
real `recordAudit` payload stamping the VK — if that payload is not wired, 054 flips
PARTIAL, never HAVE on a no-op — D8), `CompleteInstanceAuth` (calls the formerly-dead
`Engine.Complete` — STOP if it does not — D7), and (default-zero) any new column ONLY
if a server-mode path consumes it (STOP otherwise). Each new surface has a grep- AND
behavior-proven live consumer (§5) or the plan STOPS and escalates — closing both the
resolved-but-unread (Trap 1) and no-op-finalizer (Trap 2) inertness traps.
```
