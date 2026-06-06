# Backend / API Audit — g0router

**Date:** 2026-06-05  
**Scope:** `api/handlers/*`, `api/middleware.go`, `api/server.go`, `internal/usage/`, `internal/logging/`, config loading, SQLite store, SCHEMA.md contract verification  
**Method:** Read-only source audit, no edits, no network calls.

---

## Verdict

Solid foundation. No critical security holes. Several medium-severity contract bugs and silent failures that will confuse clients and make debugging hard. 4 confirmed SCHEMA.md mismatches.

---

## Summary Table

| Severity | Count |
|----------|-------|
| Critical | 0 |
| High     | 3 |
| Medium   | 5 |
| Low      | 3 |
| **SCHEMA.md mismatches** | **4** |

---

## SCHEMA.md Contract Mismatches

### Mismatch 1 — Quota response uses PascalCase field names, SCHEMA says lowercase

**File:** `internal/usage/quota.go:17-24`  
**SCHEMA.md says (line 259):** "Responses include `provider`, `limit`, `used`, `remaining`…"  
**Actual JSON output:** `{"Provider":"openai","Limit":1000,"Used":125,"Remaining":875}` — `Quota` struct has no `json:` tags on the primary fields.  
**Test explicitly locks in PascalCase** (`usage_test.go:178-186` checks `"Provider"` present, `"provider"` absent).  
Client code expecting lowercase will silently get zero values.

### Mismatch 2 — Connection response uses PascalCase, not snake_case

**File:** `api/handlers/connections.go:18-34` (`connectionResponse` struct — zero json tags)  
**Actual JSON:** `{"ID":"…","Provider":"…","AuthType":"…","IsActive":true,"ProviderSpecificData":{…}}`  
**SCHEMA.md** does not specify field names for connection responses, but REST convention and the fact that *request* bodies (`connectionRequest`) use `"auth_type"`, `"is_active"`, `"provider_specific_data"` (all snake_case) creates an asymmetric contract that will confuse any client doing a round-trip.  
Integration test workaround: `server_integration_test.go:614-622` uses `json:"ID"`, `json:"AuthType"` etc. This is deliberate but undocumented.

### Mismatch 3 — `POST /api/oauth/:provider/exchange` exists in code, absent from SCHEMA.md

**File:** `api/server.go:532-537`  
Route `POST /api/oauth/:provider/exchange` calls `handlers.OAuthExchange`. Not listed in SCHEMA.md under Management routes (line 231-232 only lists `authorize` and `poll`).  
Clients relying solely on the schema have no knowledge of this exchange endpoint.

### Mismatch 4 — `cache_write_tokens` always null despite being in DB schema and response

**File:** `internal/usage/tracker.go` (Usage struct has no `CacheWriteTokens` field), `internal/logging/requestlog.go:10-30` (no `CacheWriteTokens` in `RequestLog`), `internal/store/sqlite.go:121` (column exists), `internal/store/usage.go:21` (`CacheWriteTokens *int` in `RequestLogEntry`)  
**SCHEMA.md** (`request_log` table, line 98) shows `cache_write_tokens INTEGER`.  
The column is in the DB and `RequestLogEntry`, but the upstream `usage.Usage` struct (`tracker.go`) only has `CacheReadTokens` — no `CacheWriteTokens`. `logging/requestlog.go:Entry()` never sets `CacheWriteTokens`. It is always `NULL` in every log row, and always `null` in `/api/usage` responses. Clients cannot trust this field.

---

## Findings

### HIGH-1 — Logging errors silently swallowed

**File:** `api/server.go:216`
```go
_ = logging.NewLogger(usageStore).Log(entry)
```
**Why it matters:** Failed request log writes are silently discarded. No metric, no log line, no visibility. Operators see zero errors in logs while request_log rows silently stop accumulating — diagnosis is impossible.  
**Minimal fix:** Replace `_ =` with an actual error log: `if err := logging.NewLogger(usageStore).Log(entry); err != nil { /* log/metric */ }`. At minimum use `fmt.Fprintf(os.Stderr, …)`.  
**Verification:** Add a test with a failing `RequestStore` and assert the error is surfaced.

---

### HIGH-2 — `RTKBytesSaved`, `ComboName`, `ClientTool` never populated in request log

**File:** `api/server.go:198-216` (`logInferenceUsage`)  
**Fields in `RequestLog` struct:** `RTKBytesSaved *int`, `ComboName *string`, `ClientTool *string` (`logging/requestlog.go:25-29`)  
**What happens:** `logInferenceUsage` constructs a `logging.RequestLog` and never sets these three fields. They are always `nil` in every log entry, regardless of whether RTK is active, a combo is used, or the client tool is detectable.  
`RTKEnabled` and `CavemanEnabled` *are* set (line 211-212), but `RTKBytesSaved` is not — so a key metric for evaluating RTK savings is always zero.  
**Why it matters:** The entire purpose of `rtk_bytes_saved` and `combo_name` columns (SCHEMA.md lines 109-110) is to let operators measure savings. Both are dead columns.  
**Minimal fix:** Populate `ComboName` from `req.ComboName` if the field exists; populate `RTKBytesSaved` from the RTK compressor result; populate `ClientTool` via the user-agent or request header detection logic.  
**Verification:** Check `request_log` after a combo-routed inference call.

---

### HIGH-3 — `Usage.CacheWriteTokens` missing from `usage.Usage` struct (dead column)

**File:** `internal/usage/tracker.go:5-17` — `Usage` struct has `CacheReadTokens` but no `CacheWriteTokens`. `logging/requestlog.go:52-58` maps `Usage` fields to `RequestLogEntry` but has no path for `CacheWriteTokens`.  
**Why it matters:** Anthropic and OpenAI both report `cache_creation_input_tokens` / `cache_write_tokens`. The DB column exists. The `RequestLogEntry` field exists. But the pipeline from provider response → `Usage` → `RequestLog` → DB permanently drops cache write token counts, so cost calculations for prompt-cache-heavy workloads are wrong.  
**Minimal fix:** Add `CacheWriteTokens int` to `usage.Usage`; populate it from `providerUsage.PromptTokensDetails.CachedWriteTokens` (or equivalent) in `fromProviderUsage`; pass it through `RequestLog.Entry()`.  
**Verification:** Log a request with Anthropic cache creation headers and assert `cache_write_tokens > 0` in the DB row.

---

### MEDIUM-1 — `/api/usage`, `/api/usage/summary`, `/api/logs` accept any HTTP method

**File:** `api/server.go:537-544`; `api/handlers/usage.go:61-110`; `api/handlers/logging.go:9-33`  
None of `Usage`, `UsageSummary`, or `Logs` handlers check `ctx.Method()`. A `POST /api/usage` or `DELETE /api/logs` returns `200 OK` with data.  
**Why it matters:** Violates HTTP semantics; can mask client bugs; could be mistaken for a write endpoint. CORS pre-flight for DELETE/POST would succeed, enabling unintended cross-origin reads.  
**Minimal fix:** Add method guard at the top of each handler:
```go
if string(ctx.Method()) != fasthttp.MethodGet {
    ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
    return
}
```
Same applies to `UsageQuota` — line 112 has no method check either.  
**Verification:** `curl -X DELETE /api/usage` should return 405, currently returns 200.

---

### MEDIUM-2 — `MCPOAuthStart` returns `201 Created` but is not a resource-creation endpoint

**File:** `api/handlers/mcp.go:204`
```go
writeJSON(ctx, fasthttp.StatusCreated, mcpOAuthStartResponse{…})
```
**SCHEMA.md** (line 267): `POST /api/mcp/instances/:id/auth/start` — implies a flow initiation, not resource creation. The OAuth flow is stored but the response is the auth URL + expiry. RFC semantics: `201` should include `Location` header pointing to the created resource. Returns `201` without `Location`.  
Compare: `POST /api/oauth/:provider/authorize` (the non-MCP equivalent, `oauth.go:64`) returns `200`.  
**Why it matters:** Inconsistent status codes across similar endpoints confuse client retry logic.  
**Minimal fix:** Change to `200` to match `OAuthStart`, or add `Location` header if keeping `201`.

---

### MEDIUM-3 — `ConnectionTest` is a stub — does not test the connection

**File:** `api/handlers/connections.go:111-134`  
`ConnectionTest` fetches the connection from the store and returns `{"ok": conn.IsActive, …}`. It does not make any outbound call to verify the credentials are still valid. A connection with a revoked token but `is_active=true` returns `{"ok": true}`.  
**SCHEMA.md line 229:** `POST /api/connections/:id/test — Test connection` — implies actual testing.  
**Why it matters:** Operators use this to diagnose auth failures. It always returns success for active connections regardless of real credential state.  
**Minimal fix:** Either rename to `/status` to set accurate expectations, or route through the provider's ping/validate endpoint. At minimum document the limitation in SCHEMA.md.

---

### MEDIUM-4 — `http.DefaultClient` used in `MCPOAuthStart` without timeout

**File:** `api/handlers/mcp.go:171`
```go
authorizationURL, err = mcp.DiscoverOAuthAuthorizationURL(requestContext(ctx), http.DefaultClient, resourceURI)
```
`http.DefaultClient` has no timeout. A slow or hanging MCP OAuth discovery endpoint will hold a goroutine indefinitely.  
**Why it matters:** Slow endpoints can exhaust goroutine pool under moderate load. `requestContext(ctx)` provides a request context but there's no deadline set on it either (see `context.go:9-14` — `requestContext` just returns `ctx` which is the fasthttp `RequestCtx`, which does not cancel on connection close).  
**Minimal fix:** Pass `&http.Client{Timeout: 10 * time.Second}` instead of `http.DefaultClient`.

---

### MEDIUM-5 — `providerFromModel` allocates a new catalog on every inference log

**File:** `api/server.go:452-464`  
```go
func providerFromModel(model string) string {
    if provider, ok := modelcatalog.NewCatalog().ProviderForModel(model); ok {
```
Also `inferenceLogMetadata` at line 377 calls `modelcatalog.NewCatalog()` separately. Both called per-request.  
Depending on what `NewCatalog()` does (not audited in full depth), if it loads from disk or builds a map, this is a per-request allocation with no caching.  
**Why it matters:** Under high throughput this adds measurable overhead.  
**Minimal fix:** Cache a singleton catalog at `Server` or package level; or pass it in via `ServerConfig`.

---

### LOW-1 — `requestContext` does not propagate cancellation

**File:** `api/handlers/context.go:9-14`
```go
func requestContext(ctx *fasthttp.RequestCtx) context.Context {
    if ctx == nil {
        return context.Background()
    }
    return ctx
}
```
`*fasthttp.RequestCtx` implements `context.Context` but its `Done()` channel never closes — fasthttp does not cancel the context when the client disconnects. Long-running inference dispatches (streaming) will continue even after client drops.  
**Why it matters:** Resource waste on abandoned streams; no back-pressure to providers.  
**Minimal fix:** Wrap with a cancellable context tied to connection lifecycle, or use a middleware that monitors connection state.

---

### LOW-2 — `listener()` returns `nil` on error with no logging

**File:** `api/server.go:77-83`
```go
func (s *Server) listener() net.Listener {
    ln, err := net.Listen("tcp", ":"+strconv.Itoa(s.config.Port))
    if err != nil {
        return nil
    }
    return ln
}
```
`listener()` is not actually called in the public API (callers use `Serve(ln)` with an externally-created listener), but if ever called internally, a bind failure is silently swallowed and the caller gets `nil` with no explanation.  
**Minimal fix:** Return `error` or log the failure.

---

### LOW-3 — `settings.go` `PUT /api/settings` echoes back unsaved settings on update error

**File:** `api/handlers/settings.go:25-36`  
On `UpdateSettings` error, the handler returns `500` — correct. But on success it returns the `settings` struct decoded from the request body, not a fresh `GetSettings()` read. If `UpdateSettings` silently normalises or adjusts values (e.g. defaults), the response won't reflect the actual stored state.  
**Minimal fix:** Re-read settings after update: `s.GetSettings()` → return result, same pattern used by `Connections` and `Combos` update handlers.

---

## Areas With No Issues Found

- **Middleware auth flow** (`api/middleware.go`): Correct bearer + `X-API-Key` extraction, proper OAuth callback exemptions, CORS restricted to localhost origins only.
- **SQL injection**: All store queries use parameterised `?` placeholders throughout `internal/store/`.
- **Redaction logic**: `redactConnections`/`redactProviderSpecificData` correctly strips credential fields recursively; `redactMCPSecretMap` for MCP instances mirrors the same logic.
- **JWT/API key security**: HMAC-SHA256 key hashing, prefix stored for display, raw key returned only at creation time, never stored.
- **Error wrapping**: Consistent `fmt.Errorf("context: %w", err)` pattern throughout.
- **Streaming cleanup**: `captureStream` goroutine (`server.go:289-310`) correctly drains channel and calls `onStreamComplete` after last chunk; no goroutine leak.
- **MCP rollback**: Both `MCPClients` and `MCPInstances` `POST` handlers correctly delete the store record when runtime registration fails (`mcp.go:108-128`, `269-276`).
- **Quota caching**: `CachingQuotaFetcher` uses a mutex correctly, no lock held across I/O.
- **Integration test coverage**: `server_integration_test.go` covers auth, inference (OpenAI/Anthropic/Responses), management CRUD, MCP instance OAuth, usage/quota, connection redaction. Coverage is strong for happy paths.
- **Unit test coverage**: All handler files have corresponding `_test.go` files with CRUD and error cases. `inference_test.go` covers dispatch errors, streaming, Anthropic translation, tool use blocks.
