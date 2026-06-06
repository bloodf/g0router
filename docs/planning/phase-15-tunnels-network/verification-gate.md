# Verification Gate — Phase 15

All must pass before the phase is `DONE`:

- [ ] `go test ./... -count=1` — green
- [ ] `go vet ./...` — clean
- [ ] `go build ./cmd/g0router` — succeeds
- [ ] `go test -race ./...` — clean (health loops, supervisor)
- [ ] Coverage ≥ 95.0% (existing make coverage target)
- [ ] qa-engineer required? Yes — runtime-required scenarios (tunnel lifecycle needs a running binary); proxy-test + tailscale-409 verifiable via api method
- [ ] Manual smoke: start binary → create Cloudflare tunnel → confirm `/healthz` reachable via tunnel URL → delete → confirm process killed
- [ ] Rollback signal: any gate red or unfixable security finding → execute ./rollback.md
- [ ] Mandatory security pass recorded in phase doc `## Outcome` (§7 checklist)
- [ ] Findings flywheel? No (no prior findings)
