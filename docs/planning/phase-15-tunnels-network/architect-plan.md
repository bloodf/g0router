# Architect Plan — Phase 15: Tunnels & Network

Canonical spec: [`docs/phases/phase-15-tunnels-network.md`](../../phases/phase-15-tunnels-network.md)

This is a thin wrapper. The phase doc is authoritative for features, endpoints, schema, and test requirements. Below is the implementation summary and the security posture that gate this work.

## Summary

- Pin `cloudflared` as hardcoded Go constants: one `version` string plus a `SHA-256` per OS/arch; verify the checksum before `chmod`+exec and refuse on mismatch. Download over HTTPS from the official GitHub releases URL only.
- **No Tailscale auto-install.** g0router only drives an already-installed `tailscale` binary found on `$PATH`. Absent binary → `409` with install instructions in the `error` field.
- Validate tunnel names against `[a-z0-9-]{1,63}` at the handler boundary; all `exec.Command` calls use absolute paths + fixed arg slices, no shell interpolation, no user input in args.
- New `tunnel_config` table: `config_enc` stores tokens/credentials encrypted, reusing the OAuth-token encryption pattern (`internal/store/oauthsessions.go`). Never plaintext; API responses never expose decrypted tokens.
- Background health loops: tunnel health every 60s (HTTP GET tunnel URL `/healthz`, update `status`/`last_error`); proxy pool health every 5min (test each active pool). Both started from server startup (no `init()`), stopped via context on shutdown; child processes get a cancellation context and are killed on shutdown.
- DDD-lite: `internal/tunnel/` owns download/checksum/process-supervisor business logic; handlers stay thin (parse, validate, envelope); store persists only.
- Task list: (1) store `tunnel_config` + encrypted config + tests; (2) `internal/tunnel/` checksum-verified download + process supervisor (fake binaries in tests); (3) handlers — tunnel CRUD/health + proxy-test; (4) background health loops; (5) checkpoint incl. security pass.

## Security notes

- **Supply-chain pinning:** version + per-OS/arch SHA-256 are compile-time constants; checksum verified before any exec; mismatch is a hard refusal with nothing executed. HTTPS, official GitHub releases URL only.
- **Secrets at rest:** `config_enc` encrypted via the OAuth-token mechanism; tokens never logged, never returned in API payloads.
- **Privilege boundary:** Tailscale requires root + a system daemon and is never downloaded/installed by g0router; documented as a preinstalled-binary dependency.
- **Security review is MANDATORY** at the checkpoint per STAGE-13-19-PROCESS.md §7: input validation, authn/authz on every new route, secrets at rest, secrets in logs, supply-chain (pinned + checksummed), privilege requirements documented.
