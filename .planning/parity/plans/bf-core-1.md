# Micro-plan bf-core-1 — Provider-interface reconciliation (bifrost-core, Go)

```
program: bifrost-parity (bifrost phase — BUILDABLE-ADDITIVE only; the ~50%
  re-architecture — plugin pipeline, queue, pooling, clustering, OTEL, vector
  backends — is permanently deferred per BIFROST-MAP §1/§8 ESC set)
plan: bf-core-1
status: READY (rev 1 — authored against the LIVE tree @ <base> = ad0cb82; the
  bf-openai chain 1–4 is ALREADY MERGED. BIFROST-MAP micro-plan index row line
  305; bifrost-core disposition rows 001(subset)/027/028/020(variant)/019/002
  (.planning/parity/matrix/bifrost-core.md:10-11,29,36-37); architectural
  decisions #5/#6/#8; freeze rules BIFROST-MAP:390-393)
runs: core track. Disjoint from openai/gov/mcp tracks (run ∥). The
  provider.go/errors.go MICRO-SERIAL with bf-openai-4 (BIFROST-MAP:348-350,390)
  is DISCHARGED — bf-openai-4 is MERGED (HEAD ad0cb82 has the shipped
  CountTokens interface method + openai impl), so bf-core-1 has NO concurrent
  holder of internal/schemas/{provider,errors}.go. See §1.0.
branch: main (direct commits, per AGENTS.md "No PR workflow")
commit-prefix: phase-1/bf-core-1:
commit-footer: Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>
ref-source: BLOCKED — ESC-REF-ABSENT (BIFROST-MAP §47-68). The frozen Bifrost
  ref @ ca21298 is ABSENT on this host. The ONLY ground truth is
  .planning/parity/matrix/bifrost-core.md + g0router's own conventions. Build to
  documented matrix behavior + g0router conventions; STOP-escalate on any
  undocumented Bifrost detail. NEVER build to a guessed Bifrost wire format.
base: <base> = `git rev-parse HEAD` recorded at P0 (observed ad0cb82). Substitute
  the actual SHA everywhere §5 says <base>.
go-serial-slot: NONE for routes (this plan registers NO HTTP routes). The
  provider.go/errors.go micro-serial with bf-openai-4 is DISCHARGED (bf-openai-4
  MERGED — §1.0). bf-core-1 may freely (and additively) touch
  internal/schemas/{provider,errors}.go IF a live consumer requires it — but per
  the §0 finding, NONE does, so the expected schema-file delta is ZERO.
new-route: NO. Confirmed: bf-core-1 registers NO new HTTP routes (no
  routes_openai.go / routes_admin.go / routes_mcp.go touch).
headline: this is a NEAR-EMPTY plan. After grepping the LIVE post-bf-openai-4
  tree, EVERY row routed to bf-core-1 is either ALREADY-SATISFIED (matrix flip,
  no code) or VARIANT-by-design (label, no code) or ESCALATED (no buildable
  additive surface without dead code). There is NO additive Go feature code to
  ship that has a live consumer. The correct, honest outcome of bf-core-1 is a
  MATRIX-FLIP-ONLY closeout — NOT manufactured dead interface methods / dead
  context fields / a dead error flag. See §0 and §1.
```

---

## 0. The big finding — read before everything else

The BIFROST-MAP index (line 278) scoped bf-core-1 as "`GatewayContext` typed KV +
Compaction + CountTokens reconciliation only" plus an optional `AllowFallbacks`
flag (line 279). Grepping the LIVE tree (post-bf-openai-4 merge, HEAD ad0cb82)
collapses ALL of that to zero buildable additive code with a live consumer:

| Surface (MAP scope) | LIVE reality (file:line evidence) | bf-core-1 disposition |
|---|---|---|
| **CountTokens** interface/method reconciliation (027 subset) | **ALREADY SHIPPED by bf-openai-4.** Interface method `CountTokens(ctx *GatewayContext, key Key, request *ChatRequest) (*TokenCountResponse, *ProviderError)` (`internal/schemas/provider.go:107`); openai impl `internal/providers/openai/counttokens.go:23` (POST upstream `/v1/responses/input_tokens`); `TokenCountResponse{Tokens int}` (`provider.go:64-66`). | **SAT — already built.** No code. Verify + flip the CountTokens subdimension of 027 (the request-type/method surface). |
| **Compaction** interface/method reconciliation | **DOES NOT EXIST anywhere** — `! grep -rIn 'Compaction\|CompactRequest\|CompactResponse' internal/ --include='*.go'` → no match; no method on `provider.go:69-108`; no schema; no route. bf-openai-4 ESCALATED it (`bf-openai-4.md` §0 / §8 ESC-COMPACTION — non-additive interface change touching all 43 providers; responses-rewrite-dependent; wire shape unverifiable under ESC-REF-ABSENT). | **ESC (inherited).** No buildable surface. Adding a `Compaction` method = dead interface method on 43 providers (no route consumes it — none is funded). FORBIDDEN (§2, no-leftovers). |
| **Richer typed `GatewayContext`** (KV + metadata) (028) | `GatewayContext struct { RequestID string }` (`internal/schemas/provider.go:25-27`). Grep of ALL 20 construction sites (`internal/api/*.go`) shows EVERY caller sets ONLY `RequestID` (`grep -rn 'GatewayContext{' internal/` → all `{RequestID: fmt.Sprintf("%d", ctx.ID())}`). NO caller reads or writes any KV/metadata field; `PostHookRunner.Run(ctx, response)` (`provider.go:45-47`) reads no context field. | **ESC / VAR-record (no live consumer).** Adding a `KV map[string]any` / `Metadata` field would be a DEAD field — no handler reads it, no provider reads it. The Wave-5 `NewWithShutdown` dead-wiring lesson + BIFROST-MAP:392 "no dead interface methods" forbid it. 028 stays **PARTIAL** with a variant note. |
| **`AllowFallbacks` flag on `ProviderError`** (020) | g0router's fallback decision is driven by `inference.Classify(statusCode, body) → ErrorClass/Retryable` (`errorclass.go:73-113`) mapped to a `Verdict` by the caller; the loop `SelectionEngine.WithAccountFallback` (`selection.go:334-375`) and `AccountRunner.RunModel` (`runner.go:37-81`) branch on **`Verdict`** + **`pe.StatusCode`** (`runner.go:70`) — they read NO field off `ProviderError` to decide fallback. `ProviderError` (`errors.go:26-33`) has `Message/Type/Param/Code/StatusCode/Meta` — no `AllowFallbacks`. | **VAR-record (no live consumer).** g0router HAS account-level fallback (a different, working mechanism). An `AllowFallbacks bool` on `ProviderError` would be a DEAD field unless the entire Verdict-mapping path (`errorclass.go` + `WithAccountFallback` + `RunModel`) is rewired to honor it — a behavior change, NOT additive, and exactly the dead-flag the brief forbids. 019/020 stay **variant-HAVE**; the flag is NOT added. |
| **Request-type constants** (027) | `RequestType` is carried as a string literal (e.g. `"count_tokens"` in `counttokens.go`); there is no `RequestTypeXxx` enum that a live call site demands (bf-openai-4 explicitly declined to add one — `bf-openai-4.md` §1.4: "do NOT add one unless an existing call site requires it"). | **VAR-record.** g0router handles its live request categories via string literals + dedicated handlers. Adding 20+ Bifrost request-type constants with no consumer = dead constants. NOT added. 027 stays **PARTIAL** (variant note). |
| **postHookRunner on stream methods** (002) | **ALREADY HAVE** — `ChatCompletionStream(ctx, postHookRunner PostHookRunner, …)` etc. on every stream method (`provider.go:76,79,82,87,92,94`). `postHookSpanFinalizer` is part of the tracing ESC (BIFROST-MAP:277). Matrix already marks 002 HAVE. | **SAT — already HAVE** (matrix already correct; verify-only, no flip needed). |
| **`AllowFallbacks`/fallback-chain row 019** | `core/bifrost.go` fallback chain via PreLLMHook. g0router's `WithAccountFallback` (`selection.go:334`) + weighted smooth-WRR (`selection.go:252`) is a DIFFERENT mechanism (no per-attempt PreLLMHook). | **VAR — variant-HAVE.** Record as variant; the per-attempt-PreLLMHook mechanism is ESC (plugin-pipeline-coupled). |

**Net for bf-core-1:** BUILD = **NOTHING** (no additive code with a live consumer).
SAT = CountTokens-subset of 027 (shipped by bf-openai-4), 002 (already HAVE).
VAR = 019, 020, the request-type-constant subset of 027, the AllowFallbacks flag
(not added). ESC (inherited / no-consumer) = Compaction, the GatewayContext-KV
enrichment of 028, the 20+ request-type constants. **This is a matrix-flip-only
closeout.** A near-empty build is the CORRECT outcome — bf-openai-4 already
covered the genuinely-buildable reconciliation (CountTokens), and every remaining
MAP-suggested addition (Compaction method, GatewayContext KV, AllowFallbacks flag,
request-type constants) would be DEAD CODE with no live consumer, which §3
no-leftovers + BIFROST-MAP:392 forbid.

### 0.1 (intentionally folded into §1.0 below)

---

## 1. Decisions made (and why) — binding

### 1.0 — provider.go / errors.go micro-serial is DISCHARGED (bf-openai-4 MERGED)

BIFROST-MAP:348-350,390 declared a micro-serial: both bf-core-1 and bf-openai-4
may touch `internal/schemas/{provider,errors}.go` + compaction, and the
orchestrator must serialize that file window. **At <base> = ad0cb82, bf-openai-4
is ALREADY MERGED** — the proof is in the live tree: `CountTokens` is on the
`Provider` interface (`provider.go:107`) and the openai impl exists
(`counttokens.go:23`), and `APIError.Param`/`ProviderError.Param` already exist
(`errors.go:7,29`), all of which bf-openai-4 shipped. **There is therefore NO
concurrent holder of those schema files.** bf-core-1 is free to make an additive
edit to `provider.go`/`errors.go` without coordination — BUT per the §0 finding,
NO such edit has a live consumer, so the expected delta to both files is **ZERO**.

### D1 — Compaction: ESC (inherited from bf-openai-4); do NOT add a dead method

`Compaction` has NO method on the `Provider` interface, NO schema, and NO route
anywhere (§0 grep proof). bf-openai-4 ESCALATED row 005 (`/v1/responses/compact`)
and explicitly declined to add a `Compaction` method (`bf-openai-4.md` §0/§8/§NOT-
in-scope: "Do NOT add `Compaction` to `internal/schemas/provider.go` … that is
bf-core-1's reconcile surface ONLY if a route is funded — and none is").

**Decision:** **NO route is funded, so NO `Compaction` method is added.** Adding it
would be (a) NON-additive — it forces a method onto all 43 provider
implementations (every `internal/providers/*/` package), and (b) DEAD — nothing
calls it. Both violate §3 no-leftovers and BIFROST-MAP:392 ("only add a provider
method when its route/feature is actually being built"). Compaction STAYS the
bf-openai-4 escalation. If a `/v1/responses/compact` route is ever funded, the
plan that funds it owns adding the method + schema + impl together (route-first,
method-with-it). Recorded in `open-questions.md` (§7).

### D2 — GatewayContext: enrich ONLY where a live caller reads new fields → NONE → no change

The matrix (028, PARTIAL) wants a `BifrostContext` with typed KV storage + request
metadata. g0router's `GatewayContext` carries only `RequestID` (`provider.go:25-27`)
and EVERY one of its 20 construction sites + its sole reader (`PostHookRunner.Run`)
touches only `RequestID` (§0 grep proof). The brief's binding rule: *"enrich it
additively ONLY where a live caller reads the new fields. If it does NOT exist, do
NOT invent a speculative one."*

**Decision:** **No live caller reads any field beyond `RequestID`. Therefore
bf-core-1 adds NO field to `GatewayContext`.** A `KV map[string]any` or `Metadata`
field would be a dead field (no handler populates it from request headers, no
provider reads it, no usage/tracing path consumes it — tracing is ESC,
BIFROST-MAP:288). The minimal typed-KV that "an actual handler uses" does not
exist because no handler has a use for it under the current (non-plugin,
non-tracing) architecture. 028 stays **PARTIAL** with a variant/ESC note: the
RequestID-only context is g0router's deliberate minimal design; the typed-KV/
metadata surface is plugin-pipeline-and-tracing-coupled (both ESC,
BIFROST-MAP:282,288). If a future LIVE feature (e.g. a funded request-metadata
stamping path) needs context KV, IT adds the field with its consumer in the same
plan. Recorded in `open-questions.md` (§7).

> Soundness: this is the exact discipline bf-gov-1 applied to `AllowAllKeys`/
> `ValidateBudgetOwner` (each kept "guarded-but-live" or STOP+escalate). Here the
> honest result is that there is no guarded-but-live home for a context field, so
> none is added.

### D3 — AllowFallbacks: variant-HAVE; do NOT add a dead flag on ProviderError

The brief's binding rule on 020: *"OPTIONALLY add a small AllowFallbacks-equivalent
flag on ProviderError ONLY if it's wired into a live fallback decision in
selection.go. If it would be a dead flag, do NOT add it — mark variant-HAVE and
ESC the per-attempt-PreLLMHook mechanism."*

**Evidence the flag would be dead (§0):** g0router's fallback decision is made by
`inference.Classify(statusCode, body)` → `ErrorClass` + `Retryable`
(`errorclass.go:73-113`), which the caller maps to a `Verdict`; the fallback loop
`WithAccountFallback` (`selection.go:334-375`) branches on `Verdict`
(`VerdictUnknown`/`VerdictPermanent`/temporary) and `RunModel` (`runner.go:69-75`)
additionally reads `pe.StatusCode` for the transient-join. **Neither path reads
any boolean field off `ProviderError` to decide whether to fall back.** Adding
`AllowFallbacks bool` to `ProviderError` (`errors.go:26-33`) would be inert unless
`WithAccountFallback`/`RunModel`/the Verdict-mapping are rewired to honor it — a
behavior change to a working, tested fallback engine, NOT an additive flag.

**Decision:** **Do NOT add `AllowFallbacks` to `ProviderError`.** Record 019/020 as
**variant-HAVE**: g0router HAS account-level fallback (`WithAccountFallback`
`selection.go:334`; smooth-WRR `selection.go:252`; classify `errorclass.go`) — a
different, complete mechanism. The Bifrost per-attempt-`PreLLMHook` fallback-chain
mechanism (matrix 019 cite `core/bifrost.go:4487`; 020 cite
`core/schemas/bifrost.go:1683`) is **ESC** (plugin-pipeline-coupled; the plugin
system is BIFROST-MAP §8 / matrix 004-018 ESC). This resolves the BIFROST-MAP §467
DECISION-NEEDED explicitly: **variant-HAVE; no flag; ESC the per-attempt mechanism.**

### D4 — Request-type constants (027): variant-record; do NOT add dead constants

The matrix (027, PARTIAL) wants 20+ request-type constants
(`core/schemas/bifrost.go:686`). g0router carries request type as a string literal
at the call site (e.g. `"count_tokens"`), and bf-openai-4 explicitly declined to
add a `RequestTypeCountTokens` constant absent a consumer (`bf-openai-4.md` §1.4).

**Decision:** **No 20+-constant enum is added.** g0router handles its live request
categories via dedicated handlers + string literals; an enum of Bifrost request
types (most of which — video/container/rerank/ocr/passthrough/compaction — map to
ESC surfaces with no g0router route) would be dead constants. 027 stays
**PARTIAL** (variant note: g0router handles its live categories; the full Bifrost
request-type taxonomy is broader because it enumerates ESC surfaces).

### D5 — postHookRunner (002): already HAVE; no change

Matrix 002 is already HAVE (`provider.go:76` carries `postHookRunner
PostHookRunner` on every stream method). `postHookSpanFinalizer` is tracing-ESC
(BIFROST-MAP:277). **Decision:** verify-only; no code; no matrix change needed
(002 is already correctly HAVE).

---

## 2. Target files

### IN-SCOPE — code edits

**NONE.** bf-core-1 ships NO production Go code and NO test code. Every MAP-scoped
addition collapses to SAT (already built by bf-openai-4), VAR (variant-by-design,
label only), or ESC (no live consumer → would be dead code). This is the honest,
correct outcome (§0 headline).

### IN-SCOPE — documentation / matrix only (no compile impact)

| File | Change |
|---|---|
| `.planning/parity/matrix/bifrost-core.md` | Flip/annotate rows per §7 (CountTokens subset of 027 → SAT-cite; 019/020 → variant-HAVE note; 028/027 → PARTIAL + variant/ESC note; 002 verify-HAVE). Correct any stale cite. NO behavior claim beyond live evidence. |
| `.planning/parity/plans/open-questions.md` | Append the bf-core-1 ESC/deferred items (§7). |
| `docs/WORKFLOW.md` | Append the bf-core-1 closeout row (§7). |

### FORBIDDEN (automatic REJECT if touched)

- **`internal/schemas/provider.go`** — NO new interface method. Explicitly: **NO
  `Compaction`** (D1, dead method on 43 providers); **NO** Video*/Container*/
  Rerank/OCR/CachedContent*/Passthrough/PassthroughStream methods (all ESC — no
  funded route; adding any = dead interface method, the Wave-5 `NewWithShutdown`
  lesson + BIFROST-MAP:392). **NO** new field on `GatewayContext` (D2, dead field).
  **NO** `RequestType` enum (D4, dead constants). `CountTokens` is already present
  (bf-openai-4) — UNTOUCHED.
- **`internal/schemas/errors.go`** — **NO `AllowFallbacks`** field on `ProviderError`
  (D3, dead flag); no envelope restructuring; no `event_id`/`is_bifrost_error`
  (bf-openai-4 variant-by-design). UNTOUCHED.
- **`internal/inference/selection.go` / `runner.go` / `errorclass.go`** — UNTOUCHED.
  bf-core-1 does NOT rewire the fallback engine to honor a flag (D3). The working
  `WithAccountFallback`/`Classify` path is consumed read-only as evidence.
- **The 43 provider packages** (`internal/providers/*/`) — UNTOUCHED. No new method
  to implement (no interface method is added).
- **All route files** (`routes_openai.go`, `routes_admin.go`, `routes_mcp.go`) —
  UNTOUCHED. bf-core-1 registers no route.
- **Plugin pipeline / ProviderQueue / pooling / KVStore / clustering / CEL routing
  / adaptive LB / health states / OTEL / tracing / vector backends / semantic
  cache** — all ESC (§3). bf-core-2 owns semantic cache separately.
- **All UI / e2e / mocks** — no UI contract. No `ui/**`, no playwright, no mock/seed.
- **No `init()`, no global state, errors-as-values, snake_case** — N/A (no code),
  but binding if any reviewer believes a field IS consumable: it must be proven
  with a live reader BEFORE adding, else STOP + escalate.

---

## 3. Scope / Non-goals — explicit ESC list (the large bifrost-core escalation set)

bf-core-1 builds NOTHING; the following bifrost-core behaviors are **ESC** (out of
scope; divergent / plugin-coupled / not-applicable to a single-binary SQLite
gateway) and are recorded with reasons. (BIFROST-MAP §8, ledger lines 277-289.)

| ESC item | Matrix row(s) | Why ESC |
|---|---|---|
| **`Compaction` method/schema/route** | 001(Compaction subset); bf-openai-4's 005 | No funded route; adding the method = dead interface method on 43 providers (D1). Inherited bf-openai-4 ESC-COMPACTION; responses-rewrite + ESC-REF-ABSENT dependent. |
| **`GatewayContext` typed KV / request-metadata enrichment** | 028 (the KV/metadata half) | No live consumer (D2); plugin-pipeline + tracing coupled (both ESC). RequestID-only context is g0router's deliberate minimal design. |
| **`AllowFallbacks` flag + per-attempt PreLLMHook fallback chain** | 019, 020 | Flag would be dead (D3); the per-attempt mechanism is plugin-pipeline-coupled. g0router's account-level fallback is the variant-HAVE. |
| **20+ request-type constants enum** | 027 (enum half) | Dead constants (D4); most enumerate ESC surfaces (video/container/rerank/ocr/passthrough). |
| **`WebSocketCapableProvider`** | 003 | No WS provider abstraction (BIFROST-MAP:283). |
| **Plugin system** (BasePlugin/HTTPTransportPlugin/LLMPlugin/MCPPlugin/Observability/ConfigMarshaller; ordered pipeline; short-circuit; placement; pooling; pooled HTTP types; case-insensitive helpers) | 004-018, 049, 050 | Runtime re-architecture; g0router has fasthttp middleware only (BIFROST-MAP:282; Go-Port note #2 "largest gap"). |
| **`ProviderQueue` channel routing + lifecycle + object pooling + dropExcessRequests** | 021, 022, 023, 024 | g0router uses direct synchronous calls by design (BIFROST-MAP:284). |
| **`KeySelector` func type + `KVStore` clustering interface** | 025, 026 | Key selection already SAT via `KeyResolver`/`SelectionEngine` (025 ≈ VAR); KVStore presupposes clustering (BIFROST-MAP:285). |
| **CEL routing + rule-chain cycle detection + adaptive LB + health-state machine + weighted-random key selection** | 029, 030, 031, 032, 033 | Divergent routing engine; g0router has prefix routing + smooth-WRR + retry/cooldown (033 weighted selection SAT via `selection.go:252`) (BIFROST-MAP:286). |
| **Semantic cache + streaming accumulation** | 034, 035, 036 | **bf-core-2 owns this** (g0router-shaped, phase-19). NOT bf-core-1. |
| **`VectorStore` interface + Weaviate/Redis/Qdrant/Pinecone backends** | 037, 038 | g0router semantic cache uses SQLite + Go cosine by design (BIFROST-MAP:281). |
| **Clustering** (memberlist gossip + gRPC sync + discovery + leader election + 30-entity replication) | 039, 040, 041, 042 | Single-binary SQLite gateway; clustering is a product-category change (BIFROST-MAP:287). |
| **OTEL plugin + metrics + `Tracer` interface + trace accumulator + header capture** | 043, 044, 045, 046, 047, 048 | Tracing/observability ESC (BIFROST-MAP:288; phase-18:24 deferred OTEL). `postHookSpanFinalizer` (part of 002) is in this set. |

No-leftovers (binding, §3 CLI_ORCHESTRATOR + BIFROST-MAP:392): bf-core-1 adds a
provider interface method / `GatewayContext` field / error flag / request-type
constant ONLY if a LIVE feature consumes it. §0/§1 prove NONE does → bf-core-1
adds NONE. There is no STOP-on-dead-surface risk because no surface is added.

---

## 4. Task graph (TDD; `N. [step] -> verify: [check]`)

There is no production code, so there is no TDD red→green impl cadence (AGENTS.md
"TDD always" applies to code; this plan ships none). The task graph is a
verification + matrix-flip sequence. `go test ./... && go vet ./... && go build
./...` must be green at the single closeout commit (it is the untouched-green
baseline — bf-core-1 changes no `.go` file).

1. **[P0 baseline]** Record `<base>` = `git rev-parse HEAD`; confirm clean tree
   (`git status --porcelain` empty) and untouched-green (`go test ./... && go vet
   ./... && go build ./...` exit 0). -> verify: exit 0; <base> recorded.

2. **[verify SAT — CountTokens already shipped]** Confirm bf-openai-4's CountTokens
   reconciliation is live. -> verify:
   `grep -n 'CountTokens(ctx \*GatewayContext' internal/schemas/provider.go` (→ :107)
   AND `grep -n 'func (p \*Provider) CountTokens' internal/providers/openai/counttokens.go`
   (→ :23) AND `go test ./internal/providers/openai/ -run 'CountTokens' -v` green.

3. **[verify SAT — postHookRunner 002 already HAVE]** -> verify:
   `grep -n 'postHookRunner PostHookRunner' internal/schemas/provider.go` non-empty
   (stream methods carry it). No change.

4. **[verify ESC — Compaction has NO buildable surface]** -> verify:
   `! grep -rIn 'Compaction\|CompactRequest\|CompactResponse' internal/ --include='*.go'`
   AND `! grep -n 'Compaction' internal/schemas/provider.go` (both → no match).
   Confirms D1 (no dead method to add).

5. **[verify ESC/VAR — GatewayContext has no extra-field consumer]** -> verify:
   `grep -n 'type GatewayContext' internal/schemas/provider.go` (→ :25, only
   `RequestID`) AND every `GatewayContext{...}` construction sets only `RequestID`
   (`grep -rn 'GatewayContext{' internal/ --include='*.go' | grep -v 'RequestID:'`
   → no match). Confirms D2 (no field to add).

6. **[verify VAR — AllowFallbacks would be dead]** -> verify:
   `! grep -n 'AllowFallbacks' internal/schemas/errors.go` (absent) AND the
   fallback path reads no error field for the decision
   (`grep -n 'Verdict\|StatusCode' internal/inference/runner.go internal/inference/selection.go`
   shows the decision is Verdict/StatusCode-driven; `! grep -n 'AllowFallbacks'
   internal/inference/*.go`). Confirms D3 (no flag to add).

7. **[matrix flip + docs]** Apply §7 flips to `bifrost-core.md`; append
   `open-questions.md` + `docs/WORKFLOW.md`. -> verify: §6 green; the three doc
   files are the ONLY changed files (`git diff --name-only <base>..HEAD` lists
   exactly: `.planning/parity/matrix/bifrost-core.md`,
   `.planning/parity/plans/open-questions.md`, `docs/WORKFLOW.md`). Commit:
   `phase-1/bf-core-1: close — provider-interface reconciliation is SAT/VAR/ESC; matrix flip only; no buildable additive code (CountTokens shipped by bf-openai-4; Compaction/GatewayContext-KV/AllowFallbacks would be dead)`.

---

## 5. Acceptance criteria (binary; file:line)

**Test gates** (each yes/no, exit 0; bf-core-1 changes no `.go` file, so these are
the untouched baseline):
- `go test ./... && go vet ./... && go build ./...` → exit 0.

**No-dead-code / no-leftovers proof (the PRIMARY acceptance for this plan):**
```bash
# No new interface method was added — Compaction and the ESC method families are absent.
! grep -n 'Compaction\|Rerank\|OCR\|VideoGeneration\|ContainerCreate\|CachedContentCreate\|Passthrough' internal/schemas/provider.go && echo "no dead interface methods OK"
# GatewayContext gained NO field (still RequestID-only) — no dead context field.
grep -n 'type GatewayContext struct' internal/schemas/provider.go
[ "$(grep -c '\b[A-Z][A-Za-z]* ' <(sed -n '/type GatewayContext struct/,/}/p' internal/schemas/provider.go))" -ge 1 ]   # sanity; field set unchanged from RequestID-only
! grep -n 'KV \|Metadata \|Values ' internal/schemas/provider.go && echo "no dead GatewayContext KV/Metadata field OK"
# ProviderError gained NO AllowFallbacks flag — no dead error flag.
! grep -n 'AllowFallbacks' internal/schemas/errors.go internal/inference/*.go && echo "no dead AllowFallbacks flag OK"
# No request-type constant enum was added.
! grep -n 'RequestType[A-Z]' internal/schemas/*.go && echo "no dead request-type constants OK"
# Every NEW interface method/field/flag has a live consumer: VACUOUSLY TRUE — bf-core-1 added NONE.
```

**SAT/already-built proofs (the reconciliation bf-openai-4 shipped):**
```bash
grep -n 'CountTokens(ctx \*GatewayContext, key Key, request \*ChatRequest) (\*TokenCountResponse, \*ProviderError)' internal/schemas/provider.go   # :107
grep -n 'func (p \*Provider) CountTokens' internal/providers/openai/counttokens.go                                                                # :23
grep -n 'postHookRunner PostHookRunner' internal/schemas/provider.go                                                                              # stream methods (002 HAVE)
```

**Changed-files proof (matrix-flip-only closeout):**
```bash
git diff --name-only <base>..HEAD    # EXACTLY:
#   .planning/parity/matrix/bifrost-core.md
#   .planning/parity/plans/open-questions.md
#   docs/WORKFLOW.md
# NO internal/**/*.go file changed (this is the correct near-empty outcome).
! git diff --name-only <base>..HEAD | grep -E '\.go$' && echo "no Go file changed OK"
```

**Behavioral acceptance (binary):**
- bf-core-1 introduces NO new provider interface method, NO `GatewayContext` field,
  NO `ProviderError` flag, NO request-type constant. (Proven by the no-dead-code
  greps above.)
- CountTokens reconciliation (027 subset) is confirmed SAT (bf-openai-4 shipped it).
- The fallback engine (`WithAccountFallback`/`Classify`/`RunModel`) is byte-identical
  to pre-bf-core-1 (no rewire to honor a flag).

---

## 6. Validation commands

```bash
go test ./... && go vet ./... && go build ./...     # exit 0 (untouched-green baseline)
# no-dead-code proofs (§5)
! grep -n 'Compaction' internal/schemas/provider.go && echo "no Compaction method OK"
! grep -n 'AllowFallbacks' internal/schemas/errors.go internal/inference/*.go && echo "no AllowFallbacks flag OK"
! grep -nE 'KV |Metadata ' internal/schemas/provider.go && echo "no GatewayContext KV/Metadata field OK"
! git diff --name-only <base>..HEAD | grep -E '\.go$' && echo "matrix-flip-only (no Go change) OK"
```
No UI build / Playwright needed — bf-core-1 ships NO UI touch and NO mock
correction. No hermetic-test concern — bf-core-1 ships NO test (no code).

---

## 7. Freeze rules + matrix-flip + open-questions + WORKFLOW + no-leftovers

**Freeze rules (binding):**
- `internal/schemas/provider.go` + `errors.go` — the bf-core-1 ↔ bf-openai-4
  MICRO-SERIAL (BIFROST-MAP:348-350,390) is **DISCHARGED** (bf-openai-4 MERGED at
  <base>; §1.0). bf-core-1 has no concurrent holder. **No dead interface methods**
  (BIFROST-MAP:392) — and bf-core-1 adds none (the freeze rule is satisfied
  vacuously). Both files stay UNCHANGED by this plan.
- bf-core-1 is NOT a `routes_openai.go` / `routes_admin.go` / `routes_mcp.go`
  holder (registers no route). Takes NO serial slot.
- `internal/governance/quota.go` serial (bf-gov chain) — N/A (untouched).
- No reverse-engineering of the absent Bifrost ref (ESC-REF-ABSENT) — and bf-core-1
  builds nothing, so there is nothing to reverse-engineer.
- bf-core-2 owns semantic cache (034-036) — bf-core-1 does NOT touch it.

**Matrix-flip (at close, in `.planning/parity/matrix/bifrost-core.md`):**
- **PAR-BF-CORE-002** → stays **HAVE** (already correct; `provider.go:76`
  postHookRunner on stream methods). Verify-only; add a cite if missing.
- **PAR-BF-CORE-027** → stays **PARTIAL** with a reconciliation note: the
  **CountTokens** request surface is SAT (shipped by bf-openai-4 — interface method
  `provider.go:107` + openai impl `counttokens.go:23`); the broader 20+ request-type
  **constant enum** is VAR/ESC by design (D4 — g0router uses string literals +
  dedicated handlers; the rest enumerate ESC surfaces). Cite bf-core-1 + D4.
- **PAR-BF-CORE-028** → stays **PARTIAL** with a variant/ESC note: g0router's
  `GatewayContext` is RequestID-only by deliberate design (D2); the typed-KV /
  request-metadata surface has no live consumer and is plugin-pipeline + tracing
  coupled (both ESC). Cite bf-core-1 + D2.
- **PAR-BF-CORE-019** → **variant-HAVE** note: g0router has account-level fallback
  (`selection.go:334` `WithAccountFallback`; `errorclass.go` Classify; `runner.go`
  RunModel); the per-attempt-PreLLMHook fallback chain is ESC (plugin-coupled).
  Cite bf-core-1 + D3.
- **PAR-BF-CORE-020** → **variant-HAVE** note: g0router's fallback decision is
  Verdict/StatusCode-driven; an `AllowFallbacks` flag on `ProviderError` would be a
  dead field and is NOT added (D3). The Bifrost `AllowFallbacks`-on-`BifrostError`
  mechanism is variant-divergent. Cite bf-core-1 + D3.
- **PAR-BF-CORE-001** → stays **PARTIAL** (the CountTokens subset is now SAT via
  bf-openai-4; the 14 ESC method groups — Rerank/OCR/Video*/Container*/
  CachedContent*/Passthrough/Compaction — remain ESC, no funded route). Cite
  bf-core-1 (subset closed) + the §3 ESC list.
- All other bf-core rows (003-018, 021-026, 029-050) → **unchanged MISSING/ESC**
  per BIFROST-MAP §8 (this plan does not touch them; bf-core-2 owns 034-036).

**`open-questions.md` (append at close):**
```
## bf-core-1 — provider-interface reconciliation — 2026-06-15
- [ ] Compaction (`/v1/responses/compact`, PAR-BF-CORE-001 subset / bf-openai-4 005) — ESC. No funded route → no interface method added (would be dead on 43 providers). Owner: the plan that funds the /v1/responses/compact route adds method+schema+impl together. Why: route-first prevents a dead interface method (Wave-5 NewWithShutdown lesson). Also ESC-REF-ABSENT for the wire shape.
- [ ] GatewayContext typed KV / request-metadata (PAR-BF-CORE-028) — ESC/VAR. No live consumer today; RequestID-only is g0router's minimal design. A field is added only by a future LIVE feature (request-metadata stamping / plugin / tracing) that reads it, in the same plan. Why: avoid a dead context field.
- [ ] AllowFallbacks flag on ProviderError (PAR-BF-CORE-020) — VAR; not added (dead flag). g0router's fallback is Verdict/StatusCode-driven (errorclass.go + selection.go + runner.go). Honoring a per-attempt flag would require rewiring the working fallback engine (non-additive). Why: variant-HAVE; per-attempt PreLLMHook mechanism is plugin-ESC.
- [ ] 20+ request-type constants (PAR-BF-CORE-027 enum half) — VAR; not added (dead constants; most enumerate ESC surfaces). g0router uses string literals + dedicated handlers.
- [ ] bifrost-core large escalation set (plugin pipeline, ProviderQueue, pooling, KVStore/clustering, CEL/adaptive-LB/health-states, OTEL/tracing, vector backends — rows 003-018,021-026,029-033,037-050) — ESC per BIFROST-MAP §8; revisit only if an operator funds the respective re-architecture.
```

**`docs/WORKFLOW.md` (update at close):** add a bf-core-1 row — provider-interface
reconciliation closed as **matrix-flip-only** (NO buildable additive code): the
CountTokens reconciliation was already shipped by bf-openai-4 (SAT); Compaction is
ESC (no funded route → no dead method); GatewayContext-KV (028) and the request-type
enum (027) have no live consumer (VAR/ESC); `AllowFallbacks` (019/020) is
variant-HAVE (g0router's account-level fallback is the working mechanism; a flag
would be dead). Rows 001(subset)/002/019/020/027/028 annotated per §7; the large
bifrost-core escalation set recorded in open-questions; the provider.go/errors.go
micro-serial with bf-openai-4 is discharged (bf-openai-4 merged); ESC-REF-ABSENT
honored (built nothing). No Go file changed.

**No-leftovers confirmation (binding):** bf-core-1 introduces NO provider interface
method, NO `GatewayContext` field, NO `ProviderError` flag, and NO request-type
constant — because §0/§1 prove each would be dead code with no live consumer
(violating BIFROST-MAP:392 + §3 CLI_ORCHESTRATOR). The genuinely-buildable
reconciliation (CountTokens) was already shipped by bf-openai-4 (SAT). A
near-empty, matrix-flip-only closeout is therefore the correct, honest outcome:
the no-dead-method proof (§5) holds vacuously because no method/field/flag is added.
```
