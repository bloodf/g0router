# g0router

Single-binary Go LLM gateway/proxy with 43+ providers, OAuth flows, RTK compression, MCP gateway, embedded React dashboard; CLI + Web UI control plane.

## Activation
agentic-engineering: opt-in
<!--
  agentic-engineering governs how work is done in this project.
  Run /agentic-status to see the resolved mode, profile, and whether it is active here.

  This project is opted in (global mode is opt-in). To turn it off for this
  project only, remove the opt-in line above, or uncomment the marker line below.
-->
<!-- agentic-engineering: opt-out -->

<!--
  Review strictness for this project (relaxed | default | strict).
  Change the value below or remove the line to fall back to the global setting.
-->
agentic-engineering-profile: strict

## Decisions
- SQLite WAL store with additive-only `ensureColumn` migrations.
- Layered DDD architecture (transport→domain→repository) enforced by arch test (phase 12B).
- All API responses use snake_case JSON with a `{data, error}` envelope.
- Secrets encrypted at rest via reversible `*_enc` columns (pattern: `internal/store/oauthsessions.go`).
- Usage data lives in the `request_log` table.

## Repo structure
- `ui/` — React dashboard track; has its own AGENTS.md with deeper context.

## Tools
- GitHub operations: use `gh` CLI — do not use GitHub MCP.
- Database operations: use `sqlite3` against the data-dir DB.
- Dependency audit: `govulncheck ./...` for Go; `npm audit --json` in `ui/`.

## Docs
- `docs/planning/` — Briefs, Plans, ADRs.
- `docs/research/` — research notes and prior art.
- `docs/technical/` — technical references and design docs.
- `docs/overview/` — product vision and requirements (operator-owned).
- `docs/WORKFLOW.md` — task log / current stage status.
- `docs/phases/` — phase specs for stage 12B-19.

## Conventions
- TDD always: write test first, see it fail, write minimum code to pass.
- Every package gets `_test.go` files before implementation.
- `go test ./...` and `go vet ./...` must pass green at every commit.
- Match existing patterns — read 3 existing files before writing a new one.
- No mocks — use interfaces and fakes; test real behavior.
- No `init()` functions — explicit initialization via constructors.
- Errors are values — return `error`, never panic; wrap with `fmt.Errorf("context: %w", err)`.
- No global state — pass dependencies via struct fields or function params.
- Naming: `camelCase` locals, `PascalCase` exports; package names lowercase singular nouns.
- Ubiquitous Language: see `glossary.md` for domain terms.
- No PR workflow — commit and push directly to `main`; quality gates run locally before every push.
- Commit message format: `phase-N/task-M: <description>`.
- Update `docs/WORKFLOW.md` after completing any task.
- `.agentic/tasks.jsonl` is the task coordination surface for multi-unit orchestration plans.

## PR Workflow
# NOTE: this project pushes directly to main — no PRs. Block kept for future use.
<!--
  PR_TARGET_BRANCH: main
  PR_DRAFT: true
  PR_REVIEWERS:
  PR_LABELS:
-->

## Session start
- On the first interaction of a new session, silently check that `/init-project` scaffolding exists. Check each item only if its precondition holds:
 - Root `AGENTS.md` has required sections (`## Tools`, `## Docs`, `## Conventions`, `## Session start`) - always check.
 - `.claude/settings.json` - always check.
 - `docs/{planning,research,technical,overview}/` - always check.
 - `docs/overview/vision.md` and `docs/overview/requirements.md` - always check.
 - Seeded `MEMORY.md` at `<cwd>/.agentic/memory/MEMORY.md` - always check.
 - `.agentic/qa.md` (or legacy `.claude/qa.md`) - only if this project has a web UI.
 - `.agentic/deploy.md` (or legacy `.claude/deploy.md`) - only if release signals apply to this project.
 - `.agentic/learnings.md` - always check.
- **Parity harness (Stage 1):** if `.planning/harness/HANDOFF.md` exists, read it before parity work; VPS orchestrator is Claude Code, planner is Fable 5.
- Filesystem existence only - no LLM reasoning pass. Per-track scaffolds are out of scope for this check - do not flag them.
- Do NOT include `.agentic/preferences.json` or `.claude/settings.local.json` in the "missing" list - both are gitignored per-developer files.
- If `.agentic/preferences.json` exists and contains `"skipScaffoldingCheck": true`, skip the check entirely.
- If anything is missing, prompt the user ONCE per session on one line: `Scaffolding check: missing [list]. Re-run /init-project to fix? [y/N/never]`. `never` performs a read-modify-write on `.agentic/preferences.json` setting `skipScaffoldingCheck: true` without clobbering other keys.
