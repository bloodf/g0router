# w6-pre — Go routing carry-forward: custom models, sub-config models, VK KeyIDs pinning

Wave 6 micro-plan (rev 2 — post gate: admin CRUD removed, REF-CHECK made mandatory,
zero-skip gate added). Go-only; runs immediately, in parallel with w6-a (UI track — zero
file overlap). Branch: `main` (direct push). Commit format: `phase-1/w6-pre: <description>`.
TDD throughout: every task is STEP(a) failing test → STEP(b) minimum code to green.
`go build ./... && go vet ./... && go test ./...` green at every commit.

---

## 1. Scope

### PAR rows in scope
| Row | Title | Target state |
|---|---|---|
| PAR-ROUTE-057 | Custom model merging in `/v1/models` (settings `customModels` + alias IDs) | HAVE |
| PAR-ROUTE-058 | Sub-config model exposure (`ttsConfig.models` / `embeddingConfig.models`) | HAVE |
| PAR-ROUTE-030 | VK `ProviderConfig.KeyIDs` threading to dispatch (KeyIDs half; gate half shipped w5-g) | HAVE |

Frozen ref: `/home/cortexos/Developer/github.com/bloodf/_refs/9router` @ 827e5c3.
057 ref: `src/app/api/v1/models/route.js:316-348`. 058 ref: `route.js:364-383`.

PAR-ROUTE-057 requires ONLY the `GET /v1/models` list endpoint to merge custom/alias/static
models. Custom models are **read** from the `customModels` settings key (JSON array
`[{id, provider, type?}]`) inside the `/v1/models` handler. There are NO new CRUD endpoints
in this plan.

### NOT in scope (explicit)
- **Admin CRUD for custom models** (`GET/POST/DELETE /api/v1/models/custom`) — removed by
  gate decision: not required by PAR-ROUTE-057. No `internal/server/routes_admin.go`
  changes, no `internal/admin/` changes, no new endpoints of any kind. Operators/tests
  seed the `customModels` setting directly (`SetSetting` in tests; `sqlite3` out-of-band).
- PAR-ROUTE-051 (`preferredConnID` in `SelectConnection`) — already shipped; consumed here, not modified.
- PAR-ROUTE-031 (VK quota engine) — shipped w5-g.
- Combo-path VK pinning: the VK gate runs only on the non-combo dispatch path
  (`chat.go:347-351` branches to `handleCombo` before the gate). Pinning applies to the
  non-combo path only; combo×VK interplay is a future row if the matrix demands it.
- Any UI surface for custom models (Wave 6 UI plans own `ui/`).
- Matrix flips and `docs/WORKFLOW.md` wave rollup — wave-close decision commit owns those.
- Alias *resolution* changes (`inference.ResolveModelAlias`) — only alias *listing* is consumed.
- `/v1/models/{kind}` (PAR-ROUTE-037) — custom/sub-config models are NOT added to the
  by-kind endpoint in this plan (ref merges them in the main list handler only).

---

## 2. Precondition checks

Run all before starting. Each line states the expected result and the STOP rule.
**P5 (REF-CHECK) is a hard gate: no task may start until it has been completed and its
findings recorded. There is no defaults fallback.**

```bash
# P1 — preferred-conn pinning exists (PAR-ROUTE-051):
grep -n "preferredConnID" internal/inference/selection.go
# EXPECT: hits incl. func SelectConnection(providerID, model string, exclude []string, preferredConnID string)
# STOP if zero hits: w5 dependency missing.

# P2 — w5-g KeyIDs field exists on the persisted VK provider config:
grep -rn "KeyIDs" internal/schemas/ internal/store/ internal/api/ internal/server/
# EXPECT: ≥1 hit on the schemas/store side (schemas.ProviderConfig or store VK config JSON).
# NOTE: internal/api/vk.go's VKProviderConfig does NOT yet carry KeyIDs (verified at plan
# time) — adding it there is task T3. storeVKToAPI (routes_openai.go:111-116) currently
# drops the field — task T5 maps it.
# STOP if zero hits anywhere: the persisted field lives in a FORBIDDEN package
# (internal/schemas / internal/store); escalate to orchestrator, do not add it yourself.

# P3 — alias + settings store surface (consumed via adapters, never modified):
grep -n "func (s \*Store) ListAliases" internal/store/aliases.go      # EXPECT: 1 hit
grep -n "func (s \*Store) GetSetting\|func (s \*Store) SetSetting" internal/store/settings.go  # EXPECT: 2 hits
# NOTE: GetSetting returns store.ErrNotFound for a missing key — adapters MUST map
# ErrNotFound → empty result, not an error.

# P4 — AllowVK call-site census (signature change blast radius):
grep -rn "AllowVK(" internal/ --include=*.go
# EXPECT: definition in internal/api/vk.go + call sites ONLY in
# internal/api/{chat,messages,responses,embeddings}.go and their _test.go files.
# STOP if any other non-test caller appears: ownership violation.

# P5 — REF-CHECK (MANDATORY, blocking; no defaults):
sed -n '300,395p' /home/cortexos/Developer/github.com/bloodf/_refs/9router/src/app/api/v1/models/route.js
# MUST extract, with ref:line citations recorded in the T1/T2 commit bodies:
#   (1) ORDER — position of custom and alias entries relative to combos and catalog
#       models in the merged list (route.js:316-348), and where sub-config (tts/embedding)
#       entries are appended (route.js:364-383). The matrix row expectation is
#       combos → custom → alias → catalog, with sub-config per :364-383 — CONFIRM against
#       the ref; the ref is authoritative.
#   (2) DEDUP — direction on ID collision (route.js:316-348). Row expectation: an incoming
#       custom/alias ID that collides with a catalog ID (or an earlier merged ID) is
#       skipped, so each ID appears exactly once — CONFIRM the exact direction and the
#       seen-set seeding it implies; the ref is authoritative.
#   (3) ownedBy — exact strings for custom entries (provider field? literal?), alias
#       entries (row expectation: "alias"), and sub-config entries (row expectation: the
#       connection ID, per route.js:364-383) — CONFIRM against the ref.
#   (4) MALFORMED INPUT — what the ref does with unparseable customModels JSON /
#       providerSpecificData (route.js:316-348, 364-383), and the exact field names
#       (providerSpecificData.ttsConfig.models / .embeddingConfig.models).
# STOP RULE: if the executor sandbox cannot read the ref path, HALT and escalate to the
# orchestrator/operator to run this sed and paste the verbatim excerpt into the task
# context. Do NOT proceed on assumed semantics. T1/T2 test expectations are written from
# these findings, and every ordering/dedup/ownedBy assertion in §5 must carry a ref:line
# citation in the commit body.

# P6 — sub-config storage home:
grep -n "Metadata" internal/store/connections.go                       # EXPECT: hits (JSON blob column)
grep -rn "ttsConfig\|embeddingConfig\|providerSpecificData" internal/ ui/src/ 2>/dev/null
# Determines the JSON shape inside Connection.Metadata. Ref field names from P5(4)
# (route.js:364-383) are authoritative for what the adapter reads; accept both a
# top-level {"ttsConfig":...,"embeddingConfig":...} and a nested
# {"providerSpecificData":{...}} wrapper on read ONLY if prior art in this repo already
# stores it un-nested.

# P7 — gate present on all four handlers:
grep -n "SetVKGate" internal/api/chat.go internal/api/messages.go internal/api/responses.go internal/api/embeddings.go
# EXPECT: 4 hits.
```

---

## 3. Exclusive file ownership (exact files, no globs)

| File | Why |
|---|---|
| `internal/api/models.go` | 057/058 interfaces + merge in `List()` |
| `internal/api/models_test.go` | extend existing tests |
| `internal/api/vk.go` | `VKProviderConfig.KeyIDs`, `AllowVK` 4-value return, `VKPinnedKeyResolver` iface |
| `internal/api/vk_test.go` | extend; update `AllowVK` arity in all existing tests |
| `internal/api/chat.go` | gate keyIDs → pinned-key override before dispatch |
| `internal/api/chat_test.go` | pin/fallback tests |
| `internal/api/messages.go` | same threading |
| `internal/api/messages_test.go` | same |
| `internal/api/responses.go` | same threading |
| `internal/api/responses_test.go` | same |
| `internal/api/embeddings.go` | same threading |
| `internal/api/embeddings_test.go` | same |
| `internal/server/routes_openai.go` | adapters (custom/alias/sub-config listers, vkPinnedSelector), `storeVKToAPI` KeyIDs mapping, handler wiring |
| `internal/server/routes_openai_test.go` | adapter unit tests (create if absent) |

### FORBIDDEN (do not touch)
- `internal/admin/` (any file) and `internal/server/routes_admin.go` — admin CRUD is OUT
  of this plan by gate decision.
- `internal/inference/selection.go` — `SelectConnection` already takes `preferredConnID`; consume, don't change.
- `internal/store/settings.go` — `GetSetting`/`SetSetting` suffice.
- `internal/server/server.go` — no wiring changes outside `routes_openai.go`.
- Any `ui/` file; any other `internal/` package (schemas, store, inference, governance, auth, usage, translation).

---

## 4. Design decisions (binding)

1. **Layering**: the api package stays store-free. All three model listers and the pinned
   selector are api-package interfaces; concrete adapters live in `routes_openai.go`
   (pattern: `vkResolverAdapter`, routes_openai.go:77-118).
2. **AllowVK signature** becomes
   `AllowVK(key, model, providerID string) (ok bool, status int, reason string, keyIDs []string)`.
   `keyIDs` is the matched `VKProviderConfig.KeyIDs` (nil when no header / no match / config
   has none). Internal refactor: `providerConfigAllowed` returns `(*VKProviderConfig, bool)`
   (first matching config wins). Quota denial path returns `keyIDs=nil`.
3. **Pinned dispatch seam** (api, in vk.go):
   `type VKPinnedKeyResolver interface { ResolvePinned(providerID, model string, keyIDs []string) (connID, credential string, ok bool) }`.
   Each of the four handlers gets a `pinnedResolver VKPinnedKeyResolver` field +
   `SetVKPinnedResolver`. After a gate pass with `len(keyIDs) > 0` and a non-nil resolver:
   on `ok`, override `key.ID, key.Value = connID, credential`; on `!ok`, keep the
   originally resolved key (normal selection fallback). Nil resolver / nil gate ⇒ behavior
   unchanged.
4. **vkPinnedSelector adapter** (routes_openai.go): holds `*store.Store`, its own
   `*inference.SelectionEngine` built as `inference.NewSelectionEngine(st, st, nil, time.Now)`
   (nil Cooldown is safe — `SelectConnection` never touches `e.cd`; the global `selectionMu`
   still serializes against the production engine), a `sync.Mutex`, and an `rr map[string]int`
   round-robin cursor keyed by `providerID+":"+model`. Algorithm: starting at
   `rr[key] % len(keyIDs)` and rotating once through the list, call
   `SelectConnection(providerID, model, nil, keyID)`; accept only if the returned
   `conn.ID == keyID` (the preferred-pin path, selection.go:166-174 — a strategy-fallback
   result is rejected so pinning can't silently land elsewhere); on accept, advance the
   cursor and return `(conn.ID, credential, true)`. Credential mapping mirrors
   `comboDispatcher` (server.go:183-186): `AccessToken`, else `Secret`. All keyIDs
   ineligible ⇒ `(_, _, false)`.
5. **storeVKToAPI** (routes_openai.go) maps the persisted KeyIDs field (located by P2) into
   `api.VKProviderConfig.KeyIDs`.
6. **Models merge** (models.go `List()`): build with a `seen map[string]bool`. Order,
   dedup direction, and `ownedBy` strings come from the P5 REF-CHECK findings
   (route.js:316-348 for custom/alias, :364-383 for sub-config) — they are NOT defaults
   and may not be substituted; each is asserted in a T1/T2 test whose expected value
   carries a ref:line citation in the commit body. The catalog section keeps its existing
   `providerStart`-anchored sort (models.go:89-92) — recompute `providerStart` after the
   inserted sections. All three new listers are nil-safe (skip when unwired); a lister
   error ⇒ 500 `server_error`, matching the combo-lister error path (models.go:54-57).
7. **Adapter resilience**: `customModelsAdapter` reads setting `customModels`
   (`store.ErrNotFound` → empty), unmarshals `[{"id","provider","type"}]`. Malformed-JSON
   behavior is determined exclusively by P5(4) REF-CHECK — implement whatever the ref does
   (hard-fail or silent-empty); do NOT pre-decide. T5 test `TestCustomModelsAdapter_MalformedJSON`
   asserts whatever behavior P5(4) documents, with its expected value and a ref:line citation
   recorded in the T5 commit body. Entries with empty `id` are skipped per ref. 
   `subConfigModelsAdapter` iterates `st.ListConnections()`, parses `Metadata` per P6/P5(4);
   unparseable metadata on one connection follows P5(4) behavior (again, P5 is authoritative).
   **Parity deviation (document in commit body)**: `store.Connection` has no active/inactive
   flag, so every stored connection is treated as active — this is an explicit deviation
   recorded at T5 time; log it but do not change scope. `subConfigModelsAdapter` reads
   ONLY `providerSpecificData.ttsConfig.models` and `providerSpecificData.embeddingConfig.models`
   — the exact nested field path from ref route.js:364-383 per P5(3/4). No top-level
   fallback shape. If the key is absent the connection contributes zero sub-config entries.
8. **Nil Cooldown + selectionMu safety** (file:line evidence): `SelectConnection`
   (selection.go:132-267) never references `e.cd` — the three `e.cd.*` calls are in
   `WithAccountFallback` only (selection.go:233, 247, 262), which the pinned-selector adapter
   does NOT call. `selectionMu` is a package-level variable (selection.go:16), shared by all
   `inference.SelectionEngine` instances; the pinned selector's separate engine instance will
   acquire the same mutex, preserving the PAR-ROUTE-017 serialization guarantee.

---

## 5. TDD tasks

Every task is STEP(a) failing test FIRST (run it, paste the failure), then STEP(b) minimum
code to green. Build/vet gates and grep proofs are supplements, never substitutes for the
failing test.

### T1 — PAR-ROUTE-057: custom + alias model merging (`internal/api/models.go`)
- **(a)** In `models_test.go` add (run, see FAIL — interfaces/setters don't exist yet);
  expected order/ownedBy/dedup values are filled in from P5 findings with ref:line cited
  in the test comment and commit body:
  - `TestModelsList_MergesCustomModels` — fake `CustomModelLister` returning
    `{ID:"my-custom", Provider:"openai", Type:"llm"}`; assert exactly one entry
    `id=my-custom`, `owned_by` per P5(3) [ref route.js:316-348], positioned per P5(1).
  - `TestModelsList_MergesAliasModels` — fake `AliasModelLister` returning `["fast"]`;
    assert one entry `id=fast`, `owned_by` per P5(3) (row expectation `"alias"`; ref wins).
  - `TestModelsList_DedupCustomVsCatalog` — custom entry whose ID equals a real catalog
    model ID; assert that ID appears exactly once and the surviving entry's `owned_by`
    matches the P5(2) dedup direction [ref route.js:316-348].
  - `TestModelsList_CustomListerError` — erroring fake ⇒ 500 `server_error`.
  - `TestModelsList_NilListersUnchanged` — unwired listers ⇒ combos+catalog only (guards
    existing behavior incl. PAR-ROUTE-047 combo-first ordering).
- **(b)** Add `CustomModel{ID, Provider, Type string}`, `CustomModelLister`
  (`ListCustomModels() ([]CustomModel, error)`), `AliasModelLister`
  (`ListAliasNames() ([]string, error)`), setters `SetCustomModelLister` /
  `SetAliasModelLister`, and the seen-map merge in `List()` per design §6. Green.
- **Commit 1**: `phase-1/w6-pre: custom + alias model merging in /v1/models (PAR-ROUTE-057)`
  — body cites the P5 ref lines backing order, dedup, ownedBy.

### T2 — PAR-ROUTE-058: sub-config model exposure (`internal/api/models.go`)
- **(a)** `TestModelsList_IncludesSubConfigModels` — fake `SubConfigModelReader` returning
  `{ID:"tts-1", Kind:"tts", ProviderID:"prov-1"}` and an embedding entry; assert entries
  present with `owned_by` per P5(3) (row expectation: the connection ID [ref
  route.js:364-383]) and `type` mapped from Kind. `TestModelsList_SubConfigDedup` —
  sub-config ID colliding with catalog ⇒ exactly one entry, direction per P5(2).
  `TestModelsList_SubConfigReaderError` ⇒ 500. FAIL first.
- **(b)** `SubConfigModel{ID, Kind, ProviderID string}`, `SubConfigModelReader`
  (`ListSubConfigModels() ([]SubConfigModel, error)`), `SetSubConfigModelReader`, merge
  positioned per P5(1).
- **Commit 2**: `phase-1/w6-pre: sub-config (tts/embedding) model exposure in /v1/models (PAR-ROUTE-058)`
  — body cites route.js:364-383 for field names, ownedBy, placement.

### T3 — PAR-ROUTE-030: gate returns KeyIDs (`internal/api/vk.go`)
- **(a)** In `vk_test.go`: `TestAllowVK_ReturnsKeyIDsOnMatch` (matched config with
  `KeyIDs: []string{"c1","c2"}` ⇒ ok + those keyIDs), `TestAllowVK_NilKeyIDsWhenUnset`,
  `TestAllowVK_NilKeyIDsOnDenial` (quota/inactive/unknown paths return nil keyIDs).
  Mechanically update every existing `AllowVK` call for the new arity. FAIL on compile.
- **(b)** Add `KeyIDs []string` to `api.VKProviderConfig`; refactor
  `providerConfigAllowed` → `matchProviderConfig(configs, model, providerID) (*VKProviderConfig, bool)`;
  `AllowVK` returns the 4th value. Update the four handler call sites for arity only
  (discard keyIDs with `_` for now) so the tree compiles green.
- **Commit 3**: `phase-1/w6-pre: VKGate returns matched-config KeyIDs (PAR-ROUTE-030 gate seam)`

### T4 — PAR-ROUTE-030: handler pin threading (`chat.go`, `messages.go`, `responses.go`, `embeddings.go`)
- **(a)** Per handler (mirror existing handler-test fixtures): e.g.
  `TestChatHandle_VKPinnedKeyOverridesDispatch` — VKGate built over a fake resolver whose
  `VKInfo` config carries `KeyIDs:["conn-2"]`; fake `VKPinnedKeyResolver` returns
  `("conn-2","cred-2",true)`; fake provider captures the dispatched `schemas.Key`; assert
  `key.ID=="conn-2"` and `key.Value=="cred-2"`. Plus
  `TestChatHandle_VKPinFallbackKeepsResolvedKey` (resolver returns ok=false ⇒ original key
  dispatched) once in chat_test.go; the other three handlers each get the override test
  (`TestMessagesHandle_VKPinnedKeyOverridesDispatch`,
  `TestResponsesHandle_VKPinnedKeyOverridesDispatch`,
  `TestEmbeddingsHandle_VKPinnedKeyOverridesDispatch`). FAIL first.
- **(b)** Add `pinnedResolver VKPinnedKeyResolver` (interface defined in vk.go, design §3)
  + `SetVKPinnedResolver` to all four handlers; after the gate pass, apply the override.
  Usage attribution (`g.apiKey = vkHeader`, recordGlue calls) must keep using the
  overridden `key.ID` naturally — no extra change, just assert it in the chat test.
- **Commit 4**: `phase-1/w6-pre: thread VK KeyIDs to pinned dispatch in chat/messages/responses/embeddings (PAR-ROUTE-030)`

### T5 — production wiring (`internal/server/routes_openai.go`)
- **(a)** In `routes_openai_test.go` (create if absent; use the store-on-`t.TempDir()`
  pattern from existing internal/store or internal/server tests). Custom-model fixtures
  are seeded with `st.SetSetting("customModels", ...)` directly — no endpoint exists or
  is created:
  - `TestCustomModelsAdapter_ParsesSetting`, `TestCustomModelsAdapter_MissingSettingEmpty`,
    `TestCustomModelsAdapter_MalformedJSON` (expected behavior filled from P5(4) finding with ref:line)
  - `TestAliasModelsAdapter_ListsNames`
  - `TestSubConfigModelsAdapter_ParsesConnectionMetadata` (tts + embedding lists),
    `TestSubConfigModelsAdapter_UnparseableMetadata` (expected behavior filled from P5(4) finding with ref:line)
  - `TestVKPinnedSelector_PinsEligibleKeyID`, `TestVKPinnedSelector_RoundRobinAcrossKeyIDs`,
    `TestVKPinnedSelector_FallbackWhenAllIneligible` (locked via model_locks ⇒ ok=false)
  - `TestStoreVKToAPI_MapsKeyIDs`
  FAIL first.
- **(b)** Implement `customModelsAdapter`, `aliasModelsAdapter`, `subConfigModelsAdapter`,
  `vkPinnedSelector` (design §4, §7); map KeyIDs in `storeVKToAPI`; inside the existing
  `if st != nil` block wire `models.SetCustomModelLister/SetAliasModelLister/SetSubConfigModelReader`
  and `SetVKPinnedResolver` on all four handlers. `RegisterOpenAIRoutes` signature is
  unchanged.
- **Commit 5**: `phase-1/w6-pre: wire custom/alias/sub-config listers and VK pinned selector (PAR-ROUTE-057/058/030)`

---

## 6. Binary acceptance criteria

All of the following, verbatim, must pass:

```bash
export PATH=$PATH:/usr/local/go/bin
go build ./... && go vet ./... && go test ./...        # all green

# Zero skipped tests (wc -l always exits 0 regardless of count):
go test ./... -v 2>&1 | grep "^--- SKIP" | wc -l   # must print 0

# Interfaces exist where specified:
grep -n "ListCustomModels() (\[\]CustomModel, error)" internal/api/models.go   # ≥1
grep -n "ListAliasNames() (\[\]string, error)" internal/api/models.go          # ≥1
grep -n "ListSubConfigModels() (\[\]SubConfigModel, error)" internal/api/models.go  # ≥1
grep -n "KeyIDs \[\]string" internal/api/vk.go                                 # ≥1
grep -n "keyIDs \[\]string)" internal/api/vk.go                                # AllowVK 4-value return
grep -n "VKPinnedKeyResolver" internal/api/vk.go                               # ≥1
grep -rn "SetVKPinnedResolver" internal/api/chat.go internal/api/messages.go internal/api/responses.go internal/api/embeddings.go | wc -l  # must equal 4 (one per handler)
grep -n "KeyIDs" internal/server/routes_openai.go                              # storeVKToAPI maps it

# No admin surface was created:
grep -rn "models/custom" internal/ --include=*.go                              # must be 0 hits
git log --name-only --format= e93f0e0..HEAD | grep -c "routes_admin\|internal/admin/"  # must equal 0

# Ownership respected — check via §8 diff-gate scoped file list at closeout.
# At plan closeout the orchestrator runs the diff gate (§8) against the
# implementer's commit range; any file outside §8's file list is a REJECT.
# This replaces a base-SHA git log gate (invalid under parallel mainline commits).

# Ref evidence recorded (substitute N = number of this plan's commits):
git log -N --format=%B | grep "route.js:3" | wc -l   # must be ≥2 (T1 + T2 cite ref lines)
```

Named tests that must exist and pass (`go test ./internal/api/ ./internal/server/ -v -run <name>`):
`TestModelsList_MergesCustomModels`, `TestModelsList_MergesAliasModels`,
`TestModelsList_DedupCustomVsCatalog`, `TestModelsList_NilListersUnchanged`,
`TestModelsList_IncludesSubConfigModels`,
`TestAllowVK_ReturnsKeyIDsOnMatch`, `TestAllowVK_NilKeyIDsOnDenial`,
`TestChatHandle_VKPinnedKeyOverridesDispatch`, `TestChatHandle_VKPinFallbackKeepsResolvedKey`,
`TestMessagesHandle_VKPinnedKeyOverridesDispatch`,
`TestResponsesHandle_VKPinnedKeyOverridesDispatch`,
`TestEmbeddingsHandle_VKPinnedKeyOverridesDispatch`,
`TestVKPinnedSelector_RoundRobinAcrossKeyIDs`, `TestVKPinnedSelector_FallbackWhenAllIneligible`,
`TestCustomModelsAdapter_ParsesSetting`, `TestSubConfigModelsAdapter_ParsesConnectionMetadata`.

Behavioral invariants (asserted by the tests above; restated for the reviewer):
1. `/v1/models` with no listers wired is byte-compatible with pre-w6-pre output.
2. Every model ID appears at most once in `/v1/models`; ordering, dedup direction, and
   `owned_by` values match the P5 REF-CHECK findings, with ref:line citations in the
   T1/T2 commit bodies.
3. A VK whose matched config has no KeyIDs dispatches exactly as before (nil keyIDs ⇒ no resolver call).
4. Pinning never errors a request: ineligible KeyIDs degrade to normal selection.
5. No new HTTP endpoints exist; `internal/server/server.go`, `routes_admin.go`, and
   `internal/admin/` are untouched.

## 7. Out of scope (explicit)

Everything in §1 "NOT in scope", plus: persistence-side KeyIDs schema work (w5-g's),
new store methods of any kind, custom-model write/CRUD surface (settings key is read-only
from this plan's perspective), UI for custom models / VK pinning, alias CRUD endpoints
(already exist), `/v1/models/{kind}` merging, per-VK pinning metrics, matrix/WORKFLOW
rollup edits (wave-close decision commit owns those).

## 8. Diff-gate scope (gpt-5.5 scoped diff review — exact file list)

```
internal/api/models.go
internal/api/models_test.go
internal/api/vk.go
internal/api/vk_test.go
internal/api/chat.go
internal/api/chat_test.go
internal/api/messages.go
internal/api/messages_test.go
internal/api/responses.go
internal/api/responses_test.go
internal/api/embeddings.go
internal/api/embeddings_test.go
internal/server/routes_openai.go
internal/server/routes_openai_test.go
```

Review charge: ownership-subset check (§6 git log gate), AllowVK arity completeness,
seen-map dedup correctness vs the cited P5 ref findings (reject any ordering/dedup/ownedBy
assertion lacking a ref:line citation), nil-safety of every new seam, zero new endpoints,
and the zero-skip test gate.

## Plan gate disposition (closed by decision after 3 cycles — 2026-06-12)

**Cycle 1 REJECT** — REAL: admin CRUD endpoints not in PAR-ROUTE-057 (removed); ref-reading
was optional with fallback-to-defaults (removed; P5 is now mandatory hard gate); zero-skip
check missing (added). All fixed.

**Cycle 2 REJECT** — REAL: malformed-JSON test pre-named with behavior assumption before
ref reading (renamed to neutral `TestCustomModelsAdapter_MalformedJSON`); skip gate used
grep-c which exits 1 on 0 hits (replaced with wc -l); SetVKPinnedResolver grep ambiguous over
multiple files (replaced with grep -rn | wc -l = 4); nil Cooldown claim lacked file:line
(added: SelectConnection:132-267 never calls e.cd; e.cd is only in WithAccountFallback:233/247/262).

**Cycle 3 REJECT** — REAL: T5 still had `_MalformedJSONEmpty` and `_SkipsUnparseableMetadata`
names (fixed — neutral names with behavior from P5(4)); git log base-SHA gate invalid under
parallel commits (removed; §8 diff-gate scope list is the ownership check); top-level metadata
fallback beyond ref scope (removed; only providerSpecificData.ttsConfig/embeddingConfig per
route.js:364-383). Plan is now actionable for kimi dispatch after P5 REF-CHECK completes.
