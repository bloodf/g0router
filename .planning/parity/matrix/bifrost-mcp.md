# BF-MCP Parity Matrix

Reference: Bifrost SHA `ca21298` (frozen clone at `/Users/heitor/Developer/github.com/bloodf/_refs/bifrost`).
Target: g0router (`/Users/heitor/Developer/github.com/bloodf/g0router`).

---

## Behavior Matrix

| ID | Behavior | Evidence (file:line) | g0router status | Notes |
|---|---|---|---|---|
| PAR-BF-MCP-001 | MCP client mode: connect to upstream tool servers (HTTP, SSE, STDIO, InProcess) | `core/schemas/mcp.go:441-444` (connection-type constants); `core/mcp/mcp.go:78-177` (constructor wiring AddClient for each config) | PARTIAL | bf-mcp-sat: STDIO live (`launcher.go:54 StartStdio`) + HTTP/SSE probe-discovery (`probe.go:55 Run` real initialize→tools/list handshake + `sse.go` SSE client); InProcess ABSENT (`! grep -ri inprocess internal/mcp` → none); persistent HTTP/SSE launcher connection deferred (w7-mcp-2 no-op `recordInstance` at `launcher.go:103`). |
| PAR-BF-MCP-002 | MCP server mode: expose Bifrost as MCP server over HTTP JSON-RPC + SSE | `transports/bifrost-http/handlers/mcpserver.go:85-88` (routes `/mcp` POST+GET) | HAVE | bf-mcp-1 (D1/D2): NEW `internal/mcp/server.go` dispatcher + `POST/GET /mcp` via `internal/server/routes_mcp.go` (raw JSON-RPC 2.0 + SSE); global un-scoped surface (per-VK = bf-mcp-2) |
| PAR-BF-MCP-003 | MCP server mode: global MCP server (un-scoped tools) | `transports/bifrost-http/handlers/mcpserver.go:58-62` (globalMCPServer); `core/mcp/mcp.go:39` (server field) | HAVE | bf-mcp-1 (D3): single global server re-exposing the existing aggregated catalog (`admin/mcp.go mcpToolCatalog`, the SAME source `ListTools` serves) |
| PAR-BF-MCP-004 | MCP server mode: per-virtual-key MCP server with lazy creation | `transports/bifrost-http/handlers/mcpserver.go:43` (vkMCPServers map); `289-308` (SyncVKMCPServer); `583-595` (ensureVKMCPServer) | HAVE-by-variant | bf-mcp-2 (D1): g0router has ONE global server; the per-VK scope is computed on-demand at request time (`admin/mcp.go newMCPServer(vk)` → `scopedServerTools`), not a long-lived per-VK server object — the "lazy creation" PURPOSE (no startup stall) is met by the request-time variant. Per-VK long-lived `vkMCPServers` map architecture VAR (open-questions) |
| PAR-BF-MCP-005 | Transport: HTTP (MCPConnectionTypeHTTP) | `core/schemas/mcp.go:441` | PARTIAL | bf-mcp-sat: HTTP probe dial (`probe.go:55 Run` real 3-step handshake over injectable `*http.Client`) + transport mode recorded (`mcpinstances.go:148` persists `transport='http'`; `launcher.go:90 StartHTTP` records the mode); persistent launcher HTTP connection deferred (w7-mcp-2 no-op `recordInstance`). |
| PAR-BF-MCP-006 | Transport: SSE (MCPConnectionTypeSSE) | `core/schemas/mcp.go:443` | PARTIAL | bf-mcp-sat: SSE client (`sse.go` sets `Accept: text/event-stream`) + SSE parse (`probe.go:142 extractTools` parses `text/event-stream` bodies) + mode recorded (`mcpinstances.go` persists `transport='sse'`; `launcher.go:96 StartSSE` records the mode); persistent launcher SSE connection deferred (w7-mcp-2 no-op). |
| PAR-BF-MCP-007 | Transport: STDIO (MCPConnectionTypeSTDIO) with command/args/env | `core/schemas/mcp.go:442`; `448-452` (MCPStdioConfig) | HAVE | bf-mcp-sat: STDIO command/args/env live spawn — `launcher.go:54 StartStdio(name, command, args, env)` → `runner.Start(ProcessSpec{Command, Args, Env, ...})`; `process.go:34 exec.Command(spec.Command, spec.Args...)`, `:35 cmd.Env = mergeEnv(spec.Env)`; allowlist-gated before spawn (`launcher.go:55 isAllowedCommand`). |
| PAR-BF-MCP-008 | Transport: InProcess (MCPConnectionTypeInProcess) | `core/schemas/mcp.go:444`; `302` (InProcessServer field) | MISSING | audited: not built — `! grep -ri inprocess internal/mcp` → none; g0router launches external processes or dials HTTP/SSE; InProcess not modeled. ESC. (bf-mcp-sat) |
| PAR-BF-MCP-009 | Auth type: none (MCPAuthTypeNone) | `core/schemas/mcp.go:271` | MISSING | audited: not built — no `auth_type`/`AuthType`/`MCPAuthType` enum in `internal/store/mcp*.go` or `internal/mcp/*.go` (`! grep -rniE 'auth_type\|AuthType\|MCPAuthType'` → none); OAuth wired directly; typed auth-mode enum is per-user/enterprise surface foundation (ESC). (bf-mcp-sat) |
| PAR-BF-MCP-010 | Auth type: headers (MCPAuthTypeHeaders) server-level static headers | `core/schemas/mcp.go:272`; `292` (Headers map) | MISSING | audited: not built — no client-mode static-auth-header injection on connect; `sse.go` sets only `Content-Type`/`Accept`; `allowed_extra_headers` (`mcpclientflags.go:15`) is the SERVER-mode request-header whitelist (row 071, different row). ESC (per-user/enterprise auth surface). (bf-mcp-sat) |
| PAR-BF-MCP-011 | Auth type: oauth (MCPAuthTypeOauth) server-level OAuth 2.0 | `core/schemas/mcp.go:273`; `288-290` (OauthConfigID, OauthClientID, OauthClientSecret) | HAVE | bf-mcp-sat: full authorization-code-with-PKCE engine — `oauth.go:20 Engine`; `:46 Start` (RFC 9728/8414 discovery + PKCE flow, verifier `*_enc`); `:88 Complete` (token exchange); `:128 Refresh`; tokens `*_enc` at rest (`mcpoauth.go:45 cipher.Encrypt`). Live caller: `admin/mcp.go CompleteInstanceAuth` (bf-mcp-1). |
| PAR-BF-MCP-012 | Auth type: per_user_oauth (MCPAuthTypePerUserOauth) | `core/schemas/mcp.go:274` | MISSING | No per-user auth model |
| PAR-BF-MCP-013 | Auth type: per_user_headers (MCPAuthTypePerUserHeaders) with required key schema | `core/schemas/mcp.go:275`; `300` (PerUserHeaderKeys) | MISSING | No per-user header schema |
| PAR-BF-MCP-014 | Per-user auth modes: user, vk, session identity dimensions | `core/schemas/mcp_headers.go:38-45` (MCPHeadersUserCredential with UserID, VirtualKeyID, SessionID) | MISSING | No identity-mode enum or credential store |
| PAR-BF-MCP-015 | Inline auth flow: MCPAuthRequiredError with Kind=oauth (AuthorizeURL) | `core/schemas/mcp.go:44-46` (constants); `63-81` (struct fields) | MISSING | No error type for auth-required |
| PAR-BF-MCP-016 | Inline auth flow: MCPAuthRequiredError with Kind=headers (SubmitURL, RequiredHeaderKeys) | `core/schemas/mcp.go:44-46`; `78-80` | MISSING | No header-submission flow |
| PAR-BF-MCP-017 | Tool filtering: per-client allow-list ToolsToExecute (["*"]=all, nil=[]=deny) | `core/schemas/mcp.go:303-307` (field + semantics comments) | HAVE | bf-mcp-2 (D4): `internal/mcp/scope.go scopeTools` implements `["*"]`=all, nil/`[]`=deny over a VK's assigned patterns; live consumer is the scoped `CatalogSource` (`admin/mcp.go scopedServerTools`) |
| PAR-BF-MCP-018 | Tool filtering: auto-execute list ToolsToAutoExecute subset of ToolsToExecute | `core/schemas/mcp.go:309-315` | PARTIAL | bf-mcp-2 (D5): `ToolsToAutoExecute` STORED (`virtual_key_mcp_configs`) + subset-validated in the live assignment write path (`scope.go validateAutoExecuteSubset`). Server-mode has NO auto-exec agent loop (g0router's agent loop is the CLIENT-mode `ExecuteTool` path) → server-mode auto-exec loop ESC (open-questions) |
| PAR-BF-MCP-019 | Tool filtering: per-virtual-key executeOnlyTools with wildcard `clientName-*` and `clientName-toolName` | `transports/bifrost-http/handlers/mcpserver.go:471-520` (fetchToolsForVK) | HAVE | bf-mcp-2 (D4): `scope.go scopeTools` matches `*`, `<client>-*` (prefix wildcard), and exact `<client>-<tool>`/bare via the SHIPPED `stripServerPrefix` dual-key primitive (no re-derived prefix logic); live consumer is the scoped `CatalogSource`. Test proves a restricted VK sees strictly fewer tools |
| PAR-BF-MCP-020 | Tool filtering: AllowOnAllVirtualKeys flag on client config | `core/schemas/mcp.go:322` | HAVE | bf-mcp-2 (D6): per-client `allow_on_all_virtual_keys` flag on the `mcp_clients.config_json` blob (`store.MCPClientAllowOnAllVKs`); the scoped `CatalogSource` UNIONs an all-VK client's tools into every VK's surface, bypassing the filter. Test proves a non-assigned VK sees a flagged client's tools, and does not when false |
| PAR-BF-MCP-021 | Tool discovery: ping vs listTools health check fallback (IsPingAvailable) | `core/schemas/mcp.go:316` (IsPingAvailable) | PARTIAL | bf-mcp-sat (VAR note): g0router uses `tools/list` as the single health/discovery signal (`probe.go:80`); no `IsPingAvailable` field and no ping-first-then-listTools fallback toggle (`! grep -rniE 'IsPingAvailable\|"ping"\|ping/' internal/mcp/probe.go` → none). The tools/list health path is HAVE; the ping-vs-listTools fallback selector + `IsPingAvailable` toggle is absent (VAR by design). |
| PAR-BF-MCP-022 | Tool sync interval: per-client override with flexible duration parsing (int=nanoseconds or Go duration string) | `core/schemas/mcp.go:166-208` (MCPConfig.UnmarshalJSON); `373-422` (parseFlexibleDurationField) | MISSING | No duration parsing or sync scheduler |
| PAR-BF-MCP-023 | Tool execution timeout: bare integer treated as seconds (not nanoseconds) | `core/schemas/mcp.go:210-252` (MCPToolManagerConfig.UnmarshalJSON) | MISSING | No tool manager config exists |
| PAR-BF-MCP-024 | Agent loop: iterative tool execution up to MaxAgentDepth for Chat API | `core/mcp/mcp.go:251-273` (CheckAndExecuteAgentForChatRequest) | HAVE-by-variant | bf-mcp-sat: generic bounded agent loop `agent.go:84 Agent.Run`; `agent.go:74 NewAgent(exec, maxTurns)` with `:13 defaultAgentMaxTurns=8`, `:22 ErrMaxTurnsExceeded` (no runaway); driven by injected `:55 ModelStep`. One generic loop (not a Chat-specific `CheckAndExecuteAgentForChatRequest` entrypoint) — behavior met, entrypoint shape VAR. |
| PAR-BF-MCP-025 | Agent loop: iterative tool execution up to MaxAgentDepth for Responses API | `core/mcp/mcp.go:303-324` (CheckAndExecuteAgentForResponsesRequest) | VAR | bf-mcp-sat: same `agent.go:84 Agent.Run` generic loop covers Responses-API iteration; no separate Responses entrypoint (`! grep -rniE 'Responses' internal/mcp/agent.go` → none); single generic loop, not two API-specific entrypoints (`CheckAndExecuteAgentForChatRequest`/`...ForResponsesRequest`). |
| PAR-BF-MCP-026 | Health monitor: automatic reconnect on startup failure with retention in Disconnected state | `core/mcp/mcp.go:142-170` (constructor: retain entry + start HealthMonitor on AddClient error) | MISSING | audited: not built — g0router's `healthmonitor.go` is an OAuth-TOKEN-REFRESH sweeper (`accountsNeedingRefresh`/`sweep` at `:13`/`:25`/`:64`), NOT a connection-reconnect-on-failure monitor with Disconnected-state lifecycle. Name collision: do NOT count the OAuth sweeper as 026. The reconnect-on-startup-failure behavior is not built. ESC. (bf-mcp-sat) |
| PAR-BF-MCP-027 | Credential store interface: ConnectionHeaders, RequestHeaders, RequiresPerCallConnection | `core/schemas/mcp.go:94-140` (MCPCredentialStore interface) | MISSING | audited: not built — `! grep -rniE 'CredentialStore\|ConnectionHeaders\|RequestHeaders\|RequiresPerCall' internal/mcp/*.go internal/store/mcp*.go` → none; no credential-store abstraction exists. ESC (per-user enterprise surface foundation, BIFROST-MAP §269). (bf-mcp-sat) |
| PAR-BF-MCP-028 | Per-user header credentials table with encrypted headers_json | `framework/configstore/tables/mcp_per_user_headers.go` (schema + GORM hooks) | MISSING | No DB table for MCP header credentials |
| PAR-BF-MCP-029 | Per-user header flow table with TTL expiry | `framework/configstore/tables/mcp_per_user_headers.go` (mcp_per_user_header_flows schema) | MISSING | No flow table |
| PAR-BF-MCP-030 | OAuth token table for per-user MCP OAuth (oauth_user_tokens) | `transports/bifrost-http/handlers/mcp_sessions.go:230` (ListOauthUserTokens) | MISSING | No OAuth token table |
| PAR-BF-MCP-031 | OAuth pending flow table (oauth_user_sessions) | `transports/bifrost-http/handlers/mcp_sessions.go:233` (ListPendingOauthUserSessions) | MISSING | No OAuth flow table |
| PAR-BF-MCP-032 | MCP client config table (config_mcp_clients) with encryption at rest | `framework/configstore/tables/mcp.go` (schema + BeforeSave/AfterFind/AfterDelete hooks) | PARTIAL | bf-mcp-1 (D6): existing `mcp_clients` + `mcp_oauth_accounts`/`mcp_oauth_flows` (`*_enc`) satisfy the server-mode/encryption-at-rest subset; full Bifrost enterprise column-set (per-user/code-mode/pricing/config-hash/allow-on-all-vk) ESC. ZERO new columns added (no server-mode consumer) |
| PAR-BF-MCP-033 | VK-MCP assignment table (virtual_key_mcp_configs) | `transports/bifrost-http/handlers/mcp.go:172-176` (GetVirtualKeyMCPConfigsByMCPClientIDs); `1215-1242` (transactional diffing) | HAVE | bf-mcp-2 (D2): NEW additive `virtual_key_mcp_configs` table + `internal/store/mcpvkconfigs.go` store (many-to-many VK↔client junction, mirrors the `mcp_tool_groups` INTEGER-PK pattern); the filter's live SOURCE via `ListVKMCPConfigsByVK`. Additive CRUD admin routes (`/api/mcp/vk-configs`) |
| PAR-BF-MCP-034 | API: GET /api/mcp/clients (paginated, with VK config merge + OAuth batch fetch) | `transports/bifrost-http/handlers/mcp.go:70`; `95-294` (getMCPClients + getMCPClientsPaginated) | MISSING | `api/` directory empty except `.DS_Store` |
| PAR-BF-MCP-035 | API: POST /api/mcp/client (create with auth branching) | `transports/bifrost-http/handlers/mcp.go:72`; `387-785` (addMCPClient) | MISSING | No MCP handler routes |
| PAR-BF-MCP-036 | API: PUT /api/mcp/client/{id} (PATCH semantics, immutable fields guarded, redacted merge) | `transports/bifrost-http/handlers/mcp.go:73`; `787-1336` (updateMCPClient) | MISSING | No update endpoint |
| PAR-BF-MCP-037 | API: DELETE /api/mcp/client/{id} (DB-first delete then memory removal) | `transports/bifrost-http/handlers/mcp.go:74`; `1338-1365` (deleteMCPClient) | MISSING | No delete endpoint |
| PAR-BF-MCP-038 | API: POST /api/mcp/client/{id}/reconnect (rejects disabled and per-user types) | `transports/bifrost-http/handlers/mcp.go:75`; `296-334` (reconnectMCPClient) | MISSING | No reconnect endpoint |
| PAR-BF-MCP-039 | API: POST /api/mcp/client/{id}/complete-oauth (distinguishes create vs update flow) | `transports/bifrost-http/handlers/mcp.go:76`; `1483-1705` (completeMCPClientOAuth) | PARTIAL | bf-mcp-1 (D7): `CompleteInstanceAuth` (POST `/api/mcp/instances/{id}/auth/complete`) is the FIRST live caller of the shipped-but-dead `Engine.Complete`, returns the masked account; create flow HAVE, create-vs-update distinction ESC (g0router single create flow). Route VAR (instance-scoped) |
| PAR-BF-MCP-040 | API: GET /api/mcp/sessions (cross-table merge with de-dup + pagination) | `transports/bifrost-http/handlers/mcp_sessions.go:37`; `196-327` (list) | MISSING | No sessions API |
| PAR-BF-MCP-041 | API: POST /api/mcp/sessions/{id}/reauth (OAuth or header branch, identity gate) | `transports/bifrost-http/handlers/mcp_sessions.go:38`; `405-560` (reauth + reauthHeaderCredential) | MISSING | No reauth endpoint |
| PAR-BF-MCP-042 | API: DELETE /api/mcp/sessions/{id} (race-protected: delete flows before token/credential) | `transports/bifrost-http/handlers/mcp_sessions.go:39`; `572-667` (revoke) | MISSING | No revoke endpoint |
| PAR-BF-MCP-043 | API: GET /api/oauth/per-user/flows/{id} (flow detail with HasActiveToken hint) | `transports/bifrost-http/handlers/mcp_sessions.go:40`; `690-762` (flowDetail) | MISSING | No flow detail endpoint |
| PAR-BF-MCP-044 | API: GET /api/oauth/per-user/flows/{id}/start (rebuilds upstream URL, 410 on stale) | `transports/bifrost-http/handlers/mcp_sessions.go:41`; `764-805` (flowStart) | MISSING | No flow start endpoint |
| PAR-BF-MCP-045 | API: per-user headers flow detail (`/api/mcp/per-user-headers/flows/{id}`) | `transports/bifrost-http/handlers/mcp_per_user_headers.go` (flowDetail) | MISSING | No per-user headers handler |
| PAR-BF-MCP-046 | API: per-user headers flow submit (`PUT /api/mcp/per-user-headers/flows/{id}`) with merge + verify | `transports/bifrost-http/handlers/mcp_per_user_headers.go` (flowSubmit) | MISSING | No submit endpoint |
| PAR-BF-MCP-047 | API: per-user headers credential revoke (`DELETE /api/mcp/per-user-headers/credential/{id}`) | `transports/bifrost-http/handlers/mcp_per_user_headers.go` (revoke) | MISSING | No credential revoke endpoint |
| PAR-BF-MCP-048 | API: POST /v1/mcp/tool/execute?format=chat|responses (direct tool execution) | `transports/bifrost-http/handlers/mcpinference.go:70-71`; `executeTool` | VAR | bf-mcp-sat audit correction: direct tool execution SHIPS at `POST /api/mcp/tools/{name}/execute` (`internal/admin/mcp.go:1018 ExecuteTool`, registered `routes_admin.go:187`, w7-mcp-3) returning `{result}` — the capability the row names is built (VAR, g0router admin-route shape). The `/v1`-namespaced `format=chat|responses` routing endpoint is a separate surface (NEEDS-DECISION / no consumer; the chat-vs-responses format routing has no g0router consumer). Prior note "No direct tool execution endpoint" was factually false. |
| PAR-BF-MCP-049 | Validation: ToolsToAutoExecute must be subset of ToolsToExecute | `transports/bifrost-http/handlers/mcp.go:1394-1417` (validateToolsToAutoExecute) | HAVE | bf-mcp-2 (D5): `scope.go validateAutoExecuteSubset` (PURE) is called in the LIVE assignment create/update path (`CreateVKMCPConfig`/`UpdateVKMCPConfig`) and REJECTS an `autoExecute ⊄ execute` assignment with a 4xx `{error}` before storage. Test proves the reject |
| PAR-BF-MCP-050 | Validation: per_user_header_keys canonicalized (lowercase+trim), no empties, no duplicates | `transports/bifrost-http/handlers/mcp.go:447-461`; `935-945` | MISSING | No canonicalization logic |
| PAR-BF-MCP-051 | Validation: OAuth credential rotation explicitly disabled (400 on req.OauthConfig in update) | `transports/bifrost-http/handlers/mcp.go:948-952` | MISSING | No OAuth update path to guard |
| PAR-BF-MCP-052 | VK resolution for MCP server: x-bf-vk > Authorization Bearer vk_* > x-api-key vk_* | `transports/bifrost-http/handlers/mcpserver.go:597-618` (getVKFromRequest) | HAVE | bf-mcp-1 (D4): `resolveMCPVK` precedence `x-g0-vk > Bearer > x-api-key` (header names VAR); resolved VK genuinely consumed — provided-but-unknown/inactive VK REJECTED via `store.GetVirtualKeyByKey`, absent allowed (optional surface; per-VK scoping = bf-mcp-2) |
| PAR-BF-MCP-053 | SSE heartbeat: `: ping\n\n` every 15s to detect disconnect via reader.Send() | `transports/bifrost-http/handlers/mcpserver.go:29` (sseHeartbeatInterval); `254-266` | HAVE | bf-mcp-1 (D5): `: ping\n\n` on each tick; `mcpSSEHeartbeatInterval = 15s`; hermetic — driven by an injected ticker/clock + an in-memory sink (zero real elapsed time in tests) |
| PAR-BF-MCP-054 | Trace completion deferred for SSE to avoid fasthttp body materialization deadlock | `transports/bifrost-http/handlers/mcpserver.go:145` (BifrostContextKeyDeferTraceCompletion) | HAVE | bf-mcp-1 (D8): SSE finalizer defers completion until AFTER the stream sink closes; carries a REAL payload — a best-effort `recordAudit("mcp_server.tools_call", …)` stamping the resolved VK (written post-close, not during frame emission). Context-key name VAR; numeric usage/cost shape ESC |
| PAR-BF-MCP-055 | Tool execution audit: ToolPricing map per client + MCPCatalog lookup by "server/tool_name" | `core/schemas/mcp.go:318` (ToolPricing); `framework/mcpcatalog/main.go` (pricing map) | MISSING | No pricing catalog |
| PAR-BF-MCP-056 | Code mode binding level: server vs tool VFS binding | `core/schemas/mcp.go:259-265` (CodeModeBindingLevel) | MISSING | No code mode |
| PAR-BF-MCP-057 | Disable auto tool inject flag | `core/schemas/mcp.go:217` (DisableAutoToolInject) | HAVE | bf-mcp-2 (D7): per-client `disable_auto_tool_inject` flag on the `mcp_clients.config_json` blob (`store.MCPClientDisableAutoToolInject`); a LIVE read in `scopedServerTools` omits a flagged client's tools from the scoped served surface. Test proves the suppression narrows the surface |
| PAR-BF-MCP-058 | Credential sweep worker: orphaned header credential deletion + expired flow cleanup | `framework/mcp_headers/sweep.go` (CredentialSweepWorker) | MISSING | No sweep worker |
| PAR-BF-MCP-059 | Temp token auth for per-user header flows (optional, skipped for user mode) | `framework/mcp_headers/main.go` (InitiateUserSubmissionFlow temp-token minting) | MISSING | No temp token service for MCP |
| PAR-BF-MCP-060 | Plugin pipeline provider/release for nested tool calls in code mode | `core/mcp/mcp.go:50-51` (pluginPipelineProvider/releasePluginPipeline fields) | MISSING | No plugin pipeline integration |
| PAR-BF-MCP-061 | DB rollback on in-memory AddMCPClient failure (create path) | `transports/bifrost-http/handlers/mcp.go:519-528` (DB delete on manager failure) | MISSING | No DB + memory two-phase create |
| PAR-BF-MCP-062 | DB rollback on in-memory UpdateMCPClient failure (update path) | `transports/bifrost-http/handlers/mcp.go:1126-1137` (oldDBConfig rollback) | MISSING | No update rollback |
| PAR-BF-MCP-063 | Discovered tool key migration on client rename (prefix replacement) | `transports/bifrost-http/handlers/mcp.go:1066-1083` (old prefix -> newPrefix + suffix) | MISSING | No discovered tool persistence |
| PAR-BF-MCP-064 | Per-user header credentials flipped to `needs_update` when schema adds keys | `transports/bifrost-http/handlers/mcp.go:1151-1157` (MarkMCPPerUserHeaderCredentialsNeedsUpdate) | MISSING | No credential state machine |
| PAR-BF-MCP-065 | VK credential reconciliation on assignment change (enterprise no-op in OSS) | `transports/bifrost-http/handlers/mcp.go:1320-1330` (ReconcileOauthAfterMCPChange + ReconcileMCPHeadersAfterMCPChange) | MISSING | No reconciliation hooks |
| PAR-BF-MCP-066 | TLS config with InsecureSkipVerify and CACertPEM (env-var aware) | `core/schemas/mcp.go:454-459` (MCPTLSConfig) | MISSING | audited: not built — `! grep -rniE 'InsecureSkipVerify\|CACert\|tls\.Config\|x509' internal/mcp/*.go internal/store/mcp*.go` → none; probe/SSE clients use default `*http.Client` (`probe.go:48 defaultHTTPClient`); no per-MCP-client TLS config schema. ESC (additive-buildable if funded, not built). (bf-mcp-sat) |
| PAR-BF-MCP-067 | MCP client state machine: Connected, Disconnected, Error, PendingTools, Disabled | `core/schemas/mcp.go:479-487` (MCPConnectionState constants) | VAR | bf-mcp-sat: g0router has a connection-lifecycle status — free-form `stopped/starting/running/error` (`mcpinstances.go:34` comment; `:142` default `'stopped'`; `SetMCPInstanceStatus` at `:212`); NOT the Bifrost 5-state enum (`Connected/Disconnected/Error/PendingTools/Disabled`). Locked by `TestMCPInstanceStatusLifecycle` (full `stopped→starting→running→error` round-trip, `internal/store/mcpinstances_test.go`). |
| PAR-BF-MCP-068 | Retry loop on UpdateMCPClient blocked by in-flight reconnect (3 attempts, 500ms) | `transports/bifrost-http/handlers/mcp.go:1442-1460` (updateMCPClientWithRetry) | MISSING | No retry logic |
| PAR-BF-MCP-069 | Retry loop on UpdateMCPClientConnection blocked by in-flight reconnect | `transports/bifrost-http/handlers/mcp.go:1463-1481` (updateMCPClientConnectionWithRetry) | MISSING | No retry logic |
| PAR-BF-MCP-070 | Headers merge: preserve raw values when incoming equals redacted placeholder | `transports/bifrost-http/handlers/mcp.go:1423-1439` (mergeMCPHeaders) | MISSING | No header redaction/merge logic |
| PAR-BF-MCP-071 | AllowedExtraHeaders whitelist for request-level header forwarding | `core/schemas/mcp.go:301` (AllowedExtraHeaders WhiteList) | PARTIAL | bf-mcp-2 (D8): `allowed_extra_headers` STORED on the `mcp_clients.config_json` blob + canonicalized (lowercase/trim/dedupe) on the live client write path (`store.CanonicalizeExtraHeaders` via `normalizeMCPClientConfig`); read back via `MCPClientAllowedExtraHeaders`. g0router server-mode makes NO per-request upstream call → header-forwarding consumer ESC (open-questions) |
| PAR-BF-MCP-072 | Error: ErrMCPReconnectNotApplicable for per-user auth types | `core/schemas/mcp.go:34-38` | MISSING | No sentinel error for reconnect |
| PAR-BF-MCP-073 | Error: OAuth2 flow errors (config not found, token expired, refresh failed, etc.) | `core/schemas/mcp.go:24-33` (ErrOAuth2* vars) | MISSING | No OAuth error types |
| PAR-BF-MCP-074 | MCPAuthRequiredError implements error interface (returns Message) | `core/schemas/mcp.go:83-85` | MISSING | No auth-required error type |
| PAR-BF-MCP-075 | mcp-go server integration for JSON-RPC message handling | `transports/bifrost-http/handlers/mcpserver.go:113` (mcpServer.HandleMessage) | HAVE-by-variant | bf-mcp-1 (D1): JSON-RPC message handling built over g0router's OWN bridge (`internal/mcp/server.go Dispatch` over the shipped `splitFrames`/`NewBridgeToolExecutor`); the `mark3labs/mcp-go` DEPENDENCY is deliberately NOT added (VAR) |
| PAR-BF-MCP-076 | mcp-go client integration for upstream connections | `core/schemas/mcp.go:19` (import `github.com/mark3labs/mcp-go/client`) | VAR | bf-mcp-sat: upstream-connection client built over g0router's OWN bridge (`bridge.go`/`probe.go`/`launcher.go`/`process.go`/`sse.go`); `mark3labs/mcp-go` dependency deliberately NOT added (`! grep -rnE 'mark3labs/mcp-go' go.mod internal/` → only server.go comments noting deliberate absence). Mirrors row 075 HAVE-by-variant treatment for the server side. |
| PAR-BF-MCP-077 | Tool annotations mapping (Title, ReadOnlyHint, DestructiveHint, IdempotentHint, OpenWorldHint) | `transports/bifrost-http/handlers/mcpserver.go:406-415` | PARTIAL | bf-mcp-2 (D8): additive `omitempty` `ToolAnnotations` (Title/ReadOnlyHint/DestructiveHint/IdempotentHint/OpenWorldHint) on `mcp.ServerTool`, carried through `assembleServerCatalog` → `tools/list`. Test proves an annotated catalog surfaces annotations + omitempty omits an absent block. g0router's probe source (`ProbeTool` Name/Description only) supplies NO annotation data → shape present, no probe-sourced data |
| PAR-BF-MCP-078 | IsCodeModeClient flag on client config | `core/schemas/mcp.go:282` | PARTIAL | bf-mcp-2 (D8): per-client `is_code_mode_client` flag STORED + operator-set on the `mcp_clients.config_json` blob (`store.MCPClientIsCodeMode`). Code-mode VFS / nested-tool execution engine is NOT built and NEVER faked → execution ESC (open-questions) |
| PAR-BF-MCP-079 | ConfigHash field for reconciliation | `core/schemas/mcp.go:320` | PARTIAL | bf-mcp-2 (D8): deterministic SHA-256 `config_hash` computed on the assignment write path (`vkMCPConfigHash`) + STORED on `virtual_key_mcp_configs.config_hash` + EXPOSED in the assignment GET DTO (live drift-detection reader; test asserts the GET DTO carries it and the hash changes on update). The auto-reconciliation WORKER that acts on drift is ESC (open-questions) |
| PAR-BF-MCP-080 | ConnectionString stored as *EnvVar (encrypted at rest) | `core/schemas/mcp.go:284` | VAR (narrow) | bf-mcp-sat: `*_enc` encryption-at-rest capability exists for MCP OAuth secrets (`mcpoauth.go:45,49,140 cipher.Encrypt` → `access_token_enc`/`refresh_token_enc`/`verifier_enc`); the SPECIFIC `ConnectionString`-typed-as-`*EnvVar`-encrypted shape does NOT exist — instance `url`/`command`/`args`/`env` are plaintext (`mcpinstances.go:148`); no `connection_string` column (`! grep -nE 'connection_string\|ConnectionString\|EnvVar' internal/store/mcpinstances.go` → none). MISSING-shape note: the ConnectionString-as-EnvVar shape is NOT built. REVIEWER OPTION: downgrade 080 to MISSING/ESC if capability-VAR is too generous (D6 open question). NOT counted as HAVE. |

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
