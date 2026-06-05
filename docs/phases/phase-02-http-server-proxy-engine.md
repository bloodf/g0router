# Phase 2: HTTP Server + Proxy Engine

> **Depends on**: Phase 1  
> **Unlocks**: Phase 3, Phase 6, Phase 9, Phase 10  
> **Checkpoint**: `PHASE_2_COMPLETE`  
> **Key dependency**: `github.com/valyala/fasthttp`

---

## Prerequisites

- [x] Phase 1 complete (`PHASE_1_COMPLETE`)
- [x] `go test ./...` passes
- [x] `go vet ./...` passes

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| HTTP library | fasthttp | 10x throughput vs net/http for proxy workloads |
| Router | Manual path matching | Only ~20 routes; no framework needed |
| Request ID | UUID v4 | Unique, standard |
| CORS | Allow all origins | Gateway is behind user's network |
| Streaming | SSE via `text/event-stream` | OpenAI standard |
| Provider pool | Map-based provider registry | Registered providers looked up by name |

---

## Task 2.1: fasthttp Server Skeleton

### Completed Work

- [x] `go get github.com/valyala/fasthttp`
- [x] Write `api/server_test.go` — **test file FIRST**
- [x] Write `api/handlers/health.go` — health handler
- [x] Run tests → see RED
- [x] Write `api/server.go` — Server struct, routing, Start/Stop
- [x] Run tests → see GREEN
- [x] Run `go vet ./...` → clean
- [x] Commit: `phase-2/task-1: fasthttp server with health endpoint`

### Pre-conditions

- Phase 1 complete
- `github.com/valyala/fasthttp` in go.mod

### TDD Cycle

#### RED: Write Failing Tests First

Create `api/server_test.go`:

```go
package api

import (
    "encoding/json"
    "io"
    "net/http"
    "testing"
    "time"
)

func TestHealthz(t *testing.T) {
    srv := NewServer(ServerConfig{Port: 0, Version: "test-version"})
    ln := srv.listener()
    if ln == nil {
        t.Fatal("listener failed")
    }
    addr := ln.Addr().String()

    go func() { _ = srv.Serve(ln) }()
    t.Cleanup(func() { _ = srv.Stop() })

    client := &http.Client{Timeout: 2 * time.Second}
    resp, err := client.Get("http://" + addr + "/healthz")
    if err != nil {
        t.Fatalf("GET /healthz: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        t.Fatalf("status: %d", resp.StatusCode)
    }

    body, _ := io.ReadAll(resp.Body)
    var result map[string]string
    if err := json.Unmarshal(body, &result); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }
    if result["status"] != "ok" {
        t.Errorf("status: %q", result["status"])
    }
    if result["version"] != "test-version" {
        t.Errorf("version: %q", result["version"])
    }
}

func TestUnknownRoute(t *testing.T) {
    srv := NewServer(ServerConfig{Port: 0, Version: "test"})
    ln := srv.listener()
    go func() { _ = srv.Serve(ln) }()
    t.Cleanup(func() { _ = srv.Stop() })

    client := &http.Client{Timeout: 2 * time.Second}
    resp, err := client.Get("http://" + ln.Addr().String() + "/nope")
    if err != nil {
        t.Fatal(err)
    }
    if resp.StatusCode != 404 {
        t.Errorf("expected 404, got %d", resp.StatusCode)
    }
}
```

Run: `go test ./api/...` → RED (types don't exist).

#### GREEN: Implement

1. `api/handlers/health.go` — `func Health(ctx *fasthttp.RequestCtx, version string)`
2. `api/server.go` — `ServerConfig`, `Server` struct, `NewServer`, `Serve(ln)`, `Stop()`, `listener()`, route handler

#### REFACTOR

- Verify graceful shutdown works (Stop returns cleanly)
- No goroutine leaks

### Verification

```bash
go test ./api/... -v -count=1
go vet ./...
```

### Post-conditions

- [x] `GET /healthz` returns `{"status":"ok","version":"..."}`
- [x] Unknown routes return 404
- [x] Server starts and stops cleanly

### Commit

```
phase-2/task-1: fasthttp server with health endpoint
```

---

## Task 2.2: Middleware (CORS, Auth, Request ID)

### Completed Work

- [x] Write `api/middleware_test.go` — **test file FIRST**
- [x] Run tests → see RED
- [x] Add middleware to server handler
- [x] Run tests → see GREEN
- [x] Commit: `phase-2/task-2: middleware for CORS, request ID, and API key auth`

### TDD Cycle

#### RED: Write Failing Tests First

Tests:
- `TestCORSHeaders` — every response has `Access-Control-Allow-Origin: *`
- `TestOptionsReturns204` — OPTIONS → 204, no body
- `TestRequestIDPresent` — every response has `X-Request-ID` header (UUID format)
- `TestRequestIDUnique` — two requests → different IDs
- `TestAuthRequiredMissingKey` — REQUIRE_API_KEY=true, no key → 401
- `TestAuthRequiredValidKey` — valid Bearer token → passes through
- `TestAuthNotRequired` — REQUIRE_API_KEY=false → passes through without key
- `TestHealthzBypassesAuth` — /healthz never requires auth

#### GREEN: Implement

Middleware embedded in main handler function:
1. Generate X-Request-ID (UUID v4)
2. Set CORS headers
3. Handle OPTIONS → 204
4. Check auth for `/v1/*` routes if required
5. Route to handler

### Verification

```bash
go test ./api/... -v -count=1
go vet ./...
```

### Commit

```
phase-2/task-2: middleware for CORS, request ID, and API key auth
```

---

## Task 2.3: Proxy Engine Core

### Completed Work

- [x] Write `internal/proxy/engine_test.go` — **test file FIRST**
- [x] Run tests → see RED
- [x] Write `internal/proxy/engine.go`
- [x] Write `internal/proxy/pool.go`
- [x] Run tests → see GREEN
- [x] Commit: `phase-2/task-3: proxy engine with provider dispatch`

### TDD Cycle

#### RED: Write Failing Tests First

Create a fake provider for testing:

```go
type fakeProvider struct {
    name     providers.ModelProvider
    response *providers.ChatResponse
    err      error
}
func (f *fakeProvider) Name() providers.ModelProvider { return f.name }
func (f *fakeProvider) ChatCompletion(ctx context.Context, key providers.Key, req *providers.ChatRequest) (*providers.ChatResponse, error) {
    return f.response, f.err
}
// ... stream, list models
```

Tests:
- `TestDispatchRoutesToCorrectProvider` — register OpenAI + Anthropic, dispatch gpt-4o → reaches OpenAI
- `TestDispatchUnknownModel` — returns `ErrProviderNotFound`
- `TestDispatchNoConnections` — provider registered but no connections → `ErrNoConnections`
- `TestDispatchStreamReturnsChannel` — stream dispatch → channel with chunks

#### GREEN: Implement

- `Engine` struct with `providers` map + `store` reference
- `Register(p)` adds provider
- `resolveProvider(model)` — prefix table: `gpt-` → OpenAI, `claude-` → Anthropic, etc.
- `Dispatch` / `DispatchStream` — resolve provider → select connection → call provider method

### Commit

```
phase-2/task-3: proxy engine with provider dispatch
```

---

## Task 2.4: OpenAI Provider Implementation

### Completed Work

- [x] Write `internal/providers/openai/openai_test.go` — **test file FIRST**
- [x] Run tests → see RED
- [x] Write `internal/providers/openai/openai.go`
- [x] Write `internal/providers/openai/errors.go`
- [x] Run tests → see GREEN
- [x] Commit: `phase-2/task-4: OpenAI provider with streaming SSE parser`

### TDD Cycle

#### RED: Write Failing Tests First

**Do NOT call the real OpenAI API in tests.** Use recorded JSON fixtures and a local test HTTP server.

Tests:
- `TestBuildRequest` — ChatRequest + Key → correct URL, auth header, JSON body
- `TestParseResponse` — recorded OpenAI JSON → correct ChatResponse
- `TestParseSSEStream` — multi-line SSE recording → correct StreamChunks in order, channel closed on `[DONE]`
- `TestParseSSEWithUsage` — final chunk has usage
- `TestParseError401` — `{"error":{"message":"invalid api key"}}` → ErrAuth
- `TestParseError429` — rate limit response → ErrRateLimit with RetryAfter
- `TestParseError500` — server error → ErrServer
- `TestListModels` — recorded models response → correct []Model

#### GREEN: Implement

- `OpenAIProvider` struct with fasthttp client + base URL
- `ChatCompletion` — build request, execute, parse response
- `ChatCompletionStream` — build request, set stream=true, parse SSE line-by-line
- `ListModels` — GET /v1/models, parse response
- Error mapping: 401/403 → ErrAuth, 429 → ErrRateLimit, 5xx → ErrServer

### Commit

```
phase-2/task-4: OpenAI provider with streaming SSE parser
```

---

## Task 2.5: Shared Provider Utilities

### Completed Work

- [x] Write `internal/providers/utils/http_test.go` — **test file FIRST**
- [x] Write `internal/providers/utils/sse_test.go` — **test file FIRST**
- [x] Run tests → see RED
- [x] Write `internal/providers/utils/http.go`
- [x] Write `internal/providers/utils/sse.go`
- [x] Write `internal/providers/utils/errors.go`
- [x] Run tests → see GREEN
- [x] Commit: `phase-2/task-5: shared HTTP client and SSE parser utilities`

### TDD Cycle

#### RED: Tests for HTTP client

- `TestDoRequestSuccess` — 200 response
- `TestDoRequestRetry5xx` — first 2 calls 500, third 200 → succeeds
- `TestDoRequest429` — returns ErrRateLimit with RetryAfter
- `TestDoRequestTimeout` — context cancelled → error

#### RED: Tests for SSE parser

- `TestParseSSENormal` — receives all data payloads in order
- `TestParseSSEWithDone` — `[DONE]` stops, fn not called after
- `TestParseSSEComments` — `:comment` lines skipped
- `TestParseSSEEmptyLines` — handled gracefully

#### RED: Error types

- `TestProviderErrorUnwrap` — `errors.Is(err, ErrRateLimit)` works

### Commit

```
phase-2/task-5: shared HTTP client and SSE parser utilities
```

---

## Task 2.6: Streaming Accumulator

### Completed Work

- [x] Write `internal/streaming/accumulator_test.go` — **test file FIRST**
- [x] Run tests → see RED
- [x] Write `internal/streaming/accumulator.go`
- [x] Write `internal/streaming/chat.go`
- [x] Run tests → see GREEN
- [x] Commit: `phase-2/task-6: streaming accumulator for chunk collection`

### TDD Cycle

#### RED: Tests

- `TestAccumulateTextResponse` — 3 content delta chunks → complete message
- `TestAccumulateToolCall` — chunks with tool_call deltas → complete ToolCall
- `TestAccumulateUsageFromFinalChunk` — last chunk has usage → `Usage()` non-nil
- `TestAccumulateMultipleChoices` — chunks with index 0 and 1 → two choices
- `TestAccumulateEmpty` — no chunks → empty response, nil usage

### Commit

```
phase-2/task-6: streaming accumulator for chunk collection
```

---

## Task 2.7: Inference Handler

### Completed Work

- [x] Write `api/handlers/inference_test.go` — **test file FIRST**
- [x] Write `api/handlers/models.go`
- [x] Run tests → see RED
- [x] Write `api/handlers/inference.go`
- [x] Run tests → see GREEN
- [x] Commit: `phase-2/task-7: inference handler with sync and streaming dispatch`

### TDD Cycle

#### RED: Tests (using fake engine)

- `TestSyncInference` — POST /v1/chat/completions, stream=false → 200 + ChatResponse JSON
- `TestStreamInference` — POST /v1/chat/completions, stream=true → 200 + SSE chunks + `[DONE]`
- `TestInferenceInvalidJSON` — malformed body → 400
- `TestInferenceUnknownModel` — model="nonexistent" → 404
- `TestInferenceNoAuth` — missing key when required → 401
- `TestGetModels` — GET /v1/models → 200 + model list

### Commit

```
phase-2/task-7: inference handler with sync and streaming dispatch
```

---

## Phase Gate

```bash
go test ./... -count=1    # ALL tests pass (Phase 1 + Phase 2)
go vet ./...              # Clean
go build ./cmd/g0router   # Binary builds
```

**Expected test count**: ~50+ tests across ~12 test files.

## Phase Checklist

- [x] Task 2.1 complete (server + health)
- [x] Task 2.2 complete (middleware)
- [x] Task 2.3 complete (proxy engine)
- [x] Task 2.4 complete (OpenAI provider)
- [x] Task 2.5 complete (HTTP/SSE utils)
- [x] Task 2.6 complete (accumulator)
- [x] Task 2.7 complete (inference handler)
- [x] All tests pass: `go test ./...`
- [x] Vet clean: `go vet ./...`
- [x] Build succeeds: `go build ./cmd/g0router`
- [x] All commits follow `phase-2/task-N: description` format
- [x] Update `docs/WORKFLOW.md`: phase_2.status → `DONE`
- [x] **PHASE_2_COMPLETE**
