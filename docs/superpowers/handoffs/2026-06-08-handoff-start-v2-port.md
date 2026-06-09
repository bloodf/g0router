# Handoff: Start g0router v2.0 — Phase 1: Scaffolding

**For:** Any AI coding agent continuing this project  
**Context:** Planning complete. GSD milestone v2.0 defined with 19 phases.  
**Your immediate mission:** Execute **Phase 1: Scaffolding** end-to-end. Do not jump ahead to Phase 2.

---

## Project Context

g0router is being rebuilt from a clean slate. The previous `api/`, `internal/`, and `ui/src/` code is obsolete. The new direction is:

- **Backend:** Go (fasthttp) single binary.
- **Architecture:** BiFrost-style provider interface + converter pattern for the OpenAI-compatible layer.
- **Features:** 9router-style management features (RTK, Caveman, combos, multi-account OAuth, quota tracking, translator, MCP, proxy pools, nodes, cloud sync).
- **Frontend:** Vite + React 19 + Tailwind 4 + shadcn/ui, embedded in the Go binary.
- **E2E:** Playwright with a 1:1 mocked API layer.
- **Quality gates:** `go test ./...`, `go vet ./...`, `npm run build`, `npx playwright test` must pass on every commit.

Source documents you must read before writing code:
1. `docs/superpowers/specs/2026-06-08-9router-bifrost-port-design.md` — full design spec.
2. `.planning/ROADMAP.md` — 19-phase roadmap and dependency graph.
3. `.planning/phases/phase-01-scaffolding/PLAN.md` — this phase's plan.
4. `.planning/REQUIREMENTS.md` — requirements with REQ-IDs.

---

## Current State

- GSD milestone v2.0 initialized.
- All 19 phase plans written.
- No implementation code exists yet for the new architecture.
- The old `api/`, `internal/`, and `ui/src/` directories still exist and must be removed.

---

## Phase 1: Scaffolding — What You Must Do

### 1. Delete old code

Remove these directories entirely:
- `api/`
- `internal/`
- `ui/src/`

Keep these (do not delete):
- `cmd/g0router/`
- `embed.go`
- `go.mod`
- `Dockerfile`
- `deploy/`
- `ui/package.json` and all UI build toolchain files
- `ui/public/providers/`
- `ui/index.html`
- Project metadata: `AGENTS.md`, `CLAUDE.md`, `glossary.md`, `.agentic/`, `.claude/`, `.omc/`, `.pi/`
- Docs: `docs/` (including the new planning artifacts)
- Tests: `tests/` and E2E infrastructure

### 2. Create new Go package structure

Create empty packages under `internal/` with a `doc.go` or minimal file in each:

```
internal/
  schemas/        # Shared Go types
  server/         # fasthttp server, route registration, middleware
  api/            # OpenAI-compatible /v1/* handlers
  admin/          # Management /api/* handlers
  providers/      # Provider implementations
    openai/       # Reference provider
    anthropic/
    gemini/
    groq/
    mistral/
    cohere/
    fireworks/
    together/
    deepseek/
    minimax/
    ollama/
    bedrock/
    vertex/
    utils/        # Shared fasthttp client, SSE scanner pools
  inference/      # Routing, fallbacks, key selection
  catalog/        # Model catalog + pricing
  governance/     # Virtual keys, provider configs, quotas
  auth/           # Sessions, OAuth, API key auth
  store/          # SQLite persistence
  logging/        # Request log + audit
  mcp/            # MCP gateway
  config/         # Runtime config loading
  platform/       # 9router-specific features (RTK, caveman, combos, translator, sync, nodes, proxypool)
```

### 3. Create placeholder tests

Every package must have a `_test.go` file with at least one passing placeholder test. This satisfies the TDD convention and ensures `go test ./...` passes from day one.

Example pattern:

```go
package schemas

import "testing"

func TestPackageCompiles(t *testing.T) {
    // Phase 1 placeholder; real tests come in Phase 2+.
}
```

### 4. Update `cmd/g0router/main.go`

Replace the old main with a minimal skeleton that:
- Parses environment/config.
- Initializes the fasthttp server on the configured port.
- Serves a health check endpoint (`GET /api/health`) returning `{"status":"ok"}`.
- Serves the embedded UI catch-all (the UI build will be minimal at this phase).

Do not implement real providers, catalog, or admin APIs yet.

### 5. Update `embed.go`

Ensure `embed.go` points to the correct UI build output directory (usually `ui/dist/`).

### 6. Create minimal UI placeholder

Create the smallest possible UI so `npm run build` passes:
- `ui/src/main.tsx`
- `ui/src/App.tsx`
- `ui/src/index.css`

The placeholder can render "g0router v2.0 — coming soon" and a link to `/api/health`. Do not add TanStack Router, complex state, or dashboard pages yet.

### 7. Clean `go.mod`

Run `go mod tidy`. Remove old dependencies that are no longer imported. If `go mod tidy` removes something you need later, pin it with an explicit `require` and a comment.

### 8. Verify gates

Before declaring Phase 1 complete, these must pass:

```bash
go test ./...
go vet ./...
cd ui && npm run build
cd .. && go build ./cmd/g0router
```

Also run:

```bash
npx playwright test --list
```

This should not crash due to missing files. (No new E2E tests required in Phase 1.)

---

## Conventions You Must Follow

- **TDD:** every package gets `_test.go` before real implementation. Phase 1 uses placeholders; later phases use real tests.
- **No mocks:** use interfaces and fakes for tests.
- **No `init()` functions:** explicit constructors only.
- **Errors are values:** return `error`, never panic; wrap with `fmt.Errorf("context: %w", err)`.
- **No global state:** pass dependencies via struct fields or function params.
- **Naming:** `camelCase` locals, `PascalCase` exports; package names lowercase singular nouns.
- **Snake_case JSON:** all API responses use snake_case.
- **Commit format:** `phase-01/task-N: description` for Phase 1 work.

---

## What NOT To Do

- Do not implement any provider converter or API handler beyond the health check.
- Do not create dashboard routes or pages beyond the placeholder.
- Do not add new dependencies unless absolutely necessary for the skeleton.
- Do not skip placeholder tests in any package.
- Do not delete retained files listed in section 1.

---

## Success Criteria for Phase 1

- [ ] Old `api/`, `internal/`, `ui/src/` directories are gone.
- [ ] New `internal/` package structure exists with 14 top-level packages + provider subpackages.
- [ ] Every package has a `_test.go` that passes.
- [ ] `cmd/g0router/main.go` is a minimal skeleton serving `/api/health`.
- [ ] `embed.go` points to the correct UI build path.
- [ ] UI placeholder builds successfully.
- [ ] `go test ./...` passes.
- [ ] `go vet ./...` passes.
- [ ] `npm run build` passes in `ui/`.
- [ ] `go build ./cmd/g0router` produces a binary.
- [ ] Changes are committed with `phase-01/` prefix messages.

---

## After Phase 1

Do not start Phase 2 without explicit instruction. When Phase 1 is complete:
1. Update `.planning/STATE.md` to reflect Phase 1 complete and Phase 2 ready.
2. Update `docs/WORKFLOW.md` with Phase 1 completion note.
3. Report back with a summary of files changed and verification output.

---

## Quick Reference

| Artifact | Path |
|----------|------|
| Design spec | `docs/superpowers/specs/2026-06-08-9router-bifrost-port-design.md` |
| Roadmap | `.planning/ROADMAP.md` |
| Phase 1 plan | `.planning/phases/phase-01-scaffolding/PLAN.md` |
| Requirements | `.planning/REQUIREMENTS.md` |
| Project context | `.planning/PROJECT.md` |

---

*Handoff created: 2026-06-08*  
*Next expected action: execute Phase 1: Scaffolding.*
