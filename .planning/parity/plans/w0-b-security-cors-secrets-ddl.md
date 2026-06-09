# w0-b — security: CORS, secrets, DDL, admin validation (rev 2)

Rows: AUD-004, AUD-005, AUD-006, AUD-013, AUD-014, AUD-015. Runs AFTER w0-a merges (shared `internal/auth/oauth.go`).
Worker: M3. Reviewer: Kimi diff gate.

## Documented deviation (authorized — see `.planning/parity/GATE-RESOLUTION.md` "Addendum — AUD-004")
AUD-004's "rotate exposed ID" is not implementable: the ID is Anthropic's public Claude Code OAuth client identifier, hardcoded identically in 9router (`_refs/9router/src/lib/oauth/constants/oauth.js:21`). Authorized remediation per the logged addendum: configurability via env with the public ID as default.

## File ownership (exclusive while in flight)
- `internal/server/middleware.go`, `internal/server/server.go` (New signature), `internal/server/server_test.go`
- `cmd/g0router/main.go` (env read + pass-through only)
- `internal/auth/oauth.go` (client_id sourcing only), `internal/auth/auth_test.go`
- `internal/store/migrate.go`, `internal/store/migrate_test.go` (new)
- `internal/admin/handlers.go`, `internal/admin/connections.go`, `internal/admin/admin_test.go`

NOT `internal/config` — audit AUD-054 marks that stub package DELETE; do not resurrect it.

## Tasks (TDD order)

1. **AUD-015 CORS** (`middleware.go:33-40`): tests — (a) request with `Origin: https://evil.example` receives no `Access-Control-Allow-Origin` and no `Allow-Credentials`; (b) request with an allowlisted origin receives both. Fix: `server.New(uiFS, st)` → `server.New(uiFS, st, allowedOrigins []string)`; middleware echoes origin only if exact-match in the list. Default: empty list (UI is embedded same-origin — `internal/server/ui.go` serves it from the same host, so same-origin requests need no CORS headers at all). `cmd/g0router/main.go` populates the list from `G0ROUTER_ALLOWED_ORIGINS` (comma-separated; dev workflow uses `http://localhost:5173` for the Vite server).
2. **AUD-004 client_id** (`oauth.go:36`): tests — (a) `G0ROUTER_ANTHROPIC_CLIENT_ID=custom` → flows use `custom`; (b) unset → default `9d1c250a-e61b-44d9-88ed-5944d1962f5e` (per deviation note). Fix: read env once at construction (constructor param or struct field per existing auth construction pattern — read `internal/auth` constructors first); the literal moves to a single named constant `defaultAnthropicClientID` with a comment citing the 9router parity source.
3. **AUD-005 ensureColumn** (`migrate.go:105`): test `TestEnsureColumnRejectsBadNames` — any table/column name outside `^[a-z_][a-z0-9_]*$` returns an error before SQL executes.
4. **AUD-006 FKs** (`migrate.go:16-62`): AUD-006 requires FK constraints to exist; delete semantics per relationship:
   - `connections.provider_id → providers.id`: `ON DELETE CASCADE`. Parity evidence: PAR-ROUTE-005 (PARITY §3 maps PR #1421 "Clean up provider model aliases on provider delete" to it) — 9router cleans dependents on provider delete; current bare `DELETE` (`internal/store/providers.go:84-90`) would orphan rows once FKs are enforced.
   - Every other FK relationship found in the schema (`migrate.go:16-62`): `ON DELETE RESTRICT` — the no-data-destruction default; no parity row mandates cascade there, so none is added.
   Tests: (a) fresh DB → `PRAGMA foreign_key_list(connections)` non-empty; (b) temp DB seeded with provider+connection pre-migration → both rows survive migration; (c) `DeleteProvider` removes dependent connections; (d) `PRAGMA foreign_keys` reports ON. Implementation: versioned idempotent table-rebuild step (SQLite cannot add FKs in place) appended to the existing migration sequence.
5. **AUD-013 pathID** (`admin/handlers.go:25`): test — non-string route param → 400, not empty-string ID lookup. Fix: `pathID` returns `(string, bool)`; callers write 400 on false.
6. **AUD-014 UpdateConnection** (`admin/connections.go:124`): test — update with non-existent `ProviderID` → same status/error shape `CreateConnection` returns for that case (read `CreateConnection` first; mirror it).

## Acceptance (binary)
- Six test groups above exist; `go test ./...` green; `go vet ./...` clean.
- `grep -c "9d1c250a-e61b-44d9-88ed-5944d1962f5e" internal/auth/oauth.go` ≤ 1 and only as the named default constant.
- `PRAGMA foreign_key_list(connections)` non-empty on a fresh data dir (Task 4 test a).
- CORS tests (a)/(b) pass; no `Access-Control-Allow-Origin: *` or reflected-origin behavior remains.

## Out of scope
Rotating Anthropic's public client ID (impossible — see deviation). New OAuth flows. Session TTL. Settings-table-based origin config (env is sufficient for Wave 0; revisit only if a PAR-UI row requires runtime editing). Deleting stub packages (separate cleanup).
