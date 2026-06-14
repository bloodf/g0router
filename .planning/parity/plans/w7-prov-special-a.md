# Micro-plan w7-prov-special-a — claude-format + URL-template/simple specialized adapters (Go)

```
wave: 7
plan: w7-prov-special-a (SPLIT 1 of 2 from the WAVE-7-MAP w7-prov-special row)
status: READY (rev 1 — authored against live tree @ 28dc097; 9router frozen @ 827e5c3;
  WAVE-7-MAP w7-prov-special row ~line 177 + factory.go micro-serial §195-196,278-280;
  carries the w7-prov-openai §8 ESC-1 claude-format escalation; freeze rules MAP §267)
runs: CATALOG/PROVIDER track. Disjoint from governance/routing/mcp/platform.
  HOLDS the internal/inference/factory.go MICRO-SERIAL slot while live (additive
  switch arms only). Coordinate so only one of {w7-prov-special-a, w7-prov-special-b}
  edits factory.go at a time (sub-serial: special-a → special-b; key-disjoint arms).
  Does NOT touch internal/inference/selection.go (that is the w7-route ↔ w7-plat-1
  micro-serial — unrelated). Does NOT touch internal/server/routes_admin.go.
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w7-prov-special-a:
ref-source: 9router frozen @ 827e5c3 —
  open-sse/config/providers.js (PROVIDERS map),
  open-sse/config/providerModels.js (PROVIDER_MODELS map),
  open-sse/executors/{default,azure}.js, open-sse/executors/vertex.js.
  Per-row ref citations in §6.
base: <base> = git rev-parse HEAD recorded at P0 (28dc097b805b15cbc01281376c17df7e1a848f6 at
  authoring; if main advanced, record the actual SHA and substitute everywhere §5 says <base>).
go-serial-slot: NONE (no routes_admin.go).
factory-micro-serial: YES — this plan ADDS switch arms to buildProvider
  (internal/inference/factory.go:104-109). Additive only; the existing five built-in
  arms + the generic default are UNCHANGED. Confirm the slot is FREE at P0 (no other
  unmerged factory.go holder; w7-platnodes touched providerForModel resolution but is
  MERGED at <base> — verify with §2.6 grep).
freeze: everything outside the §3 ownership set is FROZEN.
new-route: NONE. Providers route via the inference path, NOT admin CRUD — CONFIRMED:
  these providers have no /api/<provider> admin route; they are reached through
  router.go → buildProvider → ChatCompletion (router.go:167). NO routes_admin.go,
  NO UI, NO e2e (no UI contract for providers).
```

---

## 0. Why this is a SPLIT (read first)

The WAVE-7-MAP `w7-prov-special` row bundles 11+ providers across several genuinely
different wire formats. Evidence (§2) shows two cleanly separable clusters with very
different cost/risk:

- **THIS plan (special-a) — cheap, JSON-only, low-risk:** claude-format providers
  (glm/kimi/minimax/minimax-cn — reuse the EXISTING anthropic provider path made
  catalog-driven) + the URL-template/simple-config specialized providers
  (azure resource URL, cloudflare-ai `{accountId}` template, vertex partner-openai
  path, commandcode custom-JSON, qoder, xiaomi-tokenplan region). Every adapter here
  produces/consumes JSON over HTTP. The hardest piece is the claude path generalization
  and a couple of runtime URL-build hooks — all unit-testable with canned fixtures.

- **special-b — heavy, BINARY protocols, higher-risk:** kiro (AWS eventstream binary
  frame decode) + cursor (connect+protobuf) + antigravity (multi-backend per-model
  routing). Evidence: the kiro/cursor/antigravity *message-shape converters* already
  exist in `internal/translation/` (registered in `registry.go:159,171-174`), but the
  raw binary framing (eventstream decode, protobuf encode/decode) was NEVER built — the
  ported converters explicitly assume "the scanner/executor delivers parsed maps"
  (`kiro_openai_response.go:11-12`) and "CursorExecutor already emits OpenAI-shaped
  chunks" (`cursor_openai_response.go:4`). That executor/binary layer is the genuinely
  new, large work — kept in special-b.

Splitting lets special-a ship parity for the JSON-only providers immediately without
being blocked on the binary-protocol research. The two plans share the factory.go
micro-serial (sub-serialized; key-disjoint switch arms).

---

## 1. Scope — PAR rows

### Rows this plan CLOSES (→ HAVE)

| Row | Provider(s) | ref format | New wire-format work | Disposition |
|---|---|---|---|---|
| PAR-PROV-034 | glm | claude | reuse anthropic path (catalog-driven baseURL) | claude-format adapter |
| PAR-PROV-036 | kimi | claude | reuse anthropic path | claude-format adapter |
| PAR-PROV-013 | minimax, minimax-cn | claude | reuse anthropic path | claude-format adapter |
| PAR-PROV-032 | azure | openai (resource URL) | runtime URL build from providerSpecificData | URL-build adapter |
| PAR-PROV-033 | cloudflare-ai | openai (`{accountId}` template) | runtime `{accountId}` substitution | URL-template adapter |
| PAR-PROV-012 | vertex | vertex (native) / openai (partner) | partner-openai path only (native deferred — §8 ESC-A1) | partial; URL-build adapter |
| PAR-PROV-040 | commandcode | commandcode (custom JSON) | reuse EXISTING converters (registry.go:169-170) + adapter | custom-JSON adapter |
| PAR-PROV-028 | qoder | openai (self-built URL `?Encode=1`+sigPath) | runtime URL build | URL-build adapter |
| PAR-PROV-047 | xiaomi-tokenplan | openai (region baseURL) | region→baseURL resolution | URL-resolve adapter |

### Rows DEFERRED to w7-prov-special-b (binary protocols / multi-backend)

| Row | Provider | Why |
|---|---|---|
| PAR-PROV-022 | kiro | AWS eventstream BINARY frame decode not yet built (§0) |
| PAR-PROV-023 | cursor | connect+protobuf encode/decode not yet built (§0) |
| PAR-PROV-020 | antigravity | multi-backend per-model routing + PAR-MCP-060 ride-along |

### Rows ESCALATED / DEFERRED (NOT built by either plan — §8)

| Row | Provider | Why |
|---|---|---|
| PAR-PROV-030 | perplexity-web | cookie-auth reverse-engineered web endpoint — FRAGILE (§8 ESC-A4) |
| PAR-PROV-031 | grok-web | cookie-auth reverse-engineered web endpoint — FRAGILE (§8 ESC-A4) |

### NOT in scope (explicit)

- **No openai-format catalog providers** — those are w7-prov-openai (SHIPPED).
- **No binary-protocol providers** — kiro/cursor/antigravity → special-b.
- **No web-scraped providers** — perplexity-web/grok-web → §8 escalation (defer).
- **No generic adapter rewrite** — `internal/providers/generic/**` is FROZEN. The
  claude path uses the EXISTING anthropic provider; the URL-template providers get a
  thin new adapter or a minimal generic-URL-hook (decided per-provider in §2/§6).
- **No `routes_admin.go`, no admin CRUD, no UI, no e2e, no mock** — providers route
  via the inference path; there is no provider UI contract.
- **No `ProviderConfig`/`ModelEntry` struct change** beyond the ONE additive field
  decided in §2.4 (`ProviderSpecificURL` hook is NOT a struct change — it is resolved
  at request time from `schemas.Key.ProviderSpecificData`, which already exists at
  `internal/schemas/provider.go:34`). Confirm at T0; if any field is genuinely needed
  it is ADDITIVE and asserted in a test.
- **No `New()` signature change** — `generic.New`/`anthropic.NewProvider` signatures
  are preserved; new adapters get their own constructors.
- **No secret exposure** — service-account JSON (vertex) and any persisted cookies
  use the `*_enc` reversible-column pattern (`internal/store/oauthsessions.go`) IF
  persisted; this plan does not persist new secrets (auth tokens come via
  `schemas.Key`) — assert no plaintext secret is logged or written.

---

## 2. Architectural decisions grounding (evidence)

### 2.1 buildProvider is the single dispatch seam (and it ALREADY receives the registry)

`internal/inference/factory.go:94-109`:
```go
func buildProvider(providerID string, reg *translation.Registry) (schemas.Provider, error) {
	switch providerID {
	case "openai":   return openai.NewProvider(), nil
	case "anthropic":return anthropic.NewProvider(), nil
	case "gemini":   return gemini.NewProvider(), nil
	case "ollama", "ollama-local": return ollama.New(providerID, reg)
	default:
		if _, ok := catalog.Lookup(providerID); !ok {
			return nil, fmt.Errorf("unknown provider %q", providerID)
		}
		return generic.New(providerID)   // <-- rejects non-openai formats
	}
}
```
The `default` arm calls `generic.New(providerID)`, which **HARD-REJECTS any non-openai
format** (`internal/providers/generic/provider.go:28-30`:
`if cfg.Format != "openai" { return nil, fmt.Errorf("... not supported by generic adapter") }`).
So a `format:"claude"` or `format:"commandcode"` catalog entry today produces a build
error. The fix is **additive switch arms in buildProvider** that dispatch the
specialized formats to the right adapter BEFORE the generic default. `reg` is already
in scope (passed from `router.go:167`) for adapters that need the translation registry
(commandcode). **This is the factory.go micro-serial edit — additive only.**

### 2.2 claude-format decision: REUSE the existing anthropic provider (catalog-driven baseURL)

**Decision (binding, evidence-based): there IS an existing claude/anthropic adapter
path — reuse it; do NOT build a new claude-generic adapter or per-provider claude
adapters.**

Evidence:
- `internal/providers/anthropic/` is a full Anthropic Messages-wire provider
  (`chat.go:24,84` POST `<baseURL>/v1/messages`; `ConvertRequest` openai→anthropic;
  `errorConverter`; streaming). It is constructed by `anthropic.NewProvider()`
  (`provider.go:18-25`) with a HARDCODED `baseURL: "https://api.anthropic.com"`.
- The 9router claude-format providers are EXACTLY Anthropic Messages wire format with
  a different base URL + `?beta=true` suffix + `x-api-key` auth:
  - glm → `https://api.z.ai/api/anthropic/v1/messages` (matrix PAR-PROV-034)
  - kimi → `https://api.kimi.com/coding/v1/messages` (PAR-PROV-036)
  - minimax → `https://api.minimax.io/anthropic/v1/messages` (PAR-PROV-013)
  - minimax-cn → `https://api.minimaxi.com/anthropic/v1/messages` (PAR-PROV-013)
- `anthropic.Provider.baseURL` is already a settable field (the anthropic stream tests
  set `p.baseURL = srv.URL` at `stream_test.go:28,71`), and `chat.go` builds the URL as
  `p.baseURL + "/v1/messages"`.

**Therefore the claude-format providers are: (a) catalog entries with `Format:"claude"`
+ the provider base URL, and (b) a factory dispatch arm that constructs the anthropic
provider with the catalog base URL + provider id.** The ONLY genuinely-new code is a
small additive anthropic constructor that accepts a base URL/id (e.g.
`anthropic.NewForProvider(id, baseURL)`) — purely additive, does NOT change
`NewProvider()`. The `?beta=true` query suffix and `x-api-key` auth header are
verified against the ref in §6 and asserted in tests; if the existing anthropic
`chat.go` already emits the right auth/path for these, the constructor + base-URL +
beta-suffix is the whole change. **This makes glm/kimi/minimax/minimax-cn cheap
catalog+dispatch entries — NOT a new wire format.**

### 2.3 URL-template / URL-build providers (azure, cloudflare-ai, vertex-partner, qoder, xiaomi-tokenplan)

These are all `format:"openai"` at the WIRE level — the request/response bodies are
plain OpenAI chat-completions. What differs is the **endpoint URL is computed at
request time** from per-key `ProviderSpecificData` (which already exists:
`schemas.Key.ProviderSpecificData map[string]string`, `provider.go:34`) or a region:
- **azure** (PAR-PROV-032): baseUrl `""` in ref; AzureExecutor builds a resource URL.
  Build `https://<resource>.openai.azure.com/openai/deployments/<deployment>/chat/completions?api-version=<v>`
  from `ProviderSpecificData` (resource/deployment/apiVersion). Verify exact shape at
  §6 from `open-sse/executors/azure.js`.
- **cloudflare-ai** (PAR-PROV-033): template
  `https://api.cloudflare.com/client/v4/accounts/{accountId}/ai/v1/chat/completions`;
  `{accountId}` ← `ProviderSpecificData.accountId` (ref `default.js:64-68`).
- **vertex** (PAR-PROV-012): partner-openai path only here — service-account JSON auth
  + dynamic URL; **native `vertex` format deferred (§8 ESC-A1)**. Verify the partner
  URL shape from `open-sse/executors/vertex.js`.
- **qoder** (PAR-PROV-028): executor builds full URL with `?Encode=1` + sigPath query
  params; baseUrl kept for introspection only. Verify from `open-sse/executors/qoder.js`.
- **xiaomi-tokenplan** (PAR-PROV-047): region (`sgp`/`cn`/`ams`) → baseURL resolution.

**Decision:** add a thin specialized adapter package per URL-build family that WRAPS the
generic OpenAI request/response logic but overrides URL construction at request time.
Do NOT rewrite the generic adapter. Two viable shapes (decide per-provider at T-impl,
prefer the one matching existing patterns):
- **(a) New adapter package** `internal/providers/<name>/` embedding/composing the
  generic OpenAI HTTP+JSON logic with a custom `chatURL()`/`buildURL()`. Mirrors how
  `anthropic`/`ollama` are separate packages.
- **(b) generic.NewWithURLBuilder hook** — a purely additive constructor on the generic
  package that takes a `func(key) string` URL builder (analogous to the existing
  additive `generic.NewNode`, `provider.go:44-55`, which already proves the additive-
  constructor pattern is sanctioned). This keeps the JSON/SSE logic in one place.
  **Recommend (b)** for azure/cloudflare-ai/qoder/xiaomi-tokenplan (pure URL override,
  zero body change) since `NewNode` already established the additive-constructor seam;
  use **(a)** only for vertex (service-account auth differs). Resolve at T1 with the
  §6 ref reading; whichever is chosen, `generic.New`/`NewNode` signatures are UNCHANGED.

### 2.4 ProviderSpecificData is the per-request config source (no struct change)

`schemas.Key.ProviderSpecificData map[string]string` already exists
(`internal/schemas/provider.go:32-34`). The URL-build adapters read accountId/resource/
deployment/region/apiVersion from it at request time. **No `ProviderConfig`/`ModelEntry`
struct change is required.** If T1 reveals a genuinely missing field, it is ADDITIVE and
test-asserted; default expectation is zero struct change.

### 2.5 commandcode reuses EXISTING converters

The commandcode request+response converters ALREADY exist and are registered:
`registry.go:169` (`FormatOpenAI→FormatCommandCode` `openaiToCommandCodeRequest`) and
`registry.go:170` (`FormatCommandCode→FormatOpenAI` `commandcodeToOpenAIResponse`), with
implementations at `internal/translation/openai_commandcode_request.go` (7.0K) and
`internal/translation/commandcode_openai_response.go` (7.7K) + their tests. The custom
JSON body is plain JSON (no binary). So commandcode is: catalog entry (`Format:"commandcode"`,
baseUrl `https://api.commandcode.ai/alpha/generate`, the `x-command-code-version`/
`x-cli-environment` headers) + a factory dispatch arm that builds an adapter which calls
the registry to translate request (openai→commandcode) before POST and response
(commandcode→openai) after. The adapter is thin (HTTP + two registry.Translate calls);
the heavy converter logic is already done & tested.

### 2.6 Pre-write verification greps (run at T0)

```bash
# factory.go micro-serial slot is FREE (no other unmerged holder):
git log --oneline <base>..HEAD -- internal/inference/factory.go   # expect empty at P0
# the catalog entries are absent (this plan adds them):
for p in glm kimi minimax minimax-cn azure cloudflare-ai vertex commandcode qoder xiaomi-tokenplan; do
  echo -n "$p: "; grep -c "^\s*\"$p\":" internal/providers/catalog/catalog.go; done   # all 0
# the aliases ALREADY exist (verify-only, no add needed):
grep -nE '"(glm|kimi|minimax|minimax-cn|az|azure|vertex|vx|cf|qd|qoder|cmc|commandcode)"' \
  internal/providers/catalog/aliases.go
# the commandcode converters exist (reuse, do not rebuild):
grep -n 'FormatCommandCode' internal/translation/registry.go
# the anthropic provider baseURL is settable (claude reuse):
grep -n 'baseURL' internal/providers/anthropic/{provider,chat}.go
```

---

## 3. Exclusive file ownership

**MODIFY — factory dispatch (factory.go micro-serial; ADDITIVE switch arms only):**

| File | Change |
|---|---|
| `internal/inference/factory.go` | ADD switch arms to `buildProvider` (before the `default`) dispatching `format:"claude"` providers (glm/kimi/minimax/minimax-cn) to the anthropic-based adapter, and `commandcode`/azure/cloudflare-ai/vertex/qoder/xiaomi-tokenplan to their URL-build/custom-JSON adapters. Dispatch BY catalog `Format` (look up `catalog.Lookup(providerID).Format`) so future same-format providers are free. Existing arms + generic default UNCHANGED. |
| `internal/inference/factory_test.go` | ADD tests asserting `buildProvider` returns the right adapter type / no error for each new provider id; assert the generic default + 5 built-ins still behave (regression). |

**MODIFY — catalog data (ADDITIVE map entries; no struct change):**

| File | Change |
|---|---|
| `internal/providers/catalog/catalog.go` | ADD `ProviderConfig` entries: glm/kimi/minimax/minimax-cn (`Format:"claude"`, ref base URLs, `?beta=true`, `AuthHeader:"x-api-key"`), azure (`Format:"openai"`, baseURL `""` or template), cloudflare-ai (template baseURL), vertex (partner config), commandcode (`Format:"commandcode"` + headers), qoder, xiaomi-tokenplan. Per §6. |
| `internal/providers/catalog/models.go` | ADD `Models` entries for every provider with a ref model block (glm, kimi, minimax, cloudflare-ai, vertex, commandcode, qoder, xiaomi-tokenplan — counts/IDs verbatim from ref §6). azure has NO ref model block → no entry. |
| `internal/providers/catalog/aliases.go` | VERIFY-ONLY (all target aliases already present §2.6). Add only if §2.6 grep shows a gap; if added, update `aliases_test.go` count. |

**NEW — specialized adapters (+ their tests):**

| File | Purpose |
|---|---|
| `internal/providers/anthropic/provider.go` (ADDITIVE constructor) OR new wrapper | `NewForProvider(id, baseURL)` additive constructor for the claude-format providers (does NOT change `NewProvider()`). |
| `internal/providers/<urlbuild>/...` and/or `internal/providers/generic/provider.go` (ADDITIVE `NewWithURLBuilder`) | URL-build adapters for azure/cloudflare-ai/vertex/qoder/xiaomi-tokenplan (shape decided §2.3). |
| `internal/providers/commandcode/*.go` | thin commandcode adapter (HTTP + registry translate calls; reuses existing converters). |
| `*_test.go` for each | hermetic golden fixtures (§4). |

**MODIFY — matrix + workflow (closeout):**

| File | Change |
|---|---|
| `.planning/parity/matrix/9router-providers.md` | Flip PAR-PROV-012(partial→note), 013, 028, 032, 033, 034, 036, 040, 047 → HAVE (012 with the native-format deferral note ESC-A1). Annotate 022/023/020 as → special-b; 030/031 as DEFERRED (§8). |
| `docs/WORKFLOW.md` | Record P0 base SHA, factory micro-serial window, escalations, closeout. |
| `.planning/parity/plans/open-questions.md` | Append §8 escalations. |

**FORBIDDEN (automatic REJECT):** `internal/providers/generic/chat.go` rewrite (only
the additive `NewWithURLBuilder` constructor in `provider.go` is allowed, if shape (b)
chosen); `internal/inference/selection.go`; `internal/server/routes_admin.go`; any
`internal/admin/**`; any `ui/**` or `ui/e2e/**`; any `internal/store/**` (no new
secrets persisted); kiro/cursor/antigravity adapters (special-b); any mock file; any
`ChatRequest`/`ChatResponse`/`Provider`-interface change. NO destructive edits to the
existing five factory arms or the generic default.

---

## 4. TDD tasks

Cadence (strict, AGENTS.md "TDD always: test first, see it fail, minimum code to
pass; no mocks — use fakes/httptest"): **no adapter, catalog entry, or dispatch arm is
added before the failing test that asserts it is committed RED.** All adapter tests are
HERMETIC — `net/http/httptest` fake upstreams returning canned golden request/response
pairs; NO real provider calls. `go test ./... && go vet ./... && go build ./...` green
at EVERY commit. Each adapter's request build + response parse is unit-tested with
golden fixtures.

### T0 — verify slot + facts
Run §2.6 greps; record factory micro-serial slot FREE + catalog-absent + alias-present
in WORKFLOW.md. Record P0 `<base>` SHA. No code.

### T1 — claude-format providers (glm, kimi, minimax, minimax-cn) — RED → GREEN
RED: `factory_test.go` `TestClaudeFormatProvidersDispatch` asserts `buildProvider("glm")`
(and kimi/minimax/minimax-cn) returns the anthropic-based adapter with the catalog
base URL (not the hardcoded api.anthropic.com) and no error; an anthropic-adapter test
`TestNewForProvider` asserts the constructed provider POSTs to `<catalogBaseURL>` with
`?beta=true` and `x-api-key` auth against an `httptest` server returning a canned
Anthropic-Messages response that ConvertResponse turns into the expected OpenAI body.
`catalog_test.go`/`models_test.go` assert the 4 entries' Format/BaseURL + model blocks.
Run → FAILS (entries + constructor + dispatch absent). Commit RED:
`phase-1/w7-prov-special-a: failing claude-format dispatch + catalog tests (TDD red)`.
GREEN: add catalog entries + `anthropic.NewForProvider` + factory arm. Gates green.
Commit: `phase-1/w7-prov-special-a: claude-format providers (glm, kimi, minimax, minimax-cn) via anthropic path`.

### T2 — commandcode (custom JSON via existing converters) — RED → GREEN
RED: `commandcode` adapter test with an `httptest` upstream returning a canned
commandcode response; assert the request body was translated openai→commandcode (via
registry) and the response translated back commandcode→openai. `factory_test.go` arm +
`catalog_test.go`/`models_test.go`. Run → FAILS. Commit RED. GREEN: add catalog entry +
thin adapter (HTTP + registry translate) + factory arm. Gates green.
Commit: `phase-1/w7-prov-special-a: commandcode adapter (reuses existing converters)`.

### T3 — URL-template/build openai providers (cloudflare-ai, azure, qoder, xiaomi-tokenplan) — RED → GREEN
RED: per-provider URL-builder unit tests with golden inputs (e.g.
`{accountId}=abc` → full cloudflare URL; azure resource/deployment → resource URL;
qoder `?Encode=1`+sigPath; xiaomi region `sgp`→baseURL) asserting the EXACT URL string,
plus an `httptest` round-trip asserting an OpenAI request/response passes through
unchanged at the right URL. `factory_test.go` arms + `catalog_test.go`/`models_test.go`
(azure has no model block — assert empty). Run → FAILS. Commit RED. GREEN: add the
additive `generic.NewWithURLBuilder` (or per-provider adapter) + catalog entries +
factory arms. Gates green.
Commit: `phase-1/w7-prov-special-a: URL-template openai providers (cloudflare-ai, azure, qoder, xiaomi-tokenplan)`.

### T4 — vertex (partner-openai path; native deferred) — RED → GREEN
RED: vertex partner adapter test — service-account-derived auth header + dynamically
built partner URL (golden), OpenAI round-trip via `httptest`; assert the service-account
JSON is NEVER logged/echoed (secret-safety assertion). `factory_test.go` arm +
`catalog_test.go`/`models_test.go` (partner model block). Run → FAILS. Commit RED.
GREEN: add adapter + catalog entry + factory arm. Gates green. Document native-format
deferral (§8 ESC-A1) in WORKFLOW.
Commit: `phase-1/w7-prov-special-a: vertex partner-openai adapter (native format deferred)`.

### T5 — full gates + closeout
```bash
go test ./internal/providers/... ./internal/translation/... ./internal/inference/... -run 'ClaudeFormat|CommandCode|URLTemplate|Vertex|Dispatch'
go test ./... && go vet ./... && go build ./...
```
Flip the §1 matrix rows (012-note/013/028/032/033/034/036/040/047 → HAVE; annotate
022/023/020 → special-b; 030/031 DEFERRED). Append §8 to open-questions.md. Update
docs/WORKFLOW.md. Final commit:
`phase-1/w7-prov-special-a: close — claude-format + URL-template adapters; matrix flips`.

---

## 5. Binary acceptance criteria

All yes/no. `<base>` = SHA recorded at P0 (28dc097 at authoring). Diff gate is
commit-range-scoped (§7). HERMETIC — no acceptance command performs a real provider call.

**Test gates**
- `go test ./internal/inference/... -run 'Dispatch' -v` → exit 0.
- `go test ./internal/providers/... -run 'ClaudeFormat|CommandCode|URLTemplate|Vertex' -v` → exit 0.
- `go test ./internal/providers/catalog/... -v` → exit 0.
- `go test ./... && go vet ./... && go build ./...` → exit 0.

**TDD-order proof** — each adapter's data/impl commit follows its RED test commit:
```bash
R="<first-w7-special-a>^..<last-w7-special-a>"
rc=$(git log --format=%ct -1 --grep="failing claude-format dispatch")
dc=$(git log --format=%ct -1 --grep="claude-format providers (glm")
[ "$rc" -le "$dc" ] || echo "TDD VIOLATION: claude-format"   # prints nothing
# (repeat for commandcode / URL-template / vertex)
```

**Grep proofs (per provider)**
```bash
C=internal/providers/catalog/catalog.go
F=internal/inference/factory.go
# claude-format catalog entries with claude format + ref base URLs:
grep -q 'api.z.ai/api/anthropic/v1/messages' $C        # glm (034)
grep -q 'api.kimi.com/coding/v1/messages' $C           # kimi (036)
grep -q 'api.minimax.io/anthropic/v1/messages' $C      # minimax (013)
grep -q 'api.minimaxi.com/anthropic/v1/messages' $C    # minimax-cn (013)
grep -qE '"glm":\s*\{[^}]*Format:\s*"claude"' $C || grep -q 'Format: *"claude"' $C  # claude format used
# URL-template providers:
grep -q 'accounts/{accountId}/ai/v1/chat/completions' $C   # cloudflare-ai (033)
grep -q '"azure":' $C                                       # azure (032)
grep -q '"qoder":' $C && grep -q '"xiaomi-tokenplan":' $C   # 028, 047
grep -q '"commandcode":' $C                                 # 040
grep -q '"vertex":' $C                                      # 012
# factory dispatch arms added (additive — generic default still present):
grep -q 'generic.New(providerID)' $F                        # default unchanged
grep -qE 'case "anthropic"' $F                              # built-in unchanged
# commandcode reuses existing converters (no new converter file):
grep -q 'FormatCommandCode' internal/translation/registry.go   # pre-existing registration intact
# secret-safety: no service-account/private-key string literal committed:
! git diff $R | grep -E '^\+' | grep -qiE 'PRIVATE KEY|service_account.*"private_key"' && echo "no secret committed OK"
```

**No-out-of-scope / freeze proofs (commit-range — §7)**
```bash
git diff $R --name-only | grep -vE \
  'internal/inference/factory(_test)?\.go|internal/providers/(anthropic|commandcode|vertex|azure|cloudflareai|qoder|xiaomi|generic)/.*\.go|internal/providers/catalog/(catalog|models|aliases)(_test)?\.go|\.planning/parity/(matrix/9router-providers|plans/open-questions)\.md|docs/WORKFLOW\.md' \
  | wc -l   # = 0
git diff $R --name-only -- internal/providers/generic/chat.go | wc -l        # = 0 (no generic chat rewrite)
git diff $R --name-only -- internal/inference/selection.go | wc -l           # = 0
git diff $R --name-only -- internal/server/routes_admin.go internal/admin/ ui/ internal/store/ | wc -l  # = 0
# factory edit is ADDITIVE (no removed/renamed existing arms):
git diff $R -- internal/inference/factory.go | grep -E '^\-' | grep -qE 'case "openai"|case "anthropic"|generic.New' && echo "EXISTING ARM CHANGED — REJECT" || echo "additive arms only OK"
```

---

## 6. Per-provider data table (name → format → base_url/URL-build → models → ref source)

All transcribed from 9router @ 827e5c3. **Implementers MUST re-read each ref source at
T-impl and transcribe base URLs / model IDs / header names VERBATIM — never fabricate.**

### claude-format family (reuse anthropic path)

| Provider | format | base_url (ref) | suffix/auth | models (ref) | ref source |
|---|---|---|---|---|---|
| glm | claude | `https://api.z.ai/api/anthropic/v1/messages` | `?beta=true`, `x-api-key` | glm-5.1, glm-5, glm-4.7, glm-4.6v | providers.js:131-134; providerModels.js:321-326 |
| kimi | claude | `https://api.kimi.com/coding/v1/messages` | `?beta=true`, `x-api-key` | kimi-k2.6, kimi-k2.5, kimi-latest | providers.js:141-144; providerModels.js:334-339 |
| minimax | claude | `https://api.minimax.io/anthropic/v1/messages` | `?beta=true`, `x-api-key` | MiniMax-M3, M2.7, M2.5, image model | providers.js:146-154; providerModels.js:340-347 |
| minimax-cn | claude | `https://api.minimaxi.com/anthropic/v1/messages` | `?beta=true`, `x-api-key` | (same block as minimax) | providers.js:146-154; providerModels.js:340-347 |

### URL-build / custom-JSON specialized family

| Provider | format | URL build (ref) | models (ref) | ref source |
|---|---|---|---|---|
| azure | openai | resource URL built by executor from providerSpecificData (resource/deployment/api-version); baseUrl `""` | NO ref model block → no entry | providers.js:384-387; executors/azure.js |
| cloudflare-ai | openai | template `https://api.cloudflare.com/client/v4/accounts/{accountId}/ai/v1/chat/completions`; `{accountId}`←providerSpecificData.accountId | Llama/Mistral/DeepSeek/Moonshot/Qwen + FLUX image | providers.js:390-392; providerModels.js:403-428; default.js:64-68 |
| vertex | openai (partner) | dynamic URL via VertexExecutor.buildUrl(); service-account JSON auth; **native format deferred §8 ESC-A1** | partner block (providerModels.js:580-591) | providers.js:343-352; executors/vertex.js |
| commandcode | commandcode | `https://api.commandcode.ai/alpha/generate`; headers `x-command-code-version`, `x-cli-environment` | deepseek-v4-pro, moonshotai/Kimi-K2.6, zai-org/GLM-5.1, MiniMaxAI/MiniMax-M2.7, Qwen/Qwen3.6-Max-Preview | providers.js:261-267; providerModels.js:446-458 |
| qoder | openai | executor builds full URL `?Encode=1` + sigPath; baseUrl introspection-only | tier+frontier (auto, ultimate, qmodel, dmodel, …) | providers.js:96-103; providerModels.js:147-162; executors/qoder.js |
| xiaomi-tokenplan | openai | region (`sgp`/`cn`/`ams`) → baseURL | mimo-v2.5-pro (+claude native variant — §8 ESC-A2), tts, voice clone/design | providers.js:398-401,447-457; providerModels.js:551-561; executors/xiaomi-tokenplan.js |

Aliases (all present, verify-only): glm, kimi, minimax, minimax-cn, az/azure, vx/vertex,
cf/cloudflare-ai, qd/qoder, cmc/commandcode (`aliases.go` — §2.6).

---

## 7. Diff-gate scope

Isolate this plan's commits:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w7-prov-special-a:" | awk '{print $1}'`
then `git diff <first>^..<last> --name-only` must be a subset of the §3 ownership set.
Any file outside is an automatic REJECT. `internal/providers/generic/chat.go`,
`internal/inference/selection.go`, `internal/server/routes_admin.go`, all `ui/**`, all
`internal/admin/**`, all `internal/store/**`, and the kiro/cursor/antigravity adapters
are deliberately ABSENT — touching them is an automatic REJECT.

---

## 8. Escalations / open questions

- **ESC-A1 (vertex NATIVE format deferred — recorded):** vertex supports both `vertex`
  (native Gemini-on-Vertex) and `openai` (partner-model) formats. This plan ships the
  partner-openai path (URL-build + service-account auth). The native `vertex` format
  shares the gemini-on-vertex converters (`openai_vertex_request.go` exists,
  registry.go:160) but needs the full service-account→GCP-token exchange + native URL
  build — defer to a follow-up unless cheap at T4. PAR-PROV-012 flips HAVE for the
  partner path with this note. **Open question:** does the operator accept partner-only
  vertex as PAR-PROV-012 HAVE?
- **ESC-A2 (xiaomi-tokenplan claude-native model variant — deferred):** the ref
  `mimo-v2.5-pro` carries a `targetFormat:"claude"` native variant + tts/voice models.
  `ModelEntry` has no `TargetFormat` field and this plan does not add one. The chat
  models port as openai; the claude-native + tts variants are a read-site/media concern
  (ties to w7-prov-openai §8 ESC-5 and w7-prov-media). Flag.
- **ESC-A3 (qoder/azure URL-build exact shape — VERIFY-at-impl, binding):** the exact
  qoder `sigPath`/`?Encode=1` signing and azure resource-URL template MUST be
  transcribed from `executors/qoder.js` / `executors/azure.js` at T3 — if the signing
  cannot be soundly reproduced from the ref (e.g. an opaque signature algorithm), DEFER
  that single provider as an escalation rather than fabricate it. Do not guess.
- **ESC-A4 (perplexity-web / grok-web — DEFER, binding recommendation):** PAR-PROV-030
  / PAR-PROV-031 are cookie-auth reverse-engineered web endpoints
  (`authType:"cookie"`, scraping `www.perplexity.ai/rest/...` and
  `grok.com/rest/...`). WAVE-7-MAP escalation §2 says "defer until after GA." Building a
  fragile cookie-scraper has low parity value and high breakage risk. **Recommend:
  DEFER both — leave MISSING with this rationale; do NOT build.** Carried by neither
  special-a nor special-b. Operator decision needed to formally exclude from W7
  100%-of-feasible.
- **ESC-A5 (claude-path auth/beta-suffix confirm — VERIFY-at-impl):** the existing
  `anthropic/chat.go` auth + path must be confirmed to emit `x-api-key` (not the OAuth
  bearer the real anthropic provider may use) and the `?beta=true` query for these
  providers. If the existing chat.go hardcodes anthropic-specific auth/headers, the
  additive constructor must parameterize them (additive) — assert in T1 tests. Not a
  blocker; resolved with evidence at T1.

All ESC items appended to `.planning/parity/plans/open-questions.md` at T5 (Planner
Open_Questions protocol).
