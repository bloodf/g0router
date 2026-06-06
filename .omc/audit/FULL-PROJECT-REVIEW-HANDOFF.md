# g0router — Full Project Review & Release-Readiness Audit

You are an independent senior reviewer. **Audit the ENTIRE project — not just recent
changes.** Find gaps, bugs, dead ends, spec drift, and anything that is not fully
working. Assume nothing is correct until you have verified it yourself. Goal: a
defensible PASS/FAIL verdict on whether g0router is ready to release.

---

## What g0router is

Single-binary Go LLM gateway/proxy. ~28k LOC Go across **41 packages**, **16 provider
adapters** (anthropic, azure, bedrock, cloudflare, cohere, gemini, gitlabduo, mistral,
ollama, ollamacloud, openai, openaicompat, replicate, vertex, xiaomi + utils), OAuth
flows, RTK compression, MCP gateway, Prometheus `/metrics`, admin audit log, response
cache, and an embedded **React 19 + Tailwind 4** dashboard (**22 pages**). CLI + Web UI
control plane. **33 API routes** in `api/server.go`. ~2477 Go test funcs.

## Repo state
- Path: `/Users/heitor/Developer/github.com/bloodf/g0router` · branch `main` · synced with `origin/main` · HEAD `eeebb3c`.
- Tree clean (only gitignored `.DS_Store`/`.omc`/`.pi`).

## Where to orient yourself (read these first)
- `CLAUDE.md` — behavioral + project rules (TDD, no mocks, no `init()`, errors as values, no globals, snake_case JSON end-to-end, surgical changes).
- `docs/README.md` — doc index. Then: `docs/ARCHITECTURE.md`, `docs/CONFIG.md`, `docs/DEPLOYMENT.md`, `docs/DIRECTORY_STRUCTURE.md`, `docs/PROVIDERS.md`, `docs/SCHEMA.md`, `docs/PLAN.md`, `docs/WORKFLOW.md` (large — wave history + current state at top).
- Deploy targets are **local docker / systemd / VPS only** — Railway/cloud-proxy and trusted-proxy support are explicitly **out of scope**; do not flag their absence.

---

## Audit scope — cover ALL of it

### 1. Build, test, and gates (run them, read the output)
- `make verify` (authoritative: `go test ./...` + `go vet` + `go build` + `npm test` + `npm build` + `npm e2e` + `git diff --check`).
- `make e2e-binary` (`go test -tags e2ebin -run TestE2EBinary`).
- `go test ./... -count=1 -coverprofile=/tmp/c.out && go tool cover -func` — confirm total **≥95.0%** and inspect per-package gaps. The bar is 95%; verify it holds.
- `go test -race ./...` — concurrency is the known risk area (history: 2 critical races only surfaced under `-race`). Tolerate `internal/mcp` network flakes (it has timed out at ~600s under coverage load; re-run alone if needed).
- `gitleaks detect --no-banner --redact` — must be clean.
- `go vet ./...` clean; consider `staticcheck`/`golangci-lint` if available for deeper lint.
- Docker (OrbStack): `docker build -t g0router:audit .` then
  `docker run -d -e API_KEY_SECRET=<secret> -p 20191:20128 g0router:audit serve` →
  curl `/healthz` (expect `200 {"status":"ok"}`), `/metrics`, and load the embedded dashboard.
  Container **requires `API_KEY_SECRET`** (REQUIRE_API_KEY defaults true) or exits 1.

### 2. Correctness & bugs (hunt for them)
- **Request path**: routing, middleware order, auth (`RequireAPIKey`), `allowed_sources`
  (local/lan/tailscale/public) enforcement on client IP, per-key policy enforcement
  (expiry/scopes/rate_limit_rpm/rate_limit_tpm/daily_spend_cap_usd → 401/403/429/402).
- **fasthttp pooled-ctx safety (high priority)**: handlers must never read the pooled
  `*fasthttp.RequestCtx` off-goroutine or after the handler returns. Verify every streaming
  path (`streamInference`, `handleTrafficStream`) and every goroutine snapshots values first.
  `requestContext()` detaches to `context.Background()` by design — confirm that holds everywhere.
- **Streaming**: SSE for inference + traffic; disconnect cancellation; tool-call accumulation;
  bedrock/replicate native streaming. Confirm clients can't hang the server or leak goroutines.
- **Providers**: capability flags in the provider matrix must match adapter behavior
  (consistency tests enforce this — confirm they pass and actually cover each adapter).
  Check error redaction (no upstream secrets/internal detail leaking to clients).
- **Stores**: sqlite migrations, audit log, usage logging/retention, combos, aliases, pricing,
  api keys, connections. Look for unparameterized SQL, missing error wraps, partial writes.
- **OAuth**: refresh + proactive ticker + stale `needs_reauth` flag + notify webhook;
  re-auth flow end-to-end.
- **Cache**: response cache (cache_enabled/cache_ttl_seconds, `X-Cache`), key canonicalization,
  never caches streaming, eviction correctness.
- **Concurrency**: traffic `Broker` non-blocking publish (must never block the request hot path),
  subscriber cleanup on disconnect (no leak), ratelimit limiter, settings cache, no global state.

### 3. Frontend (all 22 pages)
- Every page reachable from `App.tsx` nav; loading/empty/error/auth-expired states present.
- **UI reads the REAL Go handler JSON shapes — verify, don't trust the TS types.** Known
  inconsistency already found+patched: API keys come back snake_case (`id/name/prefix/
  is_active/last_used_at/created_at`) but the dashboard model is PascalCase; a tolerant
  `normalizeAPIKey` adapter in `ui/src/api.ts` bridges it. **Scrutinize whether other pages
  have the same Go/TS contract drift** (settings, connections, usage, combos, mcp, pricing).
  Decide if the adapter approach is acceptable for release or should be unified to snake_case
  end-to-end per CLAUDE.md.
- New pages this milestone: AuditPage, HealthPage, TrafficPage — verify they match their backends
  (`/api/audit`, `/api/connections`+`/api/providers`, `/api/traffic/stream`).
- `npm test` (vitest) + `npm e2e` (Playwright) green; note the `real-server.e2e.ts` chromium smoke
  is the only test that hits a real binary — it is the truth source for shape mismatches.
- No `console.log` in production code; immutable update patterns.

### 4. Security
- No hardcoded secrets; `API_KEY_SECRET` required; secrets never logged (audit log is
  caller-sanitized — confirm no request bodies logged).
- Auth/authz on every `/api/*` and `/v1/*` route; `/healthz` + `/metrics` intentionally pre-auth —
  confirm that is the intended exposure and `/metrics` leaks nothing sensitive.
- Input validation at boundaries; SSE endpoints auth-gated; error messages don't leak internals.
- Rate limiting present and enforced.

### 5. Docs & spec alignment
- Cross-check `docs/*` against the actual code: routes in `docs/SCHEMA.md` vs `api/server.go`;
  config keys in `docs/CONFIG.md` vs `internal/config`; provider list in `docs/PROVIDERS.md` vs
  `internal/providers`; deployment steps in `docs/DEPLOYMENT.md` actually work.
- `docs/WORKFLOW.md` top "Current State" claims `project_status: COMPLETE`. Confirm reality
  matches — no TODO/spec item left unwired, no half-built feature.
- Flag any doc that describes behavior the code does not implement (or vice versa).

### 6. Gaps & loose ends
- Dead code, unused exports, orphaned files, TODO/FIXME/XXX/`panic(`/`t.Skip`/`xfail` markers.
- Silent truncation/caps (e.g. usage charts bucket only loaded log records — partial window if
  logs are paginated). Anything that looks "covered" but isn't.
- Error paths that swallow errors; missing context wraps; `init()` functions (should be none);
  global mutable state (should be none).

---

## Known intentional coverage ceilings (don't treat as bugs unless reachable)
marshal-error branches (`writeError`/`writeOpenAIError`/`Health`/`writePolicyError`),
SSE write/flush-error + 15s-heartbeat branches, sqlite driver-fault wraps, `os.Exit` in main,
real-socket `Serve`/`Stop`, crypto/rand. Verify each is genuinely unreachable, not hiding a defect.

## Deliverable
A structured report:
1. **Verdict**: RELEASE-READY / NOT-READY, with the single most important reason.
2. **Findings** by severity (CRITICAL / HIGH / MEDIUM / LOW), each with file:line, evidence
   (test output or code), and a concrete fix.
3. **Gate results table** (every command above: pass/fail + notes).
4. **Spec/doc drift list**.
5. **Open decisions for the owner** (e.g. the `normalizeAPIKey` contract debt; usage-chart windowing).

Be adversarial. If you cannot reproduce a claim, say so. Prefer evidence over assumption.
