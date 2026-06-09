# BF-MCP Parity Matrix

Reference: Bifrost SHA `ca21298` (frozen clone at `/Users/heitor/Developer/github.com/bloodf/_refs/bifrost`).
Target: g0router (`/Users/heitor/Developer/github.com/bloodf/g0router`).

---

## Behavior Matrix

| ID | Behavior | Evidence (file:line) | g0router status | Notes |
|---|---|---|---|---|
| PAR-BF-MCP-001 | MCP client mode: connect to upstream tool servers (HTTP, SSE, STDIO, InProcess) | `core/schemas/mcp.go:441-444` (connection-type constants); `core/mcp/mcp.go:78-177` (constructor wiring AddClient for each config) | MISSING | No client manager, no transport launcher |
| PAR-BF-MCP-002 | MCP server mode: expose Bifrost as MCP server over HTTP JSON-RPC + SSE | `transports/bifrost-http/handlers/mcpserver.go:85-88` (routes `/mcp` POST+GET) | MISSING | No `/mcp` endpoint exists |
| PAR-BF-MCP-003 | MCP server mode: global MCP server (un-scoped tools) | `transports/bifrost-http/handlers/mcpserver.go:58-62` (globalMCPServer); `core/mcp/mcp.go:39` (server field) | MISSING | No server instance |
| PAR-BF-MCP-004 | MCP server mode: per-virtual-key MCP server with lazy creation | `transports/bifrost-http/handlers/mcpserver.go:43` (vkMCPServers map); `289-308` (SyncVKMCPServer); `583-595` (ensureVKMCPServer) | MISSING | No VK-scoped server logic |
| PAR-BF-MCP-005 | Transport: HTTP (MCPConnectionTypeHTTP) | `core/schemas/mcp.go:441` | MISSING | Transport string is free-form in g0router schema only |
| PAR-BF-MCP-006 | Transport: SSE (MCPConnectionTypeSSE) | `core/schemas/mcp.go:443` | MISSING | No SSE endpoint or stream writer |
| PAR-BF-MCP-007 | Transport: STDIO (MCPConnectionTypeSTDIO) with command/args/env | `core/schemas/mcp.go:442`; `448-452` (MCPStdioConfig) | MISSING | Schema has `Command`/`Args`/`Env` on MCPInstance but no launcher |
| PAR-BF-MCP-008 | Transport: InProcess (MCPConnectionTypeInProcess) | `core/schemas/mcp.go:444`; `302` (InProcessServer field) | MISSING | No in-process server wiring |
| PAR-BF-MCP-009 | Auth type: none (MCPAuthTypeNone) | `core/schemas/mcp.go:271` | MISSING | No auth-type enum exists in g0router |
| PAR-BF-MCP-010 | Auth type: headers (MCPAuthTypeHeaders) server-level static headers | `core/schemas/mcp.go:272`; `292` (Headers map) | MISSING | Schema lacks Headers field on client config |
| PAR-BF-MCP-011 | Auth type: oauth (MCPAuthTypeOauth) server-level OAuth 2.0 | `core/schemas/mcp.go:273`; `288-290` (OauthConfigID, OauthClientID, OauthClientSecret) | MISSING | No OAuth config schema or flow |
| PAR-BF-MCP-012 | Auth type: per_user_oauth (MCPAuthTypePerUserOauth) | `core/schemas/mcp.go:274` | MISSING | No per-user auth model |
| PAR-BF-MCP-013 | Auth type: per_user_headers (MCPAuthTypePerUserHeaders) with required key schema | `core/schemas/mcp.go:275`; `300` (PerUserHeaderKeys) | MISSING | No per-user header schema |
| PAR-BF-MCP-014 | Per-user auth modes: user, vk, session identity dimensions | `core/schemas/mcp_headers.go:38-45` (MCPHeadersUserCredential with UserID, VirtualKeyID, SessionID) | MISSING | No identity-mode enum or credential store |
| PAR-BF-MCP-015 | Inline auth flow: MCPAuthRequiredError with Kind=oauth (AuthorizeURL) | `core/schemas/mcp.go:44-46` (constants); `63-81` (struct fields) | MISSING | No error type for auth-required |
| PAR-BF-MCP-016 | Inline auth flow: MCPAuthRequiredError with Kind=headers (SubmitURL, RequiredHeaderKeys) | `core/schemas/mcp.go:44-46`; `78-80` | MISSING | No header-submission flow |
| PAR-BF-MCP-017 | Tool filtering: per-client allow-list ToolsToExecute (["*"]=all, nil=[]=deny) | `core/schemas/mcp.go:303-307` (field + semantics comments) | MISSING | No WhiteList type or filtering logic |
| PAR-BF-MCP-018 | Tool filtering: auto-execute list ToolsToAutoExecute subset of ToolsToExecute | `core/schemas/mcp.go:309-315` | MISSING | No agent loop |
| PAR-BF-MCP-019 | Tool filtering: per-virtual-key executeOnlyTools with wildcard `clientName-*` and `clientName-toolName` | `transports/bifrost-http/handlers/mcpserver.go:471-520` (fetchToolsForVK) | MISSING | No VK tool assignment table or logic |
| PAR-BF-MCP-020 | Tool filtering: AllowOnAllVirtualKeys flag on client config | `core/schemas/mcp.go:322` | MISSING | No VK-scoped filtering exists |
| PAR-BF-MCP-021 | Tool discovery: ping vs listTools health check fallback (IsPingAvailable) | `core/schemas/mcp.go:316` (IsPingAvailable) | MISSING | Schema lacks this field |
| PAR-BF-MCP-022 | Tool sync interval: per-client override with flexible duration parsing (int=nanoseconds or Go duration string) | `core/schemas/mcp.go:166-208` (MCPConfig.UnmarshalJSON); `373-422` (parseFlexibleDurationField) | MISSING | No duration parsing or sync scheduler |
| PAR-BF-MCP-023 | Tool execution timeout: bare integer treated as seconds (not nanoseconds) | `core/schemas/mcp.go:210-252` (MCPToolManagerConfig.UnmarshalJSON) | MISSING | No tool manager config exists |
| PAR-BF-MCP-024 | Agent loop: iterative tool execution up to MaxAgentDepth for Chat API | `core/mcp/mcp.go:251-273` (CheckAndExecuteAgentForChatRequest) | MISSING | No agent loop |
| PAR-BF-MCP-025 | Agent loop: iterative tool execution up to MaxAgentDepth for Responses API | `core/mcp/mcp.go:303-324` (CheckAndExecuteAgentForResponsesRequest) | MISSING | No agent loop |
| PAR-BF-MCP-026 | Health monitor: automatic reconnect on startup failure with retention in Disconnected state | `core/mcp/mcp.go:142-170` (constructor: retain entry + start HealthMonitor on AddClient error) | MISSING | No health monitor |
| PAR-BF-MCP-027 | Credential store interface: ConnectionHeaders, RequestHeaders, RequiresPerCallConnection | `core/schemas/mcp.go:94-140` (MCPCredentialStore interface) | MISSING | No credential store abstraction |
| PAR-BF-MCP-028 | Per-user header credentials table with encrypted headers_json | `framework/configstore/tables/mcp_per_user_headers.go` (schema + GORM hooks) | MISSING | No DB table for MCP header credentials |
| PAR-BF-MCP-029 | Per-user header flow table with TTL expiry | `framework/configstore/tables/mcp_per_user_headers.go` (mcp_per_user_header_flows schema) | MISSING | No flow table |
| PAR-BF-MCP-030 | OAuth token table for per-user MCP OAuth (oauth_user_tokens) | `transports/bifrost-http/handlers/mcp_sessions.go:230` (ListOauthUserTokens) | MISSING | No OAuth token table |
| PAR-BF-MCP-031 | OAuth pending flow table (oauth_user_sessions) | `transports/bifrost-http/handlers/mcp_sessions.go:233` (ListPendingOauthUserSessions) | MISSING | No OAuth flow table |
| PAR-BF-MCP-032 | MCP client config table (config_mcp_clients) with encryption at rest | `framework/configstore/tables/mcp.go` (schema + BeforeSave/AfterFind/AfterDelete hooks) | MISSING | No MCP client table in migrations |
| PAR-BF-MCP-033 | VK-MCP assignment table (virtual_key_mcp_configs) | `transports/bifrost-http/handlers/mcp.go:172-176` (GetVirtualKeyMCPConfigsByMCPClientIDs); `1215-1242` (transactional diffing) | MISSING | No VK-MCP junction table |
| PAR-BF-MCP-034 | API: GET /api/mcp/clients (paginated, with VK config merge + OAuth batch fetch) | `transports/bifrost-http/handlers/mcp.go:70`; `95-294` (getMCPClients + getMCPClientsPaginated) | MISSING | `api/` directory empty except `.DS_Store` |
| PAR-BF-MCP-035 | API: POST /api/mcp/client (create with auth branching) | `transports/bifrost-http/handlers/mcp.go:72`; `387-785` (addMCPClient) | MISSING | No MCP handler routes |
| PAR-BF-MCP-036 | API: PUT /api/mcp/client/{id} (PATCH semantics, immutable fields guarded, redacted merge) | `transports/bifrost-http/handlers/mcp.go:73`; `787-1336` (updateMCPClient) | MISSING | No update endpoint |
| PAR-BF-MCP-037 | API: DELETE /api/mcp/client/{id} (DB-first delete then memory removal) | `transports/bifrost-http/handlers/mcp.go:74`; `1338-1365` (deleteMCPClient) | MISSING | No delete endpoint |
| PAR-BF-MCP-038 | API: POST /api/mcp/client/{id}/reconnect (rejects disabled and per-user types) | `transports/bifrost-http/handlers/mcp.go:75`; `296-334` (reconnectMCPClient) | MISSING | No reconnect endpoint |
| PAR-BF-MCP-039 | API: POST /api/mcp/client/{id}/complete-oauth (distinguishes create vs update flow) | `transports/bifrost-http/handlers/mcp.go:76`; `1483-1705` (completeMCPClientOAuth) | MISSING | No OAuth completion endpoint |
| PAR-BF-MCP-040 | API: GET /api/mcp/sessions (cross-table merge with de-dup + pagination) | `transports/bifrost-http/handlers/mcp_sessions.go:37`; `196-327` (list) | MISSING | No sessions API |
| PAR-BF-MCP-041 | API: POST /api/mcp/sessions/{id}/reauth (OAuth or header branch, identity gate) | `transports/bifrost-http/handlers/mcp_sessions.go:38`; `405-560` (reauth + reauthHeaderCredential) | MISSING | No reauth endpoint |
| PAR-BF-MCP-042 | API: DELETE /api/mcp/sessions/{id} (race-protected: delete flows before token/credential) | `transports/bifrost-http/handlers/mcp_sessions.go:39`; `572-667` (revoke) | MISSING | No revoke endpoint |
| PAR-BF-MCP-043 | API: GET /api/oauth/per-user/flows/{id} (flow detail with HasActiveToken hint) | `transports/bifrost-http/handlers/mcp_sessions.go:40`; `690-762` (flowDetail) | MISSING | No flow detail endpoint |
| PAR-BF-MCP-044 | API: GET /api/oauth/per-user/flows/{id}/start (rebuilds upstream URL, 410 on stale) | `transports/bifrost-http/handlers/mcp_sessions.go:41`; `764-805` (flowStart) | MISSING | No flow start endpoint |
| PAR-BF-MCP-045 | API: per-user headers flow detail (`/api/mcp/per-user-headers/flows/{id}`) | `transports/bifrost-http/handlers/mcp_per_user_headers.go` (flowDetail) | MISSING | No per-user headers handler |
| PAR-BF-MCP-046 | API: per-user headers flow submit (`PUT /api/mcp/per-user-headers/flows/{id}`) with merge + verify | `transports/bifrost-http/handlers/mcp_per_user_headers.go` (flowSubmit) | MISSING | No submit endpoint |
| PAR-BF-MCP-047 | API: per-user headers credential revoke (`DELETE /api/mcp/per-user-headers/credential/{id}`) | `transports/bifrost-http/handlers/mcp_per_user_headers.go` (revoke) | MISSING | No credential revoke endpoint |
| PAR-BF-MCP-048 | API: POST /v1/mcp/tool/execute?format=chat|responses (direct tool execution) | `transports/bifrost-http/handlers/mcpinference.go:70-71`; `executeTool` | MISSING | No direct tool execution endpoint |
| PAR-BF-MCP-049 | Validation: ToolsToAutoExecute must be subset of ToolsToExecute | `transports/bifrost-http/handlers/mcp.go:1394-1417` (validateToolsToAutoExecute) | MISSING | No validation logic |
| PAR-BF-MCP-050 | Validation: per_user_header_keys canonicalized (lowercase+trim), no empties, no duplicates | `transports/bifrost-http/handlers/mcp.go:447-461`; `935-945` | MISSING | No canonicalization logic |
| PAR-BF-MCP-051 | Validation: OAuth credential rotation explicitly disabled (400 on req.OauthConfig in update) | `transports/bifrost-http/handlers/mcp.go:948-952` | MISSING | No OAuth update path to guard |
| PAR-BF-MCP-052 | VK resolution for MCP server: x-bf-vk > Authorization Bearer vk_* > x-api-key vk_* | `transports/bifrost-http/handlers/mcpserver.go:597-618` (getVKFromRequest) | MISSING | No VK resolution for MCP |
| PAR-BF-MCP-053 | SSE heartbeat: `: ping\n\n` every 15s to detect disconnect via reader.Send() | `transports/bifrost-http/handlers/mcpserver.go:29` (sseHeartbeatInterval); `254-266` | MISSING | No SSE stream |
| PAR-BF-MCP-054 | Trace completion deferred for SSE to avoid fasthttp body materialization deadlock | `transports/bifrost-http/handlers/mcpserver.go:145` (BifrostContextKeyDeferTraceCompletion) | MISSING | No SSE/tracing integration |
| PAR-BF-MCP-055 | Tool execution audit: ToolPricing map per client + MCPCatalog lookup by "server/tool_name" | `core/schemas/mcp.go:318` (ToolPricing); `framework/mcpcatalog/main.go` (pricing map) | MISSING | No pricing catalog |
| PAR-BF-MCP-056 | Code mode binding level: server vs tool VFS binding | `core/schemas/mcp.go:259-265` (CodeModeBindingLevel) | MISSING | No code mode |
| PAR-BF-MCP-057 | Disable auto tool inject flag | `core/schemas/mcp.go:217` (DisableAutoToolInject) | MISSING | No tool injection toggle |
| PAR-BF-MCP-058 | Credential sweep worker: orphaned header credential deletion + expired flow cleanup | `framework/mcp_headers/sweep.go` (CredentialSweepWorker) | MISSING | No sweep worker |
| PAR-BF-MCP-059 | Temp token auth for per-user header flows (optional, skipped for user mode) | `framework/mcp_headers/main.go` (InitiateUserSubmissionFlow temp-token minting) | MISSING | No temp token service for MCP |
| PAR-BF-MCP-060 | Plugin pipeline provider/release for nested tool calls in code mode | `core/mcp/mcp.go:50-51` (pluginPipelineProvider/releasePluginPipeline fields) | MISSING | No plugin pipeline integration |
| PAR-BF-MCP-061 | DB rollback on in-memory AddMCPClient failure (create path) | `transports/bifrost-http/handlers/mcp.go:519-528` (DB delete on manager failure) | MISSING | No DB + memory two-phase create |
| PAR-BF-MCP-062 | DB rollback on in-memory UpdateMCPClient failure (update path) | `transports/bifrost-http/handlers/mcp.go:1126-1137` (oldDBConfig rollback) | MISSING | No update rollback |
| PAR-BF-MCP-063 | Discovered tool key migration on client rename (prefix replacement) | `transports/bifrost-http/handlers/mcp.go:1066-1083` (old prefix -> newPrefix + suffix) | MISSING | No discovered tool persistence |
| PAR-BF-MCP-064 | Per-user header credentials flipped to `needs_update` when schema adds keys | `transports/bifrost-http/handlers/mcp.go:1151-1157` (MarkMCPPerUserHeaderCredentialsNeedsUpdate) | MISSING | No credential state machine |
| PAR-BF-MCP-065 | VK credential reconciliation on assignment change (enterprise no-op in OSS) | `transports/bifrost-http/handlers/mcp.go:1320-1330` (ReconcileOauthAfterMCPChange + ReconcileMCPHeadersAfterMCPChange) | MISSING | No reconciliation hooks |
| PAR-BF-MCP-066 | TLS config with InsecureSkipVerify and CACertPEM (env-var aware) | `core/schemas/mcp.go:454-459` (MCPTLSConfig) | MISSING | No TLS config schema for MCP |
| PAR-BF-MCP-067 | MCP client state machine: Connected, Disconnected, Error, PendingTools, Disabled | `core/schemas/mcp.go:479-487` (MCPConnectionState constants) | MISSING | Status is free-form string in g0router schema |
| PAR-BF-MCP-068 | Retry loop on UpdateMCPClient blocked by in-flight reconnect (3 attempts, 500ms) | `transports/bifrost-http/handlers/mcp.go:1442-1460` (updateMCPClientWithRetry) | MISSING | No retry logic |
| PAR-BF-MCP-069 | Retry loop on UpdateMCPClientConnection blocked by in-flight reconnect | `transports/bifrost-http/handlers/mcp.go:1463-1481` (updateMCPClientConnectionWithRetry) | MISSING | No retry logic |
| PAR-BF-MCP-070 | Headers merge: preserve raw values when incoming equals redacted placeholder | `transports/bifrost-http/handlers/mcp.go:1423-1439` (mergeMCPHeaders) | MISSING | No header redaction/merge logic |
| PAR-BF-MCP-071 | AllowedExtraHeaders whitelist for request-level header forwarding | `core/schemas/mcp.go:301` (AllowedExtraHeaders WhiteList) | MISSING | No extra-header allowlist |
| PAR-BF-MCP-072 | Error: ErrMCPReconnectNotApplicable for per-user auth types | `core/schemas/mcp.go:34-38` | MISSING | No sentinel error for reconnect |
| PAR-BF-MCP-073 | Error: OAuth2 flow errors (config not found, token expired, refresh failed, etc.) | `core/schemas/mcp.go:24-33` (ErrOAuth2* vars) | MISSING | No OAuth error types |
| PAR-BF-MCP-074 | MCPAuthRequiredError implements error interface (returns Message) | `core/schemas/mcp.go:83-85` | MISSING | No auth-required error type |
| PAR-BF-MCP-075 | mcp-go server integration for JSON-RPC message handling | `transports/bifrost-http/handlers/mcpserver.go:113` (mcpServer.HandleMessage) | MISSING | No mcp-go dependency or server |
| PAR-BF-MCP-076 | mcp-go client integration for upstream connections | `core/schemas/mcp.go:19` (import `github.com/mark3labs/mcp-go/client`) | MISSING | No mcp-go client usage |
| PAR-BF-MCP-077 | Tool annotations mapping (Title, ReadOnlyHint, DestructiveHint, IdempotentHint, OpenWorldHint) | `transports/bifrost-http/handlers/mcpserver.go:406-415` | MISSING | No annotation schema or mapping |
| PAR-BF-MCP-078 | IsCodeModeClient flag on client config | `core/schemas/mcp.go:282` | MISSING | No code mode flag |
| PAR-BF-MCP-079 | ConfigHash field for reconciliation | `core/schemas/mcp.go:320` | MISSING | No hash tracking |
| PAR-BF-MCP-080 | ConnectionString stored as *EnvVar (encrypted at rest) | `core/schemas/mcp.go:284` | MISSING | No EnvVar type; no encryption |

---

## Data Models

### Bifrost (reference)

**`config_mcp_clients`** (`framework/configstore/tables/mcp.go`)
- `id` uint PK
- `client_id` varchar(255) unique not null
- `name` varchar(255) unique not null
- `is_code_mode_client` bool default false
- `connection_type` varchar(20) not null
- `connection_string` text (encrypted EnvVar)
- `stdio_config_json` text
- `tls_config_json` text
- `tools_to_execute_json` text
- `tools_to_auto_execute_json` text
- `headers_json` text (encrypted)
- `allowed_extra_headers_json` text
- `is_ping_available` bool default true
- `tool_pricing_json` text
- `tool_sync_interval` int (seconds)
- `discovered_tools_json` text
- `tool_name_mapping_json` text
- `auth_type` varchar(20) default 'headers'
- `oauth_config_id` varchar(255) FK oauth_configs.id CASCADE
- `per_user_header_keys_json` text
- `allow_on_all_virtual_keys` bool default false
- `disabled` bool default false
- `config_hash` varchar(255)
- `encryption_status` varchar(20) default 'plain_text'
- `created_at` / `updated_at` datetime indexed

**`mcp_per_user_header_flows`** (`framework/configstore/tables/mcp_per_user_headers.go`)
- `id` varchar(255) PK
- `mcp_client_id` varchar(255) not null indexed
- `session_id` / `virtual_key_id` / `user_id` varchar(255) indexed
- `flow_mode` varchar(20) not null default 'vk'
- `status` varchar(50) not null indexed
- `expires_at` / `created_at` / `updated_at` datetime indexed

**`mcp_per_user_header_credentials`** (`framework/configstore/tables/mcp_per_user_headers.go`)
- `id` varchar(255) PK
- `session_id` / `virtual_key_id` / `user_id` varchar(255) indexed
- `mcp_client_id` varchar(255) not null indexed
- `auth_mode` varchar(20) not null
- `status` varchar(20) default 'active'
- `headers_json` text not null (encrypted)
- `encryption_status` varchar(20) default 'plain_text'
- `created_at` / `updated_at` datetime indexed

**`virtual_key_mcp_configs`** (implied by `transports/bifrost-http/handlers/mcp.go`)
- `virtual_key_id` + `mcp_client_id` + `tools_to_execute_json`

### g0router

**`internal/schemas/mcp.go`** — four structs, zero functions, no DB table.
- `MCPClient` {ID, Name, Type, Config map[string]any}
- `MCPInstance` {ID, ClientID, Name, Transport, URL, Command, Args, Env, Status}
- `MCPTool` {Name, Description, InputSchema}
- `MCPToolGroup` {ID, Name, ToolNames}

No migrations create MCP tables. `internal/store/migrate.go` has users, sessions, settings, providers, connections, oauth_sessions — no MCP tables.

---

## Edge Cases and Quirks

1. **Flexible duration parsing rejects fractional numeric values.** `parseFlexibleDurationField` at `core/schemas/mcp.go:373-422` caps inputs to JavaScript-safe int64 range (`±9,007,199,254,740,991`). Exponent notation (`eE`) accepted via `big.Rat`. Strings parsed with `time.ParseDuration`.

2. **Tool execution timeout intentionally diverges from generic Duration.** `MCPToolManagerConfig.UnmarshalJSON` at `core/schemas/mcp.go:223-252` treats bare integers as seconds, not nanoseconds.

3. **Per-user header keys canonicalized at request boundary.** `mcputils.CanonicalizeHeaderKeys` called in `addMCPClient` (`mcp.go:447`) and `updateMCPClient` (`mcp.go:935`). Duplicate detection runs on canonical form.

4. **Headers merge preserves redacted placeholders.** `mergeMCPHeaders` at `transports/bifrost-http/handlers/mcp.go:1423-1439` checks `incomingValue.IsRedacted() && incomingValue.Equals(redactedExisting[key])` before keeping raw value.

5. **TLS CACertPEM restored from raw on redacted placeholder match.** `updateMCPClient` at `transports/bifrost-http/handlers/mcp.go:868-878` compares redacted placeholder and swaps in existing raw PEM.

6. **Reconnect returns 400 for per-user auth types.** `ErrMCPReconnectNotApplicable` mapped to 400 at `transports/bifrost-http/handlers/mcp.go:323-325`.

7. **Revoke deletes pending flows before token/credential.** Race protection in `mcp_sessions.go:592-614` (headers) and `648-659` (OAuth).

8. **SSE heartbeat detects disconnect.** `reader.Send(ping)` returns false on client disconnect at `transports/bifrost-http/handlers/mcpserver.go:260-261`.

9. **Global MCP server registered twice with same tool filter.** `transports/bifrost-http/handlers/mcpserver.go:72-75` — duplicate `server.WithToolFilter(handler.makeIncludeClientsFilter())(handler.globalMCPServer)` call (harmless).

10. **UpdateMCPClient mutates existing config in place.** Snapshot taken before call at `transports/bifrost-http/handlers/mcp.go:826-827` to enable post-update diffing.

11. **Per-VK servers created lazily to avoid O(100k) startup stall.** `SyncAllMCPServers` clears `vkMCPServers` map; first request builds via `ensureVKMCPServer` at `mcpserver.go:583-595`.

12. **Discovered tools attached before DB persist on create.** Both per-user headers (`mcp.go:516-517`) and per-user OAuth (`completeMCPClientOAuth:1563-1564`) attach discovered tools to config before storage.

---

## Go-Port Considerations

1. Add `github.com/mark3labs/mcp-go` dependency for client and server primitives.
2. Implement `MCPCredentialStore` interface early; all auth types flow through it.
3. Use `WhiteList` type (or `[]string` with `*` semantics) for tool filtering; nil/empty = deny-all.
4. Store encrypted columns as `*_enc` per AGENTS.md decisions; use `ensureColumn` additive migrations.
5. Lazy per-VK server creation prevents startup stalls; cache invalidation on config change.
6. SSE handler requires careful fasthttp integration: `SetBodyStream` + atomic completer slot to avoid ctx recycle races.
