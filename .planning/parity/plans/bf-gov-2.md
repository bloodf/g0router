# Micro-plan bf-gov-2 — typed WhiteList/BlackList allow/block semantics (governance, Go)

```
program: bifrost-parity (Waves 6+; CLI_ORCHESTRATOR.md governs, HANDOFF.md retired)
plan: bf-gov-2
status: READY (rev 1 — authored against the LIVE merged tree AFTER bf-gov-1 shipped:
  schemas.VirtualKey.TeamID + schemas.ProviderConfig.AllowAllKeys already present
  (governance.go:7,18); api.VKProviderConfig.AllowAllKeys + AllowVK keyIDs branch
  present (vk.go:6-11,83-87); store.VirtualKey.TeamID + virtual_keys.team_id column
  present (virtualkeys.go:18; migrate.go:400). BIFROST-MAP.md ledger row bf-gov-2
  §301; gov disposition §245; architectural decision #7 §140-159; freeze rules
  §384-399. bf-gov-2 builds typed list semantics + the provider-config-level
  blacklist filter ON TOP of that shipped tree.)
runs: governance track. DISJOINT from bf-gov-1/bf-gov-3 — those own
  internal/governance/quota.go (the bf-gov-1 → bf-gov-3 internal serial).
  bf-gov-2 owns NEW internal/schemas/lists.go + the model-allow decision path in
  internal/api/vk.go (matchProviderConfig). bf-gov-2 does NOT touch quota.go.
  Runs ∥ bf-gov-1 (BIFROST-MAP §330).
branch: main (direct commits, per AGENTS.md "No PR workflow")
commit-prefix: phase-1/bf-gov-2: (matches the shipped bifrost chain prefix —
  verified: bf-gov-1 used phase-1/bf-gov-1:)
commit-footer: Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>
ref-source: BLOCKED — ESC-REF-ABSENT (BIFROST-MAP §47-68). The frozen Bifrost ref
  @ ca21298 is ABSENT on this host. The ONLY ground truth is
  .planning/parity/matrix/bifrost-governance.md rows 026-030,037,048 +
  bifrost-openai.md row 119 + g0router's own conventions. Build to documented
  matrix behavior + g0router conventions ONLY; STOP-escalate on any undocumented
  Bifrost wire detail. NEVER build to a guessed Bifrost wire format.
go-serial-slot: NONE. bf-gov-2 does NOT touch routes_admin.go nor the
  routes_openai.go ROUTE-REGISTRATION block (VK CRUD routes ALREADY EXIST — §0.3).
  bf-gov-2 holds NO internal serial on quota.go (that is bf-gov-1 → bf-gov-3).
  lists.go is a NEW disjoint file. The only shared-file touch is an ADDITIVE
  field on schemas.ProviderConfig and an ADDITIVE adapter-body edit in
  routes_openai.go's storeVKToAPI (NOT a route registration).
new-route: NO. Confirmed: bf-gov-2 registers NO new HTTP routes (§0.3).
```

---

## 0. Objective + ground truth

### 0.1 Objective

Give g0router's virtual-key model-gate **typed allow/block-list semantics** that
match the matrix's documented behavior, replacing today's bare
`[]string AllowedModels` equality scan (`vk.go:matchProviderConfig` `:100-116`)
with first-class `WhiteList`/`BlackList` types carrying `["*"]`/empty/listed
semantics + `IsAllowed`/`IsBlocked` methods, and adding a **provider-config-level
blacklist** that **wins over the allowlist** (blacklist-first 2-pass). Additive
only: a NEW `internal/schemas/lists.go` (the two list types + methods), an
additive `BlacklistedModels` field on `schemas.ProviderConfig` (rides the existing
`config_json` blob — NO new column, exactly as `AllowAllKeys` did in bf-gov-1),
and the 2-pass filter wired into the live `AllowVK` model decision. NO new tables,
NO new routes, NO quota.go touch, NO key-level provider-key blacklist store (ESC —
§3), NO model catalog, NO CEL.

This is the bifrost-governance analogue of g0router's existing flat allowlist:
g0router already enforces an allowlist (`AllowedModels []string`, empty = match-all
today) in `matchProviderConfig`; bf-gov-2 upgrades that single decision point to
typed-list semantics + a blacklist pass, with binary acceptance below.

### 0.2 Rows this plan closes (matrix = ground truth)

| Row | Behavior (matrix) | Disposition | Flip target after bf-gov-2 |
|---|---|---|---|
| PAR-BF-GOV-026 | `WhiteList` semantics: `["*"]` allows all; **empty denies all**; listed-without-`*` allows only listed (`account.go:22-30`) | **BUILD** | `schemas.WhiteList` type + `IsAllowed(v) bool` encodes exactly these three cases incl. empty=deny-all (D1, unit-tested). The GATE deliberately keeps legacy match-all on empty `AllowedModels` for backward-compat (VAR, D1). MISSING→**HAVE** (type implements the matrix contract; gate VAR noted). |
| PAR-BF-GOV-027 | `BlackList` semantics: `["*"]` blocks all; **empty blocks none**; listed-without-`*` blocks only listed (`account.go:80-106`) | **BUILD** | `schemas.BlackList` type + `IsBlocked(v) bool` with exactly these three cases (D2). MISSING→**HAVE**. |
| PAR-BF-GOV-028 | Blacklist wins over allowlist: two-pass (blacklist scan first, then allowlist) in `isModelAllowed` (`resolver.go:358-390`) | **BUILD** | `matchProviderConfig` does blacklist-pass-then-allowlist-pass; a blocked model is denied even if allowlisted (D3). MISSING→**HAVE** (VK provider-config slice). |
| PAR-BF-GOV-029 | Provider **key-level** blacklists: `TableKey.BlacklistedModels` persisted as `BlacklistedModelsJSON` (`key.go:24,83`) | **ESC** | g0router has NO provider-key model store; keys are `store.Connection` credentials (no model fields, `connections.go:13-25`) pinned via `KeyIDs []string`. A key-level blacklist field would be a DEAD field (no live single-key model-gate) → no-leftovers violation. ESC (§3). Stays MISSING. |
| PAR-BF-GOV-030 | VK **provider-config-level** blacklists: `TableVirtualKeyProviderConfig.BlacklistedModels` persisted as JSON (`virtualkey.go:32`) | **BUILD** | `schemas.ProviderConfig.BlacklistedModels schemas.BlackList` (additive, rides `config_json`); consumed by the D3 blacklist pass (D4). MISSING→**HAVE**. |
| PAR-BF-GOV-037 | `BeforeSave` on provider-config validates `AllowedModels` + `BlacklistedModels` via `Validate()` (`virtualkey.go:82-90`) | **BUILD** (behavior, not GORM hook) | `WhiteList.Validate()`/`BlackList.Validate()` methods; called inline from the VK admin create/update path (D5). GORM `BeforeSave` *mechanism* ESC; *validation behavior* built inline. MISSING→**HAVE** (validation-behavior slice). |
| PAR-BF-GOV-048 | `TableKey.AfterFind` deserializes `BlacklistedModelsJSON` into runtime `BlacklistedModels` (`key.go:662-666`) | **ESC** | Same root as 029 — there is no provider-key table/runtime field in g0router to deserialize into. ESC (§3). Stays MISSING. The provider-config-level JSON deserialization (the in-scope analogue) is handled by the existing `config_json` `json.Unmarshal` (D4, no new hook). |
| PAR-BF-OAI-119 (EXTRA) | `isModelBlockedByList` logic moved into `BlackList.IsBlocked` (`core/schemas/blacklist.go`) | **BUILD** | The block decision IS a method on the `BlackList` type (`IsBlocked`), not free-floating gate code (D2). EXTRA→**HAVE** (folds into 027). |

**Honest scoping note:** rows **026, 027, 028, 030, 037, 119** are closed
(provider-config-level + typed semantics). Rows **029 and 048** (provider **key**
level) are **ESC** because g0router has no provider-key model store to hang a
blacklist on, and inventing one with no live consumer would be a dead field
(§3 no-leftovers). The matrix's "two-pass" (028) is closed for the
provider-config tier; the model-catalog cross-provider variant of the allowlist
(031, quirk #9) stays ESC (BIFROST-MAP §256). No row is closed by inventing
un-evidenced Bifrost behavior.

### 0.3 Preconditions already satisfied (evidence — read files)

- **bf-gov-1 SHIPPED** (live tree, this is the foundation bf-gov-2 extends):
  `schemas.VirtualKey.TeamID` (`governance.go:7`), `schemas.ProviderConfig.AllowAllKeys`
  (`governance.go:18`), `api.VKProviderConfig.AllowAllKeys` + the `AllowVK` keyIDs
  branch (`vk.go:6-11,83-87`), `store.VirtualKey.TeamID` + `virtual_keys.team_id`
  column (`virtualkeys.go:18,87-88,141-142,179`; `migrate.go:400`), and the
  2-level `QuotaEngine` (`quota.go:81-190`) are all present. bf-gov-2 adds list
  semantics WITHOUT touching any of bf-gov-1's surfaces except the additive
  `ProviderConfig.BlacklistedModels` field and the `matchProviderConfig` decision.
- **The model-allow decision is EXACTLY ONE function** — `matchProviderConfig`
  (`vk.go:100-116`): it loops provider configs, matches `cfg.Provider == providerID`,
  then today treats `len(AllowedModels)==0` as match-all and otherwise does a bare
  equality scan over `AllowedModels`. **This is the single live consumer** the new
  list semantics + blacklist pass wire into. Returns `(*VKProviderConfig, bool)`.
- **`AllowVK` is the sole gate** (`vk.go:65-96`) and is called from **8 live call
  sites** (verified): `chat.go:367`, `completions.go:74`, `responses.go:101`,
  `input_tokens.go:97`, `audio.go:248`, `images.go:62`, `files.go:62`,
  `batches.go:62`, `messages.go:91`, `embeddings.go:73`. Every model-bearing
  request flows through `matchProviderConfig` → so the typed semantics have a
  guaranteed live consumer across the whole OpenAI surface. NO new wiring at call
  sites (signature unchanged).
- **`ProviderConfig` config rides `config_json`** (`virtualkeys.go:24-52`):
  `virtualKeyConfig{ProviderConfigs,Budget,RateLimitRPM}` is JSON-marshaled into
  the `config_json` blob; `AllowedModels`/`KeyIDs`/`AllowAllKeys` already ride it.
  An additive `BlacklistedModels` field rides the SAME blob — NO new column, NO
  migration (D4). `json.Unmarshal` populates it on read; this IS the analogue of
  Bifrost's `AfterFind` JSON deserialization (no new hook needed, 048-for-config).
- **`storeVKToAPI` is the store→api mapper** (`routes_openai.go:180-213`): it
  copies `pc.AllowedModels/KeyIDs/AllowAllKeys` into `api.VKProviderConfig`. The
  ONLY shared-file edit bf-gov-2 makes here is an ADDITIVE
  `BlacklistedModels: pc.BlacklistedModels` copy line (adapter body, NOT a route).
- **Validation is done inline, not via GORM** (bf-gov-1 precedent D2): the
  `ValidateBudgetOwner` value-in/error-out function (`quota.go:41-53`) is the
  established pattern — bf-gov-2 mirrors it with `WhiteList.Validate()`/
  `BlackList.Validate()` called from the VK admin path (D5).
- **No provider-key model store exists**: `schemas.Key` (`provider.go:30-35`) is a
  credential (`ID/Provider/Value/ProviderSpecificData`) with NO model fields and NO
  store file (`store.Connection`, `connections.go:13-25`, is the persisted
  credential — also no model fields). This is WHY rows 029/048 are ESC (§3).
- **Schemas/json conventions**: snake_case json tags (`governance.go`), no `init()`,
  errors-as-values (`fmt.Errorf("ctx: %w")`), no global state. lists.go follows all.

---

## 1. Decisions made (and why) — binding

### D1 — `WhiteList` type encodes matrix empty=deny-all; the GATE preserves legacy match-all (backward-compat VAR)

The matrix is explicit (PAR-BF-GOV-026, `account.go:22-30`): for a `WhiteList`,
**empty list denies all**; `["*"]` allows all; a non-empty list without `*` allows
only the listed values.

g0router's CURRENT gate treats `len(cfg.AllowedModels)==0` — **BOTH nil AND empty
`[]`** — as **match-all** (`vk.go:106-108`). The matrix's empty=deny-all is the
opposite for the empty case.

**Decision — a deliberate split between the TYPE (parity artifact) and the GATE
(backward-compat):**

1. **TYPE** — Build `schemas.WhiteList []string` with `IsAllowed(value string) bool`
   encoding the matrix contract EXACTLY:
   - contains `"*"` → `true` (allow all),
   - empty (`len==0`) → **`false`** (deny all, per matrix 026),
   - else → membership test (exact string match).

   This is unit-tested directly for all three cases (§5/D7) — the type faithfully
   encodes the documented Bifrost WhiteList contract; that is the parity artifact
   that flips PAR-BF-GOV-026 to HAVE.

2. **GATE** — `matchProviderConfig` (D3) keeps the **UNCONDITIONAL legacy
   semantics**: `len(AllowedModels)==0` (nil OR empty) → **match-all**. NO
   behavior change to the whitelist path at the gate; NO reliance on a
   nil-vs-empty distinction. The whitelist still restricts when **non-empty**
   (lists specific models → only those allowed via `WhiteList.IsAllowed`), which
   is the real use case. **The gate does NOT adopt empty=deny-all.**

**Why the gate preserves legacy match-all (autonomy rule: safest low-blast-radius
default):** adopting empty=deny-all at the gate would SILENTLY flip every
already-stored VK (and any UI/client serializing `"allowed_models": []` to mean
"no restriction") from allow-all to deny-all the moment bf-gov-2 ships — live VKs
would start returning 403. That is a high-blast-radius regression for a marginal
parity gain (an empty whitelist is a degenerate config; an operator would simply
deactivate the VK instead). The genuinely-new enforcement bf-gov-2 adds at the
gate is the **blacklist pass** (blacklist-wins-first, D3), which is entirely
unaffected by this.

This gate behavior (legacy match-all on empty `AllowedModels`) is recorded as a
deliberate g0router **VAR** for backward-compat — exactly the kind of VAR bf-gov-1
used for `IsActive` (NOT-NULL-DEFAULT-1 column). Adopting strict empty=deny-all at
the gate is a future opt-in setting, NOT built here (recorded in
`open-questions.md`).

Cross-reference: matrix quirk #2 ("deny-by-default on empty ProviderConfigs",
`resolver.go:359-362`) is about empty *ProviderConfigs* (no provider at all),
which g0router already denies in `matchProviderConfig` (returns `false` when no
config matches) — that is NOT the empty-`AllowedModels` case and stays as-is.

### D2 — `BlackList` type + empty-list semantics = **BLOCK-NONE** + `IsBlocked` method (matrix-cited, closes 119)

The matrix (PAR-BF-GOV-027, `account.go:80-106`): for a `BlackList`, `["*"]`
blocks all; **empty blocks none**; non-empty without `*` blocks only listed.
PAR-BF-OAI-119 requires the block decision to be a **method on the type**
(`BlackList.IsBlocked`), not free-floating gate code.

**Decision:** Build `schemas.BlackList []string` with
`IsBlocked(value string) bool`:
- empty (`len==0`) → **`false`** (block none),
- contains `"*"` → `true` (block all),
- else → membership test.

Note the order vs WhiteList: for BlackList, **empty is checked first** (block
none) so an unconfigured blacklist never blocks — this is the safe default and
matches the matrix's "empty blocks none". `IsBlocked` being a method on the type
closes PAR-BF-OAI-119 (the EXTRA row).

### D3 — Blacklist-WINS-over-allowlist: 2-pass in `matchProviderConfig` (the LIVE decision)

The matrix (PAR-BF-GOV-028, `resolver.go:358-390`): two-pass — **blacklist scan
first, then allowlist scan**; a model blocked by the blacklist is denied even if
the allowlist would allow it.

**Decision — precedence (deterministic, documented), inside `matchProviderConfig`
for each provider-matching config:**
1. **Blacklist pass (FIRST):** if `cfg.BlacklistedModels.IsBlocked(model)` →
   this config does NOT match (skip it; equivalent to "blocked"). Blacklist wins.
2. **Allowlist pass (SECOND):** else apply D1's GATE rule — config matches iff
   `len(AllowedModels)==0` (nil OR empty → legacy match-all VAR, backward-compat)
   OR `WhiteList(AllowedModels).IsAllowed(model)` (the non-empty whitelist
   restricts to listed models). The gate does NOT treat empty as deny-all.

A model that is both blacklisted and allowlisted is **denied** (blacklist wins) —
the binding behavioral assertion (§5). When NO provider config matches (all
blocked or none allow the model), `matchProviderConfig` returns `(nil, false)` and
`AllowVK` returns the existing `403 "provider/model not allowed for virtual key"`
(`vk.go:80-81`) — **no new HTTP status, no new error envelope** (the matrix gives
no distinct wire shape for "blocked"; ESC-REF-ABSENT forbids inventing one; record
the "distinct ModelBlocked decision/reason" question in `open-questions.md` —
that typed `Decision` enum is bf-gov-3 scope, BIFROST-MAP §249). The reason string
stays the existing one to preserve the `{data,error}` contract.

**No engine touch:** this is entirely inside `vk.go` (the api layer). quota.go is
UNTOUCHED — bf-gov-2 holds NO internal serial.

### D4 — `BlacklistedModels` placement: additive field on `ProviderConfig`, rides `config_json` (NO new column)

PAR-BF-GOV-030 wants a provider-config-level blacklist. bf-gov-1's `AllowAllKeys`
established the exact pattern: additive field on `schemas.ProviderConfig`,
serialized into the existing `config_json` blob (`virtualkeys.go:24-52`), copied
through `storeVKToAPI` into `api.VKProviderConfig`.

**Decision:** Add `BlacklistedModels schemas.BlackList \`json:"blacklisted_models,omitempty"\``
to `schemas.ProviderConfig` (`governance.go:14-20`). It rides `config_json` — NO
new column, NO migration. On read, the existing `json.Unmarshal` in
`unmarshalVirtualKeyConfig` (`virtualkeys.go:43-52`) populates it — this IS the
g0router analogue of Bifrost's `AfterFind` JSON deserialization (the in-scope
config-level half of 048; the key-level half is ESC §3). `omitempty` keeps blobs
of pre-bf-gov-2 VKs byte-stable (absent field unmarshals to nil = block-none, D2).
`api.VKProviderConfig` gains a matching `BlacklistedModels schemas.BlackList`
field; `storeVKToAPI` copies it (additive adapter line, `routes_openai.go:198-203`).

### D5 — Validation behavior inline (NOT GORM `BeforeSave`)

PAR-BF-GOV-037's *mechanism* is GORM `BeforeSave` (ESC — g0router has no GORM,
bf-gov-1 D2 precedent). The *validation behavior* (validate `AllowedModels` +
`BlacklistedModels`) is buildable inline.

**Decision:** Add `WhiteList.Validate() error` and `BlackList.Validate() error`
to lists.go. Per the matrix the only documented validation is the list-semantics
themselves; with no ref to confirm a richer rule (ESC-REF-ABSENT), `Validate`
enforces the minimal documented invariant: **a list may not mix `"*"` with other
entries** (a `["*", "gpt-4"]` is ambiguous/redundant — `*` already covers all),
returning `fmt.Errorf("whitelist: '*' cannot be combined with explicit models")`
(resp. blacklist). If impl finds the matrix/ref documents NO such rule, `Validate`
returns nil for all well-formed lists and the mix-rule question is recorded in
`open-questions.md` rather than invented. **No-leftovers (binding):** `Validate`
MUST be called from a LIVE production path — the VK admin create/update handler
(the analogue of bf-gov-1's `ValidateBudgetOwner` call in
`internal/admin/virtualkeys.go`). If T-validate finds no live caller, it MUST be
wired into the VK create/update path (reject saving a VK whose provider-config
lists are invalid) so it is genuinely consumed — else STOP + escalate (§3).

### D6 — `["*"]` is the match-all sentinel for BOTH lists (matrix-cited)

The matrix uses the literal string `"*"` as the wildcard for both `WhiteList`
(allow-all) and `BlackList` (block-all). **Decision:** the sentinel is the exact
string `"*"`; `IsAllowed`/`IsBlocked` scan for it before/with the membership test
per D1/D2. No glob/prefix matching (e.g. `gpt-*`) is built — the matrix documents
only the literal `["*"]` all-case, not pattern globbing; a glob variant is recorded
in `open-questions.md` as a possible additive follow-up, NOT built (ESC-REF-ABSENT —
no ref to confirm glob semantics).

### D7 — Hermetic, table-driven tests (no net/sleep/real-clock)

lists.go is pure (no clock, no IO). **Decision:** ALL bf-gov-2 tests are
**table-driven** over the allow/block matrix and hermetic (no `time.Now`, no
`time.Sleep`, no network, no subprocess). The list-semantics tables MUST cover, for
each of `IsAllowed`/`IsBlocked`: `["*"]`, empty, single listed (hit + miss), and
the 2-pass cross-case (blocked-overrides-allowed). The `matchProviderConfig`/
`AllowVK` tests reuse the existing `vk_test.go` fakes (no store, no DB). Binding
(Wave-7 hermetic lesson, BIFROST-MAP §494).

---

## 2. Target files

### IN-SCOPE — NEW + EXTEND (additive only)

| File | Change (additive ONLY) |
|---|---|
| `internal/schemas/lists.go` | **NEW.** `type WhiteList []string` + `IsAllowed(string) bool` (D1) + `Validate() error` (D5); `type BlackList []string` + `IsBlocked(string) bool` (D2/119) + `Validate() error` (D5). Pure, no imports beyond stdlib. snake_case n/a (slices). |
| `internal/schemas/lists_test.go` | **NEW.** RED first: table-driven `IsAllowed`/`IsBlocked` over `["*"]`/empty/listed (hit+miss) + `Validate` mix-rule cases (D7). |
| `internal/schemas/governance.go` | EXTEND: change `ProviderConfig.AllowedModels []string` → keep `[]string` (wire-compatible) and ADD `BlacklistedModels schemas.BlackList \`json:"blacklisted_models,omitempty"\`` (D4). **Do NOT retype `AllowedModels` to `WhiteList` in the struct** — keep `[]string` for blob wire-stability; convert to `WhiteList` at the decision point (D1/D3) to avoid churning every existing JSON blob and DTO. (If retyping is cleaner and proven blob-stable in T-schema, that is an allowed VAR — default to the no-retype path.) |
| `internal/api/vk.go` | EXTEND: ADD `BlacklistedModels schemas.BlackList` to `api.VKProviderConfig`; rewrite `matchProviderConfig` to the 2-pass (blacklist `IsBlocked` first → skip config; then allowlist where `len(AllowedModels)==0` (nil OR empty) → legacy match-all VAR, else `WhiteList(AllowedModels).IsAllowed`, D3). The gate does NOT adopt empty=deny-all (backward-compat, D1). PRESERVE `AllowVK`/`NewVKGate`/`matchProviderConfig` SIGNATURES (the function body changes; the `(*VKProviderConfig, bool)` return is unchanged). |
| `internal/api/vk_test.go` | EXTEND: RED first: blacklist-wins (blocked model denied even when allowlisted); `["*"]` blacklist blocks all; non-empty whitelist restricts (listed allowed, unlisted denied); empty AND nil `AllowedModels` → legacy match-all (allowed, no new denial — VAR D1). The empty=deny-all case is asserted at the TYPE level in `lists_test.go`, NOT here (D7). |
| `internal/server/routes_openai.go` | EXTEND `storeVKToAPI` (`:198-203`): ADD `BlacklistedModels: pc.BlacklistedModels` to the `api.VKProviderConfig` literal. **Adapter body only — NO route registration.** |
| `internal/admin/virtualkeys.go` (EXTEND) | Wire the D5 live consumer: call `WhiteList(pc.AllowedModels).Validate()` + `pc.BlacklistedModels.Validate()` in the VK create/update handler; reject (400) on error. (Mirror bf-gov-1's `ValidateBudgetOwner` call site here. If the handler shape differs at impl, place the call at the same store-bound validation point; STOP+escalate if no live save path exists.) |
| `internal/admin/virtualkeys_test.go` (EXTEND/CREATE) | RED first: creating/updating a VK with an invalid provider-config list (per D5 rule) is rejected; valid lists round-trip incl. `blacklisted_models` through `config_json`. |

### FORBIDDEN (automatic REJECT if touched)

- `internal/governance/quota.go` — the bf-gov-1 → bf-gov-3 INTERNAL SERIAL.
  bf-gov-2 does NOT touch it (model filtering is an api-layer/`vk.go` concern, not
  a quota concern). Touching it is an automatic reject.
- Any **new route registration** in `routes_openai.go` or `routes_admin.go`
  (only the `storeVKToAPI` adapter body changes).
- Any **new column / migration** in `internal/store/migrate.go` —
  `BlacklistedModels` rides the existing `config_json` blob (D4). NO ensureColumn
  row for blacklists.
- `internal/store/connections.go` / any provider-**key**-level blacklist field /
  any new provider-key model store — that is the ESC root of rows 029/048 (§3).
- The `Decision`/`EvaluationResult` enum, dual-dimension rate limits,
  calendar-aligned reset, streaming `UsageUpdate`, 10s sync worker — all
  **bf-gov-3**.
- `internal/schemas/lists.go` must NOT import store/governance/api (it is a pure
  leaf type package). No `init()`. No global state.
- Any model-catalog / cross-provider allowlist (031, quirk #9) — **ESC**.
- Any UI file (`ui/**`) — bf-gov-2 is Go-only; the VK admin page already consumes
  the VK CRUD; the additive `blacklisted_models` field surfaces to no asserted UI
  behavior (the page may ignore it).

---

## 3. Scope / Non-goals — explicit ESC list

**bf-gov-2 builds ONLY the typed list semantics + the provider-config-level
blacklist filter.** The following are **ESC** (recorded in `open-questions.md`):

| ESC item | Matrix row(s) | Why ESC |
|---|---|---|
| **Provider key-level blacklists** (`TableKey.BlacklistedModels` + `BlacklistedModelsJSON` + key `AfterFind` deserialize) | 029, 048 | g0router has NO provider-key model store: `schemas.Key` is a bare credential (`provider.go:30-35`), `store.Connection` (`connections.go:13-25`) persists secrets only, and VKs pin keys via `KeyIDs []string`. There is no live single-provider-key model-gate to consume a key-level blacklist → adding the field would be a DEAD field (no-leftovers violation §3). ESC until a provider-key model-policy store is funded. |
| **Model-catalog cross-provider allowlist** (`IsModelAllowedForProvider` catalog delegation) | 031, quirk #9 | g0router has no model catalog (BIFROST-MAP §256). The simple string-membership allowlist (D1) is the documented baseline; cross-provider name resolution is a divergent rework. |
| **`Decision`/`EvaluationResult` enum** mapping for a distinct "ModelBlocked" outcome | 035, 036 | bf-gov-3 (BIFROST-MAP §249). bf-gov-2 reuses the existing `(bool,int,string)` 403 tuple for a blocked model (D3); no new enum. |
| **GORM `BeforeSave` hook mechanism** for list validation | 037 (mechanism), 048 (mechanism) | g0router has no GORM (bf-gov-1 D2). The *validation behavior* (037) IS built inline (D5); the hook *mechanism* + key `AfterFind` (048) are ESC. |
| **`["*"]` glob/prefix matching** (e.g. `gpt-*`) | — (not in matrix) | The matrix documents only the literal `["*"]` all-case (D6); glob semantics are unverifiable (ESC-REF-ABSENT). Recorded as a possible additive follow-up, not built. |
| **Deny-by-default on empty ProviderConfigs** as a NEW behavior | quirk #2 | Already satisfied — `matchProviderConfig` returns `false` when no config matches (`vk.go:115`). No change; not a bf-gov-2 build item. |

**No-leftovers (binding, §3 CLI_ORCHESTRATOR):** bf-gov-2 adds the
`WhiteList`/`BlackList` types ONLY because `matchProviderConfig` (D1/D3) and the
admin validation path (D5) consume them; it adds `ProviderConfig.BlacklistedModels`
ONLY because the D3 blacklist pass reads it. D5 carries an explicit STOP-condition
if `Validate` has no live caller. NO field/type/method ships without a grep-proven
live consumer (§5) — else the plan STOPS and escalates.

---

## 4. Task graph (TDD; `N. [step] -> verify: [check]`)

Cadence (AGENTS.md "TDD always"): no impl file/field lands before its covering
`_test.go` is committed RED. `go test ./... && go vet ./... && go build ./...`
green at EVERY commit. bf-gov-2 holds NO serial slot; runs ∥ bf-gov-1.

1. **[list types, RED]** Write `internal/schemas/lists_test.go` (table-driven,
   D7): `WhiteList.IsAllowed` for `["*"]`/empty/listed-hit/listed-miss;
   `BlackList.IsBlocked` for empty/`["*"]`/listed-hit/listed-miss; `Validate`
   mix-rule cases. -> verify: `go test ./internal/schemas/ -run 'List'` FAILS
   (types undefined). Commit RED:
   `phase-1/bf-gov-2: failing WhiteList/BlackList semantics tests (TDD red)`.

2. **[list types, GREEN]** Add `internal/schemas/lists.go`: `WhiteList`/`BlackList`
   + `IsAllowed`/`IsBlocked` (D1/D2/D6, closes 119) + `Validate` (D5). -> verify:
   `go test ./internal/schemas/... && go vet ./... && go build ./...` exit 0; the
   matrix table cases pass (empty whitelist denies, empty blacklist blocks none,
   `["*"]` both, blocked-overrides in the type-level cross test). Commit:
   `phase-1/bf-gov-2: typed WhiteList/BlackList with IsAllowed/IsBlocked`.

3. **[schema blacklist field, GREEN]** Add `ProviderConfig.BlacklistedModels
   schemas.BlackList \`json:"blacklisted_models,omitempty"\`` to
   `governance.go` (D4, keep `AllowedModels []string`). -> verify:
   `go build ./...` exit 0; `grep -n 'blacklisted_models' internal/schemas/governance.go`
   non-empty; existing VK store round-trip tests still green (blob wire-stable —
   `go test ./internal/store/ -run VirtualKey`). Commit:
   `phase-1/bf-gov-2: additive provider-config blacklisted_models (rides config_json)`.

4. **[2-pass gate, RED]** Extend `internal/api/vk_test.go` (D7 hermetic): add
   `BlacklistedModels` to test configs; blacklist-wins (model in BOTH lists →
   denied 403); `["*"]` blacklist → block all; non-empty whitelist restricts
   (listed model allowed, unlisted model denied); empty AND nil `AllowedModels`
   → legacy match-all (allowed, NO new denial — the backward-compat VAR, D1).
   The empty=deny-all assertion lives in `lists_test.go` at the TYPE level (step
   1), NOT at the gate. -> verify: `go test ./internal/api/ -run 'VK|Match'`
   FAILS. Commit RED:
   `phase-1/bf-gov-2: failing 2-pass blacklist-wins gate tests (TDD red)`.

5. **[2-pass gate, GREEN]** Add `BlacklistedModels schemas.BlackList` to
   `api.VKProviderConfig`; rewrite `matchProviderConfig` to the 2-pass (D3:
   blacklist `IsBlocked` first → skip config; else allowlist where
   `len(AllowedModels)==0` (nil OR empty) → legacy match-all VAR, else
   `WhiteList(AllowedModels).IsAllowed`). PRESERVE signatures. -> verify:
   `go test ./internal/api/... && go vet ./... && go build ./...` exit 0;
   blacklist-wins case green. Commit:
   `phase-1/bf-gov-2: blacklist-wins 2-pass model filter in VK gate`.

6. **[adapter wiring, GREEN]** Extend `storeVKToAPI` (`routes_openai.go:198-203`)
   to copy `BlacklistedModels: pc.BlacklistedModels`. -> verify:
   `go test ./internal/server/... && go build ./...` exit 0; NO new route
   (grep proof §5). Commit:
   `phase-1/bf-gov-2: thread provider-config blacklist through VK resolver`.

7. **[validation, RED→GREEN]** Extend `internal/admin/virtualkeys_test.go`: VK
   create/update with an invalid list (D5 rule) is rejected (400); valid lists
   incl. `blacklisted_models` round-trip through `config_json`. Wire
   `WhiteList(...).Validate()` + `BlackList.Validate()` into the VK admin
   create/update handler (D5 live consumer). -> verify:
   `go test ./internal/admin/ -run VirtualKey` green; `Validate` has ≥1 live
   production caller (grep proof §5) OR STOP+escalate. Commit:
   `phase-1/bf-gov-2: inline provider-config list validation in VK admin path`.

8. **[close]** Run full validation (§6); flip matrix rows (§7); update
   `open-questions.md` (ESC list §3 + D-deferred items); update `docs/WORKFLOW.md`.
   -> verify: §6 all green; matrix + WORKFLOW + open-questions committed. Commit:
   `phase-1/bf-gov-2: close — typed allow/block-list semantics; matrix flip`.

---

## 5. Acceptance criteria (binary; file:line where possible)

**Test gates** (each yes/no, exit 0):
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/schemas/ -run 'List' -v` → all pass (≥4 IsAllowed cases +
  ≥4 IsBlocked cases + Validate cases).
- `go test ./internal/api/ -run 'VK|Match' -v` → blacklist-wins + `["*"]` +
  empty + nil cases pass.
- `go test ./internal/admin/ -run VirtualKey -v` → list-validation reject +
  round-trip pass.

**TDD-order proof** — each impl's covering test is in an earlier-or-equal commit:
```bash
for pair in \
  "internal/schemas/lists_test.go:internal/schemas/lists.go" \
  "internal/api/vk_test.go:internal/api/vk.go" \
  "internal/admin/virtualkeys_test.go:internal/admin/virtualkeys.go" ; do
  tf=${pair%%:*}; im=${pair##*:}
  ct=$(git log --format=%ct -1 -- "$tf"); cf=$(git log --format=%ct -1 -- "$im")
  [ "$ct" -le "$cf" ] || echo "TDD VIOLATION: $im"   # prints nothing
done
```

**Grep proofs:**
```bash
# typed list types exist with method-level decisions (closes 119)
grep -n "func (w WhiteList) IsAllowed\|func (b BlackList) IsBlocked" internal/schemas/lists.go
grep -n "func (w WhiteList) Validate\|func (b BlackList) Validate" internal/schemas/lists.go
# empty-list semantics are matrix-correct (whitelist empty denies; blacklist empty blocks none)
grep -n "IsAllowed\|IsBlocked" internal/schemas/lists.go
# blacklist field is additive on the config (rides config_json, NO new column)
grep -n "blacklisted_models" internal/schemas/governance.go                      # additive json field
! grep -niE 'blacklist|blacklisted' internal/store/migrate.go && echo "no blacklist column/migration OK"
# 2-pass blacklist-wins is in the LIVE gate (vk.go), NOT in quota.go
grep -n "IsBlocked\|IsAllowed\|BlacklistedModels" internal/api/vk.go              # consumed in matchProviderConfig
! grep -niE 'blacklist|IsBlocked|WhiteList' internal/governance/quota.go && echo "quota.go UNTOUCHED OK"
# blacklist threaded through the adapter (no new route)
grep -n "BlacklistedModels" internal/server/routes_openai.go
! grep -nE 'r\.(GET|POST|PUT|DELETE)\(' internal/server/routes_openai.go | grep -iE 'virtual|blacklist' && echo "no new route OK"
# validation has a live (non-test) caller (no-leftovers D5)
grep -rn "\.Validate()" internal/admin/virtualkeys.go                            # ≥1 production caller
# lists.go is a pure leaf (no store/governance/api import; no init/global)
! grep -nE 'internal/(store|governance|api)' internal/schemas/lists.go && echo "pure leaf OK"
! grep -n "func init(" internal/schemas/lists.go internal/api/vk.go && echo "no init() OK"
# key-level blacklist is NOT built (ESC 029/048 — no dead field)
! grep -niE 'blacklist' internal/store/connections.go internal/schemas/provider.go && echo "no key-level blacklist field OK"
```

**Behavioral acceptance (binary):**
- A model present in a provider config's `BlacklistedModels` is **DENIED** (403
  `"provider/model not allowed for virtual key"`) **even when it also appears in
  `AllowedModels`** — blacklist wins (D3). (Proven via `vk_test.go`.)
- At the **TYPE level (unit test)**: an empty `WhiteList` → `IsAllowed` returns
  false for every value (deny-all, matrix 026); a `["*"]` `WhiteList` → true for
  every value; a listed `WhiteList` → membership.
- At the **GATE level**: a config with empty OR nil `AllowedModels` is treated as
  legacy match-all (no previously-allowed VK is newly denied); a non-empty
  `AllowedModels` restricts to the listed models. The gate does NOT adopt
  empty=deny-all (backward-compat VAR, D1).
- An empty `BlackList` → `IsBlocked` returns false for every value (block-none,
  matrix 027); a `["*"]` `BlackList` → true for every value.
- A VK whose `AllowedModels` is **nil OR empty** behaves byte-identically to
  pre-bf-gov-2 (legacy match-all, VAR D1) — no previously-allowed request is
  newly denied.
- Creating/updating a VK with an invalid provider-config list is rejected (D5);
  valid `blacklisted_models` round-trips through `config_json`.

---

## 6. Validation commands

```bash
go test ./... && go vet ./... && go build ./...                 # exit 0 (binding)
go test ./internal/schemas/ -run 'List' -v
go test ./internal/api/ -run 'VK|Match' -v
go test ./internal/admin/ -run VirtualKey -v
```
No UI build / Playwright needed — bf-gov-2 ships NO UI touch and NO mock
correction (the additive `blacklisted_models` field surfaces to no asserted UI
behavior). Hermetic only (D7): no test may hit the network, sleep, or call real
`time.Now` (lists.go is pure; gate tests use existing fakes).

---

## 7. Freeze rules + matrix-flip + WORKFLOW + no-leftovers

**Freeze rules (binding):**
- `internal/governance/quota.go` — **NOT TOUCHED** by bf-gov-2 (it is the
  bf-gov-1 → bf-gov-3 internal serial; model filtering is an api-layer concern).
  bf-gov-2 takes NO internal serial slot and runs ∥ bf-gov-1 (BIFROST-MAP §330).
- bf-gov-2 is **NOT** a `routes_admin.go` holder and **NOT** a `routes_openai.go`
  route-block holder (it edits only the `storeVKToAPI` adapter body, never the
  route-registration block; §0.3 / §2 forbidden). It takes NO serial route slot.
- `internal/schemas/lists.go` is a NEW disjoint leaf file — no conflict surface.
- Migrations: **none.** `BlacklistedModels` rides `config_json` (D4); NO
  ensureColumn row, NO destructive DDL.
- No reverse-engineering of the absent Bifrost ref (ESC-REF-ABSENT) — build to
  matrix + g0router conventions only. The empty-list semantics (D1/D2), `["*"]`
  sentinel (D6), and 2-pass precedence (D3) are all matrix-cited
  (026/027/028/119); the gate's legacy-match-all-on-empty backward-compat VAR
  (D1) and the `Validate` mix-rule (D5) are conservative g0router choices
  recorded in open-questions where the matrix is silent.

**Matrix-flip (at close, in `.planning/parity/matrix/bifrost-governance.md` +
`bifrost-openai.md`):**
- PAR-BF-GOV-026 → **HAVE** (WhiteList type implements the matrix contract incl.
  empty=deny-all per D1, unit-tested; the GATE deliberately preserves legacy
  match-all on empty `AllowedModels` for backward-compat — VAR note, D1; cite
  bf-gov-2). Do NOT claim the gate adopts empty=deny-all.
- PAR-BF-GOV-027 → **HAVE** (BlackList type + empty=block-none per D2).
- PAR-BF-GOV-028 → **HAVE** (blacklist-wins 2-pass in matchProviderConfig per D3;
  VK provider-config slice).
- PAR-BF-GOV-029 → **MISSING (ESC)** (key-level blacklist — no provider-key model
  store; note bf-gov-2 ESC + rationale §3).
- PAR-BF-GOV-030 → **HAVE** (provider-config-level blacklist, rides config_json, D4).
- PAR-BF-GOV-037 → **HAVE** (inline list validation per D5; GORM BeforeSave ESC).
- PAR-BF-GOV-048 → **MISSING (ESC)** (key-level AfterFind deserialize — same root
  as 029; the config-level JSON deserialize is the existing config_json unmarshal,
  not a new hook).
- PAR-BF-OAI-119 → **HAVE** (IsBlocked is a method on the BlackList type per D2).

**`open-questions.md` (append at close):**
```
## bf-gov-2 — typed WhiteList/BlackList allow/block semantics — 2026-06-15
- [ ] Provider key-level blacklists (gov 029/048) — ESC; g0router has no provider-key model store (schemas.Key is a bare credential, store.Connection persists secrets only, keys pinned via KeyIDs). Building a key-level blacklist would be a dead field (no live single-key model-gate). Why: needs operator decision to fund a provider-key model-policy store.
- [ ] WhiteList empty=deny-all is implemented + unit-tested at the TYPE level (schemas.WhiteList.IsAllowed); the GATE (matchProviderConfig) deliberately preserves legacy match-all on empty `AllowedModels` (nil OR empty) for backward-compat (VAR, D1). Adopting strict empty=deny-all at the gate is a future opt-in setting, NOT built — to avoid silently denying every existing VK that serializes "allowed_models": [] as "no restriction". Why: confirm whether/when operators want a strict-gate opt-in.
- [ ] Distinct "ModelBlocked" decision/reason (gov 035/036 Decision enum) — bf-gov-2 reuses the existing 403 "provider/model not allowed" reason for a blacklisted model. Why: a distinct enum/reason is bf-gov-3 scope; record so the wire shape isn't invented (ESC-REF-ABSENT).
- [ ] List `["*"]` glob/prefix matching (e.g. gpt-*) — NOT built; matrix documents only the literal ["*"] all-case (D6). Why: glob semantics unverifiable without the ref; possible additive follow-up.
- [ ] WhiteList/BlackList.Validate mix-rule (D5) — defaults to "'*' cannot combine with explicit entries"; if the ref documents no such rule, Validate is a no-op for well-formed lists. Why: ESC-REF-ABSENT; confirm the intended validation contract.
```

**`docs/WORKFLOW.md` (update at close):** add a bf-gov-2 row — typed
WhiteList/BlackList allow/block semantics shipped (Go-only, no routes, no UI, no
migration, quota.go untouched); rows 026/027/028/030/037 + OAI-119 flipped to HAVE;
029/048 recorded ESC; ESC items in open-questions; ESC-REF-ABSENT honored (built to
matrix only).

**No-leftovers confirmation (binding):** bf-gov-2 adds `WhiteList`/`BlackList`
(consumed by `matchProviderConfig` D1/D3 + admin Validate D5),
`ProviderConfig.BlacklistedModels` (consumed by the D3 blacklist pass), and the
`api.VKProviderConfig.BlacklistedModels` field (consumed end-to-end through
`storeVKToAPI` → `matchProviderConfig`). No dead type, field, or method is
introduced; each new surface has a grep-proven live consumer (§5), or the plan
STOPS and escalates. Rows 029/048 are NOT built (would be dead fields) and are
honestly marked ESC rather than fabricated.
```
