# Risk Register

- **Admin lockout** — `require_login=true` with no users locks everyone out. Mitigation: reject with `409` unless ≥1 dashboard user; `DELETE` cannot remove last admin.
- **Session fixation / token leak** — raw token in DB would be replayable on dump. Mitigation: store only SHA-256(token); raw value lives solely in the HttpOnly cookie; rotate via logout/password-change invalidating other sessions.
- **CSRF bypass** — cross-origin browser mutation rides the cookie. Mitigation: `SameSite=Strict` + Origin/Referer host match on cookie-authed mutations → `403`; bearer path intentionally exempt.
- **Rate-limit bypass via XFF spoofing** — attacker forges `X-Forwarded-For` to dodge per-IP limiter. Mitigation: trust XFF first hop only when `trust_proxy_headers=true` (default false; documented tunnel-only tradeoff).
- **Coexistence regression** — new session middleware breaks existing bearer/X-API-Key clients. Mitigation: middleware accepts either credential; exempt routes preserved; tests assert bearer still works under `require_login=true`.
