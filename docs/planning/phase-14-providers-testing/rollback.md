# Rollback

- Revert via `git revert <commit-range>` for the phase; each task is an isolated commit.
- New tables (`proxy_pools`, `disabled_models`, `custom_models`) and the additive `connections.proxy_pool_id` column are safe to leave in place — `CREATE TABLE IF NOT EXISTS` / `ensureColumn` are idempotent and unused columns are inert.
- No data migration to undo; no destructive schema change to reverse.
- Proxy wiring revert restores direct outbound clients with no credential exposure.
