# MCP Parity Matrix: 9router (reference) → g0router

Reference SHA: `827e5c3` (frozen).

---

## Behavior Matrix

| ID | Behavior | Evidence (file:line) | g0router status | Notes |
|---|---|---|---|---|
| PAR-MCP-001 | SSE endpoint exposes stdio MCP plugins over HTTP | `src/app/api/mcp/[plugin]/sse/route.js:6` | PARTIAL | CLIENT SSE/message transport shipped w7-mcp-2 (`internal/mcp/sse.go`: `parseSSEFrame`/`parseSSEDataFrames` + `postMessage` + integration-only `Stream`); the SERVER `/api/mcp/{plugin}/sse` route + `endpoint`-event emission are w7-mcp-3. |
| PAR-MCP-002 | Message POST endpoint forwards JSON-RPC to stdio child | `src/app/api/mcp/[plugin]/message/route.js:7` | PARTIAL | CLIENT POST-to-`/message` transport shipped w7-mcp-2 (`internal/mcp/sse.go:postMessage`, expects 202); the SERVER route is w7-mcp-3. |
| PAR-MCP-003 | Stdio<->SSE bridge spawns one child process per plugin on demand | `src/lib/mcp/stdioSseBridge.js:146` | HAVE | `internal/mcp/launcher.go` `StartStdio` spawns one process per plugin via the injectable `ProcessRunner` and registers one `Bridge` per plugin in `Launcher.bridges`. Real `os/exec` spawn is INTEGRATION-ONLY (`internal/mcp/process.go` `osProcessRunner`, never invoked by unit tests — RCE risk, ESC-SPAWN); the registry/one-bridge-per-plugin logic is fake-runner unit-tested (`launcher_test.go`). |
| PAR-MCP-004 | Plugin command allowlist restricts spawnable binaries | `src/shared/constants/coworkPlugins.js:67` | HAVE | `internal/mcp/allowlist.go` `allowedMCPCommands` = `npx,node,uvx,python,python3,bunx,bun`; PURE `isAllowedCommand`, exhaustively unit-tested incl. every rejection (`allowlist_test.go`). Hardened over 9router's raw path.basename (rejects relative paths + shell metacharacters — ESC-ALLOWLIST). |
| PAR-MCP-005 | Custom plugin registration validates command against allowlist | `src/lib/mcp/stdioSseBridge.js:119` | HAVE | `internal/mcp/launcher.go` `StartStdio` calls `isAllowedCommand` and returns `ErrCommandNotAllowed` BEFORE any spawn; `launcher_test.go` asserts the fake runner's `Start` is NEVER called for a rejected command. |
| PAR-MCP-006 | Custom plugins persist to disk and survive restart | `src/lib/mcp/stdioSseBridge.js:130` | HAVE | `internal/store/mcpinstances.go` `UpsertMCPClient`/`CreateMCPClient` persist custom plugins to the SQLite `mcp_clients` table (survives restart — DB, not `customPlugins.json`; ESC-PERSIST). |
| PAR-MCP-007 | JSON-RPC stdout frames broadcast to all active SSE sessions | `src/lib/mcp/stdioSseBridge.js:151` | HAVE | `internal/mcp/bridge.go` PURE `splitFrames` + `Bridge.onFrame`/`broadcast` over a `sessions map[string]SessionSink`; unit-tested with canned frames + fake sinks (`bridge_test.go`). |
| PAR-MCP-008 | Tool result text filtering: drop noise nodes, collapse repeated siblings, hard-truncate at 50K chars | `src/lib/mcp/stdioSseBridge.js:20` | HAVE | `internal/mcp/filter.go` PURE `smartFilterText` (drops `generic` + empty `text` lines, collapses repeated role-prefixed siblings, `maxToolResultChars = 50_000` hard cap); unit-tested for drop/collapse/truncate/clean-unchanged (`filter_test.go`). Observable behavior pinned; ESC-FILTER. |
| PAR-MCP-009 | MCP server probe performs initialize + notifications/initialized + tools/list handshake | `src/app/api/cli-tools/cowork-mcp-tools/route.js:9` | HAVE | `internal/mcp/probe.go:Run` — three-step handshake over an injected `*http.Client`; FULLY unit-tested via a fake `http.RoundTripper` (`probe_test.go:TestProbeFullHandshakeJSON`). |
| PAR-MCP-010 | Probe uses MCP-Protocol-Version header 2025-06-18 | `src/app/api/cli-tools/cowork-mcp-tools/route.js:13` | HAVE | `internal/mcp/probe.go` const `mcpProtocolVersion="2025-06-18"`, header set on every `post`; asserted on all 3 captured requests (`probe_test.go`). |
| PAR-MCP-011 | Probe reads mcp-session-id from initialize response and replays it on tools/list | `src/app/api/cli-tools/cowork-mcp-tools/route.js:34` | HAVE | `internal/mcp/probe.go:Run` reads `mcp-session-id` from initialize + replays on initialized + tools/list; asserted in `probe_test.go`. |
| PAR-MCP-012 | Probe parses SSE responses for tools/list | `src/app/api/cli-tools/cowork-mcp-tools/route.js:60` | HAVE | `internal/mcp/probe.go:extractTools` + PURE `parseSSEDataFrames` (`sse.go`) finds `id==2` result; `probe_test.go:TestProbeToolsListSSEParse`. |
| PAR-MCP-013 | Probe returns `requiresAuth: true` on 401/403 | `src/app/api/cli-tools/cowork-mcp-tools/route.js:28` | HAVE | `internal/mcp/probe.go:Run` → `RequiresAuth=true` on 401/403 at init + tools/list; `probe_test.go:TestProbeInitRequiresAuth`/`TestProbeToolsListRequiresAuth`. |
| PAR-MCP-014 | Anthropic MCP registry client fetches servers with pagination | `src/app/api/cli-tools/cowork-mcp-registry/route.js:24` | HAVE | `internal/mcp/registry.go:fetchAll` — ≤20 pages, limit=500, cursor follow; multi-page fake transport (`registry_test.go:TestRegistryPaginationFollowsCursor`/`TestRegistryStopsAtMaxPages`). |
| PAR-MCP-015 | Registry cache TTL is 1 hour | `src/app/api/cli-tools/cowork-mcp-registry/route.js:7` | HAVE | `internal/mcp/registry.go` const `registryCacheTTL=1*time.Hour`; cache hit/miss/force tested via an injected `now` clock — NO real sleep (`registry_test.go:TestRegistryCacheHitMissForce`). |
| PAR-MCP-016 | Registry filters out claude.com and api.anthropic.com mediated servers | `src/app/api/cli-tools/cowork-mcp-registry/route.js:16` | HAVE | PURE `internal/mcp/registry.go:isDirectConnect` rejects `mcp.claude.com`, `api.anthropic.com/mcp`, `{`/`<`, non-https; `registry_test.go:TestIsDirectConnectAcceptReject`. |
| PAR-MCP-017 | Registry excludes tenant-required entries | `src/app/api/cli-tools/cowork-mcp-registry/route.js:37` | HAVE | `internal/mcp/registry.go:mapRegistryItem` skips entries with `requiredFields.length>0`; `registry_test.go:TestRegistrySkipsDirectConnectAndRequiredFields`. |
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
| PAR-MCP-032 | Database has no MCP tables | `internal/store/migrate.go:11` | HAVE | `internal/store/migrate.go` `tables` slice now declares additive `mcp_clients`, `mcp_instances`, `mcp_oauth_accounts`, `mcp_oauth_flows` (`CREATE TABLE IF NOT EXISTS`; additive-only). |
| PAR-MCP-033 | No MCP API handlers | `api/` (empty except `.DS_Store`) | PARTIAL | Store + OAuth-account persistence shipped w7-mcp-1 (`internal/store/mcpinstances.go`, `internal/store/mcpoauth.go`); the admin HANDLERS (`internal/admin/mcp.go`, `mcpoauth.go`) are w7-mcp-3. |
| PAR-MCP-034 | No MCP CLI commands | `internal/cli/` (does not exist) | MISSING | Directory absent. No `mcp add`, `mcp auth start`, `mcp auth complete`, `mcp accounts`, `mcp tools`, `mcp remove`. |
| PAR-MCP-035 | No MCP store layer | `internal/store/` (no mcp files) | HAVE | `internal/store/mcpinstances.go` (clients/instances CRUD + status) + `internal/store/mcpoauth.go` (OAuth accounts + flows; tokens/verifier `*_enc` at rest via `s.cipher`). |
| PAR-MCP-036 | No MCP launcher | `internal/mcp/` (no launcher files) | HAVE | `internal/mcp/launcher.go` `Launcher` with stdio (`StartStdio`) + http/sse (`StartHTTP`/`StartSSE`) modes behind an injectable `ProcessRunner` (`runner.go`); `NewLauncher`/`SetRunner` mirror the tunnel `Runner`/`SetRunner` seam. Real spawn INTEGRATION-ONLY (`process.go`, ESC-SPAWN). |
| PAR-MCP-037 | MCP OAuth account engine | `internal/mcp/oauth.go` | HAVE | `internal/mcp/oauth.go:Engine` — PKCE start/complete/refresh REUSING `auth.GeneratePKCE` (additive helper wrapping the in-tree PKCE primitives; no crypto copied), RFC 9728 protected-resource-metadata + RFC 8414 auth-server-metadata discovery, token exchange/refresh over the SHIPPED `mcpoauth` store (tokens `*_enc`, MASKED in returns); FULLY unit-tested via a fake transport + temp store (`oauth_test.go`). |
| PAR-MCP-038 | MCP health monitor | `internal/mcp/healthmonitor.go` | HAVE | PURE `accountHealth` + `accountsNeedingRefresh` unit-tested (`healthmonitor_test.go`); the periodic ticker `Run`/`sweep` loop is integration-only. |
| PAR-MCP-039 | MCP discovery / compact injection | `internal/mcp/discovery.go` | HAVE | `internal/mcp/discovery.go` — `toolsCache` (get/set, keyed by instance) + PURE `buildCompactManifest`; `discovery_test.go`. |
| PAR-MCP-040 | No MCP agent loop | `internal/mcp/` (no agent files) | MISSING | No `agent.go`. No multi-turn tool execution loop. |
| PAR-MCP-041 | No MCP UI pages | `ui/src/` (no MCP pages) | MISSING | grep finds no `mcp` or `MCP` references in `ui/src/`. Dashboard has no MCP instance manager. |
| PAR-MCP-042 | Cowork feature disabled globally in reference | `src/app/api/mcp/[plugin]/sse/route.js:7`, `src/app/api/mcp/[plugin]/message/route.js:8`, `src/app/api/cli-tools/cowork-settings/route.js:314` | EXTRA | All MCP bridge and settings routes return 403 with comment "Cowork disabled: MCP stdio bridge spawns arbitrary processes (RCE risk)." g0router never had this feature enabled or disabled. |
| PAR-MCP-043 | Default plugins: Exa (HTTP, no auth), Tavily (HTTP, OAuth) | `src/shared/constants/coworkPlugins.js:3` | HAVE | `internal/mcp/defaults.go` `DefaultPlugins()` returns Exa (`transport:http, oauth:false`) + Tavily (`transport:http, oauth:true`) as Go values; unit-tested (`defaults_test.go`). Consumed by w7-mcp-2/3 (no live HTTP here). |
| PAR-MCP-044 | Local stdio plugin: browsermcp via npx | `src/shared/constants/coworkPlugins.js:26` | HAVE | `internal/mcp/defaults.go` browsermcp definition: `transport:stdio, command:npx, args:["-y","@browsermcp/mcp@latest"]`, 10 tool names; test asserts the command is allowlisted + counts match. |
| PAR-MCP-045 | `buildManagedMcpServers` strips `{name}-` prefixes idempotently | `src/shared/constants/coworkPlugins.js:48` | MISSING | While-loop removes repeated prefixes, then emits bare + single-prefixed variants in `toolPolicy`. g0router has no equivalent. |
| PAR-MCP-046 | Settings GET strips `{name}-` prefixes when returning plugin tool names | `src/app/api/cli-tools/cowork-settings/route.js:288` | MISSING | Same prefix-stripping logic as buildManagedMcpServers. g0router has no settings API. |
| PAR-MCP-047 | Settings GET prefers default `toolNames` over stored `toolPolicy` keys | `src/app/api/cli-tools/cowork-settings/route.js:297` | MISSING | `const toolNames = def && Array.isArray(def.toolNames) ? def.toolNames : Array.from(bare);`. g0router has no plugin settings. |
| PAR-MCP-048 | Settings POST sanitizes custom plugin names to `[^a-zA-Z0-9_-]` and truncates to 64 chars | `src/app/api/cli-tools/cowork-settings/route.js:339` | MISSING | `String(p.name || "").replace(/[^a-zA-Z0-9_-]/g, "").slice(0, 64)`. g0router has no custom plugin input. |
| PAR-MCP-049 | Settings POST filters custom plugins by allowlist before registration | `src/app/api/cli-tools/cowork-settings/route.js:337` | MISSING | `.filter((p) => ALLOWED_MCP_COMMANDS.has(path.basename(p.command)))`. g0router has no allowlist filtering. |
| PAR-MCP-050 | Settings DELETE resets Cowork config, skip approvals, and cleans legacy 1p entries | `src/app/api/cli-tools/cowork-settings/route.js:396` | MISSING | Writes empty object to config, calls `writeSkipApprovals([])` and `cleanup1pLegacy()`. g0router has no reset flow. |
| PAR-MCP-051 | `isRunning` helper checks if bridge process is alive | `src/lib/mcp/stdioSseBridge.js:193` | HAVE | `internal/mcp/bridge.go` `Bridge.IsRunning` delegates to `Process.IsRunning` (not killed && exit code unobserved); `Launcher.IsRunning(name)` reports per-plugin liveness; unit-tested via the fake process (`bridge_test.go`/`launcher_test.go`). |
| PAR-MCP-052 | Bridge handles child stderr by logging to console | `src/lib/mcp/stdioSseBridge.js:165` | HAVE | `internal/mcp/bridge.go` `Bridge.onStderr` forwards each child stderr line to a `SetOnStderr` callback; the real stderr pipe drain is in `process.go` (INTEGRATION-ONLY); the callback wiring is unit-tested (`bridge_test.go`). |
| PAR-MCP-053 | Bridge handles child exit by deleting store entry | `src/lib/mcp/stdioSseBridge.js:167` | HAVE | `internal/mcp/bridge.go` `Bridge.onExit` fires the exit callback; `Launcher.StartStdio` wires it to `removeBridge(name)` so the registry entry is deleted on exit; unit-tested via a fake exit (`launcher_test.go` `TestLauncherExitRemovesBridge`). |
| PAR-MCP-054 | Broken SSE sessions silently ignored on send failure | `src/lib/mcp/stdioSseBridge.js:160` | HAVE | `internal/mcp/bridge.go` `Bridge.broadcast` drops a session sink that returns an error and continues the broadcast loop; unit-tested (`bridge_test.go` `TestBridgeFailingSinkDroppedWithoutAbort`). |
| PAR-MCP-055 | SSE handshake sends `endpoint` event with sessionId query param | `src/app/api/mcp/[plugin]/sse/route.js:22` | PARTIAL | CLIENT side shipped w7-mcp-2 (`internal/mcp/sse.go:parseSSEFrame` reads the `endpoint` event; `Stream` consumes it); the SERVER `endpoint`-event emission is w7-mcp-3. |
| PAR-MCP-056 | SSE response sets no-cache, keep-alive, X-Accel-Buffering headers | `src/app/api/mcp/[plugin]/sse/route.js:30` | PARTIAL | The client SSE reader honors `text/event-stream` (`internal/mcp/sse.go:Stream`); the SERVER header emission is w7-mcp-3. |
| PAR-MCP-057 | Registry dedupes by URL | `src/app/api/cli-tools/cowork-mcp-registry/route.js:57` | HAVE | `internal/mcp/registry.go:fetchAll` keeps the first occurrence per URL via a `seen` set; `registry_test.go:TestRegistryDedupesByURL`. |
| PAR-MCP-058 | Probe timeout is 8 seconds | `src/app/api/cli-tools/cowork-mcp-tools/route.js:5` | HAVE | `internal/mcp/probe.go` const `probeTimeout=8*time.Second` via `context.WithTimeout`; tested via a short ctx + a blocking fake transport — NO real 8s sleep (`probe_test.go:TestProbeTimeout`). |
| PAR-MCP-059 | Probe catches AbortError and maps to `"timeout"` | `src/app/api/cli-tools/cowork-mcp-tools/route.js:78` | HAVE | `internal/mcp/probe.go:errResult` maps `context.DeadlineExceeded` → `Error:"timeout"`; `probe_test.go:TestProbeTimeout`. |
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
