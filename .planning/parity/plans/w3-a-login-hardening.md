# w3-a — Login hardening: limiter, lockout, default password, auth mode, reset CLI

Rows: PAR-AUTH-002 (default-password fallback, `src/app/api/auth/login/route.js:49-50`), PAR-AUTH-006 (progressive lockout, `src/lib/auth/loginLimiter.js:3-5`), PAR-AUTH-014 (password reset CLI, `cli/src/cli/menus/settings.js:177-204`), PAR-AUTH-015 (auth-mode switch, `src/app/api/auth/login/route.js:40-41`, `src/app/api/auth/status/route.js:13`), PAR-AUTH-023 (1h auto-reset, `src/lib/auth/loginLimiter.js:5`), PAR-AUTH-026 (client-IP extraction, `src/lib/auth/loginLimiter.js:48-51`). Scope per `WAVE-3-MAP.md` track 1, plan 1. Frozen ref @ 827e5c3.

In-repo integration points: `internal/admin/auth.go:29` (`Login` handler — extend), `internal/auth/session.go:51` (`Sessions.Login`), `internal/auth/password.go:40` (`VerifyPassword` — PBKDF2 stays, NOT bcrypt, per auth-matrix Go-port consideration "PBKDF2 is already present and OWASP-recommended"), `internal/store/settings.go:9,31` (`GetSettings`/`SetSettings`), `cmd/g0router/main.go` (CLI; data-dir resolution :52-65).

## Ref behavior to port (read whole files first)

- **Limiter** (`loginLimiter.js` whole file): per-IP in-memory map; constants
  MAX_FAILS_BEFORE_LOCK=5, LOCK_STEPS = [30s, 2m, 10m, 30m], FAIL_WINDOW=1h.
  `getEntry` auto-deletes when `now-lastFailAt > 1h` AND not currently locked (PAR-AUTH-023).
  `checkLock` → `{locked, retryAfter(sec, ceil)}`. `recordFail`: fails+1, set
  lastFailAt; at 5 fails → lockUntil = now + LOCK_STEPS[min(lockLevel, 3)],
  lockLevel+1, fails=0. `recordSuccess` deletes the entry. Lock response: HTTP 429,
  `Retry-After` header, error text "Too many failed attempts. Try again in <N>s."
  plus a reset hint — REF BEHAVIOR, not invented: `login/route.js:9` `RESET_HINT` const, included in the 429 body at `:24` (`error`, `retryAfter`, `resetHint` fields). Port with g0router CLI wording ("… via g0router CLI: g0router reset-password").
- **Client IP** (`loginLimiter.js:48-51`): first entry of `x-forwarded-for`
  (split on ",", trim), else `x-real-ip`, else `"unknown"` (PAR-AUTH-026).
- **Default password** (`login/route.js:49-50`): the ref keys this on "no stored
  hash". g0router stores hashes on the `users` table (`internal/store/users.go:14`
  `PasswordHash`; `internal/auth/session.go:51-61` `Login` → `GetUserByUsername` →
  `VerifyPassword`). Port the rule into `Sessions.Login`: when
  `user.PasswordHash == ""`, compare (constant-time, `crypto/subtle`) against
  `INITIAL_PASSWORD` env var, default `"123456"` (PAR-AUTH-002). Non-empty hash → existing
  PBKDF2 path unchanged. (`SeedAdmin` `session.go:32-48` is unchanged — it seeds a
  hashed password; the empty-hash state arises from the reset CLI below.)
- **Auth mode** (`login/route.js:40-41`, `status/route.js:13`): settings key
  `authMode` ∈ {"password","oidc","both"} (default "password"). If `authMode=="oidc"`
  AND OIDC is configured → password login returns 403 "Password login is disabled.
  Use OIDC sign in." (PAR-AUTH-015). Stage-1 note: OIDC config lands in w3-c — until then
  `oidcConfigured()` is a helper returning whether the OIDC settings keys are
  non-empty (keys defined here, populated by w3-c); with no OIDC config, mode "oidc"
  does NOT lock the operator out (the ref's `isOidcConfigured` guard, same line).
  `/api/auth/status` response gains `auth_mode` (snake_case envelope per AGENTS.md).
- **Reset CLI** (`settings.js:177-204`): a `g0router reset-password` subcommand in
  `cmd/main.go` operating on the data-dir DB directly (no server needed): set the
  admin user's `PasswordHash` to `""` via a NEW store method
  `SetUserPasswordHash(username, hash string) error` (`internal/store/users.go` —
  follows CreateUser/GetUserByUsername patterns :20-50), so the next login takes the
  default-password path (PAR-AUTH-002). Prints "Password reset to default." (PAR-AUTH-014).

## Conventions (constraints on the worker)

Engineering constraints are AGENTS.md conventions (citable project law): "No global
state — pass dependencies via struct fields", "No mocks — use interfaces and fakes"
(hence the injected clock as a fake), "TDD always". The server is `fasthttp.Server` (`internal/server/server.go`), which serves requests on concurrent goroutines (fasthttp documented behavior), so the shared attempts map requires a mutex and `-race` coverage. The limiter is `type LoginLimiter struct { mu sync.Mutex; attempts map[string]*entry; now func() time.Time }` with `NewLoginLimiter()` (injected clock for tests; `now` defaults to `time.Now`). Owned by `internal/admin.Handlers` (constructed where Handlers is built). In-memory per ref ("Resets on process restart" — `loginLimiter.js:1`); do NOT persist. Concurrency-safe (mutex) — fasthttp handlers are concurrent; must pass `-race`.

## Preconditions (a "0 hits" grep exits 1 = pass)

- `grep -rn 'LoginLimiter\|loginLimiter' internal/` → 0 hits (new)
- `grep -c 'func (h \*Handlers) Login' internal/admin/auth.go` ≥ 1 (extending it)
- `grep -c 'func VerifyPassword' internal/auth/password.go` ≥ 1 (reuse; do NOT add bcrypt)
- `grep -rn 'authMode\|auth_mode' internal/ --include='*.go'` → 0 hits (new key)

## Exclusive file ownership

NEW: `internal/auth/limiter.go`, `internal/auth/limiter_test.go`.
TOUCH: `internal/auth/session.go` (default-password fallback inside `Login`),
`internal/auth/auth_test.go`, `internal/store/users.go` + its tests (NEW
`SetUserPasswordHash`), `internal/admin/auth.go` (Login handler: ClientIP → lock
check → mode check → Sessions.Login → record fail/success; Status: add auth_mode),
`internal/admin/auth_test.go`, `cmd/g0router/main.go` + `cmd/g0router/main_test.go` (`reset-password`).
NOT touched: `internal/auth/password.go`, `oauth.go`; the dashboard guard (w3-b —
row 027 tunnel login block lands there, where that file is owned); OIDC flow (w3-c —
only the `oidcConfigured` settings-key helper is defined here in admin/auth.go).

## Tasks (each: STEP (a) write the named failing tests, run, show fail; STEP (b) implement)

1. **LoginLimiter** (`internal/auth/limiter.go`): port the limiter verbatim with the
   exact constants/steps above; methods `CheckLock(ip) (locked bool, retryAfter int)`,
   `RecordFail(ip) (remainingBeforeLock int)`, `RecordSuccess(ip)`, plus
   `ClientIP(xff, xRealIP string) string` (pure function for 026).
   Tests (`limiter_test.go`, injected clock): `TestLimiterLocksAfterFiveFails`,
   `TestLimiterProgressiveSteps` (4 lock levels: 30s/2m/10m/30m; level capped at last),
   `TestLimiterAutoResetAfterWindow` (advance >1h since lastFail, unlocked → entry gone; PAR-AUTH-023),
   `TestLimiterNoResetWhileLocked` (window expired but still locked → stays),
   `TestLimiterRecordSuccessClears`, `TestClientIPExtraction` (XFF first-entry trim /
   x-real-ip / "unknown"), `TestLimiterConcurrent` (parallel CheckLock/RecordFail; run with -race).

2. **Login handler integration** (`internal/admin/auth.go`): order — ClientIP →
   CheckLock (locked → 429 + Retry-After + reset-hint error in the `{data,error}`
   envelope) → authMode check (oidc-only + configured → 403) → `Sessions.Login`
   (which internally does hash-verify OR, on empty hash, the default-password
   compare per the Ref-behavior section — the handler does NOT duplicate that
   logic) → fail: RecordFail; success: RecordSuccess. Status: include `auth_mode`.
   Tests (`auth_test.go` additions): `TestLoginLockout429AndRetryAfter`,
   `TestLoginDefaultPasswordWhenNoHash` (and env override),
   `TestLoginOidcModeBlocksPassword` (mode oidc + configured keys → 403; mode oidc +
   NOT configured → password still works), `TestStatusReportsAuthMode`.

3. **reset-password CLI** (`cmd/g0router/main.go`): subcommand `reset-password` — resolves the data dir EXACTLY like the server already does (`main.go:52-58`: `G0ROUTER_DATA` env else `~/.g0router`), opens the store the same way (`:61-65`), targets the FIRST/only user row (g0router seeds exactly one admin via `SeedAdmin`, `session.go:32-48`; add store helper `FirstUser()` or reuse existing accessor), `SetUserPasswordHash(user.Username, "")`, prints "Password reset to default." Tests: store-level `TestSetUserPasswordHash` (users.go) and
   `TestResetPasswordThenDefaultLogin` (hash cleared → `Sessions.Login` succeeds with
   "123456").

## Binary acceptance criteria

- `go test ./...` exits 0; `go vet ./...` exits 0; `go test -race ./internal/auth/ -run 'TestLimiter' -count=1` exits 0.
- `grep -c 'bcrypt' internal/ -r` → 0 (PBKDF2 kept).
- `grep -rn 'func init(\|panic(' internal/auth/limiter.go internal/admin/auth.go` → 0 hits.
- `TestLimiterProgressiveSteps` proves the exact 30s/2m/10m/30m step durations via the injected clock (behavioral check; no grep proxy).
- `TestLimiterProgressiveSteps`, `TestLimiterAutoResetAfterWindow`, `TestLoginDefaultPasswordWhenNoHash`, `TestLoginOidcModeBlocksPassword` pass.

## Out of scope

Dashboard guard middleware + tunnel login block (w3-b — the Login handler leaves the
hook point; 027 is w3-b's). OIDC flow itself (w3-c; only the configured-keys helper
here). API keys (w3-d). JWT (decision 2 — never). Persisting lockout state (ref is
in-memory by design). bcrypt (consideration says keep PBKDF2).
