# Architect Plan — Phase 19: Advanced Features

Canonical spec: [`docs/phases/phase-19-advanced-features.md`](../../phases/phase-19-advanced-features.md)

## Summary

- `internal/semcache/` computes cosine similarity in Go — SQLite cannot dot-product JSON embedding arrays. Repository interface for persistence; `*store.Store` satisfies it.
- Lookup is exact `cache_key` = sha256(normalized prompt + model) hit first (free); on miss, load same-model candidates (cap 500 newest, non-expired), cosine in Go, threshold `0.95` (`semantic_cache_threshold` setting).
- Embeddings come from an existing OpenAI-compatible provider connection; with no embedding-capable connection the cache is inert (logged reason) and never blocks requests. Sits AFTER guardrails, BEFORE dispatch; non-streaming `/v1/chat/completions` only; lazy expiry purge.
- `internal/update/` checks GitHub `releases/latest`, semver-compares (handles `v`-prefix + prerelease), downloads OS/arch asset + `checksums.txt`, verifies SHA-256, then stages at `DATA_DIR/update/g0router.new`; checksum mismatch aborts + audits. Swap happens on next graceful shutdown — explicit user action only, never automatic.
- WebSocket via `fasthttp/websocket`: upgrade on `/api/ws` with the same auth as `/api/*` before upgrade. Protocol v1: client `{type:"chat", session_id, model, messages}` → server `{type:"delta", content}` … `{type:"done", usage}` / `{type:"error", error}`. Reuses inference dispatch; one in-flight chat per socket; context-cancel on close.
- `internal/mitm/`: ECDSA P-256 CA, 10y, generated once → `DATA_DIR/mitm/ca.crt` + `ca.key` (mode 0600); per-host leaf certs minted on demand into an LRU cache. HTTPS proxy on `mitm_port` (default 8081), CONNECT interception routes tool hosts to local inference, tunnels others untouched.
- MITM does NO automatic `/etc/hosts` editing; `GET /api/mitm/status` returns per-tool manual instructions (hosts lines / proxy env). Tool host lists (Antigravity, Copilot, Cursor, Kiro) are config constants.
- `GET /api/skills` serves an embedded static catalog `[{name, category, description, url}]`. `GET`/`POST /api/locale` are settings-backed `{locale}`. `GET /api/version` returns `{version, go_version, build_date}`.
- All new endpoints: `{data, error}` envelope, snake_case, handler-boundary validation, audit row for every mutating route (update apply, mitm toggle, semantic cache clear, locale set).
- New `semantic_cache` table + indexes added as additive migrations in `internal/store/sqlite.go`. Flags `semantic_cache`/`websocket_chat`/`mitm_proxy` seeded `enabled=0`.
- Task order: version/locale/skills → semcache domain+store → semcache wiring → updater → WebSocket → MITM. **MITM implemented LAST** (highest risk).

## Security notes

Security review is **mandatory** for this phase (PROCESS §7). Focus areas:
- **Updater supply chain:** asset + `checksums.txt` pinned to the release, SHA-256 verified before any staging, mismatch aborts with nothing written; apply route is admin-only and audited; staged binary swapped only on graceful shutdown.
- **MITM CA handling:** `ca.key` written mode 0600, generated once and persisted; no automatic privileged `/etc/hosts` edits; CA private key never logged or returned by any endpoint (`GET /api/mitm/ca-cert` exposes the cert only).
- General: authn/authz on every new route, snake_case envelope validation, secrets-at-rest pattern reused, no secrets in logs.
