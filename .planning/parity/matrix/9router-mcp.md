# MCP Parity Matrix: 9router (reference) → g0router

Reference SHA: `827e5c3` (frozen).

---

## Behavior Matrix

| ID | Behavior | Evidence (file:line) | g0router status | Notes |
|---|---|---|---|---|
| PAR-MCP-001 | SSE endpoint exposes stdio MCP plugins over HTTP | `src/app/api/mcp/[plugin]/sse/route.js:6` | MISSING | Route returns 403 because Cowork is disabled in reference. No equivalent route exists in g0router. |
| PAR-MCP-002 | Message POST endpoint forwards JSON-RPC to stdio child | `src/app/api/mcp/[plugin]/message/route.js:7` | MISSING | Route returns 403 because Cowork is disabled. No equivalent route exists. |
| PAR-MCP-003 | Stdio<->SSE bridge spawns one child process per plugin on demand | `src/lib/mcp/stdioSseBridge.js:146` | MISSING | Uses `child_process.spawn` with `stdio: ["pipe","pipe","pipe"]`. g0router has no process launcher. |
| PAR-MCP-004 | Plugin command allowlist restricts spawnable binaries | `src/shared/constants/coworkPlugins.js:67` | MISSING | Allowlist: `npx`, `node`, `uvx`, `python`, `python3`, `bunx`, `bun`. g0router has no allowlist logic. |
| PAR-MCP-005 | Custom plugin registration validates command against allowlist | `src/lib/mcp/stdioSseBridge.js:119` | MISSING | `registerCustomPlugin` throws if `!isAllowedCommand(def?.command)`. g0router has no registration API. |
| PAR-MCP-006 | Custom plugins persist to disk and survive restart | `src/lib/mcp/stdioSseBridge.js:130` | MISSING | Reads/writes `DATA_DIR/mcp/customPlugins.json`. g0router has no MCP persistence layer. |
| PAR-MCP-007 | JSON-RPC stdout frames broadcast to all active SSE sessions | `src/lib/mcp/stdioSseBridge.js:151` | MISSING | `proc.stdout.on("data", ...)` parses newline-delimited JSON and loops over `entry.sessions.values()`. g0router has no SSE broadcaster. |
| PAR-MCP-008 | Tool result text filtering: drop noise nodes, collapse repeated siblings, hard-truncate at 50K chars | `src/lib/mcp/stdioSseBridge.js:20` | MISSING | `smartFilterText` regexes out `generic` and empty `text` lines, then `collapseRepeated` groups role-prefixed siblings. g0router has no result filter. |
| PAR-MCP-009 | MCP server probe performs initialize + notifications/initialized + tools/list handshake | `src/app/api/cli-tools/cowork-mcp-tools/route.js:9` | MISSING | Three-step handshake with 8s timeout. g0router has no probe logic. |
| PAR-MCP-010 | Probe uses MCP-Protocol-Version header 2025-06-18 | `src/app/api/cli-tools/cowork-mcp-tools/route.js:13` | MISSING | Header sent on every probe request. g0router has no MCP HTTP client. |
| PAR-MCP-011 | Probe reads mcp-session-id from initialize response and replays it on tools/list | `src/app/api/cli-tools/cowork-mcp-tools/route.js:34` | MISSING | Session state tracked across two HTTP requests. g0router has no session-aware MCP client. |
| PAR-MCP-012 | Probe parses SSE responses for tools/list | `src/app/api/cli-tools/cowork-mcp-tools/route.js:60` | MISSING | Splits on `data:` lines and finds `id === 2` result. g0router has no SSE parser for MCP. |
| PAR-MCP-013 | Probe returns `requiresAuth: true` on 401/403 | `src/app/api/cli-tools/cowork-mcp-tools/route.js:28` | MISSING | Distinguishes authless vs auth-required servers. g0router has no probe. |
| PAR-MCP-014 | Anthropic MCP registry client fetches servers with pagination | `src/app/api/cli-tools/cowork-mcp-registry/route.js:24` | MISSING | Up to 20 pages, 500 items/page, from `api.anthropic.com/mcp-registry/v0/servers`. g0router has no registry client. |
| PAR-MCP-015 | Registry cache TTL is 1 hour | `src/app/api/cli-tools/cowork-mcp-registry/route.js:7` | MISSING | `CACHE_TTL_MS = 60 * 60 * 1000`. g0router has no registry cache. |
| PAR-MCP-016 | Registry filters out claude.com and api.anthropic.com mediated servers | `src/app/api/cli-tools/cowork-mcp-registry/route.js:16` | MISSING | `isDirectConnect` rejects URLs matching `/\bmcp\.claude\.com\b/` and `/api\.anthropic\.com\/mcp/`. g0router has no registry filter. |
| PAR-MCP-017 | Registry excludes tenant-required entries | `src/app/api/cli-tools/cowork-mcp-registry/route.js:37` | MISSING | Skips entries where `meta.requiredFields?.length` is truthy. g0router has no registry logic. |
| PAR-MCP-018 | Cowork settings write `managedMcpServers` to Claude Desktop 3p config | `src/app/api/cli-tools/cowork-settings/route.js:262` | MISSING | Array of `{name, url, transport, oauth?, toolPolicy?}` injected into config. g0router has no 3p config writer. |
| PAR-MCP-019 | Managed servers generate `toolPolicy` with bare and prefixed tool names | `src/app/api/cli-tools/cowork-settings/route.js:171` | MISSING | For each tool name, emits both `t` and `{name}-{t}` as `"allow"`. g0router has no toolPolicy generator. |
| PAR-MCP-020 | `operonSkipMcpApprovals` auto-skips per-server approval prompts | `src/app/api/cli-tools/cowork-settings/route.js:240` | MISSING | Writes `skip[srv.name] = true` for every managed server. g0router has no Claude Desktop integration. |
| PAR-MCP-021 | Local stdio plugins bridged via localhost SSE endpoint | `src/app/api/cli-tools/cowork-settings/route.js:153` | MISSING | `buildLocalBridgeEntries` produces URLs like `http://localhost:${APP_PORT}/api/mcp/${name}/sse`. g0router has no local bridge. |
| PAR-MCP-022 | Custom plugins support both URL (HTTP/SSE) and command (stdio) modes | `src/app/api/cli-tools/cowork-settings/route.js:179` | MISSING | `buildCustomEntries` branches on `p.url` vs `p.command`. g0router has no custom plugin model. |
| PAR-MCP-023 | CLI token injected as `x-9r-cli-token` header into local bridge URLs | `src/app/api/cli-tools/cowork-settings/route.js:25` | MISSING | `injectAuthHeaders` adds header to URLs starting with `LOCAL_MCP_PREFIX`. g0router has no bridge auth injection. |
| PAR-MCP-024 | Tool deduplication strips built-in tools when equivalent MCP tools are present | `open-sse/utils/toolDeduper.js:33` | MISSING | Rules: Exa/Tavily MCP present → drop `WebSearch`/`WebFetch`; Browser MCP present → drop `mcp__Claude_in_Chrome__*`. g0router has no dedupe logic. |
| PAR-MCP-025 | Chat core integrates tool deduper for Claude clients only | `open-sse/handlers/chatCore.js:106` | MISSING | `if (clientTool === "claude" && Array.isArray(translatedBody.tools))` then `dedupeTools`. g0router has no client-specific dedupe. |
| PAR-MCP-026 | Cursor protobuf encoder formats MCP tool names as `mcp_{server}_{tool}` | `open-sse/utils/cursorProtobuf.js:241` | MISSING | `formatToolName` converts `mcp__server__tool` → `mcp_server_tool` or falls back to `mcp_custom_{name}`. g0router has no Cursor protobuf support. |
| PAR-MCP-027 | Cursor protobuf parses formatted tool name back to `{serverName, selectedTool}` | `open-sse/utils/cursorProtobuf.js:262` | MISSING | `parseToolName` splits on first underscore after `mcp_`. g0router has no protobuf parser. |
| PAR-MCP-028 | Cursor protobuf encodes `MCPResult` with `selected_tool` and `result` fields | `open-sse/utils/cursorProtobuf.js:295` | MISSING | Wire-type LEN encoding for both fields. g0router has no MCP protobuf encoder. |
| PAR-MCP-029 | Cursor protobuf encodes `ClientSideToolV2Call` with MCP params | `open-sse/utils/cursorProtobuf.js:332` | MISSING | Includes `tool`, `mcp_params`, `call_id`, `name`, `raw_args`, `tool_index`, `model_call_id`. g0router has no protobuf support. |
| PAR-MCP-030 | Schema defines MCPClient, MCPInstance, MCPTool, MCPToolGroup structs | `internal/schemas/mcp.go:4` | HAVE | Four structs with JSON tags. No code consumes them. |
| PAR-MCP-031 | MCP package compiles (placeholder) | `internal/mcp/mcp_test.go:7` | HAVE | `TestPackageCompiles` is an empty test. No real implementation exists. |
| PAR-MCP-032 | Database has no MCP tables | `internal/store/migrate.go:11` | MISSING | Migration creates `users`, `sessions`, `settings`, `providers`, `connections`, `oauth_sessions`. No `mcp_instances`, `mcp_oauth_accounts`, `mcp_oauth_flows`, or `mcp_clients`. |
| PAR-MCP-033 | No MCP API handlers | `api/` (empty except `.DS_Store`) | MISSING | `api/` directory contains only `.DS_Store`. No `handlers/mcp.go`, `handlers/mcpoauth.go`. |
| PAR-MCP-034 | No MCP CLI commands | `internal/cli/` (does not exist) | MISSING | Directory absent. No `mcp add`, `mcp auth start`, `mcp auth complete`, `mcp accounts`, `mcp tools`, `mcp remove`. |
| PAR-MCP-035 | No MCP store layer | `internal/store/` (no mcp files) | MISSING | Store contains `connections.go`, `crypto.go`, `migrate.go`, `oauthsessions.go`, `providers.go`, `secret.go`, `sessions.go`, `settings.go`, `store.go`, `users.go`. No `mcpinstances.go`, `mcpoauth.go`. |
| PAR-MCP-036 | No MCP launcher | `internal/mcp/` (no launcher files) | MISSING | No `launcher.go`, `process.go`, `http.go`. No `command`, `npx`, `docker`, or `http` launch types. |
| PAR-MCP-037 | No MCP OAuth account engine | `internal/mcp/` (no oauth files) | MISSING | No `oauth.go`. No PKCE, protected resource metadata, authorization server metadata, token storage, or refresh. |
| PAR-MCP-038 | No MCP health monitor | `internal/mcp/` (no health files) | MISSING | No `healthmonitor.go`. No periodic ping or auto-reconnect. |
| PAR-MCP-039 | No MCP discovery / compact injection | `internal/mcp/` (no discovery files) | MISSING | No `discovery.go`. No `tools/list` caching or compact manifest injection. |
| PAR-MCP-040 | No MCP agent loop | `internal/mcp/` (no agent files) | MISSING | No `agent.go`. No multi-turn tool execution loop. |
| PAR-MCP-041 | No MCP UI pages | `ui/src/` (no MCP pages) | MISSING | grep finds no `mcp` or `MCP` references in `ui/src/`. Dashboard has no MCP instance manager. |
| PAR-MCP-042 | Cowork feature disabled globally in reference | `src/app/api/mcp/[plugin]/sse/route.js:7`, `src/app/api/mcp/[plugin]/message/route.js:8`, `src/app/api/cli-tools/cowork-settings/route.js:314` | EXTRA | All MCP bridge and settings routes return 403 with comment "Cowork disabled: MCP stdio bridge spawns arbitrary processes (RCE risk)." g0router never had this feature enabled or disabled. |
| PAR-MCP-043 | Default plugins: Exa (HTTP, no auth), Tavily (HTTP, OAuth) | `src/shared/constants/coworkPlugins.js:3` | MISSING | Hardcoded defaults with `toolNames`, `transport`, `oauth` flags. g0router has no default MCP plugins. |
| PAR-MCP-044 | Local stdio plugin: browsermcp via npx | `src/shared/constants/coworkPlugins.js:26` | MISSING | Command `npx`, args `["-y", "@browsermcp/mcp@latest"]`, 10 tool names. g0router has no preset local plugins. |
| PAR-MCP-045 | `buildManagedMcpServers` strips `{name}-` prefixes idempotently | `src/shared/constants/coworkPlugins.js:48` | MISSING | While-loop removes repeated prefixes, then emits bare + single-prefixed variants in `toolPolicy`. g0router has no equivalent. |
| PAR-MCP-046 | Settings GET strips `{name}-` prefixes when returning plugin tool names | `src/app/api/cli-tools/cowork-settings/route.js:288` | MISSING | Same prefix-stripping logic as buildManagedMcpServers. g0router has no settings API. |
| PAR-MCP-047 | Settings GET prefers default `toolNames` over stored `toolPolicy` keys | `src/app/api/cli-tools/cowork-settings/route.js:297` | MISSING | `const toolNames = def && Array.isArray(def.toolNames) ? def.toolNames : Array.from(bare);`. g0router has no plugin settings. |
| PAR-MCP-048 | Settings POST sanitizes custom plugin names to `[^a-zA-Z0-9_-]` and truncates to 64 chars | `src/app/api/cli-tools/cowork-settings/route.js:339` | MISSING | `String(p.name || "").replace(/[^a-zA-Z0-9_-]/g, "").slice(0, 64)`. g0router has no custom plugin input. |
| PAR-MCP-049 | Settings POST filters custom plugins by allowlist before registration | `src/app/api/cli-tools/cowork-settings/route.js:337` | MISSING | `.filter((p) => ALLOWED_MCP_COMMANDS.has(path.basename(p.command)))`. g0router has no allowlist filtering. |
| PAR-MCP-050 | Settings DELETE resets Cowork config, skip approvals, and cleans legacy 1p entries | `src/app/api/cli-tools/cowork-settings/route.js:396` | MISSING | Writes empty object to config, calls `writeSkipApprovals([])` and `cleanup1pLegacy()`. g0router has no reset flow. |
| PAR-MCP-051 | `isRunning` helper checks if bridge process is alive | `src/lib/mcp/stdioSseBridge.js:193` | MISSING | Checks `!entry.proc.killed && entry.proc.exitCode === null`. g0router has no process lifecycle check. |
| PAR-MCP-052 | Bridge handles child stderr by logging to console | `src/lib/mcp/stdioSseBridge.js:165` | MISSING | `proc.stderr.on("data", (d) => console.log(...))`. g0router has no stdio bridge. |
| PAR-MCP-053 | Bridge handles child exit by deleting store entry | `src/lib/mcp/stdioSseBridge.js:167` | MISSING | `proc.on("exit", (code) => { store.delete(name); })`. g0router has no exit handler. |
| PAR-MCP-054 | Broken SSE sessions silently ignored on send failure | `src/lib/mcp/stdioSseBridge.js:160` | MISSING | `try { send(...) } catch { /* ignore broken pipe */ }`. g0router has no SSE session management. |
| PAR-MCP-055 | SSE handshake sends `endpoint` event with sessionId query param | `src/app/api/mcp/[plugin]/sse/route.js:22` | MISSING | `event: endpoint\ndata: /api/mcp/${plugin}/message?sessionId=${sid}`. g0router has no SSE handshake. |
| PAR-MCP-056 | SSE response sets no-cache, keep-alive, X-Accel-Buffering headers | `src/app/api/mcp/[plugin]/sse/route.js:30` | MISSING | Headers: `Content-Type: text/event-stream`, `Cache-Control: no-cache, no-transform`, `Connection: keep-alive`, `X-Accel-Buffering: no`. g0router has no SSE endpoint. |
| PAR-MCP-057 | Registry dedupes by URL | `src/app/api/cli-tools/cowork-mcp-registry/route.js:57` | MISSING | `seen.has(s.url)` filter after pagination loop. g0router has no registry dedupe. |
| PAR-MCP-058 | Probe timeout is 8 seconds | `src/app/api/cli-tools/cowork-mcp-tools/route.js:5` | MISSING | `TIMEOUT_MS = 8000`. g0router has no probe timeout. |
| PAR-MCP-059 | Probe catches AbortError and maps to `"timeout"` | `src/app/api/cli-tools/cowork-mcp-tools/route.js:78` | MISSING | `e.name === "AbortError" ? "timeout" : e.message`. g0router has no probe error mapping. |
| PAR-MCP-060 | Antigravity executor hardcodes `mcp_sequential-thinking_sequentialthinking` as unavailable | `open-sse/executors/antigravity.js:433` | MISSING | Tool definition with `description: "This tool is currently unavailable."`. g0router has no antigravity executor. |

---

## Data Models

### 9router (reference)

No explicit database schema. Runtime structures:

**Plugin definition (memory/disk)**
- `name`: string
- `command`: string (spawn binary)
- `args`: string[]
- `url`: string (for HTTP/SSE custom plugins)
- `transport`: `"sse" | "http"`
- `oauth`: boolean
- `toolNames`: string[]
- `toolPolicy`: Record<string, "allow"> (generated at apply time)

**Bridge store entry (global Map)**
- `proc`: ChildProcess
- `sessions`: Map<sid, sendFn>
- `buffer`: string (accumulated stdout)

**Registry item ( Anthropic API response shape )**
- `name`, `slug`, `title`, `description`, `url`, `transport`, `oauth`, `toolNames`, `toolCount`, `iconUrl`

### g0router

**`internal/schemas/mcp.go`**

```go
MCPClient {
  ID     string
  Name   string
  Type   string
  Config map[string]any
}

MCPInstance {
  ID        string
  ClientID  string
  Name      string
  Transport string
  URL       string
  Command   string
  Args      []string
  Env       map[string]string
  Status    string
}

MCPTool {
  Name        string
  Description string
  InputSchema map[string]any
}

MCPToolGroup {
  ID        string
  Name      string
  ToolNames []string
}
```

No database tables, store implementations, or API consumers exist for these types.

---

## Edge Cases and Quirks

1. **Cowork globally disabled in reference.** Every MCP bridge route and the settings POST return 403 before executing logic (`route.js:7-8`, `route.js:314-315`). The underlying code remains in place but is unreachable.

2. **Custom plugin name sanitization strips all characters outside `[a-zA-Z0-9_-]` and truncates to 64 bytes** (`cowork-settings/route.js:339`). This prevents shell injection through names but does not validate args beyond `String()` conversion.

3. **Allowlist uses `path.basename(p.command)`**, so `/usr/local/bin/npx` passes if `npx` is in the allowlist (`stdioSseBridge.js:115`). A relative path like `./npx` would fail.

4. **Tool result filtering mutates JSON-RPC result content arrays in-place** (`stdioSseBridge.js:89`). It only touches items where `type === "text"`. Non-text content passes through unmodified.

5. **Registry pagination loop caps at 20 pages** (`cowork-mcp-registry/route.js:27`). No backoff or rate-limit handling.

6. **Probe aborts the entire flow on timeout** (`cowork-mcp-tools/route.js:16`), including the `notifications/initialized` fire-and-forget step. The abort controller is shared across all three requests.

7. **Dedupe rules are hardcoded and regex-based** (`toolDeduper.js:6`). The Exa rule matches exact strings; the Browser rule matches `/^mcp__browsermcp__/`. No runtime configuration.

8. **Cursor protobuf `formatToolName` double-underscore convention** (`cursorProtobuf.js:244`): `mcp__server__tool` is parsed as server + tool. Single underscores after `mcp_` are treated as `mcp_{server}_{tool}`. A name like `mcp_foo` becomes server `foo`, selectedTool empty string (fallbacks to `"tool"`).

9. **Bridge re-spawns on every new session if the previous process exited** (`stdioSseBridge.js:141`). There is no health check before re-spawn; it simply checks `entry.proc.exitCode === null`.

10. **Settings DELETE writes an empty JSON object `{}` to the config file** (`cowork-settings/route.js:403`) rather than deleting the file. This leaves an empty managedMcpServers array absent, which the reader treats as "no config."

---

## Go-port Considerations

- Bridge spawns need `os/exec` with `StdinPipe`, `StdoutPipe`, `StderrPipe`; use `cmd.Wait` + goroutine scanners for newline-delimited JSON-RPC.
- SSE streaming maps to `http.Flush` or `text/event-stream` writers; session management needs an in-memory map with mutex or `sync.Map`.
- Tool dedupe rules are small and static; a slice of structs with `regexp.Regexp` fields works.
- Cursor protobuf encoding is wire-format specific; port the field/tag constants and `encodeField` helpers directly, or skip if Cursor support is not planned.
- Plugin allowlist should use `filepath.Base` on the command path, matching the JS `path.basename` semantics.
- The reference's `globalThis` bridge store is equivalent to a package-level `var bridges = map[string]*Bridge{}` protected by a `sync.RWMutex`.
