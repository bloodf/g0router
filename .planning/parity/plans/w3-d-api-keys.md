# w3-d — API key system: table, machineId+CRC format, /v1 gating, CLI token

Rows: PAR-AUTH-029 (apiKeys table with machineId — `src/lib/db/schema.js:74-84`: id, key UNIQUE, name, machineId, isActive, createdAt + key index), PAR-AUTH-010 (key format `sk-{machineId}-{keyId}-{crc8}` — `src/shared/utils/apiKey.js:34-38`; keyId = 6 chars [a-z0-9] `:8-15`; crc8 = first 8 hex of HMAC-SHA256(machineId+keyId, API_KEY_SECRET) `:20-26`, env `API_KEY_SECRET` default "endpoint-proxy-api-key-secret" `:3`), PAR-AUTH-009 (remote API-key validation — `src/dashboardGuard.js:106-116`), PAR-AUTH-008 (loopback no-key access — `src/dashboardGuard.js:35,102-104,118-122`), PAR-AUTH-012 (CLI token auth — `src/dashboardGuard.js:6-19`: header `x-9r-cli-token`, token = `getConsistentMachineId("9r-cli-auth")`; machineId derivation `src/shared/utils/machineId.js:49-54`: SHA256(rawMachineId + salt + cliSecret)[:16]). Scope per `WAVE-3-MAP.md` track 1, plan 4 + w3-b v4's explicit transfer: "PAR-AUTH-008/009 … are ENTIRELY w3-d's: gating + validators land together so remote /v1 is never broken in an interim state." Frozen ref @ 827e5c3. Depends on w3-b MERGED (guard live with inert `APIKeyValidator`/`CLITokenValidator` fields + passthrough /v1 step).

In-repo integration: `internal/server/guard.go` (w3-b — this plan REPLACES the passthrough step 3 with the ref's loopback/key gating and injects both validators in `server.go` wiring), `internal/store/migrate.go` (additive-only `ensureColumn` migrations per AGENTS.md decisions), HMAC key source = REF PARITY EXACTLY: env `API_KEY_SECRET`, default `"endpoint-proxy-api-key-secret"` (`apiKey.js:3`) — no alternative secret scheme.

## Ref behavior to port

- **Table** (PAR-AUTH-029): new `api_keys` store table mirroring `schema.js:74-84`
  (snake_case columns per g0router style; key UNIQUE + index). CRUD:
  `CreateAPIKey(name)` (generates the formatted key), `ListAPIKeys`,
  `SetAPIKeyActive(id, active)`, `DeleteAPIKey(id)`, `GetAPIKeyByKey(key)`.
  Admin routes are REF-EVIDENCED, not invented: `src/app/api/keys/route.js` (GET list
  :7-8; POST create :18-30 — machineId always taken server-side via
  `getConsistentMachineId()` :28-29) + `src/app/api/keys/[id]/` (per-key routes), and
  the guard PROTECTED_API_PATHS lists "/api/keys" (`dashboardGuard.js:50`). Port GET/
  POST/[id] behavior from those files (read whole) behind the guard's protected set.
- **Format** (PAR-AUTH-010): `GenerateAPIKey(machineID)` → `sk-{machineId}-{keyId}-{crc8}`
  exactly as `:8-38`; `ParseAPIKey` supports new format AND legacy `sk-{random8}`
  (`:42-48` "Supports both formats"). Validation = parse → CRC recompute match →
  DB lookup (key exists + isActive) — port the guard's `canAccessPublicLlmApi`
  remote-key path (`dashboardGuard.js:106-116`, read whole).
- **machineId** (PAR-AUTH-012 dependency): `MachineID(salt)` =
  hex(SHA256(raw + salt [+ cliSecret when salt=="9r-cli-auth"]))[:16]
  (`machineId.js:49-54`); raw machine id source = stable per-host identifier — port
  `loadRawMachineId` (read `machineId.js` whole; use the same os-level sources where
  portable, else the existing store secret as the stable raw — document choice in a
  comment with the ref cite).
- **/v1 gating** (PAR-AUTH-008/009): replace w3-b's passthrough step with
  `dashboardGuard.js:118-122` semantics: loopback request (w3-b's `isLocalRequest`)
  → allow keyless; remote → `APIKeyValidator` (Authorization Bearer or x-api-key —
  port the exact header extraction from `:106-116`) → invalid/absent → 401
  `{"error":"API key required for remote API access"}`.
- **CLI token** (PAR-AUTH-012): `CLITokenValidator` = header `x-9r-cli-token` equals
  `MachineID("9r-cli-auth")` (g0router header name: keep `x-9r-cli-token` for wire
  parity). Wire BOTH validators into the guard at `server.go` construction.

## Preconditions (a "0 hits" grep exits 1 = pass)

- `grep -c 'TestGuardPublicLlmApiPassthrough' internal/server/guard_test.go` ≥ 1 (w3-b merged with passthrough — this plan supersedes that test)
- `grep -rn 'api_keys\|GenerateAPIKey' internal/` → 0 hits (new)
- `grep -c 'ensureColumn\|CREATE TABLE' internal/store/migrate.go` ≥ 1 (additive migration pattern to follow)

## Exclusive file ownership

NEW: `internal/auth/apikey.go` + `apikey_test.go` (format/CRC/machineId/parse),
`internal/store/apikeys.go` + `apikeys_test.go` (table + CRUD),
`internal/admin/apikeys.go` + `apikeys_test.go` (/api/keys handlers).
TOUCH: `internal/server/guard.go` + `guard_test.go` (replace passthrough step 3 with
gating; replace `TestGuardPublicLlmApiPassthrough` with the gating tests below),
`internal/server/server.go` (inject the two validators), `internal/store/migrate.go`
(api_keys table), `internal/server/routes_admin.go` (register `/api/keys` + `/api/keys/{id}` routes).
NOT touched: limiter/login (w3-a), OIDC (w3-c), provider OAuth (w3-f).

## Tasks (each: STEP (a) named failing tests FIRST, run, show fail; STEP (b) implement)

1. **Key format + machineId** (`internal/auth/apikey.go`). Tests FIRST:
   `TestGenerateAPIKeyFormat` (regex `^sk-[0-9a-f]{16}-[a-z0-9]{6}-[0-9a-f]{8}$`),
   `TestCRCRecomputeMatches` (same machineId+keyId+secret → same crc; secret change
   → mismatch), `TestParseAPIKeyNewAndLegacy` (both formats per `:42-48`),
   `TestMachineIDDerivation` (16 hex chars; salt changes output; cli salt mixes
   cliSecret), `TestMachineIDStable` (two calls equal).
2. **Store** (`internal/store/apikeys.go` + migrate). Tests FIRST:
   `TestAPIKeyCRUD` (create/list/toggle/delete; key UNIQUE violation), 
   `TestAPIKeyLookupByKey`, `TestMigrationAdditive` (existing DB upgrades cleanly).
3. **Validators + guard gating** (`guard.go`, `server.go`). Tests FIRST:
   `TestGuardV1LoopbackKeyless` (loopback /v1 allowed without key — PAR-AUTH-008),
   `TestGuardV1RemoteRequiresKey` (remote /v1 no key → 401 exact error body),
   `TestGuardV1RemoteValidKey` (created+active key in Authorization Bearer AND in
   x-api-key → allowed; inactive → 401; CRC-corrupted → 401 without DB hit),
   `TestGuardCLIToken` (correct x-9r-cli-token passes ALWAYS_PROTECTED/api checks;
   wrong → 401).
4. **Admin routes** (`internal/admin/apikeys.go`). Tests FIRST:
   `TestKeysCRUDEndpoints` (session-guarded; snake_case `{data,error}` envelope;
   response shapes ported from `src/app/api/keys/route.js` GET/POST (read whole — the create response includes the full key + machineId per :30-36; the list response per the GET handler) — no invented masking policy).

## Binary acceptance criteria

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -c 'TestGuardPublicLlmApiPassthrough' internal/server/guard_test.go` → 0 (superseded by gating tests).
- `grep -rn 'func init(\|panic(' internal/auth/apikey.go internal/store/apikeys.go internal/admin/apikeys.go` → 0 hits.
- `TestGuardV1LoopbackKeyless`, `TestGuardV1RemoteRequiresKey`, `TestGuardV1RemoteValidKey`, `TestParseAPIKeyNewAndLegacy`, `TestMachineIDDerivation` pass.
- PAR-AUTH-008 and 009 flip HAVE here (gating + validators complete together).

## Out of scope

Key-scoped rate limits / per-key usage (Wave 5). UI keys page (Wave 6). The login
limiter (w3-a). Tunnel (Wave 7). Dashboard OIDC (w3-c).
