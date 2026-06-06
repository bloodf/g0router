# Verification Gate

**Tests that must pass:**
- Unit: `go test ./... -count=1` ; `go vet ./...` ; `go build ./cmd/g0router` (per-commit gate §3.1)
- Integration: `go test ./api/... -count=1` (48 KB `server_integration_test.go` net) ; per-phase `go test -race ./...` ; coverage ≥ 95.0% via the existing coverage make target (§3.2)
- E2E: n/a (no behavior change; `make e2e-binary` runs at the stage-19 exit gate, not this phase)

**qa-engineer triggered?** no — `qa_skip: type-only-refactor` (pure backend refactor, zero behavior change, no `ui/` touched; UI gates skipped per §3.2).

**Manual smoke check:** none.

**Rollback signal:** any post-merge failure of the per-commit/per-phase gate on main, a coverage drop below 95.0%, or a route-table snapshot mismatch hands off to rollback.md.

**New regression tests required by findings flywheel?** no (no prior findings)
