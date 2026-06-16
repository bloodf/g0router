# bf-mcp-3 — MCP instance env secrets encrypted at rest (PAR-BF-MCP-080)

**Parity row:** PAR-BF-MCP-080 (bifrost-mcp matrix) — "ConnectionString stored as
*EnvVar (encrypted at rest)."

**Gap:** MCP instance connection fields in `mcp_instances` are stored PLAINTEXT, but
`env_json` routinely carries SECRETS (API tokens/keys passed as env vars to stdio MCP
server processes). This violates the AGENTS.md decision: *"Secrets encrypted at rest via
reversible `*_enc` columns."* Genuine, in-scope, additive gap. Same hardening principle
just shipped for VK values in **bf-gov-5** (commit `1240e57`) — this plan mirrors that
plan's `*_enc` + idempotent in-`Open` backfill structure.

**Scope:** `env_json` only (the clearly-secret-bearing field). `args_json` weighed and
**EXCLUDED** (rationale §Open-questions Q1). `url`/`command` are NOT secrets — untouched.

**Commit prefix:** `phase-1/bf-mcp-3:` · **Footer:** `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`

---

## Verified grounding (read against code — all confirmed)

| Fact | Location | Verified |
|---|---|---|
| `mcp_instances` schema: `id, client_id, name, transport, url, command, args_json, env_json, status, created_at, updated_at` — all plaintext | `internal/store/migrate.go:232-244` | ✅ |
| `CreateMCPInstance` marshals `in.Env`→`envJSON` and INSERTs it directly | `internal/store/mcpinstances.go:124-156` | ✅ |
| `UpdateMCPInstance` marshals `in.Env`→`envJSON`, UPDATEs `env_json = ?` | `internal/store/mcpinstances.go:189-209` | ✅ |
| `GetMCPInstance`/`ListMCPInstances` SELECT `env_json` | `internal/store/mcpinstances.go:159-187` | ✅ |
| `scanMCPInstance` scans `env_json` → `unmarshalJSONStringMap` → `in.Env` | `internal/store/mcpinstances.go:248-266` | ✅ |
| Reversible cipher `Cipher.Encrypt/Decrypt` (AES-256-GCM, base64) available as `s.cipher` | `internal/store/crypto.go:43-68` | ✅ |
| Precedent to mirror: `mcpoauth.go` encrypt-on-write (`:45-52,140-143`), decrypt-on-scan (`:197-202,180`) | `internal/store/mcpoauth.go` | ✅ |
| `ensureColumn(db, table, column, decl)` additive ALTER; additive-column loop ends at `migrate.go:459` (`{"virtual_keys","key_enc",...}`), guarded `ensureColumn` at `:461` | `internal/store/migrate.go:455-465,535-537` | ✅ |
| `Open(path, secret)` builds cipher (`:27`), runs `migrate(db)` (no cipher, `:49`), constructs `&Store{db,cipher,...}` (`:54`), then runs `s.backfillVirtualKeyEncryption()` AFTER construction (`:59`) — the in-`Open` backfill seam already exists from bf-gov-5 | `internal/store/store.go:26-65` | ✅ backfill must run after Store construction (needs cipher) |
| Test harness `newMCPTestStore(t)` injects a real 32-byte secret via `LoadOrCreateSecret(t.TempDir())` + `Open(...)` — gives tests a real cipher for free | `internal/store/mcpinstances_test.go:9-22` | ✅ |

### Reader audit (CRITICAL — no consumer bypasses the store scan)

`env_json` (the column name) appears in the **entire `--type go` tree** ONLY in
`internal/store/mcpinstances.go` and `internal/store/migrate.go`. No other package
references the plaintext column. Every external consumer reads the in-memory `.Env`
field, which is populated exclusively by `scanMCPInstance`:

| Consumer | Reads | Path |
|---|---|---|
| `internal/admin/mcp.go:178-179,221` `toInstanceDTO` | `in.Env`, `in.Args` | session-gated GET/list response — via store scan ✅ |
| `internal/admin/mcp.go:295,390,411,473,518,615,1072` `GetMCPInstance`/`ListMCPInstances` | various; `.Env` only via DTO | all via store scan ✅ |
| `internal/mcp/launcher.go:53,71-74` `StartStdio(name,command,args,env)` → `runner.Start(ProcessSpec{Env:env})` | `env` param | **`internal/admin/mcp.go:342` passes `req.env()`** (HTTP create-request body — plaintext input source, NOT the stored column) ✅ |
| `internal/mcp/process.go:35,149-151` `cmd.Env = mergeEnv(spec.Env)` | `spec.Env` | env flows from launcher param, originally `req.env()` ✅ |

**Notable finding (record in close-out):** the *stored* env is **never re-fed to the
launcher** today — there is exactly one `StartStdio` call site (`admin/mcp.go:342`) and it
uses the inbound request body's env (`req.env()`), not a decrypted store row. So no live
launch path reads `.Env` from the store. The at-rest gap is nonetheless real and in scope:
the row persists secrets in plaintext on disk regardless of who reads them later, and the
DTO at `:178-179` *does* surface the stored `.Env` over the session-gated API. Encrypting
`env_json` is justified by the at-rest principle independent of the read path. No reader
bypasses the scan; **no escalation triggered.**

### Confirmed-design notes vs. brief

- **No `SumCostByTeam`-style blocker here.** Unlike bf-gov-5 (where `virtual_keys.key` was
  repurposed to a hash, breaking a raw-SQL correlation), this plan keeps `env_json` as a
  legacy column and simply stops writing/reading secrets through it. No cross-table SQL
  correlates on `env_json`. Blast radius is contained to `mcpinstances.go`.
- **`sha256hex` is NOT needed.** `env_json` is never a lookup key (no `WHERE env_json = ?`).
  Only AES `*_enc` is required — strictly simpler than the VK case (which needed both a hash
  for lookup and AES for reversibility). `Cipher.Encrypt/Decrypt` is the only primitive used.
- **Backfill seam already wired.** `store.go:59` already calls one in-`Open` backfill
  (`backfillVirtualKeyEncryption`); this plan adds a second sibling call. No `Open` signature
  change, no constructor change.

---

## Design decision: drain plaintext after backfill (mirror bf-gov-5)

After backfill, the plaintext source column must hold **no plaintext secret**. Two options
were weighed (§Open-questions Q2). **Chosen: clear `env_json` to `'{}'` post-migration and
stop writing it going forward.**

- **Write path (Create/Update):** marshal `in.Env` → encrypt → store in `env_json_enc`.
  Write `'{}'` (empty object, the column's existing default) into the legacy `env_json`
  column so it never holds a plaintext secret again.
- **Read path (`scanMCPInstance`):** SELECT `env_json_enc`; decrypt into `in.Env`. The
  legacy `env_json` column is no longer read at all (dropped from the SELECT lists).
- **Backfill:** for rows where `env_json_enc = ''`, read legacy plaintext `env_json`,
  `Encrypt` it into `env_json_enc`, and overwrite `env_json` with `'{}'`. Idempotent via the
  `env_json_enc = ''` guard.

This guarantees: (a) at-rest — no plaintext secret in any column after Create/Update or
backfill; (b) no-lockout — `.Env` still resolves to the original map so the MCP server
launches with the right secrets; (c) the legacy column is inert (kept only because
migrations are additive-only — we never DROP).

### Exact migrate change

In `migrate.go`, in the additive-column loop (currently ending at `:459` with
`{"virtual_keys", "key_enc", "TEXT NOT NULL DEFAULT ''"}`), append one entry **before**
the `ensureColumn` guard at `:461`:

```go
{"mcp_instances", "env_json_enc", "TEXT NOT NULL DEFAULT ''"},
```

Additive-only. No DROP, no constraint change. `migrate(db)` keeps its db-only signature.

### Exact store edits (`internal/store/mcpinstances.go`)

1. **`MCPInstance` struct** — no field change; `Env map[string]string` stays plaintext in
   memory (always decrypted on the launch/DTO path). Update the doc comment to note env is
   encrypted at rest in `env_json_enc` (mirror `mcpoauth.go:10-14` wording).
2. **`CreateMCPInstance`** — after marshaling `envJSON`, compute
   `envEnc, err := s.cipher.Encrypt(envJSON)`; INSERT column list becomes
   `(..., args_json, env_json, env_json_enc, status, ...)` binding `'{}'` for `env_json` and
   `envEnc` for `env_json_enc`. (Keep `args_json` as-is.)
3. **`UpdateMCPInstance`** — same: encrypt `envJSON`, `SET ... env_json = '{}', env_json_enc = ?, ...`
   binding `envEnc`. Ensures Update never leaves plaintext behind.
4. **`GetMCPInstance` / `ListMCPInstances`** — change both SELECT lists to select
   `env_json_enc` (drop `env_json` from the projection — it is no longer read).
5. **`scanMCPInstance`** — scan `env_json_enc` into a local `envEnc string`; decrypt:
   `envJSON, err := s.cipher.Decrypt(envEnc)` (fail-closed on error — see Q3), then
   `unmarshalJSONStringMap(envJSON)` → `in.Env`. (`args_json`→`in.Args` unchanged.)
   - **Empty-string guard:** a freshly-migrated-then-not-yet-resaved row, or a fresh row,
     could in principle have `env_json_enc = ''` if reached before backfill. Treat `envEnc == ""`
     as empty env (`in.Env = map[string]string{}` / nil) rather than calling `Decrypt("")`
     (which would error on `decode/too-short`). Backfill guarantees migrated rows are
     non-empty; this guard only protects the degenerate empty case. (Mirror: `mcpoauth.go`
     never has an empty `*_enc` because it always encrypts on write; here Create/Update also
     always encrypt, so `''` only ever means "no env" — map it to empty, not an error.)

### Backfill (`internal/store/mcpinstances.go` + `store.go` sequencing)

Add `func (s *Store) backfillMCPInstanceEnvEncryption() error`:
- `SELECT id, env_json FROM mcp_instances WHERE env_json_enc = ''` (the idempotency guard).
- For each row: `envEnc, err := s.cipher.Encrypt(env_json)`; then
  `UPDATE mcp_instances SET env_json_enc = ?, env_json = '{}' WHERE id = ?`.
- Idempotent: re-running skips already-migrated rows (`env_json_enc != ''`). No-op on fresh DBs.

In `store.go`, after the existing `backfillVirtualKeyEncryption()` block (`:59-62`) and
before `return s, nil` (`:64`), add a sibling:

```go
if err := s.backfillMCPInstanceEnvEncryption(); err != nil {
    db.Close()
    return nil, fmt.Errorf("backfill mcp instance env encryption: %w", err)
}
```

`Open(path, secret)` signature unchanged. Runs at most once per row across all future boots.

---

## TDD commit sequence (RED before GREEN)

> Hermetic only: `newMCPTestStore(t)` (in-memory SQLite via `t.TempDir()`, real 32-byte
> secret). NO network/sleep/subprocess. `go test ./...` + `go vet ./...` GREEN after every
> GREEN/close commit. Suite must never have a window where an existing test is red across a
> GREEN commit — collapse the interdependent column+create+update+scan changes into ONE
> commit (Step 2 GREEN), exactly as bf-gov-5 collapsed create+scan+lookup.
> NEVER `git add -A`; stage only named files. NEVER stage/revert `ui/dist/index.html`.

### Step 1 — `env_json_enc` column exists (migration) — RED then GREEN
- **RED** (`internal/store/mcpinstances_enc_test.go`, new): `TestMCPInstanceEnvEncColumnExists`
  opens via `newMCPTestStore(t)`, runs `PRAGMA table_info(mcp_instances)` over `st.DB()`,
  asserts an `env_json_enc` column is present. Fails today (column absent).
- **GREEN**: append `{"mcp_instances", "env_json_enc", "TEXT NOT NULL DEFAULT ''"}` to the
  `migrate.go` column loop (before the `:461` guard).
- Commit: `phase-1/bf-mcp-3: add additive env_json_enc column to mcp_instances`
- Stage: `internal/store/migrate.go internal/store/mcpinstances_enc_test.go`

### Step 2 — at-rest + round-trip (Create/Update/scan) — RED then GREEN (collapsed)
- **RED** (`mcpinstances_enc_test.go`):
  - `TestMCPInstanceEnvEncryptedAtRest`: `CreateMCPInstance` with
    `Env: {"API_TOKEN": "sk-secret-abc123"}`; read the raw row via
    `st.DB().QueryRow("SELECT env_json, env_json_enc FROM mcp_instances WHERE id = ?", id)`.
    Assert: (a) `env_json_enc` is non-empty; (b) the literal secret `"sk-secret-abc123"`
    appears in NEITHER `env_json` NOR `env_json_enc` (ciphertext is base64, not the literal);
    (c) `env_json` equals `"{}"` (drained). Fails today (no `_enc` write; secret in `env_json`).
  - `TestMCPInstanceEnvRoundTrip`: `created := Create(... Env:{"API_TOKEN":"sk-secret-abc123","REGION":"us"})`;
    assert `GetMCPInstance(id).Env` deep-equals the original map AND the matching
    `ListMCPInstances()` entry's `.Env` deep-equals it (scan decrypts on both paths).
  - `TestMCPInstanceEnvUpdateNoPlaintext`: create, then `UpdateMCPInstance` with a NEW env
    `{"API_TOKEN":"sk-rotated-xyz"}`; read raw row, assert `env_json == "{}"`, `env_json_enc`
    non-empty, new secret literal absent from both columns, and `GetMCPInstance(id).Env`
    returns the rotated map. (Proves Update doesn't clobber/leak.)
- **GREEN** (single commit — column write + scan must land together or existing
  `TestMCPInstanceCRUDAndStatus` / `TestMCPInstanceStatusLifecycle` would break):
  edit `CreateMCPInstance`, `UpdateMCPInstance`, `GetMCPInstance`+`ListMCPInstances` SELECT
  lists, and `scanMCPInstance` per §"Exact store edits".
- Commit: `phase-1/bf-mcp-3: encrypt mcp instance env at rest (env_json_enc; drain legacy env_json)`
- Stage: `internal/store/mcpinstances.go internal/store/mcpinstances_enc_test.go`

### Step 3 — NO-LOCKOUT backfill + Open sequencing — RED then GREEN
- **RED** (`mcpinstances_backfill_test.go`, new): `TestMCPInstanceBackfillNoLockout`:
  1. Create a `t.TempDir()` + `secret := LoadOrCreateSecret(dir)`; `dbPath := filepath.Join(dir,"test.db")`;
     `st, _ := Open(dbPath, secret)` (mirror `newMCPTestStore` but keep path+secret to reopen).
  2. Simulate an OLD plaintext row directly: `st.DB().Exec("INSERT INTO mcp_instances
     (id, client_id, name, transport, url, command, args_json, env_json, env_json_enc, status, created_at, updated_at)
     VALUES (?, '', 'legacy', 'stdio', '', 'npx', '[]', ?, '', 'stopped', ?, ?)",
     id, `{"API_TOKEN":"sk-legacy-secret"}`, now, now)` — plaintext in `env_json`,
     `env_json_enc = ''`. Capture `rawEnvJSON`.
  3. `st.Close()`; then `st2, _ := Open(dbPath, secret)` (triggers the in-`Open` backfill).
  4. Read raw row: assert `env_json_enc` non-empty, `env_json == "{}"`, and the secret literal
     absent from both columns.
  5. **Launch-survival:** assert `st2.GetMCPInstance(id).Env["API_TOKEN"] == "sk-legacy-secret"`
     (decrypted from the backfilled `_enc`, so the MCP server still launches with the secret).
  - Fails today (no backfill; `Open` doesn't migrate, and post-Step-2 scan reads `env_json_enc`
    which is still `''` for the legacy row → empty env → lockout).
- **GREEN**: add `backfillMCPInstanceEnvEncryption()` to `mcpinstances.go`; wire the sibling
  call into `Open()` after the VK backfill block, before `return s, nil`.
- Commit: `phase-1/bf-mcp-3: backfill legacy plaintext mcp instance env on Open (no lockout)`
- Stage: `internal/store/mcpinstances.go internal/store/store.go internal/store/mcpinstances_backfill_test.go`

### Step 4 — backfill idempotency — RED then GREEN
- **RED** (`mcpinstances_backfill_test.go`): `TestMCPInstanceBackfillIdempotent`: starting
  from Step 3's migrated row (or create a fresh encrypted instance), capture `env_json` +
  `env_json_enc`, call `st.backfillMCPInstanceEnvEncryption()` again (same-package `store`
  test → unexported method callable directly), assert both columns are byte-identical (the
  `env_json_enc != ''` guard skips re-encrypt — no new ciphertext, no double-drain).
- **GREEN**: the `WHERE env_json_enc = ''` guard from Step 3 already satisfies this; this step
  adds the assertion. If it trips, fix the guard.
- Commit: `phase-1/bf-mcp-3: assert mcp env backfill idempotency`
- Stage: `internal/store/mcpinstances_backfill_test.go`

### Step 5 — close-out: matrix flip + WORKFLOW
- Flip PAR-BF-MCP-080 → HAVE in the bifrost-mcp parity matrix; update `docs/WORKFLOW.md`.
- **Honest close note (matrix Notes cell):** "MCP instance env secrets now AES-encrypted at
  rest in `mcp_instances.env_json_enc`; legacy `env_json` drained to `'{}'` on
  write/backfill and no longer read. `url`/`command`/`args_json` are non-secret and remain
  plaintext (args excluded by scope — see plan Q1). One-time idempotent backfill on `Open`.
  Note: stored env is not currently re-fed to the launcher (single `StartStdio` call uses
  request-body env), but the row no longer persists plaintext secrets and the session-gated
  instance DTO sources env via the decrypting store scan."
- Commit: `phase-1/bf-mcp-3: close — MCP instance env encrypted at rest (MCP-080 → HAVE)`
- Stage: matrix file + `docs/WORKFLOW.md` (named explicitly; never `-A`).

---

## Critical correctness properties → proving test (all RED first)

| Property | Test (Step) |
|---|---|
| NO-LOCKOUT / launch-survival: old plaintext row migrated, `.Env` returns original secrets | `TestMCPInstanceBackfillNoLockout` (3) |
| At-rest: after Create, `env_json_enc` non-empty AND secret literal in NO plaintext column | `TestMCPInstanceEnvEncryptedAtRest` (2) |
| Round-trip: Create→Get/List returns original env map | `TestMCPInstanceEnvRoundTrip` (2) |
| Update doesn't clobber/leak plaintext | `TestMCPInstanceEnvUpdateNoPlaintext` (2) |
| Backfill idempotency (`env_json_enc=''` guard) | `TestMCPInstanceBackfillIdempotent` (4) |
| Migration additive (column exists) | `TestMCPInstanceEnvEncColumnExists` (1) |

---

## No-leftovers checklist

| New artifact | Live production consumer (in-plan) |
|---|---|
| `env_json_enc` column | written by Create/Update, read by scan, populated by backfill |
| encrypt-on-write in `CreateMCPInstance`/`UpdateMCPInstance` | every instance create/update via `admin/mcp.go` handlers |
| decrypt-on-scan in `scanMCPInstance` | `GetMCPInstance`/`ListMCPInstances` → DTO (`admin/mcp.go:178`) |
| `backfillMCPInstanceEnvEncryption()` | called from `Open()` on every boot |
| drained legacy `env_json` (write `'{}'`) | inert by design (additive-only — never DROP); no reader remains |

No new helper needed (`sha256hex` not used — env is never a lookup key). No dead code:
every column/method is exercised by a production path AND a hermetic test.

---

## Open questions / escalations

1. **`args_json` scope — RECOMMEND EXCLUDE.** CLI args *can* carry secrets
   (`--token=…`), but: (a) in g0router args are an allowlisted-command argv (`isAllowedCommand`
   gate, `launcher.go:54`) typically holding package names/flags, not credentials — the
   established secret-passing channel for MCP servers is env vars (which is exactly why
   `env_json` is the documented secret carrier); (b) encrypting non-secret args is gratuitous
   per the AGENTS.md principle (encrypt *secrets* at rest, not all columns) and adds a second
   `_enc` column + backfill for marginal benefit; (c) `url`/`command` are likewise non-secret
   and untouched. **Recommendation: env_json only this round.** If operators are observed
   passing tokens via args, file a follow-up `args_json_enc` row — same pattern, trivial to
   add. Flag if the reviewer wants args folded in now (mechanically identical; would extend
   Steps 1-4 to cover a second column).

2. **Drain plaintext `env_json` vs. keep-writing-only-`_enc` — CHOSE DRAIN.** Mirrors
   bf-gov-5's "plaintext source must hold no secret after backfill." Drain (`env_json='{}'`)
   is unambiguous and self-documenting in raw DB dumps; the column is kept (additive-only, no
   DROP) but inert. Alternative (leave `env_json` untouched on legacy rows, just add `_enc`)
   was rejected: it would leave plaintext secrets sitting in old rows forever, defeating the
   at-rest goal. Flag if a reviewer prefers a NULL/sentinel over `'{}'` (note: column is
   `NOT NULL DEFAULT '{}'`, so `'{}'` is the natural inert value; NULL would violate the
   constraint).

3. **`Decrypt` failure on malformed/legacy `env_json_enc` — fail-closed.** A row with
   `env_json_enc != ''` that fails to decrypt (wrong master secret / corruption) makes
   `scanMCPInstance` return an error → instance unreadable. **Recommendation: propagate the
   error (fail-closed)**, identical to `mcpoauth.go:197-202` and every other `*_enc` column.
   The only way to reach this is a changed master secret, which already breaks all other
   encrypted columns identically — no new failure mode. The `env_json_enc == ""` empty-guard
   (Step 2 scan note) handles the legitimate "no env" case without calling `Decrypt("")`.
   Flag if parity requires a softer fallback (treat-as-empty), which would silently mask a
   secret-rotation misconfig — not recommended.

4. **Backfill cost.** Single pass on `Open()`, one `Encrypt`+`UPDATE` per legacy row, guarded
   by `env_json_enc=''` so it runs at most once per row across all boots. Instance counts are
   operator-managed and small; first boot after upgrade does N AES-GCM seals + N updates —
   acceptable, one-time. Accepted.

**No blocker escalated.** All brief assumptions held: no reader bypasses the store scan
(`env_json` literal confined to `mcpinstances.go`/`migrate.go`); Update can avoid plaintext
(drain to `'{}'`); the in-`Open` backfill seam already exists from bf-gov-5 (no signature
change); the test harness (`newMCPTestStore`) injects a real secret. The only deviation from
the brief's framing is simplification: no `sha256hex`/lookup-hash needed (env is never a key),
and no `SumCostByTeam`-class cross-table SQL blocker exists for this table.
