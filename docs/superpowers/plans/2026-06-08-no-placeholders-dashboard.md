# No Placeholders Dashboard — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development. Each slice below is dispatched to an independent coder subagent. The root agent coordinates and runs final gates.

**Goal:** Remove every placeholder, mock, and no-op from the g0router React dashboard by wiring all screens, buttons, forms, and dialogs to real backend APIs.

**Architecture:** Seven independent parallel work slices, each owned by one subagent. Each subagent reads existing patterns (`CrudPage`, `Dialog`, `apiFetch`, TanStack Query), implements their slice, runs `npm run build` in `ui/`, and returns a summary. The root agent resolves cross-cutting issues and runs final integration gates.

**Tech Stack:** React 19, TanStack Router + Query, Tailwind CSS v4, shadcn/ui components, Vite, Go backend with fasthttp.

---

## Pre-Flight Checklist (Root Agent)

- [ ] Confirm spec is approved: `docs/superpowers/specs/2026-06-08-no-placeholders-dashboard-design.md`
- [ ] Stop any running g0router server
- [ ] Ensure `ui/node_modules` is installed
- [ ] Baseline build passes: `cd ui && npm run build`
- [ ] Baseline Go tests pass: `go test ./...` and `go vet ./...`

---

## Slice 1: Security + Core Shell

**Subagent:** A  
**Goal:** Remove hard-coded demo credentials, no-op notification bell, hard-coded version, and stale backup file.

**Files:**
- Modify: `ui/src/routes/login.tsx`
- Modify: `ui/src/components/layout/Header.tsx`
- Modify: `ui/src/components/layout/Sidebar.tsx`
- Delete: `ui/src/routes/_app.dashboard.tsx.bak`

**API Endpoints:**
- `GET /api/version` — returns `{current: string, latest?: string, update_available?: bool, ...}`

**Implementation Steps:**

1. **Login page cleanup**
   - Remove the demo banner block near the form.
   - Remove `defaultValue="admin"` and `defaultValue="123456"` from username/password inputs.
   - Keep form empty on first load.

2. **Header notifications bell**
   - Find the bell `<button>` in `Header.tsx`.
   - Remove the button and its surrounding wrapper entirely.

3. **Sidebar version badge**
   - Add `useQuery` to fetch `GET /api/version`.
   - Replace hard-coded `"v0.9.0"` with `data?.current ?? "…"`.
   - Show a loading ellipsis while fetching.

4. **Delete stale backup**
   - `git rm ui/src/routes/_app.dashboard.tsx.bak`

**Verification:**
- `npm run build` passes
- Login page shows empty form, no demo banner
- Sidebar shows a version string (live from `/api/version`)
- No bell icon in header

---

## Slice 2: Chat + Sessions

**Subagent:** B  
**Goal:** Remove hard-coded provider/model, fix auth header to use full key, wire recent sessions and new-chat persistence.

**Files:**
- Modify: `ui/src/routes/_app.chat.tsx`
- Modify: `ui/src/lib/types.ts`

**API Endpoints:**
- `GET /api/providers`
- `GET /api/models`
- `GET /api/keys`
- `GET /api/chat-sessions/:id`
- `POST /api/chat-sessions`
- `POST /v1/chat/completions`

**Implementation Steps:**

1. **Remove hard-coded defaults**
   - Replace `const [provider, setProvider] = useState("openai")` with `useState<string>("")`.
   - Replace `const [model, setModel] = useState("gpt-4o")` with `useState<string>("")`.
   - On mount, fetch `/api/providers` and `/api/models`.
   - Default to first available provider/model once loaded.

2. **Fix chat auth header**
   - Locate the `Authorization: Bearer ${firstKey.prefix}` line.
   - Change to `firstKey.full_key`.
   - If `full_key` is missing, toast a warning ("Copy the full key from API Keys to use chat") and disable the Send button.

3. **Wire recent sessions**
   - Locate the recent sessions list (around lines 222–231).
   - Add `onClick={() => loadSession(s.id)}`.
   - Implement `loadSession(id)`:
     ```ts
     const session = await apiFetch<ChatSession>(`/api/chat-sessions/${id}`);
     setMessages(session.messages);
     setProvider(session.provider);
     setModel(session.model);
     ```

4. **Persist new chat sessions**
   - Locate the "New chat" button handler.
   - Before clearing local state, call:
     ```ts
     await apiFetch("/api/chat-sessions", {
       method: "POST",
       body: { title: "New chat", provider, model },
     });
     ```
   - Then clear messages.

5. **Clean up mock comments in types.ts**
   - Remove `// plaintext for mock` comment.
   - Remove `// mock-only` comment.

**Verification:**
- `npm run build` passes
- Chat page loads with a real provider/model selected
- Sending a message uses `full_key` in the Authorization header
- Recent sessions are clickable and load
- New chat creates a persisted session

---

## Slice 3: Provider Detail + Connections

**Subagent:** C  
**Goal:** Add Edit/Delete actions to connections on both provider detail and connections pages; wire suggested models.

**Files:**
- Modify: `ui/src/routes/_app.providers.$id.tsx`
- Modify: `ui/src/routes/_app.connections.tsx`
- Reuse components: `ConfirmDialog` from `@/components/common/ConfirmDialog`, `Dialog` from `@/components/ui/dialog`

**API Endpoints:**
- `PUT /api/connections/:id`
- `DELETE /api/connections/:id`
- `GET /api/providers/:id/suggested-models`

**Implementation Steps:**

1. **Connection edit dialog (shared component or duplicated inline)**
   - Create a small `EditConnectionDialog` that accepts a `Connection` and `onSave`.
   - Fields: Name, Auth type (read-only or editable), Credential (API key / access token), Active toggle.
   - On save: `PUT /api/connections/:id` with the updated fields.

2. **Provider detail connections table**
   - Add an Actions column with Edit and Delete icons.
   - Edit opens the dialog.
   - Delete uses `ConfirmDialog` → `DELETE /api/connections/:id` → invalidate queries.

3. **Connections page**
   - Add the same Edit icon to each row.
   - Reuse the same dialog component.

4. **Suggested models**
   - On provider detail mount, optionally call `GET /api/providers/:id/suggested-models`.
   - Display as a compact list or an "Add all" button (optional — at minimum log/display them).

**Verification:**
- `npm run build` passes
- Edit connection updates name/credentials
- Delete connection removes it after confirmation
- Suggested models load on provider detail

---

## Slice 4: Routing Rules

**Subagent:** D  
**Goal:** Fix routing-rules form to send condition fields and align frontend type with backend flat shape.

**Files:**
- Modify: `ui/src/routes/_app.routing-rules.tsx`
- Modify: `ui/src/lib/types.ts` (if `RoutingRule` type is wrong)

**API Endpoints:**
- `GET /api/routing-rules`
- `POST /api/routing-rules`
- `PUT /api/routing-rules/:id`
- `DELETE /api/routing-rules/:id`

**Implementation Steps:**

1. **Read backend contract**
   - Inspect `api/handlers/routingrules.go` to confirm the request/response shape.
   - Expected flat fields: `id`, `name`, `priority`, `cond_field`, `cond_operator`, `cond_value`, `target_provider`, `target_model`.

2. **Update frontend type**
   - If `RoutingRule` has `condition: { field, operator, value }`, replace with flat fields.

3. **Update form**
   - Add three inputs: Condition field (text), Condition operator (select: `eq`, `contains`, `starts_with`, `regex`, etc.), Condition value (text).
   - On create/update, include `cond_field`, `cond_operator`, `cond_value` in the body.

4. **Update table columns**
   - Show condition as a readable string like "field eq value".

**Verification:**
- `npm run build` passes
- Create a routing rule with a condition; backend stores it correctly
- List view shows the condition

---

## Slice 5: Keys + Teams + Endpoint + Logs

**Subagent:** E  
**Goal:** Add regenerate action and missing form fields to API Keys; fix Teams form/columns; remove broken client-side audit; fix logs endpoint; remove no-op error reporting.

**Files:**
- Modify: `ui/src/routes/_app.keys.tsx`
- Modify: `ui/src/routes/_app.teams.tsx`
- Modify: `ui/src/routes/_app.endpoint.tsx`
- Modify: `ui/src/routes/_app.logs.tsx`
- Modify: `ui/src/routes/__root.tsx` (remove lovable import)
- Delete: `ui/src/lib/lovable-error-reporting.ts`

**API Endpoints:**
- `POST /api/keys/:id/regenerate`
- `GET /api/logs?limit=100`
- `GET /api/version`

**Implementation Steps:**

1. **API Keys regenerate**
   - Add a Regenerate row action to the keys table.
   - On click, call `POST /api/keys/:id/regenerate`.
   - Show the returned `full_key` once in a read-only dialog with a Copy button; warn that it won't be shown again.
   - Invalidate the keys query.

2. **API Key form fields**
   - Add `scopes` field: multi-select or checkboxes (`chat`, `completions`, `embeddings`, `images`, `audio`, `admin`).
   - Add `expires_at` field: date/time input. Send ISO string or unix timestamp depending on backend contract (inspect `api/handlers/apikeys.go`).

3. **Teams form fields**
   - Add `budget_period` select (`daily`, `weekly`, `monthly`).
   - Add `rate_limit_rpm` number input.

4. **Teams table columns**
   - Remove the `keys_count` and `members` columns (they render blank because the backend does not return them).

5. **Endpoint page audit fix**
   - Locate `recordAudit` in `_app.endpoint.tsx`.
   - Delete the function and all its call sites.

6. **Endpoint dynamic curl**
   - Read `provider` and `model` from local state.
   - Render sample curl using the selected provider/model; fall back to first available.

7. **Logs endpoint fix**
   - Change `apiFetch("/api/usage?limit=100")` to `apiFetch("/api/logs?limit=100")`.

8. **Remove lovable error reporting**
   - Delete `ui/src/lib/lovable-error-reporting.ts`.
   - Remove its import and usage from `__root.tsx`.

**Verification:**
- `npm run build` passes
- Regenerate key shows a new full_key
- Key create form has scopes and expires_at
- Teams form has budget_period and rate_limit_rpm
- Teams table no longer has blank columns
- Endpoint copy/export no longer calls missing `/api/audit`
- Logs page fetches from `/api/logs`

---

## Slice 6: Missing Pages (11 `<ComingSoon />` Routes)

**Subagent:** F  
**Goal:** Replace every `<ComingSoon />` page with a working CRUD/feature page that matches the backend contract.

**Files to create/modify:**
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

**Backend Endpoints to Inspect First:**
- Alerts: `api/handlers/alertchannels.go`
- Guardrails: `api/handlers/guardrails.go`
- Model limits: `api/handlers/modellimits.go`
- Prompts: `api/handlers/prompts.go`
- Feature flags: `api/handlers/featureflags.go`
- Proxy pools: `api/handlers/proxypools.go`
- MCP: `api/handlers/mcp.go`

**API Endpoints:**
- `GET/POST/PUT/DELETE /api/alert-channels`
- `POST /api/alert-channels/:id/test`
- `GET /api/guardrails`, `PUT /api/guardrails`, `POST /api/guardrails/test`
- `GET/POST/PUT/DELETE /api/model-limits`
- `GET/POST/PUT/DELETE /api/prompt-templates`
- `POST /api/prompt-templates/:id/test`
- `GET /api/feature-flags`, `PATCH /api/feature-flags/:id`
- `GET/POST/PUT/DELETE /api/proxy-pools`
- `POST /api/proxy-pools/batch`, `POST /api/proxy-pools/:id/test`
- `GET /api/mcp/clients`, `GET/POST/DELETE /api/mcp/instances`
- `GET /api/mcp/instances/:id/accounts`
- `GET /api/mcp/tools`, `POST /api/mcp/tools/:id/execute`
- `GET/POST/PUT/DELETE /api/mcp/tool-groups`

**Implementation Steps:**

1. **Read each backend handler** to determine the exact request/response shape.

2. **Simple CRUD pages** (Alerts, Model Limits, Prompts, Proxy Pools, MCP Tool Groups):
   - Use `CrudPage` if the pattern fits; otherwise build a simple table + dialog page.
   - List: `GET`
   - Create: `POST`
   - Edit: `PUT`
   - Delete: `DELETE` with `ConfirmDialog`
   - Test action where available (Alerts, Prompts, Proxy Pools)

3. **Toggle/config pages** (Guardrails, Feature Flags):
   - Guardrails: form with switches/inputs for each guardrail type; save with `PUT`; test with `POST`.
   - Feature flags: table with name + toggle; `PATCH` to toggle.

4. **MCP pages**:
   - **Index:** aggregate summary of clients, tools, instances.
   - **Instances:** list + create (with auth type selector) + delete; optional OAuth start/complete flow.
   - **Accounts:** list accounts for a selected instance.
   - **Tools:** list + execute (dialog with JSON payload).
   - **Tool Groups:** CRUD.

5. **Update sidebar navigation** if any new routes need explicit links (check `ui/src/components/layout/Sidebar.tsx`).

**Verification:**
- `npm run build` passes
- Each page loads real data from its backend endpoint
- Create/edit/delete actions work where applicable
- Test actions produce a toast result

---

## Slice 7: Cleanup + Polish

**Subagent:** G  
**Goal:** Fix topology legend, document hard-coded settings arrays, ensure no leftover mock language.

**Files:**
- Modify: `ui/src/components/topology/TopologyLegend.tsx`
- Modify: `ui/src/routes/_app.settings.tsx` (add comment only)

**Implementation Steps:**

1. **TopologyLegend dynamic window**
   - Read `window_sec` from the topology filters context/state.
   - Replace hard-coded "last 30s" with `last ${window_sec}s` (or humanized like "last 2 min" when appropriate).

2. **Settings arrays comment**
   - Add a code comment above `SOURCE_OPTIONS`, `LOCALE_OPTIONS`, `CAVEMAN_LEVELS` noting they are UI defaults because no backend enumeration endpoint exists.

**Verification:**
- `npm run build` passes
- Topology legend text changes when window selector changes

---

## Cross-Cutting Concerns

### Type Consistency
- All new/changed frontend types go in `ui/src/lib/types.ts`.
- Prefer extending existing types rather than creating ad-hoc inline types.

### API Contract Mismatches
- If a subagent finds a mismatch (backend returns snake_case but frontend expects camelCase, missing fields, etc.), they must pause and report it to the root agent.
- The root agent decides whether to fix the backend view or the frontend type.

### Error Handling
- All API calls go through `apiFetch`, which already toasts on error and redirects on 401.
- Subagents should not add additional try/catch unless they need special handling.

### Styling
- Reuse existing Tailwind classes and `cn()` utility.
- Do not introduce new color names or spacing values.

---

## Coordinator Final Gates

After all subagents report complete:

1. **Merge conflicts**
   - Run `git status`.
   - Resolve any conflicts in shared files (`types.ts`, `__root.tsx`, `Sidebar.tsx`).

2. **Full test suite**
   ```bash
   cd /Users/heitor/Developer/github.com/bloodf/g0router
   go test ./... -count=1
   go vet ./...
   cd ui && npm run build
   ```

3. **Server restart and spot checks**
   - Build binary: `go build -o g0router ./cmd/g0router`
   - Start server: `env DATA_DIR=./debug_data PORT=20128 ./g0router serve`
   - Spot-check at least 3 previously broken flows, e.g.:
     - Login page has no prefilled credentials
     - `/alerts` page loads real alert channels
     - Chat page can send a message with a real key
     - Routing rules can be created with a condition

4. **Update WORKFLOW.md**
   - Append a hotfix entry summarizing all changes.

5. **Commit**
   ```bash
   git add -A
   git commit -m "hotfix: wire all dashboard placeholders and mocks to real APIs"
   ```

---

## Execution Order

1. Dispatch all 7 subagents in parallel.
2. Each subagent completes their slice and runs `npm run build`.
3. Root agent resolves conflicts and runs final gates.
4. Restart server and verify.
5. Update WORKFLOW.md and commit.
