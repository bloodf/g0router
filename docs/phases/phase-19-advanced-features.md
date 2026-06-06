# Phase 19: Advanced Features

> Process, contracts, gates, architecture: see `docs/phases/STAGE-13-19-PROCESS.md`.
> Security review: **mandatory** at checkpoint (MITM CA, auto-updater).

## Goal
Semantic cache, version check + auto-update, locale persistence, WebSocket
chat channel, MITM proxy, skills catalog.

## Architecture
- `internal/semcache/` — domain: embedding fetch, cosine similarity, TTL,
  lookup/store decisions. Repository interface for persistence.
- `internal/update/` — domain: version compare, release fetch, checksum
  verify, staged self-replace.
- `internal/mitm/` — domain: CA generation, cert minting, proxy server.

## Deferred (decided now)
- **WebRTC** — WebSocket only this stage.
- **i18n translations** — backend stores locale preference only. UI ships
  `en` + `pt-BR` complete; other locales fall back to `en`. "33 locales" is
  out of scope (no translation source).
- **Landing page, skills page UI** — Lovable's job. Backend ships only the
  skills catalog endpoint.

## Features (6 backend)
1. Semantic cache `[flag: semantic_cache]`
2. Version info + update check + opt-in self-update
3. Locale get/set (settings-backed)
4. WebSocket chat endpoint `[flag: websocket_chat]`
5. MITM proxy `[flag: mitm_proxy]`
6. Skills catalog endpoint (static embedded JSON)

## New Database Tables
```sql
CREATE TABLE IF NOT EXISTS semantic_cache (
    id INTEGER PRIMARY KEY,
    cache_key TEXT NOT NULL,           -- sha256(normalized prompt + model)
    embedding_json TEXT NOT NULL,
    model TEXT NOT NULL,
    response_json TEXT NOT NULL,
    expires_at DATETIME,
    hit_count INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_semantic_cache_model ON semantic_cache(model);
CREATE INDEX IF NOT EXISTS idx_semantic_cache_expires ON semantic_cache(expires_at);
```

## New API Endpoints
- `GET /api/cache/semantic` — stats + entries (key, model, hits, expires; NOT full responses)
- `DELETE /api/cache/semantic` — clear (audited)
- `GET /api/version` — `{version, go_version, build_date}`
- `POST /api/update/check` — `{current, latest, update_available, changelog_url}`
- `POST /api/update/apply` — opt-in: download + verify + stage (audited; admin only)
- `GET /api/locale` / `POST /api/locale` — settings-backed `{locale}`
- `GET /api/ws` — WebSocket upgrade
- `GET /api/mitm/status`, `POST /api/mitm/toggle`, `GET /api/mitm/ca-cert`, `PUT /api/mitm/tools/:tool`
- `GET /api/skills` — embedded static catalog `[{name, category, description, url}]`

## Semantic Cache Implementation
- **Similarity computed in Go**, not SQL — SQLite cannot dot-product JSON
  arrays. Lookup: exact `cache_key` hit first (free); else load candidate
  embeddings for same model (cap 500 newest, non-expired), cosine in Go,
  threshold 0.95 (single global setting `semantic_cache_threshold`).
- Embeddings via existing OpenAI-compatible provider connection; if no
  embedding-capable connection configured → cache disabled with logged reason
  (never blocks requests).
- Only non-streaming `/v1/chat/completions`; sits AFTER guardrails, BEFORE
  dispatch. TTL from `cache_ttl_seconds` setting. Expired rows purged lazily.

## Auto-Updater Implementation
- Check: GitHub releases API `repos/bloodf/g0router/releases/latest`, compare
  semver tag vs build version.
- Apply (explicit user action only, never automatic):
  1. Download release asset for OS/arch + its `checksums.txt`.
  2. Verify SHA-256 against checksum file. Mismatch → abort + audit.
  3. Stage at `DATA_DIR/update/g0router.new`; swap on next graceful shutdown.
- Release pipeline must publish `checksums.txt` — note as release-eng dependency.

## WebSocket Implementation
- `fasthttp/websocket` (gorilla port). Upgrade on `/api/ws`; auth same as
  `/api/*` (session cookie or bearer before upgrade).
- Protocol v1: client `{type:"chat", session_id, model, messages}` → server
  streams `{type:"delta", content}` … `{type:"done", usage}` / `{type:"error", error}`.
- Reuses inference dispatch; one in-flight chat per socket; context-cancel on close.

## MITM Implementation (highest risk — last)
- CA: ECDSA P-256, 10y, generated once → `DATA_DIR/mitm/ca.crt` + `ca.key`
  (key file mode 0600). Per-host leaf certs minted on demand, LRU cache.
- HTTPS proxy on `mitm_port` (default 8081), CONNECT interception → routes
  matching tool hosts to local inference; others tunneled through untouched.
- **No automatic /etc/hosts editing** (needs root, crash-rollback risk).
  `GET /api/mitm/status` returns per-tool instructions (hosts lines / proxy
  env vars) for the user to apply manually.
- Tools: Antigravity, Copilot, Cursor, Kiro — host lists as config constants.

## Tasks
1. `phase-19/task-1`: version/update-check + locale + skills endpoints + tests
2. `phase-19/task-2`: `internal/semcache/` domain + store + tests
3. `phase-19/task-3`: semantic cache dispatch wiring + handlers + tests
4. `phase-19/task-4`: `internal/update/` apply path (checksum verify, staged swap; fake release server in tests) + tests
5. `phase-19/task-5`: WebSocket endpoint + protocol + tests
6. `phase-19/task-6`: `internal/mitm/` CA + cert minting + proxy + handlers + tests
7. `phase-19/checkpoint` (incl. security pass)

## Test Requirements (minimum)
- Cosine: identical=1, orthogonal=0; threshold boundary; candidate cap respected
- Exact-key hit skips embedding call; expired entry not served; hit_count increments; flag off → bypass
- No embedding connection → requests still served, cache inert
- Update: checksum mismatch aborts, nothing staged; valid download staged; version compare handles `v`-prefix + prerelease
- WS: upgrade rejected without auth; chat round-trip streams deltas (fake provider); close cancels in-flight; flag off → 404
- MITM: CA generates once + persists; leaf valid for host, signed by CA; key file 0600; toggle off stops listener; non-tool hosts pass through
- Skills/locale/version endpoints return envelope shapes

## Commit Message (final)
`phase-19/advanced-features: semcache, updater, websocket, mitm, skills`
