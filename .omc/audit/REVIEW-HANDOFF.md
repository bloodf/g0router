# g0router — Independent Release-Readiness Review (Agent-Team Handoff)

You are an AI engineering orchestrator with a team of specialist subagents. Independently re-audit the **g0router** LLM gateway and decide if it is releasable. Assume nothing is correct until source, tests, and runtime behavior prove it. Do not trust this handoff's claims — verify them.

## Repository
- GitHub: `git@github.com:bloodf/g0router.git`
- Branch: `main`
- Expected HEAD at review time: `a3b8fd0` (confirm with `git rev-parse HEAD`; if newer, review the actual tip and note drift)
- Go backend in `api/ internal/ cmd/`; React/TS dashboard in `ui/`; single embedded binary (`embed.go` embeds `ui/dist`).

## Mode
Start **read-only**. No edits, no commits, no pushes, no live provider calls, no secrets printed or requested. If optional live checks are run, require credentials from environment variables only. Act as a grumpy, picky principal engineer: source evidence beats docs and beats this handoff. Every finding needs `file:line`.

## What changed recently (verify each claim against source — do NOT assume true)
This repo just completed a large remediation + feature pass. Confirm these landed correctly and introduced no regressions:

1. **Brand removal (R1/R6)** — legacy upstream brand strings removed from source, UI, and shipping docs. Verify none remain outside `docs/evaluations/` (historical audit trail).
2. **OpenAI/Anthropic compatibility (R2/R9/R15)** — `/v1/chat/completions` and `/v1/messages` both ingress and egress. Streaming **tool calls** translated in Anthropic egress; **tool ids** preserved on ingress; **tool definitions + tool_choice** translated on ingress (was 501). Verify the full agentic loop (define tools → `tool_use` → `tool_result`) survives both formats, streaming and non-streaming.
3. **Concurrency fixes (R3/R5/R8)** — RWMutex on `refreshers`/`quotaFetchers`/`providerPool`; refresh single-flight ordering + panic recovery; streaming backoff now records success only after first chunk; **`requestContext` no longer leaks the pooled `*fasthttp.RequestCtx`** (use-after-recycle race — confirm under `-race`); MCP stdio deadlock/cancellation/call-after-close/session-teardown.
4. **Security/hardening (R4/R10)** — internal errors redacted from client responses (~38 sites); SQL identifier whitelist in `ensureColumn`; MCP OAuth flow expiry; settings cache.
5. **Usage/cost (R7)** — negative-input-cost clamp; `cache_write_tokens` plumbed.
6. **UI (R11)** — multi-step combos (was truncating); origin-relative endpoint URLs.
7. **Deploy (R14)** — container binds `0.0.0.0` (`Dockerfile` `ENV BIND_ADDRESS`); binary defaults to `127.0.0.1` (secure local).
8. **Full request-logging system (Wave L)** — THE NEWEST, SCRUTINIZE HARDEST:
   - Retention setting `log_retention_days` (presets 5/15/30/60/90/180, 0=forever, custom) in `internal/store/settings.go`, surfaced via `/api/settings` (snake_case JSON).
   - Hourly background cleanup goroutine: `Server.StartLogRetention` / `runLogRetentionOnce` / `store.DeleteRequestLogsOlderThan` — verify it deletes the RIGHT rows, races nothing under `-race`, stops on ctx cancel, and respects retention=0.
   - `GET /api/logs` rich query: `provider/model/auth_type/source_format/status_class/search/start/end/limit/offset` + `total`. Verify **all queries are parameterized** (no SQL injection via `search`/filters), `status_class` validation, pagination/`total` correctness, and `limit` clamping.
   - Operational fields `client_tool` / `rtk_bytes_saved` / `combo_name` populated at the log site WITHOUT touching the recycled fasthttp ctx (snapshot pattern) — verify no new use-after-recycle.
   - UI: `ui/src/pages/LogsPage.tsx` (filters/search/pagination/detail) + `SettingsPage.tsx` retention control. Verify UI↔backend contract (snake_case fields, `total`, query params) matches actual handlers.

## Known/disclosed residuals (confirm these are acceptable, or escalate if worse than stated)
- "Kinds of logs" = HTTP status-class filter over **inference request logs only**; no separate MCP/access/system log streams.
- `bedrock`/`replicate` advertised as non-streaming (matrix flag matches adapter); native event-stream not implemented (needs live creds).
- Coverage 95.0% total; some functions <95% are irreducible driver-fault/marshal-error wraps, `os.Exit` main, and the serve-forever loop. Project rule: **no mocks** — fakes/httptest only.
- Live provider calls not exercised (paths covered by fake upstreams).

## Agent Team + Orchestration

Spawn the following specialist subagents. Run **read-only review agents in parallel** (disjoint scopes, each writes findings to `.omc/review/<area>.md` and returns a ≤12-line summary). Then run a **synthesis/merge pass**, then **deterministic gates**, then an **adversarial verification pass** on every Critical/High before it goes in the final report.

### Wave 1 — Parallel specialist review (read-only, fan-out)
Spawn these concurrently; each must produce `file:line` evidence and classify Critical/High/Medium/Low, and explicitly call out areas where NO issue was found:

1. **logging-reviewer** — `internal/store/usage.go` (filters/`CountUsage`/`DeleteRequestLogsOlderThan`/whereClause), `api/server.go` retention ticker + M11 population, `api/handlers/{logging,usage,settings}.go`. Hunt: SQL injection in search/filters, pagination/total off-by-one, retention deleting wrong rows or racing, ctx-after-recycle in M11, settings contract drift.
2. **concurrency-reviewer** — run the suite under `-race`; audit `internal/proxy` (maps, fallback rotation, backoff, refresh single-flight), `internal/mcp` (transport mutexes, cancellation, toolmanager refcount), `api/server.go` (fasthttp ctx, streaming snapshot, cleanup goroutine). Hunt: races, deadlocks, goroutine/body leaks.
3. **compat-reviewer** — `api/handlers/inference.go` + `internal/providers/*` + `internal/translate`. Hunt: OpenAI/Anthropic ingress+egress correctness, streaming + tool-call translation gaps, tool id loss, usage extraction, broken SSE.
4. **security-reviewer** — auth on `/api/*` and `/v1/*`, error/secret redaction, OAuth state/PKCE, token storage, CORS, bind defaults, `ensureColumn` whitelist, MCP OAuth expiry. Run `gitleaks detect --source . --redact` (incl. history).
5. **dashboard-reviewer** — every `ui/src/pages/*` against `ui/src/api.ts` and the Go handlers. Hunt: contract drift (esp. new logs/settings snake_case), inert controls, missing states, the LogsPage filter/pagination/search wiring, retention selector persistence.
6. **provider-parity-reviewer** — `internal/provider/matrix.go` vs adapters: advertised capability flags vs real auth/list/dispatch/stream/refresh. Hunt: dishonest matrix entries, stub-but-advertised providers, dynamic-route allowlist drift.
7. **test-coverage-reviewer** — measure coverage (`go test ./... -coverprofile`), find under-tested critical paths, weak/tautological tests, and any test that asserts buggy behavior. Verify the new logging code is genuinely covered, not just line-touched.

### Wave 2 — Synthesis
Spawn **synthesis-merger**: dedupe Wave 1 findings, resolve disagreements, rank by severity, and produce a single consolidated list with a coverage/parity matrix.

### Wave 3 — Deterministic gates (must all pass; any failure ⇒ FAIL)
Run and capture exact output:
```
gitleaks detect --source . --redact --verbose
go vet ./...
go build ./cmd/g0router
go test ./... -race -count=1
go test ./... -coverprofile=cover.out && go tool cover -func=cover.out | tail -1   # expect ~95%
npm --prefix ui test -- --run
npm --prefix ui run build
npm --prefix ui run e2e
make verify
make e2e-binary            # opt-in real-binary smoke (builds binary, mints key, hits /healthz /v1/models /api/*)
git diff --check
```
Optional deployment check (OrbStack/Docker): `docker build -t g0router:review . && docker run -d -e API_KEY_SECRET=x -p 28131:20128 g0router:review`, then curl `/healthz` (expect 200) and confirm reachability (image must bind 0.0.0.0).

### Wave 4 — Adversarial verification
For EVERY Critical/High candidate, spawn 2-3 independent **verifier** agents prompted to REFUTE the finding (write a failing test or trace the code path). Keep only findings that survive majority refutation attempts. This kills plausible-but-wrong findings before they reach the report.

### Orchestration rules
- Review agents are read-only; only the gate runner executes commands; nobody edits source in the review.
- Scope agents to disjoint packages; have each run only its own package tests to avoid shared-tree `go test` collisions; the orchestrator runs the full suite once in Wave 3.
- Each agent writes its report to `.omc/review/<area>.md` and returns a compact summary so the orchestrator's context stays lean.
- Merge into ONE consolidated report.

## Required Output

```
# Verdict
PASS or FAIL for release.

# Executive Summary
Blunt: can this ship?

# Blocking Findings (Critical + High)
Ordered by severity. Each: severity, evidence (file:line), why it matters, minimal fix, suggested verification. Only findings that survived adversarial verification.

# Non-Blocking Findings (Medium + Low)

# Logging System Review
Retention correctness, cleanup-job safety, query parameterization/injection, pagination/total, M11 population, UI↔backend contract — tested/untested per item.

# OpenAI/Anthropic Compatibility Matrix
Ingress + egress, streaming + non-streaming, tool calls + tool definitions — status + evidence.

# Provider Parity Matrix
provider | auth | models | dispatch | stream | refresh | status | evidence.

# Dashboard Coverage
Every page + action: wired/tested status.

# Security & Secret Review
Auth, redaction, CORS, bind defaults, tokens, gitleaks (incl. history).

# Gate Results
Exact command output. Any failure/skip ⇒ FAIL.

# Coverage Report
Total + per-package; under-tested critical paths.

# Final Release Recommendation
Exactly one: "Release approved" / "Release blocked" / "Release conditionally approved with named caveats".
```

## Cross-checks for THIS review (don't just rubber-stamp)
- Run `go test ./... -race` yourself — the prior audit found 2 use-after-recycle **Criticals only visible under `-race`**. Confirm none remain.
- Try to SQL-inject the log `search`/filter params (e.g. `search=' OR 1=1 --`) against a running binary; confirm parameterized.
- Confirm retention=0 deletes nothing and retention=N deletes only older-than-N rows; confirm the cleanup goroutine exits on shutdown (no leaked goroutine).
- Confirm `/api/logs` `total` is correct across pages and matches `CountUsage` independent of `limit/offset`.
- Confirm the settings JSON is snake_case end-to-end and the UI sends/reads matching keys.
- Confirm no secret (API keys, OAuth `ClientSecret`, tokens) appears in any client response or server log line.
