# Micro-plan w7-prov-media — Media/embedding specialist providers (Go)

```
wave: 7
plan: w7-prov-media (the WAVE-7-MAP "scope-call" row ~line 179; flagged
  "MAY be deferred wholesale to a media escalation if Stage-1 chat-only ranking
  holds")
status: READY (rev 1 — authored against live tree @ 0533032; 9router frozen @ 827e5c3)
runs: CATALOG/PROVIDER track. Disjoint from governance/routing/mcp/platform.
  Does NOT touch internal/server/routes_admin.go (no admin route). HOLDS the
  internal/inference/factory.go MICRO-SERIAL slot ONLY for the single additive
  voyage-ai dispatch arm (coordinate with w7-prov-special-a/-b — sub-serialize;
  key-disjoint arm). Does NOT touch internal/inference/selection.go.
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w7-prov-media:
ref-source: 9router frozen @ 827e5c3 —
  open-sse/config/providers.js (PROVIDERS map: deepgram/assemblyai/nanobanana only),
  open-sse/config/providerModels.js (PROVIDER_MODELS map: all 12 media blocks),
  open-sse/handlers/embeddingProviders/{openai,index}.js (the embedding registry),
  open-sse/handlers/imageProviders/*.js + handlers/sttCore.js (image/STT executors —
  evidence for the DEFER decisions). Per-row ref citations in §6.
base: <base> = git rev-parse HEAD recorded at P0 (0533032 at authoring; if main
  advanced, record the actual SHA and substitute everywhere §5 says <base>).
go-serial-slot: NONE (no routes_admin.go).
factory-micro-serial: YES (minimal) — adds ONE additive dispatch arm for voyage-ai
  to buildProvider (factory.go:99-140). Additive only; the existing five built-in
  arms, the special-a/-b format switch, the urltemplate predicate, and the generic
  default are UNCHANGED. Confirm the slot is FREE / coordinate with concurrent
  factory.go holders at P0 (§2.7 grep).
freeze: everything outside the §3 ownership set is FROZEN. Explicitly FROZEN:
  internal/api/** (embeddings.go and all v1 handlers), internal/server/** (route
  registration), internal/schemas/** (Provider interface + media request/response
  structs), the generic/openai/urltemplate adapters, all UI.
```

---

## 0. The scope call — READ FIRST (binding)

The WAVE-7-MAP flags this plan as a possible **wholesale DEFER**. This plan makes
the call **per provider, grounded in two hard architectural facts** discovered in
the live tree (cited in §2), not in a blanket judgement:

**FACT 1 — only the embedding media-method is reachable today.** The gateway exposes
exactly ONE non-chat media route: `POST /v1/embeddings`
(`internal/server/routes_openai.go:98` → `internal/api/embeddings.go:53`), which
resolves a provider via the router and calls `provider.Embedding(...)`
(`embeddings.go:64,100`). There is **NO** `/v1/images/generations`,
**NO** `/v1/audio/transcriptions`, and **NO** `/v1/audio/speech` route registered
anywhere in `internal/server/` or `internal/api/` (the only occurrences of those
strings are inert endpoint LABELS in a model-listing map, `internal/api/models.go:406-409`
— they register no handler). `ImageGeneration`/`Speech`/`Transcription` exist on the
`schemas.Provider` interface (`internal/schemas/provider.go:86-94`) but are stubbed
to 501 in every adapter (`generic/stubs.go:36-66`, `openai/stubs.go:25-55`) and have
no caller.

**Consequence:** an image/STT/TTS adapter built now would be **dead, unreachable
code** — no HTTP path dispatches it, and adding one means editing
`internal/api/**` + `internal/server/routes_openai.go`, which are **out of this
plan's scope** (that is a transport-layer wave, not a provider-adapter micro-plan).
It would also be un-exercisable end-to-end. Per the orchestrator rule "honest about
what's a real adapter vs a catalog stub," those are DEFERRED, not faked.

**FACT 2 — the embedding wire format is OpenAI-compatible and already templated.**
9router routes voyage-ai through its OpenAI-compatible embedding adapter
(`handlers/embeddingProviders/openai.js:8` base `https://api.voyageai.com/v1/embeddings`,
`index.js:6-8` lists `voyage-ai` in `OPENAI_COMPAT_PROVIDERS`, bearer auth, plain
`{model,input,encoding_format?,dimensions?}` body, `normalize: identity`). g0router
already has the exact Go template for this: `internal/providers/openai/embedding.go:12`
(POST `<baseURL>/v1/embeddings`, bearer via `utils.SetAuthHeader`, decode
`schemas.EmbeddingResponse`). So a voyage-ai embedding adapter is **cheap, real, and
HERMETICALLY testable** against an `httptest` server through the live
`provider.Embedding` path.

**Net decision: BUILD voyage-ai (the one cheap, reachable, soundly-testable
embedding adapter). DEFER the other 11** (1 image-chat, 7 image, 1 video, 2 STT,
and huggingface which is image+STT — none has a reachable route; several are
async-polling/upload APIs that cannot be soundly built+tested now). This is NOT a
wholesale defer — it ships the single provider that has a real method to implement
and a real path to exercise, and records the rest as explicit, reasoned escalations
(§8) so a future media-transport wave can pick them up against a real route.

---

## 1. Scope — PAR rows

### Row this plan CLOSES (→ HAVE, real tested adapter)

| Row | Provider | Interface method | New work | Disposition |
|---|---|---|---|---|
| PAR-PROV-066 | **voyage-ai** | `Embedding` | catalog entry + NEW `internal/providers/voyageai/` embedding adapter (mirrors `openai/embedding.go`) + factory dispatch arm + missing aliases | embedding adapter — BUILD |

### Rows DEFERRED (catalog-stub or no-op; stay MISSING — recorded as escalations §8)

| Row | Provider | ref method | Reason (binding) | ESC |
|---|---|---|---|---|
| PAR-PROV-053 | deepgram | Transcription (STT) | no `/v1/audio/transcriptions` route; deepgram `/v1/listen` is a distinct non-OpenAI wire | ESC-M1 |
| PAR-PROV-054 | assemblyai | Transcription (STT) | no STT route; assemblyai is **upload-then-poll** (`sttCore.js:62` two-step `/v2/upload`→transcript poll) — async, not hermetically determinable | ESC-M1 |
| PAR-PROV-055 | nanobanana | ImageGeneration | no `/v1/images/generations` route; image method stubbed 501 | ESC-M2 |
| PAR-PROV-058 | fal-ai | ImageGeneration | no image route; **queue/poll** API (`queue.fal.run`, `imageProviders/falAi.js`) — async | ESC-M2 |
| PAR-PROV-059 | stability-ai | ImageGeneration | no image route; bespoke `v2beta/stable-image/generate` multipart wire | ESC-M2 |
| PAR-PROV-060 | black-forest-labs | ImageGeneration (+edit) | no image route; **poll** API (`api.bfl.ai/v1`, `imageProviders/blackForestLabs.js`) — async | ESC-M2 |
| PAR-PROV-061 | recraft | ImageGeneration | no image route; bespoke `external.api.recraft.ai/v1/images/generations` | ESC-M2 |
| PAR-PROV-062 | runwayml | ImageGeneration + **video** | no image route; video has NO interface method at all; `api.dev.runwayml.com` poll API | ESC-M3 |
| PAR-PROV-063 | sdwebui | ImageGeneration | no image route; self-hosted, runtime base URL, no fixed catalog endpoint | ESC-M2 |
| PAR-PROV-064 | comfyui | ImageGeneration | no image route; self-hosted graph API, runtime base URL | ESC-M2 |
| PAR-PROV-065 | huggingface | ImageGeneration + Transcription | no image/STT route; NOT in the embedding registry (`embeddingProviders/index.js:6-8`) — its catalog block is image+STT only (`providerModels.js:632-638`); routed via `imageProviders/huggingface.js` + sttCore | ESC-M2 |

### NOT in scope (explicit)

- **No image/STT/TTS adapter code.** Building one is dead code without a route
  (FACT 1). The Image/Speech/Transcription methods stay stubbed (§0/§8).
- **No new v1 routes / no `internal/api/**` edit / no `internal/server/**` edit.**
  Adding `/v1/images/generations` etc. is a transport wave, not this plan.
- **No `schemas.Provider` interface change, no media request/response struct change**
  (`internal/schemas/provider.go`, `images.go`, `audio.go`, `embedding.go` are
  FROZEN). voyage-ai reuses the existing `EmbeddingRequest`/`EmbeddingResponse`.
- **No generic/openai/urltemplate adapter rewrite.** The voyage adapter is a NEW
  package; it does not touch the existing stubs.
- **No video support.** `schemas.Provider` has no video method; runwayml's video
  models are unmappable (ESC-M3).
- **No UI, no e2e, no mock.** WAVE-7-MAP e2e impact = "none (no UI page —
  media-providers UI was W6-deferred PAR-UI-022..024)".
- **No `New()` signature change.** voyage adapter gets its own constructor
  (`voyageai.New(providerID)`); the existing constructors are untouched.
- **No secret exposure.** The API key arrives via `schemas.Key.Value`
  (`provider.go:33`) and is set on the request via `utils.SetAuthHeader`; it is
  never logged or echoed (assert in tests + §5 grep).

---

## 2. Architectural decisions grounding (evidence — cite file:line)

### 2.1 The media-method interface surface

`internal/schemas/provider.go:84-94` defines the non-chat media methods on the
`Provider` interface:
- `Embedding(ctx, key, *EmbeddingRequest) (*EmbeddingResponse, *ProviderError)` (:84)
- `ImageGeneration` / `ImageEdit` / `ImageVariation` (:86-89)
- `Speech` / `Transcription` (:91-94)

The request/response shapes: `EmbeddingRequest{Input,Model,EncodingFormat,Dimensions,User}`
+ `EmbeddingResponse{Object,Data[],Model,Usage}` (`internal/schemas/embedding.go:4-18`);
`ImageGenerationRequest`/`ImageGenerationResponse` (`images.go:4-26`);
`TranscriptionRequest{File[]byte,...}`/`SpeechRequest` (`audio.go:4-58`). These are
FROZEN — voyage reuses Embedding* verbatim.

### 2.2 Only `Embedding` is reachable — the dispatch chain (the BUILD gate)

`POST /v1/embeddings` is registered at `internal/server/routes_openai.go:98`
(`embeddings.Handle`). `EmbeddingsHandler.Handle` (`internal/api/embeddings.go:53`)
unmarshals an `EmbeddingRequest`, calls `h.router.Resolve(req.Model)`
(`embeddings.go:64`), and dispatches `provider.Embedding(gatewayCtx, key, &req)`
(`embeddings.go:100`). `Router.Resolve` (`internal/inference/router.go:64`) →
`providerForModel(model)` (`router.go:87`) → `buildProvider(providerID, registry)`
(`router.go:167`). **So an embedding adapter returned by `buildProvider` for a
voyage model is fully wired and exercisable end-to-end.**

There is **no** equivalent for image/STT/TTS. Grep proof (§5): no `/v1/images` /
`/v1/audio` route in `internal/server/` or `internal/api/` (the only matches are the
inert label strings at `internal/api/models.go:406-409`). Hence image/STT/TTS
adapters are unreachable → DEFER (§0 FACT 1).

### 2.3 The generic adapter STUBS Embedding — voyage needs a real adapter + dispatch

`internal/providers/generic/stubs.go:32-34` returns `notImplemented("embedding")`
(501) for the generic OpenAI-compatible provider. So a voyage-ai catalog entry alone
would route through `generic.New` (`factory.go:138`) and return 501 on embeddings.
voyage-ai therefore needs (a) a NEW adapter package that IMPLEMENTS `Embedding`, and
(b) a factory dispatch arm that constructs it before the generic default. This
mirrors how `urltemplate` providers are dispatched ahead of generic
(`factory.go:135-137`).

### 2.4 The embedding adapter template — `openai/embedding.go` (read 3 before writing)

`internal/providers/openai/embedding.go:12-67` is the exact shape to mirror:
- `req.SetRequestURI(p.baseURL + "/v1/embeddings")` (:18) — for voyage the full URL
  is `https://api.voyageai.com/v1/embeddings`, so the adapter sets the URI from the
  catalog `BaseURL` directly (the catalog BaseURL IS the full embeddings endpoint,
  matching the ref `embeddingProviders/openai.js:8`).
- `utils.SetAuthHeader(req, key.Value)` (:20) — bearer auth (matches ref `bearerAuth`).
- `utils.SetJSONBody(req, request)` (:22) — the `EmbeddingRequest` marshals to
  `{input,model,encoding_format?,dimensions?,user?}` snake_case (`embedding.go:4-10`),
  matching the ref `buildBody` (`embeddingProviders/openai.js:28-36`).
- error path via `p.errorConverter.Convert(...)` (`openai.NewErrorConverter()`),
  status check `!= 200` (:46), decode `schemas.EmbeddingResponse` (:56). The ref
  `normalize` is identity (`embeddingProviders/openai.js:38`), so no body
  transformation is needed — decode straight into `EmbeddingResponse`.

The voyage adapter copies this method into a new package, parameterized by the
catalog BaseURL + provider id. `openai.NewErrorConverter()` + `utils.ClientPool`
are reused (they are already imported by `generic`/`urltemplate`, `provider.go:8-9`).

### 2.5 The provider struct + constructor template — `urltemplate`/`generic`

`internal/providers/urltemplate/provider.go:37-65` and `generic/provider.go:13-37`
show the canonical small-adapter shape: a `Provider` struct with
`{id, config, client *utils.ClientPool, networkConfig, errorConverter}`, a `New`
catalog-bound constructor, `GetProvider()`, and `SetNetworkConfig` (pushes the proxy
override, `provider.go:65-68`). The voyage adapter mirrors this, plus its own
`Embedding` method and the remaining interface methods delegated to a `stubs.go`
(copy the `generic/stubs.go` 501 stubs for every non-embedding method so the package
satisfies `schemas.Provider`). Compile-time assertion `var _ schemas.Provider =
(*Provider)(nil)` (`generic/stubs.go:10`).

### 2.6 Catalog + model + alias evidence

- **Catalog (`internal/providers/catalog/catalog.go`):** `ProviderConfig{Name,BaseURL,
  Format,Headers,AuthHeader,NoAuth,Retry}` (`catalog.go:6-18`) — voyage needs only
  `{Name,BaseURL,Format:"openai"}`. No struct change. (The `deepgram`/`assemblyai`/
  `nanobanana` catalog entries from the ref `providers.js:317-327` are NOT added —
  their providers are DEFERRED; adding a catalog entry with no adapter + no route
  would be a misleading stub.)
- **Models (`internal/providers/catalog/models.go`):** `ModelEntry{ID,Name,
  UpstreamModelID,Type,Params}` (`models.go:8-14`); `Type:"embedding"` is already in
  use (e.g. `nebius` Qwen3-Embedding, `models.go:247`). voyage's 7 entries port
  verbatim with `Type:"embedding"` (ref `providerModels.js:524-532`).
- **Aliases (`internal/providers/catalog/aliases.go`):** a single
  `map[string]string` (`aliases.go:5`); `ProviderAliasCount()` is asserted == **138**
  (`aliases_test.go:6`). `voyage-ai`/`voyage` aliases are **ABSENT** (grep §2.7) and
  MUST be added; the count assertion is updated to the new total in the same commit.

### 2.7 Pre-write verification greps (run at P0/T1)

```bash
# voyage-ai is NOT yet in catalog/models/aliases (this plan adds it):
grep -c '"voyage-ai"' internal/providers/catalog/catalog.go      # expect 0
grep -c '"voyage-ai"' internal/providers/catalog/models.go       # expect 0
grep -nE '"voyage(-ai)?"' internal/providers/catalog/aliases.go  # expect EMPTY (add)
# the embedding route + dispatch exist (BUILD gate):
grep -n '/v1/embeddings' internal/server/routes_openai.go        # :98 present
grep -n 'provider.Embedding(' internal/api/embeddings.go         # :100 present
# the DEFER gate: no image/STT/TTS route registered (only inert labels):
grep -rn 'v1/images/generations\|v1/audio/transcriptions\|v1/audio/speech' \
  internal/server/ internal/api/ | grep -v models.go | grep -v _test
  # expect EMPTY (no real route)
# the embedding template + 501 stub:
grep -n 'p.baseURL + "/v1/embeddings"' internal/providers/openai/embedding.go  # :18
grep -n 'notImplemented("embedding")' internal/providers/generic/stubs.go      # :33
# factory micro-serial: who else is editing factory.go right now?
git log --oneline <base>..HEAD -- internal/inference/factory.go   # coordinate if non-empty
# confirm the ref embedding base url + voyage-ai compat membership:
grep -n 'voyage-ai' /home/cortexos/Developer/github.com/bloodf/_refs/9router/open-sse/handlers/embeddingProviders/openai.js   # :8 base url
grep -n 'voyage-ai' /home/cortexos/Developer/github.com/bloodf/_refs/9router/open-sse/handlers/embeddingProviders/index.js    # OPENAI_COMPAT list
```

---

## 3. Exclusive file ownership

**NEW — voyage embedding adapter (+ tests):**

| File | Contract |
|---|---|
| `internal/providers/voyageai/provider.go` | `Provider` struct {id, config, client `*utils.ClientPool`, networkConfig, errorConverter} + `New(providerID string) (*Provider, error)` (catalog-bound; rejects a non-`voyage-ai` id / non-openai format) + `GetProvider()` + `SetNetworkConfig()`. Mirrors `urltemplate/provider.go:37-65,82-92`. No `init()`; errors-as-values; no global state. |
| `internal/providers/voyageai/embedding.go` | `Embedding(ctx, key, *EmbeddingRequest) (*EmbeddingResponse, *ProviderError)` — copy of `openai/embedding.go:12-67`, URI from `p.config.BaseURL` (the full `…/v1/embeddings`), bearer auth via `utils.SetAuthHeader`, `errorConverter.Convert` on non-200, decode `EmbeddingResponse`. |
| `internal/providers/voyageai/stubs.go` | the remaining `schemas.Provider` methods (Chat, Image*, Speech*, Transcription*, File*, Batch*, Responses*, TextCompletion*, ListModels, CountTokens) → `notImplemented(...)` 501. Copy `generic/stubs.go:12-114` (minus Embedding). Compile-time assert `var _ schemas.Provider = (*Provider)(nil)`. |
| `internal/providers/voyageai/embedding_test.go` | HERMETIC: `httptest` server returns a canned OpenAI-shaped embeddings response; assert the adapter POSTs to the server URL (test seam: a `urlOverride`/`baseURL` field set by the test, mirroring `urltemplate/provider.go:43-46`), sends bearer auth + the `{model,input}` body, decodes `EmbeddingResponse` (Data/Model/Usage); non-200 → `ProviderError`; **assert the key value never appears in any logged/echoed output**. RED first. |
| `internal/providers/voyageai/provider_test.go` | `New("voyage-ai")` succeeds; `New("bogus")` / a non-openai id errors; `GetProvider()=="voyage-ai"`. RED first. |

**MODIFY — factory dispatch (factory.go micro-serial; ONE additive arm):**

| File | Change |
|---|---|
| `internal/inference/factory.go` | ADD a dispatch for voyage-ai before the generic default (e.g. `if providerID == "voyage-ai" { return voyageai.New(providerID) }`, placed alongside the `urltemplate.IsURLTemplateProvider` check at :135-137). Import `…/providers/voyageai`. Existing arms + generic default UNCHANGED. |
| `internal/inference/factory_test.go` | ADD a test asserting `buildProvider("voyage-ai")` returns a `*voyageai.Provider` (no error) and that the 5 built-ins + generic default still behave (regression). RED first. |

**MODIFY — catalog data (ADDITIVE map entries; no struct change):**

| File | Change |
|---|---|
| `internal/providers/catalog/catalog.go` | ADD ONE `ProviderConfig`: `"voyage-ai": {Name:"voyage-ai", BaseURL:"https://api.voyageai.com/v1/embeddings", Format:"openai"}`. NO other media providers added. |
| `internal/providers/catalog/models.go` | ADD the `"voyage-ai"` model block: 7 entries, all `Type:"embedding"`, verbatim from `providerModels.js:524-532` (§6). |
| `internal/providers/catalog/aliases.go` | ADD missing aliases `"voyage-ai"→"voyage-ai"` and `"voyage"→"voyage-ai"` (verify exact gap at T1; sdwebui/comfyui aliases are NOT added — those providers are deferred). |

**MODIFY — tests (TDD; written RED before the data, §4):**

| File | Change |
|---|---|
| `internal/providers/catalog/catalog_test.go` | ADD a test asserting `Lookup("voyage-ai")` returns BaseURL `…/v1/embeddings` + `Format=="openai"`. |
| `internal/providers/catalog/models_test.go` | ADD a test asserting `ModelsFor("voyage-ai")` count==7 and every entry has `Type=="embedding"`; key IDs present (voyage-3-large, voyage-3.5, voyage-code-3). |
| `internal/providers/catalog/aliases_test.go` | UPDATE the `ProviderAliasCount()` expected total (138 → new total after adding the voyage aliases) in the SAME commit; ADD sample assertions for `voyage-ai`/`voyage`. |

**MODIFY — matrix + workflow (closeout):**

| File | Change |
|---|---|
| `.planning/parity/matrix/9router-providers.md` | Flip PAR-PROV-066 (voyage-ai) → HAVE (cite the adapter + dispatch). Annotate PAR-PROV-053/054/055/058/059/060/061/062/063/064/065 as DEFERRED with the §8 escalation tag (stay MISSING). |
| `docs/WORKFLOW.md` | Record P0 base SHA, the BUILD-vs-DEFER call + facts, the alias-count delta, the §8 escalations, closeout. |
| `.planning/parity/plans/open-questions.md` | Append §8 escalations (ESC-M1/M2/M3 + the media-transport-route open question). |

**FORBIDDEN (automatic REJECT):** `internal/api/**` (embeddings.go and all v1
handlers), `internal/server/**` (route registration), `internal/schemas/**`
(interface + media structs), `internal/providers/generic/**`,
`internal/providers/openai/**`, `internal/providers/urltemplate/**`,
`internal/inference/selection.go`, `internal/inference/router.go`,
`internal/server/routes_admin.go`, any `internal/admin/**`, any `ui/**`, any mock
file. NO image/STT/TTS adapter packages. NO `ProviderConfig`/`ModelEntry`/`Provider`
struct change. NO new v1 route.

---

## 4. TDD tasks

Cadence (strict, AGENTS.md "TDD always: test first, see it fail, minimum code to
pass; no mocks — use `net/http/httptest` fakes"): **no adapter, catalog entry, alias,
or dispatch arm is added before the failing test that asserts it is committed RED.**
All adapter tests are HERMETIC — a fake upstream returning a canned golden
embeddings response; NO real provider calls. `go test ./... && go vet ./... &&
go build ./...` green at EVERY commit.

### T0 — verify facts (no code)
Run the §2.7 greps; record in WORKFLOW.md: (a) `/v1/embeddings` route + dispatch
present (BUILD gate met); (b) no `/v1/images|audio` route (DEFER gate met);
(c) voyage-ai absent from catalog/models/aliases; (d) factory micro-serial slot
state. Record P0 `<base>` SHA.

### T1 — catalog + model + alias for voyage-ai — RED → GREEN
STEP(a) RED: add `catalog_test.go` `TestVoyageAIProvider` (Lookup BaseURL/Format),
`models_test.go` `TestVoyageAIModels` (count==7, all `Type:"embedding"`),
`aliases_test.go` voyage assertions + the updated count. Run
`go test ./internal/providers/catalog/ -run 'Voyage|Alias'` → **FAILS**. Commit RED:
`phase-1/w7-prov-media: failing voyage-ai catalog/model/alias tests (TDD red)`.
STEP(b) GREEN: add the catalog entry + 7-model block + the two aliases + count
update. Gates: `go test ./... && go vet ./... && go build ./...` green. Commit:
`phase-1/w7-prov-media: voyage-ai catalog + embedding model block + aliases`.

### T2 — voyage embedding adapter + factory dispatch — RED → GREEN
STEP(a) RED: write `internal/providers/voyageai/embedding_test.go` +
`provider_test.go` (httptest round-trip + constructor) and
`internal/inference/factory_test.go` `TestVoyageAIDispatch`
(`buildProvider("voyage-ai")` → `*voyageai.Provider`). The package must compile-fail
or assert-fail (impl absent). Run `go test ./internal/providers/voyageai/...
./internal/inference/ -run 'Voyage|Dispatch'` → **FAILS**. Commit RED:
`phase-1/w7-prov-media: failing voyage-ai embedding adapter + dispatch tests (TDD red)`.
STEP(b) GREEN: implement `voyageai/provider.go` + `embedding.go` + `stubs.go`; add
the additive factory arm + import. Gates green. Commit:
`phase-1/w7-prov-media: voyage-ai embedding adapter + factory dispatch`.

### T3 — full gates + closeout
```bash
go test ./internal/providers/voyageai/... -run Voyage -v
go test ./internal/providers/catalog/... -run 'Voyage|Alias' -v
go test ./internal/inference/ -run 'Voyage|Dispatch' -v
go test ./... && go vet ./... && go build ./...
```
Flip `.planning/parity/matrix/9router-providers.md`: PAR-PROV-066 → HAVE; annotate
053/054/055/058/059/060/061/062/063/064/065 DEFERRED (ESC-M1/M2/M3). Append §8 to
`.planning/parity/plans/open-questions.md`. Update `docs/WORKFLOW.md`. Final commit:
`phase-1/w7-prov-media: close — voyage-ai embedding HAVE; media image/stt/tts deferred; matrix flips`.

---

## 5. Binary acceptance criteria

All yes/no. `<base>` = SHA recorded at P0 (0533032 at authoring). Diff gate is
commit-range-scoped (§7). HERMETIC — no acceptance command performs a real provider call.

**Test gates**
- `go test ./internal/providers/voyageai/... -run Voyage -v` → exit 0.
- `go test ./internal/providers/catalog/... -run 'Voyage|Alias' -v` → exit 0.
- `go test ./internal/inference/ -run 'Voyage|Dispatch' -v` → exit 0.
- `go test ./... && go vet ./... && go build ./...` → exit 0.

**TDD-order proof** — each impl commit follows its RED test commit:
```bash
R="<first-w7-prov-media>^..<last-w7-prov-media>"
rc=$(git log --format=%ct -1 --grep="failing voyage-ai embedding adapter")
dc=$(git log --format=%ct -1 --grep="voyage-ai embedding adapter + factory dispatch")
[ "$rc" -le "$dc" ] || echo "TDD VIOLATION: adapter"   # prints nothing
rc2=$(git log --format=%ct -1 --grep="failing voyage-ai catalog")
dc2=$(git log --format=%ct -1 --grep="voyage-ai catalog + embedding model")
[ "$rc2" -le "$dc2" ] || echo "TDD VIOLATION: catalog"  # prints nothing
```

**Grep proofs (voyage-ai built, real adapter not stub)**
```bash
C=internal/providers/catalog/catalog.go
M=internal/providers/catalog/models.go
F=internal/inference/factory.go
V=internal/providers/voyageai
grep -q 'api.voyageai.com/v1/embeddings' $C                         # catalog base url
grep -q '"voyage-ai"' $M                                            # model block present
grep -c 'Type: *"embedding"' $M | grep -q '[0-9]'                   # embedding-typed entries
grep -q 'func (p \*Provider) Embedding' $V/embedding.go             # REAL embedding impl
grep -q 'utils.SetAuthHeader' $V/embedding.go                       # bearer auth reused
grep -q 'notImplemented("embedding")' $V/embedding.go && echo "FAKE STUB — REJECT" || echo "real embedding OK"
grep -qE 'voyageai\.New|"voyage-ai"' $F                             # factory dispatch arm
grep -q 'var _ schemas.Provider = (\*Provider)(nil)' $V/stubs.go    # interface satisfied
# aliases:
grep -qE '"voyage(-ai)?":' internal/providers/catalog/aliases.go    # voyage aliases added
# no init(); no global state in the new package:
! grep -rn 'func init(' $V/ && echo "no init() OK"
# secret-safety: the API key is never written to a literal/log:
! grep -rnE 'key\.Value' $V/ | grep -iE 'log\.|fmt\.Print|Errorf.*key\.Value' && echo "no key leak OK"
```

**DEFER honesty proofs (image/STT/TTS NOT faked)**
```bash
# no image/STT/TTS adapter package was created:
test ! -d internal/providers/deepgram && test ! -d internal/providers/falai \
  && test ! -d internal/providers/stabilityai && test ! -d internal/providers/nanobanana \
  && echo "no deferred-provider adapters created OK"
# deferred providers were NOT added to the catalog as misleading stubs:
! grep -qE '"deepgram"|"assemblyai"|"fal-ai"|"stability-ai"|"black-forest-labs"|"nanobanana"|"runwayml"|"sdwebui"|"comfyui"|"recraft"' \
  internal/providers/catalog/catalog.go && echo "no deferred catalog stubs OK"
# huggingface NOT added as an embedding provider (it is image+STT — ESC-M2):
! grep -q '"huggingface"' internal/providers/catalog/catalog.go && echo "huggingface not stubbed OK"
```

**No-out-of-scope / freeze proofs (commit-range — §7)**
```bash
git diff $R --name-only | grep -vE \
  'internal/providers/voyageai/.*\.go|internal/inference/factory(_test)?\.go|internal/providers/catalog/(catalog|models|aliases)(_test)?\.go|\.planning/parity/(matrix/9router-providers|plans/open-questions)\.md|docs/WORKFLOW\.md' \
  | wc -l   # = 0
# transport + interface + sibling adapters untouched:
git diff $R --name-only -- internal/api/ internal/server/ internal/schemas/ \
  internal/providers/generic/ internal/providers/openai/ internal/providers/urltemplate/ \
  internal/inference/router.go internal/inference/selection.go | wc -l   # = 0
# factory edit is ADDITIVE (no removed/renamed existing arms):
git diff $R -- internal/inference/factory.go | grep -E '^-' | grep -v '^---' \
  | grep -qE 'case "openai"|case "anthropic"|generic.New|IsURLTemplateProvider' \
  && echo "EXISTING ARM CHANGED — REJECT" || echo "additive arm only OK"
# no struct change:
git diff $R -- internal/providers/catalog/catalog.go internal/providers/catalog/models.go \
  | grep -E '^\+' | grep -qE 'type ProviderConfig|type ModelEntry|func ' \
  && echo "STRUCT/FUNC CHANGE — REJECT" || echo "additive entries only OK"
```

---

## 6. Per-provider data table (name → method → base_url → models → ref source)

All transcribed from 9router @ 827e5c3. **The implementer MUST re-read the ref at
T-impl and transcribe IDs verbatim — never fabricate.**

### BUILT — voyage-ai (embedding)

| Provider | method | base_url | format/auth | models (count; all `Type:"embedding"`) | ref source |
|---|---|---|---|---|---|
| voyage-ai | `Embedding` | `https://api.voyageai.com/v1/embeddings` | openai / bearer | 7 — voyage-3-large, voyage-3.5, voyage-3.5-lite, voyage-code-3, voyage-finance-2, voyage-law-2, voyage-multilingual-2 | `providerModels.js:524-532`; `handlers/embeddingProviders/openai.js:8` (base url) + `index.js:6-8` (OPENAI_COMPAT membership) |

Aliases to ADD: `voyage-ai`→`voyage-ai`, `voyage`→`voyage-ai` (verify gap §2.7).

### DEFERRED — image / video / STT (recorded for a future media-transport wave)

| Provider | ref method | ref base_url / shape | models (ref) | why deferred | ref source |
|---|---|---|---|---|---|
| deepgram | Transcription | `https://api.deepgram.com/v1/listen` (distinct STT wire) | nova-3, nova-2, whisper-large | no STT route; non-OpenAI wire | `providers.js:317-319`; `providerModels.js:785-788` |
| assemblyai | Transcription | `https://api.assemblyai.com/v1/audio/transcriptions` (+ `/v2/upload` poll) | universal-3-pro, universal-2 | no STT route; **upload-then-poll** async (`sttCore.js:62`) | `providers.js:321-323`; `providerModels.js:790-793` |
| nanobanana | ImageGeneration | `https://api.nanobananaapi.ai/v1/chat/completions` | nanobanana-flash, nanobanana-pro | no image route; image method stubbed 501 | `providers.js:325-327`; `providerModels.js:620-623` |
| fal-ai | ImageGeneration | `https://queue.fal.run` (**queue/poll**) | fal-ai/flux/schnell, …/flux/dev, …/flux-pro/v1.1(-ultra), …/recraft-v3, …/ideogram/v2, …/stable-diffusion-v35-large | no image route; async queue | `providerModels.js:794-802`; `imageProviders/falAi.js:4` |
| stability-ai | ImageGeneration | `https://api.stability.ai/v2beta/stable-image/generate` (multipart) | stable-image-ultra/-core, sd3.5-large/-large-turbo/-medium | no image route; bespoke multipart wire | `providerModels.js:803-809`; `imageProviders/stabilityAi.js:4` |
| black-forest-labs | ImageGeneration (+edit) | `https://api.bfl.ai/v1` (**poll**) | flux-pro-1.1(-ultra), flux-pro, flux-dev, flux-kontext-pro/max (edit) | no image route; async poll | `providerModels.js:810-817`; `imageProviders/blackForestLabs.js:4` |
| recraft | ImageGeneration | `https://external.api.recraft.ai/v1/images/generations` | recraftv3, recraftv2 | no image route | `providerModels.js:818-821`; `imageProviders/openai.js:7` |
| runwayml | ImageGeneration + **video** | `https://api.dev.runwayml.com/v1` (poll) | gen4_image(-turbo) (image), gen4_turbo / gen3a_turbo (**video**) | no image route; **video has no interface method** | `providerModels.js:822-827`; `imageProviders/runwayml.js:4` |
| sdwebui | ImageGeneration | self-hosted (runtime base url) | stable-diffusion-v1-5, sdxl-base-1.0 | no image route; no fixed catalog endpoint | `providerModels.js:624-627` |
| comfyui | ImageGeneration | self-hosted graph API (runtime base url) | flux-dev, sdxl | no image route; graph-workflow API | `providerModels.js:628-631` |
| huggingface | ImageGeneration + Transcription | `https://api-inference.huggingface.co/models` | FLUX.1-schnell, SDXL-base (image); whisper-large-v3, whisper-small (stt) | no image/STT route; NOT in embedding registry | `providerModels.js:632-638`; `imageProviders/huggingface.js:4` |

---

## 7. Diff-gate scope

Isolate this plan's commits:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w7-prov-media:" | awk '{print $1}'`
then `git diff <first>^..<last> --name-only` must be a subset of:
```
internal/providers/voyageai/provider.go
internal/providers/voyageai/provider_test.go
internal/providers/voyageai/embedding.go
internal/providers/voyageai/embedding_test.go
internal/providers/voyageai/stubs.go
internal/inference/factory.go              (ONE additive arm + import)
internal/inference/factory_test.go
internal/providers/catalog/catalog.go      (ONE additive entry)
internal/providers/catalog/catalog_test.go
internal/providers/catalog/models.go       (ONE additive block)
internal/providers/catalog/models_test.go
internal/providers/catalog/aliases.go      (two additive aliases + count)
internal/providers/catalog/aliases_test.go
.planning/parity/matrix/9router-providers.md
.planning/parity/plans/open-questions.md
docs/WORKFLOW.md
```
Any file outside is an automatic REJECT. `internal/api/**`, `internal/server/**`,
`internal/schemas/**`, the generic/openai/urltemplate adapters,
`internal/inference/router.go`/`selection.go`, and any image/STT/TTS adapter package
are deliberately ABSENT — touching them is an automatic REJECT. If a concurrent
factory.go holder (w7-prov-special-a/-b) is unmerged, sub-serialize the factory edit
(key-disjoint arm).

---

## 8. Escalations / open questions

- **ESC-M1 (STT — deepgram PAR-PROV-053, assemblyai PAR-PROV-054 — DEFERRED,
  binding).** The `Transcription` interface method exists
  (`schemas/provider.go:93`) but NO `/v1/audio/transcriptions` route is registered
  (`internal/server/`/`internal/api/` have none; `models.go:407` is an inert label).
  An STT adapter would be unreachable dead code. Additionally assemblyai is a
  two-step **upload-then-poll** flow (`sttCore.js:62`) that is not hermetically
  determinable as a single request/response, and deepgram's `/v1/listen` is a
  distinct non-OpenAI wire. **Decision:** DEFER both; stay MISSING. Pick up when a
  `/v1/audio/transcriptions` transport route exists.
- **ESC-M2 (image — nanobanana 055, fal-ai 058, stability-ai 059,
  black-forest-labs 060, recraft 061, sdwebui 063, comfyui 064, huggingface 065 —
  DEFERRED, binding).** No `/v1/images/generations` route exists; `ImageGeneration`
  is stubbed 501 in every adapter. Several are async (fal-ai queue/poll, bfl poll),
  bespoke-wire (stability multipart), or self-hosted-runtime-URL (sdwebui, comfyui).
  huggingface is image+STT (NOT in the embedding registry `embeddingProviders/index.js:6-8`),
  so it does not qualify as the cheap embedding case. **Decision:** DEFER all;
  do NOT add catalog stubs (a base_url with no adapter and no route is misleading).
- **ESC-M3 (video — runwayml 062 — DEFERRED, binding).** runwayml's `gen4_turbo`/
  `gen3a_turbo` are `type:"video"` (`providerModels.js:824-825`). The
  `schemas.Provider` interface has **no video method** — video is unmappable to the
  current interface without a schema change (which is out of scope: `schemas/**` is
  frozen). Its image models share the ESC-M2 no-route blocker. **Decision:** DEFER.
- **ESC-M4 (the media-transport open question — for the orchestrator).** The real
  blocker for image/STT/TTS parity is the **absence of v1 media routes**
  (`/v1/images/generations`, `/v1/audio/transcriptions`, `/v1/audio/speech`) and any
  router media-dispatch path — a TRANSPORT/API-layer concern, not a provider-adapter
  one. **Open question:** should a future wave add the media transport routes
  (`internal/api/images.go`, `internal/api/audio.go` + `routes_openai.go`
  registration) so the 11 deferred adapters become buildable+reachable? Until then,
  those PAR rows are correctly MISSING. This plan ships the one provider
  (voyage-ai) whose method (`Embedding`) is already routed.
- **ESC-M5 (the wholesale-defer alternative — NOT taken, recorded).** The
  WAVE-7-MAP authorized deferring this plan entirely "if Stage-1 chat-only ranking
  holds." This plan deliberately does NOT wholesale-defer: voyage-ai is a real,
  reachable, hermetically-testable embedding adapter at near-zero cost (one ~55-line
  method copied from `openai/embedding.go` + a catalog/model/alias/dispatch arm), so
  shipping it closes one parity row honestly. If the operator prefers zero media
  scope this wave, the fallback is to drop T1/T2 and record PAR-PROV-066 alongside
  the others under ESC-M4. **Recommended: BUILD voyage-ai as planned.**

All ESC items appended to `.planning/parity/plans/open-questions.md` at T3
(Planner Open_Questions protocol).
```
