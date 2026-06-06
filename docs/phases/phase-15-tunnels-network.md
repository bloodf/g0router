# Phase 15: Tunnels & Network

> Process, contracts, gates: see `docs/phases/STAGE-13-19-PROCESS.md`.
> Security review: **mandatory** at checkpoint (binary downloads, CLI shelling).

## Goal
Cloudflare and Tailscale tunnel management with safe CLI integration, tunnel
health checks, and proxy connectivity testing.

## Features (7)
1. Cloudflare Tunnel creation/management
2. Tailscale funnel enable/disable
3. Tunnel health ping (background, 60s)
4. Tunnel endpoint display
5. Tunnel dashboard access toggle
6. Outbound proxy test
7. Proxy pool auto health checks (background)

## Supply-Chain & Privilege Rules (NON-NEGOTIABLE)
- `cloudflared` download: **pinned version + pinned SHA-256 per OS/arch**,
  hardcoded constants in Go. Verify checksum before chmod+exec. Refuse on
  mismatch. Download over HTTPS from the official GitHub releases URL only.
- **Tailscale is NOT downloaded/installed by g0router.** It requires root and a
  system daemon. g0router only drives an already-installed `tailscale` binary
  found on `$PATH`. If absent → `409` with install instructions in `error`.
- All `exec.Command` calls use absolute paths + fixed arg slices. No shell
  interpolation, no user input in args (validate tunnel names: `[a-z0-9-]{1,63}`).
- Child processes get a context for cancellation; killed on shutdown.
- Never log tokens printed by the CLIs.

## New Database Tables
```sql
CREATE TABLE IF NOT EXISTS tunnel_config (
    id INTEGER PRIMARY KEY,
    type TEXT NOT NULL UNIQUE,        -- 'cloudflare' | 'tailscale'
    is_enabled INTEGER DEFAULT 0,
    config_enc TEXT,                  -- encrypted (tokens/credentials inside)
    url TEXT,
    status TEXT DEFAULT 'inactive',   -- 'inactive' | 'starting' | 'active' | 'error'
    last_error TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## New API Endpoints
- `GET /api/tunnels` — `[{type, is_enabled, url, status, last_error}]`
- `POST /api/tunnels/cloudflare` — download (verified) cloudflared if missing, create + run tunnel
- `DELETE /api/tunnels/cloudflare` — stop process, mark inactive
- `POST /api/tunnels/tailscale` — enable `tailscale funnel` (requires preinstalled binary)
- `DELETE /api/tunnels/tailscale` — disable funnel
- `GET /api/tunnels/health` — live reachability check of active tunnel URLs
- `POST /api/settings/proxy-test` — `{url}` test outbound proxy connectivity

All tunnel mutations: admin-session or bearer auth + audit rows.

## Background Jobs
- Tunnel health: every 60s, HTTP GET tunnel URL `/healthz`, update `status`/`last_error`.
- Proxy pool health: every 5min, test each active pool, update `last_check_*`.
- Both: started from server startup (no `init()`), stopped via context on shutdown.

## Tasks
1. `phase-15/task-1`: store — tunnel_config (encrypted config) + tests
2. `phase-15/task-2`: tunnel package — `internal/tunnel/`: checksum-verified download, process supervisor (fake binaries in tests) + tests
3. `phase-15/task-3`: handlers — tunnel CRUD/health + proxy-test + tests
4. `phase-15/task-4`: background health loops (tunnel + proxy pools) + tests
5. `phase-15/checkpoint` (incl. security pass)

## Test Requirements (minimum)
- Checksum mismatch → download rejected, nothing executed
- Tunnel name validation rejects shell metacharacters
- Process supervisor: start, capture URL from output (fake script), stop on context cancel
- Tailscale absent from PATH → 409 with instructions
- Health loop flips status on unreachable URL; stops cleanly on shutdown
- Config round-trips encrypted; API responses never expose tokens
- Proxy-test returns `{ok, latency_ms, error}`, never 500 on bad proxy

## Outcome

All 7 features implemented. Supply-chain security enforced with pinned checksums. No privilege escalation.

### Shipped
- **task-1**: `internal/store/tunnels.go` — `tunnel_config` CRUD with AES-GCM encrypted `config_enc`
- **task-2**: `internal/tunnel/download.go` — pinned `cloudflared` download with SHA-256 verification per OS/arch; `internal/tunnel/supervisor.go` — process supervisor with URL capture from stdout, context cancellation kill; `internal/tunnel/tunnel.go` — Manager orchestrating store + supervisor
- **task-3**: `api/handlers/tunnels.go` — tunnel CRUD (cloudflare/tailscale create/delete), health check, proxy-test; `api/handlers/tunnels_test.go`
- **task-4**: `api/server.go` — background tunnel health (60s) and proxy pool health (5min) loops with panic recovery
- **task-coverage**: Tunnel package error branch coverage to maintain 95.0%

### Security Review
- Input validation: ✅ Tunnel names `[a-z0-9-]{1,63}`; port numeric validation; no shell interpolation
- Authn/authz: ✅ Tunnel mutations require admin session or bearer auth + audit rows
- Secrets at rest: ✅ Tunnel config encrypted with AES-GCM
- Secrets in logs: ✅ CLI stderr discarded; tokens never logged
- Supply-chain: ✅ Cloudflared pinned version + SHA-256; HTTPS from GitHub releases only; checksum verified before chmod+exec
- Privilege requirements: ✅ Tailscale not downloaded by g0router; requires preinstalled binary on PATH; 409 with instructions if absent

### Gates
- `go test ./... -count=1`: PASS
- `go vet ./...`: PASS
- `go test -race ./...`: PASS
- `go build ./cmd/g0router`: PASS
- Coverage: 95.0%

### Commit Range
`f8e2943` → `39aa6d7`

## Commit Message (final)
`phase-15/tunnels-network: cloudflare, tailscale funnel, health loops`
