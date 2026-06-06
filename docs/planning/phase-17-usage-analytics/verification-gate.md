# Verification Gate — Phase 17

- [ ] `go test ./... -count=1` — green
- [ ] `go vet ./...` — clean
- [ ] `go build ./cmd/g0router` — succeeds
- [ ] `go test -race ./...` — green; coverage ≥ 95.0%
- [ ] qa-engineer? yes — scenarios `usage-chart-shape`, `bulk-disable` (method: api)
- [ ] manual smoke — curl `/api/usage/chart` + `/api/connections/bulk-disable` on seeded DB; confirm envelope, zero-fill, audit row
- [ ] rollback signal — incorrect bucket sums, mass-disable of active connections, or coverage < 95.0%
- [ ] findings flywheel? no (no prior findings)
