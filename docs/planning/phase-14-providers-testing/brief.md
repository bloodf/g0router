# Brief

**Problem:** g0router lacks provider detail APIs, live model testing, encrypted proxy pools, and disabled/custom model management. Operators cannot inspect a provider, test a model, route outbound traffic through a proxy, or curate the advertised model set.

**Success criteria:**
- Proxy pools CRUD with `password_enc` encrypted at rest; list/get never return plaintext; outbound provider requests use the assigned pool.
- Disabled models absent from `/v1/models`, `/api/models`, and routing candidates; custom models appear in listings.
- Provider detail/connections/suggested-models + single and batch (SSE) model-test endpoints return `{ok, latency_ms, error}` and never 500 on upstream failure.
- Per-phase gate green: `go test -race`, coverage ≥ 95.0%, audit rows for every mutating endpoint.

**Non-goals:**
- No UI work (lands in phases 20-21).
- No feature-flag-gated features (semantic_cache, guardrails, etc.).
- No migration of existing endpoint shapes (Phase 21).

**Constraints:** snake_case JSON + `{data, error}` envelope; secrets encrypted via `*_enc` columns (oauthsessions.go pattern); additive migrations only; DDD-lite layering; direct push to main. fasthttp proxy caveat — inspect `internal/providers/utils` client construction before wiring.

**Verification:** `go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && go test -race ./...` all green with coverage ≥ 95.0%.

**QA criteria:**
```yaml
qa_skip: null
scenarios:
  - id: 1
    description: Proxy pool create/get omits plaintext password; round-trips encrypted
    method: api
    evidence: POST /api/proxy-pools then GET /api/proxy-pools/:id — response has no password field
  - id: 2
    description: Disabled model absent from /v1/models and rejected by routing with clear error
    method: api
    evidence: POST /api/models/disabled then GET /v1/models (model absent) + routing call returns 400/error not 500
  - id: 3
    description: Single model test on upstream failure returns {ok:false, error}, never 500
    method: api
    evidence: POST /api/providers/:id/models/:model/test against dead upstream — HTTP 200, body ok:false
manual_smoke: curl proxy-pools CRUD + model-test + disabled-models list against a local g0router binary; confirm envelopes and no plaintext secrets.
```

**Linked artifacts:** architect-plan: ./architect-plan.md; orchestration: ./orchestration.jsonl
