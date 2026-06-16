# bf-refactor-1 — consolidate the two *_enc backfills (ponytail shrink)

**Origin:** ponytail over-engineering review of bf-gov-5 + bf-mcp-3 (`net: -35 lines`).
**Type:** pure refactor — behavior-preserving, NO functional change. The existing
backfill tests are the regression guard.

## Gap
`backfillVirtualKeyEncryption` (`internal/store/virtualkeys.go`) and
`backfillMCPInstanceEnvEncryption` (`internal/store/mcpinstances.go`) are the same
~30-line collect-then-update skeleton (query `WHERE <encCol> = ''` → scan rows into a
`pending` slice → `rows.Close()` → loop `Encrypt` + `UPDATE`). Only the table, source
column, and per-row transform differ. The single-conn collect-then-update correctness
concern is duplicated in two places.

## Design (one shared helper + two thin wrappers)
Add to `internal/store/crypto.go` (or wherever fits the existing store style) ONE helper:

```go
// backfillEnc migrates legacy rows of `table` whose `encCol` is empty (''): it reads
// the plaintext `srcCol`, applies transform to produce the new src value and the
// reversible ciphertext, and writes both back in one UPDATE. Idempotent via the
// encCol='' guard. Rows are collected before updating to avoid iterating a result set
// while writing on the single-conn DB. table/srcCol/encCol are compile-time constants
// from the call sites (not user input) — safe to interpolate (SQL has no bind param
// for identifiers).
func (s *Store) backfillEnc(table, srcCol, encCol string,
    transform func(raw string) (newSrc, enc string, err error)) error
```

Body: `SELECT id, <srcCol> FROM <table> WHERE <encCol> = ''` → collect `{id, raw}` →
`rows.Close()` → for each: `newSrc, enc, err := transform(raw)`; `UPDATE <table> SET
<encCol> = ?, <srcCol> = ? WHERE id = ?` bound `(enc, newSrc, id)`. Wrap errors with
context exactly as the originals do.

**Keep both named wrappers** (the idempotency tests call `st.backfillVirtualKeyEncryption()`
/ `st.backfillMCPInstanceEnvEncryption()` by name, and `Open()` calls them) — reduce each
to a one-line delegation:

```go
func (s *Store) backfillVirtualKeyEncryption() error {
    return s.backfillEnc("virtual_keys", "key", "key_enc", func(raw string) (string, string, error) {
        enc, err := s.cipher.Encrypt(raw)
        return sha256hex(raw), enc, err   // key column repurposed to the lookup hash
    })
}

func (s *Store) backfillMCPInstanceEnvEncryption() error {
    return s.backfillEnc("mcp_instances", "env_json", "env_json_enc", func(raw string) (string, string, error) {
        enc, err := s.cipher.Encrypt(raw)
        return "{}", enc, err              // legacy env_json drained to '{}'
    })
}
```

This preserves: the `Open()` call sites, the test-callable method names, the UPDATE
column order (`encCol` first, then `srcCol`), and the exact migration behavior.

## Constraints
- Behavior-preserving. NO test edits expected (existing tests are the guard). If a test
  must change, STOP and escalate — that would mean behavior drifted.
- `go test ./...` + `go vet ./...` GREEN. Commit prefix `phase-1/bf-refactor-1:`;
  footer `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`.
- Explicit `git add <named files>` only; NEVER `-A`; NEVER touch `ui/dist/index.html`.
- Worktree-isolated; do NOT push (orchestrator merges after verifying).

## Steps
1. Add `backfillEnc` helper; rewrite the two wrappers as one-line delegations.
   Stage: `internal/store/crypto.go internal/store/virtualkeys.go internal/store/mcpinstances.go`
   (exact files depend on where the helper lands — name them in the commit).
   Commit: `phase-1/bf-refactor-1: consolidate *_enc backfills into one backfillEnc helper`
2. Close-out: this phase needs no matrix change (no parity row). Update `docs/WORKFLOW.md`
   bf-followups table: bf-refactor-1 PENDING → DONE.
   Commit: `phase-1/bf-refactor-1: close — backfill consolidation (WORKFLOW updated)`

## Regression guard (must stay GREEN, unchanged)
`TestBackfillNoLockout`, `TestBackfillIdempotent` (VK) and
`TestMCPInstanceBackfillNoLockout`, `TestMCPInstanceBackfillIdempotent` (MCP) — they
exercise both wrappers end-to-end; if they pass unchanged, the refactor is behavior-preserving.

## No-leftovers
The helper has exactly two live callers (the wrappers), each called from `Open()`. No new
flexibility beyond what the two existing call sites need.
