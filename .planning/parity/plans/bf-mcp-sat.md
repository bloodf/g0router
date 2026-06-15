# Micro-plan bf-mcp-sat — bifrost-mcp client-mode SAT verification + matrix flip (Go)

```
program: bifrost-parity (Waves 6+; CLI_ORCHESTRATOR.md governs, HANDOFF.md retired).
  bifrost phase — this is the FINAL bifrost buildable unit. After bf-mcp-sat the
  bifrost-mcp matrix has no further buildable rows (the residual is the ESC
  enterprise/per-user surface, recorded permanently in open-questions.md).
plan: bf-mcp-sat
status: READY (rev 1 — authored against the LIVE post-bf-mcp-2 tree @ <base> =
  f043592 (`git rev-parse HEAD`): the full SHIPPED Wave-7 CLIENT-mode tree
  internal/mcp/{launcher,process,bridge,probe,registry,oauth,agent,discovery,
  healthmonitor,toolpolicy,filter,allowlist,sse,runner,defaults}.go +
  internal/store/{mcpinstances,mcpoauth,mcptoolgroups,mcpvkconfigs,mcpclientflags}.go
  + internal/admin/mcp.go + the SHIPPED bf-mcp-1 server.go + bf-mcp-2 scope.go.
  BIFROST-MAP.md bifrost-mcp disposition §259-271 (the SAT/VAR row §263);
  architectural decision #6 §121-138; freeze rules §384-399. PRIMARY structural
  template = bf-core-1.md (the matrix-flip-only closeout pattern — verify the LIVE
  tree, flip HONESTLY, add a small regression test ONLY where a flip claims an
  untested behavior; ship NO production code).)
runs: MCP track, AFTER bf-mcp-1 + bf-mcp-2 (both MERGED — their dispositions are
  already in bifrost-mcp.md). Disjoint from openai/gov/core tracks (run ∥).
branch: main (direct commits, per AGENTS.md "No PR workflow")
commit-prefix: phase-1/bf-mcp-sat:
footer: Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>
ref-source: BLOCKED — ESC-REF-ABSENT (BIFROST-MAP §47-68). The frozen Bifrost ref
  @ ca21298 is ABSENT on this host. The ONLY ground truth is
  .planning/parity/matrix/bifrost-mcp.md + the LIVE g0router tree. A row is flipped
  ONLY on file:line evidence in the live CLIENT-mode tree. NO row is flipped on the
  strength of the MAP's optimistic claim alone — the MAP §263 over-claimed (it
  predates the honest live-tree audit in §0); where the live tree does NOT evidence
  the behavior, the row STAYS MISSING and is escalated. NEVER inflate the parity
  count.
go-serial-slot: NONE. bf-mcp-sat registers NO HTTP routes and edits NO route file.
  The routes_admin.go / routes_mcp.go MCP serial chain TERMINATED at bf-core-2
  (BIFROST-MAP §343-346: bf-mcp-1 → bf-mcp-2 → bf-core-2). bf-mcp-sat is NOT a
  routes serial holder — it adds no routes.
new-route: NO.
headline: this is a VERIFY-THEN-FLIP closeout (bf-core-1 pattern). It ships NO
  production Go code. The MAP routed 17 rows here as "SAT/VAR — flip to HAVE/PARTIAL"
  (001,005,006,007,008,009,010,011,021,024,025,026,027,066,067,076,080). The honest
  live-tree audit (§0) finds the MAP OVER-CLAIMED: only TWO are clean HAVE (007 STDIO,
  011 OAuth); four are PARTIAL (001,005,006,021); four are VAR (024,025,067,076); and
  SEVEN STAY MISSING/ESC (008,009,010,026,027,066,080) because the live client-mode
  tree does NOT evidence the behavior. Where a flip to HAVE/PARTIAL/VAR claims a
  behavior with NO existing regression test, bf-mcp-sat adds a SMALL hermetic
  additive regression test (no production code) to LOCK it. Most candidate behaviors
  are ALREADY tested (launcher_test/oauth_test/agent_test/probe/transport_test) →
  pure-flip; only the cases below need a flip+regression-test.
```

---

## 0. Objective + the PER-ROW verification table (the heart of this plan)

### 0.1 Objective

bf-mcp-2 closed the server-mode + per-VK + config-flag rows
(002/003/004/017/018/019/020/032/033/039/049/052/053/054/057/071/075/077/078/079 —
all already HAVE/HAVE-by-variant/PARTIAL in `bifrost-mcp.md`, cited to bf-mcp-1/2).
bf-mcp-sat is the FINAL unit: it audits the 17 rows the MAP §263 claimed were
ALREADY-SATISFIED by g0router's shipped Wave-7 **CLIENT-mode** MCP, and flips the
matrix **HONESTLY** against file:line evidence in the live tree — never against the
MAP's claim. Where a true flip asserts an untested behavior, it adds a small additive
regression test (no production `.go` change). Where the behavior is NOT evidenced in
the live tree, the row **STAYS MISSING** and is escalated (do NOT inflate parity).

The MAP's claim was: "g0router's client-mode covers HTTP/SSE/STDIO transports, STDIO
command/args/env, none/headers/oauth auth concepts, ping-vs-listTools health, agent
loop, TLS, connection-state, mcp-go-equivalent, EnvVar-encrypted connection string —
flip these to HAVE/PARTIAL." The audit below shows that claim was partly true
(STDIO, OAuth, agent loop, the HTTP/SSE probe path) and partly OPTIMISTIC (no
auth-type enum, no static-header injection, no InProcess, no reconnect-state health
monitor, no credential-store interface, no TLS config, no EnvVar-encrypted
ConnectionString). The honest flip is the deliverable.

### 0.2 Per-row verification table (row → matrix behavior → LIVE file:line evidence → disposition → pure-flip vs flip+regression-test)

Disposition keys: **HAVE** (fully covered, CLIENT direction) · **PARTIAL** (covered
for client; a piece — InProcess / launcher live-connection / ping-fallback toggle —
is separate/deferred) · **VAR** (g0router's own-mechanism variant; behavior met,
shape differs) · **STILL-MISSING/ESC** (NOT evidenced in the live tree — NOT flipped).

| Row | Matrix behavior (current status) | LIVE evidence (file:line) | Honest disposition | Flip kind |
|---|---|---|---|---|
| **001** | client mode connect: HTTP, SSE, STDIO, **InProcess** (MISSING) | STDIO live: `launcher.go:54 StartStdio` spawns via `runner.Start(ProcessSpec{...})`. HTTP/SSE discovery: `probe.go:55 Run` does real `initialize`→`notifications/initialized`→`tools/list` over an injectable `*http.Client`; `sse.go` SSE client. InProcess: ABSENT (`! grep -ri inprocess internal/mcp` → none). | **PARTIAL** (STDIO live + HTTP/SSE probe-discovery; **InProcess absent**, launcher HTTP/SSE persistent-connection is a no-dial placeholder `launcher.go:103 recordInstance`). | **pure-flip** (STDIO covered by `launcher_test.go TestLauncherStartStdio*`; probe by `probe_test.go`/`transport_test.go`). |
| **005** | Transport: HTTP (MISSING) | `probe.go:55 Run` real HTTP dial (3-step handshake) over injectable client; `mcpinstances.go:148` persists `transport='http'`; `launcher.go:90 StartHTTP` records the mode (no-dial placeholder, comment `:88-90` "the real HTTP/SSE client is w7-mcp-2"). | **PARTIAL** (transport modeled + probe-dialed; the persistent launcher connection deferred — w7-mcp-2 launcher live-dial is a no-op `recordInstance`). | **pure-flip** (`transport_test.go` + `launcher_test.go:172 TestLauncherStartHTTPSSEModes`). |
| **006** | Transport: SSE (MISSING) | `sse.go` SSE client (`req.Header Accept: text/event-stream`); `probe.go:142 extractTools` parses `text/event-stream` bodies; `mcpinstances.go` persists `transport='sse'`; `launcher.go:96 StartSSE` records the mode (no-dial placeholder). | **PARTIAL** (SSE wire helpers + probe SSE parse; launcher persistent SSE connection deferred). | **pure-flip** (`sse_test.go`, `transport_test.go`, `launcher_test.go:172`). |
| **007** | Transport: STDIO with command/args/env (MISSING) | `launcher.go:54 StartStdio(name, command, args, env)` → `runner.Start(ProcessSpec{Command, Args, Env, ...})`; `process.go:34 exec.Command(spec.Command, spec.Args...)`, `:35 cmd.Env = mergeEnv(spec.Env)`. Allowlist-gated before spawn (`launcher.go:55 isAllowedCommand`). | **HAVE** (genuine live STDIO spawn with command/args/env). | **pure-flip** (`launcher_test.go` 14 `StartStdio` hits; `process` covered via runner fakes). |
| **008** | Transport: InProcess (MISSING) | ABSENT. `! grep -rniE 'inprocess\|in_process\|in-process' internal/mcp/*.go` → no match. No in-process server wiring exists. | **STILL-MISSING / ESC** (not built; not applicable to g0router's process-launcher model). NOT FLIPPED. | n/a (stays MISSING; escalated §3). |
| **009** | Auth type: none (`MCPAuthTypeNone`) (MISSING) | ABSENT. No `auth_type`/`AuthType`/`MCPAuthType` enum anywhere (`! grep -rniE 'auth_type\|AuthType\|MCPAuthType' internal/store/mcp*.go internal/schemas/mcp.go internal/mcp/*.go` → none). OAuth is wired directly via the `Engine` (`oauth.go`), NOT via an auth-type switch with a `none` member. | **STILL-MISSING / ESC** (no auth-type enum/state; the "none" concept does not exist as a typed mode). NOT FLIPPED. | n/a (stays MISSING; escalated §3). |
| **010** | Auth type: headers (server-level static headers) (MISSING) | ABSENT for the CLIENT-connect direction. No static-auth-header injection from client config onto the upstream connect: `sse.go` sets only `Content-Type`/`Accept`; no `headers_json`/`Headers map` consumed on connect. (NOTE: bf-mcp-2's `allowed_extra_headers` `mcpclientflags.go:15` is the SERVER-mode request-header whitelist — a different row 071, already PARTIAL — NOT client-mode static auth headers.) | **STILL-MISSING / ESC** (no client-mode static-header auth on connect). NOT FLIPPED. | n/a (stays MISSING; escalated §3). |
| **011** | Auth type: oauth (server-level OAuth 2.0) (MISSING) | `oauth.go:20 Engine` — full authorization-code-with-PKCE flow: `:46 Start` (RFC 9728/8414 discovery + persists PKCE flow, verifier `*_enc`), `:88 Complete` (token exchange), `:128 Refresh`; tokens `*_enc` at rest (`mcpoauth.go:45 cipher.Encrypt`). Live caller wired in bf-mcp-1 (`admin/mcp.go CompleteInstanceAuth`). | **HAVE** (full server-level OAuth 2.0 for the CLIENT direction). | **pure-flip** (`oauth_test.go` 28 Start/Complete/Refresh/PKCE/verifier hits). |
| **021** | Tool discovery: **ping vs listTools** health-check fallback (`IsPingAvailable`) (MISSING) | `probe.go:80` uses `tools/list` as the discovery/health signal; there is NO `IsPingAvailable` field and NO ping-first-then-listTools FALLBACK toggle (`! grep -rniE 'IsPingAvailable\|"ping"\|ping/' internal/mcp/probe.go` → no ping method). g0router's health signal IS tools/list (one path, not a ping-vs-listTools selector). | **VAR** (g0router uses tools/list as the single health/discovery signal; the ping-vs-listTools *fallback selector* + `IsPingAvailable` toggle is a Bifrost shape g0router does not have — variant by design). | **pure-flip** (the tools/list health path is covered by `probe_test.go`; the VAR note records the absent toggle — no new test needed because nothing new is claimed HAVE). |
| **024** | Agent loop: iterative tool exec up to MaxAgentDepth for **Chat** API (MISSING) | `agent.go:84 Agent.Run` — bounded multi-turn tool-execution loop, `:74 NewAgent(exec, maxTurns)` with `:13 defaultAgentMaxTurns=8`, `:22 ErrMaxTurnsExceeded` (no runaway). Driven by an injected `:55 ModelStep`. | **HAVE-by-variant** (a generic bounded agent loop exists and is the Chat-API consumer; g0router has ONE generic loop, not a Chat-specific `CheckAndExecuteAgentForChatRequest` entrypoint — the behavior is met, the entrypoint shape is VAR). | **pure-flip** (`agent_test.go` 18 Run/maxTurns/ModelStep hits). |
| **025** | Agent loop: iterative tool exec up to MaxAgentDepth for **Responses** API (MISSING) | Same `agent.go:84 Agent.Run` generic loop; there is NO separate Responses-API agent entrypoint (`! grep -rniE 'Responses' internal/mcp/agent.go` → none). The single generic loop serves both API shapes via the injected `ModelStep`. | **VAR** (the iterative-tool-exec BEHAVIOR is covered by the same generic `Agent.Run`; a Responses-specific entrypoint à la `CheckAndExecuteAgentForResponsesRequest` does not exist — variant: one generic loop, not two API-specific entrypoints). | **pure-flip** (covered by `agent_test.go`; the VAR note records the single-loop shape — no new HAVE claim, no new test). |
| **026** | Health monitor: automatic **reconnect on startup failure** with retention in **Disconnected** state (MISSING) | g0router's `healthmonitor.go` is an OAuth-**token-refresh** sweeper: `:13 accountHealth`, `:25 accountsNeedingRefresh`, `:45 NewHealthMonitor(st, interval, lead)`, `:64 sweep` — it monitors token EXPIRY and refreshes, NOT connection reconnect-on-failure with a Disconnected-state lifecycle. No reconnect-on-startup-failure path exists. | **STILL-MISSING / ESC** (g0router's "health monitor" is a *different* thing — OAuth token refresh — NOT connection-reconnect with Disconnected retention; the 026 behavior is not built). NOT FLIPPED. | n/a (stays MISSING; escalated §3 — the name collision is recorded so no reviewer mistakes the OAuth sweeper for 026). |
| **027** | Credential store interface: `ConnectionHeaders`, `RequestHeaders`, `RequiresPerCallConnection` (MISSING) | ABSENT. `! grep -rniE 'CredentialStore\|ConnectionHeaders\|RequestHeaders\|RequiresPerCall' internal/mcp/*.go internal/store/mcp*.go` → no match. No credential-store abstraction. | **STILL-MISSING / ESC** (no credential-store interface; the abstraction does not exist). NOT FLIPPED. | n/a (stays MISSING; escalated §3 — it is the per-user enterprise surface's foundation, BIFROST-MAP §269 ESC). |
| **066** | TLS config with `InsecureSkipVerify` and `CACertPEM` (env-var aware) (MISSING) | ABSENT in the MCP tree. `! grep -rniE 'InsecureSkipVerify\|CACert\|tls\.Config\|x509' internal/mcp/*.go internal/store/mcp*.go` → no match. The probe/SSE clients use the default `*http.Client` (`probe.go:48 defaultHTTPClient`); no per-client TLS config schema. | **STILL-MISSING / ESC** (no MCP TLS config schema/field). NOT FLIPPED. | n/a (stays MISSING; escalated §3). |
| **067** | MCP client **state machine**: Connected, Disconnected, Error, PendingTools, Disabled (MISSING) | g0router's instance status is a **free-form** string `'stopped' \| 'starting' \| 'running' \| 'error'` (`mcpinstances.go:34` comment; `:142` default `'stopped'`; `SetMCPInstanceStatus` tested `mcpinstances_test.go:117`). It is NOT the Bifrost 5-state enum (no `Connected`/`Disconnected`/`PendingTools`/`Disabled`). | **VAR** (g0router HAS a connection-lifecycle status — its own free-form `stopped/starting/running/error` set — not the Bifrost 5-state enum; the lifecycle-status BEHAVIOR exists, the enum shape differs by design). | **flip+regression-test** — see §0.3: the free-form status lifecycle (`stopped→starting→running→error` via `SetMCPInstanceStatus`) is locked by `mcpinstances_test.go:117` (status→running) but the FULL set is not asserted; add one small additive store-test case asserting each of the four statuses round-trips, to LOCK the variant lifecycle the flip claims. |
| **076** | **mcp-go client** integration for upstream connections (MISSING) | g0router rolls its OWN bridge/client — no `mark3labs/mcp-go` dep (`! grep -rnE 'mark3labs/mcp-go' go.mod internal/` → only the server.go COMMENTS noting the deliberate absence). The client direction (probe/launcher/process/sse/bridge) is g0router's own code. | **VAR** (the upstream-connection client BEHAVIOR is built over g0router's OWN bridge; the `mark3labs/mcp-go` DEPENDENCY is deliberately NOT added — mirrors row 075's HAVE-by-variant treatment for the server side). | **pure-flip** (the own-bridge client is covered by `bridge_test.go`/`probe_test.go`/`launcher_test.go`; the no-dep invariant is asserted in §5 grep + the existing bf-mcp-1/2 `! grep mcp-go` proof). |
| **080** | ConnectionString stored as `*EnvVar` (encrypted at rest) (MISSING) | g0router has NO `connection_string` column and NO `EnvVar` type; the instance `url`/`command`/`args`/`env` are PLAINTEXT (`mcpinstances.go:148`). What IS `*_enc` at rest is the OAuth **token/verifier** material (`mcpoauth.go:45,49,140 cipher.Encrypt` → `access_token_enc`/`refresh_token_enc`/`verifier_enc`) — a DIFFERENT field set, not a ConnectionString-as-EnvVar. | **VAR (narrow)** (g0router encrypts MCP *secret* material at rest via the `*_enc` precedent — the encryption-at-rest CAPABILITY the row's spirit names exists for OAuth secrets; but the SPECIFIC "`ConnectionString` typed as `*EnvVar`, the whole connection string encrypted" does NOT exist — instance URLs/env are plaintext). Honestly this is a **narrow VAR on the encryption-at-rest capability only; the ConnectionString-as-EnvVar shape is MISSING**. | **pure-flip to VAR with an explicit MISSING-shape note** (no new HAVE claim; `mcpoauth_test.go` already covers the `*_enc` round-trip; no new test). Reviewer option in §1: leave 080 fully **MISSING** if the narrow capability-VAR is judged too generous — see D6. |

**Net (honest):**
- **HAVE (2):** 007 (STDIO command/args/env), 011 (OAuth 2.0).
- **PARTIAL (4):** 001 (client-mode minus InProcess + persistent HTTP/SSE), 005 (HTTP — probe yes, launcher connection deferred), 006 (SSE — same), 021 (tools/list health; ping-fallback toggle absent → see VAR note; recorded PARTIAL/VAR — §1 D2).
- **VAR (4):** 024 (generic agent loop = Chat consumer), 025 (same loop, no Responses entrypoint), 067 (own free-form status lifecycle, not the 5-state enum), 076 (own-bridge client, no mcp-go dep). Plus 080 as a **narrow VAR** (encryption-at-rest capability) with an explicit MISSING-shape note (D6).
- **STILL-MISSING / ESC (6, NOT flipped):** 008 (InProcess), 009 (auth none), 010 (auth headers), 026 (reconnect-state health monitor — g0router's monitor is OAuth-refresh, a different thing), 027 (credential-store interface), 066 (TLS config).

This is materially MORE CONSERVATIVE than the MAP's "flip all 17 to HAVE/PARTIAL." The
MAP §263 over-claimed (it predates this live-tree audit). bf-mcp-sat flips ONLY what
the live tree evidences and leaves the rest MISSING — honest parity, not inflated.

### 0.3 Regression-test decision (per row) — pure-flip vs flip+regression-test

The brief: where a flip asserts a behavior with NO existing regression test, add a
SMALL hermetic additive regression test (no production code) to lock it. Audit:

| Behavior the flip claims | Existing test? | Decision |
|---|---|---|
| 007 STDIO spawn (HAVE) | `launcher_test.go` (14 `StartStdio` hits) | **pure-flip** (locked). |
| 011 OAuth Start/Complete/Refresh (HAVE) | `oauth_test.go` (28 hits) | **pure-flip** (locked). |
| 001/005/006 HTTP/SSE probe + STDIO + transport-mode record (PARTIAL) | `probe_test.go`, `transport_test.go`, `launcher_test.go:172 TestLauncherStartHTTPSSEModes` | **pure-flip** (locked). |
| 021 tools/list health (PARTIAL/VAR) | `probe_test.go` (tools/list path) | **pure-flip** (no new HAVE claim; VAR note records the absent ping toggle). |
| 024/025 agent loop (HAVE-by-variant / VAR) | `agent_test.go` (18 hits) | **pure-flip** (locked; VAR note records the single-loop shape). |
| 076 own-bridge client / no mcp-go (VAR) | `bridge_test.go`/`probe_test.go` + the `! grep mcp-go` invariant | **pure-flip** (locked). |
| 067 free-form status lifecycle (VAR) | `mcpinstances_test.go:117` asserts only `→running`; the FULL `stopped/starting/running/error` set is NOT asserted | **flip+regression-test** — add ONE additive store-test case `TestMCPInstanceStatusLifecycle` asserting each of the four statuses round-trips through `SetMCPInstanceStatus`/`GetMCPInstance`, locking the variant lifecycle the 067 flip claims. Additive test only; NO production change. |
| 080 narrow VAR (encryption-at-rest capability) | `mcpoauth_test.go` covers the `*_enc` token round-trip | **pure-flip to VAR** (the capability is locked by the existing OAuth `*_enc` test; the flip adds an explicit MISSING-shape note for the ConnectionString-as-EnvVar; no new HAVE claim → no new test). |

**Only ONE row (067) needs a flip+regression-test.** Everything else is pure-flip
(already locked) or STAYS MISSING. The 067 test is the ONLY production-adjacent change
and it is a purely additive test function in an existing `_test.go` — no `.go`
production edit anywhere in the plan.

---

## 1. Decisions made (and why) — binding

### D1 — The MAP's "flip all 17" is OVERRIDDEN by the live-tree audit (honesty over the MAP)

BIFROST-MAP §263 listed all 17 as SAT/VAR and said "flip these to HAVE/PARTIAL for the
client direction." The §0 audit proves that claim is OPTIMISTIC: 6 of the 17
(008/009/010/026/027/066) have NO live evidence and STAY MISSING. **The matrix is the
ground truth, and the live tree is the only evidence for a flip** (BIFROST-MAP §60-62).
**Decision:** bf-mcp-sat flips a row ONLY on a cited live-tree file:line; it does NOT
flip a row because the MAP predicted it. The honest count (2 HAVE / 4 PARTIAL / 4+1 VAR
/ 6 MISSING) supersedes the MAP's "17 flips." This is the bf-core-1 discipline (a
near-empty, honest closeout beats a manufactured one).

### D2 — 021 ping-vs-listTools: PARTIAL recorded as VAR (tools/list IS the health signal; the ping-fallback toggle is absent)

The matrix 021 behavior is the ping-VS-listTools FALLBACK with `IsPingAvailable`.
g0router uses `tools/list` as its single health/discovery signal (`probe.go:80`) and
has NO ping method and NO `IsPingAvailable` toggle. **Decision:** flip 021 to
**PARTIAL** with a VAR note — the health-via-tools/list behavior is HAVE-shaped (it IS
the discovery/health probe), but the ping-first-with-listTools-fallback selector is a
Bifrost shape g0router deliberately does not have (no per-client ping toggle). Recorded
config-/shape-VAR; the ping selector is ESC (§3). No new HAVE is claimed; no new test.

### D3 — 024 HAVE-by-variant, 025 VAR: one generic agent loop, not two API-specific entrypoints

`agent.go:84 Agent.Run` is ONE generic bounded loop driven by an injected `ModelStep`.
Bifrost has two entrypoints (`CheckAndExecuteAgentForChatRequest` /
`...ForResponsesRequest`). **Decision:** 024 → **HAVE-by-variant** (the bounded
iterative-tool-exec loop IS built and IS the Chat consumer; the entrypoint shape is
VAR). 025 → **VAR** (the SAME generic loop covers the Responses-API iteration; a
distinct Responses entrypoint does not exist and is not needed under g0router's design).
Neither is inflated to a clean Bifrost-shaped HAVE.

### D4 — 026 STAYS MISSING: g0router's "health monitor" is a DIFFERENT thing (name collision)

g0router HAS a file named `healthmonitor.go`, but it is an OAuth-token-EXPIRY sweeper
(`accountsNeedingRefresh`, `sweep`), NOT the 026 connection-reconnect-on-startup-failure
monitor with a Disconnected-state lifecycle. **Decision (binding honesty):** the name
collision must NOT trick a flip. 026 STAYS **MISSING/ESC** — the reconnect-on-failure
behavior is not built. The matrix note will explicitly say "g0router's healthmonitor.go
is an OAuth-refresh sweeper, not connection reconnect; 026 not built" so no future
reviewer double-counts the OAuth monitor as 026.

### D5 — 008/009/010/027/066 STAY MISSING (no evidence; per-user/enterprise foundation)

InProcess (008), auth-type `none` (009), auth-type static `headers` (010), the
credential-store interface (027), and MCP TLS config (066) have ZERO live evidence
(§0 greps return nothing). 009/010/027 are the typed-auth + credential-store foundation
of the per-user enterprise surface that BIFROST-MAP §269 already escalates. **Decision:**
all five STAY **MISSING/ESC**, recorded in `open-questions.md`. NOT flipped.

### D6 — 080: narrow VAR on the encryption-at-rest capability, with an explicit MISSING-shape note (reviewer may downgrade to MISSING)

g0router encrypts MCP OAuth secrets at rest (`*_enc`, `mcpoauth.go`) — the
encryption-at-rest CAPABILITY the row's spirit names exists for the secret material.
But the SPECIFIC `ConnectionString`-typed-as-`*EnvVar`-encrypted shape does NOT exist
(instance URLs/env are plaintext; there is no ConnectionString column). **Decision:**
flip 080 to **VAR (narrow)** with an EXPLICIT note: "encryption-at-rest exists for OAuth
secrets via `*_enc`; the ConnectionString-as-EnvVar shape is NOT built (instance
url/env plaintext)." **Open-question / reviewer option (recorded §7):** if the parity
council judges the capability-VAR too generous (the row is specifically about the
ConnectionString field), downgrade 080 to **MISSING/ESC**. Default: narrow-VAR with the
MISSING-shape note. Either way the parity count is NOT inflated — VAR is not counted as
HAVE; the MISSING shape is explicit.

### D7 — NO production code; the ONLY change is the 067 additive regression test + the matrix/docs

Per the brief ("NO new code where already satisfied"; consume-only client-mode
primitives, BIFROST-MAP §397). **Decision:** bf-mcp-sat edits NO production `.go` file.
The sole code-adjacent change is ONE additive test function
(`TestMCPInstanceStatusLifecycle` in `internal/store/mcpinstances_test.go`, D-067) that
locks the free-form status lifecycle the 067 VAR flip claims. Adding a regression test
to an existing `_test.go` is additive and forbidden-list-safe (it touches no
`internal/mcp/*.go` production file). Everything else is matrix + `open-questions.md` +
`docs/WORKFLOW.md`.

---

## 2. Target files

### IN-SCOPE — additive TEST only (no production code)

| File | Change (additive TEST ONLY) |
|---|---|
| `internal/store/mcpinstances_test.go` | ADD ONE hermetic additive test `TestMCPInstanceStatusLifecycle` (D-067): create an instance, drive `SetMCPInstanceStatus` through each of `stopped`→`starting`→`running`→`error`, and assert `GetMCPInstance().Status` round-trips each value. Locks the free-form connection-lifecycle the 067 VAR flip claims. Uses the in-memory SQLite test store (hermetic — no network/sleep/subprocess). NO production `.go` change. PRESERVE every existing test. |

### IN-SCOPE — documentation / matrix only (no compile impact)

| File | Change |
|---|---|
| `.planning/parity/matrix/bifrost-mcp.md` | Flip/annotate the 11 evidenced rows per §7 (007/011 → HAVE; 001/005/006/021 → PARTIAL; 024 → HAVE-by-variant; 025/067/076 → VAR; 080 → VAR-narrow). Each flip carries its §0 file:line cite + a `bf-mcp-sat` tag. The 6 MISSING rows (008/009/010/026/027/066) get a one-line "audited: not built — <reason>" annotation but KEEP status MISSING. |
| `.planning/parity/plans/open-questions.md` | APPEND the bf-mcp-sat ESC/MISSING items + the 080-downgrade reviewer option (§7). |
| `docs/WORKFLOW.md` | APPEND the bf-mcp-sat closeout row (§7). |

### FORBIDDEN (automatic REJECT if touched)

- **Any production `.go` file** — bf-mcp-sat ships NO production code. Explicitly
  FORBIDDEN: `internal/mcp/*.go` (consume-only, BIFROST-MAP §397), `internal/store/mcp*.go`
  (the `_test.go` ADDITION for 067 is allowed; the `.go` production file is NOT touched),
  `internal/admin/mcp.go`, `internal/schemas/mcp.go`. No new InProcess transport, no
  auth-type enum, no credential-store interface, no TLS config, no ConnectionString
  column, no reconnect health monitor — those are the MISSING/ESC rows; building any of
  them is OUT OF SCOPE (a separate funded plan, not this closeout).
- **All route files** (`routes_admin.go`, `routes_mcp.go`, `routes_openai.go`) — bf-mcp-sat
  registers NO route and is NOT a serial holder (the MCP serial chain terminated at
  bf-core-2). UNTOUCHED.
- **The `mark3labs/mcp-go` dependency** — NOT added (076 VAR; own bridge).
- **`internal/store/migrate.go`** — NO new column/table (no MISSING row is built).
- **Any flip to HAVE/PARTIAL on a row with no live evidence** — 008/009/010/026/027/066
  STAY MISSING. Flipping any of them is an automatic REJECT (inflated parity).
- **All UI / e2e / mocks** — no UI contract.
- **No `init()`, no global state** — N/A (the one test addition uses the existing test
  store constructor).

---

## 3. Rows left MISSING / ESC (and exactly why)

bf-mcp-sat leaves SIX of the MAP's 17 candidates MISSING — the honest residual. Each is
recorded in `open-questions.md` at close.

| Row | Matrix behavior | Why it STAYS MISSING (live-tree evidence of ABSENCE) |
|---|---|---|
| **008** | Transport: InProcess | `! grep -ri inprocess internal/mcp/*.go` → none. No in-process server wiring; g0router launches external processes or dials HTTP/SSE — InProcess is not modeled. ESC (not applicable to g0router's launcher model). |
| **009** | Auth type: none | No `auth_type` enum exists. There is no typed auth-mode with a `none` member; OAuth is wired directly. The typed-auth enum is the per-user-surface foundation (BIFROST-MAP §269 ESC). |
| **010** | Auth type: headers (static) | No client-config static-auth-header injection on connect (`sse.go` sets only transport headers). The `allowed_extra_headers` whitelist (071) is the SERVER-mode request-forwarding shape, a different row. ESC (per-user/enterprise auth surface). |
| **026** | Health monitor: reconnect on startup failure + Disconnected state | g0router's `healthmonitor.go` is an OAuth-token-refresh sweeper (`accountsNeedingRefresh`/`sweep`), NOT a connection-reconnect monitor with a Disconnected-state lifecycle. The reconnect-on-failure behavior is not built. ESC (name collision recorded so it is not double-counted). |
| **027** | Credential store interface (`ConnectionHeaders`/`RequestHeaders`/`RequiresPerCallConnection`) | `! grep -ri CredentialStore internal/mcp internal/store` → none. No credential-store abstraction. It is the per-user enterprise surface's foundation (BIFROST-MAP §269 ESC). |
| **066** | TLS config (`InsecureSkipVerify`/`CACertPEM`, env-var aware) | `! grep -riE 'InsecureSkipVerify\|CACert\|tls.Config' internal/mcp internal/store/mcp*` → none. The probe/SSE clients use the default `*http.Client`; no per-MCP-client TLS config schema. ESC (additive-buildable later if funded, but not built — not flipped). |

**080 boundary case** (recorded but flipped VAR-narrow, see D6): the encryption-at-rest
CAPABILITY exists (`*_enc` OAuth secrets) but the ConnectionString-as-EnvVar SHAPE does
not. Flipped **VAR (narrow)** with an explicit MISSING-shape note; the reviewer may
downgrade to MISSING (§7 open question). NOT counted as HAVE.

**No-leftovers / no-inflation (binding):** a row is flipped HAVE/PARTIAL/VAR ONLY with a
cited live file:line proving the behavior; the six rows above have a cited ABSENCE proof
and STAY MISSING. NO row is flipped on the MAP's claim alone (D1).

---

## 4. Task graph (TDD; `N. [step] -> verify: [check]`)

bf-mcp-sat ships ONE additive test (067) and otherwise verifies + flips. `go test ./...
&& go vet ./... && go build ./...` green at every commit (the verify steps are the
untouched-green baseline; the 067 test must pass GREEN against the EXISTING production
code — it locks shipped behavior, it does not drive new code).

1. **[P0 baseline]** Record `<base>` = `git rev-parse HEAD` (observed f043592); confirm
   clean tree (`git status --porcelain` empty) and untouched-green
   (`go test ./... && go vet ./... && go build ./...` exit 0). -> verify: exit 0; `<base>`
   recorded.

2. **[VERIFY the HAVE/PARTIAL/VAR evidence — the §0 greps]** Run every §0/§5 grep to
   confirm: 007 STDIO spawn present; 011 OAuth Engine present; 001/005/006 probe+launcher
   modes present; 024/025 `Agent.Run` present; 067 free-form status present; 076 no
   mcp-go dep; AND the ABSENCE proofs for 008/009/010/026/027/066/080-shape. -> verify:
   every "present" grep non-empty AND every "absent" grep empty (the §5 block). This is
   the gate that the flips match the live tree.

3. **[067 regression test, GREEN against existing code]** Add
   `TestMCPInstanceStatusLifecycle` to `internal/store/mcpinstances_test.go` (D-067):
   asserts `stopped/starting/running/error` round-trip through
   `SetMCPInstanceStatus`/`GetMCPInstance`. -> verify: `go test ./internal/store/ -run
   MCPInstanceStatusLifecycle -v` GREEN against the existing `mcpinstances.go` (locks the
   shipped lifecycle — no production change); `go test ./... && go vet ./... && go build
   ./...` exit 0; hermetic (`! grep -nE 'time.Sleep|net.Dial' internal/store/mcpinstances_test.go`).
   Commit: `phase-1/bf-mcp-sat: regression test locking MCP instance status lifecycle (067 VAR)`.

4. **[matrix flip + docs]** Apply §7 flips/annotations to `bifrost-mcp.md` (11 evidenced
   rows flipped with cites; 6 MISSING rows annotated "audited: not built", status
   KEPT MISSING); append `open-questions.md` + `docs/WORKFLOW.md`. -> verify: §6 green;
   the changed files are EXACTLY `internal/store/mcpinstances_test.go`,
   `.planning/parity/matrix/bifrost-mcp.md`, `.planning/parity/plans/open-questions.md`,
   `docs/WORKFLOW.md` (`git diff --name-only <base>..HEAD`); NO production `.go` file
   changed (`! git diff --name-only <base>..HEAD | grep -E 'internal/.*[^_]\.go$' | grep -v _test`).
   Commit:
   `phase-1/bf-mcp-sat: close — bifrost-mcp client-mode SAT audit; honest matrix flip (2 HAVE / 4 PARTIAL / 5 VAR / 6 stay MISSING); no production code`.

---

## 5. Acceptance criteria (binary; file:line / grep proofs)

**Test gates** (each yes/no, exit 0; HERMETIC):
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/store/ -run MCPInstanceStatusLifecycle -v` → GREEN (locks 067).

**No-production-code proof (PRIMARY acceptance — matches bf-core-1):**
```bash
# Only an additive TEST + matrix/docs changed. NO production .go file.
git diff --name-only <base>..HEAD    # EXACTLY:
#   internal/store/mcpinstances_test.go
#   .planning/parity/matrix/bifrost-mcp.md
#   .planning/parity/plans/open-questions.md
#   docs/WORKFLOW.md
! git diff --name-only <base>..HEAD | grep -E 'internal/.*\.go$' | grep -v '_test\.go$' && echo "no production .go changed OK"
# the ONLY .go touched is a _test.go
git diff --name-only <base>..HEAD | grep -E 'internal/.*\.go$'   # → only internal/store/mcpinstances_test.go
```

**Every FLIPPED row has cited live evidence (the flips are honest):**
```bash
# 007 HAVE — STDIO spawn with command/args/env
grep -nE 'func \(l \*Launcher\) StartStdio' internal/mcp/launcher.go            # :54
grep -nE 'exec\.Command\(spec\.Command|cmd\.Env = mergeEnv' internal/mcp/process.go  # :34-35
# 011 HAVE — OAuth engine Start/Complete/Refresh
grep -nE 'func \(e \*Engine\) (Start|Complete|Refresh)' internal/mcp/oauth.go  # :46/:88/:128
# 001/005/006 PARTIAL — probe dial + transport modes
grep -nE 'method":"initialize"|method":"tools/list"' internal/mcp/probe.go
grep -nE 'func \(l \*Launcher\) (StartHTTP|StartSSE)' internal/mcp/launcher.go  # :90/:96
# 024/025 VAR/HAVE-variant — generic agent loop
grep -nE 'func \(a \*Agent\) Run|ErrMaxTurnsExceeded|maxTurns' internal/mcp/agent.go
# 067 VAR — free-form status lifecycle (+ the new lock)
grep -nE "'stopped'|status = \"stopped\"|SetMCPInstanceStatus" internal/store/mcpinstances.go
grep -nE 'func TestMCPInstanceStatusLifecycle' internal/store/mcpinstances_test.go
# 076 VAR — own bridge, no mcp-go dep
! grep -rnE 'mark3labs/mcp-go' go.mod internal/ --include='*.go' | grep -v '//' && echo "no mcp-go dep (076 VAR) OK"
# 080 VAR-narrow — *_enc encryption-at-rest exists (OAuth secrets); ConnectionString shape absent
grep -nE 'access_token_enc|refresh_token_enc|verifier_enc|cipher\.Encrypt' internal/store/mcpoauth.go
! grep -nE 'connection_string|ConnectionString|EnvVar' internal/store/mcpinstances.go && echo "no ConnectionString-as-EnvVar shape (080 MISSING-shape) OK"
```

**Every MISSING row that STAYS MISSING has a cited ABSENCE proof (no inflation):**
```bash
! grep -rniE 'inprocess|in_process' internal/mcp/*.go && echo "008 InProcess absent OK"
! grep -rniE 'auth_type|AuthType|MCPAuthType' internal/store/mcp*.go internal/schemas/mcp.go internal/mcp/*.go && echo "009/010 auth-type enum absent OK"
# 026: the healthmonitor is OAuth-refresh, NOT reconnect-state (name collision honesty)
grep -nE 'accountsNeedingRefresh|accountHealth' internal/mcp/healthmonitor.go      # it IS the OAuth sweeper
! grep -rniE 'Reconnect|Disconnected|PendingTools' internal/mcp/healthmonitor.go && echo "026 reconnect-state monitor absent OK"
! grep -rniE 'CredentialStore|ConnectionHeaders|RequiresPerCall' internal/mcp/*.go internal/store/mcp*.go && echo "027 credential-store interface absent OK"
! grep -rniE 'InsecureSkipVerify|CACert|tls\.Config|x509' internal/mcp/*.go internal/store/mcp*.go && echo "066 TLS config absent OK"
# the matrix keeps these MISSING (annotated, not flipped)
grep -nE 'PAR-BF-MCP-(008|009|010|026|027|066)' .planning/parity/matrix/bifrost-mcp.md | grep -iE 'MISSING' # all six still MISSING
```

**Hermetic proof (the one new test):**
```bash
! grep -nE 'time\.Sleep|net\.Dial|exec\.Command' internal/store/mcpinstances_test.go && echo "067 test hermetic OK"
```

**Behavioral acceptance (binary):**
- bf-mcp-sat introduces NO production Go code (proven by the no-production-code grep).
- Exactly 11 rows flipped (007/011 HAVE; 001/005/006/021 PARTIAL; 024 HAVE-variant;
  025/067/076 VAR; 080 VAR-narrow), each with a cited live file:line.
- Exactly 6 rows STAY MISSING (008/009/010/026/027/066), each with a cited ABSENCE.
- The 067 status lifecycle is locked by a passing hermetic regression test.

---

## 6. Validation commands

```bash
go test ./... && go vet ./... && go build ./...     # exit 0
go test ./internal/store/ -run MCPInstanceStatusLifecycle -v    # GREEN (locks 067)
# no-production-code proof (§5)
! git diff --name-only <base>..HEAD | grep -E 'internal/.*\.go$' | grep -v '_test\.go$' && echo "matrix+test-only (no production .go) OK"
# the flips are evidence-backed + the MISSING rows stay MISSING (§5 grep blocks)
```
No UI build / Playwright needed — bf-mcp-sat ships NO UI touch and NO mock correction.
The single added test is hermetic (in-memory SQLite store; no net/sleep/subprocess).

---

## 7. Freeze rules + the exact matrix-flip list + open-questions + WORKFLOW + no-leftovers

### Freeze rules (binding)

- **Wave-7 MCP CLIENT-mode is consume-only** (BIFROST-MAP §397). bf-mcp-sat edits NO
  `internal/mcp/*.go` production file; it only ADDS one regression test to an existing
  `_test.go` in `internal/store/`.
- bf-mcp-sat is **NOT a routes serial holder** — it registers no route. The
  routes_admin.go / routes_mcp.go MCP serial chain TERMINATED at bf-core-2
  (BIFROST-MAP §343-346); bf-mcp-sat adds nothing to it.
- **No double-flip** (binding): rows already addressed by bf-mcp-1/bf-mcp-2
  (002/003/004/017/018/019/020/032/033/039/049/052/053/054/057/071/075/077/078/079) are
  NOT touched — their dispositions stand. bf-mcp-sat touches ONLY the 17 MAP-routed
  client-mode candidates, of which it flips 11 and leaves 6 MISSING.
- **No reverse-engineering of the absent Bifrost ref** (ESC-REF-ABSENT) — every flip is
  grounded in a g0router live file:line, never a guessed Bifrost shape.
- **No inflation** — a row is flipped ONLY on cited live evidence; the 6 MISSING rows
  keep MISSING with a cited absence (D1/§3).

### Exact matrix-flip list (row → new status; applied at close in `bifrost-mcp.md`)

| Row | MISSING → | Evidence cite (the flip's basis) |
|---|---|---|
| PAR-BF-MCP-001 | **PARTIAL** | STDIO live (`launcher.go:54`) + HTTP/SSE probe (`probe.go:55`); InProcess absent, persistent HTTP/SSE launcher deferred. |
| PAR-BF-MCP-005 | **PARTIAL** | HTTP probe dial (`probe.go:55`) + transport mode (`launcher.go:90`); persistent launcher connection deferred (w7-mcp-2 no-op `recordInstance`). |
| PAR-BF-MCP-006 | **PARTIAL** | SSE client (`sse.go`) + SSE parse (`probe.go:142`) + mode (`launcher.go:96`); persistent launcher connection deferred. |
| PAR-BF-MCP-007 | **HAVE** | STDIO command/args/env live spawn (`launcher.go:54` → `process.go:34-35`). |
| PAR-BF-MCP-011 | **HAVE** | OAuth 2.0 PKCE Engine (`oauth.go:46/88/128`; tokens `*_enc` `mcpoauth.go:45`). |
| PAR-BF-MCP-021 | **PARTIAL** (VAR note) | tools/list health signal (`probe.go:80`); ping-vs-listTools fallback + `IsPingAvailable` toggle absent (VAR). |
| PAR-BF-MCP-024 | **HAVE-by-variant** | generic bounded agent loop `Agent.Run` (`agent.go:84`); one generic loop, not a Chat-specific entrypoint (VAR). |
| PAR-BF-MCP-025 | **VAR** | same `Agent.Run` covers Responses iteration; no separate Responses entrypoint. |
| PAR-BF-MCP-067 | **VAR** | own free-form status lifecycle `stopped/starting/running/error` (`mcpinstances.go:34`, locked by new `TestMCPInstanceStatusLifecycle`); not the Bifrost 5-state enum. |
| PAR-BF-MCP-076 | **VAR** | own-bridge client (`bridge.go`/`probe.go`); no `mark3labs/mcp-go` dep (mirrors 075). |
| PAR-BF-MCP-080 | **VAR (narrow)** | `*_enc` encryption-at-rest for OAuth secrets (`mcpoauth.go:45,49,140`); ConnectionString-as-EnvVar shape NOT built (instance url/env plaintext) — MISSING-shape note. |
| PAR-BF-MCP-008 | **stays MISSING** (annotate) | InProcess not modeled (no evidence). ESC. |
| PAR-BF-MCP-009 | **stays MISSING** (annotate) | no auth-type enum; `none` mode does not exist. ESC. |
| PAR-BF-MCP-010 | **stays MISSING** (annotate) | no client-mode static-header auth on connect. ESC. |
| PAR-BF-MCP-026 | **stays MISSING** (annotate) | `healthmonitor.go` is an OAuth-refresh sweeper, NOT connection-reconnect-with-Disconnected-state. ESC. |
| PAR-BF-MCP-027 | **stays MISSING** (annotate) | no credential-store interface. ESC (per-user surface foundation). |
| PAR-BF-MCP-066 | **stays MISSING** (annotate) | no MCP TLS config schema. ESC. |

### `open-questions.md` (append at close)

```
## bf-mcp-sat — bifrost-mcp client-mode SAT audit — 2026-06-15
- [ ] PAR-BF-MCP-008 InProcess transport — MISSING/ESC. Not modeled; g0router launches external processes or dials HTTP/SSE. Build only if an in-process MCP server use-case is funded.
- [ ] PAR-BF-MCP-009/010 auth-type none/headers — MISSING/ESC. No typed auth-mode enum; OAuth is wired directly. The typed-auth enum + static-header injection are the per-user/enterprise auth surface foundation (BIFROST-MAP §269). — what would consume a typed auth-mode?
- [ ] PAR-BF-MCP-026 health monitor reconnect-on-failure + Disconnected state — MISSING/ESC. NOTE: g0router's internal/mcp/healthmonitor.go is an OAuth-TOKEN-REFRESH sweeper (accountsNeedingRefresh/sweep), NOT a connection-reconnect monitor — do NOT double-count it as 026. The reconnect-on-startup-failure behavior is not built.
- [ ] PAR-BF-MCP-027 credential-store interface (ConnectionHeaders/RequestHeaders/RequiresPerCallConnection) — MISSING/ESC. No abstraction; foundation of the per-user enterprise surface (BIFROST-MAP §269).
- [ ] PAR-BF-MCP-066 MCP TLS config (InsecureSkipVerify/CACertPEM) — MISSING/ESC. Probe/SSE use the default *http.Client; additive-buildable later if funded, not built here.
- [ ] PAR-BF-MCP-080 ConnectionString-as-EnvVar encryption — flipped VAR-narrow (encryption-at-rest exists via *_enc for OAuth secrets; the ConnectionString-as-EnvVar shape is NOT built — instance url/env plaintext). REVIEWER OPTION: downgrade 080 to MISSING/ESC if the capability-VAR is judged too generous (the row is specifically about the ConnectionString field). Default: VAR-narrow with MISSING-shape note. Either way 080 is NOT counted as HAVE.
- [ ] PAR-BF-MCP-001/005/006 client-mode transports — PARTIAL. The persistent HTTP/SSE launcher CONNECTION is a no-dial placeholder (launcher.go:103 recordInstance; w7-mcp-2 deferred the live dial); the probe path is real. Promote to HAVE only if/when the launcher live HTTP/SSE connection is funded. InProcess (008) stays separate-MISSING.
- [ ] PAR-BF-MCP-021 ping-vs-listTools fallback — PARTIAL/VAR. g0router uses tools/list as the single health signal; the IsPingAvailable ping-first-fallback selector is absent (VAR). — does any consumer need a ping toggle?
- [ ] PAR-BF-MCP-024/025 agent loop — HAVE-by-variant / VAR. One generic bounded Agent.Run, not two API-specific entrypoints (CheckAndExecuteAgentFor{Chat,Responses}Request). The iterative-tool-exec behavior is met; the dual-entrypoint shape is VAR.
- [ ] PAR-BF-MCP-067 connection state machine — VAR. g0router uses a free-form stopped/starting/running/error status (locked by TestMCPInstanceStatusLifecycle), not the Bifrost 5-state Connected/Disconnected/Error/PendingTools/Disabled enum.
- [ ] PAR-BF-MCP-076 mcp-go client — VAR. Own bridge; mark3labs/mcp-go deliberately not added (mirrors 075).
- [ ] MAP §263 over-claim recorded: the MAP listed all 17 as SAT/VAR "flip to HAVE/PARTIAL"; the live-tree audit (bf-mcp-sat §0) found 6 (008/009/010/026/027/066) have no evidence and stay MISSING. Parity count NOT inflated.
```

### `docs/WORKFLOW.md` (update at close)

Add a bf-mcp-sat row — bifrost-mcp client-mode SAT verification closed as
**matrix-flip-only + one additive regression test** (NO production code): audited the 17
MAP-routed client-mode candidates against the LIVE Wave-7 tree; honestly flipped 11
(007/011 → HAVE; 001/005/006/021 → PARTIAL; 024 → HAVE-by-variant; 025/067/076 → VAR;
080 → VAR-narrow) each with a cited file:line, and left 6 MISSING/ESC
(008/009/010/026/027/066) with cited absence proofs — the MAP §263 "flip all 17"
over-claim corrected, parity NOT inflated. Locked the 067 free-form status lifecycle
with `TestMCPInstanceStatusLifecycle`. No `internal/mcp/*.go` production edit
(consume-only, BIFROST-MAP §397); not a routes serial holder (MCP chain terminated at
bf-core-2). ESC-REF-ABSENT honored. **bifrost-mcp has no further buildable rows** — the
residual is the ESC per-user/enterprise surface.

### No-leftovers confirmation (binding)

bf-mcp-sat adds NO production code, NO route, NO column, NO dead surface. The ONE
additive test (`TestMCPInstanceStatusLifecycle`) locks an ALREADY-shipped behavior (the
067 VAR flip's basis) — it is not new functionality. Every flipped row (11) cites a live
file:line; every MISSING row (6) cites a live absence. No row is flipped on the MAP's
prediction (D1); no behavior is inflated to HAVE that the tree does not evidence
(008/009/010/026/027/066 stay MISSING; 080 is VAR-narrow, not HAVE). A near-empty,
honest, matrix-flip-plus-one-regression-test closeout is the correct outcome — the
bf-core-1 discipline applied to the final bifrost-mcp unit.
```
