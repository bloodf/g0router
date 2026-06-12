# PARITY.md — g0router master parity index

Reference SHAs frozen per `SOURCES.md`:
- 9router `827e5c3` (v0.4.71)
- Bifrost `ca21298`

North star: g0router is a total replacement. v1.0 = drop-in 9router replacement. v1.1 = Bifrost OpenAI surface + MCP gateway. Stage 3 = Bifrost governance, adaptive LB, semantic cache, observability, cluster mode.

---

## 1. Rollup

| Domain | File | Total rows | HAVE | PARTIAL | MISSING | EXTRA | Stage |
|---|---|---:|---:|---:|---:|---:|---|
| 9router Translation | `matrix/9router-translation.md` | 55 | 4 | 5 | 46 | 0 | Stage 1 |
| 9router Providers | `matrix/9router-providers.md` | 67 | 3 | 0 | 63 | 1 | Stage 1 |
| 9router Routing | `matrix/9router-routing.md` | 60 | 13 | 2 | 45 | 0 | Stage 1 |
| 9router Auth | `matrix/9router-auth.md` | 30 | 6 | 1 | 23 | 0 | Stage 1 |
| 9router Usage | `matrix/9router-usage.md` | 40 | 0 | 0 | 40 | 0 | Stage 1 |
| 9router UI | `matrix/9router-ui.md` | 133 | 9 | 2 | 117 | 5 | Stage 1 |
| 9router MCP | `matrix/9router-mcp.md` | 60 | 2 | 0 | 57 | 1 | Stage 1 |
| 9router Platform | `matrix/9router-platform.md` | 50 | 2 | 2 | 46 | 0 | Stage 1 |
| Bifrost OpenAI | `matrix/bifrost-openai.md` | 91 | 5 | 2 | 83 | 1 | Stage 2 |
| Bifrost MCP | `matrix/bifrost-mcp.md` | 80 | 0 | 0 | 80 | 0 | Stage 2 |
| Bifrost Core | `matrix/bifrost-core.md` | 50 | 1 | 3 | 46 | 0 | Stage 3 |
| Bifrost Governance | `matrix/bifrost-governance.md` | 50 | 0 | 2 | 48 | 0 | Stage 3 |
| g0router Audit | `matrix/g0router-audit.md` | 89 findings | — | — | 7 PARTIAL | 22 BROKEN / 32 DEBT / 27 EXTRA / 1 OK | Wave 0 |
| PR ports (Stage 1) | PARITY.md §3 | 101 | 0 | 0 | 101 | 0 | Stage 1 |

Stage 1 implementable rows (HAVE+PARTIAL+MISSING): 488 matrix + 101 PR-port = 589. Stage 1 EXTRA rows (g0router-only, keep-or-drop decisions): 7.
Stage 2 implementable rows (HAVE+PARTIAL+MISSING): 170. Stage 2 EXTRA rows: 1.
Stage 3 implementable rows (HAVE+PARTIAL+MISSING): 100. Stage 3 EXTRA rows: 0.
PR-port accounting: 129 ACCEPT PRs total = 101 new PAR-PR-* rows (counted in the rollup line above) + 28 PRs that amend existing PAR-<DOMAIN>-NNN rows (no new rows; tracked as amendments on those rows). 101 + 28 = 129; "Coverage: 129/129" in §3 counts PRs, not rows.

Stage 1 exit covers matrix rows AND PAR-PR rows; both gate at 100% HAVE.

---

## 2. Wave 0 — remediation

**STATUS: COMPLETE (2026-06-09).** All five bundles (A-E → plans w0-a..w0-e) merged to main with diff gates closed; dispositions in `GATE-RESOLUTION.md`. All 22 BROKEN + 10 co-located DEBT rows are fixed.

These bundles group audit findings for plan factory consumption. Each bundle becomes one micro-plan in `.planning/parity/plans/` with TDD task ordering and binary acceptance criteria; no implementation happens from this document.

All **BROKEN** findings from `g0router-audit.md`, grouped into fix bundles. DEBT findings included only when they share a file with a BROKEN fix.

### Bundle A — crypto rand + marshal/write error handling
**Files:** `internal/store/store.go`, `internal/auth/session.go`, `internal/auth/oauth.go`, `internal/api/chat.go`, `internal/api/embeddings.go`, `internal/api/models.go`, `internal/api/errors.go`, `internal/providers/openai/chat.go`, `internal/providers/anthropic/chat.go`, `internal/providers/gemini/chat.go`

| AUD ID | Finding (verbatim from g0router-audit.md) | Binary acceptance check |
|---|---|---|
| AUD-001 | `rand.Read` error discarded in `newID()` — produces weak/empty IDs on failure | unit test proves `newID` returns error when `rand.Read` fails |
| AUD-002 | `rand.Read` error discarded in `newToken()` — session tokens can be empty | unit test proves `newToken` returns error when `rand.Read` fails |
| AUD-003 | `rand.Read` error discarded in `randomURLSafe()` — OAuth state/verifier can be empty | unit test proves `randomURLSafe` returns error when `rand.Read` fails |
| AUD-007 | `json.Marshal` error ignored — can emit empty/invalid SSE chunks | unit test proves SSE handler aborts on `json.Marshal` error |
| AUD-008 | `ctx.WriteString`/`ctx.Write` errors discarded during SSE streaming — client disconnects undetected | unit test proves stream aborts when `ctx.Write` returns error |
| AUD-009 | `json.Marshal` error ignored in non-streaming chat response | unit test proves chat handler returns 500 on `json.Marshal` error |
| AUD-010 | `json.Marshal` error ignored in embeddings response | unit test proves embeddings handler returns 500 on `json.Marshal` error |
| AUD-011 | `json.Marshal` error ignored in models response | unit test proves models handler returns 500 on `json.Marshal` error |
| AUD-012 | `json.Marshal` error ignored in error response writer | unit test proves error writer uses plain-text fallback when `json.Marshal` fails |
| AUD-045 | Streaming SSE JSON unmarshal errors silently skipped via `continue` in all three providers | unit test proves provider stream aborts on JSON unmarshal error |

### Bundle B — security: CORS, secrets, DDL
**Files:** `internal/server/middleware.go`, `internal/auth/oauth.go`, `internal/store/migrate.go`, `internal/admin/handlers.go`, `internal/admin/connections.go`

| AUD ID | Finding (verbatim from g0router-audit.md) | Binary acceptance check |
|---|---|---|
| AUD-004 | Hardcoded production Anthropic OAuth `client_id` baked into source | `go test ./...` green; no hardcoded `client_id` in `internal/auth/oauth.go` |
| AUD-005 | `ensureColumn` constructs SQL via `fmt.Sprintf` with unsanitized params — injection vector if ever called dynamically | `go test ./...` green; `ensureColumn` whitelists column names |
| AUD-006 | SQLite schema has `NOT NULL` on foreign-key columns but no `FOREIGN KEY` constraints | `go test ./...` green; schema migration adds `REFERENCES ... ON DELETE` clauses |
| AUD-013 | `pathID` silently returns `""` for non-string route params | unit test proves `pathID` returns error on bad type assertion |
| AUD-014 | `UpdateConnection` does not validate `ProviderID` exists when changed | unit test proves `UpdateConnection` rejects non-existent `ProviderID` |
| AUD-015 | CORS middleware reflects any `Origin` header and sets `Allow-Credentials: true` — CSRF risk | `go test ./...` green; CORS whitelists origins or removes credentials header |

### Bundle C — provider converter silent data loss
**Files:** `internal/providers/anthropic/converter.go`, `internal/providers/gemini/converter.go`, `internal/providers/gemini/chat.go`

| AUD ID | Finding (verbatim from g0router-audit.md) | Binary acceptance check |
|---|---|---|
| AUD-031 | Anthropic `ConvertRequest` silently drops multiple system messages — only last kept | unit test proves multiple system messages are concatenated or error |
| AUD-035 | Gemini `ConvertChatRequest` silently drops multiple system messages — only last kept | unit test proves multiple system messages are concatenated or error |
| AUD-038 | Gemini `convertMessages` loses `Message.ToolCallID` for tool-role messages | unit test proves `ToolCallID` propagates into Gemini function response metadata |
| AUD-039 | Gemini `ConvertChatResponse` leaves `ChatResponse.ID` empty | unit test proves `ConvertChatResponse` returns non-empty `ID` |
| AUD-040 | Gemini `ConvertStreamChunk` leaves `StreamChunk.ID` empty | unit test proves `ConvertStreamChunk` returns non-empty `ID` |
| AUD-041 | Gemini tool-call IDs collide when same function invoked twice: `"call_" + name` | unit test proves repeated tool calls receive unique IDs |
| AUD-042 | Gemini `json.Unmarshal` error ignored in tool argument parsing — passes nil args silently | unit test proves malformed JSON tool arguments return error |
| AUD-049 | `gemini/chat.go` does not call `sanitizeModelName` on chat model — passes `"gemini/gemini-1.5-pro"` to URI | unit test proves chat path calls `sanitizeModelName` before URI build |

### Bundle D — UI scaffolding
**Files:** `ui/src/lib/types.ts` (new), `ui/src/routes/`, `ui/src/App.tsx`

| AUD ID | Finding (verbatim from g0router-audit.md) | Binary acceptance check |
|---|---|---|
| AUD-075 | `ui/src/lib/types` missing — 36 types imported by e2e mocks do not exist | `ui/src/lib/types.ts` exists and exports the 36 types; `npm run build` green |
| AUD-076 | `ui/src/routes/` has only `__root.tsx` — zero page routes for 30+ e2e specs | at least one route file exists under `ui/src/routes/` besides `__root.tsx`; `npm run build` green |
| AUD-077 | `ui/src/App.tsx` is a static placeholder — no components, no routing, no API client | `App.tsx` imports and mounts TanStack Router root component; `npm run build` green |

### Bundle E — provider streaming and SSE shape
**Files:** `internal/api/chat.go`, all provider `chat.go`

| AUD ID | Finding (verbatim from g0router-audit.md) | Binary acceptance check |
|---|---|---|
| AUD-032 | Anthropic `ConvertRequest` ignores `schemas.ChatRequest.N` | unit test maps `N` or documents unsupported; `go vet + test green` |
| AUD-033 | Anthropic `ConvertRequest` ignores `PresencePenalty`, `FrequencyPenalty`, `LogitBias`, `User`, `ResponseFormat`, `Seed` | unit test maps each field or documents unsupported; `go vet + test green` |
| AUD-034 | Anthropic `convertMessages` ignores `Message.Name` | unit test maps `Name` or documents unsupported; `go vet + test green` |
| AUD-036 | Gemini `ConvertChatRequest` ignores `N`, `Stream`, `PresencePenalty`, `FrequencyPenalty`, `LogitBias`, `User`, `ResponseFormat`, `Seed` | unit test maps each field or documents unsupported; `go vet + test green` |
| AUD-037 | Gemini `convertMessages` ignores `Message.Name` | unit test maps `Name` or documents unsupported; `go vet + test green` |
| AUD-046 | Streaming non-EOF scanner errors break loop but do not propagate to caller | unit test proves scanner error is propagated to caller |
| AUD-047 | `postHookRunner.Run` errors discarded in all three providers | unit test proves hook errors are surfaced instead of discarded |
| AUD-081 | `EmbeddingRequest.EncodingFormat`, `Dimensions`, `User` never read by Gemini embedding converter | unit test maps each field or documents unsupported; `go vet + test green` |
| AUD-085 | Anthropic `ConvertStreamEventToChunk` maps `input_json_delta` to `Delta.Content` — should map to tool-call partial JSON | unit test proves `input_json_delta` maps to tool-call `Delta`; `go vet + test green` |

---

## 3. PR ports

All 129 ACCEPT PRs from `a3-shortlist.md` mapped to the domain matrix where the behavior lands. PR-port rows use ID `PAR-PR-<number>`; they are first-class parity rows owned by this document; rows citing `PAR-<DOMAIN>-NNN` are amendments to that existing row.

### Mapped to `9router-translation.md` (TRANS)

| PR | Related PAR row(s) or PAR-PR-ID | One-line behavior |
|---|---|---|
| #1752 | PAR-PR-1752 | Fix Gemini thought parts leaking into assistant content |
| #1742 | PAR-PR-1742 | Strip unsupported `client_metadata` from cerebras/mistral requests |
| #1717 | PAR-TRANS-031 | Codex API compatibility and SSE translation fixes |
| #1701 | PAR-PR-1701 | Forward connection-level proxy to Gemini/OpenAI embedding requests |
| #1666 | PAR-PR-1666 | Mask request debug logs |
| #1652 | PAR-PR-1652 | Add mappable "auto" model slot for Kiro agent mode |
| #1626 | PAR-PR-1626 | OpenAI-compatible token parameter auto-learning fallback |
| #1615 | PAR-TRANS-039, PAR-TRANS-040 | Antigravity protocol fidelity (ideVersion, User-Agent, host, headers) |
| #1599 | PAR-PR-1599 | Strip reasoning blobs from agentic context |
| #1585 | PAR-TRANS-010, PAR-TRANS-055 | Sanitize system role in passthrough mode for Claude provider |
| #1573 | PAR-PR-1573 | Emit valid concatenable tool_calls.arguments deltas for Kiro |
| #1568 | PAR-PR-1568 | Prevent false stall aborts on large-context reasoning streams |
| #1506 | PAR-PR-1506 | Reject unsupported Kiro 1m context suffix |
| #1505 | PAR-PR-1505 | Dedupe Anthropic version headers |
| #1500 | PAR-TRANS-055 | Strip Claude context management for compatible providers |
| #1488 | PAR-TRANS-031 | Responses API `max_tokens` mapping |
| #1463 | PAR-PR-1463 | Resolve Cursor cu/default empty responses |
| #1460 | PAR-PR-1460 | Preserve reasoning effort for Codex translations |
| #1412 | PAR-PR-1412 | Replay reasoning content for thinking tool calls |
| #1410 | PAR-PR-1410 | Ensure correct `anthropic-version` header for variations |
| #1402 | PAR-PR-1402 | Route Codex auto-review to Codex provider |
| #1401 | PAR-TRANS-041 | Fix `/v1/messages` non-streaming JSON mode |
| #1397 | PAR-TRANS-031 | Preserve Codex `custom_tool_call` shape through translator |
| #1387 | PAR-TRANS-017 | Inject `json_schema` into system prompt for openai-compatible providers |
| #1384 | PAR-PR-1384 | Add `input_type` param for NVIDIA nv-embedqa-e5-v5 |
| #1383 | PAR-TRANS-040 | Preserve required fields in Antigravity tool schemas |
| #1349 | PAR-PR-1349 | Normalize `anthropic-version` header |
| #1347 | PAR-PR-1347 | Persist custom model token limits |
| #1344 | PAR-TRANS-017 | Downgrade `json_schema` to `json_object` for non-OpenAI providers |
| #1337 | PAR-PR-1337 | Fix Xiaomi reasoning content echo |
| #1331 | PAR-PR-1331 | Support AI SDK image parts in translators |
| #1316 | PAR-PR-1316 | Strip Cursor Composer `<｜final｜>` sentinel markers |
| #1272 | PAR-ROUTE-043 | Default `stream` to false per OpenAI spec |
| #1264 | PAR-PR-1264 | Strip temperature for Claude models with extended thinking |
| #1193 | PAR-PR-1193 | Responses API MCP namespace round-trip + DeepSeek thinking suffix |
| #1170 | PAR-PR-1170 | Rescue Kiro compaction requests missing `toolConfig` |
| #1148 | PAR-PR-1148 | Drop empty `data: null` SSE events between chunks |
| #1115 | PAR-PR-1115 | Prevent `reasoning_content` from being deleted unconditionally |
| #1095 | PAR-PR-1095 | Strip built-in tools when functionDeclarations present (Antigravity) |
| #1084 | PAR-PR-1084 | Drop literal `<think>`/`</think>` markers from Claude→OpenAI stream |
| #1054 | PAR-PR-1054 | Support OpenAI `max_completion_tokens` parameter |
| #1007 | PAR-TRANS-031 | Normalize Codex custom tools (`apply_patch`) to `{input:string}` schema |
| #1004 | PAR-PR-1004 | Stabilize Codex continue sessions |
| #980 | PAR-PR-980 | Stop leaking think tags in OpenAI streams |
| #976 | PAR-PR-976 | Preserve Codex reasoning summary deltas |
| #952 | PAR-PR-952 | Preserve Claude thinking signatures |
| #909 | PAR-PR-909 | Fix BytePlus endpoint and hide disabled models |
| #882 | PAR-PR-882 | Decloak Claude tool names on every client-bound path |
| #875 | PAR-PR-875 | Fix empty Anthropic thinking blocks |
| #873 | PAR-PR-873 | Strip unsupported n8n Responses API params for Codex |
| #721 | PAR-PR-721 | Suppress null Responses SSE frames and preserve completed output |
| #717 | PAR-PR-717 | Align Codex OAuth models and request contract |
| #664 | PAR-PR-664 | Translate `max_tokens` to `max_completion_tokens` for openai-compatible providers |
| #656 | PAR-PR-656 | Strip X-Stainless headers and normalize SDK User-Agent |
| #651 | PAR-PR-651 | Translate non-streaming Ollama `tool_calls` for OpenAI SDK |
| #586 | PAR-PR-586 | Forward client headers in native passthrough mode |
| #485 | PAR-PR-485 | Use providerId for passthrough model alias lookup |
| #468 | PAR-TRANS-007, PAR-TRANS-008, PAR-TRANS-009, PAR-TRANS-055 | Skip modifications for same-format requests (lossless passthrough) |
| #422 | PAR-TRANS-028 | Coerce string numeric JSON Schema constraints to integers |
| #421 | PAR-TRANS-028 | Coerce tool description to string in all translation paths |
| #420 | PAR-TRANS-040 | Convert Gemini body to OpenAI format in antigravity MITM handler |
| #349 | PAR-TRANS-028 | Sanitize OpenAI tool schemas for strict Codex validation |
| #345 | PAR-PR-345 | Preserve `tool_calls` during SSE-to-JSON reassembly |
| #337 | PAR-ROUTE-020 | Retry transient errors before falling through combo chain |
| #645 | PAR-PR-645 | Force Agent mode when Claude CLI routes through Cursor provider |
| #1209 | PAR-PR-1209 | Resolve Kiro improperly formed request when tool_calls in history |
| #1425 | PAR-PR-1425 | Default Codex reasoning to medium |
| #286 | PAR-TRANS-047, PAR-TRANS-050 | Fix SSE `data: [DONE]` sentinel for non-streaming requests |

### Mapped to `9router-providers.md` (PROV)

| PR | Related PAR row(s) or PAR-PR-ID | One-line behavior |
|---|---|---|
| #1560 | PAR-PR-1560 | Add Crof AI provider support |
| #1700 | PAR-PR-1700 | Show edited provider names consistently across views |
| #1450 | PAR-PR-1450 | Bound provider connection tests |
| #1448 | PAR-PR-1448 | Show disabled connection errors |
| #1427 | PAR-PR-1427 | Keep quota accounts active until all pools empty |
| #1421 | PAR-ROUTE-005 | Clean up provider model aliases on provider delete |
| #1338 | PAR-PR-1338 | Fix provider client priority sorting |
| #1211 | PAR-PR-1211 | Use public v1 endpoints for model reachability tests |
| #1164 | PAR-PR-1164 | Migrate Cohere provider to v2 API |
| #1375 | PAR-PR-1375 | Fix Kiro "Improperly formed request" |
| #1374 | PAR-PR-1374 | Remove AI-generated intro |
| #702 | PAR-PR-702 | Separate usage tracking for Ollama Cloud vs local |
| #693 | PAR-PROV-010 | Mark Ollama provider as noAuth |
| #665 | PAR-PR-665 | Add API key support for OpenCode Go provider |
| #657 | PAR-PR-657 | Strip provider prefixes from remote `/models` IDs |
| #655 | PAR-PR-655 | Strip `enumDescriptions` from VSCode Copilot tool schemas for Antigravity |
| #652 | PAR-PR-652 | Guard against corrupt JSON in request-details DB |
| #649 | PAR-PR-649 | Allow per-connection refresh lead time override |
| #644 | PAR-PR-644 | Make version update banner clickable |
| #642 | PAR-PR-642 | Detect localhost Docker networking and suggest host IP |
| #641 | PAR-PR-641 | Add qwen-code provider for Qwen OAuth with user_code |
| #637 | PAR-PR-637 | Fetch OpenRouter embedding/tts models dynamically |
| #629 | PAR-PR-629 | Guard against null content parts |
| #628 | PAR-PR-628 | Strip default values from tool schema in antigravity-to-openai |
| #518 | PAR-PR-518 | Fetch API Key Compatible provider models for `/v1/models` |

### Mapped to `9router-routing.md` (ROUTE)

| PR | Related PAR row(s) or PAR-PR-ID | One-line behavior |
|---|---|---|
| #1685 | PAR-PR-1685 | Resolve nested standalone build path |
| #1672 | PAR-PR-1672 | Increase validation `max_tokens` to 10 |
| #1434 | PAR-ROUTE-010 | Prevent circular dependencies in model combos |
| #1418 | PAR-PR-1418 | Abort stalled initial upstream responses for Codex |
| #1385 | PAR-AUTH-024 | CORS preflight improvements for browser/WebView clients |
| #1346 | PAR-PR-1346 | Migrate deprecated Codex hooks config |
| #1249 | PAR-PR-1249 | Fix OAuth redirect URI handling for remote deployments |
| #1207 | PAR-PR-1207 | Fix SSRF in suggested-models endpoint |
| #995 | PAR-PR-995 | Add response headers for model tracking and observability |
| #953 | PAR-ROUTE-010 | Reject circular combo references |
| #947 | PAR-ROUTE-002 | Advance round-robin pointer past fallback-served model |
| #879 | PAR-ROUTE-014, PAR-ROUTE-045 | Parse `retryAfter` timestamp for precise 429 backoff |
| #640 | PAR-PR-640 | Prevent infinite retry loop when all accounts error |
| #648 | PAR-PR-648 | Reset models state on combo prop change |
| #474 | PAR-PR-474 | Resolve bare model names to connection `defaultModel` |
| #339 | PAR-PR-339 | Show model name instead of raw ID in combo list |
| #520 | PAR-PR-520 | Secure dangerous unauthenticated global package install endpoint |
| #1121 | PAR-PR-1121 | Restore automatic Update Now relaunch |
| #1455 | PAR-PR-1455 | Add Codex subagent description |

### Mapped to `9router-auth.md` (AUTH)

| PR | Related PAR row(s) or PAR-PR-ID | One-line behavior |
|---|---|---|
| #1711 | PAR-PR-1711 | Persistent dashboard session cookie — 30d TTL, unified parser |
| #1498 | PAR-AUTH-014 | Reset CLI password against SQLite settings |
| #1458 | PAR-PR-1458 | Validate HuggingFace tokens via whoami |
| #1445 | PAR-PR-1445 | Distinguish API-banned account from invalid token; preserve refreshed RT |
| #1388 | PAR-PR-1388 | Fix Kiro social refresh endpoint for token import |
| #1247 | PAR-PR-1247 | Harden public API and local-only access gates |

### Mapped to `9router-usage.md` (USAGE)

| PR | Related PAR row(s) or PAR-PR-ID | One-line behavior |
|---|---|---|
| #1738 | PAR-USAGE-010 | Count alternate usage token fields (input_tokens/output_tokens synonyms) |
| #424 | PAR-PR-424 | Correct stats truncation at 10k requests and make history cap configurable |

### Mapped to `9router-platform.md` (PLAT)

| PR | Related PAR row(s) or PAR-PR-ID | One-line behavior |
|---|---|---|
| #1509 | PAR-PR-1509 | Autofill proxy URL for connectivity tests |
| #1198 | PAR-PR-1198 | Stop tryRestart orphan race breaking Hide to Tray |
| #1100 | PAR-PR-1100 | Fix droid settings and streaming passthrough |
| #884 | PAR-PR-884 | Preserve proxy pool on token refresh; use proxy-aware fetch in model sync |
| #447 | PAR-PR-447 | Bootstrap `db.json` on first run and set `DATA_DIR` in Docker |
| #399 | PAR-PR-399 | Respect `DATA_DIR` and `XDG_CONFIG_HOME` in all data paths |

### Mapped to `9router-ui.md` (UI)

| PR | Related PAR row(s) or PAR-PR-ID | One-line behavior |
|---|---|---|
| #1416 | PAR-PR-1416 | Add Codex subagent role description in UI |
| #1348 | PAR-PR-1348 | Honor `TAILSCALE_AUTHKEY` in Tailscale login UI |
| #650 | PAR-PR-650 | Remove false `fs` dependency from `package.json` |

### Unmapped

None. All 129 ACCEPT PRs map to a Stage 1 domain matrix. No PRs are unmappable.

Coverage: 129/129 ACCEPT PRs mapped (101 as new PAR-PR rows, 28 as amendments to existing PAR rows), 0 duplicate PR entries.

---

## 4. Issue cross-check

KNOWN-BROKEN issues from `a3-shortlist.md` that describe 9router bugs g0router must **not** replicate. These become negative test cases. All listed issues intersect Stage 1 domains. Issues are grouped by domain tag; each issue ID and headline traces directly to the `KNOWN-BROKEN` table in `a3-shortlist.md`.

### TRANS — translation / inference correctness (74 issues)

| Issue | Behavior to avoid |
|---|---|
| #1758 | Codex subagent tools fail when `multi_agent_v2` uses encrypted params |
| #1756 | OpenAI-compatible endpoint returns "No active credentials" incorrectly |
| #1749 | Gemma 4 thinking output leaks into final response content |
| #1748 | `temperature` causes HTTP 400 on Claude Opus 4 and newer Anthropic models |
| #1745 | `max_tokens` unsupported on newer OpenAI models (gpt-5.4 family) |
| #1703 | Stream stall timeout not configurable for Qoder; proxy ignored |
| #1702 | OpenAI-compatible endpoint failure from `chat.b.ai` |
| #1667 | Qoder Error 403 Code 112 on any prompt |
| #1663 | Codex OAuth auto-refresh incomplete; token revoked after days |
| #1649 | Mistral fails with `reasoning_content` when Codex `/responses` used |
| #1645 | Cloudflare Workers AI truncates responses prematurely |
| #1639 | Overview zero-token usage bug |
| #1625 | GitHub provider Gemini responses empty / prompt-echo |
| #1621 | Stream stall timeout proxying Claude Code to Xiaomi mimo-v2.5 |
| #1603 | Claude returns "role 'system' is not supported on this model" |
| #1580 | `cc/claude-sonnet-4-6` always fails with 400 system role |
| #1565 | `qd/*` models report identity as Qwen regardless of selection |
| #1564 | Claude Code fails with Antigravity due to invalid tool schema |
| #1563 | Claude Cowork missing; version rollback broken; forced update |
| #1553 | `/v1/models` misses models from noAuth providers (opencode) |
| #1545 | Tunnel errors (generic) |
| #1543 | DeepSeek V4 Flash thinking mode issue |
| #1537 | Antigravity and Gemini 400 errors |
| #1512 | Model IDs from `/v1/models` not directly usable in chat |
| #1503 | Kiro silently forwards `[1m]` context suffix → 400 |
| #1480 | kimi-k2.6 error |
| #1476 | Antigravity MITM problem |
| #1475 | Duplicate `anthropic-version` header failure |
| #1382 | DeepSeek ~0 output tokens for tool-heavy Claude→OpenAI; GLM 13% empty |
| #1371 | Codex `apply_patch` broken via `/v1/responses` |
| #1368 | Antigravity glob/grep schemas lose required `pattern` param |
| #1357 | Kiro provider timeout |
| #1356 | Antigravity IDE auth failure via MITM |
| #1355 | Kiro IDE workspace tools fail via MITM |
| #1352 | Cloudflare Tunnel fails on Docker/Podman |
| #1343 | `json_schema` response_format rejected with 400 |
| #1330 | AI SDK image parts dropped by multimodal OpenAI translators |
| #1321 | Xiaomi thinking models require `reasoning_content` passthrough |
| #1294 | Custom model `max_input_tokens`/`max_output_tokens` dropped |
| #1276 | Claude Code does not work in Combo |
| #1255 | Kiro 429 despite single project |
| #1237 | ETIMEDOUT on providers when IPv6 unreachable |
| #1227 | Duplicate `anthropic-version: "2023-06-01, 2023-06-01"` |
| #1206 | Hide to Tray shuts down app on Ubuntu/Linux |
| #1197 | Hide to Tray loses tray icon after update |
| #1192 | Migration 0.4.41→0.4.50 drops provider connections |
| #1189 | Gateway fails to handle DeepSeek `reasoning_content` param |
| #1188 | Antigravity missing from UI after update |
| #1187 | `INVALID_MODEL_ID` on Opencode |
| #1173 | Opencode returns Claude format on OpenAI endpoint |
| #1157 | OpenAI models fail with "Unknown parameter: 'client_metadata'" |
| #1145 | Wrong model list in Claude Code Extension |
| #1093 | OpenCode DeepSeek v4 flash does not work with Claude Code |
| #1078 | Missing vision/context metadata causes multimodal models treated as text-only |
| #1061 | Claude Code constant empty/incomplete responses in Antigravity |
| #1059 | Claude Code / Antigravity provider 403 |
| #1052 | Factory Droid BYOK crashes on `/v1/responses` for cx/gpt-5.5 |
| #1046 | OpenCode Free models missing from `/v1/models` |
| #1038 | System role ignored in Codex OAuth API calls |
| #1037 | Vercel Relay with Codex returns 403 without diagnostics |
| #1036 | 400 Invalid JSON: unknown name `ref` in function_declarations |
| #1028 | DeepSeek provider does not work on Codex |
| #972 | Azure OpenAI error |
| #966 | Double token usage |
| #948 | Round-robin combo pointer does not advance past fallback-served model |
| #943 | Kimi API key problem |
| #905 | Droid BYOK null type error |
| #896 | Tailscale failing |
| #876 | MITM server failed to start |
| #708 | Kiro invalid model error |
| #558 | MITM certificate install failed |
| #319 | Claude Code for VSCode on GA pointing issue |
| #1117 | Compact API endpoint fails with "Unsupported parameter: _compact" |
| #1008 | Cursor "Switch to Agent mode" stop flow |

### ROUTE — routing / gateway behavior (32 issues)

| Issue | Behavior to avoid |
|---|---|
| #1757 | OpenRouter API keys stop the process |
| #1753 | WSL starts slowly; dashboard inaccessible in background |
| #1682 | Combo with `kind="llm"` hidden in Dashboard |
| #1707 | 9router models do not provide editing tools |
| #1641 | Auto retry issue |
| #1534 | MCP tools collapsed into empty namespace via `/v1/responses` |
| #1519 | Tunnel connection fails repeatedly |
| #1470 | DNS resolve timeout for Kiro endpoint |
| #1468 | Claude Code fails with "context_management: Extra inputs are not permitted" |
| #1452 | `--model` flag only available for Amp employees |
| #1447 | Hidden per-connection lastError when `isActive=false` |
| #1444 | Codex wrong "Token invalid" label on API-banned account |
| #1438 | Memory not loaded across sessions |
| #1407 | Unable to load Codex ChatGPT Plus usage limits |
| #1398 | Error when using Auto-review codex |
| #1328 | OAuth account fails to consume `credit_freetrial` after main credit exhausted |
| #1326 | PyCharm + 9router issue |
| #1307 | `POST /v1/search` returns 502 when SearXNG down |
| #1299 | API empty/malformed response (HTTP 200) |
| #1284 | Hide to Tray on macOS not a real background handoff |
| #1263 | Tailscale login ignores `TAILSCALE_AUTHKEY` |
| #1253 | Kiro token import fails with Bad credentials while IDE session active |
| #1244 | Kiro Pro detected as Free / Claude Opus unavailable |
| #1235 | Infinite loop / circular dependency allowed in Combo |
| #1214 | MITM failed to start |
| #1160 | Dashboard HTTP 401 for provider model list when upstream returns 200 |
| #1123 | Claude cannot work with sub-agents (UI/TRANS boundary) |
| #1120 | Update Now falls back to Copy & Shutdown |
| #1101 | Cannot find module 'next' |
| #1025 | API empty/malformed response |
| #1022 | Cannot connect to GitHub Copilot provider |
| #1070 | Codex displays 'Authentication Failed' |

### PROV — provider adapter / credential behavior (10 issues)

| Issue | Behavior to avoid |
|---|---|
| #1696 | Failed to fetch providers: unexpected token 'I' |
| #1679 | Codex connected — Usage API temporarily unavailable (401) |
| #1640 | Qoder fail to prompt error 413 due to model ID change |
| #1617 | qwen3.7-max not supported in opencode go |
| #1442 | Opencode Go kimi-2.6 model error |
| #1323 | `GET /api/providers/client` 500 when sorting by priority |
| #1275 | Opencode Zen HTTP 500 system error |
| #1240 | Copilot MITM proxy.individual.githubcopilot.com |
| #1139 | OpenCode Go does not connect |
| #993 | Entering API key for fal does not work |

### AUTH — authentication / access control (1 issue)

| Issue | Behavior to avoid |
|---|---|
| #1482 | Password reset function invalid |

### PLAT — platform / runtime / deployment (21 issues)

| Issue | Behavior to avoid |
|---|---|
| #1692 | Hardcoded timeouts cause 85%+ errors |
| #1657 | better-sqlite3 install fails on Node 26 |
| #1605 | better-sqlite3 pruned by tray install → no SQLite driver |
| #1572 | Next.js 10MB body limit truncates `/v1/chat/completions` |
| #1529 | Large `/v1/responses` truncated by Next proxy 10MB limit |
| #1469 | `ReferenceError: Readable is not defined` on Node.js v24 |
| #1467 | `ReferenceError: Readable is not defined` v0.4.62 |
| #1457 | Kiro regression `ReferenceError: Readable is not defined` |
| #1451 | Kiro fails with `Readable.toWeb` missing |
| #1449 | Test Connection hangs forever when probe never returns |
| #1443 | Model list inaccessible after v0.4.62 update |
| #1409 | Deleting custom provider leaves orphaned model aliases |
| #1390 | sql.js fallback broken: sql-wasm.wasm missing |
| #1245 | Severe memory leak |
| #1106 | Almost all models failing despite quota |
| #1012 | Update banner compares against bundled app version |
| #1003 | No SQLite driver available |
| #987 | No SQLite driver available |
| #958 | Pipeline stream error with NVIDIA NIM |
| #1050 | Intermittent failure after 1-2 prompts |
| #940 | Cowork cannot call tool `ask_question` in plugin runtime |

### UI — dashboard / interface (0 issues)

#1123 spans the UI/TRANS boundary; counted once under ROUTE above. No UI-only issues.

**Skip summary:** 0 issues skipped. All KNOWN-BROKEN issues in `a3-shortlist.md` fall within Stage 1 domains. Total: 138 unique issues (74+32+10+1+21).

---

## 5. Conflicts and dedupe

Pairs of rows across matrices that describe the same or overlapping behavior. Canonical row chosen; deferred row noted.

| Pair | Canonical row | Deferred row | Rationale |
|---|---|---|---|
| `PAR-TRANS-053` (`dedupeTools`) ↔ `PAR-MCP-024` (strip built-in tools when MCP present) | `PAR-TRANS-053` | `PAR-MCP-024` | Tool deduplication is a translation-layer concern; `PAR-MCP-024` is deferred to Stage 2/3 because Bifrost MCP has broader VK-scoped tool filtering. |
| `PAR-TRANS-054` (Claude CLI warmup/count/title bypass) ↔ `PAR-ROUTE-034` (bypass patterns for Claude CLI) | `PAR-ROUTE-034` | `PAR-TRANS-054` | Bypass is a routing decision; translation matrix defers to routing. |
| `PAR-AUTH-019` (OAuth credential manager for provider connections) ↔ `PAR-PLAT-047` (g0router OAuth flow partial) | `PAR-AUTH-019` | `PAR-PLAT-047` | Auth domain owns OAuth flows; platform row is a partial-status observation. |
| `PAR-TRANS-046`–`050` (central SSE stream processor, passthrough, flush, reasoning injection) ↔ `PAR-BF-OAI-201`–`204` (SSE header timing, reader bypass, event format, `[DONE]` skip) | `PAR-BF-OAI-201`–`204` | `PAR-TRANS-046`–`050` | Bifrost defines the target SSE implementation; 9router stream utilities are Stage 1 interim and will be superseded. |
| `PAR-PROV-002` (anthropic provider adapter) ↔ `PAR-TRANS-012`–`022` (OpenAI→Claude translation details) | `PAR-PROV-002` | `PAR-TRANS-012`–`022` | Provider matrix owns the adapter existence; translation rows detail field mappings that hang under the adapter. |
| `PAR-TRANS-047` (SSE parser handles `data:`, `[DONE]`, `event:`, NDJSON) ↔ `PAR-BF-OAI-206` (chat completion streaming supported) | `PAR-TRANS-047` for Stage 1; `PAR-BF-OAI-206` for Stage 2 | `PAR-TRANS-047` eventually | Stage 1 needs parser parity first; Stage 2 unifies with Bifrost endpoint behavior. |
| `PAR-AUTH-024` (CORS middleware) ↔ `PAR-BF-OAI-303` (status-code fallback) | N/A — no overlap | — | Not a real conflict; listed to note these are independent. |

**Disagreement between input files:** `a3-shortlist.md` SEED DISAGREEMENTS note that several PRs have seed domain `routing` but were deferred as "out of Stage 1 scope." Those PRs are excluded from the ACCEPT port list above because they were deferred, not accepted. No active conflict remains in the ACCEPT set.

---

## 6. Stage gates

### Stage 1 — drop-in 9router replacement (v1.0)

**Entry criteria**
- All Wave 0 BROKEN findings resolved and `go test ./...` + `go vet ./...` green.
- No swallowed `rand.Read`, `json.Marshal`, or SSE parse errors remain.
- CORS no longer reflects arbitrary origins with credentials.

**Exit criteria (measurable)**
- 100% of Stage 1 parity rows at HAVE, except rows on an explicit user-approved exclusion list (created at the Stage 0 checkpoint, default empty).
- All UI e2e specs that correspond to 9router pages pass (`ui/e2e/*.spec.ts`).
- Negative test cases from Section 4 are wired and passing.
- `go test ./...` and `go vet ./...` green; govulncheck clean.

**User checkpoint**
- User runs a representative 9router client config against g0router and confirms no regression in chat, embeddings, `/v1/models`, dashboard provider/connection CRUD, and usage viewing.

### Stage 2 — Bifrost OpenAI surface + MCP gateway (v1.1)

**Entry criteria**
- Stage 1 signed off.
- Provider interface expanded to cover Bifrost OpenAI endpoints: text completions, responses, audio, images, files, batches (at minimum routes registered with 501 stubs eliminated).

**Exit criteria (measurable)**
- 100% of Stage 2 parity rows at HAVE, except rows on an explicit user-approved exclusion list (created at the Stage 2 entry checkpoint, default empty).
- All OpenAI-compatible endpoints from `PAR-BF-OAI-001`–`044` registered and return correct envelopes.
- MCP server + client mode scaffold exists with `mcp-go` integration and at least HTTP/SSE/STDIO transports.
- Bifrost-style error envelope (`event_id`, `is_bifrost_error`) available on `/v1` surface.

**User checkpoint**
- User validates that an OpenAI SDK client and an MCP client (e.g., Claude Desktop) can target g0router without code changes.

### Stage 3 — Bifrost enterprise capabilities

**Entry criteria**
- Stage 2 signed off.
- Decision made on vector backend for semantic cache (SQLite + Go vector extension vs external Weaviate/Qdrant/Redis/Pinecone).
- Decision made on cluster membership (HashiCorp memberlist vs none).

**Exit criteria (measurable)**
- 100% of Stage 3 parity rows at HAVE, except rows on an explicit user-approved exclusion list (created at the Stage 3 entry checkpoint, default empty).
- `WhiteList`/`BlackList` semantics enforced.
- Budget and rate-limit CAS-increment workers running with 10s background sync.

**User checkpoint**
- User load-tests multi-key routing, verifies governance decisions (budget/rate-limit/block), and confirms OTEL traces/metrics export.

---

## 7. Open questions — DECIDED at Stage 0 checkpoint (2026-06-09)

| # | Decision |
|---|---|
| 1 | Monolithic per-provider OAuth handlers, implementation ordered by provider popularity. No generic manager abstraction. |
| 2 | Keep g0router opaque SQLite session tokens (7-day TTL). No JWT. |
| 3 | Port all 39 locales as files; react-i18next hooks; drop runtime DOM MutationObserver approach. |
| 4 | Skip Cowork 403 stub in Stage 1. Bifrost MCP gateway (Stage 2) supersedes. PAR rows for Cowork move to the Stage 1 exclusion list. |
| 5 | Live smoke tests REQUIRED in CI for reverse-engineered providers. Implication: CI needs provisioned live accounts/credentials; failures from upstream drift block the gate by design. |
| 6 | Raw OpenAI-compatible shapes authoritative on `/v1` routes. Bifrost `{data,error}` envelope only on the management API. |
| 7 | SQLite-native vector path (sqlite-vec) is the default for semantic cache; external stores optional behind an interface. |
| 8 | Cluster mode is a HARD Stage 3 requirement: full Bifrost parity (memberlist gossip + gRPC replication). |
| 9 | Expand provider interface to full Bifrost size (50+ methods). Stubs return typed not-implemented errors until each capability lands. |
| 10 | Go-native platform equivalents: systemd/launchd service for tray/auto-start, go-selfupdate for auto-update, `crypto/tls` CA reverse proxy for MITM, download-on-demand tunnel binaries (cloudflared/tailscale). |

Stage 1 exclusion list (user-approved per decision 4): 9router Cowork/MCP-bridge rows in `matrix/9router-mcp.md` that exist only to serve the disabled Cowork feature. All other rows gate at 100% HAVE.

Original questions preserved below for traceability.

### Original questions

1. **OAuth flow implementation order and shared infrastructure.** 9router has ~15 OAuth flows (Claude cc, Codex cx, Gemini CLI gc, Qwen, GitHub, Kiro, Cursor, etc.). What is the implementation priority order (device-code vs PKCE vs cookie-auth) and shared-infrastructure approach (generic OAuth manager with per-provider executor plugins vs monolithic per-provider handlers)?

2. **Dashboard session model?** 9router uses JWT (HS256, 24h). g0router uses opaque SQLite tokens (7-day). Do we preserve g0router's approach for parity, or adopt 9router's JWT to ease client migration?

3. **i18n strategy?** 9router has 39 locales with runtime DOM translation via MutationObserver. g0router depends on `react-i18next`. Do we port the 39 locales and runtime DOM approach, or switch to hook-based i18n and drop runtime translation?

4. **Cowork MCP bridge?** 9router's MCP/Cowork feature is globally disabled (returns 403). Do we port it as a disabled-compatible stub in Stage 1, or jump directly to Bifrost's MCP gateway in Stage 2?

5. **Reverse-engineered/cookie-auth providers — testing approach and fragility disclaimer.** Scope is fixed by the north star: every 9router provider ships in v1.0, including reverse-engineered/cookie-auth ones (`grok-web`, `perplexity-web`, Chinese OAuth providers). What testing strategy minimizes fragility (recorded HTTP fixtures vs live smoke tests vs contract tests), and what disclaimer ships in docs?

6. **Bifrost error envelope on `/v1`?** g0router uses raw OpenAI-compatible success/error shapes. Bifrost uses `{data,error}` envelope with `event_id`. Which shape is authoritative on `/v1` routes in Stage 2?

7. **Vector backend for semantic cache?** Bifrost supports Weaviate/Redis/Qdrant/Pinecone. g0router is SQLite-native. Do we build a SQLite vector extension path, or require an external vector store?

8. **Cluster mode commitment?** Bifrost has memberlist gossip + gRPC replication. Is cluster mode a hard Stage 3 requirement, or optional/experimental?

9. **Provider interface size?** Bifrost defines 50+ provider methods. g0router's interface is ~16 methods. Do we expand to full Bifrost size, or keep a smaller core with optional interface upgrades?

10. **Go-native platform equivalents per feature.** 9router is an Electron/Node app with tray, auto-update, MITM proxy, cloudflared/Tailscale tunnels. g0router is a single Go binary. Which Go-native mechanism replaces each feature (systemd/launchd service for tray/auto-start, `crypto/tls` reverse proxy with `mkcert`-style CA for MITM, bundled `cloudflared`/`tailscale` binaries or download-on-demand for tunnels, `go-selfupdate` or binary patch for auto-update)?

---

SYNTHESIS-COMPLETE
