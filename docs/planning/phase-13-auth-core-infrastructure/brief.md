# Brief: Dashboard Auth & Core Infrastructure

**Problem:** The dashboard control plane has no password-based human authentication — anyone with network reach to `/api/*` can act once bearer auth is bypassed, and there is no first-run setup, session, or CSRF protection for browser users.

**Success criteria:**
- First-run setup creates the first admin, then login issues an HTTP-only `g0router_session` cookie that survives restart.
- `/api/*` accepts either existing bearer/X-API-Key auth OR a valid session cookie; existing CLI/bearer clients keep working unchanged.
- Cookie-authenticated mutating requests with a mismatched Origin are rejected `403`; bearer requests skip the check.
- Login rate limiting returns `429` with `retry_after_seconds` after 5 failed attempts per IP per 15 min.

**Non-goals:**
- No new dashboard UI (toast/theme/breadcrumbs/Cmd+K are Lovable's job, phases 20-21).
- No change to the existing `admin/123456` control-plane API key seeded in `root.go`.
- No OAuth/SSO/external identity providers.

**Constraints:** snake_case JSON + `{data, error}` envelope on every `/api/*` response; passwords bcrypt-hashed, session tokens SHA-256-hashed at rest; additive migrations only; coverage ≥95.0%; `require_login` defaults false. Security review mandatory (process doc §7).

**Verification:** `go test ./... -count=1 && go vet ./... && go test -race ./... && go build ./cmd/g0router`, coverage ≥95.0%, plus the auth test matrix in the phase doc §Test Requirements (setup-once-409, login 200/401/429, session+bearer coexistence under `require_login=true`, Origin 403, logout invalidation, last-admin guard, restart survival, XFF gating).

**QA criteria:**
```yaml
qa_skip: null
scenarios:
  - id: 1
    description: First-run POST /api/auth/setup creates an admin and a second call returns 409.
    method: api
    evidence: curl transcript showing 200 with Set-Cookie on first call, 409 on second.
  - id: 2
    description: POST /api/auth/login with correct password returns 200 and sets the g0router_session HttpOnly cookie; subsequent /api/* request with that cookie succeeds under require_login=true.
    method: api
    evidence: curl transcript of login response headers and an authenticated follow-up request.
  - id: 3
    description: Cookie-authenticated mutating request with a cross-origin Origin header is rejected 403; the same request with a bearer key succeeds.
    method: api
    evidence: paired curl transcripts (cookie+bad-Origin → 403, bearer → 2xx).
  - id: 4
    description: Six failed logins within the window return 429 with retry_after_seconds in the body.
    method: api
    evidence: curl transcript of the 6th attempt showing 429 and retry_after_seconds.
manual_smoke: Build the binary, run first-run setup, log in via curl, confirm the session cookie grants /api/* access after a process restart, then exercise logout and confirm the next request is 401.
```

**Linked artifacts:** architect-plan: ./architect-plan.md; orchestration: ./orchestration.jsonl
