# Fix — w3-a + w3-e + w3-f diff-gate findings (2026-06-11)

Author: Fable 5. Implementer: kimi. All findings REAL. Dispatched AFTER w3-b merges
(shared admin_test.go — no concurrent writers).

## w3-a (limiter semantics + handler coverage)
1. `internal/auth/limiter.go` `RecordFail`: the lock-triggering (5th) call must return
   `remainingBeforeLock = 0`, not 5. Compute remaining BEFORE resetting `fails`, or
   return 0 explicitly when the call trips the lock. The semantics: "attempts left
   before lock" — at the moment of locking, 0.
2. `internal/auth/limiter_test.go`: fix `TestLimiterLocksAfterFiveFails` (and any sibling)
   to assert the lock-triggering call returns 0 (codify the correct contract).
3. `internal/admin/admin_test.go` `TestLoginDefaultPasswordWhenNoHash`: add the
   handler-level env-override case (set `INITIAL_PASSWORD`, no user hash → that
   password logs in through the Login HANDLER, not just Sessions.Login).

## w3-e (proxy dialer address + test race)
4. `internal/providers/utils/client.go`: `fasthttpproxy.FasthttpHTTPDialer` expects
   `host:port`, not a full URL — pass `proxyURL.Host` (the BLOCKER; proxy dials
   currently fail). Keep the per-proxy client cache keyed by the proxy URL string.
5. `internal/providers/utils/proxy_test.go`: the `targetStub`/`seen` shared state must
   be mutex- or atomic-guarded like `proxyLog` so `-race` is clean.

## w3-f (refresh single-flight ordering + xai scope encoding)
6. `internal/auth/credentials.go`: the single-flight refresh must call `c.wg.Done()`
   BEFORE (or atomically with) deleting `r.calls[conn.ID]` under the lock — current
   order (delete then Done) opens a window where a concurrent caller misses the
   in-flight entry and starts a 2nd refresh. Correct pattern: hold `r.mu`, look up or
   create the call; the leader runs refresh, then under `r.mu` deletes the entry and
   calls Done; waiters Wait on wg AFTER releasing r.mu. Ensure `TestRefreshSingleFlight`
   actually exercises concurrency (N goroutines, exactly 1 token request) under -race.
7. `internal/auth/oauth.go` xAI authorize URL: scope spaces must be `%20`, not `+`
   (`xai.js:103-130`). Do NOT use bare `url.Values.Encode()` for the scope param;
   build the query so the scope's spaces become `%20` (e.g. `url.QueryEscape` then
   replace, or construct manually). Add/extend a test asserting the authorize URL
   contains `%20` (not `+`) in scope.

## Acceptance (binary)
- `go test ./... && go vet ./...` green; `go test -race ./internal/auth/ ./internal/providers/utils/` green.
- `TestLimiterLocksAfterFiveFails` asserts remaining==0 on the locking call.
- `TestRefreshSingleFlight`: N concurrent ResolveKey → exactly 1 upstream token request (-race clean).
- xai authorize-URL test asserts `%20` scope encoding (no `+`).
- `grep -c 'proxyURL.Host\|\.Host)' internal/providers/utils/client.go` ≥ 1.
- Files touched ONLY: internal/auth/limiter.go, limiter_test.go, internal/admin/admin_test.go, internal/providers/utils/client.go, proxy_test.go, internal/auth/credentials.go, credentials_test.go, internal/auth/oauth.go, oauth_test.go. Do NOT git commit.
