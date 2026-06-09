# Design: Remove All Placeholders, Mocks, and No-Ops from the g0router Dashboard

**Date:** 2026-06-08  
**Status:** Approved for implementation  
**Scope:** React dashboard (`ui/src/`) — wire every button, form, table action, and page to the real backend API. Remove `<ComingSoon />`, mocked data, hard-coded defaults, and no-op handlers.

---

## Background

The g0router backend exposes ~60 management endpoints under `/api/*`. The React dashboard has functional pages for providers, connections, settings, endpoint, usage, combos, models, and chat, but many other pages are still `<ComingSoon />` or contain no-op buttons/mocked data. This spec consolidates the audit findings and defines the exact implementation for all 33 issues.

---

## Goals

1. Every interactive element in the dashboard does something real (API call, navigation, or state change).
2. Every page that currently renders `<ComingSoon />` is replaced with a working CRUD/feature page.
3. All hard-coded/mock data is replaced with live data from `/api/*`.
4. Existing UI patterns (`CrudPage`, `Dialog`, `apiFetch`, TanStack Query, `toast`) are reused; no new design system.
5. All gates pass: `go test ./...`, `go vet ./...`, `npm run build` in `ui/`.

---

## Non-Goals

- New visual design or rebranding.
- New backend features; we only consume what exists.
- Mobile-responsive overhaul.
- Playwright E2E tests (the user's separate goal; this work unblocks it).

---

## High-Level Approach

Parallel subagent implementation. Six independent slices, each owned by one coder subagent. A coordinator (root agent) resolves cross-cutting API contract mismatches and runs final gates.

---

## Slice Assignments

### Slice 1 — Security + Core Shell
**Owner:** Subagent A  
**Files:** `ui/src/routes/login.tsx`, `ui/src/components/layout/Header.tsx`, `ui/src/components/layout/Sidebar.tsx`, `ui/src/routes/_app.dashboard.tsx.bak`

| Issue | Fix |
|-------|-----|
| Login form pre-fills `admin`/`123456` and shows a demo banner | Remove prefill and banner; keep empty fields. Add a first-run hint that points the user to the default credentials only when no user exists, but do not hard-code them into the form. |
| Notifications bell in Header has no `onClick` | Remove the bell button until a notifications backend exists. |
| Sidebar version badge is hard-coded `v0.9.0` | Wire to `GET /api/version` and display `current`. |
| `_app.dashboard.tsx.bak` is stale | Delete the file. |

**API endpoints used:** `GET /api/version`

---

### Slice 2 — Chat + Sessions
**Owner:** Subagent B  
**Files:** `ui/src/routes/_app.chat.tsx`, `ui/src/lib/types.ts` (mock comments)

| Issue | Fix |
|-------|-----|
| Hard-coded `provider="openai"`, `model="gpt-4o"` | On mount, fetch `/api/providers` and `/api/models`, default to first available provider/model, and let the user switch. |
| Authorization header uses `firstKey.prefix` instead of full key | Use `firstKey.full_key` if available; otherwise toast a warning and block send. |
| Recent chat sessions have clickable buttons with no `onClick` | `onClick` calls `GET /api/chat-sessions/:id` and hydrates `messages`, `provider`, `model` state. |
| New chat only clears local state | `POST /api/chat-sessions` to create a persisted session, then clear local draft state. |
| Mock comments in `types.ts` | Remove `// plaintext for mock` and `// mock-only` language. |

**API endpoints used:** `GET /api/providers`, `GET /api/models`, `GET /api/keys`, `GET /api/chat-sessions/:id`, `POST /api/chat-sessions`, `POST /v1/chat/completions`

---

### Slice 3 — Provider Detail + Connections
**Owner:** Subagent C  
**Files:** `ui/src/routes/_app.providers.$id.tsx`, `ui/src/routes/_app.connections.tsx`

| Issue | Fix |
|-------|-----|
| Provider detail connection table has no Edit/Remove | Add row actions: Edit (dialog → `PUT /api/connections/:id`), Remove (`DELETE /api/connections/:id` with `ConfirmDialog`). |
| No Suggested-models feature | Add a "Load suggested models" button or populate on mount: `GET /api/providers/:id/suggested-models`. |
| Connections page has no Edit action | Add Edit dialog same as above. |

**API endpoints used:** `PUT /api/connections/:id`, `DELETE /api/connections/:id`, `GET /api/providers/:id/suggested-models`

---

### Slice 4 — Routing Rules
**Owner:** Subagent D  
**Files:** `ui/src/routes/_app.routing-rules.tsx`

| Issue | Fix |
|-------|-----|
| Form only sends `name`, `priority`, `target_provider`, `target_model` | Add `cond_field`, `cond_operator`, `cond_value` fields. |
| Frontend type expects nested `condition` object | Align with backend flat fields: `cond_field`, `cond_operator`, `cond_value`. |

**API endpoints used:** `GET /api/routing-rules`, `POST /api/routing-rules`, `PUT /api/routing-rules/:id`, `DELETE /api/routing-rules/:id`

---

### Slice 5 — Keys + Teams + Endpoint + Logs
**Owner:** Subagent E  
**Files:** `ui/src/routes/_app.keys.tsx`, `ui/src/routes/_app.teams.tsx`, `ui/src/routes/_app.endpoint.tsx`, `ui/src/routes/_app.logs.tsx`, `ui/src/lib/lovable-error-reporting.ts`

| Issue | Fix |
|-------|-----|
| API Keys missing Regenerate action | Add row action: Regenerate → `POST /api/keys/:id/regenerate`, show one-time `full_key` in a dialog. |
| API Key form omits `scopes` and `expires_at` | Add multi-select/checkboxes for scopes and a date input for `expires_at`. |
| Teams form omits `budget_period` and `rate_limit_rpm` | Add `budget_period` select and `rate_limit_rpm` number input. |
| Teams table shows blank `keys_count`/`members` | Remove those columns until the backend returns them, or call `GET /api/keys?team_id=...` to compute counts. Decision: remove the columns to keep the table honest. |
| Endpoint `recordAudit()` calls `POST /api/audit` which does not exist | Remove client-side `recordAudit` calls; server already records mutations via `withAudit`. Delete the `recordAudit` function. |
| Endpoint sample curl hard-codes `gpt-4o` | Use currently selected provider/model from state. |
| Logs page calls `/api/usage` | Switch to `GET /api/logs?limit=100`. |
| `lovable-error-reporting.ts` is a no-op | Remove the file and its import in `__root.tsx`. |

**API endpoints used:** `POST /api/keys/:id/regenerate`, `GET /api/logs`, `GET /api/version`

---

### Slice 6 — Missing Pages (11 `<ComingSoon />` routes)
**Owner:** Subagent F  
**Files:**
- `ui/src/routes/_app.alerts.tsx`
- `ui/src/routes/_app.guardrails.tsx`
- `ui/src/routes/_app.model-limits.tsx`
- `ui/src/routes/_app.prompts.tsx`
- `ui/src/routes/_app.feature-flags.tsx`
- `ui/src/routes/_app.proxy-pools.tsx`
- `ui/src/routes/_app.mcp.index.tsx`
- `ui/src/routes/_app.mcp.instances.tsx`
- `ui/src/routes/_app.mcp.accounts.tsx`
- `ui/src/routes/_app.mcp.tools.tsx`
- `ui/src/routes/_app.mcp.tool-groups.tsx`

For each page, implement a working CRUD/feature UI using `CrudPage` for list+create+edit+delete where applicable, or a bespoke page where the feature is more complex (e.g., MCP instances with OAuth flow).

**API endpoints used:**
- Alerts: `GET/POST/PUT/DELETE /api/alert-channels`, `POST /api/alert-channels/:id/test`
- Guardrails: `GET /api/guardrails`, `PUT /api/guardrails`, `POST /api/guardrails/test`
- Model limits: `GET/POST/PUT/DELETE /api/model-limits`
- Prompts: `GET/POST/PUT/DELETE /api/prompt-templates`, `POST /api/prompt-templates/:id/test`
- Feature flags: `GET /api/feature-flags`, `PATCH /api/feature-flags/:id`
- Proxy pools: `GET/POST/PUT/DELETE /api/proxy-pools`, `POST /api/proxy-pools/batch`, `POST /api/proxy-pools/:id/test`
- MCP: aggregate `GET /api/mcp/clients`, `GET/POST/DELETE /api/mcp/instances`, `GET /api/mcp/instances/:id/accounts`, `GET /api/mcp/tools`, `POST /api/mcp/tools/:id/execute`, `GET/POST/PUT/DELETE /api/mcp/tool-groups`

---

### Slice 7 — Cleanup + Polish
**Owner:** Subagent G (or root agent)  
**Files:** `ui/src/routes/_app.settings.tsx`, `ui/src/components/topology/TopologyLegend.tsx`, `ui/src/lib/types.ts`

| Issue | Fix |
|-------|-----|
| Settings hard-coded arrays | Document that these are UI defaults; no backend endpoint exists for locale/caveman levels. No code change unless a setting endpoint is added. |
| TopologyLegend says "last 30s" statically | Read `window_sec` from filters context and render dynamically. |

---

## Coordination Rules

1. **No new dependencies.** Reuse existing shadcn/ui components, `apiFetch`, TanStack Query, and `toast`.
2. **No backend changes unless a contract mismatch is found.** If a mismatch is discovered, the subagent stops and escalates to the coordinator.
3. **Each subagent must run `npm run build` in `ui/` before returning.** Build errors are fixed inline.
4. **Each subagent returns a summary:** files changed, API endpoints used, known limitations, manual test steps.
5. **Coordinator final gates:** `go test ./...`, `go vet ./...`, `npm run build`, server restart, spot-check 3+ formerly broken flows.

---

## Success Criteria

- [ ] Zero occurrences of `<ComingSoon />` in `ui/src/routes/`.
- [ ] Zero no-op buttons (every `onClick` or `to` does something real).
- [ ] Zero hard-coded demo credentials, versions, or mock data in user-facing UI.
- [ ] `npm run build` succeeds with no new errors.
- [ ] `go test ./...` and `go vet ./...` pass.
- [ ] Server starts and at least 3 previously broken flows work end-to-end.

---

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Backend API contract mismatch | Subagent escalates; coordinator fixes backend view or frontend type. |
| Large diff is hard to review | Each subagent produces a focused slice; summary table above makes ownership clear. |
| Race conditions on shared files | Subagents work on disjoint files; only `types.ts` is touched by multiple slices — coordinator handles merge. |
| Build bloat from new pages | Existing `CrudPage` is reused; no new heavy dependencies. |

---

## Open Questions / Decisions

1. **Settings hard-coded arrays** — Keep as UI defaults; no backend endpoint exists today.
2. **Notifications bell** — Remove rather than implement a backend feed.
3. **Teams table columns** — Remove `keys_count`/`members` until backend supports them.
4. **Client-side audit** — Remove; server already records mutations.
5. **Lovable error reporting** — Remove the no-op module.
