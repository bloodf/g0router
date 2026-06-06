# MCP Gateway Audit — g0router

**Auditor**: grumpy principal engineer  
**Date**: 2026-06-05  
**Scope**: `internal/mcp/`, `internal/cli/mcp_*.go`, `internal/store/mcp*.go`, `api/handlers/mcp*.go`  
**Method**: read-only source analysis, no edits, no network

---

## Summary Counts

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High     | 3 |
| Medium   | 5 |
| Low      | 4 |

---

## CRITICAL — None

No critical data-races or memory-safety issues found.

---

## HIGH

### H1 — No MCP Cancellation Support (Missing `notifications/cancelled`)
**Files**: `internal/mcp/stdio.go`, `internal/mcp/httpclient.go`  
**Why**: The MCP 2025-11-25 spec requires clients to send `notifications/cancelled` when the caller's context is cancelled mid-call. Neither `StdioClient.callLocked` (stdio.go:114–156) nor `StreamableHTTPClient.callLocked` (httpclient.go:90–113) nor `SSEClient.callLocked` (httpclient.go:253–267) do this. When the caller's `ctx` is cancelled, the in-flight request simply returns `ctx.Err()` and the server-side tool keeps running. For stdio this leaks a subprocess computation; for HTTP it wastes server resources and can corrupt the next call if the response arrives later.  
**Fix**: In `callLocked`, on `ctx.Done()`, send `notifications/cancelled` with the in-flight `id` before returning. For SSE/streamable-HTTP this must be a fire-and-forget POST; for stdio a non-blocking `notifyLocked`.  
**Verify**: Add a test that cancels a context mid-call and asserts the cancelled notification was sent.

---

### H2 — Stdio Client Locks I/O for Full Call Duration (no backpressure / context interruptibility)
**File**: `internal/mcp/stdio.go:131`  
**Why**: `c.reader.ReadBytes('\n')` inside `callLocked` holds `c.mu` and blocks the goroutine with no timeout or `ctx`-aware interrupt mechanism. `bufio.Reader.ReadBytes` does not accept a context. If the subprocess hangs, the mutex is held forever, and all subsequent `ListTools`/`CallTool` calls deadlock. The `ctx.Err()` check at line 128 only fires between iterations.  
**Fix**: Wrap the underlying `io.ReadCloser` in a context-aware reader that closes itself (or is paired with a `context.AfterFunc` kill) so the blocking read unblocks on cancellation. Alternatively use a dedicated read goroutine with a channel.  
**Verify**: Test: subprocess never responds — `ctx` with 100ms timeout must return, not deadlock.

---

### H3 — `HealthMonitor.registeredClients` reaches into `ClientManager.mu` directly (lock aliasing)
**File**: `internal/mcp/healthmonitor.go:206–215`  
**Why**: `registeredClients()` acquires `m.manager.mu` directly (`m.manager.mu.Lock()`). This couples `HealthMonitor` to the internal field layout of `ClientManager`, bypassing encapsulation. More critically, if `HealthMonitor.CheckAll` is called concurrently with `ClientManager.Register` or `ClientManager.Close` — both of which call `client.ListTools` while holding `manager.mu` — a goroutine calling `Register` that itself calls `ListTools` (which may block on I/O) while another goroutine in `registeredClients` also holds `manager.mu` produces a potential deadlock chain: `Register` holds `mu`, calls `ListTools` which blocks on stdio I/O; `registeredClients` tries to acquire `mu` — that is fine because `Register` holds and releases it. However, if `ListTools` (called under `manager.mu` at line 96 in `clientmanager.go:96`) blocks long-term, all other manager operations serialise behind it, starving health checks.  
**Fix**: Add a `Clients() map[string]Client` snapshot method to `ClientManager` that takes the lock, copies the map, and returns. `HealthMonitor.registeredClients` uses only that public method.  
**Verify**: `go test -race ./internal/mcp/...` with concurrent register + health check.

---

## MEDIUM

### M1 — Schema validation skips `array`, `allOf`, `oneOf`, `anyOf`, `$ref`, `enum`
**File**: `internal/mcp/toolmanager.go:283–313`  
**Why**: `validateSchemaValue` only validates `type`, `required`, `properties`, and `additionalProperties`. It silently accepts anything for `array` items, any `$ref`, any `allOf`/`oneOf`/`anyOf` combinator, and any `enum`. A tool with a schema like `{"type":"array","items":{"type":"string"}}` will accept `[1,2,3]` without error. This means malformed arguments reach the downstream MCP server.  
**Fix**: Handle `items` for array type; validate `enum` membership; for `$ref`/combinators either reject unsupported schemas with a clear error or expand them.  
**Verify**: Unit test: schema with `enum: ["a","b"]`, call with `"c"` — expect validation error.

---

### M2 — `rehydrateMCPRuntime` ignores errors silently; startup failures logged nowhere
**File**: `internal/cli/root.go:224–249`  
**Why**: Both `mcpInstanceForRuntime` errors and `RegisterInstance` errors are handled only by calling `s.UpdateMCPInstanceHealth(instance.ID, "unhealthy")`, with `continue`. There is no log line, no structured error capture. An operator restarting the server has no way to know which instances failed rehydration or why (bad creds, dead process, network unreachable). The `UpdateMCPInstanceHealth` call itself discards its error with `_`.  
**Fix**: Accept a `*log.Logger` or `slog.Logger` in `rehydrateMCPRuntime`; log each failure with instanceID and error. Return a summary if needed.  
**Verify**: Integration test: one bad instance config in DB → log output contains instance ID and error.

---

### M3 — OAuth discovery uses only `/.well-known/oauth-authorization-server` path; ignores RFC 8414 path variants
**File**: `internal/mcp/oauth.go:513–519`  
**Why**: `authorizationServerMetadataURL` always constructs `{scheme}://{host}/.well-known/oauth-authorization-server`, stripping all path components from the authorization server URL. RFC 8414 §3 allows the metadata endpoint to include the issuer path (e.g., `/.well-known/oauth-authorization-server/tenant/v2`). Some AS implementations (Azure AD, Okta) require the path-based form. This causes discovery failure for multi-tenant AS configurations.  
**Fix**: Implement RFC 8414 §3 path-prefixed discovery: try `{issuer}/.well-known/oauth-authorization-server` first (path-preserving), then fall back to root.  
**Verify**: Test with an issuer URL that has a non-empty path.

---

### M4 — SSEClient reconnect drops `initialized` state on endpoint reset
**File**: `internal/mcp/httpclient.go:224–251`  
**Why**: `ensureEndpoint` guards itself with `c.endpoint != "" && c.reader != nil` (line 225). However, there is no reconnect path: if the underlying SSE connection drops (EOF on `readSSEEvent`), `callLocked` returns an error but `c.endpoint` and `c.reader` remain set to the old (now broken) values. The next `CallTool` call will attempt to POST to the stale endpoint and fail; there is no automatic re-establish. `c.initialized` remains `true`, so `ensureInitialized` will not retry.  
**Fix**: On any I/O error in `callLocked`, clear `c.endpoint`, `c.reader`, `c.body`, and `c.initialized` so the next call triggers a fresh connection.  
**Verify**: Test: close SSE server mid-stream, next `ListTools` must succeed after reconnect.

---

### M5 — `mcpInstanceForRuntime` calls `selectMCPRuntimeOAuthAccount` which iterates all accounts without expiry check on the selected account path
**File**: `internal/cli/root.go:300–316`  
**Why**: `selectMCPRuntimeOAuthAccount` picks the first account matching label and URL without checking whether the token is expired. It returns an expired account, which `shouldRefreshMCPAccount` (line 343–347) then checks — but the refresh path returns an error, causing the instance to be marked unhealthy at rehydration, even though a valid non-expired account might exist later in the list (e.g., second account with same label after a re-auth).  
**Fix**: In `selectMCPRuntimeOAuthAccount`, prefer a non-expired account; only use an expired account if no valid one exists (for refresh purposes).  
**Verify**: Test: two accounts for same instance, first expired with refresh token, second valid — must use second.

---

## LOW

### L1 — `ClientManager.Register` calls `client.ListTools` while holding `mu`
**File**: `internal/mcp/clientmanager.go:84–106`  
**Why**: Lines 84–106: `m.mu.Lock()` is taken, `connector.Connect` is called (which may spawn a subprocess or make HTTP calls), then `client.ListTools(ctx)` is called — all while holding the mutex. Any other goroutine calling `Client()` or `Close()` is blocked for the full duration of the network/process startup. For slow MCP servers this is a UX issue; for pathological servers it serialises the entire gateway.  
**Fix**: Release the mutex before `Connect` and `ListTools`; re-acquire to write `m.clients[cfg.ID]`, with a double-check for duplicate registration.  
**Verify**: Benchmark: concurrent Register calls for 3 slow servers — latency should not serialise.

---

### L2 — `decodeSchemaObject` called `validateToolArguments` will panic on `schema["required"]` being `null` JSON
**File**: `internal/mcp/toolmanager.go:316–327`  
**Why**: `validateRequiredProperties` calls `json.Unmarshal(raw, &required)` where `raw` is `schema["required"]`. If `schema["required"]` is the JSON literal `null` (valid JSON Schema), `json.Unmarshal` sets `required` to `nil` and returns `nil` error, then the loop does nothing — this is fine. However if the key is absent, `schema["required"]` is a nil `json.RawMessage`; `json.Unmarshal(nil, ...)` returns an error which is silently swallowed (`return nil` on line 319). This silent swallow masks encoding bugs. Not a crash, but misleading.  
**Fix**: Distinguish absent key (`len(raw) == 0`) from `null` explicitly; only attempt unmarshal when key is present and non-null.  
**Verify**: Unit test: schema with absent `required` field — validation must still enforce nothing.

---

### L3 — `StdioCredentialEnv` exposes `MCP_REFRESH_TOKEN` in process environment
**File**: `internal/mcp/oauth.go:370–381`  
**Why**: `StdioCredentialEnv` injects `MCP_REFRESH_TOKEN` into the subprocess env (line 375–377). Subprocess environments are readable by any process running as the same user (`/proc/PID/environ` on Linux). Refresh tokens are long-lived credentials. The MCP spec only requires `MCP_ACCESS_TOKEN`.  
**Fix**: Do not inject `MCP_REFRESH_TOKEN` into subprocess env. The gateway should handle refresh itself and re-launch the process with a fresh access token.  
**Verify**: `StdioCredentialEnv` unit test asserts refresh token is absent from returned env map.

---

### L4 — `CompactToolsForRequest` drops `InputSchema` from returned `providers.Tool`
**File**: `internal/mcp/toolmanager.go:103–123`  
**Why**: `CompactToolsForRequest` builds `providers.Tool` with only `Type`, `Function.Name`, and `Function.Description` (lines 114–121). The `InputSchema` (parameter schema) is never included. Any downstream LLM call using this compact list cannot pass correct function-calling schemas to the provider, falling back to no schema. This causes inference engines to omit parameter hints to the model.  
**Fix**: Include the tool's `InputSchema` in `providers.ToolFunction.Parameters` when building the compact list.  
**Verify**: Test: register tool with schema, `CompactToolsForRequest` must return `Parameters` populated.

---

## NO-ISSUE Areas

- **Concurrency in `ToolManager`**: `sync.RWMutex` used correctly throughout (`mu.RLock` for reads, `mu.Lock` for writes). No map iteration during mutation.
- **`ClientManager.Close`**: Correctly deletes from map before releasing mutex, then calls `client.Close()` outside the lock — no lock-held I/O.
- **PKCE**: Implemented correctly — S256 challenge method, 32-byte random verifier, `base64.RawURLEncoding` for both verifier and challenge (`oauth.go:561–564`).
- **OAuth state CSRF**: State is 24-byte random, stored as SHA-256 hash in DB (`store/mcpoauth.go:308–311`), consumed atomically in a transaction — correct.
- **Token endpoint discovery fallback**: Both `derivedTokenEndpointForFlow` heuristic (`oauth.go:390–405`) and `discoverTokenEndpoint` metadata fetch are present with correct fallback order.
- **Secret redaction**: Both `internal/mcp/instances.go:isSecretKey` and `api/handlers/mcp.go:isMCPSecretKey` redact env/header keys containing `token`, `secret`, `key`, `authorization`, `password`.
- **OAuth flow expiry**: `CompleteCallback` checks `flow.ExpiresAt` (oauth.go:221–223). DB stores and DB scan both handle RFC3339 correctly.
- **JSON-RPC ID matching**: All three transports (`stdio.go:139`, `httpclient.go:337–339`) check response ID matches request ID.
- **SSE-to-streamable-HTTP fallback**: `Launcher.launchHTTP` (launcher.go:93–108) falls back from streamable-HTTP to SSE on 400/404/405/406/415 — appropriate set.
- **`http.ErrUseLastResponse`** set on OAuth HTTP clients — prevents redirect-following on token endpoints, correct per spec.
- **Store transactions for flow consumption**: `ConsumeMCPOAuthFlow` uses a transaction to atomically read+delete flow — prevents replay attacks.
- **Account label rehydration**: `AccountLabelForInstance` correctly queries the instance's `account_label` column and threads it through `accountLabelFromToken`.

---

## Transport Coverage

| Transport | Initialize | tools/list | tools/call | Reconnect |
|-----------|-----------|-----------|-----------|-----------|
| stdio | ✅ (`ensureInitialized`, `notifications/initialized`) | ✅ | ✅ | ❌ (no reconnect path) |
| SSE | ✅ (`ensureEndpoint` + initialize) | ✅ | ✅ | ❌ (H2, M4) |
| Streamable HTTP | ✅ (`legacyInitializeStreamable` + notify) | ✅ | ✅ | N/A (stateless POST) |

`notifications/cancelled` — **absent on all transports** (H1).

## OAuth Coverage

| Feature | Status |
|---------|--------|
| Discovery (protected resource metadata) | ✅ |
| Discovery (auth server metadata) | ✅ (M3: path variant gap) |
| PKCE S256 | ✅ |
| State CSRF | ✅ |
| Authorization code exchange | ✅ |
| Token refresh | ✅ |
| Account label selection | ✅ |
| Rehydration on restart | ✅ (M5: expired account selection order) |
| Refresh token in subprocess env | ⚠️ L3 |
