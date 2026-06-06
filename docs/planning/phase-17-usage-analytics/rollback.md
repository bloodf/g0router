# Rollback — Phase 17

- Endpoints are additive (new routes, new store methods); no schema migration to reverse.
- Revert the phase-17 commits to remove `/api/usage/chart` and the bulk connection endpoints; existing `/api/quota` and usage-summary paths are untouched.
- Bulk-disable/enable are reversible via the existing per-connection enable/disable controls; affected ids are recorded in `audit_log` to identify what to restore.
- Signal to roll back: incorrect bucket sums, mass-disable of active connections, or coverage drop below 95.0%.
