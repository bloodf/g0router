# g0router Wave 7.J Evaluation Prompt

Evaluate completed Wave `7.J` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

## Review Inputs

- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/DEPLOYMENT.md`
- `README.md`
- Relevant phase docs:
  - `docs/phases/phase-10-dashboard-ui.md`
  - `docs/phases/phase-11-packaging-deployment-polish.md`
  - `docs/phases/phase-12-advanced-mcp-gateway.md`
- Wave commits:
  - `13db10d phase-7/task-j1: add dashboard control-plane auth`
  - `d8de874 phase-7/task-j2: harden installer bootstrap`
  - `677b6ff phase-7/task-j3: harden docker bootstrap`
  - `407933c phase-7/task-j4: wire live mcp instance lifecycle`
- Merge commits:
  - `3c20d48 merge wave 7j dashboard auth`
  - `d7782a0 merge wave 7j installer`
  - `35ac82c merge wave 7j docker bootstrap`
  - `4728044 merge wave 7j mcp runtime`

## Checks

- Dashboard control-plane auth:
  - Saved control-plane key is sent to `/api/*` as `Authorization: Bearer ...`.
  - Explicit caller auth headers are not overwritten.
  - UI offers save/clear controls and retries active page requests after auth changes.
  - Full provider credentials are still never displayed.
- Installer bootstrap:
  - `g0router install` is self-contained and does not require checkout-relative deploy templates.
  - Generated systemd defaults contain nonblank `JWT_SECRET` and `API_KEY_SECRET` before enabling `--now`.
  - Installing over the same executable path does not truncate the running binary.
- Docker bootstrap:
  - `/data` is writable for the distroless nonroot image with a fresh named volume.
  - Compose refuses to start without stable `JWT_SECRET` and `API_KEY_SECRET`.
  - Docker docs explain first API key bootstrap and stable secret requirements.
  - Compose healthcheck syntax is valid.
- MCP runtime:
  - `POST /api/mcp/instances` registers live runtime clients, discovers tools, persists manifests, and rolls back store rows on registration failure.
  - `DELETE /api/mcp/instances/:id` closes runtime clients and removes live tools before deleting store rows.
  - MCP OAuth callback/complete uses request context, persists tokens, reapplies credentials to the live runtime, updates manifest cache, and records health.
  - `ToolManager.UnregisterClient` removes only the selected client's tools and preserves siblings.
  - Runtime instance config storage is concurrency-safe.
- Workflow:
  - `docs/WORKFLOW.md` accurately marks Wave 7.J tasks complete and project status as evaluation pending, not release-ready.
  - No unowned or unrelated files were committed.
  - Existing protected local dirt such as `.DS_Store`, `.pi/`, and untracked `AGENTS.md` was not cleaned up or committed.

## Required Gates

Run:

```bash
go test ./... -count=1
go vet ./...
go build ./cmd/g0router
npm --prefix ui test -- --run
npm --prefix ui run build
make build
JWT_SECRET=test-jwt API_KEY_SECRET=test-api docker compose config
docker build -t g0router:wave-7j-eval .
```

If Docker is unavailable, report the exact daemon/tooling failure instead of marking the Docker gate passed.

## Return Format

## Verdict

PASS or FAIL

## Blocking Findings

Issues that must be fixed before advancing beyond Wave 7.J.

## Non-Blocking Findings

Risks or cleanup notes.

## Gate Results

Command results with exact failures if any.

## Workflow Status Review

Whether `docs/WORKFLOW.md` accurately reflects Wave 7.J completion and evaluation-pending state.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
