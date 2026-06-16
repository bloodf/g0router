# bf-gov-5 — VK bearer value encrypted at rest (PAR-BF-GOV-006)

**Parity row:** PAR-BF-GOV-006 (bifrost-governance matrix) — "VK value encrypted at rest:
SHA-256 hash for lookup, AES encryption at rest."

**Gap:** The VK bearer value (`virtual_keys.key`, e.g. `g0vk-<hex>`) is stored AND served in
PLAINTEXT, violating the AGENTS.md decision: *"Secrets encrypted at rest via reversible `*_enc`
columns (pattern: `internal/store/oauthsessions.go`)."* This is a genuine, in-scope, additive gap.

**Scope:** The VK BEARER VALUE only. `config_json` (budgets/limits/provider configs) is OUT of
scope.

**Commit prefix:** `phase-1/bf-gov-5:` · **Footer:** `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`

---

## Verified grounding (read against code — all confirmed correct)

| Fact | Location | Verified |
|---|---|---|
| `Cipher` AES-256-GCM, `Encrypt(string)(string,error)` / `Decrypt(string)(string,error)`, reversible | `internal/store/crypto.go:13,34,44` | ✅ |
| `Open(path, secret)` creates cipher (`:27`), runs `migrate(db)` db-only no cipher (`:49`), then `return &Store{db, cipher, apiKeyGenerator}` (`:54`) | `internal/store/store.go:26-55` | ✅ migrate **cannot** backfill (no cipher); backfill must run AFTER Store construction |
| `CreateVirtualKey`: builds `Key: "g0vk-"+key`, `INSERT ... (id, key, name, config_json, is_active, team_id, created_at, updated_at)` storing raw key | `internal/store/virtualkeys.go:58-97` | ✅ |
| `GetVirtualKeyByKey(key)` → `WHERE key = ?` with raw key | `internal/store/virtualkeys.go:130-133` | ✅ |
| `scanVirtualKey` scans `id, key, name, config_json, is_active, team_id, created_at, updated_at` | `internal/store/virtualkeys.go:176-194` | ✅ |
| All 4 SELECTs (`List`/`GetByID`/`GetByKey` + scan) select `key` directly | `virtualkeys.go:100-133,182` | ✅ |
| `UpdateVirtualKey` UPDATE sets `name, config_json, is_active, team_id, updated_at` — does **NOT** touch `key` | `internal/store/virtualkeys.go:144-147` | ✅ **no clobber risk** — no fix needed |
| `toVirtualKeyDTO` returns `Key: vk.Key` (plaintext one-time + on every list/get) — MUST be preserved | `internal/admin/virtualkeys.go:31` | ✅ |
| schema `key TEXT NOT NULL UNIQUE` + `idx_virtual_keys_key` | `migrate.go:75-83,383` | ✅ |
| `ensureColumn(db, table, column, decl)` — additive ALTER ADD COLUMN, idempotent | `migrate.go:532-569` | ✅ |
| `team_id` is the existing additive-column precedent in the same `virtual_keys` block | `migrate.go:454` | ✅ |
| Test harness: `newTestStore(t)` → `LoadOrCreateSecret(dir)` → `Open(path, secret)` (real 32-byte secret injected) | `internal/store/store_test.go:26-39`, `secret.go:15` | ✅ tests get a real cipher for free |

### Call-site audit (no path reads the raw `key` column expecting plaintext — confirmed)

All production consumers of the VK value go through the store methods being changed and consume
`vk.Key`, which the decrypt-into-`.Key` design keeps as **plaintext**:

| Consumer | Uses | Needs plaintext `.Key`? |
|---|---|---|
| `internal/server/routes_openai.go:179` `ResolveVK` → `storeVKToAPI` → `api.VKInfo{Key: vk.Key}` | `GetVirtualKeyByKey(rawKey)` | ✅ yes (spend attribution + gate) |
| `internal/admin/mcp.go:740` MCP tool-scoping | `GetVirtualKeyByKey(vk)` | only reads `.ID` (no `.Key`) |
| `internal/admin/mcp.go:865` `admitMCPVK` | `GetVirtualKeyByKey(key)` | reads `.IsActive` (no `.Key`) |
| `internal/admin/virtualkeys.go:31` DTO display | `vk.Key` | ✅ yes (UI shows plaintext) |
| `internal/governance/quota.go:238,256,259,286,305` spend ledger | `vk.Key` keys `SumCostByAPIKey` / `rpmHits` | ✅ yes — **must equal raw key** |
| `internal/store/requestlog.go:269-285` `SumCostByTeam` (team-budget aggregate, called by `quota.go:323` `checkTeamBudget`) | **raw SQL correlates `request_log.api_key IN (SELECT key FROM virtual_keys WHERE team_id = ?)`** | ⛔ **BROKEN by the repurpose** — see BLOCKER below |

**Critical attribution chain (verified via `internal/api/usage_glue_test.go:714`: `entry.APIKey == vkKey`):**
`request_log.api_key` records the **raw VK value**, and `quota.go` sums spend keyed on `vk.Key`.
If `vk.Key` ever became the hash, every budget would silently reset to zero. The decrypt-into-`.Key`
design keeps `vk.Key` the raw plaintext at every read, so `SumCostByAPIKey` (binds raw `vk.Key` as
`WHERE api_key = ?`) stays consistent. **A test must pin this.**

### ⛔ BLOCKER — `SumCostByTeam` raw-SQL correlation breaks silently (call-site audit miss)

`internal/store/requestlog.go:269-285` (`SumCostByTeam`) is the live team-budget aggregate
(`quota.go:323` `checkTeamBudget`). It correlates the two columns **inside SQL**:

```sql
SELECT SUM(cost) FROM request_log
WHERE api_key IN (SELECT key FROM virtual_keys WHERE team_id = ?)
  AND timestamp >= ?
```

`request_log.api_key` stores the **RAW** VK value; after the repurpose `virtual_keys.key` holds the
**HASH**, so the subquery matches NOTHING → `SumCostByTeam` silently returns 0 for every team →
**team budget enforcement silently stops**. SQLite has no built-in `sha256`, so this CANNOT be fixed
inside the SQL. **Verified blast radius:** this is the ONLY broken correlation —
`SumCostByAPIKey` / `SumTokensByAPIKey` / `SumRequestsByAPIKey` (`requestlog.go`) all bind the raw key
as `WHERE api_key = ?` (fine), and every other `FROM virtual_keys` read is a store VK method already
covered above. **Fix:** rewrite `SumCostByTeam` to a Go-mediated two-step (Step B GREEN), sequenced in
the SAME commit group that makes `virtual_keys.key` the hash, so the suite never has a window where
team-cost is silently 0.

### Confirmed-design corrections vs. the brief

- **BLOCKER (added in review): `SumCostByTeam` must be rewritten** — see the BLOCKER box above. The brief
  did not cover the raw-SQL `api_key IN (SELECT key FROM virtual_keys …)` correlation in
  `requestlog.go:269-285`; it silently zeroes every team budget once `key` holds the hash. Fixed in the
  Step B GREEN commit (same group as the hash repurpose).
- **UpdateVirtualKey needs NO change** — its UPDATE already omits `key`. The plan keeps a *test* that
  proves Update does not clobber `key`/`key_enc` (regression guard), but no impl edit there.
- **Existing test breakage identified:** `virtualkeys_test.go:137-148` ("Duplicate key value rejected")
  inserts `created1.Key` (raw) directly into the `key` column expecting a UNIQUE violation. After the
  repurpose, the `key` column holds the *hash*, so a raw-value insert will NOT collide and the test
  will fail. **This test must be updated** to insert the hash (`sha256hex(created1.Key)`) to still
  prove UNIQUE-on-`key`. (Tracked as Step D, and must land no later than Step B to keep the suite green.)

---

## Design (confirmed)

1. **migrate.go** — add to the existing additive-column loop:
   `{"virtual_keys", "key_enc", "TEXT NOT NULL DEFAULT ''"}`. Additive, idempotent, no constraint change.
2. **Repurpose `key` column** to store `sha256hex(rawKey)` (stays NOT NULL + UNIQUE — no schema change,
   the hash is non-empty and unique). New helper `sha256hex(s string) string` (`crypto/sha256` +
   `encoding/hex`).
3. **CreateVirtualKey** — store `key = sha256hex(raw)`, `key_enc = s.cipher.Encrypt(raw)`; the RETURNED
   struct keeps `.Key = raw` (one-time reveal in the create response — preserves current behavior).
4. **scanVirtualKey + every SELECT** — also select `key_enc`; decrypt into `.Key` so the returned
   struct's `.Key` is ALWAYS plaintext (DTO display + spend attribution + gate). The `key` column (hash)
   is used ONLY for lookup and never surfaced in the struct.
5. **GetVirtualKeyByKey(rawKey)** — `WHERE key = sha256hex(rawKey)`.
6. **UpdateVirtualKey** — already leaves `key`/`key_enc` intact; no change (regression test only).
7. **`SumCostByTeam` rewrite (BLOCKER fix)** — replace the single raw-SQL subquery with a Go-mediated
   two-step on `*Store` (`s.cipher` available): (1) `SELECT key_enc FROM virtual_keys WHERE team_id = ?`
   → `s.cipher.Decrypt` each → slice of raw keys; (2) if the slice is EMPTY, return 0 (guard —
   `api_key IN ()` is invalid SQL); (3) `SELECT SUM(cost) FROM request_log WHERE api_key IN (?,?,…) AND
   timestamp >= ?` with the decrypted raw keys + `sinceISO` as bound params (dynamic placeholder list).
   Must land in the SAME commit group as the hash repurpose.
8. **One-time backfill** — `(s *Store) backfillVirtualKeyEncryption() error`: for rows `WHERE key_enc = ''`,
   read plaintext `key`, set `key_enc = Encrypt(plaintext)` and `key = sha256hex(plaintext)`. Idempotent
   via the `key_enc=''` guard. Sequenced in `Open()` AFTER Store construction (signature unchanged).

### Exact migrate change

In `migrate.go`, the additive-column loop (currently ending at `{"virtual_keys", "team_id", ...}`,
`:454`), append one entry:

```go
{"virtual_keys", "key_enc", "TEXT NOT NULL DEFAULT ''"},
```

No index change. `key` stays `TEXT NOT NULL UNIQUE` with `idx_virtual_keys_key` (now indexing hashes —
still correct for the `WHERE key = ?` lookup).

### Exact Open() restructure (`store.go:54`)

Replace the single `return` with:

```go
s := &Store{db: db, cipher: cipher, apiKeyGenerator: defaultAPIKeyGenerator}
if err := s.backfillVirtualKeyEncryption(); err != nil {
    db.Close()
    return nil, fmt.Errorf("backfill virtual key encryption: %w", err)
}
return s, nil
```

Signature `Open(path string, secret []byte) (*Store, error)` is unchanged. No New()/constructor
signature changes.

---

## TDD commit sequence (RED before GREEN at every step)

> Hermetic only: in-memory/`t.TempDir()` SQLite via `newTestStore(t)` (injects a real 32-byte secret).
> NO network/sleep/subprocess. `go test ./...` + `go vet ./...` GREEN at every commit.
> NEVER `git add -A`; stage only the named files. NEVER stage/revert `ui/dist/index.html`.

### Step 1 — `key_enc` column exists (migration) — RED then GREEN
- **RED** (`internal/store/virtualkeys_enc_test.go`, new): `TestVirtualKeyKeyEncColumnExists` opens a
  store via `newTestStore(t)`, runs `PRAGMA table_info(virtual_keys)`, asserts a `key_enc` column is
  present. Fails today (column absent).
- **GREEN**: add `{"virtual_keys", "key_enc", "TEXT NOT NULL DEFAULT ''"}` to the `migrate.go` loop.
- Commit: `phase-1/bf-gov-5: add additive key_enc column to virtual_keys`
- Stage: `internal/store/migrate.go internal/store/virtualkeys_enc_test.go`

### Step 2 — `sha256hex` helper — RED then GREEN
- **RED** (`internal/store/crypto_hash_test.go`, new): `TestSHA256Hex` asserts `sha256hex("g0vk-abc")`
  equals the known 64-char lowercase hex digest of that string and is deterministic. Fails (undefined).
- **GREEN**: add `func sha256hex(s string) string` to `internal/store/crypto.go` (`crypto/sha256` +
  `encoding/hex`). LIVE consumer arrives in Steps 3/5 (no-leftovers — helper is exercised by the
  Create/Get impl, not just its unit test).
- Commit: `phase-1/bf-gov-5: add sha256hex lookup-hash helper`
- Stage: `internal/store/crypto.go internal/store/crypto_hash_test.go`

### Step A (RED) — add ALL encryption tests in one failing commit
> **Why one pair (Revision 2 — suite-green invariant):** the existing suite at
> `virtualkeys_test.go:57` and `:191` does `Create → GetVirtualKeyByKey(created.Key)`. The moment Create
> stores the hash, `GetVirtualKeyByKey(raw)` breaks and stays broken until the lookup is rehashed AND
> scan decrypts. If those landed as separate GREEN commits the suite would be RED in between. So: one RED
> commit adds every enc test, then ONE GREEN commit (Step B) implements create-hash+enc, scan-decrypt,
> getbykey-by-hash, AND the SumCostByTeam rewrite together — the full suite goes green again at Step B.
- **RED** — add to `internal/store/virtualkeys_enc_test.go` (new) the following, all failing today:
  - `TestCreateVirtualKeyStoredAtRest` — create a VK, read the raw row via
    `st.DB().QueryRow("SELECT key, key_enc FROM virtual_keys WHERE id = ?", id)`. Assert: (a) `key` ==
    `sha256hex(created.Key)`, 64 hex chars; (b) `key` != raw `created.Key`; (c) `key_enc` non-empty;
    (d) raw plaintext appears in NEITHER column.
  - `TestVirtualKeyRoundTripPlaintext` — capture `raw := created.Key`; assert
    `GetVirtualKeyByID(id).Key == raw` and the `ListVirtualKeys()` entry `.Key == raw` (scan decrypts).
  - `TestGetVirtualKeyByKeyResolvesRaw` — `GetVirtualKeyByKey(created.Key)` (raw) resolves
    `.ID == created.ID` and `.Key == created.Key`.
  - `TestSumCostByTeamSurvivesEncryption` (load-bearing — proves team budgets still enforce):
    1. create a team VK with `TeamID: "T"` (captures `raw := created.Key`);
    2. insert a `request_log` row under the **raw** key — mirror how `request_log.api_key` is written
       (per `internal/api/usage_glue_test.go:670-714`, `api_key == vkKey`): a direct
       `INSERT INTO request_log (timestamp, api_key, cost, ...)` with `api_key = raw`, a known `cost`
       (e.g. `1.25`), and a `timestamp` >= the `since` used below;
    3. assert `SumCostByTeam("T", since)` returns that cost AFTER encryption (i.e. with
       `virtual_keys.key` holding the hash). With the un-rewritten SQL this returns 0 → RED.
- Commit: `phase-1/bf-gov-5: failing tests — VK at-rest hash+enc, round-trip, lookup, team-cost survival`
- Stage: `internal/store/virtualkeys_enc_test.go`

### Step B (GREEN) — implement hash+enc + scan-decrypt + lookup-by-hash + SumCostByTeam rewrite (one commit)
- **GREEN — all in one commit so the suite returns to green:**
  - `CreateVirtualKey` (`virtualkeys.go`): `hash := sha256hex(vk.Key)`, `enc, err := s.cipher.Encrypt(vk.Key)`;
    INSERT now writes `key = hash`, `key_enc = enc` (add the column to the INSERT). Returned struct keeps
    `.Key = vk.Key` (raw — one-time reveal preserved).
  - `scanVirtualKey` + SELECTs: add `key_enc` to all three SELECT column lists (`ListVirtualKeys` `:102`,
    `GetVirtualKeyByID` `:126`, `GetVirtualKeyByKey` `:132`); scan an extra `keyEnc string`; set
    `vk.Key = s.cipher.Decrypt(keyEnc)` (the hash column scans into a throwaway local, never surfaced).
    **`scanVirtualKey` is a package func — promote to method `(s *Store) scanVirtualKey(row ...)` so it
    has `s.cipher`** (Open question #1); update its 3 call-sites.
  - `GetVirtualKeyByKey`: change to `WHERE key = ?` with `sha256hex(key)`.
  - **`SumCostByTeam` (`requestlog.go:269-285`) — BLOCKER fix:** rewrite to the Go-mediated two-step:
    (1) `SELECT key_enc FROM virtual_keys WHERE team_id = ?` → `s.cipher.Decrypt` each into a `[]string`
    of raw keys; (2) if empty, `return 0, nil` (guard — `api_key IN ()` is invalid SQL); (3) build a
    dynamic placeholder list and run `SELECT SUM(cost) FROM request_log WHERE api_key IN (?,?,…) AND
    timestamp >= ?` binding the decrypted raw keys + `sinceISO`. Keep the existing
    `sql.NullFloat64`/`!total.Valid → 0` handling.
- Commit: `phase-1/bf-gov-5: encrypt VK value at rest (hash lookup + AES key_enc) and Go-correlate team cost`
- Stage: `internal/store/virtualkeys.go internal/store/requestlog.go`

### Step C — backfill + Open sequencing (NO-LOCKOUT, the load-bearing test) — RED then GREEN
- **RED** (`virtualkeys_backfill_test.go`, new): `TestBackfillNoLockout`:
  1. `newTestStore(t)`; capture `secret` + db path (use a `t.TempDir()` path + `LoadOrCreateSecret`
     directly so the same secret/path can be reopened — mirror `newTestStore` but keep the path/secret).
  2. Simulate an OLD row: `st.DB().Exec("INSERT INTO virtual_keys (id, key, name, config_json, is_active, team_id, created_at, updated_at) VALUES (?, ?, ?, ?, 1, '', ?, ?)", id, rawKey, "legacy", "{}", now, now)` — i.e. plaintext in `key`, `key_enc` defaults to `''`. `rawKey := "g0vk-legacyraw"`.
  3. Close, then `Open(path, secret)` again (triggers the in-`Open` backfill).
  4. Read the raw row: assert `key == sha256hex(rawKey)`, `key_enc != ''`, raw not in either column.
  5. **No-lockout:** `GetVirtualKeyByKey(rawKey)` resolves the row (`.ID == id`).
  6. **DTO/display:** the resolved `.Key == rawKey` (the original raw key — so the UI still shows it).
  - Fails today (no backfill; `GetVirtualKeyByKey(rawKey)` already broken after Step B for legacy rows,
    and `key_enc` stays empty).
- **GREEN**:
  - Add `(s *Store) backfillVirtualKeyEncryption() error` (`internal/store/virtualkeys.go`): in a single
    pass, `SELECT id, key FROM virtual_keys WHERE key_enc = ''`; for each row, `enc := Encrypt(key)`,
    `hash := sha256hex(key)`, `UPDATE virtual_keys SET key_enc = ?, key = ? WHERE id = ?`. Collect rows
    first then update (avoid iterating while writing on the single-conn DB). Idempotent via the
    `key_enc=''` guard.
  - Restructure `Open()` (`store.go:54`) per the snippet above: construct `s` first, call backfill,
    `db.Close()`+return on error, else `return s, nil`. Signature unchanged.
- Commit: `phase-1/bf-gov-5: backfill legacy plaintext VKs to hash+enc on Open (no lockout)`
- Stage: `internal/store/virtualkeys.go internal/store/store.go internal/store/virtualkeys_backfill_test.go`

### Step D — backfill idempotency + repair existing duplicate-key test — RED then GREEN
- **RED** (`virtualkeys_backfill_test.go`): `TestBackfillIdempotent` — after Step C's migrated row,
  capture `key`+`key_enc`, call `st.backfillVirtualKeyEncryption()` again, assert the `key` column and
  `key_enc` are byte-identical (no re-encrypt, no re-hash — the `key_enc != ''` guard skips it). This is
  a same-package `store` test so it can call the unexported method directly.
- **Also in this step (test repair, not new behavior):** update the existing
  `virtualkeys_test.go:137-148` duplicate-key insert to use `sha256hex(created1.Key)` instead of the raw
  `created1.Key`, so it still proves the UNIQUE constraint fires on the (now hash-valued) `key` column.
  Without this the existing test breaks once Step B lands. (Since Step B makes `key` the hash, this
  one-line test repair is REQUIRED to keep the suite green — fold it into Step B's commit if you prefer
  to keep that suite green within the same commit; either way it must land no later than Step B. Listed
  here so the duplicate-key invariant is explicitly re-proven.)
- **GREEN**: the idempotency guard already exists from Step C; this step only adds the assertion + repairs
  the legacy test. If the assertion fails, fix the guard in `backfillVirtualKeyEncryption`.
- Commit: `phase-1/bf-gov-5: prove backfill idempotency; repair duplicate-key test for hashed column`
- Stage: `internal/store/virtualkeys_backfill_test.go internal/store/virtualkeys_test.go`

### Step E — full-stack regression: spend-attribution + Update-no-clobber + green gate
- **RED/Guard** (`virtualkeys_enc_test.go`):
  - `TestUpdateDoesNotClobberKeyOrEnc` — create VK, capture `key`+`key_enc` raw columns, run
    `UpdateVirtualKey` (rename), re-read raw columns, assert `key` and `key_enc` unchanged AND
    `GetVirtualKeyByKey(raw)` still resolves. (Regression guard — should pass given Update omits `key`;
    if it ever fails, fix Update to re-write `key_enc`+hash from the existing decrypted key.)
  - `TestSpendAttributionKeyIsRaw` — assert `created.Key` (and a re-read `.Key`) has the `g0vk-` prefix
    and equals the raw value, documenting that `quota.go`'s `SumCostByAPIKey(vk.Key,...)` /
    `request_log.api_key` chain keeps summing on the raw key (no budget reset). Cross-refs
    `internal/api/usage_glue_test.go:714`.
- **GREEN**: no impl change expected; if a guard trips, fix per its note. Run full `go test ./...` +
  `go vet ./...`.
- Commit: `phase-1/bf-gov-5: regression guards — spend attribution raw key + update no-clobber`
- Stage: `internal/store/virtualkeys_enc_test.go`

### Step F — close-out: HONEST matrix flip + WORKFLOW (Revision 3 — record the residual)
- Flip PAR-BF-GOV-006 to HAVE in the bifrost-governance parity matrix; update `docs/WORKFLOW.md`.
- **The close note AND the matrix Notes cell MUST state the residual (do NOT imply total at-rest secrecy
  of the VK everywhere):**
  > "VK value is now hash-for-lookup + AES-`key_enc` at rest in the `virtual_keys` table.
  > `request_log.api_key` retains the raw VK for spend attribution (now correlated in Go via decrypt in
  > `SumCostByTeam`, not via the broken hash-column subquery) — encrypting the VK value in `request_log`
  > is a separate, larger hardening, not claimed here. GOV-006 scope = the VK record/table; the
  > request_log audit row is out of scope."
- Commit: `phase-1/bf-gov-5: close — VK value encrypted at rest in virtual_keys (GOV-006 → HAVE; request_log raw VK residual noted)`
- Stage: the matrix file + `docs/WORKFLOW.md` (named explicitly; never `-A`).

---

## No-leftovers checklist (CLI_ORCHESTRATOR §3 — every new symbol has a LIVE consumer)

| New symbol / column | Live consumer (same plan) |
|---|---|
| `virtual_keys.key_enc` column | written by `CreateVirtualKey` (Step B) + `backfill` (Step C); read+decrypted by `scanVirtualKey` (Step B) AND by the rewritten `SumCostByTeam` (Step B) |
| `sha256hex()` helper | `CreateVirtualKey` insert (Step B), `GetVirtualKeyByKey` lookup (Step B), `backfill` (Step C) |
| `backfillVirtualKeyEncryption()` | called by `Open()` (Step C) |
| decrypt-into-`.Key` in `scanVirtualKey` | every read path → DTO display, `storeVKToAPI`, quota spend attribution |
| `SumCostByTeam` Go-mediated correlation | `quota.go:323` `checkTeamBudget` (live team-budget enforcement) — rewritten in Step B |
| hash in `key` column | `GetVirtualKeyByKey` lookup; nothing surfaces it in the struct (intentional) |

No dead code: every column/method/helper is exercised by a production path **and** a hermetic test.

---

## Open questions / escalations

1. **`scanVirtualKey` needs the cipher to decrypt `key_enc`.** It is currently a package-level func
   (`virtualkeys.go:176`). Two options: (a) promote to method `(s *Store) scanVirtualKey(...)` and update
   its 3 call-sites (preferred — matches `oauthsessions.go` which decrypts inline with `s.cipher`);
   (b) keep it a func taking `cipher *Cipher` param. **Recommendation: (a).** No signature change leaks
   outside the package. Flag if the reviewer prefers (b).

2. **`Decrypt` failure on a malformed/legacy `key_enc`.** A row with `key_enc != ''` that fails to
   decrypt (wrong secret / corruption) would make `scanVirtualKey` error and the VK unreadable. Options:
   (a) propagate the error (fail-closed — safest for an auth secret; a wrong secret should not silently
   succeed); (b) fall back to treating the row as un-migrated. **Recommendation: (a) fail-closed**, since
   the only way to reach this is a changed master secret, which already breaks every other `*_enc`
   column identically (`oauthsessions`, `connections`). No new behavior. Flag if parity requires a
   softer fallback.

3. **Backfill cost on large `virtual_keys` tables.** Single pass on `Open()`, one UPDATE per legacy
   row, guarded by `key_enc=''` so it runs at most once per row across all future boots. VK counts are
   small (operator-managed). If a deployment has thousands of VKs, the first boot after upgrade does N
   AES-GCM seals + N updates — acceptable, one-time. No batching needed; flag if the reviewer wants a
   transaction wrapper around the backfill UPDATEs (single-conn DB already serializes).

4. **Index now covers hashes, not raw keys.** `idx_virtual_keys_key` is unchanged and still serves the
   `WHERE key = ?` equality lookup (now hash equality). No action; noted for completeness.

5. **Per-request hot-path cost** (`routes_openai.go` `ResolveVK`): one `sha256` (for the lookup) +,
   on hit, one AES-GCM open (decrypt `key_enc` into `.Key`). Both are microsecond-scale; acceptable.
   No caching introduced (out of scope).

6. **`SumCostByTeam` per-check cost (accepted, documented).** The rewrite decrypts ALL of a team's VK
   `key_enc` values on every team-budget check (`checkTeamBudget`, one per gated request for a teamed VK).
   That is N AES-GCM opens where N = VKs in the team — small (teams hold a handful of VKs), microsecond
   each, and it runs only on the team-budget path. Accepted as the cost of correctness (the alternative —
   a sha256 column join — would re-introduce an at-rest correlation we are removing, or require storing a
   second hash on `request_log`, a larger change). Flag if profiling later shows team-budget checks are
   hot enough to warrant caching the decrypted team→keys set.

**BLOCKER resolved in-plan (found in review):** `SumCostByTeam` (`requestlog.go:269-285`) correlated
`request_log.api_key` against the now-hashed `virtual_keys.key` in raw SQL, silently zeroing every team
budget. Fixed by the Go-mediated two-step rewrite in Step B, pinned by `TestSumCostByTeamSurvivesEncryption`
(Step A). Otherwise every brief assumption held; the remaining deviations are corrections, not blockers:
(a) `UpdateVirtualKey` needs no impl change, and (b) the existing duplicate-key test must be repaired
(Step D / no later than Step B).
