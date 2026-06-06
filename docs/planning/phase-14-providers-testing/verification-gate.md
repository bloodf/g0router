# Verification Gate

- **Unit/Integration:** `go test ./... -count=1` green.
- **Vet:** `go vet ./...` clean.
- **Build:** `go build ./cmd/g0router` succeeds.
- **Per-phase:** `go test -race ./...` green; coverage ≥ 95.0% (current baseline).
- **E2E:** not required this phase (stage exit gate `make e2e-binary` runs after phase 19).
- **qa-engineer triggered?** Yes — `qa_skip: null`, 3 api-method scenarios (proxy-pool secret omission, disabled-model routing, model-test no-500).
- **Manual smoke:** curl proxy-pools CRUD + model-test + disabled-models list against a local g0router binary; confirm `{data,error}` envelopes, snake_case, no plaintext secrets.
- **Rollback signal:** any per-phase gate red, coverage < 95.0%, or plaintext secret in a response → phase BLOCKED, revert per rollback.md.
- **New regression tests required by findings flywheel?** no (no prior findings).
