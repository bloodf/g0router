# g0router Routing/Runtime Audit — 04-routing.md

**Date:** 2026-06-05  
**Scope:** model resolution, aliases, combos, fallback/backoff, connection rotation, token refresh, streaming (SSE), tool-call translation, usage extraction, cost calculation, quota checks, request logging, OpenAI↔Anthropic ingress/egress compatibility.  
**Method:** source-only, no edits, no network.

---

## Verdict

Functional for happy-path text completions. Three correctness bugs in the streaming and tool-call surface; two design issues around success recording and cost units. No data races in the hot path with normal concurrency; one theoretical data race on shared Connection pointers under high concurrency. OpenAI egress correct. Anthropic egress broken for streaming tool calls. Anthropic ingress functional but incomplete.

---

## Findings

### CRITICAL

None.

---

### HIGH

#### H1 — Premature success recording in `dispatchStreamRoute`
**File:** `internal/proxy/engine.go:209`

```go
e.recordProviderSuccess(conn, upstreamModel)
return stream, nil
```

`recordProviderSuccess` (which resets `BackoffLevel` to 0 and deletes the model lock) is called the moment `ChatCompletionStream` returns a non-error channel — before a single token has been consumed. If the upstream stream subsequently errors mid-way (network drop, provider-side cut-off), the connection's backoff is already cleared. The next request will route to the same connection with no penalty, defeating the exponential backoff.

**Compare:** `dispatchRoute` (non-streaming) records success only after `chatCompletion` returns a full non-error response — semantically correct.

**Fix:** Record success from a goroutine that wraps the channel, e.g. inside `captureStream` in `server.go` when the stream drains without a `StreamChunk.Error`. The simplest approach: pass `conn` and `upstreamModel` into a wrapper goroutine, record failure if any error chunk is seen, record success on clean drain.

**Verify:** Add a test that injects a mid-stream error and asserts `BackoffLevel > 0` after the stream is consumed.

---

#### H2 — Anthropic SSE egress silently drops streaming tool calls
**File:** `api/handlers/inference.go:163–220` (`streamMessages`)

The `streamMessages` function iterates `chunk.Choices` and handles only `Delta.Content` (text delta) and `FinishReason`. When the upstream (any provider via `anthropic.go`'s `parseSSE`) emits a completed tool-call block via `content_block_stop`, it arrives as a `StreamChunk` with `Delta.ToolCalls` set. `streamMessages` never reads `Delta.ToolCalls` — those tool call events are silently dropped. An Anthropic-format client (`/v1/messages`) requesting streaming tool use receives no `content_block_start/delta/stop` events for `tool_use` blocks, violating the Anthropic SSE protocol.

**Affected path:** POST `/v1/messages` with `stream: true` and tools.

**Fix:** After handling `Delta.Content`, add a branch:
```go
for i, toolCall := range choice.Delta.ToolCalls {
    writeAnthropicStreamEvent(w, "content_block_start", map[string]any{
        "type": "content_block_start", "index": blockIndex + 1 + i,
        "content_block": map[string]any{"type": "tool_use", "id": toolCall.ID, "name": toolCall.Function.Name, "input": map[string]any{}},
    })
    writeAnthropicStreamEvent(w, "content_block_delta", map[string]any{
        "type": "content_block_delta", "index": blockIndex + 1 + i,
        "delta": map[string]any{"type": "input_json_delta", "partial_json": toolCall.Function.Arguments},
    })
    writeAnthropicStreamEvent(w, "content_block_stop", map[string]any{
        "type": "content_block_stop", "index": blockIndex + 1 + i,
    })
}
```

**Verify:** Stream a tool-using request to `/v1/messages`, assert `content_block_start` with `type: tool_use` appears in SSE output.

---

#### H3 — `RefreshManager`: delete-then-close exposes a narrow use-after-free window
**File:** `internal/provider/refresh.go:56–59`

```go
m.mu.Lock()
delete(m.inflight, key)   // (1) entry removed
m.mu.Unlock()
close(call.done)           // (2) waiters unblocked
```

Between (1) and (2): a new goroutine for the same refresh key arrives, finds no inflight entry, allocates a **new** `refreshCall`, and starts a new refresh. This is not a crash, but it results in redundant concurrent refreshes for the same token when there is contention. More critically, the old `call.token` / `call.err` are written before (1) and the close happens after — waiters read from the already-populated struct correctly — but the sequencing guarantees are informal. The fix is to close the channel before deleting from the map so the invariant "an inflight entry always has an open done channel" holds.

```go
close(call.done)           // close first
m.mu.Lock()
delete(m.inflight, key)
m.mu.Unlock()
```

**Severity justification:** Demoted from race/critical because reads of `call.token`/`call.err` happen only after `<-call.done` unblocks, which only happens after both writes. No actual data corruption occurs in practice. Kept HIGH because it causes silent redundant OAuth refreshes under load.

**Verify:** Race detector (`go test -race`) on refresh concurrency test.

---

### MEDIUM

#### M1 — Cost override uses per-token units; catalog uses per-million; no documentation or validation
**Files:** `internal/usage/cost.go:12–13,55–56` vs `internal/modelcatalog/catalog.go:22–24`

```go
// catalog path (correct):
inputCost := float64(inputTokens) * price.InputPerMillionUSD / 1_000_000

// override path:
inputCost := float64(usage.InputTokens) * price.InputCostPerToken  // no /1_000_000
```

The override stores and applies cost as **dollars per token** (e.g. `0.00000015`). The catalog stores **dollars per million tokens** and divides by `1_000_000`. These are intentionally different units. This is internally consistent — the test at `pricing_test.go:14` uses `0.00000015` which is indeed `$0.15/million` expressed per-token. The problem: the API surface (`/api/pricing`) and `pricingRequest` struct use field names `input_cost_per_token` / `output_cost_per_token` with no unit documentation. Any operator who enters `0.15` (intending $/million, matching catalog convention) instead of `0.00000015` will get costs 1,000,000× too high. There is no validation or unit hint.

**Fix:** Add a comment/docstring to `PricingOverride`, add a sanity-check bounds validation in `SetPricingOverride` (e.g. reject values > 0.01), or rename fields to `input_cost_per_million_usd` to match catalog convention.

**Verify:** Unit test covering a known model's catalog price vs override price producing equal output when set consistently.

---

#### M2 — `FallbackManager.RecordFailure` / `RecordSuccess` mutate shared `*Connection` without lock
**File:** `internal/provider/fallback.go:76–98`

`Next()` acquires `m.mu` to protect the `cursors` map, but `RecordFailure` and `RecordSuccess` mutate `conn.BackoffLevel` and `conn.ModelLocks` directly on the pointer returned from `GetActiveConnections`. If two concurrent requests both fail on the same connection and both call `RecordFailure` simultaneously, `conn.BackoffLevel++` is an unsynchronized read-modify-write on an `int` field — a data race under `-race`.

In practice, each call to `GetActiveConnections` returns freshly deserialized pointers from SQLite, so the same request's pointer is not shared across goroutines unless they get the same pointer from the same query result. The store's `queryConnections` allocates new `*Connection` per scan, so concurrent requests each get their own pointer. Race is unlikely in production but possible if callers cache the pointer (none currently do, but this is a fragile assumption).

**Fix:** Either document that each caller owns its pointer (and ensure it stays that way), or copy the struct in `RecordFailure`/`RecordSuccess` before mutation.

**Verify:** `go test -race ./internal/provider/...` with concurrent failure injection.

---

#### M3 — Anthropic ingress (`/v1/messages`) missing translation of multi-part content arrays
**File:** `api/handlers/inference.go:107–133`

The `Messages` handler deserializes the body directly into `providers.ChatRequest` (OpenAI wire format). Anthropic clients send `content` as an array of typed blocks, e.g. `[{"type":"text","text":"..."}]`. After `json.Unmarshal`, `Message.Content` is `[]interface{}` (since the field is `any`). The `rejectUnsupportedAnthropicContent` guard rejects non-text block types at ingress, so image/tool blocks return 501. Text-only multi-part arrays reach `toAnthropicRequest` where `contentBlocksFromContent` handles `[]any` via marshal/unmarshal — this works. However, tool result messages (`{"role":"tool","content":"...","tool_call_id":"..."}`) sent in Anthropic format (`tool_use_id` field) are not mapped to `ToolCallID`. The client sends `"tool_use_id"` but the struct uses `"tool_call_id"` — the field name differs between Anthropic and OpenAI wire formats, so tool results from Anthropic clients arrive with `ToolCallID == nil` and `toToolResultBlock` returns `"tool result missing tool_call_id"`.

**Affected path:** Any Anthropic client calling `/v1/messages` with tool results.

**Fix:** In the `Messages` handler, after parsing, remap Anthropic's `tool_use_id` field to `ToolCallID`. This requires either a separate Anthropic-schema struct for ingress, or a post-parse fixup pass.

**Verify:** POST to `/v1/messages` with a message `{"role":"tool","tool_use_id":"tu_abc","content":"result"}`, assert no 400/500.

---

### LOW

#### L1 — `maxConnectionAttempts` queries store without lock; could diverge from `FallbackManager.Next`
**File:** `internal/proxy/engine.go:446–459`

`maxConnectionAttempts` calls `store.GetActiveConnections` to count connections, then the loop calls `providerForRoute` → `connectionForModel` → `FallbackManager.Next` → `GetActiveConnections` again. Between the two queries, a connection could be deactivated, making `maxConnectionAttempts` return N but `Next` find N-1. This causes one extra no-op loop iteration, not a crash. Low severity, but worth noting.

#### L2 — `streamMessages` Anthropic SSE missing final `w.Flush()` guard for no-content streams
**File:** `api/handlers/inference.go:222`

If a stream yields zero choices (e.g. empty upstream response), `started` stays false, no events are written, and the SSE response ends with no `message_stop`. Some clients will hang waiting for the terminal event. The non-streaming `anthropicMessageResponse` path handles nil/empty gracefully. For streaming, add a post-loop: if `!started`, emit a minimal `message_start` + `message_stop`.

#### L3 — `DetectFormat` misclassifies OpenAI requests with `system` string field as Anthropic
**File:** `internal/translate/detect.go:30`

```go
if rawSystem, ok := fields["system"]; ok && isAnthropicSystem(rawSystem) {
    return FormatAnthropic, nil
}
```

`isAnthropicSystem` returns true for a plain string value. An OpenAI-format request with `"system": "..."` (a valid extension, used in some clients) would be misclassified as Anthropic format. Only affects `NormalizeOpenAI` callers; the main `Inference` handler (`/v1/chat/completions`) does not call `NormalizeOpenAI` — it parses directly. `Messages` handler also parses directly. `NormalizeOpenAI` is used in `translate` package tests. Low impact currently, but a latent bug if `NormalizeOpenAI` is ever added to the main request path.

#### L4 — OpenAI Responses streaming drops tool call outputs
**File:** `api/handlers/inference.go:336–370` (`streamResponses`)

`streamResponses` only accumulates `Delta.Content` text. Tool call chunks in `Delta.ToolCalls` are ignored. The `response.completed` event will have no `function_call` output items. Non-streaming `Responses` path correctly calls `OpenAIChatToResponsesResponse` which handles tool calls.

---

## NO-ISSUE Areas

- **Model resolution / alias cache**: correct. `aliasCache` is internally locked; TTL-based expiry is consistent.
- **Combo fallback**: correct. `ComboResolver.Dispatch` tries each step in order, short-circuits on `ErrQuotaExhausted` only, passes other errors through to next step.
- **Token refresh before dispatch**: correct. `refreshConnectionIfNeeded` is called in `keyForModel` which is called before every dispatch. `RefreshManager` deduplicates concurrent refreshes for the same connection.
- **OAuth refresh window (5 min)**: correct. `connectionNeedsRefresh` checks `ExpiresAt - refreshWindow`.
- **Non-streaming fallback**: correct. `dispatchRoute` loops up to `maxConnectionAttempts`, records failure and continues on `fallbackWorthyError`, returns `lastErr` when exhausted.
- **`fallbackWorthyError` coverage**: good. Covers rate-limit, quota, server errors, timeouts, gateway errors.
- **Anthropic egress (non-stream)**: correct. `toAnthropicRequest` translates system, tools, tool_choice, stop sequences. `toChatResponse` maps stop reason, usage, tool_use blocks.
- **OpenAI egress (non-stream + stream)**: correct. Passes `providers.ChatRequest` directly; SSE parsed and forwarded without buffering.
- **Anthropic stream tool-call assembly** (provider-side): correct. `parseSSE` state machine accumulates `input_json_delta` fragments per block index, emits complete `ToolCall` on `content_block_stop`. Input tokens preserved from `message_start`.
- **Usage extraction**: correct. `fromProviderUsage` maps `PromptTokens`→`InputTokens`, `CompletionTokens`→`OutputTokens`, `CachedTokens` from `PromptTokensDetails`.
- **Cost calculation (catalog path)**: correct. Subtracts `CacheReadTokens` from input, applies separate cache price.
- **Quota check**: correct. Per-request, before dispatch, checks `Remaining <= 0` unless `Unlimited`. Swallows `ErrQuotaUnsupported` gracefully.
- **Request logging**: correct. Captures usage from both non-stream response and stream `onStreamComplete` callback. Does not double-log streams.
- **RTK / Caveman preprocessing**: correct. Applied as a wrapping engine layer before dispatch; does not mutate original request (copies struct).
- **SSE streaming passthrough (OpenAI)**: true streaming — `SetBodyStreamWriter` with per-chunk `Flush()`, no buffering.
- **Connection pool (`providerPool`)**: no lock needed (write-once at startup, read-only thereafter).

---

## OpenAI / Anthropic Compatibility Matrix

| Direction | Protocol | Endpoint | Status | Notes |
|-----------|----------|----------|--------|-------|
| Ingress | OpenAI Chat | `POST /v1/chat/completions` | ✅ Compatible | Full parse of ChatRequest struct |
| Ingress | OpenAI Responses | `POST /v1/responses` | ✅ Compatible | Translated to ChatRequest |
| Ingress | Anthropic Messages | `POST /v1/messages` | ⚠️ Partial | Text works; tool results with `tool_use_id` broken (M3) |
| Egress | OpenAI provider | ChatCompletion | ✅ Compatible | Passes ChatRequest verbatim |
| Egress | OpenAI provider | Stream | ✅ Compatible | SSE parse→forward, no buffering |
| Egress | Anthropic provider | ChatCompletion | ✅ Compatible | Full schema translation |
| Egress | Anthropic provider | Stream | ✅ Compatible (provider-side) | SSE assembled correctly upstream |
| Client ← | OpenAI SSE response | `/v1/chat/completions stream` | ✅ Compatible | |
| Client ← | Anthropic SSE response | `/v1/messages stream` text | ✅ Compatible | |
| Client ← | Anthropic SSE response | `/v1/messages stream` tools | ❌ Broken | H2: tool_use blocks dropped |
| Client ← | OpenAI Responses SSE | `/v1/responses stream` tools | ❌ Broken | L4: tool outputs dropped |

---

## Summary Table

| ID | Severity | File:Line | Issue |
|----|----------|-----------|-------|
| H1 | High | `internal/proxy/engine.go:209` | Premature success recorded before stream consumed |
| H2 | High | `api/handlers/inference.go:163` | Streaming tool calls silently dropped in Anthropic SSE egress |
| H3 | High | `internal/provider/refresh.go:56–59` | delete-before-close causes redundant concurrent OAuth refreshes |
| M1 | Medium | `internal/usage/cost.go:12,55` | Override cost units (per-token) undocumented vs catalog (per-million); operator trap |
| M2 | Medium | `internal/provider/fallback.go:76` | `BackoffLevel` mutation without lock; data race under `-race` |
| M3 | Medium | `api/handlers/inference.go:107` | Anthropic `tool_use_id` not mapped to `ToolCallID` on ingress |
| L1 | Low | `internal/proxy/engine.go:446` | `maxConnectionAttempts` count can diverge from actual available connections |
| L2 | Low | `api/handlers/inference.go:222` | Empty Anthropic stream sends no terminal `message_stop` |
| L3 | Low | `internal/translate/detect.go:30` | OpenAI `system` string misdetected as Anthropic format |
| L4 | Low | `api/handlers/inference.go:347` | Responses API streaming drops tool call outputs |

**Counts:** Critical: 0, High: 3, Medium: 3, Low: 4
