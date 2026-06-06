# Verification Gate

**Tests that must pass:**
- Unit / Integration: `go test ./... -count=1`; `go vet ./...`; `go build ./cmd/g0router`; per-phase: `go test -race ./...` and coverage ≥95.0% (existing make coverage target). Full auth matrix from the phase doc §Test Requirements must be covered: setup-once-then-409; login 200+cookie / wrong-401 / 6th-fail-429 with `retry_after_seconds`; session grants `/api/*` under `require_login=true`; garbage/expired cookie 401; bearer still works under `require_login=true`; exempt routes open; cookie+mismatched-Origin mutation 403; bearer mutation passes; logout invalidation; password-change wrong-current-403 + invalidates other sessions; `require_login=true` zero-users 409; cannot-delete-last-admin; sessions survive store reopen; XFF honored only when `trust_proxy_headers=true`.
- E2E: n/a (backend-only phase; no `ui/` changes; verified via Go tests + curl per process doc §6).

**qa-engineer triggered?** yes — api scenarios 1-4 (first-run setup→409, login→cookie session, CSRF cross-origin 403 vs bearer pass, rate-limit 429).

**Manual smoke check:** Build `go build ./cmd/g0router`, start the binary, run `POST /api/auth/setup` to create the first admin, `POST /api/auth/login` via curl and capture the `g0router_session` cookie, confirm an authenticated `/api/*` request succeeds with that cookie under `require_login=true` and continues to succeed after a process restart (server-side session persistence), then `POST /api/auth/logout` and confirm the next request returns `401`.

**Rollback signal:** existing bearer-auth `/api/*` API clients receiving `401`/`403` post-deploy (coexistence path regression).

**New regression tests required by findings flywheel?** no (no prior findings).
