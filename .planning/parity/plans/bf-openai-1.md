# Micro-plan bf-openai-1 — Text completions (`/v1/completions` +stream) + models-get filter fix (Go)

```
program: bifrost-parity (bifrost phase — BUILDABLE-ADDITIVE only; the ~50%
  re-architecture is permanently deferred per BIFROST-MAP §1/§8 ESC set)
plan: bf-openai-1
status: READY (rev 1 — authored against the live tree @ <base>; BIFROST-MAP
  micro-plan index row ~line 296; bifrost-openai disposition §210-237;
  architectural decisions §72-200; serial chain §323-351)
runs: OpenAI-surface track. HOLDS the internal/server/routes_openai.go SERIAL
  SLOT while live (decision 3). Serial chain:
  **bf-openai-1** → bf-openai-2 → bf-openai-3 → bf-openai-4 (each appends /v1/*).
  Disjoint from the governance / mcp / core tracks (run ∥).
branch: main (direct commits, per AGENTS.md "No PR workflow")
commit-prefix: phase-1/bf-openai-1:
ref-source: ESC-REF-ABSENT (BIFROST-MAP §47-68) — the frozen Bifrost ref
  (@ca21298) is NOT on this host. The matrix rows + g0router's own conventions
  are the ONLY ground truth. /v1/completions is a documented, stable OpenAI
  legacy endpoint, so its wire shape is g0router's own schema
  (internal/schemas/completions.go) + OpenAI's public spec — NOT a guessed
  Bifrost internal. No Bifrost handler internals are reconstructed.
base: <base> = git rev-parse HEAD recorded at P0. Substitute the actual SHA
  everywhere §5 says <base>.
go-serial-slot: this plan holds the SINGLE in-flight edit to
  internal/server/routes_openai.go while live (decision 3; static-before-{param}
  precedence, §97). Slot must be FREE at P5 before T-routes. RELEASE to
  bf-openai-2 on close.
new-route: API routes only. NO UI contract — /v1/* are OpenAI-compatible API
  routes (NOT the {data,error} admin envelope). No e2e, no UI, no mocks.
```

---

## 1. Scope — PAR rows + the two deliverables

### Rows this plan closes

| Row | Claim (matrix text) | Current state (evidence) | Target after bf-openai-1 |
|---|---|---|---|
| **PAR-BF-OAI-002** | `POST /v1/completions` registered with fasthttp (`bifrost-openai.md:13`) | MISSING — "Schema exists (`internal/schemas/completions.go:4-21`) but no route registered". Confirmed: no `/v1/completions` anywhere in `internal/` except the schema comment (P1 grep). | HAVE — route registered (`routes_openai.go`), handler dispatches to `provider.TextCompletion`, returns the OpenAI completions shape. |
| **PAR-BF-OAI-207** | Text completion streaming supported (`bifrost-openai.md:95`) | MISSING — "Provider stubbed (`stubs.go:13-15`)". Confirmed: `TextCompletionStream` returns `notImplemented("text_completion_stream")` (`internal/providers/openai/stubs.go:13-15`). | HAVE — `stream:true` dispatches to `provider.TextCompletionStream`, SSE-framed via the shared passthrough processor; `[DONE]` terminator. |
| **PAR-BF-OAI-019** | `GET /v1/models/{id}` registered with fasthttp (`bifrost-openai.md:30`) | PARTIAL — matrix note says "handler delegates to `List` with no filtering (`internal/api/models.go:51-53`)". **THIS NOTE IS STALE** (see §1.3): the live `Get` handler (`internal/api/models.go:449-482`) ALREADY filters to one model by id and 404s on miss, wired via `GetOrByKind` (`models.go:387-399`, registered `routes_openai.go:101`). | HAVE — verified by a hermetic test that `GET /v1/models/{id}` returns exactly one entry for a known model and 404 for an unknown one; add the missing regression test; flip PARTIAL→HAVE. If the live `Get` is found NOT to filter (it does, per the read), the fix is the filter — but the read shows it already filters. |

Matrix flips at closeout (§4 T-close), in `.planning/parity/matrix/bifrost-openai.md`:
- PAR-BF-OAI-002 → HAVE (cite the new route + handler).
- PAR-BF-OAI-207 → HAVE (cite the implemented `TextCompletionStream` + handler stream path).
- PAR-BF-OAI-019 → HAVE (cite `models.go:449-482` + the new regression test).

### 1.1 The OpenAI-shape vs admin-envelope decision (BINDING — decision per AGENTS.md + BIFROST-MAP §118/§224)

**`/v1/*` routes return OpenAI shapes, NOT the `{data,error}` admin envelope.**
This is g0router's existing, verified convention on every `/v1/*` route:

- Success bodies are the bare OpenAI object (e.g. chat returns the marshalled
  `*schemas.ChatResponse` directly — `internal/api/chat.go:477-479`; embeddings
  returns the bare `*schemas.EmbeddingResponse` — `internal/api/embeddings.go:119-121`;
  models returns bare `ListModelsResponse`/`ModelEntry` — `internal/api/models.go:309-311,474-476`).
- Error bodies use the api-layer `writeError(ctx, status, errType, message, code)`
  (`internal/api/errors.go:18`) which emits the OpenAI `{"error":{...}}` shape —
  NOT `internal/admin/respond.go`'s `{data,error}`. The api package does not import
  `internal/admin`.

Therefore the completions handler:
- On success writes the bare `*schemas.TextCompletionResponse`
  (`internal/schemas/completions.go:24-31`) as JSON (object `"text_completion"`),
  mirroring `embeddings.go:119-121`.
- On error calls `writeError(...)` (the OpenAI error shape), mirroring
  `embeddings.go:60,66,80,107`.
- Streaming uses `text/event-stream` + the shared
  `translation.ProcessPassthroughStream` SSE framing + `data: [DONE]`, mirroring
  `chat.go:416-449`.

The `{data,error}` admin envelope is **FORBIDDEN** on this route (§ NOT in scope).
PAR-BF-OAI-301/302/303 (Bifrost's `BifrostError`/`is_bifrost_error`/`event_id`)
are a VARIANT-by-design escalation (BIFROST-MAP §224) and ride bf-openai-4 if at
all — NOT this plan.

### 1.2 The TextCompletion implementation approach (BINDING — soundness analysis)

The MAP marks 002/207 BUILD because the schema + interface method already exist;
the gap is "wire the route over the stubbed provider method". But the openai
provider method is a **real stub** that returns 501
(`internal/providers/openai/stubs.go:9-15` → `notImplemented("text_completion")`).
Two sound options were considered; the recommended default is **Option A**.

**Option A (RECOMMENDED — implement TextCompletion/TextCompletionStream for the
openai provider).** OpenAI's legacy `POST /v1/completions` is a real, documented,
stable upstream endpoint. The implementation is a near-verbatim copy of the
existing, tested transport patterns:
- `TextCompletion` mirrors `internal/providers/openai/embedding.go:12-67` exactly:
  `p.client.AcquireRequest/Response`, `req.SetRequestURI(p.baseURL + "/v1/completions")`,
  `req.Header.SetMethod(fasthttp.MethodPost)`, `utils.SetAuthHeader(req, key.Value)`
  (`internal/providers/utils/helpers.go:30`), `utils.SetJSONBody(req, request)`
  (`helpers.go:11`), `p.client.Do`, status check → `p.errorConverter.Convert(...)`,
  `utils.ReadJSONBody(resp, &result)` (`helpers.go:22`) into a
  `schemas.TextCompletionResponse`. `RequestType: "text_completion"` in `ErrorMeta`.
- `TextCompletionStream` mirrors `internal/providers/openai/chat.go:77-166`:
  set `streamReq.Stream = true`, POST to `/v1/completions`, then the goroutine
  drains via `utils.NewSSEScanner` (`chat.go:136`), `[DONE]` terminates, malformed
  chunk → `streamError(...)` (AUD-045 parity), post-hook honored (AUD-047). Uses
  the same `chan *schemas.StreamChunk` (legacy completion stream chunks carry
  `choices[].text` deltas; `StreamChunk` already models `choices[].delta`/`text`
  — confirm the field at impl; if `StreamChunk` lacks a `text` delta field, the
  passthrough still forwards the raw chunk shape, so no schema change is needed —
  the handler forwards bytes via `ProcessPassthroughStream`).

  This is **sound** because: it is the identical transport the openai provider
  already ships for chat/embeddings; it is hermetically testable with
  `httptest.NewServer` + `p.baseURL = srv.URL` (the exact pattern at
  `internal/providers/openai/stream_test.go:26-27`); and it makes NO claim about
  Bifrost internals (ESC-REF-ABSENT-safe).

  **Binding consequence:** implementing these two methods means the existing
  `TestNotImplementedStubs` sub-cases for `TextCompletion` + `TextCompletionStream`
  (`internal/providers/openai/openai_test.go:36-37`) WILL break (they assert 501).
  Those two sub-cases MUST be REMOVED from that table and REPLACED by the new
  hermetic success/error tests in a new `internal/providers/openai/completions_test.go`.
  The other 17 sub-cases stay untouched. This is the ONLY edit to an existing
  openai-provider test file (§3).

**Option B (FALLBACK — handler-only, surface the 501 cleanly; do NOT implement
the provider method).** Register the route + handler; the handler calls
`provider.TextCompletion`, which returns the existing `not_implemented`/501; the
handler maps that to `writeError(ctx, 501, "not_implemented", ...)`. This closes
PAR-BF-OAI-002 (route registered) but NOT 207 (streaming "supported" would still
501). Use this ONLY if, at impl, the openai legacy completions transport cannot be
made to pass hermetically (it can — it is identical to embeddings). If Option B is
taken, PAR-BF-OAI-207 STAYS MISSING and is escalated honestly in WORKFLOW.md (do
NOT mark 207 HAVE on a 501).

**Default: Option A.** Other 42 providers' `TextCompletion` stubs are NOT touched
(they keep returning 501; only the openai provider gains the real method — §3
FORBIDDEN list). The route works for any provider whose `TextCompletion` is
implemented; for the rest it surfaces their own 501 cleanly (no crash).

### 1.3 The models-get "fix" — stale matrix note reconciliation (BINDING)

The matrix note for PAR-BF-OAI-019 (`bifrost-openai.md:30`) cites
`internal/api/models.go:51-53` "delegates to `List` with no filtering" and
`routes_openai.go:18`. Both cites are STALE against the live tree:
- `GET /v1/models/{id}` is registered via `r.GET("/v1/models/{param}", models.GetOrByKind)`
  (`internal/server/routes_openai.go:101`), and `GetOrByKind` (`models.go:387-399`)
  dispatches non-kind params to `Get` (`models.go:449-482`).
- `Get` ALREADY filters to a single model by id (loops the catalog, returns the
  one matching `ModelEntry`, else `writeError(... 404 "model not found")`).

So 019 is, on the live tree, **already-HAVE in behavior** but mislabeled PARTIAL
with no regression test pinning it. The "fix" this plan ships is therefore:
1. A hermetic regression test (`internal/api/models_test.go` ADDITIVE) asserting
   `GET /v1/models/{id}` returns exactly one entry for a known catalog model and
   404 for an unknown id, AND that it does NOT return the full list.
2. The matrix flip PARTIAL→HAVE with the corrected cite.

If — and ONLY if — the test surfaces that `Get` does not actually filter (it does,
per the read at `models.go:449-482`), the plan adds the filter. The read is
unambiguous: no filter code change is expected; this is a verify + pin + flip.
No new file for models-get; the regression test is appended to the existing
`models_test.go`.

### 1.4 Completions Go contract (NEW, TDD)

**Schema (REUSE — already exists, no change expected):**
`internal/schemas/completions.go` provides `TextCompletionRequest`
(`:4-21`), `TextCompletionResponse` (`:24-31`), `TextCompletionChoice` (`:34-39`),
`Logprobs` (`internal/schemas/chat.go:112`), `Usage` (existing). If a field is
missing for the OpenAI completions response (e.g. `Object` defaulting to
`"text_completion"`), set it in the provider, NOT via a schema edit, unless a
genuinely-absent field forces an additive struct field (additive-only, decision 2).

**Provider transport (Option A — NEW file `internal/providers/openai/completions.go`):**

| Method | Mirrors | Behavior |
|---|---|---|
| `TextCompletion(ctx, key, *TextCompletionRequest) (*TextCompletionResponse, *ProviderError)` | `embedding.go:12-67` | POST `p.baseURL + "/v1/completions"`; auth header; JSON body; status check → `errorConverter.Convert`; read into `TextCompletionResponse`. `RequestType: "text_completion"`. |
| `TextCompletionStream(ctx, postHookRunner, key, *TextCompletionRequest) (chan *StreamChunk, *ProviderError)` | `chat.go:77-166` | `streamReq.Stream = true`; POST `/v1/completions`; goroutine drains `utils.NewSSEScanner`; `[DONE]` terminates; malformed → `streamError` (AUD-045); post-hook honored (AUD-047). `RequestType: "text_completion_stream"`. |

These REPLACE the two stubs in `internal/providers/openai/stubs.go:9-15`
(delete those two funcs from stubs.go; move them — now implemented — to
completions.go). The 17 other stubs in stubs.go are UNTOUCHED.

**Handler (NEW file `internal/api/completions.go`):** `CompletionsHandler`, mirrors
`EmbeddingsHandler` (`internal/api/embeddings.go`) for non-stream + `ChatHandler`
(`internal/api/chat.go:416-449`) for the stream path. Resolver seam =
`Resolve(model string)` (the embeddings-style `completionsResolver` interface,
mirroring `embeddings.go:22-27`; `inference.Router.Resolve` satisfies it,
`internal/inference/router.go:64`).

| Aspect | Contract |
|---|---|
| Route | `POST /v1/completions` |
| Parse | `json.Unmarshal(raw, &schemas.TextCompletionRequest)`; invalid JSON → `writeError(400, "invalid_request_error", "invalid JSON body", nil)` (mirror `embeddings.go:59-62`). |
| Resolve | `h.router.Resolve(req.Model)`; error → `writeError(400, "invalid_request_error", err.Error(), nil)`. |
| VK gate | `x-g0-vk` gate + pinned-key override, mirroring `embeddings.go:70-90` (REUSE `h.vkGate` / `h.pinnedResolver` seams + `NewVKGate`). Apply BEFORE dispatch. |
| Usage glue | OPTIONAL: wire `usageRecorder`/`pendingTracker`/`detailCapture` via the same additive setters as embeddings (`embeddings.go:34-43`). Default: include the setters + `recordGlue()` so `routes_openai.go` can wire usage symmetrically. record under endpoint `"/v1/completions"`. |
| Non-stream dispatch | `provider.TextCompletion(gatewayCtx, key, &req)`; on `*ProviderError` → `writeError(perr.StatusCode|502, perr.Type, perr.Message, perr.Code)` (mirror `embeddings.go:101-108`); on success marshal bare `*TextCompletionResponse` via `jsonMarshal` → 200 `application/json` (mirror `embeddings.go:111-121`). |
| Stream dispatch | when `req.Stream`: set `text/event-stream` + `Cache-Control: no-cache` + `Connection: keep-alive` (mirror `chat.go:417-419`); `provider.TextCompletionStream(gatewayCtx, nil, key, &req)`; pre-stream `*ProviderError` → `writeError`; else `writeSSEStream(streamCtx, ctx, ch)` (`chat.go:35-38`) with `withRequestCancel` (`chat.go:53-58`). |
| Marshal failure | fall back to plain-text 500 "internal error" (mirror `embeddings.go:112-118`, AUD-010 contract). |

**Construction:** `NewCompletionsHandler(router *inference.Router) *CompletionsHandler`
(mirror `NewEmbeddingsHandler`, `embeddings.go:30-32`). NO `New(...)`/
`RegisterOpenAIRoutes(...)` signature change beyond the additive symmetry already
present (decision 9) — the handler is constructed INSIDE `RegisterOpenAIRoutes`
like `embeddings := api.NewEmbeddingsHandler(router_)` (`routes_openai.go:40`).

### 1.5 routes_openai.go registration (serial-slot additive, §3)

Construct + wire the handler alongside the existing ones, and append ONE route
line. Static-before-`{param}` is irrelevant here (`/v1/completions` is a distinct
static path), but the append MUST stay grouped with the other `/v1/*` POSTs
(`routes_openai.go:95-98`):

```go
// (in the handler-construction block, mirroring embeddings)
completions := api.NewCompletionsHandler(router_)
if recorder != nil { completions.SetUsageRecorder(recorder) }
if tracker != nil { completions.SetPendingTracker(tracker) }
if detail != nil { completions.SetDetailCapture(detail) }
if st != nil {
    completions.SetVKGate(vkGate)            // reuse the vkGate built at :77
    completions.SetVKPinnedResolver(selector) // reuse the selector built at :84
}

// (in the route block, after :95-98)
r.POST("/v1/completions", completions.Handle)
```

The `vkGate`/`selector` are already constructed in the `if st != nil` block
(`routes_openai.go:77,84`); reuse them — do NOT rebuild. The new construction +
wiring is additive; no existing line is deleted.

### NOT in scope (explicit — FORBIDDEN)

- **The ESC rows.** No responses rewrite, no normalization layer
  (PAR-BF-OAI-101..118), no rerank/ocr (039/040), no video/containers/async/WS,
  no raw-passthrough (205). Touching any ESC surface is an automatic REJECT.
- **Other bf-openai plans' surfaces.** No audio/images (bf-openai-2), no
  files/batches (bf-openai-3), no responses-extras/SSE-correctness/compaction
  (bf-openai-4). Do not touch their schemas/handlers/stubs.
- **The `{data,error}` admin envelope on `/v1/*`** (§1.1) — FORBIDDEN. No import
  of `internal/admin` from `internal/api`.
- **The 17 other provider stubs** in `internal/providers/openai/stubs.go`
  (Responses/Image*/Speech/Transcription/File*/Batch*/CountTokens) — untouched.
- **The other 42 providers' `TextCompletion`** — untouched (they keep their 501).
- **All UI / e2e / mocks** — these are API routes with NO UI contract. No
  `ui/**` edit, no playwright, no mock/seed.
- **No `New(...)`/`RegisterOpenAIRoutes(...)` signature change**, no `init()`, no
  global state, errors-as-values (`fmt.Errorf("ctx: %w")`), no panics.
- **No destructive DDL** (this plan touches no store/migrate — there is no DB
  surface in completions/models-get).

---

## 2. Precondition checks

Run all before any edit; abort and report to the orchestrator on any failure.

```bash
# P0 — clean tree; record <base>
git status --porcelain     # must be empty (explicit `git add <file>`, never -A)
git rev-parse HEAD         # record as <base> for §5

# P1 — the completions gap is REAL (no route/handler anywhere)
grep -rn '/v1/completions' internal/ | grep -v _test.go
# ^ expect ONLY internal/schemas/completions.go (the schema comment). No route, no handler.
test ! -e internal/api/completions.go && echo "api handler gap OK"
test ! -e internal/providers/openai/completions.go && echo "provider impl gap OK"
grep -n 'notImplemented("text_completion")' internal/providers/openai/stubs.go  # the two stubs to replace (:9-15)

# P2 — reused surfaces present (the de-risk)
grep -n 'type TextCompletionRequest\|type TextCompletionResponse\|type TextCompletionChoice' internal/schemas/completions.go
grep -n 'type Logprobs' internal/schemas/chat.go
grep -n 'func (r \*Router) Resolve\b' internal/inference/router.go            # :64
grep -n 'func SetAuthHeader\|func SetJSONBody\|func ReadJSONBody\|func NewSSEScanner' internal/providers/utils/*.go
grep -n 'func writeError\|func jsonMarshal\|func writeSSEStream\|func withRequestCancel' internal/api/*.go
grep -n 'func NewEmbeddingsHandler\|func NewVKGate\|func requestHeadersFromCtx' internal/api/*.go
grep -n 'func streamError' internal/providers/openai/*.go

# P3 — the models-get behavior is already correct (verify the stale matrix note)
grep -n 'func (h \*ModelsHandler) Get\b\|func (h \*ModelsHandler) GetOrByKind' internal/api/models.go  # :449,:387
grep -n '/v1/models/{param}' internal/server/routes_openai.go                 # :101 (wired to GetOrByKind)

# P4 — provider transport templates present (Option A)
grep -n 'func (p \*Provider) Embedding\b' internal/providers/openai/embedding.go   # :12
grep -n 'func (p \*Provider) ChatCompletionStream' internal/providers/openai/chat.go  # :77
grep -n 'p.baseURL = srv.URL' internal/providers/openai/stream_test.go        # hermetic pattern :27
grep -n 'TextCompletion\b\|TextCompletionStream' internal/providers/openai/openai_test.go  # :36-37 (the two sub-cases to remove)

# P5 — routes_openai.go SERIAL SLOT is FREE
git log --oneline -5 -- internal/server/routes_openai.go
# Orchestrator MUST confirm no concurrent bf-openai-* plan holds an unmerged
# routes_openai.go edit before bf-openai-1 begins T-routes. bf-openai-1 is FIRST
# in the chain (§323-351). bf-openai-1 TAKES the slot, RELEASES to bf-openai-2.

# P6 — green at base (HERMETIC; no network)
go test ./... && go vet ./... && go build ./...     # exit 0 (untouched-green baseline)
```

---

## 3. Exclusive file ownership

After bf-openai-1 merges, CREATE files are owned by bf-openai-1; later plans
consume, never edit (decision 7).

**CREATE — provider transport (NEW, Option A):**

| File | Contract |
|---|---|
| `internal/providers/openai/completions.go` | `TextCompletion` + `TextCompletionStream` (Option A, §1.4) — moved from stubs.go, now implemented; mirror `embedding.go` + `chat.go` transport. No `init()`; errors-as-values; `RequestType: "text_completion"`/`"text_completion_stream"`. |
| `internal/providers/openai/completions_test.go` | RED first. Hermetic `httptest.NewServer` + `p.baseURL = srv.URL` (mirror `stream_test.go:16-48`): `TextCompletion` success → maps body to `TextCompletionResponse` (`choices[].text`); upstream-500 → `*ProviderError` with status 500; `TextCompletionStream` yields N content chunks then `[DONE]`; malformed chunk → one error chunk + abort (AUD-045 parity). NO real network. |

**CREATE — api transport (NEW):**

| File | Contract |
|---|---|
| `internal/api/completions.go` | `CompletionsHandler` + `NewCompletionsHandler` + `completionsResolver` seam + the additive setters (VK/usage) + `Handle` (non-stream + stream), §1.4. OpenAI shapes only (§1.1); `writeError` for errors; bare `*TextCompletionResponse` for success. |
| `internal/api/completions_test.go` | RED first. Hermetic, fake provider/resolver mirroring `embeddings_test.go:14-34` (`fakeCompletionsResolver` returns a fake provider implementing `TextCompletion`/`TextCompletionStream`; embed a shared base to satisfy the full `schemas.Provider` interface like `fakeEmbeddingsProvider` embeds `fakeMessagesProvider`): non-stream success → bare OpenAI body (assert NO `data`/`error` wrapper key); invalid JSON → 400 OpenAI error shape; provider 501 → 501 passthrough; stream → `text/event-stream` + `[DONE]`; VK-denied → 429 and provider NOT called (mirror `embeddings_test.go:67-99`); VK-pinned key override (mirror `embeddings_test.go:103-137`); marshal failure → plain 500 (mirror `embeddings_test.go:42-65`). |

**EXTEND — provider stubs (REMOVE the two now-implemented stubs):**

| File | Change |
|---|---|
| `internal/providers/openai/stubs.go` | DELETE `TextCompletion` (`:9-11`) + `TextCompletionStream` (`:13-15`) ONLY (they move to completions.go, implemented). The 17 other stubs + `notImplemented` helper are PRESERVED verbatim. |
| `internal/providers/openai/openai_test.go` | REMOVE the `TextCompletion` + `TextCompletionStream` sub-cases from the `TestNotImplementedStubs` table (`:36-37`) — they no longer 501. The other 17 sub-cases are PRESERVED. (This is the ONLY edit to a pre-existing openai test.) |

**EXTEND — models-get regression test (ADDITIVE):**

| File | Change |
|---|---|
| `internal/api/models_test.go` | ADD a hermetic test pinning `GET /v1/models/{id}`: known catalog model → exactly one `ModelEntry` (not the full list), 404 on unknown id. Drives `ModelsHandler.Get`/`GetOrByKind` directly with a `fasthttp.RequestCtx` + `SetUserValue("param", id)` (mirror existing models_test.go patterns). RED-then-green ONLY if `Get` doesn't already filter (it does — so this is a green-confirming regression test; commit it after confirming it passes, noting in WORKFLOW that 019 was already-HAVE behaviorally). |

**MODIFY — serial-slot route registration (additive only):**

| File | Change |
|---|---|
| `internal/server/routes_openai.go` | ADD the `completions` handler construction + wiring (reuse `vkGate`/`selector`/`recorder`/`tracker`/`detail`) + the ONE `r.POST("/v1/completions", completions.Handle)` line (§1.5). NOTHING else changes. SERIAL SLOT — only holder while live; RELEASE to bf-openai-2 on close. |

**FORBIDDEN:** everything else. Explicitly: the 17 other openai stubs; the 42
other providers' `TextCompletion`; all other `internal/api/*.go`
(chat/embeddings/messages/responses/models bodies — models_test.go is the only
ADD); `internal/schemas/*` (REUSE completions.go; edit ONLY if a genuinely-absent
field forces an additive struct field, §1.4); all bf-openai-2/3/4 surfaces; all
`internal/admin/*`, `internal/store/*`, `internal/governance/*`, `internal/mcp/*`;
all `ui/**`. Touching any of these is an automatic REJECT.

---

## 4. TDD tasks

Cadence (strict, AGENTS.md "TDD always"): **no Go impl file may exist before its
`_test.go` is committed RED.** `go test ./... && go vet ./... && go build ./...`
green at EVERY commit (a RED commit may fail ONLY the new package's targeted run;
prefer table/assertion failures over compile failures — scaffold the signatures so
the package compiles and the assertion fails). Order: provider impl → api handler
→ models-get regression → serial-slot route → closeout.

### T-provider — STEP(a) RED, STEP(b) impl
STEP(a): write `internal/providers/openai/completions_test.go` (hermetic
httptest, §3). Run `go test ./internal/providers/openai/ -run Completion` → FAIL
(impl missing). Commit RED:
`phase-1/bf-openai-1: failing openai TextCompletion(+stream) tests (TDD red)`.
STEP(b): create `internal/providers/openai/completions.go` (Option A); DELETE the
two stubs from `stubs.go`; REMOVE the two sub-cases from `openai_test.go`. Gates:
`go test ./... && go vet ./... && go build ./...` green. Commit:
`phase-1/bf-openai-1: implement openai TextCompletion + TextCompletionStream`.

*If Option A cannot pass hermetically (it should — identical to embeddings),
STOP and ESCALATE (§8 ESC-TEXTCOMP); fall to Option B for 002 only, leave 207
MISSING. Do NOT fabricate a green.*

### T-handler — STEP(a) RED, STEP(b) impl
STEP(a): write `internal/api/completions_test.go` (fake provider/resolver, §3).
Run `go test ./internal/api/ -run Completion` → FAIL. Commit RED:
`phase-1/bf-openai-1: failing /v1/completions handler tests (TDD red)`.
STEP(b): create `internal/api/completions.go` (§1.4). Gates green. Commit:
`phase-1/bf-openai-1: /v1/completions handler (non-stream + SSE, OpenAI shape)`.

### T-modelsget — regression test + flip
Add the `GET /v1/models/{id}` regression test to `internal/api/models_test.go`
(§3). Run `go test ./internal/api/ -run Models` → green (Get already filters,
§1.3). If RED (Get doesn't filter — not expected), add the filter to `Get` and
re-run. Commit:
`phase-1/bf-openai-1: pin GET /v1/models/{id} single-model filter (regression)`.

### T-routes — serial-slot route registration
TAKE the serial slot (orchestrator confirms FREE at P5). Add the construction +
wiring + the ONE `r.POST("/v1/completions", ...)` line to `routes_openai.go`
(§1.5). Gates: `go test ./... && go vet ./... && go build ./...` green. Commit
(ONE commit touches the serial file):
`phase-1/bf-openai-1: register POST /v1/completions (serial slot)`.

### T-close — full gates + closeout
```bash
go test ./... && go vet ./... && go build ./...
go test ./internal/api/... ./internal/server/... -run 'Completion|Models' -v
go test ./internal/providers/openai/ -run 'Completion|NotImplemented' -v
```
Flip `.planning/parity/matrix/bifrost-openai.md`: PAR-BF-OAI-002 → HAVE,
PAR-BF-OAI-207 → HAVE (Option A) / STAYS MISSING + escalated (Option B),
PAR-BF-OAI-019 → HAVE; correct the stale 019 cite. Update `docs/WORKFLOW.md`
(P6 base observation, the Option A vs B decision actually taken, the OpenAI-shape
decision, the 019 stale-note reconciliation, the serial-slot take/release).
Append any open items to `.planning/parity/plans/open-questions.md`. Final commit:
`phase-1/bf-openai-1: close — completions HAVE; models-get HAVE; matrix flip`.
**On the close commit, RELEASE the routes_openai.go serial slot to bf-openai-2.**

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0. Diff gate is
**bf-openai-1 commit-range-scoped** (§7).

**Test gates (HERMETIC — no network)**
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/api/... ./internal/server/... -run 'Completion|Models' -v`
  → exit 0, all pass (completions: non-stream + stream + invalid-JSON + provider-err
  + VK-denied + VK-pinned + marshal-fail; models-get: single + 404).
- `go test ./internal/providers/openai/ -run 'Completion|NotImplemented' -v`
  → exit 0 (TextCompletion success/err + stream chunks/abort; the 17 remaining
  NotImplemented sub-cases still pass).

**TDD-order proof** — each impl file's covering test appears in an earlier-or-equal commit:
```bash
for pair in \
  "internal/providers/openai/completions_test.go:internal/providers/openai/completions.go" \
  "internal/api/completions_test.go:internal/api/completions.go" ; do
  tf=${pair%%:*}; im=${pair##*:}
  ct=$(git log --format=%ct --diff-filter=A -1 -- "$tf")
  cf=$(git log --format=%ct --diff-filter=A -1 -- "$im")
  [ "$ct" -le "$cf" ] || echo "TDD VIOLATION: $im"     # prints nothing
done
```

**Grep proofs**
```bash
# route registered (serial-slot)
grep -n '/v1/completions' internal/server/routes_openai.go                    # the POST line
grep -n 'api.NewCompletionsHandler' internal/server/routes_openai.go
# handler exists, OpenAI shape (NOT admin envelope)
grep -n 'func (h \*CompletionsHandler) Handle' internal/api/completions.go
grep -n 'writeError\|jsonMarshal' internal/api/completions.go                 # OpenAI error + bare body
! grep -rn 'internal/admin' internal/api/completions.go && echo "no admin-envelope import OK"
! grep -n '"data"' internal/api/completions.go && echo "no {data} wrapper OK"
# provider method implemented (Option A) — no longer a stub
grep -n 'func (p \*Provider) TextCompletion\b\|func (p \*Provider) TextCompletionStream' internal/providers/openai/completions.go
grep -n 'p.baseURL + "/v1/completions"' internal/providers/openai/completions.go
! grep -n 'notImplemented("text_completion")' internal/providers/openai/stubs.go && echo "stubs removed OK"
# models-get filters to one model (verify the live behavior is pinned)
grep -n 'func (h \*ModelsHandler) Get\b' internal/api/models.go               # :449 (filters by id)
grep -n 'GetOrByKind\|/v1/models/{param}' internal/server/routes_openai.go    # wired :101
# no init(); errors-as-values
! grep -rn 'func init(' internal/api/completions.go internal/providers/openai/completions.go && echo "no init() OK"
! grep -rn 'panic(' internal/api/completions.go internal/providers/openai/completions.go && echo "no panic OK"
```

**SSE / OpenAI-shape proofs**
```bash
grep -n 'text/event-stream' internal/api/completions.go                       # stream content-type
grep -n 'writeSSEStream\|ProcessPassthroughStream' internal/api/completions.go internal/providers/openai/completions.go
grep -n '\[DONE\]\|streamError\|NewSSEScanner' internal/providers/openai/completions.go  # terminator + abort parity
```
Plus a runtime assertion in `completions_test.go`: the non-stream success body
unmarshals to a `TextCompletionResponse` with a populated `choices[].text` and
contains NEITHER a top-level `"data"` NOR a top-level `"error"` key (proves
OpenAI-shape, NOT admin-envelope).

**Negative / freeze proofs (bf-openai-1 commit-range — §7)**
```bash
R="<first-bf-openai-1>^..<last-bf-openai-1>"
# Only the sanctioned files changed:
git diff $R --name-only -- internal/ | grep -vE \
 'internal/providers/openai/(completions|completions_test|stubs|openai_test)\.go|internal/api/(completions|completions_test|models_test)\.go|internal/server/routes_openai\.go' \
 | wc -l                                                                       # = 0
# Other bf-openai stubs untouched (only the two text-completion stubs removed):
git diff $R -- internal/providers/openai/stubs.go | grep -E '^-' | grep -ivE '^---|TextCompletion|notImplemented\("text_completion' | grep -iE 'func \(p \*Provider\)' | wc -l   # = 0
# No other api handler body changed (models_test ADD is the only models touch):
git diff $R --name-only -- internal/api/ | grep -vE 'internal/api/(completions|completions_test|models_test)\.go' | wc -l   # = 0
# No store/admin/governance/mcp/ui touched:
git diff $R --name-only -- internal/store/ internal/admin/ internal/governance/ internal/mcp/ ui/ | wc -l   # = 0
# routes_openai.go = exactly ONE commit, additive (no route deletions):
git log --oneline $R -- internal/server/routes_openai.go | wc -l              # = 1
git diff $R -- internal/server/routes_openai.go | grep -E '^-' | grep -vE '^---|^-$' | wc -l   # = 0 (no deletions)
```

---

## 6. Out of scope (restated, binding)

No `{data,error}` admin envelope on `/v1/*` (§1.1 — FORBIDDEN). No ESC rows
(responses-rewrite/normalization/rerank/ocr/video/containers/async/WS/
raw-passthrough). No other bf-openai plans' surfaces (audio/images/files/batches/
responses-extras). No edits to the 17 other openai stubs or the 42 other
providers' TextCompletion. No schema rewrite (REUSE completions.go; additive field
ONLY if forced). No UI / e2e / mocks (API routes, no UI contract). No
`New(...)`/`RegisterOpenAIRoutes(...)` signature change, no `init()`, no global
state, no panics. No store/migrate (no DB surface). Matrix-vs-code contradiction
(e.g. the stale 019 note) → reconcile honestly in the plan + WORKFLOW (§1.3),
never fabricate. If Option A can't pass hermetically → escalate (§8), do not mark
207 HAVE on a 501.

## 7. Diff-gate scope

bf-openai-* plans commit to main concurrently (disjoint from gov/mcp/core, serial
within the openai track), so a broad `<base>..HEAD` range can sweep in siblings.
The diff gate MUST be scoped to bf-openai-1's own commits:
`git log --oneline main | grep "^[0-9a-f]* phase-1/bf-openai-1:" | awk '{print $1}'`
then `git diff <first-bf-openai-1>^..<last-bf-openai-1> -- [file list]`.

`git diff <first>^..<last> --name-only` must be exactly a subset of:
```
internal/providers/openai/completions.go
internal/providers/openai/completions_test.go
internal/providers/openai/stubs.go            (DELETE the two text-completion stubs only)
internal/providers/openai/openai_test.go      (REMOVE the two text-completion sub-cases only)
internal/api/completions.go
internal/api/completions_test.go
internal/api/models_test.go                   (ADD the models-get regression test only)
internal/server/routes_openai.go              (serial-slot additive; ONE commit)
.planning/parity/matrix/bifrost-openai.md
.planning/parity/plans/open-questions.md
docs/WORKFLOW.md
```
Any file outside this list in the scoped diff is an automatic review REJECT.
`internal/schemas/completions.go` is deliberately ABSENT (REUSE; additive-field
edit only if §1.4 forces it — then add it to the list with a WORKFLOW note). All
other api handlers, all other providers, store/admin/governance/mcp, and all
`ui/**` are ABSENT — touching them is an automatic REJECT. The `routes_openai.go`
edit must appear in exactly ONE commit (§5) and the serial slot is released to
bf-openai-2 on close.

## 8. Escalations / decisions (explicit — recommended defaults, do not fabricate)

- **ESC-TEXTCOMP (RESOLVED at authoring — TextCompletion implementation, binding
  default = Option A).** §1.2. OpenAI's legacy `/v1/completions` is a real,
  stable upstream endpoint; `TextCompletion`/`TextCompletionStream` are
  implementable for the openai provider as a near-verbatim copy of the existing,
  tested `embedding.go` + `chat.go` transport, hermetically testable via
  `httptest.NewServer` + `p.baseURL = srv.URL` (`stream_test.go:27`). This closes
  BOTH 002 and 207 soundly without any Bifrost-internal claim (ESC-REF-ABSENT-safe).
  **Fallback Option B** (handler surfaces the 501) closes only 002 and leaves 207
  MISSING — use ONLY if Option A cannot pass hermetically at impl. **Never mark
  207 HAVE while the method returns 501.** RECOMMENDED: Option A.

- **ESC-MODELSGET-STALE (RESOLVED at authoring — stale matrix note, binding).**
  §1.3. The PAR-BF-OAI-019 matrix note (`bifrost-openai.md:30`) cites
  `models.go:51-53` "delegates to List with no filtering" — STALE. The live `Get`
  (`models.go:449-482`) already filters to one model by id and 404s on miss, wired
  via `GetOrByKind` (`models.go:387-399`, route `routes_openai.go:101`). This plan
  ships a regression test pinning that behavior + flips 019 PARTIAL→HAVE with the
  corrected cite. NO filter code change is expected (the read is unambiguous). If
  the regression test surprises and shows no filter, add the filter — but do not
  fabricate; record the actual finding in WORKFLOW.

- **ESC-OPENAI-SHAPE (RESOLVED at authoring — envelope decision, binding).** §1.1.
  `/v1/*` routes return OpenAI shapes (bare success object + OpenAI `{"error":{}}`),
  NOT the `{data,error}` admin envelope. Verified across chat/embeddings/models.
  The admin envelope is FORBIDDEN here. Bifrost's `BifrostError`/`event_id`/`param`
  (301/302/303) is a VARIANT escalation (BIFROST-MAP §224) — NOT this plan.

- **ESC-STREAMCHUNK-FIELD (CONDITIONAL — at impl).** Legacy completion stream
  chunks carry `choices[].text` deltas (vs chat's `choices[].delta`). If
  `schemas.StreamChunk` cannot represent a `text` delta, the passthrough
  (`ProcessPassthroughStream`) still forwards the raw chunk bytes, so NO schema
  change is needed. Only if a typed-decode path requires it, add an additive
  `text` field to the relevant stream-choice struct (additive-only, decision 2) —
  decide at T-provider; default = forward raw, no schema edit.

- **Serial-slot dependency (§1.5 / P5).** bf-openai-1 is FIRST in the
  routes_openai.go serial chain (§323-351); it TAKES the slot and RELEASES it to
  bf-openai-2 on close. Orchestrator confirms exactly one unmerged holder
  (decision 3) before T-routes.

- **No other blocking dependency.** All reused surfaces (completions schema,
  Router.Resolve, utils helpers, writeError/jsonMarshal/writeSSEStream,
  NewVKGate, embeddings/chat templates, httptest hermetic pattern) are in-tree at
  <base> (P2/P4). bf-openai-1 is unblocked once the serial slot is free.
```
