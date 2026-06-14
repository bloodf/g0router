# Micro-plan w7-prov-special-b — binary-protocol + multi-backend specialized adapters (Go)

```
wave: 7
plan: w7-prov-special-b (SPLIT 2 of 2 from the WAVE-7-MAP w7-prov-special row)
status: READY (rev 1 — authored against live tree @ 28dc097; 9router frozen @ 827e5c3;
  WAVE-7-MAP w7-prov-special row ~line 177 + factory.go micro-serial §195-196,278-280;
  PAR-MCP-060 ride-along RESOLVED in w7-mcp-3 §895 — antigravity executor is THIS plan)
runs: CATALOG/PROVIDER track. Disjoint from governance/routing/mcp/platform.
  HOLDS the internal/inference/factory.go MICRO-SERIAL slot while live (additive
  switch arms only). Sub-serial AFTER w7-prov-special-a (special-a → special-b);
  key-disjoint switch arms (special-a adds claude/commandcode/azure/cloudflare/vertex/
  qoder/xiaomi arms; special-b adds kiro/cursor/antigravity arms — no overlap).
  Does NOT touch selection.go or routes_admin.go.
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w7-prov-special-b:
ref-source: 9router frozen @ 827e5c3 —
  open-sse/config/providers.js, open-sse/config/providerModels.js,
  open-sse/executors/{kiro,cursor,antigravity}.js, open-sse/executors/default.js:259-268
  (kiro refresh). Per-row ref citations in §6.
base: <base> = git rev-parse HEAD recorded at P0 (after w7-prov-special-a merges; record
  the actual SHA and substitute everywhere §5 says <base>).
go-serial-slot: NONE (no routes_admin.go).
factory-micro-serial: YES — ADDS kiro/cursor/antigravity switch arms to buildProvider.
  Additive only; special-a's arms + the five built-ins + generic default UNCHANGED.
  Confirm slot FREE at P0 (special-a merged + released).
freeze: everything outside the §3 ownership set is FROZEN.
new-route: NONE. Inference path only. NO routes_admin.go, NO UI, NO e2e, NO mock.
```

---

## 0. Why this plan is the HARD half (read first)

The kiro/cursor/antigravity *message-shape converters already exist and are registered*
in the translation registry:
- `registry.go:171` `FormatOpenAI→FormatKiro` (`buildKiroPayload`),
  `:172` `FormatKiro→FormatOpenAI` (`kiroToOpenAIResponse`).
- `registry.go:173` `FormatOpenAI→FormatCursor` (`buildCursorRequest`),
  `:174` `FormatCursor→FormatOpenAI` (`cursorToOpenAIResponse`).
- `registry.go:159,163-164` antigravity request/response converters
  (`antigravityToOpenAIRequest`, `openaiToAntigravityResponse`, + `geminiToOpenAIResponse`).

But the converters were ported assuming a runtime layer that DOES NOT EXIST yet:
- `kiro_openai_response.go:11-12` (verbatim): *"the raw-SSE-string parsing branch is
  NOT ported — g0router's scanner/executor delivers parsed maps carrying _eventType or
  wrapped event keys."* → **the AWS eventstream BINARY frame decoder that turns
  `application/vnd.amazon.eventstream` bytes into those parsed maps is the missing,
  genuinely-new work.**
- `cursor_openai_response.go:3-5` (verbatim): *"Since CursorExecutor already emits
  OpenAI-shaped chunks, this is a passthrough translator."* → **the CursorExecutor that
  speaks `application/connect+proto` (connect-protocol + protobuf framing) over
  `api2.cursor.sh` is the missing work.** `buildCursorRequest`/`openai_cursor_request.go`
  uses `json.Marshal` only (grep §2.5) — it builds the *logical* request, NOT the
  protobuf wire frames.
- antigravity needs multi-backend per-model routing (gemini / claude / gpt-oss backends
  behind one provider) + the PAR-MCP-060 unavailable-tool ride-along.

So this plan is the binary-protocol + multi-backend executor layer. It is HIGH-RISK
because the wire formats (AWS eventstream binary framing; connect+protobuf) must be
reproduced exactly. **The honesty rule (§8) is load-bearing: build ONLY what can be
soundly reproduced from the ref with real, tested decoders; escalate the rest — never
fabricate a wire protocol.**

---

## 1. Scope — PAR rows

### Rows this plan TARGETS (→ HAVE only if the wire format is soundly reproducible)

| Row | Provider | wire format (the genuinely-new work) | Existing reuse | Risk |
|---|---|---|---|---|
| PAR-PROV-022 | kiro | AWS eventstream binary frame DECODE (`application/vnd.amazon.eventstream`) + custom-JSON token refresh | `buildKiroPayload`, `kiroToOpenAIResponse`, headers/catalog entry already exist (catalog.go:81-93, registry.go:171-172) | MEDIUM — eventstream framing is documented (AWS) and decodable |
| PAR-PROV-023 | cursor | connect-protocol + protobuf encode/decode over `api2.cursor.sh` (`/aiserver.v1.ChatService/StreamUnifiedChatWithTools`) | `buildCursorRequest`, `cursorToOpenAIResponse` (passthrough) exist (registry.go:173-174) | HIGH — protobuf schema must be reverse-engineered from ref; if unsound → ESC-B1 DEFER |
| PAR-PROV-020 | antigravity | multi-backend per-model routing (gemini/claude/gpt-oss) + fallback URL list + PAR-MCP-060 unavailable-tool ride-along | `antigravityToOpenAIRequest`/`openaiToAntigravityResponse` + `geminiToOpenAIResponse` exist (registry.go:159,163-164) | MEDIUM-HIGH — JSON over HTTP but per-model backend dispatch |

### Rows ESCALATED / DEFERRED (§8)

| Row | Provider | Why |
|---|---|---|
| PAR-PROV-023 | cursor (conditional) | DEFER if the connect+protobuf schema cannot be soundly reproduced from `executors/cursor.js` (§8 ESC-B1) — do NOT fabricate protobuf |
| PAR-PROV-030/031 | perplexity-web / grok-web | cookie-scraper web endpoints — DEFER (carried in special-a §8 ESC-A4; not built here either) |

### NOT in scope (explicit)

- **No claude-format / URL-template providers** — those are w7-prov-special-a.
- **No openai-format catalog providers** — w7-prov-openai (SHIPPED).
- **No generic adapter rewrite** — `internal/providers/generic/chat.go` FROZEN.
- **No converter rewrite** — the kiro/cursor/antigravity message-shape converters in
  `internal/translation/` are REUSED, not rewritten. New code is the binary
  framing/decoder + executor adapters that FEED those converters parsed maps.
- **No routes_admin.go, no admin CRUD, no UI, no e2e, no mock** — inference path only.
- **No `ProviderConfig`/`ModelEntry`/`Provider`-interface/`ChatRequest` struct change**
  beyond ADDITIVE fields proven necessary at impl (default: none; assert in tests).
- **No `New()` signature change** for existing constructors.
- **No PAR-MCP-060 route/store work** — that was resolved in w7-mcp-3; this plan only
  supplies the antigravity executor behavior the ride-along references (w7-mcp-3 §895
  ESC-ANTIGRAVITY explicitly assigns the antigravity executor to w7-prov-special).
- **No secret exposure** — kiro OAuth refresh tokens use `schemas.Key` /
  `ProviderSpecificData`; if any token is persisted use the `*_enc` pattern
  (`internal/store/oauthsessions.go`). This plan does not add new persisted secrets;
  assert no token/private-key is logged.

---

## 2. Architectural decisions grounding (evidence)

### 2.1 buildProvider dispatch seam (same as special-a; additive arms)

`internal/inference/factory.go:94-109` — the `default` arm calls `generic.New`, which
rejects non-openai formats (`generic/provider.go:28-30`). kiro already has a catalog
entry with `Format:"kiro"` (catalog.go:81-93) so it currently FAILS to build. Add
additive switch arms for `format:"kiro"`, `format:"cursor"`, `format:"antigravity"`
(dispatch by `catalog.Lookup(providerID).Format`) constructing the new executor
adapters, which take `reg *translation.Registry` (already passed at router.go:167) to
invoke the existing converters. Existing arms UNCHANGED.

### 2.2 kiro — REUSE converters, BUILD the eventstream decoder + executor

Existing & reused:
- catalog entry `kiro` `Format:"kiro"` baseURL
  `https://codewhisperer.us-east-1.amazonaws.com/generateAssistantResponse` + the
  AWS eventstream headers (`Accept: application/vnd.amazon.eventstream`, `X-Amz-Target`,
  User-Agent) — `catalog.go:81-93` (ALREADY PRESENT — verify, do not duplicate).
- `buildKiroPayload` (`openai_kiro_request.go`, 643+ lines, registry.go:171) builds the
  request body (JSON via `json.Marshal`, grep §2.5).
- `kiroToOpenAIResponse` (`kiro_openai_response.go`, registry.go:172) consumes ALREADY-
  PARSED event maps (`resolveKiroEvent`).
- model catalog block — VERIFY present (kiro models may already be in models.go from a
  Wave-1 pairing; if absent, add per §6).

Genuinely-new (THIS plan):
- **AWS eventstream binary frame decoder** — parse the `vnd.amazon.eventstream` response
  stream: 4-byte total length, 4-byte headers length, 4-byte prelude CRC, headers,
  payload, 4-byte message CRC; extract `:event-type`/`:message-type` headers + JSON
  payload → the parsed maps `kiroToOpenAIResponse` expects. This is a documented,
  deterministic binary format → unit-testable with canned `[]byte` golden frames.
- **kiro executor adapter** (`internal/providers/kiro/`): HTTP POST (sigv4/bearer per
  ref) → read eventstream → decode frames → feed `kiroToOpenAIResponse` via registry →
  emit `schemas.StreamChunk`. Custom-JSON token refresh per `default.js:259-268`
  (reuse `schemas.Key`; refresh only if a token is supplied — do not build a new OAuth
  flow here, that is w7-prov-oauth; §8 ESC-B3).

### 2.3 cursor — the protobuf risk (build only if reproducible)

Existing: `buildCursorRequest` (logical request, JSON), `cursorToOpenAIResponse`
(passthrough). The MISSING piece is the `application/connect+proto` framing: connect-
protocol envelope (1-byte flags + 4-byte msg length prefix per frame) wrapping protobuf-
encoded `StreamUnifiedChatWithTools` request/response messages over `api2.cursor.sh`.
**Decision (binding):** read `open-sse/executors/cursor.js` at T-impl. IF the protobuf
message schema + connect framing can be soundly reconstructed (the JS likely hand-rolls
the protobuf field encoding — reproducible), build a `internal/providers/cursor/` adapter
with a hand-rolled protobuf encoder/decoder for exactly the fields the ref uses, unit-
tested with golden wire-byte fixtures. IF the schema is opaque/unverifiable → **DEFER
PAR-PROV-023 (§8 ESC-B1)** and document; ship kiro + antigravity only. Never fabricate
the protobuf layout.

### 2.4 antigravity — multi-backend routing + MCP-060 ride-along

Existing converters: `antigravityToOpenAIRequest`, `openaiToAntigravityResponse`,
`geminiToOpenAIResponse` (registry.go:159,163-164); also
`openai_claude_antigravity.go`, `antigravity_openai_request.go` exist. antigravity
calls DIFFERENT backends per model (gemini / claude / gpt-oss) behind one provider id,
with a fallback baseURL list (`daily-cloudcode-pa.googleapis.com` + sandbox) and the
`User-Agent: antigravity/1.107.0` header (matrix PAR-PROV-020). Build
`internal/providers/antigravity/`: per-model backend selection → pick converter pair →
JSON over HTTP with fallback URL ordering (reuse the multi-URL `chatURLs()` pattern,
`generic/chat.go:29-35`). The PAR-MCP-060 ride-along (antigravity returns certain tools
as hardcoded-unavailable) is resolved as part of the antigravity executor behavior
(w7-mcp-3 §895 assigns it here) — implement the unavailable-tool filtering in the
antigravity request/response path; unit-test the filter with a golden tool list.

### 2.5 Verification greps (run at T0)

```bash
# converters exist (reuse):
grep -nE 'FormatKiro|FormatCursor|FormatAntigravity' internal/translation/registry.go
# kiro catalog entry + eventstream headers ALREADY present (verify, do not dup):
grep -n 'codewhisperer\|vnd.amazon.eventstream\|X-Amz-Target' internal/providers/catalog/catalog.go
# kiro/cursor request builders are JSON-only today (the binary layer is missing):
grep -c 'protobuf\|connect+proto\|eventstream' internal/translation/openai_cursor_request.go internal/translation/openai_kiro_request.go  # expect 0
# factory micro-serial slot FREE (special-a merged & released):
git log --oneline <base>..HEAD -- internal/inference/factory.go
# special-a did NOT already add kiro/cursor/antigravity arms (key-disjoint):
grep -nE 'kiro|cursor|antigravity' internal/inference/factory.go  # expect only via catalog dispatch, not special-a arms
# aliases present (verify-only):
grep -nE '"(kr|kiro|cu|cursor|ag|antigravity)"' internal/providers/catalog/aliases.go
```

---

## 3. Exclusive file ownership

**MODIFY — factory dispatch (micro-serial; ADDITIVE arms; key-disjoint from special-a):**

| File | Change |
|---|---|
| `internal/inference/factory.go` | ADD switch arms dispatching `format:"kiro"`/`"cursor"`/`"antigravity"` to the new executor adapters (pass `reg`). Existing arms (incl. special-a's) UNCHANGED. |
| `internal/inference/factory_test.go` | ADD tests for the 3 new dispatch arms; regression-assert built-ins + generic default + special-a arms unchanged. |

**NEW — executor adapters + binary codecs (+ tests):**

| File | Purpose |
|---|---|
| `internal/providers/kiro/eventstream.go` | AWS eventstream binary frame decoder (length/CRC/headers/payload → parsed maps). |
| `internal/providers/kiro/provider.go` + `chat.go` | kiro executor adapter (HTTP + decode + registry translate + stream). |
| `internal/providers/cursor/*.go` | cursor connect+protobuf adapter (ONLY if ESC-B1 reproducible; else not created). |
| `internal/providers/antigravity/*.go` | antigravity multi-backend executor + unavailable-tool filter (PAR-MCP-060). |
| `*_test.go` for each | HERMETIC golden fixtures: canned eventstream `[]byte` frames, canned protobuf wire bytes, canned antigravity backend responses. |

**MODIFY — catalog data (ADDITIVE; mostly verify-only for kiro):**

| File | Change |
|---|---|
| `internal/providers/catalog/catalog.go` | VERIFY kiro present (catalog.go:81-93). ADD cursor (`Format:"cursor"`, `api2.cursor.sh`, connect+proto headers — only if built) + antigravity (`Format:"antigravity"`, fallback baseURL list, UA header) entries per §6. |
| `internal/providers/catalog/models.go` | ADD `Models` blocks for kiro (if absent), cursor (if built), antigravity per §6. |
| `internal/providers/catalog/aliases.go` | VERIFY-ONLY (kr/kiro, cu/cursor, ag/antigravity present §2.5). |

**MODIFY — matrix + workflow (closeout):**

| File | Change |
|---|---|
| `.planning/parity/matrix/9router-providers.md` | Flip PAR-PROV-022 (kiro) → HAVE; PAR-PROV-020 (antigravity) → HAVE; PAR-PROV-023 (cursor) → HAVE or keep MISSING with ESC-B1 note. |
| `docs/WORKFLOW.md` | Record P0 SHA, factory micro-serial window, the cursor build/defer decision, escalations. |
| `.planning/parity/plans/open-questions.md` | Append §8. |

**FORBIDDEN (automatic REJECT):** `internal/providers/generic/chat.go` rewrite;
`internal/translation/*` converter rewrites (reuse only — registry registrations are
consume-only); special-a's adapter packages; `internal/inference/selection.go`;
`internal/server/routes_admin.go`; `internal/admin/**`; `ui/**`; `internal/store/**`
(no new secrets); any `Provider`/`ChatRequest` interface change; removing/renaming any
existing factory arm.

---

## 4. TDD tasks

Cadence (strict; AGENTS.md TDD; HERMETIC — canned binary/wire fixtures, NO real provider
calls, no mocks-use fakes/httptest). `go test ./... && go vet ./... && go build ./...`
green at EVERY commit.

### T0 — verify slot + facts + cursor go/no-go
Run §2.5 greps. Read `executors/cursor.js` and DECIDE ESC-B1 (build cursor protobuf vs
defer); record the decision + P0 `<base>` in WORKFLOW.md. No code.

### T1 — kiro eventstream decoder (pure unit) — RED → GREEN
RED: `kiro/eventstream_test.go` feeds canned `[]byte` eventstream frames (golden:
known length/CRC/headers/JSON payload) and asserts the decoder yields the exact parsed
event maps. Run → FAILS (no decoder). Commit RED:
`phase-1/w7-prov-special-b: failing kiro eventstream decoder tests (TDD red)`.
GREEN: implement the decoder. Gates green.
Commit: `phase-1/w7-prov-special-b: kiro AWS eventstream frame decoder`.

### T2 — kiro executor adapter + factory arm — RED → GREEN
RED: `kiro/chat_test.go` with an `httptest` server returning a canned eventstream byte
body; assert the adapter POSTs the kiro payload (via `buildKiroPayload`), decodes frames,
runs `kiroToOpenAIResponse` via registry, and emits the expected `schemas.StreamChunk`s.
`factory_test.go` asserts `buildProvider("kiro")` returns the kiro adapter (not an error).
Run → FAILS. Commit RED. GREEN: adapter + factory arm + verify catalog/models. Gates green.
Commit: `phase-1/w7-prov-special-b: kiro executor adapter + factory dispatch`.

### T3 — antigravity multi-backend executor + MCP-060 filter — RED → GREEN
RED: `antigravity/*_test.go` — per-model backend selection (gemini/claude/gpt-oss →
correct converter pair) via `httptest` canned backend responses; fallback URL ordering;
the unavailable-tool filter (golden tool list → filtered output, PAR-MCP-060).
`factory_test.go` arm. `catalog_test.go`/`models_test.go` for the antigravity entry.
Run → FAILS. Commit RED. GREEN: executor + filter + catalog entry + factory arm. Gates green.
Commit: `phase-1/w7-prov-special-b: antigravity multi-backend executor + unavailable-tool filter (PAR-MCP-060)`.

### T4 — cursor connect+protobuf adapter (CONDITIONAL on ESC-B1) — RED → GREEN
ONLY if T0 decided cursor is reproducible. RED: `cursor/*_test.go` — protobuf
encode/decode round-trip with golden wire-byte fixtures + connect-frame envelope
(flags+length prefix); `httptest` round-trip emitting OpenAI chunks via
`cursorToOpenAIResponse`; `factory_test.go` arm. Run → FAILS. Commit RED. GREEN: adapter +
codec + catalog entry + factory arm. Gates green.
Commit: `phase-1/w7-prov-special-b: cursor connect+protobuf executor`.
If DEFERRED: skip T4; record ESC-B1 in WORKFLOW + open-questions; PAR-PROV-023 stays MISSING.

### T5 — full gates + closeout
```bash
go test ./internal/providers/... ./internal/translation/... ./internal/inference/... -run 'Kiro|Cursor|Antigravity|Eventstream|Dispatch'
go test ./... && go vet ./... && go build ./...
```
Flip §1 matrix rows (022, 020 → HAVE; 023 → HAVE or ESC-B1 note). Append §8 to
open-questions.md. Update docs/WORKFLOW.md. Final commit:
`phase-1/w7-prov-special-b: close — binary-protocol adapters; matrix flips`.

---

## 5. Binary acceptance criteria

All yes/no. `<base>` = SHA at P0 (post-special-a). HERMETIC — no acceptance command
performs a real provider call (canned `[]byte`/wire fixtures + `httptest` only).

**Test gates**
- `go test ./internal/providers/kiro/... -v` → exit 0 (incl. eventstream decoder).
- `go test ./internal/providers/antigravity/... -v` → exit 0.
- `go test ./internal/providers/cursor/... -v` → exit 0 (IF built; else dir absent).
- `go test ./internal/inference/... -run 'Dispatch' -v` → exit 0.
- `go test ./... && go vet ./... && go build ./...` → exit 0.

**TDD-order proof**
```bash
R="<first-w7-special-b>^..<last-w7-special-b>"
rc=$(git log --format=%ct -1 --grep="failing kiro eventstream decoder")
dc=$(git log --format=%ct -1 --grep="kiro AWS eventstream frame decoder")
[ "$rc" -le "$dc" ] || echo "TDD VIOLATION: kiro decoder"   # prints nothing
# (repeat for antigravity / cursor if built)
```

**Grep proofs**
```bash
C=internal/providers/catalog/catalog.go
F=internal/inference/factory.go
grep -q 'codewhisperer.us-east-1.amazonaws.com' $C                # kiro (022, pre-existing)
grep -q 'vnd.amazon.eventstream' $C                               # kiro eventstream headers
test -f internal/providers/kiro/eventstream.go                    # decoder exists (022)
grep -q 'api2.cursor.sh' $C || echo "cursor deferred (ESC-B1)"    # cursor built OR deferred
grep -q '"antigravity":' $C                                       # antigravity (020)
grep -q 'antigravity/1.107' $C                                    # antigravity UA header
grep -q 'generic.New(providerID)' $F                              # generic default unchanged
# converters reused (no rewrite):
git diff $R --name-only -- internal/translation/ | grep -vE '_test\.go$' | wc -l   # = 0 (no converter src changed)
# no fabricated protobuf if cursor deferred:
test -d internal/providers/cursor && echo "cursor BUILT" || echo "cursor DEFERRED"
# secret-safety:
! git diff $R | grep -E '^\+' | grep -qiE 'PRIVATE KEY|"refresh_token":\s*"[A-Za-z0-9]' && echo "no secret committed OK"
```

**Freeze proofs (commit-range — §7)**
```bash
git diff $R --name-only | grep -vE \
  'internal/inference/factory(_test)?\.go|internal/providers/(kiro|cursor|antigravity)/.*\.go|internal/providers/catalog/(catalog|models|aliases)(_test)?\.go|\.planning/parity/(matrix/9router-providers|plans/open-questions)\.md|docs/WORKFLOW\.md' \
  | wc -l   # = 0
git diff $R --name-only -- internal/providers/generic/chat.go internal/inference/selection.go internal/server/routes_admin.go internal/admin/ ui/ internal/store/ | wc -l  # = 0
git diff $R -- internal/inference/factory.go | grep -E '^\-' | grep -qE 'case "openai"|case "anthropic"|generic.New' && echo "EXISTING ARM CHANGED — REJECT" || echo "additive arms only OK"
```

---

## 6. Per-provider data table (name → format → wire work → models → ref source)

Transcribe VERBATIM at impl — never fabricate base URLs, model IDs, header names, or
wire layouts.

| Provider | format | base_url (ref) | wire work (genuinely-new) | models (ref) | ref source |
|---|---|---|---|---|---|
| kiro | kiro | `https://codewhisperer.us-east-1.amazonaws.com/generateAssistantResponse` (catalog.go:83 — present) | AWS eventstream binary decode; custom-JSON refresh | thinking/agentic variants + strip lists | providers.js:194-206; providerModels.js:127-146; executors/kiro.js; default.js:259-268 |
| cursor | cursor | `https://api2.cursor.sh` + chatPath `/aiserver.v1.ChatService/StreamUnifiedChatWithTools` | connect+protobuf encode/decode (ESC-B1: build only if reproducible) | claude-4.5-opus, gpt-5.2-codex, kimi-k2.5 | providers.js:208-218; providerModels.js:163-178; executors/cursor.js |
| antigravity | antigravity | fallback list: `daily-cloudcode-pa.googleapis.com` + sandbox; UA `antigravity/1.107.0` | multi-backend per-model routing (gemini/claude/gpt-oss) + fallback URL ordering + unavailable-tool filter (PAR-MCP-060) | per-backend block | providers.js:105-113; providerModels.js:84-94; executors/antigravity.js |

Aliases (verify-only): kr/kiro, cu/cursor, ag/antigravity (`aliases.go` §2.5).

---

## 7. Diff-gate scope

Isolate commits:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w7-prov-special-b:" | awk '{print $1}'`
→ `git diff <first>^..<last> --name-only` must be a subset of the §3 ownership set.
Any file outside is an automatic REJECT. `internal/translation/*` source (converters),
`internal/providers/generic/chat.go`, `selection.go`, `routes_admin.go`, `admin/**`,
`ui/**`, `store/**`, and special-a's adapter packages are deliberately ABSENT.

---

## 8. Escalations / open questions

- **ESC-B1 (cursor protobuf — CONDITIONAL build, binding):** the
  `application/connect+proto` + protobuf wire layout for cursor's
  `StreamUnifiedChatWithTools` MUST be soundly reconstructable from
  `open-sse/executors/cursor.js` at T0/T4. IF the JS hand-rolls reproducible protobuf
  field encoding → BUILD (T4) with golden wire-byte tests. IF opaque/unverifiable →
  **DEFER PAR-PROV-023; do NOT fabricate the protobuf.** Decision recorded at T0.
  **Open question:** operator acceptance of cursor-deferred if unsound.
- **ESC-B2 (antigravity backend matrix — VERIFY-at-impl):** the exact per-model →
  backend (gemini/claude/gpt-oss) mapping + fallback URL order MUST be transcribed from
  `executors/antigravity.js`. If a backend's auth is OAuth-only (antigravity is OAuth,
  matrix PAR-PROV-020), the executor consumes a supplied `schemas.Key` token; the OAuth
  ACQUISITION flow is w7-prov-oauth, not this plan (§ESC-B3). Catalog+executor HAVE is
  satisfied by the wire path; flag the OAuth dependency.
- **ESC-B3 (kiro/antigravity OAuth acquisition — out of scope, recorded):** kiro custom-
  JSON refresh + antigravity OAuth token ACQUISITION belong to w7-prov-oauth
  (`internal/auth/oauth.go` generalization). This plan builds the executors that USE a
  supplied token (`schemas.Key`) + kiro's refresh-on-401 if a refresh token is present;
  it does NOT build the initial OAuth login. PAR-PROV-022/020 catalog+executor parity is
  satisfied by the wire path. Flag the cross-plan dependency.
- **ESC-B4 (perplexity-web / grok-web — DEFER, binding):** PAR-PROV-030/031 cookie-
  scraper web endpoints — NOT built by special-a or special-b (carried in special-a §8
  ESC-A4). Recommend formal DEFER; operator decision to exclude from W7 100%-feasible.
- **ESC-B5 (kiro model block presence — VERIFY):** kiro models may already be in
  models.go from a Wave-1 kiro pairing (w1-i-kiro-pair). VERIFY at T2; add only if
  absent. Avoid duplicate-key panic in the Models map.

All ESC items appended to `.planning/parity/plans/open-questions.md` at T5 (Planner
Open_Questions protocol).
