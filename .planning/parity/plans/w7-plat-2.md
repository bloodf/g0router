# Micro-plan w7-plat-2 — Tunnels backend (cloudflared + tailscale) (Go)

```
wave: 7
plan: w7-plat-2
status: READY (rev 1 — authored against merged Waves 0–6 + the shipped W7
  governance plans + the SHIPPED w7-plat-1 (proxy-pools, gate-verified — its
  files are live in-tree: internal/platform/{proxypools,outboundproxy}.go,
  internal/admin/proxypools.go, internal/store/proxypools.go, the proxy_pools
  table @ migrate.go:191, and the SetProber/SetProxyProber injection seam
  @ handlers.go:86-89 — this plan REUSES that exact injection philosophy);
  live tree @ <base>; WAVE-7-MAP w7-plat-2 row ~line 184; serial chain §219-224;
  reconciliation §245; freeze rules §267)
runs: platform track. Disjoint domain/store/admin files from w7-plat-1 (proxy-pools,
  SHIPPED), w7-plat-3 (mitm) — run ∥ w7-plat-3. TAKES the
  internal/server/routes_admin.go SERIAL SLOT in chain order
  (… → w7-mcp-3 → w7-plat-1 → **w7-plat-2** → w7-plat-3 → w7-misc; MAP §219-224).
  w7-plat-1 RELEASED the slot to w7-plat-2 on its close (open-questions w7-plat-1
  §116). NO secondary micro-serial: w7-plat-2 does NOT touch selection.go /
  factory.go / runner.go (tunnels are a standalone process-managed subsystem,
  not an inference-path concern).
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w7-plat-2:
ref-source: 9router frozen @ 827e5c3 — cloudflared + tailscale tunnel surfaces.
  The BINDING contract for W7 is the W6 e2e mock (decision 1: real Go wins, mock
  corrected to mirror it). Mock sources:
    ui/e2e/mocks/handlers/tunnels.ts + ui/e2e/mocks/seed/tunnels.ts.
  9router's ref paths (`/api/tunnel/{enable,disable,tailscale-*}`) DIVERGE from
  the in-tree mock (`/api/tunnels/{type}`, REMAPPED — open-questions w6-m ESC-1c);
  the in-tree mock paths are CANONICAL (the w6-m UI calls them).
base: <base> = git rev-parse HEAD recorded at P0. Substitute the actual SHA
  everywhere §5 says <base>. (At authoring, HEAD = a41664063…; recompute at P0.)
go-serial-slot: this plan holds the SINGLE in-flight edit to
  internal/server/routes_admin.go while live (W3/W4/W5/W6 lesson; MAP decision 3).
  Slot must be FREE at P5 before T-routes (no concurrent W7 plan with an unmerged
  routes_admin.go edit); RELEASE to w7-plat-3 on close.
new-route: NO UI route files. The /tunnels page ALREADY SHIPPED in w6-m (PARTIAL)
  against the mock; this plan builds the REAL Go so the page flips PARTIAL→HAVE
  and corrects the mock body to mirror the Go DTO.
```

---

## 1. Scope — PAR rows + the two subsystems

### Rows this plan closes

| Row / item | Claim | Target state after w7-plat-2 |
|---|---|---|
| PAR-PLAT-015 | cloudflared binary download (+ magic-byte validate) | HAVE (integration-only impl — §1.9; binary download is NOT unit-tested) |
| PAR-PLAT-016 | cloudflared named-tunnel run (`tunnel run --token`) | HAVE (state machine unit-tested via fake runner; real spawn integration-only) |
| PAR-PLAT-017 | cloudflared quick-tunnel + `*.trycloudflare.com` URL extraction from stderr | HAVE (URL-parse is PURE + unit-tested on canned stderr; spawn integration-only) |
| PAR-PLAT-018 | cloudflared kill / stop | HAVE (Stop on the runner; fake-runner tested; real kill integration-only) |
| PAR-PLAT-019 | tailscale install | HAVE (integration-only impl — §1.9; OS-privileged, escalated) |
| PAR-PLAT-020 | tailscale daemon (userspace / TUN) + login poll | HAVE (state machine unit-tested via fake; daemon/poll integration-only; TUN escalated) |
| PAR-PLAT-021 | tailscale funnel | HAVE (state machine unit-tested via fake; real funnel integration-only) |
| PAR-PLAT-022 | tailscale cert | HAVE (state machine unit-tested via fake; real cert integration-only) |
| PAR-PLAT-023 | tunnel status / enable / disable + health admin API | HAVE (Go — `GET /api/tunnels`, `GET /api/tunnels/health`, `POST/DELETE /api/tunnels/{type}`; FULLY unit-tested via fake runner) |
| open-questions w6-m **ESC-1c** (tunnels backend absent) | real `/api/tunnels*` status/enable/disable + health over Cloudflare + Tailscale | RESOLVED (cite this plan) |
| PAR-UI-112 | tunnels page (PARTIAL) | PARTIAL → HAVE |
| PAR-UI-113 | tunnel enable/toggle (PARTIAL) | PARTIAL → HAVE |
| PAR-UI-114 | tunnel status/health (PARTIAL) | PARTIAL → HAVE |

Matrix flips at closeout (§4 T-close): in `.planning/parity/matrix/`, PAR-PLAT-015..023
→ HAVE (real Go; binary-download/spawn/OS-privileged parts integration-only +
escalation-recorded — §1.9); PAR-UI-112/113/114 PARTIAL → HAVE. Mark
`open-questions.md` w6-m ESC-1c RESOLVED with a cite to this plan; append any new open
items (§8).

### 1.1 Preconditions already satisfied by merged waves (evidence — cite file:line)

- **W6-m UI is SHIPPED (PARTIAL) and FROZEN (consume-only, MAP decision 8 / §267).**
  The `/tunnels` page renders against the registered mock. The UI type is
  `Tunnel` (`ui/src/lib/types.ts:284-289`):
  `{type:string, is_enabled:boolean, url:string, status:string}` — a 4-field shape.
  The binding acceptance contract is the existing spec (must stay green at closeout):
  `ui/e2e/tunnels.spec.ts` — it asserts: the page contains "Tunnels", exactly **2**
  `[data-testid='tunnel-card']`, the strings "cloudflare", "tailscale",
  "trycloudflare.com"; that toggling fires a **POST** then a **DELETE** on
  `/\/api\/tunnels\/[^/]+$/`. It does NOT assert `/health`, NOR any field beyond the
  4-field `Tunnel` shape.
- **The mock contract (CANONICAL — the page calls these in-tree paths)**
  — `ui/e2e/mocks/handlers/tunnels.ts`:
  - `GET /api/tunnels` → bare JSON array of `Tunnel` (`Array.from(store.tunnels.values())`).
  - `GET /api/tunnels/health` → `{healthy: true}`.
  - `POST /api/tunnels/{type}` (regex `\/api\/tunnels\/[^/]+$`) → sets the stored
    tunnel `is_enabled=true,status="active"` and returns the tunnel object (or `{}`).
  - `DELETE /api/tunnels/{type}` → sets `is_enabled=false,status="inactive"`,
    returns `{}`.
  - `{type}` ∈ `{cloudflare, tailscale}` (the URL's last segment).
- **The seed shape (CANONICAL)** — `ui/e2e/mocks/seed/tunnels.ts` returns two
  `Tunnel` rows:
  `{type:"cloudflare", is_enabled:false, url:"https://g0router-demo.trycloudflare.com", status:"inactive"}`
  and `{type:"tailscale", is_enabled:false, url:"http://g0router.tailnet.ts.net", status:"inactive"}`.
  **This is the canonical Go list DTO** (4 fields). The spec asserts the
  "trycloudflare.com" substring → the cloudflare seed/list `url` MUST keep that
  domain (the mock/Go must surface a cloudflare URL containing `trycloudflare.com`).
- **The settings-driven tunnel host-access guard EXISTS — CONSUME, do NOT edit**
  — `internal/server/guard.go:135-141`: when `tunnelDashboardAccess != "true"`, a
  request whose host matches `urlHostname(settings["tunnelUrl"])` or
  `urlHostname(settings["tailscaleUrl"])` is redirected to `/login`. This is a
  FORWARD-LOOKING inbound host-access guard keyed off settings, NOT a CRUD route.
  w7-plat-2 COEXISTS with it (it may, optionally, write `settings["tunnelUrl"]`/
  `settings["tailscaleUrl"]` when a tunnel goes active so the guard sees the live
  host — §8 ESC-GUARD-SETTINGS) but NEVER edits `guard.go`.
- **The `internal/platform` package + the SHIPPED injection precedent (the central
  reuse).** `internal/platform/doc.go` names "the proxy pool" as a platform feature;
  the package now holds the SHIPPED w7-plat-1 files. The KEY precedent this plan
  mirrors EXACTLY:
  - `internal/platform/proxypools.go:18` `type Prober func(proxyURL, target string)
    (latencyMs int, err error)` — an injectable seam "so tests run without network
    access".
  - `internal/platform/proxypools.go:25` the service holds `prober Prober` as a
    FIELD; `NewProxyPoolService(st)` constructs WITHOUT it (`:30`); `SetProber(p)`
    (`:36`) injects it post-construction; `TestConnectivity` falls back to a
    `defaultProber` when the field is nil (`:99-101`).
  - `internal/admin/handlers.go:22` the `Handlers` struct holds
    `proxyPools *platform.ProxyPoolService`; `New(...)` constructs it via
    `platform.NewProxyPoolService(st)` (`handlers.go:53`) with NO `New(...)`
    signature change; `SetProxyProber(p platform.Prober)` (`handlers.go:86-89`) is
    the post-construction injector that forwards to `h.proxyPools.SetProber(p)`.
  - **This is the IDENTICAL shape w6-j's `SetShutdownFunc` established
    (`internal/admin/handlers.go:78-83`; w6-j §1.5b) for "external effect, testable
    without performing it." w7-plat-2's `TunnelRunner` REUSES this exact philosophy
    (§1.4).**
- **Secret-at-rest precedent (`*_enc`)** — `internal/store/migrate.go`: the
  `tables` slice declares reversible `*_enc` columns written/read via `s.cipher`:
  `connections.secret_enc` (`migrate.go:51`), `oauth_sessions.verifier_enc`
  (`migrate.go:62`), `alert_channels.config_enc` (`migrate.go:186`), and the
  SHIPPED `proxy_pools.password_enc` (`migrate.go:198`). The encrypt/decrypt round
  is `s.cipher.Encrypt/Decrypt` (proxy-pools store uses it for `password_enc`). The
  tunnel TOKEN follows this precedent: `token_enc` (§1.3).
- **Additive migrations only** — `migrate.go` new tables via the `tables []struct`
  slice with `CREATE TABLE IF NOT EXISTS` (`migrate.go:15-200`); new columns via the
  `ensureColumn(db, table, column, decl)` loop (`migrate.go:255-257`, helper at
  `:333`). The SHIPPED `proxy_pools` table (`migrate.go:191`) is the immediate
  precedent for adding a `tunnels` table the same way.
- **Envelope + handler patterns** (`internal/admin/respond.go`):
  `writeData(ctx, status, data)` (`respond.go:19`) / `writeError(ctx, status,
  message)` (`respond.go:23`) → `{data,error:{message}}` snake_case.
  `pathID(ctx.UserValue("id"))` extracts a path param (`handlers.go:93`) — for
  tunnels the param is `{type}`, so use `ctx.UserValue("type")` (mirror the same
  cast). CRUD/handler template = `internal/admin/proxypools.go` (the freshest
  sibling: DTO, request structs, `writeData/writeError`, `h.recordAudit`,
  nil-safe service field).
- **Admin test harness** (`internal/admin/admin_test.go:24` `newTestEnv`): real
  `store.Open(tempDB, secret)` (`:31`) + `auth.NewSessions` + `SeedAdmin("admin",
  "123456")` (`:38`) + `New(...)`. NO mocks. `call(t, h, method, uri, body,
  userValues, headers)` (`:72`) drives a handler + decodes the `{data,error}`
  envelope. This is the authoritative proof surface.
- **The audit seam** — `internal/admin/audit.go:64`
  `func (h *Handlers) recordAudit(ctx, action, target, details string)`. REUSE
  `h.recordAudit` on every tunnel enable/disable mutation (NO audit retrofit into
  other files; NO edit to audit.go).
- **Handlers injection** — the `Handlers` struct composes `h.store` directly; new
  domains use `h.store` with NO new global state and NO `New(...)` signature change
  (MAP decision 9). New tunnel-service field is constructed in `New` over the
  existing `st` (mirror `proxyPools: platform.NewProxyPoolService(st)` at
  `handlers.go:53`).

### 1.2 The mock contract this flip must mirror (binding — decision 1)

**Decision 1 (MAP §36, §245):** real Go wins; the W6 mock body + seed are corrected
IN THIS PLAN to mirror the real Go `{data,error}` snake_case DTO. The page is FROZEN
(decision 8); prefer matching the mock's existing field names in the Go DTO; only
ESCALATE if impossible.

**Tunnels** (`ui/e2e/mocks/handlers/tunnels.ts` + `seed/tunnels.ts`):
- Routes the page consumes (canonical, in-tree):
  - `GET /api/tunnels` → bare array under `{data}` of `tunnelDTO`.
  - `GET /api/tunnels/health` → `{data:{healthy:bool}}`.
  - `POST /api/tunnels/{type}` → enable; returns the updated `{data:tunnelDTO}`.
  - `DELETE /api/tunnels/{type}` → disable; returns `{data:{...}}` (page ignores body).
- DTO shape = the UI `Tunnel` type (`types.ts:284-289`):
  `{type:string, is_enabled:bool, url:string, status:string}` — **this is the
  canonical 4-field Go list DTO**. `type` ∈ `{cloudflare, tailscale}`; `status` ∈
  `{active, inactive, ...}` (the seed uses `"inactive"`/`"active"`); `url` is the
  active tunnel URL (cloudflare must surface a `*.trycloudflare.com` URL for a quick
  tunnel; the spec asserts the substring).
- **Mock divergences to reconcile (mock mirrors Go — decision 1):**
  - **Envelope:** the mock returns BARE bodies (`json(route, array)` /
    `json(route, {healthy:true})`); the Go returns the `{data}` envelope
    (`respond.go:19`). **VERIFY** whether the mock's `json()` helper already wraps in
    `{data}` (it likely does, mirroring the other handlers) — if `json()` wraps, no
    change; if not, the mock is already what the page expects (the page's `apiFetch`
    unwraps `{data}`). **Reconciliation:** confirm the mock body matches the Go's
    `{data}` shape at T-mocks; correct ONLY on a real divergence (§8 ESC-MOCK).
  - **POST/DELETE body:** the mock POST returns the tunnel object;
    DELETE returns `{}`. Go: POST returns `{data:tunnelDTO}` (the enabled tunnel),
    DELETE returns `{data:{message:"Tunnel disabled"}}` or the disabled `tunnelDTO`.
    The page ignores the mutation body (the spec only asserts the request FIRES, not
    its response). **Reconciliation:** mirror the Go return shape iff a spec asserts
    it (none does — the spec asserts only the request method/path); default = leave
    the mock POST/DELETE bodies as-is if `json()` envelope-compatible, else mirror.
  - **`/health`:** the mock returns `{healthy:true}`; Go returns
    `{data:{healthy:bool}}` where `healthy` reflects the live runner status (true iff
    a tunnel is active/reachable, else false; deterministic in unit tests via the
    fake runner). The page does NOT call `/health` in the spec (VERIFY at P4 via
    `grep -nE 'tunnels/health' ui/src`); if unused, `/health` is a parity endpoint
    proven by the Go admin test only (§8 ESC-HEALTH-USE).
  - **Seed `url`:** the cloudflare seed `url` contains `trycloudflare.com` (spec
    asserts the substring) — KEEP it. The corrected Go list DTO for a seeded/known
    cloudflare tunnel must surface a `trycloudflare.com` URL when active; the mock
    seed stays as the 2-row `Tunnel` shape (verify field names; no change expected).

### 1.3 Tunnels Go contract (NEW, TDD)

Table `tunnels` (additive, `migrate.go` `tables` slice — mirror the SHIPPED
`proxy_pools` table at `migrate.go:191`). **DECIDE typed-vs-JSON columns (§8
ESC-SCHEMA).** RECOMMENDED default = **typed columns** (the DTO is a fixed small
shape; typed columns enable the `WHERE type=?` lookup cleanly; matches the
proxy-pools/gov precedent). The tunnel TOKEN is `token_enc` at rest (mirror
`proxy_pools.password_enc`, `migrate.go:198`):

```sql
CREATE TABLE IF NOT EXISTS tunnels (
  type TEXT PRIMARY KEY,                 -- 'cloudflare' | 'tailscale' (fixed 2 rows)
  is_enabled INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'inactive', -- inactive|starting|active|error
  url TEXT NOT NULL DEFAULT '',          -- active tunnel URL (e.g. *.trycloudflare.com)
  token_enc TEXT NOT NULL DEFAULT '',    -- cloudflared named-tunnel token, ENCRYPTED at rest; NEVER echoed
  mode TEXT NOT NULL DEFAULT '',         -- cloudflare: 'named'|'quick'; tailscale: 'funnel'|'serve'|''
  last_error TEXT NOT NULL DEFAULT '',   -- last start/poll error (surfaced as status detail; no secrets)
  updated_at INTEGER NOT NULL DEFAULT 0
)
```

`internal/store/tunnels.go` (NEW): `Tunnel` struct
`{Type,IsEnabled,Status,URL,Token,Mode,LastError,UpdatedAt}` (`Token` plaintext in
memory, encrypted at rest via `s.cipher` into `token_enc`, mirroring the proxy-pools
`password`/`password_enc` round) + methods:
- `GetTunnel(typ string) (Tunnel, error)` — `ErrNotFound` on `sql.ErrNoRows`.
- `ListTunnels() ([]Tunnel, error)` — both rows; deterministic order (cloudflare,
  tailscale).
- `UpsertTunnel(t Tunnel) error` — `INSERT … ON CONFLICT(type) DO UPDATE` (fixed
  2-key table; mirror the proxy-pools store patterns: `s.cipher.Encrypt(token)`,
  `boolToInt`, `time.Now().Unix()`).
- `SetTunnelState(typ, status, url, lastErr string, enabled bool) error` — the
  state-transition writer the state machine calls after a runner Start/Stop/poll.
- `EnsureTunnelRows() error` (OPTIONAL — §8 ESC-SEED-ROWS) — seed the two
  rows on first migrate so `ListTunnels` always returns 2 (matches the mock's 2-card
  spec). Alternatively the list handler synthesizes the 2 fixed rows from a
  constant + overlays stored state — DECIDE at T-tunnelsstore (default: list
  handler returns the 2 known types, overlaying any stored row → always 2 entries,
  no seed migration needed).

**Secret note:** the cloudflared named-tunnel `token` is encrypted at rest
(`token_enc`) and the read DTO MASKS it (`token_set bool`, NEVER the cleartext) —
mirror the proxy-pools `password`→`password_set` masking. The mock seed has no token
field; the corrected mock seed need not add one.

`internal/admin/tunnels.go` (NEW):

| Handler | Route | Shape (snake_case, `{data}`) | Notes |
|---|---|---|---|
| `ListTunnels` | `GET /api/tunnels` | bare array under `{data}` of `tunnelDTO` (mirror `tunnels.ts` GET). ALWAYS 2 entries (cloudflare, tailscale) | `tunnelDTO{type,is_enabled,url,status}` (+ `token_set` if surfaced; NEVER `token`/`token_enc`). The 4-field Tunnel shape is canonical. |
| `TunnelHealth` | `GET /api/tunnels/health` | `{data:{healthy:bool}}` | `healthy` from the runner status; deterministic via fake in tests (§1.4). The page may not call it (ESC-HEALTH-USE) |
| `EnableTunnel` | `POST /api/tunnels/{type}` | body `{token?, mode?}` (cloudflare named needs a token; quick/tailscale may omit); enables + starts the runner; returns `{data:tunnelDTO}` | 400 on unknown `{type}`; cloudflared named-mode requires a token (else 400 or fall back to quick — ESC-CF-MODE) |
| `DisableTunnel` | `DELETE /api/tunnels/{type}` | stops the runner; returns `{data:tunnelDTO}` (status="inactive") or `{data:{message:"Tunnel disabled"}}` | 400 on unknown `{type}`; idempotent (disabling an inactive tunnel → 200, status stays inactive) |

`{type}` is validated against `{cloudflare, tailscale}`; any other value → 400
`{error:{message:"unknown tunnel type"}}` (the runner map has exactly two keys).

### 1.4 THE CENTRAL DESIGN PROBLEM — injectable `TunnelRunner` (binding, REUSE the SHIPPED Prober/SetProber + w6-j SetShutdownFunc philosophy)

cloudflared and tailscale are **external binaries that CANNOT be downloaded or
spawned in tests** (AGENTS.md "No mocks; use interfaces and fakes; test real
behavior"; w6-j §1.5b "testable WITHOUT performing the external effect"). The admin
handlers + store + the enable/disable/status/health **state machine** MUST be
unit-testable deterministically with NO network, NO binary download, NO process
spawn, NO sleep-on-a-real-process. The mechanism — REUSED VERBATIM from the SHIPPED
proxy-pools `Prober`/`SetProber` seam (`proxypools.go:18,25,36,99-101` +
`handlers.go:86-89`) and w6-j's `SetShutdownFunc` (`handlers.go:78-83`):

**The interface (NEW — `internal/platform/tunnel/runner.go`):**
```go
package tunnel

// Runner abstracts the lifecycle of a single tunnel process (cloudflared or
// tailscale). The REAL impl shells out to the external binary; the TEST impl is a
// deterministic fake returning canned status/URL — so the admin handlers + the
// enable/disable/status/health state machine are unit-tested WITHOUT spawning any
// process or touching the network. Mirrors platform.Prober (proxypools.go:18).
type Runner interface {
    // Start enables the tunnel; for cloudflared-quick it returns the extracted
    // *.trycloudflare.com URL. Returns the resolved public URL (may be "").
    Start(opts StartOpts) (url string, err error)
    // Stop disables/kills the tunnel process. Idempotent.
    Stop() error
    // Status reports the live state without side effects (running, url, lastErr).
    Status() (RunnerStatus, error)
}

type StartOpts struct {
    Type  string // "cloudflare" | "tailscale"
    Token string // cloudflared named-tunnel token (from token_enc); "" → quick tunnel
    Mode  string // "named"|"quick" (cf) | "funnel"|"serve" (ts)
}
type RunnerStatus struct {
    Running bool
    URL     string
    Status  string // "inactive"|"starting"|"active"|"error"
    LastErr string // human-readable; NO secrets
}
```

**The tunnel SERVICE (NEW — `internal/platform/tunnel/service.go` OR a struct in
the package):** holds a `runners map[string]Runner` (keyed `cloudflare`/`tailscale`)
+ the `*store.Store`. The state machine lives HERE and is the unit-test core:
- `Enable(typ, token, mode)` → write `status="starting"` → `runner.Start(opts)` →
  on success write `status="active", url=<extracted>, is_enabled=true,
  last_error=""`; on error write `status="error", last_error=<msg>,
  is_enabled=true` (enabled-but-failing) → persist via `store.SetTunnelState`.
- `Disable(typ)` → `runner.Stop()` → write `status="inactive", url="",
  is_enabled=false` → persist.
- `Status(typ)` / `List()` → overlay `runner.Status()` (or the stored row) → DTO.
- `Health()` → `{healthy: anyRunnerActive}` (or per-spec: healthy iff no enabled
  tunnel is in `error`).

**Injection (binding — NO `New(...)` signature change; mirror SetProxyProber):**
- The service holds the runner map as a FIELD with a REAL default constructed at
  `New` (the real cloudflared/tailscale runners) — exactly as
  `NewProxyPoolService(st)` constructs WITHOUT the prober and falls back to
  `defaultProber` when nil (`proxypools.go:30,99-101`).
- A setter `SetRunner(typ string, r Runner)` (or `SetRunners(map[string]Runner)`)
  overrides the runner(s) for tests — mirror `SetProber` (`proxypools.go:36`).
- On `Handlers`: add a `tunnels *platform/tunnel.Service` (or `*tunnel.Service`)
  FIELD constructed in `New` over the existing `st` (mirror
  `proxyPools: platform.NewProxyPoolService(st)`, `handlers.go:53`) — NO `New(...)`
  signature change. Add `SetTunnelRunner(typ string, r Runner)` on `Handlers`
  forwarding to the service (mirror `SetProxyProber`, `handlers.go:86-89`). Tests
  call `env.handlers.SetTunnelRunner("cloudflare", fakeRunner)` after `newTestEnv`.
- The REAL default runner's constructor wires the real cloudflared/tailscale shell
  impl (§1.5/§1.6); it is the ONLY place a real process is referenced, and it is
  NOT exercised by any unit test.

**What is UNIT-TESTED (deterministic, hermetic — `go test ./...` with NO network /
NO process / NO download / NO real-process sleep):**
- The full state machine via the FAKE runner: enable→`status="active"` +
  url set + persisted; enable-with-fake-error→`status="error"` + last_error +
  persisted; disable→`status="inactive"` + url cleared + persisted; idempotent
  disable; unknown `{type}`→400; health reflects fake runner state.
- The cloudflared **quick-tunnel URL extraction** as a PURE function on canned
  stderr (§1.5) — fully deterministic, the highest-value unit test.
- The tunnel **token never leaks**: token stored `token_enc`; no DTO/response/log
  contains the cleartext token or any ciphertext prefix.
- The store: token round-trips encrypted (raw column ≠ cleartext); upsert/get/list.

**What is INTEGRATION-ONLY (NOT unit-tested — thin, isolated, escalation-recorded —
§1.9):** the real binary download + magic-byte validate; the real
`cloudflared`/`tailscale` process spawn/kill; tailscale install + daemon (TUN) +
login poll + funnel + cert. These live in `cloudflared.go`/`tailscale.go` behind the
`Runner` interface; their bodies shell out and are excluded from `go test ./...`
determinism (guarded so they are never invoked in unit tests; see §5 grep proof
"no-real-spawn-in-test").

### 1.5 cloudflared runner (`internal/platform/tunnel/cloudflared.go`, NEW)

Implements `Runner` for cloudflare. Two modes:
- **Named tunnel:** `cloudflared tunnel run --token <token>` (token from
  `token_enc`). `Start` spawns the process; `Status` reports running.
- **Quick tunnel:** `cloudflared tunnel --url http://localhost:<port>` (no token);
  cloudflared prints the assigned `https://<random>.trycloudflare.com` URL to
  **stderr**. `Start` reads stderr, extracts the URL, returns it.
- **Stop/kill:** terminate the spawned process (`cmd.Process.Kill()` /
  context-cancel), idempotent.

**THE URL EXTRACTION (PURE + UNIT-TESTED — binding).** Factor the
`*.trycloudflare.com` extraction into a PURE function:
```go
// extractQuickTunnelURL scans cloudflared stderr text and returns the assigned
// https://<sub>.trycloudflare.com URL, or ("", false) if absent. PURE — no I/O.
func extractQuickTunnelURL(stderr string) (string, bool)
```
Implemented via a `regexp` matching `https://[a-z0-9-]+\.trycloudflare\.com`.
**Unit tests (deterministic, on CANNED stderr strings — NO process):** a realistic
cloudflared stderr block containing
`... |  https://brave-tree-1234.trycloudflare.com  | ...` → extracts that URL;
stderr with no URL → `("", false)`; multiple lines → first match. The real `Start`
in quick mode calls `extractQuickTunnelURL` over the live stderr pipe — but the
PARSER is what the unit test covers, NOT the spawn.

**Binary download (PAR-PLAT-015 — integration-only, §1.9).** A `ensureBinary()`
helper downloads the platform cloudflared binary to the data-dir, validates the
**magic bytes** (ELF `0x7f454c46` / Mach-O / PE per GOOS/GOARCH) before chmod+exec.
This is NOT unit-tested (no network in tests); the magic-byte VALIDATOR itself MAY
be a pure unit-tested helper (`isValidExecutable(head []byte, goos string) bool`)
on canned byte slices (cheap, deterministic — §8 ESC-MAGICBYTE: default = unit-test
the pure validator, integration-only the download). NEVER spawn/download in `go
test ./...`.

### 1.6 tailscale runner (`internal/platform/tunnel/tailscale.go`, NEW)

Implements `Runner` for tailscale. Operations (all behind `Runner`, the state
machine is fake-tested; the real impl is integration-only — §1.9):
- **Install (PAR-PLAT-019):** fetch/install the tailscale binary (OS-privileged —
  ESCALATED, §1.9/§8 ESC-OS-PRIV). Integration-only.
- **Daemon (PAR-PLAT-020):** start `tailscaled` in **userspace-networking** mode by
  default (no TUN, no root — the server-binary-friendly path) OR TUN mode
  (OS-privileged, escalated). `Start` brings up the daemon + `tailscale up`.
- **Login poll (PAR-PLAT-020):** `tailscale up` emits a login URL; poll
  `tailscale status` until authenticated. The login-URL extraction MAY be a pure
  unit-tested parser on canned output (mirror the cloudflared URL parser — §8
  ESC-TS-URLPARSE; cheap, deterministic). The poll loop itself (with real sleeps) is
  integration-only.
- **Funnel (PAR-PLAT-021):** `tailscale funnel <port>` to expose publicly; the
  public `*.ts.net` URL is reported via `Status().URL`.
- **Cert (PAR-PLAT-022):** `tailscale cert <host>` to provision a TLS cert.
  Integration-only.
- **Stop:** `tailscale down` / stop the daemon. Idempotent.

**Default mode (binding default — §8 ESC-TS-MODE):** userspace-networking (no TUN,
no root) so the server binary can run a tunnel without OS privilege; TUN mode is an
escalated opt-in (§1.9). The state machine treats tailscale identically to
cloudflare via the `Runner` interface — enable/disable/status/health are
fake-tested; the privileged install/TUN parts are integration-only.

### 1.7 health endpoint (`GET /api/tunnels/health`)

`{data:{healthy:bool}}`. `healthy` = derived from the runner statuses:
RECOMMENDED default = `true` iff every ENABLED tunnel reports a non-`error` status
(an all-disabled gateway is healthy; an enabled-but-erroring tunnel is unhealthy).
Deterministic in unit tests via the fake runner's canned `RunnerStatus`. The page
may not call `/health` (VERIFY at P4 — §8 ESC-HEALTH-USE); it is proven by the Go
admin test regardless.

### 1.8 routes_admin.go registration (serial-slot additive, §3)

Add (additive appends; static/deeper-before-param precedence honored by the file —
the `/health` static route MUST precede the `{type}` param route, mirroring the
`/api/proxy-pools/batch`-before-`{id}` precedent the SHIPPED w7-plat-1 set at
`routes_admin.go:133-134`, and the `…/{id}/test` deeper-route pattern at `:137`):
```go
// Tunnels (static collection + /health before {type} param).
r.GET("/api/tunnels", h.RequireSession(h.ListTunnels))
r.GET("/api/tunnels/health", h.RequireSession(h.TunnelHealth))   // static BEFORE {type}
r.POST("/api/tunnels/{type}", h.RequireSession(h.EnableTunnel))
r.DELETE("/api/tunnels/{type}", h.RequireSession(h.DisableTunnel))
```
Route-precedence note: `/api/tunnels/health` (static) vs `/api/tunnels/{type}`
(param) follow the file's existing static/deeper-before-param ordering. A genuine
`fasthttp/router` collision (`health` matching the `{type}` slot) is §8 ESC-ROUTE,
not a silent path change. The 4 lines append BELOW the proxy-pools block
(`routes_admin.go:131-137`). Diff bound §5: the route block is ONE commit, additive
only.

### NOT in scope (explicit)

- **No UI page/component/route/store edits** — `/tunnels` and all w6-m components are
  FROZEN consume-only (decision 8). The ONLY UI-tree touches are the tunnels
  mock-body + seed corrections (§1.2 / §3), and ONLY if a real Go-vs-mock divergence
  exists (default: verify, correct minimally).
- **No edits to other platform domains** — proxy-pools (w7-plat-1, SHIPPED — CONSUME
  the injection precedent, do NOT edit its files), mitm (w7-plat-3) are disjoint.
- **No edits to `internal/server/guard.go`** — the `tunnelDashboardAccess` settings
  guard (`guard.go:135-141`) is CONSUMED/coexisted-with, NEVER edited.
- **No edits to pre-existing admin handlers' bodies** — apikeys, virtualkeys,
  providers*, connections, combos, auth, version, usage, **proxypools** are
  FORBIDDEN. The ONLY `handlers.go` touch is the ADDITIVE `tunnels` service field +
  the `SetTunnelRunner` setter (mirroring the SHIPPED `proxyPools` field +
  `SetProxyProber` — NOT a frozen handler body; NO `New(...)` signature change).
- **No edits to inference (`selection.go`/`factory.go`/`runner.go`)** — tunnels are
  NOT an inference-path concern; w7-plat-2 holds NO selection.go micro-serial.
- **No interface change** — `New(...)` signature PRESERVED (additive setters / a
  default-constructed field only; MAP decision 9).
- **No destructive DDL / column renames** — additive `ensureTable`/`ensureColumn`
  ONLY (decision 2).
- **No new global state** — handlers compose `h.store`; the tunnel service is
  constructed over `h.store` and holds its runner map as a field.
- **No secret exposure** — tunnel `token` `*_enc` at rest + MASKED in responses
  (`token_set`, NEVER `token`); status/health/error responses carry no secrets;
  `last_error` is scrubbed of any token (§5 grep proofs).
- **No real binary download / process spawn / network / OS-privileged op in any
  unit test** — those paths are integration-only behind `Runner` (§1.9); the unit
  suite is fully hermetic.

---

## 2. Precondition checks

Run all before any edit; abort and report to orchestrator on any failure.

```bash
# P0 — clean tree; record <base>
git status --porcelain     # expect empty EXCEPT a possibly-dirty ui/dist/index.html
                           # (gitignored build artifact — NEVER stage it, NEVER revert it).
                           # If anything ELSE is dirty, STOP. Use explicit `git add <file>`, never -A.
git rev-parse HEAD         # record as <base> for §5

# P1 — the gap is REAL (no Go for tunnels)
grep -nE '/api/tunnels' internal/server/routes_admin.go ; echo "^ expect EMPTY"
grep -rniE '"/api/tunnels|TunnelHandler|ListTunnels' internal/ cmd/ ; echo "^ expect EMPTY (no tunnel Go)"
test ! -e internal/store/tunnels.go && test ! -e internal/admin/tunnels.go && echo "tunnel admin/store gap OK"
test ! -d internal/platform/tunnel && echo "platform/tunnel pkg gap OK"
grep -nE 'tunnels' internal/store/migrate.go ; echo "^ expect EMPTY (no tunnels table)"

# P2 — the SHIPPED injection precedent to MIRROR is present (w7-plat-1)
grep -n "type Prober\|func (s \*ProxyPoolService) SetProber\|func NewProxyPoolService\|defaultProber" internal/platform/proxypools.go
grep -n "proxyPools\|func (h \*Handlers) SetProxyProber\|func (h \*Handlers) SetShutdownFunc\|func New(" internal/admin/handlers.go
grep -n "platform.NewProxyPoolService(st)" internal/admin/handlers.go   # the New-constructs-field precedent

# P3 — reused surfaces present
grep -n "func writeData\|func writeError" internal/admin/respond.go
grep -n "func pathID" internal/admin/handlers.go
grep -n "func (h \*Handlers) recordAudit" internal/admin/audit.go
grep -n "func newTestEnv\|func call(" internal/admin/admin_test.go
grep -n "s.cipher.Encrypt\|s.cipher.Decrypt\|password_enc" internal/store/proxypools.go   # *_enc round precedent

# P4 — migrate pattern + secret-at-rest precedent + the tunnel guard (consume, not edit)
grep -n "tables := \|CREATE TABLE IF NOT EXISTS proxy_pools\|password_enc\|ensureColumn(db, col" internal/store/migrate.go | head
grep -n "tunnelDashboardAccess\|tunnelUrl\|tailscaleUrl" internal/server/guard.go   # 135-141 — CONSUME, never edit

# P5 — the W6-m UI + spec present (consume-only) and the mock to mirror
test -f ui/e2e/tunnels.spec.ts && echo "spec present"
test -f ui/e2e/mocks/handlers/tunnels.ts && test -f ui/e2e/mocks/seed/tunnels.ts && echo "mock+seed present"
grep -n "trycloudflare\|healthy\|is_enabled\|status\|/api/tunnels" ui/e2e/mocks/handlers/tunnels.ts ui/e2e/mocks/seed/tunnels.ts
grep -nE "tunnels/health|/api/tunnels|trycloudflare" ui/src/**/*.ts* 2>/dev/null | head   # what the PAGE actually calls (resolves ESC-HEALTH-USE)

# P6 — routes_admin.go serial slot FREE (released by w7-plat-1)
git log --oneline -5 -- internal/server/routes_admin.go    # last touch = w7-plat-1 (merged); slot free
# Orchestrator MUST confirm: no concurrent W7 plan holds an unmerged routes_admin.go edit before T-routes.

# P7 — green at base
go test ./... && go vet ./... && go build ./...     # exit 0 (Go untouched-green; HERMETIC — no net/process)
cd ui && npm run build                               # exit 0 (build BEFORE e2e — e2e-hygiene)
cd ui && npx playwright test e2e/tunnels.spec.ts     # PASS at base against the W6 mock; record in WORKFLOW.md
```

---

## 3. Exclusive file ownership

After w7-plat-2 merges, all CREATE files are owned by w7-plat-2; later plans consume,
never edit (MAP decision 7).

**CREATE — store (NEW):**

| File | Contract |
|---|---|
| `internal/store/tunnels.go` | `Tunnel` struct + `GetTunnel`/`ListTunnels`/`UpsertTunnel`/`SetTunnelState` (+ optional `EnsureTunnelRows`); `s.cipher.Encrypt/Decrypt` for `token`↔`token_enc`; `boolToInt`, `time.Now().Unix()`, `ErrNotFound`. Mirrors `proxypools.go`/`connections.go` `*_enc` round. |
| `internal/store/tunnels_test.go` | Table-driven, temp `store.Open`: upsert→get→list(=2 deterministic order)→state-transition; **token round-trips encrypted (raw `token_enc` column ≠ cleartext)**. RED first. |

**EXTEND — store (additive only):**

| File | Change (additive ONLY) |
|---|---|
| `internal/store/migrate.go` | ADD the `tunnels` table to the `tables` slice (mirror the SHIPPED `proxy_pools` block @ :191). ADDITIVE ONLY — no DROP/RENAME. |

**CREATE — domain (NEW package `internal/platform/tunnel`):**

| File | Contract |
|---|---|
| `internal/platform/tunnel/runner.go` | `Runner` interface (`Start/Stop/Status`) + `StartOpts` + `RunnerStatus` (§1.4). No `init()`; errors-as-values. The injectable seam. |
| `internal/platform/tunnel/service.go` | `Service` (or struct): `runners map[string]Runner` + `*store.Store`; `Enable/Disable/Status/List/Health` state machine; `NewService(st)` constructs REAL default runners; `SetRunner(typ, r)` overrides for tests (mirror `SetProber` `proxypools.go:36`). Persists via `store.SetTunnelState`. No `init()`. |
| `internal/platform/tunnel/cloudflared.go` | `cloudflaredRunner` implementing `Runner`: named (`tunnel run --token`) + quick (`tunnel --url`) + Stop/kill; `extractQuickTunnelURL(stderr) (string,bool)` PURE; `ensureBinary` + magic-byte validate (integration-only, §1.9). |
| `internal/platform/tunnel/tailscale.go` | `tailscaleRunner` implementing `Runner`: install/daemon(userspace default)/login-poll/funnel/cert/Stop (integration-only real impl, §1.9); optional pure login-URL parser. |
| `internal/platform/tunnel/service_test.go` | State machine via a FAKE `Runner`: enable→active+url+persist; enable-error→error+last_error; disable→inactive; idempotent disable; unknown type; health from fake status. RED first. Deterministic, NO process/network. |
| `internal/platform/tunnel/cloudflared_test.go` | `extractQuickTunnelURL` on CANNED stderr (has-URL → extracts `*.trycloudflare.com`; no-URL → false; multi-line → first); `isValidExecutable` magic-byte validator on canned bytes (ESC-MAGICBYTE). RED first. NO spawn/download. |

**CREATE — transport (NEW):**

| File | Contract |
|---|---|
| `internal/admin/tunnels.go` | `ListTunnels`/`TunnelHealth`/`EnableTunnel`/`DisableTunnel` + `tunnelDTO` + request/validate; `writeData`/`writeError`; 400 on unknown `{type}`; `h.recordAudit` after enable/disable (best-effort). NEVER echoes `token`. Reads `{type}` via `ctx.UserValue("type")`. |
| `internal/admin/tunnels_test.go` | via `newTestEnv` + `SetTunnelRunner(typ, fakeRunner)`: list→2 entries; enable cloudflare (fake)→active+url; disable→inactive; enable unknown type→400; health→`{healthy}`; **no response leaks `token`/`token_enc`**; an audit entry on enable. RED first. Deterministic — fake runner, NO process/network. |

**MODIFY — handlers wiring (additive only — mirror the SHIPPED proxyPools/SetProxyProber):**

| File | Change |
|---|---|
| `internal/admin/handlers.go` | ADDITIVE: add a `tunnels *tunnel.Service` field; construct it in `New` via `tunnel.NewService(st)` (mirror `proxyPools: platform.NewProxyPoolService(st)` @ :53). ADD `SetTunnelRunner(typ string, r tunnel.Runner)` forwarding to `h.tunnels.SetRunner` (mirror `SetProxyProber` @ :86-89). NO `New(...)` signature change. |

**MODIFY — serial-slot route registration (additive only):**

| File | Change |
|---|---|
| `internal/server/routes_admin.go` | ADD the 4 route lines (§1.8) BELOW the proxy-pools block (:131-137). NOTHING else. ONE commit. SERIAL SLOT — only holder while live; RELEASE to w7-plat-3 on close. |

**MODIFY — e2e mock corrections (mirror real Go, decision 1 — ONLY on real divergence):**

| File | Change |
|---|---|
| `ui/e2e/mocks/handlers/tunnels.ts` (BODY) | VERIFY the mock body matches the Go `{data}` envelope + DTO field names; correct ONLY a real divergence (`/health` shape, POST/DELETE body) — the spec asserts only request method/path, not bodies. Do NOT add fields the page ignores. (ESC-MOCK if a correction reds a spec.) |
| `ui/e2e/mocks/seed/tunnels.ts` (BODY) | Already the 4-field `Tunnel` shape (2 rows, cloudflare `url` contains `trycloudflare.com`) — verify; correct only on field-name divergence. No token field needed. |

**FORBIDDEN:** everything else. Explicitly: all pre-existing `internal/admin/*.go`
except the NEW tunnels files + the ADDITIVE handlers.go field/setter; all
`internal/store/*.go` except tunnels (NEW) + migrate (additive table); all
pre-existing `internal/platform/*` (proxypools/outboundproxy — SHIPPED, CONSUME the
precedent, do NOT edit); `internal/server/guard.go` (CONSUME the tunnel guard, no
edit); all `internal/inference/*` (tunnels are not an inference concern); all UI
`ui/src/**` (FROZEN, decision 8); all other mocks/seeds/specs;
`ui/package.json` + lockfile; `ui/vite.config.ts`; `ui/playwright.config.ts`;
`ui/dist/**` (gitignored — NEVER stage, NEVER revert `ui/dist/index.html`). Touching
any of these is an automatic REJECT.

---

## 4. TDD tasks

Cadence (strict, AGENTS.md "TDD always: write test first, see it fail, write minimum
code to pass"): **no Go impl file may exist before its `_test.go` is committed RED.**
`go test ./... && go vet ./... && go build ./...` green at EVERY commit, FULLY
HERMETIC (no network, no binary download, no process spawn, no real-process sleep).
The e2e spec stays green throughout (real Go is additive; mock corrections — if any —
mirror it). Order: runner interface + URL-extract pure tests → tunnels store →
service state machine → admin handlers → routes serial slot → mock verify/correct →
closeout.

### T-runner — STEP(a) RED, STEP(b) impl (the pure parser + interface)
STEP(a): write `internal/platform/tunnel/runner.go` (interface + types — compiles)
and `internal/platform/tunnel/cloudflared_test.go` for `extractQuickTunnelURL`
(canned stderr) + `isValidExecutable` (canned magic bytes).
`go test ./internal/platform/tunnel/ -run 'URL|Executable'` → FAIL. Commit RED:
`phase-1/w7-plat-2: failing cloudflared URL-extract + magic-byte tests (TDD red)`.
STEP(b): implement `extractQuickTunnelURL` + `isValidExecutable` (the pure helpers)
in `cloudflared.go` (process/download bodies stubbed/integration-only). Gates green.
Commit: `phase-1/w7-plat-2: cloudflared quick-tunnel URL extraction + magic-byte validate`.

### T-tunnelsstore — STEP(a) RED store, STEP(b) impl
STEP(a): write `internal/store/tunnels_test.go`; ADD the `tunnels` table to
`migrate.go` (so the test compiles + the table exists).
`go test ./internal/store/ -run Tunnel` → FAIL. Commit RED:
`phase-1/w7-plat-2: failing tunnels store tests (TDD red)`.
STEP(b): implement `internal/store/tunnels.go` (upsert/get/list/state +
`token`↔`token_enc` via `s.cipher`). Gates green. Commit:
`phase-1/w7-plat-2: tunnels store (token *_enc at rest)`.

### T-service — STEP(a) RED state machine, STEP(b) impl (fake runner)
STEP(a): write `internal/platform/tunnel/service_test.go` against a FAKE `Runner`
(enable/disable/status/health/error/idempotent/unknown-type). → FAIL. Commit RED:
`phase-1/w7-plat-2: failing tunnel service state-machine tests (TDD red)`.
STEP(b): implement `service.go` (state machine + `NewService(st)` real-default
runners + `SetRunner` test override) + thin `cloudflared.go`/`tailscale.go` `Runner`
impls (real process/install/funnel/cert bodies = integration-only, §1.9; guarded so
never invoked in unit tests). Gates green (fake runner only). Commit:
`phase-1/w7-plat-2: tunnel service state machine + injectable runner`.

### T-admin — STEP(a) RED handlers, STEP(b) impl
STEP(a): write `internal/admin/tunnels_test.go` (via `newTestEnv` +
`SetTunnelRunner` fake): list=2, enable→active+url, disable→inactive, unknown→400,
health, no-token-leak, audit-on-enable. ADD the ADDITIVE `handlers.go` field +
`SetTunnelRunner` (so the test compiles). → FAIL. Commit RED:
`phase-1/w7-plat-2: failing tunnels admin handler tests (TDD red)`.
STEP(b): implement `internal/admin/tunnels.go` (CRUD-ish handlers + DTO + 400 guard
+ `h.recordAudit`). Gates green. Commit:
`phase-1/w7-plat-2: tunnels admin API (list/health/enable/disable)`.

### T-routes — serial-slot route registration
TAKE the routes_admin.go serial slot (orchestrator confirms FREE at P6 — released by
w7-plat-1). Add the 4 route lines (§1.8). Gates green. Commit (ONE commit touches the
serial file):
`phase-1/w7-plat-2: register tunnels admin routes (serial slot)`.

### T-mocks — mock-body verify/correct (mirror real Go, decision 1)
VERIFY `tunnels.ts` + seed match the Go `{data}` envelope + 4-field DTO; correct ONLY
a real divergence (`/health` shape; POST/DELETE body) — do NOT add fields the page
ignores. Gates: `cd ui && npm run build` green (BEFORE playwright);
`npx playwright test e2e/tunnels.spec.ts` green (still). If a correction reds a
non-w7-plat-2 spec, STOP + ESCALATE (§8 ESC-MOCK). Commit (only if a change is made):
`phase-1/w7-plat-2: correct tunnels mock to mirror real Go DTOs`.

### T-close — full gates + closeout
```bash
go test ./... && go vet ./... && go build ./...                      # HERMETIC — no net/process
go test ./internal/platform/tunnel/... ./internal/admin/ -run Tunnel -v
go test ./internal/store/ -run Tunnel -v
cd ui && npm run build                                               # BEFORE playwright (e2e-hygiene)
cd ui && npx playwright test e2e/tunnels.spec.ts                     # green (ISOLATED, no concurrent playwright)
cd ui && npx playwright test                                        # full suite green (mod the known pre-existing comprehensive.spec flake — open-questions w7-plat-1 §115)
```
Flip the matrix: PAR-PLAT-015..023 → HAVE (real Go; binary-download/spawn/OS-privileged
parts integration-only + escalation-recorded — §1.9 / §8); PAR-UI-112/113/114 PARTIAL
→ HAVE. Mark `open-questions.md` w6-m ESC-1c RESOLVED with a cite; append any new open
items (§8 — OS-privileged tailscale install/TUN escalation; binary-download
integration-only note; ESC-CF-MODE/ESC-TS-MODE decisions). Update `docs/WORKFLOW.md`
(P7 base observation; the ESC-SCHEMA / ESC-CF-MODE / ESC-TS-MODE / ESC-HEALTH-USE
decisions; the serial-slot take/release; the mock verify/correct). Final commit:
`phase-1/w7-plat-2: close — tunnels (cloudflared+tailscale) Go; matrix flip; mock mirror`.
**On the close commit, RELEASE the routes_admin.go serial slot to w7-plat-3.**

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0. Diff gate is
**w7-plat-2 commit-range-scoped** (§7).

**Test gates (HERMETIC — no network, no binary download, no process spawn, no
real-process sleep)**
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/platform/tunnel/... ./internal/admin/ -run Tunnel -v` → exit 0,
  all pass (URL-extract ≥3 cases; magic-byte ≥2; state machine ≥6 incl error +
  idempotent + unknown-type; admin list=2/enable/disable/health/no-token-leak).
- `go test ./internal/store/ -run Tunnel -v` → exit 0 (incl token encrypted-at-rest).
- `cd ui && npm run build` → exit 0 (BEFORE playwright).
- `cd ui && npx playwright test e2e/tunnels.spec.ts` → exit 0, all pass, 0 skipped
  (ISOLATED; no concurrent playwright; `ui/dist/index.html` NEVER reverted).
- `cd ui && npx playwright test` → exit 0 mod the known pre-existing
  `comprehensive.spec.ts` flake (open-questions w7-plat-1 §115); no tunnels-related
  green-at-base spec goes red.

**TDD-order proof** — each impl file's covering test appears in an earlier-or-equal
commit:
```bash
for pair in \
  "internal/platform/tunnel/cloudflared_test.go:internal/platform/tunnel/cloudflared.go" \
  "internal/platform/tunnel/service_test.go:internal/platform/tunnel/service.go" \
  "internal/store/tunnels_test.go:internal/store/tunnels.go" \
  "internal/admin/tunnels_test.go:internal/admin/tunnels.go" ; do
  tf=${pair%%:*}; im=${pair##*:}
  ct=$(git log --format=%ct --diff-filter=A -1 -- "$tf")
  cf=$(git log --format=%ct --diff-filter=A -1 -- "$im")
  [ "$ct" -le "$cf" ] || echo "TDD VIOLATION: $im"     # prints nothing
done
```

**Grep proofs**
```bash
# tunnels transport + store
grep -n "func (h \*Handlers) ListTunnels\|TunnelHealth\|EnableTunnel\|DisableTunnel" internal/admin/tunnels.go
grep -n "type\|is_enabled\|url\|status\|healthy" internal/admin/tunnels.go        # canonical 4-field DTO + health
grep -n "func (s \*Store) GetTunnel\|ListTunnels\|UpsertTunnel\|SetTunnelState" internal/store/tunnels.go
grep -n "writeData\|writeError\|recordAudit" internal/admin/tunnels.go             # envelope + audit
grep -n "ctx.UserValue(\"type\")\|unknown tunnel type\|400\|StatusBadRequest" internal/admin/tunnels.go  # {type} guard
# runner injection seam (mirrors SHIPPED Prober/SetProber + w6-j SetShutdownFunc)
grep -n "type Runner interface\|func.*SetRunner\|func NewService" internal/platform/tunnel/*.go
grep -n "tunnels\b\|func (h \*Handlers) SetTunnelRunner\|tunnel.NewService(st)" internal/admin/handlers.go
# cloudflared URL extraction (pure)
grep -n "func extractQuickTunnelURL\|trycloudflare" internal/platform/tunnel/cloudflared.go
# routes
grep -nE '/api/tunnels' internal/server/routes_admin.go
# no init(); no global state
! grep -rn "func init(" internal/admin/tunnels.go internal/store/tunnels.go internal/platform/tunnel/*.go && echo "no init() OK"
```

**No-real-spawn / no-download-in-test proofs (binding — the hermeticity guarantee)**
```bash
# unit tests NEVER spawn the external binary nor download nor sleep on a real process:
! grep -nE 'exec\.Command|os/exec|http\.Get|http\.Client|\.Download|cmd\.Start|cmd\.Run' \
   internal/platform/tunnel/service_test.go internal/platform/tunnel/cloudflared_test.go \
   internal/admin/tunnels_test.go internal/store/tunnels_test.go && echo "no real spawn/download/net in tests OK"
# the real spawn/download lives ONLY in the runner impls (cloudflared.go/tailscale.go), behind Runner:
grep -nE 'exec\.Command|os/exec' internal/platform/tunnel/cloudflared.go internal/platform/tunnel/tailscale.go  # expect MATCHES here only
# the fake runner used by tests implements Runner without any process:
grep -n "Runner\b" internal/platform/tunnel/service_test.go internal/admin/tunnels_test.go  # fake impl present
```

**No-secret-exposure proofs (binding)**
```bash
# token/cleartext never appears in any tunnel DTO/response field
! grep -nE 'json:"token"' internal/admin/tunnels.go && echo "no token json field OK"
grep -nA10 'type tunnelDTO struct' internal/admin/tunnels.go ; echo "^ must NOT contain token (only token_set, if any)"
# token encrypted at rest
grep -n "token_enc\|s.cipher.Encrypt\|s.cipher.Decrypt" internal/store/tunnels.go
# runtime no-leak: tunnels_test.go marshals every tunnel response + asserts it
# contains neither the cleartext token nor any ciphertext prefix; last_error carries no token.
# additive migrations only (no DROP/RENAME introduced by this plan)
git diff <base>..HEAD -- internal/store/migrate.go | grep -E '^\+' | grep -iE 'DROP COLUMN|RENAME COLUMN|DROP TABLE' | wc -l   # = 0
```

**Negative / freeze proofs (w7-plat-2 commit-range — §7)**
```bash
R="<first-w7-plat-2>^..<last-w7-plat-2>"
# Only the sanctioned Go files changed:
git diff $R --name-only -- internal/ | grep -vE \
 'internal/store/(tunnels|migrate)(_test)?\.go|internal/platform/tunnel/.*\.go|internal/admin/(tunnels)(_test)?\.go|internal/admin/handlers\.go|internal/server/routes_admin\.go' \
 | wc -l                                                                  # = 0
# Frozen admin handlers untouched (incl the SHIPPED proxypools):
git diff $R --name-only -- internal/admin/auth.go internal/admin/apikeys.go internal/admin/virtualkeys.go internal/admin/providers.go internal/admin/connections.go internal/admin/combos.go internal/admin/proxypools.go internal/admin/version.go | wc -l   # = 0
# Frozen guard (tunnel guard CONSUMED, not edited) untouched:
git diff $R --name-only -- internal/server/guard.go | wc -l              # = 0
# SHIPPED platform proxy-pools untouched (precedent consumed, not edited):
git diff $R --name-only -- internal/platform/proxypools.go internal/platform/outboundproxy.go internal/store/proxypools.go | wc -l   # = 0
# inference untouched (tunnels are not an inference concern):
git diff $R --name-only -- internal/inference/ | wc -l                   # = 0
# handlers.go = additive only (no deletions of existing logic):
git diff $R -- internal/admin/handlers.go | grep -E '^-' | grep -v '^---' | wc -l   # = 0
# UI frozen except the sanctioned mock/seed bodies (if touched at all):
git diff $R --name-only -- ui/ | grep -vE \
 'ui/e2e/mocks/handlers/tunnels\.ts|ui/e2e/mocks/seed/tunnels\.ts' | wc -l   # = 0
git diff $R --name-only -- ui/src/ | wc -l                               # = 0 (src frozen)
git diff $R --name-only -- ui/dist/ | wc -l                             # = 0 (dist gitignored/never staged)
# routes_admin.go = exactly ONE commit, additive:
git log --oneline $R -- internal/server/routes_admin.go | wc -l          # = 1
git diff $R -- internal/server/routes_admin.go | grep -E '^-' | grep -v '^---' | wc -l   # = 0 (no deletions)
```

---

## 6. Out of scope (restated, binding)

No UI src edits (decision 8 — pages/components/routes/stores frozen); only the
sanctioned tunnels mock-body + seed corrections, and ONLY on a real Go-vs-mock
divergence. No edits to pre-existing admin handler bodies (auth, apikeys,
virtualkeys, providers*, connections, combos, version, usage, **proxypools**). No
edits to the SHIPPED `internal/platform/{proxypools,outboundproxy}.go` (CONSUME the
`Prober`/`SetProber`/`NewService`-field injection precedent, do NOT edit). No
`guard.go` edit (the `tunnelDashboardAccess` guard @ :135-141 is CONSUMED/
coexisted-with, never modified). No inference edits (tunnels are not an
inference-path concern — NO selection.go micro-serial). No interface / `New(...)`
signature change (additive setters / default-constructed field only). No JWT. No
destructive DDL — additive `ensureTable`/`ensureColumn` only. No new global state. No
other platform domains (proxy-pools SHIPPED, mitm w7-plat-3). **No real binary
download / process spawn / network / OS-privileged op in any unit test** — those are
integration-only behind `Runner` (§1.9); the unit suite is fully hermetic. No secret
exposure (tunnel token `*_enc` + masked `token_set`; status/health/error responses +
`last_error` carry no token). Mock-vs-Go contradiction → escalate (§8), never fudge a
mock or edit a frozen handler. NEVER revert `ui/dist/index.html`; NEVER run
concurrent playwright; `npm run build` before e2e (e2e-hygiene).

## 7. Diff-gate scope

W7 platform plans (plat-1 SHIPPED / plat-2 / plat-3) commit to main concurrently, so
a broad `<base>..HEAD` range sweeps in sibling commits. The diff gate MUST be scoped
to w7-plat-2's own commits. Isolate them:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w7-plat-2:" | awk '{print $1}'`
then `git diff <first-w7-plat-2>^..<last-w7-plat-2> -- [file list]`.

`git diff <first>^..<last> --name-only` must be exactly a subset of:
```
internal/store/tunnels.go
internal/store/tunnels_test.go
internal/store/migrate.go                 (additive tunnels table; ONE concern)
internal/platform/tunnel/runner.go
internal/platform/tunnel/service.go
internal/platform/tunnel/service_test.go
internal/platform/tunnel/cloudflared.go
internal/platform/tunnel/cloudflared_test.go
internal/platform/tunnel/tailscale.go
internal/admin/tunnels.go
internal/admin/tunnels_test.go
internal/admin/handlers.go                (ADDITIVE tunnels field + SetTunnelRunner; no New() sig change)
internal/server/routes_admin.go           (serial-slot additive routes; ONE commit)
ui/e2e/mocks/handlers/tunnels.ts          (body only — verify/correct on real divergence)
ui/e2e/mocks/seed/tunnels.ts              (verify; correct only on divergence)
.planning/parity/matrix/*                  (row flips)
.planning/parity/plans/open-questions.md
docs/WORKFLOW.md
```
Any file outside this list in the scoped diff is an automatic review REJECT.
`internal/server/guard.go`, the SHIPPED `internal/platform/{proxypools,outboundproxy}.go`,
the pre-existing admin handlers, `internal/inference/**`, and all `ui/src/**` are
deliberately ABSENT — touching them is an automatic REJECT. The `routes_admin.go`
edit must appear in exactly ONE commit (§5); the serial slot is released to
w7-plat-3 on close.

## 8. Escalations / decisions (explicit — recommended defaults, do not fabricate)

- **ESC-OS-PRIV (ESCALATED — tailscale install / TUN + cloudflared binary download,
  recommended default).** tailscale **install** (PAR-PLAT-019) and **TUN-mode
  daemon** (PAR-PLAT-020) are OS-privileged (package install / root / `/dev/net/tun`)
  and ill-suited to a server binary's unit tests; the cloudflared **binary download**
  (PAR-PLAT-015) needs network. **Decision: the parity bar is the admin API + state
  machine + injectable runner (FULLY unit-tested via the fake); the real binary
  download / process spawn / install / TUN are a THIN real impl behind `Runner` that
  is NOT unit-tested (integration-only).** RECOMMENDED tailscale default =
  **userspace-networking** (no TUN, no root — the server-friendly path); TUN is an
  escalated opt-in. Record this disposition in `open-questions.md` at closeout:
  PAR-PLAT-015/019/020(TUN) marked HAVE with an "integration-only / OS-privileged"
  footnote. Flag for orchestrator confirmation.
- **ESC-SCHEMA (RESOLVED at authoring — typed vs JSON columns, binding default).**
  The `tunnels` row is a fixed small shape with a `WHERE type=?` lookup over exactly
  2 keys. **Decision: typed columns** (clean lookup; matches the SHIPPED proxy-pools
  + gov-plan precedent of typed columns for fixed-shape domains). Flag for
  confirmation.
- **ESC-CF-MODE (RESOLVED at authoring — cloudflared named vs quick, binding
  default).** `POST /api/tunnels/cloudflare` with a `token` → **named** tunnel
  (`tunnel run --token`); without a token → **quick** tunnel (`tunnel --url`,
  extract `*.trycloudflare.com`). **Decision: support both; token-presence selects
  the mode** (or an explicit `mode` in the body wins). The seed `url` containing
  `trycloudflare.com` implies quick mode is the demoed path. Flag for confirmation.
- **ESC-TS-MODE (RESOLVED at authoring — tailscale userspace vs TUN, binding
  default).** **Decision: userspace-networking default** (no TUN, no root); TUN is an
  escalated opt-in (ESC-OS-PRIV). Flag.
- **ESC-MAGICBYTE (RESOLVED at authoring — magic-byte validator surface, binding
  default).** **Decision: factor a PURE `isValidExecutable(head []byte, goos string)
  bool` validator (unit-tested on canned bytes — ELF/Mach-O/PE) out of the
  integration-only download path.** The DOWNLOAD is integration-only; the VALIDATOR
  is a cheap deterministic unit test. Flag.
- **ESC-SEED-ROWS (RESOLVED at authoring — how `ListTunnels` always returns 2,
  binding default).** **Decision: the list handler returns the 2 known types
  (cloudflare, tailscale) overlaying any stored row → always exactly 2 entries**
  (matches the spec's 2-card assertion) WITHOUT a seed migration. Alternative:
  `EnsureTunnelRows` seeds 2 rows on migrate. RECOMMENDED: the overlay (no migration
  side-effect). Flag.
- **ESC-HEALTH-USE (CONDITIONAL — does the page call `/health`).** VERIFY at P5 via
  `grep -nE 'tunnels/health' ui/src`. If the page never calls it, `/health` is a
  parity endpoint proven by the Go admin test only (the spec asserts only
  list/POST/DELETE). Default: ship `/health` (cheap, mirrors the mock) regardless.
  Flag if the spec surprises.
- **ESC-GUARD-SETTINGS (CONDITIONAL — write `settings["tunnelUrl"]` on enable).** The
  `guard.go:135-141` host-access guard reads `settings["tunnelUrl"]`/`["tailscaleUrl"]`.
  w7-plat-2 MAY write those settings when a tunnel goes active so the guard sees the
  live host. **Default: do NOT write settings in w7-plat-2** (keep the surface
  minimal; the guard is forward-looking and the operator sets those settings
  manually today) — record as a follow-up. NEVER edit `guard.go`. Flag.
- **ESC-MOCK (CONDITIONAL — mock ripple).** `tunnels.ts` is w6-m-owned and
  tunnels-only (not shared). VERIFY the mock body matches the Go `{data}` envelope +
  4-field DTO; correct ONLY a real divergence. If a body correction reds a
  non-w7-plat-2 spec or a seed correction ripples, STOP and ESCALATE for orchestrator
  serialization — no fudge, no frozen-branch edit.
- **ESC-ROUTE (CONDITIONAL — fasthttp/router precedence).** `/api/tunnels/health`
  (static) vs `/api/tunnels/{type}` (param) follow the file's
  static/deeper-before-param precedent (the SHIPPED `/api/proxy-pools/batch`-before-
  `{id}` @ `routes_admin.go:133-134`). If the matcher mis-disambiguates (`health`
  caught by `{type}`) or panics on a conflict, STOP and ESCALATE for a path
  arrangement — never silently diverge page/mock/Go.
- **ESC-ARCH (CONDITIONAL — layering).** Per the w7-gov-1 ESC-ARCH finding
  (open-questions §95: no in-tree arch test enforces transport→domain→repository;
  proxypools/teams/virtualkeys layering is by convention), build the
  `internal/platform/tunnel` domain service because the state machine + runner
  injection warrant a seam (reused beyond the handler — exactly as proxy-pools got
  `platform/proxypools.go`). Do NOT pre-guess a stricter rule; follow the SHIPPED
  proxy-pools precedent.
- **Serial-slot dependency (§1.8 / P6).** w7-plat-2 TAKES the routes_admin.go slot
  in chain order (… → w7-plat-1 → **w7-plat-2** → w7-plat-3 → w7-misc; w7-plat-1
  released it on its close — open-questions w7-plat-1 §116) and RELEASES it to
  w7-plat-3 on close. NO selection.go micro-serial (tunnels do not touch inference).
  Orchestrator confirms exactly one unmerged routes_admin.go holder (decision 3 /
  MAP §219-224) before T-routes.
```
