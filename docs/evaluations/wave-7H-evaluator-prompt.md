# g0router Wave 7.H Evaluation Prompt

Evaluate completed wave `7.H` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

## Review Inputs

- `AGENTS.md`
- `CLAUDE.md`
- `README.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/SCHEMA.md`
- Relevant code:
  - `ui/src/api.ts`
  - `ui/src/api.test.ts`
  - `ui/src/App.tsx`
  - `ui/src/App.test.tsx`
  - `ui/src/components/Primitives.tsx`
  - `ui/src/components/Primitives.test.tsx`
  - `ui/src/pages/DashboardPage.tsx`
  - `ui/src/pages/EndpointPage.tsx`
  - `ui/src/pages/ProvidersPage.tsx`
  - `ui/src/pages/UsagePage.tsx`
  - `ui/src/pages/QuotaPage.tsx`
  - `ui/src/pages/CombosPage.tsx`
  - `ui/src/pages/McpPage.tsx`
  - `ui/src/pages/SettingsPage.tsx`
  - corresponding page `*.test.tsx` files
  - `ui/dist/**`
- Commit refs:
  - `d8635ee phase-7/task-h0: plan dashboard wave`
  - `f2dcfca phase-7/task-h1: wire dashboard api contracts`
  - `973e9a9 phase-7/task-h2: wire providers and endpoint dashboard pages`
  - `9c375b1 phase-7/task-h3: wire usage quota logs dashboard pages`
  - `4704dae phase-7/task-h4: wire combos and settings dashboard pages`
  - `83cce34 phase-7/task-h5: wire mcp dashboard page`
  - range `d8635ee..HEAD`

## Check

- Dashboard pages no longer use static fixtures for providers, connections, API keys, usage, logs, quotas, combos, settings, or MCP.
- `ui/src/api.ts` matches the real management API routes, especially `/api/usage/quota/:provider` rather than `/api/quota`.
- API helpers use same-origin credentials, parse empty 204 responses, expose useful API errors, and distinguish 401/403 auth-expired states.
- Every dashboard page has loading, empty, error, and auth-expired states where applicable.
- Provider credentials, API keys, MCP env/header values, access tokens, and refresh tokens are not displayed from stored API data.
- Newly created gateway API key raw values are shown only as transient create output, not as stored key data.
- Combos and settings forms call the real management APIs with the documented request bodies.
- MCP dashboard reads clients, instances, tools, and per-instance accounts, and instance/OAuth actions call real MCP API routes.
- The app shell mounts only the selected page; the dashboard view must not mount every management page and spam all API surfaces.
- Wide dashboard tables, especially MCP instances/accounts/tools, must be horizontally scrollable on mobile rather than clipped or causing page-wide overflow.
- UI tests cover actual API contracts rather than static copy only.
- `ui/dist/**` matches the final built UI after all page slices were merged.
- Existing `.DS_Store`, `.pi/`, and untracked `AGENTS.md` state was not cleaned up or committed.
- `docs/WORKFLOW.md` accurately marks Wave 7.H complete and does not advance to Wave 7.I unless all gates pass.

## Known Deferred Work

- Deeper request logging, pricing expansion, and quota enforcement remain Wave 7.I work.
- Further dashboard polish can be non-blocking if the current UI is functional, API-backed, and does not leak credentials.

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

Also smoke-test the UI in a browser against a local dev or served build and verify:

- dashboard first screen renders without console errors
- navigation to Endpoint, Providers, Usage, Quota, Combos, MCP, and Settings works
- no obvious text overlap or mobile overflow on a narrow viewport

## Return

```markdown
## Verdict

PASS or FAIL

## Blocking Findings

Issues that must be fixed before Wave 7.I implementation advances.

## Non-Blocking Findings

Risks or cleanup notes.

## Gate Results

Command results with exact failures if any.

## Workflow Status Review

Whether `docs/WORKFLOW.md` accurately reflects Wave 7.H and advances to Wave 7.I.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
```
