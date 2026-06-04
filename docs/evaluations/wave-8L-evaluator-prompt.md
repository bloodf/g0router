# g0router Wave 8.L Evaluation

Evaluate completed wave `8.L` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PROVIDERS.md`
- `docs/SCHEMA.md`
- Commits: `7022836`, `7633953`, `009117f`

Start read-only. Do not edit files.

Check:
- Real-server API integration coverage for authenticated management mutations: API keys, aliases, combos, pricing overrides, and settings.
- Real-server MCP integration coverage for instance creation, secret redaction, OAuth start, token exchange via local fake token endpoint, account redaction, credential reapply, and deletion.
- CLI `login <provider> --key --api-key KEY --name NAME` persists active provider API-key connections, validates matrix-declared API-key support, and never prints the key.
- `auth list` includes OAuth and API-key auth-capable providers from the provider matrix without legacy split-brain aliases such as bare `github`.
- No MiniMax or other real provider tokens are committed, logged, or embedded.
- Workflow status accurately marks Stage 8 as active rather than release-complete.
- Gates pass:
  - `go test ./... -count=1`
  - `go vet ./...`
  - `go build ./cmd/g0router`
  - `npm --prefix ui test -- --run`
  - `npm --prefix ui run build`
  - `npm --prefix ui run e2e`
  - `make build`

Return:

## Verdict

PASS or FAIL

## Blocking Findings

Issues that must be fixed before advancing.

## Non-Blocking Findings

Risks or cleanup notes.

## Gate Results

Command results with exact failures if any.

## Workflow Status Review

Whether `docs/WORKFLOW.md` is accurate.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
