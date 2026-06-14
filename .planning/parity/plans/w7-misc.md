# Micro-plan w7-misc — Remaining W6-deferred small backends (Go)

```
wave: 7
plan: w7-misc
status: READY (rev 1 — authored against merged Waves 0–6 + the shipped W7 gov/plat/mcp
  chain, live tree @ <base>; WAVE-7-MAP w7-misc row ~line 186; serial chain §219-224
  (w7-misc is the LAST routes_admin holder); reconciliation decision 1 §36/§245;
  freeze rules §267)
runs: misc/cleanup track. Disjoint NEW domain/store/admin files; the ONLY shared hot
  file is internal/server/routes_admin.go (SERIAL SLOT). TAKES the slot LAST in the
  chain (w7-platnodes → w7-route → w7-gov-1 → w7-gov-2 → w7-gov-3 → w7-mcp-3 →
  w7-plat-1 → w7-plat-2 → w7-plat-3 → **w7-misc**; MAP §219-224). Releases to NOBODY
  (last holder) OR to w7-prov-oauth's trivial /api/oauth append if the orchestrator
  sequences it after (MAP §222-224).
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w7-misc:
ref-source: 9router frozen @ 827e5c3 — console/translator/models/OIDC surfaces; the
  BINDING contract for W7 is the W6 e2e mock (decision 1: real Go wins, mock corrected
  to mirror it). Mock sources:
    ui/e2e/mocks/handlers/{translator,nodes,models,logs,version}.ts + the EventSource
    fixture (ui/e2e/mocks/fixture.ts:78-97 — console-logs branch) + seeds.
base: <base> = git rev-parse HEAD recorded at P0. Substitute the actual SHA everywhere
  §5 says <base>. (At authoring HEAD = 2f6fb00; record the real P0 SHA.)
go-serial-slot: this plan holds the SINGLE in-flight edit to
  internal/server/routes_admin.go while live (W3/W4/W5/W6/W7 lesson; MAP decision 3).
  Slot must be FREE at P-check (the prior chain holder merged + slot released) before
  T-routes.
new-route: NO UI route files. All affected UI pages (/console, /translator, /models,
  the ModelSelectModal, the OIDC settings panel) ALREADY SHIPPED in W6 against mocks;
  this plan builds the REAL Go so the pages flip mock/variant-HAVE → true-HAVE and
  corrects the relevant mock bodies to mirror the Go DTOs.
```

---

## 0. BUILD-vs-DEFER decisions (executive summary — read first)

The brief lists FIVE candidate small backends. Per the w7-prov-media dead-route lesson
("DEFER honestly rather than build dead code"), each was scoped against the real in-tree
pipeline. Decisions, with the deciding evidence:

| Item | Decision | Deciding evidence |
|---|---|---|
| **1. console-logs SSE** (`GET /api/console-logs/stream`) | **BUILD** (additive log-capture seam + SSE endpoint, reusing the w5-e usagestream SSE precedent) | SSE precedent exists (`internal/admin/usagestream.go:33-38,171-189` — `SetBodyStreamWriter` + `data: ` frames + `Flush`). BUT `internal/logging/` has NO log buffer/broadcast — only `Debug.Logf` to an `io.Writer` (`logging/debug.go:24-35`) and the server uses `log.Printf` to stderr (`cmd/g0router/main.go:117,136,145`). So this plan ADDS a small in-process ring-buffer/broadcaster (`internal/logging/console.go`) that the SSE handler subscribes to, and a hermetic frame-emit test over a FAKE log source. See §1.1 + §8 ESC-CONSOLE-SRC. NOT dead code: the endpoint streams real captured server log lines. |
| **2. translator backend** (`/api/translator/{load,save,translate}`) | **BUILD** `load` + `translate`; **DEFER** `save` + `send` | `translate` has a real hermetic seam: `Registry.TranslateRequest(from, to Format, model, body, stream, credentials)` (`internal/translation/registry.go:208`) over the in-tree converters. The mock consumes ONLY `GET /api/translator/load` + `POST /api/translator/translate` (`translator.ts:21,31`); there is NO `save`/`send` consumer in the page or spec. Building `save`/`send` would be dead code. See §1.2 + §8 ESC-TRANS-SCOPE. |
| **3. models test/availability/custom** (`/api/models/{test,availability,custom}`) | **BUILD** all three | Catalog seam exists (`catalog.ModelsFor(provider)` `models.go:520`, `catalog.ResolveModel(provider,id)` `models.go:531`). Mock contracts are explicit: `/api/models/test`→`{ok,latency_ms}` (`nodes.ts:119-123`), `/api/models/availability`→`{available:[{id,available}]}` (`nodes.ts:125-133`), `/api/models/custom` GET/POST/DELETE (`models.ts:35-55`). `test` = injectable prober (hermetic, NO live network); `availability` = catalog/connection reachability; `custom` = additive custom-model store. See §1.3 + §8 ESC-MODELS-TEST. |
| **4. OIDC secret-at-rest** (`oidc_secret_enc`) | **BUILD** (additive column + store helpers + minimal read-site redirect) | `oidc_client_secret` is stored PLAINTEXT in the `settings` table via `SetSettings` (`settings.go:51`) and read in exactly TWO production sites (`auth.go:36` `oidcConfigured`, `oidc.go:182`). The `*_enc` precedent is well-worn (`migrate.go:51-53,186,198,210`; `oauthsessions.go:21,58` `cipher.Encrypt/Decrypt`). Build is additive at the store boundary; the two read sites get a MINIMAL surgical redirect to the encrypted accessor (the ONE sanctioned pre-existing-handler touch — see §1.4 + §8 ESC-OIDC-READSITE). |
| **5. chat-sessions admin CRUD** (`/api/chat-sessions`) | **DEFER** (recommended) | 9router keeps sessions in localStorage (open-questions w6-i ESC-1a, line 17). The chat send/receive turn uses the REAL gateway route `/v1/chat/completions` (`routes_openai.go:84`); no g0router row genuinely requires server-side persistence. Building `/api/chat-sessions` CRUD would add a table + handlers + routes with NO consuming requirement = exactly the dead-code the brief warns against. DEFER with a recorded follow-up. See §8 ESC-CHATSESS. |

**Net build set:** console-logs SSE + translator(load/translate) + models(test/availability/custom)
+ OIDC secret-at-rest. **Deferred:** translator save/send, chat-sessions CRUD.

**Rows that flip at closeout (§4 T-close):** PAR-UI-017 (console-logs, w6-i ESC-1),
PAR-UI-018/086 (translator, w6-i ESC-2), PAR-UI-117/119/120 (models custom/test/
availability, w6-f ESC-3) → variant-HAVE → true-HAVE; w6-j ESC-4 (OIDC secret-at-rest)
→ RESOLVED. open-questions w6-i ESC-1a (chat-sessions) stays OPEN with the DEFER
rationale recorded.

---

## 1. Scope — the four built domains

### Rows this plan closes

| Row / item | Claim | Target state after w7-misc |
|---|---|---|
| open-questions w6-i **ESC-1** / PAR-UI-017 (console-logs stream absent) | real `GET /api/console-logs/stream` SSE over a server log-capture pipeline | true-HAVE (Go — NEW `internal/logging/console.go` capture seam + `internal/admin/console.go` SSE, §1.1) |
| open-questions w6-i **ESC-2** / PAR-UI-018/086 (translator backend absent) | real `GET /api/translator/load` + `POST /api/translator/translate` over the translation registry | true-HAVE (Go — NEW `internal/admin/translator.go` over `internal/translation`, §1.2); save/send DEFERRED |
| open-questions w6-f **ESC-3** / PAR-UI-117/119/120 (models test/availability/custom mock-only) | real `POST /api/models/test`, `GET /api/models/availability`, `GET/POST /api/models/custom` + `DELETE /api/models/custom/{id}` | true-HAVE (Go — NEW `internal/store/custommodels.go` + `internal/admin/models.go` over the catalog, §1.3) |
| open-questions w6-j **ESC-4** (OIDC secret-at-rest) | OIDC client secret encrypted at rest (`oidc_secret_enc`), never echoed | RESOLVED (Go — additive column + `internal/store/oidcsecret.go` helpers + minimal read-site redirect, §1.4) |

### 1.1 Preconditions already satisfied by merged waves (evidence)

- **W6 UI pages SHIPPED and FROZEN (consume-only, MAP decision 8 / §267).** The
  `/console`, `/translator`, `/models` pages, the ModelSelectModal, and the OIDC
  settings panel render against the registered mocks / fixture. Binding acceptance
  contracts (must stay green at closeout):
  - `ui/e2e/console.spec.ts` — 3 tests: page loads ("Console", `console.spec.ts:11`);
    `[data-testid="console-log-row"]` appears (`:18-19`); each row shows a
    `[data-testid="console-log-level"]` badge (`:22-29`). **CRITICAL:** these are
    driven by the e2e `MockEventSource` (`fixture.ts:78-97`), NOT an HTTP route — the
    fixture stubs `window.EventSource` and pushes synthetic `{level,message,timestamp}`
    lines every 2500ms. The spec NEVER hits the real Go SSE endpoint. So the real Go is
    PURELY additive; the spec stays green untouched (the fixture is FROZEN — do NOT edit
    `fixture.ts`).
  - `ui/e2e/translator.spec.ts` — 4 tests: page loads ("Translator"); step textareas
    render (`textarea[aria-label="Client Request"]`); `[data-testid="translator-load"]`
    populates the first textarea with `/gpt-4o/` (from `GET /api/translator/load`);
    `[data-testid="translator-translate"]` makes the body contain "translated" (from
    `POST /api/translator/translate`). The page consumes `load` + `translate` ONLY.
  - `ui/e2e/models.spec.ts` — 3 tests: page loads ("Models"); `[data-testid="model-row"]`
    renders a `$` cost; toggling a row POSTs `/api/models/disabled`. (The disabled-model
    surface ALREADY has real Go — `disabledmodels.go`; NOT this plan. This plan adds the
    `custom`/`test`/`availability` siblings that the ModelSelectModal consumes via the
    `nodes.ts`/`models.ts` mocks.)
- **SSE precedent IN-TREE (the console-logs de-risk)** — `internal/admin/usagestream.go`
  (w5-e): `ctx.SetContentType("text/event-stream")` (`:34`) + `ctx.SetBodyStreamWriter(
  func(w *bufio.Writer){...})` (`:38`) + a `serve(w, ctx.Done(), done)` loop that selects
  on `clientDone` + a keepalive ticker (`:45-57`) + frame writes `"data: " + json + "\n\n"`
  then `Flush()` (`:171-189`). The keepalive/ping pattern (`writePing` `:158-168`) is the
  reuse target. **REUSE this shape** for the console SSE handler.
- **The log SOURCE does NOT exist yet (the build, not a wire-up).** `internal/logging/`
  has ONLY `Debug.Logf` writing to an injected `io.Writer` (`logging/debug.go:13-35`)
  and a `doc.go` that ASPIRES to own "the request log and the audit trail." The server's
  operational logs go to stderr via `log.Printf` (`cmd/g0router/main.go:117,136,145`).
  There is NO ring-buffer, broadcaster, or subscribe surface (`grep -rn 'Subscribe|
  Broadcast|chan' internal/logging/` → ZERO). So this plan ADDS a small capture seam
  (§1.1 contract). See §8 ESC-CONSOLE-SRC for the source decision.
- **Translation registry IN-TREE (the translator de-risk)** — `internal/translation/
  registry.go`: `type Registry` (`:132`), `NewRegistry()` (`:138`), and
  `TranslateRequest(from, to Format, model string, body map[string]any, stream bool,
  credentials map[string]any) (map[string]any, error)` (`:208`). `Format` is a string
  enum (`formats.go:6-22`: `FormatOpenAI`, `FormatClaude`, `FormatGemini`, …) with a
  `ParseFormat` helper (`formats.go:28+`). This is the EXACT hermetic seam for the
  translator `translate` endpoint — canned `{from,to,model,payload}` in, transformed
  payload out, NO network.
- **Catalog IN-TREE (the models de-risk)** — `internal/providers/catalog/models.go`:
  `type ModelEntry` (`:8`), `ModelsFor(provider string) []ModelEntry` (`:520`),
  `ResolveModel(provider, id string) (ModelEntry, bool)` (`:531`); `internal/providers/
  catalog/aliases.go` `ProviderAliasCount` etc. Provider rows live on
  `store.ProviderRecord{ID,Name,Type,BaseURL,Enabled,Prefix,APIType,…}` (`providers.go:
  12-21`). These power `availability` (which catalog/connection models are reachable) and
  `test` (resolve a model then ping via an injectable prober).
- **OIDC secret storage IN-TREE (plaintext today — the secret-at-rest target)** —
  `oidc_client_secret` is written via the generic flat `PutSettings` →
  `store.SetSettings` (`settings.go:51`) and read in exactly TWO production sites:
  `oidcConfigured(settings)` checks `settings["oidc_client_secret"] != ""`
  (`auth.go:34-36`) and the callback builds the token-exchange with `ClientSecret:
  strings.TrimSpace(settings["oidc_client_secret"])` (`oidc.go:182`). The OIDC TEST
  handler (`oidc.go:247-300`) operates on a CALLER-PROVIDED body secret, NOT the stored
  one ("stored OIDC secrets" comment `oidc.go:247`), so it does NOT read the stored
  value. The `*_enc` precedent: `migrate.go:51-53` (`secret_enc`/`access_token_enc`/
  `refresh_token_enc`), `:186` (`config_enc`), `:198` (`password_enc`), `:210`
  (`token_enc`); written/read via `s.cipher.Encrypt`/`Decrypt` (`oauthsessions.go:21,58`).
  The `Store` carries `cipher *Cipher` (`store.go`).
- **Envelope + handler patterns** (`internal/admin/respond.go`): `writeData(ctx, status,
  data)` / `writeError(ctx, status, message)` → `{data, error:{message}}` snake_case
  (`respond.go:19,23`). `pathID(ctx.UserValue("id"))` extracts `{id}` (`handlers.go:158`).
- **Audit write seam IN-TREE (reuse, NO new audit code)** — w7-gov-1 added `h.recordAudit(
  ctx, action, target, details string)` (`internal/admin/audit.go:64`, best-effort,
  swallows errors, actor from `ctx.UserValue(userKey).(*store.User)`). REUSE it on the
  custom-model create/delete + the OIDC-secret update (best-effort, post-success, NEVER
  fails the parent). NO edit to audit.go.
- **Admin test harness** (`internal/admin/admin_test.go` `newTestEnv` + `call` +
  `dataField[T]` + `errMessage` + `loginToken`): real `store.Open(tempDB, secret)` +
  `auth.NewSessions` + `SeedAdmin` + `New(...)`. NO mocks. The authoritative proof
  surface. The OIDC test env (`oidc_test.go:16` `oidcTestEnv`) seeds `oidc_*` settings.
- **Handlers injection** (`internal/admin/handlers.go`): `New(st, sessions, flows)`
  (`:39`) + additive setters (`SetUsageServices` `:76`, … `SetMCPProbe` `:153`). New
  domains compose `h.store` directly; injectables (the console broadcaster, the model
  prober, the translation registry) are wired via NEW additive setters mirroring
  `SetNodeProber` (`:107`) — **NO `New(...)` signature change** (MAP decision 9).
- **Migrations additive-only** (`internal/store/migrate.go`): new tables via the `tables`
  slice with `CREATE TABLE IF NOT EXISTS` (`:15-32`); new columns via `ensureColumn(db,
  table, column, decl)` (`:454-456`, applied in the loop ending `:380`). ADDITIVE ONLY
  (decision 2).

### 1.2 The mock contracts these flips must mirror (binding — decision 1)

**Decision 1 (MAP §36, §245):** real Go wins; the W6 mock body + seed are corrected IN
THIS PLAN to mirror the real Go `{data,error}` snake_case DTO. The page is FROZEN
(decision 8); prefer matching the mock's existing field names in the Go DTO; only
ESCALATE if impossible.

**Console-logs** (`ui/e2e/mocks/fixture.ts:78-97` — the `MockEventSource` console branch;
NOT an HTTP handler):
- The fixture pushes `{ data: JSON.stringify({ level, message, timestamp }) }` events
  over `EventSource("/api/console-logs/stream")` (`fixture.ts:78-97`). The page reads
  `event.data` as JSON and renders `[data-testid="console-log-row"]` with a
  `[data-testid="console-log-level"]` badge.
- **The canonical SSE frame DTO** = `{level:string, message:string, timestamp:string}`
  (ISO-8601). The real Go SSE MUST emit `data: {"level":..,"message":..,"timestamp":..}\n\n`
  with these exact field names so the page renders identically when pointed at real Go.
- **Reconciliation:** the fixture is the shared SSE-stub for ALL specs (it also has the
  `/api/traffic/stream` branch, `fixture.ts:60-77`) — **do NOT edit `fixture.ts`** (out
  of scope, FROZEN; same disposition as the shared `utils.ts`). The Go frame shape is
  proven by the Go unit test (§1.1 / §3), not by an e2e edit. There is no `logs.ts`
  console-stream HTTP handler to correct (`logs.ts` serves `/api/usage/request-logs`
  ONLY — unrelated request-log history, `logs.ts:14-26`).

**Translator** (`ui/e2e/mocks/handlers/translator.ts`):
- Routes the page consumes: `GET /api/translator/load?file=<name>` →
  `{data:{file, payload:<json string>}}` (`translator.ts:21-29`); `POST /api/translator/
  translate` body `{from?,to?,payload?}` → `{data:{payload:<json string with
  "translated":true>}}` (`translator.ts:31-47`). NO `save`, NO `send` consumer.
- The `load` sample is a `SAMPLE_CLIENT_REQUEST` containing `"gpt-4o"` (`translator.ts:
  13-18`); the spec asserts the first textarea gets `/gpt-4o/` (`translator.spec.ts:34`).
- The `translate` mock echoes a `{translated:true, from, to, payload}` marker; the spec
  asserts the body contains "translated" (`translator.spec.ts:48`).
- **Reconciliation:** the Go `GET /api/translator/load` returns `{data:{file, payload}}`
  where `payload` is a JSON-serialized sample client request CONTAINING `"gpt-4o"` (so
  the spec's `/gpt-4o/` assertion holds against real Go). The Go `POST /api/translator/
  translate` runs `Registry.TranslateRequest` and returns `{data:{payload, translated:
  true, from, to}}` — the response MUST carry a `"translated"` token (mirror the mock
  marker) so the spec's "translated" assertion holds. The corrected `translator.ts` keeps
  these two routes and DROPS nothing (it never had save/send). See §8 ESC-TRANS-SCOPE for
  the field reconciliation if `TranslateRequest`'s output shape diverges from the mock's
  echo shape (the Go wraps the real transformed body under `payload` + a `translated:true`
  sibling, preserving the mock's asserted token).

**Models** (`ui/e2e/mocks/handlers/nodes.ts` + `ui/e2e/mocks/handlers/models.ts`):
- `POST /api/models/test` → `{data:{ok:true, latency_ms:42}}` (`nodes.ts:119-123`). Body
  carries the model to test. The Go returns `{data:{ok:bool, latency_ms:int}}` (mirror).
- `GET /api/models/availability` → `{data:{available:[{id, available:true}]}}`
  (`nodes.ts:125-133`). The Go returns `{data:{available:[{id, available}]}}` (mirror).
- `GET /api/models/custom` → `{data:[customModel]}`; `POST /api/models/custom` body →
  `{data:{id, ...body, is_disabled:false, is_custom:true}}`; `DELETE /api/models/custom/
  {id}` → `{data:{}}` (`models.ts:35-55`). The Go custom-model DTO mirrors `{id, …,
  is_custom:true}`.
- **Reconciliation:** correct `nodes.ts` (`test`/`availability` branches) + `models.ts`
  (`custom` branches) SUCCESS bodies to mirror the Go DTOs (field names already match —
  likely verification-only). Do NOT touch the `nodes.ts` provider-node CRUD branches
  (w7-platnodes-owned, FROZEN) or the `models.ts` `/api/models` + `/api/models/disabled`
  branches (w6-e/disabledmodels-owned, FROZEN). See §8 ESC-MODELS-MOCK.

**OIDC** — NO mock change. The OIDC secret is persisted via the REAL flat `PUT /api/
settings` (no dedicated OIDC mock route, `grep oidc ui/e2e/mocks/handlers/settings.ts`
→ ZERO) and the secret-at-rest change is INVISIBLE to the wire DTO (the secret is never
echoed in any response, before or after). No e2e/mock touch for OIDC.

### 1.3 Architecture (binding — layered DDD, decision 4)

w7-gov-1/gov-2/gov-3 RESOLVED ESC-ARCH: **no in-tree arch test enforces transport→
domain→repository** (CRUD handlers call `h.store` directly). This plan follows the
precedent — a domain seam is added ONLY where non-trivial logic must be unit-tested in
isolation:

```
console-logs: admin/console.go  → logging/console.go (NEW capture seam: ring buffer + broadcaster)
              (the SSE handler subscribes to the broadcaster; the broadcaster is the
               testable seam — hermetic frame-emit test over a fake/in-memory source)
translator:   admin/translator.go → translation.Registry (EXISTING — TranslateRequest)
              (thin transport over the existing registry; NO new domain file — the
               registry IS the domain logic, unit-tested already in internal/translation)
models:       admin/models.go     → store/custommodels.go (NEW custom-model CRUD)
                                  → catalog.ModelsFor/ResolveModel (EXISTING)
                                  → an injectable model Prober (test) — hermetic
oidc-secret:  store/oidcsecret.go (NEW: GetOIDCSecret/SetOIDCSecret over cipher)
              + a MINIMAL surgical redirect at the two read sites (auth.go, oidc.go)
```

- **console-logs broadcaster (`internal/logging/console.go`) is WARRANTED** — it holds a
  bounded ring buffer of recent lines + a fan-out to subscribers (channels). It is the
  testable seam: a hermetic test PUSHES synthetic lines and asserts the subscriber
  receives the JSON-shaped frame `{level,message,timestamp}`. Constructor
  `NewConsoleLog(capacity int)`; methods `Append(level, message string)`, `Subscribe()
  (<-chan ConsoleLine, func())` (the unsubscribe closure), `Recent() []ConsoleLine`. No
  `init()`; errors-as-values; no global state (the instance is held on `Handlers` via an
  additive setter). See §8 ESC-CONSOLE-SRC for what FEEDS `Append` (the source decision).
- **translator needs NO new domain file** — `translation.Registry.TranslateRequest` is
  the existing, already-unit-tested domain logic. `admin/translator.go` is a thin
  transport that parses `{from,to,model,payload}`, calls the registry, and wraps the
  result. The registry is injected via an additive `SetTranslationRegistry` setter (NO
  `New` change). If a per-handler registry construction is cleaner (`translation.
  NewRegistry()` is cheap + stateless), construct it lazily in the handler — decide at
  T-translator.
- **models test prober is an injectable seam** — `test` must NOT hit a live network in
  unit tests. Define a tiny `ModelProber interface { Probe(ctx, providerID, modelID
  string) (ok bool, latencyMS int, err error) }`; the default impl does a best-effort
  reachability check; tests inject a fake. Mirror the w7-platnodes `NodeProber` injection
  precedent (`SetNodeProber`, `handlers.go:107`). `availability` + `custom` are pure
  catalog/store reads — no prober needed.

### 1.4 Console-logs Go contract (NEW, TDD)

`internal/logging/console.go` (NEW):
```go
type ConsoleLine struct { Level, Message, Timestamp string } // Timestamp = RFC3339
type ConsoleLog struct { /* ring buffer + subscriber set + mutex */ }
func NewConsoleLog(capacity int) *ConsoleLog
func (c *ConsoleLog) Append(level, message string)            // ring-append + fan-out
func (c *ConsoleLog) Subscribe() (<-chan ConsoleLine, func()) // chan + unsubscribe
func (c *ConsoleLog) Recent() []ConsoleLine                   // snapshot for replay
```
- Bounded ring (drop oldest at capacity). `Subscribe` returns a buffered channel; a slow
  consumer drops frames rather than blocking `Append` (never block the log path).
- No `init()`, no global state — the instance lives on `Handlers` (additive
  `SetConsoleLog(*logging.ConsoleLog)` setter mirroring `SetNodeProber`).

`internal/admin/console.go` (NEW):

| Handler | Route | Shape | Notes |
|---|---|---|---|
| `ConsoleLogStream` | `GET /api/console-logs/stream` | SSE: `data: {"level":..,"message":..,"timestamp":..}\n\n` per line + periodic keepalive ping | MIRROR `usagestream.go`: `SetContentType("text/event-stream")` + `SetBodyStreamWriter` + select on `ctx.Done()` + a keepalive ticker; on connect, REPLAY `Recent()` then stream `Subscribe()` frames; on `ctx.Done()` call the unsubscribe closure. 501-safe if `h.console` is nil (unwired) — mirrors the w6-j nil-safe Shutdown precedent. |

**Source feeding `Append` (§8 ESC-CONSOLE-SRC — RECOMMENDED default).** Capture the
server's existing `log` output by setting `log.SetOutput(io.MultiWriter(os.Stderr,
consoleLogWriter))` in `cmd/g0router/main.go` where the `ConsoleLog` is constructed (an
`io.Writer` adapter whose `Write` parses the line into `{level,message}` and calls
`Append`). This is a MINIMAL additive `main.go` wiring (mirrors the w6-j ESC-1
`SetVersionInfo`/`SetShutdownFunc` additive forward) — NOT a frozen-handler edit. The
hermetic test does NOT exercise `main.go`; it drives `ConsoleLog.Append` directly and
asserts the SSE handler emits the frame. If the orchestrator prefers zero `main.go`
touch, the fallback is to expose `ConsoleLog` and let future log call-sites push to it
explicitly (the endpoint still works, just with fewer captured lines) — RECOMMENDED:
the `MultiWriter` capture (cheap, captures everything, additive).

### 1.5 Translator Go contract (NEW, TDD)

`internal/admin/translator.go` (NEW):

| Handler | Route | Body / response | Notes |
|---|---|---|---|
| `TranslatorLoad` | `GET /api/translator/load?file=<name>` | `{data:{file, payload}}` where `payload` is a JSON-serialized sample client request CONTAINING `"gpt-4o"` (mirror `translator.ts:13-29`) | sample payloads are in-handler constants (NO new store, mirrors the self-contained mock). `file` defaults to `"sample"`. |
| `TranslatorTranslate` | `POST /api/translator/translate` | body `{from?,to?,model?,payload}`; parse `payload` (JSON), `ParseFormat(from)`/`ParseFormat(to)` (defaults openai→claude), call `registry.TranslateRequest(from,to,model,body,false,nil)`; return `{data:{payload:<json of transformed>, translated:true, from, to}}` | the response MUST carry a `"translated"` token (mirror the mock marker, §1.2). NO credentials, NO network — `TranslateRequest` is a pure body transform. 400 on unparseable payload/format. |

**DEFERRED (§8 ESC-TRANS-SCOPE):** `POST /api/translator/save` + `POST /api/translator/
send`. NO page/spec/mock consumer exists; building them is dead code. Recorded as a
follow-up (`save` = persist a named sample; `send` = execute the translated request
against a live provider — the latter overlaps the real gateway path and needs a scope
decision). Do NOT register these routes.

### 1.6 Models Go contract (NEW, TDD)

Table `custom_models` (additive, `migrate.go` tables slice):
```sql
CREATE TABLE IF NOT EXISTS custom_models (
  id TEXT PRIMARY KEY,
  provider TEXT NOT NULL DEFAULT '',
  model_id TEXT NOT NULL,
  name TEXT NOT NULL DEFAULT '',
  config_json TEXT NOT NULL DEFAULT '{}',   -- JSON blob for misc fields (cost/context)
  created_at INTEGER NOT NULL
)
```
(No secret fields — custom-model metadata is not sensitive; no `*_enc`.)

`internal/store/custommodels.go` (NEW): `CustomModel{ID, Provider, ModelID, Name string,
Config map[string]any, CreatedAt int64}` + `CreateCustomModel`/`ListCustomModels`/
`DeleteCustomModel(id)` + `scanCustomModel`; `newID()`, unix ts, `ErrNotFound`,
`config_json` JSON blob (mirror `virtualkeys.go` config-blob precedent).

`internal/admin/models.go` (NEW):

| Handler | Route | Shape (snake_case, `{data}`) | Notes |
|---|---|---|---|
| `TestModel` | `POST /api/models/test` | body `{provider?, model_id} ` ; resolve via `catalog.ResolveModel`; call `h.modelProber.Probe(...)`; returns `{data:{ok:bool, latency_ms:int}}` | INJECTABLE prober (hermetic). 400 on empty model_id. mirror `nodes.ts:119-123` |
| `ModelAvailability` | `GET /api/models/availability` | `{data:{available:[{id, available:bool}]}}` | derive from catalog models + (optionally) enabled provider connections; deterministic in tests. mirror `nodes.ts:125-133` |
| `ListCustomModels` | `GET /api/models/custom` | `{data:[customModelDTO]}` (bare array under data) | mirror `models.ts:37` |
| `CreateCustomModel` | `POST /api/models/custom` | body `{provider?, model_id, name?, ...}`; returns `{data:customModelDTO}` with `is_custom:true` | `recordAudit(ctx,"custom_model.create",model_id,…)`. 400 empty model_id. mirror `models.ts:38-44` |
| `DeleteCustomModel` | `DELETE /api/models/custom/{id}` | `{data:{}}` or 404 | `recordAudit(ctx,"custom_model.delete",id,…)`. mirror `models.ts:47-54` |

`customModelDTO{id, provider, model_id, name, is_custom:true, is_disabled:false, …}`
(plus any config fields the page reads — verify against the UI `CustomModel`/`Model`
type at impl).

**Route-ownership note:** the existing `/api/models/disabled` routes (`routes_admin.go:
201-203`, `disabledmodels.go`) are FROZEN — do NOT touch. This plan ADDS the disjoint
`/api/models/test`, `/api/models/availability`, `/api/models/custom[/{id}]` routes.
Static `/api/models/test` + `/api/models/availability` + `/api/models/custom` register
BEFORE `/api/models/custom/{id}` (static-before-param). See §8 ESC-MODELS-ROUTE.

### 1.7 OIDC secret-at-rest Go contract (additive column + minimal read-site redirect)

Additive column on the existing `settings`-adjacent storage. **DECISION (§8
ESC-OIDC-STORE):** add a dedicated single-value encrypted holder rather than widening
the flat `settings` table semantics. RECOMMENDED: a tiny additive store seam that keeps
the encrypted secret in its own column on a dedicated single-row table OR an
`ensureColumn` on a config table — the cheapest additive form is a dedicated
`oidc_secret` single-row table with an `oidc_secret_enc` column (mirrors the singleton
pattern + the `*_enc` precedent):
```sql
CREATE TABLE IF NOT EXISTS oidc_secret (
  id INTEGER PRIMARY KEY,            -- always 1
  oidc_secret_enc TEXT NOT NULL DEFAULT ''
)
```

`internal/store/oidcsecret.go` (NEW):
- `GetOIDCSecret() (string, error)` — read the singleton row, `s.cipher.Decrypt` the
  `oidc_secret_enc`; empty string if unset.
- `SetOIDCSecret(secret string) error` — `s.cipher.Encrypt`, UPSERT the singleton row.
- **Migration of the existing plaintext value:** on first `GetOIDCSecret` (or in a
  one-shot additive migration step), if `oidc_secret_enc` is empty AND the legacy
  `settings["oidc_client_secret"]` is non-empty, encrypt-and-move it, then BLANK the
  plaintext settings key. Additive + idempotent. See §8 ESC-OIDC-MIGRATE.

**Minimal read-site redirect (the ONE sanctioned pre-existing-handler touch).** Two
production reads of `settings["oidc_client_secret"]` must consult the encrypted accessor:
- `internal/admin/auth.go:34-36` `oidcConfigured` — change the secret check to consult
  `h.store.GetOIDCSecret()` (non-empty) instead of `settings["oidc_client_secret"]`.
- `internal/admin/oidc.go:182` callback — set `ClientSecret:` from `h.store.GetOIDCSecret()`
  instead of `settings["oidc_client_secret"]`.

And one write redirect:
- `internal/admin/settings.go` `PutSettings` (`:19-26`) — when the incoming flat settings
  body carries `oidc_client_secret`, route it to `h.store.SetOIDCSecret(value)` and strip
  it from the plaintext `SetSettings` map (so it is NEVER persisted plaintext). All other
  keys pass through unchanged.

These three surgical edits are the MINIMUM to make the secret encrypted-at-rest while
preserving every existing behavior (login-mode gating, callback exchange, the OIDC test
handler which uses a body-provided secret and is untouched). They are explicitly
in-scope for w7-misc as the OIDC-secret-handling exception (§3 / §6). The OIDC TEST
handler (`oidc.go:247-300`) is UNTOUCHED (it reads a caller-provided body secret, not the
stored one).

**No-echo guarantee:** the secret is never returned by any handler. `GetSettings`
(`settings.go:9-17`) returns the flat settings map — after the migration BLANKS
`oidc_client_secret`, the map no longer carries it (the panel shows an empty/placeholder
secret field, the standard "secret already set" UX). Add a §5 grep proof that no handler
echoes `oidc_secret_enc` or the decrypted secret.

### 1.8 routes_admin.go registration (serial-slot additive, §3)

Append AFTER the last existing block (the file ends ~line 257; the models/disabled block
is `:201-203`), static-before-`{id}`:
```go
// Console-logs SSE (real server log stream; mirrors usagestream SSE).
r.GET("/api/console-logs/stream", h.RequireSession(h.ConsoleLogStream))
// Translator (load sample + translate over the translation registry).
r.GET("/api/translator/load", h.RequireSession(h.TranslatorLoad))
r.POST("/api/translator/translate", h.RequireSession(h.TranslatorTranslate))
// Models test/availability/custom (static before {id}).
r.POST("/api/models/test", h.RequireSession(h.TestModel))
r.GET("/api/models/availability", h.RequireSession(h.ModelAvailability))
r.GET("/api/models/custom", h.RequireSession(h.ListCustomModels))
r.POST("/api/models/custom", h.RequireSession(h.CreateCustomModel))
r.DELETE("/api/models/custom/{id}", h.RequireSession(h.DeleteCustomModel))
```
(OIDC secret-at-rest adds NO new route — it redirects existing `/api/settings` +
`/api/auth/login`/OIDC-callback internals.) Route-precedence note: `/api/models/test`,
`/api/models/availability`, `/api/models/custom` (static) vs the existing `/api/models/
disabled` (static) and the new `/api/models/custom/{id}` (param) — all static siblings,
no collision; `custom/{id}` is the only param path. A genuine mis-disambiguation is §8
ESC-MODELS-ROUTE. Diff bound §5: the route block is ONE commit, additive only.

### NOT in scope (explicit)

- **chat-sessions CRUD — DEFERRED** (§0 / §8 ESC-CHATSESS). NO `/api/chat-sessions`
  route, table, or handler. The `chat-sessions.ts` mock stays as-is (consumed only by
  the chat page's optional session list; 9router keeps these client-side).
- **translator save/send — DEFERRED** (§1.5 / §8 ESC-TRANS-SCOPE). NO `/api/translator/
  save` or `/api/translator/send`.
- **No UI src edits** — `/console`, `/translator`, `/models`, ModelSelectModal, OIDC
  panel FROZEN (decision 8). Only the sanctioned `nodes.ts`/`models.ts`/`translator.ts`
  mock-body verifications/corrections (§1.2 / §3).
- **No `fixture.ts` edit** — the console `MockEventSource` branch + the `/api/traffic/
  stream` branch are shared, FROZEN e2e infra (§1.2).
- **No edit to the shared mock `utils.ts`/index/seed-index/`store.ts`.**
- **No edits to pre-existing admin handlers EXCEPT the three sanctioned OIDC-secret
  read/write redirects** (`auth.go:34-36`, `oidc.go:182`, `settings.go` PutSettings) —
  these are the explicit OIDC-secret-handling exception (§1.7). Every other pre-existing
  handler body (disabledmodels.go, nodes.go, version.go, the OIDC test handler
  `oidc.go:247-300`, login/logout/me, providers*, connections, …) is FORBIDDEN.
- **No `New(...)` signature change / no new global state** (decision 9) — the console
  broadcaster, model prober, and translation registry are wired via NEW additive setters
  mirroring `SetNodeProber`.
- **No destructive DDL / column renames** — additive `ensureTable`/`ensureColumn` ONLY
  (decision 2).
- **No secret exposure** — OIDC secret encrypted at rest, never echoed; custom-model +
  console + translator carry no secrets (§5 grep proofs).

---

## 2. Precondition checks

Run all before any edit; abort and report to orchestrator on any failure.

```bash
# P0 — clean tree; record <base>
git status --porcelain     # must be empty (explicit `git add <file>`, never -A;
                           # ui/dist/** gitignored — never stage it)
git rev-parse HEAD         # record as <base> for §5

# P1 — the gaps are REAL (no Go for any built domain)
grep -nE '/api/console-logs|/api/translator|/api/models/(test|availability|custom)' internal/server/routes_admin.go ; echo "^ expect EMPTY"
test ! -e internal/admin/console.go && test ! -e internal/admin/translator.go && test ! -e internal/admin/models.go && echo "admin gap OK"
test ! -e internal/logging/console.go && echo "console source gap OK"
test ! -e internal/store/custommodels.go && test ! -e internal/store/oidcsecret.go && echo "store gap OK"
grep -rnE 'chat-sessions|ChatSession|console-logs|ConsoleLog|/api/translator' internal/ ; echo "^ expect EMPTY (no Go yet)"

# P2 — reused seams present (the de-risk)
grep -n "SetBodyStreamWriter\|text/event-stream\|func (s \*streamState) writeData\|writePing" internal/admin/usagestream.go
grep -n "func (r \*Registry) TranslateRequest\|func NewRegistry\|type Format" internal/translation/registry.go internal/translation/formats.go
grep -n "func ModelsFor\|func ResolveModel\|type ModelEntry" internal/providers/catalog/models.go
grep -n "func (h \*Handlers) recordAudit" internal/admin/audit.go
grep -n "func writeData\|func writeError" internal/admin/respond.go
grep -n "func pathID" internal/admin/handlers.go
grep -n "func newTestEnv\|func call\|func dataField\|func loginToken" internal/admin/admin_test.go
grep -n "func (h \*Handlers) SetNodeProber" internal/admin/handlers.go   # additive-setter precedent

# P3 — OIDC plaintext secret read-sites (the redirect targets) + *_enc precedent + cipher
grep -n 'settings\["oidc_client_secret"\]' internal/admin/auth.go internal/admin/oidc.go
grep -n "func (h \*Handlers) PutSettings\|func (s \*Store) SetSettings\|func (s \*Store) GetSettings" internal/admin/settings.go internal/store/settings.go
grep -nE "_enc TEXT NOT NULL|func ensureColumn|CREATE TABLE IF NOT EXISTS" internal/store/migrate.go | head
grep -n "s.cipher.Encrypt\|s.cipher.Decrypt" internal/store/oauthsessions.go
# CONFIRM the OIDC TEST handler does NOT read the stored secret (must stay untouched):
grep -n "stored OIDC secrets\|body.ClientSecret\|clientSecret := body" internal/admin/oidc.go

# P4 — the W6 UI + specs + mocks/fixture present (consume-only)
test -f ui/e2e/console.spec.ts && test -f ui/e2e/translator.spec.ts && test -f ui/e2e/models.spec.ts && echo "specs present"
test -f ui/e2e/mocks/handlers/translator.ts && test -f ui/e2e/mocks/handlers/nodes.ts && test -f ui/e2e/mocks/handlers/models.ts && echo "mocks present"
grep -n "console-logs/stream\|level\|timestamp" ui/e2e/mocks/fixture.ts ; echo "^ the SSE frame DTO to mirror (FROZEN fixture)"
grep -n "models/test\|models/availability" ui/e2e/mocks/handlers/nodes.ts ; echo "^ test/availability mock shapes (§1.6)"
grep -n "models/custom" ui/e2e/mocks/handlers/models.ts ; echo "^ custom mock shapes (§1.6)"
grep -n "translator/load\|translator/translate\|gpt-4o\|translated" ui/e2e/mocks/handlers/translator.ts ; echo "^ load/translate contract (§1.5)"

# P5 — routes_admin.go serial slot is FREE (prior chain holder merged + released)
git log --oneline -5 -- internal/server/routes_admin.go   # last touch = prior chain holder (merged)
# Orchestrator MUST confirm no concurrent W7 plan holds an unmerged routes_admin.go edit
# before w7-misc begins T-routes. w7-misc is the LAST chain holder (MAP §219-224).

# P6 — green at base
go test ./... && go vet ./... && go build ./...     # exit 0 (Go untouched-green)
# e2e ISOLATED — kill stale chromium/vite-preview first (e2e-hygiene); never run
# concurrently with another playwright invocation; NEVER revert ui/dist/index.html.
pkill -f 'chromium|vite preview' 2>/dev/null ; true
cd ui && npx playwright test e2e/console.spec.ts e2e/translator.spec.ts e2e/models.spec.ts
# Record base: these PASS at base against the W6 mocks/fixture. They must STAY green
# after the mock-body verifications. Record exact pass/fail in WORKFLOW.md.
cd ui && npm run build                               # exit 0
```

---

## 3. Exclusive file ownership

After w7-misc merges, all CREATE files are owned by w7-misc; later plans consume, never
edit (MAP decision 7).

**CREATE — logging (NEW):**

| File | Contract |
|---|---|
| `internal/logging/console.go` | `ConsoleLine` + `ConsoleLog` (ring buffer + subscriber fan-out) + `NewConsoleLog`/`Append`/`Subscribe`/`Recent` (§1.4). No `init()`; no global state. |
| `internal/logging/console_test.go` | hermetic: `Append` N lines → `Recent()` returns the bounded newest set; `Subscribe()` receives a pushed `{level,message,timestamp}` frame; slow-consumer drop does not block `Append`; unsubscribe stops delivery. RED first. |

**CREATE — store (NEW):**

| File | Contract |
|---|---|
| `internal/store/custommodels.go` | `CustomModel` struct + `CreateCustomModel`/`ListCustomModels`/`DeleteCustomModel` + `scanCustomModel`; mirrors `virtualkeys.go` config-blob CRUD. `newID()`, unix ts, `ErrNotFound`. |
| `internal/store/custommodels_test.go` | table-driven via temp `store.Open`: create→list→delete→404. RED first. |
| `internal/store/oidcsecret.go` | `GetOIDCSecret`/`SetOIDCSecret` over `s.cipher.Encrypt/Decrypt` (singleton `oidc_secret` row); first-read migration of the legacy plaintext `settings["oidc_client_secret"]` (encrypt-move-blank, idempotent, §1.7). |
| `internal/store/oidcsecret_test.go` | Set→Get round-trips through encrypt/decrypt; **the raw `oidc_secret_enc` column is NOT plaintext** (read it directly, assert != the secret); the legacy-plaintext migration encrypts + blanks `settings["oidc_client_secret"]`; empty default. RED first. |

**EXTEND — store (additive table/column registration only):**

| File | Change (additive ONLY) |
|---|---|
| `internal/store/migrate.go` | ADD `custom_models` + `oidc_secret` tables to the `tables` slice. ADDITIVE ONLY — no DROP/RENAME. |
| `internal/store/migrate_test.go` (if present — EXTEND additively; else rely on store tests) | assert the two new tables exist post-migrate. |

**CREATE — transport (NEW):**

| File | Contract |
|---|---|
| `internal/admin/console.go` | `ConsoleLogStream` SSE handler (mirror `usagestream.go`: stream-writer + keepalive + `Recent()` replay + `Subscribe()` + `ctx.Done()` unsubscribe). nil-safe (501 if `h.console` unwired). |
| `internal/admin/console_test.go` | via `newTestEnv` + an injected `ConsoleLog`: append a line, open the stream, assert the response carries a `data: {"level":..,"message":..,"timestamp":..}` frame with the exact field names; nil-console → 501. Hermetic (no real log source; drive `Append` directly). RED first. |
| `internal/admin/translator.go` | `TranslatorLoad` (sample with `"gpt-4o"`) + `TranslatorTranslate` (`registry.TranslateRequest`; response carries a `"translated"` token). `writeData`/`writeError`. |
| `internal/admin/translator_test.go` | via `newTestEnv`: `load` returns a payload containing `"gpt-4o"`; `translate` with a canned `{from:"openai",to:"claude",payload:<openai body>}` returns `{data:{payload,...,translated:true}}` (assert the transformed payload differs from input AND carries `translated`); bad payload→400. Hermetic (no network). RED first. |
| `internal/admin/models.go` | `TestModel`/`ModelAvailability`/`ListCustomModels`/`CreateCustomModel`/`DeleteCustomModel` + `customModelDTO`; injectable `ModelProber`; `catalog.ResolveModel`/`ModelsFor`; `h.recordAudit` on custom create/delete. |
| `internal/admin/models_test.go` | via `newTestEnv` + a FAKE `ModelProber`: `test` returns `{ok,latency_ms}` deterministically (no network); `availability` returns `{available:[...]}`; custom create→list(≥1)→delete→404; create empty model_id→400; **assert an audit entry on custom create** (`GetAudit`). RED first. |

**MODIFY — handlers wiring (additive setters only, NO `New` sig change):**

| File | Change |
|---|---|
| `internal/admin/handlers.go` | ADD additive fields `console *logging.ConsoleLog`, `modelProber ModelProber`, (optional) `translation *translation.Registry`; ADD additive setters `SetConsoleLog`, `SetModelProber`, (optional) `SetTranslationRegistry` — mirror `SetNodeProber` (`:107`). `New(...)` signature UNCHANGED. |

**MODIFY — main/server wiring (minimal additive forward, §1.4 ESC-CONSOLE-SRC):**

| File | Change |
|---|---|
| `cmd/g0router/main.go` | Construct `logging.NewConsoleLog(...)`, `h.SetConsoleLog(...)`, and `log.SetOutput(io.MultiWriter(os.Stderr, consoleWriter))` (additive; mirrors the w6-j ESC-1 `SetVersionInfo`/`SetShutdownFunc` forward). If this forces a non-additive change, ESCALATE (§8 ESC-CONSOLE-SRC). |

**MODIFY — the THREE sanctioned OIDC-secret redirects (the explicit exception):**

| File | Change |
|---|---|
| `internal/admin/auth.go` (`oidcConfigured`, `:34-36`) | the secret check consults `h.store.GetOIDCSecret() != ""` instead of `settings["oidc_client_secret"]`. NOTHING else in auth.go changes. |
| `internal/admin/oidc.go` (callback, `:182`) | `ClientSecret:` sourced from `h.store.GetOIDCSecret()` instead of `settings["oidc_client_secret"]`. The OIDC TEST handler (`:247-300`) is UNTOUCHED. |
| `internal/admin/settings.go` (`PutSettings`, `:19-26`) | route an incoming `oidc_client_secret` to `h.store.SetOIDCSecret` + strip it from the plaintext `SetSettings` map. All other keys pass through. |

**MODIFY — serial-slot route registration (additive only):**

| File | Change |
|---|---|
| `internal/server/routes_admin.go` | ADD the 8 route lines (§1.8). NOTHING else changes. ONE commit. SERIAL SLOT — only holder while live; LAST chain holder (release to nobody, or to w7-prov-oauth's trivial append per MAP §222-224). |

**MODIFY — e2e mock corrections (mirror real Go, decision 1):**

| File | Change |
|---|---|
| `ui/e2e/mocks/handlers/translator.ts` (BODY) | Verify `load`→`{file,payload}` (payload contains `"gpt-4o"`) + `translate`→`{payload,...,translated:true}` mirror the Go DTOs. Likely verification-only. DO NOT add save/send. |
| `ui/e2e/mocks/handlers/nodes.ts` (BODY — test/availability branches ONLY) | Verify `/api/models/test`→`{ok,latency_ms}` + `/api/models/availability`→`{available:[{id,available}]}` mirror Go. DO NOT touch the provider-node CRUD branches (w7-platnodes-owned, FROZEN). |
| `ui/e2e/mocks/handlers/models.ts` (BODY — custom branches ONLY) | Verify `/api/models/custom` GET/POST + `/api/models/custom/{id}` DELETE mirror the Go `customModelDTO`. DO NOT touch `/api/models` or `/api/models/disabled` branches (FROZEN). |

**FORBIDDEN:** everything else. Explicitly: all pre-existing `internal/admin/*.go` EXCEPT
the NEW console/translator/models files + the additive `handlers.go` setters + the THREE
sanctioned OIDC-secret redirects (auth.go/oidc.go/settings.go); the OIDC TEST handler
(`oidc.go:247-300`); `disabledmodels.go`, `nodes.go` (provider-node CRUD), `version.go`,
login/logout/me; `internal/admin/audit.go` (REUSE `h.recordAudit` read-only); all other
`internal/store/*.go` except custommodels/oidcsecret (NEW) + migrate (additive); all
other `internal/logging/*` except the NEW console.go; `internal/translation/*` (REUSE the
registry, no edit); `ui/e2e/mocks/fixture.ts` (FROZEN); the shared mock `utils.ts`/index/
seed-index/`store.ts`; the `chat-sessions.ts`/`logs.ts` mocks (DEFER/unrelated); all UI
`ui/src/**` (FROZEN, decision 8); `ui/package.json` + lockfile; `ui/vite.config.ts`;
`ui/playwright.config.ts`; and **`ui/dist/index.html` MUST NOT be reverted** (e2e-hygiene).
Touching any of these is an automatic REJECT.

---

## 4. TDD tasks

Cadence (strict, AGENTS.md "TDD always"): **no Go impl file may exist before its
`_test.go` is committed RED.** `go test ./... && go vet ./... && go build ./...` green at
EVERY commit (RED test commits fail only the new package's targeted run; prefer table
tests that fail on assertion, not compile). The three e2e specs stay green throughout
(real Go is additive; mock corrections mirror it; the console fixture is FROZEN). The four
domains are independent — order is console → translator → models → oidc-secret, then the
single serial-slot routes commit, then mock verifications + closeout.

### T-console — STEP(a) RED, STEP(b) impl
STEP(a): write `internal/logging/console_test.go` + `internal/admin/console_test.go`
(`newTestEnv` + injected `ConsoleLog`; assert the SSE frame field names). Run targeted →
FAIL. Commit RED: `phase-1/w7-misc: failing console-log capture + SSE tests (TDD red)`.
STEP(b): implement `internal/logging/console.go` + `internal/admin/console.go` (mirror
`usagestream.go`) + the additive `SetConsoleLog` setter + the minimal `main.go`
MultiWriter wiring. Gates: `go test ./... && go vet ./... && go build ./...` green.
Commit: `phase-1/w7-misc: console-log ring buffer + SSE stream endpoint`.

### T-translator — STEP(a) RED, STEP(b) impl
STEP(a): write `internal/admin/translator_test.go` (load contains `gpt-4o`; translate
returns `translated:true` + a transformed payload; bad payload→400). Run targeted →
FAIL. Commit RED: `phase-1/w7-misc: failing translator load/translate tests (TDD red)`.
STEP(b): implement `internal/admin/translator.go` over `translation.Registry.
TranslateRequest` (+ optional `SetTranslationRegistry` setter). Gates green. Commit:
`phase-1/w7-misc: translator load + translate over translation registry`.

### T-models — STEP(a) RED, STEP(b) impl
STEP(a): write `internal/store/custommodels_test.go` + `internal/admin/models_test.go`
(fake `ModelProber`; test/availability/custom-CRUD; audit-on-create); add the
`custom_models` table to `migrate.go`. Run targeted → FAIL. Commit RED:
`phase-1/w7-misc: failing models test/availability/custom tests (TDD red)`.
STEP(b): implement `internal/store/custommodels.go` + `internal/admin/models.go` (+ the
additive `SetModelProber` setter; reuse `h.recordAudit`). Gates green. Commit:
`phase-1/w7-misc: models test/availability + custom-model store & admin`.

### T-oidc-secret — STEP(a) RED, STEP(b) impl
STEP(a): write `internal/store/oidcsecret_test.go` (encrypt round-trip; not-plaintext;
legacy-plaintext migration encrypt+blank) + extend `internal/admin/oidc_test.go`'s env
expectations IF needed (prefer asserting via the store test + a new admin-level test that
configuring the secret via `PUT /api/settings` then reading `GetSettings` shows NO
plaintext `oidc_client_secret`, while `oidcConfigured` is true). Add the `oidc_secret`
table to `migrate.go`. Run targeted → FAIL. Commit RED:
`phase-1/w7-misc: failing OIDC secret-at-rest tests (TDD red)`.
STEP(b): implement `internal/store/oidcsecret.go` + the THREE sanctioned redirects
(auth.go `oidcConfigured`, oidc.go callback, settings.go `PutSettings`). Gates: full
`go test ./...` green (the existing oidc_test.go must STAY green — verify the redirect
preserves login-mode gating + callback exchange). Commit:
`phase-1/w7-misc: OIDC client secret encrypted at rest (oidc_secret_enc)`.

### T-routes — serial-slot route registration
TAKE the serial slot (orchestrator confirms FREE at P5). Add the 8 route lines to
`routes_admin.go` (§1.8). Gates: `go test ./... && go vet ./... && go build ./...` green.
Commit (ONE commit touches the serial file):
`phase-1/w7-misc: register console/translator/models admin routes (serial slot)`.

### T-mocks — mock-body verifications (mirror real Go, decision 1)
Verify `translator.ts` (load/translate shapes), `nodes.ts` (test/availability branches),
`models.ts` (custom branches) mirror the Go DTOs. Gates: `cd ui && npm run build` green;
isolated `npx playwright test e2e/console.spec.ts e2e/translator.spec.ts e2e/models.spec.ts`
green (still). If a correction reds a non-w7-misc spec, STOP + ESCALATE (§8 ESC-MOCK).
Commit (skip if no body change needed — record "verified, no change" in WORKFLOW.md):
`phase-1/w7-misc: verify translator/nodes/models mocks mirror real Go DTOs`.

### T-close — full gates + closeout
```bash
go test ./... && go vet ./... && go build ./...
go test ./internal/admin/ -run 'Console|Translator|Models|OIDC' -v
go test ./internal/store/ -run 'CustomModel|OIDC' -v
go test ./internal/logging/ -run 'Console' -v
cd ui && npm run build
pkill -f 'chromium|vite preview' 2>/dev/null ; true
cd ui && npx playwright test e2e/console.spec.ts e2e/translator.spec.ts e2e/models.spec.ts   # green
pkill -f 'chromium|vite preview' 2>/dev/null ; true
cd ui && npx playwright test                                                                 # full suite green (no regressions)
cd ui && npx vitest run src/                                                                 # unaffected, green
```
Flip `.planning/parity/matrix/9router-ui.md`: PAR-UI-017 (console), PAR-UI-018/086
(translator), PAR-UI-117/119/120 (custom/test/availability) → variant-HAVE→true-HAVE
(real Go, cite §1.1/§1.5/§1.6). Mark `open-questions.md` w6-i ESC-1/ESC-2, w6-f ESC-3,
w6-j ESC-4 RESOLVED with a cite to this plan; keep w6-i ESC-1a (chat-sessions) OPEN with
the DEFER rationale; append the translator save/send DEFER follow-up. Update
`docs/WORKFLOW.md` (P6 base observation; the ESC-CONSOLE-SRC MultiWriter decision; the
ESC-TRANS-SCOPE save/send defer; the ESC-MODELS-TEST injectable-prober decision; the
ESC-OIDC-READSITE three-redirect decision + the legacy-plaintext migration; the
ESC-CHATSESS defer; the serial-slot take-from-prior / LAST-holder release). Final commit:
`phase-1/w7-misc: close — console/translator/models Go + OIDC secret-at-rest; matrix flip; W6-misc cluster complete`.
**On the close commit, RELEASE the routes_admin.go serial slot (w7-misc is the LAST chain
holder — release to nobody, or to w7-prov-oauth's trivial /api/oauth append per MAP
§222-224).**

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0. Diff gate is
**w7-misc commit-range-scoped** (§7).

**Test gates**
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/admin/ -run 'Console|Translator|Models|OIDC' -v` → exit 0, all
  pass (console SSE frame + nil-501; translator load-gpt4o + translate-translated +
  400; models test/availability/custom-CRUD ≥6 incl audit-on-create; OIDC no-plaintext +
  oidcConfigured-true).
- `go test ./internal/store/ -run 'CustomModel|OIDC' -v` → exit 0 (custom CRUD; OIDC
  encrypt round-trip + not-plaintext + legacy migration).
- `go test ./internal/logging/ -run 'Console' -v` → exit 0 (ring bound + subscribe frame
  + slow-drop + unsubscribe).
- `cd ui && npx playwright test e2e/console.spec.ts e2e/translator.spec.ts e2e/models.spec.ts`
  → exit 0, all pass (3 console + 4 translator + 3 models), 0 skipped. (Run ISOLATED;
  kill stale chromium/vite-preview first; NEVER revert ui/dist/index.html.)
- `cd ui && npx playwright test` → exit 0, no spec green-at-base goes red.
- `cd ui && npm run build` → exit 0. `cd ui && npx vitest run src/` → exit 0.

**TDD-order proof** — each impl file's covering test appears in an earlier-or-equal commit:
```bash
for pair in \
  "internal/logging/console_test.go:internal/logging/console.go" \
  "internal/store/custommodels_test.go:internal/store/custommodels.go" \
  "internal/store/oidcsecret_test.go:internal/store/oidcsecret.go" \
  "internal/admin/console_test.go:internal/admin/console.go" \
  "internal/admin/translator_test.go:internal/admin/translator.go" \
  "internal/admin/models_test.go:internal/admin/models.go" ; do
  tf=${pair%%:*}; im=${pair##*:}
  ct=$(git log --format=%ct --diff-filter=A -1 -- "$tf")
  cf=$(git log --format=%ct --diff-filter=A -1 -- "$im")
  [ "$ct" -le "$cf" ] || echo "TDD VIOLATION: $im"     # prints nothing
done
```

**Grep proofs (per domain)**
```bash
# console
grep -n "func (h \*Handlers) ConsoleLogStream" internal/admin/console.go
grep -n "text/event-stream\|SetBodyStreamWriter\|data: " internal/admin/console.go          # SSE shape
grep -n "func NewConsoleLog\|func (c \*ConsoleLog) Append\|Subscribe\|Recent" internal/logging/console.go
grep -n "level\|message\|timestamp" internal/admin/console.go                               # frame field names
# translator
grep -n "func (h \*Handlers) TranslatorLoad\|TranslatorTranslate" internal/admin/translator.go
grep -n "TranslateRequest\|translated" internal/admin/translator.go                          # registry reuse + marker
! grep -nE '/api/translator/(save|send)|TranslatorSave|TranslatorSend' internal/ && echo "no deferred translator routes OK"
# models
grep -n "func (h \*Handlers) TestModel\|ModelAvailability\|ListCustomModels\|CreateCustomModel\|DeleteCustomModel" internal/admin/models.go
grep -n "ResolveModel\|ModelsFor\|ModelProber" internal/admin/models.go                      # catalog + injectable prober
grep -n "func (s \*Store) CreateCustomModel\|ListCustomModels\|DeleteCustomModel" internal/store/custommodels.go
# oidc-secret
grep -n "func (s \*Store) GetOIDCSecret\|SetOIDCSecret" internal/store/oidcsecret.go
grep -n "s.cipher.Encrypt\|s.cipher.Decrypt" internal/store/oidcsecret.go                     # at-rest
grep -n "GetOIDCSecret" internal/admin/auth.go internal/admin/oidc.go                         # the redirects
grep -n "SetOIDCSecret" internal/admin/settings.go                                            # the write redirect
# routes
grep -nE '/api/console-logs/stream|/api/translator/(load|translate)|/api/models/(test|availability|custom)' internal/server/routes_admin.go
# no init(); no new global state
! grep -rn "func init(" internal/admin/console.go internal/admin/translator.go internal/admin/models.go internal/logging/console.go internal/store/custommodels.go internal/store/oidcsecret.go && echo "no init() OK"
grep -n "func New(" internal/admin/handlers.go ; echo "^ signature MUST be unchanged from <base>"
```

**No-secret-exposure proofs (binding)**
```bash
# OIDC secret never echoed: no handler returns the decrypted secret or the enc column
! grep -rnE 'oidc_secret_enc|GetOIDCSecret' internal/admin/*.go | grep -iE 'writeData|json:' && echo "no oidc secret in any response OK"
# after migration, GetSettings no longer carries plaintext oidc_client_secret (asserted in test)
# additive migrations only (no DROP/RENAME introduced by this plan)
git diff <base>..HEAD -- internal/store/migrate.go | grep -E '^\+' | grep -iE 'DROP COLUMN|RENAME COLUMN|DROP TABLE' | wc -l   # = 0
# console/translator/custom carry no secret fields
grep -n "_enc" internal/store/custommodels.go ; echo "^ expect EMPTY (no secret fields in custom models)"
# the OIDC TEST handler is UNTOUCHED (body-provided secret, not the stored one)
git diff <base>..HEAD -- internal/admin/oidc.go | grep -E '^[-+]' | grep -iE 'ProbeOIDCClientSecret|body.ClientSecret|stored OIDC secrets' | wc -l   # = 0 (test handler region unchanged)
```
Plus a runtime no-leak assertion in `oidcsecret_test.go`: read the raw `oidc_secret_enc`
column and assert it is NEITHER the cleartext secret NOR empty after a `SetOIDCSecret`.

**Negative / freeze proofs (w7-misc commit-range — §7)**
```bash
R="<first-w7-misc>^..<last-w7-misc>"
# Only the sanctioned Go files changed:
git diff $R --name-only -- internal/ cmd/ | grep -vE \
 'internal/logging/console(_test)?\.go|internal/store/(custommodels|oidcsecret)(_test)?\.go|internal/store/migrate\.go|internal/admin/(console|translator|models)(_test)?\.go|internal/admin/handlers\.go|internal/admin/auth\.go|internal/admin/oidc\.go|internal/admin/settings\.go|internal/server/routes_admin\.go|cmd/g0router/main\.go' \
 | wc -l                                                                  # = 0
# Frozen admin handlers untouched (the OIDC TEST handler + disabledmodels + nodes + version + audit):
git diff $R --name-only -- internal/admin/disabledmodels.go internal/admin/nodes.go internal/admin/version.go internal/admin/audit.go | wc -l   # = 0
# auth.go / oidc.go / settings.go changed ONLY for the sanctioned redirects (manual review: diff is the 3 redirects + nothing else):
git diff $R -- internal/admin/auth.go internal/admin/oidc.go internal/admin/settings.go | grep -E '^\+' | grep -ivE 'GetOIDCSecret|SetOIDCSecret|oidc_client_secret' | grep -vE '^\+\+\+' | wc -l   # ~0 (only the redirect lines)
# Frozen translation pkg untouched (reused, not edited):
git diff $R --name-only -- internal/translation/ | wc -l                 # = 0
# UI is frozen except the sanctioned mock bodies; fixture untouched:
git diff $R --name-only -- ui/ | grep -vE \
 'ui/e2e/mocks/handlers/(translator|nodes|models)\.ts' | wc -l           # = 0
git diff $R --name-only -- ui/e2e/mocks/fixture.ts | wc -l               # = 0 (fixture frozen)
git diff $R --name-only -- ui/src/ | wc -l                               # = 0 (src frozen)
# No deferred surfaces built:
! git diff $R --name-only | grep -E 'chat-sessions|chatsessions' && echo "no chat-sessions Go OK"
# routes_admin.go = exactly ONE commit, additive:
git log --oneline $R -- internal/server/routes_admin.go | wc -l          # = 1
git diff $R -- internal/server/routes_admin.go | grep -E '^-' | grep -v '^---' | wc -l   # = 0 (no deletions)
```

---

## 6. Out of scope (restated, binding)

No UI src edits (decision 8 — pages/components/routes/stores frozen); only the sanctioned
translator/nodes/models mock-body verifications. No `fixture.ts` edit (console
MockEventSource + traffic-stream branches frozen). No edits to pre-existing admin handlers
EXCEPT the three sanctioned OIDC-secret redirects (auth.go `oidcConfigured`, oidc.go
callback `ClientSecret` source, settings.go `PutSettings` write redirect) — the OIDC TEST
handler, disabledmodels, nodes (provider-node CRUD), version, login/logout/me, audit are
FORBIDDEN. No `internal/translation` edit (reuse the registry). No `New(...)` signature
change / no new global state (additive setters only). No destructive DDL — additive
`ensureTable`/`ensureColumn` only. **DEFERRED (do NOT build): translator save/send,
chat-sessions CRUD.** No secret exposure (OIDC secret encrypted/never echoed; console/
translator/custom carry no secrets). Mock-vs-Go contradiction → escalate (§8), never
fudge a mock or edit a frozen handler/fixture.

## 7. Diff-gate scope

W7 plans commit to main concurrently, so a broad `<base>..HEAD` range sweeps in sibling
commits. The diff gate MUST be scoped to w7-misc's own commits. Isolate them:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w7-misc:" | awk '{print $1}'`
then `git diff <first-w7-misc>^..<last-w7-misc> -- [file list]`.

`git diff <first>^..<last> --name-only` must be exactly a subset of:
```
internal/logging/console.go
internal/logging/console_test.go
internal/store/custommodels.go
internal/store/custommodels_test.go
internal/store/oidcsecret.go
internal/store/oidcsecret_test.go
internal/store/migrate.go              (additive tables; ONE commit per domain ok)
internal/admin/console.go
internal/admin/console_test.go
internal/admin/translator.go
internal/admin/translator_test.go
internal/admin/models.go
internal/admin/models_test.go
internal/admin/handlers.go             (additive setters; NO New() sig change)
internal/admin/auth.go                 (ONLY the oidcConfigured secret-check redirect)
internal/admin/oidc.go                 (ONLY the callback ClientSecret source redirect; TEST handler untouched)
internal/admin/settings.go             (ONLY the PutSettings oidc-secret write redirect)
internal/admin/oidc_test.go            (additive assertions if needed — additive only)
cmd/g0router/main.go                   (additive ConsoleLog + MultiWriter wiring)
internal/server/routes_admin.go        (serial-slot additive routes; ONE commit)
ui/e2e/mocks/handlers/translator.ts    (body only — load/translate mirror Go)
ui/e2e/mocks/handlers/nodes.ts         (body only — test/availability branches; node CRUD untouched)
ui/e2e/mocks/handlers/models.ts        (body only — custom branches; /api/models + disabled untouched)
.planning/parity/matrix/9router-ui.md
.planning/parity/plans/open-questions.md
docs/WORKFLOW.md
```
Any file outside this list in the scoped diff is an automatic review REJECT. The OIDC TEST
handler region of `oidc.go`, `disabledmodels.go`, `nodes.go`, `version.go`, `audit.go`,
`internal/translation/*`, `ui/e2e/mocks/fixture.ts`, and all `ui/src/**` are deliberately
ABSENT — touching them is an automatic REJECT. The `routes_admin.go` edit must appear in
exactly ONE commit (§5) and the serial slot is released as the LAST chain holder.

## 8. Escalations / decisions (explicit — recommended defaults, do not fabricate)

- **ESC-CONSOLE-SRC (RESOLVED at authoring — what feeds the console stream, binding
  default).** `internal/logging/` has NO log buffer; the server logs via `log.Printf` to
  stderr (`main.go:117,136,145`). **Decision:** ADD a small in-process `ConsoleLog` ring
  buffer + broadcaster (`internal/logging/console.go`) and capture the existing `log`
  output via `log.SetOutput(io.MultiWriter(os.Stderr, consoleWriter))` in `main.go`
  (additive forward, mirroring the w6-j ESC-1 setter pattern). The SSE handler replays
  `Recent()` then streams `Subscribe()` frames. Hermetic test drives `Append` directly
  (no `main.go` in the test path). If the `main.go` capture forces a non-additive change,
  fall back to exposing `ConsoleLog` for explicit push call-sites (endpoint still works).
  RECOMMENDED: the MultiWriter capture. Flag for orchestrator confirmation.
- **ESC-TRANS-SCOPE (RESOLVED at authoring — translator endpoint set, binding default).**
  The mock + page consume ONLY `load` + `translate` (`translator.ts:21,31`;
  `translator.spec.ts`). `save`/`send` have NO consumer. **Decision:** BUILD load +
  translate (over `Registry.TranslateRequest`); DEFER save/send (dead code otherwise).
  `send` additionally overlaps the real gateway request path and needs its own scope
  decision. Recorded as a follow-up in `open-questions.md`. RECOMMENDED; flag.
- **ESC-MODELS-TEST (RESOLVED at authoring — `/api/models/test` reachability, binding
  default).** `test` must be hermetic. **Decision:** an injectable `ModelProber` interface
  (mirror `NodeProber`, `handlers.go:107`); the default impl does a best-effort
  reachability check, tests inject a fake returning `{ok,latency_ms}`. `availability` +
  `custom` are pure catalog/store reads (no prober). RECOMMENDED; flag.
- **ESC-OIDC-READSITE (RESOLVED at authoring — the three sanctioned redirects, binding
  default).** Encrypting `oidc_client_secret` at rest requires the two production READ
  sites (`auth.go:36` `oidcConfigured`, `oidc.go:182` callback) + the WRITE site
  (`settings.go` `PutSettings`) to consult the encrypted accessor. **Decision:** these
  THREE surgical edits are the explicit OIDC-secret-handling exception to the
  frozen-handler rule (§3/§6); the OIDC TEST handler (`oidc.go:247-300`, body-provided
  secret) is UNTOUCHED. If any redirect reds the existing `oidc_test.go`, the redirect is
  wrong — fix the redirect, never weaken the test. RECOMMENDED; flag.
- **ESC-OIDC-STORE / ESC-OIDC-MIGRATE (RESOLVED at authoring — storage shape + legacy
  migration).** **Decision:** a dedicated single-row `oidc_secret` table with an
  `oidc_secret_enc` column (singleton + `*_enc` precedent), plus an idempotent first-read
  migration that encrypt-moves the legacy plaintext `settings["oidc_client_secret"]` then
  BLANKS it (so the plaintext never persists and `GetSettings` no longer echoes it). The
  `ensureColumn`-on-settings alternative is documented but rejected (mixing a secret into
  the plaintext flat map is the very thing we're fixing). RECOMMENDED; flag.
- **ESC-CHATSESS (RESOLVED at authoring — DEFER, binding).** 9router keeps chat sessions
  in localStorage (open-questions w6-i ESC-1a). No g0router row requires server-side
  persistence; the chat turn uses the real `/v1/chat/completions` gateway. **Decision:**
  DEFER `/api/chat-sessions` CRUD — building it is dead code. The `chat-sessions.ts` mock
  stays. Recorded OPEN with this rationale. If a future feature genuinely needs
  server-side session history, a follow-up plan builds it (table + CRUD + routes).
- **ESC-MODELS-ROUTE / ESC-MODELS-MOCK (CONDITIONAL).** Route precedence: the new static
  `/api/models/{test,availability,custom}` siblings vs the existing static `/api/models/
  disabled` + the new `/api/models/custom/{id}` (param) — all static-before-param; no
  expected collision. If `fasthttp/router` mis-disambiguates, STOP + ESCALATE (never a
  silent path change). Mock ripple: `nodes.ts`/`models.ts` are shared — w7-misc edits ONLY
  the test/availability + custom branches. If a correction reds a non-w7-misc spec, STOP +
  ESCALATE for orchestrator serialization (ESC-MOCK).
- **Serial-slot dependency (§1.8 / P5).** w7-misc TAKES the routes_admin.go slot LAST in
  the chain (MAP §219-224) after the prior holder releases it, and RELEASES to nobody (or
  to w7-prov-oauth's trivial /api/oauth append). Orchestrator confirms exactly one
  unmerged holder (decision 3) before T-routes.
- **No other blocking dependency.** All reused surfaces (usagestream SSE, translation
  registry, catalog, cipher, recordAudit, newTestEnv, additive-setter + migrate patterns)
  are in-tree at <base>. w7-misc is unblocked once the serial slot is free.
```
