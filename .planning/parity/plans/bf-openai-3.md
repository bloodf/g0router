# Micro-plan bf-openai-3 — Files (`/v1/files` CRUD: upload/list/get/delete/content) + Batches (`/v1/batches` CRUD: create/list/get/cancel) (Go)

```
program: bifrost-parity (bifrost phase — BUILDABLE-ADDITIVE only; the ~50%
  re-architecture is permanently deferred per BIFROST-MAP §1/§8 ESC set)
plan: bf-openai-3
status: READY (rev 1 — authored against the live tree @ <base>; BIFROST-MAP
  micro-plan index row ~line 298; bifrost-openai disposition rows
  020/021/022/023/024/025/026/027/028/029 = .planning/parity/matrix/bifrost-openai.md:31-40;
  serial chain BIFROST-MAP:323-351)
runs: OpenAI-surface track. HOLDS the internal/server/routes_openai.go SERIAL
  SLOT while live (decision 3). Serial chain:
  bf-openai-1 (SHIPPED) → bf-openai-2 (SHIPPED) → **bf-openai-3** → bf-openai-4
  (each appends /v1/*). Disjoint from the governance / mcp / core tracks (run ∥).
branch: main (direct commits, per AGENTS.md "No PR workflow")
commit-prefix: phase-1/bf-openai-3:
ref-source: ESC-REF-ABSENT (BIFROST-MAP §47-68) — the frozen Bifrost ref
  (@ca21298) is NOT on this host. The matrix rows + g0router's own conventions
  are the ONLY ground truth. /v1/files and /v1/batches are documented, stable
  OpenAI public endpoints, so their wire shapes are g0router's own schemas
  (internal/schemas/files.go, internal/schemas/batch.go) + OpenAI's public spec
  — NOT a guessed Bifrost internal. No Bifrost handler internals are reconstructed.
base: <base> = git rev-parse HEAD recorded at P0. Substitute the actual SHA
  everywhere §5 says <base>.
go-serial-slot: this plan holds the SINGLE in-flight edit to
  internal/server/routes_openai.go while live (decision 3). bf-openai-2 RELEASED
  the slot on its close; bf-openai-3 TAKES it. Slot must be FREE at P5 before
  T-routes. RELEASE to bf-openai-4 on close.
new-route: API routes only. NO UI contract — /v1/* are OpenAI-compatible API
  routes (NOT the {data,error} admin envelope). No e2e, no UI, no mocks.
pattern: MIRRORS the SHIPPED bf-openai-1/bf-openai-2
  (`.planning/parity/plans/bf-openai-1.md`, `bf-openai-2.md`): /v1-route +
  OpenAI-shape (not admin envelope) + provider-method-impl-over-stub (Option A).
  bf-openai-2 implemented Speech/Transcription/ImageGeneration (+streams) over the
  stubs; bf-openai-3 implements FileUpload/FileList/FileRetrieve/FileDelete/
  FileContent + BatchCreate/BatchList/BatchRetrieve/BatchCancel the same way.
statefulness-decision: **Option A — STATELESS PASSTHROUGH to upstream OpenAI's
  files/batches API.** Resolved with evidence at authoring (§1.2). g0router does
  NOT host a local file/batch store; OpenAI is the system of record. No new
  SQLite tables. (Full Option-A-vs-B analysis + evidence in §1.2.)
```

---

## 1. Scope — PAR rows + the deliverables

### Rows this plan closes

| Row | Claim (matrix text) | Current state (evidence) | Target after bf-openai-3 |
|---|---|---|---|
| **PAR-BF-OAI-025** | `POST /v1/files` (upload) registered with fasthttp (`bifrost-openai.md:36`) | MISSING — "Schema exists (`internal/schemas/files.go:15-20`) but no route; provider stubbed (`stubs.go:57-75`)". Confirmed: no `/v1/files` route anywhere; `FileUpload` → `notImplemented("file_upload")` (`internal/providers/openai/stubs.go:17-19`). | HAVE — route registered (`routes_openai.go`), handler parses **multipart/form-data** (file + purpose, §1.5), dispatches to `provider.FileUpload`, returns the bare OpenAI `FileObject` JSON. |
| **PAR-BF-OAI-026** | `GET /v1/files` (list) registered with fasthttp (`bifrost-openai.md:37`) | MISSING — "Schema exists (`internal/schemas/files.go:22-26`) but no route; provider stubbed". `FileList` → `notImplemented("file_list")` (`stubs.go:21-23`). | HAVE — GET route + handler → `provider.FileList`, returns bare `FileListResponse`. |
| **PAR-BF-OAI-027** | `GET /v1/files/{file_id}` (retrieve) registered (`bifrost-openai.md:38`) | MISSING — "Schema exists (`internal/schemas/files.go:4-13`) but no route; provider stubbed". `FileRetrieve` → `notImplemented("file_retrieve")` (`stubs.go:25-27`). | HAVE — GET route (`{file_id}` path param) + handler → `provider.FileRetrieve`, returns bare `FileObject`. |
| **PAR-BF-OAI-028** | `DELETE /v1/files/{file_id}` registered (`bifrost-openai.md:39`) | MISSING — "No route; provider stubbed". `FileDelete` → `notImplemented("file_delete")` (`stubs.go:29-31`). | HAVE — DELETE route + handler → `provider.FileDelete`, returns bare `FileDeleteResponse`. |
| **PAR-BF-OAI-029** | `GET /v1/files/{file_id}/content` registered (`bifrost-openai.md:40`) | MISSING — "No route; provider stubbed". `FileContent` → `notImplemented("file_content")` (`stubs.go:33-35`). | HAVE — GET route + handler → `provider.FileContent`, returns RAW file bytes with the upstream `Content-Type` (NOT a JSON envelope, §1.4). |
| **PAR-BF-OAI-020** | `POST /v1/batches` registered with fasthttp (`bifrost-openai.md:31`) | MISSING — "Schema exists (`internal/schemas/batch.go:45-51`) but no route; provider stubbed (`stubs.go:77-91`)". `BatchCreate` → `notImplemented("batch_create")` (`stubs.go:37-39`). | HAVE — JSON route + handler → `provider.BatchCreate`, returns bare `Batch`. |
| **PAR-BF-OAI-021** | `GET /v1/batches` registered with fasthttp (`bifrost-openai.md:32`) | MISSING — "Schema exists (`internal/schemas/batch.go:53-57`) but no route; provider stubbed". `BatchList` → `notImplemented("batch_list")` (`stubs.go:41-43`). | HAVE — GET route + handler → `provider.BatchList`, returns bare `BatchListResponse`. |
| **PAR-BF-OAI-022** | `GET /v1/batches/{batch_id}` registered (`bifrost-openai.md:33`) | MISSING — "Schema exists (`internal/schemas/batch.go:4-22`) but no route; provider stubbed". `BatchRetrieve` → `notImplemented("batch_retrieve")` (`stubs.go:45-47`). | HAVE — GET route (`{batch_id}` path param) + handler → `provider.BatchRetrieve`, returns bare `Batch`. |
| **PAR-BF-OAI-023** | `POST /v1/batches/{batch_id}/cancel` registered (`bifrost-openai.md:34`) | MISSING — "No route; provider stubbed". `BatchCancel` → `notImplemented("batch_cancel")` (`stubs.go:49-51`). | HAVE — POST route (`{batch_id}` path param) + handler → `provider.BatchCancel`, returns bare `Batch`. |

### Row this plan ESCALATES (does NOT flip — see §8 ESC-BATCH-RESULTS)

| Row | Claim (matrix text) | Why it can NOT be built additively here |
|---|---|---|
| **PAR-BF-OAI-024** | `GET /v1/batches/{batch_id}/results` registered (`bifrost-openai.md:35`) | NO provider method on the interface and NO schema. `internal/schemas/provider.go:102-105` declares EXACTLY four batch methods — `BatchCreate`/`BatchList`/`BatchRetrieve`/`BatchCancel` (grep-confirmed: there is no `BatchResults`/`batch_results` symbol anywhere — `internal/schemas/{provider,batch}.go` + `internal/providers/openai/stubs.go`). The MAP index row says "020..029", but the buildable-additive surface (interface methods + schemas) covers only 020/021/022/023 + the FILES rows. **Canonically, OpenAI has no `GET /v1/batches/{id}/results` endpoint** — a completed batch exposes its results via its `output_file_id`, fetched with `GET /v1/files/{output_file_id}/content` (= **PAR-BF-OAI-029**, which DOES ship in this plan). So the *capability* the row describes is reachable through 029; the *literal route* requires a new interface method + schema, which is a non-additive interface change (touches all 43 providers' Provider implementations) and is therefore OUT of the buildable-additive bifrost phase. Escalated honestly in §8; STAYS MISSING in the matrix with the cross-reference to 029. |

> NOTE on the MAP "020..029" span: the BIFROST-MAP index row (line 298) lists the
> rows as "PAR-BF-OAI-020..029", a contiguous shorthand. The MATRIX
> (`bifrost-openai.md:31-40`) enumerates all ten. Of those ten, NINE have a
> stubbed provider method + schema (020/021/022/023 batches; 025/026/027/028/029
> files) and are buildable-additive HERE. ONE (024) has neither and is escalated
> (above). This is a matrix-shorthand-vs-interface reconciliation, recorded
> honestly per §6; never fabricate a 024 method/schema to "hit the span".

> NOTE on stub line numbers: the matrix cites `stubs.go:57-91` (files) and
> `stubs.go:77-91` (batches); the LIVE `internal/providers/openai/stubs.go` (read
> @ authoring) places the methods at: `FileUpload` :17-19, `FileList` :21-23,
> `FileRetrieve` :25-27, `FileDelete` :29-31, `FileContent` :33-35, `BatchCreate`
> :37-39, `BatchList` :41-43, `BatchRetrieve` :45-47, `BatchCancel` :49-51. The
> matrix cites are stale on the exact spans (they describe the original Bifrost
> handler line numbers in `transports/bifrost-http/handlers/inference.go`) but
> correct on the substance (all nine are 501 stubs). Use the LIVE spans; the
> executor must re-grep at P1.

Matrix flips at closeout (§4 T-close), in `.planning/parity/matrix/bifrost-openai.md`:
- PAR-BF-OAI-025 → HAVE (cite the files upload route + multipart parse + impl).
- PAR-BF-OAI-026 → HAVE (cite the files list route + handler + impl).
- PAR-BF-OAI-027 → HAVE (cite the files retrieve route + `{file_id}` param + impl).
- PAR-BF-OAI-028 → HAVE (cite the files delete route + impl).
- PAR-BF-OAI-029 → HAVE (cite the files content route + raw-bytes body + impl).
- PAR-BF-OAI-020 → HAVE (cite the batches create route + handler + impl).
- PAR-BF-OAI-021 → HAVE (cite the batches list route + handler + impl).
- PAR-BF-OAI-022 → HAVE (cite the batches retrieve route + `{batch_id}` param + impl).
- PAR-BF-OAI-023 → HAVE (cite the batches cancel route + `{batch_id}` param + impl).
- **PAR-BF-OAI-024 → STAYS MISSING + ESCALATED** (cross-reference: results reachable via 029; §8).

### 1.1 The OpenAI-shape vs admin-envelope decision (BINDING — inherited from bf-openai-1/2 §1.1)

**`/v1/*` routes return OpenAI shapes, NOT the `{data,error}` admin envelope.**
This is g0router's existing, verified convention (chat returns bare
`*ChatResponse`; completions bare `*TextCompletionResponse`; transcription bare
`*TranscriptionResponse` — `internal/api/audio.go:212-222`; speech raw bytes —
`internal/api/audio.go:110-116`).

Therefore the bf-openai-3 handlers:
- **FileContent** is the ONLY non-JSON success case: on success it writes the RAW
  file bytes (the `[]byte` returned by `provider.FileContent`) with the upstream
  `Content-Type` (default `application/octet-stream`), status 200. NO `jsonMarshal`,
  NO `{data}`, NO `{error}` wrapper. This is the EXACT analog of the SHIPPED Speech
  bytes-out path (`internal/api/audio.go:110-116`, ESC-SPEECH-BYTES). (`FileContent`
  returns `([]byte, *ProviderError)` — `internal/schemas/provider.go:100` —
  proving the body is raw bytes, not a JSON object.)
- **FileUpload / FileList / FileRetrieve / FileDelete / BatchCreate / BatchList /
  BatchRetrieve / BatchCancel** on success write the bare OpenAI object
  (`*FileObject`, `*FileListResponse`, `*FileDeleteResponse`, `*Batch`,
  `*BatchListResponse`) via `jsonMarshal` → 200 `application/json`, mirroring
  `internal/api/audio.go:212-222`.
- **All errors** call `writeError(ctx, status, errType, message, code)`
  (`internal/api/errors.go:18`) — the OpenAI `{"error":{...}}` shape — NOT the admin
  envelope. The api package does not import `internal/admin`.
- **No streaming.** None of the files/batches endpoints stream (no `*Stream`
  provider method exists for files/batches — `internal/schemas/provider.go:96-105`).
  There is NO `text/event-stream` / `writeSSEStream` path in this plan.

The `{data,error}` admin envelope is **FORBIDDEN** on these routes (§6).

### 1.2 Statefulness — Option A (passthrough) vs Option B (local store): RESOLVED → Option A (BINDING)

OpenAI files/batches are STATEFUL at the protocol level: a client uploads a file →
gets a `file_id` → references that `file_id` as `input_file_id` in a batch → polls
the batch by `batch_id` → fetches the result via the batch's `output_file_id`. The
question is **WHERE that state lives**: upstream at OpenAI (Option A, stateless
proxy) or in g0router (Option B, local SQLite store).

**DECISION: Option A — stateless passthrough to upstream OpenAI's files/batches
API.** Evidence (all grep/read-confirmed @ authoring):

1. **The provider-method signatures are passthrough-shaped, identical to the SHIPPED
   stateless media methods.** `internal/schemas/provider.go:96-105` declares:
   `FileUpload(ctx, key Key, *FileUploadRequest) (*FileObject, *ProviderError)`,
   `FileRetrieve(ctx, key Key, fileID string) (*FileObject, *ProviderError)`,
   `FileContent(ctx, key Key, fileID string) ([]byte, *ProviderError)`,
   `BatchCreate(ctx, key Key, *BatchCreateRequest) (*Batch, *ProviderError)`, etc.
   They take `key schemas.Key` (the UPSTREAM credential) and a bare id/request, and
   return the upstream object. This is the EXACT shape of the stateless
   `Speech`/`Transcription`/`ImageGeneration` methods bf-openai-2 just shipped
   (`internal/schemas/provider.go:90-94`). A LOCAL store would not need `key Key`
   at all (it would key off the g0router VK + a local table), and would not return
   the upstream `*FileObject`/`*Batch` directly. The signatures encode Option A.

2. **The stub impls live in the provider layer, not a store layer.** All nine are
   methods on `*openai.Provider` (`internal/providers/openai/stubs.go:17-51`),
   returning `notImplemented(...)` — the SAME 501 pattern the media stubs used
   before bf-openai-2 implemented them as upstream HTTP proxies
   (`internal/providers/openai/audio.go`, `images.go`). The architecture places
   files/batches in the PROVIDER (upstream-proxy) layer, NOT in `internal/store`.

3. **Every shipped openai-provider method is a thin upstream proxy.** `Speech`
   (`internal/providers/openai/audio.go:19-70`), `Transcription` (:106-151),
   `ImageGeneration` (`images.go:16-41`), `Embedding`, `TextCompletion` all do:
   `req.SetRequestURI(p.baseURL + "/v1/...")` + `utils.SetAuthHeader(req, key.Value)`
   + `p.client.Do` + status-check → `errorConverter.Convert` + decode. Files/batches
   proxy to `p.baseURL + "/v1/files..."` / `/v1/batches...` IDENTICALLY. The
   `FileObject.ID`/`Batch.ID` are OpenAI's ids, returned verbatim — g0router never
   mints or stores them.

4. **There is no files/batches store seam.** Grep-confirmed: no `files`/`batches`
   table, no `internal/store/files.go`/`batches.go`, no `FileStore`/`BatchStore`
   interface anywhere. Option B would require inventing all of that. The MAP marks
   these rows BUILDABLE-ADDITIVE precisely because the additive surface
   (interface + schemas + provider stubs) is already present for a PROXY — not a
   store.

**Why Option A is the sound, in-scope choice:** it is ADDITIVE (no schema change,
no new tables, no `New(...)` signature change), STATELESS (g0router holds no
file/batch state; OpenAI is the system of record; the client's own `file_id`/
`batch_id` round-trips through), HERMETICALLY TESTABLE (`httptest.NewServer` +
`p.baseURL = srv.URL`, the SHIPPED pattern at
`internal/providers/openai/audio_test.go:33-34`), and makes NO claim about Bifrost
internals (ESC-REF-ABSENT-safe — `/v1/files` & `/v1/batches` are documented stable
OpenAI public endpoints).

**Why Option B is REJECTED here:** a g0router-local file/batch store is a large,
non-additive feature (new SQLite tables + migrations + an async batch-execution
worker that fans a JSONL input file out to per-line completions, accrues
`request_counts`, writes an `output_file_id`, and transitions `status`
validating→in_progress→finalizing→completed). That is a STATEFUL SUBSYSTEM, not a
route-over-stub wiring, and there is NO parity-intent evidence for it (no store
seam, no worker, no table, no schema field that only a local store would need — the
schemas are pure OpenAI wire shapes). Building it would (a) exceed the
buildable-additive bifrost phase, (b) duplicate state OpenAI already owns, and (c)
contradict the passthrough signatures (#1). If a FUTURE wave genuinely wants a
g0router-managed batch store, it is a SEPARATE, scoped, deliberate plan — recorded
as an open question (§8 ESC-LOCAL-STORE), NOT smuggled in here.

**Binding consequence:** bf-openai-3 implements the nine methods as upstream
proxies for the **openai** provider ONLY (mirroring how bf-openai-2 implemented
Speech/Transcription/Image* for openai only). The other 42 providers' File*/Batch*
stubs are UNTOUCHED (they keep their 501). No store, no migrate, no worker, no
new table.

### 1.3 The provider-method implementation approach (BINDING — Option A, mirrors bf-openai-2 §1.2)

The MAP marks 020..023/025..029 BUILD because the schemas + interface methods
already exist (`internal/schemas/provider.go:96-105`); the gap is "wire the route
over the stubbed provider method". The nine openai-provider methods are real 501
stubs (`internal/providers/openai/stubs.go:17-51`). Two sound options, per the
bf-openai-1/2 precedent:

**Option A (RECOMMENDED — implement the methods for the openai provider).** Near-
verbatim copies of the SHIPPED `embedding.go` / `audio.go` / `images.go` transport:
`p.client.AcquireRequest/Response`, `req.SetRequestURI(p.baseURL + "/v1/...")`,
`req.Header.SetMethod(...)` (POST/GET/DELETE per row), `utils.SetAuthHeader`,
`utils.SetJSONBody` (JSON-body endpoints: batch create) OR a multipart body builder
(file upload — §1.5) OR no body (GET/DELETE/cancel), `p.client.Do`, status-check →
`p.errorConverter.Convert(...)`, then `utils.ReadJSONBody(resp, &result)` (JSON
responses) OR a `resp.Body()` copy + `Content-Type` capture (file content bytes —
§1.4).

This is **sound** because it is the identical transport the openai provider already
ships for chat/embeddings/completions/audio/images; it is hermetically testable
with `httptest.NewServer` + `p.baseURL = srv.URL`
(`internal/providers/openai/audio_test.go:20-34`); and it makes NO claim about
Bifrost internals.

**Binding consequence (test-table edit):** `TestNotImplementedStubs` in
`internal/providers/openai/openai_test.go` asserts the stubs 501. The executor MUST
re-grep that table at P1 (`go test ./internal/providers/openai/ -run 'NotImplemented' -v`)
and REMOVE exactly the File*/Batch* sub-cases that are now implemented (if they are
listed there), leaving the genuinely-remaining stubs (Responses, ResponsesStream,
CountTokens) UNTOUCHED. (bf-openai-2 already removed the media sub-cases; verify
which File*/Batch* sub-cases survive before deleting — do NOT delete a Responses/
CountTokens sub-case.) This is the ONLY edit to a pre-existing openai-provider test.

**Option B (FALLBACK — handler-only, surface the 501 cleanly; per-method, narrow).**
For any single method whose upstream transport cannot be made to pass hermetically
at impl, register the route + handler; the handler calls the provider method, which
returns the existing `not_implemented`/501; the handler maps that to
`writeError(ctx, 501, "not_implemented", ...)`. This closes the route-registration
row but leaves the method 501. Use ONLY per-method if Option A cannot pass
hermetically (it should — these are documented, simple JSON/bytes/multipart upstream
calls). If Option B is taken for a method, mark its row HAVE-route/escalate-impl
honestly in WORKFLOW.md (NEVER claim full parity on a 501).

**Default: Option A for all nine methods.**

### 1.4 File-content-bytes handling decision (BINDING — analog of SHIPPED ESC-SPEECH-BYTES)

`GET /v1/files/{file_id}/content` is the ONE route in this plan whose success body
is **NOT JSON**. OpenAI's `/v1/files/{id}/content` returns the raw file bytes (e.g.
the JSONL of a batch output or an uploaded training file). The decision:

1. The provider `FileContent(ctx, key, fileID) ([]byte, *ProviderError)` method
   copies the upstream `resp.Body()` into the returned `[]byte` (clone, do NOT
   alias the pooled response — `append([]byte(nil), resp.Body()...)`, exactly like
   Speech at `internal/providers/openai/audio.go:65`). It does NOT `ReadJSONBody`.
   Because the interface returns only `[]byte` (no Content-Type field — unlike
   `SpeechResponse` which carries `ContentType`), the handler sets a fixed
   `application/octet-stream` Content-Type on the response (see §1.6 for the
   Content-Type seam decision).
2. The handler writes those bytes verbatim with `Content-Type:
   application/octet-stream` — NO JSON marshal, NO `{data}`/`{error}` envelope.
3. Proof obligation (test): the file-content success body equals the upstream bytes
   exactly AND the body is NOT valid JSON `{...}` with a `data`/`error` key.

This is sound and ESC-REF-ABSENT-safe (OpenAI's documented binary/JSONL response;
no Bifrost internal). It is the direct analog of the SHIPPED Speech bytes-out path.

### 1.5 Multipart-file-upload handling decision (BINDING — reuses the SHIPPED ESC-MULTIPART pattern)

`POST /v1/files` takes **multipart/form-data** (file + `purpose`). bf-openai-2
introduced the first multipart handling in the tree
(`internal/api/audio.go:124-173` inbound parse, `internal/providers/openai/audio.go:176-241`
outbound build). bf-openai-3 REUSES that exact pattern (do NOT reinvent it).

**Inbound parse (handler side):** mirror `AudioHandler.Transcription`
(`internal/api/audio.go:124-173`):
- Detect `multipart/form-data` via the SHIPPED helper `isMultipart(ctx)`
  (`internal/api/audio.go:283-285`); if absent → `writeError(400,
  "invalid_request_error", "expected multipart/form-data", nil)`.
- Parse via `ctx.MultipartForm()`. On error → `writeError(400, ...)`.
- **Explicit field whitelist** (Go-port note #6, matrix:287): file part `file`
  (required) → read into `FileUploadRequest.File` via the SHIPPED helper
  `readMultipartFile(form, "file")` (`internal/api/audio.go:297-312`); value
  `purpose` (required) → `FileUploadRequest.Purpose` via `formValue(form, "purpose")`
  (`internal/api/audio.go:288-293`); optional `filename` value, else derive from the
  multipart part header → `FileUploadRequest.Filename`. Missing required `file` or
  `purpose` → `writeError(400, ...)`.
  - `FileUploadRequest.Filename` is `json:"-"` (`internal/schemas/files.go:18`); if
    the form has no explicit `filename` value, read it from the file part header
    (`form.File["file"][0].Filename`) so the outbound multipart carries the real
    filename. Default to `"file"` only if absent.

**Outbound build (provider side):** mirror `setTranscriptionBody`
(`internal/providers/openai/audio.go:176-241`): stdlib `mime/multipart.NewWriter`
over a `bytes.Buffer`; write the `file` part (`mw.CreateFormFile("file",
request.Filename)` + `fw.Write(request.File)`) + the `purpose` form value; set
`req.Header.SetContentType(mw.FormDataContentType())`, `req.SetBody(buf.Bytes())`.
(Do NOT reuse `utils.SetJSONBody` for upload — it is multipart.)

**Hermetic test for multipart:** the provider test uses a canned in-memory
multipart body and an `httptest.NewServer` that asserts the inbound `Content-Type`
starts with `multipart/form-data`, that the `file` part round-trips, and that
`purpose` is present (mirror `internal/providers/openai/audio_test.go:74-90`); the
handler test builds a `fasthttp.RequestCtx` with a multipart body + the multipart
Content-Type and asserts the parsed fields reach the fake provider. NO real
network, NO real files on disk (in-memory `bytes` only).

### 1.6 Method/HTTP-verb + path-param + Content-Type seams (BINDING)

| Provider method | HTTP verb | Upstream URI | Body in | Body out |
|---|---|---|---|---|
| `FileUpload` | POST | `p.baseURL + "/v1/files"` | multipart (§1.5) | JSON `*FileObject` |
| `FileList` | GET | `p.baseURL + "/v1/files"` | none | JSON `*FileListResponse` |
| `FileRetrieve` | GET | `p.baseURL + "/v1/files/" + fileID` | none | JSON `*FileObject` |
| `FileDelete` | DELETE | `p.baseURL + "/v1/files/" + fileID` | none | JSON `*FileDeleteResponse` |
| `FileContent` | GET | `p.baseURL + "/v1/files/" + fileID + "/content"` | none | RAW `[]byte` (§1.4) |
| `BatchCreate` | POST | `p.baseURL + "/v1/batches"` | JSON `*BatchCreateRequest` | JSON `*Batch` |
| `BatchList` | GET | `p.baseURL + "/v1/batches"` | none | JSON `*BatchListResponse` |
| `BatchRetrieve` | GET | `p.baseURL + "/v1/batches/" + batchID` | none | JSON `*Batch` |
| `BatchCancel` | POST | `p.baseURL + "/v1/batches/" + batchID + "/cancel"` | none | JSON `*Batch` |

**Path params (handler side):** read via `ctx.UserValue("file_id").(string)` /
`ctx.UserValue("batch_id").(string)`, the SHIPPED fasthttp pattern
(`internal/api/models.go:451` reads `ctx.UserValue("id").(string)`). The route
declarations name the params (`/v1/files/{file_id}`, `/v1/batches/{batch_id}`,
`/v1/batches/{batch_id}/cancel`, `/v1/files/{file_id}/content`). A missing/empty id
→ `writeError(400, "invalid_request_error", "missing file id"/"missing batch id",
nil)`.

**fileID safety:** the `fileID`/`batchID` is interpolated into the upstream URI
path. OpenAI ids are opaque tokens (`file-...`, `batch_...`); the handler MUST
reject an empty id (above). The provider builds the URI with the id as-received
(the upstream rejects malformed ids with its own 4xx, surfaced via
`errorConverter.Convert`). Do NOT attempt local id validation beyond non-empty —
that would diverge from OpenAI's id format silently.

**Content-Type seam for FileContent (§1.4):** since `FileContent` returns only
`[]byte` (no Content-Type), the handler sets a fixed
`Content-Type: application/octet-stream`. If a future row needs the upstream
Content-Type preserved, that is an additive interface change (out of scope here) —
record as an open question if the executor finds the fixed type insufficient for a
hermetic test. Default = `application/octet-stream`.

### 1.7 Go contract (mirrors bf-openai-2 §1.3)

**Schemas (REUSE — already exist, no change expected):**
`internal/schemas/files.go` provides `FileObject` (:4-13), `FileUploadRequest`
(:15-20, `File []byte` + `Filename string`, both `json:"-"`, + `Purpose string`),
`FileListResponse` (:22-26), `FileDeleteResponse` (:28-33). `internal/schemas/batch.go`
provides `Batch` (:4-22), `BatchErrors`/`BatchError` (:24-36),
`BatchRequestCounts` (:38-43), `BatchCreateRequest` (:45-51), `BatchListResponse`
(:53-57). If a field is genuinely missing at impl, set it in the provider, NOT via a
schema edit, unless absence forces an additive struct field (additive-only,
decision 2) — record in WORKFLOW + add the file to §7.

> Soundness note: no files/batches schema carries a `Usage` field (grep-confirmed).
> So the usage-recording glue (`recordNonStream`) records 0 prompt/0 completion
> tokens for these routes (mirroring how `audio.go:118`/`224` records 0 for
> speech/transcription). These endpoints do not return token usage in the OpenAI
> wire shape. Do NOT invent a Usage field. (Files/batches are control-plane
> operations, not token-billed inference.)

**Provider transport (Option A — NEW files):** the nine methods per §1.6, in
`internal/providers/openai/files.go` (5 file methods) + `internal/providers/openai/batches.go`
(4 batch methods). Each mirrors `embedding.go`/`audio.go`/`images.go` transport;
correct `RequestType` per method (`"file_upload"`, `"file_list"`, `"file_retrieve"`,
`"file_delete"`, `"file_content"`, `"batch_create"`, `"batch_list"`,
`"batch_retrieve"`, `"batch_cancel"`). No `init()`; errors-as-values; no panics.

**Handlers (NEW files `internal/api/files.go`, `internal/api/batches.go`):** mirror
`AudioHandler` (`internal/api/audio.go:21-52`) — same struct fields
(`router completionsResolver`-style seam, `usageRecorder`, `pendingTracker`,
`detailCapture`, `vkGate`, `pinnedResolver`), same additive setters, same
`recordGlue()`, same VK gate placement (after Resolve, before dispatch — REUSE the
SHIPPED `resolveAndGate` shape, `internal/api/audio.go:231-259`), same
`gatewayCtx := &schemas.GatewayContext{RequestID: fmt.Sprintf("%d", ctx.ID())}`.

| Aspect | Contract |
|---|---|
| Resolver seam | REUSE the existing `completionsResolver` interface (`internal/api/audio.go:22` uses it as `router completionsResolver`); `inference.Router.Resolve` satisfies it. Do NOT widen the seam. |
| Model for Resolve | Files/batches have NO `model` field. `h.router.Resolve("")` (or a fixed sentinel) selects the openai provider/key. **The executor MUST verify** at P2 how `inference.Router.Resolve` behaves with an empty/sentinel model and pick the minimal sound key-resolution that lands the openai provider (see §8 ESC-RESOLVE-NO-MODEL). If Resolve cannot resolve without a model, use the SHIPPED VK/pinned path to obtain the openai key, or escalate honestly — do NOT fabricate a key. |
| FileUpload parse | **multipart** (§1.5): `file` (required) → `File`; `purpose` (required) → `Purpose`; `filename` (optional / part header) → `Filename`. Missing required → `writeError(400, ...)`. |
| FileList / BatchList parse | none (GET, no body). |
| FileRetrieve / FileDelete / FileContent parse | path param `file_id` (§1.6); empty → `writeError(400, ...)`. |
| BatchCreate parse | JSON: `json.Unmarshal(raw, &schemas.BatchCreateRequest)`; invalid JSON → `writeError(400, "invalid_request_error", "invalid JSON body", nil)`. |
| BatchRetrieve / BatchCancel parse | path param `batch_id` (§1.6); empty → `writeError(400, ...)`. |
| VK gate | `x-g0-vk` gate + pinned-key override, IDENTICAL to `audio.go:231-259` (`resolveAndGate`, reuse `h.vkGate`/`h.pinnedResolver`). Apply BEFORE dispatch. Record endpoint = the route path. |
| Usage glue | include the additive setters + `recordGlue()` so `routes_openai.go` wires usage symmetrically (`audio.go:36-52`). Record under the route's endpoint path; 0/0 tokens (no Usage field). |
| Non-stream dispatch (JSON-out) | call provider method; on `*ProviderError` → `writeProviderError(ctx, perr)` (`internal/api/audio.go:262-268`); on success `jsonMarshal(resp)` → 200 `application/json`; marshal failure → plain-text 500 "internal error" (`audio.go:212-219`). |
| Non-stream dispatch (FileContent bytes-out) | call `provider.FileContent`; on `*ProviderError` → `writeProviderError`; on success: `ctx.SetStatusCode(200)`; `ctx.SetContentType("application/octet-stream")`; `ctx.SetBody(bytes)`. NO jsonMarshal. (§1.4.) |
| Streaming | NONE (§1.1). |

**Construction:** `NewFilesHandler(router *inference.Router) *FilesHandler`,
`NewBatchesHandler(router *inference.Router) *BatchesHandler` (mirror
`NewAudioHandler`, `audio.go:31-33`). NO `New(...)`/`RegisterOpenAIRoutes(...)`
signature change beyond the additive symmetry already present (decision 9) — the
handlers are constructed INSIDE `RegisterOpenAIRoutes` like
`audio := api.NewAudioHandler(router_)` (`routes_openai.go:42`).

> Handler-count decision: ONE handler per file/domain — `FilesHandler` owns
> `Upload` + `List` + `Retrieve` + `Delete` + `Content` (`internal/api/files.go`),
> `BatchesHandler` owns `Create` + `List` + `Retrieve` + `Cancel`
> (`internal/api/batches.go`). Each public method is a distinct
> `func (h *FilesHandler) Upload(ctx)` etc. (so route lines read
> `r.POST("/v1/files", files.Upload)`). This matches the index row
> "`internal/api/{files,batches}.go`" (BIFROST-MAP:298).

### 1.8 routes_openai.go registration (serial-slot additive, §3)

Construct + wire the two handlers alongside the existing ones, and append the route
lines, grouped with the other `/v1/*` routes (after the bf-openai-2 image lines at
`routes_openai.go:120-122`). All paths are distinct; note the `{file_id}` /
`{batch_id}` param routes coexist with the existing `/v1/models/{param}` style —
fasthttp routes them by static prefix (`/v1/files`, `/v1/batches`) so there is no
precedence clash with `/v1/models/*`.

```go
// (in the handler-construction block, after `images := api.NewImagesHandler(router_)` at :43)
files := api.NewFilesHandler(router_)
batches := api.NewBatchesHandler(router_)
// usage glue — extend the existing if-blocks (mirror audio/images wiring :49-50,:57-58,:65-66)
if recorder != nil { files.SetUsageRecorder(recorder); batches.SetUsageRecorder(recorder) }
if tracker  != nil { files.SetPendingTracker(tracker);  batches.SetPendingTracker(tracker)  }
if detail   != nil { files.SetDetailCapture(detail);    batches.SetDetailCapture(detail)    }
if st != nil {
    files.SetVKGate(vkGate);             batches.SetVKGate(vkGate)             // reuse vkGate built at :89
    files.SetVKPinnedResolver(selector); batches.SetVKPinnedResolver(selector) // reuse selector built at :99
}

// (in the route block, after :122 `r.POST("/v1/images/variations", images.Variations)`)
r.POST("/v1/files", files.Upload)
r.GET("/v1/files", files.List)
r.GET("/v1/files/{file_id}", files.Retrieve)
r.DELETE("/v1/files/{file_id}", files.Delete)
r.GET("/v1/files/{file_id}/content", files.Content)
r.POST("/v1/batches", batches.Create)
r.GET("/v1/batches", batches.List)
r.GET("/v1/batches/{batch_id}", batches.Retrieve)
r.POST("/v1/batches/{batch_id}/cancel", batches.Cancel)
```

The `vkGate`/`selector`/`recorder`/`tracker`/`detail` are already constructed
(`routes_openai.go:89,99` + the params); REUSE them — do NOT rebuild. The new
construction + wiring is additive; no existing line is deleted. **Verify the
`r.DELETE`/`r.GET` helper names against the fasthttp `*router.Router` API at P2**
(the existing file uses `r.GET`/`r.POST` — `routes_openai.go:113-125`; `r.DELETE`
is the analogous method).

### NOT in scope (explicit — FORBIDDEN)

- **PAR-BF-OAI-024 (`GET /v1/batches/{batch_id}/results`)** — no provider method,
  no schema; results reachable via 029 (`/v1/files/{output_file_id}/content`).
  Adding a `BatchResults` interface method is a non-additive change touching all 43
  providers — ESCALATED (§8 ESC-BATCH-RESULTS), STAYS MISSING. Do NOT add a
  `BatchResults` method/schema/route.
- **Any local files/batches STORE / SQLite table / batch-execution worker**
  (Option B, §1.2) — REJECTED for this plan. No `internal/store/*` touch, no
  migrate, no new table, no worker, no async fan-out.
- **The other 42 providers' File*/Batch* methods** — UNTOUCHED (they keep their
  501). bf-openai-3 implements ONLY the **openai** provider's nine methods.
- **The remaining openai stubs** in `internal/providers/openai/stubs.go`
  (Responses, ResponsesStream, CountTokens) — UNTOUCHED.
- **Other bf-openai plans' surfaces** — no completions (bf-openai-1, SHIPPED), no
  audio/images (bf-openai-2, SHIPPED — do NOT re-touch), no responses-extras/
  SSE-correctness/compaction (bf-openai-4). Do not touch their schemas/handlers/stubs.
- **The ESC rows** — no responses-rewrite/normalization, no rerank/ocr, no
  containers/async/WS, no raw-passthrough. Touching any ESC surface = REJECT.
- **The `{data,error}` admin envelope on `/v1/*`** (§1.1) — FORBIDDEN. No import of
  `internal/admin` from `internal/api`.
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

# P1 — the files/batches gap is REAL (no route/handler anywhere)
grep -rn '/v1/files\|/v1/batches' internal/ | grep -v _test.go | grep -v internal/schemas/
# ^ expect NOTHING (no files/batches route or handler; only schema comments mention them).
test ! -e internal/api/files.go    && echo "files handler gap OK"
test ! -e internal/api/batches.go  && echo "batches handler gap OK"
test ! -e internal/providers/openai/files.go    && echo "provider files impl gap OK"
test ! -e internal/providers/openai/batches.go  && echo "provider batches impl gap OK"
grep -n 'notImplemented("file_upload")\|notImplemented("file_list")\|notImplemented("file_retrieve")\|notImplemented("file_delete")\|notImplemented("file_content")\|notImplemented("batch_create")\|notImplemented("batch_list")\|notImplemented("batch_retrieve")\|notImplemented("batch_cancel")' internal/providers/openai/stubs.go  # the 9 stubs to replace (:17-51)
# Confirm 024 has NO buildable surface (escalation evidence):
! grep -rn 'BatchResults\|batch_results' internal/schemas/ internal/providers/openai/ && echo "024 has no method/schema — escalate OK"

# P2 — reused surfaces present (the de-risk)
grep -n 'type FileObject\|type FileUploadRequest\|type FileListResponse\|type FileDeleteResponse' internal/schemas/files.go
grep -n 'type Batch\b\|type BatchCreateRequest\|type BatchListResponse' internal/schemas/batch.go
grep -n 'FileUpload\|FileList\|FileRetrieve\|FileDelete\|FileContent\|BatchCreate\|BatchList\|BatchRetrieve\|BatchCancel' internal/schemas/provider.go   # :96-105 interface methods
grep -n 'func (r \*Router) Resolve\b' internal/inference/router.go   # confirm Resolve("") behavior for no-model routes (§1.7)
grep -n 'func SetAuthHeader\|func SetJSONBody\|func ReadJSONBody' internal/providers/utils/*.go
grep -n 'func writeError\|func jsonMarshal\|func (h \*AudioHandler) resolveAndGate\|func writeProviderError\|func isMultipart\|func readMultipartFile\|func formValue\|func requestHeadersFromCtx' internal/api/*.go
grep -n 'func NewAudioHandler\|type completionsResolver\|func NewVKGate' internal/api/*.go
grep -n 'ctx.UserValue(' internal/api/models.go   # :451 path-param read pattern
grep -n 'r.GET\|r.POST\|r.DELETE' internal/server/routes_openai.go   # confirm r.DELETE availability (§1.8)

# P3 — the bf-openai-2 pattern (SHIPPED) is present to mirror
grep -n 'func (h \*AudioHandler) Transcription\|func (h \*AudioHandler) Speech\|func (h \*AudioHandler) recordGlue' internal/api/audio.go
grep -n 'func (p \*Provider) Transcription\b\|func (p \*Provider) Speech\b\|func (p \*Provider) setTranscriptionBody' internal/providers/openai/audio.go
grep -n 'r.POST("/v1/images/variations"' internal/server/routes_openai.go   # :122 (the line after which bf-openai-3 appends)

# P4 — provider transport templates present (Option A)
grep -n 'func (p \*Provider) Embedding\b' internal/providers/openai/embedding.go
grep -n 'p.baseURL = srv.URL' internal/providers/openai/audio_test.go        # hermetic pattern :34
grep -n 'append(\[\]byte(nil), resp.Body()' internal/providers/openai/audio.go  # :65 bytes-clone pattern (FileContent analog)
go test ./internal/providers/openai/ -run 'NotImplemented' -v   # see which File*/Batch* sub-cases exist to remove (§1.3)

# P5 — routes_openai.go SERIAL SLOT is FREE (bf-openai-2 released it on close)
git log --oneline -8 -- internal/server/routes_openai.go
# Orchestrator MUST confirm no concurrent bf-openai-* plan holds an unmerged
# routes_openai.go edit before bf-openai-3 begins T-routes. bf-openai-3 is THIRD
# in the chain (after SHIPPED bf-openai-1, bf-openai-2). TAKES the slot, RELEASES to bf-openai-4.

# P6 — green at base (HERMETIC; no network)
go test ./... && go vet ./... && go build ./...     # exit 0 (untouched-green baseline)
```

---

## 3. Exclusive file ownership

After bf-openai-3 merges, CREATE files are owned by bf-openai-3; later plans
consume, never edit (decision 7).

**CREATE — provider transport (NEW, Option A):**

| File | Contract |
|---|---|
| `internal/providers/openai/files.go` | `FileUpload` + `FileList` + `FileRetrieve` + `FileDelete` + `FileContent` (Option A, §1.6) — moved from stubs.go, now implemented; multipart outbound for upload (§1.5); raw-bytes out for content (§1.4, clone `resp.Body()`); JSON for list/retrieve/delete. No `init()`; errors-as-values; correct `RequestType`. |
| `internal/providers/openai/files_test.go` | RED first. Hermetic `httptest.NewServer` + `p.baseURL = srv.URL` (mirror `audio_test.go:20-34`): `FileUpload` success (canned multipart in test; assert inbound CT starts `multipart/form-data` + `file` part round-trips + `purpose` present) → `*FileObject`; `FileList` → `*FileListResponse`; `FileRetrieve` (assert GET `/v1/files/<id>`) → `*FileObject`; `FileDelete` (assert DELETE `/v1/files/<id>`) → `*FileDeleteResponse`; `FileContent` (assert GET `/v1/files/<id>/content`) → bytes == upstream body; each upstream-non-200 → `*ProviderError` carrying the status. NO real network/files. |
| `internal/providers/openai/batches.go` | `BatchCreate` + `BatchList` + `BatchRetrieve` + `BatchCancel` (Option A, §1.6) — moved from stubs.go, now implemented; JSON body for create; GET for list/retrieve; POST for cancel. No `init()`; errors-as-values; correct `RequestType`. |
| `internal/providers/openai/batches_test.go` | RED first. Hermetic: `BatchCreate` (assert POST `/v1/batches` + JSON body round-trips `input_file_id`/`endpoint`/`completion_window`) → `*Batch`; `BatchList` → `*BatchListResponse`; `BatchRetrieve` (assert GET `/v1/batches/<id>`) → `*Batch`; `BatchCancel` (assert POST `/v1/batches/<id>/cancel`) → `*Batch`; each upstream-non-200 → `*ProviderError`. NO real network. |

**CREATE — api transport (NEW):**

| File | Contract |
|---|---|
| `internal/api/files.go` | `FilesHandler` + `NewFilesHandler` + additive setters (VK/usage) + `recordGlue` + `Upload(ctx)` (multipart-in/JSON-out, §1.5) + `List(ctx)` (JSON-out) + `Retrieve(ctx)` (`{file_id}` param, JSON-out) + `Delete(ctx)` (`{file_id}` param, JSON-out) + `Content(ctx)` (`{file_id}` param, bytes-out, §1.4), §1.7. OpenAI shapes only (§1.1); `writeError`/`writeProviderError` for errors. REUSE the SHIPPED `resolveAndGate`/`isMultipart`/`readMultipartFile`/`formValue`/`writeProviderError`/`jsonMarshal` helpers (do NOT duplicate them). |
| `internal/api/files_test.go` | RED first. Hermetic, fake provider/resolver mirroring `audio_test.go` (embed a base fake to satisfy the full `schemas.Provider` interface): Upload multipart success → bare `FileObject` JSON (assert NO `data`/`error` wrapper); non-multipart upload → 400; missing `file`/`purpose` → 400; List → bare `FileListResponse`; Retrieve/Delete with `{file_id}` → bare object; empty id → 400; Content → raw bytes body + `application/octet-stream` (assert body is NOT JSON `{data}`/`{error}`); provider 501 → 501 passthrough; VK-denied → 429 + provider NOT called; VK-pinned override; marshal failure → plain 500. |
| `internal/api/batches.go` | `BatchesHandler` + `NewBatchesHandler` + additive setters + `recordGlue` + `Create(ctx)` (JSON-in/out) + `List(ctx)` (JSON-out) + `Retrieve(ctx)` (`{batch_id}` param) + `Cancel(ctx)` (`{batch_id}` param, POST), §1.7. OpenAI shapes only; bare `*Batch`/`*BatchListResponse` success. |
| `internal/api/batches_test.go` | RED first. Hermetic fake provider/resolver: Create success → bare `Batch` (assert NO `data`/`error` wrapper); invalid JSON → 400; List → bare `BatchListResponse`; Retrieve/Cancel with `{batch_id}` → bare `Batch`; empty id → 400; provider 501 → 501; VK-denied → 429 + provider NOT called; VK-pinned override; marshal failure → plain 500. |

**EXTEND — provider stubs (REMOVE the nine now-implemented stubs):**

| File | Change |
|---|---|
| `internal/providers/openai/stubs.go` | DELETE `FileUpload`(:17-19), `FileList`(:21-23), `FileRetrieve`(:25-27), `FileDelete`(:29-31), `FileContent`(:33-35), `BatchCreate`(:37-39), `BatchList`(:41-43), `BatchRetrieve`(:45-47), `BatchCancel`(:49-51) ONLY (they move to files.go/batches.go, implemented). The remaining stubs (Responses, ResponsesStream, CountTokens) + the `notImplemented` helper are PRESERVED verbatim. Re-grep live spans at P1. |
| `internal/providers/openai/openai_test.go` | REMOVE the `TestNotImplementedStubs` sub-cases for the File*/Batch* methods that are now implemented (re-grep at P1/P4 to see which survive — bf-openai-2 already removed media sub-cases). The Responses/ResponsesStream/CountTokens sub-cases are PRESERVED. This is the ONLY edit to a pre-existing openai test. |

**MODIFY — serial-slot route registration (additive only):**

| File | Change |
|---|---|
| `internal/server/routes_openai.go` | ADD the `files`/`batches` handler construction + wiring (reuse `vkGate`/`selector`/`recorder`/`tracker`/`detail`) + the NINE route lines (§1.8). NOTHING else changes. SERIAL SLOT — only holder while live; RELEASE to bf-openai-4 on close. |

**FORBIDDEN:** everything else. Explicitly: the remaining openai stubs
(Responses/ResponsesStream/CountTokens); the 42 other providers' File*/Batch*; any
`internal/store/*` (no local file/batch store — §1.2); a `BatchResults` method/
schema/route (024 escalated — §8); all other `internal/api/*.go`
(chat/embeddings/messages/responses/models/completions/audio/images bodies);
`internal/schemas/*` (REUSE files.go/batch.go; edit ONLY if a genuinely-absent
field forces an additive field, §1.7 — then add to §7 + WORKFLOW); all bf-openai-4
surfaces; all `internal/admin/*`, `internal/governance/*`, `internal/mcp/*`,
`internal/providers/catalog/*`; all `ui/**`; all video. Touching any of these is an
automatic REJECT.

---

## 4. TDD tasks

Cadence (strict, AGENTS.md "TDD always"): **no Go impl file may exist before its
`_test.go` is committed RED.** `go test ./... && go vet ./... && go build ./...`
green at EVERY commit (a RED commit may fail ONLY the new package's targeted run;
prefer table/assertion failures over compile failures — scaffold the signatures so
the package compiles and the assertion fails). Order: provider impls → api handlers
→ serial-slot routes → closeout.

### T-prov-files — STEP(a) RED, STEP(b) impl
STEP(a): write `internal/providers/openai/files_test.go` (hermetic httptest +
canned multipart, §3). Run `go test ./internal/providers/openai/ -run 'File'`
→ FAIL. Commit RED:
`phase-1/bf-openai-3: failing openai File upload/list/retrieve/delete/content tests (TDD red)`.
STEP(b): create `internal/providers/openai/files.go` (Option A; multipart out for
upload §1.5; bytes-out for content §1.4); DELETE the five file stubs from
`stubs.go`; REMOVE the File* sub-cases from `openai_test.go`. Gates:
`go test ./... && go vet ./... && go build ./...` green. Commit:
`phase-1/bf-openai-3: implement openai File* methods (upload multipart, content bytes-out)`.

*If a method cannot pass hermetically, STOP and ESCALATE (§8); fall to Option B for
that method only. Do NOT fabricate a green.*

### T-prov-batches — STEP(a) RED, STEP(b) impl
STEP(a): write `internal/providers/openai/batches_test.go` (§3). Run
`go test ./internal/providers/openai/ -run 'Batch'` → FAIL. Commit RED:
`phase-1/bf-openai-3: failing openai Batch create/list/retrieve/cancel tests (TDD red)`.
STEP(b): create `internal/providers/openai/batches.go` (Option A); DELETE the four
batch stubs from `stubs.go`; REMOVE the Batch* sub-cases from `openai_test.go`.
Gates green. Commit:
`phase-1/bf-openai-3: implement openai Batch* methods (create/list/retrieve/cancel)`.

### T-handler-files — STEP(a) RED, STEP(b) impl
STEP(a): write `internal/api/files_test.go` (fake provider/resolver + multipart
request body + path-param ctx, §3). Run `go test ./internal/api/ -run 'File'`
→ FAIL. Commit RED:
`phase-1/bf-openai-3: failing /v1/files handler tests (TDD red)`.
STEP(b): create `internal/api/files.go` (REUSE the SHIPPED multipart/resolveAndGate
helpers). Gates green. Commit:
`phase-1/bf-openai-3: /v1/files handlers (upload multipart, content bytes-out, CRUD)`.

### T-handler-batches — STEP(a) RED, STEP(b) impl
STEP(a): write `internal/api/batches_test.go` (§3). Run
`go test ./internal/api/ -run 'Batch'` → FAIL. Commit RED:
`phase-1/bf-openai-3: failing /v1/batches handler tests (TDD red)`.
STEP(b): create `internal/api/batches.go`. Gates green. Commit:
`phase-1/bf-openai-3: /v1/batches handlers (create/list/retrieve/cancel)`.

### T-routes — serial-slot route registration
TAKE the serial slot (orchestrator confirms FREE at P5). Add the construction +
wiring + the NINE route lines to `routes_openai.go` (§1.8). Gates:
`go test ./... && go vet ./... && go build ./...` green. Commit (ONE commit touches
the serial file):
`phase-1/bf-openai-3: register /v1/files + /v1/batches routes (serial slot)`.

### T-close — full gates + closeout
```bash
go test ./... && go vet ./... && go build ./...
go test ./internal/api/... ./internal/providers/openai/... -run 'File|Batch' -v
go test ./internal/providers/openai/ -run 'NotImplemented' -v   # remaining stubs (Responses/CountTokens) still 501
```
Flip `.planning/parity/matrix/bifrost-openai.md`: 025/026/027/028/029 (files) +
020/021/022/023 (batches) → HAVE; cite the new routes + handlers + impls.
**PAR-BF-OAI-024 → STAYS MISSING + ESCALATED** (cross-reference to 029, §8). Update
`docs/WORKFLOW.md` (P6 base observation, the Option-A passthrough statelessness
decision + evidence, the file-content bytes-out + upload-multipart decisions, the
no-model Resolve outcome, the OpenAI-shape decision, the 024 escalation, the
serial-slot take/release). Append the 024 escalation + the
Option-B-local-store open question to `.planning/parity/plans/open-questions.md`.
Final commit:
`phase-1/bf-openai-3: close — files+batches routes HAVE; 024 escalated; matrix flip`.
**On the close commit, RELEASE the routes_openai.go serial slot to bf-openai-4.**

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0. Diff gate is
**bf-openai-3 commit-range-scoped** (§7). NO e2e.

**Test gates (HERMETIC — no network, no real files)**
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/api/... ./internal/providers/openai/... -run 'File|Batch' -v`
  → exit 0, all pass (upload multipart + list + retrieve + delete + content bytes
  + batch create/list/retrieve/cancel + invalid-input + provider-err + VK-denied +
  VK-pinned + marshal-fail).
- `go test ./internal/providers/openai/ -run 'NotImplemented' -v` → exit 0 (the
  remaining NotImplemented sub-cases — Responses/ResponsesStream/CountTokens — still pass).

**TDD-order proof** — each impl file's covering test appears in an earlier-or-equal commit:
```bash
for pair in \
  "internal/providers/openai/files_test.go:internal/providers/openai/files.go" \
  "internal/providers/openai/batches_test.go:internal/providers/openai/batches.go" \
  "internal/api/files_test.go:internal/api/files.go" \
  "internal/api/batches_test.go:internal/api/batches.go" ; do
  tf=${pair%%:*}; im=${pair##*:}
  ct=$(git log --format=%ct --diff-filter=A -1 -- "$tf")
  cf=$(git log --format=%ct --diff-filter=A -1 -- "$im")
  [ "$ct" -le "$cf" ] || echo "TDD VIOLATION: $im"     # prints nothing
done
```

**Grep proofs**
```bash
# routes registered (serial-slot)
grep -n '"/v1/files"\|"/v1/files/{file_id}"\|"/v1/files/{file_id}/content"' internal/server/routes_openai.go
grep -n '"/v1/batches"\|"/v1/batches/{batch_id}"\|"/v1/batches/{batch_id}/cancel"' internal/server/routes_openai.go
grep -n 'api.NewFilesHandler\|api.NewBatchesHandler' internal/server/routes_openai.go
# handlers exist, OpenAI shape (NOT admin envelope)
grep -n 'func (h \*FilesHandler) Upload\|func (h \*FilesHandler) List\|func (h \*FilesHandler) Retrieve\|func (h \*FilesHandler) Delete\|func (h \*FilesHandler) Content' internal/api/files.go
grep -n 'func (h \*BatchesHandler) Create\|func (h \*BatchesHandler) List\|func (h \*BatchesHandler) Retrieve\|func (h \*BatchesHandler) Cancel' internal/api/batches.go
grep -n 'writeError\|writeProviderError' internal/api/files.go internal/api/batches.go
! grep -rn 'internal/admin' internal/api/files.go internal/api/batches.go && echo "no admin-envelope import OK"
! grep -n '"data"' internal/api/files.go internal/api/batches.go && echo "no {data} wrapper OK"
# file-content bytes-out (NOT jsonMarshal) — octet-stream Content-Type
grep -n 'application/octet-stream\|SetBody' internal/api/files.go
# multipart parse present (REUSE bf-openai-2 helpers)
grep -n 'isMultipart\|MultipartForm\|readMultipartFile' internal/api/files.go
# NO local store touch (Option A, §1.2)
! grep -rn 'internal/store' internal/api/files.go internal/api/batches.go internal/providers/openai/files.go internal/providers/openai/batches.go && echo "no store touch OK"
# provider methods implemented (Option A) — no longer stubs
grep -n 'func (p \*Provider) FileUpload\b\|func (p \*Provider) FileContent\b' internal/providers/openai/files.go
grep -n 'func (p \*Provider) BatchCreate\b\|func (p \*Provider) BatchCancel\b' internal/providers/openai/batches.go
grep -n 'p.baseURL + "/v1/files"\|p.baseURL + "/v1/files/"' internal/providers/openai/files.go
grep -n 'p.baseURL + "/v1/batches"\|p.baseURL + "/v1/batches/"' internal/providers/openai/batches.go
! grep -n 'notImplemented("file_upload")\|notImplemented("batch_create")\|notImplemented("file_content")\|notImplemented("batch_cancel")' internal/providers/openai/stubs.go && echo "9 stubs removed OK"
# remaining stubs preserved; 024 NOT fabricated
grep -n 'notImplemented("count_tokens")\|notImplemented("responses")' internal/providers/openai/stubs.go
! grep -rn 'BatchResults\|batch_results\|/v1/batches/{batch_id}/results' internal/ && echo "024 not fabricated OK"
# no init(); errors-as-values
! grep -rn 'func init(' internal/api/files.go internal/api/batches.go internal/providers/openai/files.go internal/providers/openai/batches.go && echo "no init() OK"
! grep -rn 'panic(' internal/api/files.go internal/api/batches.go internal/providers/openai/files.go internal/providers/openai/batches.go && echo "no panic OK"
```
Plus runtime assertions in the tests:
- FileContent success: response body == upstream file bytes AND `Content-Type ==
  application/octet-stream` AND body is NOT JSON with a `data`/`error` key.
- FileUpload/List/Retrieve/Delete + Batch* success: body unmarshals to the bare
  OpenAI object AND contains NEITHER top-level `"data"` NOR `"error"` key.
- Multipart upload: provider receives a request whose Content-Type starts
  `multipart/form-data`, whose `file` part round-trips the input bytes, and whose
  `purpose` is present.
- Path-param: Retrieve/Delete/Content/Cancel build the upstream URI with the
  received id; empty id → 400 before any provider call.

**Negative / freeze proofs (bf-openai-3 commit-range — §7)**
```bash
R="<first-bf-openai-3>^..<last-bf-openai-3>"
# Only the sanctioned files changed:
git diff $R --name-only -- internal/ | grep -vE \
 'internal/providers/openai/(files|files_test|batches|batches_test|stubs|openai_test)\.go|internal/api/(files|files_test|batches|batches_test)\.go|internal/server/routes_openai\.go' \
 | wc -l                                                                       # = 0
# Remaining openai stubs untouched (only the nine file/batch stubs removed):
git diff $R -- internal/providers/openai/stubs.go | grep -E '^-' | grep -ivE '^---|File(Upload|List|Retrieve|Delete|Content)|Batch(Create|List|Retrieve|Cancel)|notImplemented\("(file|batch)' | grep -iE 'func \(p \*Provider\)' | wc -l   # = 0
# No other api handler body changed:
git diff $R --name-only -- internal/api/ | grep -vE 'internal/api/(files|files_test|batches|batches_test)\.go' | wc -l   # = 0
# No store / catalog / other-provider / admin/governance/mcp/ui touched:
git diff $R --name-only -- internal/store/ internal/providers/catalog/ internal/admin/ internal/governance/ internal/mcp/ ui/ | wc -l   # = 0
git diff $R --name-only -- internal/providers/ | grep -vE 'internal/providers/openai/(files|files_test|batches|batches_test|stubs|openai_test)\.go' | wc -l   # = 0
# routes_openai.go = exactly ONE commit, additive (no route deletions):
git log --oneline $R -- internal/server/routes_openai.go | wc -l              # = 1
git diff $R -- internal/server/routes_openai.go | grep -E '^-' | grep -vE '^---|^-$' | wc -l   # = 0 (no deletions)
# 024 NOT added; no new schema/store table:
git diff $R --name-only -- internal/schemas/ internal/store/ | wc -l         # = 0
```

---

## 6. Out of scope (restated, binding)

No `{data,error}` admin envelope on `/v1/*` (§1.1 — FORBIDDEN; file content returns
raw bytes, the JSON routes return bare OpenAI objects). No local files/batches store
/ SQLite table / batch-execution worker (Option A passthrough, §1.2 — REJECTED for
this plan). No `BatchResults` method/schema/route — 024 escalated (§8), STAYS
MISSING. No ESC rows. No other bf-openai plans' surfaces (completions/audio/images/
responses-extras). No edits to the remaining openai stubs
(Responses/ResponsesStream/CountTokens) or the 42 other providers' File*/Batch*. No
schema rewrite (REUSE files.go/batch.go; additive field ONLY if forced). No UI / e2e
/ mocks. No `New(...)`/`RegisterOpenAIRoutes(...)` signature change, no `init()`, no
global state, no panics. No store/migrate. Matrix-vs-code contradiction (e.g. the
"020..029" span shorthand vs the missing 024 surface, or the stale stub line spans)
→ reconcile honestly in the plan + WORKFLOW (§1 NOTEs), never fabricate. If a method
can't pass hermetically → escalate (§8), do not mark its row HAVE on a 501.

## 7. Diff-gate scope

bf-openai-* plans commit to main concurrently (disjoint from gov/mcp/core, serial
within the openai track), so a broad `<base>..HEAD` range can sweep in siblings.
The diff gate MUST be scoped to bf-openai-3's own commits:
`git log --oneline main | grep "^[0-9a-f]* phase-1/bf-openai-3:" | awk '{print $1}'`
then `git diff <first-bf-openai-3>^..<last-bf-openai-3> -- [file list]`.

`git diff <first>^..<last> --name-only` must be exactly a subset of:
```
internal/providers/openai/files.go
internal/providers/openai/files_test.go
internal/providers/openai/batches.go
internal/providers/openai/batches_test.go
internal/providers/openai/stubs.go            (DELETE the nine file/batch stubs only)
internal/providers/openai/openai_test.go      (REMOVE the File*/Batch* sub-cases only)
internal/api/files.go
internal/api/files_test.go
internal/api/batches.go
internal/api/batches_test.go
internal/server/routes_openai.go               (serial-slot additive; ONE commit)
.planning/parity/matrix/bifrost-openai.md
.planning/parity/plans/open-questions.md
docs/WORKFLOW.md
```
Any file outside this list in the scoped diff is an automatic review REJECT.
`internal/schemas/{files,batch}.go` are deliberately ABSENT (REUSE; additive-field
edit only if §1.7 forces it — then add it to the list with a WORKFLOW note).
`internal/store/*` is deliberately ABSENT (Option A passthrough — no local store).
All other api handlers, all other providers, `internal/providers/catalog`,
store/admin/governance/mcp, all `ui/**`, and all video are ABSENT — touching them is
an automatic REJECT. The `routes_openai.go` edit must appear in exactly ONE commit
(§5) and the serial slot is released to bf-openai-4 on close.

## 8. Escalations / decisions (explicit — recommended defaults, do not fabricate)

- **ESC-STATEFULNESS (RESOLVED at authoring — passthrough, binding default =
  Option A).** §1.2. OpenAI files/batches are stateful at the protocol level, but
  the STATE lives UPSTREAM at OpenAI, not in g0router. Evidence: the provider-method
  signatures take `key Key` + return upstream objects (passthrough-shaped, identical
  to the SHIPPED stateless media methods — `internal/schemas/provider.go:96-105`);
  the stubs live in the PROVIDER layer not a store layer
  (`internal/providers/openai/stubs.go:17-51`); every shipped openai method is a
  thin upstream proxy (`audio.go`/`images.go`/`embedding.go`); there is NO
  files/batches store seam/table/worker anywhere (grep-confirmed). **Option B (a
  g0router-local file/batch store + async batch-execution worker) is REJECTED** for
  this plan as a large non-additive stateful subsystem with no parity-intent
  evidence; it is recorded as a future open question (ESC-LOCAL-STORE, below).
  RECOMMENDED + BINDING: Option A.

- **ESC-LOCAL-STORE (DEFERRED — future open question, NOT this plan).** §1.2. IF a
  future wave genuinely wants a g0router-MANAGED file/batch store (local SQLite
  tables, additive migrations, an async batch-execution worker that fans a JSONL
  input out to per-line completions, accrues `BatchRequestCounts`, writes an
  `output_file_id`, transitions `status`), that is a SEPARATE, deliberate,
  consensus-gated plan (new tables + worker + lifecycle = stateful subsystem, far
  beyond route-over-stub). bf-openai-3 does NOT build it. Recorded in
  `.planning/parity/plans/open-questions.md` at closeout so the option is not lost.

- **ESC-BATCH-RESULTS (RESOLVED at authoring — 024 escalated, binding).** §1 +
  "NOT in scope". `GET /v1/batches/{batch_id}/results` (PAR-BF-OAI-024) has NO
  provider method and NO schema (`internal/schemas/provider.go:102-105` declares
  only Create/List/Retrieve/Cancel; no `BatchResults`/`batch_results` symbol
  anywhere — grep-confirmed). Canonically OpenAI exposes batch results via the
  batch's `output_file_id` fetched through `GET /v1/files/{output_file_id}/content`
  (= PAR-BF-OAI-029, which DOES ship here), so the capability is reachable. Adding a
  literal `/results` route requires a new interface method + schema — a non-additive
  change touching all 43 providers' Provider implementations — which is OUT of the
  buildable-additive bifrost phase. **024 STAYS MISSING + escalated** with the
  cross-reference to 029. NEVER fabricate a `BatchResults` method/schema to "hit the
  020..029 span".

- **ESC-FILE-CONTENT-BYTES (RESOLVED at authoring — non-JSON success body,
  binding).** §1.4. `GET /v1/files/{id}/content` returns RAW file bytes
  (`FileContent` returns `[]byte` — `internal/schemas/provider.go:100`), NOT a JSON
  envelope. The provider clones `resp.Body()` (mirror Speech, `audio.go:65`); the
  handler writes bytes verbatim with `Content-Type: application/octet-stream` — NO
  jsonMarshal, NO `{data}`/`{error}`. The only non-JSON success route in the plan.
  (Content-Type seam: the interface returns no Content-Type, so a fixed
  octet-stream is used; preserving the upstream Content-Type would be an additive
  interface change — out of scope, §1.6.)

- **ESC-MULTIPART-UPLOAD (RESOLVED at authoring — reuses SHIPPED pattern,
  binding).** §1.5. `POST /v1/files` takes `multipart/form-data` (file + purpose).
  bf-openai-2 SHIPPED the multipart pattern (`internal/api/audio.go` inbound +
  `internal/providers/openai/audio.go` outbound); bf-openai-3 REUSES the SHIPPED
  helpers (`isMultipart`/`readMultipartFile`/`formValue` inbound; stdlib
  `mime/multipart.NewWriter` outbound) with an EXPLICIT field whitelist (`file`,
  `purpose`, optional `filename`). Hermetic tests use canned in-memory multipart —
  NO real files, NO real network. Do NOT reinvent the helpers.

- **ESC-RESOLVE-NO-MODEL (CONDITIONAL — at impl).** §1.7. Files/batches requests
  carry NO `model` field, but the handler must obtain the openai provider + upstream
  key via `h.router.Resolve(...)`. The executor MUST verify at P2 how
  `inference.Router.Resolve` behaves with an empty/sentinel model and pick the
  minimal sound resolution that lands the openai provider (e.g. `Resolve("")`, a
  fixed sentinel, or the VK/pinned key path). If Resolve cannot resolve without a
  model AND the VK/pinned path cannot supply the openai key hermetically, STOP and
  ESCALATE honestly (record in WORKFLOW) — do NOT fabricate a key or hard-code a
  credential. Default = the minimal Resolve path that the SHIPPED handlers already
  use; confirm against the live `internal/inference/router.go`.

- **ESC-OPENAI-SHAPE (RESOLVED at authoring — envelope decision, binding,
  inherited).** §1.1. `/v1/*` return OpenAI shapes (bare object or raw bytes +
  OpenAI `{"error":{}}`), NOT the `{data,error}` admin envelope. Verified across
  chat/embeddings/completions/audio/images. The admin envelope is FORBIDDEN here.

- **Serial-slot dependency (§1.8 / P5).** bf-openai-3 is THIRD in the
  routes_openai.go serial chain (after the SHIPPED bf-openai-1, bf-openai-2, which
  RELEASED the slot on close); it TAKES the slot and RELEASES it to bf-openai-4 on
  close. Orchestrator confirms exactly one unmerged holder (decision 3) before
  T-routes.

- **No other blocking dependency.** All reused surfaces (files/batch schemas,
  Router.Resolve, utils helpers, writeError/jsonMarshal/writeProviderError/
  resolveAndGate/isMultipart/readMultipartFile/formValue/requestHeadersFromCtx,
  NewVKGate, the SHIPPED audio handler + provider templates, httptest hermetic
  pattern, stdlib `mime/multipart`, the `ctx.UserValue` path-param pattern) are
  in-tree at <base> (P2/P3/P4). bf-openai-3 is unblocked once the serial slot is
  free.
```
