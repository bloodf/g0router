# Micro-plan w7-gov-3 — Governance backends C: guardrails + alert-channels (Go)

```
wave: 7
plan: w7-gov-3
status: READY (rev 1 — authored against merged Waves 0–6 + w7-gov-1 + w7-gov-2
  (both shipped, gate-green), live tree @ <base>; WAVE-7-MAP w7-gov-3 row ~line 173;
  serial chain §219-224; reconciliation decision 1 §36/§245; freeze rules §267)
runs: governance+routing track. Disjoint domain/store/admin files from w7-gov-1
  (merged) and w7-gov-2 (merged). TAKES the internal/server/routes_admin.go SERIAL
  SLOT after w7-gov-2 RELEASES it (chain: w7-platnodes → w7-route → w7-gov-1 →
  w7-gov-2 → **w7-gov-3** → w7-mcp-3 → w7-plat-1 → w7-plat-2 → w7-plat-3 → w7-misc;
  MAP §219-224). w7-gov-3 is the LAST gov holder — it RELEASES the slot to the next
  chain holder (w7-mcp-3, or whichever non-gov plan the orchestrator sequences next)
  on close.
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w7-gov-3:
ref-source: 9router frozen @ 827e5c3 — governance guardrails + alert-channels
  surfaces; the BINDING contract for W7 is the W6 e2e mock (decision 1: real Go wins,
  mock corrected to mirror it). Mock sources:
    ui/e2e/mocks/handlers/{guardrails,alert-channels}.ts + seed/{guardrails,alert-channels}.ts.
base: <base> = git rev-parse HEAD recorded at P0. Substitute the actual SHA
  everywhere §5 says <base>.
go-serial-slot: this plan holds the SINGLE in-flight edit to
  internal/server/routes_admin.go while live (W3/W4/W5/W6/w7-gov-1/w7-gov-2 lesson;
  MAP decision 3). Slot must be FREE at P-check (w7-gov-2 merged + slot released)
  before T-routes. RELEASE to the next chain holder (w7-mcp-3) on close.
new-route: NO UI route files. Both UI pages (/guardrails, /alerts) ALREADY SHIPPED
  in w6-k against mocks; this plan builds the REAL Go so the pages flip variant-HAVE
  → true-HAVE and corrects the mock bodies/seeds to mirror the Go DTOs.
```

---

## 1. Scope — PAR rows + the two domains

### Rows this plan closes

| Row / item | Claim | Target state after w7-gov-3 |
|---|---|---|
| open-questions w6-k **ESC-1d** (guardrails backend absent) | real `GET /api/guardrails` (config) + `PUT /api/guardrails` (update config) + `POST /api/guardrails/test` (standalone blocklist/PII evaluator → `{blocked, redacted_prompt, matches}`) | true-HAVE (Go — NEW `internal/store/guardrails.go` + `internal/governance/guardrails.go` (evaluator seam) + `internal/admin/guardrails.go`, §1.4) |
| open-questions w6-k **ESC-1f** (alert-channels backend absent) | real `GET/POST /api/alert-channels` + `GET/PUT/DELETE /api/alert-channels/{id}` + `POST /api/alert-channels/{id}/test` (real, deterministic-in-tests test-notification) | true-HAVE (Go — NEW `internal/store/alertchannels.go` + `internal/governance/alertchannels.go` (dispatcher seam) + `internal/admin/alerts.go`, §1.5) |

Matrix flips at closeout (§4 T-close): in `.planning/parity/matrix/9router-ui.md`, the
guardrails + alerts governance rows (the last two of the w6-k governance cluster,
mirroring how w7-gov-1 flipped teams/audit and w7-gov-2 flipped feature-flags/prompts)
→ variant-HAVE→true-HAVE with a cite to this plan. Mark `open-questions.md` w6-k
ESC-1d/1f RESOLVED. **w7-gov-3 closes the final two governance domains; with it the
entire w6-k governance cluster (ESC-1a..1f + ESC-2) is backed by real Go.**

### 1.1 Preconditions already satisfied by merged waves (evidence)

- **W6-k UI is SHIPPED and FROZEN (consume-only, MAP decision 8 / §267).** The
  `/guardrails` page (`ui/src/routes/guardrails.tsx` + the prompt tester
  `ui/src/components/governance/guardrails-tester.tsx`) and the `/alerts` page
  (`ui/src/routes/alerts.tsx` + `ui/src/components/governance/alert-channel-form-modal.tsx`)
  render against the registered mocks. The binding acceptance contracts are the
  existing specs (must stay green at closeout):
  - `ui/e2e/guardrails.spec.ts` — 3 tests: page loads ("Guardrails",
    `guardrails.spec.ts:10`); **prompt tester** — fills `input[aria-label="Test prompt"]`
    with `"my secret password"`, clicks `button:has-text("Test")`, asserts the body
    matches `/blocked/i` (`:14-22`); config form — `[data-testid="guardrails-enabled"]`
    + `[data-testid="guardrails-blocklist"]` visible, `[data-testid="guardrails-save"]`
    click fires a `PUT /api/guardrails` (`:24-37`). **CRITICAL:** the tester asserts
    `/blocked/i` against `"my secret password"` — the real Go `/test` MUST compute
    `blocked:true` here (the seed enables guardrails + blocklists `password`/`secret`,
    so a case-insensitive substring match fires; §1.4 / §8 ESC-GR-EVAL).
  - `ui/e2e/alerts.spec.ts` — 5 tests: page loads ("Alerts", `alerts.spec.ts:10`);
    rows render — `[data-testid="alert-channel-row"]` ≥2 from seed + renders
    "Webhook Alerts" + "webhook" (`:15-24`); create — `[data-testid="alert-channel-new"]`
    modal (traffic-lights) + `#alert-channel-name` + `[data-testid="alert-channel-save"]`
    fires `POST /api/alert-channels` (`:26-41`); per-channel test —
    `[data-testid="alert-channel-test"]` fires `POST /api/alert-channels/{id}/test`
    (regex `/\/api\/alert-channels\/[^/]+\/test$/`, `:43-52`); delete —
    `[data-testid="alert-channel-delete"]` + confirm dialog ("Delete channel")
    decrements the row count (`:54-65`).
  - **Component vitest (consume-only, must stay green):**
    `ui/src/components/governance/guardrails-tester.test.tsx` mocks the `/test` fetch
    to return `{blocked, redacted_prompt, matches}` and asserts the rendered HTML
    matches `/blocked/i` on `blocked:true` (`:65-81`). This nails the `/test` DTO
    field names: `{blocked:bool, redacted_prompt:string, matches:[]string}`.
- **w7-gov-1 audit-write seam is IN-TREE and reusable (the de-risk).** w7-gov-1 added
  `Handlers.audit *governance.AuditService` (`internal/admin/handlers.go:20`), the
  `auditService()` accessor (`:55`), and the best-effort write helper
  `func (h *Handlers) recordAudit(ctx, action, target, details string)`
  (`internal/admin/audit.go:64-72` — resolves the actor from
  `ctx.UserValue(userKey).(*store.User)`, logs + swallows write errors). **REUSE
  `h.recordAudit` on this plan's mutations** (guardrails config update; alert-channel
  create/update/delete; alert test-send) — best-effort, post-success, NEVER fails the
  parent (mirrors w7-gov-1 ESC-AUDIT-WRITE + w7-gov-2 ESC-AUDIT-REUSE). NO new audit
  code; NO edit to audit.go / governance/audit.go / handlers.go.
- **CRUD templates EXIST.** Store CRUD template with a numeric (int64 autoincrement)
  PK + JSON-blob column = `internal/store/prompttemplates.go` (w7-gov-2:
  `models_json` blob, `boolToInt`/`intToBool`, `scanPromptTemplate`, `ErrNotFound`).
  TEXT-PK CRUD template = `internal/store/teams.go` (w7-gov-1). Secret-at-rest CRUD
  template = `internal/store/connections.go` (`secret_enc`/`access_token_enc` via
  `s.cipher.Encrypt`/`Decrypt`, `connections.go:43,118,141`). Transport CRUD template
  = `internal/admin/prompttemplates.go` / `internal/admin/teams.go` (DTO + request
  structs + validate + `writeData`/`writeError` + `pathID`/`strconv.ParseInt`).
- **Singleton-config storage primitives EXIST (the guardrails de-risk).** Two
  options in-tree: `internal/store/settings.go` (`GetSetting`/`SetSetting`/
  `GetSettings`/`SetSettings` flat key→value over the `settings` table) and
  `internal/store/kv.go` (`GetKV`/`SetKV`/`ListKV` scoped key→value over the `kv`
  table, with a JSON-blob precedent — `UserPricing` stores a JSON map under
  `scope='pricing'`, `kv.go:80-99`). The guardrails config is a SINGLETON (one
  config object, not a list) — see §8 ESC-GR-STORE for the storage decision
  (RECOMMENDED: a dedicated single-row `guardrails` table mirroring the 4-field mock
  shape; the KV/settings JSON-blob is the documented fallback).
- **Envelope + handler patterns** (`internal/admin/respond.go`): `writeData(ctx,
  status, data)` / `writeError(ctx, status, message)` → `{data, error:{message}}`
  snake_case (`respond.go:19,23`). `pathID(ctx.UserValue("id"))` extracts `{id}`
  (`handlers.go:84`); numeric ids parse via `strconv.ParseInt` (the w7-gov-2
  ESC-IDTYPE precedent).
- **Migrations are additive-only** (`internal/store/migrate.go`): new tables via the
  `tables []struct{name,create}` slice with `CREATE TABLE IF NOT EXISTS` (the
  w7-gov-1/gov-2 `teams`/`audit_log`/`feature_flags`/`prompt_templates` additions end
  at `migrate.go:173`; the additive-index block at `:182-205`; `ensureColumn` loop at
  `:225`). The at-rest `*_enc` precedent column decls are at `migrate.go:51-53,62`.
  ADDITIVE ONLY (decision 2).
- **At-rest cipher EXISTS on the Store** (`internal/store/store.go:19` `cipher
  *Cipher`; `internal/store/crypto.go` `Cipher.Encrypt`/`Decrypt`). REUSE it for the
  alert-channel `config` secret-at-rest decision (§8 ESC-ALERT-SECRET) — do NOT add a
  new cipher.
- **Admin test harness** (`internal/admin/admin_test.go` `newTestEnv` + `call` +
  `dataField[T]` + `errMessage` + `loginToken`): real `store.Open(tempDB, secret)` +
  `auth.NewSessions` + `SeedAdmin` + `New(...)`. NO mocks. This is the authoritative
  proof surface (mirrors w7-gov-1/gov-2 §1.1).
- **Handlers injection** (`internal/admin/handlers.go`): the `Handlers` struct holds
  `store`/`sessions`/`audit`/… ; new domains compose `h.store` directly (like
  teams/prompttemplates) — NO new global state, NO `New(...)` signature change (MAP
  decision 9, `handlers.go:28`).

### 1.2 The mock contracts these flips must mirror (binding — decision 1)

**Decision 1 (MAP §36, §245):** real Go wins; the W6 mock body + seed are corrected
IN THIS PLAN to mirror the real Go `{data,error}` snake_case DTO. The page is FROZEN
(decision 8); where the real DTO and the mock disagree, **prefer matching the mock's
existing field names in the Go DTO** (modeled to match 9router); only ESCALATE if
impossible.

**Envelope reconciliation (both domains).** The mock `json(route, data)` util wraps
every payload as `{data}` (`ui/e2e/mocks/handlers/utils.ts`); the page reads lists as
a bare array under `data` (the teams/feature-flags precedent). So the Go list endpoint
returns `{data:[...]}`. The shared `error()` helper (`{error:<string>}`) diverges from
the Go `{error:{message}}` but is a tolerated, shared-infra divergence — **do NOT edit
`utils.ts`** (out of scope; same disposition as w7-gov-1/gov-2). Only correct the
per-route SUCCESS bodies/seeds.

**Guardrails** (`ui/e2e/mocks/handlers/guardrails.ts` + `seed/guardrails.ts`):
- Routes: `GET /api/guardrails` (config read, `guardrails.ts:8`), `PUT /api/guardrails`
  (config update — spreads `{...store.guardrails, ...body}`, `:9-13`), `POST
  /api/guardrails/test` (evaluator, `:16-27`). **There is NO `/{id}` and NO list** —
  guardrails is a SINGLETON config object (NOT a CRUD list); the Go MUST match.
- Mock/seed config shape = the UI `Guardrails` type (`ui/src/lib/types.ts:103-108`):
  **`{guardrails_enabled:bool, guardrails_blocklist:[]string, pii_redaction_enabled:bool,
  pii_redaction_types:[]string}`** — this is the canonical Go config DTO. The page
  consumes `guardrails_blocklist` (`guardrails.tsx:32,58`); save PUTs the 4-field
  object (`guardrails.tsx:58`).
- **`/test` evaluator contract (CRITICAL — the blocked-computation, §8 ESC-GR-EVAL).**
  The mock `/test` (`guardrails.ts:16-27`): body `{prompt}`; `matches =
  guardrails_enabled ? blocklist.filter(w => prompt.toLowerCase().includes(w.toLowerCase())) : []`;
  `blocked = matches.length > 0`; returns `{blocked, redacted_prompt: prompt, matches}`
  (`:18-24`). The Go `/test` MUST compute IDENTICALLY:
  `blocked = guardrails_enabled && (∃ blocklist word that is a case-insensitive
  substring of prompt)`; `matches` = the matching blocklist words (case-insensitive
  substring), in blocklist order; `redacted_prompt` = the prompt with PII redaction
  applied when `pii_redaction_enabled` (see below), else the prompt verbatim (the mock
  echoes the prompt verbatim — the Go default may also echo verbatim when PII is off;
  §8 ESC-GR-EVAL for the PII-on behavior).
- **Seed is the binding green-state** (`seed/guardrails.ts`, w6-k path B correction):
  `{guardrails_enabled:true, guardrails_blocklist:["password","secret","badword1"],
  pii_redaction_enabled:false, pii_redaction_types:["email","phone","ssn"]}`. With
  `"my secret password"` → matches `["password","secret"]` (or in blocklist order
  `["password","secret"]`), `blocked:true` → the spec's `/blocked/i` assertion passes.
  **KEEP this seed correction intact** (do NOT revert w6-k path B). The Go `/test`,
  given this config, must produce `blocked:true` for that prompt. **Reconciliation:**
  the mock + seed already produce the required green state; the only change to
  `guardrails.ts` is verification (likely none) — the Go must mirror the mock's
  evaluator logic so the spec stays green when the page eventually talks to real Go.

**Alert-channels** (`ui/e2e/mocks/handlers/alert-channels.ts` + `seed/alert-channels.ts`):
- Routes: `GET /api/alert-channels` (list — `Array.from(store.alertChannels.values())`,
  `alert-channels.ts:8`), `POST` (create — `{id:Date.now(), created_at:..., ...body}`,
  `:9-14`), `GET|PUT|DELETE /api/alert-channels/{id}` (`:17-37`), `POST
  /api/alert-channels/{id}/test` (returns `{ok:true, message:"Test notification sent"}`,
  `:38-41`).
- Mock/seed shape = the UI `AlertChannel` type (`ui/src/lib/types.ts:12-20`):
  **`{id:number, name, channel_type, config:Record<string,unknown>, events:[]string,
  is_active:bool, created_at:string}`** — the canonical Go DTO. **`id` is NUMERIC**
  (`Date.now()` in create; seed `id:1,2`; §8 ESC-IDTYPE — INTEGER autoincrement PK,
  the w7-gov-2 precedent).
- Seed (`seed/alert-channels.ts`): two channels — `{id:1, name:"Webhook Alerts",
  channel_type:"webhook", config:{url:"https://hooks.example.com/g0router"},
  events:["quota_exceeded","provider_error"], is_active:true, created_at:...}` and
  `{id:2, name:"Discord Alerts", channel_type:"discord",
  config:{webhook_url:"https://discord.com/api/webhooks/xxx"},
  events:["provider_error"], is_active:false, created_at:...}`. The `config` is a
  free-form JSON object (webhook URL / discord webhook URL — potential secrets;
  §8 ESC-ALERT-SECRET).
- Create body (`alert-channel-form-modal.tsx:55-58`): `{name, channel_type, config:
  (channel_type==="discord" ? {webhook_url:url} : {url}), events, is_active}`. Returns
  the created DTO.
- **`/{id}/test` contract** (`alert-channels.ts:38-41`): the mock returns
  `{ok:true, message:"Test notification sent"}`. The spec only asserts the POST fires
  (`alerts.spec.ts:43-52`) — it does NOT assert the response body. **Reconciliation:**
  the Go `/test` returns `{data:{ok:bool, message:string}}` mirroring the mock keys;
  the real test-notification scope is §8 ESC-ALERT-TEST (best-effort send to the
  channel's configured target, deterministic in tests via an injectable sender; NEVER
  echoes the secret config). The DELETE mock returns `{}` (page decrements on success,
  ignores body); the Go DELETE returns `{data:{message:"Alert channel deleted
  successfully"}}` (page-tolerated, the prompttemplates DELETE precedent).

### 1.3 Architecture (binding — layered DDD, decision 4 + the w7-gov-1/gov-2 ESC-ARCH finding)

w7-gov-1 RESOLVED ESC-ARCH and w7-gov-2 confirmed it: **no in-tree arch test enforces
transport→domain→repository** (teams/feature-flags/prompttemplates handlers call
`h.store` DIRECTLY with no enforcing test). BUT the brief mandates a domain seam for
BOTH this plan's domains (the evaluator + the dispatcher), and both warrant one on
merit (non-trivial logic that must be unit-tested in isolation):

```
guardrails:     admin/guardrails.go  → governance/guardrails.go (the EVALUATOR seam) → store/guardrails.go (singleton config)
                (the /test evaluator — blocklist substring + PII redaction — is a pure,
                 dependency-free function unit-tested in governance/guardrails_test.go)
alert-channels: admin/alerts.go      → governance/alertchannels.go (the DISPATCHER seam) → store/alertchannels.go (CRUD, config_enc)
                (the test-notification dispatcher — an injectable Sender interface —
                 unit-tested deterministically in governance/alertchannels_test.go)
```

- **Guardrails domain (`internal/governance/guardrails.go`) is WARRANTED** (brief:
  "the evaluator seam"): it holds `GuardrailEngine` with a constructor
  `NewGuardrailEngine(st)` and a PURE evaluator `Evaluate(cfg, prompt) (blocked bool,
  redacted string, matches []string)` that the transport calls. The evaluator is
  dependency-free (no store, no pipeline) so it is unit-tested directly with table
  cases proving the blocked-computation matches the mock (§1.4). This is the brief's
  REQUIRED "the guardrails `/test` evaluator MUST have unit tests proving the
  blocked-computation deterministically."
- **Standalone evaluator, NOT a pipeline hook (DECISION — §8 ESC-GR-PIPELINE).** The
  brief asks: does `/api/guardrails/test` run a real `internal/inference` pipeline
  guardrail pass, or a standalone blocklist/PII evaluator? **The mock is a standalone
  evaluator; mirror that.** w7-gov-3 ships the STANDALONE evaluator
  (`governance/guardrails.go` `Evaluate`) — NO `internal/inference` edit. A full
  request-pipeline guardrail integration (applying the config to live inference
  requests) is a LARGER, separate concern recorded as a follow-up in
  `open-questions.md` (the config storage + evaluator this plan ships are the
  prerequisite for that future wiring).
- **Alert-channels domain (`internal/governance/alertchannels.go`) is WARRANTED**
  (brief: "the dispatcher seam"): it holds `AlertDispatcher` with a constructor
  `NewAlertDispatcher(...)` taking an injectable `Sender` interface (the real impl
  does a best-effort HTTP POST to the channel's configured URL; tests inject a fake
  Sender that records the call WITHOUT network I/O — "deterministic in tests", brief).
  `Dispatch(ctx, channel) (ok bool, message string, err error)` is the seam the
  transport `/test` handler calls. This is the brief's REQUIRED "alert `/test` MUST
  have unit tests proving … notification dispatch deterministically."

### 1.4 Guardrails Go contract (NEW, TDD)

**Singleton config storage (DECISION — §8 ESC-GR-STORE).** RECOMMENDED: a dedicated
SINGLE-ROW `guardrails` table mirroring the 4-field mock shape (a fixed `id=1`
sentinel row; `GetGuardrails` upserts a default row on first read so the config always
exists). This mirrors the table precedent (teams/feature_flags), gives typed columns,
and avoids JSON-blob parsing in the hot read path. The KV/settings JSON-blob is the
documented fallback if the orchestrator prefers zero new tables (§8 ESC-GR-STORE).

Table `guardrails` (additive, `migrate.go` tables slice). **Single-row config** (a
fixed `id` PK; the store guarantees exactly one row):
```sql
CREATE TABLE IF NOT EXISTS guardrails (
  id INTEGER PRIMARY KEY,                          -- always 1 (singleton sentinel)
  guardrails_enabled INTEGER NOT NULL DEFAULT 0,   -- SQLite bool 0/1
  guardrails_blocklist_json TEXT NOT NULL DEFAULT '[]',   -- JSON []string
  pii_redaction_enabled INTEGER NOT NULL DEFAULT 0,
  pii_redaction_types_json TEXT NOT NULL DEFAULT '[]',    -- JSON []string
  updated_at INTEGER NOT NULL DEFAULT 0
)
```
(No secret fields — the blocklist words are policy config, NOT secrets; no `*_enc`.)

`internal/store/guardrails.go` (NEW): `Guardrails{Enabled bool, Blocklist []string,
PIIRedactionEnabled bool, PIIRedactionTypes []string}` + methods:
- `GetGuardrails() (*Guardrails, error)` — read the singleton row; if absent, INSERT a
  zero-value default row (enabled=false, empty lists) then return it (so the config
  always exists). JSON-decode the two `*_json` columns.
- `SetGuardrails(g *Guardrails) error` — UPSERT the singleton row (`INSERT … ON
  CONFLICT(id) DO UPDATE`, fixed `id=1`); JSON-encode the lists; bump `updated_at`.
- `boolToInt`/`intToBool` for the SQLite bools; `encoding/json` for the lists.

`internal/governance/guardrails.go` (NEW): `GuardrailEngine` with constructor
`NewGuardrailEngine(st *store.Store)` and:
- `Config() (*store.Guardrails, error)` — wraps `st.GetGuardrails`.
- `Save(g *store.Guardrails) error` — wraps `st.SetGuardrails`.
- **`Evaluate(cfg *store.Guardrails, prompt string) (blocked bool, redacted string,
  matches []string)`** — the PURE evaluator (no store, no I/O). Logic (MIRROR the mock
  `guardrails.ts:18-24` EXACTLY):
  - `matches`: if `cfg.Enabled`, the blocklist words `w` where
    `strings.Contains(strings.ToLower(prompt), strings.ToLower(w))`, in blocklist
    order; else empty.
  - `blocked`: `len(matches) > 0`.
  - `redacted`: when `cfg.PIIRedactionEnabled`, apply deterministic PII redaction over
    `cfg.PIIRedactionTypes` (email/phone/ssn → replace matches with `[REDACTED]`,
    dependency-free regex; §8 ESC-GR-EVAL); else the prompt VERBATIM (mirrors the mock,
    which echoes `redacted_prompt: prompt`).

`internal/admin/guardrails.go` (NEW):

| Handler | Route | Shape (snake_case, `{data}`) | Notes |
|---|---|---|---|
| `GetGuardrails` | `GET /api/guardrails` | `{data:configDTO}` | `configDTO{guardrails_enabled, guardrails_blocklist, pii_redaction_enabled, pii_redaction_types}` (the 4-field mock shape) |
| `UpdateGuardrails` | `PUT /api/guardrails` | body = the 4-field config (partial allowed — merge over current, mirroring the mock spread `{...store.guardrails, ...body}`); returns the full updated `{data:configDTO}` | `recordAudit(ctx,"guardrails.update","guardrails",<summary>)` best-effort; summary = e.g. `fmt.Sprintf("enabled=%v blocklist=%d", enabled, len(blocklist))` — NEVER the blocklist words verbatim if treated as sensitive (they are policy, not secrets, but keep the summary terse) |
| `TestGuardrails` | `POST /api/guardrails/test` | body `{prompt:string}`; loads the current config, calls `engine.Evaluate(cfg, prompt)`; returns `{data:{blocked, redacted_prompt, matches}}` | NO audit (read-only test). **MUST return `blocked:true` for `"my secret password"` under the seed config** (the spec contract, §1.1) |

**SINGLETON — NO list, NO POST/DELETE, NO `/{id}`.** Registering any CRUD route for
guardrails is a scope violation.

### 1.5 Alert-channels Go contract (NEW, TDD)

Table `alert_channels` (additive). **INTEGER autoincrement PK** (§8 ESC-IDTYPE) +
**`config_enc`** for the secret-bearing config blob (§8 ESC-ALERT-SECRET):
```sql
CREATE TABLE IF NOT EXISTS alert_channels (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  channel_type TEXT NOT NULL DEFAULT 'webhook',
  config_enc TEXT NOT NULL DEFAULT '',          -- encrypted JSON object (may hold webhook URLs/tokens)
  events_json TEXT NOT NULL DEFAULT '[]',       -- JSON []string (NOT secret)
  is_active INTEGER NOT NULL DEFAULT 1,          -- SQLite bool
  created_at TEXT NOT NULL                       -- ISO-8601 (RFC3339), mirrors mock seed
)
```

`internal/store/alertchannels.go` (NEW): `AlertChannel{ID int64, Name string,
ChannelType string, Config map[string]any, Events []string, IsActive bool, CreatedAt
string}` + methods (MIRROR `connections.go` for the `*_enc` encrypt-on-write /
decrypt-on-read pattern; `prompttemplates.go` for the JSON-blob + numeric-PK shape):
- `CreateAlertChannel(in *AlertChannel) (*AlertChannel, error)` — JSON-encode
  `Config`, `s.cipher.Encrypt` → `config_enc`; JSON-encode `Events` → `events_json`;
  RFC3339 `created_at`.
- `ListAlertChannels() ([]*AlertChannel, error)` — `ORDER BY id ASC`; decrypt + decode.
- `GetAlertChannelByID(id int64) (*AlertChannel, error)` — `ErrNotFound`.
- `UpdateAlertChannel(id int64, in *AlertChannel) (*AlertChannel, error)` —
  re-encrypt config; `ErrNotFound`.
- `DeleteAlertChannel(id int64) error` — `ErrNotFound` on 0 rows.
- `scanAlertChannel` helper (decrypt `config_enc` via `s.cipher.Decrypt`, JSON-decode);
  `boolToInt`/`intToBool`.

`internal/governance/alertchannels.go` (NEW): `AlertDispatcher` with constructor
`NewAlertDispatcher(sender Sender)` where:
```go
type Sender interface {
    Send(ctx context.Context, channelType string, config map[string]any) error
}
```
- The real `Sender` impl (`httpSender` in the same file, or a tiny default) does a
  best-effort POST to the channel's configured target (the `url`/`webhook_url` in
  `config`) with a small fixed timeout; a non-2xx or transport error → `err`.
- `Dispatch(ctx, ch *store.AlertChannel) (ok bool, message string)` — calls
  `sender.Send`; returns `(true, "Test notification sent")` on success, `(false,
  "Test notification failed: …<sanitized>")` on error. **NEVER includes the secret
  config (URL/token) in the message** (§5 no-secret-exposure proof).
- **Deterministic in tests** (brief): `alertchannels_test.go` injects a FAKE `Sender`
  that records the `(channelType, config)` it was asked to send and returns a
  configurable error — NO network I/O. Tests assert: a configured channel dispatches
  `ok:true` + the fake recorded the call; a Sender error → `ok:false` + the message
  carries NO secret.

`internal/admin/alerts.go` (NEW):

| Handler | Route | Body / response | Notes |
|---|---|---|---|
| `ListAlertChannels` | `GET /api/alert-channels` | `{data:[channelDTO]}` (bare array under data) | `channelDTO{id,name,channel_type,config,events,is_active,created_at}` — see §8 ESC-ALERT-CONFIG-ECHO for whether `config` is echoed in the LIST/GET DTO |
| `CreateAlertChannel` | `POST /api/alert-channels` | body `{name, channel_type?, config?, events?, is_active?}`; 400 on empty name; returns `{data:channelDTO}` | `recordAudit(ctx,"alert_channel.create",name,<summary, no secrets>)` |
| `GetAlertChannel` | `GET /api/alert-channels/{id}` | `{data:channelDTO}` or 404 | numeric id (`strconv.ParseInt`) |
| `UpdateAlertChannel` | `PUT /api/alert-channels/{id}` | body = create body; returns updated `{data:channelDTO}` or 404 | `recordAudit(ctx,"alert_channel.update",name,…)` |
| `DeleteAlertChannel` | `DELETE /api/alert-channels/{id}` | `{data:{message:"Alert channel deleted successfully"}}` or 404 | mock returns `{}`; page tolerates. `recordAudit(ctx,"alert_channel.delete",id,…)` |
| `TestAlertChannel` | `POST /api/alert-channels/{id}/test` | loads the channel, calls `dispatcher.Dispatch(ctx, ch)`; returns `{data:{ok:bool, message:string}}` (mirrors mock `{ok,message}`). 404 if channel missing | `recordAudit(ctx,"alert_channel.test",name,"test notification sent")` best-effort; **response NEVER echoes the secret config** |

**Route precedence note:** `/api/alert-channels/{id}/test` (the deeper static segment)
must resolve correctly vs `/api/alert-channels/{id}` — register the `…/{id}/test`
route; `fasthttp/router` distinguishes a 3-segment param-then-static path from the
2-segment param path. Register the static collection (`/api/alert-channels`) and the
`…/{id}` + `…/{id}/test` routes following the file's existing static-before-param
ordering (§1.6). A genuine mis-disambiguation is §8 ESC-ROUTE, not a silent path change.

### 1.6 routes_admin.go registration (serial-slot additive, §3)

Append AFTER the w7-gov-2 prompt-templates block (`routes_admin.go:73`),
static-before-`{id}`:
```go
// Guardrails (SINGLETON config — no list/{id}). Static /test BEFORE the bare PUT/GET.
r.GET("/api/guardrails", h.RequireSession(h.GetGuardrails))
r.PUT("/api/guardrails", h.RequireSession(h.UpdateGuardrails))
r.POST("/api/guardrails/test", h.RequireSession(h.TestGuardrails))
// Alert channels CRUD (+ per-channel test). Static collection before {id}; {id}/test deeper.
r.GET("/api/alert-channels", h.RequireSession(h.ListAlertChannels))
r.POST("/api/alert-channels", h.RequireSession(h.CreateAlertChannel))
r.POST("/api/alert-channels/{id}/test", h.RequireSession(h.TestAlertChannel))
r.GET("/api/alert-channels/{id}", h.RequireSession(h.GetAlertChannel))
r.PUT("/api/alert-channels/{id}", h.RequireSession(h.UpdateAlertChannel))
r.DELETE("/api/alert-channels/{id}", h.RequireSession(h.DeleteAlertChannel))
```
Diff bound §5: the route block is ONE commit, additive only.

### NOT in scope (explicit)

- **No UI page/component/route/store edits** — `/guardrails`, `/alerts`, the
  guardrails-tester, the alert-channel-form-modal, and all w6-k components are FROZEN
  consume-only (decision 8). The ONLY UI-tree touches are the mock-body + seed
  VERIFICATION/corrections (§1.2 / §3). **NEVER revert the w6-k path-B
  `seed/guardrails.ts` correction** (it keeps the tester green).
- **No `internal/inference` pipeline edit** — the guardrails `/test` is a STANDALONE
  evaluator, NOT a live request-pipeline hook (§8 ESC-GR-PIPELINE; pipeline
  integration is a tracked follow-up).
- **No list / POST / DELETE / `{id}` for guardrails** — it is a SINGLETON config
  (brief-mandated; mock has none).
- **No edits to other governance domains** — teams/audit/user-management (w7-gov-1,
  merged), feature-flags/prompts (w7-gov-2, merged) are disjoint; do not touch their
  Go/mocks/seeds.
- **No edits to pre-existing admin handlers' bodies** — teams.go, audit.go,
  usermgmt.go, featureflags.go, prompttemplates.go, apikeys.go, virtualkeys.go,
  providers*.go, connections.go, combos.go, disabledmodels.go, auth.go, version.go,
  usage/pricing handlers are FORBIDDEN.
- **No edit to `internal/admin/audit.go` / `internal/governance/audit.go` /
  `internal/admin/handlers.go`** — REUSE `h.recordAudit` only (read-only consumption
  of the w7-gov-1 seam; the `audit` field + accessor already exist).
- **No edit to the shared mock `utils.ts` / index / seed-index / `store.ts` /
  `fixture.ts`** (§1.2 envelope note).
- **No destructive DDL / column renames** — additive `ensureTable`/`ensureColumn` ONLY
  (decision 2).
- **No new global state / no `New(...)` signature change** (decision 9) — handlers
  compose `h.store` + the two new domain engines (constructed via a thin accessor over
  `h.store`, NO `New` signature change) + reuse `h.recordAudit`.
- **No secret exposure** — the alert-channel `config` (webhook URLs/tokens) is
  encrypted at rest via `config_enc`; the test-notification response NEVER echoes the
  secret config; audit `details` carry no secrets (§5 grep proofs).

---

## 2. Precondition checks

Run all before any edit; abort and report to orchestrator on any failure.

```bash
# P0 — clean tree; record <base>
git status --porcelain     # must be empty (explicit `git add <file>`, never -A;
                           # ui/dist/** gitignored — never stage it)
git rev-parse HEAD         # record as <base> for §5

# P1 — the two gaps are REAL (no Go for either domain)
grep -nE '/api/guardrails|/api/alert-channels' internal/server/routes_admin.go ; echo "^ expect EMPTY"
test ! -e internal/store/guardrails.go && test ! -e internal/store/alertchannels.go && echo "store gap OK"
test ! -e internal/governance/guardrails.go && test ! -e internal/governance/alertchannels.go && echo "governance gap OK"
test ! -e internal/admin/guardrails.go && test ! -e internal/admin/alerts.go && echo "admin gap OK"

# P2 — w7-gov-1/gov-2 seams present (the de-risk) + reused surfaces
grep -n "func (h \*Handlers) recordAudit" internal/admin/audit.go
grep -nE "audit .*governance|auditService" internal/admin/handlers.go
grep -n "func writeData\|func writeError" internal/admin/respond.go
grep -n "func pathID" internal/admin/handlers.go
grep -n "func newTestEnv\|func call\|func dataField\|func loginToken" internal/admin/admin_test.go
grep -n "func (s \*Store) CreatePromptTemplate\|GetPromptTemplateByID" internal/store/prompttemplates.go   # numeric-PK + JSON-blob template
grep -n "s.cipher.Encrypt\|s.cipher.Decrypt" internal/store/connections.go   # *_enc template
grep -n "func (s \*Store) GetKV\|SetKV\|GetSetting\|SetSetting" internal/store/kv.go internal/store/settings.go   # singleton-store fallbacks

# P3 — migrate pattern (additive tables slice + index + ensureColumn) + cipher on Store
grep -nE '"prompt_templates"|"feature_flags"|CREATE TABLE IF NOT EXISTS|func ensureColumn|CREATE INDEX IF NOT EXISTS' internal/store/migrate.go | head
grep -n "cipher .*\*Cipher" internal/store/store.go

# P4 — the W6-k UI + specs are present (consume-only) and the mocks to correct
test -f ui/e2e/guardrails.spec.ts && test -f ui/e2e/alerts.spec.ts && echo "specs present"
test -f ui/e2e/mocks/handlers/guardrails.ts && test -f ui/e2e/mocks/handlers/alert-channels.ts && echo "mocks present"
test -f ui/e2e/mocks/seed/guardrails.ts && test -f ui/e2e/mocks/seed/alert-channels.ts && echo "seeds present"
grep -n "guardrails_enabled.*true\|blocklist" ui/e2e/mocks/seed/guardrails.ts ; echo "^ w6-k path-B green seed — DO NOT revert"
grep -n "blocked\|redacted_prompt\|matches" ui/e2e/mocks/handlers/guardrails.ts ; echo "^ the evaluator contract to mirror (§1.4)"
grep -n "id: Date.now()\|id: 1\|id: 2" ui/e2e/mocks/handlers/alert-channels.ts ui/e2e/mocks/seed/alert-channels.ts ; echo "^ NUMERIC ids (ESC-IDTYPE)"
grep -n "url\|webhook_url" ui/e2e/mocks/seed/alert-channels.ts ; echo "^ config may carry secrets (ESC-ALERT-SECRET)"

# P5 — routes_admin.go serial slot is FREE (w7-gov-2 merged + released)
git log --oneline -5 -- internal/server/routes_admin.go   # last touch = w7-gov-2 (merged)
# Orchestrator MUST confirm no concurrent W7 plan holds an unmerged routes_admin.go
# edit before w7-gov-3 begins T-routes (chain: …→w7-gov-2→**w7-gov-3**→w7-mcp-3).
# w7-gov-3 TAKES the slot, then RELEASES it to the next chain holder (w7-mcp-3) — it is
# the LAST gov holder.

# P6 — green at base
go test ./... && go vet ./... && go build ./...     # exit 0 (Go untouched-green)
# e2e ISOLATED — kill stale chromium/vite-preview first (e2e-hygiene rule); never
# run concurrently with another playwright invocation; NEVER revert ui/dist/index.html.
pkill -f 'chromium|vite preview' 2>/dev/null ; true
cd ui && npx playwright test e2e/guardrails.spec.ts e2e/alerts.spec.ts
# Record base: these PASS at base against the W6 mocks. They must STAY green after
# the mock-body verifications. Record exact pass/fail in WORKFLOW.md.
cd ui && npx vitest run src/components/governance/guardrails-tester.test.tsx   # consume-only; green
cd ui && npm run build                               # exit 0
```

---

## 3. Exclusive file ownership

After w7-gov-3 merges, all CREATE files are owned by w7-gov-3; later plans consume,
never edit (MAP decision 7).

**CREATE — store (NEW):**

| File | Contract |
|---|---|
| `internal/store/guardrails.go` | `Guardrails` struct + `GetGuardrails` (default-on-first-read) + `SetGuardrails` (singleton upsert, fixed `id=1`); JSON-encode the two lists; `boolToInt`/`intToBool`. NO `*_enc` (blocklist is policy, not secret). |
| `internal/store/guardrails_test.go` | Table-driven via temp `store.Open`: first `GetGuardrails` returns a default (disabled, empty); `SetGuardrails` then `GetGuardrails` round-trips enabled + blocklist + PII lists; second `Set` overwrites (still one row). RED first. |
| `internal/store/alertchannels.go` | `AlertChannel` struct (INT64 id, `map[string]any` config, `[]string` events) + `CreateAlertChannel`/`ListAlertChannels`/`GetAlertChannelByID`/`UpdateAlertChannel`/`DeleteAlertChannel` + `scanAlertChannel`; `config_enc` via `s.cipher.Encrypt`/`Decrypt`; `events_json` JSON blob; `boolToInt`. `ErrNotFound`. |
| `internal/store/alertchannels_test.go` | create→get→list→update→delete→404; config round-trips through encrypt/decrypt (assert the decrypted map equals the input); **assert the raw `config_enc` column is NOT plaintext** (read the column directly, assert it != the JSON). RED first. |

**EXTEND — store (additive table registration only):**

| File | Change (additive ONLY) |
|---|---|
| `internal/store/migrate.go` | ADD the `guardrails` + `alert_channels` tables to the `tables` slice (after the w7-gov-2 `prompt_templates` entry, `migrate.go:173`). ADDITIVE ONLY — no DROP/RENAME. |
| `internal/store/migrate_test.go` (if present — EXTEND additively; else rely on store tests) | assert the two new tables exist post-migrate. |

**CREATE — domain (NEW):**

| File | Contract |
|---|---|
| `internal/governance/guardrails.go` | `GuardrailEngine` + `NewGuardrailEngine(st)` + `Config`/`Save` (wrap store) + the PURE `Evaluate(cfg, prompt) (blocked, redacted, matches)` mirroring the mock logic (§1.4). No `init()`; errors-as-values; no global state. |
| `internal/governance/guardrails_test.go` | **The blocked-computation proof (brief-mandated):** table cases — enabled + blocklist `["password","secret"]` + prompt `"my secret password"` → `blocked:true`, `matches:["password","secret"]` (blocklist order); disabled → `blocked:false`, `matches:[]`; no match → `blocked:false`; case-insensitivity (`"SECRET"` matches `"secret"`); PII-on redacts an email/phone/ssn in `redacted`; PII-off echoes verbatim. RED first. |
| `internal/governance/alertchannels.go` | `AlertDispatcher` + `Sender` interface + a real `httpSender` (best-effort POST, small timeout) + `NewAlertDispatcher(sender)` + `Dispatch(ctx, ch) (ok, message)`. Message NEVER carries the secret config. No `init()`. |
| `internal/governance/alertchannels_test.go` | **The dispatch proof (brief-mandated):** inject a FAKE `Sender` (records call, returns configurable err) — NO network I/O; assert a configured channel → `ok:true` + the fake recorded `(channel_type, config)`; a Sender error → `ok:false` + the message contains NEITHER the URL NOR any token from config. RED first. |

**CREATE — transport (NEW):**

| File | Contract |
|---|---|
| `internal/admin/guardrails.go` | `GetGuardrails`/`UpdateGuardrails`/`TestGuardrails` + `guardrailsConfigDTO` (4 fields) + request/validate; `writeData`/`writeError`; constructs `governance.NewGuardrailEngine(h.store)` via a thin accessor (no `New` sig change); `h.recordAudit` on update (best-effort). |
| `internal/admin/guardrails_test.go` | via `newTestEnv`: GET returns default config; PUT updates + GET reflects it; **`/test` with `{prompt:"my secret password"}` against an enabled+blocklisted config returns `blocked:true` + `matches` + `redacted_prompt`** (the spec-binding case); `/test` with disabled config → `blocked:false`; **assert an audit entry on update** (`GetAudit`). RED first. |
| `internal/admin/alerts.go` | `ListAlertChannels`/`CreateAlertChannel`/`GetAlertChannel`/`UpdateAlertChannel`/`DeleteAlertChannel`/`TestAlertChannel` + `alertChannelDTO` + request/validate; numeric `{id}` parse; constructs `governance.NewAlertDispatcher(...)` via a thin accessor; `h.recordAudit` on create/update/delete/test. |
| `internal/admin/alerts_test.go` | via `newTestEnv`: create→list(≥1)→get→update→delete→404; create empty-name→400; config round-trips; `/{id}/test` returns `{ok,message}` using an injected fake Sender (deterministic; no network); **the test-notification response contains NO secret (URL/token) from config**; **assert an audit entry on create** (`GetAudit`). RED first. |

**MODIFY — serial-slot route registration (additive only):**

| File | Change |
|---|---|
| `internal/server/routes_admin.go` | ADD the 9 route lines (§1.6). NOTHING else changes. ONE commit. SERIAL SLOT — only holder while live; RELEASE to the next chain holder (w7-mcp-3) on close (w7-gov-3 is the LAST gov holder). |

**MODIFY — e2e mock corrections (mirror real Go, decision 1):**

| File | Change |
|---|---|
| `ui/e2e/mocks/handlers/guardrails.ts` (BODY) | Verify GET/PUT return the 4-field config under `{data}` (they already do) and `/test` returns `{blocked, redacted_prompt, matches}` with the mock's substring logic. Likely NO change beyond confirming the field names match the Go DTO. DO NOT change the evaluator logic (it is the binding contract the Go mirrors). |
| `ui/e2e/mocks/handlers/alert-channels.ts` (BODY) | Verify GET/POST/PUT return the 7-field `AlertChannel` shape under `{data}`; `/{id}/test` returns `{ok,message}` (matches Go). DELETE returns `{}` (page tolerates; Go returns `{message}`). Likely NO change beyond confirming the field set. |
| `ui/e2e/mocks/seed/guardrails.ts` (BODY) | **DO NOT revert the w6-k path-B correction** (`guardrails_enabled:true` + blocklist incl `password`/`secret` — keeps the tester green). Verify the 4-field shape matches the Go DTO; correct ONLY if a field name diverges (none expected). |
| `ui/e2e/mocks/seed/alert-channels.ts` (BODY) | Already `{id:number,name,channel_type,config,events,is_active,created_at}` — verify; correct only on divergence (none expected). |

**FORBIDDEN:** everything else. Explicitly: all pre-existing `internal/admin/*.go`
(including teams.go, audit.go, usermgmt.go, featureflags.go, prompttemplates.go,
handlers.go — NO edit; reuse `h.recordAudit` read-only); all other `internal/store/*.go`
except guardrails/alertchannels (NEW) + migrate (additive table registration only);
`internal/governance/audit.go` (NO edit); `internal/inference/*` (NO pipeline hook —
ESC-GR-PIPELINE); all UI `ui/src/**` (FROZEN, decision 8); the shared mock
`utils.ts`/index/seed-index/`store.ts`/`fixture.ts`; all other mocks/seeds/specs;
`ui/package.json` + lockfile; `ui/vite.config.ts`; `ui/playwright.config.ts`; and
**`ui/dist/index.html` MUST NOT be reverted** (e2e-hygiene rule). Touching any of these
is an automatic REJECT.

---

## 4. TDD tasks

Cadence (strict, AGENTS.md "TDD always"): **no Go impl file may exist before its
`_test.go` is committed RED.** `go test ./... && go vet ./... && go build ./...` green
at EVERY commit (RED test commits fail only the new package's targeted run; prefer
table tests that fail on assertion, not compile). The two e2e specs + the guardrails
component vitest stay green throughout (real Go is additive; mock corrections mirror
it). The two domains are independent — order is guardrails → alert-channels, then the
single serial-slot routes commit, then mock verifications + closeout.

### T-guardrails — STEP(a) RED store+domain+admin tests, STEP(b) impl
STEP(a): write `internal/store/guardrails_test.go` + `internal/governance/guardrails_test.go`
(the blocked-computation table cases — brief-mandated) + `internal/admin/guardrails_test.go`
(`newTestEnv`, incl the `"my secret password"`→`blocked:true` case); add the
`guardrails` table to `migrate.go` (so tests compile + the table exists).
`go test ./internal/store/ -run Guardrail`, `go test ./internal/governance/ -run Guardrail`,
`go test ./internal/admin/ -run Guardrail` → FAIL. Commit RED:
`phase-1/w7-gov-3: failing guardrails store+domain+admin tests (TDD red)`.
STEP(b): implement `internal/store/guardrails.go` + `internal/governance/guardrails.go`
(the pure `Evaluate`) + `internal/admin/guardrails.go` (reuse `h.recordAudit` on
update). Gates: `go test ./... && go vet ./... && go build ./...` green. Commit:
`phase-1/w7-gov-3: guardrails singleton config + standalone evaluator + admin`.

### T-alerts — STEP(a) RED, STEP(b) impl
STEP(a): write `internal/store/alertchannels_test.go` (incl the config_enc-not-plaintext
assertion) + `internal/governance/alertchannels_test.go` (the fake-Sender dispatch
proof + no-secret-in-message — brief-mandated) + `internal/admin/alerts_test.go`; add
the `alert_channels` table to `migrate.go`. Run targeted tests → FAIL. Commit RED:
`phase-1/w7-gov-3: failing alert-channels store+domain+admin tests (TDD red)`.
STEP(b): implement `internal/store/alertchannels.go` (config_enc encrypt/decrypt) +
`internal/governance/alertchannels.go` (`Sender` + `Dispatch`) +
`internal/admin/alerts.go` (CRUD + `/{id}/test`; reuse `h.recordAudit`). Gates green.
Commit: `phase-1/w7-gov-3: alert-channels store (config_enc) + dispatcher + admin CRUD + test`.

### T-routes — serial-slot route registration
TAKE the serial slot (orchestrator confirms FREE at P5). Add the 9 route lines to
`routes_admin.go` (§1.6). Gates: `go test ./... && go vet ./... && go build ./...`
green. Commit (ONE commit touches the serial file):
`phase-1/w7-gov-3: register guardrails + alert-channels admin routes (serial slot)`.

### T-mocks — mock-body verifications (mirror real Go, decision 1)
Verify `guardrails.ts` (4-field config + `/test` evaluator shape), `alert-channels.ts`
(7-field DTO + `/{id}/test` `{ok,message}`), and both seeds (no change expected;
**NEVER revert the w6-k path-B guardrails seed**). Gates: `cd ui && npm run build`
green; isolated `npx playwright test e2e/guardrails.spec.ts e2e/alerts.spec.ts` green
(still); `npx vitest run src/components/governance/guardrails-tester.test.tsx` green.
If a correction reds a non-w7-gov-3 spec, STOP + ESCALATE (§8 ESC-MOCK). Commit (skip
if no body change was needed — record "verified, no change" in WORKFLOW.md instead):
`phase-1/w7-gov-3: verify guardrails/alert-channels mocks mirror real Go DTOs`.

### T-close — full gates + closeout
```bash
go test ./... && go vet ./... && go build ./...
go test ./internal/admin/ -run 'Guardrail|Alert' -v
go test ./internal/governance/ -run 'Guardrail|Alert' -v
go test ./internal/store/ -run 'Guardrail|Alert' -v
cd ui && npm run build
pkill -f 'chromium|vite preview' 2>/dev/null ; true
cd ui && npx playwright test e2e/guardrails.spec.ts e2e/alerts.spec.ts        # green
pkill -f 'chromium|vite preview' 2>/dev/null ; true
cd ui && npx playwright test                                                  # full suite green (no regressions)
cd ui && npx vitest run src/                                                  # unaffected, green
```
Flip `.planning/parity/matrix/9router-ui.md`: the guardrails + alerts governance rows
→ variant-HAVE→true-HAVE (real Go, cite §1.4/§1.5). Mark `open-questions.md` w6-k
ESC-1d/1f RESOLVED with a cite to this plan; append any new open items (§8 — the
guardrails request-pipeline-integration follow-up). Update `docs/WORKFLOW.md` (P6 base
observation, the ESC-GR-STORE singleton-table decision, the ESC-GR-PIPELINE
standalone-evaluator decision, the ESC-ALERT-SECRET config_enc decision, the
ESC-ALERT-TEST dispatcher-with-injectable-Sender decision, the serial-slot
take-from-w7-gov-2 / release-to-w7-mcp-3 as the LAST gov holder, the mock
verifications). Final commit:
`phase-1/w7-gov-3: close — guardrails/alert-channels Go; matrix flip; mock mirror; gov cluster complete`.
**On the close commit, RELEASE the routes_admin.go serial slot to the next chain
holder (w7-mcp-3).**

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0. Diff gate is
**w7-gov-3 commit-range-scoped** (§7).

**Test gates**
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/admin/ -run 'Guardrail|Alert' -v` → exit 0, all pass
  (guardrails: GET-default + PUT-update + `/test` blocked:true (the spec case) + `/test`
  disabled:false + audit-on-update ≥5 cases; alerts: CRUD ≥6 cases incl audit-on-create
  + `/{id}/test` ok + no-secret-in-response).
- `go test ./internal/governance/ -run 'Guardrail|Alert' -v` → exit 0
  (guardrails Evaluate blocked-computation table incl `"my secret password"`→true,
  case-insensitive, disabled→false, PII redact/echo; alerts Dispatch via fake Sender
  ok + error-no-secret).
- `go test ./internal/store/ -run 'Guardrail|Alert' -v` → exit 0
  (guardrails singleton round-trip; alert config_enc encrypt/decrypt + not-plaintext).
- `cd ui && npx playwright test e2e/guardrails.spec.ts e2e/alerts.spec.ts` → exit 0,
  all pass (3 guardrails + 5 alerts), 0 skipped. (Run ISOLATED; kill stale
  chromium/vite-preview first; NEVER revert ui/dist/index.html.)
- `cd ui && npx vitest run src/components/governance/guardrails-tester.test.tsx` → exit 0.
- `cd ui && npx playwright test` → exit 0, no spec green-at-base goes red.
- `cd ui && npm run build` → exit 0. `cd ui && npx vitest run src/` → exit 0.

**TDD-order proof** — each impl file's covering test appears in an earlier-or-equal
commit:
```bash
for pair in \
  "internal/store/guardrails_test.go:internal/store/guardrails.go" \
  "internal/store/alertchannels_test.go:internal/store/alertchannels.go" \
  "internal/governance/guardrails_test.go:internal/governance/guardrails.go" \
  "internal/governance/alertchannels_test.go:internal/governance/alertchannels.go" \
  "internal/admin/guardrails_test.go:internal/admin/guardrails.go" \
  "internal/admin/alerts_test.go:internal/admin/alerts.go" ; do
  tf=${pair%%:*}; im=${pair##*:}
  ct=$(git log --format=%ct --diff-filter=A -1 -- "$tf")
  cf=$(git log --format=%ct --diff-filter=A -1 -- "$im")
  [ "$ct" -le "$cf" ] || echo "TDD VIOLATION: $im"     # prints nothing
done
```

**Grep proofs (per domain)**
```bash
# guardrails
grep -n "func (h \*Handlers) GetGuardrails\|UpdateGuardrails\|TestGuardrails" internal/admin/guardrails.go
grep -n "guardrails_enabled\|guardrails_blocklist\|pii_redaction_enabled\|pii_redaction_types" internal/admin/guardrails.go   # 4-field config DTO
grep -n "blocked\|redacted_prompt\|matches" internal/admin/guardrails.go                  # /test response shape
grep -n "func (e \*GuardrailEngine) Evaluate\|func.*Evaluate" internal/governance/guardrails.go
grep -nE "strings.Contains|strings.ToLower" internal/governance/guardrails.go             # case-insensitive substring blocked-computation
grep -n "func (s \*Store) GetGuardrails\|SetGuardrails" internal/store/guardrails.go
grep -n "writeData\|writeError" internal/admin/guardrails.go                              # {data,error}
grep -n "h.recordAudit" internal/admin/guardrails.go                                      # audit on update
! grep -nE "func \(h \*Handlers\) (List|Create|Delete)Guardrail" internal/admin/guardrails.go && echo "no list/POST/DELETE OK (singleton)"
# alert-channels
grep -n "func (h \*Handlers) ListAlertChannels\|CreateAlertChannel\|GetAlertChannel\|UpdateAlertChannel\|DeleteAlertChannel\|TestAlertChannel" internal/admin/alerts.go
grep -n "name\|channel_type\|config\|events\|is_active\|created_at" internal/admin/alerts.go
grep -n "ok\|message" internal/admin/alerts.go                                            # /test response shape
grep -n "type Sender interface\|func.*Dispatch" internal/governance/alertchannels.go
grep -n "func (s \*Store) CreateAlertChannel\|ListAlertChannels\|GetAlertChannelByID\|UpdateAlertChannel\|DeleteAlertChannel" internal/store/alertchannels.go
grep -nE "s.cipher.Encrypt|s.cipher.Decrypt|config_enc" internal/store/alertchannels.go   # secret-at-rest
grep -n "h.recordAudit" internal/admin/alerts.go                                          # audit on mutations
# numeric id handling
grep -nE "ParseInt|strconv" internal/admin/alerts.go ; echo "^ numeric id parse (ESC-IDTYPE)"
# no init(); no new global state
! grep -rn "func init(" internal/admin/guardrails.go internal/admin/alerts.go internal/governance/guardrails.go internal/governance/alertchannels.go internal/store/guardrails.go internal/store/alertchannels.go && echo "no init() OK"
```

**No-secret-exposure proofs (binding)**
```bash
# alert config secret-at-rest: stored encrypted via config_enc (NOT a plaintext config column)
grep -n "config_enc" internal/store/migrate.go internal/store/alertchannels.go ; echo "^ config stored encrypted"
! grep -nE 'config_json' internal/store/alertchannels.go && echo "no plaintext config column OK"
# the test-notification dispatcher message NEVER carries the secret config
grep -nA6 "func.*Dispatch" internal/governance/alertchannels.go ; echo "^ message must NOT interpolate config url/token"
# guardrails carries NO secret fields → no *_enc
grep -n "_enc" internal/store/guardrails.go ; echo "^ expect EMPTY (blocklist is policy, not secret)"
# additive migrations only (no DROP/RENAME introduced by this plan)
git diff <base>..HEAD -- internal/store/migrate.go | grep -E '^\+' | grep -iE 'DROP COLUMN|RENAME COLUMN|DROP TABLE' | wc -l   # = 0
# audit details carry no secrets (human-readable summaries only)
grep -n "h.recordAudit" internal/admin/guardrails.go internal/admin/alerts.go ; echo "^ details = name/flag summaries, never raw config/blocklist payloads"
```
Plus a runtime no-leak assertion in `alerts_test.go`: marshal the `/{id}/test` response
(and the error path) and assert it contains NEITHER the channel's `url`/`webhook_url`
value NOR any token substring from `config`.

**Blocked-computation proof (binding — the brief's CRITICAL requirement)**
```bash
# the governance evaluator test proves blocked:true for "my secret password" under the seed config
grep -n "my secret password\|blocked.*true\|matches" internal/governance/guardrails_test.go
# the admin /test handler test proves the same end-to-end through newTestEnv
grep -n "my secret password\|/api/guardrails/test\|blocked" internal/admin/guardrails_test.go
```

**Negative / freeze proofs (w7-gov-3 commit-range — §7)**
```bash
R="<first-w7-gov-3>^..<last-w7-gov-3>"
# Only the sanctioned Go files changed:
git diff $R --name-only -- internal/ | grep -vE \
 'internal/store/(guardrails|alertchannels|migrate)(_test)?\.go|internal/governance/(guardrails|alertchannels)(_test)?\.go|internal/admin/(guardrails|alerts)(_test)?\.go|internal/server/routes_admin\.go' \
 | wc -l                                                                  # = 0
# Frozen w7-gov-1/gov-2 + pre-existing admin handlers untouched:
git diff $R --name-only -- internal/admin/teams.go internal/admin/audit.go internal/admin/usermgmt.go internal/admin/featureflags.go internal/admin/prompttemplates.go internal/admin/handlers.go internal/admin/auth.go internal/governance/audit.go | wc -l   # = 0
# inference pipeline untouched (standalone evaluator, ESC-GR-PIPELINE):
git diff $R --name-only -- internal/inference/ | wc -l                   # = 0
# Other store files untouched (except migrate additive):
git diff $R --name-only -- internal/store/ | grep -vE 'internal/store/(guardrails|alertchannels|migrate)(_test)?\.go' | wc -l   # = 0
# UI is frozen except the sanctioned mock/seed bodies:
git diff $R --name-only -- ui/ | grep -vE \
 'ui/e2e/mocks/handlers/(guardrails|alert-channels)\.ts|ui/e2e/mocks/seed/(guardrails|alert-channels)\.ts' | wc -l   # = 0
git diff $R --name-only -- ui/src/ | wc -l                               # = 0 (src frozen)
git diff $R --name-only -- ui/e2e/mocks/handlers/utils.ts ui/e2e/mocks/store.ts ui/e2e/mocks/fixture.ts | wc -l   # = 0 (shared infra frozen)
git diff $R --name-only -- ui/dist/index.html | wc -l                    # = 0 (never reverted — e2e-hygiene rule)
# routes_admin.go = exactly ONE commit, additive:
git log --oneline $R -- internal/server/routes_admin.go | wc -l          # = 1
git diff $R -- internal/server/routes_admin.go | grep -E '^-' | grep -v '^---' | wc -l   # = 0 (no deletions)
```

---

## 6. Out of scope (restated, binding)

No UI src edits (decision 8 — pages/components/routes/stores frozen); only the
sanctioned guardrails/alert-channels mock-body + seed VERIFICATIONS (and NOT the shared
`utils.ts`/`store.ts`/`fixture.ts`); NEVER revert the w6-k path-B `seed/guardrails.ts`
correction or `ui/dist/index.html`. No `internal/inference` pipeline hook (standalone
evaluator — ESC-GR-PIPELINE; pipeline integration is a tracked follow-up). No
list/POST/DELETE/`{id}` for guardrails (singleton config). No edits to w7-gov-1/gov-2
files (teams/audit/usermgmt/featureflags/prompttemplates/handlers/governance-audit) —
`h.recordAudit` is consumed read-only. No edits to pre-existing admin handlers. No JWT.
No destructive DDL — additive `ensureTable`/`ensureColumn` only. No `New(...)` signature
change / no new global state. No other governance domains (gov-1/gov-2 merged). No
secret exposure (alert config encrypted at rest via `config_enc`; test-notification
response never echoes secrets; audit details carry no secrets). Mock-vs-Go
contradiction → escalate (§8), never fudge a mock or edit a frozen handler.

## 7. Diff-gate scope

W7 governance plans (gov-1/gov-2 merged; sibling W7 plans concurrent) commit to main,
so a broad `<base>..HEAD` range sweeps in sibling commits. The diff gate MUST be scoped
to w7-gov-3's own commits. Isolate them:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w7-gov-3:" | awk '{print $1}'`
then `git diff <first-w7-gov-3>^..<last-w7-gov-3> -- [file list]`.

`git diff <first>^..<last> --name-only` must be exactly a subset of:
```
internal/store/guardrails.go
internal/store/guardrails_test.go
internal/store/alertchannels.go
internal/store/alertchannels_test.go
internal/store/migrate.go               (additive table registration; additive ONLY)
internal/store/migrate_test.go          (CONDITIONAL — only if it already exists)
internal/governance/guardrails.go
internal/governance/guardrails_test.go
internal/governance/alertchannels.go
internal/governance/alertchannels_test.go
internal/admin/guardrails.go
internal/admin/guardrails_test.go
internal/admin/alerts.go
internal/admin/alerts_test.go
internal/server/routes_admin.go         (serial-slot additive routes; ONE commit)
ui/e2e/mocks/handlers/guardrails.ts     (body only — verify; minimal/no change)
ui/e2e/mocks/handlers/alert-channels.ts (body only — verify; minimal/no change)
ui/e2e/mocks/seed/guardrails.ts         (verify; NEVER revert path-B; correct only on divergence)
ui/e2e/mocks/seed/alert-channels.ts     (verify; correct only on divergence)
.planning/parity/matrix/9router-ui.md
.planning/parity/plans/open-questions.md
docs/WORKFLOW.md
```
Any file outside this list in the scoped diff is an automatic review REJECT.
`internal/admin/{teams,audit,usermgmt,featureflags,prompttemplates,handlers,auth}.go`,
`internal/governance/audit.go`, `internal/inference/**`, all other pre-existing
admin/store handlers, all `ui/src/**`, the shared mock infra, and `ui/dist/index.html`
are deliberately ABSENT — touching them is an automatic REJECT. The `routes_admin.go`
edit must appear in exactly ONE commit (§5) and the serial slot is released to the next
chain holder (w7-mcp-3) on close — w7-gov-3 is the LAST gov holder.

## 8. Escalations / decisions (explicit — recommended defaults, do not fabricate)

- **ESC-GR-STORE (RESOLVED at authoring — guardrails singleton storage, binding
  default).** The brief asks: a single-row table or a settings/KV blob for the
  guardrails SINGLETON config? Two in-tree fallbacks exist: `settings.go` (flat
  key→value) and `kv.go` (scoped key→value with a JSON-blob precedent, `UserPricing`
  `kv.go:80-99`). **Decision: a dedicated single-row `guardrails` table** (fixed `id=1`
  sentinel; `GetGuardrails` default-on-first-read; `SetGuardrails` upsert). Rationale:
  it mirrors the 4-field mock shape with typed columns (the teams/feature_flags table
  precedent), avoids JSON-blob parse on every read, and keeps the config queryable.
  **Documented fallback:** if the orchestrator prefers ZERO new tables, store the
  config as one JSON blob under `kv` scope `'guardrails'` key `'config'` (reusing
  `SetKV`/`GetKV`) — functionally equivalent, one fewer table. RECOMMENDED: the
  dedicated table; flag for confirmation; the plan proceeds on this default.
- **ESC-GR-PIPELINE (RESOLVED at authoring — evaluator scope, binding default).** The
  brief asks: does `/api/guardrails/test` run a real `internal/inference` request-
  pipeline guardrail pass, or a STANDALONE blocklist/PII evaluator? **The mock is a
  standalone evaluator (`guardrails.ts:16-27`); mirror that.** **Decision:** w7-gov-3
  ships the STANDALONE evaluator (`governance/guardrails.go` `Evaluate` — pure,
  dependency-free) over the stored config; NO `internal/inference` edit. A full
  request-pipeline guardrail integration (applying the stored config to live inference
  requests — blocking/redacting real traffic) is a LARGER concern, recorded as a
  follow-up in `open-questions.md`; the config storage + evaluator this plan ships are
  the prerequisite for it. RECOMMENDED as stated; flag for confirmation.
- **ESC-GR-EVAL (RESOLVED at authoring — the blocked-computation + PII semantics,
  binding default).** **The blocked-computation MUST mirror the mock EXACTLY**
  (`guardrails.ts:18-24`): `blocked = guardrails_enabled && (∃ blocklist word that is a
  case-insensitive substring of the prompt)`; `matches` = the matching words in
  blocklist order; this is non-negotiable (the `guardrails.spec.ts` tester + the
  `guardrails-tester.test.tsx` vitest both ride on `"my secret password"`→`blocked:true`
  under the seed). **PII redaction semantics** (the mock echoes `redacted_prompt:
  prompt` verbatim and never exercises PII): **Decision:** when
  `pii_redaction_enabled` is false, `redacted_prompt` = the prompt VERBATIM (mirrors
  the mock — keeps both specs green); when true, apply a deterministic, dependency-free
  regex redaction over `pii_redaction_types` (email/phone/ssn → `[REDACTED]`). The
  PII-on path is NOT exercised by any current spec, so it is a forward-compatible
  addition that does not risk a green spec; flag for confirmation. RECOMMENDED as
  stated.
- **ESC-IDTYPE (RESOLVED at authoring — alert-channels PK, binding default).** The
  alert-channels mock + UI type use NUMERIC ids (`AlertChannel.id:number`
  `types.ts:13`; create uses `Date.now()`; seed `id:1,2`). **Decision: `INTEGER
  PRIMARY KEY AUTOINCREMENT`** for `alert_channels` (id int64 in Go; handlers parse
  `{id}` via `strconv.ParseInt`, the w7-gov-2 precedent). Guardrails has NO id (it is a
  singleton; the table's `id` is a fixed internal sentinel, never surfaced). RECOMMENDED.
- **ESC-ALERT-SECRET (RESOLVED at authoring — alert config secret-at-rest, binding
  default).** The alert-channel `config` may hold webhook URLs / tokens (the seed has
  `{url}` / `{webhook_url}`; a Slack/Discord webhook URL IS a credential). The brief
  mandates "mask/encrypt at rest via `*_enc` if it carries secrets, and never echo
  secrets in the test-notification response." **Decision: store the entire `config`
  JSON blob ENCRYPTED at rest in a `config_enc` column** via `s.cipher.Encrypt`/
  `Decrypt` (the `connections.go` `secret_enc` precedent) — treat the whole config as
  potentially secret-bearing (simplest + safest; no per-field key allow/deny-list). The
  GET/LIST/GET-by-id DTO DOES surface `config` (the form modal needs to re-display the
  URL on edit — `alert-channel-form-modal.tsx:39-40`), but the **test-notification
  response NEVER echoes config** (§5). See ESC-ALERT-CONFIG-ECHO for the DTO nuance.
  RECOMMENDED as stated; flag for confirmation.
- **ESC-ALERT-CONFIG-ECHO (RESOLVED at authoring — config in the read DTO, binding
  default).** The form modal re-displays the saved URL on edit (`alert-channel-form-
  modal.tsx:39-40` reads `channel.config.url`/`webhook_url`), so the GET/LIST DTO MUST
  surface `config`. **Decision:** the read DTO echoes `config` (the page is an
  authenticated admin surface behind `RequireSession`, the same trust level that
  already returns connection secrets in some admin flows — but verify: if the operator
  wants the config masked in LIST and full only on GET-by-id, that is a hardening
  option). The **test-notification response** still NEVER echoes config (a non-admin-
  display path). RECOMMENDED: echo config in the read DTO (required for the edit form);
  flag the LIST-masking option for confirmation.
- **ESC-ALERT-TEST (RESOLVED at authoring — test-notification scope, binding default).**
  The brief asks the `/{id}/test` scope: "a best-effort send to the channel's
  configured target, deterministic in tests." **Decision:** `internal/governance/
  alertchannels.go` defines a `Sender` interface; the real `httpSender` does a
  best-effort HTTP POST to the config's `url`/`webhook_url` with a short fixed timeout
  (a non-2xx/transport error → `ok:false`, sanitized message). Tests inject a FAKE
  `Sender` (records the call, returns a configurable error) — NO network I/O, fully
  deterministic. The handler returns `{data:{ok, message}}` mirroring the mock
  `{ok,message}`. The message NEVER carries the secret config. If the operator wants a
  richer per-channel-type dispatch (Slack vs Discord vs generic webhook payload
  shaping), that is a follow-up; the default is a generic best-effort POST. RECOMMENDED
  as stated; flag for confirmation.
- **ESC-DOMAIN-WIRING (RESOLVED at authoring — how the handlers reach the engines,
  binding default).** Both domains get a governance engine (the brief-mandated
  evaluator + dispatcher seams). **Decision:** construct them via a thin per-call (or
  lazily-cached) accessor over `h.store` — e.g. `governance.NewGuardrailEngine(h.store)`
  and `governance.NewAlertDispatcher(governance.NewHTTPSender())` inside the handler —
  with NO `New(...)` signature change and NO new global state (decision 9; the
  `auditService()` accessor at `handlers.go:55` is the precedent for a domain accessor
  that does NOT widen `New`). Do NOT add an `h.guardrails`/`h.alerts` field if a free
  accessor suffices; if a field is cleaner, add it constructed inside the existing
  `New` body (NO signature change). Decide at impl; default = free accessor.
- **ESC-ROUTE (CONDITIONAL — fasthttp/router precedence).** `/api/alert-channels/{id}/test`
  (param-then-static, 3 segments) vs `/api/alert-channels/{id}` (param, 2 segments) and
  the singleton `/api/guardrails/test` (static, 2 segments) vs `/api/guardrails` (1
  segment) follow the file's existing static-before-param ordering. If the matcher
  mis-disambiguates (the `{id}` param swallows `test`, returning a 404 for the test
  path), STOP and ESCALATE for a path arrangement — never silently diverge page/mock/Go.
  Add an explicit Go handler test that `/api/alert-channels/1/test` resolves to
  `TestAlertChannel` (not `GetAlertChannel`) and `/api/guardrails/test` resolves to
  `TestGuardrails` (not the bare GET/PUT).
- **ESC-MOCK (CONDITIONAL — shared mock ripple).** `guardrails.ts`/`alert-channels.ts`
  are consumed only by the w7-gov-3 specs + the guardrails-tester vitest (verify with
  `grep -rn "guardrails\|alert-channels" ui/e2e/*.spec.ts ui/src/**/*.test.tsx`). The
  shared `utils.ts` `json()`/`error()` helpers are NOT edited. The w6-k path-B
  `seed/guardrails.ts` correction is the binding green-state — NEVER revert it. If a
  body correction reds a non-w7-gov-3 spec, STOP and ESCALATE for orchestrator
  serialization — no fudge, no frozen-branch edit.
- **ESC-AUDIT-REUSE (RESOLVED at authoring — reuse w7-gov-1 seam).** This plan's
  mutations (guardrails update; alert create/update/delete/test) call
  `h.recordAudit(ctx, action, target, details)` (`audit.go:64-72`) best-effort,
  post-success, mirroring w7-gov-1/gov-2. NO edit to audit.go / governance/audit.go /
  handlers.go. `details` are human-readable summaries (channel name, "enabled=true",
  flag counts) — NEVER raw config/blocklist payloads.
- **Serial-slot dependency (§1.6 / P5).** w7-gov-3 TAKES the routes_admin.go slot after
  w7-gov-2 releases it (chain MAP §219-224) and, as the LAST gov holder, RELEASES it to
  the next chain holder (w7-mcp-3) on close. Orchestrator confirms exactly one unmerged
  holder (decision 3) before T-routes.
- **No other blocking dependency.** All reused surfaces (store/prompttemplates.go
  numeric-PK + JSON-blob template, store/connections.go `*_enc` template, the Store
  cipher, respond.go, pathID, newTestEnv, migrate additive pattern, the w7-gov-1
  `h.recordAudit` seam) are in-tree at <base>. w7-gov-3 is unblocked once the serial
  slot is free.
```
