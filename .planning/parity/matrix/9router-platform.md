# 9router → g0router Platform Parity Matrix

## Row Table

| ID | Behavior | Evidence (file:line) | g0router status | Notes |
|----|----------|----------------------|-----------------|-------|
| PAR-PLAT-001 | Proxy pool CRUD: list with `isActive` filter and `includeUsage` bound-connection count | `src/app/api/proxy-pools/route.js:44-75` | HAVE | w7-plat-1: `GET /api/proxy-pools[?isActive]`, `GET /{id}`; `internal/admin/proxypools.go`, `internal/store/proxypools.go` |
| PAR-PLAT-002 | Proxy pool create with validated `type` ∈ {http,vercel,cloudflare,deno} | `src/app/api/proxy-pools/route.js:10,78-92` | HAVE | w7-plat-1: `POST /api/proxy-pools` with validated name/host/port + protocol (http/https/socks5). Relay-deploy types (vercel/cloudflare/deno) remain PAR-PLAT-006/007/008 |
| PAR-PLAT-003 | Proxy pool update/delete with bound-connection guard (409 if in use) | `src/app/api/proxy-pools/[id]/route.js:94-122` | HAVE | w7-plat-1: `PUT/DELETE /{id}`; DELETE returns 409 via `CountConnectionsUsingProxyPool` (connections.proxy_pool_id) |
| PAR-PLAT-004 | Proxy pool health test: HTTP uses `undici` ProxyAgent HEAD to google.com; relay types use `x-relay-target` probe | `src/app/api/proxy-pools/[id]/test/route.js:6-48` | HAVE | w7-plat-1: `POST /{id}/test` HEAD-via-proxy probe (SSRF-guarded, injectable prober); `platform.ProxyPoolService.TestConnectivity` |
| PAR-PLAT-005 | Proxy pool test writes `testStatus`, `lastTestedAt`, `lastError`, toggles `isActive` | `src/app/api/proxy-pools/[id]/test/route.js:49-56` | HAVE | w7-plat-1: test persists `last_check_status`/`last_check_at` via `SetProxyPoolCheck` |
| PAR-PLAT-006 | Cloudflare Workers relay deploy: uploads worker script via multipart/form-data, enables subdomain, creates pool entry | `src/app/api/proxy-pools/cloudflare-deploy/route.js:50-145` | MISSING | |
| PAR-PLAT-007 | Vercel relay deploy: posts to `/v13/deployments` with edge function files, polls readyState, disables ssoProtection, creates pool entry | `src/app/api/proxy-pools/vercel-deploy/route.js:58-141` | MISSING | |
| PAR-PLAT-008 | Deno Deploy relay deploy: creates app via Deno v2 API, deploys asset, polls revision status (max 30×2s), rollback on failure | `src/app/api/proxy-pools/deno-deploy/route.js:47-175` | MISSING | |
| PAR-PLAT-009 | Proxy pool SQLite schema: `proxyPools(id,isActive,testStatus,data,createdAt,updatedAt)` with JSON `data` column | `src/lib/db/schema.js:60-73` | HAVE | w7-plat-1: additive `proxy_pools` table (typed columns; password_enc at rest) + connections.proxy_pool_id (per-connection proxy resolution, selection.go hook); `internal/store/migrate.go` |
| PAR-PLAT-010 | Provider node CRUD: types `openai-compatible`/`anthropic-compatible`/`custom-embedding` with prefix + baseUrl | `src/app/api/provider-nodes/route.js:20-103` | HAVE | w7-platnodes: full CRUD (list/create/get/update/delete) over the providers table extended with `prefix`/`api_type` (`internal/admin/nodes.go`, `internal/platform/providernodes.go`, `internal/store/providernodes.go`); list filters the three node types; `{id}` routes registered (`routes_admin.go`). |
| PAR-PLAT-011 | Provider node baseUrl sanitization: strips trailing `/messages` for anthropic, `/embeddings` for custom-embedding | `src/app/api/provider-nodes/route.js:66-69,83-87` | HAVE | w7-platnodes: `platform.SanitizeNodeBaseURL(nodeType, raw)` strips trailing `/messages` (anthropic-compatible) and `/embeddings` (custom-embedding), trims a trailing slash, idempotent; applied on create+update (`internal/platform/providernodes.go`). |
| PAR-PLAT-012 | Provider node update cascades prefix/baseUrl/apiType to bound `providerConnections` | `src/app/api/provider-nodes/[id]/route.js:61-74` | HAVE | w7-platnodes: a node IS a providers row; `ProviderNodeService.Update` persists the sanitized base_url/api_type/prefix on the row, from which bound connections (no base URL of their own) resolve transitively at request time (`internal/platform/providernodes.go`). |
| PAR-PLAT-013 | Provider node validation: tests `/models` first, falls back to `/chat/completions` if `modelId` provided; custom-embedding tests `/embeddings` POST | `src/app/api/provider-nodes/validate/route.js:52-201` | HAVE | w7-platnodes: `ValidateProviderNode` runs the probe through an injectable `NodeProber` (hermetic in tests), SSRF-guarded via `platform.IsBlockedTarget` before dialing; api_key used transiently, never persisted/echoed (`internal/admin/nodes.go`, `internal/platform/providernodes.go`). |
| PAR-PLAT-014 | Provider node SQLite schema: `providerNodes(id,type,name,data,createdAt,updatedAt)` with JSON `data` | `src/lib/db/schema.js:49-58` | HAVE | w7-platnodes (ESC-SCHEMA): additive typed columns `prefix`/`api_type` on the existing `providers` table (`ensureColumn`, `internal/store/migrate.go`) rather than a separate JSON-`data` table — continues the w6-f node-on-providers mapping; enables prefix lookup + cascade. |
| PAR-PLAT-015 | cloudflared tunnel: auto-downloads platform-specific binary from GitHub releases, validates magic bytes | `src/lib/tunnel/cloudflare/cloudflared.js:43-184` | HAVE | w7-plat-2 — pure magic-byte validator `isValidExecutable` (ELF/Mach-O/PE) unit-tested (`internal/platform/tunnel/cloudflared.go`); binary download is integration-only (needs network — §1.9 / ESC-OS-PRIV) |
| PAR-PLAT-016 | cloudflared spawns `tunnel run --token <token>` with DNS resolver 1.1.1.1:53, resolves after 4 `Registered tunnel connection` log lines | `src/lib/tunnel/cloudflare/cloudflared.js:195-269` | HAVE | w7-plat-2 — named-tunnel state machine (enable→active) unit-tested via fake `Runner` (`internal/platform/tunnel/service.go`); real `tunnel run --token` spawn is integration-only (`cloudflared.go`, §1.9) |
| PAR-PLAT-017 | cloudflared quick tunnel: `tunnel --url http://127.0.0.1:<port> --no-autoupdate`, extracts `*.trycloudflare.com` URL from logs | `src/lib/tunnel/cloudflare/cloudflared.js:275-404` | HAVE | w7-plat-2 — pure `extractQuickTunnelURL` (`*.trycloudflare.com`) unit-tested on canned stderr (`internal/platform/tunnel/cloudflared.go`); real quick-tunnel spawn is integration-only (§1.9) |
| PAR-PLAT-018 | cloudflared kill: intentionalKill flag suppresses unexpected-exit handler, pkill by port regex | `src/lib/tunnel/cloudflare/cloudflared.js:186-188,407-438` | HAVE | w7-plat-2 — `Runner.Stop` + disable→inactive state machine unit-tested via fake (`service.go`); real process kill (context-cancel) is integration-only (`cloudflared.go`, §1.9) |
| PAR-PLAT-019 | Tailscale install: brew/pkg on mac, install.sh on Linux, MSI + UAC PowerShell on Windows | `src/lib/tunnel/tailscale/tailscale.js:210-410` | HAVE | w7-plat-2 — integration-only / OS-privileged (package install / root); behind `Runner` (`internal/platform/tunnel/tailscale.go`); escalated (ESC-OS-PRIV) |
| PAR-PLAT-020 | Tailscale daemon: userspace-networking fallback (no sudo) or TUN mode with sudo password; custom socket in data dir | `src/lib/tunnel/tailscale/tailscale.js:465-547` | HAVE | w7-plat-2 — enable/disable/status state machine unit-tested via fake (`service.go`); default userspace-networking (no TUN/root); daemon/poll + TUN mode integration-only / OS-privileged (ESC-OS-PRIV / ESC-TS-MODE) |
| PAR-PLAT-021 | Tailscale login: parses auth URL from stdout or `status --json` (Windows), polls 15s | `src/lib/tunnel/tailscale/tailscale.js:573-669` | HAVE | w7-plat-2 — pure `extractTailscaleLoginURL` unit-tested on canned output (`tailscale.go`); the 15s poll loop is integration-only (real sleeps — §1.9) |
| PAR-PLAT-022 | Tailscale funnel: `funnel --bg <port>`, resolves URL from `Self.DNSName`, handles `Funnel is not enabled` with enableUrl | `src/lib/tunnel/tailscale/tailscale.js:672-747` | HAVE | w7-plat-2 — funnel-mode state machine (URL via `Status().URL`) unit-tested via fake (`service.go`); real `tailscale funnel` is integration-only (§1.9) |
| PAR-PLAT-023 | Tailscale cert provisioning: `tailscale cert --cert-file/--key-file` best-effort | `src/lib/tunnel/tailscale/tailscale.js:750-766` | HAVE | w7-plat-2 — behind `Runner`; real `tailscale cert` is integration-only / best-effort (§1.9) |
| PAR-PLAT-024 | MITM HTTPS server on port 443: SNI cert cache, ALPN h2/1.1 negotiation, upstream IP resolution via 8.8.8.8 | `src/mitm/server.js:36-206` | HAVE | w7-plat-3 — SNI leaf minting + CA signing + chain verify + leaf cache + the `GetCertificate` closure unit-tested hermetically (`internal/platform/mitm/{ca,service}_test.go`); the live TLS reverse-proxy listener (`proxy.go` net.Listen+tls.NewListener+intercept-forward) is integration-only (§1.9) |
| PAR-PLAT-025 | MITM Root CA: generates on first start, auto-installs to system trust store (mac keychain, Win certmgr, Linux update-ca-trust) | `src/mitm/manager.js:512-550` | HAVE | w7-plat-3 — root CA gen + CA-cert serving as raw PEM (`GET /api/mitm/ca-cert`, `application/x-pem-file`) fully unit-tested; key at rest as a 0o600 file under the data dir (ESC-CA-STORE A, mirrors secret.go), never served/logged. System-trust-store auto-install is OS-privileged → DEFERRED/escalated (ESC-OS-PRIV) |
| PAR-PLAT-026 | MITM hosts file patching: per-tool DNS entries for antigravity/copilot/kiro/cursor; atomic write on Windows; sudo tee on Unix | `src/mitm/dns/dnsConfig.js:146-180` | MISSING | |
| PAR-PLAT-027 | MITM hosts file restore on startup and cleanup on shutdown (sync + async paths) | `src/mitm/manager.js:248-259,715-767` | MISSING | |
| PAR-PLAT-028 | MITM restart backoff: 5 attempts with delays [5s,10s,20s,30s,60s], reset after 60s | `src/mitm/manager.js:401-443` | HAVE | w7-plat-3 — per-tool MITM config + global/per-tool toggle (`GET /api/mitm/status`, `POST /api/mitm/toggle`, `POST /api/mitm/tools/{id}`) fully unit-tested via `newTestEnv`; pure `nextBackoff` policy (1s doubling, capped 30s, max 5) unit-tested; the start retry loop's real sleeps are integration-only (§1.9 / ESC-BACKOFF) |
| PAR-PLAT-029 | MITM sudo password: AES-256-GCM encrypted at rest with machine-id derived key | `src/mitm/manager.js:96-183` | MISSING | |
| PAR-PLAT-030 | Tray: macOS/Linux via systray2/systray Go binary; Windows via PowerShell NotifyIcon | `cli/src/cli/tray/tray.js:44-54,114-139` | MISSING | |
| PAR-PLAT-031 | Tray menu: status, open dashboard, auto-start toggle, quit | `cli/src/cli/tray/tray.js:59-69` | MISSING | |
| PAR-PLAT-032 | Auto-start: launchd plist (mac), VBS in Startup folder (Win), .desktop (Linux) | `cli/src/cli/tray/autostart.js:47-299` | MISSING | |
| PAR-PLAT-033 | Auto-start self-awareness: macOS `isAgentSelfMacOS` skips `launchctl unload` to avoid SIGTERM on self | `cli/src/cli/tray/autostart.js:127-139,197,225` | MISSING | |
| PAR-PLAT-034 | Auto-update: detached updater process runs `npm i -g <pkg>@latest`, exposes progress via HTTP server on 127.0.0.1:20129 | `src/lib/updater/updater.js:67-88,131-173` | MISSING | |
| PAR-PLAT-035 | Auto-update wait-for-exit: polls app port until free (3-15s), retries install up to 3×, relaunches app, opens dashboard | `src/lib/updater/updater.js:90-125,184-196` | MISSING | |
| PAR-PLAT-036 | Auto-update kill strategy: kills MITM by PID file, collects all app PIDs (node.exe, cloudflared, tray) before update | `src/lib/appUpdater.js:139-157` | MISSING | |
| PAR-PLAT-037 | CLI tools auto-config: Claude Code (env vars), Codex CLI (TOML), Factory Droid, Open Claw, OpenCode, Hermes | `cli/src/cli/menus/cliTools.js:583-616` | MISSING | |
| PAR-PLAT-038 | CLI tools API: `GET/POST /api/cli/providers/<tool>`, reads/writes tool-specific config files | `cli/src/cli/menus/cliTools.js:29-120` | MISSING | |
| PAR-PLAT-039 | Cursor OAuth auto-import: scans `state.vscdb` SQLite (better-sqlite3 → sqlite3 CLI fallback), extracts accessToken + machineId | `src/app/api/oauth/cursor/auto-import/route.js:177-258` | MISSING | |
| PAR-PLAT-040 | Cursor token validation: length check, UUID machineId regex, skips API call (protobuf) | `src/lib/oauth/services/cursor.js:95-125` | MISSING | |
| PAR-PLAT-041 | Cursor checksum: XOR timestamp bytes with rolling key 165, base64 encode | `src/lib/oauth/services/cursor.js:27-40` | MISSING | |
| PAR-PLAT-042 | Kiro auto-import: scans `~/.aws/sso/cache` for refreshToken starting with `aorAAAAAG` | `src/app/api/oauth/kiro/auto-import/route.js:10-84` | MISSING | |
| PAR-PLAT-043 | Codex import-token: accepts raw access token, decodes JWT payload for email/workspace/plan, stores as `access_token` authType | `src/app/api/oauth/codex/import-token/route.js:12-96` | MISSING | |
| PAR-PLAT-044 | Database export/import: `GET/POST /api/settings/database` exports full localDb JSON, re-applies proxy env after import | `src/app/api/settings/database/route.js:5-36` | MISSING | |
| PAR-PLAT-045 | g0router provider CRUD: flat providers table (id,name,type,base_url,enabled) with no prefix/node abstraction | `internal/store/providers.go:10-116` | HAVE | Equivalent to 9router base provider list, not provider-nodes |
| PAR-PLAT-046 | g0router connection CRUD: stores secret_enc/access_token_enc/refresh_token_enc with at-rest encryption | `internal/store/connections.go:10-147` | HAVE | Equivalent to 9router providerConnections |
| PAR-PLAT-047 | g0router OAuth flow: PKCE authorization-code for anthropic only; start + callback + refresh endpoints | `internal/auth/oauth.go:33-160` | PARTIAL | Only anthropic; no cursor/codex/kiro/iflow/gitlab flows |
| PAR-PLAT-048 | g0router admin routes: `/api/providers`, `/api/connections`, `/api/settings`, `/api/oauth/{provider}` | `internal/server/routes_admin.go:27-51` | PARTIAL | No proxy-pool, provider-node, tunnel, MITM, or CLI-tool routes |
| PAR-PLAT-049 | g0router platform package is a placeholder doc.go referencing RTK, caveman, combos, translator, sync, proxy-pool | `internal/platform/doc.go:1-5` | MISSING | Phase 11+ per comment |
| PAR-PLAT-050 | g0router no tray, no auto-start, no auto-update, no MITM, no tunnel binaries | `cmd/g0router/main.go:1-80` | MISSING | Single binary HTTP server only |

## Data models

### 9router proxyPools
```
proxyPools
  id TEXT PRIMARY KEY
  isActive INTEGER DEFAULT 1
  testStatus TEXT
  data TEXT NOT NULL        -- JSON: name, proxyUrl, noProxy, type, strictProxy, lastTestedAt, lastError
  createdAt TEXT NOT NULL
  updatedAt TEXT NOT NULL
```

### 9router providerNodes
```
providerNodes
  id TEXT PRIMARY KEY
  type TEXT                 -- openai-compatible | anthropic-compatible | custom-embedding
  name TEXT
  data TEXT NOT NULL        -- JSON: prefix, apiType, baseUrl
  createdAt TEXT NOT NULL
  updatedAt TEXT NOT NULL
```

### 9router providerConnections (g0router equivalent: connections)
```
providerConnections
  id TEXT PRIMARY KEY
  provider TEXT NOT NULL
  authType TEXT NOT NULL
  name TEXT
  email TEXT
  priority INTEGER
  isActive INTEGER DEFAULT 1
  data TEXT NOT NULL        -- JSON: accessToken, refreshToken, expiresAt, providerSpecificData, testStatus, ...
  createdAt TEXT NOT NULL
  updatedAt TEXT NOT NULL
```

### g0router providers
```
providers
  id TEXT PRIMARY KEY
  name TEXT NOT NULL
  type TEXT NOT NULL
  base_url TEXT NOT NULL DEFAULT ''
  enabled INTEGER NOT NULL DEFAULT 1
  created_at INTEGER NOT NULL
  updated_at INTEGER NOT NULL
```

### g0router connections
```
connections
  id TEXT PRIMARY KEY
  provider_id TEXT NOT NULL
  name TEXT NOT NULL
  kind TEXT NOT NULL        -- api_key | oauth
  secret_enc TEXT NOT NULL DEFAULT ''
  access_token_enc TEXT NOT NULL DEFAULT ''
  refresh_token_enc TEXT NOT NULL DEFAULT ''
  expires_at INTEGER NOT NULL DEFAULT 0
  metadata TEXT NOT NULL DEFAULT ''
  created_at INTEGER NOT NULL
  updated_at INTEGER NOT NULL
```

## Edge cases and quirks

- **Proxy pool type validation inconsistency**: POST allows `deno` type; PUT update allows only `http`/`vercel`/`cloudflare` (`src/app/api/proxy-pools/[id]/route.js:41`). Deno pools cannot change type after creation.
- **Proxy test auto-deactivates pool on failure**: `updateProxyPool(id, { isActive: result.ok })` silently disables the pool if the test fails (`src/app/api/proxy-pools/[id]/test/route.js:55`).
- **Vercel deploy disables SSO protection unconditionally**: PATCHes `ssoProtection: null` on the project without checking if the user wants it (`src/app/api/proxy-pools/vercel-deploy/route.js:114-121`).
- **Deno deploy rollback**: on failure, deletes the created app via Deno v2 API (`src/app/api/proxy-pools/deno-deploy/route.js:116-119,147-150`).
- **cloudflared binary magic-byte validation**: checks PE/ELF/Mach-O magic to detect truncated downloads (`src/lib/tunnel/cloudflare/cloudflared.js:120-135`).
- **Tailscale sudo password injection guard**: rejects passwords containing newlines to prevent stdin command injection (`src/lib/tunnel/tailscale/tailscale.js:307-308`).
- **Tailscale install script persisted to temp file**: instead of piping curl→sudo sh, writes to tmp file with 0o700 mode before sudo execution (`src/lib/tunnel/tailscale/tailscale.js:324-327`).
- **MITM port 443 kill with forceKillPort443 flag**: manager exposes `forceKillPort443` param that kills the owning process without confirmation (`src/mitm/manager.js:467-510`).
- **MITM Windows spawns on port 8443, not 443**: `MITM_WIN_NODE_PORT = 8443` is declared but never used; actual server still listens on 443 (`src/mitm/manager.js:42`).
- **Cursor auto-import Linux guard**: checks `which cursor` or desktop file to avoid importing from stale configs of uninstalled IDE (`src/app/api/oauth/cursor/auto-import/route.js:201-219`).
- **Kiro token prefix heuristic**: looks for refreshToken starting with `aorAAAAAG` to distinguish from other AWS SSO cache entries (`src/app/api/oauth/kiro/auto-import/route.js:35,54`).
- **Updater process outlives parent**: detached + unref Node process survives Next.js server exit to run npm install (`src/lib/appUpdater.js:177-179`).

## Go-port considerations

- Replace Node `child_process` spawns with `os/exec`; bundle cloudflared/tailscale binaries or download to `os.UserCacheDir()`.
- MITM CA proxy becomes a Go `crypto/tls` + `net/http` reverse proxy; use `mkcert`-style local CA instead of Node `https` module.
- Tray/auto-start: drop for server binary; add `g0router service install` command generating systemd/launchd plist.
- Auto-update: `go-selfupdate` or in-app binary patch via `github.com/inconshreveable/go-update`; no npm lifecycle.
- CLI tool config: shell out to `jq`/`plutil` or write TOML/JSON directly; avoid Node `fs` abstractions.
- hosts file patching: use `os.Hostname()` loopback guard; on Windows call `iphlpapi.DnsFlushResolverCache` via syscall or `exec ipconfig`.
