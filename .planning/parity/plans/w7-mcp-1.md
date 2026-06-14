# Micro-plan w7-mcp-1 — MCP foundation (store + launcher + stdio↔SSE bridge) (Go)

```
wave: 7
plan: w7-mcp-1
status: READY (rev 1 — authored against merged Waves 0–6 + the SHIPPED W7
  platform plans, using the SHIPPED w7-plat-2 (tunnels) as the PRIMARY template:
  it solved the SAME hard problem — external-process management behind an
  injectable runner for hermetic unit tests + a thin integration-only real spawn.
  Its files are LIVE in-tree and CONSUMED here as the pattern of record:
  internal/platform/tunnel/{runner,service,cloudflared}.go + service_test.go +
  the SHIPPED `tunnels` table @ migrate.go:205 + the SetTunnelRunner injection
  seam @ handlers.go:98-101 — this plan REUSES that exact philosophy for the MCP
  launcher/process. live tree @ <base>; WAVE-7-MAP w7-mcp-1 row ~line 180; MCP
  track split §109-110; serial chain §219-224 (mcp-1 holds NO routes_admin slot);
  reconciliation §245; freeze rules §267.)
runs: MCP track. GREENFIELD `internal/mcp` — disjoint from every other domain/store/
  admin file; runs ∥ governance + providers + platform tracks. INTERNALLY SERIAL:
  w7-mcp-1 (THIS — foundation) ──▶ w7-mcp-2 (client/probe/registry/OAuth engine) ──▶
  w7-mcp-3 (admin transport + routes + tools). mcp-2 and mcp-3 DEPEND on this
  foundation's constructors; mcp-3 is the ONLY MCP plan that takes the
  routes_admin.go serial slot (MAP §208). THIS plan takes NO serial slot.
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w7-mcp-1:
ref-source: 9router frozen @ 827e5c3 — the stdio↔SSE bridge + plugin allowlist.
  Authoritative ref files (cited from `.planning/parity/matrix/9router-mcp.md`):
    src/lib/mcp/stdioSseBridge.js (spawn @146, broadcast @151, smartFilterText @20,
      stderr-log @165, exit-delete @167, broken-pipe-ignore @160, isRunning @193,
      registerCustomPlugin allowlist gate @119, customPlugins.json persist @130);
    src/shared/constants/coworkPlugins.js (ALLOWLIST @67, defaults Exa/Tavily @3,
      browsermcp-via-npx local plugin @26).
  NO W6 UI/e2e mock contract binds this plan: the MCP foundation has NO HTTP
  surface (PAR-MCP-001/002 SSE/message routes = w7-mcp-2/3; the w6-l MCP mocks are
  reconciled by w7-mcp-3 — MAP §182). This plan ships PURE Go consumed by mcp-2/3.
base: <base> = git rev-parse HEAD recorded at P0. Substitute the actual SHA
  everywhere §5 says <base>.
go-serial-slot: NONE. This plan does NOT edit internal/server/routes_admin.go
  (foundation only — confirmed §1.9 / §6 / §7). The routes_admin.go MCP edit lands
  in w7-mcp-3 (MAP §182, §208). No selection.go / factory.go / runner.go (inference)
  micro-serial either — MCP is a standalone gateway subsystem, not an inference-path
  concern.
new-route: NONE. NO UI route files, NO admin handler, NO `/api/mcp/*` registration.
  Foundation only.
```

---

## 1. Scope — PAR rows + the three subsystems

### Rows this plan closes

| Row / item | Claim (from `9router-mcp.md`) | Target state after w7-mcp-1 |
|---|---|---|
| PAR-MCP-032 | DB has no MCP tables (`migrate.go:11`) | HAVE (additive `mcp_instances`, `mcp_clients`, `mcp_oauth_accounts`, `mcp_oauth_flows` tables) |
| PAR-MCP-035 | No MCP store layer (`internal/store/`) | HAVE (`internal/store/mcpinstances.go` + `internal/store/mcpoauth.go`; OAuth tokens `*_enc`) |
| PAR-MCP-036 | No MCP launcher (`internal/mcp/`) — no `command`/`npx`/`http` launch types | HAVE (launcher with stdio/HTTP/SSE modes behind an injectable runner; real spawn integration-only) |
| PAR-MCP-003 | Stdio↔SSE bridge spawns one child process per plugin on demand (`stdioSseBridge.js:146`) | HAVE (launcher spawns one process per plugin via the injectable runner; real `os/exec` integration-only) |
| PAR-MCP-004 | Plugin command allowlist restricts spawnable binaries (`coworkPlugins.js:67`) | HAVE (PURE `isAllowedCommand`; allowlist = `npx,node,uvx,python,python3,bunx,bun`; FULLY unit-tested incl. rejections) |
| PAR-MCP-005 | Custom plugin registration validates command against allowlist (`stdioSseBridge.js:119`) | HAVE (the launcher's `RegisterCustomPlugin`/`StartStdio` rejects a non-allowlisted command BEFORE any spawn; unit-tested) |
| PAR-MCP-006 | Custom plugins persist to disk and survive restart (`stdioSseBridge.js:130`) | HAVE (persisted to the `mcp_clients` table via the store — DB, not `customPlugins.json` — ESC-PERSIST) |
| PAR-MCP-007 | JSON-RPC stdout frames broadcast to all active SSE sessions (`stdioSseBridge.js:151`) | HAVE (PURE frame-split + broadcast over a session map; FULLY unit-tested with canned frames + fake sinks) |
| PAR-MCP-008 | Tool result text filter: drop noise nodes, collapse repeated siblings, 50K truncate (`stdioSseBridge.js:20`) | HAVE (PURE `smartFilterText` + `collapseRepeated` + 50_000-char hard-truncate; FULLY unit-tested) |
| PAR-MCP-051 | `isRunning` checks bridge process alive (`stdioSseBridge.js:193`) | HAVE (`IsRunning()` on the process abstraction: not-killed && exitCode==nil; unit-tested via fake) |
| PAR-MCP-052 | Bridge logs child stderr (`stdioSseBridge.js:165`) | HAVE (stderr drained + logged; the drain-callback wiring is unit-tested via fake; real pipe integration-only) |
| PAR-MCP-053 | Bridge deletes store entry on child exit (`stdioSseBridge.js:167`) | HAVE (exit handler removes the bridge from the registry; unit-tested via a fake exit signal) |
| PAR-MCP-054 | Broken SSE sessions silently ignored on send failure (`stdioSseBridge.js:160`) | HAVE (a failing session sink is dropped, never aborts the broadcast; unit-tested) |
| PAR-MCP-043 | Default plugins: Exa (HTTP, no auth), Tavily (HTTP, OAuth) (`coworkPlugins.js:3`) | HAVE (default plugin DEFINITIONS as Go constants/seed consumed by mcp-2/3; no live HTTP here) |
| PAR-MCP-044 | Local stdio plugin: browsermcp via npx (`coworkPlugins.js:26`) | HAVE (preset stdio plugin definition as a Go constant; allowlisted command `npx`) |
| PAR-MCP-033 (store half) | No MCP API handlers — `handlers/mcp.go`, `handlers/mcpoauth.go` absent | PARTIAL→ (store + OAuth-account persistence land here; the HANDLERS are w7-mcp-3 — this plan supplies the store the handlers consume) |

Matrix flips at closeout (§4 T-close): in `.planning/parity/matrix/9router-mcp.md`,
PAR-MCP-003,004,005,006,007,008,032,035,036,043,044,051,052,053,054 → HAVE (real Go;
the real `os/exec` spawn / real binary download / live HTTP-SSE client are
integration-only behind the runner — §1.9). PAR-MCP-033 stays MISSING/PARTIAL with a
footnote "store + OAuth-account persistence shipped w7-mcp-1; admin handlers
w7-mcp-3". Append new open items (§8) to `open-questions.md`.

### 1.1 Preconditions already satisfied by merged waves (evidence — cite file:line)

- **`internal/mcp` is a Phase-1 placeholder — greenfield to build out.**
  `internal/mcp/doc.go:1-4` documents the package intent ("MCP gateway … bridge for
  agentic workflows") with NO code. `internal/mcp/mcp_test.go:7` is a no-op
  `TestPackageCompiles` placeholder explicitly slated for replacement ("real MCP
  gateway tests arrive in Phase 12+"). This plan replaces that no-op with real
  hermetic tests (or adds new test files and removes the placeholder).
- **The MCP types ALREADY EXIST (consume — do NOT redefine).**
  `internal/schemas/mcp.go` (PAR-MCP-030 = HAVE per `9router-mcp.md:40`):
  - `MCPClient{ID,Name,Type string; Config map[string]any}` (`mcp.go:4-9`).
  - `MCPInstance{ID,ClientID,Name,Transport,URL,Command string; Args []string;
    Env map[string]string; Status string}` (`mcp.go:12-22`).
  - `MCPTool{Name,Description string; InputSchema map[string]any}` (`mcp.go:25-29`).
  - `MCPToolGroup{ID,Name string; ToolNames []string}` (`mcp.go:32-36`).
  The store records MIRROR these field shapes (§1.3). `Transport` ∈ {stdio, http,
  sse}; `Command`/`Args`/`Env` drive the stdio launcher; `URL` drives HTTP/SSE modes.
- **The injectable-runner pattern is SHIPPED and CONSUMED VERBATIM (w7-plat-2).**
  This is the central reuse — the SAME hard problem (external process behind an
  injectable runner for hermetic tests + thin integration-only real spawn):
  - `internal/platform/tunnel/runner.go:15-23` — `type Runner interface
    {Start(StartOpts)(string,error); Stop() error; Status()(RunnerStatus,error)}`:
    "real impl shells out … test impl is a deterministic fake … so the … state
    machine … are unit-tested WITHOUT spawning any process or touching the network."
  - `internal/platform/tunnel/service.go:24-46` — the `Service` holds
    `runners map[string]Runner` as a FIELD; `NewService(st)` constructs REAL defaults
    (`newCloudflaredRunner()`, `:36`); `SetRunner(typ, r)` (`:44-46`) overrides for
    tests. **This is the EXACT shape the MCP launcher copies (§1.4).**
  - `internal/platform/tunnel/cloudflared.go:18-26,33-54` — the PURE helpers
    (`extractQuickTunnelURL`, `isValidExecutable`) are factored OUT of the
    integration-only spawn (`:75-128` `os/exec` bodies, "INTEGRATION-ONLY — not
    exercised by unit tests"). **The MCP bridge's frame-split/broadcast/filter are
    the analogous PURE core (§1.5/§1.6).**
  - `internal/platform/tunnel/service_test.go:14-45` — `fakeRunner` implements
    `Runner` with NO process; the whole state machine is unit-tested through it.
    **The MCP `_test.go` files use the IDENTICAL fake-process pattern (§4).**
- **Secret-at-rest precedent (`*_enc`) — REUSE for OAuth tokens.**
  - `internal/store/oauthsessions.go:20-33,37-62` — `s.cipher.Encrypt(o.Verifier)`
    on write into `verifier_enc`; `s.cipher.Decrypt(verifierEnc)` on read. The
    in-flight OAuth state row pattern this plan mirrors for `mcp_oauth_flows`.
  - `internal/store/proxypools.go:30-33,97-100,149-165` — `s.cipher.Encrypt/Decrypt`
    round for `password_enc`; `scanProxyPool` decrypts on read; the masked-read DTO
    discipline. The MCP OAuth **access/refresh tokens** follow this for
    `access_token_enc`/`refresh_token_enc` (§1.3). The SHIPPED `tunnels` table's
    `token_enc` (`migrate.go:210`) is the freshest sibling.
- **Additive migrations only.** `internal/store/migrate.go:14-18` — the `tables
  []struct{name,create string}` slice with `CREATE TABLE IF NOT EXISTS`
  (`:11-13` "Migrations are additive-only … never modified or dropped"). New columns
  via `ensureColumn(db, table, column, decl)` (`:349-351`, ALTER ... ADD COLUMN). The
  SHIPPED `proxy_pools` (`:191`) + `tunnels` (`:205`) blocks are the immediate
  precedents for appending four `mcp_*` tables the same way.
- **Store helpers to reuse (do NOT re-implement).**
  `internal/store/store.go:14` `var ErrNotFound`; `store.go:71` `func newID()`;
  `internal/store/providers.go:96` `type rowScanner`; `:114` `func boolToInt`;
  `:121` `func requireRowAffected`. `s.cipher.Encrypt/Decrypt` for `*_enc`.
- **Store test harness precedent.** `internal/platform/tunnel/service_test.go:47-60`
  `newServiceTestStore` = `store.LoadOrCreateSecret(dir)` + `store.Open(filepath.
  Join(dir,"test.db"), secret)` + `t.Cleanup(st.Close)`. The MCP store/launcher
  tests use the SAME temp-store helper. (The admin harness
  `internal/admin/admin_test.go:24 newTestEnv` / `:72 call` is NOT needed — this plan
  ships NO admin handler.)

### 1.2 No UI / mock contract binds this plan (binding — confirm)

The MCP foundation has **no HTTP surface**: PAR-MCP-001/002 (SSE/message routes) are
w7-mcp-2/3; the w6-l MCP UI mocks (`ui/e2e/mocks/handlers/mcp.ts` + skills) are
reconciled by **w7-mcp-3** (MAP §182). Therefore w7-mcp-1:
- adds **NO** `internal/admin/*` handler, **NO** `internal/server/routes_admin.go`
  edit, **NO** `ui/**` touch (src, mocks, seeds, specs), **NO** e2e.
- exposes ONLY exported Go constructors/functions consumed by w7-mcp-2 (probe/
  registry/OAuth engine over the bridge transport) and w7-mcp-3 (admin handlers +
  routes + tools). The acceptance is Go-test-only (no playwright — §5).

### 1.3 MCP store Go contract (NEW, TDD) — four additive tables

Four additive tables in the `migrate.go` `tables` slice (mirror the SHIPPED
`proxy_pools`/`tunnels` blocks @ `:191`/`:205`). **DECIDE typed-vs-JSON columns
(§8 ESC-SCHEMA).** RECOMMENDED default = **typed columns for fixed-shape fields,
a JSON `*_json` column for the free-form bits** (`MCPClient.Config map[string]any`,
`MCPInstance.Args []string`, `MCPInstance.Env map[string]string`) — matches the
SHIPPED precedent (`alert_channels.config_enc`/`events_json` @ `migrate.go:186-187`).
OAuth access/refresh tokens are `*_enc` at rest (mirror `tunnels.token_enc` /
`connections.access_token_enc`/`refresh_token_enc` @ `migrate.go:52-53`):

```sql
-- PAR-MCP-006: custom plugins persist + survive restart (DB, not customPlugins.json — ESC-PERSIST)
CREATE TABLE IF NOT EXISTS mcp_clients (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  type        TEXT NOT NULL DEFAULT '',   -- 'default'|'custom' (registration origin)
  config_json TEXT NOT NULL DEFAULT '{}', -- MCPClient.Config map[string]any
  created_at  INTEGER NOT NULL DEFAULT 0,
  updated_at  INTEGER NOT NULL DEFAULT 0
);

-- a running/registered MCP server instance (mirrors schemas.MCPInstance)
CREATE TABLE IF NOT EXISTS mcp_instances (
  id         TEXT PRIMARY KEY,
  client_id  TEXT NOT NULL DEFAULT '',
  name       TEXT NOT NULL,
  transport  TEXT NOT NULL DEFAULT 'stdio', -- 'stdio'|'http'|'sse'
  url        TEXT NOT NULL DEFAULT '',       -- http/sse modes
  command    TEXT NOT NULL DEFAULT '',       -- stdio mode (allowlist-validated)
  args_json  TEXT NOT NULL DEFAULT '[]',     -- MCPInstance.Args []string
  env_json   TEXT NOT NULL DEFAULT '{}',     -- MCPInstance.Env map[string]string
  status     TEXT NOT NULL DEFAULT 'stopped',-- 'stopped'|'starting'|'running'|'error'
  created_at INTEGER NOT NULL DEFAULT 0,
  updated_at INTEGER NOT NULL DEFAULT 0
);

-- a persisted OAuth account for an MCP server (tokens ENCRYPTED at rest)
CREATE TABLE IF NOT EXISTS mcp_oauth_accounts (
  id                TEXT PRIMARY KEY,
  instance_id       TEXT NOT NULL DEFAULT '',
  server_url        TEXT NOT NULL DEFAULT '',
  access_token_enc  TEXT NOT NULL DEFAULT '', -- ENCRYPTED; NEVER echoed
  refresh_token_enc TEXT NOT NULL DEFAULT '', -- ENCRYPTED; NEVER echoed
  expires_at        INTEGER NOT NULL DEFAULT 0,
  scope             TEXT NOT NULL DEFAULT '',
  status            TEXT NOT NULL DEFAULT '',  -- 'connected'|'expired'|'error' (no secrets)
  created_at        INTEGER NOT NULL DEFAULT 0,
  updated_at        INTEGER NOT NULL DEFAULT 0
);

-- an in-flight MCP OAuth authorization (PKCE verifier ENCRYPTED; mirrors oauth_sessions)
CREATE TABLE IF NOT EXISTS mcp_oauth_flows (
  state         TEXT PRIMARY KEY,
  instance_id   TEXT NOT NULL DEFAULT '',
  server_url    TEXT NOT NULL DEFAULT '',
  verifier_enc  TEXT NOT NULL DEFAULT '', -- PKCE verifier ENCRYPTED; NEVER echoed
  redirect_uri  TEXT NOT NULL DEFAULT '',
  expires_at    INTEGER NOT NULL DEFAULT 0,
  created_at    INTEGER NOT NULL DEFAULT 0
);
```

**`internal/store/mcpinstances.go` (NEW)** — `MCPClient`-ish + `MCPInstance`-ish
store records + methods (JSON-marshal the `map`/`[]string` fields into the `*_json`
columns; reuse `newID`, `boolToInt`, `time.Now().Unix()`, `ErrNotFound`,
`requireRowAffected`):
- clients: `CreateMCPClient`, `GetMCPClient(id)`, `ListMCPClients()`,
  `UpsertMCPClient`, `DeleteMCPClient(id)`.
- instances: `CreateMCPInstance`, `GetMCPInstance(id)`, `ListMCPInstances()`,
  `UpdateMCPInstance`, `SetMCPInstanceStatus(id, status)`, `DeleteMCPInstance(id)`.

**`internal/store/mcpoauth.go` (NEW)** — OAuth accounts + flows (tokens/verifier via
`s.cipher.Encrypt/Decrypt`, mirroring `oauthsessions.go`/`proxypools.go`):
- accounts: `UpsertMCPOAuthAccount`, `GetMCPOAuthAccount(id)`,
  `GetMCPOAuthAccountByInstance(instanceID)`, `ListMCPOAuthAccounts()`,
  `DeleteMCPOAuthAccount(id)` — tokens decrypted on read, NEVER returned in any DTO
  field a handler will echo (the masked-read discipline — §1.3 secret note).
- flows: `CreateMCPOAuthFlow`, `ConsumeMCPOAuthFlow(state)` (return+delete; expired →
  `ErrNotFound`; verifier decrypted on consume) — a direct analogue of
  `CreateOAuthSession`/`ConsumeOAuthSession` (`oauthsessions.go:20,37`).

**Secret note (binding):** `access_token`/`refresh_token`/`verifier` are plaintext in
memory, `*_enc` at rest. NO store method returns a struct whose token fields are
populated into a JSON-tagged response by a downstream handler; the read records carry
the decrypted token ONLY for the OAuth engine's internal use (w7-mcp-2). The store
unit test asserts the raw `*_enc` column ≠ cleartext (§4 T-mcpoauth, §5 grep proof).

### 1.4 THE CENTRAL DESIGN PROBLEM — injectable launcher/process (binding; COPY the SHIPPED tunnel Runner pattern)

MCP stdio servers are **external binaries that CANNOT be spawned in unit tests**
(AGENTS.md "No mocks; use interfaces and fakes; test real behavior"; 9router's own
ref disables the bridge globally precisely because it "spawns arbitrary processes
(RCE risk)" — `9router-mcp.md:52`/PAR-MCP-042). The launcher, the allowlist gate, the
bridge framing/broadcast/filter, and the process lifecycle (`IsRunning`, exit-delete,
stderr-drain) MUST be unit-testable deterministically with NO process spawn, NO
network, NO port bind, NO sleep-on-a-real-process. The mechanism is COPIED from the
SHIPPED tunnel `Runner`/`SetRunner` seam (`tunnel/runner.go:15`, `service.go:24-46`):

**The process abstraction (NEW — `internal/mcp/runner.go`):**
```go
package mcp

// ProcessRunner abstracts spawning + driving one MCP stdio child process. The
// REAL impl shells out via os/exec (StdinPipe/StdoutPipe/StderrPipe + cmd.Wait);
// the TEST impl is a deterministic in-memory fake (canned stdout frames, a
// controllable exit) — so the launcher, the bridge broadcast, the filter, and the
// lifecycle are unit-tested WITHOUT spawning any process. Mirrors tunnel.Runner
// (tunnel/runner.go:15) and platform.Prober (proxypools.go:18).
type ProcessRunner interface {
    // Start spawns the child for spec (command+args+env). Returns a Process whose
    // stdout frames are delivered to onFrame and whose exit invokes onExit.
    Start(spec ProcessSpec) (Process, error)
}

// Process is one running (or fake) child; the bridge drives it.
type Process interface {
    // Write sends a JSON-RPC frame to the child's stdin (newline-delimited).
    Write(frame []byte) error
    // IsRunning reports liveness: not killed && exit code not yet observed
    // (PAR-MCP-051; mirrors !proc.killed && proc.exitCode === null).
    IsRunning() bool
    // Stop kills the child. Idempotent.
    Stop() error
}

type ProcessSpec struct {
    Command string            // allowlist-validated base name (npx/node/uvx/...)
    Args    []string
    Env     map[string]string
    // OnFrame receives each newline-delimited stdout JSON-RPC frame (PAR-MCP-007).
    OnFrame func(frame []byte)
    // OnStderr receives child stderr lines for logging (PAR-MCP-052).
    OnStderr func(line string)
    // OnExit fires once when the child exits, with its code (PAR-MCP-053).
    OnExit func(code int)
}
```

**The command allowlist (PURE — security boundary — binding, FULLY unit-tested;
NEW in `internal/mcp/allowlist.go`):**
```go
// allowedMCPCommands mirrors 9router's ALLOWED_MCP_COMMANDS (coworkPlugins.js:67).
var allowedMCPCommands = map[string]struct{}{
    "npx": {}, "node": {}, "uvx": {}, "python": {}, "python3": {},
    "bunx": {}, "bun": {},
}

// isAllowedCommand reports whether cmd resolves to an allowlisted launcher. It
// matches on filepath.Base so an absolute path to a permitted binary
// (/usr/local/bin/npx) passes while a relative path (./npx) or a path with shell
// metacharacters is rejected. PURE — no I/O. Mirrors path.basename semantics
// (9router-mcp.md quirk #3) but HARDENS it: reject empty, reject any cmd whose
// base != cmd unless cmd is an absolute clean path, reject shell metacharacters.
func isAllowedCommand(cmd string) bool
```
RECOMMENDED hardening (binding default — §8 ESC-ALLOWLIST): permit iff
`filepath.Base(cmd)` is in the map AND the command contains NO shell metacharacters
(`; | & $ \` > < ( ) newline`) AND is either a bare base name or an **absolute**
clean path (`filepath.IsAbs && filepath.Clean(cmd)==cmd`). This rejects: `rm`,
`bash`, `bash -c`, `sh`, `./npx`, `../npx`, `npx; rm -rf /`, `$(...)`, an absolute
path to an arbitrary binary (`/bin/rm`), and an empty command. (9router's looser
`path.basename` would pass `/bin/rm`'s base `rm` — NO; `rm` isn't in the map — and
would pass `./npx`'s base `npx` — we REJECT that; the hardening is a deliberate
parity improvement, recorded §8.) **This is THE security-critical surface; it is
PURE and exhaustively unit-tested incl. every rejection case (§4 T-allowlist, §5).**

**The launcher (NEW — `internal/mcp/launcher.go`):** holds a `ProcessRunner` FIELD
with a REAL default (`osProcessRunner`) constructed at `NewLauncher`, overridable via
`SetRunner(ProcessRunner)` for tests — EXACTLY as `tunnel.NewService`/`SetRunner`
(`service.go:32-46`). Modes (mirror PAR-MCP-036's `command`/`http` launch types +
PAR-MCP-022's url-vs-command branch):
- **stdio:** `StartStdio(spec)` — FIRST `isAllowedCommand(spec.Command)` → reject
  with a typed error BEFORE any spawn (PAR-MCP-005); then `runner.Start(spec)` →
  hand the `Process` to a new `Bridge` (§1.5). Registers the bridge in the launcher's
  `bridges map[string]*Bridge` under a `sync.RWMutex` (the package-level registry the
  Go-port note §170 prescribes; one bridge per plugin — PAR-MCP-003).
- **HTTP / SSE client modes:** `StartHTTP(url)` / `StartSSE(url)` — record the
  instance's transport+url for the mcp-2 probe/transport; NO live HTTP dial here
  (that is w7-mcp-2). The launcher exposes the URL+headers the mcp-2 client will use.
  (The real HTTP-SSE client is integration-only / mcp-2; this plan only models the
  mode + persists the instance.)
- **lifecycle:** `IsRunning(name)` (PAR-MCP-051 via `Process.IsRunning`); on child
  exit the `OnExit` handler removes the bridge from the registry (PAR-MCP-053);
  stderr lines are drained to a logger callback (PAR-MCP-052). Re-spawn on a new
  request only if the prior process exited (9router quirk #9) — DECIDE re-spawn
  policy at T-launcher (default: re-spawn iff `!IsRunning`).

**Injection (binding — mirror tunnel SetRunner):** `NewLauncher(st *store.Store)`
constructs the REAL `osProcessRunner`; `SetRunner(r ProcessRunner)` overrides it for
tests. The REAL `osProcessRunner` (the ONLY place `os/exec` appears — §1.7) is NOT
exercised by any unit test. There is **NO** `Handlers` wiring and **NO**
`SetTunnelRunner`-style admin setter in THIS plan (no admin handler exists yet);
w7-mcp-3 will construct/inject the launcher into its handlers using this same seam.

### 1.5 stdio↔SSE bridge (NEW — `internal/mcp/bridge.go`)

One `Bridge` per plugin (PAR-MCP-003) wrapping a `Process` (§1.4) + a session map.
The framing/broadcast/lifecycle wiring is the unit-test core (the PURE analogue of
the tunnel URL parser):

```go
type Bridge struct {
    mu       sync.RWMutex
    proc     Process
    sessions map[string]SessionSink // sid -> sink
    buffer   []byte                  // accumulated partial stdout (Go-port note)
}

// SessionSink delivers a frame to one SSE session. A real sink writes to the
// http.ResponseWriter/flusher (w7-mcp-2/3); the test sink is an in-memory recorder.
type SessionSink func(frame []byte) error
```
- **AddSession(sid, sink)/RemoveSession(sid)** — register/unregister an SSE consumer.
- **onFrame(frame)** (wired as the `Process.OnFrame` callback) — broadcast the frame
  to every session sink (PAR-MCP-007). A sink that errors is silently dropped from
  the map and does NOT abort the broadcast loop (PAR-MCP-054 "ignore broken pipe").
- **Send(frame)** — write a JSON-RPC frame to the child stdin (`Process.Write`).
- **onExit(code)** — invoke the launcher's registry-delete (PAR-MCP-053).
- **IsRunning()** — delegate to `proc.IsRunning()` (PAR-MCP-051).

**THE FRAME SPLITTER (PURE + UNIT-TESTED — binding).** Newline-delimited JSON-RPC
framing is a PURE function over the accumulated buffer (the analogue of
`extractQuickTunnelURL`):
```go
// splitFrames consumes complete newline-delimited JSON frames from buf, returning
// the complete frames and the remaining partial tail. PURE — no I/O. Mirrors
// 9router's newline-split of proc.stdout (stdioSseBridge.js:151).
func splitFrames(buf []byte) (frames [][]byte, rest []byte)
```
Unit-tested on canned byte streams: a single complete frame; two frames in one chunk;
a partial frame held in `rest` until its newline arrives across two chunks; empty
input; blank lines skipped. The real stdout pipe in `osProcessRunner` feeds
`splitFrames` over the buffer — but the SPLITTER is what the unit test covers, NOT
the pipe.

### 1.6 smart text filter (PURE + UNIT-TESTED — binding) — `internal/mcp/filter.go`

Port 9router's `smartFilterText` + `collapseRepeated` (PAR-MCP-008,
`stdioSseBridge.js:20`) as PURE functions (no I/O):
```go
// smartFilterText drops noise nodes (role==generic, empty text lines), collapses
// repeated role-prefixed sibling lines, and hard-truncates the result at 50_000
// chars. PURE. Mirrors stdioSseBridge.js:20 + collapseRepeated.
func smartFilterText(s string) string
const maxToolResultChars = 50_000 // 9router hard cap
```
Behaviors to port (from `9router-mcp.md:18` + Edge Cases #4): regex-drop `generic`
nodes + empty `text` lines; `collapseRepeated` groups consecutive role-prefixed
siblings; hard-truncate at 50_000 chars. The filter MUTATES only `type==="text"`
content; non-text content passes through (quirk #4) — model this at the call site
(applied to text fields of a JSON-RPC tool result), but the STRING transform itself
is the PURE unit-tested core. Unit tests on canned strings: noise dropped; repeated
siblings collapsed; a >50K input truncated to exactly 50_000; a clean short string
unchanged; non-text untouched. **DECIDE the exact 9router regexes at T-filter
(§8 ESC-FILTER): default = port the ref regexes verbatim; if a regex is ambiguous in
the matrix note, ESCALATE with the ref `stdioSseBridge.js:20-89` text — never
fabricate the filter semantics.**

### 1.7 the real process runner (`internal/mcp/process.go`, NEW — INTEGRATION-ONLY)

`osProcessRunner` implements `ProcessRunner`/`Process` by shelling out via `os/exec`
(`StdinPipe`/`StdoutPipe`/`StderrPipe` + `cmd.Wait` + goroutine scanners — Go-port
note §165). It is the ONLY place `os/exec` is referenced; it is guarded so it is
NEVER invoked in `go test ./...` (tests inject the fake runner). Its stdout goroutine
feeds `splitFrames` (§1.5); its stderr goroutine feeds `OnStderr` (PAR-MCP-052); its
`cmd.Wait` goroutine fires `OnExit` (PAR-MCP-053); `IsRunning` checks
`cmd.ProcessState == nil || !cmd.ProcessState.Exited()` (PAR-MCP-051). **This file
carries NO unit-tested logic — the testable logic (split/broadcast/filter/allowlist/
lifecycle wiring) lives in the PURE helpers + the fake-driven bridge tests (§1.9).**

### 1.8 default plugin definitions (PAR-MCP-043/044) — `internal/mcp/defaults.go` (NEW)

Port the default plugin DEFINITIONS as Go constants/values (no live HTTP — the
clients are mcp-2/3): Exa (`transport:http, oauth:false`), Tavily
(`transport:http, oauth:true`), browsermcp (`transport:stdio, command:npx,
args:["-y","@browsermcp/mcp@latest"]`, 10 tool names — `coworkPlugins.js:3,26`). A
trivial unit test asserts the browsermcp command is allowlisted
(`isAllowedCommand("npx")==true`) and the definition counts match the ref. These
definitions are consumed by w7-mcp-2 (registry/probe) and w7-mcp-3 (seed/UI).

### 1.9 What is UNIT-TESTED vs INTEGRATION-ONLY (binding — the hermeticity guarantee)

**UNIT-TESTED (deterministic, hermetic — `go test ./...` with NO process / NO
network / NO port bind / NO download / NO real-process sleep):**
- The command allowlist `isAllowedCommand` — EVERY accept + EVERY reject (`rm`,
  `bash`, `bash -c`, `sh`, `./npx`, `../npx`, `/bin/rm`, `npx; rm -rf /`, `$(x)`,
  empty) — the security boundary, exhaustively (§4 T-allowlist).
- `splitFrames` — newline-delimited framing incl. partial-frame carryover (§1.5).
- `smartFilterText` — noise-drop, sibling-collapse, 50K truncate, non-text passthrough
  (§1.6).
- The `Bridge` broadcast/lifecycle via a FAKE `Process`: add/remove session;
  broadcast reaches all sinks; a failing sink is dropped without aborting
  (PAR-MCP-054); `IsRunning` reflects the fake's liveness (PAR-MCP-051); a fake exit
  fires `OnExit` → registry-delete (PAR-MCP-053); stderr line → `OnStderr`
  (PAR-MCP-052).
- The launcher: `StartStdio` REJECTS a non-allowlisted command BEFORE any spawn
  (PAR-MCP-005 — assert the fake runner's `Start` was NEVER called); one bridge per
  plugin (PAR-MCP-003); re-spawn-iff-not-running.
- The store: clients/instances CRUD + status transitions; OAuth account/flow
  round-trip with **tokens encrypted at rest** (raw `*_enc` column ≠ cleartext);
  flow consume deletes + expires.

**INTEGRATION-ONLY (NOT unit-tested — thin, isolated, escalation-recorded):** the
real `os/exec` spawn/kill + stdin/stdout/stderr pipes + `cmd.Wait`
(`osProcessRunner`, §1.7); the live HTTP/SSE client dial (deferred to w7-mcp-2). These
live behind `ProcessRunner` and are excluded from `go test ./...` determinism
(guarded so they are never invoked in unit tests; §5 grep proof
"no-real-spawn-in-test").

### NOT in scope (explicit — foundation only)

- **No admin handler** — NO `internal/admin/mcp.go` / `mcpoauth.go` (w7-mcp-3).
- **No route registration** — NO `internal/server/routes_admin.go` edit (w7-mcp-3
  takes the serial slot). Confirmed: this plan touches NO serial file.
- **No probe / registry / OAuth-flow engine / health monitor / discovery / agent
  loop** — those are w7-mcp-2 (probe/registry/OAuth) and w7-mcp-3 (agent). This plan
  ships ONLY the store + launcher + bridge + filter + allowlist + defaults the later
  plans consume.
- **No live HTTP/SSE I/O** — the SSE/message HTTP endpoints (PAR-MCP-001/002/055/056)
  are w7-mcp-2/3; this plan models the transport mode + the session-sink seam only.
- **No UI / mock / seed / spec / e2e** — the MCP foundation has no UI contract; the
  w6-l MCP mocks are reconciled by w7-mcp-3 (MAP §182).
- **No edits to `internal/schemas/mcp.go`** — the MCP types are CONSUMED as-is
  (PAR-MCP-030 HAVE).
- **No edits to SHIPPED platform/tunnel files** — CONSUME the Runner/SetRunner
  precedent, do NOT edit `internal/platform/tunnel/*`.
- **No edits to pre-existing store files** except the additive `migrate.go` tables.
- **No `New(...)` signature change anywhere** — the launcher exposes `NewLauncher` +
  `SetRunner`; constructors are NEW (foundation package), not modifications.
- **No destructive DDL** — additive `ensureTable`/`ensureColumn` only.
- **No new global state beyond the documented package-level bridge registry** behind a
  mutex (Go-port note §170) — owned by the launcher instance, not a free global.
- **No secret exposure** — OAuth tokens + PKCE verifier `*_enc` at rest, never echoed.
- **No real process spawn / network / port bind / download in any unit test.**

---

## 2. Precondition checks

Run all before any edit; abort and report to orchestrator on any failure.

```bash
# P0 — clean tree; record <base>
git status --porcelain     # expect empty EXCEPT a possibly-dirty ui/dist/index.html
                           # (gitignored build artifact — NEVER stage it, NEVER revert it).
                           # If anything ELSE is dirty, STOP. Use explicit `git add <file>`, never -A.
git rev-parse HEAD         # record as <base> for §5

# P1 — the gap is REAL (greenfield internal/mcp; no MCP store/launcher)
test ! -e internal/store/mcpinstances.go && test ! -e internal/store/mcpoauth.go && echo "mcp store gap OK"
test ! -e internal/mcp/launcher.go && test ! -e internal/mcp/bridge.go && test ! -e internal/mcp/filter.go && echo "mcp launcher gap OK"
grep -nE 'mcp_instances|mcp_clients|mcp_oauth' internal/store/migrate.go ; echo "^ expect EMPTY (no mcp tables)"
ls internal/mcp/   # expect only doc.go + mcp_test.go (the placeholders)

# P2 — the SHIPPED injectable-runner precedent to COPY is present (w7-plat-2 tunnels)
grep -nE 'type Runner interface|func .*SetRunner|func NewService' internal/platform/tunnel/runner.go internal/platform/tunnel/service.go
grep -nE 'extractQuickTunnelURL|isValidExecutable|INTEGRATION-ONLY' internal/platform/tunnel/cloudflared.go   # pure-helper-vs-integration split
grep -nE 'type fakeRunner|newServiceTestStore' internal/platform/tunnel/service_test.go                      # the fake-process + temp-store test pattern

# P3 — the MCP types exist (consume, do not redefine)
grep -nE 'type MCPClient|type MCPInstance|type MCPTool|type MCPToolGroup' internal/schemas/mcp.go

# P4 — reused store surfaces + secret-at-rest precedent
grep -nE 'var ErrNotFound|func newID|func boolToInt|func requireRowAffected|type rowScanner' internal/store/store.go internal/store/providers.go
grep -nE 's\.cipher\.Encrypt|s\.cipher\.Decrypt|verifier_enc' internal/store/oauthsessions.go
grep -nE 's\.cipher\.Encrypt|password_enc' internal/store/proxypools.go
grep -nE 'token_enc|CREATE TABLE IF NOT EXISTS tunnels|CREATE TABLE IF NOT EXISTS proxy_pools|tables :=|func ensureColumn' internal/store/migrate.go | head

# P5 — NO routes/admin/UI surface this plan (confirm foundation only)
grep -nE '/api/mcp' internal/server/routes_admin.go ; echo "^ expect EMPTY (mcp-3 adds routes)"
test ! -e internal/admin/mcp.go && echo "no mcp admin handler (correct — mcp-3)"

# P6 — green at base (HERMETIC)
go test ./... && go vet ./... && go build ./...     # exit 0 (no net/process)
```

---

## 3. Exclusive file ownership

After w7-mcp-1 merges, all CREATE files are owned by w7-mcp-1; later MCP plans consume,
never edit (MAP decision 7). w7-mcp-2 EXTENDS `internal/mcp` with NEW files (sse.go,
probe.go, registry.go, oauth.go, …) and CONSUMES this plan's constructors.

**CREATE — store (NEW):**

| File | Contract |
|---|---|
| `internal/store/mcpinstances.go` | `MCPClient`/`MCPInstance` store records + CRUD/status methods; `*_json` columns for map/slice fields; `newID`/`boolToInt`/`ErrNotFound`/`requireRowAffected`. Mirrors `proxypools.go`/`providers.go`. |
| `internal/store/mcpinstances_test.go` | Temp `store.Open` (mirror `service_test.go:47-60`): client+instance create→get→list→status-transition→delete. RED first. |
| `internal/store/mcpoauth.go` | OAuth accounts + flows; `s.cipher.Encrypt/Decrypt` for `access_token_enc`/`refresh_token_enc`/`verifier_enc`; `ConsumeMCPOAuthFlow` (return+delete+expire). Mirrors `oauthsessions.go`. |
| `internal/store/mcpoauth_test.go` | Account upsert→get; **token round-trips encrypted (raw `*_enc` column ≠ cleartext)**; flow create→consume→deleted; expired flow → `ErrNotFound`. RED first. |

**EXTEND — store (additive only):**

| File | Change (additive ONLY) |
|---|---|
| `internal/store/migrate.go` | ADD `mcp_clients`, `mcp_instances`, `mcp_oauth_accounts`, `mcp_oauth_flows` to the `tables` slice (mirror the SHIPPED `tunnels` block @ :205). ADDITIVE ONLY — no DROP/RENAME. |

**CREATE — domain (NEW files in package `internal/mcp`):**

| File | Contract |
|---|---|
| `internal/mcp/runner.go` | `ProcessRunner`/`Process` interfaces + `ProcessSpec` (§1.4). The injectable seam. No `init()`; errors-as-values. |
| `internal/mcp/allowlist.go` | `allowedMCPCommands` map + PURE `isAllowedCommand(cmd) bool` (§1.4 hardening). The security boundary. |
| `internal/mcp/allowlist_test.go` | EVERY accept + EVERY reject (rm/bash/bash -c/sh/./npx/../npx//bin/rm/`npx; rm -rf /`/`$(x)`/empty). RED first. PURE — no spawn. |
| `internal/mcp/bridge.go` | `Bridge` (session map + buffer) + `AddSession`/`RemoveSession`/`onFrame` broadcast/`Send`/`onExit`/`IsRunning` + PURE `splitFrames` (§1.5). |
| `internal/mcp/bridge_test.go` | Via a FAKE `Process`: `splitFrames` framing incl. partial carryover; broadcast reaches all sinks; failing sink dropped without abort (PAR-MCP-054); exit→registry-delete; stderr→callback; `IsRunning`. RED first. NO process/network. |
| `internal/mcp/filter.go` | PURE `smartFilterText` + `collapseRepeated` + `maxToolResultChars=50_000` (§1.6). |
| `internal/mcp/filter_test.go` | noise-drop; sibling-collapse; >50K → exactly 50_000; clean-short unchanged; non-text passthrough. RED first. |
| `internal/mcp/launcher.go` | `Launcher{ runner ProcessRunner; st *store.Store; bridges map[string]*Bridge; mu sync.RWMutex }`; `NewLauncher(st)` (real `osProcessRunner`); `SetRunner(r)` test override (mirror `tunnel.SetRunner`); `StartStdio` (allowlist-gate-BEFORE-spawn + bridge register), `StartHTTP`/`StartSSE` (mode+instance persist, no live dial), `IsRunning`, exit-delete wiring. No `New()` sig change (NEW pkg). |
| `internal/mcp/launcher_test.go` | Via `SetRunner(fakeRunner)`: `StartStdio` rejects non-allowlisted BEFORE spawn (assert fake `Start` NOT called); one bridge per plugin; re-spawn-iff-not-running; exit removes bridge from registry. RED first. NO process/network. |
| `internal/mcp/process.go` | `osProcessRunner`/`osProcess` real `os/exec` impl (StdinPipe/StdoutPipe/StderrPipe + cmd.Wait + goroutine scanners feeding splitFrames/OnStderr/OnExit; IsRunning via ProcessState). INTEGRATION-ONLY — never invoked by unit tests (§1.7/§1.9). |
| `internal/mcp/defaults.go` | Exa/Tavily/browsermcp default plugin definitions as Go values (PAR-MCP-043/044). |
| `internal/mcp/defaults_test.go` | browsermcp command allowlisted; definition counts/tool-name counts match the ref. RED first. |

**REPLACE — the Phase-1 placeholder:**

| File | Change |
|---|---|
| `internal/mcp/mcp_test.go` | REPLACE the no-op `TestPackageCompiles` with real tests (or delete it once the new `_test.go` files cover the package; keep the package test-covered at every commit). |

**FORBIDDEN:** everything else. Explicitly: ALL `internal/admin/*` (no MCP handler —
w7-mcp-3); `internal/server/routes_admin.go` (no route — w7-mcp-3 serial slot);
`internal/server/guard.go` (the `LOCAL_ONLY_PATHS` mcp entry is w7-mcp-3);
`internal/schemas/mcp.go` (consume, don't redefine); ALL SHIPPED
`internal/platform/tunnel/*` + `internal/platform/proxypools.go` (consume the
precedent, do NOT edit); ALL pre-existing `internal/store/*.go` except the NEW mcp
files + the additive `migrate.go` tables; ALL `internal/inference/*` (MCP is not an
inference concern); ALL `ui/**` (src, mocks, seeds, specs, dist); `internal/auth/*`
(the PKCE engine reuse is w7-mcp-2). Touching any of these is an automatic REJECT.

---

## 4. TDD tasks

Cadence (strict, AGENTS.md "TDD always: write test first, see it fail, write minimum
code to pass"): **no Go impl file may exist before its `_test.go` is committed RED.**
`go test ./... && go vet ./... && go build ./...` green at EVERY commit, FULLY
HERMETIC (no network, no process spawn, no port bind, no real-process sleep). Order:
allowlist (security first) → frame splitter + filter (pure cores) → bridge (fake
process) → mcp store → mcp oauth store → launcher (fake runner) → defaults → closeout.

### T-allowlist — STEP(a) RED, STEP(b) impl (the security boundary FIRST)
STEP(a): write `internal/mcp/allowlist_test.go` (every accept + every reject case).
`go test ./internal/mcp/ -run Allow` → FAIL. Commit RED:
`phase-1/w7-mcp-1: failing command-allowlist tests (TDD red)`.
STEP(b): implement `internal/mcp/allowlist.go` (`allowedMCPCommands` + PURE
`isAllowedCommand` with the §1.4 hardening). Gates green. Commit:
`phase-1/w7-mcp-1: MCP command allowlist (rejects rm/bash/relative/abs-arbitrary)`.

### T-pure — STEP(a) RED, STEP(b) impl (frame splitter + smart filter)
STEP(a): write `internal/mcp/bridge_test.go` (the `splitFrames` cases) +
`internal/mcp/filter_test.go`. `go test ./internal/mcp/ -run 'Frame|Filter'` → FAIL.
Commit RED: `phase-1/w7-mcp-1: failing frame-split + smart-filter tests (TDD red)`.
STEP(b): implement PURE `splitFrames` (`bridge.go`) + `smartFilterText`
(`filter.go`). Gates green. Commit:
`phase-1/w7-mcp-1: JSON-RPC frame split + smart text filter (pure)`.

### T-bridge — STEP(a) RED, STEP(b) impl (bridge over a FAKE process)
STEP(a): write `internal/mcp/runner.go` (interfaces — compiles) + extend
`bridge_test.go` (broadcast/failing-sink-drop/exit-delete/stderr/IsRunning via a fake
`Process`). → FAIL. Commit RED:
`phase-1/w7-mcp-1: failing bridge broadcast/lifecycle tests (TDD red)`.
STEP(b): implement the `Bridge` (session map, broadcast, onExit, IsRunning) in
`bridge.go`. Gates green (fake process only). Commit:
`phase-1/w7-mcp-1: stdio<->SSE bridge broadcast + lifecycle (injectable process)`.

### T-mcpstore — STEP(a) RED store, STEP(b) impl
STEP(a): write `internal/mcp/instances`/clients tests in
`internal/store/mcpinstances_test.go`; ADD `mcp_clients` + `mcp_instances` to
`migrate.go`. `go test ./internal/store/ -run MCP` → FAIL. Commit RED:
`phase-1/w7-mcp-1: failing mcp clients/instances store tests (TDD red)`.
STEP(b): implement `internal/store/mcpinstances.go`. Gates green. Commit:
`phase-1/w7-mcp-1: mcp clients + instances store (additive tables)`.

### T-mcpoauth — STEP(a) RED store, STEP(b) impl
STEP(a): write `internal/store/mcpoauth_test.go` (token encrypted-at-rest; flow
consume+expire); ADD `mcp_oauth_accounts` + `mcp_oauth_flows` to `migrate.go`.
`go test ./internal/store/ -run MCPOAuth` → FAIL. Commit RED:
`phase-1/w7-mcp-1: failing mcp oauth store tests (TDD red)`.
STEP(b): implement `internal/store/mcpoauth.go` (`*_enc` round via `s.cipher`).
Gates green. Commit:
`phase-1/w7-mcp-1: mcp oauth accounts + flows store (tokens *_enc at rest)`.

### T-launcher — STEP(a) RED, STEP(b) impl (launcher over a FAKE runner)
STEP(a): write `internal/mcp/launcher_test.go` (allowlist-gate-before-spawn via fake
runner; one bridge per plugin; re-spawn-iff-not-running; exit→registry-delete). →
FAIL. Commit RED: `phase-1/w7-mcp-1: failing launcher tests (TDD red)`.
STEP(b): implement `internal/mcp/launcher.go` (`NewLauncher`/`SetRunner`/`StartStdio`/
`StartHTTP`/`StartSSE`/`IsRunning`) + the INTEGRATION-ONLY `internal/mcp/process.go`
(`osProcessRunner` real `os/exec`, guarded so no unit test invokes it). Gates green
(fake runner only). Commit:
`phase-1/w7-mcp-1: MCP launcher (stdio/http/sse modes; injectable process runner)`.

### T-defaults — STEP(a) RED, STEP(b) impl
STEP(a): write `internal/mcp/defaults_test.go` (browsermcp allowlisted; counts match
ref). → FAIL. Commit RED:
`phase-1/w7-mcp-1: failing default-plugin tests (TDD red)`.
STEP(b): implement `internal/mcp/defaults.go` (Exa/Tavily/browsermcp). REPLACE/remove
the no-op `mcp_test.go` placeholder (package stays test-covered). Gates green. Commit:
`phase-1/w7-mcp-1: default MCP plugin definitions (exa/tavily/browsermcp)`.

### T-close — full gates + closeout
```bash
go test ./... && go vet ./... && go build ./...                       # HERMETIC — no net/process
go test ./internal/mcp/... -v                                        # all pass
go test ./internal/store/ -run 'MCP|Mcp' -v                          # incl token encrypted-at-rest
go test ./internal/mcp/... ./internal/store/ -run 'Mcp|MCP'          # the binary-acceptance subset
```
Flip the matrix in `.planning/parity/matrix/9router-mcp.md`: PAR-MCP-003,004,005,006,
007,008,032,035,036,043,044,051,052,053,054 → HAVE (real Go; the real `os/exec` spawn
/ binary download / live HTTP-SSE client are integration-only behind the runner —
§1.9). PAR-MCP-033 footnote (store half shipped; handlers w7-mcp-3). Append the §8
open items to `open-questions.md` (ESC-ALLOWLIST hardening, ESC-PERSIST DB-vs-file,
ESC-SCHEMA typed+JSON, ESC-FILTER regex-port, the integration-only-spawn note, the
foundation-confirms-no-routes/no-UI note, the mcp-2/3 dependency handoff). Update
`docs/WORKFLOW.md` (P6 base observation; the ESC decisions; the constructors mcp-2/3
consume). Final commit:
`phase-1/w7-mcp-1: close — MCP foundation (store+launcher+bridge); matrix flip`.

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0. Diff gate is
**w7-mcp-1 commit-range-scoped** (§7).

**Test gates (HERMETIC — no network, no process spawn, no port bind, no real-process
sleep)**
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/mcp/... ./internal/store/ -run 'Mcp|MCP' -v` → exit 0, all pass
  (allowlist: every accept + every reject; splitFrames incl. partial carryover;
  filter: drop/collapse/50K/passthrough; bridge: broadcast + failing-sink-drop + exit
  + IsRunning; launcher: reject-before-spawn + one-bridge-per-plugin; store: CRUD +
  status; oauth: token encrypted-at-rest + flow consume/expire).
- NO playwright (no UI surface this plan).

**TDD-order proof** — each impl file's covering test appears in an earlier-or-equal
commit:
```bash
for pair in \
  "internal/mcp/allowlist_test.go:internal/mcp/allowlist.go" \
  "internal/mcp/filter_test.go:internal/mcp/filter.go" \
  "internal/mcp/bridge_test.go:internal/mcp/bridge.go" \
  "internal/mcp/launcher_test.go:internal/mcp/launcher.go" \
  "internal/store/mcpinstances_test.go:internal/store/mcpinstances.go" \
  "internal/store/mcpoauth_test.go:internal/store/mcpoauth.go" ; do
  tf=${pair%%:*}; im=${pair##*:}
  ct=$(git log --format=%ct --diff-filter=A -1 -- "$tf")
  cf=$(git log --format=%ct --diff-filter=A -1 -- "$im")
  [ "$ct" -le "$cf" ] || echo "TDD VIOLATION: $im"     # prints nothing
done
```

**Grep proofs**
```bash
# allowlist (security boundary) present + the exact 7-command set
grep -nE 'func isAllowedCommand|"npx"|"node"|"uvx"|"python"|"python3"|"bunx"|"bun"' internal/mcp/allowlist.go
# pure cores present
grep -nE 'func splitFrames' internal/mcp/bridge.go
grep -nE 'func smartFilterText|50000|50_000' internal/mcp/filter.go
# injectable runner seam (mirrors tunnel.Runner/SetRunner)
grep -nE 'type ProcessRunner interface|type Process interface|func .*SetRunner|func NewLauncher' internal/mcp/runner.go internal/mcp/launcher.go
# bridge lifecycle (PAR-MCP-051/053/054)
grep -nE 'func .*IsRunning|func .*AddSession|func .*onExit|func .*onFrame' internal/mcp/bridge.go
# store methods + *_enc
grep -nE 'func .*MCPClient|func .*MCPInstance' internal/store/mcpinstances.go
grep -nE 'func .*MCPOAuth|access_token_enc|refresh_token_enc|verifier_enc|s\.cipher\.(Encrypt|Decrypt)' internal/store/mcpoauth.go
# additive mcp tables
grep -nE 'mcp_clients|mcp_instances|mcp_oauth_accounts|mcp_oauth_flows' internal/store/migrate.go
# no init(); no free global state
! grep -rn 'func init(' internal/mcp/ internal/store/mcpinstances.go internal/store/mcpoauth.go && echo "no init() OK"
```

**Allowlist-rejection proofs (binding — the security guarantee)** — the
`allowlist_test.go` MUST assert `isAllowedCommand` returns false for each of:
`rm`, `bash`, `bash -c`, `sh`, `./npx`, `../npx`, `/bin/rm`, `npx; rm -rf /`,
`$(touch x)`, `` `id` ``, `""` (empty); and true for `npx`, `node`, `uvx`, `python`,
`python3`, `bunx`, `bun`, `/usr/local/bin/npx`.
```bash
grep -nE 'rm|bash|sh|\./npx|/bin/rm|;|\$\(' internal/mcp/allowlist_test.go   # rejection cases present
```

**No-real-spawn / no-network-in-test proofs (binding — hermeticity)**
```bash
# unit tests NEVER spawn a process, dial the network, bind a port, or download:
! grep -nE 'exec\.Command|os/exec|http\.Get|http\.Client|net\.Listen|net\.Dial|\.Download|cmd\.Start|cmd\.Run' \
   internal/mcp/allowlist_test.go internal/mcp/bridge_test.go internal/mcp/filter_test.go \
   internal/mcp/launcher_test.go internal/mcp/defaults_test.go \
   internal/store/mcpinstances_test.go internal/store/mcpoauth_test.go && echo "no real spawn/net in tests OK"
# the real os/exec lives ONLY in process.go, behind ProcessRunner:
grep -nE 'exec\.Command|os/exec' internal/mcp/process.go                      # expect MATCHES here only
! grep -rnE 'exec\.Command|os/exec' internal/mcp/launcher.go internal/mcp/bridge.go internal/mcp/filter.go internal/mcp/allowlist.go && echo "os/exec confined to process.go OK"
# the fake runner/process used by tests implements the interfaces without any process:
grep -nE 'ProcessRunner|Process\b' internal/mcp/launcher_test.go internal/mcp/bridge_test.go   # fake impl present
```

**No-secret-exposure proofs (binding)**
```bash
# tokens/verifier encrypted at rest; the mcpoauth_test.go marshals a stored row +
# asserts the raw *_enc column != cleartext and no DTO field carries the cleartext.
grep -nE 'access_token_enc|refresh_token_enc|verifier_enc|s\.cipher\.(Encrypt|Decrypt)' internal/store/mcpoauth.go
# additive migrations only (no DROP/RENAME introduced by this plan)
git diff <base>..HEAD -- internal/store/migrate.go | grep -E '^\+' | grep -iE 'DROP COLUMN|RENAME COLUMN|DROP TABLE' | wc -l   # = 0
```

**Negative / freeze proofs (w7-mcp-1 commit-range — §7)**
```bash
R="<first-w7-mcp-1>^..<last-w7-mcp-1>"
# Only the sanctioned Go files changed:
git diff $R --name-only -- internal/ | grep -vE \
 'internal/store/(mcpinstances|mcpoauth)(_test)?\.go|internal/store/migrate\.go|internal/mcp/.*\.go' \
 | wc -l                                                                  # = 0
# FOUNDATION-ONLY: NO routes_admin.go, NO admin handler, NO guard edit:
git diff $R --name-only -- internal/server/routes_admin.go | wc -l       # = 0  (confirm NO route this plan)
git diff $R --name-only -- internal/admin/ | wc -l                       # = 0  (confirm NO admin handler)
git diff $R --name-only -- internal/server/guard.go | wc -l              # = 0
# schemas + SHIPPED platform + inference untouched:
git diff $R --name-only -- internal/schemas/mcp.go internal/platform/ internal/inference/ internal/auth/ | wc -l   # = 0
# NO UI change at all (confirm NO UI this plan):
git diff $R --name-only -- ui/ | wc -l                                   # = 0
# migrate.go = additive only (no deletions of existing tables/logic):
git diff $R -- internal/store/migrate.go | grep -E '^-' | grep -v '^---' | wc -l   # = 0
```

---

## 6. Out of scope (restated, binding)

NO admin handler (`internal/admin/mcp.go` — w7-mcp-3). NO route registration
(`routes_admin.go` — w7-mcp-3 holds the serial slot; THIS plan holds NONE). NO
`guard.go` `LOCAL_ONLY_PATHS` edit (w7-mcp-3). NO probe / registry / OAuth-flow
engine / health monitor / discovery / agent loop (w7-mcp-2/3). NO live HTTP/SSE I/O
(the SSE/message endpoints are w7-mcp-2/3; this plan models the mode + the session-
sink seam only). NO edits to `internal/schemas/mcp.go` (consume the types). NO edits
to SHIPPED `internal/platform/tunnel/*` or `internal/platform/proxypools.go` (CONSUME
the Runner/SetRunner injection precedent, do NOT edit). NO `internal/inference/*`
(MCP is not an inference concern — no selection.go/factory.go micro-serial). NO
`internal/auth/*` (PKCE reuse is w7-mcp-2). NO UI (src/mocks/seeds/specs/dist —
foundation has no UI contract; the w6-l mocks are reconciled by w7-mcp-3). NO
`New(...)` signature change (NEW package constructors only). NO destructive DDL —
additive `ensureTable`/`ensureColumn` only. NO free global state (the bridge registry
is owned by the launcher instance behind a mutex). NO secret exposure (OAuth tokens +
PKCE verifier `*_enc`, never echoed). **NO real process spawn / network / port bind /
download in any unit test** — confined to `process.go` behind `ProcessRunner` (§1.7/
§1.9); the unit suite is fully hermetic. Ambiguity (allowlist contents, schema field
mismatch, filter regex, bridge protocol detail) → ESCALATE (§8) with the recommended
default from the 9router ref — NEVER fabricate a protocol.

## 7. Diff-gate scope

The MCP track runs concurrently with governance/providers/platform, so a broad
`<base>..HEAD` range sweeps in sibling commits. The diff gate MUST be scoped to
w7-mcp-1's own commits. Isolate them:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w7-mcp-1:" | awk '{print $1}'`
then `git diff <first-w7-mcp-1>^..<last-w7-mcp-1> -- [file list]`.

`git diff <first>^..<last> --name-only` must be exactly a subset of:
```
internal/store/mcpinstances.go
internal/store/mcpinstances_test.go
internal/store/mcpoauth.go
internal/store/mcpoauth_test.go
internal/store/migrate.go                 (additive 4 mcp tables; ONE concern)
internal/mcp/runner.go
internal/mcp/allowlist.go
internal/mcp/allowlist_test.go
internal/mcp/bridge.go
internal/mcp/bridge_test.go
internal/mcp/filter.go
internal/mcp/filter_test.go
internal/mcp/launcher.go
internal/mcp/launcher_test.go
internal/mcp/process.go                    (integration-only os/exec; no unit test)
internal/mcp/defaults.go
internal/mcp/defaults_test.go
internal/mcp/mcp_test.go                    (placeholder replaced/removed)
.planning/parity/matrix/9router-mcp.md      (row flips)
.planning/parity/plans/open-questions.md
docs/WORKFLOW.md
```
Any file outside this list in the scoped diff is an automatic review REJECT.
`internal/server/routes_admin.go`, `internal/admin/**`, `internal/server/guard.go`,
`internal/schemas/mcp.go`, the SHIPPED `internal/platform/**`, `internal/inference/**`,
`internal/auth/**`, and ALL `ui/**` are deliberately ABSENT — touching them is an
automatic REJECT. This plan holds NO routes_admin.go serial slot (the MCP slot is
w7-mcp-3's).

## 8. Escalations / decisions (explicit — recommended defaults, do not fabricate)

- **ESC-ALLOWLIST (RESOLVED at authoring — allowlist contents + hardening, binding
  default).** Contents = the exact 9router set (`coworkPlugins.js:67`,
  `9router-mcp.md:14`): `npx, node, uvx, python, python3, bunx, bun`. Hardening
  (§1.4): match on `filepath.Base`, REJECT shell metacharacters, REJECT relative
  paths (`./npx`), permit a bare base name OR an absolute clean path. This is
  STRICTER than 9router's raw `path.basename` (which would pass `/bin/rm`'s base — but
  `rm` isn't in the map; we additionally reject `./npx`). The unit test asserts every
  rejection. Flag for orchestrator confirmation of the metacharacter/abs-path policy.
- **ESC-PERSIST (RESOLVED at authoring — where custom plugins persist, binding
  default).** 9router writes `DATA_DIR/mcp/customPlugins.json` (PAR-MCP-006,
  `stdioSseBridge.js:130`). **Decision: persist to the SQLite `mcp_clients` table**
  (the g0router store is the canonical persistence layer; survives restart; matches
  the WAL-store decision in AGENTS.md). Parity is the BEHAVIOR (custom plugins
  survive restart), not the file. Flag.
- **ESC-SCHEMA (RESOLVED at authoring — typed vs JSON columns, binding default).**
  Typed columns for fixed-shape fields; a JSON `*_json` column for the free-form
  `Config map[string]any` / `Args []string` / `Env map[string]string` (mirror
  `alert_channels.events_json` @ `migrate.go:187`). Flag.
- **ESC-FILTER (CONDITIONAL — the exact smart-filter regexes).** The matrix note
  (`9router-mcp.md:18` + Edge Case #4) summarizes `smartFilterText`/`collapseRepeated`
  but does NOT quote every regex. **Default: port the ref `stdioSseBridge.js:20-89`
  semantics verbatim (drop `generic` + empty `text`; collapse repeated role-prefixed
  siblings; 50_000-char hard cap; mutate only `type==="text"`).** If a regex is
  ambiguous when implementing, STOP and ESCALATE with the ref source text — never
  fabricate the filter semantics. The unit test pins the OBSERVABLE behavior (drop/
  collapse/truncate/passthrough), so a minor regex variance that preserves behavior
  is acceptable; a behavioral divergence is an escalation.
- **ESC-SPAWN (RESOLVED at authoring — real spawn integration-only, recommended
  default; mirrors w7-plat-2 ESC-OS-PRIV).** The real `os/exec` spawn/kill +
  stdin/stdout/stderr pipes + `cmd.Wait` (`osProcessRunner`, §1.7) are
  INTEGRATION-ONLY (spawning arbitrary MCP servers in CI is the exact RCE risk
  9router cites at PAR-MCP-042). The parity bar = the launcher + allowlist + bridge +
  filter + lifecycle FULLY unit-tested via the fake runner; the thin real spawn is a
  guarded integration surface NOT exercised by `go test ./...`. Record at closeout:
  PAR-MCP-003/036 HAVE with an "integration-only spawn" footnote. Flag.
- **ESC-HTTP-SSE-CLIENT (DEFERRED to w7-mcp-2).** The live HTTP/SSE MCP client (probe
  handshake, session-id replay, SSE parse — PAR-MCP-009..013) is w7-mcp-2. This plan
  models the transport MODE + persists the instance + exposes the session-sink seam
  ONLY; no live dial. Record as the mcp-2 handoff.
- **ESC-RESPAWN (RESOLVED at authoring — re-spawn policy, binding default).** Mirror
  9router quirk #9: re-spawn a plugin's process on a new request ONLY if the prior
  process exited (`!IsRunning`). Default: re-spawn-iff-not-running; no health-check
  before re-spawn. Flag.
- **Foundation confirms NO routes / NO UI (binding).** This plan adds NO admin
  handler, NO `routes_admin.go` edit, NO UI/mock/seed/spec/e2e. The MCP routes_admin
  serial slot belongs to w7-mcp-3 (MAP §182/§208); this plan holds NONE. The w6-l MCP
  UI mocks are reconciled by w7-mcp-3, not here. Recorded for orchestrator: w7-mcp-1
  is fully parallelizable with every non-MCP track and takes no serial file.
- **mcp-2/3 dependency handoff (binding).** w7-mcp-2 (probe/registry/OAuth engine)
  and w7-mcp-3 (admin transport + routes + tools/agent) DEPEND on this foundation's
  exported constructors: `NewLauncher(st)`/`SetRunner`, the `Bridge` + `SessionSink`
  seam, `smartFilterText`, `isAllowedCommand`, the store methods
  (`*MCPClient`/`*MCPInstance`/`*MCPOAuth*`), and the default plugin definitions. Keep
  these exported + stable; mcp-2 EXTENDS `internal/mcp` with NEW files (no edits to
  this plan's files). Record in `open-questions.md` at closeout.
```
