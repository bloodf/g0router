# Verification Gate

- [ ] `go test ./... -count=1` green
- [ ] `go vet ./...` clean
- [ ] `go build ./cmd/g0router` succeeds
- [ ] `go test -race ./...` green
- [ ] Coverage ≥ 95.0% (per-phase baseline, PROCESS §3.2)
- [ ] qa-engineer? yes — semcache hit path (api), updater staged-swap + WebSocket protocol (runtime-required), manual_smoke on version/locale/skills/mitm
- [ ] Manual smoke: version/locale/skills/mitm endpoints return `{data, error}` envelope shapes
- [ ] Rollback signal: feature flags → `enabled=0` no-ops all three flagged features; staged updater binary inert until swap
- [ ] Security pass recorded in phase `## Outcome` (mandatory, PROCESS §7): updater supply chain + MITM CA handling
- [ ] Findings flywheel? no (no prior findings)
