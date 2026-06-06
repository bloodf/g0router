# Risk Register

- **Proxy credential leakage** — `password_enc` must encrypt at rest (oauthsessions.go pattern); CRUD/test/batch responses and logs must never emit plaintext. Test asserts no password field in responses.
- **fasthttp proxy support limits** — provider adapters use fasthttp clients; `http.Transport.Proxy` may not apply. Inspect `internal/providers/utils` first; use fasthttp dialer equivalent for SOCKS5/HTTP. Record actual approach in phase `## Outcome`; deferral allowed if a protocol is unsupported.
- **Model-disable affecting live routing** — disabling a model must reject routing with a clear 400-class error, not a 500 or silent drop. Filtering must cover `/v1/models`, `/api/models`, and routing candidate sets consistently.
- **Migration safety** — all new tables/columns additive (`ensureColumn`); no destructive alters.
