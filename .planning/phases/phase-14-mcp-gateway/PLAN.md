# Phase 14: MCP Gateway

**Phase:** 14  
**Goal:** Implement the MCP gateway with clients, instances, tools, and tool groups.  
**Requirements:** MGMT-12, UI-14, PLAT-11  
**Estimated duration:** 5–6 days  
**Wave:** 5 — 9router Features

---

## Why

MCP enables AI models to use external tools (filesystem, web search, databases). This is a core 9router feature for agentic workflows.

---

## Scope

### In scope
- `internal/mcp/server.go` — MCP server lifecycle.
- `internal/mcp/client.go` — MCP client management (STDIO, SSE).
- `internal/mcp/tools.go` — tool registration, discovery, filtering, execution.
- `internal/admin/mcp.go` — `/api/mcp/clients`, `/api/mcp/instances`, `/api/mcp/tools`, `/api/mcp/tool-groups`.
- Dashboard pages:
  - `routes/_app.mcp.index.tsx`
  - `routes/_app.mcp.instances.tsx`
  - `routes/_app.mcp.accounts.tsx`
  - `routes/_app.mcp.tools.tsx`
  - `routes/_app.mcp.tool-groups.tsx`

### Out of scope
- Advanced MCP agent orchestration with multi-turn loops (future).

---

## Verification

### Tests
1. MCP client registration persists configuration.
2. MCP instance supports STDIO and SSE connection types.
3. Tool discovery fetches tool list from connected server.
4. Tool execution endpoint invokes tool and returns result.
5. Tool groups filter tools correctly.

### Manual verification
1. Register a filesystem MCP server and list tools.
2. Execute `read_file` through the gateway.

---

## Tasks

1. Implement MCP protocol types.
2. Implement client lifecycle manager.
3. Implement tool manager.
4. Implement admin handlers.
5. Implement dashboard pages.
6. Write tests and E2E coverage.
7. Verify gates.

---

## Risks

| Risk | Mitigation |
|------|------------|
| MCP STDIO process management is brittle | Use process supervision with health checks and restart limits. |
| Tool schema version drift | Store schema with tool registration and refresh on demand. |
