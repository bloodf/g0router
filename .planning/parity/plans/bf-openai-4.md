# Micro-plan bf-openai-4 — Responses extras (`/v1/responses/input_tokens` CountTokens) + SSE correctness + error-field reconciliation (Go)

```
program: bifrost-parity (bifrost phase — BUILDABLE-ADDITIVE only; the ~50%
  re-architecture is permanently deferred per BIFROST-MAP §1/§8 ESC set)
plan: bf-openai-4
status: READY (rev 1 — authored against the LIVE tree @ <base>; BIFROST-MAP
  micro-plan index row line 299; bifrost-openai disposition rows 004/005/201/202/
  203/204/301/302/303/304 (.planning/parity/matrix/bifrost-openai.md:15-16,89-92,
  107-110) + the SAT/flip rows 003/208 (matrix:14,96); serial chain BIFROST-MAP:
  324,341)
runs: OpenAI-surface track. HOLDS the internal/server/routes_openai.go SERIAL
  SLOT while live (decision 3). Serial chain:
  bf-openai-1 (SHIPPED) → bf-openai-2 (SHIPPED) → bf-openai-3 (SHIPPED) →
  **bf-openai-4** (LAST in the chain — appends /v1/responses/input_tokens, fixes
  SSE setup ordering, surfaces the existing APIError.Param). Disjoint from the
  governance / mcp / core tracks (run ∥), EXCEPT the provider.go/errors.go
  MICRO-SERIAL with bf-core-1 (BIFROST-MAP:348-350) — see §0 coordination note.
branch: main (direct commits, per AGENTS.md "No PR workflow")
commit-prefix: phase-1/bf-openai-4:
ref-source: ESC-REF-ABSENT (BIFROST-MAP §47-68) — the frozen Bifrost ref
  (@ca21298) is NOT on this host. The matrix rows + g0router's own conventions
  are the ONLY ground truth. `/v1/responses/input_tokens` is a documented OpenAI
  Responses-API endpoint whose g0router shape is its OWN schemas
  (internal/schemas/responses.go + TokenCountResponse) + the existing responses
  translation path — NOT a reconstructed Bifrost handler internal.
base: <base> = `git rev-parse HEAD` recorded at P0. Substitute the actual SHA
  everywhere §5 says <base>.
go-serial-slot: this plan holds the SINGLE in-flight edit to
  internal/server/routes_openai.go while live (decision 3). bf-openai-3 RELEASED
  the slot on its close; bf-openai-4 TAKES it. Slot must be FREE at P5 before
  T-routes. bf-openai-4 is the FINAL holder of the OpenAI serial chain — on close
  the chain is COMPLETE (no successor to release to).
new-route: API route only. NO UI contract — `/v1/responses/input_tokens` is an
  OpenAI-compatible API route (NOT the {data,error} admin envelope). No e2e, no
  UI, no mocks.
pattern: MIRRORS the SHIPPED bf-openai-1/2/3 (`.planning/parity/plans/
  bf-openai-1.md`, `bf-openai-2.md`, `bf-openai-3.md`): /v1-route + OpenAI-shape
  (not admin envelope) + provider-method-impl-over-stub (Option A). bf-openai-1
  implemented TextCompletion over the stub; bf-openai-4 implements CountTokens
  the SAME way + reuses the SHIPPED responses-translation path for the
  responses→chat shape.
```

---

## 0. The big finding — read before everything else

Three of the rows the BIFROST-MAP routed to bf-openai-4 are **already built** in
the LIVE tree (the MAP index was authored before / independent of the SHIPPED
bf-openai-1/2/3 work, and overstated the buildable surface for one row). The plan
therefore splits into FOUR honest dispositions, each grounded at file:line:

| Surface | MAP said | LIVE reality (evidence) | bf-openai-4 disposition |
|---|---|---|---|
| `/v1/responses` route + responses streaming (003, 208) | MISSING / stubbed | **HAVE** — route registered `routes_openai.go:127`; handler `internal/api/responses.go:56-148` translates responses→chat and streams via `ChatCompletionStream` + `ProcessTranslateStream`; tested `internal/api/responses_test.go:146,183`. | **FLIP to HAVE** (regression test already green — assert + cite, no code). |
| `event: <type>` SSE typing for responses streams (203) | MISSING | **HAVE for responses** — `FormatSSE(FormatOpenAIResponses, …)` emits `event: <name>\ndata: <json>\n\n` (`internal/translation/sse.go:144-153`); `responses_test.go:203,206,209` already assert `event: response.created`/`response.output_text.delta`/`response.completed`. | **FLIP to HAVE for the responses path**; the image-gen-stream half of the row is bf-openai-2's passthrough domain → VARIANT (§8 ESC-IMG-EVENT-TYPE). |
| `[DONE]` terminator on responses streams (204) | MISSING | **HAVE** — `ProcessTranslateStream` writes `data: [DONE]\n\n` (`internal/translation/stream.go:136`); `responses_test.go:214` asserts it. The 204 *negative* clause (skip `[DONE]` when `includeEventType`/`skipDoneMarker`) is Bifrost-specific (no g0router toggle). | **FLIP the positive half to HAVE**; the skip-toggle is VARIANT-by-design (§8 ESC-DONE-SKIP). |
| `/v1/responses/input_tokens` (CountTokens) (004) | MISSING / stub | **stub** — `CountTokens` 501 (`internal/providers/openai/stubs.go:17-19`); NO route; the translation path to reach it exists (`responses.go:70`). | **BUILD** (route + handler + openai CountTokens impl over the stub). |
| SSE-setup-ordering bug (201) | MISSING (real bug) | **real bug** — headers set `text/event-stream` (`chat.go:417-419`) BEFORE the stream-open error check (`chat.go:436-439` → `writeError` emits JSON under the SSE content-type); same in `responses.go:128-136`. | **BUILD** (fix the ordering; provider-open errors return JSON with the correct content-type). |
| SSE error frames mid-stream (304) | MISSING | mid-stream errors abort the channel (`stream.go:60-61`) and are *logged* (`chat.go:445-447`), not re-framed as `event: error`. | **BUILD** the SSE-open-error JSON fix (= 201); the mid-stream `event: error` *re-frame* is VARIANT (§8 ESC-SSE-MIDSTREAM-FRAME — the channel is already torn down; re-framing is a passthrough-processor change coupled to the responses-rewrite). |
| `/v1/responses/compact` (Compaction) (005) | BUILD ("interface method exists") | **FALSE premise** — there is NO `Compaction` method on the `Provider` interface (`internal/schemas/provider.go:69-108` — grep-confirmed absent), NO `Compaction`/`Compact*` schema anywhere (`grep -rI Compaction internal/schemas/` → no match), NO stub. | **ESCALATE** (§8 ESC-COMPACTION) — non-additive interface change + behavior depends on the responses-rewrite ESC. STAYS MISSING. |
| `BifrostError`/`is_bifrost_error`/`event_id` envelope (301/302/303) | MISSING/PARTIAL | g0router's `{data,error}` (admin) + flat OpenAI `{"error":{…}}` (`internal/api/errors.go:18-38`) is CANONICAL by design (AGENTS.md). **`APIError.Param` ALREADY EXISTS** (`internal/schemas/errors.go:7` — the matrix cite "lacks Param" is STALE), but `writeError` does not surface it. `event_id`/`is_bifrost_error` are a different contract. | **VARIANT-record** (301/302/303 stay MISSING-by-design with a variant note). Optional tiny additive: surface the *existing* `APIError.Param` through `writeError` (§1.6, DECISION below). Do NOT restructure the envelope. |
| fasthttp internal-pipe bypass SSE reader (202) | MISSING (perf) | g0router writes directly to `ctx`; no `lib.NewSSEStreamReader` analog. Pure performance/transport-abstraction, not correctness. | **ESCALATE** (§8 ESC-SSE-PIPE) — optional perf, divergent transport. STAYS MISSING. |
| raw upstream-bytes passthrough (205) | MISSING | g0router always unmarshals/remarshals; cross-cutting transport change. | **ESCALATE** (§8, already MAP-ESC :235) — not in bf-openai-4 scope. STAYS MISSING. |

**Net for bf-openai-4:** BUILD = 004 + 201 (the real bug). FLIP = 003, 208, 203,
204 (positive halves). VARIANT = 301, 302, 303, 304 (the optional `param`-surface
is a sub-decision). ESCALATE = 005, 202, the negative/midstream halves of
203/204/304, and (already MAP-ESC) 205. This is the honest closeout of the
bifrost-openai surface.

### 0.1 provider.go / errors.go micro-serial with bf-core-1 (coordination — BINDING)

BIFROST-MAP:348-350 declares a micro-serial: both bf-core-1 and bf-openai-4 may
touch `internal/schemas/{provider,errors}.go`. **bf-openai-4 touches NEITHER of
those schema files** under this plan's chosen scope:
- It does NOT add a `Compaction` interface method to `provider.go` (ESCALATED — §8).
- It does NOT add `CountTokens` to `provider.go` (the method ALREADY exists at
  `provider.go:107` — bf-openai-4 only implements it in the openai package).
- The optional `param`-surface (§1.6) edits `internal/api/errors.go` (the
  *handler* writer), NOT `internal/schemas/errors.go` (the *struct*) — the struct
  already has `Param` (`schemas/errors.go:7`). **No `internal/schemas/errors.go`
  edit.**

Therefore bf-openai-4 needs **NO micro-serial coordination** with bf-core-1 on
the schema files. If, at impl, the executor discovers a genuinely-unavoidable
additive field on `schemas/errors.go` (it should not — see §1.6), STOP and
coordinate the edit window with bf-core-1 per the orchestrator before touching
`internal/schemas/errors.go`. Default: no schema-file edit at all.

---

## 1. Scope — PAR rows + the deliverables

### Rows this plan BUILDS (flips MISSING → HAVE via new code)

| Row | Claim (matrix text) | Current state (evidence) | Target after bf-openai-4 |
|---|---|---|---|
| **PAR-BF-OAI-004** | `POST /v1/responses/input_tokens` (count tokens) registered (`bifrost-openai.md:15`) | MISSING — "`CountTokens` stubbed in OpenAI provider". Confirmed: `CountTokens` → `notImplemented("count_tokens")` (`internal/providers/openai/stubs.go:17-19`); NO `/v1/responses/input_tokens` route (grep-confirmed). The reach path exists: the responses→chat translation (`internal/api/responses.go:70`) + `CountTokens(ctx, key, *ChatRequest)` (`provider.go:107`). | HAVE — route registered (`routes_openai.go`), handler translates the responses-shaped body → `*ChatRequest` (REUSE the SHIPPED responses translation, §1.3), dispatches to `provider.CountTokens`, returns the bare `*TokenCountResponse` JSON (`{"tokens": N}`). openai `CountTokens` implemented over the former stub (§1.4). |
| **PAR-BF-OAI-201** | SSE headers set **after** stream setup so provider errors return JSON (`bifrost-openai.md:89`) | MISSING — "g0router sets SSE headers before calling provider … so provider errors return `text/event-stream` with JSON body mismatch". Confirmed real bug: `chat.go:417-419` sets `text/event-stream` THEN `chat.go:436-439` calls `writeError` (JSON) on stream-open failure; identical in `responses.go:128-136`, `completions.go` (bf-openai-1 stream path), `audio.go`/`images.go` (bf-openai-2 stream paths). | HAVE — the SSE content-type/headers are set ONLY AFTER the provider stream channel opens successfully (`perr == nil`). On a stream-open `*ProviderError`, `writeError` runs with the DEFAULT (JSON) content-type + the real status code — no `text/event-stream` mismatch. Applied to every `/v1/*` streaming handler bf-openai-1..4 own (§1.5). |

### Rows this plan FLIPS (already-satisfied — assert + cite, NO new feature code; a regression test is added/confirmed)

| Row | Why it is already HAVE in the LIVE tree (evidence) |
|---|---|
| **PAR-BF-OAI-003** | `POST /v1/responses` registered: `routes_openai.go:127` (`r.POST("/v1/responses", responses.Handle)`); handler `internal/api/responses.go:56`. Matrix:14 says MISSING — STALE. **FLIP to HAVE.** |
| **PAR-BF-OAI-208** | Responses streaming supported: `responses.go:132-147` opens `ChatCompletionStream` and pipes through `ProcessTranslateStream(…, FormatOpenAIResponses, …)`; `responses_test.go:183` (`TestResponsesEndpointStreamsEvents`) is green. Matrix:96 says MISSING/stubbed — STALE (the `Responses` provider stub is NOT on the live responses path; the handler uses translation, not `provider.Responses`). **FLIP to HAVE.** |
| **PAR-BF-OAI-203** (responses half) | `event: <type>` SSE typing for the Responses stream: `FormatSSE(FormatOpenAIResponses, …)` emits `event: %s\ndata: %s\n\n` (`internal/translation/sse.go:144-153`); `responses_test.go:203,206,209` assert `event: response.created`/`response.output_text.delta`/`response.completed`. **FLIP to HAVE for the responses path** (image-gen half → VARIANT §8). |
| **PAR-BF-OAI-204** (positive half) | `[DONE]` terminator on the Responses stream: `ProcessTranslateStream` writes `data: [DONE]\n\n` (`internal/translation/stream.go:136`); `responses_test.go:214` asserts it. **FLIP the positive half to HAVE** (the skip-toggle negative half → VARIANT §8). |

### Rows this plan records as VARIANT (no code, or one tiny optional additive — see §1.6)

| Row | Disposition |
|---|---|
| **PAR-BF-OAI-301** | `BifrostError`/`is_bifrost_error`/`event_id` envelope. g0router's flat OpenAI `{"error":{message,type,code[,param]}}` + the admin `{data,error}` envelope is CANONICAL (AGENTS.md). **VARIANT-by-design** — STAYS MISSING with a variant note (do NOT restructure). |
| **PAR-BF-OAI-302** | `ErrorField` with `Param`/`EventID`. **`APIError.Param` ALREADY EXISTS** (`internal/schemas/errors.go:7` — matrix "lacks Param" is STALE). `EventID` is Bifrost-specific. **VARIANT**; the only optional code is surfacing the existing `Param` through `writeError` (§1.6 DECISION). `event_id` = variant-by-design. |
| **PAR-BF-OAI-303** | Status-code fallback + `IsBifrostError` discriminator. g0router sets explicit status in `writeError` (`errors.go:35`) and has no `IsBifrostError` flag by design. **VARIANT/PARTIAL-by-design** — no change. |
| **PAR-BF-OAI-304** (open-error half) | SSE error frames: the SSE-OPEN error JSON-content-type fix IS built (= 201, §1.5). The mid-stream `event: error` *re-frame* is VARIANT (§8 ESC-SSE-MIDSTREAM-FRAME). |

### Rows this plan ESCALATES (STAY MISSING — honest, not built)

| Row | Why it can NOT be built additively here |
|---|---|
| **PAR-BF-OAI-005** | `POST /v1/responses/compact` (compaction). **NO `Compaction` method on `Provider`** (`internal/schemas/provider.go:69-108` — grep-confirmed absent; the MAP's "interface methods exist" is FALSE for Compaction — it lists CountTokens, which DOES exist, but never Compaction). **NO `Compaction`/`Compact*` schema** (`grep -rI 'Compaction\|CompactRequest\|CompactResponse' internal/schemas/` → no match). Adding a `Compaction` interface method is a NON-ADDITIVE change touching all 43 providers' `Provider` implementations (violates the no-leftovers rule §3 and the "no dead interface methods" guard BIFROST-MAP:278). Its *behavior* (server-side conversation/transcript compaction for the Responses API) is part of the responses subsystem whose full fidelity is the responses-rewrite ESC (BIFROST-MAP §5(c):118, §1 ESC-REF-ABSENT — the compaction wire shape is unverifiable without the ref). **ESCALATED (§8 ESC-COMPACTION); STAYS MISSING.** Do NOT add a `Compaction` method/schema/route/stub. |
| **PAR-BF-OAI-202** | SSE reader bypasses fasthttp internal pipe (`lib.NewSSEStreamReader`). Pure performance/transport-abstraction; g0router writes directly to `ctx` by design. Not a correctness gap. **ESCALATED (§8 ESC-SSE-PIPE); STAYS MISSING.** |
| **PAR-BF-OAI-203** (image-gen half) | `event: <type>` for image-gen streams. Image streams (bf-openai-2, SHIPPED) use `writeSSEStream`→`ProcessPassthroughStream` (plain `data:`), not the responses translator. Re-framing them is a passthrough-processor change in bf-openai-2's domain. **VARIANT/ESC (§8 ESC-IMG-EVENT-TYPE).** |
| **PAR-BF-OAI-204** (skip-toggle half) | Skip `[DONE]` when `includeEventType`/`skipDoneMarker`. g0router has no such toggle and always terminates with `[DONE]` by design. **VARIANT-by-design (§8 ESC-DONE-SKIP).** |
| **PAR-BF-OAI-205** | Raw upstream-bytes passthrough. Already MAP-ESC (:235); cross-cutting transport change. **STAYS MISSING.** |

Matrix flips at closeout (§4 T-close), in `.planning/parity/matrix/bifrost-openai.md`:
- PAR-BF-OAI-004 → HAVE (cite the input_tokens route + responses-translation reuse + openai CountTokens impl + tests).
- PAR-BF-OAI-201 → HAVE (cite the headers-after-open fix across the streaming handlers + the stream-open-error-returns-JSON test).
- PAR-BF-OAI-003 → HAVE (cite `routes_openai.go:127` + `responses.go:56` + `responses_test.go:146`).
- PAR-BF-OAI-208 → HAVE (cite the streaming path + `responses_test.go:183`).
- PAR-BF-OAI-203 → HAVE (responses path: `sse.go:144-153` + `responses_test.go:203-209`); add a one-line matrix note that the image-gen-stream half is VARIANT (§8).
- PAR-BF-OAI-204 → HAVE (positive `[DONE]`: `stream.go:136` + `responses_test.go:214`); note the skip-toggle is VARIANT-by-design.
- PAR-BF-OAI-301/302/303 → STAY MISSING + VARIANT note (envelope is canonical; `APIError.Param` already exists; `event_id`/`is_bifrost_error` are a different contract). If the optional `param`-surface ships (§1.6), add the cite to 302's note (still VARIANT — the envelope is unchanged in shape).
- PAR-BF-OAI-304 → the SSE-open-error half folds into 201 HAVE; the mid-stream re-frame STAYS MISSING + VARIANT note.
- **PAR-BF-OAI-005 → STAYS MISSING + ESCALATED** (no interface method/schema; responses-rewrite-dependent; §8).
- **PAR-BF-OAI-202, 205 → STAY MISSING** (perf / raw-passthrough; §8 / MAP-ESC).

### 1.1 The OpenAI-shape vs admin-envelope decision (BINDING — inherited from bf-openai-1/2/3 §1.1)

**`/v1/*` routes return OpenAI shapes, NOT the `{data,error}` admin envelope.**
The bf-openai-4 input_tokens handler returns the bare `*TokenCountResponse`
(`{"tokens": N}` — `internal/schemas/provider.go:64-66`) via `jsonMarshal` → 200
`application/json`, mirroring the SHIPPED non-stream JSON-out path
(`internal/api/audio.go:212-222`, `completions.go`). All errors call
`writeError(ctx, status, errType, message, code)` (`internal/api/errors.go:18`) —
the OpenAI `{"error":{…}}` shape — NOT the admin envelope. The api package does
not import `internal/admin`. The `{data,error}` admin envelope is **FORBIDDEN** on
these routes (§6).

### 1.2 input_tokens is NON-streaming (BINDING)

`/v1/responses/input_tokens` returns a single `{"tokens": N}` object — it does NOT
stream (`CountTokens` returns `(*TokenCountResponse, *ProviderError)` —
`provider.go:107` — a single value, no channel). There is NO `text/event-stream`
path in the input_tokens handler. (This is the ONE responses-family route in this
plan that is non-streaming; `/v1/responses` itself stays streaming-only and is
UNTOUCHED — bf-openai-4 does not edit `responses.go`'s Handle body except for the
shared SSE-ordering fix §1.5.)

### 1.3 Responses→Chat translation reuse for input_tokens (BINDING — Option A)

`CountTokens` takes a `*ChatRequest` (`provider.go:107`), but the
`/v1/responses/input_tokens` request body is a **Responses-API shape** (model +
input + instructions + tools …). The handler MUST translate the responses body
to a `*ChatRequest` before calling `CountTokens`, REUSING the exact path the
SHIPPED `ResponsesHandler.Handle` already uses:

```
raw → json.Unmarshal → body map
model, _ := body["model"].(string)
translated, err := h.registry.TranslateRequest(
    translation.FormatOpenAIResponses, translation.FormatOpenAI, model, body,
    /*stream=*/false, nil)            // NOTE stream=false (count is non-streaming)
b, _ := json.Marshal(translated)
var req schemas.ChatRequest; json.Unmarshal(b, &req)
translation.PreprocessChatRequest(&req)
```

This is verbatim the SHIPPED `responses.go:57-90` block (with `stream=false`),
proving the translation path exists and is the in-scope, ESC-REF-ABSENT-safe way
to reach `CountTokens` from a responses body. The handler then resolves the
provider/key (`h.router.ResolveForModel(&req)`, `responses.go:92`), applies the
SHIPPED `x-g0-vk` gate (§1.7), and dispatches `provider.CountTokens`.

> Soundness: this REUSES `translation.Registry`/`PreprocessChatRequest`/
> `ResolveForModel` — all SHIPPED. It introduces NO new translation logic and
> NO new schema. The count is computed by the upstream openai `CountTokens` impl
> (§1.4), not by g0router estimating locally.

### 1.4 openai `CountTokens` implementation (BINDING — Option A, mirrors bf-openai-1 TextCompletion-over-stub)

The openai `CountTokens` stub (`internal/providers/openai/stubs.go:17-19`) is
replaced with a real upstream proxy, mirroring the SHIPPED `Embedding`/
`TextCompletion`/`Speech` transport (`internal/providers/openai/embedding.go`,
`completions.go`, `audio.go`):

- `p.client.AcquireRequest/Response`; `req.SetRequestURI(p.baseURL +
  "/v1/responses/input_tokens")`; `req.Header.SetMethod("POST")`;
  `utils.SetAuthHeader(req, key.Value)`; `utils.SetJSONBody(req, …)` with the
  count request body (see the body note below); `p.client.Do`; status-check →
  `p.errorConverter.Convert(...)` on non-2xx; `utils.ReadJSONBody(resp, &result)`
  into a struct that decodes the upstream `{"input_tokens": N}` (or `{"tokens":
  N}`) into `schemas.TokenCountResponse{Tokens: N}`. Correct `RequestType` =
  `"count_tokens"` (a string literal — there is NO `RequestTypeCountTokens`
  constant in `internal/schemas/catalog.go:8-13`; do NOT add one unless an
  existing call site requires it, per no-leftovers §3).
- **Upstream body note (ESC-REF-ABSENT-bounded):** the openai upstream
  `/v1/responses/input_tokens` accepts a Responses-shaped body. Because
  bf-openai-4 receives a *translated `ChatRequest`* (§1.3) but the upstream count
  endpoint is Responses-shaped, the executor MUST resolve, at P2 with evidence,
  ONE of: (a) the upstream `CountTokens` accepts the chat-shaped body directly
  (openai's tokenizer is model-keyed, body-shape-tolerant), OR (b) the openai
  provider sends the ORIGINAL responses body through (in which case `CountTokens`
  needs the raw body, not the translated `ChatRequest` — a signature mismatch
  that would require coordinating an interface change with bf-core-1, OUT of this
  plan's additive scope). **If neither (a) nor a hermetic equivalent can be made
  to pass, STOP and ESCALATE (§8 ESC-COUNT-BODY) — fall to Option B (handler
  surfaces the 501 cleanly) for CountTokens; do NOT fabricate a green.** The
  hermetic test asserts the handler→provider→upstream round-trip with a fake
  upstream returning `{"input_tokens": 42}` and the handler emitting
  `{"tokens": 42}`.

> Other 42 providers' `CountTokens` stay 501 (UNTOUCHED) — bf-openai-4 implements
> ONLY the **openai** provider's `CountTokens`, exactly as bf-openai-1/2/3
> implemented only the openai provider's methods.

### 1.5 SSE-setup-ordering fix (BINDING — PAR-BF-OAI-201, the real bug)

**Current (buggy) shape** — every streaming `/v1/*` handler sets the SSE
content-type/headers BEFORE confirming the provider stream opened:

```go
// chat.go:417-440 (and structurally identical in responses.go:128-136,
// completions.go stream path, audio.go/images.go stream paths)
ctx.SetContentTypeBytes([]byte("text/event-stream"))   // ← set first
ctx.Response.Header.Set("Cache-Control", "no-cache")
ctx.Response.Header.Set("Connection", "keep-alive")
ch, perr := provider.ChatCompletionStream(...)
if perr != nil {
    writeError(ctx, fasthttp.StatusBadGateway, perr.Type, perr.Message, perr.Code) // ← JSON under text/event-stream
    return
}
```

**Fixed shape** — open the stream FIRST; set SSE headers ONLY on success:

```go
ch, perr := provider.ChatCompletionStream(...)
// (refresh-retry loop stays where it is, before the header set)
if perr != nil {
    // content-type is still the default; writeError emits application/json + real status
    g.recordError(endpoint, ...)            // preserve the existing recordError call
    status := perr.StatusCode; if status == 0 { status = fasthttp.StatusBadGateway }
    writeError(ctx, status, perr.Type, perr.Message, perr.Code)
    return
}
ctx.SetContentTypeBytes([]byte("text/event-stream"))   // ← set only after open succeeds
ctx.Response.Header.Set("Cache-Control", "no-cache")
ctx.Response.Header.Set("Connection", "keep-alive")
// then writeSSEStream… / ProcessTranslateStream…
```

**Binding consequences:**
- Apply the SAME reordering to EVERY `/v1/*` streaming handler that bf-openai-1..4
  own: `chat.go` (stream branch :416-449), `responses.go` (:128-147),
  `completions.go` (stream path), `audio.go` (Speech/Transcription stream paths),
  `images.go` (Generations stream path). The executor MUST re-grep at P2 for the
  `SetContentTypeBytes([]byte("text/event-stream"))`-then-`ChatCompletionStream`/
  `*Stream`-then-`writeError` shape across `internal/api/*.go` and fix each
  occurrence. (The `messages.go` Claude path: fix it too IF it has the same
  shape; verify at P2.)
- The status code on a stream-open error must be the provider's real status
  (`perr.StatusCode`, falling back to `StatusBadGateway` when 0) — mirror the
  non-stream branch (`chat.go:461-465`), NOT a hardcoded 502 — so the open-error
  path matches the non-stream error path exactly.
- This fix is BEHAVIOR-PRESERVING for the success path (headers still get set,
  just one line later) and CORRECTNESS-FIXING for the error path. The existing
  passing stream tests (`responses_test.go:183`, the bf-openai-1/2 stream tests)
  MUST stay green; a NEW test asserts: stream-open `*ProviderError` →
  `Content-Type: application/json` + `{"error":{…}}` body + the real status code
  (NOT `text/event-stream`).
- **bf-openai-2/3 SHIPPED handlers (`audio.go`, `images.go`)** are co-owned only
  for THIS ordering fix. The edit is the minimal reorder above — do NOT refactor
  their bodies otherwise. (This is the one place bf-openai-4 touches a prior
  plan's file; it is in-scope because 201 is a cross-handler correctness row and
  the OpenAI chain is serial — bf-openai-4 is the sole live holder.)

### 1.6 The `param`/`event_id` error-field DECISION (RESOLVED — variant-record, with one optional tiny additive)

**DECISION (binding): g0router's `{data,error}` / flat OpenAI `{"error":{…}}`
envelope is CANONICAL (AGENTS.md: "All API responses use snake_case JSON with a
`{data, error}` envelope"). Do NOT restructure it. Do NOT add `event_id` /
`is_bifrost_error`.** PAR-BF-OAI-301/302/303 are recorded as **VARIANT-by-design**
(the g0router envelope already conveys the error; `event_id`/`is_bifrost_error`
are Bifrost-specific contract surface, not a g0router gap).

**Sub-decision on `param` (the ONE tiny optional additive):**
- `APIError.Param *string` ALREADY EXISTS (`internal/schemas/errors.go:7`); so does
  `ProviderError.Param` (`errors.go:29`). The matrix:108 cite "lacks Param" is
  STALE — verify and correct it at closeout.
- The gap is purely that `writeError` (`internal/api/errors.go:18-38`) does not
  EMIT `param` even when present. `writeProviderError` (`audio.go:262`) forwards
  `perr.Code` but the executor must check whether it forwards `perr.Param`.
- **OPTIONAL, tiny, non-breaking additive (RECOMMENDED if it stays trivial):**
  add a `param *string` parameter to `writeError` OR a sibling that, when
  non-nil, sets `resp["error"]["param"]`, and have `writeProviderError` pass
  `perr.Param` through. This is genuinely useful (it surfaces the upstream
  `param` openai already returns on validation errors — PAR-BF-OAI-305 already
  preserves `type`/`code`/`message`; `param` completes the OpenAI error object)
  and is additive (a new optional field on an EXISTING struct shape, `omitempty`,
  no envelope restructuring).
- **GUARDRAIL:** if surfacing `param` requires changing `writeError`'s signature
  at MANY call sites (it is called ~30× across `internal/api/*.go`), prefer a
  NEW helper `writeErrorWithParam(ctx, status, errType, message, code, param)`
  that `writeError` delegates to (keeping the existing signature stable for all
  current callers), and have `writeProviderError` use the new helper. If even
  that is not clean/tiny, RECORD `param`-surfacing as VARIANT (the field exists
  on the struct; surfacing it is deferred) and ship NOTHING — do NOT force it.
- **`event_id` / `is_bifrost_error`:** NOT added under any circumstance
  (variant-by-design — they are a different envelope contract; adding them would
  diverge g0router's canonical shape).

This resolves the §224 DECISION-NEEDED explicitly: **variant-record the envelope;
optionally surface the already-existing `param`; never restructure; never add
`event_id`.**

### 1.7 Go contract (mirrors bf-openai-1/2/3)

**Schemas (REUSE — no new schema, no schema-file edit):**
`internal/schemas/provider.go:64-66` provides `TokenCountResponse{Tokens int}`
(the input_tokens success shape). `internal/schemas/responses.go` provides
`ResponsesRequest`. `internal/schemas/errors.go:4-9` provides `APIError` (with the
existing `Param`). **No `internal/schemas/*` edit** (the `param`-surface §1.6 edits
the `internal/api` *writer*, not the struct). If a genuinely-absent field forces an
additive change, STOP + coordinate the provider.go/errors.go micro-serial with
bf-core-1 (§0.1) — default is no schema edit.

**Provider transport (Option A — NEW file `internal/providers/openai/counttokens.go`):**
the `CountTokens` method per §1.4, mirroring `embedding.go` transport; correct
`RequestType` (`"count_tokens"`); no `init()`; errors-as-values; no panics. DELETE
the `CountTokens` stub from `stubs.go` (it MOVES here, implemented).

**Handler (NEW file `internal/api/input_tokens.go`):** a `InputTokensHandler`
(or extend the existing `ResponsesHandler` with a `CountTokens(ctx)` method —
DECISION: a NEW dedicated handler file mirrors the bf-openai-1/2/3 one-file-per-
surface convention and avoids re-touching `responses.go`'s Handle; the route reads
`r.POST("/v1/responses/input_tokens", inputTokens.Handle)`). Same struct fields as
`ResponsesHandler` (`router modelResolver`, `registry *translation.Registry`,
`usageRecorder`, `pendingTracker`, `detailCapture`, `vkGate`, `pinnedResolver`),
same additive setters, same `recordGlue()`, same `x-g0-vk` gate placement (after
Resolve, before dispatch — REUSE the SHIPPED gate block `responses.go:98-118`),
same `gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d",
ctx.ID())}`.

| Aspect | Contract |
|---|---|
| Resolver seam | REUSE `modelResolver` (`internal/api/chat.go:19`); `inference.Router.ResolveForModel` satisfies it. Do NOT widen. |
| Body parse | Responses-shape JSON → translate to `*ChatRequest` (§1.3, `stream=false`); invalid JSON / translate error → `writeError(400, "invalid_request_error", …, nil)`. |
| Resolve | `h.router.ResolveForModel(&req)`; error → `writeError(400, "invalid_request_error", err.Error(), nil)` (mirror `responses.go:92-96`). |
| VK gate | `x-g0-vk` gate + pinned-key override, IDENTICAL to `responses.go:98-118`. Apply BEFORE dispatch. Record endpoint `/v1/responses/input_tokens`. |
| Usage glue | additive setters + `recordGlue()` so `routes_openai.go` wires usage symmetrically (`responses.go:36-52`). Non-stream record; 0/0 tokens (count is not a billed inference; mirror audio's 0/0 — `audio.go:118`). |
| Dispatch | `provider.CountTokens(gatewayCtx, key, &req)`; on `*ProviderError` → `writeProviderError(ctx, perr)` (or `writeError` with `perr.StatusCode`); on success `jsonMarshal(resp)` → 200 `application/json` (bare `*TokenCountResponse`); marshal failure → plain-text 500 (`responses.go:77-81`). |
| Streaming | NONE (§1.2). |

**Construction:** `NewInputTokensHandler(router *inference.Router)
*InputTokensHandler` (mirror `NewResponsesHandler`, `responses.go:28-33`).
Constructed INSIDE `RegisterOpenAIRoutes` like the SHIPPED handlers
(`routes_openai.go:39-45`). NO `New(...)`/`RegisterOpenAIRoutes(...)` signature
change beyond the additive symmetry already present (decision 9).

### 1.8 routes_openai.go registration (serial-slot additive, §3)

Construct + wire the new handler alongside the existing ones, and append the route
line, grouped with the other `/v1/*` routes (after the bf-openai-3 batches lines).
`/v1/responses/input_tokens` is a STATIC path → no `{param}` precedence concern
with `/v1/responses` (static `/v1/responses/input_tokens` vs static
`/v1/responses` — distinct exact paths; verify fasthttp routes both at P2).

```go
// (handler-construction block, after `batches := api.NewBatchesHandler(router_)` :45)
inputTokens := api.NewInputTokensHandler(router_)
// usage glue — extend the existing if-blocks (mirror :46-75)
if recorder != nil { inputTokens.SetUsageRecorder(recorder) }
if tracker  != nil { inputTokens.SetPendingTracker(tracker) }
if detail   != nil { inputTokens.SetDetailCapture(detail)   }
if st != nil {
    inputTokens.SetVKGate(vkGate)             // reuse vkGate built at :97
    inputTokens.SetVKPinnedResolver(selector) // reuse selector built at :109
}

// (route block, after the bf-openai-3 batches route lines)
r.POST("/v1/responses/input_tokens", inputTokens.Handle)
```

The SSE-ordering fix (§1.5) edits `internal/api/*.go` handler bodies, NOT
`routes_openai.go` — the only `routes_openai.go` change is the additive
construction + the one new route line. REUSE
`vkGate`/`selector`/`recorder`/`tracker`/`detail`; do NOT rebuild. **Verify the
`r.POST` helper + static-path coexistence at P2.**

### NOT in scope (explicit — FORBIDDEN)

- **PAR-BF-OAI-005 (`/v1/responses/compact`)** — no interface method, no schema;
  responses-rewrite-dependent. ESCALATED (§8 ESC-COMPACTION), STAYS MISSING. Do
  NOT add a `Compaction` method/schema/route/stub. Do NOT add `Compaction` to
  `internal/schemas/provider.go` (no-leftovers §3; that is bf-core-1's reconcile
  surface ONLY if a route is funded — and none is).
- **Any responses-rewrite / normalization layer** (BIFROST-MAP §5(c), §1
  ESC-REF-ABSENT, ESC rows 101-118) — REJECTED. Do NOT touch
  `internal/translation/*` except to READ it; the SSE typing/[DONE] already work
  (§0). Do NOT add `event:` typing to image streams (203 image half — VARIANT),
  do NOT add a `[DONE]`-skip toggle (204 — VARIANT), do NOT add mid-stream
  `event: error` re-framing (304 — VARIANT), do NOT add the fasthttp pipe-bypass
  reader (202 — ESC), do NOT add raw-bytes passthrough (205 — ESC).
- **Restructuring the `{data,error}` / flat OpenAI error envelope** (§1.6) —
  FORBIDDEN. No `event_id`, no `is_bifrost_error`, no envelope shape change. The
  ONLY permitted error-writer change is the optional, tiny, additive `param`
  surface of the ALREADY-EXISTING `APIError.Param` (§1.6) — and only if it stays
  trivial; else variant-record and ship nothing.
- **`internal/schemas/{provider,errors,responses}.go` edits** — none expected
  (§1.7); if forced, coordinate the bf-core-1 micro-serial (§0.1).
- **The other 42 providers' `CountTokens`** — UNTOUCHED (they keep their 501).
  bf-openai-4 implements ONLY the **openai** provider's `CountTokens`.
- **The remaining openai stubs** (`Responses`, `ResponsesStream` in
  `internal/providers/openai/stubs.go:9-15`) — UNTOUCHED (the live `/v1/responses`
  path uses translation, not these stubs; they stay 501 as dead-but-required
  interface satisfiers). Do NOT implement them (that is the responses-rewrite ESC).
- **`responses.go`'s Handle body** — touched ONLY for the shared SSE-ordering fix
  (§1.5); no other change.
- **Other bf-openai plans' surfaces** beyond the SSE-ordering co-edit (§1.5) — no
  completions/audio/images/files/batches feature changes; only the minimal
  stream-header reorder in their stream paths.
- **All UI / e2e / mocks** — API route, no UI contract. No `ui/**`, no playwright,
  no mock/seed.
- **No `New(...)`/`RegisterOpenAIRoutes(...)` signature change**, no `init()`, no
  global state, errors-as-values (`fmt.Errorf("ctx: %w")`), no panics.
- **No store / migrate touch** (this route touches no store).

---

## 2. Precondition checks

Run all before any edit; abort and report to the orchestrator on any failure.

```bash
# P0 — clean tree; record <base>
git status --porcelain     # must be empty (explicit `git add <file>`, never -A)
git rev-parse HEAD         # record as <base> for §5

# P1 — the input_tokens gap is REAL; the compaction premise is FALSE; the flips are real
grep -rn '/v1/responses/input_tokens' internal/ | grep -v _test.go    # expect NOTHING (no route/handler)
test ! -e internal/api/input_tokens.go            && echo "input_tokens handler gap OK"
test ! -e internal/providers/openai/counttokens.go && echo "provider counttokens impl gap OK"
grep -n 'notImplemented("count_tokens")' internal/providers/openai/stubs.go   # :18 — the stub to replace
# Compaction has NO buildable surface (escalation evidence):
! grep -rIn 'Compaction\|CompactRequest\|CompactResponse' internal/schemas/ internal/providers/openai/ && echo "005 has no method/schema — escalate OK"
! grep -n 'Compaction' internal/schemas/provider.go && echo "no Compaction interface method — escalate OK"
# The FLIP rows ARE already satisfied:
grep -n 'r.POST("/v1/responses"' internal/server/routes_openai.go             # :127 (003 HAVE)
grep -n 'event: %s' internal/translation/sse.go                               # :149,151 (203 responses HAVE)
grep -n 'data: \[DONE\]' internal/translation/stream.go                        # :136 (204 positive HAVE)
grep -n 'event: response.created\|event: response.completed\|data: \[DONE\]' internal/api/responses_test.go  # :203,209,214 (regression already green)

# P2 — reused surfaces present (the de-risk)
grep -n 'type TokenCountResponse\|CountTokens' internal/schemas/provider.go   # :64-66, :107
grep -n 'type APIError\|Param ' internal/schemas/errors.go                     # :4-9, :7 (Param ALREADY EXISTS — 302 cite stale)
grep -n 'func (h \*ResponsesHandler) Handle\|TranslateRequest\|PreprocessChatRequest\|ResolveForModel' internal/api/responses.go internal/inference/router.go
grep -n 'func writeError\|func writeProviderError\|func jsonMarshal\|func requestHeadersFromCtx' internal/api/*.go
grep -n 'func NewResponsesHandler\|type modelResolver' internal/api/*.go
grep -n 'func SetAuthHeader\|func SetJSONBody\|func ReadJSONBody' internal/providers/utils/*.go
grep -n 'func (p \*Provider) Embedding\b' internal/providers/openai/embedding.go   # transport template
grep -n 'p.baseURL = srv.URL' internal/providers/openai/*_test.go             # hermetic pattern

# P3 — the SSE-ordering bug shape across the streaming handlers (201 target sites)
grep -rn 'SetContentTypeBytes(\[\]byte("text/event-stream"))' internal/api/*.go   # every site that sets SSE headers
# For each, confirm it precedes the provider *Stream call + a writeError on perr (the bug):
grep -n 'ChatCompletionStream\|SpeechStream\|TranscriptionStream\|ImageGenerationStream\|TextCompletionStream' internal/api/*.go

# P4 — the param-surface call-site count (§1.6 guardrail)
grep -rcn 'writeError(' internal/api/*.go    # if many, prefer writeErrorWithParam sibling
grep -n 'func writeProviderError' internal/api/audio.go ; sed -n '262,270p' internal/api/audio.go 2>/dev/null || grep -A8 'func writeProviderError' internal/api/audio.go  # does it forward perr.Param?

# P5 — routes_openai.go SERIAL SLOT is FREE (bf-openai-3 released it on close)
git log --oneline -8 -- internal/server/routes_openai.go
# Orchestrator MUST confirm no concurrent bf-openai-* plan holds an unmerged
# routes_openai.go edit. bf-openai-4 is LAST in the chain (after SHIPPED
# bf-openai-1/2/3). TAKES the slot; on close the OpenAI chain is COMPLETE.
# Confirm NO bf-core-1 in-flight edit to schemas/provider.go|errors.go (§0.1) —
# bf-openai-4 touches NEITHER, but verify before T-impl.

# P6 — green at base (HERMETIC; no network)
go test ./... && go vet ./... && go build ./...     # exit 0 (untouched-green baseline)
```

---

## 3. Exclusive file ownership

After bf-openai-4 merges, CREATE files are owned by bf-openai-4; later plans
consume, never edit (decision 7).

**CREATE — provider transport (NEW, Option A):**

| File | Contract |
|---|---|
| `internal/providers/openai/counttokens.go` | `CountTokens` (Option A, §1.4) — moved from stubs.go, now implemented; POST upstream `/v1/responses/input_tokens`; JSON body; decode `{input_tokens|tokens: N}` → `*TokenCountResponse`. No `init()`; errors-as-values; `RequestType "count_tokens"`. |
| `internal/providers/openai/counttokens_test.go` | RED first. Hermetic `httptest.NewServer` + `p.baseURL = srv.URL` (mirror the SHIPPED `embedding_test.go`/`audio_test.go` pattern): `CountTokens` success (fake upstream returns `{"input_tokens":42}`) → `*TokenCountResponse{Tokens:42}`; upstream-non-200 → `*ProviderError` carrying the status. NO real network. |

**CREATE — api transport (NEW):**

| File | Contract |
|---|---|
| `internal/api/input_tokens.go` | `InputTokensHandler` + `NewInputTokensHandler` + additive setters (VK/usage) + `recordGlue` + `Handle(ctx)` (responses-body → translate → `*ChatRequest` → `CountTokens`, §1.3/§1.7). OpenAI shapes only (§1.1); bare `*TokenCountResponse` success; `writeError`/`writeProviderError` for errors. REUSE the SHIPPED `translation.Registry`/`PreprocessChatRequest`/`ResolveForModel`/VK-gate/`jsonMarshal` (do NOT duplicate). |
| `internal/api/input_tokens_test.go` | RED first. Hermetic fake provider/resolver (mirror `responses_test.go`): valid responses body → bare `{"tokens":N}` JSON (assert NO `data`/`error` wrapper); invalid JSON → 400; translate error → 400; provider 501 → 501 passthrough; VK-denied → 429 + provider NOT called; VK-pinned override; marshal failure → plain 500; NON-streaming (assert content-type `application/json`, no `text/event-stream`). |

**MODIFY — SSE-setup-ordering fix (201, co-owned cross-handler — §1.5):**

| File | Change |
|---|---|
| `internal/api/chat.go` | Reorder the STREAM branch (:416-449): open `ChatCompletionStream` (keep the refresh-retry loop) → on `perr` `recordError` + `writeError(perr.StatusCode‖502, …)` with DEFAULT content-type → ONLY THEN set `text/event-stream`+headers → `writeSSEStreamWithSource`. NOTHING else changes. |
| `internal/api/responses.go` | Reorder (:128-147): open `ChatCompletionStream` → on `perr` `recordError`+`writeError` with default content-type → THEN set SSE headers → `ProcessTranslateStream`. Handle body otherwise UNCHANGED. |
| `internal/api/completions.go` | Reorder the stream path identically (verify the exact shape at P3). |
| `internal/api/audio.go` | Reorder the Speech + Transcription stream paths identically (bf-openai-2 SHIPPED; minimal reorder only — §1.5). |
| `internal/api/images.go` | Reorder the Generations stream path identically (bf-openai-2 SHIPPED; minimal reorder only). |
| `internal/api/messages.go` | IF it has the same SSE-headers-before-open shape (verify P3), reorder identically; else leave UNTOUCHED. |

**MODIFY — optional param-surface (302, §1.6 — ONLY if trivial; else SKIP):**

| File | Change |
|---|---|
| `internal/api/errors.go` | OPTIONAL: add `writeErrorWithParam(ctx, status, errType, message, code, param *string)` that sets `error.param` when non-nil; `writeError` delegates (signature stable for all current callers). If even this is not clean/tiny → SKIP (variant-record §1.6). |
| `internal/api/audio.go` (`writeProviderError`) | OPTIONAL: forward `perr.Param` via `writeErrorWithParam`. Only if the above ships. |

**EXTEND — provider stubs (REMOVE the now-implemented stub):**

| File | Change |
|---|---|
| `internal/providers/openai/stubs.go` | DELETE `CountTokens`(:17-19) ONLY (it moves to counttokens.go, implemented). The remaining stubs (`Responses` :9-11, `ResponsesStream` :13-15) + the `notImplemented` helper are PRESERVED verbatim. Re-grep live spans at P1. |
| `internal/providers/openai/openai_test.go` | IF `TestNotImplementedStubs` has a `count_tokens` sub-case, REMOVE exactly that sub-case (re-grep at P1/P4 — bf-openai-1/2/3 already removed their sub-cases). The `Responses`/`ResponsesStream` sub-cases are PRESERVED. This is the ONLY edit to a pre-existing openai-provider test (other than the stream-ordering test additions). |

**MODIFY — serial-slot route registration (additive only):**

| File | Change |
|---|---|
| `internal/server/routes_openai.go` | ADD the `inputTokens` handler construction + wiring (reuse `vkGate`/`selector`/`recorder`/`tracker`/`detail`) + the ONE route line `r.POST("/v1/responses/input_tokens", inputTokens.Handle)` (§1.8). NOTHING else changes. SERIAL SLOT — sole holder while live; FINAL holder of the OpenAI chain. |

**FORBIDDEN:** everything else. Explicitly: a `Compaction` method/schema/route/
stub (005 ESCALATED — §8); the openai `Responses`/`ResponsesStream` stubs (stay
501); the 42 other providers' `CountTokens`; any `internal/translation/*` edit
(SSE typing/[DONE] already work — §0); any `internal/store/*`/migrate; any
`internal/schemas/*` edit (REUSE; coordinate bf-core-1 micro-serial if forced —
§0.1); `event_id`/`is_bifrost_error`/envelope restructuring (§1.6); all
`internal/admin/*`, `internal/governance/*`, `internal/mcp/*`,
`internal/providers/catalog/*`; all `ui/**`; all video/containers/rerank/ocr/
async/WS. Touching any of these is an automatic REJECT.

---

## 4. TDD tasks

Cadence (strict, AGENTS.md "TDD always"): **no Go impl file may exist before its
`_test.go` is committed RED.** `go test ./... && go vet ./... && go build ./...`
green at EVERY commit (a RED commit may fail ONLY the new package's targeted run;
scaffold signatures so the package compiles and the assertion fails). Order:
flip-regression (cheap, no code) → provider impl → api handler → SSE-ordering fix →
optional param-surface → serial-slot route → closeout.

### T-flip — regression-lock the already-satisfied rows (003, 208, 203, 204)
NO production code. Confirm the SHIPPED `responses_test.go` already asserts:
route registration is exercised (`TestResponsesEndpointTranslatesRequest` :146),
streaming works (`TestResponsesEndpointStreamsEvents` :183), `event: response.*`
framing (:203,206,209), `data: [DONE]` (:214). Run
`go test ./internal/api/ -run 'Responses' -v` → GREEN. If any of the four
assertions is NOT already covered, ADD the missing assertion to a NEW test
function `TestResponsesStreamEventTypingAndDone` (regression lock) — RED only if
the behavior were absent (it is present, so the test passes immediately, locking
the flip). Commit:
`phase-1/bf-openai-4: regression-lock responses route+streaming+event-typing+[DONE] (003/208/203/204 flip)`.

### T-prov-counttokens — STEP(a) RED, STEP(b) impl
STEP(a): write `internal/providers/openai/counttokens_test.go` (hermetic httptest,
§3). Run `go test ./internal/providers/openai/ -run 'CountTokens'` → FAIL. Commit
RED: `phase-1/bf-openai-4: failing openai CountTokens test (TDD red)`.
STEP(b): create `internal/providers/openai/counttokens.go` (Option A, §1.4);
DELETE the `CountTokens` stub from `stubs.go`; REMOVE the `count_tokens` sub-case
from `openai_test.go` (if present). Gates green. Commit:
`phase-1/bf-openai-4: implement openai CountTokens (POST /v1/responses/input_tokens upstream)`.
*If the upstream body shape can't pass hermetically (§1.4), STOP + ESCALATE
(§8 ESC-COUNT-BODY); fall to Option B for CountTokens. Do NOT fabricate a green.*

### T-handler-input-tokens — STEP(a) RED, STEP(b) impl
STEP(a): write `internal/api/input_tokens_test.go` (fake provider/resolver +
responses-body request + VK cases, §3). Run `go test ./internal/api/ -run
'InputTokens'` → FAIL. Commit RED:
`phase-1/bf-openai-4: failing /v1/responses/input_tokens handler test (TDD red)`.
STEP(b): create `internal/api/input_tokens.go` (REUSE the SHIPPED
translation/resolve/VK-gate helpers). Gates green. Commit:
`phase-1/bf-openai-4: /v1/responses/input_tokens handler (responses→chat translate → CountTokens)`.

### T-sse-ordering — STEP(a) RED, STEP(b) fix (201)
STEP(a): write a NEW test (in `internal/api/chat_test.go` or a new
`internal/api/sse_ordering_test.go`) asserting: a stream-open `*ProviderError`
yields `Content-Type: application/json` + `{"error":{…}}` body + the real status
code (NOT `text/event-stream`), for chat (and at least one of responses/
completions). Run → FAIL (current code sets `text/event-stream` first). Commit
RED: `phase-1/bf-openai-4: failing SSE-open-error content-type test (201 TDD red)`.
STEP(b): apply the reorder (§1.5) to every streaming handler bf-openai-1..4 own
(chat/responses/completions/audio/images, + messages if same shape). Keep the
existing stream-success tests green. Gates green. Commit:
`phase-1/bf-openai-4: fix SSE header ordering — provider-open errors return JSON (201)`.

### T-param — OPTIONAL param-surface (302, §1.6) — ONLY if trivial
If the §1.6 guardrail passes (trivial sibling helper): STEP(a) write a test that
a `*ProviderError` carrying `Param` surfaces `error.param` in the JSON; RED;
STEP(b) add `writeErrorWithParam` + forward `perr.Param` in `writeProviderError`;
green. Commit:
`phase-1/bf-openai-4: surface existing APIError.Param in error envelope (302 variant-augment)`.
*If NOT trivial → SKIP this task; variant-record in closeout.*

### T-routes — serial-slot route registration
TAKE the serial slot (orchestrator confirms FREE at P5; confirm no bf-core-1
in-flight schema edit per §0.1). Add the construction + wiring + the ONE route
line to `routes_openai.go` (§1.8). Gates green. Commit (ONE commit touches the
serial file):
`phase-1/bf-openai-4: register /v1/responses/input_tokens route (serial slot — OpenAI chain final)`.

### T-close — full gates + closeout
```bash
go test ./... && go vet ./... && go build ./...
go test ./internal/api/... ./internal/providers/openai/... -run 'CountTokens|Compact|Responses|SSE|InputTokens' -v
go test ./internal/providers/openai/ -run 'NotImplemented' -v   # remaining stubs (Responses/ResponsesStream) still 501
```
Flip `.planning/parity/matrix/bifrost-openai.md`: 004 + 201 → HAVE (new code);
003, 208, 203(responses), 204(positive) → HAVE (flip; cite the SHIPPED evidence +
the regression lock); 301/302/303 → STAY MISSING + VARIANT note (envelope
canonical; `APIError.Param` already exists — correct the stale 302 cite; if the
optional param-surface shipped, cite it as variant-augment); 304 open-error half
folds into 201, mid-stream re-frame STAYS MISSING + VARIANT; **005 → STAYS MISSING
+ ESCALATED** (no method/schema; responses-rewrite-dependent); 202, 205 → STAY
MISSING. Update `docs/WORKFLOW.md` (P6 base observation; the 003/208/203/204 flip
discovery; the CountTokens Option-A + responses-translation-reuse decision; the
upstream-count-body resolution; the SSE-ordering fix scope; the param/event_id
variant-record decision; the Compaction escalation; the serial-slot take + the
OpenAI-chain-complete note). Append the Compaction + (if any) ESC-COUNT-BODY +
the image-event-type / done-skip / midstream-frame variants to
`.planning/parity/plans/open-questions.md`. **Append the FINAL bifrost-openai
HAVE/MISSING projection (§9) to WORKFLOW.** Final commit:
`phase-1/bf-openai-4: close — input_tokens HAVE; SSE-order fixed; 003/208/203/204 flipped; 005 escalated; matrix flip; OpenAI chain complete`.
**On the close commit, the routes_openai.go OpenAI serial chain is COMPLETE — no
successor.**

---

## 5. Binary acceptance criteria

All evaluated at the close commit; `<base>` = `git rev-parse HEAD` recorded at P0.

```bash
# A1 — targeted suites green (the brief's command)
go test ./internal/api/... ./internal/providers/openai/... -run 'CountTokens|Compact|Responses|SSE|InputTokens'   # exit 0

# A2 — full hermetic gates green (no network)
go test ./... && go vet ./... && go build ./...                  # exit 0

# A3 — the new route exists and is static
grep -n 'r.POST("/v1/responses/input_tokens"' internal/server/routes_openai.go   # exactly 1

# A4 — the CountTokens stub is GONE from stubs.go; impl exists; other stubs remain
! grep -n 'notImplemented("count_tokens")' internal/providers/openai/stubs.go    # gone
grep -n 'func (p \*Provider) CountTokens' internal/providers/openai/counttokens.go  # implemented here
grep -n 'notImplemented("responses")\|notImplemented("responses_stream")' internal/providers/openai/stubs.go  # PRESERVED

# A5 — Compaction was NOT added anywhere (escalation honored)
! grep -rIn 'Compaction\|/v1/responses/compact\|CompactRequest\|CompactResponse' internal/   # no match

# A6 — SSE-ordering fix: no streaming handler sets text/event-stream BEFORE the *Stream open error path
#   (manual proof: for each SetContentTypeBytes("text/event-stream") site, the preceding lines are the
#    provider *Stream open + perr check; the new test A7 is the executable proof)

# A7 — SSE-open-error returns JSON (the 201 regression test) is green
go test ./internal/api/ -run 'SSE|StreamOpenError' -v            # exit 0

# A8 — the flip rows' regression lock is green
go test ./internal/api/ -run 'Responses' -v                      # exit 0 (event typing + [DONE] + route + stream)

# A9 — envelope NOT restructured: no event_id / is_bifrost_error introduced
! grep -rn 'event_id\|is_bifrost_error\|IsBifrostError' internal/api/ internal/schemas/   # no match

# A10 — TDD order proof (RED before GREEN per surface)
git log --oneline <base>..HEAD -- internal/providers/openai/counttokens_test.go internal/providers/openai/counttokens.go
git log --oneline <base>..HEAD -- internal/api/input_tokens_test.go internal/api/input_tokens.go
#   each test file's first commit predates its impl file's first commit

# A11 — freeze proofs (no out-of-scope churn)
git diff --name-only <base>..HEAD     # ⊆ the §3 ownership set ONLY
git diff <base>..HEAD -- internal/translation/   # EMPTY (no translation edits)
git diff <base>..HEAD -- internal/schemas/       # EMPTY (no schema edits) — UNLESS bf-core-1-coordinated (§0.1)
git diff <base>..HEAD -- ui/                      # EMPTY
git diff <base>..HEAD -- internal/store/          # EMPTY

# A12 — commit prefix
git log --oneline <base>..HEAD | grep -vc '^[0-9a-f]\+ phase-1/bf-openai-4:'   # 0 (every commit prefixed)

# A13 — NO e2e
git diff --name-only <base>..HEAD | grep -E 'e2e|playwright|\.spec\.' && echo FAIL || echo "no e2e OK"
```

Stop conditions: any A-criterion red → STOP, report to orchestrator, do not
self-approve. Verification is a separate pass (code-reviewer/verifier), per the
authoring/review separation rule.

---

## 6. Guardrails (Must Have / Must NOT Have)

**Must Have:** strict TDD (RED before impl, hermetic, no network); `/v1/*` bare
OpenAI shapes (no admin envelope); additive only; no `init()`; errors-as-values;
no `New(...)`/`RegisterOpenAIRoutes(...)` signature change; reuse the SHIPPED
translation/resolve/VK-gate/error helpers; the SSE-ordering fix is behavior-
preserving on success + correctness-fixing on error; honest dispositions
(flip the satisfied rows, build the buildable, variant-record the envelope,
escalate Compaction).

**Must NOT Have:** a `Compaction` method/schema/route/stub; any
`internal/translation/*` or `internal/schemas/*` edit (unless bf-core-1-
coordinated §0.1); envelope restructuring / `event_id` / `is_bifrost_error`;
implementing the openai `Responses`/`ResponsesStream` stubs; touching the 42 other
providers; the fasthttp pipe-bypass reader (202) / raw passthrough (205) / image
`event:` typing (203 image half) / `[DONE]`-skip toggle (204 half) / mid-stream
`event: error` re-frame (304 half); any UI/e2e/mock; any store/migrate.

---

## 7. Files touched (summary)

CREATE: `internal/providers/openai/counttokens.go` (+ `_test.go`),
`internal/api/input_tokens.go` (+ `_test.go`),
`internal/api/sse_ordering_test.go` (or assertions in `chat_test.go`).
MODIFY: `internal/api/{chat,responses,completions,audio,images}.go` (SSE reorder,
§1.5; + `messages.go` if same shape), `internal/providers/openai/stubs.go` (delete
CountTokens stub), `internal/providers/openai/openai_test.go` (drop count_tokens
sub-case if present), `internal/server/routes_openai.go` (construction + 1 route).
OPTIONAL: `internal/api/errors.go` + `internal/api/audio.go` (param-surface §1.6).
DOCS: `.planning/parity/matrix/bifrost-openai.md`, `docs/WORKFLOW.md`,
`.planning/parity/plans/open-questions.md`.
NO edit: `internal/translation/*`, `internal/schemas/*` (default), `ui/**`,
`internal/store/*`.

---

## 8. Escalations + open questions (honest)

- **ESC-COMPACTION (PAR-BF-OAI-005)** — `/v1/responses/compact` has NO `Compaction`
  interface method (`provider.go:69-108` grep-confirmed absent), NO schema, NO
  stub. Adding the method is non-additive (43 providers) and violates the
  no-leftovers/no-dead-interface-method guard (§3, BIFROST-MAP:278); the behavior
  is responses-subsystem-coupled and its wire shape is unverifiable under
  ESC-REF-ABSENT. STAYS MISSING. If a future wave funds the responses-rewrite +
  restores the ref, Compaction rides that plan (bf-core-1 adds the interface
  method only when a route is funded). Open question recorded.
- **ESC-COUNT-BODY (conditional, PAR-BF-OAI-004)** — if the upstream openai
  `/v1/responses/input_tokens` rejects the translated chat-shaped body and
  requires the original responses body (§1.4), the additive `CountTokens(…,
  *ChatRequest)` signature is insufficient and an interface change would be needed
  (bf-core-1 micro-serial). In that case fall to Option B (handler surfaces the
  501 cleanly; route HAVE, impl escalated) and record honestly — never claim full
  parity on a 501. Resolve at P2 with evidence.
- **ESC-SSE-PIPE (PAR-BF-OAI-202)** — fasthttp internal-pipe-bypass SSE reader;
  pure perf/transport abstraction, divergent. STAYS MISSING.
- **ESC-IMG-EVENT-TYPE (PAR-BF-OAI-203 image half)** — `event: <type>` typing for
  image-gen streams (bf-openai-2 passthrough domain). The responses half is HAVE;
  image half is VARIANT (image streams use `ProcessPassthroughStream` plain
  `data:` by design). Defer.
- **ESC-DONE-SKIP (PAR-BF-OAI-204 negative half)** — skip-`[DONE]` toggle
  (`includeEventType`/`skipDoneMarker`); g0router always terminates with `[DONE]`
  by design. VARIANT-by-design.
- **ESC-SSE-MIDSTREAM-FRAME (PAR-BF-OAI-304 mid-stream half)** — re-framing a
  mid-stream error as `event: error`; the channel is already torn down
  (`stream.go:60-61`) and re-framing is a passthrough-processor change coupled to
  the responses-rewrite. The SSE-OPEN error half IS built (= 201). Defer.
- **ENVELOPE-VARIANT (PAR-BF-OAI-301/302/303)** — `BifrostError`/`is_bifrost_error`/
  `event_id` is a different contract; g0router's `{data,error}`/flat OpenAI error
  is canonical (AGENTS.md). Variant-by-design; the only optional augment is
  surfacing the ALREADY-EXISTING `APIError.Param` (§1.6). `event_id`/
  `is_bifrost_error` never added.

Append all of the above to `.planning/parity/plans/open-questions.md` at T-close.

---

## 9. FINAL bifrost-openai state projection (after bf-openai-4 closes)

The bifrost-openai matrix (89 rows incl. 1 EXTRA) after the full bf-openai-1..4
chain:

**HAVE (closed by the chain or pre-existing):**
- Pre-existing SAT: 001, 006, 018(models list), 206, 305 (+ 305 param-preserve).
- bf-openai-1: 002, 207 (completions +stream), 019 (models-get filter).
- bf-openai-2: 007, 008, 009, 010, 011, 209, 210, 211 (audio/images +streams).
- bf-openai-3: 020, 021, 022, 023, 025, 026, 027, 028, 029 (batches/files).
- bf-openai-4: **004** (input_tokens), **201** (SSE-open-error JSON), **003**,
  **208** (responses route + streaming — flip), **203** (responses event-typing —
  flip), **204** (positive `[DONE]` — flip).
- → roughly **35 HAVE** on the OpenAI surface.

**MISSING / VARIANT / ESC tail (documented, NOT built — the deliberate divergence):**
- VARIANT-by-design: 301, 302 (`param` exists; `event_id` not added), 303
  (no `IsBifrostError`), 203(image half), 204(skip-toggle), 304(mid-stream frame).
- ESC (this plan): 005 (Compaction — no method/schema, responses-rewrite-dep),
  202 (pipe bypass), 205 (raw passthrough).
- ESC (MAP, prior): 024 (batch results — reachable via 029), 101-118
  (normalization), 401-405/504-507 (capability matrices/WS), 012-017/030-043/
  501-503 (videos/containers/rerank/ocr/async/aliases), 044.

**HAVE ≈ 35 vs the documented ESC/VAR tail ≈ 54** — the tail is the intentional
single-binary-SQLite-gateway divergence (normalization layer, capability
matrices, WS/async/video/containers, in-memory raw passthrough, the Bifrost error
envelope) recorded honestly per ESC-REF-ABSENT, NOT silent gaps. The
**buildable-additive OpenAI surface is now COMPLETE**: every row with an existing
interface method + schema + stub (completions/audio/images/files/batches/
count-tokens) is HAVE; the responses route/streaming/event-typing/[DONE] are HAVE
(flipped); the SSE-open-error correctness bug is fixed; the only remaining
buildable row (Compaction) is honestly escalated because its interface
method/schema do not exist and its behavior depends on the escalated
responses-rewrite. The OpenAI serial chain (bf-openai-1→2→3→4) is closed.
```
