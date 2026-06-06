# Rollback — Phase 15

- **Trigger:** failing per-phase gate, coverage <95.0%, or a security-pass finding that can't be fixed forward.
- **Process state:** stop any running tunnel child processes (DELETE endpoints / context cancel) before reverting; orphaned `cloudflared` processes must be killed.
- **Code:** `git revert` the phase-15 task commits on main (direct-push); revert leaves `tunnel_config` migration in place (additive, inert when no rows).
- **Data:** `tunnel_config` rows reference encrypted config only; mark `is_enabled=0`/`status=inactive` if rows persist post-revert. No destructive migration.
- **Verify:** `go test ./... -count=1 && go build ./cmd/g0router` green after revert; confirm no lingering tunnel processes.
