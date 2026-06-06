# g0router — Session Continuation Handoff

You are continuing autonomous work on **g0router** (Go LLM gateway/proxy, single embedded binary; Go in `api/ internal/ cmd/`, React/TS dashboard in `ui/`). A long prior session did a full audit + remediation + many features. Context was reset for a clean slate. Pick up exactly here.

## Repo / state
- Path: `/Users/heitor/Developer/github.com/bloodf/g0router` · branch `main` · remote `git@github.com:bloodf/g0router.git`
- **HEAD = `d8b08f5`** (confirm `git rev-parse HEAD`; everything below is committed + pushed; working tree clean except `.DS_Store`/`.omc`/`.pi` noise).
- Go **~2658 tests**, `-race` clean, `go vet` clean, build clean. **Coverage 94.6% (statements)** — the standing goal is **≥95%**, so coverage is the first debt to clear.
- UI: vitest + Playwright (~33 e2e) green as of the last UI wave, BUT the newest backend features (this session's Round 1–5) have **NO UI yet** — see Remaining.

## Active goal (do not stop until met)
Drive autonomously until the project is 100% working: all features wired end-to-end (backend AND dashboard), all bugs fixed, **≥95% Go unit coverage**, everything e2e-testable tested + green, `make verify` + `make e2e-binary` + gitleaks + OrbStack container smoke all pass. The user deploys **only local docker / systemd / VPS** (never Railway/cloud-proxy — so trusted-proxy/XFF is out of scope).

## House rules (follow exactly)
- **TDD**: test first (see it fail), then implement. **No mocks** — use interfaces, fakes, `httptest`, temp sqlite. No `init()`, errors wrapped `%w`, no global state, constructors for deps.
- `go test ./...` and `go vet ./...` green at every commit. Commit per green slice; messages like `feat(scope): …` / `fix(scope): …` (no AI attribution, write as the user would).
- **Run `-race`** — two real Criticals this project only surfaced under `-race`.
- Settings JSON is **snake_case** end-to-end; UI must match the Go handler shapes exactly (read `api/handlers/*.go`, don't guess).
- **fasthttp pooled-ctx rule**: never read `*fasthttp.RequestCtx` off the request goroutine / after handler return — snapshot needed values first. `requestContext` is deliberately detached to `context.Background()`.
- Proxy `/v1/*` **always requires a valid API key** (independent of the `RequireAPIKey` toggle, which gates `/api/*`). Source policy (`allowed_sources`: local/lan/tailscale/public) is enforced on client IP.
- **Orchestration that works here**: split work into **file-disjoint** waves, run parallel agents each scoped to its own packages with **package-scoped test commands only** (NOT `go test ./...` — concurrent full-suite runs collide on the shared tree). After agents return, YOU run the full `go build/vet/test -race` + coverage, fix fallout, then commit. `api/server.go` and `api/middleware.go` are hot files — only ONE wave may edit them per round; serialize those.
- npm/vitest under parallel load flakes with "worker forks emitted error" (fnm env) — re-run once; `make verify` is authoritative for UI. Some UI tests flake if they `getByRole` async-loaded rows synchronously — use `await findBy…`.
- Coverage ceilings that are acceptable (don't chase): `json.Marshal` error branches on fixed structs, `crypto/rand` failure, sqlite driver-fault wraps, `os.Exit` main, real-socket `Serve/Stop`, serve-forever loops.

## What already exists (DONE — don't rebuild; wire UI to it)
Backend, all committed:
- **Logging system**: retention (`log_retention_days`) + hourly cleanup; `/api/logs` rich filters (provider/model/auth_type/source_format/status_class/search/start/end/limit/offset + `total`); operational fields populated (`client_tool`, `rtk_bytes_saved`, `combo_name`).
- **Access control**: proxy key mandate; `allowed_sources` policy; usage attribution (`api_key_name`, `connection_name`, `connection_provider`, `account_email` resolved via joins on `/api/usage` + `/api/logs`).
- **Combos**: strategy field — `fallback`, `round_robin`, `least_used`, `auto` (request-task heuristic: vision/tools/code/large→most-capable, else cheapest), **`fastest`** + **`cheapest`** (telemetry-ordered by recent avg latency/cost). API: `strategy` on `/api/combos`.
- **OAuth**: refresh-before-dispatch + token persistence for codex/anthropic/gemini/cursor/copilot/etc (already worked); **proactive background refresh ticker** + **stale notification** via new `internal/notify` (generic/discord/telegram webhook; settings `notify_webhook_url`, `notify_on_reauth`); **needs-reauth flag** on connections (`needs_reauth`, `last_refresh_error`).
- **Per-key policy**: `expires_at`, `scopes` (model globs), `rate_limit_rpm`, `rate_limit_tpm`, `daily_spend_cap_usd` — enforced pre-dispatch (scope→403, RPM→429, spend→402) + recorded post-dispatch via `internal/ratelimit`. On `/api/keys`.
- **Providers**: native streaming for **bedrock** (ConverseStream event-stream decoder) + **replicate** (SSE); matrix flags flipped.
- **Endpoints**: `/v1/embeddings`, `/v1/images/generations`, `/v1/audio/transcriptions`, `/v1/audio/speech` (optional capability interfaces; openai + openaicompat-embeddings; 501 `capability_unsupported` otherwise).
- **Response cache**: `cache_enabled` + `cache_ttl_seconds`; TTL+LRU keyed by canonical request hash; `X-Cache: HIT/MISS`; non-streaming only.
- **Observability**: `internal/metrics` Prometheus `GET /metrics` (management-gated); **audit log** (`audit_log` table, `GET /api/audit`) recording successful admin mutations with actor key id.
- Earlier session: full audit remediation (R1–R15), Anthropic/OpenAI compat incl. streaming + ingress tool definitions/tool_choice, MCP fixes, error redaction, etc.

## Remaining work (do these)
**1. Coverage to ≥95%** (currently 94.6%). Target the newest under-covered code: `internal/cache`, `internal/metrics`, `internal/store/audit.go`, `api/server.go` (cache/metrics/audit wiring), `api/policy.go`, `internal/notify`, multimodal handlers/providers. Add real tests (no mocks). This is the quickest win — do it first or alongside.

**2. Dashboard UI for every new backend feature** (none wired yet). Read the real Go JSON first. Build/extend:
   - **API Keys page**: full policy form — expiry date, model scopes (list), RPM, TPM, daily spend cap; show per-key usage/spend.
   - **Settings**: notification config (`notify_webhook_url`, `notify_on_reauth`); response cache (`cache_enabled`, `cache_ttl_seconds`). (`allowed_sources` + retention already in UI.)
   - **Combos page**: add `fastest` + `cheapest` to the strategy selector (fallback/round_robin/least_used/auto already there).
   - **Audit log page** (new): paginated `/api/audit` viewer (timestamp, actor key, action, target).
   - **Metrics**: surface a link/snippet for `/metrics` (Prometheus scrape) on a Diagnostics/Observability page.
   - **#12 Per-provider health page**: last success, backoff level, `needs_reauth`, quota, recent latency/cost — from connections + telemetry.
   - **#16 Usage charts**: time-series cost/tokens by key/provider/model on the Usage history page.
   - **#15 Auto discoverability**: surface the `auto` combo/strategy as a first-class routable model with a "what it picks" explainer.
   - **#17 One-click re-auth**: from the "Needs re-auth" badge straight into the provider OAuth flow.
   - (Optional) a small **embeddings/audio playground** to exercise the new endpoints.

**3. NEW — live traffic topology view (the big one):** a 9router-style **animated connection graph** showing where the proxy is talking to in real time — client/keys → gateway → providers/models, with **animated flowing lines** on active requests. Needs: a backend live feed of recent/active dispatches (an SSE endpoint e.g. `GET /api/traffic/stream`, or poll recent `/api/logs`), and a UI canvas/SVG force-or-radial graph with animated edges (pulse/particles) keyed to live request volume per provider. Make it performant + degrade gracefully when idle.

**4. Final release gate** (run after the above): `gitleaks detect --source . --redact`, `go vet ./...`, `go test ./... -race`, coverage ≥95%, `make verify` (go+ui+playwright+git-diff), `make e2e-binary`, OrbStack `docker build` + run + `/healthz` 200, `git diff --check`, then push. Update `docs/WORKFLOW.md` with the new waves.

## Suggested next order
(a) coverage→95% + commit. (b) UI wave: keys-policy + settings(notify+cache) + combos(fastest/cheapest) + audit page (file-disjoint within ui/, one agent or a couple). (c) UI wave: health page + usage charts + auto-discoverability + one-click reauth. (d) traffic topology (backend feed wave, then UI viz wave). (e) final gate + push. Verify + commit + push after each wave; keep `make verify` green.

## Notes
- A read-only review handoff for an independent team exists at `.omc/audit/REVIEW-HANDOFF.md`; the consolidated audit at `.omc/audit/CONSOLIDATED-REPORT.md`. `.omc/` is gitignored (local only).
- `kimi -p "<prompt>"` is available for an independent second-opinion code review (slow, agentic) — used twice already; both rounds' findings were triaged + fixed.
- Provider matrix is intentionally honest (advertised capability flags must match adapter behavior; there are consistency tests enforcing it — keep them green when changing capabilities).
