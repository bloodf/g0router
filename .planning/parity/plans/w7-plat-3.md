# Micro-plan w7-plat-3 — MITM backend (root CA + HTTPS MITM proxy + per-tool admin) (Go)

```
wave: 7
plan: w7-plat-3
status: READY (rev 1 — authored against merged Waves 0–6 + the shipped W7
  governance plans + the SHIPPED w7-plat-1 (proxy-pools) and SHIPPED w7-plat-2
  (tunnels). At authoring, w7-plat-1 + w7-plat-2 are LIVE in-tree:
  internal/platform/{proxypools,outboundproxy}.go, internal/platform/tunnel/*.go,
  internal/admin/{proxypools,tunnels}.go, internal/store/{proxypools,tunnels}.go,
  the proxy_pools table @ migrate.go:191, and on internal/admin/handlers.go the
  proxyPools field (:23) + tunnels field (:24) constructed in New (:55-56) with the
  SetProxyProber (:91) + SetTunnelRunner (:99-100) post-construction injectors.
  THIS plan REUSES that exact injection philosophy (default-constructed service
  field + a SetX setter; NO New() signature change). live tree @ <base>;
  WAVE-7-MAP w7-plat-3 row ~line 185; serial chain §219-224; reconciliation §245;
  freeze rules §267)
runs: platform track. Disjoint domain/store/admin files from w7-plat-1 (proxy-pools,
  SHIPPED) and w7-plat-2 (tunnels, SHIPPED) — run ∥ those (its files are disjoint).
  TAKES the internal/server/routes_admin.go SERIAL SLOT in chain order
  (… → w7-plat-1 → w7-plat-2 → **w7-plat-3** → w7-misc; MAP §219-224). w7-plat-2
  RELEASES the slot to w7-plat-3 on its close. NO secondary micro-serial: w7-plat-3
  does NOT touch selection.go / factory.go / runner.go (MITM is a standalone
  listener subsystem, NOT an inference-path concern).
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w7-plat-3:
ref-source: 9router frozen @ 827e5c3 — MITM (man-in-the-middle) proxy surface.
  The BINDING contract for W7 is the W6 e2e mock (decision 1: real Go wins, mock
  corrected to mirror it). Mock sources:
    ui/e2e/mocks/handlers/mitm.ts + ui/e2e/mocks/seed/mitm.ts.
base: <base> = git rev-parse HEAD recorded at P0. Substitute the actual SHA
  everywhere §5 says <base>.
go-serial-slot: this plan holds the SINGLE in-flight edit to
  internal/server/routes_admin.go while live (W3/W4/W5/W6 lesson; MAP decision 3).
  Slot must be FREE at P6 before T-routes (no concurrent W7 plan with an unmerged
  routes_admin.go edit — w7-plat-2 must have closed + released); RELEASE to w7-misc
  on close.
new-route: NO UI route files. The /mitm page ALREADY SHIPPED in w6-m (PARTIAL)
  against the mock; this plan builds the REAL Go so the page flips PARTIAL→HAVE
  and corrects the mock body to mirror the Go DTO (keeping the raw-PEM ca-cert
  contract intact).
```

---

## 1. Scope — PAR rows + the three surfaces

### Rows this plan closes

| Row / item | Claim | Target state after w7-plat-3 |
|---|---|---|
| PAR-PLAT-024 | root CA generation (self-signed CA, persisted; key at rest) + CA-cert serving as raw PEM | HAVE (CA-gen + PEM encode are PURE crypto, FULLY unit-tested; `GET /api/mitm/ca-cert` serves raw PEM `application/x-pem-file`, NOT `{data}`) |
| PAR-PLAT-025 | HTTPS MITM reverse proxy: SNI per-host leaf-cert minting signed by the root CA, cert cache, ALPN, intercept+forward | HAVE (leaf-cert minting + CA signing + chain verify are PURE crypto, FULLY unit-tested; the live reverse-proxy LISTENER is integration-only — §1.9, like w7-plat-2's spawn split). System-trust-store auto-install + hosts-file patching are OS-privileged → DEFERRED/escalated (§1.9 / §8 ESC-OS-PRIV) |
| PAR-PLAT-028 | per-tool MITM config + enable/toggle (global + per-tool) | HAVE (`GET /api/mitm/status`→`{enabled,tools[]}`, `POST /api/mitm/toggle`, `POST /api/mitm/tools/{id}`; FULLY unit-tested via `newTestEnv`) |
| open-questions w6-m **ESC-1a** (mitm backend absent) | real `/api/mitm/*` status/toggle/ca-cert/tools + MITM proxy config + CA-cert serving + per-tool enable | RESOLVED (cite this plan) |
| PAR-UI-013 | mitm page (PARTIAL) | PARTIAL → HAVE |

Matrix flips at closeout (§4 T-close): in `.planning/parity/matrix/`, PAR-PLAT-024/025/028
→ HAVE (real Go; the live MITM listener + OS-privileged trust-store/hosts-file parts
integration-only + escalation-recorded — §1.9); PAR-UI-013 PARTIAL → HAVE. Mark
`open-questions.md` w6-m ESC-1a RESOLVED with a cite to this plan; append any new open
items (§8 — OS-privileged trust-store/hosts-file escalation; listener integration-only
note; ESC-CA-STORE/ESC-CACHE decisions).

### 1.1 Preconditions already satisfied by merged waves (evidence — cite file:line)

- **W6-m UI is SHIPPED (PARTIAL) and FROZEN (consume-only, MAP decision 8 / §267).**
  The `/mitm` page renders against the registered mock (`ui/src/routes/mitm.tsx`).
  The UI type is `MitmTool` (`ui/src/lib/types.ts:151-157`):
  `{id:string, name:string, enabled:boolean, dns_override:string, status:"active"|"inactive"}`
  — a 5-field shape. The status panel is a local `MitmStatus` interface
  (`mitm.tsx:16-19`): `{enabled:boolean, tools:MitmTool[]}`.
  The binding acceptance contract is the existing spec (must stay green at closeout):
  `ui/e2e/mitm.spec.ts` — it asserts: the page body contains "MITM"; a visible
  `[data-testid='mitm-enable-toggle']`; exactly **2** `[data-testid='mitm-tool-row']`;
  the strings "Request Inspector" and "Response Modifier"; that toggling a tool fires
  a **POST** on `/\/api\/mitm\/tools\/[^/]+$/`; and that a
  `[data-testid='mitm-ca-cert-download']` control is present. It does NOT assert the
  ca-cert response body, NOR the global-toggle request, NOR any field beyond the
  5-field `MitmTool` + `{enabled,tools}` shape.
- **The mock contract (CANONICAL — the page calls these in-tree paths)**
  — `ui/e2e/mocks/handlers/mitm.ts`:
  - `GET /api/mitm/status` → `json(route, {enabled: store.mitmEnabled, tools: store.mitmTools})`
    (the page reads `status.enabled` + `status.tools` via `apiFetch`, `mitm.tsx:35-38`).
  - `POST /api/mitm/toggle` → flips `store.mitmEnabled`, returns `{enabled}`
    (the page calls it via `apiFetch`, ignores the body, `mitm.tsx:53-61`).
  - `GET /api/mitm/ca-cert` → **`route.fulfill`** with status 200,
    `Content-Type: application/x-pem-file`, body = RAW PEM (NOT `{data}`). **The page
    downloads it via a PLAIN `fetch(${origin}/api/mitm/ca-cert)` + anchor, NOT
    `apiFetch`** (`mitm.tsx:80-101`, comment `mitm.tsx:82-83`). THIS is the binding
    raw-PEM contract the Go handler MUST mirror.
  - `POST /api/mitm/tools/{id}` (regex `\/api\/mitm\/tools\/[^/]+$`) → flips the
    stored tool `enabled`, sets `status = enabled ? "active":"inactive"`, returns the
    tool object; 404 `{error}` if the tool id is unknown.
- **The seed shape (CANONICAL)** — `ui/e2e/mocks/seed/mitm.ts` (`seedMitmStatus()`):
  `{enabled:false, ca_cert:"-----BEGIN CERTIFICATE-----\n...", tools:[...]}` with two
  `MitmTool` rows:
  `{id:"mitm-1", name:"Request Inspector", enabled:true,  dns_override:"localhost", status:"active"}`
  and
  `{id:"mitm-2", name:"Response Modifier", enabled:false, dns_override:"",          status:"inactive"}`.
  **This is the canonical Go list shape** (5-field tool + `{enabled,tools}` status).
  The spec asserts the "Request Inspector"/"Response Modifier" substrings + 2 rows →
  the corrected Go status/seed MUST surface those two named tools. Note: the seed
  carries a top-level `ca_cert` field, but the page's `MitmStatus`
  (`mitm.tsx:16-19`) does NOT read it (CA cert is fetched via the raw-PEM endpoint) —
  so the Go `/api/mitm/status` DTO need NOT include `ca_cert` (the page ignores it).
- **The `internal/platform` package + the SHIPPED injection precedent (the central
  reuse).** `internal/platform/doc.go` names the platform features; the package now
  holds the SHIPPED w7-plat-1 (proxypools) + w7-plat-2 (tunnel/) files. The KEY
  precedent this plan mirrors EXACTLY:
  - `internal/admin/handlers.go:23` the `Handlers` struct holds
    `proxyPools *platform.ProxyPoolService`; `:24` holds `tunnels *tunnel.Service`.
  - `New(st, sessions, flows)` (`handlers.go:32`) constructs both via
    `platform.NewProxyPoolService(st)` (`:55`) and `tunnel.NewService(st)` (`:56`)
    with NO `New(...)` signature change.
  - `SetProxyProber(p platform.Prober)` (`handlers.go:91`) and
    `SetTunnelRunner(typ string, r tunnel.Runner)` (`handlers.go:99-100`) are the
    post-construction injectors that forward into the service. This is the IDENTICAL
    "external effect, testable without performing it" shape w6-j's `SetShutdownFunc`
    (`handlers.go:85`) established. **w7-plat-3's MITM service REUSES this exact
    philosophy (§1.4): a default-constructed `mitm *mitm.Service` field + a
    `SetMitmCA`/`SetMitmProxy` setter for test injection; NO `New(...)` change.**
- **Secret-at-rest precedents (TWO, the CA-key-storage decision draws on both):**
  - **Reversible `*_enc` columns** — `internal/store/migrate.go`: the `tables` slice
    declares `*_enc` columns written/read via `s.cipher`: `connections.secret_enc`
    (`migrate.go:51`), `oauth_sessions.verifier_enc` (`migrate.go:62`),
    `alert_channels.config_enc` (`migrate.go:186`), `proxy_pools.password_enc`
    (`migrate.go:198`). The encrypt/decrypt round is `s.cipher.Encrypt/Decrypt`
    (`oauthsessions.go:21,58`; `proxypools.go:30,162`).
  - **Key-material as a 0600 file under the data dir** —
    `internal/store/secret.go:15-36` `LoadOrCreateSecret(dataDir)`: `os.MkdirAll(dataDir,
    0o700)` (`:16`) then `os.WriteFile(path, key, 0o600)` (`:36`); the master secret
    key itself lives as `dataDir/secret.key` (the file that backs `s.cipher`). The
    store exposes the dir via `store.DataDir()` (`apikeys.go:92`). `store_test.go:64`
    asserts the `0o600` perm. **THIS is the precedent for the CA key-at-rest
    decision (§1.3 / §8 ESC-CA-STORE).**
- **Additive migrations only** — `migrate.go` new tables via the `tables []struct`
  slice with `CREATE TABLE IF NOT EXISTS` (`migrate.go:15-200`); new columns via
  `ensureColumn(db, table, column, decl)` (`migrate.go:267`, helper at `:343`). The
  SHIPPED `proxy_pools` table (`migrate.go:191`) is the immediate precedent for adding
  a `mitm_tools` (+ optional `mitm_ca`) table the same way.
- **Envelope + handler patterns** (`internal/admin/respond.go`):
  `writeData(ctx, status, data)` (`respond.go:19`) / `writeError(ctx, status,
  message)` (`respond.go:23`) → `{data,error:{message}}` snake_case. `pathID(ctx
  .UserValue("id"))` extracts a path param (`handlers.go:104`) — for mitm tools the
  param is `{id}`, so use `pathID(ctx.UserValue("id"))` directly (mirror the existing
  cast). **The CA-cert handler is the EXCEPTION — it writes raw PEM via the fasthttp
  ctx directly (`ctx.SetContentType("application/x-pem-file")` + `ctx.SetBody(pem)` /
  `ctx.Write(pem)`), NOT `writeData` (§1.5).** CRUD/handler template = the SHIPPED
  `internal/admin/tunnels.go` / `internal/admin/proxypools.go` (DTO, request structs,
  `writeData/writeError`, `h.recordAudit`, nil-safe service field, `{type}`/`{id}`
  path-param cast).
- **Admin test harness** (`internal/admin/admin_test.go` `newTestEnv`): real
  `store.Open(tempDB, secret)` + `auth.NewSessions` + `SeedAdmin("admin","123456")` +
  `New(...)`. NO mocks. `call(...)` drives a handler + decodes the `{data,error}`
  envelope. This is the authoritative proof surface for the status/toggle/tool admin
  API. (The raw-PEM ca-cert handler is asserted by reading `ctx` body + content-type,
  NOT the envelope decoder — §1.5.)
- **The audit seam** — `internal/admin/audit.go:64`
  `func (h *Handlers) recordAudit(ctx, action, target, details string)` (resolves the
  actor via `ctx.UserValue(userKey)`, `audit.go:66`). REUSE `h.recordAudit` on every
  mitm mutation (global toggle + per-tool toggle). NO audit retrofit into other files;
  NO edit to audit.go.
- **Handlers injection** — the `Handlers` struct composes `h.store` directly; new
  domains use `h.store` with NO new global state and NO `New(...)` signature change
  (MAP decision 9). The new `mitm` service field is constructed in `New` over the
  existing `st` (mirror `tunnels: tunnel.NewService(st)`, `handlers.go:56`).
- **No pre-existing TLS/cert code in-tree** — a repo-wide
  `grep -rn "x509|tls.Certificate|GenerateKey|crypto/tls|crypto/x509"
  internal/ cmd/` returns ZERO matches at authoring (verified). The MITM CA + leaf
  minting is GREENFIELD pure stdlib (`crypto/x509`, `crypto/tls`, `crypto/rsa` or
  `crypto/ecdsa`, `crypto/rand`, `encoding/pem`) — nothing to reuse, nothing to
  collide with. (Re-confirm at P1.)

### 1.2 The mock contract this flip must mirror (binding — decision 1)

**Decision 1 (MAP §36, §245):** real Go wins; the W6 mock body + seed are corrected
IN THIS PLAN to mirror the real Go DTO. The page is FROZEN (decision 8); prefer
matching the mock's existing field names in the Go DTO; only ESCALATE if impossible.

**MITM** (`ui/e2e/mocks/handlers/mitm.ts` + `seed/mitm.ts`):
- Routes the page consumes (canonical, in-tree):
  - `GET /api/mitm/status` → `{enabled:bool, tools:[mitmToolDTO]}`. **VERIFY** whether
    the mock's `json()` helper wraps in `{data}` (the page reads via `apiFetch`, which
    unwraps `{data}`, `mitm.tsx:35`). If `json()` wraps, the Go writes the SAME
    `{data:{enabled,tools}}` envelope (`writeData`). The page reads `status.enabled`
    + `status.tools` after the `apiFetch` unwrap, so the Go MUST place `{enabled,
    tools}` under `{data}`. ALWAYS 2 tools (Request Inspector, Response Modifier — to
    keep the 2-row spec green).
  - `POST /api/mitm/toggle` → flips global enable; returns `{data:{enabled:bool}}`.
    The page ignores the body (`mitm.tsx:56`), only that the request fires (the spec
    does NOT even assert this request — it asserts only the per-tool POST).
  - `GET /api/mitm/ca-cert` → **RAW PEM**, `Content-Type: application/x-pem-file`,
    **NOT** `{data}`. The page fetches via plain `fetch` (`mitm.tsx:84`). **This is
    the binding raw-PEM exception (§1.5) — the corrected mock keeps `route.fulfill`
    with `application/x-pem-file`; the Go handler writes raw PEM bytes with the same
    content-type.**
  - `POST /api/mitm/tools/{id}` (regex `\/api\/mitm\/tools\/[^/]+$`) → flips the
    tool `enabled` + `status`, returns `{data:mitmToolDTO}`; 404 on unknown id. The
    spec asserts the request method/path fires (`mitm.spec.ts:35-42`), not the body.
- DTO shape = the UI `MitmTool` type (`types.ts:151-157`):
  `{id:string, name:string, enabled:bool, dns_override:string, status:"active"|
  "inactive"}` — **this is the canonical 5-field Go tool DTO.** The status DTO is
  `{enabled:bool, tools:[mitmToolDTO]}`.
- **Mock divergences to reconcile (mock mirrors Go — decision 1):**
  - **Envelope (`status`/`toggle`/`tools/{id}`):** VERIFY whether the mock `json()`
    helper already wraps in `{data}` (it likely does, mirroring the other handlers —
    `ui/e2e/mocks/handlers/utils.ts`). If `json()` wraps, no change is needed; if not,
    confirm the page's `apiFetch` unwrap still matches. **Reconciliation:** confirm
    the mock body matches the Go's `{data}` shape at T-mocks; correct ONLY on a real
    divergence (§8 ESC-MOCK).
  - **ca-cert (raw PEM — DO NOT enveloping):** the mock uses `route.fulfill` with
    `application/x-pem-file` + a raw-PEM body (NOT `json()`). The Go handler MUST do
    the SAME (raw bytes, that content-type, NO `{data}`). **NO mock change here** — it
    already mirrors the binding contract; just verify the content-type string and that
    the Go body is a valid PEM block (§1.5 / §5 grep proof).
  - **Seed `ca_cert` top-level field:** the seed has a top-level `ca_cert` the page
    does NOT read (`MitmStatus`, `mitm.tsx:16-19`, has only `{enabled,tools}`). The
    Go `/api/mitm/status` DTO need NOT include `ca_cert` (the page ignores it). LEAVE
    the seed's `ca_cert` as-is OR drop it — default: LEAVE the seed unchanged (it is
    harmless; the page ignores it; removing it is a no-op risk). VERIFY no spec reads
    `ca_cert` (none does).
  - **Seed `dns_override`:** the page renders `tool.dns_override || "no DNS override"`
    (`mitm.tsx:158`). The Go tool DTO MUST carry `dns_override` (default `""`). The
    seed keeps `dns_override:"localhost"`/`""` — KEEP. NOTE: actual DNS-override
    ENFORCEMENT (mapping a host to the MITM listener) is part of the deferred
    OS/hosts-file scope (§1.9 / §8 ESC-OS-PRIV); the `dns_override` field is stored +
    surfaced (config), not enforced at the OS level in this plan.

### 1.3 MITM Go contract (NEW, TDD) — store + CA storage + key-at-rest

The MITM domain has TWO persisted concerns: (a) the per-tool config rows (+ a global
enable flag), and (b) the root CA cert+key. **DECIDE typed-vs-JSON columns + the
CA-key-at-rest mechanism (§8 ESC-SCHEMA / ESC-CA-STORE).**

**Per-tool config table `mitm_tools`** (additive, `migrate.go` `tables` slice — mirror
the SHIPPED `proxy_pools` table @ `migrate.go:191`). RECOMMENDED default = **typed
columns** (the DTO is a fixed small 5-field shape; typed columns enable the
`WHERE id=?` per-tool toggle cleanly; matches the proxy-pools/tunnels precedent):

```sql
CREATE TABLE IF NOT EXISTS mitm_tools (
  id TEXT PRIMARY KEY,                      -- 'mitm-1' | 'mitm-2' | ...
  name TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 0,
  dns_override TEXT NOT NULL DEFAULT '',    -- config only (OS-level enforcement deferred — §1.9)
  status TEXT NOT NULL DEFAULT 'inactive',  -- 'active' | 'inactive'
  updated_at INTEGER NOT NULL DEFAULT 0
)
```

**Global enable flag.** The `/api/mitm/status` `enabled` is a single global boolean.
RECOMMENDED default = store it in the existing `settings` key-value surface (mirror
how `tunnelDashboardAccess`/`tunnelUrl` live in `settings`) under a key
`mitmEnabled` — NO new table for one boolean. **DECIDE (§8 ESC-GLOBAL-FLAG):** if a
`settings` accessor is not cleanly available to the store layer, fall back to a
single-row `mitm_state` table or a sentinel row. Default: `settings["mitmEnabled"]`
if the store exposes settings get/set; else a tiny `mitm_state(enabled INTEGER)`
single-row table. Decide at T-mitmstore against the live `settings` API.

**CA storage + key-at-rest (THE binding secret decision — §8 ESC-CA-STORE).** The
root CA is a self-signed cert (PUBLIC — served as raw PEM) + a PRIVATE key (SECRET —
NEVER served, NEVER echoed). Two viable storage shapes, BOTH grounded in shipped
precedent:
- **(A — RECOMMENDED DEFAULT) Key + cert as files under the data dir, restricted
  perms.** Mirror `internal/store/secret.go:15-36` EXACTLY: the master `secret.key`
  already lives as `dataDir/secret.key` written with `os.WriteFile(path, key, 0o600)`
  after `os.MkdirAll(dataDir, 0o700)`. The CA follows the identical pattern:
  `dataDir/mitm-ca.key` (PEM-encoded private key, `0o600`) + `dataDir/mitm-ca.crt`
  (PEM-encoded public cert, `0o644`/`0o600`). `store.DataDir()` (`apikeys.go:92`)
  gives the dir. **Generate-on-first-use** (like `LoadOrCreateSecret`): if the key
  file is absent, generate the CA and persist both files; else load. The key bytes
  NEVER leave the process; only the public `mitm-ca.crt` is read + served.
  RATIONALE: a CA private key is large PEM key material that fits the "key file under
  the data dir, 0600" precedent (secret.go) better than an `*_enc` DB column; it also
  keeps the CA usable for TLS without a per-read decrypt round.
- **(B — alternative) Cert+key in a `mitm_ca` DB table with the key `*_enc`.** A
  single-row `mitm_ca(cert_pem TEXT, key_enc TEXT, created_at INTEGER)` table; the
  private key encrypted at rest via `s.cipher.Encrypt` (the `verifier_enc`/
  `password_enc` precedent — `oauthsessions.go:21`, `proxypools.go:30`); the public
  `cert_pem` stored cleartext. RATIONALE: keeps everything in the single SQLite WAL
  store (one backup surface; consistent with the additive-migration discipline).
- **DECISION (binding default): (A) key+cert files under the data dir** (`0o700` dir /
  `0o600` key / `0o644` cert), generate-on-first-use, mirroring `secret.go`. It is the
  closest shipped precedent for raw key MATERIAL (vs a short secret string in a DB
  column), avoids a per-TLS-handshake decrypt, and the public cert file is exactly
  what the raw-PEM endpoint serves. **Flag for orchestrator confirmation; if the
  operator prefers single-store backup semantics, switch to (B) with `key_enc`.**
  EITHER WAY: the private key is NEVER served, NEVER logged, NEVER placed in any DTO
  (§5 no-key-echo grep proof).

`internal/store/mitm.go` (NEW): `MitmTool` struct
`{ID, Name, Enabled, DNSOverride, Status, UpdatedAt}` + methods:
- `ListMitmTools() ([]MitmTool, error)` — deterministic order (by id);
  ALWAYS surfaces the seeded tools (see EnsureMitmTools / overlay below).
- `GetMitmTool(id string) (MitmTool, error)` — `ErrNotFound` on `sql.ErrNoRows`.
- `UpsertMitmTool(t MitmTool) error` — `INSERT … ON CONFLICT(id) DO UPDATE` (mirror
  the proxy-pools/tunnels upsert patterns: `boolToInt`, `time.Now().Unix()`).
- `SetMitmToolEnabled(id string, enabled bool) (MitmTool, error)` — flips `enabled`
  + derives `status = enabled ? "active":"inactive"`; persists; returns the updated
  row (404 via `ErrNotFound` on unknown id).
- `GetMitmEnabled() (bool, error)` / `SetMitmEnabled(bool) error` — the global flag
  (settings-backed per ESC-GLOBAL-FLAG).
- `EnsureMitmTools() error` (OPTIONAL — §8 ESC-SEED-ROWS) — seed the two known tools
  (Request Inspector, Response Modifier) on first migrate so `ListMitmTools` always
  returns ≥2 (matches the mock's 2-row spec). Alternatively the status handler
  overlays the 2 known tools from a constant + any stored row — DECIDE at
  T-mitmstore (default: `EnsureMitmTools` seeds the 2 rows once on migrate, since
  these are named domain tools — Request Inspector / Response Modifier — not synthetic
  placeholders; the seed keeps `ListMitmTools` honest and the 2-row spec green).

**Secret note:** the `mitm_tools` table holds NO secret (dns_override is config, not a
credential). The ONLY secret in this domain is the CA private key, handled per
ESC-CA-STORE above — NEVER in a DTO, NEVER in `mitm_tools`.

`internal/admin/mitm.go` (NEW):

| Handler | Route | Shape (snake_case) | Notes |
|---|---|---|---|
| `MitmStatus` | `GET /api/mitm/status` | `{data:{enabled:bool, tools:[mitmToolDTO]}}` | `mitmToolDTO{id,name,enabled,dns_override,status}` (5 fields; NEVER any key material). ALWAYS ≥2 tools. Reads `store.GetMitmEnabled()` + `store.ListMitmTools()`. |
| `MitmToggle` | `POST /api/mitm/toggle` | `{data:{enabled:bool}}` | flips the global flag; `h.recordAudit("mitm.toggle", ...)`; starts/stops the MITM proxy listener best-effort via the service (§1.6) — the LISTENER start is integration-only but the FLAG flip + persist + audit is unit-tested |
| `MitmCACert` | `GET /api/mitm/ca-cert` | **RAW PEM**, `Content-Type: application/x-pem-file`, NOT `{data}` | the raw-PEM EXCEPTION (§1.5). Loads/lazily-generates the CA (service), serves the PUBLIC cert PEM ONLY. NEVER the key. |
| `MitmToolToggle` | `POST /api/mitm/tools/{id}` | `{data:mitmToolDTO}` | `pathID(ctx.UserValue("id"))`; 404 on unknown id; flips the tool; `h.recordAudit("mitm.tool.toggle", id, ...)`. |

`{id}` is read via `pathID(ctx.UserValue("id"))` (mirror the SHIPPED handlers); an
unknown id → 404 `{error:{message:"tool not found"}}` (mirror the mock's 404).

### 1.4 THE CENTRAL DESIGN PROBLEM — pure-crypto core (unit-tested) vs live listener (integration-only) (binding, REUSE the SHIPPED service-field + SetX-injector philosophy)

The MITM proxy is a LIVE TLS LISTENER that binds a port and performs real TLS
interception — it CANNOT be exercised in unit tests (AGENTS.md "No mocks; use
interfaces and fakes; test real behavior"; w7-plat-2 §1.9 "the live reverse-proxy
listener is integration-only"). BUT the crypto CORE — CA generation, leaf-cert
minting, CA signing, chain verification, PEM encoding — is PURE `crypto/x509` /
`crypto/tls` and is FULLY unit-testable deterministically with NO port binding, NO
network, NO real TLS handshake. The admin status/toggle/tool API is FULLY unit-tested
via `newTestEnv`. The split:

**The CA core (NEW — `internal/platform/mitm/ca.go`) — PURE crypto, FULLY unit-tested:**
```go
package mitm

// CA holds the self-signed root CA used to mint per-host leaf certificates for
// MITM interception. The private key is SECRET — never serialized into any
// response/DTO/log; only CertPEM() (the public cert) is served.
type CA struct {
    cert    *x509.Certificate
    key     crypto/* PrivateKey   // RSA or ECDSA (decide §8 ESC-KEYTYPE; default ECDSA P-256)
    certPEM []byte                // cached public PEM
}

// GenerateCA creates a fresh self-signed root CA (IsCA=true, KeyUsageCertSign).
// PURE — no I/O. Deterministic-shape (unit-tested: parses back, IsCA, KeyUsage).
func GenerateCA(opts CAOpts) (*CA, error)

// LoadOrCreateCA loads the CA from the data-dir files (mitm-ca.{key,crt}), or
// generates+persists one on first use (mirror store.LoadOrCreateSecret).
// The FILE I/O wrapper (the only I/O) is integration-thin; GenerateCA is the
// pure unit-tested core. (ESC-CA-STORE)
func LoadOrCreateCA(dataDir string) (*CA, error)

// CertPEM returns the PUBLIC root CA cert as PEM (application/x-pem-file body).
// PURE. Unit-tested: output is a valid PEM CERTIFICATE block, parses via
// x509.ParseCertificate, IsCA==true. NEVER returns key material.
func (c *CA) CertPEM() []byte

// MintLeaf mints a leaf cert for the given SNI host, signed by the CA.
// PURE crypto — no I/O. Unit-tested: x509.Verify(leaf, pool{CA}) succeeds;
// leaf.DNSNames contains host; leaf NOT a CA.
func (c *CA) MintLeaf(host string) (tls.Certificate, error)
```

**The leaf-cert cache (NEW — `internal/platform/mitm/ca.go` or proxy.go) — PURE +
unit-tested:** a `map[string]tls.Certificate` guarded by a `sync.RWMutex`, keyed by
SNI host; `getLeaf(host)` mints-and-caches on miss, returns the cached cert on hit.
Unit-tested deterministically: first call mints (cache miss), second call returns the
SAME cert object (cache hit) — NO listener, NO handshake.

**The MITM proxy LISTENER (NEW — `internal/platform/mitm/proxy.go`) — integration-only
(NOT unit-tested — §1.9):** a `Proxy` that holds the `*CA` + the leaf cache + a
`tls.Config` whose `GetCertificate(hello *tls.ClientHelloInfo)` mints/returns the
per-SNI leaf (this is the ALPN/SNI interception seam); `Start(addr)` binds the
listener, `Stop()` closes it, `Running()` reports state. **The `GetCertificate`
closure logic (given a `ClientHelloInfo.ServerName`, return the right leaf) is PURE
and CAN be unit-tested by calling it with a synthetic `*tls.ClientHelloInfo{ServerName:
"example.com"}` and asserting the returned `*tls.Certificate` verifies against the CA —
this does NOT bind a port.** The `Start`/`Stop`/`net.Listen`/`tls.NewListener` body +
the actual intercept-and-forward proxying are integration-only (a real port bind + a
real TLS handshake), excluded from `go test ./...` determinism (§5 no-listen-in-test
grep proof).

**The MITM SERVICE (NEW — `internal/platform/mitm/service.go`):** holds the `*store
.Store` + the lazily-loaded `*CA` + the `*Proxy` (the listener). It is the seam the
admin handlers call. RESTART BACKOFF lives here (§1.6).
- `Status() (enabled bool, tools []store.MitmTool, err error)` — overlays store state.
- `Toggle() (enabled bool, err error)` — flips the global flag; on enable, best-effort
  `proxy.Start` (integration-only) with restart backoff; on disable, `proxy.Stop`.
- `ToggleTool(id) (store.MitmTool, error)` — flips a tool, persists.
- `CACertPEM() ([]byte, error)` — `LoadOrCreateCA(store.DataDir()).CertPEM()` (the
  public cert; lazily generates on first call).
- `NewService(st)` constructs WITHOUT binding any listener (mirror
  `tunnel.NewService(st)` constructing real-default runners but not spawning —
  `handlers.go:56`). The CA + proxy are created lazily / on enable.

**Injection (binding — NO `New(...)` signature change; mirror SetTunnelRunner):**
- On `Handlers`: add a `mitm *mitm.Service` FIELD constructed in `New` via
  `mitm.NewService(st)` (mirror `tunnels: tunnel.NewService(st)`, `handlers.go:56`) —
  NO `New(...)` signature change.
- A test-injection setter on the service for the listener/CA so the admin tests run
  WITHOUT binding a port: `SetProxy(p MitmProxy)` (a `MitmProxy` interface
  `{Start(addr) error; Stop() error; Running() bool}` whose REAL impl is the live
  listener and whose TEST impl is a deterministic fake — mirror the `tunnel.Runner`
  seam). Forward via a `Handlers` setter `SetMitmProxy(p mitm.MitmProxy)` (mirror
  `SetTunnelRunner`, `handlers.go:99-100`). The admin toggle test injects the fake so
  `Toggle` records the flag flip + audit WITHOUT a real bind. **The CA itself is real
  in tests** (it is pure crypto — generating a real test CA is cheap + deterministic;
  no injection needed for the CA, only for the listener). For the ca-cert handler
  test, the service may use a temp data dir (`newTestEnv` already uses a tempDB; the
  CA files land beside it via `store.DataDir()`).

**What is UNIT-TESTED (deterministic, hermetic — `go test ./...` with NO port bind /
NO network / NO real TLS handshake):**
- `GenerateCA` → parses back via `x509.ParseCertificate`; `IsCA==true`;
  `KeyUsage` has `CertSign`.
- `CertPEM()` → a valid `CERTIFICATE` PEM block; `pem.Decode` + `x509.ParseCertificate`
  round-trips; the block contains NO `PRIVATE KEY` (no-key-echo).
- `MintLeaf(host)` → `x509.Verify(leaf, roots=pool{CA})` succeeds; `leaf.DNSNames`
  contains `host`; leaf is NOT a CA. **This is the highest-value unit test (CA signing
  + chain verification, fully deterministic).**
- the leaf CACHE: miss mints, hit returns the same cert (no re-mint).
- the `GetCertificate` closure: given `ClientHelloInfo{ServerName:"example.com"}`
  returns a leaf that verifies against the CA (no port bind).
- the admin status/toggle/tool API via `newTestEnv` (+ fake `MitmProxy`): status→
  enabled+2 tools; toggle→flips + audit; tool toggle→flips + status derived; unknown
  tool id→404; ca-cert→raw PEM + content-type (§1.5); **no response/DTO leaks key
  material**.

**What is INTEGRATION-ONLY (NOT unit-tested — thin, isolated, escalation-recorded —
§1.9):** `proxy.Start`/`Stop` (real `net.Listen` + `tls.NewListener` + the
intercept-and-forward loop); the OS system-trust-store auto-install + hosts-file
patching (OS-privileged — DEFERRED/escalated). These live in `proxy.go` (+ a deferred
trust-store helper) behind the `MitmProxy` interface; their bodies bind ports / shell
out / require OS privilege and are excluded from `go test ./...` determinism (§5 grep
proof "no-listen-in-test").

### 1.5 CA-cert raw-PEM response (`GET /api/mitm/ca-cert`) — the binding raw-PEM exception

The CA-cert handler is the ONE handler that does NOT use `writeData`. It mirrors the
w6-m page's plain-`fetch` download (`mitm.tsx:80-101`) + the mock's `route.fulfill`
with `application/x-pem-file` (`mitm.ts` ca-cert branch). The handler:
```go
func (h *Handlers) MitmCACert(ctx *fasthttp.RequestCtx) {
    pem, err := h.mitm.CACertPEM()        // public cert PEM ONLY; lazily generates the CA
    if err != nil { writeError(ctx, 500, "failed to load CA certificate"); return }
    ctx.SetContentType("application/x-pem-file")  // NOT application/json
    ctx.SetStatusCode(200)
    ctx.SetBody(pem)                       // RAW PEM bytes; NOT a {data} envelope
}
```
**Binding asserts (admin test + §5 grep proofs):** the response `Content-Type` is
`application/x-pem-file` (NOT `application/json`); the body begins
`-----BEGIN CERTIFICATE-----` and is a valid PEM CERTIFICATE block that
`x509.ParseCertificate` accepts; the body contains NO `PRIVATE KEY` and NO key
material (no-key-echo). The mock already mirrors this exact contract — verify, do not
change (§1.2). The error path uses `writeError` (a `{data,error}` envelope) only on
failure; the SUCCESS path is raw PEM.

### 1.6 Global toggle + per-tool toggle + restart backoff

- **Global toggle (`POST /api/mitm/toggle`).** Flips `store.SetMitmEnabled(!cur)`,
  records audit, and best-effort starts/stops the proxy listener via the service. The
  FLAG flip + persist + audit is unit-tested (via the fake `MitmProxy`); the real
  `proxy.Start` is integration-only.
- **Per-tool toggle (`POST /api/mitm/tools/{id}`).** `SetMitmToolEnabled(id,
  !cur)` → derives `status` → persists → returns `{data:mitmToolDTO}`; 404 on unknown
  id; `h.recordAudit`.
- **Restart backoff (binding — §8 ESC-BACKOFF).** The proxy listener can fail to bind
  (port in use) or crash. The service wraps `proxy.Start` in a bounded
  exponential-backoff restart loop (e.g. 1s, 2s, 4s … capped, max N attempts) so a
  transient bind failure self-heals without hammering. **DECIDE (§8 ESC-BACKOFF):**
  there is NO existing backoff helper in-tree (verified — `grep -rn backoff
  internal/` is EMPTY). RECOMMENDED default = a small in-package backoff
  (`time.After` with a doubling delay capped at e.g. 30s, max 5 attempts, abort on
  `Stop`/disable). The backoff TIMING is integration-only (it sleeps on a real
  listener); the backoff POLICY (the delay sequence / cap / max-attempts as a pure
  function `nextBackoff(attempt) time.Duration`) CAN be a cheap pure unit test
  (deterministic — assert the doubling + cap). Default: factor `nextBackoff` pure +
  unit-test it; the loop that calls `proxy.Start` between sleeps is integration-only.

### 1.7 routes_admin.go registration (serial-slot additive, §3)

Add (additive appends; static/deeper-before-param precedence honored by the file — the
specific static routes `/api/mitm/status`, `/api/mitm/toggle`, `/api/mitm/ca-cert`
have no `{param}` collision; the `/api/mitm/tools/{id}` param route is distinct). The
4 lines append BELOW the tunnels block (`routes_admin.go:140-143`):
```go
// MITM (status/toggle/ca-cert static; tools/{id} param).
r.GET("/api/mitm/status", h.RequireSession(h.MitmStatus))
r.POST("/api/mitm/toggle", h.RequireSession(h.MitmToggle))
r.GET("/api/mitm/ca-cert", h.RequireSession(h.MitmCACert))     // raw PEM, NOT {data}
r.POST("/api/mitm/tools/{id}", h.RequireSession(h.MitmToolToggle))
```
Route-precedence note: the three `/api/mitm/<word>` static routes vs
`/api/mitm/tools/{id}` (param under a distinct `tools/` segment) do not collide. A
genuine `fasthttp/router` collision is §8 ESC-ROUTE, not a silent path change. The 4
lines append BELOW the tunnels block (`routes_admin.go:143`). **NOTE the ca-cert
route is still `RequireSession`-guarded** (the page calls it via authenticated plain
`fetch` from the dashboard origin; the raw-PEM body is the only difference from the
other handlers, not the auth). Diff bound §5: the route block is ONE commit, additive
only.

### NOT in scope (explicit)

- **No UI page/component/route/store edits** — `/mitm` and all w6-m components are
  FROZEN consume-only (decision 8). The ONLY UI-tree touches are the mitm mock-body +
  seed verification (§1.2 / §3), and ONLY if a real Go-vs-mock divergence exists
  (default: verify, the raw-PEM contract already matches — likely NO change).
- **No edits to other platform domains** — proxy-pools (w7-plat-1, SHIPPED) +
  tunnels (w7-plat-2, SHIPPED) are disjoint; CONSUME their injection precedent, do
  NOT edit their files (`internal/platform/{proxypools,outboundproxy}.go`,
  `internal/platform/tunnel/*.go`, `internal/store/{proxypools,tunnels}.go`,
  `internal/admin/{proxypools,tunnels}.go`).
- **No OS-privileged trust-store auto-install / hosts-file patching** — DEFERRED +
  escalated (§1.9 / §8 ESC-OS-PRIV). The parity bar is CA-gen + CA-cert serving +
  proxy core (mint/sign/verify/cache) + the admin API. The `dns_override` field is
  STORED + surfaced (config), not OS-enforced.
- **No edits to pre-existing admin handlers' bodies** — apikeys, virtualkeys,
  providers*, connections, combos, auth, version, usage, **proxypools**, **tunnels**
  are FORBIDDEN. The ONLY `handlers.go` touch is the ADDITIVE `mitm` service field +
  the `SetMitmProxy` setter (mirroring the SHIPPED `tunnels` field + `SetTunnelRunner`
  — NOT a frozen handler body; NO `New(...)` signature change).
- **No edits to `internal/server/guard.go`** — no MITM guard exists/required here.
- **No edits to inference (`selection.go`/`factory.go`/`runner.go`)** — MITM is NOT
  an inference-path concern; w7-plat-3 holds NO selection.go micro-serial.
- **No interface change** — `New(...)` signature PRESERVED (additive setters /
  default-constructed field only; MAP decision 9).
- **No destructive DDL / column renames** — additive `ensureTable`/`ensureColumn`
  ONLY (decision 2).
- **No new global state** — handlers compose `h.store`; the mitm service is
  constructed over `h.store` and holds its CA/proxy as fields.
- **No secret exposure** — the CA PRIVATE KEY is NEVER served, NEVER logged, NEVER in
  any DTO; only the PUBLIC CA cert PEM is served (§1.5 / §5 grep proofs). Stored per
  ESC-CA-STORE (file `0o600` default, or `key_enc`).
- **No real port bind / TLS handshake / OS-privileged op in any unit test** — those
  are integration-only behind the `MitmProxy` interface (§1.9); the unit suite is
  fully hermetic.

### 1.9 Integration-only / deferred surface (binding — the hermeticity boundary)

UNIT-TESTED (hermetic): `GenerateCA`, `CertPEM`, `MintLeaf` + chain `x509.Verify`,
the leaf cache, the `GetCertificate` closure (synthetic ClientHello), `nextBackoff`
(pure), and the full status/toggle/tool/ca-cert admin API via `newTestEnv` (+ fake
`MitmProxy`). INTEGRATION-ONLY (NOT unit-tested, thin, behind `MitmProxy`):
`proxy.Start`/`Stop` (real `net.Listen` + `tls.NewListener` + intercept-and-forward),
the restart-backoff loop's real sleeps. DEFERRED + ESCALATED (OS-privileged, out of
scope — §8 ESC-OS-PRIV): system-trust-store auto-install (PAR-PLAT-025 trust half) +
hosts-file patching (the `dns_override` OS-enforcement). Recorded at closeout in
`open-questions.md` with a "integration-only / OS-privileged" footnote on the affected
rows.

---

## 2. Precondition checks

Run all before any edit; abort and report to orchestrator on any failure.

```bash
# P0 — clean tree; record <base>
git status --porcelain     # expect empty EXCEPT a possibly-dirty ui/dist/index.html
                           # (gitignored build artifact — NEVER stage it, NEVER revert it).
                           # If anything ELSE is dirty, STOP. Use explicit `git add <file>`, never -A.
git rev-parse HEAD         # record as <base> for §5

# P1 — the gap is REAL (no Go for mitm; no TLS/cert code to collide with)
grep -nE '/api/mitm' internal/server/routes_admin.go ; echo "^ expect EMPTY"
grep -rniE '"/api/mitm|MitmHandler|MitmStatus|MitmToggle' internal/ cmd/ ; echo "^ expect EMPTY (no mitm Go)"
test ! -e internal/store/mitm.go && test ! -e internal/admin/mitm.go && echo "mitm admin/store gap OK"
test ! -d internal/platform/mitm && echo "platform/mitm pkg gap OK"
grep -nE 'mitm_tools|mitm_ca' internal/store/migrate.go ; echo "^ expect EMPTY (no mitm tables)"
grep -rnE 'x509|tls\.Certificate|crypto/x509|crypto/tls|GenerateKey' internal/ cmd/ ; echo "^ expect EMPTY (greenfield crypto)"

# P2 — the SHIPPED injection precedent to MIRROR is present (w7-plat-1 + w7-plat-2)
grep -n "proxyPools\|tunnels\|func (h \*Handlers) SetProxyProber\|func (h \*Handlers) SetTunnelRunner\|func (h \*Handlers) SetShutdownFunc\|func New(" internal/admin/handlers.go
grep -n "platform.NewProxyPoolService(st)\|tunnel.NewService(st)" internal/admin/handlers.go   # the New-constructs-field precedent

# P3 — reused surfaces present
grep -n "func writeData\|func writeError" internal/admin/respond.go
grep -n "func pathID" internal/admin/handlers.go
grep -n "func (h \*Handlers) recordAudit" internal/admin/audit.go
grep -n "func newTestEnv\|func call(" internal/admin/admin_test.go
grep -n "func LoadOrCreateSecret\|os.WriteFile(path, key, 0o600)\|os.MkdirAll(dataDir, 0o700)" internal/store/secret.go   # key-at-rest FILE precedent (ESC-CA-STORE A)
grep -n "func (s \*Store) DataDir" internal/store/apikeys.go                                                              # data-dir accessor
grep -n "s.cipher.Encrypt\|s.cipher.Decrypt" internal/store/oauthsessions.go internal/store/proxypools.go                # *_enc round (ESC-CA-STORE B)

# P4 — migrate pattern + global-flag (settings) surface
grep -n "tables := \|CREATE TABLE IF NOT EXISTS proxy_pools\|ensureColumn(db, col\|func ensureColumn" internal/store/migrate.go | head
grep -rnE 'func \(s \*Store\) .*Setting|settings\b' internal/store/*.go | head   # resolves ESC-GLOBAL-FLAG (is there a settings get/set?)

# P5 — the W6-m UI + spec present (consume-only) and the mock to mirror
test -f ui/e2e/mitm.spec.ts && echo "spec present"
test -f ui/e2e/mocks/handlers/mitm.ts && test -f ui/e2e/mocks/seed/mitm.ts && echo "mock+seed present"
grep -n "application/x-pem-file\|ca-cert\|enabled\|tools\|/api/mitm" ui/e2e/mocks/handlers/mitm.ts
grep -n "Request Inspector\|Response Modifier\|dns_override\|enabled\|status" ui/e2e/mocks/seed/mitm.ts
grep -nE "api/mitm|ca-cert|apiFetch|fetch\(" ui/src/routes/mitm.tsx   # confirm the raw-PEM plain-fetch + the {enabled,tools} status read

# P6 — routes_admin.go serial slot FREE (released by w7-plat-2)
git log --oneline -5 -- internal/server/routes_admin.go    # last touch = w7-plat-2 (merged); slot free
# Orchestrator MUST confirm: w7-plat-2 has CLOSED + released the slot; no concurrent
# W7 plan holds an unmerged routes_admin.go edit before T-routes.

# P7 — green at base
go test ./... && go vet ./... && go build ./...     # exit 0 (Go untouched-green; HERMETIC — no net/port/handshake)
cd ui && npm run build                               # exit 0 (build BEFORE e2e — e2e-hygiene)
cd ui && npx playwright test e2e/mitm.spec.ts        # PASS at base against the W6 mock; record in WORKFLOW.md
```

---

## 3. Exclusive file ownership

After w7-plat-3 merges, all CREATE files are owned by w7-plat-3; later plans consume,
never edit (MAP decision 7).

**CREATE — store (NEW):**

| File | Contract |
|---|---|
| `internal/store/mitm.go` | `MitmTool` struct + `ListMitmTools`/`GetMitmTool`/`UpsertMitmTool`/`SetMitmToolEnabled`/`GetMitmEnabled`/`SetMitmEnabled` (+ optional `EnsureMitmTools`); `boolToInt`, `time.Now().Unix()`, `ErrNotFound`. NO secret in this table (CA key handled separately — §1.3). Mirrors `proxypools.go`/`tunnels.go` patterns. |
| `internal/store/mitm_test.go` | Table-driven, temp `store.Open`: ensure/upsert→get→list(≥2 deterministic order)→tool-toggle (enabled+status derived)→unknown-id `ErrNotFound`; global enable get/set round-trips. RED first. |

**EXTEND — store (additive only):**

| File | Change (additive ONLY) |
|---|---|
| `internal/store/migrate.go` | ADD the `mitm_tools` table to the `tables` slice (mirror the SHIPPED `proxy_pools` block @ :191). IF ESC-GLOBAL-FLAG resolves to a table (not settings), ADD a tiny `mitm_state` single-row table too. ADDITIVE ONLY — no DROP/RENAME. |

**CREATE — domain (NEW package `internal/platform/mitm`):**

| File | Contract |
|---|---|
| `internal/platform/mitm/ca.go` | `CA` (cert + private key + cached cert PEM) + `GenerateCA(opts)` (PURE) + `LoadOrCreateCA(dataDir)` (file I/O wrapper, ESC-CA-STORE A; or DB via store if B) + `CertPEM()` (PURE, public cert only) + `MintLeaf(host)` (PURE, CA-signed leaf) + the leaf cache (`getLeaf(host)`, mutex-guarded). No `init()`; errors-as-values. The PRIVATE KEY is never serialized into any returned value except internal TLS use. |
| `internal/platform/mitm/ca_test.go` | PURE-crypto unit tests: `GenerateCA`→parses back, `IsCA`, `KeyUsage CertSign`; `CertPEM()`→valid PEM CERTIFICATE block, NO PRIVATE KEY in output; `MintLeaf(host)`→`x509.Verify(leaf, pool{CA})` SUCCEEDS, `DNSNames` has host, leaf not a CA; cache miss-mints/hit-returns-same. RED first. NO port/network/handshake. |
| `internal/platform/mitm/proxy.go` | `MitmProxy` interface (`Start(addr) error; Stop() error; Running() bool`) + the REAL `listenerProxy` impl: a `tls.Config` with a `GetCertificate(hello)` closure that mints/returns the per-SNI leaf via the CA+cache; `Start` = `net.Listen`+`tls.NewListener`+intercept-forward (INTEGRATION-ONLY, §1.9); `Stop`/`Running`. The `GetCertificate` closure is factored so it is callable (and unit-testable) without a bind. No `init()`. |
| `internal/platform/mitm/service.go` | `Service`: `*store.Store` + lazy `*CA` + a `MitmProxy` field; `Status`/`Toggle`/`ToggleTool`/`CACertPEM` + the restart-backoff loop (`nextBackoff` PURE); `NewService(st)` constructs WITHOUT binding (real default `listenerProxy` built lazily / on enable); `SetProxy(MitmProxy)` overrides for tests (mirror `SetRunner`). No `init()`. |
| `internal/platform/mitm/service_test.go` | Via a FAKE `MitmProxy` + a real (cheap) CA in a temp dir: `Toggle`→flips+persists (fake Start not really binding); `ToggleTool`→flips+status; `CACertPEM`→valid PEM; the `GetCertificate` closure with a synthetic `ClientHelloInfo{ServerName}`→leaf verifies against CA; `nextBackoff` doubling+cap. RED first. Deterministic, NO port/network/handshake. |

**CREATE — transport (NEW):**

| File | Contract |
|---|---|
| `internal/admin/mitm.go` | `MitmStatus`/`MitmToggle`/`MitmCACert`/`MitmToolToggle` + `mitmToolDTO` + request/validate; `writeData`/`writeError` for status/toggle/tool; **raw-PEM ctx write (NOT writeData) for ca-cert (§1.5)**; 404 on unknown tool id; `h.recordAudit` after each mutation (best-effort). NEVER echoes any key material. Reads `{id}` via `pathID(ctx.UserValue("id"))`. |
| `internal/admin/mitm_test.go` | via `newTestEnv` + `SetMitmProxy(fakeProxy)`: status→enabled+2 tools; toggle→flips+audit; tool toggle→flips+status; unknown id→404; **ca-cert→`Content-Type: application/x-pem-file` + raw PEM body that `x509.ParseCertificate` accepts + contains NO PRIVATE KEY**; **no status/toggle/tool response leaks key material**; an audit entry on toggle. RED first. Deterministic — fake proxy + real cheap CA in tempdir, NO port/network/handshake. |

**MODIFY — handlers wiring (additive only — mirror the SHIPPED tunnels/SetTunnelRunner):**

| File | Change |
|---|---|
| `internal/admin/handlers.go` | ADDITIVE: add a `mitm *mitm.Service` field; construct it in `New` via `mitm.NewService(st)` (mirror `tunnels: tunnel.NewService(st)` @ :56). ADD `SetMitmProxy(p mitm.MitmProxy)` forwarding to `h.mitm.SetProxy` (mirror `SetTunnelRunner` @ :99-100). NO `New(...)` signature change. |

**MODIFY — serial-slot route registration (additive only):**

| File | Change |
|---|---|
| `internal/server/routes_admin.go` | ADD the 4 route lines (§1.7) BELOW the tunnels block (:140-143). NOTHING else. ONE commit. SERIAL SLOT — only holder while live; RELEASE to w7-misc on close. |

**MODIFY — e2e mock corrections (mirror real Go, decision 1 — ONLY on real divergence):**

| File | Change |
|---|---|
| `ui/e2e/mocks/handlers/mitm.ts` (BODY) | VERIFY the status/toggle/tool bodies match the Go `{data}` envelope + DTO field names; **the ca-cert branch ALREADY mirrors the binding raw-PEM contract (`route.fulfill` + `application/x-pem-file`) — verify, do NOT change.** Correct ONLY a real divergence in status/toggle/tool envelope. (ESC-MOCK if a correction reds a spec.) |
| `ui/e2e/mocks/seed/mitm.ts` (BODY) | Already the 5-field `MitmTool` shape (2 named rows). The top-level `ca_cert` is ignored by the page — LEAVE as-is. Verify field names; correct only on a real field-name divergence. |

**FORBIDDEN:** everything else. Explicitly: all pre-existing `internal/admin/*.go`
except the NEW mitm files + the ADDITIVE handlers.go field/setter; all
`internal/store/*.go` except mitm (NEW) + migrate (additive table); all pre-existing
`internal/platform/*` (proxypools/outboundproxy/tunnel — SHIPPED, CONSUME the
precedent, do NOT edit); `internal/store/secret.go` (CONSUME the key-at-rest
precedent, do NOT edit); `internal/server/guard.go`; all `internal/inference/*` (MITM
is not an inference concern); all UI `ui/src/**` (FROZEN, decision 8); all other
mocks/seeds/specs; `ui/package.json` + lockfile; `ui/vite.config.ts`;
`ui/playwright.config.ts`; `ui/dist/**` (gitignored — NEVER stage, NEVER revert
`ui/dist/index.html`). Touching any of these is an automatic REJECT.

---

## 4. TDD tasks

Cadence (strict, AGENTS.md "TDD always: write test first, see it fail, write minimum
code to pass"): **no Go impl file may exist before its `_test.go` is committed RED.**
`go test ./... && go vet ./... && go build ./...` green at EVERY commit, FULLY
HERMETIC (no network, no port bind, no real TLS handshake, no OS-privileged op). The
e2e spec stays green throughout (real Go is additive; mock corrections — if any —
mirror it). Order: CA pure-crypto core → mitm store → service state machine (+ fake
proxy) → admin handlers (incl raw-PEM ca-cert) → routes serial slot → mock verify →
closeout.

### T-ca — STEP(a) RED, STEP(b) impl (the pure-crypto core — highest value)
STEP(a): write `internal/platform/mitm/ca.go` (types only — compiles) and
`internal/platform/mitm/ca_test.go` for `GenerateCA` / `CertPEM` / `MintLeaf` +
chain-verify + leaf-cache.
`go test ./internal/platform/mitm/ -run 'CA|Leaf|Cert'` → FAIL. Commit RED:
`phase-1/w7-plat-3: failing MITM CA-gen + leaf-mint + chain-verify tests (TDD red)`.
STEP(b): implement `GenerateCA` (self-signed root, IsCA, KeyUsageCertSign),
`CertPEM` (public PEM), `MintLeaf` (CA-signed leaf), the leaf cache, and
`LoadOrCreateCA` (file I/O wrapper per ESC-CA-STORE; the I/O is thin, the pure core is
tested). Gates green. Commit:
`phase-1/w7-plat-3: MITM root CA generation + leaf-cert minting + CA signing`.

### T-mitmstore — STEP(a) RED store, STEP(b) impl
STEP(a): write `internal/store/mitm_test.go`; ADD the `mitm_tools` table to
`migrate.go` (+ `mitm_state` iff ESC-GLOBAL-FLAG → table). `go test ./internal/store/
-run Mitm` → FAIL. Commit RED:
`phase-1/w7-plat-3: failing MITM store tests (TDD red)`.
STEP(b): implement `internal/store/mitm.go` (list/get/upsert/tool-toggle/global flag
+ optional EnsureMitmTools). Gates green. Commit:
`phase-1/w7-plat-3: MITM tool store + global enable flag`.

### T-service — STEP(a) RED state machine, STEP(b) impl (fake proxy + real CA)
STEP(a): write `internal/platform/mitm/service_test.go` against a FAKE `MitmProxy` +
a real cheap CA in a temp dir (toggle/tool/ca-cert/GetCertificate-closure/nextBackoff).
→ FAIL. Commit RED:
`phase-1/w7-plat-3: failing MITM service tests (TDD red)`.
STEP(b): implement `service.go` (Status/Toggle/ToggleTool/CACertPEM + `NewService(st)`
+ `SetProxy` test override + `nextBackoff` pure) + `proxy.go` (the `MitmProxy`
interface + the `GetCertificate` closure + the real `listenerProxy` whose
`Start`/`Stop` bind/close — INTEGRATION-ONLY, §1.9, never invoked in unit tests).
Gates green (fake proxy only). Commit:
`phase-1/w7-plat-3: MITM service + SNI cert cache + injectable proxy listener`.

### T-admin — STEP(a) RED handlers, STEP(b) impl
STEP(a): write `internal/admin/mitm_test.go` (via `newTestEnv` + `SetMitmProxy` fake):
status=enabled+2 tools, toggle→flips+audit, tool toggle→flips+status, unknown id→404,
**ca-cert→raw PEM + `application/x-pem-file` + no-key-echo**, no-key-leak. ADD the
ADDITIVE `handlers.go` field + `SetMitmProxy` (so the test compiles). → FAIL. Commit
RED: `phase-1/w7-plat-3: failing MITM admin handler tests (TDD red)`.
STEP(b): implement `internal/admin/mitm.go` (status/toggle/tool handlers + DTO + 404
guard + `h.recordAudit` + the raw-PEM ca-cert handler — `ctx.SetContentType(
"application/x-pem-file")` + `ctx.SetBody(pem)`, NOT writeData). Gates green. Commit:
`phase-1/w7-plat-3: MITM admin API (status/toggle/ca-cert/tool-toggle)`.

### T-routes — serial-slot route registration
TAKE the routes_admin.go serial slot (orchestrator confirms FREE at P6 — released by
w7-plat-2). Add the 4 route lines (§1.7). Gates green. Commit (ONE commit touches the
serial file):
`phase-1/w7-plat-3: register MITM admin routes (serial slot)`.

### T-mocks — mock-body verify/correct (mirror real Go, decision 1)
VERIFY `mitm.ts` status/toggle/tool bodies match the Go `{data}` envelope + 5-field
DTO; **the ca-cert raw-PEM branch ALREADY mirrors the contract — verify, do NOT
change.** Verify the seed (2 named tools; `ca_cert` ignored — leave). Correct ONLY a
real divergence. Gates: `cd ui && npm run build` green (BEFORE playwright);
`npx playwright test e2e/mitm.spec.ts` green (still). If a correction reds a
non-w7-plat-3 spec, STOP + ESCALATE (§8 ESC-MOCK). Commit (only if a change is made):
`phase-1/w7-plat-3: correct MITM mock to mirror real Go DTOs`.

### T-close — full gates + closeout
```bash
go test ./... && go vet ./... && go build ./...                      # HERMETIC — no net/port/handshake
go test ./internal/platform/mitm/... ./internal/admin/ -run Mitm -v
go test ./internal/store/ -run Mitm -v
cd ui && npm run build                                               # BEFORE playwright (e2e-hygiene)
cd ui && npx playwright test e2e/mitm.spec.ts                        # green (ISOLATED, no concurrent playwright)
cd ui && npx playwright test                                        # full suite green (mod the known pre-existing comprehensive.spec flake — open-questions w7-plat-1 §115)
```
Flip the matrix: PAR-PLAT-024/025/028 → HAVE (real Go; the live MITM listener +
OS-privileged trust-store/hosts-file parts integration-only + escalation-recorded —
§1.9 / §8); PAR-UI-013 PARTIAL → HAVE. Mark `open-questions.md` w6-m ESC-1a RESOLVED
with a cite; append any new open items (§8 — OS-privileged trust-store/hosts-file
escalation; listener integration-only note; ESC-CA-STORE / ESC-GLOBAL-FLAG /
ESC-KEYTYPE / ESC-BACKOFF decisions). Update `docs/WORKFLOW.md` (P7 base observation;
the ESC-* decisions; the serial-slot take/release; the mock verify/correct). Final
commit:
`phase-1/w7-plat-3: close — MITM (CA + proxy core + admin) Go; matrix flip; mock mirror`.
**On the close commit, RELEASE the routes_admin.go serial slot to w7-misc.**

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0. Diff gate is
**w7-plat-3 commit-range-scoped** (§7).

**Test gates (HERMETIC — no network, no port bind, no real TLS handshake, no
OS-privileged op)**
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/platform/mitm/... ./internal/admin/ -run Mitm -v` → exit 0,
  all pass (CA-gen ≥1; CertPEM valid+no-key ≥1; MintLeaf+x509.Verify ≥1; leaf cache
  ≥1; GetCertificate closure ≥1; nextBackoff ≥1; admin status=2 tools/toggle/tool/
  unknown-404/ca-cert-raw-PEM/no-key-leak).
- `go test ./internal/store/ -run Mitm -v` → exit 0.
- `cd ui && npm run build` → exit 0 (BEFORE playwright).
- `cd ui && npx playwright test e2e/mitm.spec.ts` → exit 0, all pass, 0 skipped
  (ISOLATED; no concurrent playwright; `ui/dist/index.html` NEVER reverted).
- `cd ui && npx playwright test` → exit 0 mod the known pre-existing
  `comprehensive.spec.ts` flake (open-questions w7-plat-1 §115); no mitm-related
  green-at-base spec goes red.

**TDD-order proof** — each impl file's covering test appears in an earlier-or-equal
commit:
```bash
for pair in \
  "internal/platform/mitm/ca_test.go:internal/platform/mitm/ca.go" \
  "internal/platform/mitm/service_test.go:internal/platform/mitm/service.go" \
  "internal/store/mitm_test.go:internal/store/mitm.go" \
  "internal/admin/mitm_test.go:internal/admin/mitm.go" ; do
  tf=${pair%%:*}; im=${pair##*:}
  ct=$(git log --format=%ct --diff-filter=A -1 -- "$tf")
  cf=$(git log --format=%ct --diff-filter=A -1 -- "$im")
  [ "$ct" -le "$cf" ] || echo "TDD VIOLATION: $im"     # prints nothing
done
```

**Grep proofs**
```bash
# mitm transport + store
grep -n "func (h \*Handlers) MitmStatus\|MitmToggle\|MitmCACert\|MitmToolToggle" internal/admin/mitm.go
grep -n "id\|name\|enabled\|dns_override\|status" internal/admin/mitm.go        # canonical 5-field DTO
grep -n "func (s \*Store) ListMitmTools\|GetMitmTool\|UpsertMitmTool\|SetMitmToolEnabled\|GetMitmEnabled\|SetMitmEnabled" internal/store/mitm.go
grep -n "writeData\|writeError\|recordAudit" internal/admin/mitm.go             # envelope + audit (NON-ca-cert handlers)
grep -n "pathID(ctx.UserValue(\"id\"))\|tool not found\|404\|StatusNotFound" internal/admin/mitm.go  # {id} guard
# CA pure-crypto core + x509.Verify in tests
grep -n "func GenerateCA\|func (c \*CA) CertPEM\|func (c \*CA) MintLeaf\|func LoadOrCreateCA" internal/platform/mitm/ca.go
grep -n "x509.Verify\|x509.NewCertPool\|x509.ParseCertificate" internal/platform/mitm/ca_test.go   # chain verify in tests
# service injection seam (mirrors SHIPPED Runner/SetRunner + tunnels field)
grep -n "type MitmProxy interface\|func.*SetProxy\|func NewService" internal/platform/mitm/*.go
grep -n "mitm\b\|func (h \*Handlers) SetMitmProxy\|mitm.NewService(st)" internal/admin/handlers.go
# raw-PEM ca-cert handler (the binding exception)
grep -n "application/x-pem-file\|SetContentType\|SetBody" internal/admin/mitm.go   # raw PEM, NOT writeData
! grep -n "writeData" internal/admin/mitm.go | grep -i "cacert\|ca_cert\|MitmCACert" && echo "ca-cert does NOT use writeData OK"
# routes
grep -nE '/api/mitm' internal/server/routes_admin.go
# no init(); no global state
! grep -rn "func init(" internal/admin/mitm.go internal/store/mitm.go internal/platform/mitm/*.go && echo "no init() OK"
```

**No-listen / no-handshake-in-test proofs (binding — the hermeticity guarantee)**
```bash
# unit tests NEVER bind a port nor perform a real TLS handshake nor OS-privileged op:
! grep -nE 'net\.Listen|tls\.NewListener|tls\.Dial|\.Listen\(|http\.Get|exec\.Command|os/exec' \
   internal/platform/mitm/ca_test.go internal/platform/mitm/service_test.go \
   internal/admin/mitm_test.go internal/store/mitm_test.go && echo "no listen/dial/handshake/exec in tests OK"
# the real listen/bind lives ONLY in the proxy impl (proxy.go), behind MitmProxy:
grep -nE 'net\.Listen|tls\.NewListener' internal/platform/mitm/proxy.go  # expect MATCHES here only
# the fake proxy used by tests implements MitmProxy without any bind:
grep -n "MitmProxy\b" internal/platform/mitm/service_test.go internal/admin/mitm_test.go  # fake impl present
```

**No-secret-exposure proofs (binding — the CA private key NEVER leaks)**
```bash
# the CA private key never appears in any DTO/response/served body:
! grep -nE 'json:"key"|json:"private_key"|json:"ca_key"|PRIVATE KEY' internal/admin/mitm.go && echo "no key json field / no PRIVATE KEY in handler OK"
grep -nA8 'type mitmToolDTO struct' internal/admin/mitm.go ; echo "^ must NOT contain any key material (only id/name/enabled/dns_override/status)"
# the served ca-cert body is the PUBLIC cert only (CertPEM never emits a key):
grep -n "PRIVATE KEY" internal/platform/mitm/ca.go ; echo "^ may appear ONLY in the persist-key path (LoadOrCreateCA write), NEVER in CertPEM"
# the runtime no-key-leak test marshals every status/toggle/tool response + the
# ca-cert body and asserts none contains 'PRIVATE KEY' nor the raw key bytes
# (asserted in mitm_test.go).
# key-at-rest: per ESC-CA-STORE — (A) file 0o600 (mirror secret.go) OR (B) key_enc:
grep -nE '0o600|key_enc|s.cipher.Encrypt' internal/platform/mitm/ca.go internal/store/mitm.go   # the chosen mechanism present
# additive migrations only (no DROP/RENAME introduced by this plan)
git diff <base>..HEAD -- internal/store/migrate.go | grep -E '^\+' | grep -iE 'DROP COLUMN|RENAME COLUMN|DROP TABLE' | wc -l   # = 0
```

**Negative / freeze proofs (w7-plat-3 commit-range — §7)**
```bash
R="<first-w7-plat-3>^..<last-w7-plat-3>"
# Only the sanctioned Go files changed:
git diff $R --name-only -- internal/ | grep -vE \
 'internal/store/(mitm|migrate)(_test)?\.go|internal/platform/mitm/.*\.go|internal/admin/(mitm)(_test)?\.go|internal/admin/handlers\.go|internal/server/routes_admin\.go' \
 | wc -l                                                                  # = 0
# Frozen admin handlers untouched (incl the SHIPPED proxypools + tunnels):
git diff $R --name-only -- internal/admin/auth.go internal/admin/apikeys.go internal/admin/virtualkeys.go internal/admin/providers.go internal/admin/connections.go internal/admin/combos.go internal/admin/proxypools.go internal/admin/tunnels.go internal/admin/version.go | wc -l   # = 0
# Frozen guard + the key-at-rest precedent (secret.go) CONSUMED, not edited:
git diff $R --name-only -- internal/server/guard.go internal/store/secret.go | wc -l   # = 0
# SHIPPED platform proxy-pools + tunnels untouched (precedent consumed, not edited):
git diff $R --name-only -- internal/platform/proxypools.go internal/platform/outboundproxy.go internal/store/proxypools.go internal/store/tunnels.go internal/platform/tunnel/ internal/admin/tunnels.go | wc -l   # = 0
# inference untouched (mitm is not an inference concern):
git diff $R --name-only -- internal/inference/ | wc -l                   # = 0
# handlers.go = additive only (no deletions of existing logic):
git diff $R -- internal/admin/handlers.go | grep -E '^-' | grep -v '^---' | wc -l   # = 0
# UI frozen except the sanctioned mock/seed bodies (if touched at all):
git diff $R --name-only -- ui/ | grep -vE \
 'ui/e2e/mocks/handlers/mitm\.ts|ui/e2e/mocks/seed/mitm\.ts' | wc -l     # = 0
git diff $R --name-only -- ui/src/ | wc -l                               # = 0 (src frozen)
git diff $R --name-only -- ui/dist/ | wc -l                             # = 0 (dist gitignored/never staged)
# routes_admin.go = exactly ONE commit, additive:
git log --oneline $R -- internal/server/routes_admin.go | wc -l          # = 1
git diff $R -- internal/server/routes_admin.go | grep -E '^-' | grep -v '^---' | wc -l   # = 0 (no deletions)
```

---

## 6. Out of scope (restated, binding)

No UI src edits (decision 8 — pages/components/routes/stores frozen); only the
sanctioned mitm mock-body + seed verification, and ONLY on a real Go-vs-mock
divergence (the raw-PEM ca-cert contract already matches — likely NO change). No
edits to pre-existing admin handler bodies (auth, apikeys, virtualkeys, providers*,
connections, combos, version, usage, **proxypools**, **tunnels**). No edits to the
SHIPPED `internal/platform/{proxypools,outboundproxy}.go` + `internal/platform/tunnel/*`
(CONSUME the service-field + SetX-injector precedent, do NOT edit). No edit to
`internal/store/secret.go` (CONSUME the key-at-rest FILE precedent, never modify). No
`guard.go` edit. No inference edits (MITM is not an inference-path concern — NO
selection.go micro-serial). No interface / `New(...)` signature change (additive
setters / default-constructed field only). No destructive DDL — additive
`ensureTable`/`ensureColumn` only. No new global state. No other platform domains
(proxy-pools + tunnels SHIPPED). **No OS-privileged trust-store auto-install /
hosts-file patching** — DEFERRED + escalated (§1.9 / §8 ESC-OS-PRIV). **No real port
bind / TLS handshake / OS-privileged op in any unit test** — those are
integration-only behind `MitmProxy` (§1.9); the unit suite is fully hermetic. **No CA
private-key exposure** — only the PUBLIC CA cert PEM is served; the key is stored at
rest (file `0o600` default, or `key_enc`) and NEVER echoed/logged/DTO'd. Mock-vs-Go
contradiction → escalate (§8), never fudge a mock or edit a frozen handler. NEVER
revert `ui/dist/index.html`; NEVER run concurrent playwright; `npm run build` before
e2e (e2e-hygiene).

## 7. Diff-gate scope

W7 platform plans (plat-1 SHIPPED / plat-2 SHIPPED / plat-3) commit to main, so a
broad `<base>..HEAD` range sweeps in sibling commits. The diff gate MUST be scoped to
w7-plat-3's own commits. Isolate them:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w7-plat-3:" | awk '{print $1}'`
then `git diff <first-w7-plat-3>^..<last-w7-plat-3> -- [file list]`.

`git diff <first>^..<last> --name-only` must be exactly a subset of:
```
internal/store/mitm.go
internal/store/mitm_test.go
internal/store/migrate.go                 (additive mitm_tools [+mitm_state] table; ONE concern)
internal/platform/mitm/ca.go
internal/platform/mitm/ca_test.go
internal/platform/mitm/proxy.go
internal/platform/mitm/service.go
internal/platform/mitm/service_test.go
internal/admin/mitm.go
internal/admin/mitm_test.go
internal/admin/handlers.go                (ADDITIVE mitm field + SetMitmProxy; no New() sig change)
internal/server/routes_admin.go           (serial-slot additive routes; ONE commit)
ui/e2e/mocks/handlers/mitm.ts             (body only — verify/correct on real divergence; ca-cert raw-PEM unchanged)
ui/e2e/mocks/seed/mitm.ts                 (verify; correct only on divergence)
.planning/parity/matrix/*                  (row flips)
.planning/parity/plans/open-questions.md
docs/WORKFLOW.md
```
Any file outside this list in the scoped diff is an automatic review REJECT.
`internal/server/guard.go`, `internal/store/secret.go`, the SHIPPED
`internal/platform/{proxypools,outboundproxy}.go` + `internal/platform/tunnel/**`,
the pre-existing admin handlers, `internal/inference/**`, and all `ui/src/**` are
deliberately ABSENT — touching them is an automatic REJECT. The `routes_admin.go`
edit must appear in exactly ONE commit (§5); the serial slot is released to w7-misc on
close.

## 8. Escalations / decisions (explicit — recommended defaults, do not fabricate)

- **ESC-OS-PRIV (ESCALATED — system-trust-store auto-install + hosts-file patching,
  recommended default).** Auto-installing the root CA into the OS/browser trust store
  (PAR-PLAT-025 trust half) and patching the hosts file (the `dns_override`
  OS-enforcement) are OS-privileged (root / registry / `/etc/hosts`) and ill-suited to
  a server binary's unit tests. **Decision: the parity bar is CA-gen + CA-cert serving
  (raw PEM) + the proxy CRYPTO core (mint/sign/verify/cache) + the admin API (FULLY
  unit-tested); the live listener is integration-only; the trust-store auto-install +
  hosts-file patching are DEFERRED to a "desktop/agent" escalation** (mirrors the w6-m
  ESC at open-questions §80 — "drop for server binary; add g0router service install").
  The `dns_override` field is STORED + surfaced (config), not OS-enforced. Record at
  closeout: PAR-PLAT-025 marked HAVE with an "integration-only listener / OS-privileged
  trust-store+hosts deferred" footnote. Flag for orchestrator confirmation.
- **ESC-CA-STORE (RESOLVED at authoring — CA key-at-rest mechanism, binding default).**
  The root CA cert is PUBLIC (served raw PEM); the key is SECRET. **Decision: (A) key
  + cert as files under the data dir** — `dataDir/mitm-ca.key` (`0o600`) +
  `dataDir/mitm-ca.crt`, generate-on-first-use, mirroring `internal/store/secret.go:15-36`
  (the master `secret.key` precedent: `os.MkdirAll(dataDir, 0o700)` + `os.WriteFile(…,
  0o600)`; `store.DataDir()` @ `apikeys.go:92`). RATIONALE: the closest shipped
  precedent for raw key MATERIAL; avoids a per-handshake decrypt; the public cert file
  is exactly the raw-PEM endpoint body. **Alternative (B): a `mitm_ca(cert_pem,
  key_enc, created_at)` single-row DB table** with the key `*_enc` via `s.cipher`
  (the `verifier_enc`/`password_enc` precedent — `oauthsessions.go:21`,
  `proxypools.go:30`) — keeps everything in the single WAL store (one backup surface).
  If only ONE option survives: (A) is recommended; (B) is fully viable and is the
  switch if the operator wants single-store backup semantics. EITHER WAY the key is
  NEVER served/logged/DTO'd. Flag for orchestrator confirmation.
- **ESC-SCHEMA (RESOLVED at authoring — typed vs JSON columns, binding default).** The
  `mitm_tools` row is a fixed 5-field shape with a `WHERE id=?` per-tool toggle.
  **Decision: typed columns** (clean lookup; matches the SHIPPED proxy-pools/tunnels
  precedent of typed columns for fixed-shape domains). Flag.
- **ESC-GLOBAL-FLAG (RESOLVED at authoring — where the global `enabled` lives, binding
  default).** The `/api/mitm/status` `enabled` is a single global boolean.
  **Decision: store it in the existing `settings` key-value surface** under
  `mitmEnabled` (mirror how `tunnelDashboardAccess`/`tunnelUrl` live in settings —
  `guard.go:135-141`) if the store exposes a settings get/set (VERIFY at P4).
  **Alternative:** a tiny single-row `mitm_state(enabled INTEGER)` table (additive).
  Default: settings if cleanly available; else the single-row table. Flag.
- **ESC-KEYTYPE (RESOLVED at authoring — RSA vs ECDSA for the CA, binding default).**
  **Decision: ECDSA P-256** (`crypto/ecdsa` + `elliptic.P256`) — fast key-gen (cheap
  deterministic unit tests), modern, widely trusted by clients for MITM leaf certs.
  **Alternative: RSA-2048** (maximally compatible with ancient clients) — slower
  key-gen, heavier tests. Default: ECDSA P-256; switch to RSA-2048 only if a target
  client rejects ECDSA leaves. Flag.
- **ESC-SEED-ROWS (RESOLVED at authoring — how `ListMitmTools` always returns ≥2,
  binding default).** **Decision: `EnsureMitmTools()` seeds the 2 named tools (Request
  Inspector, Response Modifier) once on migrate** — they are named domain tools (not
  synthetic placeholders), the seed keeps `ListMitmTools` honest and the 2-row spec
  green. **Alternative:** the status handler overlays 2 known tools from a constant +
  any stored row (no seed migration). Default: seed via `EnsureMitmTools`. Flag.
- **ESC-BACKOFF (RESOLVED at authoring — restart backoff, binding default).** No
  existing backoff helper in-tree (verified — `grep -rn backoff internal/` EMPTY).
  **Decision: a small in-package bounded exponential backoff** — `nextBackoff(attempt)
  time.Duration` doubling from 1s, capped at 30s, max 5 attempts, abort on
  `Stop`/disable. The `nextBackoff` POLICY is PURE + unit-tested (doubling + cap); the
  loop that sleeps between `proxy.Start` attempts is integration-only (real sleeps on a
  real listener). Flag.
- **ESC-CACHE (RESOLVED at authoring — leaf-cert cache eviction, binding default).**
  **Decision: an unbounded `map[host]tls.Certificate` guarded by a `sync.RWMutex`**,
  minted-on-miss. RATIONALE: the host set a MITM proxy sees is bounded in practice; an
  LRU/TTL is premature. **Alternative:** an LRU with a size cap or a per-leaf TTL
  honoring the leaf NotAfter. Default: unbounded map (simplest, deterministic to
  unit-test); record an LRU/TTL follow-up in open-questions if memory becomes a
  concern. Flag.
- **ESC-MOCK (CONDITIONAL — mock ripple).** `mitm.ts`/`seed/mitm.ts` are w6-m-owned and
  mitm-only (not shared). VERIFY the status/toggle/tool bodies match the Go `{data}`
  envelope + 5-field DTO; **the ca-cert raw-PEM branch already mirrors the binding
  contract — verify, do NOT change.** Correct ONLY a real divergence. If a body
  correction reds a non-w7-plat-3 spec or a seed correction ripples, STOP and ESCALATE
  for orchestrator serialization — no fudge, no frozen-branch edit.
- **ESC-ROUTE (CONDITIONAL — fasthttp/router precedence).** The three
  `/api/mitm/<word>` static routes vs `/api/mitm/tools/{id}` (param under a distinct
  `tools/` segment) do not collide (distinct path prefixes). If the matcher
  mis-disambiguates or panics on a conflict, STOP and ESCALATE for a path arrangement
  — never silently diverge page/mock/Go.
- **ESC-ARCH (CONDITIONAL — layering).** Per the w7-gov-1/plat-2 ESC-ARCH finding (no
  in-tree arch test strictly enforces transport→domain→repository; layering is by
  convention), build the `internal/platform/mitm` domain service because the CA + cert
  cache + listener + backoff warrant a seam (reused beyond the handler — exactly as
  proxy-pools got `platform/proxypools.go` and tunnels got `platform/tunnel/`). Follow
  the SHIPPED precedent; do NOT pre-guess a stricter rule.
- **Serial-slot dependency (§1.7 / P6).** w7-plat-3 TAKES the routes_admin.go slot in
  chain order (… → w7-plat-1 → w7-plat-2 → **w7-plat-3** → w7-misc; w7-plat-2 releases
  it on its close) and RELEASES it to w7-misc on close. NO selection.go micro-serial
  (MITM does not touch inference). Orchestrator confirms exactly one unmerged
  routes_admin.go holder (decision 3 / MAP §219-224) before T-routes — specifically
  that w7-plat-2 has CLOSED + released.
```
