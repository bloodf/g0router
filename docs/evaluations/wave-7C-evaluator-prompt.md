# g0router Wave 7.C Evaluation Prompt

Evaluate completed wave `7.C` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

## Review Inputs

- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- Relevant phase/remediation docs in `docs/`
- Commit refs:
  - `39c481a phase-7/task-c1: persist oauth sessions`
  - `d996bcd phase-7/task-c2: persist cli oauth login`
  - `e5d8367 phase-7/task-c3: refresh oauth credentials before dispatch`
  - `86d2c51 phase-7/task-c4: normalize provider identifiers`
  - range `eb89f36..HEAD`

## Check

- OAuth authorize stores callback session state server-side, including provider, verifier, redirect URI, expiry, and account label.
- OAuth callback and exchange consume stored state and reconstruct PKCE verifier without accepting `state.verifier` from the client.
- HTTP OAuth callback, exchange, and poll persist `store.Connection` rows and return redacted connection metadata, not raw token JSON.
- CLI `login --device` persists completed OAuth connections in the configured `--data-dir`, does not persist pending polls, and does not leak token material in output.
- API-key style auth flows persist as `api_key` connections, not fake OAuth access tokens.
- OAuth refresh runs before direct dispatch, streaming dispatch, combo dispatch, and provider model listing when OAuth credentials are near expiry.
- Refresh uses the stored OAuth origin metadata, so Codex/OpenAI rows refresh through the Codex flow instead of a nonexistent OpenAI OAuth flow.
- Refreshed access token, refresh token, and expiry are persisted without rewriting unrelated connection metadata.
- Provider IDs are canonicalized consistently:
  - runtime/store `codex` -> `openai`
  - OAuth flow `openai` -> `codex`
  - `github` -> `github-copilot`
- Legacy `codex` and `github` rows remain usable for routing/logout through alias lookup rather than being stranded.
- `/api/connections` lists stored auth-only provider rows instead of filtering them out through the runtime provider list.
- Existing `.DS_Store`, `.pi/`, and untracked `AGENTS.md` state was not cleaned up or committed.

## Known Deferred Work

- OAuth device login currently polls once; a user-friendly long-poll loop remains follow-up work.
- Provider ID normalization does not claim unsupported providers are implemented; provider parity/status belongs in Wave 7.D.
- Model dispatch is still prefix-based and must be replaced in Wave 7.E.
- Real MCP OAuth and JSON-RPC client behavior remain Wave 7.G work.

## Gates

Run:

```bash
go test ./... -count=1
go vet ./...
go build ./cmd/g0router
npm --prefix ui test -- --run
npm --prefix ui run build
make build
```

## Return

```markdown
## Verdict

PASS or FAIL

## Blocking Findings

Issues that must be fixed before Wave 7.D.

## Non-Blocking Findings

Risks or cleanup notes.

## Gate Results

Command results with exact failures if any.

## Workflow Status Review

Whether `docs/WORKFLOW.md` accurately reflects Wave 7.C and advances to Wave 7.D.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
```
