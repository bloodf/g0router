# Rollback

- **Feature flags first:** set `semantic_cache`, `websocket_chat`, `mitm_proxy` to `enabled=0` — all three features no-op at their boundary, restoring prior behavior without a redeploy.
- **Updater:** delete staged `DATA_DIR/update/g0router.new`; no swap occurs until next graceful shutdown, so an un-swapped stage is inert.
- **MITM:** toggle off stops the listener; delete `DATA_DIR/mitm/` to discard the CA (regenerated on next enable).
- **Code:** `semantic_cache` table is additive (no destructive migration); revert via `git revert` of the task commit range. Endpoints (version/locale/skills) are read-mostly and safe to revert independently.
