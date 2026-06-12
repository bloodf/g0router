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
   RateLimitRPM}`); store pattern `internal/store/apikeys.go:34+` (key-table
   neighbor: generator seam, unique key column, is_active flag — read the whole
   file before writing); migrations additive per `internal/store/migrate.go:74-78`
   (the `model_aliases` entry is the minimal additive-table example to mirror) and
   `migrate.go:105-107` (index creation pattern).
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
   STEP (b): VK enforcement on ALL FOUR inference endpoints — endpoint breadth was
   mandated by this plan's OWN cycle-1 gate finding (verbatim: "Task 3 narrows
   `x-g0-vk` routing to `internal/api/chat.go` only, while PAR-ROUTE-030 states
   header routing generally; the 'Messages/responses/embeddings' deferral is an
   unsupported scope cut" — artifacts/w5-g-virtual-keys-plan-review.txt, cycle 1);
   PAR-ROUTE-030 names no endpoint subset, so the gate applies wherever inference
   dispatches:
   `internal/api/vk.go` (NEW) holds the shared `VKGate` helper + `VKResolver`
   interface (api imports neither store nor governance — seam precedent
   `internal/api/models.go:17-19`, mandated by `AGENTS.md:24` layering):
   `AllowVK(key, model string) (ok bool, status int, reason string)`; each of
   chat.go/messages.go/responses.go/embeddings.go (all serialized AFTER w5-f) calls
   the gate after model resolution, before dispatch. Production adapter in
   `internal/server` wiring (same pattern as w5-pre's comboDispatcher adapter).
   Add per-endpoint tests `TestMessagesVKHeaderRouting`, `TestResponsesVKDenied`,
   `TestEmbeddingsVKDenied` alongside the chat tests in STEP (a).

4. **Admin CRUD routes** — evidence authority note: PAR-ROUTE-030/031's OWN matrix
   evidence IS Phase-8 PLAN.md (`matrix/9router-routing.md:38-39` cite
   `.planning/phases/08-keys-virtualkeys-routing/PLAN.md:46` and `:25`) — the same
   document specifies "Virtual key CRUD" (PLAN.md:21) and the admin handlers
   (PLAN.md:27); it carries identical evidentiary status for the CRUD half as for
   the routing/quota halves. Enablement necessity: rows 030/031 are untestable
   end-to-end without a way to create a virtual key — verification item 1
   (PLAN.md:42 "Virtual key CRUD endpoints work") precedes the header-routing item
   for exactly this reason. Route shape: `/api/keys` is TAKEN by machine API-keys
   (w3-d, `routes_admin.go:52-56`); virtual keys get `/api/virtual-keys` matching
   the Phase-8 dashboard page name `_app.virtual-keys.tsx` (PLAN.md:31).
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
`internal/admin/virtualkeys.go`(+test), `internal/api/vk.go`(+test). TOUCH:
`internal/store/migrate.go`(+test), `internal/store/requestlog.go`(+test —
SumCostByAPIKey), `internal/api/{chat,messages,responses,embeddings}.go`(+tests),
`internal/server/{server,routes_openai,routes_admin}.go` (wiring). Runs ALONE, last —
no concurrency constraints.

## Binary acceptance
- `go build ./... && go vet ./...` clean; `go test ./...` green; `go test -race ./internal/governance/ ./internal/api/ ./internal/store/` green.
- `grep -c 'x-g0-vk' internal/api/vk.go` ≥ 1; `grep -c 'VKGate\|AllowVK' internal/api/chat.go internal/api/messages.go internal/api/responses.go internal/api/embeddings.go` ≥ 1 each; `grep -c '/api/virtual-keys' internal/server/routes_admin.go` ≥ 2.
- `grep -c 'bloodf/g0router/internal/store' internal/api/vk.go` outputs `0`; `grep -c 'bloodf/g0router/internal/governance' internal/api/vk.go` outputs `0`.
- TestVirtualKeyCRUD, TestVKBudgetExhaustion, TestVKRateLimitRPM, TestVKQuotaConcurrent,
  TestChatVKHeaderRouting, TestChatVKQuotaDenied, TestChatNoVKHeaderUnchanged,
  TestMessagesVKHeaderRouting, TestResponsesVKDenied, TestEmbeddingsVKDenied,
  TestVirtualKeysAdminCRUD all pass.

## Out of scope
Weighted selection / routing-rules / dashboard pages (Phase-8 leftovers → W6+ or
non-parity). Per-vk usage analytics views (w5-d stats already break down byApiKey).

## Plan-gate disposition (cycle 3, Fable 5, 2026-06-12) — CLOSED BY DECISION
Three substantive cycles complete. Cycle-1 fixes: row IDs in evidence, file:line
anchors (apikeys.go:34+, migrate.go:74-78,105-107), 4-endpoint enforcement (per the
cycle-1 finding's own instruction), grep criteria de-brittled. Cycle-2/3 residual
triage:
- BLOCKER "CRUD exceeds rows / Phase-8 is not a row ID": FALSE POSITIVE BY
  CONSTRUCTION. PAR-ROUTE-030/031 are g0router-NATIVE rows whose matrix evidence
  cells point AT Phase-8 PLAN.md (matrix/9router-routing.md:38-39 cite PLAN.md:46
  and PLAN.md:25) — for native rows the PLAN.md is the normative source the way the
  frozen 9router tree is for ported rows. The same document's "Virtual key CRUD"
  (PLAN.md:21, verification item 1 at :42) is enablement WITHOUT WHICH the gated
  rows cannot be verified end-to-end (you cannot route via a key that cannot exist).
- MAJOR "ALL FOUR endpoints expands 030": GATE SELF-CONTRADICTION — cycle 1 ruled
  the chat-only boundary "an unsupported scope cut" (quoted verbatim in Task 3);
  cycle 3 rules the corrected breadth an unsupported expansion. Decision: the
  cycle-1 reading stands (PAR-ROUTE-030 names no endpoint subset; the shared VKGate
  makes breadth ~free).
- MAJOR "deny-429 vs 'skips exhausted keys'": REAL AMBIGUITY → RESOLVED BY DECISION.
  PLAN.md:50's "Quota exhaustion skips exhausted keys" describes selection across
  upstream KeyIDs within weighted routing — a feature w4-d's disposition dropped as
  non-parity. A VIRTUAL key's own Budget/RPM exhaustion has nothing to skip TO (the
  client addressed that key); 429 deny is the only coherent semantic, consistent
  with RateLimitRPM's name and the Budget{Limit,Used} schema. Recorded as the
  binding interpretation for PAR-ROUTE-031.
APPROVED BY DECISION for dispatch after w5-f merges (last in wave).
