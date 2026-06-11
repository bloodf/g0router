# w3-a — Login hardening: limiter, lockout, default password, auth mode, reset CLI

Rows: PAR-AUTH-002 (default-password fallback, `src/app/api/auth/login/route.js:49-50`), 006 (progressive lockout, `src/lib/auth/loginLimiter.js:3-5`), 014 (password reset CLI, `cli/src/cli/menus/settings.js:177-204`), 015 (auth-mode switch, `src/app/api/auth/login/route.js:40-41`, `src/app/api/auth/status/route.js:13`), 023 (1h auto-reset, `loginLimiter.js:5`), 026 (client-IP extraction, `loginLimiter.js:48-51`). Scope per `WAVE-3-MAP.md` track 1, plan 1. Frozen ref @ 827e5c3.

In-repo integration points: `internal/admin/auth.go:29` (`Login` handler — extend), `internal/auth/session.go:51` (`Sessions.Login`), `internal/auth/password.go:40` (`VerifyPassword` — PBKDF2 stays, NOT bcrypt, per auth-matrix Go-port consideration "PBKDF2 is already present and OWASP-recommended"), `internal/store/settings.go:9,31` (`GetSettings`/`SetSettings`), `cmd/main.go` (CLI).

## Ref behavior to port (read whole files first)

- **Limiter** (`loginLimiter.js` whole file): per-IP in-memory map; constants
  MAX_FAILS_BEFORE_LOCK=5, LOCK_STEPS = [30s, 2m, 10m, 30m], FAIL_WINDOW=1h.
  `getEntry` auto-deletes when `now-lastFailAt > 1h` AND not currently locked (023).
  `checkLock` → `{locked, retryAfter(sec, ceil)}`. `recordFail`: fails+1, set
  lastFailAt; at 5 fails → lockUntil = now + LOCK_STEPS[min(lockLevel, 3)],
  lockLevel+1, fails=0. `recordSuccess` deletes the entry. Lock response: HTTP 429,
  `Retry-After` header, error text "Too many failed attempts. Try again in <N>s."
  plus a reset hint mentioning the CLI reset path (g0router wording).
- **Client IP** (`loginLimiter.js:48-51`): first entry of `x-forwarded-for`
  (split on ",", trim), else `x-real-ip`, else `"unknown"` (026).
- **Default password** (`login/route.js:49-50`): when NO password hash is stored in
  settings, compare against `INITIAL_PASSWORD` env var, default `"123456"` (002).
  When a hash IS stored, verify via the existing PBKDF2 `VerifyPassword`.
- **Auth mode** (`login/route.js:40-41`, `status/route.js:13`): settings key
  `authMode` ∈ {"password","oidc","both"} (default "password"). If `authMode=="oidc"`
  AND OIDC is configured → password login returns 403 "Password login is disabled.
  Use OIDC sign in." (015). Stage-1 note: OIDC config lands in w3-c — until then
  `oidcConfigured()` is a helper returning whether the OIDC settings keys are
  non-empty (keys defined here, populated by w3-c); with no OIDC config, mode "oidc"
  does NOT lock the operator out (the ref's `isOidcConfigured` guard, same line).
  `/api/auth/status` response gains `auth_mode` (snake_case envelope per AGENTS.md).
- **Reset CLI** (`settings.js:177-204`): a `g0router` CLI action that clears the
  stored password hash in settings (next login uses the default-password path) and
  prints confirmation. Go: a subcommand `reset-password` in `cmd/main.go` operating
  on the data-dir DB directly (no server needed), mirroring the ref's menu action.

## Conventions (constraints on the worker)

No `init()`, no globals: the limiter is `type LoginLimiter struct { mu sync.Mutex; attempts map[string]*entry; now func() time.Time }` with `NewLoginLimiter()` (injected clock for tests; `now` defaults to `time.Now`). Owned by `internal/admin.Handlers` (constructed where Handlers is built). In-memory per ref ("Resets on process restart" — `loginLimiter.js:1`); do NOT persist. Concurrency-safe (mutex) — fasthttp handlers are concurrent; must pass `-race`.

## Preconditions (a "0 hits" grep exits 1 = pass)

- `grep -rn 'LoginLimiter\|loginLimiter' internal/` → 0 hits (new)
- `grep -c 'func (h \*Handlers) Login' internal/admin/auth.go` ≥ 1 (extending it)
- `grep -c 'func VerifyPassword' internal/auth/password.go` ≥ 1 (reuse; do NOT add bcrypt)
- `grep -rn 'authMode\|auth_mode' internal/ --include='*.go'` → 0 hits (new key)

## Exclusive file ownership

NEW: `internal/auth/limiter.go`, `internal/auth/limiter_test.go`.
TOUCH: `internal/admin/auth.go` (Login handler: lock check → tunnel-block hook
point left for w3-b → mode check → default-pw/hash verify → record fail/success;
Status handler: add auth_mode), `internal/admin/auth_test.go`, `cmd/main.go`
(+`cmd/main_test.go`) for `reset-password`.
NOT touched: `internal/auth/session.go`, `password.go`, `oauth.go` (reused as-is);
the dashboard guard (w3-b); OIDC implementation (w3-c — only the settings-key helper
`oidcConfigured` is defined here, in `internal/admin/auth.go`).

## Tasks (each: STEP (a) write the named failing tests, run, show fail; STEP (b) implement)

1. **LoginLimiter** (`internal/auth/limiter.go`): port the limiter verbatim with the
   exact constants/steps above; methods `CheckLock(ip) (locked bool, retryAfter int)`,
   `RecordFail(ip) (remainingBeforeLock int)`, `RecordSuccess(ip)`, plus
   `ClientIP(xff, xRealIP string) string` (pure function for 026).
   Tests (`limiter_test.go`, injected clock): `TestLimiterLocksAfterFiveFails`,
   `TestLimiterProgressiveSteps` (4 lock levels: 30s/2m/10m/30m; level capped at last),
   `TestLimiterAutoResetAfterWindow` (advance >1h since lastFail, unlocked → entry gone; 023),
   `TestLimiterNoResetWhileLocked` (window expired but still locked → stays),
   `TestLimiterRecordSuccessClears`, `TestClientIPExtraction` (XFF first-entry trim /
   x-real-ip / "unknown"), `TestLimiterConcurrent` (parallel CheckLock/RecordFail; run with -race).

2. **Login handler integration** (`internal/admin/auth.go`): order — ClientIP →
   CheckLock (locked → 429 + Retry-After + reset-hint error in the `{data,error}`
   envelope) → authMode check (oidc-only + configured → 403) → password verify:
   stored hash → `VerifyPassword`; no hash → constant-time compare against
   `INITIAL_PASSWORD` env or "123456" (002) → fail: RecordFail; success:
   RecordSuccess + existing session issue. Status: include `auth_mode`.
   Tests (`auth_test.go` additions): `TestLoginLockout429AndRetryAfter`,
   `TestLoginDefaultPasswordWhenNoHash` (and env override),
   `TestLoginOidcModeBlocksPassword` (mode oidc + configured keys → 403; mode oidc +
   NOT configured → password still works), `TestStatusReportsAuthMode`.

3. **reset-password CLI** (`cmd/main.go`): subcommand `reset-password` (flag-compatible
   with existing arg parsing in main.go) — opens the store, clears the password-hash
   settings key, prints "Password reset to default." Test (`cmd/main_test.go` or a
   store-level test): `TestResetPasswordClearsHash` (hash set → run → key empty →
   login takes default-pw path).

## Binary acceptance criteria

- `go test ./...` exits 0; `go vet ./...` exits 0; `go test -race ./internal/auth/ -run 'TestLimiter' -count=1` exits 0.
- `grep -c 'bcrypt' internal/ -r` → 0 (PBKDF2 kept).
- `grep -rn 'func init(\|panic(' internal/auth/limiter.go internal/admin/auth.go` → 0 hits.
- `grep -c '30000\|30_000\|30 \* time.Second' internal/auth/limiter.go` ≥ 1 AND the four steps 30s/2m/10m/30m present (verbatim constants).
- `TestLimiterProgressiveSteps`, `TestLimiterAutoResetAfterWindow`, `TestLoginDefaultPasswordWhenNoHash`, `TestLoginOidcModeBlocksPassword` pass.

## Out of scope

Dashboard guard middleware + tunnel login block (w3-b — the Login handler leaves the
hook point; 027 is w3-b's). OIDC flow itself (w3-c; only the configured-keys helper
here). API keys (w3-d). JWT (decision 2 — never). Persisting lockout state (ref is
in-memory by design). bcrypt (consideration says keep PBKDF2).
