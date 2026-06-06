# Stage 13-19 — Execution Process, Contracts & Checkpoints

This document is the single source of truth for HOW phases 13-19 are executed.
Phase docs (`phase-13-*` … `phase-19-*`) define WHAT is built. This doc defines
the process, the cross-cutting contracts every phase must follow, and the
checkpoint protocol between phases.

---

## 1. Cross-Cutting Contracts (apply to EVERY new/changed endpoint)

### 1.1 JSON shape contract
- **All JSON field names are `snake_case`** — request bodies, response bodies, SSE payloads.
- The TypeScript types in `docs/lovable-prompt.md` are the contract. Backend must
  emit exactly those shapes. No PascalCase. No `normalizeAPIKey`-style bridges for
  new endpoints. (Existing endpoints keep their shape until Phase 21 migration.)

### 1.2 Response envelope
- Every `/api/*` JSON response uses the existing envelope:
  ```json
  { "data": <payload>, "error": null }
  ```
  On failure: `{ "data": null, "error": "human-readable message" }` with the
  appropriate HTTP status. Match the pattern in `api/handlers/apikeys.go` before
  writing a new handler.
- SSE endpoints are exempt (raw `event:`/`data:` frames).

### 1.3 Error statuses
- `400` validation failure, `401` unauthenticated, `403` authenticated but not
  allowed (role / CSRF / source policy), `404` missing resource, `409` conflict,
  `429` rate limited (body includes `retry_after_seconds`), `500` internal.

### 1.4 Audit logging
- Every **mutating** endpoint added in stages 13-19 writes an `audit_log` row:
  actor (session username or API-key name), action (`<resource>.<verb>`, e.g.
  `virtual_key.create`), target, details. Read `internal/store/audit.go` first.
- Mandatory for: auth/user changes, backup/restore, MITM toggle, tunnel
  start/stop, key/team/budget changes, settings writes.

### 1.5 Validation
- Validate all input at the handler boundary. Fail fast with `400` + specific
  message. Never pass unvalidated input to `exec.Command`, SQL, or file paths.

### 1.6 Secrets at rest
- Reversible secrets (proxy credentials, alert tokens, tunnel config) are stored
  encrypted using the same mechanism as OAuth tokens — read
  `internal/store/oauthsessions.go` and reuse that pattern. Never plaintext.
- Irreversible secrets (passwords, virtual key material) are hashed
  (bcrypt for passwords, SHA-256 like `api_keys` for keys).

### 1.7 Feature flag gating
- Features marked `[flag: <key>]` in a phase doc check the flag at the
  middleware/dispatch boundary and no-op (pass-through) when disabled.
  Flags are seeded in the `feature_flags` migration with `enabled=0`.
- Flagged features: `semantic_cache`, `guardrails`, `pii_redaction`,
  `websocket_chat`, `mitm_proxy`.

---

## 1.8 Architecture (DDD-lite — mandatory for new code)

Layers, dependency direction strictly inward:

```
api/handlers/   → HTTP transport ONLY: parse, validate, envelope, status codes.
                  No business rules. Thin — delegate to domain.
internal/<domain>/ → domain packages own ALL business logic:
                  auth (sessions, CSRF, rate limit), tunnel, console,
                  governance (virtual keys, budgets, limits), guardrails,
                  semcache, update. Pure Go, no fasthttp imports.
internal/store/ → repository layer: persistence only, no business decisions.
                  One file per aggregate. Domain packages depend on store
                  via narrow interfaces they define themselves.
```

Rules:
- New business logic NEVER lives in a handler or in store. New domain concept
  → new `internal/<domain>` package with its own tests.
- Domain packages define the repository interface they need; `*store.Store`
  satisfies it. Tests use fakes implementing the same interface.
- Handlers depend on domain packages, not the reverse.
- Files 200-400 lines typical, 800 max. Split by aggregate, not by type.
- Immutability: return new values, don't mutate inputs.

## 2. Per-Task TDD Loop (unchanged from AGENTS.md, restated)

For every task inside a phase:

1. Read 3 existing analogous files (store / handler / middleware).
2. Write `_test.go` first. Run it. **See it fail.**
3. Write minimum code to pass.
4. `go test ./... -count=1` + `go vet ./...` green.
5. Commit: `phase-N/task-M: <description>`.

No mocks — interfaces and fakes. Temp SQLite DBs in tests. `httptest`-style
handler tests via fasthttp `serveCtx` helpers already used in `api/*_test.go`.

---

## 3. Gates

### 3.1 Per-commit gate (every commit)
```bash
go test ./... -count=1
go vet ./...
go build ./cmd/g0router
```

### 3.2 Per-phase gate (before phase checkpoint)
```bash
go test ./... -count=1
go vet ./...
go test -race ./...
go build ./cmd/g0router
```
- Go coverage must not drop below **95.0%** (current baseline). Check with the
  existing coverage make target.
- UI gates (`npm --prefix ui test -- --run`, `npm --prefix ui run build`) run
  **only if `ui/` was touched** in the phase. Backend phases 13-19 should not
  touch `ui/`; if one does, both UI gates must pass.

### 3.3 Stage exit gate (after phase 19)
All per-phase gates + `make e2e-binary` + `gitleaks` scan clean.

---

## 4. Checkpoint Protocol (end of every phase)

1. Run per-phase gate (3.2). All green or phase is NOT done.
2. Update `docs/WORKFLOW.md` Stage 13-19 phase table: status → `DONE`,
   add commit range, note any deviations from the phase doc.
3. Update the phase doc itself: append a `## Outcome` section listing
   what shipped, what was deferred, and why.
4. Commit docs: `phase-N/checkpoint: workflow + outcome notes`.
5. **Stop and reassess**: re-read the next phase doc against the now-current
   codebase. If assumptions broke, update the phase doc BEFORE starting it.

A phase with failing gates, missing tests, or undocumented deviations is
`BLOCKED`, not `DONE`. Never start phase N+1 from a `BLOCKED` phase.

---

## 5. Definition of Done (per phase checklist)

- [ ] Every feature in the phase doc implemented or explicitly deferred in `## Outcome`
- [ ] Every new table in `internal/store/sqlite.go` migrations (additive only)
- [ ] Every new store file has `_test.go` written first
- [ ] Every new endpoint: envelope + snake_case + validation + audit (if mutating)
- [ ] Per-phase gate green (3.2), coverage ≥ 95.0%
- [ ] `docs/WORKFLOW.md` phase table updated
- [ ] Phase doc `## Outcome` section written
- [ ] All commits pushed

---

## 6. Scope Control

- **Deferred ≠ deleted.** Anything cut goes in the phase doc `## Outcome` with a
  one-line reason. Don't silently drop features.
- **No UI work** in phases 13-19. The Lovable UI lands in phases 20-21.
  New APIs are verified via Go tests + curl.
- **Don't break**: source IP policy, RTK/Caveman dispatch wiring, MCP agent
  loop, existing `/v1/*` auth, existing `/api/*` bearer auth (coexists with
  sessions — see phase 13).
- If a phase doc contradicts the codebase, the codebase wins; update the doc
  and note it at the checkpoint.

---

## 7. Security Review Triggers

Run a focused security pass (and record it in the phase `## Outcome`) for:
- Phase 13 (auth/sessions/CSRF) — mandatory
- Phase 15 (binary downloads, CLI shelling) — mandatory
- Phase 18 (backup/restore secret export, budget enforcement) — mandatory
- Phase 19 (MITM CA, auto-updater self-replace) — mandatory

Checklist per pass: input validation, authn/authz on every new route, secrets
at rest, secrets in logs, supply-chain (downloads pinned + checksummed),
privilege requirements documented.

---

## 8. Recovery Protocol

1. `go test ./...` → identify failures
2. `git log --oneline -10` → last good commit
3. Read WORKFLOW.md Stage 13-19 table → active phase/task
4. Fix failing tests before proceeding. Never skip — fix or revert.
