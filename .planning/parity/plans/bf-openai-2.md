# Micro-plan bf-openai-2 — Audio (`/v1/audio/{speech,transcriptions}` +stream) + Images (`/v1/images/{generations,edits,variations}` +stream) (Go)

```
program: bifrost-parity (bifrost phase — BUILDABLE-ADDITIVE only; the ~50%
  re-architecture is permanently deferred per BIFROST-MAP §1/§8 ESC set)
plan: bf-openai-2
status: READY (rev 1 — authored against the live tree @ <base>; BIFROST-MAP
  micro-plan index row ~line 297; bifrost-openai disposition rows 007/008/209/210
  + 009/010/011/211 = BIFROST-MAP:217-218; serial chain BIFROST-MAP:323-351;
  matrix rows .planning/parity/matrix/bifrost-openai.md:18-22,97-99,209-211)
runs: OpenAI-surface track. HOLDS the internal/server/routes_openai.go SERIAL
  SLOT while live (decision 3). Serial chain:
  bf-openai-1 (SHIPPED) → **bf-openai-2** → bf-openai-3 → bf-openai-4
  (each appends /v1/*). Disjoint from the governance / mcp / core tracks (run ∥).
branch: main (direct commits, per AGENTS.md "No PR workflow")
commit-prefix: phase-1/bf-openai-2:
ref-source: ESC-REF-ABSENT (BIFROST-MAP §47-68) — the frozen Bifrost ref
  (@ca21298) is NOT on this host. The matrix rows + g0router's own conventions
  are the ONLY ground truth. /v1/audio/{speech,transcriptions} and
  /v1/images/{generations,edits,variations} are documented, stable OpenAI public
  endpoints, so their wire shapes are g0router's own schemas
  (internal/schemas/audio.go, internal/schemas/images.go) + OpenAI's public spec
  — NOT a guessed Bifrost internal. No Bifrost handler internals are reconstructed.
base: <base> = git rev-parse HEAD recorded at P0. Substitute the actual SHA
  everywhere §5 says <base>.
go-serial-slot: this plan holds the SINGLE in-flight edit to
  internal/server/routes_openai.go while live (decision 3). bf-openai-1 RELEASED
  the slot on its close; bf-openai-2 TAKES it. Slot must be FREE at P5 before
  T-routes. RELEASE to bf-openai-3 on close.
new-route: API routes only. NO UI contract — /v1/* are OpenAI-compatible API
  routes (NOT the {data,error} admin envelope). No e2e, no UI, no mocks.
pattern: MIRRORS the SHIPPED bf-openai-1 (`.planning/parity/plans/bf-openai-1.md`):
  completions-handler shape + OpenAI-shape (not admin envelope) +
  provider-method-impl-over-stub (Option A). bf-openai-1 implemented
  TextCompletion/TextCompletionStream for the openai provider over the stubs;
  bf-openai-2 implements Speech/Transcription/ImageGeneration (+ Image edit/var,
  + streams) the same way.
```

---

## 1. Scope — PAR rows + the deliverables

### Rows this plan closes

| Row | Claim (matrix text) | Current state (evidence) | Target after bf-openai-2 |
|---|---|---|---|
| **PAR-BF-OAI-007** | `POST /v1/audio/speech` registered with fasthttp (`bifrost-openai.md:18`) | MISSING — "Schema exists (`internal/schemas/audio.go:4-10`) but no route; provider stubbed (`stubs.go:33-35`)". Confirmed: no `/v1/audio` route anywhere; `Speech` returns `notImplemented("speech")` (`internal/providers/openai/stubs.go:33-35`). | HAVE — route registered (`routes_openai.go`), handler dispatches to `provider.Speech`, returns RAW audio bytes with the upstream `Content-Type` (NOT a JSON envelope). |
| **PAR-BF-OAI-209** | Speech streaming supported (`bifrost-openai.md:97`) | MISSING — "Provider stubbed (`stubs.go:37-39`)". Confirmed: `SpeechStream` → `notImplemented("speech_stream")`. | HAVE — `stream:true` dispatches to `provider.SpeechStream`, SSE-framed via the shared `writeSSEStream`; `[DONE]` terminator. (See §1.6 — speech audio-stream caveat: default = pass upstream SSE/chunk bytes through; if the upstream returns raw audio chunks not SSE, the impl forwards raw frames — soundness analysis in §1.6.) |
| **PAR-BF-OAI-008** | `POST /v1/audio/transcriptions` registered with fasthttp (`bifrost-openai.md:19`) | MISSING — "Schema exists (`internal/schemas/audio.go:18-27`) but no route; provider stubbed (`stubs.go:41-43`)". Confirmed: `Transcription` → `notImplemented("transcription")`. | HAVE — route registered, handler parses **multipart/form-data** (file upload, §1.5), dispatches to `provider.Transcription`, returns the bare OpenAI `TranscriptionResponse` JSON. |
| **PAR-BF-OAI-210** | Transcription streaming supported (`bifrost-openai.md:98`) | MISSING — "Provider stubbed (`stubs.go:45-47`)". `TranscriptionStream` → `notImplemented("transcription_stream")`. | HAVE — `stream` form-field set → dispatches to `provider.TranscriptionStream`, SSE-framed; `[DONE]` terminator. |
| **PAR-BF-OAI-009** | `POST /v1/images/generations` registered with fasthttp (`bifrost-openai.md:20`) | MISSING — "Schema exists (`internal/schemas/images.go:4-13`) but no route; provider stubbed (`stubs.go:17-19`)". `ImageGeneration` → `notImplemented("image_generation")`. | HAVE — JSON route + handler → `provider.ImageGeneration`, returns bare `ImageGenerationResponse`. |
| **PAR-BF-OAI-211** | Image generation streaming supported (`bifrost-openai.md:99`) | MISSING — "Provider stubbed (`stubs.go:21-23`)". `ImageGenerationStream` → `notImplemented("image_generation_stream")`. | HAVE — `stream:true` → `provider.ImageGenerationStream`, SSE-framed; `[DONE]`. |
| **PAR-BF-OAI-010** | `POST /v1/images/edits` registered with fasthttp (`bifrost-openai.md:21`) | MISSING — "Schema exists (`internal/schemas/images.go:28-38`) but no route; provider stubbed (`stubs.go:25-27`)". `ImageEdit` → `notImplemented("image_edit")`. | HAVE — **multipart** route + handler → `provider.ImageEdit`, returns bare `ImageGenerationResponse`. |
| **PAR-BF-OAI-011** | `POST /v1/images/variations` registered with fasthttp (`bifrost-openai.md:22`) | MISSING — "Schema exists (`internal/schemas/images.go:40-48`) but no route; provider stubbed (`stubs.go:29-31`)". `ImageVariation` → `notImplemented("image_variation")`. | HAVE — **multipart** route + handler → `provider.ImageVariation`, returns bare `ImageGenerationResponse`. |

> NOTE on stub line numbers: the matrix cites `stubs.go:41-47` etc.; the live
> `internal/providers/openai/stubs.go` (read @ authoring) places the methods at:
> `ImageGeneration` :17-19, `ImageGenerationStream` :21-23, `ImageEdit` :25-27,
> `ImageVariation` :29-31, `Speech` :33-35, `SpeechStream` :37-39, `Transcription`
> :41-43, `TranscriptionStream` :45-47. The matrix cites are slightly stale on the
> exact line spans but correct on the substance (all eight are 501 stubs). Use the
> live spans; the executor must re-grep at P1.

Matrix flips at closeout (§4 T-close), in `.planning/parity/matrix/bifrost-openai.md`:
- PAR-BF-OAI-007 → HAVE (cite the speech route + handler + impl, raw-bytes body).
- PAR-BF-OAI-209 → HAVE (Option A) / STAYS MISSING + escalated (Option B; §1.6).
- PAR-BF-OAI-008 → HAVE (cite the transcriptions route + multipart parse + impl).
- PAR-BF-OAI-210 → HAVE (Option A) / STAYS MISSING + escalated (Option B; §1.6).
- PAR-BF-OAI-009 → HAVE (cite the generations route + handler + impl).
- PAR-BF-OAI-211 → HAVE (Option A) / STAYS MISSING + escalated (Option B; §1.6).
- PAR-BF-OAI-010 → HAVE (cite the edits route + multipart + impl).
- PAR-BF-OAI-011 → HAVE (cite the variations route + multipart + impl).

### 1.1 The OpenAI-shape vs admin-envelope decision (BINDING — inherited from bf-openai-1 §1.1)

**`/v1/*` routes return OpenAI shapes, NOT the `{data,error}` admin envelope.**
This is g0router's existing, verified convention (chat returns bare
`*ChatResponse`; embeddings bare `*EmbeddingResponse`; completions bare
`*TextCompletionResponse` — `internal/api/completions.go:135-145`).

Therefore the bf-openai-2 handlers:
- **Speech** is the ONLY non-JSON success case: on success it writes the RAW
  `SpeechResponse.Audio` bytes with `Content-Type = SpeechResponse.ContentType`
  (e.g. `audio/mpeg`), status 200. NO `jsonMarshal`, NO `{data}`, NO `{error}`
  wrapper. (`SpeechResponse` is `{Audio []byte (json:"-"), ContentType string
  (json:"-")}` — `internal/schemas/audio.go:13-16`; both fields are `json:"-"`,
  so JSON-marshalling it would emit `{}` — proof the body MUST be the raw bytes.)
- **Transcription / ImageGeneration / ImageEdit / ImageVariation** on success
  write the bare OpenAI object (`*TranscriptionResponse`, `*ImageGenerationResponse`)
  via `jsonMarshal` → 200 `application/json`, mirroring completions.go:135-145.
- **All errors** call `writeError(ctx, status, errType, message, code)`
  (`internal/api/errors.go`) — the OpenAI `{"error":{...}}` shape — NOT the admin
  envelope. The api package does not import `internal/admin`.
- **Streaming** uses `text/event-stream` + `Cache-Control: no-cache` +
  `Connection: keep-alive` + `writeSSEStream(streamCtx, ctx, ch)` + `[DONE]`,
  mirroring `internal/api/completions.go:100-122`.

The `{data,error}` admin envelope is **FORBIDDEN** on these routes (§6). Bifrost's
`BifrostError`/`event_id` (301/302/303) is a VARIANT-by-design escalation
(BIFROST-MAP §224, matrix:301-303 VAR) and rides bf-openai-4 if at all — NOT this plan.

### 1.2 The provider-method implementation approach (BINDING — Option A, mirrors bf-openai-1 §1.2)

The MAP marks 007..011/209..211 BUILD because the schemas + interface methods
already exist (`internal/schemas/provider.go:86-94`); the gap is "wire the route
over the stubbed provider method". But the eight openai-provider methods are real
501 stubs (`internal/providers/openai/stubs.go:17-47`). Two sound options, per the
bf-openai-1 precedent:

**Option A (RECOMMENDED — implement the methods for the openai provider).** OpenAI's
`/v1/audio/speech`, `/v1/audio/transcriptions`, `/v1/images/generations`,
`/v1/images/edits`, `/v1/images/variations` are real, documented, stable upstream
endpoints. The non-multipart, non-speech impls are near-verbatim copies of the
SHIPPED `embedding.go:12-67` / `completions.go:16-71` transport:
`p.client.AcquireRequest/Response`, `req.SetRequestURI(p.baseURL + "/v1/...")`,
`req.Header.SetMethod(POST)`, `utils.SetAuthHeader`, `utils.SetJSONBody` (JSON
endpoints) or a multipart body builder (speech is JSON-in/bytes-out; transcriptions
& image edits/variations are multipart-in), `p.client.Do`, status-check →
`p.errorConverter.Convert(...)`, then either `utils.ReadJSONBody(resp, &result)`
(JSON responses) or `resp.Body()` copy + `Content-Type` capture (speech bytes).
Streams mirror `completions.go:75-166` (SSE drain via `utils.NewSSEScanner`,
`[DONE]` terminator, malformed → `streamError` per AUD-045, post-hook per AUD-047).

This is **sound** because: it is the identical transport the openai provider
already ships for chat/embeddings/completions; it is hermetically testable with
`httptest.NewServer` + `p.baseURL = srv.URL` (the exact pattern at
`internal/providers/openai/stream_test.go:27`, reused by completions_test.go); and
it makes NO claim about Bifrost internals (ESC-REF-ABSENT-safe). For multipart
(transcriptions/edits/variations) the provider builds the outbound multipart body
from the already-parsed `[]byte` fields on the request schema
(`TranscriptionRequest.File`, `ImageEditRequest.{Image,Mask}`,
`ImageVariationRequest.Image`) — see §1.5.

**Binding consequence:** implementing these eight methods means the existing
`TestNotImplementedStubs` sub-cases for `ImageGeneration`, `ImageEdit`,
`ImageVariation`, `Speech`, `Transcription` (`internal/providers/openai/openai_test.go:38-42`)
WILL break (they assert 501). Those FIVE sub-cases MUST be REMOVED from that table
and REPLACED by the new hermetic success/error tests in new
`internal/providers/openai/{audio,images}_test.go`. (The stream methods —
`SpeechStream`/`TranscriptionStream`/`ImageGenerationStream` — are NOT in the
`TestNotImplementedStubs` table today, so no table edit is needed for them; they
are covered freshly in the new test files.) The remaining sub-cases (Responses,
ResponsesStream, File*, Batch*, CountTokens) stay UNTOUCHED. This is the ONLY edit
to a pre-existing openai-provider test file (§3).

**Option B (FALLBACK — handler-only, surface the 501 cleanly; per-method, narrow).**
For any single method whose upstream transport cannot be made to pass hermetically
at impl, register the route + handler; the handler calls the provider method, which
returns the existing `not_implemented`/501; the handler maps that to
`writeError(ctx, 501, "not_implemented", ...)`. This closes the route-registration
row (007/008/009/010/011) but NOT the corresponding streaming row (209/210/211).
Use ONLY per-method if Option A cannot pass hermetically (it should — the JSON
methods are identical to embeddings; speech bytes + multipart are bounded
extensions). If Option B is taken for a method, its streaming row STAYS MISSING and
is escalated honestly in WORKFLOW.md (NEVER mark a streaming row HAVE on a 501).

**Default: Option A for all eight methods.** Other 42 providers' Speech/
Transcription/Image* stubs are NOT touched (they keep returning 501; only the
openai provider gains the real methods — §3 FORBIDDEN list). This PARTIALLY
unblocks the w7-prov-media deferral — see §1.7.

### 1.3 Audio/Images Go contract (mirrors bf-openai-1 §1.4)

**Schemas (REUSE — already exist, no change expected):**
`internal/schemas/audio.go` provides `SpeechRequest` (:4-10), `SpeechResponse`
(:13-16, `Audio []byte` + `ContentType string`, both `json:"-"`),
`TranscriptionRequest` (:18-27, `File []byte` `json:"-"` + form fields),
`TranscriptionResponse` (:29-37), `TranscriptionWord`/`TranscriptionSegment`.
`internal/schemas/images.go` provides `ImageGenerationRequest` (:4-13),
`ImageGenerationResponse` (:15-19) + `ImageData` (:22-26), `ImageEditRequest`
(:28-38, `Image []byte`+`Mask []byte` `json:"-"`), `ImageVariationRequest`
(:40-48, `Image []byte` `json:"-"`). If a field is genuinely missing at impl, set
it in the provider, NOT via a schema edit, unless absence forces an additive struct
field (additive-only, decision 2) — record in WORKFLOW + add the file to §7.

> Soundness note: `SpeechResponse` and `ImageGenerationResponse` have **no `Usage`
> field** (grep-confirmed). So the usage-recording glue (`recordNonStream`) records
> 0 prompt/0 completion tokens for these routes (mirroring how completions.go:147-152
> guards `resp.Usage != nil`). That is acceptable — these endpoints do not return
> token usage in the OpenAI wire shape. Do NOT invent a Usage field.

**Provider transport (Option A — NEW files):**

| Method | Mirrors | Behavior |
|---|---|---|
| `Speech(ctx, key, *SpeechRequest) (*SpeechResponse, *ProviderError)` | `embedding.go:12-67` | POST `p.baseURL + "/v1/audio/speech"`; auth; JSON body; status → `errorConverter.Convert`; on 200 copy `resp.Body()` into `SpeechResponse.Audio` and `string(resp.Header.ContentType())` into `.ContentType`. NO `ReadJSONBody`. `RequestType:"speech"`. |
| `SpeechStream(ctx, postHookRunner, key, *SpeechRequest) (chan *StreamChunk, *ProviderError)` | `completions.go:75-166` | `streamReq.Stream`-equivalent; POST; goroutine drains via `NewSSEScanner`; `[DONE]`; malformed → `streamError` (AUD-045); post-hook (AUD-047). `RequestType:"speech_stream"`. (§1.6 caveat.) |
| `Transcription(ctx, key, *TranscriptionRequest) (*TranscriptionResponse, *ProviderError)` | `embedding.go` + multipart builder (§1.5) | POST `p.baseURL + "/v1/audio/transcriptions"`; **multipart body** (file + model + optional fields); auth; status check; `ReadJSONBody` → `TranscriptionResponse`. `RequestType:"transcription"`. |
| `TranscriptionStream(ctx, postHookRunner, key, *TranscriptionRequest) (chan *StreamChunk, *ProviderError)` | `completions.go:75-166` + multipart | multipart body w/ `stream=true`; SSE drain; `[DONE]`; AUD-045/047. `RequestType:"transcription_stream"`. |
| `ImageGeneration(ctx, key, *ImageGenerationRequest) (*ImageGenerationResponse, *ProviderError)` | `embedding.go:12-67` | POST `p.baseURL + "/v1/images/generations"`; JSON body; `ReadJSONBody` → `ImageGenerationResponse`. `RequestType:"image_generation"`. |
| `ImageGenerationStream(ctx, postHookRunner, key, *ImageGenerationRequest) (chan *StreamChunk, *ProviderError)` | `completions.go:75-166` | `stream:true` JSON body; SSE drain; `[DONE]`; AUD-045/047. `RequestType:"image_generation_stream"`. |
| `ImageEdit(ctx, key, *ImageEditRequest) (*ImageGenerationResponse, *ProviderError)` | `embedding.go` + multipart (§1.5) | POST `p.baseURL + "/v1/images/edits"`; **multipart** (image + optional mask + prompt + fields); `ReadJSONBody` → `ImageGenerationResponse`. `RequestType:"image_edit"`. |
| `ImageVariation(ctx, key, *ImageVariationRequest) (*ImageGenerationResponse, *ProviderError)` | `embedding.go` + multipart (§1.5) | POST `p.baseURL + "/v1/images/variations"`; **multipart** (image + fields); `ReadJSONBody`. `RequestType:"image_variation"`. |

These REPLACE the eight stubs in `internal/providers/openai/stubs.go:17-47` (delete
those eight funcs from stubs.go; move them — now implemented — to audio.go/images.go
in the openai provider package). The remaining stubs in stubs.go (Responses,
ResponsesStream, File*, Batch*, CountTokens) + the `notImplemented` helper are
UNTOUCHED.

**Handlers (NEW files `internal/api/audio.go`, `internal/api/images.go`):** mirror
`CompletionsHandler` (`internal/api/completions.go`) — same struct fields
(`router completionsResolver`-style seam, `usageRecorder`, `pendingTracker`,
`detailCapture`, `vkGate`, `pinnedResolver`), same additive setters
(`SetUsageRecorder`/`SetPendingTracker`/`SetDetailCapture`/`SetVKGate`/`SetVKPinnedResolver`),
same `recordGlue()`, same VK gate placement (after Resolve, before dispatch),
same `gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}`.

| Aspect | Contract |
|---|---|
| Resolver seam | REUSE the existing `completionsResolver` interface (`internal/api/completions.go:26-28`: `Resolve(model string) (schemas.Provider, schemas.Key, error)`) OR declare a private alias per handler if package-locality is cleaner; `inference.Router.Resolve` satisfies it. Do NOT widen the seam. |
| Speech parse | JSON: `json.Unmarshal(raw, &schemas.SpeechRequest)`; invalid JSON → `writeError(400,"invalid_request_error","invalid JSON body",nil)`. |
| Transcription parse | **multipart** (§1.5): `ctx.Request.Header.ContentType()` must be `multipart/form-data`; parse via fasthttp `ctx.MultipartForm()`; map `file` → `TranscriptionRequest.File`, `model`/`language`/`prompt`/`response_format`/`temperature`/`timestamp_granularities[]`/`stream` form values → fields; missing `file` or `model` → `writeError(400,"invalid_request_error",...)`. |
| ImageGeneration parse | JSON: `json.Unmarshal(raw, &schemas.ImageGenerationRequest)`. |
| ImageEdit parse | **multipart**: `image` (required) + optional `mask` files → `Image`/`Mask`; `prompt`/`model`/`n`/`size`/`response_format`/`user` form values. |
| ImageVariation parse | **multipart**: `image` (required) file → `Image`; `model`/`n`/`size`/`response_format`/`user` form values. |
| Resolve | `h.router.Resolve(req.Model)`; error → `writeError(400,"invalid_request_error",err.Error(),nil)`. |
| VK gate | `x-g0-vk` gate + pinned-key override, IDENTICAL to `completions.go:71-91` (reuse `h.vkGate`/`h.pinnedResolver`). Apply BEFORE dispatch. Record endpoint = the route path. |
| Usage glue | include the additive setters + `recordGlue()` so `routes_openai.go` wires usage symmetrically (completions.go:155-158). record under the route's endpoint path. |
| Non-stream dispatch (JSON-out) | call provider method; on `*ProviderError` → `writeError(perr.StatusCode|502, perr.Type, perr.Message, perr.Code)` (completions.go:124-133); on success `jsonMarshal(resp)` → 200 `application/json`; marshal failure → plain-text 500 "internal error" (completions.go:135-145, AUD-010). |
| Non-stream dispatch (SPEECH bytes-out) | call `provider.Speech`; on `*ProviderError` → `writeError`; on success: `ctx.SetStatusCode(200)`; `ctx.SetContentType(resp.ContentType)` (fallback `"application/octet-stream"` if empty); `ctx.SetBody(resp.Audio)`. NO jsonMarshal. (§1.1.) |
| Stream dispatch | when stream requested (`req.Stream` for speech/image JSON; `stream` form-field for transcription): set `text/event-stream`+`Cache-Control: no-cache`+`Connection: keep-alive` (completions.go:101-103); call the `*Stream` provider method; pre-stream `*ProviderError` → `writeError`; else `writeSSEStream(streamCtx, ctx, ch)` with `withRequestCancel(ctx)` (completions.go:116-120). |

**Construction:** `NewAudioHandler(router *inference.Router) *AudioHandler`,
`NewImagesHandler(router *inference.Router) *ImagesHandler` (mirror
`NewCompletionsHandler`, completions.go:31-33). NO `New(...)`/
`RegisterOpenAIRoutes(...)` signature change beyond the additive symmetry already
present (decision 9) — the handlers are constructed INSIDE `RegisterOpenAIRoutes`
like `completions := api.NewCompletionsHandler(router_)` (`routes_openai.go:41`).

> Handler-count decision: ONE handler per file/domain — `AudioHandler` owns
> `Speech` + `Transcription` (`internal/api/audio.go`), `ImagesHandler` owns
> `Generations` + `Edits` + `Variations` (`internal/api/images.go`). Each public
> method is a distinct `func (h *AudioHandler) Speech(ctx)` etc. (so route lines
> read `r.POST("/v1/audio/speech", audio.Speech)`). This matches the index row
> "`internal/api/{audio,images}.go`" (BIFROST-MAP:297).

### 1.4 Speech-audio-bytes handling decision (BINDING)

`/v1/audio/speech` is the ONE route in this plan whose success body is **NOT JSON**.
OpenAI's `/v1/audio/speech` returns raw audio bytes (default `audio/mpeg`; format
governed by the request `response_format`). The decision:

1. The provider `Speech` method copies the upstream `resp.Body()` into
   `SpeechResponse.Audio` and the upstream `Content-Type` header into
   `SpeechResponse.ContentType` (do NOT `ReadJSONBody` — the body is binary).
2. The handler writes those bytes verbatim with that Content-Type — NO JSON marshal,
   NO `{data}`/`{error}` envelope. Fallback Content-Type when empty =
   `application/octet-stream`.
3. Proof obligation (test): the speech success body equals the upstream bytes
   exactly AND the response `Content-Type` equals the upstream Content-Type AND the
   body is NOT valid JSON `{...}` with a `data`/`error` key (it is raw audio).

This is sound and ESC-REF-ABSENT-safe (OpenAI's documented binary response; no
Bifrost internal). The `SpeechResponse` schema already models exactly this
(`Audio []byte`, `ContentType string`, both `json:"-"` — proving the design intent
is a non-JSON body).

### 1.5 Multipart-transcription / image-edit / image-variation handling decision (BINDING)

Three routes take **multipart/form-data** (file upload): `/v1/audio/transcriptions`,
`/v1/images/edits`, `/v1/images/variations`. There is **no existing multipart
parsing anywhere in `internal/`** (grep-confirmed) — this plan introduces the first
multipart handling. Scope it tightly per Go-port note #6 (matrix:287: "Multipart
parsing for images/audio/video … should be implemented per-endpoint with explicit
form field whitelists").

**Inbound parse (handler side):**
- Detect `multipart/form-data` via `bytes.HasPrefix(ctx.Request.Header.ContentType(), []byte("multipart/form-data"))`; if absent → `writeError(400,"invalid_request_error","expected multipart/form-data",nil)`.
- Parse via fasthttp's `ctx.MultipartForm()` (returns `*multipart.Form`, error). On error → `writeError(400,...)`.
- **Explicit field whitelist** (reject-by-omission; unknown fields ignored, never forwarded blindly):
  - transcriptions: file part `file` (required) → read into `TranscriptionRequest.File`; values `model`(required), `language`, `prompt`, `response_format`, `temperature`(parse float), `timestamp_granularities[]`, `stream`(parse bool).
  - image edits: file parts `image`(required), `mask`(optional) → `Image`/`Mask`; values `prompt`(required), `model`, `n`(int), `size`, `response_format`, `user`.
  - image variations: file part `image`(required) → `Image`; values `model`, `n`(int), `size`, `response_format`, `user`.
- A multipart file part is read via `fh.Open()` + `io.ReadAll` into the `[]byte`
  schema field. Missing required part/field → `writeError(400,...)`.
- A small private helper (e.g. `readMultipartFile(form, field) ([]byte, bool, error)`)
  MAY live in `internal/api/audio.go` or a shared `internal/api/multipart.go`
  (executor's choice; if shared, add `multipart.go`+`multipart_test.go` to §7). Keep
  it api-package-local; no new exported surface beyond the handlers.

**Outbound build (provider side):** the openai provider methods build the OUTBOUND
multipart body from the already-parsed `[]byte` fields, using stdlib
`mime/multipart.NewWriter` over a `bytes.Buffer`: write the file part(s) +
whitelisted form values, set `req.Header.SetContentType(writer.FormDataContentType())`,
`req.SetBody(buf.Bytes())`. (Do NOT reuse `utils.SetJSONBody` for these — they are
multipart, not JSON.) An additive `utils.SetMultipartBody`-style helper MAY be added
to `internal/providers/utils/helpers.go` IF it stays generic and additive (then add
`helpers.go`+a `helpers_test.go` to §7 with a WORKFLOW note); default = inline in
the provider methods to avoid touching the shared utils file.

**Hermetic test for multipart:** the provider tests use a canned in-memory multipart
body (build with `mime/multipart` in the test) and an `httptest.NewServer` that
asserts the inbound `Content-Type` starts with `multipart/form-data` and that the
file part round-trips; the handler tests build a `fasthttp.RequestCtx` with a
multipart body + the multipart Content-Type and assert the parsed fields reach the
fake provider. NO real network, NO real files on disk (use `bytes`/in-memory only).

### 1.6 Streaming-soundness caveat for speech/transcription/image streams (CONDITIONAL — decide at impl)

The streaming rows (209/210/211) assert "<kind> streaming supported". OpenAI's
public streaming for these surfaces is heterogeneous:
- Image generation streaming (`stream:true` on `/v1/images/generations`,
  newer models) emits SSE `partial_image`/`completed` events → SSE drain is the
  correct shape; mirror `completions.go` exactly.
- Transcription streaming (`stream:true` on `/v1/audio/transcriptions`, gpt-4o
  transcribe) emits SSE transcript delta events → SSE drain is correct.
- Speech streaming is the weakest fit: OpenAI's speech streaming may emit raw audio
  chunks (chunked transfer) rather than SSE `data:` frames.

**Binding decision:** implement all three `*Stream` methods using the SSE-drain
template (`completions.go:75-166`) and pass chunks through `writeSSEStream`. This is
sound for image + transcription streaming (real SSE). For SPEECH streaming, if at
impl the upstream does NOT emit SSE-framed bytes, the SSE-drain template will not
cleanly frame raw audio — in that case take per-method **Option B for 209 only**
(register the route, `SpeechStream` returns its 501; mark 209 STAYS MISSING +
escalate honestly; 007 non-stream speech still ships HAVE). NEVER mark 209 HAVE on a
501 and NEVER fabricate an SSE frame around raw audio. Default = Option A SSE-drain;
fall to Option B for 209 ONLY if the upstream shape forbids hermetic SSE framing.
Record the actual decision taken in WORKFLOW.md.

(For the JSON-shaped streams — image-gen, transcription — there is no `text` delta
concern: `ProcessPassthroughStream`/`writeSSEStream` forward the raw chunk shape, so
no schema change is needed, per bf-openai-1 ESC-STREAMCHUNK-FIELD.)

### 1.7 PARTIALLY unblocks w7-prov-media (CROSS-REFERENCE — informational, NOT scope)

w7-prov-media (`.planning/parity/plans/w7-prov-media.md`) DEFERRED 11 media
providers. Its binding rationale (ESC-M2 §508, ESC-M4 §514-525) is explicit:

> "The real blocker for image/STT/TTS parity is the **absence of v1 media routes**
> (`/v1/images/generations`, `/v1/audio/transcriptions`, `/v1/audio/speech`) and
> any router media-dispatch path … **Open question:** should a future wave add the
> media transport routes (`internal/api/images.go`, `internal/api/audio.go` +
> `routes_openai.go` registration) so the 11 deferred adapters become
> buildable+reachable?"

**bf-openai-2 answers ESC-M4: it ADDS exactly those routes** — `internal/api/audio.go`,
`internal/api/images.go`, and the `routes_openai.go` registration. This **PARTIALLY
unblocks** the w7-prov-media deferral: after bf-openai-2, the `/v1/audio/*` and
`/v1/images/*` ROUTES EXIST and are reachable, so the deferred media providers'
methods become *route-reachable*. **It does NOT build the deferred media provider
ADAPTERS** — those (deepgram/assemblyai STT, nanobanana/fal-ai/stability-ai/
black-forest-labs/recraft/sdwebui/comfyui/huggingface image, runwayml video) remain
the w7-prov-media deferred scope and are a SEPARATE follow-up wave. bf-openai-2
implements ONLY the **openai** provider's Speech/Transcription/Image* methods (the
one provider with real, documented audio/image HTTP APIs). The video gap (runwayml,
ESC-M3) is NOT touched — there is no video method on the interface and no video
route in this plan.

**Action at closeout:** add a note to `.planning/parity/plans/open-questions.md`
(and the w7-prov-media ESC-M4 follow-up) recording that the ROUTE-existence blocker
is now resolved; the remaining w7-prov-media work is purely provider-adapter
transport for the 11 deferred providers.

### 1.8 routes_openai.go registration (serial-slot additive, §3)

Construct + wire the two handlers alongside the existing ones, and append the route
lines, grouped with the other `/v1/*` POSTs (`routes_openai.go:101-105`). All
`/v1/audio/*` and `/v1/images/*` paths are distinct static paths (no `{param}`
precedence concern):

```go
// (in the handler-construction block, after `completions := ...` at :41)
audio := api.NewAudioHandler(router_)
images := api.NewImagesHandler(router_)
// usage glue — extend the existing if-blocks (mirror completions wiring :46,:52,:58)
if recorder != nil { audio.SetUsageRecorder(recorder); images.SetUsageRecorder(recorder) }
if tracker  != nil { audio.SetPendingTracker(tracker);  images.SetPendingTracker(tracker)  }
if detail   != nil { audio.SetDetailCapture(detail);    images.SetDetailCapture(detail)    }
if st != nil {
    audio.SetVKGate(vkGate);            images.SetVKGate(vkGate)            // reuse vkGate built at :81
    audio.SetVKPinnedResolver(selector); images.SetVKPinnedResolver(selector) // reuse selector built at :89
}

// (in the route block, after :105 `r.POST("/v1/completions", completions.Handle)`)
r.POST("/v1/audio/speech", audio.Speech)
r.POST("/v1/audio/transcriptions", audio.Transcription)
r.POST("/v1/images/generations", images.Generations)
r.POST("/v1/images/edits", images.Edits)
r.POST("/v1/images/variations", images.Variations)
```

The `vkGate`/`selector`/`recorder`/`tracker`/`detail` are already constructed
(`routes_openai.go:81,89` + the params); REUSE them — do NOT rebuild. The new
construction + wiring is additive; no existing line is deleted.

### NOT in scope (explicit — FORBIDDEN)

- **The w7-prov-media DEFERRED provider adapters** — deepgram/assemblyai (STT),
  nanobanana/fal-ai/stability-ai/black-forest-labs/recraft/sdwebui/comfyui/
  huggingface (image), runwayml (video). bf-openai-2 implements ONLY the **openai**
  provider's Speech/Transcription/ImageGeneration/ImageEdit/ImageVariation methods.
  Touching `internal/providers/{deepgram,assemblyai,...}` or
  `internal/providers/catalog/*` is an automatic REJECT (§1.7).
- **Video** (`/v1/videos*`, runwayml) — ESC (BIFROST-MAP:012-017,112; matrix:012-013
  MISSING). No video method on the interface; no video route here.
- **The ESC rows** — no responses-rewrite/normalization, no rerank/ocr,
  no containers/async/WS, no raw-passthrough (205). Touching any ESC surface = REJECT.
- **Other bf-openai plans' surfaces** — no completions (bf-openai-1, SHIPPED — do
  NOT re-touch), no files/batches (bf-openai-3), no responses-extras/SSE-correctness/
  compaction (bf-openai-4). Do not touch their schemas/handlers/stubs.
- **The `{data,error}` admin envelope on `/v1/*`** (§1.1) — FORBIDDEN. No import of
  `internal/admin` from `internal/api`.
- **The remaining provider stubs** in `internal/providers/openai/stubs.go`
  (Responses/ResponsesStream/File*/Batch*/CountTokens) — UNTOUCHED.
- **The other 42 providers' Speech/Transcription/Image* methods** — UNTOUCHED
  (they keep their 501).
- **All UI / e2e / mocks** — API routes, no UI contract. No `ui/**` edit, no
  playwright, no mock/seed.
- **No `New(...)`/`RegisterOpenAIRoutes(...)` signature change**, no `init()`, no
  global state, errors-as-values (`fmt.Errorf("ctx: %w")`), no panics.
- **No destructive DDL / no store touch** (these routes touch no store/migrate).

---

## 2. Precondition checks

Run all before any edit; abort and report to the orchestrator on any failure.

```bash
# P0 — clean tree; record <base>
git status --porcelain     # must be empty (explicit `git add <file>`, never -A)
git rev-parse HEAD         # record as <base> for §5

# P1 — the audio/images gap is REAL (no route/handler anywhere)
grep -rn '/v1/audio\|/v1/images' internal/ | grep -v _test.go
# ^ expect NOTHING (no audio/images route or handler). bf-openai-1 added only /v1/completions.
test ! -e internal/api/audio.go   && echo "audio handler gap OK"
test ! -e internal/api/images.go  && echo "images handler gap OK"
test ! -e internal/providers/openai/audio.go  && echo "provider audio impl gap OK"
test ! -e internal/providers/openai/images.go && echo "provider images impl gap OK"
grep -n 'notImplemented("speech")\|notImplemented("transcription")\|notImplemented("image_generation")\|notImplemented("image_edit")\|notImplemented("image_variation")' internal/providers/openai/stubs.go  # the 5 non-stream + 3 stream stubs to replace (:17-47)
grep -rn 'multipart' internal/ | grep -v _test.go   # expect NOTHING — bf-openai-2 introduces the first multipart parse

# P2 — reused surfaces present (the de-risk)
grep -n 'type SpeechRequest\|type SpeechResponse\|type TranscriptionRequest\|type TranscriptionResponse' internal/schemas/audio.go
grep -n 'type ImageGenerationRequest\|type ImageGenerationResponse\|type ImageEditRequest\|type ImageVariationRequest' internal/schemas/images.go
grep -n 'Speech\|Transcription\|ImageGeneration\|ImageEdit\|ImageVariation' internal/schemas/provider.go   # :86-94 interface methods
grep -n 'func (r \*Router) Resolve\b' internal/inference/router.go
grep -n 'func SetAuthHeader\|func SetJSONBody\|func ReadJSONBody\|func NewSSEScanner' internal/providers/utils/*.go
grep -n 'func writeError\|func jsonMarshal\|func writeSSEStream\|func withRequestCancel\|func requestHeadersFromCtx' internal/api/*.go
grep -n 'func NewCompletionsHandler\|type completionsResolver\|func NewVKGate' internal/api/*.go
grep -n 'func streamError' internal/providers/openai/chat.go   # :15

# P3 — the completions pattern (bf-openai-1, SHIPPED) is present to mirror
grep -n 'func (h \*CompletionsHandler) Handle\|func (h \*CompletionsHandler) recordGlue' internal/api/completions.go
grep -n 'func (p \*Provider) TextCompletion\b\|func (p \*Provider) TextCompletionStream' internal/providers/openai/completions.go
grep -n 'r.POST("/v1/completions"' internal/server/routes_openai.go   # :105 (the slot bf-openai-1 released)

# P4 — provider transport templates present (Option A)
grep -n 'func (p \*Provider) Embedding\b' internal/providers/openai/embedding.go     # :12
grep -n 'p.baseURL = srv.URL' internal/providers/openai/stream_test.go               # hermetic pattern :27
grep -n 'ImageGeneration\|ImageEdit\|ImageVariation\|Speech\|Transcription' internal/providers/openai/openai_test.go  # :38-42 (5 sub-cases to remove)

# P5 — routes_openai.go SERIAL SLOT is FREE (bf-openai-1 released it on close)
git log --oneline -8 -- internal/server/routes_openai.go
# Orchestrator MUST confirm no concurrent bf-openai-* plan holds an unmerged
# routes_openai.go edit before bf-openai-2 begins T-routes. bf-openai-2 is SECOND
# in the chain (after the SHIPPED bf-openai-1). bf-openai-2 TAKES the slot, RELEASES to bf-openai-3.

# P6 — green at base (HERMETIC; no network)
go test ./... && go vet ./... && go build ./...     # exit 0 (untouched-green baseline)
```

---

## 3. Exclusive file ownership

After bf-openai-2 merges, CREATE files are owned by bf-openai-2; later plans
consume, never edit (decision 7).

**CREATE — provider transport (NEW, Option A):**

| File | Contract |
|---|---|
| `internal/providers/openai/audio.go` | `Speech` + `SpeechStream` + `Transcription` + `TranscriptionStream` (Option A, §1.3) — moved from stubs.go, now implemented; mirror `embedding.go`+`completions.go` transport; multipart outbound for transcription (§1.5); speech bytes-out (§1.4). No `init()`; errors-as-values; correct `RequestType`. |
| `internal/providers/openai/audio_test.go` | RED first. Hermetic `httptest.NewServer` + `p.baseURL = srv.URL` (mirror `stream_test.go:27`, `completions_test.go`): `Speech` success → `Audio` bytes == upstream body + `ContentType` == upstream header; upstream-500 → `*ProviderError` status 500; `Transcription` success (canned multipart in test; assert inbound CT starts `multipart/form-data`) → `TranscriptionResponse`; stream methods yield N chunks then `[DONE]` + malformed → one error chunk + abort (AUD-045). NO real network/files. |
| `internal/providers/openai/images.go` | `ImageGeneration` + `ImageGenerationStream` + `ImageEdit` + `ImageVariation` (Option A, §1.3) — moved from stubs.go, now implemented; JSON for generations, multipart for edits/variations (§1.5). No `init()`; errors-as-values; correct `RequestType`. |
| `internal/providers/openai/images_test.go` | RED first. Hermetic: `ImageGeneration` success → `ImageGenerationResponse`; upstream-500 → `*ProviderError`; `ImageEdit`/`ImageVariation` success (canned multipart; assert inbound multipart CT + file round-trip) → `ImageGenerationResponse`; `ImageGenerationStream` chunks + `[DONE]` + malformed abort. NO real network/files. |

**CREATE — api transport (NEW):**

| File | Contract |
|---|---|
| `internal/api/audio.go` | `AudioHandler` + `NewAudioHandler` + additive setters (VK/usage) + `recordGlue` + `Speech(ctx)` (JSON-in/bytes-out, §1.4) + `Transcription(ctx)` (multipart-in/JSON-out, §1.5), §1.3. OpenAI shapes only (§1.1); `writeError` for errors. |
| `internal/api/audio_test.go` | RED first. Hermetic, fake provider/resolver mirroring `completions_test.go`/`embeddings_test.go:14-34` (embed a shared base — e.g. `fakeMessagesProvider` — to satisfy the full `schemas.Provider` interface): Speech success → raw audio bytes body + Content-Type from provider (assert body is NOT JSON `{data}`/`{error}`); Transcription multipart success → bare `TranscriptionResponse` JSON (assert NO `data`/`error` wrapper); invalid JSON (speech) → 400 OpenAI error; non-multipart (transcription) → 400; provider 501 → 501 passthrough; stream → `text/event-stream` + `[DONE]`; VK-denied → 429 and provider NOT called; VK-pinned override; marshal failure → plain 500. |
| `internal/api/images.go` | `ImagesHandler` + `NewImagesHandler` + additive setters + `recordGlue` + `Generations(ctx)` (JSON) + `Edits(ctx)` (multipart) + `Variations(ctx)` (multipart), §1.3. OpenAI shapes only; bare `*ImageGenerationResponse` success. |
| `internal/api/images_test.go` | RED first. Hermetic fake provider/resolver: Generations success → bare `ImageGenerationResponse` (assert NO `data`/`error` wrapper); invalid JSON → 400; Edits/Variations multipart success → bare `ImageGenerationResponse`; non-multipart edits/variations → 400; provider 501 → 501; stream (generations) → `text/event-stream`+`[DONE]`; VK-denied → 429 + provider NOT called; VK-pinned override; marshal failure → plain 500. |

**OPTIONAL CREATE — shared multipart helpers (executor's choice, §1.5):**

| File | Change |
|---|---|
| `internal/api/multipart.go` (+ `_test.go`) | ONLY if a shared inbound-parse helper is cleaner than inline. api-package-local; no exported surface beyond the handlers. If created, add both to §7 + a WORKFLOW note. Default = inline in audio.go/images.go (no extra file). |

**EXTEND — provider stubs (REMOVE the eight now-implemented stubs):**

| File | Change |
|---|---|
| `internal/providers/openai/stubs.go` | DELETE `ImageGeneration`(:17-19), `ImageGenerationStream`(:21-23), `ImageEdit`(:25-27), `ImageVariation`(:29-31), `Speech`(:33-35), `SpeechStream`(:37-39), `Transcription`(:41-43), `TranscriptionStream`(:45-47) ONLY (they move to images.go/audio.go, implemented). The remaining stubs (Responses, ResponsesStream, File*, Batch*, CountTokens) + `notImplemented` helper are PRESERVED verbatim. Re-grep live spans at P1. |
| `internal/providers/openai/openai_test.go` | REMOVE the `ImageGeneration`, `ImageEdit`, `ImageVariation`, `Speech`, `Transcription` sub-cases from the `TestNotImplementedStubs` table (`:38-42`) — they no longer 501. The other sub-cases are PRESERVED. (This is the ONLY edit to a pre-existing openai test.) |

**OPTIONAL EXTEND — provider utils (executor's choice, §1.5):**

| File | Change |
|---|---|
| `internal/providers/utils/helpers.go` (+ `_test.go`) | ONLY if a generic `SetMultipartBody` helper is preferred over inline multipart construction. Additive, generic. If created, add both to §7 + a WORKFLOW note. Default = inline in the provider methods (no shared-util touch). |

**MODIFY — serial-slot route registration (additive only):**

| File | Change |
|---|---|
| `internal/server/routes_openai.go` | ADD the `audio`/`images` handler construction + wiring (reuse `vkGate`/`selector`/`recorder`/`tracker`/`detail`) + the FIVE route lines (§1.8). NOTHING else changes. SERIAL SLOT — only holder while live; RELEASE to bf-openai-3 on close. |

**FORBIDDEN:** everything else. Explicitly: the remaining openai stubs; the 42
other providers' Speech/Transcription/Image*; `internal/providers/catalog/*` and
the w7-prov-media deferred provider packages; all other `internal/api/*.go`
(chat/embeddings/messages/responses/models/completions bodies); `internal/schemas/*`
(REUSE audio.go/images.go; edit ONLY if a genuinely-absent field forces an additive
field, §1.3 — then add to §7 + WORKFLOW); all bf-openai-3/4 surfaces; all
`internal/admin/*`, `internal/store/*`, `internal/governance/*`, `internal/mcp/*`;
all `ui/**`; all video. Touching any of these is an automatic REJECT.

---

## 4. TDD tasks

Cadence (strict, AGENTS.md "TDD always"): **no Go impl file may exist before its
`_test.go` is committed RED.** `go test ./... && go vet ./... && go build ./...`
green at EVERY commit (a RED commit may fail ONLY the new package's targeted run;
prefer table/assertion failures over compile failures — scaffold the signatures so
the package compiles and the assertion fails). Order: provider impls → api handlers
→ serial-slot routes → closeout.

### T-prov-audio — STEP(a) RED, STEP(b) impl
STEP(a): write `internal/providers/openai/audio_test.go` (hermetic httptest +
canned multipart, §3). Run `go test ./internal/providers/openai/ -run 'Speech|Transcri'`
→ FAIL. Commit RED:
`phase-1/bf-openai-2: failing openai Speech/Transcription(+stream) tests (TDD red)`.
STEP(b): create `internal/providers/openai/audio.go` (Option A; speech bytes-out
§1.4; multipart out §1.5); DELETE the four audio stubs from `stubs.go`; REMOVE the
`Speech`+`Transcription` sub-cases from `openai_test.go`. Gates:
`go test ./... && go vet ./... && go build ./...` green. Commit:
`phase-1/bf-openai-2: implement openai Speech/Transcription (+streams)`.

*If a method cannot pass hermetically (speech-stream SSE caveat §1.6), STOP and
ESCALATE (§8); fall to Option B for that method only, leave its streaming row
MISSING. Do NOT fabricate a green.*

### T-prov-images — STEP(a) RED, STEP(b) impl
STEP(a): write `internal/providers/openai/images_test.go` (§3). Run
`go test ./internal/providers/openai/ -run 'Image'` → FAIL. Commit RED:
`phase-1/bf-openai-2: failing openai ImageGeneration/Edit/Variation(+stream) tests (TDD red)`.
STEP(b): create `internal/providers/openai/images.go` (Option A; multipart out for
edits/variations §1.5); DELETE the four image stubs from `stubs.go`; REMOVE the
`ImageGeneration`+`ImageEdit`+`ImageVariation` sub-cases from `openai_test.go`.
Gates green. Commit:
`phase-1/bf-openai-2: implement openai ImageGeneration/Edit/Variation (+stream)`.

### T-handler-audio — STEP(a) RED, STEP(b) impl
STEP(a): write `internal/api/audio_test.go` (fake provider/resolver + multipart
request bodies, §3). Run `go test ./internal/api/ -run 'Audio|Speech|Transcri'`
→ FAIL. Commit RED:
`phase-1/bf-openai-2: failing /v1/audio handler tests (TDD red)`.
STEP(b): create `internal/api/audio.go` (+ optional `multipart.go`, §1.5). Gates
green. Commit:
`phase-1/bf-openai-2: /v1/audio handlers (speech bytes-out, transcriptions multipart, SSE)`.

### T-handler-images — STEP(a) RED, STEP(b) impl
STEP(a): write `internal/api/images_test.go` (§3). Run
`go test ./internal/api/ -run 'Image'` → FAIL. Commit RED:
`phase-1/bf-openai-2: failing /v1/images handler tests (TDD red)`.
STEP(b): create `internal/api/images.go`. Gates green. Commit:
`phase-1/bf-openai-2: /v1/images handlers (generations/edits/variations, multipart, SSE)`.

### T-routes — serial-slot route registration
TAKE the serial slot (orchestrator confirms FREE at P5). Add the construction +
wiring + the FIVE route lines to `routes_openai.go` (§1.8). Gates:
`go test ./... && go vet ./... && go build ./...` green. Commit (ONE commit touches
the serial file):
`phase-1/bf-openai-2: register /v1/audio + /v1/images routes (serial slot)`.

### T-close — full gates + closeout
```bash
go test ./... && go vet ./... && go build ./...
go test ./internal/api/... ./internal/providers/openai/... -run 'Audio|Speech|Transcri|Image' -v
go test ./internal/providers/openai/ -run 'NotImplemented' -v   # remaining stubs still 501
```
Flip `.planning/parity/matrix/bifrost-openai.md`: 007/008/009/010/011 → HAVE;
209/210/211 → HAVE (Option A) / STAYS MISSING + escalated (Option B per method,
§1.6); cite the new routes + handlers + impls. Update `docs/WORKFLOW.md` (P6 base
observation, the per-method Option A vs B decision actually taken, the speech
bytes-out + multipart decisions, the speech-stream SSE caveat outcome, the
OpenAI-shape decision, the serial-slot take/release, the w7-prov-media partial-unblock
note). Append the w7-prov-media partial-unblock + any open items to
`.planning/parity/plans/open-questions.md`. Final commit:
`phase-1/bf-openai-2: close — audio+images routes HAVE; matrix flip; w7-prov-media route-blocker resolved`.
**On the close commit, RELEASE the routes_openai.go serial slot to bf-openai-3.**

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0. Diff gate is
**bf-openai-2 commit-range-scoped** (§7). NO e2e.

**Test gates (HERMETIC — no network, no real files)**
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/api/... ./internal/providers/openai/... -run 'Audio|Speech|Transcri|Image' -v`
  → exit 0, all pass (speech bytes-out + transcription multipart + image gen/edit/var
  + streams + invalid-input + provider-err + VK-denied + VK-pinned + marshal-fail).
- `go test ./internal/providers/openai/ -run 'NotImplemented' -v` → exit 0
  (the remaining NotImplemented sub-cases — Responses/File*/Batch*/CountTokens — still pass).

**TDD-order proof** — each impl file's covering test appears in an earlier-or-equal commit:
```bash
for pair in \
  "internal/providers/openai/audio_test.go:internal/providers/openai/audio.go" \
  "internal/providers/openai/images_test.go:internal/providers/openai/images.go" \
  "internal/api/audio_test.go:internal/api/audio.go" \
  "internal/api/images_test.go:internal/api/images.go" ; do
  tf=${pair%%:*}; im=${pair##*:}
  ct=$(git log --format=%ct --diff-filter=A -1 -- "$tf")
  cf=$(git log --format=%ct --diff-filter=A -1 -- "$im")
  [ "$ct" -le "$cf" ] || echo "TDD VIOLATION: $im"     # prints nothing
done
```

**Grep proofs**
```bash
# routes registered (serial-slot)
grep -n '/v1/audio/speech\|/v1/audio/transcriptions' internal/server/routes_openai.go
grep -n '/v1/images/generations\|/v1/images/edits\|/v1/images/variations' internal/server/routes_openai.go
grep -n 'api.NewAudioHandler\|api.NewImagesHandler' internal/server/routes_openai.go
# handlers exist, OpenAI shape (NOT admin envelope)
grep -n 'func (h \*AudioHandler) Speech\|func (h \*AudioHandler) Transcription' internal/api/audio.go
grep -n 'func (h \*ImagesHandler) Generations\|func (h \*ImagesHandler) Edits\|func (h \*ImagesHandler) Variations' internal/api/images.go
grep -n 'writeError' internal/api/audio.go internal/api/images.go
! grep -rn 'internal/admin' internal/api/audio.go internal/api/images.go && echo "no admin-envelope import OK"
! grep -n '"data"' internal/api/audio.go internal/api/images.go && echo "no {data} wrapper OK"
# speech bytes-out (NOT jsonMarshal) — Content-Type set from provider
grep -n 'SetContentType\|resp.ContentType\|resp.Audio\|SetBody' internal/api/audio.go
# multipart parse present (the first in the tree)
grep -n 'MultipartForm\|multipart/form-data' internal/api/audio.go internal/api/images.go
# provider methods implemented (Option A) — no longer stubs
grep -n 'func (p \*Provider) Speech\b\|func (p \*Provider) Transcription\b' internal/providers/openai/audio.go
grep -n 'func (p \*Provider) ImageGeneration\b\|func (p \*Provider) ImageEdit\b\|func (p \*Provider) ImageVariation\b' internal/providers/openai/images.go
grep -n 'p.baseURL + "/v1/audio/speech"\|p.baseURL + "/v1/audio/transcriptions"' internal/providers/openai/audio.go
grep -n 'p.baseURL + "/v1/images/generations"\|p.baseURL + "/v1/images/edits"\|p.baseURL + "/v1/images/variations"' internal/providers/openai/images.go
! grep -n 'notImplemented("speech")\|notImplemented("transcription")\|notImplemented("image_generation")\|notImplemented("image_edit")\|notImplemented("image_variation")' internal/providers/openai/stubs.go && echo "8 stubs removed OK"
# remaining stubs preserved
grep -n 'notImplemented("count_tokens")\|notImplemented("file_upload")\|notImplemented("batch_create")' internal/providers/openai/stubs.go
# no init(); errors-as-values
! grep -rn 'func init(' internal/api/audio.go internal/api/images.go internal/providers/openai/audio.go internal/providers/openai/images.go && echo "no init() OK"
! grep -rn 'panic(' internal/api/audio.go internal/api/images.go internal/providers/openai/audio.go internal/providers/openai/images.go && echo "no panic OK"
```

**SSE / OpenAI-shape / bytes proofs**
```bash
grep -n 'text/event-stream' internal/api/audio.go internal/api/images.go          # stream content-type
grep -n 'writeSSEStream' internal/api/audio.go internal/api/images.go
grep -n '\[DONE\]\|streamError\|NewSSEScanner' internal/providers/openai/audio.go internal/providers/openai/images.go
```
Plus runtime assertions in the tests:
- Speech success: response body == upstream audio bytes AND response Content-Type ==
  upstream Content-Type AND body is NOT JSON with a `data`/`error` key.
- Transcription/ImageGeneration/ImageEdit/ImageVariation success: body unmarshals to
  the bare OpenAI object AND contains NEITHER top-level `"data"` NOR `"error"` key.
- Multipart: provider receives a request whose Content-Type starts
  `multipart/form-data` and whose file part round-trips the input bytes.

**Negative / freeze proofs (bf-openai-2 commit-range — §7)**
```bash
R="<first-bf-openai-2>^..<last-bf-openai-2>"
# Only the sanctioned files changed:
git diff $R --name-only -- internal/ | grep -vE \
 'internal/providers/openai/(audio|audio_test|images|images_test|stubs|openai_test)\.go|internal/api/(audio|audio_test|images|images_test|multipart|multipart_test)\.go|internal/providers/utils/helpers(_test)?\.go|internal/server/routes_openai\.go' \
 | wc -l                                                                       # = 0
# Remaining openai stubs untouched (only the eight audio/image stubs removed):
git diff $R -- internal/providers/openai/stubs.go | grep -E '^-' | grep -ivE '^---|Speech|Transcription|ImageGeneration|ImageEdit|ImageVariation|notImplemented\("(speech|transcription|image)' | grep -iE 'func \(p \*Provider\)' | wc -l   # = 0
# No other api handler body changed:
git diff $R --name-only -- internal/api/ | grep -vE 'internal/api/(audio|audio_test|images|images_test|multipart|multipart_test)\.go' | wc -l   # = 0
# No catalog / other-provider / store/admin/governance/mcp/ui touched:
git diff $R --name-only -- internal/providers/catalog/ internal/store/ internal/admin/ internal/governance/ internal/mcp/ ui/ | wc -l   # = 0
git diff $R --name-only -- internal/providers/ | grep -vE 'internal/providers/openai/(audio|audio_test|images|images_test|stubs|openai_test)\.go|internal/providers/utils/helpers(_test)?\.go' | wc -l   # = 0
# routes_openai.go = exactly ONE commit, additive (no route deletions):
git log --oneline $R -- internal/server/routes_openai.go | wc -l              # = 1
git diff $R -- internal/server/routes_openai.go | grep -E '^-' | grep -vE '^---|^-$' | wc -l   # = 0 (no deletions)
```

---

## 6. Out of scope (restated, binding)

No `{data,error}` admin envelope on `/v1/*` (§1.1 — FORBIDDEN; speech returns raw
bytes, the JSON routes return bare OpenAI objects). No ESC rows. No w7-prov-media
DEFERRED provider adapters (only the openai provider's methods, §1.7). No video. No
other bf-openai plans' surfaces (completions/files/batches/responses-extras). No
edits to the remaining openai stubs or the 42 other providers' media methods. No
schema rewrite (REUSE audio.go/images.go; additive field ONLY if forced). No UI /
e2e / mocks. No `New(...)`/`RegisterOpenAIRoutes(...)` signature change, no `init()`,
no global state, no panics. No store/migrate. Matrix-vs-code contradiction (e.g. the
slightly-stale stub line spans) → reconcile honestly in the plan + WORKFLOW (§1
NOTE), never fabricate. If a method/stream can't pass hermetically → escalate (§8),
do not mark its row HAVE on a 501.

## 7. Diff-gate scope

bf-openai-* plans commit to main concurrently (disjoint from gov/mcp/core, serial
within the openai track), so a broad `<base>..HEAD` range can sweep in siblings.
The diff gate MUST be scoped to bf-openai-2's own commits:
`git log --oneline main | grep "^[0-9a-f]* phase-1/bf-openai-2:" | awk '{print $1}'`
then `git diff <first-bf-openai-2>^..<last-bf-openai-2> -- [file list]`.

`git diff <first>^..<last> --name-only` must be exactly a subset of:
```
internal/providers/openai/audio.go
internal/providers/openai/audio_test.go
internal/providers/openai/images.go
internal/providers/openai/images_test.go
internal/providers/openai/stubs.go            (DELETE the eight audio/image stubs only)
internal/providers/openai/openai_test.go      (REMOVE the five media sub-cases only)
internal/api/audio.go
internal/api/audio_test.go
internal/api/images.go
internal/api/images_test.go
internal/api/multipart.go                      (OPTIONAL — only if a shared parse helper is used)
internal/api/multipart_test.go                 (OPTIONAL — pairs with multipart.go)
internal/providers/utils/helpers.go            (OPTIONAL — only if SetMultipartBody added)
internal/providers/utils/helpers_test.go       (OPTIONAL — pairs with helpers.go)
internal/server/routes_openai.go               (serial-slot additive; ONE commit)
.planning/parity/matrix/bifrost-openai.md
.planning/parity/plans/open-questions.md
docs/WORKFLOW.md
```
Any file outside this list in the scoped diff is an automatic review REJECT.
`internal/schemas/{audio,images}.go` are deliberately ABSENT (REUSE; additive-field
edit only if §1.3 forces it — then add it to the list with a WORKFLOW note). The
two OPTIONAL pairs (`multipart.go`/`_test.go`, `utils/helpers.go`/`_test.go`) are
included ONLY if the executor chooses the shared-helper path (§1.5); default omits
them. All other api handlers, all other providers, `internal/providers/catalog`,
store/admin/governance/mcp, all `ui/**`, and all video are ABSENT — touching them is
an automatic REJECT. The `routes_openai.go` edit must appear in exactly ONE commit
(§5) and the serial slot is released to bf-openai-3 on close.

## 8. Escalations / decisions (explicit — recommended defaults, do not fabricate)

- **ESC-MEDIA-IMPL (RESOLVED at authoring — provider-method impl, binding default =
  Option A).** §1.2. OpenAI's `/v1/audio/{speech,transcriptions}` and
  `/v1/images/{generations,edits,variations}` are real, stable upstream endpoints;
  the openai provider's Speech/Transcription/ImageGeneration/ImageEdit/ImageVariation
  (+ streams) are implementable as near-verbatim copies of the SHIPPED
  `embedding.go`+`completions.go` transport, hermetically testable via
  `httptest.NewServer` + `p.baseURL = srv.URL`. **Fallback Option B** (handler
  surfaces the 501) closes only the route-registration row and leaves the streaming
  row MISSING — use ONLY per-method if Option A cannot pass hermetically. **Never
  mark a streaming row HAVE while the method returns 501.** RECOMMENDED: Option A.

- **ESC-SPEECH-BYTES (RESOLVED at authoring — non-JSON success body, binding).**
  §1.4. `/v1/audio/speech` returns RAW audio bytes with the upstream Content-Type,
  NOT a JSON envelope — `SpeechResponse{Audio []byte, ContentType string}` (both
  `json:"-"`) proves the design intent. The handler writes bytes verbatim; the
  provider copies `resp.Body()` + `resp.Header.ContentType()`. NO jsonMarshal, NO
  `{data}`/`{error}`. This is the only non-JSON success route in the plan.

- **ESC-MULTIPART (RESOLVED at authoring — multipart parse, binding).** §1.5.
  `/v1/audio/transcriptions`, `/v1/images/edits`, `/v1/images/variations` take
  `multipart/form-data`. There is NO existing multipart parse in `internal/`
  (grep-confirmed) — bf-openai-2 introduces the first, scoped per-endpoint with an
  EXPLICIT field whitelist (Go-port note #6, matrix:287). Inbound: fasthttp
  `ctx.MultipartForm()` → whitelisted fields → schema `[]byte`/value fields.
  Outbound: stdlib `mime/multipart.NewWriter` over a `bytes.Buffer`. Hermetic tests
  use canned in-memory multipart bodies — NO real files, NO real network. Shared
  helpers are OPTIONAL (default inline).

- **ESC-SPEECH-STREAM (CONDITIONAL — at impl, per-method).** §1.6. Image-gen and
  transcription streaming are real SSE → SSE-drain template is sound (209-image,
  210). SPEECH streaming may emit raw audio chunks, not SSE frames; if at impl the
  upstream shape forbids hermetic SSE framing, take Option B for **209 only**
  (register route, `SpeechStream` returns 501, 209 STAYS MISSING + escalated; 007
  non-stream speech still ships HAVE). NEVER fabricate an SSE frame around raw audio.
  Default = SSE-drain; fall to Option B for 209 only. Record the actual outcome in
  WORKFLOW.

- **ESC-OPENAI-SHAPE (RESOLVED at authoring — envelope decision, binding,
  inherited).** §1.1. `/v1/*` return OpenAI shapes (bare object or raw bytes +
  OpenAI `{"error":{}}`), NOT the `{data,error}` admin envelope. Verified across
  chat/embeddings/completions. The admin envelope is FORBIDDEN here. Bifrost's
  `BifrostError`/`event_id` (301/302/303) is a VARIANT escalation (BIFROST-MAP §224)
  — NOT this plan.

- **W7-PROV-MEDIA PARTIAL UNBLOCK (RESOLVED at authoring — cross-reference,
  informational).** §1.7. w7-prov-media DEFERRED 11 media providers BECAUSE no
  `/v1/audio` or `/v1/images` route existed (ESC-M4). bf-openai-2 ADDS those routes,
  resolving the ROUTE-existence blocker. The 11 deferred provider ADAPTERS remain a
  SEPARATE follow-up (still out of scope here — only the openai provider's methods
  ship). Record the partial-unblock in open-questions.md at closeout. The video gap
  (runwayml, ESC-M3) is NOT touched (no video method/route).

- **Serial-slot dependency (§1.8 / P5).** bf-openai-2 is SECOND in the
  routes_openai.go serial chain (after the SHIPPED bf-openai-1, which RELEASED the
  slot on close); it TAKES the slot and RELEASES it to bf-openai-3 on close.
  Orchestrator confirms exactly one unmerged holder (decision 3) before T-routes.

- **No other blocking dependency.** All reused surfaces (audio/images schemas,
  Router.Resolve, utils helpers, writeError/jsonMarshal/writeSSEStream/
  withRequestCancel/requestHeadersFromCtx, NewVKGate, the SHIPPED completions
  handler + provider templates, httptest hermetic pattern, stdlib `mime/multipart`)
  are in-tree at <base> (P2/P3/P4). bf-openai-2 is unblocked once the serial slot is
  free.
```
