# Verification Gate

Per-phase gate (STAGE-13-19-PROCESS §3.2) — all must pass before checkpoint:

- `go test ./... -count=1` — green.
- `go vet ./...` — clean.
- `go build ./cmd/g0router` — succeeds.
- `go test -race ./...` — green.
- Go coverage ≥ **95.0%** (current baseline, must not drop).
- qa-engineer? — yes: `method: api` scenarios (vk-budget-enforcement, routing-rule-application, feature-flag-gating, backup-secret-redaction) per brief QA criteria.
- Manual smoke — create gvk- key + team via curl, drive `/v1/*`, confirm `budget_used_usd` accrues on key + team; run `POST /api/settings/backup` and grep output for secret leakage.
- Security pass — mandatory (§7): input validation, authn/authz on every new route, secrets at rest (`config_enc` encrypted, key material hashed), secrets-in-logs scan, privilege requirements documented; recorded in phase `## Outcome`.
- Rollback signal — feature_flags default 0 + additive-only tables; revert safe.
- Findings flywheel? — no (no prior findings).
