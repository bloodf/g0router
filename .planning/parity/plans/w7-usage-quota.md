# Micro-plan w7-usage-quota — Provider quota/usage fetchers (remaining 6) (Go)

```
wave: 7
plan: w7-usage-quota
status: READY (rev 1 — authored against the SHIPPED w5-e usage dispatcher
  (FetchProviderUsage + the claude/gemini fetchers, LIVE in-tree @
  internal/usage/providerusage.go:129-481) and the SHIPPED w5-e hermetic test
  precedent (httptest.NewServer + the `baseURL ...string` variadic seam +
  snake_case-only assertions @ internal/usage/providerusage_test.go:16-90). The
  admin dispatcher (GET /api/usage/{connectionId}) is ALREADY ROUTED and ALREADY
  delegates to usage.FetchProviderUsage with a refresh-before-fetch + retry-once
  envelope (internal/admin/connectionusage.go:45-147). THIS plan adds the remaining
  6 provider fetchers behind the SAME dispatcher switch — ADDITIVELY (a new
  `case` arm per provider + a NEW file per provider); it does NOT rewrite the
  w5-e claude/gemini fetchers, does NOT change the dispatcher's refresh/retry
  envelope, does NOT touch the admin handler, and does NOT change any New()
  signature. live tree @ <base>; WAVE-7-MAP w7-usage-quota row ~line 187;
  PAR-USAGE-032 (PARTIAL) @ 9router-usage.md:43; concurrency note §205.)
runs: catalog/usage track. EXTENDS the SHIPPED internal/usage package with NEW
  per-provider files + an ADDITIVE switch arm + an ADDITIVE constant per provider
  in providerusage.go. Disjoint from every other domain/store/admin file; runs ∥
  governance + MCP + platform tracks. NO routes_admin.go slot (the route exists —
  MAP w7-usage-quota row: "NO (dispatcher already routed)"). MAY DEPEND on
  w7-prov-oauth for Codex/Kiro/GitHub OAuth tokens (MAP §187 "may depend on
  w7-prov-oauth") — but ONLY for live-token availability at runtime, NOT for any
  code seam: every fetcher is unit-tested HERMETICALLY with a canned token, so this
  plan is code-unblocked today; the dependency is a deferral signal (§8 ESC-OAUTH-DEP),
  not a build blocker.
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w7-usage-quota:
ref-source: 9router frozen @ 827e5c3 — the provider-usage route
  `src/app/api/usage/[connectionId]/route.js:16,122-188` (the dispatcher + the eight
  per-provider fetchers, of which claude+gemini are SHIPPED in w5-e). PAR-USAGE-032
  @ 9router-usage.md:43 names the full set: GitHub, Gemini, Antigravity, Claude,
  Codex, Kiro, GLM, MiniMax. The 6 remaining (Gemini+Claude done) = GitHub,
  Antigravity, Codex, Kiro, GLM, MiniMax.
  ⚠ EVIDENCE CONSTRAINT (binding — §8 ESC-REF-ABSENT): the frozen 9router JS source
  is NOT present on this machine (it lives at the macOS path
  /Users/heitor/Developer/github.com/bloodf/_refs/9router; this Linux tree carries
  only the parity matrix + the SHIPPED Go). The exact per-provider usage endpoint
  URL / auth scheme / response shape for the 6 remaining providers CANNOT be read
  from route.js here. Therefore the per-provider endpoint table (§1.4) is authored
  from the matrix claim + the g0router catalog auth facts + each provider's
  PUBLICLY-DOCUMENTED quota endpoint, and EACH provider carries an explicit
  build-vs-defer verdict. Any provider whose usage endpoint cannot be soundly
  determined is DEFERRED (a graceful fallback message), NEVER fabricated. The
  orchestrator/executor MUST re-read route.js:122-188 against the frozen ref at impl
  time and reconcile per §8 ESC-REF-ABSENT before flipping a provider from DEFER→BUILD.
base: <base> = git rev-parse HEAD recorded at P0. Substitute the actual SHA
  everywhere §5 says <base>. (At authoring, recompute at P0.)
go-serial-slot: NONE. This plan does NOT edit internal/server/routes_admin.go (the
  GET /api/usage/{connectionId} route is already registered and consumed by the
  SHIPPED ConnectionUsageHandler). No selection.go / factory.go micro-serial. The
  usage package is a standalone data surface, not an inference-path concern.
new-route: NONE. NO route file, NO admin handler edit, NO UI touch. The quota page
  consumes the EXISTING GET /api/usage/{connectionId} endpoint with NO contract
  change (the response is the same `map[string]any` quota DTO the page already
  renders for claude/gemini). Library-only extension.
```

---

## 1. Scope — the row + the dispatcher-arm approach

### Row this plan closes

| Row / item | Claim (from `9router-usage.md:43`) | Target state after w7-usage-quota |
|---|---|---|
| PAR-USAGE-032 | Provider usage API fetches external quotas for GitHub, Gemini, Antigravity, Claude, Codex, Kiro, GLM, MiniMax (`route.js:16,122-188`) | HAVE *iff* enough of the remaining 6 are SOUNDLY built; else stays PARTIAL with a per-provider DEFER footnote (§4 T-close, §8 ESC-REF-ABSENT). Gemini+Claude already HAVE (w5-e). |

**Matrix flip rule (binding):** flip PAR-USAGE-032 PARTIAL→HAVE ONLY if every one of
the 6 remaining providers is EITHER soundly built (real fetcher, hermetically tested)
OR soundly DEFERRED with a graceful fallback whose behaviour is unit-tested (the
dispatcher returns a clear "not available" message, no crash). If any provider is
left in an unsound half-state, PAR-USAGE-032 stays PARTIAL with a footnote listing
built-vs-deferred. The closeout note (§4 T-close) records the exact split.

### 1.1 The dispatcher is ALREADY routed + the arm approach (evidence — cite file:line)

- **The route exists and is consumed (do NOT edit).** `GET /api/usage/{connectionId}`
  is served by `ConnectionUsageHandler.GetConnectionUsage`
  (`internal/admin/connectionusage.go:45-147`). It resolves the connection
  (`:68`), guards `Kind=="oauth"` (`:78`), resolves the provider type (`:85`),
  refreshes credentials proactively within the OAuth lead window
  (`:107-115`, `oauthTokenNeedsRefresh` @ `:38`), calls the injectable `Fetcher`
  (default `usage.FetchProviderUsage`, `:124-127`), and on an auth-expired message
  force-refreshes + retries once (`:135-144`). **This entire envelope is REUSED
  verbatim** — the 6 new fetchers slot in BELOW it via the dispatcher switch; the
  admin handler is NOT touched.
- **The dispatcher is `usage.FetchProviderUsage` — a `switch providerType`**
  (`internal/usage/providerusage.go:132-146`): `case "anthropic"` → claude (`:137`),
  `case "gemini"` → gemini (`:139`), `default` → `{"message": "Usage API not
  implemented for <provider>"}` (`:142-144`). **The 6 new providers are 6 new `case`
  arms ADDED before the `default`; the `default` stays as the final catch-all.** The
  claude/gemini arms are NOT touched.
- **The test seam is the `baseURL ...string` variadic** (`providerusage.go:132`,
  threaded via `firstBaseURL` @ `:148`). Each fetcher takes a `baseURL` so the
  hermetic test points it at an `httptest.NewServer`. The w5-e tests pass `srv.URL`
  as the 4th arg (`providerusage_test.go:39`). **Every new fetcher MUST accept the
  same baseURL seam** so its test is hermetic with NO real network.
- **Snake_case-only is enforced** by the existing tests (assert no camelCase keys
  leak: `providerusage_test.go:68-73`). The quota DTO each fetcher returns is the
  same shape claude/gemini emit: a top-level `map[string]any` with `plan` and/or
  `quotas` (a map of label→`{used,total,remaining,remaining_percentage,reset_at,
  unlimited}`) and/or a `message` fallback. **New fetchers MUST emit snake_case keys
  only** and reuse the shared `parseResetTime` helper (`providerusage.go:450-480`)
  for any timestamp normalisation.
- **The Connection carries the tokens (read-only, decrypted on read).**
  `store.Connection{AccessToken, RefreshToken, ExpiresAt, Metadata, Secret}`
  (`internal/store/connections.go:13-26`); secrets are `*_enc` at rest and decrypted
  by `GetConnection` (`connections.go:145-148`). The dispatcher passes the
  decrypted `conn.AccessToken` (and, for api-key providers, `conn.Secret`) into the
  fetcher. **Tokens are used transiently inside the HTTP call and NEVER echoed back
  in any returned map** (the secret-safety contract, §5 grep proof).

### 1.2 The shipped reuse surface (consume — do NOT edit)

- `internal/usage/providerusage.go:132 FetchProviderUsage` — the dispatcher switch.
- `internal/usage/providerusage.go:148 firstBaseURL` — the baseURL-or-default helper
  (REUSE for every new fetcher's default-vs-test-seam).
- `internal/usage/providerusage.go:450 parseResetTime` — the timestamp normaliser
  (epoch s/ms + RFC3339 + RFC1123 → RFC3339; REUSE for any reset field).
- `internal/usage/providerusage.go:281 createClaudeQuotaObject` / `:276 hasUtilization`
  — the utilisation→quota-object shape (REUSE the OUTPUT SHAPE for any
  percentage-based provider; do NOT call these directly if the input fields differ —
  build an analogous `create<Provider>QuotaObject` that emits the SAME snake_case
  keys).
- `internal/usage/providerusage_test.go:16-90` — the hermetic test template
  (`httptest.NewServer` + `srv.Client()` + `srv.URL` seam + snake_case assertions).
- `internal/admin/connectionusage.go:45-147` — the dispatcher's caller envelope
  (refresh-before-fetch, retry-once on auth-expired). CONSUME — do NOT edit.

### 1.3 The auth facts per provider (from the g0router catalog + store, cite file:line)

These are the auth seams the fetchers consume; they are GROUNDED in-tree, independent
of the absent JS ref:

- **glm** — `internal/providers/catalog/catalog.go:377-386`: api-key provider,
  `AuthHeader: "x-api-key"`, base `https://api.z.ai/api/anthropic/v1/messages`
  (Anthropic-format upstream). The connection key is `conn.Secret` (api-key), not an
  OAuth token. Provider `Type` string = `"glm"`.
- **minimax** — `catalog.go:397-405`: api-key, `AuthHeader: "x-api-key"`, base
  `https://api.minimax.io/anthropic/v1/messages`. Key = `conn.Secret`. Type =
  `"minimax"` (and `"minimax-cn"` @ `:407` → `https://api.minimaxi.com`).
- **kiro** — `catalog.go:81-93`: format `"kiro"` (AWS CodeWhisperer eventstream),
  base `https://codewhisperer.us-east-1.amazonaws.com/...`. OAuth/AWS-token provider.
  Type = `"kiro"`. Token = `conn.AccessToken`.
- **antigravity** — `catalog.go:98-105`: format `"antigravity"` (Google Cloud-Code
  envelope), base `https://daily-cloudcode-pa.googleapis.com`. OAuth provider. Type =
  `"antigravity"`. Token = `conn.AccessToken`; project-id may live in
  `conn.Metadata` (mirror the gemini `projectIDFromMetadata` precedent,
  `providerusage.go:427-436`).
- **codex** — OAuth provider (`internal/auth/oauth_providers.go:37 Provider:"codex"`;
  flow registered `routes_admin.go:87`). Type = `"codex"`. Token = `conn.AccessToken`
  (ChatGPT/Codex OAuth). NOT in the chat catalog (it is an OAuth/openai-responses
  upstream).
- **github** — GitHub Copilot OAuth provider. Token = `conn.AccessToken`. (Provider
  `Type` string to confirm at impl — `"github"` per the matrix wording; verify the
  exact store Type the connection records, §8 ESC-PROVIDER-TYPE.)

### 1.4 Per-provider usage-endpoint table (build-vs-defer — binding; §8 ESC-REF-ABSENT)

⚠ The endpoint URL / auth / response-shape columns below are the AUTHORING-TIME best
determination from the matrix claim + the catalog auth facts (§1.3) + publicly
documented provider quota APIs. They are NOT read from the frozen `route.js` (absent
here). **At impl, the executor re-reads `route.js:122-188` against the frozen ref and
reconciles; a provider flips DEFER→BUILD only when its endpoint is soundly confirmed.**

| Provider | Type str | Auth (from §1.3) | Best-known usage/quota endpoint (TO CONFIRM vs route.js) | Response → g0router quota DTO | Verdict (authoring) |
|---|---|---|---|---|---|
| **github** | `github` | `Bearer <conn.AccessToken>` (Copilot OAuth) | GitHub Copilot usage is exposed via the Copilot token/quota endpoint the CLI uses (e.g. `GET https://api.github.com/copilot_internal/v2/token` returns a `{quota_snapshots:{...}}` / limited-user payload) — **CONFIRM the exact path + payload vs route.js**. | `{plan, quotas:{...}}` or `{plan, message}` if the plan has no metered quota. | **BUILD** (sound public endpoint + clear token) — gated on route.js confirm. |
| **antigravity** | `antigravity` | `Bearer <conn.AccessToken>` (Cloud-Code OAuth); project from `conn.Metadata` | Mirrors the gemini Cloud-Code quota call against the antigravity base (`daily-cloudcode-pa.googleapis.com`), i.e. a `…retrieveUserQuota`-shaped POST with `{project}` — **CONFIRM the exact internal path + request body vs route.js** (the gemini fetcher @ `providerusage.go:296-384` is the structural twin). | `{plan, quotas:{modelId→{used,total,reset_at,remaining_percentage,unlimited}}}` (reuse the gemini bucket-mapping shape). | **BUILD** (strong gemini-twin precedent) — gated on route.js confirm. |
| **codex** | `codex` | `Bearer <conn.AccessToken>` (ChatGPT/Codex OAuth) | ChatGPT/Codex usage is the `…/backend-api/…` rate-limit/usage payload the Codex CLI reads — **CONFIRM the exact host+path+payload vs route.js** (codex is not in the chat catalog; the OAuth token is the only in-tree fact). | `{plan, quotas:{...}}` or `{plan, message}`. | **BUILD-IF-CONFIRMED** — endpoint shape is ChatGPT-internal; build only if route.js gives a concrete URL+shape; else **DEFER**. |
| **kiro** | `kiro` | AWS/CodeWhisperer token (`conn.AccessToken`) | Kiro/CodeWhisperer usage/limits endpoint (AWS-signed or bearer; the chat path is the eventstream `generateAssistantResponse` — usage is a SEPARATE limits call) — **CONFIRM vs route.js**; if it requires AWS SigV4 signing not present in-tree, that is a deferral signal. | `{plan, quotas:{...}}` or `{plan, message}`. | **BUILD-IF-CONFIRMED** — defer if the limits call needs SigV4 machinery absent in-tree (§8 ESC-KIRO-SIGV4). |
| **glm** | `glm` | `x-api-key: <conn.Secret>` (api-key, `catalog.go:381`) | z.ai / GLM coding-plan usage endpoint (an api-key GET against the z.ai host returning plan/quota) — **CONFIRM the exact path + payload vs route.js**. | `{plan, quotas:{...}}` or `{plan, message}`. | **BUILD-IF-CONFIRMED** — api-key auth is clear; the usage path needs route.js confirm; else **DEFER**. |
| **minimax** | `minimax` | `x-api-key: <conn.Secret>` (api-key, `catalog.go:401`) | MiniMax account/quota endpoint (api-key GET against `api.minimax.io`) — **CONFIRM the exact path + payload vs route.js**. | `{plan, quotas:{...}}` or `{plan, message}`. | **BUILD-IF-CONFIRMED** — api-key auth is clear; the usage path needs route.js confirm; else **DEFER**. |

**Authoring summary:** 2 high-confidence BUILD (github, antigravity), 4
BUILD-IF-CONFIRMED (codex, kiro, glm, minimax — each becomes BUILD when route.js
confirms a concrete endpoint+shape, else DEFER). **No endpoint is fabricated:** a
provider whose endpoint cannot be soundly confirmed ships the graceful fallback arm
(§1.5) and is recorded DEFERRED. The minimum bar for the matrix flip is in §1 (flip
rule).

### 1.5 The graceful-deferral arm (binding — for any provider not soundly built)

A DEFERRED provider still gets a dispatcher `case` arm, but the arm returns a clear,
snake_case fallback WITHOUT a network call:
```go
case "kiro":
    return map[string]any{
        "message": "Usage API not yet available for Kiro.",
    }, nil
```
This is unit-tested (the dispatcher returns the message, no panic, no network). It is
distinct from the catch-all `default` only in that it is provider-named (so the page
shows a provider-specific message). A DEFERRED provider's arm is the ONLY code it
contributes; no `quota_<provider>.go` HTTP file is created for it until confirmed.

### 1.6 What is UNIT-TESTED (binding — the hermeticity guarantee)

**UNIT-TESTED (deterministic, hermetic — `go test ./...` with NO real network):**
- Each BUILT fetcher via `httptest.NewServer` + the `baseURL` seam: asserts the
  request method, path, and auth header the fetcher sends (Bearer token OR x-api-key);
  asserts the canned response body maps to the snake_case quota DTO
  (`{plan,quotas:{label→{used,total,remaining,remaining_percentage,reset_at,
  unlimited}}}`); asserts NO camelCase key leaks (mirror
  `providerusage_test.go:68-73`); asserts the token/secret value does NOT appear in
  the returned map (secret-safety).
- Each fetcher's error/degraded paths: non-2xx → graceful `{plan?,message}` (no
  crash); body-decode error → `{message}`; missing token/secret → `{message}` (mirror
  the gemini `accessToken==""` guard, `providerusage.go:302-307`).
- Each DEFERRED provider's dispatcher arm: returns the provider-named fallback message.
- The dispatcher routing itself: `FetchProviderUsage("github",…)` reaches the github
  fetcher (table-driven, one sub-test per new provider).

**NO real network, no live token, no SigV4 dial is ever opened by a unit test** — all
HTTP goes through the `httptest` server via the `baseURL` seam.

### NOT in scope (explicit — fetchers + dispatcher arms only)

- **No admin handler edit** — `internal/admin/connectionusage.go` is CONSUMED, NOT
  edited (the refresh/retry envelope + the `Fetcher`/`HTTPClient` seams already exist).
- **No route registration** — NO `internal/server/routes_admin.go` edit (the
  `/api/usage/{connectionId}` route is registered + live). This plan holds NO serial slot.
- **No rewrite of the w5-e fetchers** — `fetchClaudeUsage` (`providerusage.go:157`)
  and `fetchGeminiUsage` (`:296`) + their helpers + the `anthropic`/`gemini` switch
  arms are FROZEN consume-only. The new arms are ADDED before `default`.
- **No change to `FetchProviderUsage`'s signature** — the `(providerType, conn,
  client, baseURL ...string)` signature is preserved; only NEW `case` arms are added
  to its body.
- **No other domains** — NO `internal/store/*` edit (tokens are read-only via the
  existing `GetConnection`); NO `internal/auth/*` edit; NO `internal/providers/*`
  edit; NO `internal/inference/*`.
- **No UI / mock / seed / spec / e2e** — the quota page consumes the EXISTING
  endpoint with the SAME DTO shape; NO UI contract change, so NO `ui/**` touch and
  NO playwright.
- **No new package** — all new files live in the existing `internal/usage` package.
- **No `init()`** — fetchers are plain functions; no global state; errors-as-values
  (a fetcher returns `(map[string]any, error)` but, mirroring w5-e, prefers a
  `{message}` map over a hard error for degraded-but-connected states —
  `providerusage.go:166,172` precedent).
- **No fabricated endpoint** — a provider whose usage endpoint cannot be soundly
  confirmed against the frozen ref ships the §1.5 deferral arm and is recorded DEFERRED.
- **No secret exposure** — tokens/api-keys are used transiently in the request,
  NEVER placed in the returned map (§5 grep proof).

---

## 2. Precondition checks

Run all before any edit; abort and report to orchestrator on any failure.

```bash
# P0 — clean tree; record <base>
git status --porcelain     # expect empty EXCEPT a possibly-dirty ui/dist/index.html
                           # (gitignored build artifact — NEVER stage it, NEVER revert it).
                           # If anything ELSE is dirty, STOP. Use explicit `git add <file>`, never -A.
git rev-parse HEAD         # record as <base> for §5

# P1 — the w5-e dispatcher + claude/gemini fetchers are SHIPPED + present (consume)
grep -nE 'func FetchProviderUsage|case "anthropic"|case "gemini"|func firstBaseURL|func parseResetTime' internal/usage/providerusage.go
grep -nE 'func fetchClaudeUsage|func fetchGeminiUsage' internal/usage/providerusage.go
grep -nE 'baseURL \.\.\.string|httptest\.NewServer|srv\.URL' internal/usage/providerusage_test.go

# P2 — the admin dispatcher route is ALREADY routed (NOT touched by this plan)
grep -nE 'GetConnectionUsage|FetchProviderUsage|Fetcher func|HTTPClient' internal/admin/connectionusage.go
grep -nE '/api/usage/\{connectionId\}|GetConnectionUsage' internal/server/routes_admin.go ; echo "^ route exists (do NOT edit)"

# P3 — the gap is REAL (no per-provider files yet; default arm still catches the 6)
for f in github antigravity codex kiro glm minimax ; do test ! -e internal/usage/quota_$f.go && echo "internal/usage/quota_$f.go gap OK" ; done
grep -nE 'case "github"|case "antigravity"|case "codex"|case "kiro"|case "glm"|case "minimax"' internal/usage/providerusage.go ; echo "^ expect EMPTY (default catches them today)"

# P4 — the auth facts to consume (catalog + store, read-only)
grep -nE '"glm"|"minimax"|"kiro"|"antigravity"' internal/providers/catalog/catalog.go | head
grep -nE 'Provider:\s*"codex"|Provider:\s*"github"' internal/auth/oauth_providers.go ; echo "^ confirm OAuth provider Type strings (§8 ESC-PROVIDER-TYPE)"
grep -nE 'AccessToken|Secret|Metadata|ExpiresAt' internal/store/connections.go | head

# P5 — the frozen 9router ref for route.js (§8 ESC-REF-ABSENT)
test -e /Users/heitor/Developer/github.com/bloodf/_refs/9router/src/app/api/usage/\[connectionId\]/route.js \
  && echo "ref present — read :122-188 per provider" \
  || echo "REF ABSENT on this host — apply ESC-REF-ABSENT: confirm each provider endpoint vs the frozen ref BEFORE DEFER→BUILD; do NOT fabricate"

# P6 — green at base (HERMETIC)
go test ./internal/usage/... ./internal/admin/ -run 'Usage|Quota|ConnectionUsage'
go test ./... && go vet ./... && go build ./...     # exit 0
```

---

## 3. Exclusive file ownership

After w7-usage-quota merges, all CREATE files are owned by this plan; later plans
consume, never edit.

**CREATE — usage fetchers (NEW files in package `internal/usage`, one per BUILT provider):**

| File | Contract |
|---|---|
| `internal/usage/quota_github.go` | `fetchGithubUsage(accessToken string, client *http.Client, baseURL string) (map[string]any, error)`; Bearer auth; GET the confirmed Copilot quota endpoint; map → `{plan,quotas}`/`{message}`; non-2xx + decode-error → `{message}`; missing token → `{message}`. No `init()`; errors-as-values; token never echoed. |
| `internal/usage/quota_github_test.go` | `httptest.NewServer` + `baseURL` seam: asserts method/path/Bearer header; canned payload → snake_case DTO; non-2xx → `{message}`; no camelCase leak; token absent from result. RED first. NO real network. |
| `internal/usage/quota_antigravity.go` | `fetchAntigravityUsage(accessToken, metadata string, client *http.Client, baseURL string)`; Bearer auth; project-id from `metadata` (mirror `projectIDFromMetadata`); POST the confirmed Cloud-Code quota path; map buckets → `{plan,quotas}` (reuse the gemini bucket shape). |
| `internal/usage/quota_antigravity_test.go` | hermetic: project-from-metadata path + canned bucket payload → snake_case quotas; missing-project → `{plan,message}`; non-2xx → `{message}`. RED first. |
| `internal/usage/quota_codex.go` (CONDITIONAL — BUILD only if route.js confirms) | `fetchCodexUsage(accessToken string, client *http.Client, baseURL string)`; Bearer auth; confirmed ChatGPT/Codex usage path → `{plan,quotas}`/`{message}`. If unconfirmed, this file is NOT created and codex ships the §1.5 deferral arm. |
| `internal/usage/quota_codex_test.go` (CONDITIONAL) | hermetic per above. RED first. |
| `internal/usage/quota_kiro.go` (CONDITIONAL — BUILD only if route.js confirms + no SigV4 needed) | `fetchKiroUsage(accessToken string, client *http.Client, baseURL string)`; confirmed CodeWhisperer limits endpoint → DTO. DEFER if SigV4 signing is required (§8 ESC-KIRO-SIGV4). |
| `internal/usage/quota_kiro_test.go` (CONDITIONAL) | hermetic. RED first. |
| `internal/usage/quota_glm.go` (CONDITIONAL — BUILD only if route.js confirms) | `fetchGLMUsage(apiKey string, client *http.Client, baseURL string)`; `x-api-key` auth (api-key = `conn.Secret`); confirmed z.ai usage endpoint → DTO. |
| `internal/usage/quota_glm_test.go` (CONDITIONAL) | hermetic; asserts `x-api-key` header. RED first. |
| `internal/usage/quota_minimax.go` (CONDITIONAL — BUILD only if route.js confirms) | `fetchMiniMaxUsage(apiKey string, client *http.Client, baseURL string)`; `x-api-key` auth; confirmed minimax account/quota endpoint → DTO. |
| `internal/usage/quota_minimax_test.go` (CONDITIONAL) | hermetic; asserts `x-api-key` header. RED first. |

**EXTEND — the dispatcher (additive switch arms + constants ONLY):**

| File | Change (additive ONLY) |
|---|---|
| `internal/usage/providerusage.go` | ADD one `case "<provider>":` arm per provider BEFORE `default` in `FetchProviderUsage` (`:136-145`): BUILT providers delegate to their `fetch<Provider>Usage(...)` with `firstBaseURL(baseURL, default<Provider>BaseURL)`; DEFERRED providers return the §1.5 provider-named fallback. ADD a `default<Provider>BaseURL` const per BUILT provider to the const block (`:20-24`). NO edit to `fetchClaudeUsage`/`fetchGeminiUsage`/`firstBaseURL`/`parseResetTime`/`createClaudeQuotaObject`/the `anthropic`/`gemini` arms/the signature. |
| `internal/usage/providerusage_test.go` (EXTEND additively) | ADD a `TestProviderUsageDispatch` table covering each new arm routes correctly + each DEFERRED arm returns its fallback. (Per-provider deep tests live in the per-provider `_test.go` files.) RED first for any new assertion. |

**FORBIDDEN:** everything else. Explicitly: `internal/admin/connectionusage.go`
(CONSUME — the route + refresh/retry envelope are done); `internal/server/routes_admin.go`
(route exists — NO slot); the w5-e `fetchClaudeUsage`/`fetchGeminiUsage` + their
helpers + the `anthropic`/`gemini` arms + the `FetchProviderUsage` signature
(FROZEN); ALL `internal/store/*` (tokens read-only via existing `GetConnection`); ALL
`internal/auth/*`; ALL `internal/providers/*`; ALL `internal/inference/*`; ALL `ui/**`
(src, mocks, seeds, specs, dist — no contract change); any new package. Touching any
of these is an automatic REJECT.

---

## 4. TDD tasks

Cadence (strict, AGENTS.md "TDD always: write test first, see it fail, write minimum
code to pass"): **no Go impl file may exist before its `_test.go` is committed RED.**
`go test ./... && go vet ./... && go build ./...` green at EVERY commit, FULLY
HERMETIC (no real network, no live token). Order: dispatch-table RED → per-provider
(high-confidence first: github → antigravity) → conditional providers (codex, kiro,
glm, minimax — each gated on route.js confirm; DEFER arm if unconfirmed) → closeout.

### T-dispatch — STEP(a) RED, STEP(b) arms scaffold
STEP(a): write/extend `internal/usage/providerusage_test.go` with
`TestProviderUsageDispatch` asserting each of the 6 provider types routes away from
the generic `default` (BUILT → its fetcher's behaviour via a stub server; DEFERRED →
its provider-named fallback). `go test ./internal/usage/ -run Dispatch` → FAIL.
Commit RED: `phase-1/w7-usage-quota: failing dispatcher-arm routing tests (TDD red)`.
STEP(b): add the 6 `case` arms (DEFER fallback placeholders for any not-yet-built;
BUILT arms wired in their own task below). Gates green. Commit:
`phase-1/w7-usage-quota: dispatcher arms for github/antigravity/codex/kiro/glm/minimax`.

### T-github — STEP(a) RED, STEP(b) impl (BUILD)
RE-READ `route.js:122-188` for the github fetcher; confirm endpoint+auth+shape.
STEP(a): write `internal/usage/quota_github_test.go` (httptest server + baseURL seam;
method/path/Bearer assertions; canned payload → snake_case DTO; non-2xx → `{message}`;
no-camelCase; token-absent). `go test ./internal/usage/ -run Github` → FAIL. Commit RED:
`phase-1/w7-usage-quota: failing github usage fetcher tests (TDD red)`.
STEP(b): implement `internal/usage/quota_github.go` + wire the `case "github"` arm +
`defaultGithubBaseURL` const. Gates green. Commit:
`phase-1/w7-usage-quota: github copilot usage fetcher`.

### T-antigravity — STEP(a) RED, STEP(b) impl (BUILD)
RE-READ `route.js` for antigravity; confirm the Cloud-Code quota path vs the gemini twin.
STEP(a): write `internal/usage/quota_antigravity_test.go` (project-from-metadata +
canned bucket payload → snake_case quotas; missing-project → `{plan,message}`;
non-2xx). → FAIL. Commit RED:
`phase-1/w7-usage-quota: failing antigravity usage fetcher tests (TDD red)`.
STEP(b): implement `internal/usage/quota_antigravity.go` + wire `case "antigravity"`
+ const. Gates green. Commit:
`phase-1/w7-usage-quota: antigravity cloud-code usage fetcher`.

### T-conditional — codex / kiro / glm / minimax (BUILD-IF-CONFIRMED, else DEFER)
For EACH provider: RE-READ `route.js:122-188`. **If the endpoint+auth+shape are
soundly confirmed** → run the STEP(a) RED / STEP(b) impl cycle exactly like T-github
(its own `quota_<provider>_test.go` then `quota_<provider>.go` + arm + const), commit
per provider: `phase-1/w7-usage-quota: <provider> usage fetcher`. **If NOT soundly
confirmable** → leave the §1.5 provider-named DEFER arm (already in T-dispatch),
record the provider DEFERRED in WORKFLOW.md + open-questions.md (§8 ESC-REF-ABSENT),
and do NOT create its HTTP file. The DEFER arm's fallback-message test (from
T-dispatch) is its only coverage. NEVER fabricate an endpoint to force a BUILD.
- kiro extra gate: if the limits call needs AWS SigV4 not present in-tree → DEFER
  (§8 ESC-KIRO-SIGV4), do not import a signer.
- glm/minimax: `x-api-key` from `conn.Secret` (not a Bearer token).

### T-close — full gates + closeout
```bash
go test ./internal/usage/... ./internal/admin/ -run 'Usage|Quota|ConnectionUsage'    # HERMETIC
go test ./... && go vet ./... && go build ./...                                        # exit 0
go test ./internal/usage/ -run 'Github|Antigravity|Codex|Kiro|GLM|MiniMax|Dispatch' -v
```
**Matrix flip (§1 flip rule):** in `.planning/parity/matrix/9router-usage.md`,
PAR-USAGE-032 → HAVE *iff* all 6 are soundly built-or-deferred (every BUILT fetcher
hermetically tested; every DEFER arm returning a tested fallback). Record the exact
built-vs-deferred split in the matrix note. If any provider is in an unsound
half-state, leave PAR-USAGE-032 PARTIAL with the per-provider footnote.
Append the §8 open items (ESC-REF-ABSENT outcome per provider, ESC-PROVIDER-TYPE
resolution, ESC-KIRO-SIGV4 outcome, ESC-OAUTH-DEP note, the built-vs-deferred list)
to `.planning/parity/plans/open-questions.md`. Update `docs/WORKFLOW.md` (P0 base
observation; the ref-absent constraint; which providers built vs deferred and why;
the constructors/arms added). Final commit:
`phase-1/w7-usage-quota: close — provider usage fetchers; matrix flip/footnote`.

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0. Diff gate is
**w7-usage-quota commit-range-scoped** (§7).

**Test gates (HERMETIC — no real network, no live token)**
- `go test ./internal/usage/... ./internal/admin/ -run 'Usage|Quota|ConnectionUsage'`
  → exit 0, all pass.
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/usage/ -run 'Github|Antigravity|Codex|Kiro|GLM|MiniMax|Dispatch' -v`
  → exit 0; each BUILT provider asserts method/path/auth-header + canned→snake_case
  DTO + non-2xx fallback + no-camelCase + token-absent; each DEFERRED provider
  asserts its provider-named fallback message; the dispatch table asserts every new
  type routes away from the generic `default`.

**TDD-order proof** — each impl file's covering test appears in an earlier-or-equal commit:
```bash
for pair in \
  "internal/usage/quota_github_test.go:internal/usage/quota_github.go" \
  "internal/usage/quota_antigravity_test.go:internal/usage/quota_antigravity.go" \
  "internal/usage/quota_codex_test.go:internal/usage/quota_codex.go" \
  "internal/usage/quota_kiro_test.go:internal/usage/quota_kiro.go" \
  "internal/usage/quota_glm_test.go:internal/usage/quota_glm.go" \
  "internal/usage/quota_minimax_test.go:internal/usage/quota_minimax.go" ; do
  tf=${pair%%:*}; im=${pair##*:}
  [ -e "$im" ] || continue                              # skip DEFERRED providers (no impl file)
  ct=$(git log --format=%ct --diff-filter=A -1 -- "$tf")
  cf=$(git log --format=%ct --diff-filter=A -1 -- "$im")
  [ "$ct" -le "$cf" ] || echo "TDD VIOLATION: $im"      # prints nothing
done
```

**Grep proofs (per BUILT fetcher — skip line for any DEFERRED provider)**
```bash
# each BUILT fetcher exists + takes the baseURL seam + uses the shared helpers
grep -nE 'func fetchGithubUsage|func fetchAntigravityUsage' internal/usage/quota_github.go internal/usage/quota_antigravity.go
grep -nE 'baseURL string' internal/usage/quota_*.go                       # every fetcher has the test seam
grep -nE 'firstBaseURL|parseResetTime' internal/usage/providerusage.go    # arms reuse the shared helpers
# the 6 dispatcher arms exist (BUILT delegate or DEFER fallback)
grep -nE 'case "github"|case "antigravity"|case "codex"|case "kiro"|case "glm"|case "minimax"' internal/usage/providerusage.go   # = 6 lines
# snake_case only in returned DTOs (no camelCase quota keys)
! grep -nE 'remainingPercentage|resetAt' internal/usage/quota_*.go && echo "snake_case only OK"
# no init(); no global state
! grep -rn 'func init(' internal/usage/quota_*.go && echo "no init() OK"
# the w5-e fetchers + signature are UNTOUCHED
grep -nE 'func fetchClaudeUsage|func fetchGeminiUsage' internal/usage/providerusage.go   # still present, unchanged
grep -nE 'func FetchProviderUsage\(providerType string, conn \*store.Connection, client \*http.Client, baseURL \.\.\.string\)' internal/usage/providerusage.go   # signature intact
```

**No-secret-exposure proofs (binding)**
```bash
# the token/api-key is never placed into the returned map (used transiently only).
# Each test marshals the fetcher result and asserts the canned token string is ABSENT:
grep -nE 'AccessToken|conn.Secret|apiKey|accessToken' internal/usage/quota_*.go | grep -iE 'map\[string\]any\{' && echo "REJECT: token in a returned map" || echo "no token in returned map OK"
# runtime no-leak assertion lives in each per-provider test (marshal result, assert
# the canned bearer/api-key value is not a substring of the JSON).
```

**Freeze / negative proofs (w7-usage-quota commit-range — §7)**
```bash
R="<first-w7-usage-quota>^..<last-w7-usage-quota>"
# Only the sanctioned usage files changed:
git diff $R --name-only -- internal/ | grep -vE \
 'internal/usage/quota_(github|antigravity|codex|kiro|glm|minimax)(_test)?\.go|internal/usage/providerusage(_test)?\.go' \
 | wc -l                                                                  # = 0
# The admin dispatcher + routes + w5-e auth/store are UNTOUCHED:
git diff $R --name-only -- internal/admin/connectionusage.go internal/server/routes_admin.go internal/store/ internal/auth/ internal/providers/ internal/inference/ | wc -l   # = 0
# UI is untouched (no contract change):
git diff $R --name-only -- ui/ | wc -l                                    # = 0
# providerusage.go change is additive (no deletions of the w5-e arms/helpers):
git diff $R -- internal/usage/providerusage.go | grep -E '^-' | grep -v '^---' | grep -iE 'fetchClaudeUsage|fetchGeminiUsage|case "anthropic"|case "gemini"|firstBaseURL|parseResetTime' | wc -l   # = 0
```

---

## 6. Out of scope (restated, binding)

No admin-handler edit (`connectionusage.go` consumed — route + refresh/retry done).
No routes_admin.go edit (route exists — NO serial slot). No rewrite of the w5-e
claude/gemini fetchers or the `FetchProviderUsage` signature (FROZEN; new arms added
before `default`). No store/auth/providers/inference edits (tokens read-only via
existing `GetConnection`). No UI/mock/seed/spec/e2e (quota page consumes the existing
endpoint with the same DTO — no contract change). No new package. No `init()` / no
global state. No fabricated endpoint — unconfirmable providers ship the graceful
deferral arm and are recorded DEFERRED. No secret exposure (tokens used transiently,
never echoed). Ref-vs-determination contradiction → escalate (§8), never fudge.

## 7. Diff-gate scope

W7 catalog/usage plans commit to main concurrently, so a broad `<base>..HEAD` range
sweeps in sibling commits. The diff gate MUST be scoped to w7-usage-quota's own
commits. Isolate them:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w7-usage-quota:" | awk '{print $1}'`
then `git diff <first-w7-usage-quota>^..<last-w7-usage-quota> -- [file list]`.

`git diff <first>^..<last> --name-only` must be exactly a subset of:
```
internal/usage/quota_github.go
internal/usage/quota_github_test.go
internal/usage/quota_antigravity.go
internal/usage/quota_antigravity_test.go
internal/usage/quota_codex.go            (CONDITIONAL — only if BUILT)
internal/usage/quota_codex_test.go       (CONDITIONAL)
internal/usage/quota_kiro.go             (CONDITIONAL — only if BUILT)
internal/usage/quota_kiro_test.go        (CONDITIONAL)
internal/usage/quota_glm.go              (CONDITIONAL — only if BUILT)
internal/usage/quota_glm_test.go         (CONDITIONAL)
internal/usage/quota_minimax.go          (CONDITIONAL — only if BUILT)
internal/usage/quota_minimax_test.go     (CONDITIONAL)
internal/usage/providerusage.go          (additive arms + consts; w5-e arms/helpers untouched)
internal/usage/providerusage_test.go     (additive dispatch-table tests)
.planning/parity/matrix/9router-usage.md
.planning/parity/plans/open-questions.md
docs/WORKFLOW.md
```
Any file outside this list in the scoped diff is an automatic review REJECT.
`internal/admin/connectionusage.go`, `internal/server/routes_admin.go`,
`internal/store/**`, `internal/auth/**`, `internal/providers/**`,
`internal/inference/**`, and all `ui/**` are deliberately ABSENT — touching them is
an automatic REJECT.

## 8. Escalations / decisions (explicit — recommended defaults, do not fabricate)

- **ESC-REF-ABSENT (BINDING — the frozen 9router JS ref is not on this host).** The
  per-provider usage endpoint URL/auth/response-shape (route.js:122-188) cannot be
  read here (the ref lives at the macOS `/Users/heitor/.../_refs/9router` path). The
  §1.4 table is the AUTHORING-TIME best determination from the matrix + the catalog
  auth facts + public docs. **Decision/protocol:** at impl, the executor MUST read
  `route.js:122-188` against the frozen ref and, per provider, confirm the concrete
  endpoint+auth+shape BEFORE flipping DEFER→BUILD. A provider that cannot be soundly
  confirmed ships the §1.5 graceful deferral arm and is recorded DEFERRED — NEVER
  fabricate an endpoint. RECOMMENDED minimum: github + antigravity BUILT (high
  confidence: clear token + strong gemini-twin precedent); codex/kiro/glm/minimax
  BUILD-IF-CONFIRMED. Flag the built-vs-deferred split for orchestrator confirmation.
- **ESC-PROVIDER-TYPE (CONDITIONAL — confirm the store Type strings).** The
  dispatcher switches on `provider.Type` (`connectionusage.go:85` → `FetchProviderUsage`
  `:129`). The Type strings for the api-key providers (`glm`, `minimax`) are confirmed
  in the catalog (`catalog.go:377,397`); `codex` is confirmed via the OAuth flow
  (`oauth_providers.go:37`); `kiro`/`antigravity` via the catalog (`catalog.go:81,98`).
  **`github` (GitHub Copilot) Type string is NOT confirmed in-tree** — verify the
  exact `provider.Type` a github/copilot connection records at impl (it may be
  `"github"`, `"github-copilot"`, or `"copilot"`). If it diverges, the `case` literal
  must match the recorded Type, not the matrix wording — escalate if ambiguous.
- **ESC-KIRO-SIGV4 (CONDITIONAL — kiro limits-call auth).** If the CodeWhisperer
  usage/limits endpoint requires AWS SigV4 request signing (not present in-tree), kiro
  is DEFERRED (the §1.5 arm) rather than importing a signer — that is a larger,
  separate concern. If route.js shows a simple bearer-token limits call, BUILD.
- **ESC-OAUTH-DEP (NOTE — w7-prov-oauth dependency is runtime-only).** MAP §187 notes
  this plan "may depend on w7-prov-oauth" for Codex/Kiro/GitHub OAuth tokens. That
  dependency is for LIVE token availability at runtime, NOT a code seam: every fetcher
  is unit-tested with a canned token via the httptest seam, so this plan is
  code-unblocked today. If a provider has no OAuth flow shipped yet, its fetcher still
  builds + tests; it simply returns the graceful `{message}` at runtime when no
  connection token exists (mirror the gemini `accessToken==""` guard,
  `providerusage.go:302-307`).
- **No serial-slot dependency.** The `/api/usage/{connectionId}` route is registered
  and live (`connectionusage.go` shipped in w5-e). This plan holds NO routes_admin.go
  slot and is unblocked once P0-P6 pass.
```
