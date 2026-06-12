# w5-g — Virtual keys: x-g0-vk routing + per-key quota

PAR rows: PAR-ROUTE-030 (`x-g0-vk` header routing), PAR-ROUTE-031 (per-key quota
tracking). PAR-ROUTE-032 (VirtualKey schema) is already HAVE
(`internal/schemas/governance.go:4-25`). These are g0router-NATIVE rows (the matrix
notes "9router has no virtual key header routing" — evidence is the in-repo Phase-8
plan, `.planning/phases/08-keys-virtualkeys-routing/PLAN.md:46-49` verification items
4-5: "`x-g0-vk` header routes through the correct virtual key"; "Quota exhaustion
skips exhausted keys"). Deferral provenance: `WAVE-4-MAP.md` §Stage-1 scope
("030/031 virtual-key routing → Wave 5 (governance/usage adjacency)").
NOT in scope (explicitly, from Phase-8's broader list): weighted provider selection
(dropped as non-parity by w4-d plan-gate disposition), fallback chain generation
(w4-d/e shipped the parity fallback engines), routing-rules CRUD, dashboard pages
(W6).
Frozen ref @ 827e5c3 (not used — native rows). Depends: w5-b (request_log spend
source), w5-d (routes_admin.go serialization), w5-f (internal/api serialization).
Runs LAST in Wave 5, ALONE.

## Tasks

1. **virtual_keys store** — evidence: schema `internal/schemas/governance.go:4-25`
   (`VirtualKey{ID, Name, ProviderConfigs[], Budget{Limit, Period, Used},
   RateLimitRPM}`); store pattern `internal/store/apikeys.go` (key-table neighbor);
   migrations additive per `internal/store/migrate.go`.
   STEP (a): `TestVirtualKeyCRUD` (create/get-by-id/get-by-key/list/update/delete
   round-trip; key value stored UNIQUE; unknown id → ErrNoRows-mapped per
   `settings.go:33-40` convention) — fails.
   STEP (b): migrate.go additive table `virtual_keys` (id TEXT PRIMARY KEY, key TEXT
   NOT NULL UNIQUE, name TEXT NOT NULL, config_json TEXT NOT NULL DEFAULT '{}'
   (provider_configs + budget + rate_limit_rpm as one snake_case JSON blob — additive
   evolution without column churn), is_active INTEGER NOT NULL DEFAULT 1, created_at
   INTEGER NOT NULL, updated_at INTEGER NOT NULL) + `idx_virtual_keys_key`; NEW
   `internal/store/virtualkeys.go` CRUD.

2. **Quota engine (PAR-ROUTE-031)** — evidence: Phase-8 PLAN.md:50 ("Quota
   exhaustion skips exhausted keys") + risks table ("Quota race conditions → atomic
   counters or SQLite transactions"); spend source = `request_log` cost attribution
   (AGENTS.md:27; w5-b SaveUsage rows carry `api_key`).
   STEP (a): `TestVKBudgetExhaustion` (vk budget limit 1.00, seeded request_log rows
   attributed to the vk key summing 1.10 in the period → Allow()=false with
   budget-exhausted reason; under limit → true), `TestVKRateLimitRPM` (rpm=2,
   injected clock; 3rd request inside the same minute denied, next minute allowed),
   `TestVKQuotaConcurrent` (-race; parallel Allow calls never over-admit RPM) — fail.
   STEP (b): NEW `internal/governance/quota.go`: `QuotaEngine{spend SpendReader,
   clock}` — `SpendReader` interface implemented by store:
   `SumCostByAPIKey(key, sinceISO string) (float64, error)` (added to
   `internal/store/requestlog.go` — serial, w5-b long merged); budget periods:
   daily/weekly/monthly window start derived from Period + clock; RPM = mutex-guarded
   per-key minute window counter (in-memory, matching Budget.Used semantics
   refreshed from request_log). Replaces the placeholder `internal/governance`
   doc.go-only package (its doc comment names exactly this responsibility).

3. **x-g0-vk header routing (PAR-ROUTE-030)** — evidence: Phase-8 PLAN.md:49
   (verification 4) + `internal/schemas/governance.go:13-18` (`ProviderConfig
   {Provider, AllowedModels, KeyIDs, Weight}`).
   STEP (a): `TestChatVKHeaderRouting` (request with `x-g0-vk: <key>` and a model
   allowed by the vk's ProviderConfigs → dispatched; model NOT in AllowedModels →
   403 envelope), `TestChatVKQuotaDenied` (exhausted vk → 429 envelope, provider
   never called), `TestChatNoVKHeaderUnchanged` (no header → existing path
   untouched) — fail.
   STEP (b): in `internal/api/chat.go` (serialized AFTER w5-f): `VKResolver`
   interface (api imports neither store nor governance — w4-e seam precedent):
   `ResolveVK(key string) (vk-info, ok)` + `AllowVK(key, model string) (ok, status,
   reason)`; check after model resolution, before dispatch. Production adapter in
   `internal/server` wiring (same pattern as w5-pre's comboDispatcher adapter).

4. **Admin CRUD routes** — evidence: Phase-8 PLAN.md:27 (`internal/admin/keys.go` —
   note: `/api/keys` is TAKEN by machine API-keys (w3-d, `routes_admin.go:52-56`);
   virtual keys get `/api/virtual-keys` matching the Phase-8 dashboard page name
   `_app.virtual-keys.tsx`, PLAN.md:31).
   STEP (a): `TestVirtualKeysAdminCRUD` (POST validates name + provider_configs
   non-empty + budget fields non-negative → 400 on violation; GET list; PUT update;
   DELETE; envelope + snake_case) — fails.
   STEP (b): NEW `internal/admin/virtualkeys.go` handlers; register
   GET/POST `/api/virtual-keys`, GET/PUT/DELETE `/api/virtual-keys/{id}` under
   `RequireSession` in `internal/server/routes_admin.go`.

## Preconditions (each states its own pass condition)
- `grep -c 'api_key' internal/store/requestlog.go` ≥ 1 (w5-b merged — spend attribution exists).
- `grep -c '/api/usage/stream' internal/server/routes_admin.go` ≥ 1 (w5-e merged — serialization done).
- `grep -rc 'Recorder' internal/api/chat.go` ≥ 1 (w5-f merged — api serialization done).
- `grep -rc 'x-g0-vk' internal/ --include='*.go'` outputs `0` (the gap; flips ≥1).
- `ls internal/store/virtualkeys.go 2>/dev/null | wc -l` outputs `0` (gap).

## Exclusive file ownership
NEW: `internal/store/virtualkeys.go`(+test), `internal/governance/quota.go`(+test),
`internal/admin/virtualkeys.go`(+test). TOUCH: `internal/store/migrate.go`(+test),
`internal/store/requestlog.go`(+test — SumCostByAPIKey), `internal/api/chat.go`(+test),
`internal/server/{server,routes_openai,routes_admin}.go` (wiring). Runs ALONE, last —
no concurrency constraints.

## Binary acceptance
- `go build ./... && go vet ./...` clean; `go test ./...` green; `go test -race ./internal/governance/ ./internal/api/ ./internal/store/` green.
- `grep -c 'x-g0-vk' internal/api/chat.go` ≥ 1; `grep -c '/api/virtual-keys' internal/server/routes_admin.go` ≥ 2.
- `grep -rc 'bloodf/g0router/internal/store\|bloodf/g0router/internal/governance' internal/api/chat.go` → `:0`.
- TestVirtualKeyCRUD, TestVKBudgetExhaustion, TestVKRateLimitRPM, TestVKQuotaConcurrent,
  TestChatVKHeaderRouting, TestChatVKQuotaDenied, TestChatNoVKHeaderUnchanged,
  TestVirtualKeysAdminCRUD all pass.

## Out of scope
Weighted selection / routing-rules / dashboard pages (Phase-8 leftovers → W6+ or
non-parity). Per-vk usage analytics views (w5-d stats already break down byApiKey).
Messages/responses/embeddings VK enforcement (chat is the Phase-8 verification
surface; extending to sibling endpoints is a follow-up noted for W6).
