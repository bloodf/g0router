# g0router Wave 7.H Remediation Evaluation Prompt

Evaluate Wave `7.H` remediation in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

## Review Inputs

- `AGENTS.md`
- `CLAUDE.md`
- `README.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/evaluations/wave-7H-evaluator-prompt.md`
- Remediation diff: `9d07f19..HEAD`
- Relevant files:
  - `api/handlers/providers.go`
  - `api/handlers/providers_test.go`
  - `api/handlers/connections.go`
  - `api/handlers/connections_test.go`
  - `ui/src/api.ts`
  - `ui/src/pages/ProvidersPage.tsx`
  - `ui/src/pages/ProvidersPage.test.tsx`
  - `ui/src/pages/EndpointPage.tsx`
  - `ui/src/pages/EndpointPage.test.tsx`
  - `ui/src/pages/UsagePage.tsx`
  - `ui/src/pages/UsagePage.test.tsx`
  - `ui/src/pages/CombosPage.tsx`
  - `ui/src/pages/CombosPage.test.tsx`
  - `ui/dist/**`

## Failed Findings To Recheck

- `/api/providers` must never serialize `"auth_types": null`; unsupported providers with no auth support should return an empty array, and the dashboard must also tolerate legacy/null payloads without crashing.
- Management connection responses must redact credential-shaped keys inside `ProviderSpecificData`, including nested token, secret, key, authorization, and password fields, without mutating stored credential data.
- Endpoint, Providers, Usage, and Combos dashboard tables must be horizontally scrollable on mobile and must not use `overflow-hidden` wrappers that clip wide content.
- The UI must not display provider access tokens, refresh tokens, API keys, provider-specific credential values, or stored gateway API key raw material.
- `docs/WORKFLOW.md` must record the remediation and must not advance past Wave 7.H until this remediation evaluation passes.

## Required Gates

Run:

```bash
go test ./... -count=1
go vet ./...
go build ./cmd/g0router
npm --prefix ui test -- --run
npm --prefix ui run build
make build
```

Also smoke-test the built UI or dev server in a browser:

- Providers page renders against a real `/api/providers` response that includes unsupported/no-auth providers.
- Endpoint, Providers, Usage, Combos, and MCP pages have no document-level mobile overflow at a narrow viewport.
- Table content remains reachable via horizontal scroll containers.
- Browser console has no application errors during dashboard navigation.

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

Whether `docs/WORKFLOW.md` accurately records the Wave 7.H remediation and whether Wave 7.I can begin.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
```
