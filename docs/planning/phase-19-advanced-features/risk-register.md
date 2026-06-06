# Risk Register

- **Updater bricks install:** corrupt/partial download or wrong-arch asset swapped in. Mitigate: SHA-256 verify before staging, stage to `g0router.new` (never overwrite live binary), swap only on graceful shutdown, mismatch aborts + audits.
- **MITM CA key leak:** `ca.key` readable or returned by an endpoint. Mitigate: mode 0600, never log, `ca-cert` endpoint exposes cert only, no auto privileged hosts edits.
- **semcache serves wrong answer:** loose threshold or stale embedding returns a non-matching cached response. Mitigate: 0.95 threshold, exact-key first, lazy expiry purge, inert when no embedding connection, flag-gated.
- **WebSocket connection leaks:** sockets/goroutines not reclaimed on client disconnect. Mitigate: one in-flight chat per socket, context-cancel on close, dispatch cancellation propagated.
