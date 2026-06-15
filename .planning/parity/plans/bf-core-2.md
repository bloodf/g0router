# Micro-plan bf-core-2 — Semantic cache (g0router-shaped) (core, Go)

```
program: bifrost-parity (Waves 6+; CLI_ORCHESTRATOR.md governs, HANDOFF.md retired)
plan: bf-core-2
status: READY (rev 1 — authored against merged Waves 0–7 tree, live tree;
  BIFROST-MAP.md ledger row bf-core-2 §306; core disposition table §273-289;
  architectural decision #8 §161-177; routes_admin serial chain §343-346;
  freeze rules §384-399. AUTHORITATIVE design = docs/phases/phase-19-advanced-features.md:60-69,
  g0router's OWN roadmap — NOT a guessed Bifrost vector shape.)
runs: core track. Greenfield-disjoint domain (internal/semcache/ does NOT exist).
  Runs ∥ everything EXCEPT the routes_admin.go serial chain (see go-serial-slot).
branch: main (direct commits, per AGENTS.md "No PR workflow")
commit-prefix: phase-1/bf-core-2: (matches the shipped bifrost chain prefix — verified
  in git log: `phase-1/bf-openai-4: ...`, `phase-1/bf-gov-3: ...`)
footer: Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>
ref-source: BLOCKED — ESC-REF-ABSENT (BIFROST-MAP §47-68). The frozen Bifrost ref
  @ ca21298 is ABSENT on this host. Bifrost's semantic-cache rows (034/035/036) cite
  `plugins/semanticcache/main.go` — UNREADABLE. Per BIFROST-MAP §280, bf-core-2 does
  NOT build to Bifrost's vector-plugin shape; it builds to g0router's OWN already-planned
  design in docs/phases/phase-19-advanced-features.md:60-69. bf-core-2 ships ONLY the
  deterministic EXACT-KEY-HASH half (SQLite semantic_cache table, [flag: semantic_cache],
  non-streaming chat only, after guardrails); the SEMANTIC-SIMILARITY half (cosine-in-Go
  over embeddings, ≤500 candidates, threshold 0.95) is DEFERRED — not built — pending the
  operator embedding-source decision (D2). STOP-escalate on any detail not in phase-19.
  NEVER build to a guessed Bifrost wire format or invent a vector-backend pipeline.
go-serial-slot: routes_admin.go — bf-core-2 is the **LAST HOLDER** of the
  bf-mcp-1 → bf-mcp-2 → bf-core-2 serial chain (BIFROST-MAP §343-346). It appends
  the /api/cache/semantic GET+DELETE routes (append-only, static collection) and
  **RELEASES to NOBODY** (chain terminus). bf-core-2 must NOT start its routes_admin.go
  edit until bf-mcp-2 has merged and released the slot.
new-route: YES — `GET /api/cache/semantic` + `DELETE /api/cache/semantic`
  (per phase-19:50-51), {data,error} snake_case envelope.
chat-hook: YES — bf-core-2 adds a flag-gated read-through/write-through cache hook
  in internal/api/chat.go's NON-STREAMING path only, injected via an additive setter
  (mirrors chat.SetVKGate / SetUsageRecorder). Flag OFF ⇒ clean no-op (hook nil).
```

---

## 0. Objective + ground truth

### 0.1 Objective

Build g0router's **own** phase-19 semantic cache as a greenfield-disjoint domain
package, satisfying the Bifrost *concept* (PAR-BF-CORE-034/035/036) without
Bifrost's external vector backends (037/038 = ESC by design).

Per the **honest-scoping decision (D1/D2, below)**, bf-core-2 ships ONLY the
deterministic half and DEFERS the semantic half entirely:

1. **EXACT-KEY-HASH cache — HAVE (live, deterministic, fully hermetic). THE ONLY
   THING BUILT.** A `sha256(normalized prompt + model)` key → O(1) SQLite lookup.
   On a hit (non-expired, same model), the cached `response_json` is returned and
   the provider call is **short-circuited** (proven by test). On a miss, the
   response is written through. Flag-gated `[semantic_cache]`; non-streaming
   `/v1/chat/completions` only; positioned where a guardrail check would sit (D5).

2. **COSINE / SEMANTIC half — DEFERRED, NOT BUILT (D2).** The cosine engine, the
   `Embedder` interface/seam, the ≤500-candidate loader, and the semantic branch
   in `Lookup` are **NOT built in bf-core-2**. Rationale: the production embedder
   has no funded source decision (phase-19:65 says "existing connection" but not
   which; ESC-REF-ABSENT blocks the Bifrost shape), so a semantic path built now
   would be **production-inert** (nil embedder ⇒ the branch never executes in
   production; only a fake-embedder test would exercise it). The parity outcome is
   IDENTICAL either way — 034/035/036 flip to PARTIAL whether or not the cosine
   machinery exists now, because the semantic lookup is not live in production
   regardless. Building it now adds inert code for ZERO parity gain. Deferring the
   whole semantic half costs nothing and lets the future embedder-wiring plan build
   AND test the cosine path END-TO-END against the real (operator-decided) embedder
   rather than a fake.

**Forward-compatibility kept (not inert):** the `semantic_cache` table retains its
`embedding_json` column (D4) — write-through stores `[]` for now; an empty column
default is data, not executable dead code, and it lets the deferred semantic plan
land without a migration. That is the ONLY accommodation; no cosine code, no
embedder seam, no candidate loader ships.

Additive-only: a NEW `internal/semcache/` domain package, a NEW
`internal/store/semcache.go` + `semantic_cache` table (additive `ensureTable`), a
NEW `internal/admin/cache.go` (GET/DELETE), and a flag-gated hook in
`internal/api/chat.go` injected via an additive setter. NO `NewChatHandler`
signature change, NO `init()`, NO global state, NO destructive DDL.

### 0.2 Rows this plan closes (matrix = ground truth)

| Row | Behavior (matrix) | Disposition | Acceptance after bf-core-2 |
|---|---|---|---|
| PAR-BF-CORE-034 | Semantic cache plugin with dual-path lookup: direct hash + semantic similarity | **BUILD (hash half only)** | Direct-hash path LIVE (exact-key read/write-through, short-circuits provider — D1). The SEMANTIC-SIMILARITY half (cosine over embeddings) is **DEFERRED — not built** (D2), pending the operator embedding-source decision (it would be production-inert until a live embedder is wired, so it is deferred to the plan that wires the embedder and tests it end-to-end). MISSING → **PARTIAL** (hash half HAVE; semantic half deferred). |
| PAR-BF-CORE-035 | Semantic cache config: Provider, EmbeddingModel, TTL, Threshold, Dimension | **BUILD (g0router-shaped, TTL/Threshold only)** | g0router config via settings store (D6): `semantic_cache_threshold` (default 0.95, reserved for the deferred semantic plan), `cache_ttl_seconds`; flag `semantic_cache`. Provider/EmbeddingModel/Dimension are **out of scope now** — they belong to the deferred semantic-similarity plan (D2). MISSING → **PARTIAL** (TTL/Threshold/flag HAVE; Provider/EmbeddingModel/Dimension deferred). |
| PAR-BF-CORE-036 | Streaming response accumulation with background reaper and TTL bookkeeping | **BUILD (g0router-shaped, no streaming)** | Streaming cache is OUT by phase-19:68 design (non-streaming chat ONLY — D7); the "stream accumulation" sub-behavior is **VAR/ESC** (g0router does not cache streamed responses). TTL bookkeeping IS built: `expires_at` column + **lazy purge on read** (phase-19:69 "Expired rows purged lazily") — NO background reaper goroutine (g0router has no global state / no init; D8). MISSING → **PARTIAL** (TTL + lazy purge HAVE; stream accumulation + background reaper = VAR/ESC). |
| PAR-BF-CORE-037 | `VectorStore` abstraction (Ping/CreateNamespace/GetNearest/Add/Delete) | **ESC** | g0router uses SQLite + Go cosine by design (phase-19:61; BIFROST-MAP §281). NO VectorStore interface. Recorded §3. |
| PAR-BF-CORE-038 | Vector store backends: Weaviate, Redis, Qdrant, Pinecone | **ESC** | External vector backends out of scope by design (phase-19:61). Recorded §3. |

**Honest scoping note:** 034 closes only to **PARTIAL** — the exact-key half is
genuinely HAVE (live + short-circuits), the cosine/semantic half is **deferred —
not built** (D2). 035/036 close to PARTIAL for the same reason. 037/038 are ESC by
g0router's deliberate SQLite+Go design. No row is closed by inventing un-evidenced
Bifrost vector behavior, and NO production-inert code ships. The residual (the
whole semantic-similarity half, streaming, background reaper) is recorded in
`open-questions.md` (§7).

### 0.3 Preconditions already satisfied (evidence — read 5 files, AGENTS.md)

- **Chat non-streaming path + the hook insertion point** (`internal/api/chat.go`):
  `ChatHandler.Handle` (`chat.go:318`) resolves the provider/key
  (`ResolveForModel`, `:358`), runs the **x-g0-vk gate** (`:364-384`), the
  pending-tracker (`:386-389`), the stream decision (`:391-402`), then for the
  NON-STREAM branch calls `provider.ChatCompletion(gatewayCtx, key, &req)`
  (`:461`). **The cache read-through hook sits immediately before `:461`
  (after VK gate, after stream decision confirms non-stream); the write-through
  sits immediately after a successful `resp` (`:486-488`), before/around
  `recordNonStream` (`:495`).** The handler already uses ADDITIVE setters
  (`SetVKGate :159`, `SetUsageRecorder :148`, `SetPendingTracker :152`,
  `SetDetailCapture :156`) constructed from `NewChatHandler(router_)` (`:132`) —
  bf-core-2's `SetSemanticCache` mirrors this EXACTLY. `NewChatHandler` signature
  is NOT changed.
- **GUARDRAILS DO NOT RUN INLINE IN CHAT (critical finding — D5).** Verified:
  `grep -rln "Evaluate|Guardrail|Blocklist" internal/api/` returns NOTHING. The
  `GuardrailEngine.Evaluate` (`internal/governance/guardrails.go:44`) is a pure
  evaluator consumed ONLY by the admin `POST /api/guardrails/test` route
  (`routes_admin.go:165`) and the config CRUD (`:163-164`). **There is no live
  guardrail check in the chat request path today.** Consequence (D5): "after
  guardrails" is satisfied **positionally** — the cache hook is placed exactly
  where a guardrail check WOULD sit (after VK gate, before dispatch), so when a
  future plan wires `Evaluate` into chat.go inline, the cache is already correctly
  ordered after it. Because no guardrail runs inline now, there is no current
  blocked-prompt-served-from-cache risk; this is documented honestly, NOT papered
  over with a fabricated guardrail call.
- **Feature-flag store EXISTS** (`internal/store/featureflags.go`): `FeatureFlag
  {ID,Key,Enabled,Description,CreatedAt}` (`:9-16`); `ListFeatureFlags` (`:19`),
  `GetFeatureFlagByID` (`:43`), `SetFeatureFlagEnabled(id,enabled)` (`:50`). The
  `feature_flags` table (`migrate.go:158-164`): `id, key UNIQUE, enabled INTEGER
  DEFAULT 0, description, created_at`. **There is NO `GetFeatureFlagByKey`** —
  flags are looked up by ID only. bf-core-2 adds an additive
  `IsFeatureEnabled(key string) (bool, error)` (D9) so the hook can gate on the
  `semantic_cache` flag by its KEY string. The flag row itself is seeded additively
  (D9).
- **Settings store EXISTS** (`internal/store/settings.go`): `GetSetting(key)
  (string,error)` (`:33`), `SetSetting(key,value)` (`:46`) — backs
  `semantic_cache_threshold` + `cache_ttl_seconds` (D6).
- **Migrations are additive `ensureTable`** (`internal/store/migrate.go`): a
  `{name, CREATE TABLE IF NOT EXISTS ...}` slice (`:15-...`, e.g. `feature_flags`
  `:158`, `guardrails` `:174`, `proxy_pools` `:191`). bf-core-2 appends a
  `semantic_cache` entry to this slice (the exact phase-19:34-46 DDL incl. the two
  indexes). `ensureColumn` never alters/drops. NO destructive DDL.
- **Store CRUD template** (`internal/store/featureflags.go`, `guardrails.go`):
  `*Store` method receivers, `s.db.Query`/`QueryRow`/`Exec`, `scanX` helpers,
  `boolToInt`, `errors.Is(err, sql.ErrNoRows) → ErrNotFound`, `fmt.Errorf("ctx:
  %w")`. bf-core-2's `internal/store/semcache.go` mirrors this shape.
- **Admin handler template** (`internal/admin/featureflags.go`): `func (h
  *Handlers) X(ctx *fasthttp.RequestCtx)`, DTO structs with snake_case `json`
  tags, `writeData(ctx, status, data)` (`respond.go:19`), `writeError(ctx,
  status, message)` (`respond.go:23`), `h.recordAudit(ctx, action, target,
  details)` (`audit.go:64`). bf-core-2's `internal/admin/cache.go` mirrors this;
  DELETE is audited (phase-19:51 "clear (audited)").
- **Admin `Handlers` struct** (`internal/admin/handlers.go:18-38`) holds
  `store *store.Store`; `New(st,sessions,flows)` (`:42`). The cache handler reads
  the `semantic_cache` table via `h.store` — NO new field needed on `Handlers`
  (it already has `store`). The semcache store methods hang off `*store.Store`.
- **routes_admin.go LAST serial holder** (`internal/server/routes_admin.go`):
  static collection routes register straightforwardly (e.g.
  `r.GET("/api/feature-flags", ...)` `:150`; `r.GET("/api/guardrails", ...)`
  `:163`). bf-core-2 appends `GET`+`DELETE /api/cache/semantic` (a bare
  collection, no `{id}` — no static-vs-param precedence concern). Serial chain
  bf-mcp-1 → bf-mcp-2 → **bf-core-2 (terminus)** per BIFROST-MAP §343-346.
- **Embeddings are provider-routed, NOT an internal callable service**
  (`internal/api/embeddings.go`): `EmbeddingsHandler.Handle` resolves a provider
  via the router and calls `provider.Embedding(gatewayCtx, key, &req)` (`:100`).
  `schemas.EmbeddingRequest{Input,Model,...}` → `EmbeddingResponse{Data
  []Embedding{Embedding []float64}}` (`schemas/embedding.go`). There is NO
  internal "embed this prompt" function the cache could call without going through
  full provider/key resolution + a real network round-trip — **this is why the
  semantic-similarity half is deferred** (D2): there is no funded, hermetic
  embedding source today, so any cosine path built now would be production-inert.

---

## 1. Decisions made (and why) — binding

### D1 — Ship EXACT-KEY-HASH cache HAVE (live); it is the ONLY thing built

**Decision:** Ship the deterministic, fully-buildable half LIVE — and nothing else:
- **EXACT-KEY-HASH cache = HAVE.** `sha256(normalized prompt + model)` → O(1)
  SQLite lookup. Read-through short-circuits the provider on hit; write-through
  stores on miss. No embedding needed (free, deterministic). This is the
  Bifrost "direct hash path" (034) AND the matrix quirk #5 "direct-only mode"
  (`Provider=""`, lookup goes through the deterministic hash path only). Genuinely
  live, proven by a short-circuit test (§5). The `Cache` struct has NO embedder
  field, NO semantic branch — `Lookup` does exactly one thing: exact-key SQLite
  read (non-expired) → hit returns cached bytes + increments hit_count; miss
  returns no-hit. `Store` is the write-through.

**Why nothing else:** building the cosine/semantic machinery now would add
production-inert code for zero parity gain (D2). The honest, fully-live outcome is
the exact-key cache alone.

### D2 — Semantic-similarity half DEFERRED (not built); same parity, no inert code

**The embedding-source decision is the crux.** phase-19:65 says embeddings come
"via existing OpenAI-compatible provider connection; if no embedding-capable
connection configured → cache disabled." Verified against the live tree (§0.3):
there is NO internal embedder service — embedding requires full provider
resolution (`router.ResolveForModel`) + a real `provider.Embedding(...)` network
round-trip. The source connection/model is **un-decided** (phase-19 says "existing
connection" but not which; ESC-REF-ABSENT blocks the Bifrost shape), and a live
embedding call **cannot be proven hermetically** (no real embedding call allowed
in tests).

**Decision:** Build NONE of the semantic-similarity machinery in bf-core-2 — DEFER
it entirely to a future "semantic-similarity" plan gated on the embedding-source
decision. NOT built here: `internal/semcache/cosine.go` (Cosine + bestMatch), the
`Embedder` interface, any embedder field on `Cache`, `SetEmbedder`, the semantic
branch in `Lookup`, `store.LoadSemanticCandidates` (the ≤500 cosine candidate
loader), and all fake-embedder tests.

**Why defer rather than build-behind-a-nil-embedder:** a semantic path built now
would be **production-inert** — the production embedder would be nil by default
(no funded source), so the branch would NEVER execute in production; only a
fake-embedder test would exercise it. That is the tested-but-never-live pattern
this program rejects (inert team-budget accumulator, vestigial sync worker, dead
AllowFallbacks flag, dead interface methods). **The parity outcome is IDENTICAL
either way** — 034/035/036 flip to PARTIAL whether or not the cosine machinery
exists now, because the semantic lookup is not live in production regardless. So
building it now is inert code for zero parity gain; deferring the whole half costs
nothing and lets the future plan build AND test the cosine path END-TO-END against
the real (operator-decided) embedder rather than a fake. Recorded in
open-questions (§7).

**Forward-compatibility (data, not code):** the `semantic_cache` table keeps its
`embedding_json` column (D3) — write-through stores `[]` for now. An empty column
value is data, not executable dead code, and it lets the deferred semantic plan
land its cosine path without a migration.

### D3 — `semantic_cache` table: phase-19 DDL verbatim; exact-key lookup + lazy purge in the query

**Decision:** append to the `migrate.go` tables slice EXACTLY the phase-19:34-46
DDL (no invented columns):
```sql
CREATE TABLE IF NOT EXISTS semantic_cache (
    id INTEGER PRIMARY KEY,
    cache_key TEXT NOT NULL,            -- sha256(normalized prompt + model)
    embedding_json TEXT NOT NULL,
    model TEXT NOT NULL,
    response_json TEXT NOT NULL,
    expires_at DATETIME,
    hit_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_semantic_cache_model ON semantic_cache(model);
CREATE INDEX IF NOT EXISTS idx_semantic_cache_expires ON semantic_cache(expires_at);
```
(The two `CREATE INDEX` statements ride the same table entry's exec, or as
adjacent additive entries — both are `IF NOT EXISTS`, additive-only.)
Store methods (`internal/store/semcache.go`):
- `GetSemanticCacheByKey(cacheKey string, nowISO string) (*SemanticCacheEntry,
  error)` — exact-key lookup, **filters `expires_at IS NULL OR expires_at >
  now`** (non-expired only). Returns `ErrNotFound` on miss.
- `InsertSemanticCacheEntry(e SemanticCacheEntry) error` — write-through.
- `IncrementSemanticCacheHit(id int64) error` — `hit_count = hit_count + 1` on hit
  (test: hit_count increments — phase-19:110).
- `PurgeExpiredSemanticCache(nowISO string) (int64, error)` — lazy purge of expired
  rows (phase-19:69); called opportunistically on read (D8), NO background
  goroutine.
- `ListSemanticCacheEntries() / SemanticCacheStats()` — for the admin GET (keys,
  model, hits, expires — NOT full responses, phase-19:50).
- `ClearSemanticCache() error` — for the admin DELETE.

**NOT built:** `LoadSemanticCandidates` (the ≤500 cosine candidate loader) is the
semantic path's loader — DEFERRED with the rest of the semantic half (D2).

Mirror `featureflags.go` scan/`boolToInt`/`ErrNotFound` patterns. The
`embedding_json` column is retained for forward-compatibility (D2): write-through
stores `[]` (empty array) for the hash-only cache — the column is data the deferred
semantic plan will populate, not executable dead code. Write-through always stores
the key + response.

### D4 — "After guardrails" = positional placement (guardrails do not run inline today)

Per §0.3, NO guardrail check runs in chat.go's request path. **Decision:** place
the cache read-through hook at the exact point a guardrail check would sit — after
the x-g0-vk gate (`chat.go:384`) and after the stream decision confirms
non-streaming (`:391-402`), immediately before `provider.ChatCompletion` (`:461`).
This is the "after guardrails, before dispatch" position from phase-19:68. Because
no guardrail runs inline now, there is no current path by which a blocked prompt
could be served from / written to cache. When a future plan wires
`GuardrailEngine.Evaluate` into chat.go inline (the §7 open item), it MUST run
BEFORE this cache hook — documented here and in open-questions so the ordering
invariant is preserved. bf-core-2 does NOT fabricate an inline guardrail call.
The cache hook only runs on the NON-STREAM branch (D6).

### D5 — Config via settings store (g0router-shaped, not Bifrost's config struct)

phase-19:64 — "threshold 0.95 (single global setting `semantic_cache_threshold`)";
:69 — "TTL from `cache_ttl_seconds` setting". **Decision:** the cache reads
`cache_ttl_seconds` (TTL → `expires_at = now + ttl`; `0`/unset ⇒ no expiry / a
documented default) via the existing `store.GetSetting`. `semantic_cache_threshold`
(default `0.95`) is reserved for the deferred semantic plan (D2) — bf-core-2 reads
no threshold (the exact-key cache has no similarity step). The Bifrost
`Provider`/`EmbeddingModel`/`Dimension` config fields (035) are out of scope now —
they belong to the deferred semantic-similarity plan (D2, §7). No new config
table — settings store + the feature flag are the config surface.

### D6 — Non-streaming chat ONLY; streaming cache is OUT (VAR/ESC)

phase-19:68 — "Only non-streaming `/v1/chat/completions`". **Decision:** the cache
hook is invoked ONLY in chat.go's non-stream branch (after the `useStream` decision
resolves false). The streaming branch (`chat.go:418-459`) is UNTOUCHED — no cache
read, no write. Bifrost's "streaming response accumulation" (036) is therefore
VAR/ESC for g0router (recorded §3). Rationale to document: streamed responses are
incrementally framed SSE; caching them requires reassembly + replay framing that
phase-19 deliberately deferred. The cache also applies ONLY to `/v1/chat/
completions` — NOT `/v1/messages`, `/v1/responses`, `/v1/embeddings`, etc. (the
hook is wired into `ChatHandler` only).

### D7 — Lazy TTL purge on read; NO background reaper goroutine

phase-19:69 — "Expired rows purged lazily". **Decision:** expired rows are
excluded from reads by the `expires_at` filter (D3) and purged opportunistically
via `PurgeExpiredSemanticCache` called on the read path (e.g. once per lookup or
rate-limited). NO background reaper goroutine, NO `time.Ticker`, NO `init()` (the
Bifrost "background reaper" of 036 is ESC — g0router has no global state / no
init, AGENTS.md). The clock is injected (a `func() time.Time` on the `Cache`,
defaulting to `time.Now`) so TTL/expiry tests are hermetic (D-tests advance a
fixed clock — NO real `time.Now`, NO sleep). Mirrors the quota engine's injectable
`clock` (`quota.go:25`).

### D8 — Flag gating: additive `IsFeatureEnabled(key)` + seed the `semantic_cache` flag

The hook must be OFF by default and active only when `[flag: semantic_cache]` is
on. There is no `GetFeatureFlagByKey` today (§0.3). **Decision:** add an additive
`store.IsFeatureEnabled(key string) (bool, error)` (single-row `SELECT enabled
FROM feature_flags WHERE key = ?`; missing flag ⇒ `false, nil` — fail-OFF). Seed
the `semantic_cache` flag row additively (an idempotent `INSERT ... WHERE NOT
EXISTS` / `INSERT OR IGNORE` in the seed path that already seeds flags, or via the
migration seed step — match the existing flag-seed mechanism; if flags are seeded
elsewhere, add the row there). The chat hook checks `IsFeatureEnabled("semantic_
cache")` first; `false` ⇒ the hook is a clean no-op and `provider.ChatCompletion`
runs unchanged (proven by a flag-off-no-op test, §5). The injected cache being nil
(handler-level, when the store/cache isn't wired) is ALSO a clean no-op — two
layers of OFF (nil cache, or flag off), both byte-identical to pre-bf-core-2 chat.

### D9 — Hermetic TDD (binding)

ALL bf-core-2 tests: the store uses temp/in-memory SQLite via the existing
`store.Open` test pattern; the chat hook is tested with an INJECTED fake cache (or
a real cache over in-memory SQLite) + a fake provider. **NO real network, NO real
`provider.Embedding` call, NO real `time.Now`, NO `time.Sleep`, NO subprocess.**
The clock is injected (D7). This is binding (Wave-7 hermetic lesson, BIFROST-MAP
§494).

---

## 2. Target files

### IN-SCOPE — NEW (greenfield, additive)

| File | Contents |
|---|---|
| `internal/semcache/cache.go` | NEW domain package. `Cache` struct (holds a `repo` repository interface, `clock func() time.Time`, ttl getter via settings reader); `NewCache(...)` constructor (additive deps); `Lookup(ctx, model, prompt) (response []byte, hit bool, err error)` (exact-key SQLite read only — no embedder, no semantic branch); `Store(ctx, model, prompt, response []byte) error` (write-through, stores `embedding_json=[]`). Errors-as-values, no global state, no init. |
| `internal/semcache/keys.go` | NEW. `CacheKey(model, prompt string) string` = `sha256(normalized prompt + model)` (D1; normalization = the documented prompt-normalization, e.g. trimmed/lowercased messages — match phase-19 intent; if normalization detail is undocumented, default to a deterministic JSON-of-messages+model and record the normalization choice in open-questions rather than inventing). |
| `internal/semcache/cache_test.go` | RED-first. Exact-key hit short-circuit; miss → write-through; expired entry not served; hit_count increments. Hermetic (D9). |
| `internal/store/semcache.go` | NEW. `SemanticCacheEntry` struct + the D3 methods (`GetSemanticCacheByKey`, `InsertSemanticCacheEntry`, `IncrementSemanticCacheHit`, `PurgeExpiredSemanticCache`, `ListSemanticCacheEntries`, `SemanticCacheStats`, `ClearSemanticCache`). **NO `LoadSemanticCandidates`** (the ≤500 cosine loader is deferred with the semantic half, D2). Mirrors `featureflags.go` patterns. |
| `internal/store/semcache_test.go` | RED-first: insert→get-by-key round-trip; expired filtered out; hit_count increment; clear empties; stats. Temp/in-mem SQLite. |
| `internal/admin/cache.go` | NEW. `func (h *Handlers) GetSemanticCache(ctx)` (stats + entries: key, model, hits, expires — NOT full responses, phase-19:50; `writeData`); `func (h *Handlers) ClearSemanticCache(ctx)` (DELETE; `h.recordAudit(ctx,"semantic_cache.clear",...)`; phase-19:51). Mirrors `featureflags.go`. |
| `internal/admin/cache_test.go` | RED-first: GET returns envelope shape (no full responses leaked); DELETE clears + audits. |

### IN-SCOPE — EXTEND (additive only)

| File | Change (additive ONLY) |
|---|---|
| `internal/store/migrate.go` | ADD the `semantic_cache` table + 2 indexes to the tables slice (D3, phase-19:34-46 verbatim). ADD/locate the `semantic_cache` feature-flag seed (D8). NOTHING destructive. |
| `internal/store/featureflags.go` | ADD `IsFeatureEnabled(key string) (bool, error)` (D8, fail-OFF on missing). Additive method. |
| `internal/store/featureflags_test.go` (EXTEND) | RED: `IsFeatureEnabled` true/false/missing-key cases. |
| `internal/api/chat.go` | ADD a `semanticCache SemanticCache` field on `ChatHandler` + `SetSemanticCache(c SemanticCache)` additive setter (mirrors `SetVKGate :159`). ADD the flag-gated read-through hook BEFORE `provider.ChatCompletion` (`:461`, NON-STREAM branch only, D4/D6) + write-through after a successful `resp` (`:486-488`). `SemanticCache` is a small api-local interface (Lookup/Store) so the api package does not import store/semcache directly (mirrors `modelResolver`/`ComboDispatcher` seams, `chat.go:19,114`). PRESERVE `NewChatHandler` signature. |
| `internal/api/chat_test.go` (EXTEND/CREATE) | RED: hook nil ⇒ provider called (no-op); flag-off ⇒ provider called (no-op); cache hit ⇒ provider NOT called, cached bytes returned (short-circuit proof); miss ⇒ provider called + Store invoked (write-through). Injected fake cache + fake provider. Hermetic. |
| `internal/server/routes_openai.go` | EXTEND the chat wiring block (`:22-37`): construct the semcache `Cache` (over `st`) and `chat.SetSemanticCache(...)` when `st != nil`. Adapter wiring only — NO new /v1 route. (This is the api↔semcache adapter, mirroring the vkGate wiring `:101-102`.) |
| `internal/server/routes_admin.go` | ADD `r.GET("/api/cache/semantic", h.RequireSession(h.GetSemanticCache))` + `r.DELETE("/api/cache/semantic", h.RequireSession(h.ClearSemanticCache))` (append-only static collection). **LAST holder of the routes_admin serial chain.** |

### FORBIDDEN (automatic REJECT if touched)

- Any `VectorStore` interface / Weaviate/Redis/Qdrant/Pinecone backend — **ESC**
  (037/038, §3).
- Any streaming-cache code in `chat.go`'s stream branch (`:418-459`) — streaming
  cache is OUT (D6).
- **Any cosine engine, `Embedder` interface/seam, `SetEmbedder`, semantic branch
  in `Lookup`, `LoadSemanticCandidates`, or fake-embedder test** — the entire
  semantic-similarity half is DEFERRED, not built (D2). Building it now is
  production-inert (nil embedder ⇒ never executes in production) for zero parity
  gain. Automatic REJECT.
- Any `provider.Embedding` / `router.ResolveForModel` call in the cache path or in
  any test — bf-core-2 does no embedding at all (D1/D2).
- Any background reaper goroutine / `time.Ticker` / `init()` for TTL — lazy purge
  only (D7).
- A fabricated inline `GuardrailEngine.Evaluate` call in chat.go — guardrails do
  not run inline today; placement is positional only (D4).
- `NewChatHandler` signature change — additive setter only.
- Caching on any endpoint other than `/v1/chat/completions` non-stream (D6).
- Any UI file (`ui/**`) — bf-core-2 is Go-only; `/api/cache/semantic` has no UI
  page (BIFROST-MAP §378, new surfaces ship a Go integration test, no UI touch).
- Destructive DDL (DROP/RENAME) in `migrate.go`.

---

## 3. Scope / Non-goals — explicit ESC list

| ESC / VAR item | Matrix row(s) | Why |
|---|---|---|
| **`VectorStore` interface** (Ping/CreateNamespace/GetNearest/Add/Delete) | 037 | g0router uses SQLite + Go cosine by design (phase-19:61; BIFROST-MAP §281). |
| **External vector backends** (Weaviate, Redis, Qdrant, Pinecone) | 038 | Out of scope by design (phase-19:61). |
| **Streaming response accumulation** (cache streamed responses) | 036 (stream half) | Non-streaming chat ONLY (phase-19:68; D6). VAR. |
| **Background reaper goroutine** for TTL | 036 (reaper half) | Lazy purge on read only (phase-19:69; D7); no global state / init (AGENTS.md). ESC. |
| **SEMANTIC-SIMILARITY half — DEFERRED, NOT BUILT** (cosine engine, `Embedder` interface/seam, `LoadSemanticCandidates` ≤500 loader, semantic branch in `Lookup`, embedding pipeline) | 034 (semantic half), 035 (Provider/EmbeddingModel/Dimension) | embedding-source decision is OPEN (D2) + un-hermetic to test live + ESC-REF-ABSENT blocks the Bifrost shape. Building it now would be production-inert (nil embedder ⇒ never live) for ZERO parity gain — 034/035 flip to PARTIAL regardless. DEFERRED to a future plan that builds AND tests the cosine path end-to-end against the operator-decided embedder (§7). |
| **Bifrost semantic-cache plugin wire shape** (`plugins/semanticcache/main.go`) | 034/035/036 cites | UNREADABLE (ESC-REF-ABSENT). bf-core-2 builds to phase-19, not the Bifrost plugin. |

No-leftovers (binding, §3 CLI_ORCHESTRATOR): bf-core-2 adds the cache hook ONLY if
it actually short-circuits the provider on a hit and writes through on a miss
(proven §5). EVERYTHING bf-core-2 builds is live in production — there is NO
production-inert code: no cosine engine, no embedder seam, no semantic branch (all
DEFERRED, D2). If at impl the hook does not short-circuit on a hit, STOP +
escalate. The deferred semantic half is recorded in open-questions (§7) — it is a
clean deferral (nothing built), not silent dead code.

---

## 4. Task graph (TDD; `N. [step] -> verify: [check]`)

Cadence (AGENTS.md "TDD always"): no impl file lands before its covering
`_test.go` is committed RED. `go test ./... && go vet ./... && go build ./...`
green at EVERY commit. Footer on every commit:
`Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`.
**Do NOT touch routes_admin.go until bf-mcp-2 releases the serial slot (§7).**

1. **[cache key, RED→GREEN]** Write `keys_test.go` (deterministic key for
   same model+prompt; differs across model/prompt; normalization stable). Add
   `keys.go` (`sha256(normalized prompt + model)`, D1). -> verify:
   `go test ./internal/semcache/ -run Key` green. Commit RED/GREEN:
   `phase-1/bf-core-2: failing cache-key test (TDD red)` /
   `phase-1/bf-core-2: sha256 exact-key cache-key`.

2. **[store table + CRUD, RED→GREEN]** Write `internal/store/semcache_test.go`
   (insert→get-by-key; expired filtered; hit_count++; clear; stats; temp/in-mem
   SQLite). Add the `semantic_cache` table + indexes to `migrate.go` (D3 verbatim)
   + `internal/store/semcache.go` methods (NO `LoadSemanticCandidates`, D2). ->
   verify: `go test ./internal/store/ -run Semantic` green; migration additive
   (grep §5); `go vet ./... && go build ./...` exit 0. Commit RED then GREEN:
   `phase-1/bf-core-2: failing semantic_cache store test (TDD red)` /
   `phase-1/bf-core-2: semantic_cache table + exact-key store CRUD (additive)`.

3. **[feature-flag by-key gate, RED→GREEN]** Extend `featureflags_test.go`
   (`IsFeatureEnabled` true/false/missing→false). Add `IsFeatureEnabled` to
   `featureflags.go` + seed the `semantic_cache` flag row additively (D8). ->
   verify: `go test ./internal/store/ -run Feature` green; flag seeded (grep §5).
   Commit RED/GREEN: `phase-1/bf-core-2: IsFeatureEnabled by-key + semantic_cache flag seed`.

4. **[cache domain Lookup/Store, RED→GREEN]** Write `internal/semcache/cache_test.go`
   (exact-key hit short-circuit returns cached bytes; miss → Store write-through;
   expired not served; hit_count++; injected fixed clock D7/D9). Add
   `internal/semcache/cache.go` (`NewCache`, `Lookup` — exact-key only, no embedder
   / no semantic branch; `Store`; ttl from settings D5; lazy purge D7). -> verify:
   `go test ./internal/semcache/` green; no embedding/network in the package (grep
   §5); `go vet ./... && go build ./...` exit 0. Commit RED then GREEN:
   `phase-1/bf-core-2: failing semcache Lookup/Store tests (TDD red)` /
   `phase-1/bf-core-2: exact-key read-through/write-through cache (semantic half deferred)`.

5. **[chat hook, RED→GREEN]** Extend `internal/api/chat_test.go`: nil cache ⇒
   provider called (no-op); flag-off ⇒ provider called (no-op); HIT ⇒ provider
   NOT called + cached bytes returned (SHORT-CIRCUIT PROOF); MISS ⇒ provider
   called + Store invoked (write-through); stream branch ⇒ cache NEVER consulted
   (D6). Add the `SemanticCache` api-local interface + `semanticCache` field +
   `SetSemanticCache` setter + the flag-gated hook before `:461` and write-through
   after `:486-488` (D4/D6/D8). -> verify: `go test ./internal/api/ -run Chat`
   green; short-circuit + flag-off-no-op proven; `NewChatHandler` signature
   unchanged (grep §5); `go vet ./... && go build ./...` exit 0. Commit RED then
   GREEN: `phase-1/bf-core-2: failing chat semantic-cache hook tests (TDD red)` /
   `phase-1/bf-core-2: flag-gated read-through/write-through cache hook in non-stream chat`.

6. **[admin GET/DELETE handlers, RED→GREEN]** Write `internal/admin/cache_test.go`
   (GET envelope shape — keys/model/hits/expires, NO full responses; DELETE clears
   + audits). Add `internal/admin/cache.go` (D3 stats/list + clear). -> verify:
   `go test ./internal/admin/ -run Cache` green; no full `response_json` in the
   GET DTO (grep §5). Commit RED then GREEN:
   `phase-1/bf-core-2: failing /api/cache/semantic handler tests (TDD red)` /
   `phase-1/bf-core-2: GET/DELETE /api/cache/semantic admin handlers (audited)`.

7. **[wiring: api↔semcache adapter, RED→GREEN]** Extend `routes_openai.go` chat
   wiring (`:22-37`) to construct the `Cache` over `st` and
   `chat.SetSemanticCache(...)` when `st != nil`. Add a server-level test proving
   an exact-key hit through the real handler short-circuits the provider (flag on)
   and a flag-off request does not. -> verify: `go test ./internal/server/...`
   green; NO new /v1 route added (grep §5); `go test ./... && go vet ./... &&
   go build ./...` exit 0. Commit RED then GREEN:
   `phase-1/bf-core-2: wire exact-key semantic cache into chat handler`.

8. **[routes_admin serial slot — LAST, RED→GREEN]** AFTER bf-mcp-2 releases the
   slot: append `GET`+`DELETE /api/cache/semantic` to `routes_admin.go`. Add/extend
   a server route test that the routes resolve to the handlers. -> verify:
   `go test ./internal/server/... && go test ./...` green; routes registered
   (grep §5); additive append only. Commit:
   `phase-1/bf-core-2: register GET/DELETE /api/cache/semantic (routes_admin serial terminus)`.

9. **[close]** Full validation (§6); flip matrix rows (§7); update
   `open-questions.md` (semantic-half-deferred + streaming + reaper +
   guardrail-inline + normalization items); update `docs/WORKFLOW.md`; the
   routes_admin serial chain TERMINATES (releases to nobody). -> verify: §6 all
   green; matrix + WORKFLOW + open-questions committed. Commit:
   `phase-1/bf-core-2: close — exact-key cache HAVE, semantic half deferred; matrix flip; serial chain terminus`.

---

## 5. Acceptance criteria (binary; file:line / grep where possible)

**Test gates** (each yes/no, exit 0):
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/semcache/ -v` → exact-key cache + cache-key pass.
- `go test ./internal/store/ -run 'Semantic|Feature' -v` → store + IsFeatureEnabled pass.
- `go test ./internal/api/ -run Chat -v` → hook tests pass (incl. short-circuit + flag-off no-op).
- `go test ./internal/admin/ -run Cache -v` → GET/DELETE pass.
- `go test ./internal/server/ -v` → wiring + route tests pass.

**TDD-order proof** — each impl's covering test is in an earlier-or-equal commit:
```bash
for pair in \
  "internal/semcache/cache_test.go:internal/semcache/cache.go" \
  "internal/store/semcache_test.go:internal/store/semcache.go" \
  "internal/api/chat_test.go:internal/api/chat.go" \
  "internal/admin/cache_test.go:internal/admin/cache.go" ; do
  tf=${pair%%:*}; im=${pair##*:}
  ct=$(git log --format=%ct -1 -- "$tf"); cf=$(git log --format=%ct -1 -- "$im")
  [ "$ct" -le "$cf" ] || echo "TDD VIOLATION: $im"   # prints nothing
done
```

**Grep proofs:**
```bash
# greenfield domain exists (exact-key only)
test -f internal/semcache/cache.go && test -f internal/semcache/keys.go
# table + indexes are additive, verbatim phase-19
grep -n "semantic_cache" internal/store/migrate.go
grep -n "idx_semantic_cache_model\|idx_semantic_cache_expires" internal/store/migrate.go
git diff <base>..HEAD -- internal/store/migrate.go | grep -E '^\+' | grep -iE 'DROP|RENAME' | wc -l   # = 0
# flag gating: by-key lookup + the hook checks the flag (fail-OFF)
grep -n "func (s \*Store) IsFeatureEnabled" internal/store/featureflags.go
grep -n "semantic_cache" internal/api/chat.go internal/server/routes_openai.go   # flag key + wiring
# LIVE hit short-circuit proof — a hit returns WITHOUT calling the provider
grep -niE "short.?circuit|cache hit|provider NOT called|notCalled|providerCalls" internal/api/chat_test.go
# flag-off / nil-cache no-op proof
grep -niE "flag.?off|nil cache|no-?op" internal/api/chat_test.go
# after-guardrails positional: hook sits AFTER the vk gate, BEFORE ChatCompletion (non-stream)
grep -n "semanticCache\|SetSemanticCache" internal/api/chat.go
# SEMANTIC HALF NOT BUILT — no cosine engine, no embedder seam, no candidate loader (D2)
! test -f internal/semcache/cosine.go && echo "no cosine engine OK"
! grep -rn "Embedder\|Cosine\|bestMatch\|SetEmbedder\|LoadSemanticCandidates" internal/semcache/ internal/store/semcache.go && echo "no semantic-half code OK"
# NO embedding network call anywhere (bf-core-2 does no embedding at all)
! grep -rn "provider.Embedding\|ResolveForModel" internal/semcache/ && echo "no embedding in semcache OK"
! grep -rnE 'http\.Get|net\.Dial|time\.Sleep|time\.Now\(\)' internal/semcache/*_test.go internal/api/chat_test.go && echo "hermetic OK"
# NO background reaper / init / global state
! grep -rn "func init(\|go func()\|time.NewTicker\|time.Tick(" internal/semcache/ && echo "no reaper/init OK"
# NO streaming-cache (hook only in non-stream branch)
grep -n "semanticCache" internal/api/chat.go   # references appear only around the non-stream ChatCompletion call, not the stream branch
# NewChatHandler signature unchanged (additive setter only)
grep -n "func NewChatHandler" internal/api/chat.go   # still NewChatHandler(router *inference.Router)
grep -n "func (h \*ChatHandler) SetSemanticCache" internal/api/chat.go
# admin GET does NOT leak full responses (phase-19:50)
! grep -niE "response_json|ResponseJSON" internal/admin/cache.go | grep -i "json:\"" && echo "GET omits full responses OK"
grep -n "recordAudit" internal/admin/cache.go   # DELETE audited
# routes registered (terminus), no new /v1 route
grep -n "/api/cache/semantic" internal/server/routes_admin.go   # GET + DELETE
! grep -nE 'r\.(GET|POST)\("/v1/' internal/server/routes_openai.go | grep -i cache && echo "no new /v1 cache route OK"
# ESC: no vector backends
! grep -rniE "weaviate|qdrant|pinecone|VectorStore" internal/semcache/ internal/store/semcache.go && echo "no vector backend OK"
```

**Behavioral acceptance (binary):**
- With `semantic_cache` flag ON and a prior write-through for `(model, prompt)`, a
  subsequent identical request returns the CACHED `response_json` and
  `provider.ChatCompletion` is **NOT called** (short-circuit proven via a fake
  provider whose call count stays 0). hit_count increments.
- With the flag OFF (or the cache nil), chat behavior is byte-identical to
  pre-bf-core-2: the provider is called, no cache read/write occurs.
- An expired entry (`expires_at` < injected clock now) is NOT served; a fresh
  request misses and writes through.
- The cache hook runs ONLY in the non-stream branch; a streaming request never
  consults the cache.
- `GET /api/cache/semantic` returns `{data: {stats, entries:[{key,model,hits,
  expires}]}}` with NO full `response_json`; `DELETE` clears the table and writes
  an audit row.
- bf-core-2 makes NO embedding call and contains NO cosine/embedder code — the
  semantic-similarity half is deferred (D2), so there is no production-inert path.

---

## 6. Validation commands

```bash
go test ./... && go vet ./... && go build ./...                 # exit 0 (binding)
go test ./internal/semcache/ -v
go test ./internal/store/ -run 'Semantic|Feature' -v
go test ./internal/api/ -run Chat -v
go test ./internal/admin/ -run Cache -v
go test ./internal/server/ -v
```
No UI build / Playwright needed — bf-core-2 ships NO UI touch and NO mock
correction (`/api/cache/semantic` has no UI page; BIFROST-MAP §378). Hermetic only
(D9): no test may hit the network, call real `provider.Embedding`, sleep, or call
real `time.Now`.

---

## 7. Freeze rules + matrix-flip + WORKFLOW + open-questions + no-leftovers

**Freeze rules (binding):**
- `internal/server/routes_admin.go` — bf-core-2 is the **LAST HOLDER** of the
  bf-mcp-1 → bf-mcp-2 → bf-core-2 serial chain (BIFROST-MAP §343-346). bf-core-2
  MUST NOT begin its routes_admin.go edit (task 8) until bf-mcp-2 has merged and
  released the slot. On close, the chain TERMINATES — bf-core-2 releases to
  NOBODY. Additive append only (static `/api/cache/semantic` collection).
- `internal/api/chat.go` — additive: a field + setter + a hook block; NO
  `NewChatHandler` signature change. Not a serial-route file edit (no route
  registered here).
- `internal/store/migrate.go` — additive `ensureTable` only (the `semantic_cache`
  entry + 2 `CREATE INDEX IF NOT EXISTS`); no destructive DDL.
- `internal/semcache/` is greenfield-disjoint — runs ∥ all other bf-* plans.
- No reverse-engineering of the absent Bifrost ref (ESC-REF-ABSENT) — build to
  phase-19:60-69 + g0router conventions only. NO vector-backend pipeline.

**Matrix-flip (at close, in `.planning/parity/matrix/bifrost-core.md`):**
- PAR-BF-CORE-034 → **PARTIAL** (exact-key-hash path LIVE + short-circuiting per
  D1; the SEMANTIC-SIMILARITY half is DEFERRED — not built — per D2, pending the
  operator embedding-source decision). Note cite bf-core-2 + D1/D2.
- PAR-BF-CORE-035 → **PARTIAL** (TTL/flag config via settings store + feature flag
  per D5/D8; Threshold reserved, Provider/EmbeddingModel/Dimension out-of-scope-now
  with the deferred semantic half per D2).
- PAR-BF-CORE-036 → **PARTIAL** (TTL + lazy purge HAVE per D7; streaming
  accumulation + background reaper = VAR/ESC per D6/D7).
- PAR-BF-CORE-037 → **MISSING/ESC** (VectorStore — g0router SQLite+Go cosine by
  design). Annotate ESC + cite bf-core-2 §3.
- PAR-BF-CORE-038 → **MISSING/ESC** (vector backends — out of scope by design).
  Annotate ESC + cite bf-core-2 §3.

**`open-questions.md` (append at close):**
```
## bf-core-2 — Semantic cache (g0router-shaped) — 2026-06-15
- [ ] Semantic-similarity cache half DEFERRED (not built) — a future plan builds the cosine engine + embedder wiring TOGETHER once the embedding source is decided, so the cosine path is tested LIVE (end-to-end against the real operator-decided embedder), not against a fake. bf-core-2 shipped only the deterministic exact-key-hash cache. Needs operator decision: WHICH connection/model embeds prompts (phase-19:65 "existing OpenAI-compatible connection") + acceptance of a non-hermetic live embedding round-trip. Why: building it in bf-core-2 would be production-inert (nil embedder ⇒ never live) for zero parity gain — 034/035 flip to PARTIAL regardless; ESC-REF-ABSENT blocks the Bifrost shape; a fabricated vector source is forbidden.
- [ ] Inline guardrail enforcement in chat.go — guardrails (GuardrailEngine.Evaluate) do NOT run in the chat request path today (admin/test-only). bf-core-2 places the cache hook POSITIONALLY where a guardrail check belongs (after VK gate, before dispatch). When a future plan wires Evaluate inline, it MUST run BEFORE the cache hook so a blocked prompt is never served/cached. Why: preserve the phase-19:68 "after guardrails" invariant once guardrails go inline.
- [ ] Streaming-response cache (036 stream-accumulation half) — OUT by phase-19:68 (non-streaming only). Why: streamed SSE reassembly/replay deferred.
- [ ] Background TTL reaper (036 reaper half) — bf-core-2 uses lazy purge on read (phase-19:69); a background reaper would need a goroutine/Ticker (no global state/init per AGENTS.md). Why: deferred unless a sweep cadence is funded.
- [ ] Prompt normalization detail for cache_key — if phase-19's exact normalization (trim/lowercase/message-shape) is under-specified, bf-core-2 defaults to a deterministic JSON-of-messages+model hash. Why: the normalization choice affects hit rate; revisit if a canonical form is decided.
- [ ] VectorStore + external backends (037/038) — ESC by g0router's SQLite+Go-cosine design. Why: product-design divergence from Bifrost.
```

**`docs/WORKFLOW.md` (update at close):** add a bf-core-2 row — semantic cache
(g0router-shaped) shipped: exact-key-hash cache LIVE (flag-gated, short-circuits
provider, non-stream chat only); the SEMANTIC-SIMILARITY half (cosine + embedder)
is DEFERRED — not built — pending the operator embedding-source decision;
`semantic_cache` table + GET/DELETE `/api/cache/semantic` (audited) added; rows
034/035/036 → PARTIAL, 037/038 → ESC; deferred/ESC items recorded in
open-questions; routes_admin serial chain TERMINATES at bf-core-2; ESC-REF-ABSENT
honored (built to phase-19 only, no Bifrost vector shape).

**No-leftovers confirmation (binding):** bf-core-2 adds the cache hook (consumed:
proven to short-circuit the provider on a hit and write through on a miss, §5),
the `semantic_cache` table + exact-key store methods (consumed by the hook + admin
handlers), `IsFeatureEnabled` (consumed by the hook's flag gate), and the admin
GET/DELETE (consumed by the registered routes). EVERYTHING bf-core-2 builds is LIVE
in production — there is NO production-inert code: no cosine engine, no `Embedder`
seam, no semantic branch, no candidate loader (the whole semantic-similarity half
is DEFERRED, not built — D2). The deferral is a DOCUMENTED clean cut (open-questions
§7), not silent dead code. The retained `embedding_json` column stores `[]` (data,
forward-compatible — not executable dead code). No dead column, field, route, or
interface method is introduced; each new surface has a grep-proven live consumer
(§5). If at impl the cache hook does not short-circuit on a hit, the plan STOPS and
escalates.
```
