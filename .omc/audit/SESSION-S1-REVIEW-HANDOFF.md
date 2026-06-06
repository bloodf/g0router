# g0router — Session S1 Review Handoff (release-readiness audit)

You are an independent reviewer. Prior session (S1) wired all Round 1–5 backend
features into the dashboard, added a live traffic topology view, and restored the
≥95% Go coverage gate. **Your job: verify it is correct, complete, matches the
docs/spec, and is release-ready. Do not assume the prior agent was right — re-check.**

---

## Repo state

- Path: `/Users/heitor/Developer/github.com/bloodf/g0router` · branch `main` · synced with `origin/main`.
- HEAD: `eeebb3c`. Session range: `d8b08f5..eeebb3c` (8 commits).
- Tree clean (only gitignored `.DS_Store`/`.omc`/`.pi` untracked).
- Diffstat: **31 files, +3346 / −46**.

### Commits (oldest→newest)
```
cd25a28 test: cover multimodal handlers, metrics, cache, audit, and server helpers
c3747e1 feat(ui): per-key policy form, notify+cache settings, combo strategies, audit page
9ad684d feat(ui): provider health page, usage summary chart, strategy hints, one-click reauth
1aaf194 feat(traffic): live traffic topology with SSE feed and animated graph
9cdd91d feat(ui): daily usage time-series chart on usage page
cb5ef6c test(traffic): cover nil-broker guard, shutdown path, and empty ring replay
6c58a61 fix(ui): render real API-key fields and key the policy row fragment
eeebb3c docs(workflow): record session S1 ...
```

### New files
- Go tests: `api/coverage_server_pure_test.go`, `api/handlers/coverage_extra_handlers_test.go`, `api/server_traffic_test.go`, `internal/cache/coverage_cache_test.go`, `internal/metrics/coverage_metrics_test.go`, `internal/store/coverage_audit_test.go`
- Go src: `internal/traffic/broker.go` (+ `broker_test.go`)
- UI: `ui/src/pages/{AuditPage,HealthPage,TrafficPage}.tsx` (+ `.test.tsx`)

---

## What was built, by goal (verify each against the spec in the original handoff)

### (a) Coverage → 95.0% (was 94.6%)
Added real tests (no mocks) for: multimodal handlers (embeddings/images/audio/speech
guards), `statusClassFor`/`parseCacheableRequest`/`handleMetrics`, metrics nil-receiver
guards + `sortedRequestKeys` tie-breakers, cache nil-clock + zero-capacity, audit
`clampAuditLimit` bounds + closed-store error paths, catalog unknown-provider, traffic broker.
- **Check:** `go test ./... -count=1 -coverprofile` → total **95.0%**. Confirm it still reads ≥95.0%.

### (b) UI wave b — `c3747e1`
- **APIKeysPage / EndpointPage**: per-key policy editor (expires_at, scopes, rate_limit_rpm,
  rate_limit_tpm, daily_spend_cap_usd) via `PUT /api/keys/{id}`.
- **SettingsPage**: `notify_webhook_url`, `notify_on_reauth`, `cache_enabled`, `cache_ttl_seconds`.
- **CombosPage**: added `fastest` + `cheapest` strategies.
- **AuditPage (new)**: `GET /api/audit` → `{object,data[],limit,offset,total}`.

### (c) UI wave c — `9ad684d` + `9cdd91d`
- **HealthPage (new)**: per-provider/connection health from `/api/connections` + `/api/providers`
  (needs_reauth, backoff_level, unavailable_until, last_refresh_error, expires_at).
- **UsagePage**: summary panel (`/api/usage/summary` = `{request_count,total_tokens,total_cost_usd}`,
  aggregates only — **no time buckets in backend**) **plus** a client-side daily time-series chart
  bucketed from the loaded usage log records.
- **CombosPage**: strategy discoverability hint text.
- **ProvidersPage**: one-click "Re-authenticate" on needs_reauth connections, reusing the existing
  OAuth authorize→poll→exchange flow (no new endpoint).

### (d) Traffic topology — `1aaf194` + `cb5ef6c`
- **`internal/traffic`**: `Broker` with fixed-size ring buffer, non-blocking `Publish` (drops to slow
  subscribers — never blocks the request path), `Subscribe`/`Unsubscribe`, `Recent()` replay.
  `Event{timestamp,key_id,provider,model,status_class,status_code,latency_ms}` (snake_case).
- **`api/server.go`**: `Server.trafficBroker` constructed in `NewServer(256)`; publishes an Event from
  `observeRequestMetric` (key id = `metadata.apiKeyID`); new SSE route `GET /api/traffic/stream`
  (`handleTrafficStream`) mirroring `streamInference` — `: connected` flush, `Recent()` replay, select on
  subscriber channel + 15s heartbeat (`: ping`) + `stopCh`; `Stop()` closes `stopCh` via `sync.Once`
  before fasthttp shutdown.
- **`ui/src/pages/TrafficPage.tsx`**: hand-rolled SVG topology (keys→gateway→providers), edges pulse +
  thicken with live volume; consumes SSE via **authed `fetch` + `ReadableStream` reader** (NOT
  EventSource, because EventSource cannot send the `Authorization` header); idle "Waiting…" degrade.

### Bug fixes — `6c58a61` (found by the real-server e2e smoke)
- **API-key casing mismatch**: Go `apiKeyView` returns snake_case (`id/name/prefix/is_active/
  last_used_at/created_at`) for BOTH list and create, but the TS `APIKeyResponse` + dashboard read
  PascalCase (`ID/Name/Prefix/...`). Mocked tests passed (fixtures use PascalCase); only the
  real-server chromium smoke exposed it (created key rendered an empty name). **Fix: a casing-tolerant
  `normalizeAPIKey` at the `api.ts` boundary** (prefers snake, falls back to Pascal) applied in
  `listAPIKeys` / `createAPIKey` / `updateAPIKeyPolicy`.
- React list-key warning: keyed the keys-table policy-row `Fragment`.

---

## Verification gates (prior session claims — RE-RUN and confirm)

| Gate | Claimed |
|---|---|
| `gitleaks detect` | no leaks (473 commits) |
| `go vet ./...` | clean |
| `go test ./... -count=1` | 2701 pass; coverage **95.0%** |
| `go test -race ./api/ ./internal/traffic/` | clean |
| `go test -tags e2ebin -run TestE2EBinary` | pass |
| `npm --prefix ui test` | 147 pass (23 files) |
| `npm --prefix ui run build` | clean |
| `npm --prefix ui run e2e` | 33 pass, 1 skipped, 0 fail, 0 key warnings |
| docker build + run + `/healthz` (OrbStack) | `200 {"status":"ok","version":"0.1.0-dev"}` |
| `git diff --check` | clean |

Authoritative one-shot: `make verify` (does go test/vet/build + npm test/build/e2e + git diff --check).
Plus `make e2e-binary`. NOTE: `internal/mcp` tests hit the network and flaked once at a 600s timeout
under coverage instrumentation; a clean re-run passed. Re-run mcp alone if it times out.

Docker runtime note: container **requires `API_KEY_SECRET`** env (REQUIRE_API_KEY defaults true) or it
exits 1 with `API_KEY_SECRET required when REQUIRE_API_KEY=true`. Smoke command:
`docker run -d -e API_KEY_SECRET=<secret> -p 20191:20128 g0router:<tag> serve` then curl `/healthz`.

---

## Highest-scrutiny items (look here first)

1. **`normalizeAPIKey` (`ui/src/api.ts`)** — the casing-tolerant adapter. Is the boundary the right
   place, or should the Go `apiKeyView`/TS type be made consistently snake_case end-to-end (per the
   CLAUDE.md rule "Settings JSON snake_case end-to-end; UI reads real Go handler shapes")? The current
   fix is minimal-blast-radius but papers over a Go/TS contract inconsistency. Decide if that debt is
   acceptable for release or should be unified. Also confirm no OTHER API-key consumer (DashboardPage,
   UsagePage `api_key_*`) still reads a field the real server doesn't send.
2. **SSE pooled-ctx safety** — `handleTrafficStream` runs in `SetBodyStreamWriter`. Confirm it never
   reads the pooled `*fasthttp.RequestCtx` off-goroutine/after return (it snapshots the broker ref and
   only writes). Confirm `Publish` in `observeRequestMetric` snapshots all values before any goroutine.
   Run `go test -race ./api/` and ideally a race run with an active SSE client + concurrent inference.
3. **Broker back-pressure** — verify `Publish` truly never blocks the request hot path (drops on full
   subscriber channel) and that a disconnecting SSE client is unsubscribed (no goroutine/subscriber leak).
4. **Auth gating of new routes** — `/api/traffic/stream` and `/api/audit` must obey the same
   `RequireAPIKey` + `allowed_sources` gating as other `/api/*` routes (they route through `handleAPI`).
   Confirm an unauthenticated request is rejected.
5. **Usage time-series** — it buckets only the records the page already loaded; if the logs endpoint is
   capped/paginated the chart silently covers a partial window. Spec wanted "cost/tokens time-series";
   confirm this is acceptable or whether a backend bucketed endpoint is required.
6. **Coverage ceilings (intentionally uncovered)** — marshal-error branches (`writeError`,
   `writeOpenAIError`, `Health`, `writePolicyError`), SSE write/flush-error + 15s-heartbeat branches in
   `handleTrafficStream`/`streamInference`, sqlite driver-fault wraps, `os.Exit` main, real-socket
   Serve/Stop, crypto/rand. Confirm none of these hide a real reachable bug.

## Review checklist
- [ ] Re-run `make verify` + `make e2e-binary` clean; coverage still ≥95.0%.
- [ ] `go test -race ./...` (tolerate mcp network flake; re-run if needed).
- [ ] Every new UI page wired into `App.tsx` nav and reachable; states (loading/empty/error/auth-expired) present.
- [ ] New `/api/*` routes auth-gated; `/healthz` + `/metrics` still pre-auth.
- [ ] No secrets in diff; `gitleaks` clean.
- [ ] Docker image builds, runs with `API_KEY_SECRET`, serves `/healthz` and the embedded dashboard.
- [ ] `docs/WORKFLOW.md` S1 entry matches reality; no stale TODO/spec item left unwired.
- [ ] Decide on the `normalizeAPIKey` contract debt (item 1) before tagging a release.

## Out of scope (per original handoff)
Railway/cloud-proxy + trusted-proxy support (deploy targets are local docker / systemd / VPS only).
