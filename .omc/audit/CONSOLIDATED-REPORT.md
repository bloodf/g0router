# g0router — Principal Engineer Release Audit

HEAD `8070b5e` · branch `main` · date 2026-06-05 · mode read-only

---

# Verdict

**FAIL** for release (conditionally fixable — no Critical security leak; blocked by High correctness + false-advertising + brand-pollution).

# Executive Summary

Build is real, not vapor: `go vet`, `go build`, **852 Go tests**, **97 UI tests**, **23/24 Playwright e2e** (1 skipped), `make verify`, `git diff --check` all green; gitleaks clean across 420 commits. Structurally sound: good mutex discipline in MCP, honest provider matrix that gates auth-only stubs out of routing, real CRUD dashboard wired to real endpoints.

But it cannot ship as advertised:
- **Anthropic streaming egress drops tool calls** — `/v1/messages` SSE emits only text. Any agentic client using Anthropic format over streaming loses all tool use. Directly contradicts "Anthropic compatible + full API usage."
- **Advertised-but-broken providers**: bedrock + replicate marked `supported` but streaming is a hard stub; cloudflare-ai-gateway silently dead without `account_id`.
- **Concurrency reals**: token-refresh map race + refresh stampede; streaming marks provider-success before any byte flows (defeats backoff).
- **Brand pollution everywhere**: `oh-my-pi/omp`, `9router`, `bifrost` in 30+ source/doc files incl. a `PROVIDERS.md` table with vestigial `OMP ID / 9Router ID / Bifrost ID` columns. User wants these gone.
- **Docs lie in spots**: ARCHITECTURE claims "23+ providers" (actual 43) and an `api/integrations/openai.go` package that was never created; DIRECTORY_STRUCTURE lists 6 phantom CLI files.

Kimi CLI review corroborated the concurrency/leak cluster (over-graded some as Critical; verified down to High/Med).

---

# Blocking Findings (Critical + High)

Ordered by severity. No true Criticals after verification; the following Highs block release.

### B1 — Anthropic streaming egress silently drops tool calls
- **Sev**: High · **Evidence**: `api/handlers/inference.go:151-222` — SSE loop handles only `choice.Delta.Content`; zero `tool_use` / `input_json_delta` blocks emitted. Verified by direct read.
- **Why**: `/v1/messages` streaming is unusable for tool-using agents. Breaks the headline "Anthropic compatible" claim.
- **Fix**: Emit `content_block_start{type:"tool_use"}` + `input_json_delta` + `content_block_stop` when `choice.Delta.ToolCalls` present; track tool block index separately from text block.
- **Verify**: Golden SSE test: OpenAI-format tool-call stream in → assert Anthropic tool_use events out.

### B2 — Token-refresh map data race + stampede
- **Sev**: High · **Evidence**: `internal/provider/refresh.go:34,44,56-59` (delete-before-close window; routing agent H3) and `internal/proxy/engine.go:78,82` (`refreshers`/`quotaFetchers` maps written by `Register*` without lock while read on the request path; kimi #6,#16,#17).
- **Why**: Concurrent requests → duplicate refresh calls / `-race` failures / possible token thrash under load.
- **Fix**: `sync.RWMutex` around refresher/quota maps (or freeze maps before serving). In `refresh.go`, `close(call.done)` before `delete(m.inflight,key)`.
- **Verify**: `go test -race` with concurrent dispatch; test asserting single refresh under N concurrent callers.

### B3 — Streaming marks provider success before data flows
- **Sev**: High · **Evidence**: `internal/proxy/engine.go:209` — `recordProviderSuccess` fires when `ChatCompletionStream` returns the channel, before any chunk. Mid-stream errors never penalize backoff.
- **Why**: Exponential backoff defeated for streaming; a flapping provider keeps getting traffic.
- **Fix**: Record success only after first successful chunk (or stream completion without error).
- **Verify**: Test: stream that errors after channel open → assert backoff incremented, next request rotates.

### B4 — Advertised providers with broken/stub dispatch paths
- **Sev**: High · **Evidence**: bedrock streaming returns `ErrStreamingUnsupported` (`internal/providers/bedrock/bedrock.go:109-111`) yet matrix says `supported`; replicate stream + ListModels hard stubs (`internal/providers/replicate/replicate.go:114-120`); cloudflare-ai-gateway dead without `account_id`, no catalog (`internal/providers/cloudflare/cloudflare.go:56-60`). `internal/provider/matrix.go:42-86`.
- **Why**: "Full usage of providers" is false for these; clients requesting streaming get errors on a `supported` provider.
- **Fix**: Either implement native streaming (bedrock `InvokeModelWithResponseStream`, replicate SSE) or downgrade matrix capability flags to advertise no-stream. Make cloudflare require+validate `account_id` at construction with a clear error.
- **Verify**: Matrix capability test asserting advertised stream flag ⇔ adapter implements streaming.

### B5 — `/v1/models` aborts when any one provider errors
- **Sev**: High · **Evidence**: `internal/proxy/engine.go:222` returns on first `ListModels` error; `:245` swallows errors silently into static catalog (kimi #14,#15).
- **Why**: One misconfigured provider blanks the whole model list for every harness.
- **Fix**: Log and `continue` per-provider; aggregate partial results.
- **Verify**: Test: 1 failing + 2 ok providers → `/v1/models` returns the 2.

### B6 — Internal errors leaked verbatim to clients
- **Sev**: High · **Evidence**: `api/handlers/connections.go:62`, `api/handlers/mcp.go:91,201`, ~12 sites — `fmt.Sprintf("...: %v", err)` into `writeError` exposes SQLite internals, file paths, subprocess errors; MCP OAuth `ClientSecret` may surface on store failure (security H3 + kimi #19).
- **Why**: Info disclosure to authenticated callers; secret-leak risk in driver errors.
- **Fix**: Static client-facing messages; log detail server-side only. Audit all `writeError(... %v, err)`.
- **Verify**: Handler test asserting 500 body contains no `err.Error()` substring.

# Non-Blocking Findings (Medium + Low)

- **M1 cost math** `internal/usage/cost.go:45` — `InputTokens - CacheReadTokens` underflows to negative cost if a provider reports input exclusive of cache reads. Guard `max(0, …)` and pin the per-provider convention. (kimi #12, verified provider-dependent.)
- **M2 cache_write untracked** — `CacheWriteTokens` absent from `internal/usage/tracker.go` pipeline; DB column always NULL; Anthropic/OpenAI cache-write cost lost. (backend agent.)
- **M3 Anthropic ingress tool_use_id loss** — non-stream tool results drop `tool_use_id` (routing M3).
- **M4 SCHEMA.md mismatches (4)** — Quota PascalCase vs documented lowercase (`internal/usage/quota.go:17-24`); `connectionResponse` no json tags (`api/handlers/connections.go:18-34`); `POST /api/oauth/:provider/exchange` undocumented (`api/server.go:532`); `cache_write_tokens` documented but never populated.
- **M5 streaming clients no timeout** `internal/providers/openai/openai.go:33` (+anthropic/gemini/azure/vertex/ollamacloud) — stalled upstream hangs SSE reader forever. (kimi #13.)
- **M6 MCP cancellation absent** — `notifications/cancelled` never sent on any transport (`internal/mcp/stdio.go:114`, `httpclient.go:90,253`); server tools run after client cancel. (mcp H1.)
- **M7 MCP stdio read under lock** `internal/mcp/stdio.go:131` — `ReadBytes` blocks holding `c.mu`, no ctx interrupt; hung subprocess deadlocks that client. (mcp H2.)
- **M8 ToolManager call after close** `internal/mcp/toolmanager.go:207` — read lock released before `CallTool`; client can be closed mid-call. (kimi #22.)
- **M9 combos UI truncation** `ui/src/pages/CombosPage.tsx:46` — edit form rebuilds `steps=[oneStep]`, silently truncates multi-step combos on save. Verified. (dashboard H2.)
- **M10 hardcoded endpoint URL** `ui/src/pages/EndpointPage.tsx:91` — `http://127.0.0.1:8080` literal breaks non-8080 deploys; use `window.location.origin`. (dashboard H1.)
- **M11 operational log fields null** `api/server.go:198-216` — `RTKBytesSaved`, `ComboName`, `ClientTool` never populated. (backend.)
- **M12 SQL identifier concat** `internal/store/sqlite.go:243` — `ensureColumn` concatenates identifiers; whitelist them. (kimi #23, currently hardcoded callers.)
- **L1** usage log + flush errors swallowed (`api/server.go:216`, `inference.go:66`).
- **L2** settings fetched from DB every request, no cache (`api/server.go:313`).
- **L3** MCP refresh token passed into subprocess env (mcp L3).
- **L4** SSE MCP client no reconnect (mcp M4); RFC 8414 path-variant metadata not tried (mcp M3).

# OMP / 9Router / Bifrost Parity Matrix (summary)

`internal/provider/matrix.go:42-86` = **43 entries**. Honest self-labeling (`auth_only` stubs gated out of routing at `engine.go:348-354`).
- **Fully functional** (auth+models+dispatch+stream): ~30.
- **Native-API providers**: anthropic, gemini, vertex, openai, bedrock, ollama-cloud, replicate, xiaomi, gitlab-duo. The other **24 share one `openaicompat` shell** (`internal/providers/openaicompat/provider.go:96`) — lowest-common-denominator, not full native capability.
- **Partial (6)**: missing stream and/or model listing.
- **Stub/auth-only (5)**: zero dispatch.
- **Worst offenders**: replicate (stream+ListModels stub), bedrock (no stream), cloudflare-ai-gateway (dead w/o account_id), opencode/kilo (dispatch-only no discovery), alibaba/minimax/qianfan/zhipu (OAuth login, no refresh → silent expiry).
- **Padding**: kagi + tavily are MCP-search, not inference (`matrix.go:75,83`).
- **Fragility**: matrix + `providerQualifiedDynamicRoute` allowlist (`engine.go:577-597`) must stay in sync or a provider is silently non-routable.

Brand columns `OMP ID / 9Router ID / Bifrost ID` in PROVIDERS.md are vestigial (identical to g0router ID for all 43 rows) → delete.

# Dashboard Coverage Review

19 pages. **All 40 UI→backend endpoint mappings verified correct — zero broken contracts, zero inert buttons.** 13 pages have dedicated unit tests; 6 are thin wrappers under parent tests; 2 e2e suites. Issues: M9 (combos truncation), M10 (hardcoded URL), DataDir editable free-text with no guard (`SettingsPage.tsx:154-162`), LogsPage/DiagnosticsPage bypass `api.ts` with inline URLs, McpPage local-duplicated OAuth type (contract-drift risk).

# API Integration Coverage Review

Public `/v1/*` (chat/completions, messages, models) + protected `/api/*` (connections, keys, aliases, combos, quotas, pricing, usage, logs, settings, mcp, oauth) all implemented and integration-tested. Gaps: SCHEMA.md mismatches (M4), `/api/oauth/:provider/exchange` undocumented, error-leak sites (B6).

# MCP Coverage Review

Transports stdio / SSE / Streamable-HTTP all implement initialize + tools/list + tools/call correctly. OAuth: discovery, PKCE, state CSRF, code exchange, refresh, labels, rehydration — all present. Gaps: cancellation absent everywhere (M6), stdio read-under-lock deadlock (M7), call-after-close (M8), no SSE reconnect, RFC 8414 path variants, refresh token in subprocess env.

# Security And Secret Review

No Critical. gitleaks clean (420 commits, 7.86 MB). OAuth state+PKCE correct. Highs: error-leakage (B6), MCP OAuth flow expiry not checked at store layer (`internal/store/mcpoauth.go:67` vs `oauthsessions.go:50`), auth-exempt OAuth callback paths undocumented (`api/middleware.go:100`). Note: control plane defaults — confirm `RequireAPIKey` default and bind address before exposing.

# Gate Results

| Gate | Result |
|---|---|
| `gitleaks detect` | PASS — no leaks, 420 commits |
| `go vet ./...` | PASS |
| `go build ./cmd/g0router` | PASS |
| `go test ./... -count=1` | PASS — 852 tests, 36 pkgs |
| `npm ui test` | PASS — 97 tests, 20 files (via make verify; standalone bash run flaked on fnm/worker-fork under parallel load — env artifact, not a real failure) |
| `npm ui build` | PASS |
| `npm ui e2e` | PASS — 24 tests, 23 passed, 1 skipped |
| `make build` | PASS |
| `make verify` | PASS (exit 0; runs go+ui+e2e+git diff) |
| `git diff --check` | PASS |

All deterministic gates pass. (Note: a green gate suite with the B-class correctness bugs present = the tests don't cover those paths — see remediation.)

# Suggested Remediation Plan

- **Wave R1 — Brand removal** (docs+code+UI). Files: PROVIDERS.md (drop 3 columns), REFERENCES.md (tombstone), ARCHITECTURE/PLAN/DIRECTORY_STRUCTURE/README/AGENTS/CLAUDE.md, `internal/provider/matrix.go`+`matrix_test.go`, `api/handlers/providers.go`, ui api/tests/e2e, phase docs. Tests-first: matrix_test asserting no legacy IDs. Gate: `go test ./... && npm test`.
- **Wave R2 — Anthropic streaming tool calls** (B1). Golden SSE test first. Gate: new test + `make verify`.
- **Wave R3 — Refresh race + backoff timing** (B2,B3). `go test -race` first. Gate: `-race` green.
- **Wave R4 — `/v1/models` resilience + error redaction** (B5,B6). Handler tests first.
- **Wave R5 — Provider full-usage / matrix honesty** (B4): implement bedrock+replicate native streaming or correct capability flags; cloudflare validation. Capability test first.
- **Wave R6 — Docs truth pass** (phantom files/packages, 23→43, SCHEMA.md fields).

Each wave: TDD red→green, commit `phase-R/task-N`, run `make verify` after, then re-run this audit's gate set.

# Final Release Recommendation

**Release blocked.** Conditionally approvable once B1–B6 land and brand references are stripped. No security Critical; the blockers are correctness, false-advertising parity, and unmet "OpenAI+Anthropic full API" goal (Anthropic streaming tool calls).
