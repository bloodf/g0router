# Rollback

- Revert is safe by design: `require_login` defaults `false`, so reverting the auth code leaves `/api/*` behaving exactly as before (bearer/X-API-Key only). No live deployment depends on session auth being present.
- `git revert <commit-range>` for phase-13 commits (direct push to main, no PR). Revert handler/middleware/settings changes together to avoid a half-wired auth path.
- Additive tables `dashboard_users` and `dashboard_sessions` may remain — they are unused after revert and harm nothing; dropping them is optional cleanup, not required for rollback.
- New settings keys `require_login` / `trust_proxy_headers` are inert once the reading code is gone; leave or delete.
- Rollback signal: existing bearer-auth `/api/*` clients receiving unexpected `401`/`403` after deploy → revert immediately, then diagnose the coexistence path offline.
