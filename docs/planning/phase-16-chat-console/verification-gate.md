# Verification Gate

Per-phase gate (STAGE-13-19-PROCESS §3.2) — all must pass:

- `go test ./... -count=1`
- `go vet ./...`
- `go build ./cmd/g0router`
- `go test -race ./...`
- Coverage ≥ 95.0% (no drop below baseline; existing coverage make target).
- qa-engineer? no (backend-only phase; API scenarios covered by Go tests; UI is phases 20-21).
- Manual smoke: curl chat-sessions CRUD round-trip and `GET /api/console-logs/stream` (replay then live) + `DELETE /api/console-logs`.
- Rollback signal: any gate red or coverage below 95.0% → phase BLOCKED, apply `rollback.md`.
- Findings flywheel? no (no prior findings).
