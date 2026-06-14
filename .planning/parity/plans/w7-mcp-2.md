# Micro-plan w7-mcp-2 — MCP client (SSE/message transport + probe + registry + OAuth engine) (Go)

```
wave: 7
plan: w7-mcp-2
status: READY (rev 1 — authored against the SHIPPED w7-mcp-1 foundation
  (store + launcher + bridge + filter + allowlist + defaults — LIVE in-tree @
  internal/mcp/{runner,bridge,launcher,filter,allowlist,defaults,process}.go +
  internal/store/{mcpinstances,mcpoauth}.go) and the SHIPPED w7-plat-2/w7-mcp-1
  injectable-runner pattern, here REUSED for NETWORK I/O: every probe/registry/OAuth
  HTTP call goes through an INJECTABLE http transport (or http.RoundTripper) so unit
  tests feed canned responses with NO real network — exactly as the tunnel Runner and
  the MCP ProcessRunner gate external effects behind a fake. The in-tree
  internal/auth/oauth.go is the PKCE engine of record (NewOAuthFlow injects a nil-able
  *http.Client @ oauth.go:128-136; pkceChallenge @ oauth.go:274; randomURLSafe @
  oauth.go:266) and is REUSED — not edited. live tree @ <base>; WAVE-7-MAP w7-mcp-2 row
  ~line 181; MCP track split §207-208; serial chain §219-224 (mcp-2 holds NO
  routes_admin slot — w7-mcp-3 is the only MCP route holder); reconciliation §245;
  freeze rules §267.)
runs: MCP track. EXTENDS the greenfield internal/mcp package with NEW files —
  disjoint from every other domain/store/admin file; runs ∥ governance + providers +
  platform tracks. INTERNALLY SERIAL: w7-mcp-1 (SHIPPED — foundation) ──▶ w7-mcp-2
  (THIS — client/probe/registry/OAuth engine) ──▶ w7-mcp-3 (admin transport + routes +
  tools). DEPENDS on w7-mcp-1's exported constructors (NewLauncher/SetRunner, the
  Bridge + SessionSink seam, smartFilterText, isAllowedCommand, the *MCP* store
  methods, DefaultPlugins). mcp-3 DEPENDS on this plan's probe/registry/OAuth engine
  + the SSE-message transport seam. THIS plan takes NO serial slot (no routes_admin
  edit; the SSE/message + admin routes are w7-mcp-3 — MAP §181/§208).
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w7-mcp-2:
ref-source: 9router frozen @ 827e5c3 — the probe handshake + registry client.
  Authoritative ref files (read for THIS plan):
    src/app/api/cli-tools/cowork-mcp-tools/route.js (TIMEOUT_MS=8000 @5; headers incl
      MCP-Protocol-Version "2025-06-18" @10-14; initialize id:1 @19-27; 401/403 →
      {requiresAuth:true} @28-30; mcp-session-id read @34; notifications/initialized
      @41-46; tools/list id:2 @49-54; SSE parse "data:" lines find id===2 @60-69;
      JSON fallback @71; AbortError → "timeout" @78);
    src/app/api/cli-tools/cowork-mcp-registry/route.js (REGISTRY_URL @5;
      VISIBILITY="commercial,gsuite,gsuite-google" @6; CACHE_TTL_MS=3600000 @7;
      isDirectConnect rejects mcp.claude.com + api.anthropic.com/mcp + {< } @16-22;
      pagination loop ≤20 pages limit=500 + cursor @24-55; requiredFields skip @37;
      remote.type==="sse"?"sse":"http" @38; oauth=!meta.isAuthless @47; URL dedupe via
      Set @56-58; nextCursor @53);
    src/app/api/mcp/[plugin]/sse/route.js (endpoint event w/ sessionId @22;
      text/event-stream + no-cache + keep-alive + X-Accel-Buffering headers @30-35;
      registerSession/unregisterSession @20,25);
    src/app/api/mcp/[plugin]/message/route.js (sendToChild + 202 @16-17).
  NO 9router MCP-OAuth engine exists in the frozen ref (Cowork disabled): the MCP
  OAuth account engine (PAR-MCP-037) has NO direct probe — it is authored against the
  MCP authorization spec (RFC 9728 protected-resource-metadata + RFC 8414
  authorization-server-metadata + RFC 7636 PKCE) REUSING the in-tree
  internal/auth/oauth.go PKCE engine. This is ESC-MCP-OAUTH-PROTOCOL (§8) — recommended
  default recorded, NEVER fabricated; escalate any ambiguous discovery detail.
base: <base> = git rev-parse HEAD recorded at P0. Substitute the actual SHA
  everywhere §5 says <base>. (At authoring, HEAD = ca15297…; recompute at P0.)
go-serial-slot: NONE. This plan does NOT edit internal/server/routes_admin.go. The
  SSE/message routes (PAR-MCP-001/002/055/056) and all /api/mcp/* admin routes land in
  w7-mcp-3 (MAP §181/§208). No selection.go / factory.go micro-serial either — MCP is
  a standalone gateway subsystem, not an inference-path concern.
new-route: NONE. NO UI route files, NO admin handler, NO /api/mcp/* registration.
  Client/engine library only — consumed by w7-mcp-3's handlers.
```

---

## 1. Scope — PAR rows + the four subsystems

### Rows this plan closes

| Row / item | Claim (from `9router-mcp.md`) | Target state after w7-mcp-2 |
|---|---|---|
| PAR-MCP-009 | Probe performs `initialize` + `notifications/initialized` + `tools/list` handshake (`cowork-mcp-tools/route.js:9`) | HAVE (probe state machine; FULLY unit-tested over an INJECTED fake http transport returning canned initialize + tools/list) |
| PAR-MCP-010 | Probe sends `MCP-Protocol-Version: 2025-06-18` header (`…:13`) | HAVE (constant `mcpProtocolVersion = "2025-06-18"`; header asserted in the captured request via the fake transport) |
| PAR-MCP-011 | Probe reads `mcp-session-id` from `initialize` + replays on `tools/list` (`…:34`) | HAVE (session-id captured + replayed; unit-tested via a two-response fake transport asserting the replayed header) |
| PAR-MCP-012 | Probe parses SSE responses for `tools/list` (`…:60`) | HAVE (PURE `parseSSEDataFrames` — split `data:` lines, find `id==2` result; unit-tested on canned SSE text) |
| PAR-MCP-013 | Probe returns `requiresAuth:true` on 401/403 (`…:28`) | HAVE (401/403 from the fake transport → `RequiresAuth=true`; unit-tested) |
| PAR-MCP-014 | Registry client fetches with pagination (`cowork-mcp-registry/route.js:24`) | HAVE (≤20 pages, limit=500, cursor follow; unit-tested via a multi-page fake transport returning canned cursors) |
| PAR-MCP-015 | Registry cache TTL = 1h (`…:7`) | HAVE (`registryCacheTTL = 1*time.Hour`; in-memory cache hit/miss/expire unit-tested via an INJECTED clock — no real sleep) |
| PAR-MCP-016 | Registry filters claude.com + api.anthropic.com/mcp (`…:16`) | HAVE (PURE `isDirectConnect`; rejects `mcp.claude.com`, `api.anthropic.com/mcp`, `{`/`<`; unit-tested incl. every reject) |
| PAR-MCP-017 | Registry excludes tenant-required entries (`…:37`) | HAVE (skip entries with `meta.requiredFields.length>0`; unit-tested) |
| PAR-MCP-037 | No MCP OAuth account engine (`internal/mcp/`) | HAVE (`internal/mcp/oauth.go`: PKCE start/exchange/refresh REUSING `internal/auth/oauth.go`; protected-resource-metadata + auth-server-metadata discovery; token storage/refresh over the SHIPPED `mcpoauth` store, tokens `*_enc`; FULLY unit-tested via the fake transport) |
| PAR-MCP-038 | No MCP health monitor (`internal/mcp/`) | HAVE (`internal/mcp/healthmonitor.go`: per-account health check + token-expiry/refresh-needed status; PURE status derivation unit-tested; the periodic ping loop is integration-only) |
| PAR-MCP-039 | No MCP discovery / compact injection (`internal/mcp/`) | HAVE (`internal/mcp/discovery.go`: `tools/list` result caching keyed by instance + compact manifest assembly; PURE manifest build unit-tested) |
| PAR-MCP-055 | SSE handshake sends `endpoint` event w/ sessionId (`sse/route.js:22`) | HAVE (the CLIENT side of the bridge transport: `internal/mcp/sse.go` connects to a remote `/sse` endpoint, reads the `endpoint` event, POSTs to `/message`; the SERVER `endpoint`-event emission is w7-mcp-3 — this plan models the client transport + the frame parse) |
| PAR-MCP-056 | SSE response headers no-cache/keep-alive/X-Accel-Buffering (`sse/route.js:30`) | HAVE (the client SSE reader honors `text/event-stream`; the server header emission is w7-mcp-3) |
| PAR-MCP-057 | Registry dedupes by URL (`…:57`) | HAVE (PURE URL-dedupe after pagination; unit-tested with duplicate-URL pages) |
| PAR-MCP-058 | Probe timeout = 8s (`…:5`) | HAVE (`probeTimeout = 8*time.Second` via `context.WithTimeout`; unit-tested via a short-timeout context + a blocking fake transport — NO real 8s sleep) |
| PAR-MCP-059 | Probe AbortError → `"timeout"` (`…:78`) | HAVE (`context.DeadlineExceeded` → `Error:"timeout"`; unit-tested via a canceled context) |
| PAR-MCP-001 (client half) | SSE endpoint exposes stdio plugins over HTTP (`mcp/[plugin]/sse/route.js:6`) | PARTIAL→ (the CLIENT SSE/message HTTP transport lands here; the SERVER `/api/mcp/{plugin}/sse` route + `endpoint`-event emission is w7-mcp-3) |
| PAR-MCP-002 (client half) | Message POST forwards JSON-RPC to child (`mcp/[plugin]/message/route.js:7`) | PARTIAL→ (the CLIENT POST-to-`/message` transport lands here; the SERVER route is w7-mcp-3) |

Matrix flips at closeout (§4 T-close): in `.planning/parity/matrix/9router-mcp.md`,
PAR-MCP-009,010,011,012,013,014,015,016,017,037,038,039,057,058,059 → HAVE (real Go;
all HTTP I/O hermetically unit-tested via an injected transport; the periodic
health-ping loop is integration-only — §1.9). PAR-MCP-001,002,055,056 stay
MISSING/PARTIAL with a footnote "client SSE/message transport shipped w7-mcp-2; the
server `/api/mcp/{plugin}/{sse,message}` routes + handshake-event emission are
w7-mcp-3". Append new open items (§8) to `open-questions.md`.

> **Row note (binding):** PAR-MCP-018/019 (Cowork-settings `managedMcpServers` /
> `toolPolicy`) named loosely under the MCP track are **NOT** in this plan — the
> matrix maps them to Cowork-settings (`cowork-settings/route.js`), assigned to
> w7-mcp-3. This plan closes exactly the 19 rows above
> (001/002/009-017/037/038/039/055-059).

### 1.1 Preconditions already satisfied by the SHIPPED w7-mcp-1 (evidence — cite file:line)

- **The launcher/bridge transport seam is LIVE (consume — do NOT edit).**
  - `internal/mcp/bridge.go:12` `type SessionSink func(frame []byte) error` — the
    per-session frame sink; `bridge.go:37 AddSession`, `:44 RemoveSession`,
    `:122 Send` (write to child stdin), `:127 IsRunning`. The SSE/message CLIENT
    transport (§1.4) produces frames a `SessionSink` consumes and consumes frames a
    remote server emits.
  - `internal/mcp/bridge.go:141 splitFrames` — the PURE newline-delimited JSON-RPC
    frame splitter. The SSE client REUSES the same framing philosophy; the SSE
    `data:`-line parse (§1.5) is the analogous PURE function for the SSE wire format.
  - `internal/mcp/launcher.go:31 NewLauncher`, `:41 SetRunner`, `:90 StartHTTP`,
    `:96 StartSSE`, `:103 recordInstance` — the HTTP/SSE mode seam. THIS plan
    supplies the live client the launcher's HTTP/SSE modes deferred to mcp-2 (the
    `recordInstance` placeholder @ `launcher.go:103-108` is where mcp-2/3 wire the
    real transport). **NO edit to launcher.go in THIS plan** — the client is a NEW
    file consumed by mcp-3's handlers (the launcher wiring is mcp-3).
- **The OAuth store is LIVE + complete for accounts + flows (consume; extend additively only).**
  - `internal/store/mcpoauth.go:44 UpsertMCPOAuthAccount` (tokens `*_enc`),
    `:92 GetMCPOAuthAccount`, `:99 GetMCPOAuthAccountByInstance`,
    `:106 ListMCPOAuthAccounts`, `:130 DeleteMCPOAuthAccount`,
    `:139 CreateMCPOAuthFlow` (verifier `*_enc`), `:158 ConsumeMCPOAuthFlow`
    (return+delete+expire→`ErrNotFound`). **The OAuth engine (§1.6) consumes these
    verbatim.** The account `Status` field (`mcpoauth.go:23`: `connected|expired|
    error`) is the health-monitor's write target. If the engine needs a
    status-only update or a list-by-status, ADD an additive method
    (`SetMCPOAuthAccountStatus`, `ListMCPOAuthAccountsByStatus`) — §3 / ESC-STORE-ADD.
- **The instances store is LIVE (consume).** `internal/store/mcpinstances.go:122
  CreateMCPInstance`, `:157 GetMCPInstance`, `:164 ListMCPInstances`,
  `:188 UpdateMCPInstance`, `:210 SetMCPInstanceStatus`. The probe/discovery bind
  results to an instance via these (instance bookkeeping is mcp-3's handler concern;
  this plan's engine accepts an instance/url and returns results — it does not own
  instance CRUD wiring).
- **The default plugin definitions are LIVE (consume).** `internal/mcp/defaults.go:20
  DefaultPlugins()` returns Exa (`http`, no-oauth), Tavily (`http`, oauth),
  browsermcp (`stdio`). The registry client (§1.5) returns the SAME
  `PluginDefinition`-shaped marketplace entries (or a registry-specific DTO that
  mcp-3 maps to a `PluginDefinition` — DECIDE at T-registry, §8 ESC-REG-DTO; default:
  a NEW `RegistryServer` DTO mirroring the 9router registry fields, NOT a reuse of
  `PluginDefinition`).
- **The in-tree PKCE OAuth engine is LIVE — REUSE, do NOT edit (`internal/auth/oauth.go`).**
  - `oauth.go:128 NewOAuthFlow(cfg OAuthConfig, st *store.Store, client *http.Client)`
    — **the http client is INJECTABLE (nil → a default 30s client @ :129-134).** This
    is the EXACT injectable-HTTP precedent the MCP client reuses for hermetic tests.
  - `oauth.go:274 pkceChallenge(verifier)` (S256), `:266 randomURLSafe(n)` (state +
    verifier generation), `:151 StartWithRedirect` (builds the authorize URL with
    `code_challenge`/`code_challenge_method=S256`), `:198 ExchangeWithRedirect`
    (code→token), `:221 Refresh` (refresh→token), `:232 requestToken` (PostForm +
    parse). The MCP OAuth engine (§1.6) REUSES these primitives — it does NOT
    re-implement PKCE. **If a primitive needs generalizing (e.g. the authorize URL
    builder must accept a discovered endpoint instead of `cfg.AuthorizeURL`), prefer
    an ADDITIVE helper in `internal/mcp/oauth.go` that calls the existing primitives;
    escalate (ESC-PKCE-GENERALIZE, §8) BEFORE changing any `internal/auth/oauth.go`
    signature.**
- **Secret-at-rest precedent (`*_enc`) — already used by `mcpoauth.go`.** Tokens +
  verifier are encrypted at rest via `s.cipher.Encrypt/Decrypt` (`mcpoauth.go:45-52,
  140-143, 180-182, 197-203`). The OAuth engine NEVER returns the cleartext token in
  any DTO a handler echoes (the masked-read discipline — w7-mcp-1 §1.3 secret note).
- **Store helpers to reuse.** `internal/store/store.go:14 ErrNotFound`,
  `:71 newID`; `internal/store/providers.go:121 requireRowAffected`. `s.cipher.*` for
  any additive `*_enc` (none expected — tokens already enc'd by mcp-1).
- **Test harness precedent (hermetic).** The mcp-1 tests inject a fake
  `ProcessRunner` (`internal/mcp/launcher_test.go`, `bridge_test.go`); the tunnel
  tests inject a fake `Runner` (`internal/platform/tunnel/service_test.go:14-45`).
  The store tests use a temp `store.Open` (`mcpoauth_test.go`,
  `service_test.go:47-60`). THIS plan adds the analogous **fake `http.RoundTripper`**
  (canned-response transport) + an **injectable clock** for the registry-cache TTL +
  a **short-timeout context** for the probe — so NO real network, NO real 8s sleep
  (§1.4 / §4).

### 1.2 No UI / mock contract binds this plan (binding — confirm)

The MCP client/engine has **no HTTP surface of its own**: the SSE/message SERVER
routes (PAR-MCP-001/002/055/056) and all `/api/mcp/*` admin routes are w7-mcp-3; the
w6-l MCP UI mocks (`ui/e2e/mocks/handlers/mcp.ts` + `skills.ts`) are reconciled by
**w7-mcp-3** (MAP §182). Therefore w7-mcp-2:
- adds **NO** `internal/admin/*` handler, **NO** `internal/server/routes_admin.go`
  edit, **NO** `ui/**` touch (src, mocks, seeds, specs), **NO** e2e.
- exposes ONLY exported Go constructors/functions consumed by w7-mcp-3 (the probe, the
  registry client, the OAuth engine, the SSE/message client transport, the health
  monitor, the discovery cache). The acceptance is Go-test-only (no playwright — §5).

### 1.3 The MCP protocol constants (binding — ported verbatim from the ref)

```go
const (
    mcpProtocolVersion = "2025-06-18"        // cowork-mcp-tools/route.js:13 (PAR-MCP-010)
    probeTimeout       = 8 * time.Second     // cowork-mcp-tools/route.js:5  (PAR-MCP-058)
    registryURL        = "https://api.anthropic.com/mcp-registry/v0/servers" // registry:5
    registryVisibility = "commercial,gsuite,gsuite-google"                   // registry:6
    registryPageLimit  = 500                  // registry:28
    registryMaxPages   = 20                   // registry:27 (PAR-MCP-014)
    registryCacheTTL   = 1 * time.Hour        // registry:7  (PAR-MCP-015)
)
```
JSON-RPC ids: `initialize` = id 1; `tools/list` = id 2 (the SSE parse keys on
`id==2` — `cowork-mcp-tools/route.js:67`). Probe request headers (every request):
`Content-Type: application/json`, `Accept: application/json, text/event-stream`,
`MCP-Protocol-Version: 2025-06-18` (`route.js:10-14`).

### 1.4 THE CENTRAL DESIGN PROBLEM — injectable HTTP transport (binding; COPY the SHIPPED Runner/SetRunner + the in-tree NewOAuthFlow nil-able-client philosophy)

The probe, registry, and OAuth engine all perform **network HTTP calls that CANNOT be
made in unit tests** (AGENTS.md "No mocks; use interfaces and fakes; test real
behavior"; the whole MCP track is hermetic — w7-mcp-1 §1.9). The probe state machine,
the registry pagination+cache+dedupe, and the OAuth PKCE+discovery+refresh+health MUST
be unit-testable deterministically with NO real network, NO real timeout sleep, NO
real clock. The mechanism is COPIED from two SHIPPED precedents:

1. **The tunnel/ProcessRunner injection seam** (a struct holds the effect-doer as a
   FIELD with a real default + a `Set…` override for tests):
   `internal/platform/tunnel/service.go:32,44` + `internal/mcp/launcher.go:31,41`.
2. **The in-tree `NewOAuthFlow` nil-able `*http.Client`** (`internal/auth/oauth.go:128`):
   the http client is a constructor param; `nil` → a real default. THIS is the exact
   shape every MCP client constructor uses.

**The injection point (binding decision): an `*http.Client` field constructed at the
component constructor, nil-able to a real default — REUSING `internal/auth/oauth.go`'s
pattern verbatim.** Each network component (`Probe`, `Registry`, OAuth `Engine`) holds
its own `client *http.Client`; the constructor signature is
`New…(…, client *http.Client)` with `nil → defaultHTTPClient()`. Tests pass
`&http.Client{Transport: <fakeRoundTripper>}`. The fake `http.RoundTripper` is the
single canned-response seam:

```go
// internal/mcp/transport_test.go (TEST ONLY — never in non-test code)
// fakeTransport returns canned *http.Response values keyed by request, captures the
// requests it receives (so tests assert headers/body/method), and can return an
// error or block (for the timeout test). Implements http.RoundTripper.
type fakeTransport struct {
    responses []fakeResp            // consumed in order, or matched by URL+method
    captured  []*http.Request       // every request seen (header/body assertions)
    block     bool                  // when true, blocks until ctx is canceled (timeout test)
}
func (f *fakeTransport) RoundTrip(r *http.Request) (*http.Response, error) { /* canned */ }
```

**Why `http.RoundTripper`, not a custom interface (binding — ESC-HTTP-SEAM resolved):**
an injected `*http.Client{Transport: rt}` (a) matches `internal/auth/oauth.go`'s
existing `*http.Client` field exactly (the PKCE engine is reused; its client is already
`*http.Client`), (b) requires NO new interface in non-test code, (c) lets the fake
capture the real `*http.Request` so the tests assert the MCP-Protocol-Version header,
the replayed `mcp-session-id`, the registry cursor query, and the PKCE `code_verifier`
form field — the highest-value assertions. The probe's **8s timeout** is a
`context.WithTimeout` on the request (`http.NewRequestWithContext`), tested via a
**short-timeout context + a blocking fake transport** (NO real 8s sleep — §4
T-probe). The registry's **1h cache TTL** is gated by an **injectable clock**
(`now func() time.Time` field, real `time.Now` default, a fake in tests — NO real
sleep — §4 T-registry).

```go
// internal/mcp/probe.go
type Probe struct {
    client *http.Client    // nil → defaultHTTPClient() (mirrors auth/oauth.go:128)
}
func NewProbe(client *http.Client) *Probe { /* nil → default */ }
// Run probes url: initialize → notifications/initialized → tools/list, honoring the
// 8s timeout via ctx. Returns ProbeResult{Tools, RequiresAuth, Error}.
func (p *Probe) Run(ctx context.Context, url string) ProbeResult

// internal/mcp/registry.go
type Registry struct {
    client *http.Client
    now    func() time.Time   // injectable clock for the 1h cache (test override)
    mu     sync.Mutex
    cache  *registryCacheEntry // {servers, fetchedAt}
}
func NewRegistry(client *http.Client) *Registry { /* nil → default; now = time.Now */ }
func (r *Registry) List(ctx context.Context, force bool) ([]RegistryServer, error)

// internal/mcp/oauth.go
type Engine struct {
    store  *store.Store
    client *http.Client
}
func NewEngine(st *store.Store, client *http.Client) *Engine { /* nil → default */ }
```

### 1.5 The four subsystems (binding contracts)

**(A) SSE/message HTTP client transport (`internal/mcp/sse.go`, NEW) — PAR-MCP-001/002/055/056 (client half).**
The CLIENT side of the bridge transport: connect to a remote MCP server over SSE,
read the `endpoint` event (`event: endpoint\ndata: <messageURL>` —
`sse/route.js:22`), then POST JSON-RPC messages to that message URL. The PURE,
unit-tested core is the SSE frame parser (the analogue of `splitFrames`):
```go
// parseSSEFrame parses one SSE event block ("event: X\ndata: Y\n\n") into its event
// name + data payload. PURE — no I/O. Mirrors the client side of sse/route.js:22.
func parseSSEFrame(block []byte) (event, data string)
// parseSSEDataFrames extracts every "data:" JSON payload from an SSE text body
// (PAR-MCP-012; mirrors cowork-mcp-tools/route.js:60-69 — split on "\n", keep
// lines starting "data:", strip the prefix). PURE.
func parseSSEDataFrames(body string) []string
```
The live SSE reader (streaming the response body, dispatching frames to a
`SessionSink` from the SHIPPED `bridge.go:12`) is the thin integration-only surface;
the FRAME PARSE is the unit-tested core. The message-POST client (`POST <messageURL>`
with a JSON-RPC body, expect 202 — `message/route.js:17`) is unit-tested via the fake
transport asserting the method/URL/body. **NO server route here** — the server
`/api/mcp/{plugin}/sse` + `endpoint`-event emission + the no-cache/keep-alive/
X-Accel-Buffering headers (PAR-MCP-056) are w7-mcp-3.

**(B) Probe — the MCP handshake (`internal/mcp/probe.go`, NEW) — PAR-MCP-009/010/011/012/013/058/059.**
The three-step handshake (`cowork-mcp-tools/route.js`), as a state machine over the
injected client:
1. **initialize** (id 1) POST with `protocolVersion`, `capabilities:{}`,
   `clientInfo:{name:"g0router",version:"1"}` (`route.js:22-25`). On **401/403** →
   `ProbeResult{RequiresAuth:true, Tools:nil}` (PAR-MCP-013, `route.js:28`). On other
   non-2xx → `ProbeResult{Error:"init <status>"}` (`route.js:31-33`). Read the
   `mcp-session-id` response header (`route.js:34`).
2. **notifications/initialized** POST (no id; best-effort — errors swallowed)
   replaying the session-id header if present (`route.js:41-46`).
3. **tools/list** (id 2) POST replaying the session-id. On 401/403 →
   `RequiresAuth:true` (`route.js:55`). Parse the response: if
   `Content-Type: text/event-stream` → `parseSSEDataFrames` + find the JSON with
   `id==2 && result` (PAR-MCP-012, `route.js:60-69`); else `json.Unmarshal`
   (`route.js:71`). Extract `result.tools[] → {name, description}` (`route.js:73-76`).
Timeout: an 8s `context.WithTimeout` wraps all three requests (PAR-MCP-058,
`route.js:5,16`); `context.DeadlineExceeded` → `ProbeResult{Error:"timeout"}`
(PAR-MCP-059, `route.js:78`).
```go
type ProbeResult struct {
    Tools        []ProbeTool // {Name, Description}
    RequiresAuth bool        // 401/403 detected (PAR-MCP-013)
    Error        string      // "timeout" | "init <status>" | <msg> ("" on success)
}
type ProbeTool struct{ Name, Description string }
```

**(C) Registry client (`internal/mcp/registry.go`, NEW) — PAR-MCP-014/015/016/017/057.**
Anthropic mcp-registry pagination + 1h cache + direct-connect filter + URL dedupe
(`cowork-mcp-registry/route.js`):
- **pagination** (`route.js:24-55`): up to `registryMaxPages` (20) requests to
  `registryURL?limit=500&visibility=…[&cursor=…]`; follow `metadata.nextCursor`; stop
  when absent or a page errors. PURE per-page mapper extracts each
  `item.server` + `item._meta["com.anthropic.api/mcp-registry"]` into a
  `RegistryServer` (`route.js:32-51`).
- **filter** (`route.js:36-37`): drop entries whose first `remote.url` fails
  `isDirectConnect` (PURE — rejects `mcp.claude.com`, `api.anthropic.com/mcp`, URLs
  with `{`/`<`, non-`https`; `route.js:16-22`, PAR-MCP-016) OR whose
  `meta.requiredFields.length>0` (tenant-required, PAR-MCP-017, `route.js:37`).
- **transport** (`route.js:38`): `remote.type=="sse" ? "sse" : "http"`;
  `oauth = !meta.isAuthless` (`route.js:47`).
- **dedupe** (`route.js:56-58`): after pagination, keep first occurrence per URL
  (PAR-MCP-057).
- **cache** (`route.js:7,64-73`): an in-memory `{servers, fetchedAt}`; on `List(ctx,
  force=false)` within `registryCacheTTL` of `r.now()` → return cached; else refetch
  + store. `force=true` bypasses (the `?refresh=1` analogue, `route.js:63`).
```go
type RegistryServer struct {
    Name, Slug, Title, Description, URL, Transport string
    OAuth     bool
    ToolNames []string
    ToolCount int
    IconURL   string
}
func isDirectConnect(url string) bool // PURE — the §16-22 reject set
```

**(D) OAuth account engine (`internal/mcp/oauth.go`, NEW) — PAR-MCP-037.**
PKCE authorization-code flow for MCP servers that returned `RequiresAuth` from the
probe, REUSING `internal/auth/oauth.go` primitives + the SHIPPED `mcpoauth` store. The
flow (per the MCP authorization spec — ESC-MCP-OAUTH-PROTOCOL §8):
1. **discovery** — fetch the server's protected-resource-metadata
   (`<resource>/.well-known/oauth-protected-resource`, RFC 9728) → the
   `authorization_servers[]`; fetch each authorization-server-metadata
   (`<as>/.well-known/oauth-authorization-server`, RFC 8414) → `authorization_endpoint`
   + `token_endpoint`. PURE JSON parsers (`parseProtectedResourceMetadata`,
   `parseAuthServerMetadata`) unit-tested on canned JSON; the fetch goes through the
   injected client.
2. **start** — `randomURLSafe(32)` state + `randomURLSafe(64)` verifier
   (`auth/oauth.go:155-162` primitives), `pkceChallenge(verifier)`
   (`auth/oauth.go:274`), persist via `store.CreateMCPOAuthFlow` (verifier `*_enc` —
   `mcpoauth.go:139`), build the authorize URL against the DISCOVERED
   `authorization_endpoint` (the additive helper — NOT `cfg.AuthorizeURL`;
   ESC-PKCE-GENERALIZE §8). Return `{authURL, state}`.
3. **complete** — `store.ConsumeMCPOAuthFlow(state)` (`mcpoauth.go:158`; expired →
   `ErrNotFound`), POST `grant_type=authorization_code` + `code_verifier=sess.Verifier`
   to the discovered `token_endpoint` (reuse the `requestToken` parse shape —
   `auth/oauth.go:232-264`), `store.UpsertMCPOAuthAccount` (tokens `*_enc`,
   `Status:"connected"` — `mcpoauth.go:44`). Return the account (tokens MASKED).
4. **refresh** — when an account's `ExpiresAt` is within a lead window, POST
   `grant_type=refresh_token`, re-`UpsertMCPOAuthAccount`. PURE
   `needsRefresh(expiresAt, now, lead)` unit-tested.
5. **health** — derive an account's health/status from `ExpiresAt` vs `now`
   (`connected` / `expired` / `error`); the PURE status derivation is unit-tested.
**Binding: REUSE `pkceChallenge`/`randomURLSafe` from `internal/auth/oauth.go`; do NOT
re-implement PKCE.** If those are unexported and inaccessible from package `mcp`,
ESCALATE (ESC-PKCE-GENERALIZE §8) — recommended default: add an exported additive
helper in `internal/auth` (e.g. `auth.GeneratePKCE() (verifier, challenge string)`)
that wraps the existing private funcs, WITHOUT changing any existing signature; never
copy-paste the crypto.

**(E) Health monitor (`internal/mcp/healthmonitor.go`, NEW) — PAR-MCP-038.**
Per-account health: PURE `accountHealth(account, now) Status` (expiry-aware) +
`accountsNeedingRefresh(accounts, now, lead) []*MCPOAuthAccount` — unit-tested. The
periodic ping/refresh loop (a goroutine + ticker calling the engine's refresh) is the
thin integration-only surface (§1.9) — NOT unit-tested with a real ticker; the
PURE derivation it calls IS unit-tested.

**(F) Discovery / compact injection (`internal/mcp/discovery.go`, NEW) — PAR-MCP-039.**
A `tools/list` result cache keyed by instance/url + a PURE compact-manifest assembler
(`buildCompactManifest(tools []ProbeTool) string` — the token-frugal tool listing for
agent injection). The cache get/set + the PURE manifest build are unit-tested; no
network of its own (it caches probe results).

### 1.6 What is UNIT-TESTED vs INTEGRATION-ONLY (binding — the hermeticity guarantee)

**UNIT-TESTED (deterministic, hermetic — `go test ./...` with NO real network / NO
real timeout sleep / NO real clock):**
- **Probe** via the fake `http.RoundTripper`: full handshake → tools extracted;
  initialize 401/403 → `RequiresAuth`; tools/list 401/403 → `RequiresAuth`; the
  `MCP-Protocol-Version: 2025-06-18` header present on every captured request
  (PAR-MCP-010); the `mcp-session-id` read from the initialize response + replayed on
  notifications/initialized + tools/list (PAR-MCP-011); SSE-content-type tools/list →
  `parseSSEDataFrames` finds `id==2` (PAR-MCP-012); JSON-content-type fallback; the 8s
  timeout via a short-ctx + blocking fake transport → `Error:"timeout"` (PAR-MCP-058/
  059, NO real sleep).
- **Registry** via the fake transport + injectable clock: multi-page pagination
  follows `nextCursor` ≤20 pages (PAR-MCP-014); `isDirectConnect` every accept +
  every reject (`mcp.claude.com`, `api.anthropic.com/mcp`, `{`/`<`, `http://`)
  (PAR-MCP-016); `requiredFields`-skip (PAR-MCP-017); URL-dedupe across duplicate
  pages (PAR-MCP-057); cache hit within TTL (no second fetch), cache miss after
  advancing the fake clock past 1h, `force` bypass (PAR-MCP-015 — NO real sleep).
- **OAuth engine** via the fake transport + the temp store: protected-resource +
  auth-server metadata JSON parsed from canned bodies; start persists a flow (verifier
  `*_enc`) + returns an authorize URL keyed on the DISCOVERED endpoint with a valid
  S256 `code_challenge`; complete consumes the flow, exchanges (canned token
  response), upserts an account with tokens `*_enc` + `Status:"connected"`; refresh
  exchanges a canned refresh response + re-upserts; `needsRefresh`/`accountHealth`
  PURE derivations; **no response/DTO carries the cleartext token** (raw `*_enc`
  column ≠ cleartext).
- **SSE transport** PURE cores: `parseSSEFrame` (endpoint event), `parseSSEDataFrames`
  (data-line extraction); the message-POST client asserts method/URL/202 via the fake
  transport.
- **Discovery**: `buildCompactManifest`; the tools cache get/set.

**INTEGRATION-ONLY (NOT unit-tested — thin, isolated, escalation-recorded):** the live
streaming SSE response reader (long-lived `text/event-stream` body dispatch to a
`SessionSink`); the health-monitor periodic ticker goroutine. These call the
PURE/fake-tested cores; their thin live wiring is excluded from `go test ./...`
determinism (§5 grep proof "no-real-network-in-test"). No real network, registry
fetch, OAuth dial, or SSE stream is ever opened by a unit test.

### NOT in scope (explicit — client/engine only)

- **No admin handler** — NO `internal/admin/mcp.go` / `mcpoauth.go` (w7-mcp-3).
- **No route registration** — NO `internal/server/routes_admin.go` edit; NO
  `/api/mcp/{plugin}/{sse,message}` SERVER routes; NO `/api/mcp/*` admin routes; NO
  `guard.go` `LOCAL_ONLY_PATHS` edit (all w7-mcp-3). This plan holds NO serial slot.
- **No agent loop** — `internal/mcp/agent.go` (multi-turn tool execution) is w7-mcp-3.
- **No edits to the SHIPPED w7-mcp-1 files** — `internal/mcp/{runner,bridge,launcher,
  filter,allowlist,defaults,process}.go` are CONSUMED, not edited (mcp-2 adds NEW
  files only — MAP decision 7).
- **No edits to `internal/auth/oauth.go` body/signatures** — the PKCE engine is
  REUSED; if a primitive must be reachable from package `mcp`, prefer an ADDITIVE
  exported helper (ESC-PKCE-GENERALIZE §8); a signature change is an ESCALATION.
- **No edits to `internal/schemas/mcp.go`** — consume the types.
- **No edits to pre-existing store files** except (if needed) ADDITIVE methods on
  `internal/store/mcpoauth.go` (status-only update / list-by-status — §3
  ESC-STORE-ADD); NO migration change expected (the four `mcp_*` tables shipped in
  mcp-1; no new columns anticipated).
- **No UI / mock / seed / spec / e2e** — the w6-l MCP mocks are reconciled by w7-mcp-3.
- **No `New(...)` signature change anywhere** — the components expose NEW constructors
  (`NewProbe`/`NewRegistry`/`NewEngine`) in the foundation package.
- **No real network / SSE stream / OAuth dial / registry fetch in any unit test** —
  all HTTP goes through the injected transport; all timeouts via short ctx; all cache
  TTL via the injectable clock.
- **No secret exposure** — OAuth tokens + PKCE verifier stay `*_enc` at rest (already
  enforced by the mcp-1 store), never echoed.

---

## 2. Precondition checks

Run all before any edit; abort and report to orchestrator on any failure.

```bash
# P0 — clean tree; record <base>
git status --porcelain     # expect empty EXCEPT a possibly-dirty ui/dist/index.html
                           # (gitignored build artifact — NEVER stage it, NEVER revert it).
                           # If anything ELSE is dirty, STOP. Use explicit `git add <file>`, never -A.
git rev-parse HEAD         # record as <base> for §5

# P1 — the w7-mcp-1 FOUNDATION is SHIPPED + present (this plan consumes it)
test -e internal/mcp/launcher.go && test -e internal/mcp/bridge.go && test -e internal/mcp/runner.go && echo "mcp-1 launcher/bridge/runner OK"
test -e internal/store/mcpoauth.go && test -e internal/store/mcpinstances.go && echo "mcp-1 store OK"
grep -nE 'type SessionSink|func splitFrames|func .*AddSession' internal/mcp/bridge.go
grep -nE 'func NewLauncher|func .*StartHTTP|func .*StartSSE|func .*recordInstance' internal/mcp/launcher.go
grep -nE 'func .*UpsertMCPOAuthAccount|func .*CreateMCPOAuthFlow|func .*ConsumeMCPOAuthFlow|func .*GetMCPOAuthAccountByInstance' internal/store/mcpoauth.go
grep -nE 'func DefaultPlugins' internal/mcp/defaults.go

# P2 — the gap is REAL (no probe/registry/oauth/sse/health/discovery yet)
for f in sse probe registry oauth healthmonitor discovery ; do test ! -e internal/mcp/$f.go && echo "internal/mcp/$f.go gap OK" ; done

# P3 — the in-tree PKCE engine to REUSE is present (consume, don't edit)
grep -nE 'func NewOAuthFlow|func pkceChallenge|func randomURLSafe|func .*requestToken|client \*http\.Client' internal/auth/oauth.go

# P4 — NO routes/admin/UI surface this plan (confirm client/engine only)
grep -nE '/api/mcp' internal/server/routes_admin.go ; echo "^ expect EMPTY (mcp-3 adds routes)"
test ! -e internal/admin/mcp.go && echo "no mcp admin handler (correct — mcp-3)"

# P5 — green at base (HERMETIC)
go test ./... && go vet ./... && go build ./...     # exit 0 (no net/process)
```

---

## 3. Exclusive file ownership

After w7-mcp-2 merges, all CREATE files are owned by w7-mcp-2; later MCP plans consume,
never edit (MAP decision 7). w7-mcp-3 EXTENDS `internal/mcp` with NEW files
(admin.go-adjacent handlers live in `internal/admin`, agent.go in `internal/mcp`) and
CONSUMES this plan's `Probe`/`Registry`/`Engine`/SSE-transport/health/discovery.

**CREATE — domain (NEW files in package `internal/mcp`):**

| File | Contract |
|---|---|
| `internal/mcp/sse.go` | CLIENT SSE/message transport: PURE `parseSSEFrame` + `parseSSEDataFrames`; message-POST client (`POST <messageURL>`, expect 202); the live streaming reader dispatching to a `SessionSink` (integration-only body). No `init()`; errors-as-values. |
| `internal/mcp/sse_test.go` | `parseSSEFrame` (endpoint event); `parseSSEDataFrames` (multi `data:` lines, blank-line skip); message-POST via fake transport asserts method/URL/202. RED first. NO real network. |
| `internal/mcp/probe.go` | `Probe{client *http.Client}` + `NewProbe(client)` (nil→default) + `Run(ctx,url) ProbeResult`; the 3-step handshake, session-id replay, 8s ctx timeout, requiresAuth on 401/403, SSE/JSON tools parse. |
| `internal/mcp/probe_test.go` | Via fake `http.RoundTripper`: full handshake→tools; init 401/403→RequiresAuth; tools/list 401/403→RequiresAuth; MCP-Protocol-Version header captured; session-id replayed; SSE-content-type parse (id==2); JSON fallback; short-ctx+blocking transport→`"timeout"` (NO real 8s sleep). RED first. |
| `internal/mcp/registry.go` | `Registry{client,now,cache,mu}` + `NewRegistry(client)` (nil→default; now=time.Now) + `List(ctx,force)`; pagination (≤20, limit=500, cursor), PURE `isDirectConnect`, requiredFields-skip, URL-dedupe, 1h cache via injectable clock; `RegistryServer` DTO. |
| `internal/mcp/registry_test.go` | Via fake transport + fake clock: multi-page cursor follow; `isDirectConnect` accept/reject set; requiredFields-skip; URL-dedupe; cache hit (no 2nd fetch) / miss after clock+1h / force-bypass (NO real sleep). RED first. |
| `internal/mcp/oauth.go` | `Engine{store,client}` + `NewEngine(st,client)` (nil→default); PURE `parseProtectedResourceMetadata` + `parseAuthServerMetadata`; start (REUSE `auth` PKCE primitives + `store.CreateMCPOAuthFlow`); complete (`store.ConsumeMCPOAuthFlow` + token exchange + `store.UpsertMCPOAuthAccount`); refresh; PURE `needsRefresh`. Tokens MASKED in returns. |
| `internal/mcp/oauth_test.go` | Via fake transport + temp store: metadata parse; start persists flow (verifier `*_enc`) + authorize URL on discovered endpoint w/ S256 challenge; complete consumes flow + upserts account (tokens `*_enc`, status connected); refresh re-upserts; `needsRefresh`; NO cleartext token in any return. RED first. |
| `internal/mcp/healthmonitor.go` | PURE `accountHealth(account, now) string` + `accountsNeedingRefresh(accounts, now, lead)`; the periodic ticker loop calling the engine refresh (integration-only). |
| `internal/mcp/healthmonitor_test.go` | `accountHealth` (connected/expired/near-expiry); `accountsNeedingRefresh` selection. RED first. NO real ticker. |
| `internal/mcp/discovery.go` | `tools/list` result cache keyed by instance/url + PURE `buildCompactManifest(tools) string`. |
| `internal/mcp/discovery_test.go` | cache get/set; `buildCompactManifest` shape. RED first. |
| `internal/mcp/transport_test.go` | TEST-ONLY shared `fakeTransport` (`http.RoundTripper`) + `fakeClock` helpers used by probe/registry/oauth tests. (No non-test counterpart.) |

**EXTEND — store (additive only, IF NEEDED):**

| File | Change (additive ONLY — only if the engine/health needs it) |
|---|---|
| `internal/store/mcpoauth.go` | MAY add `SetMCPOAuthAccountStatus(id, status)` and/or `ListMCPOAuthAccountsByStatus(status)` — ADDITIVE methods only, NO column/migration change, NO edit to existing methods. Default: prefer `UpsertMCPOAuthAccount` (already updates `status`) and `ListMCPOAuthAccounts` + in-memory filter, adding a method ONLY if a query is genuinely needed (ESC-STORE-ADD §8). |
| `internal/store/mcpoauth_test.go` | If a method is added, its RED test goes here first. |

**EXTEND — auth (additive ONLY, ONLY on escalation):**

| File | Change |
|---|---|
| `internal/auth/oauth.go` | FORBIDDEN to change any existing signature/body. ONLY if `pkceChallenge`/`randomURLSafe` are unreachable from package `mcp`: add an ADDITIVE exported helper (e.g. `func GeneratePKCE() (verifier, challenge string, err error)`) wrapping the existing private funcs — ESC-PKCE-GENERALIZE, escalate BEFORE adding. Default attempt order: (1) reach via an additive `internal/auth` helper; (2) escalate. NEVER copy the crypto into `internal/mcp`. |

**FORBIDDEN:** everything else. Explicitly: ALL `internal/admin/*` (no MCP handler —
w7-mcp-3); `internal/server/routes_admin.go` (no route — w7-mcp-3 serial slot);
`internal/server/guard.go` (the `LOCAL_ONLY_PATHS` mcp entry is w7-mcp-3); the SHIPPED
`internal/mcp/{runner,bridge,launcher,filter,allowlist,defaults,process}.go` (CONSUME,
do NOT edit); `internal/schemas/mcp.go` (consume); `internal/auth/oauth.go` BODY +
existing signatures (REUSE; additive helper only on escalation); ALL pre-existing
`internal/store/*.go` except ADDITIVE `mcpoauth.go` methods; ALL SHIPPED
`internal/platform/*` (consume the precedent); ALL `internal/inference/*`; ALL `ui/**`
(src, mocks, seeds, specs, dist). Touching any of these (beyond the sanctioned
additive `mcpoauth.go`/`oauth.go` helpers) is an automatic REJECT.

---

## 4. TDD tasks

Cadence (strict, AGENTS.md "TDD always: write test first, see it fail, write minimum
code to pass"): **no Go impl file may exist before its `_test.go` is committed RED.**
`go test ./... && go vet ./... && go build ./...` green at EVERY commit, FULLY
HERMETIC (no real network, no real timeout sleep, no real clock, no SSE stream).
Order: shared fake transport/clock + SSE pure parsers → probe (fake transport) →
registry (fake transport + fake clock) → OAuth engine (fake transport + temp store) →
health + discovery (pure) → closeout.

### T-transport — STEP(a) RED, STEP(b) impl (the SSE pure parsers + the test seam)
STEP(a): write `internal/mcp/transport_test.go` (the `fakeTransport`/`fakeClock`
helpers) + `internal/mcp/sse_test.go` (`parseSSEFrame`, `parseSSEDataFrames`,
message-POST). `go test ./internal/mcp/ -run 'SSE|Frame'` → FAIL. Commit RED:
`phase-1/w7-mcp-2: failing SSE parse + message-POST tests (TDD red)`.
STEP(b): implement `internal/mcp/sse.go` (PURE parsers + the message-POST client;
the live streaming reader stubbed integration-only). Gates green. Commit:
`phase-1/w7-mcp-2: SSE/message client transport (pure frame parse)`.

### T-probe — STEP(a) RED, STEP(b) impl (probe over a FAKE transport)
STEP(a): write `internal/mcp/probe_test.go` (handshake→tools; 401/403→RequiresAuth;
protocol-version header; session-id replay; SSE/JSON parse; short-ctx timeout via a
blocking transport). `go test ./internal/mcp/ -run Probe` → FAIL. Commit RED:
`phase-1/w7-mcp-2: failing MCP probe handshake tests (TDD red)`.
STEP(b): implement `internal/mcp/probe.go` (`NewProbe`/`Run`, the 3-step handshake,
8s ctx, requiresAuth, SSE/JSON tools parse). Gates green (fake transport only).
Commit: `phase-1/w7-mcp-2: MCP probe (initialize+initialized+tools/list, 8s, requiresAuth)`.

### T-registry — STEP(a) RED, STEP(b) impl (registry over a FAKE transport + clock)
STEP(a): write `internal/mcp/registry_test.go` (pagination cursor follow;
`isDirectConnect` accept/reject; requiredFields-skip; URL-dedupe; cache hit/miss via
the fake clock; force-bypass). → FAIL. Commit RED:
`phase-1/w7-mcp-2: failing MCP registry pagination/cache/dedupe tests (TDD red)`.
STEP(b): implement `internal/mcp/registry.go` (`NewRegistry`/`List`, pagination, PURE
`isDirectConnect`, dedupe, 1h injectable-clock cache). Gates green. Commit:
`phase-1/w7-mcp-2: MCP registry client (pagination + 1h cache + direct-connect + dedupe)`.

### T-oauth — STEP(a) RED, STEP(b) impl (OAuth engine; REUSE auth PKCE + mcp-1 store)
STEP(a): write `internal/mcp/oauth_test.go` (metadata parse; start→flow persisted
+ authorize URL on discovered endpoint; complete→account upserted tokens `*_enc`;
refresh; `needsRefresh`; no-cleartext-token). If `auth` PKCE primitives are
unreachable, ESCALATE (ESC-PKCE-GENERALIZE) before adding the additive `auth` helper;
if a store method is needed, write its RED test in `mcpoauth_test.go` first
(ESC-STORE-ADD). → FAIL. Commit RED:
`phase-1/w7-mcp-2: failing MCP OAuth engine tests (TDD red)`.
STEP(b): implement `internal/mcp/oauth.go` (discovery parsers + start/complete/refresh
REUSING `auth` PKCE + the SHIPPED `mcpoauth` store) + any sanctioned additive helper.
Gates green. Commit:
`phase-1/w7-mcp-2: MCP OAuth engine (PKCE reuse + PRM/ASM discovery + token refresh)`.

### T-health-discovery — STEP(a) RED, STEP(b) impl (pure health + discovery)
STEP(a): write `internal/mcp/healthmonitor_test.go` (`accountHealth`,
`accountsNeedingRefresh`) + `internal/mcp/discovery_test.go` (cache get/set,
`buildCompactManifest`). → FAIL. Commit RED:
`phase-1/w7-mcp-2: failing health-monitor + discovery tests (TDD red)`.
STEP(b): implement `internal/mcp/healthmonitor.go` (PURE derivations + the
integration-only ticker loop) + `internal/mcp/discovery.go` (tools cache + compact
manifest). Gates green. Commit:
`phase-1/w7-mcp-2: MCP health monitor + discovery/compact-injection`.

### T-close — full gates + closeout
```bash
go test ./... && go vet ./... && go build ./...                       # HERMETIC — no net/sleep
go test ./internal/mcp/... -run 'Probe|Registry|OAuth|SSE|Health|Discovery|Frame' -v
go test ./internal/store/ -run 'MCPOAuth|Mcp' -v                      # if a store method was added
```
Flip the matrix in `.planning/parity/matrix/9router-mcp.md`: PAR-MCP-009,010,011,012,
013,014,015,016,017,037,038,039,057,058,059 → HAVE (real Go; all HTTP hermetically
unit-tested via an injected transport; the periodic health-ping loop + live SSE stream
are integration-only — §1.6/§1.9). PAR-MCP-001,002,055,056 footnote (client SSE/message
transport shipped w7-mcp-2; server routes + handshake-event emission w7-mcp-3). Append
the §8 open items to `open-questions.md` (ESC-HTTP-SEAM resolution, ESC-MCP-OAUTH-PROTOCOL
recommended default, ESC-PKCE-GENERALIZE outcome, ESC-STORE-ADD outcome, ESC-REG-DTO,
the integration-only SSE-stream/health-ticker note, the mcp-3 dependency handoff).
Update `docs/WORKFLOW.md` (P5 base observation; the ESC decisions; the constructors
mcp-3 consumes). Final commit:
`phase-1/w7-mcp-2: close — MCP client (probe+registry+oauth+sse); matrix flip`.

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0. Diff gate is
**w7-mcp-2 commit-range-scoped** (§7).

**Test gates (HERMETIC — no real network, no real timeout sleep, no real clock, no SSE
stream)**
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/mcp/... -run 'Probe|Registry|OAuth|SSE|Health|Discovery|Frame' -v`
  → exit 0, all pass (probe: handshake + 401/403-requiresAuth + protocol-version
  header + session-id replay + SSE/JSON parse + timeout; registry: pagination +
  isDirectConnect accept/reject + requiredFields-skip + URL-dedupe + cache
  hit/miss/force; oauth: metadata parse + start/complete/refresh + tokens `*_enc` +
  no-cleartext; sse: parseSSEFrame/parseSSEDataFrames + message-POST 202; health:
  accountHealth + needs-refresh; discovery: cache + manifest).
- `go test ./internal/store/ -run 'MCPOAuth|Mcp' -v` → exit 0 (if an additive store
  method was added; else N/A).
- NO playwright (no UI surface this plan).

**TDD-order proof** — each impl file's covering test appears in an earlier-or-equal
commit:
```bash
for pair in \
  "internal/mcp/sse_test.go:internal/mcp/sse.go" \
  "internal/mcp/probe_test.go:internal/mcp/probe.go" \
  "internal/mcp/registry_test.go:internal/mcp/registry.go" \
  "internal/mcp/oauth_test.go:internal/mcp/oauth.go" \
  "internal/mcp/healthmonitor_test.go:internal/mcp/healthmonitor.go" \
  "internal/mcp/discovery_test.go:internal/mcp/discovery.go" ; do
  tf=${pair%%:*}; im=${pair##*:}
  ct=$(git log --format=%ct --diff-filter=A -1 -- "$tf")
  cf=$(git log --format=%ct --diff-filter=A -1 -- "$im")
  [ "$ct" -le "$cf" ] || echo "TDD VIOLATION: $im"     # prints nothing
done
```

**Grep proofs**
```bash
# protocol constants ported verbatim
grep -nE 'mcpProtocolVersion|2025-06-18|probeTimeout|8 \* time\.Second|registryCacheTTL|1 \* time\.Hour|limit=500|registryMaxPages|20' internal/mcp/probe.go internal/mcp/registry.go
# injectable HTTP client + clock seams (mirror auth/oauth.go:128)
grep -nE 'client \*http\.Client|func NewProbe|func NewRegistry|func NewEngine|now +func\(\) time\.Time' internal/mcp/probe.go internal/mcp/registry.go internal/mcp/oauth.go
# probe handshake + requiresAuth + session-id replay + timeout
grep -nE 'initialize|notifications/initialized|tools/list|RequiresAuth|mcp-session-id|MCP-Protocol-Version|DeadlineExceeded|"timeout"' internal/mcp/probe.go
# registry filter + dedupe (PAR-MCP-016/017/057)
grep -nE 'func isDirectConnect|mcp\.claude\.com|api\.anthropic\.com|requiredFields|nextCursor|seen' internal/mcp/registry.go
# OAuth REUSES auth PKCE (no re-implemented crypto in mcp)
grep -nE 'auth\.(NewOAuthFlow|GeneratePKCE)|pkceChallenge|sha256' internal/mcp/oauth.go ; echo "^ no raw sha256 PKCE in mcp/oauth.go (REUSE auth)"
! grep -nE 'crypto/sha256' internal/mcp/oauth.go && echo "PKCE crypto NOT copied into mcp OK"
# OAuth consumes the SHIPPED mcp-1 store (tokens *_enc enforced there)
grep -nE 'CreateMCPOAuthFlow|ConsumeMCPOAuthFlow|UpsertMCPOAuthAccount' internal/mcp/oauth.go
# no init(); no free global state
! grep -rn 'func init(' internal/mcp/sse.go internal/mcp/probe.go internal/mcp/registry.go internal/mcp/oauth.go internal/mcp/healthmonitor.go internal/mcp/discovery.go && echo "no init() OK"
```

**No-real-network / no-real-sleep proofs (binding — hermeticity)**
```bash
# unit tests NEVER dial the network, open a real http transport, or sleep on a real timeout:
! grep -nE 'http\.Get|http\.Post|http\.DefaultClient|net\.Dial|net\.Listen|time\.Sleep\([^)]*Second' \
   internal/mcp/sse_test.go internal/mcp/probe_test.go internal/mcp/registry_test.go \
   internal/mcp/oauth_test.go internal/mcp/healthmonitor_test.go internal/mcp/discovery_test.go \
   && echo "no real net / no real-second sleep in tests OK"
# tests inject a fake RoundTripper + a fake clock:
grep -nE 'http\.RoundTripper|fakeTransport|RoundTrip|fakeClock' internal/mcp/transport_test.go
grep -nE 'Transport:|fakeTransport|fakeClock|now =' internal/mcp/probe_test.go internal/mcp/registry_test.go internal/mcp/oauth_test.go
# the probe 8s timeout is tested via a SHORT ctx + a blocking transport (NOT a real 8s wait):
grep -nE 'context\.WithTimeout|WithCancel|block' internal/mcp/probe_test.go
# the registry 1h cache is tested via the injectable clock (NOT a real 1h wait):
grep -nE 'now =|Add\(.*Hour\)|fakeClock' internal/mcp/registry_test.go
```

**No-secret-exposure proofs (binding)**
```bash
# the OAuth engine consumes the *_enc store; the oauth_test asserts the raw column !=
# cleartext and that no returned account DTO carries the cleartext token:
grep -nE 'access_token_enc|refresh_token_enc|verifier_enc|s\.cipher' internal/store/mcpoauth.go   # (already shipped — consumed)
grep -nE 'AccessToken|RefreshToken|token_set|cleartext|!= ' internal/mcp/oauth_test.go            # the masking assertion present
```

**Negative / freeze proofs (w7-mcp-2 commit-range — §7)**
```bash
R="<first-w7-mcp-2>^..<last-w7-mcp-2>"
# Only the sanctioned Go files changed:
git diff $R --name-only -- internal/ | grep -vE \
 'internal/mcp/(sse|probe|registry|oauth|healthmonitor|discovery)(_test)?\.go|internal/mcp/transport_test\.go|internal/store/mcpoauth(_test)?\.go|internal/auth/oauth\.go' \
 | wc -l                                                                  # = 0
# CLIENT/ENGINE-ONLY: NO routes_admin.go, NO admin handler, NO guard edit:
git diff $R --name-only -- internal/server/routes_admin.go | wc -l       # = 0  (confirm NO route this plan)
git diff $R --name-only -- internal/admin/ | wc -l                       # = 0  (confirm NO admin handler)
git diff $R --name-only -- internal/server/guard.go | wc -l              # = 0
# SHIPPED mcp-1 files + schemas + platform + inference untouched:
git diff $R --name-only -- internal/mcp/launcher.go internal/mcp/bridge.go internal/mcp/runner.go internal/mcp/filter.go internal/mcp/allowlist.go internal/mcp/process.go internal/mcp/defaults.go internal/schemas/mcp.go internal/platform/ internal/inference/ | wc -l   # = 0
# auth/oauth.go: if touched, ADDITIVE only (no deletions of existing logic):
git diff $R -- internal/auth/oauth.go | grep -E '^-' | grep -v '^---' | wc -l   # = 0 (additive helper only, or untouched)
# mcpoauth.go: if touched, ADDITIVE only:
git diff $R -- internal/store/mcpoauth.go | grep -E '^-' | grep -v '^---' | wc -l   # = 0
# NO UI change at all:
git diff $R --name-only -- ui/ | wc -l                                   # = 0
```

---

## 6. Out of scope (restated, binding)

NO admin handler (`internal/admin/mcp.go` — w7-mcp-3). NO route registration
(`routes_admin.go` — w7-mcp-3 holds the serial slot; THIS plan holds NONE). NO
`/api/mcp/{plugin}/{sse,message}` SERVER routes + handshake-event emission + SSE
response headers (PAR-MCP-001/002/055/056 server half — w7-mcp-3). NO `guard.go`
`LOCAL_ONLY_PATHS` edit (w7-mcp-3). NO agent loop (`internal/mcp/agent.go` — w7-mcp-3).
NO edits to the SHIPPED w7-mcp-1 `internal/mcp/*` files (consume; add NEW files only).
NO edits to `internal/schemas/mcp.go` (consume the types). NO edits to
`internal/auth/oauth.go` body/signatures (REUSE the PKCE engine; ADDITIVE exported
helper only on ESC-PKCE-GENERALIZE). NO edits to pre-existing store files except
ADDITIVE `mcpoauth.go` methods (ESC-STORE-ADD; no migration/column change). NO
`internal/inference/*` / `internal/platform/*` edits. NO UI (src/mocks/seeds/specs/
dist — the w6-l mocks are reconciled by w7-mcp-3). NO `New(...)` signature change (NEW
package constructors only). NO secret exposure (OAuth tokens + PKCE verifier `*_enc`,
never echoed). **NO real network / SSE stream / OAuth dial / registry fetch / real
timeout sleep / real clock in any unit test** — all HTTP through the injected
transport, all timeouts via short ctx, all cache TTL via the injectable clock; the live
SSE stream + the health-ticker loop are integration-only (§1.6/§1.9); the unit suite is
fully hermetic. Ambiguity (registry URL/shape, MCP-OAuth discovery detail, PKCE
generalization, the SSE wire format) → ESCALATE (§8) with the recommended default from
the ref/spec — NEVER fabricate a protocol.

## 7. Diff-gate scope

The MCP track runs concurrently with governance/providers/platform, so a broad
`<base>..HEAD` range sweeps in sibling commits. The diff gate MUST be scoped to
w7-mcp-2's own commits. Isolate them:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w7-mcp-2:" | awk '{print $1}'`
then `git diff <first-w7-mcp-2>^..<last-w7-mcp-2> -- [file list]`.

`git diff <first>^..<last> --name-only` must be exactly a subset of:
```
internal/mcp/sse.go
internal/mcp/sse_test.go
internal/mcp/probe.go
internal/mcp/probe_test.go
internal/mcp/registry.go
internal/mcp/registry_test.go
internal/mcp/oauth.go
internal/mcp/oauth_test.go
internal/mcp/healthmonitor.go
internal/mcp/healthmonitor_test.go
internal/mcp/discovery.go
internal/mcp/discovery_test.go
internal/mcp/transport_test.go              (test-only fake http transport + clock)
internal/store/mcpoauth.go                  (ADDITIVE methods only — IF NEEDED)
internal/store/mcpoauth_test.go             (RED for any added method — IF NEEDED)
internal/auth/oauth.go                       (ADDITIVE exported helper only — ONLY on ESC-PKCE-GENERALIZE)
.planning/parity/matrix/9router-mcp.md       (row flips)
.planning/parity/plans/open-questions.md
docs/WORKFLOW.md
```
Any file outside this list in the scoped diff is an automatic review REJECT.
`internal/server/routes_admin.go`, `internal/admin/**`, `internal/server/guard.go`,
`internal/schemas/mcp.go`, the SHIPPED `internal/mcp/{launcher,bridge,runner,filter,
allowlist,process,defaults}.go`, `internal/platform/**`, `internal/inference/**`, and
ALL `ui/**` are deliberately ABSENT — touching them is an automatic REJECT. This plan
holds NO routes_admin.go serial slot (the MCP slot is w7-mcp-3's).

## 8. Escalations / decisions (explicit — recommended defaults, do not fabricate)

- **ESC-HTTP-SEAM (RESOLVED at authoring — the injection mechanism, binding default).**
  Inject an `*http.Client` field per network component (`Probe`/`Registry`/`Engine`),
  nil-able to a real default — REUSING `internal/auth/oauth.go:128`'s exact
  `NewOAuthFlow(cfg, st, client *http.Client)` shape (no new non-test interface).
  Tests pass `&http.Client{Transport: <fakeRoundTripper>}`. The fake `http.RoundTripper`
  captures requests (header/body/method assertions) + returns canned responses (incl.
  401/403, SSE bodies, paginated registry pages, token responses) and can block (for
  the probe-timeout test). The probe 8s timeout is `context.WithTimeout`; tests use a
  short ctx. The registry 1h cache is gated by an injectable `now func() time.Time`;
  tests advance a fake clock. NO real network, NO real timeout sleep, NO real clock in
  any unit test. Flag for orchestrator confirmation.
- **ESC-MCP-OAUTH-PROTOCOL (CONDITIONAL — the MCP OAuth flow has NO 9router probe).**
  The frozen 9router ref DISABLED Cowork and ships NO MCP-OAuth engine; PAR-MCP-037 is
  a MISSING row with no reference implementation to port. **Default: author the engine
  against the MCP authorization spec — RFC 9728 protected-resource-metadata discovery
  (`<resource>/.well-known/oauth-protected-resource` → `authorization_servers[]`),
  RFC 8414 authorization-server-metadata (`<as>/.well-known/oauth-authorization-server`
  → `authorization_endpoint` + `token_endpoint`), RFC 7636 PKCE (S256), reusing
  `internal/auth/oauth.go` primitives + the SHIPPED `mcpoauth` store.** If a discovery
  detail is ambiguous (e.g. the exact well-known path, the `resource` param, dynamic
  client registration), STOP and ESCALATE with the spec section — never fabricate a
  protocol. The unit test pins the OBSERVABLE behavior (flow persisted, authorize URL
  on the discovered endpoint with a valid S256 challenge, code→token exchange, refresh,
  tokens `*_enc`), so a discovery-shape refinement that preserves behavior is
  acceptable; a behavioral divergence is an escalation. Flag.
- **ESC-PKCE-GENERALIZE (CONDITIONAL — reuse the in-tree PKCE primitives).**
  `pkceChallenge` (`auth/oauth.go:274`) + `randomURLSafe` (`:266`) are unexported.
  **Default attempt order: (1) if reachable, reuse directly; (2) if not (different
  package), add an ADDITIVE exported helper in `internal/auth` — e.g.
  `func GeneratePKCE() (verifier, challenge string, err error)` wrapping the existing
  private funcs, WITHOUT changing any existing signature/body — and reuse that; (3) if
  even that is contentious, escalate.** NEVER copy the SHA-256/base64 crypto into
  `internal/mcp`. The grep proof asserts no `crypto/sha256` in `internal/mcp/oauth.go`.
  Flag the chosen path for orchestrator review.
- **ESC-STORE-ADD (CONDITIONAL — additive mcpoauth store method).** The SHIPPED
  `mcpoauth.go` already supports account upsert (which updates `status`) + list. The
  health monitor may want `SetMCPOAuthAccountStatus(id, status)` and/or
  `ListMCPOAuthAccountsByStatus(status)`. **Default: prefer the existing
  `UpsertMCPOAuthAccount` + `ListMCPOAuthAccounts` (filter in memory); add an additive
  method ONLY if a genuine query need exists — and TDD it RED-first in
  `mcpoauth_test.go`.** No column/migration change (the four `mcp_*` tables shipped in
  mcp-1). Flag if a method is added.
- **ESC-REG-DTO (RESOLVED at authoring — registry DTO shape, binding default).** The
  registry returns a NEW `RegistryServer` DTO mirroring the 9router registry fields
  (`name, slug, title, description, url, transport, oauth, toolNames, toolCount,
  iconUrl` — `cowork-mcp-registry/route.js:40-51`), NOT a reuse of the SHIPPED
  `PluginDefinition` (`defaults.go:7`). mcp-3 maps `RegistryServer` → its
  marketplace/UI DTO. Flag.
- **ESC-SSE-STREAM (RESOLVED at authoring — live SSE reader integration-only,
  recommended default; mirrors w7-mcp-1 ESC-SPAWN).** The long-lived streaming SSE
  response reader (dispatching `text/event-stream` frames to a `SessionSink`) is
  INTEGRATION-ONLY; the PURE frame parsers (`parseSSEFrame`/`parseSSEDataFrames`) + the
  message-POST client are FULLY unit-tested via the fake transport. The health-monitor
  periodic ticker loop is likewise integration-only; its PURE derivations are
  unit-tested. Record at closeout: PAR-MCP-001/002/055/056 (client half) + PAR-MCP-038
  HAVE with an "integration-only stream/ticker" footnote. Flag.
- **Client confirms NO routes / NO UI (binding).** This plan adds NO admin handler, NO
  `routes_admin.go` edit, NO UI/mock/seed/spec/e2e. The MCP routes_admin serial slot
  belongs to w7-mcp-3 (MAP §181/§208); this plan holds NONE. The w6-l MCP UI mocks are
  reconciled by w7-mcp-3, not here. Recorded for orchestrator: w7-mcp-2 is fully
  parallelizable with every non-MCP track and takes no serial file; it is serially
  AFTER w7-mcp-1 (consumes its constructors) and BEFORE w7-mcp-3 (which consumes this
  plan's probe/registry/engine/transport).
- **mcp-3 dependency handoff (binding).** w7-mcp-3 (admin transport + routes + tools/
  agent) DEPENDS on this plan's exported surface: `NewProbe`/`Probe.Run`,
  `NewRegistry`/`Registry.List` + `RegistryServer`, `NewEngine` + start/complete/
  refresh + the OAuth account flow, the SSE/message client transport + `parseSSEFrame`/
  `parseSSEDataFrames`, the health derivations, the discovery cache + compact manifest.
  Keep these exported + stable; mcp-3 EXTENDS `internal/mcp` with NEW files (agent.go)
  + adds the SERVER SSE/message routes + admin handlers — no edits to this plan's files.
  Record in `open-questions.md` at closeout.
```
