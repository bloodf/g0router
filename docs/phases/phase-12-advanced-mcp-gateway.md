# Phase 12: Advanced MCP Gateway

> **Depends on**: Phase 11
> **Unlocks**: PROJECT COMPLETE
> **Checkpoint**: `PHASE_12_COMPLETE`

---

## Prerequisites

- [x] Phase 11 complete
- [x] Phase 9 MCP base complete
- [x] Dashboard, CLI, management API, packaging, and E2E suite complete
- [x] `go test ./...` passes
- [x] `go vet ./...` passes

## Scope

Phase 9 builds the baseline MCP gateway: client manager, tool manager, compact discovery, agent loop, health checks, and basic handlers. Phase 12 extends that gateway after the rest of the product is complete.

The goal is that a user can add any practical MCP server shape:

- `stdio` command servers, including local binaries.
- `npx` launched MCP servers, such as Expo MCP.
- `docker` launched MCP servers, modeled as controlled stdio subprocesses.
- Remote `streamable-http` MCP servers.
- Legacy HTTP+SSE MCP servers when discovery falls back to the older transport.

This phase also adds proper OAuth account handling for MCP servers:

- Multiple instances of the same MCP server definition are allowed.
- Each instance can be authenticated to a different account.
- Tokens, callback state, manifests, health, and process lifecycle are scoped to the instance.
- A user can paste an OAuth callback URL when the redirect returns to a localhost URL that g0router cannot receive directly.
- HTTP-based MCP authorization follows the MCP authorization flow. Stdio MCP servers receive credentials through per-instance environment, files, or explicit helper flows instead of pretending stdio runs HTTP OAuth.

## Non-Goals

- No plugin architecture.
- No changes to the core LLM provider fallback behavior.
- No speculative support for non-MCP tool protocols.
- No shared global MCP account state.
- No token logging, callback URL logging, or environment value logging.

## Reference Specs

- MCP transports: `https://modelcontextprotocol.io/specification/2025-11-25/basic/transports`
- MCP authorization: `https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization`

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Unit of config | MCP instance | Lets the same MCP server be added more than once, each with its own account, auth, env, manifest, and health state. |
| Launch types | `command`, `npx`, `docker`, `http` | Covers local binaries, npm packages, containerized servers, and remote hosted servers without a plugin system. |
| Transport mapping | `command`, `npx`, and `docker` use stdio; `http` uses streamable HTTP with legacy SSE fallback | Matches MCP transport semantics while keeping launch mechanics separate from protocol transport. |
| OAuth owner | Per MCP instance | Supports Atlassian-style multiple accounts by creating multiple instances rather than overwriting one token set. |
| OAuth callback completion | Accept local callback route or pasted callback URL | Handles normal browser redirects and flows where the redirect lands on another localhost process. |
| HTTP OAuth | Protected Resource Metadata, authorization server metadata, PKCE, state, resource parameter, bearer tokens | Keeps HTTP MCP auth aligned with the MCP authorization spec. |
| Stdio OAuth | Instance-scoped helper flows and credential injection | Stdio servers should get credentials from the environment or their own local flow, not HTTP MCP authorization. |
| Tool naming | Stable instance-qualified tool identity | Prevents collisions when two instances expose the same tool names. |

---

## Data Model Plan

Phase 12 should add or migrate toward explicit instance-oriented tables. Exact DDL can be adjusted during implementation, but the implementation must preserve these concepts:

```sql
-- One row per configured MCP server instance.
mcp_instances (
    id,
    name,
    server_key,
    launch_type,       -- command | npx | docker | http
    transport,         -- stdio | streamable-http | sse
    command,
    args,
    url,
    headers,
    env,
    cwd,
    account_label,
    is_active,
    health_status,
    last_health_check,
    tool_manifest,
    manifest_updated_at,
    created_at,
    updated_at
)

-- One row per authenticated account/session for an MCP instance.
mcp_oauth_accounts (
    id,
    instance_id,
    account_label,
    subject,
    email,
    issuer,
    resource_uri,
    scopes,
    access_token,
    refresh_token,
    expires_at,
    auth_metadata,
    created_at,
    updated_at
)

-- Short-lived pending OAuth attempts.
mcp_oauth_flows (
    id,
    instance_id,
    state_hash,
    code_verifier_secret,
    redirect_uri,
    authorization_url,
    resource_uri,
    expires_at,
    created_at
)
```

## Implemented API Surface

```text
GET    /api/mcp/instances
POST   /api/mcp/instances
DELETE /api/mcp/instances/:id

GET    /api/mcp/instances/:id/accounts
POST   /api/mcp/instances/:id/auth/start
POST   /api/mcp/instances/:id/oauth/complete
GET    /api/mcp/oauth/callback
```

`GET`/`PUT` per-instance mutation, explicit refresh-tools endpoints, and account
delete endpoints are not exposed as separate management routes in the current
release. Instance deletion cascades stored OAuth accounts and flows.

## Implemented CLI Surface

```text
g0router mcp add <name> --launch-type command --transport stdio --command <command> [--arg <arg>...]
g0router mcp add <name> --launch-type npx --transport stdio --command <package> [--arg <arg>...]
g0router mcp add <name> --launch-type docker --transport stdio --command <image> [--arg <arg>...]
g0router mcp add <name> --launch-type http --transport streamable-http --url <url>
g0router mcp auth start <instance>
g0router mcp auth complete <instance> <callback-url>
g0router mcp accounts <instance>
g0router mcp tools <instance>
g0router mcp remove <instance>
```

---

## Task 12.1: MCP Instance Model + Store

### Completed Work

- [x] Write `internal/store/mcpinstances_test.go` first.
- [x] Write `internal/mcp/instances_test.go` first.
- [x] Define instance identity, launch config, account labels, and manifest ownership.
- [x] Migrate existing `mcp_clients` behavior into instance-oriented APIs without breaking Phase 9 handlers.
- [x] Commit: `phase-12/task-1: mcp instance model and store`

### Required Tests

- Creating two instances with the same `server_key` but different names is valid.
- Creating two instances with the same name is rejected.
- A manifest belongs to one instance and is not shared with sibling instances.
- Instance env and headers are redacted in list responses.
- Invalid launch types and transports are rejected.

---

## Task 12.2: MCP Launchers for Command, Npx, Docker, and HTTP

### Completed Work

- [x] Write `internal/mcp/launcher_test.go` first.
- [x] Implement launch specs for `command`, `npx`, `docker`, and `http`.
- [x] Use fakes for process runners and HTTP servers; do not use mocks.
- [x] Keep docker and npx as launch modes, not protocol transports.
- [x] Commit: `phase-12/task-2: mcp launcher matrix`

### Required Tests

- `npx` launch builds a stdio subprocess spec without shell interpolation.
- `docker` launch builds a stdio subprocess spec with `docker run -i` and explicit args.
- `command` launch preserves args as an argv array.
- `http` launch sends MCP requests through streamable HTTP and stores returned session IDs.
- Legacy SSE fallback is attempted only after streamable HTTP initialization fails with the documented fallback status codes.
- Stderr from stdio servers is captured for diagnostics without marking every stderr line as failure.

---

## Task 12.3: MCP OAuth Account Engine

### Completed Work

- [x] Write `internal/mcp/oauth_test.go` first.
- [x] Write `internal/store/mcpoauth_test.go` first.
- [x] Implement pending auth flow state, PKCE verifier storage, resource URI handling, token storage, and refresh.
- [x] Support HTTP MCP auth discovery from `WWW-Authenticate` and well-known resource metadata.
- [x] Support user-provided client credentials when metadata documents and dynamic registration are unavailable.
- [x] Commit: `phase-12/task-3: mcp oauth account engine`

### Required Tests

- OAuth state is single-use and scoped to one instance.
- A pasted callback for one instance cannot complete another instance's flow.
- Tokens are stored under `instance_id` and account identity.
- Multiple Atlassian MCP instances can hold different account tokens at the same time.
- HTTP MCP requests include bearer tokens and the negotiated MCP protocol version.
- Invalid, expired, or wrong-resource tokens trigger reauth or refresh rather than leaking another account's token.
- Stdio instances can receive credentials through redacted env material without using HTTP MCP auth.

---

## Task 12.4: OAuth Callback URL Completion

### Completed Work

- [x] Write `api/handlers/mcpoauth_test.go` first.
- [x] Write CLI tests for pasted callback URL completion first.
- [x] Implement `/api/mcp/oauth/callback`.
- [x] Implement `POST /api/mcp/instances/:id/oauth/complete`.
- [x] Implement `g0router mcp auth complete <instance> <callback-url>`.
- [x] Commit: `phase-12/task-4: mcp oauth callback completion`

### Required Tests

- Normal redirect callback completes the pending flow.
- Pasted callback URL completion extracts query params, validates state, and exchanges the code.
- Callback URLs missing `code` are rejected.
- Callback URLs with mismatched `state` are rejected.
- Callback URLs are never persisted or logged with raw authorization codes.
- Expo-style `npx` flow can be completed by pasting the localhost callback URL into g0router.

---

## Task 12.5: MCP Management Surfaces

### Completed Work

- [x] Write API tests first for instance CRUD, auth start, auth complete, accounts, and tool listing.
- [x] Write CLI tests first for add/auth/accounts/tools/remove commands.
- [x] Update the dashboard MCP page to manage instances and accounts.
- [x] Show per-instance health, auth status, launch type, tool count, and account label.
- [x] Commit: `phase-12/task-5: mcp instance management surfaces`

### Required Tests

- API list responses redact secrets and tokens.
- Tool list can filter by instance and account.
- Tool identities remain stable when two instances expose the same tool name.
- Deleting one instance removes its OAuth accounts and cached tools without affecting sibling instances.
- Dashboard and CLI can add one Atlassian instance for account A and another for account B.

---

## Task 12.6: Advanced MCP Integration Tests + Docs

### Completed Work

- [x] Write integration tests first with fake MCP servers and fake OAuth endpoints.
- [x] Add a documented manual verification path for real `npx`, `docker`, and remote HTTP MCP servers.
- [x] Update `docs/SCHEMA.md`, `docs/CONFIG.md`, `docs/DEPLOYMENT.md`, and README examples with the final implemented contracts.
- [x] Commit: `phase-12/task-6: advanced mcp integration docs`

### Required Tests

- Fake HTTP MCP OAuth flow completes through redirect and pasted callback URL.
- Fake stdio MCP server launched through command mode lists and executes tools.
- Fake npx launch spec is verified without requiring network access.
- Docker verification is skipped only when Docker is unavailable, with a clear test reason.
- Token refresh works without changing the selected MCP account.
- `go test ./...`, `go vet ./...`, and `go build ./cmd/g0router` pass.

---

## Phase Gate

```bash
go test ./... -count=1
go vet ./...
go build ./cmd/g0router
```

## Phase Checklist

- [x] Task 12.1 complete (MCP Instance Model + Store)
- [x] Task 12.2 complete (MCP Launchers for Command, Npx, Docker, and HTTP)
- [x] Task 12.3 complete (MCP OAuth Account Engine)
- [x] Task 12.4 complete (OAuth Callback URL Completion)
- [x] Task 12.5 complete (MCP Management Surfaces)
- [x] Task 12.6 complete (Advanced MCP Integration Tests + Docs)
- [x] All tests pass: `go test ./...`
- [x] Vet clean: `go vet ./...`
- [x] Build succeeds: `go build ./cmd/g0router`
- [x] All commits follow `phase-12/task-N: description` format
- [x] Update `docs/WORKFLOW.md`: phase_12.status -> `DONE`
- [x] **PHASE_12_COMPLETE** -> **PROJECT COMPLETE**
