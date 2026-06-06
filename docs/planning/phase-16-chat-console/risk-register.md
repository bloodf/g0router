# Risk Register

- **Ring memory growth** — 1000-entry cap with `attrs` of unbounded size could bloat memory. Mitigate: fixed-capacity ring overwrites oldest; bound/serialize attrs at insert.
- **Oversized payloads** — large `messages_json` or base64 images exhaust memory/DB. Mitigate: validate at store boundary (≤2MB json, ≤5MB/image, ≤4/message, mime allowlist) before persist; reject 400.
- **Broker goroutine lifecycle** — leaked subscriber goroutines on client disconnect or shutdown. Mitigate: mirror `internal/traffic` broker incl. nil-broker guard and shutdown path; test disconnect-releases-subscriber and clean shutdown.
