# Fix micro-plan — w2-d diff-gate findings (2026-06-11)

Author: Fable 5. Implementer: kimi. Both findings REAL.
Artifact: `artifacts/w2-d-catalog-router-diff-scoped-gpt.txt`.

## Task 1 — synchronize the lazy provider cache (BLOCKER, concurrency)

`internal/inference/router.go` `providerForID` reads `r.providers[id]` then writes
`r.providers[id]=p` with no lock — a data race / concurrent-map-write panic on the
fasthttp request path under load. Fix:
- Add `mu sync.RWMutex` to the `Router` struct (init in `NewRouter`; the map stays).
- `providerForID`: take `r.mu.RLock()` for the read; if found, RUnlock + return. On
  miss, RUnlock, build the provider (outside the lock), then `r.mu.Lock()`,
  double-check the map (another goroutine may have inserted), insert if still absent,
  Unlock, return. (Standard double-checked lazy-init; do NOT hold the lock across
  `buildProvider`.)
- Add `TestRouterConcurrentResolve` to `router_test.go`: launch N goroutines all
  calling `Resolve("deepseek-chat")` (and a mix of models); assert no race/panic and
  all return non-nil providers. Must pass under `go test -race`.

## Task 2 — explicit unknown-ID rejection in the factory (MAJOR)

`internal/inference/factory.go` `buildProvider` `default:` calls `generic.New(providerID)`
for any non-special id, relying on generic.New's internal catalog check. Make the
contract explicit: in `default`, first `if _, ok := catalog.Lookup(providerID); !ok {
return nil, fmt.Errorf("unknown provider %q", providerID) }`, THEN `return generic.New(providerID)`.
Add `TestBuildProviderUnknownErrors` (a bogus id → error).

## Acceptance (binary)

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `go test -race ./internal/inference/ -run 'TestRouterConcurrentResolve' -count=1` passes.
- `grep -c 'sync.RWMutex\|sync.Mutex' internal/inference/router.go` ≥ 1.
- `TestBuildProviderUnknownErrors` passes (unknown id → error, no generic fallthrough).
- Files touched ONLY: `internal/inference/router.go`, `internal/inference/router_test.go`,
  `internal/inference/factory.go`, `internal/inference/factory_test.go`. Do NOT git commit.

## Out of scope

Any provider/catalog change. Routing policy beyond thread-safety + unknown rejection.
