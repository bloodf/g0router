# g0router Audit — Phases 1-6

**Auditor:** strict code audit  
**Scope:** full repo (`go test ./...` passes, `go vet ./...` clean)  
**Date:** 2026-06-09  
**Lines:** ~8,400 Go + ~3,100 test  

---

## Findings

| ID | Finding | Evidence (file:line) | Severity | Remediation |
|---|---|---|---|---|
| AUD-001 | `rand.Read` error discarded in `newID()` — produces weak/empty IDs on failure | `internal/store/store.go:69` | BROKEN | Check `rand.Read` return and propagate error or retry. |
| AUD-002 | `rand.Read` error discarded in `newToken()` — session tokens can be empty | `internal/auth/session.go:103` | BROKEN | Check `rand.Read` return and propagate error or retry. |
| AUD-003 | `rand.Read` error discarded in `randomURLSafe()` — OAuth state/verifier can be empty | `internal/auth/oauth.go:164` | BROKEN | Check `rand.Read` return and propagate error or retry. |
| AUD-004 | Hardcoded production Anthropic OAuth `client_id` baked into source | `internal/auth/oauth.go:36` | BROKEN | Move to configuration/env var; rotate exposed ID. |
| AUD-005 | `ensureColumn` constructs SQL via `fmt.Sprintf` with unsanitized params — injection vector if ever called dynamically | `internal/store/migrate.go:105` | DEBT | Use parameterized DDL or whitelist table/column names. |
| AUD-006 | SQLite schema has `NOT NULL` on foreign-key columns but no `FOREIGN KEY` constraints | `internal/store/migrate.go:16-62` | DEBT | Add `REFERENCES ... ON DELETE ...` clauses to enforce referential integrity. |
| AUD-007 | `json.Marshal` error ignored — can emit empty/invalid SSE chunks | `internal/api/chat.go:53` | BROKEN | Handle `json.Marshal` error; abort stream or log. |
| AUD-008 | `ctx.WriteString`/`ctx.Write` errors discarded during SSE streaming — client disconnects undetected | `internal/api/chat.go:54-58` | DEBT | Check write errors and abort stream on failure. |
| AUD-009 | `json.Marshal` error ignored in non-streaming chat response | `internal/api/chat.go:72` | BROKEN | Handle `json.Marshal` error; return 500. |
| AUD-010 | `json.Marshal` error ignored in embeddings response | `internal/api/embeddings.go:50` | BROKEN | Handle `json.Marshal` error; return 500. |
| AUD-011 | `json.Marshal` error ignored in models response | `internal/api/models.go:44` | BROKEN | Handle `json.Marshal` error; return 500. |
| AUD-012 | `json.Marshal` error ignored in error response writer | `internal/api/errors.go:20` | BROKEN | Handle `json.Marshal` error; write plain-text fallback. |
| AUD-013 | `pathID` silently returns `""` for non-string route params | `internal/admin/handlers.go:25` | DEBT | Return error or panic on bad type assertion. |
| AUD-014 | `UpdateConnection` does not validate `ProviderID` exists when changed | `internal/admin/connections.go:124` | DEBT | Add `GetProvider` check matching `CreateConnection`. |
| AUD-015 | CORS middleware reflects any `Origin` header and sets `Allow-Credentials: true` — CSRF risk | `internal/server/middleware.go:33-40` | BROKEN | Whitelist allowed origins or remove credentials header. |
| AUD-016 | `ErrNoProvider` defined but never returned; `Resolve` has no error path | `internal/inference/router.go:63` | DEBT | Remove dead var or add unknown-model error path. |
| AUD-017 | `ModelsHandler.Get` ignores `{id}` param and delegates to `List` | `internal/api/models.go:51` | DEBT | Implement per-model lookup or remove route until Phase 9. |
| AUD-018 | `DeleteExpiredSessions` exported but never called — no cleanup cron | `internal/store/sessions.go:57` | DEBT | Add background ticker or admin endpoint to invoke cleanup. |
| AUD-019 | `oauth_sessions.created_at` written but `OAuthSession` struct lacks the field | `internal/store/oauthsessions.go:25-28` | DEBT | Add `CreatedAt` to struct or drop column. |
| AUD-020 | `settings.updated_at` written but `GetSettings` returns `map[string]string`, dropping timestamp | `internal/store/settings.go:40-43` | DEBT | Add timestamp to return type or drop column. |
| AUD-021 | `sessions.created_at` scanned but never read by callers | `internal/store/sessions.go:33-35` | EXTRA | Remove field if unused, or expose via API. |
| AUD-022 | `users.created_at` / `users.updated_at` scanned but `userDTO` only exposes `ID`, `Username` | `internal/store/users.go:51-53` | EXTRA | Add timestamps to DTO or drop columns. |
| AUD-023 | `Connection.CreatedAt` / `UpdatedAt` passed through DTO but never used for business logic | `internal/admin/connections.go:37-50` | EXTRA | Remove from DTO if UI does not need them yet. |
| AUD-024 | `newTestStore` duplicated verbatim across `store`, `auth`, `server` test files | `store/store_test.go:12`, `auth/auth_test.go:17`, `server/server_test.go:20` | DEBT | Extract to `internal/testutil`. |
| AUD-025 | HTTP call helper duplicated between `admin_test.go` and `integration_test.go` | `admin/admin_test.go:72`, `server/integration_test.go:72` | DEBT | Extract to shared test helper. |
| AUD-026 | Fake OAuth token server duplicated across `auth_test.go`, `admin_test.go`, `integration_test.go` | `auth/auth_test.go:144`, `admin/admin_test.go:44`, `server/integration_test.go:38` | DEBT | Extract to `internal/testutil`. |
| AUD-027 | Handler success-response pattern copy-pasted in `chat.go`, `embeddings.go`, `models.go` | `api/chat.go:72-75`, `api/embeddings.go:50-53`, `api/models.go:44-47` | DEBT | Extract `writeJSON(ctx, status, v)` helper. |
| AUD-028 | Handler provider-error pattern copy-pasted in `chat.go`, `embeddings.go`, `models.go` | `api/chat.go:63-69`, `api/embeddings.go:41-47`, `api/models.go:35-41` | DEBT | Extract `writeProviderError(ctx, perr)` helper. |
| AUD-029 | `healthHandler` returns raw `{"status":"ok"}` — violates `{data,error}` envelope convention | `internal/server/ui.go:19` | DEBT | Wrap in envelope or document OpenAI-surface exception. |
| AUD-030 | OpenAI-compatible API returns raw success/error shapes, not project `{data,error}` envelope | `api/errors.go:10-24`, `api/chat.go:72-75` | DEBT | Document exception in AGENTS.md or add admin envelope variant. |
| AUD-031 | Anthropic `ConvertRequest` silently drops multiple system messages — only last kept | `internal/providers/anthropic/converter.go:113-119` | BROKEN | Concatenate or error on multiple system messages. |
| AUD-032 | Anthropic `ConvertRequest` ignores `schemas.ChatRequest.N` | `internal/providers/anthropic/converter.go:91-131` | PARTIAL | Add mapping or document unsupported; confirm via A1 matrix. |
| AUD-033 | Anthropic `ConvertRequest` ignores `PresencePenalty`, `FrequencyPenalty`, `LogitBias`, `User`, `ResponseFormat`, `Seed` | `internal/providers/anthropic/converter.go:91-131` | PARTIAL | Map or document each; confirm via A1 matrix. |
| AUD-034 | Anthropic `convertMessages` ignores `Message.Name` | `internal/providers/anthropic/converter.go:133-164` | PARTIAL | Map to Anthropic name field or document drop. |
| AUD-035 | Gemini `ConvertChatRequest` silently drops multiple system messages — only last kept | `internal/providers/gemini/converter.go:132-140` | BROKEN | Concatenate or error on multiple system messages. |
| AUD-036 | Gemini `ConvertChatRequest` ignores `N`, `Stream`, `PresencePenalty`, `FrequencyPenalty`, `LogitBias`, `User`, `ResponseFormat`, `Seed` | `internal/providers/gemini/converter.go:104-152` | PARTIAL | Map or document each; confirm via A1 matrix. |
| AUD-037 | Gemini `convertMessages` ignores `Message.Name` | `internal/providers/gemini/converter.go:154-186` | PARTIAL | Map to Gemini name field or document drop. |
| AUD-038 | Gemini `convertMessages` loses `Message.ToolCallID` for tool-role messages | `internal/providers/gemini/converter.go:161` | BROKEN | Propagate tool call ID into function response metadata. |
| AUD-039 | Gemini `ConvertChatResponse` leaves `ChatResponse.ID` empty | `internal/providers/gemini/converter.go:215` | BROKEN | Generate a response ID. |
| AUD-040 | Gemini `ConvertStreamChunk` leaves `StreamChunk.ID` empty | `internal/providers/gemini/converter.go:322` | BROKEN | Generate a chunk ID. |
| AUD-041 | Gemini tool-call IDs collide when same function invoked twice: `"call_" + name` | `internal/providers/gemini/converter.go:244` | BROKEN | Use random or sequential IDs. |
| AUD-042 | Gemini `json.Unmarshal` error ignored in tool argument parsing — passes nil args silently | `internal/providers/gemini/converter.go:169` | BROKEN | Return error on malformed JSON arguments. |
| AUD-043 | Gemini error converter drops integer `Code` field — never mapped to `ProviderError.Code` | `internal/providers/gemini/errors.go:41` | DEBT | Map `envelope.Error.Code` to string code. |
| AUD-044 | `NetworkConfig` stored by all providers but `Timeout`/`ProxyURL`/`MaxRetries` never applied to `ClientPool` | `openai/provider.go:33`, `anthropic/provider.go:33`, `gemini/provider.go:33` | DEBT | Wire `NetworkConfig` into `utils.NewClientPool` or remove field. |
| AUD-045 | Streaming SSE JSON unmarshal errors silently skipped via `continue` in all three providers | `openai/chat.go:144`, `anthropic/chat.go:149`, `gemini/chat.go:150` | BROKEN | Abort stream or send error chunk on parse failure. |
| AUD-046 | Streaming non-EOF scanner errors break loop but do not propagate to caller | `openai/chat.go:134-139`, `anthropic/chat.go:138-142`, `gemini/chat.go:139-143` | DEBT | Close channel with error sentinel or log. |
| AUD-047 | `postHookRunner.Run` errors discarded in all three providers | `openai/chat.go:149`, `anthropic/chat.go:164`, `gemini/chat.go:158` | DEBT | Log hook errors or abort on failure. |
| AUD-048 | `gemini/chat.go` defines `setAuthHeader` and `buildModelURI` but neither is called | `gemini/chat.go:167-175` | EXTRA | Delete dead functions. |
| AUD-049 | `gemini/chat.go` does not call `sanitizeModelName` on chat model — passes `"gemini/gemini-1.5-pro"` to URI | `gemini/chat.go:23`, `gemini/embedding.go:18` | BROKEN | Call `sanitizeModelName` in chat path or unify model prep. |
| AUD-050 | Provider struct + constructor + stubs copy-pasted verbatim across openai/anthropic/gemini | `openai/provider.go:9-35`, `anthropic/provider.go:9-35`, `gemini/provider.go:9-35` | DEBT | Extract generic provider base struct to `internal/providers/base`. |
| AUD-051 | Chat non-streaming boilerplate copy-pasted across openai/anthropic/gemini | `openai/chat.go:15-70`, `anthropic/chat.go:15-72`, `gemini/chat.go:16-74` | DEBT | Extract generic HTTP JSON round-trip helper. |
| AUD-052 | Chat streaming boilerplate copy-pasted across openai/anthropic/gemini | `openai/chat.go:73-155`, `anthropic/chat.go:75-174`, `gemini/chat.go:77-165` | DEBT | Extract generic SSE stream helper. |
| AUD-053 | `internal/catalog` stub: doc.go + empty test only; zero imports | `internal/catalog/doc.go`, `internal/catalog/catalog_test.go` | EXTRA | DELETE — no seed value. |
| AUD-054 | `internal/config` stub: doc.go + empty test only; zero imports | `internal/config/doc.go`, `internal/config/config_test.go` | EXTRA | DELETE — no seed value. |
| AUD-055 | `internal/governance` stub: doc.go + empty test only; zero imports | `internal/governance/doc.go`, `internal/governance/governance_test.go` | EXTRA | DELETE — no seed value. |
| AUD-056 | `internal/logging` stub: doc.go + empty test only; zero imports | `internal/logging/doc.go`, `internal/logging/logging_test.go` | EXTRA | DELETE — no seed value. |
| AUD-057 | `internal/mcp` stub: doc.go + empty test only; zero imports | `internal/mcp/doc.go`, `internal/mcp/mcp_test.go` | EXTRA | DELETE — no seed value. |
| AUD-058 | `internal/platform` stub: doc.go + empty test only; zero imports | `internal/platform/doc.go`, `internal/platform/platform_test.go` | EXTRA | DELETE — no seed value. |
| AUD-059 | `internal/providers/bedrock` stub: doc.go + empty test only; zero imports | `internal/providers/bedrock/doc.go`, `internal/providers/bedrock/bedrock_test.go` | EXTRA | DELETE — no seed value. |
| AUD-060 | `internal/providers/cohere` stub: doc.go + empty test only; zero imports | `internal/providers/cohere/doc.go`, `internal/providers/cohere/cohere_test.go` | EXTRA | DELETE — no seed value. |
| AUD-061 | `internal/providers/deepseek` stub: doc.go + empty test only; zero imports | `internal/providers/deepseek/doc.go`, `internal/providers/deepseek/deepseek_test.go` | EXTRA | DELETE — no seed value. |
| AUD-062 | `internal/providers/fireworks` stub: doc.go + empty test only; zero imports | `internal/providers/fireworks/doc.go`, `internal/providers/fireworks/fireworks_test.go` | EXTRA | DELETE — no seed value. |
| AUD-063 | `internal/providers/groq` stub: doc.go + empty test only; zero imports | `internal/providers/groq/doc.go`, `internal/providers/groq/groq_test.go` | EXTRA | DELETE — no seed value. |
| AUD-064 | `internal/providers/minimax` stub: doc.go + empty test only; zero imports | `internal/providers/minimax/doc.go`, `internal/providers/minimax/minimax_test.go` | EXTRA | DELETE — no seed value. |
| AUD-065 | `internal/providers/mistral` stub: doc.go + empty test only; zero imports | `internal/providers/mistral/doc.go`, `internal/providers/mistral/mistral_test.go` | EXTRA | DELETE — no seed value. |
| AUD-066 | `internal/providers/ollama` stub: doc.go + empty test only; zero imports | `internal/providers/ollama/doc.go`, `internal/providers/ollama/ollama_test.go` | EXTRA | DELETE — no seed value. |
| AUD-067 | `internal/providers/together` stub: doc.go + empty test only; zero imports | `internal/providers/together/doc.go`, `internal/providers/together/together_test.go` | EXTRA | DELETE — no seed value. |
| AUD-068 | `internal/providers/vertex` stub: doc.go + empty test only; zero imports | `internal/providers/vertex/doc.go`, `internal/providers/vertex/vertex_test.go` | EXTRA | DELETE — no seed value. |
| AUD-069 | `internal/api/api_test.go` is compile-only placeholder with zero assertions | `internal/api/api_test.go:5-9` | EXTRA | Write real handler tests or delete file. |
| AUD-070 | `internal/inference/inference_test.go` is compile-only placeholder with zero assertions | `internal/inference/inference_test.go:5-9` | EXTRA | Write real router tests or delete file. |
| AUD-071 | `cmd/g0router/main_test.go` is compile-only placeholder with zero assertions | `cmd/g0router/main_test.go:5-11` | EXTRA | Write real CLI tests or delete file. |
| AUD-072 | `internal/schemas/schemas_test.go:251-289` `TestSchemaTypesCompile` is redundant — types already compiled by preceding tests | `internal/schemas/schemas_test.go:251` | EXTRA | Delete `TestSchemaTypesCompile`. |
| AUD-073 | `internal/inference/router_test.go` covers only happy path — no unknown/empty model error cases | `internal/inference/router_test.go:9-77` | DEBT | Add error-case table tests. |
| AUD-074 | `ui/AGENTS.md` missing — no UI-track agent instructions | N/A | DEBT | Create `ui/AGENTS.md` with UI conventions. |
| AUD-075 | `ui/src/lib/types` missing — 36 types imported by e2e mocks do not exist | `ui/e2e/mocks/store.ts:3-36` | BROKEN | Create `ui/src/lib/types.ts` with all mock-assumed types. |
| AUD-076 | `ui/src/routes/` has only `__root.tsx` — zero page routes for 30+ e2e specs | `ui/src/routeTree.gen.ts` lines show `fullPaths: never` | BROKEN | Implement routes or mark e2e specs as pending. |
| AUD-077 | `ui/src/App.tsx` is a static placeholder — no components, no routing, no API client | `ui/src/App.tsx:1-34` | BROKEN | Build real UI or remove e2e specs until UI exists. |
| AUD-078 | `ui/package.json` declares ~55 dependencies but `src/` imports none of them (placeholder UI) | `ui/package.json` dependencies vs `grep -r` in `ui/src/` | EXTRA | Remove unused deps or implement features that use them. |
| AUD-079 | `ui/package.json` `test` script is `echo 'No unit tests configured' && exit 0` | `ui/package.json:7` | EXTRA | Configure vitest/jest or remove script. |
| AUD-080 | Schema types `VirtualKey`, `PricingEntry`, `ModelCapability`, `Cost`, `MCPClient`, `MCPInstance`, `MCPTool`, `MCPToolGroup` defined but never referenced outside `schemas_test.go` | `internal/schemas/governance.go`, `internal/schemas/catalog.go`, `internal/schemas/mcp.go` | EXTRA | Delete until phase that implements them, or document as forward-declared. |
| AUD-081 | `EmbeddingRequest.EncodingFormat`, `Dimensions`, `User` never read by Gemini embedding converter | `internal/schemas/embedding.go:7-9`, `internal/providers/gemini/converter.go:284-299` | PARTIAL | Map or document unsupported fields; confirm via A1 matrix. |
| AUD-082 | `ListModelsResponse` / `ModelEntry` used only by OpenAI provider; Anthropic/Gemini stubs return `not_implemented` | `internal/providers/openai/models.go:10-51` | OK | Acceptable for Phase 5; only OpenAI has live model list. |
| AUD-083 | `server/server.go` sets `ReadTimeout: 0`, `WriteTimeout: 0` — no request timeouts | `internal/server/server.go:38-40` | DEBT | Set finite timeouts or document infinite intent. |
| AUD-084 | `admin/respond.go` `writeEnvelope` handles `json.Marshal` error with fallback but fallback itself is hardcoded JSON | `internal/admin/respond.go:28-32` | DEBT | Use `fmt.Fprintf` fallback to avoid second marshal failure. |
| AUD-085 | Anthropic `ConvertStreamEventToChunk` maps `input_json_delta` to `Delta.Content` — should map to tool-call partial JSON | `internal/providers/anthropic/converter.go:265-267` | PARTIAL | Verify against A1 matrix for tool-call streaming shape. |
| AUD-086 | All provider stub files (openai/anthropic/gemini) contain 20+ methods returning `nil, notImplemented(...)` with no TODO comments | `openai/stubs.go`, `anthropic/stubs.go`, `gemini/stubs.go` | DEBT | Add `// TODO(phase-N): implement` to each stub method. |
| AUD-087 | `TestNewWithoutStoreSkipsAdminRoutes` weak assertion — passes on 404, 500, or any non-200 body | `internal/server/server_test.go:149-162` | DEBT | Assert exact 404 or route-not-found behavior. |
| AUD-088 | Protected-route 401 loop in `server_test.go` omits `PUT/DELETE /api/providers/{id}`, `PUT/DELETE /api/connections/{id}`, `POST /api/connections/{id}/refresh`, `POST /api/oauth/{provider}/callback`, `POST /api/auth/logout` | `internal/server/server_test.go:69-94` | DEBT | Extend loop to cover all registered protected routes. |
| AUD-089 | `TestOpenEnablesWALAndIsIdempotent` does not assert WAL is still on after second open | `internal/store/store_test.go:55-85` | DEBT | Add second `PRAGMA journal_mode` check after reopen. |

---

## Remediation plan seeds

### Wave 0.1 — Error handling (BROKEN)
**Covers:** AUD-001, AUD-002, AUD-003, AUD-007, AUD-009, AUD-010, AUD-011, AUD-012, AUD-031, AUD-035, AUD-038, AUD-039, AUD-040, AUD-041, AUD-042, AUD-045, AUD-049, AUD-077
- Fix all swallowed `rand.Read` errors in `store`, `auth`.
- Fix all ignored `json.Marshal`/`Write` errors in `api/*`.
- Fix converter silent drops: multiple system messages (anthropic + gemini), tool-call ID loss, empty response IDs, colliding tool-call IDs, ignored JSON unmarshal in Gemini tool args.
- Fix Gemini chat model sanitization (call `sanitizeModelName` or unify).
- Fix SSE parse errors: abort stream instead of `continue`.
- Decision on UI: either stub e2e specs as pending or scaffold routes + types so 30 specs can begin passing.

### Wave 0.2 — Security & conventions (BROKEN + DEBT)
**Covers:** AUD-004, AUD-005, AUD-006, AUD-013, AUD-014, AUD-015, AUD-029, AUD-030
- Move hardcoded OAuth `client_id` to env/config.
- Harden `ensureColumn` against injection (whitelist or parameterized).
- Add SQLite `FOREIGN KEY` constraints.
- Fix CORS origin reflection + credentials.
- Fix `pathID` type assertion.
- Add `ProviderID` validation in `UpdateConnection`.
- Document OpenAI-surface envelope exception in AGENTS.md or unify envelope shapes.

### Wave 0.3 — Dead code & schema cleanup (EXTRA + DEBT)
**Covers:** AUD-016, AUD-017, AUD-018, AUD-019, AUD-020, AUD-021, AUD-022, AUD-023, AUD-048, AUD-053-AUD-068, AUD-069, AUD-070, AUD-071, AUD-072, AUD-078, AUD-079, AUD-080, AUD-086
- Delete all 15 stub packages (30 files).
- Delete redundant `TestSchemaTypesCompile` and compile-only placeholders in `api`, `inference`, `cmd/g0router`.
- Remove dead functions `setAuthHeader`, `buildModelURI` from `gemini/chat.go`.
- Remove or fill unused schema columns/fields (`oauth_sessions.created_at`, `settings.updated_at`, etc.).
- Remove unused UI dependencies (~55 packages) or implement the features that justify them.
- Remove forward-declared schema types (`VirtualKey`, `PricingEntry`, `MCPClient`, etc.) until their phases arrive.
- Add TODO comments to all provider stub methods.

### Wave 0.4 — Test quality (DEBT + EXTRA)
**Covers:** AUD-073, AUD-087, AUD-088, AUD-089
- Add error-case table tests to `router_test.go` (unknown model, empty string).
- Strengthen `TestNewWithoutStoreSkipsAdminRoutes` assertion.
- Extend protected-route 401 loop to cover all admin routes.
- Add second WAL assertion in `TestOpenEnablesWALAndIsIdempotent`.

### Wave 0.5 — Duplication & abstraction (DEBT)
**Covers:** AUD-024, AUD-025, AUD-026, AUD-027, AUD-028, AUD-050, AUD-051, AUD-052
- Extract `newTestStore`, fake OAuth server, HTTP call helpers into `internal/testutil`.
- Extract `writeJSON` and `writeProviderError` helpers in `api` package.
- Extract generic provider base struct/constructor to reduce openai/anthropic/gemini duplication.
- Extract generic HTTP JSON round-trip and SSE stream helpers to reduce chat/embedding duplication.

### Wave 0.6 — Converter completeness (PARTIAL)
**Covers:** AUD-032, AUD-033, AUD-034, AUD-036, AUD-037, AUD-043, AUD-081, AUD-085
- Build A1 translation matrix for every `schemas.ChatRequest` field → Anthropic/Gemini mapping.
- Confirm each PARTIAL row: map field, document unsupported, or add `X-Unsupported-Header`.
- Fix Gemini error converter integer-code drop.
- Verify `input_json_delta` streaming mapping for Anthropic tool calls.

---

## Keep/Delete verdicts

| Package | Verdict | Rationale |
|---|---|---|
| `internal/catalog` | **DELETE** | doc.go + empty test; zero imports; no types, no interfaces, no functions. |
| `internal/config` | **DELETE** | doc.go + empty test; zero imports; no types, no interfaces, no functions. |
| `internal/governance` | **DELETE** | doc.go + empty test; zero imports; no types, no interfaces, no functions. |
| `internal/logging` | **DELETE** | doc.go + empty test; zero imports; no types, no interfaces, no functions. |
| `internal/mcp` | **DELETE** | doc.go + empty test; zero imports; no types, no interfaces, no functions. |
| `internal/platform` | **DELETE** | doc.go + empty test; zero imports; no types, no interfaces, no functions. |
| `internal/providers/bedrock` | **DELETE** | doc.go + empty test; zero imports; no types, no interfaces, no functions. |
| `internal/providers/cohere` | **DELETE** | doc.go + empty test; zero imports; no types, no interfaces, no functions. |
| `internal/providers/deepseek` | **DELETE** | doc.go + empty test; zero imports; no types, no interfaces, no functions. |
| `internal/providers/fireworks` | **DELETE** | doc.go + empty test; zero imports; no types, no interfaces, no functions. |
| `internal/providers/groq` | **DELETE** | doc.go + empty test; zero imports; no types, no interfaces, no functions. |
| `internal/providers/minimax` | **DELETE** | doc.go + empty test; zero imports; no types, no interfaces, no functions. |
| `internal/providers/mistral` | **DELETE** | doc.go + empty test; zero imports; no types, no interfaces, no functions. |
| `internal/providers/ollama` | **DELETE** | doc.go + empty test; zero imports; no types, no interfaces, no functions. |
| `internal/providers/together` | **DELETE** | doc.go + empty test; zero imports; no types, no interfaces, no functions. |
| `internal/providers/vertex` | **DELETE** | doc.go + empty test; zero imports; no types, no interfaces, no functions. |

---

*End of audit.*
