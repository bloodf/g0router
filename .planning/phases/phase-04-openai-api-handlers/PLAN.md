# Phase 4: OpenAI API Handlers

**Phase:** 04  
**Goal:** Expose `/v1/chat/completions`, `/v1/embeddings`, `/v1/models` and validate drop-in OpenAI SDK compatibility.  
**Requirements:** OPENAI-01..05, OPENAI-11..12  
**Estimated duration:** 3–4 days  
**Wave:** 1 — Foundation

---

## Why

This is the first user-visible milestone. A working `/v1/chat/completions` proves the architecture end-to-end.

---

## Scope

### In scope
- `internal/server/server.go` — fasthttp server setup.
- `internal/server/routes_openai.go` — route registration for `/v1/*`.
- `internal/server/middleware.go` — CORS, request ID, auth middleware.
- `internal/api/chat.go` — `POST /v1/chat/completions` handler (streaming + non-streaming).
- `internal/api/embeddings.go` — `POST /v1/embeddings` handler.
- `internal/api/models.go` — `GET /v1/models` and `/v1/models/:id` handlers.
- `internal/inference/router.go` — basic routing: resolve provider, select key, call provider.
- Wire catalog and OpenAI provider into `cmd/g0router/main.go`.

### Out of scope
- Virtual keys and weighted routing (Phase 8).
- Anthropic/Gemini providers (Phase 5).
- Management API.

---

## Verification

### Tests
1. Integration test: `POST /v1/chat/completions` non-streaming returns OpenAI-shaped JSON.
2. Integration test: `POST /v1/chat/completions` streaming returns valid SSE.
3. Integration test: `POST /v1/embeddings` returns correct shape.
4. Integration test: `GET /v1/models` returns entries from catalog.
5. Error responses use OpenAI error envelope.

### Manual verification
1. Point OpenAI Python SDK at `http://localhost:20128/v1` and run a chat completion.
2. Repeat with streaming enabled.

---

## Tasks

1. Implement fasthttp server with route registration.
2. Implement middleware chain.
3. Implement chat completion handler with stream branching.
4. Implement embeddings handler.
5. Implement models handler.
6. Implement basic router (provider resolution + key selection).
7. Wire dependencies in main.
8. Write integration tests with in-memory SQLite.
9. Verify gates.

---

## Risks

| Risk | Mitigation |
|------|------------|
| Streaming handler leaks goroutines | Use `context.Done` + defer close; test with `-race`. |
| Auth middleware blocks OpenAI SDK | Accept bearer tokens and validate against API keys/endpoint keys only after Phase 6. |
| Route conflicts with dashboard | Serve `/v1/*` before catch-all UI handler. |
