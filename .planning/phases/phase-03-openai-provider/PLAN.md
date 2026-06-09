# Phase 3: OpenAI Provider

**Phase:** 03  
**Goal:** Implement the reference OpenAI provider with chat, embeddings, models, and streaming.  
**Requirements:** PROV-01..03, OPENAI-01..02  
**Estimated duration:** 4–5 days  
**Wave:** 1 — Foundation

---

## Why

OpenAI is the reference provider. If this provider is correct, every other provider has a clear pattern to follow.

---

## Scope

### In scope
- `internal/providers/openai/provider.go` — implements `Provider` interface using fasthttp.
- `internal/providers/openai/chat.go` — request/response converters and streaming chunk mapper.
- `internal/providers/openai/embedding.go` — embedding converters.
- `internal/providers/openai/models.go` — list models converter.
- `internal/providers/openai/errors.go` — `ErrorConverter` for OpenAI HTTP errors.
- `internal/providers/utils/` — shared fasthttp client setup, SSE scanner pool, common helpers.
- Non-streaming chat completion.
- Streaming chat completion via SSE.
- Embeddings.
- List models.

### Out of scope
- Images, audio, files, batch (Phase 11/12).
- Responses API (Phase 12).
- Other providers.

---

## Verification

### Tests
1. Non-streaming chat completion returns correct response shape against recorded fixture.
2. Streaming chat completion emits valid SSE chunks and `[DONE]`.
3. Embeddings return correct vector shape.
4. List models returns expected entries.
5. Error converter maps 4xx/5xx to uniform error schema.

### Manual verification
1. Run provider tests with recorded fixtures.
2. If API key available, run one live request and compare to fixture.

---

## Tasks

1. Set up shared provider utilities.
2. Implement fasthttp client lifecycle in `openai/provider.go`.
3. Implement chat request/response converters.
4. Implement streaming handler.
5. Implement embedding and models converters.
6. Implement error converter.
7. Write fixture files and table-driven tests.
8. Verify gates.

---

## Risks

| Risk | Mitigation |
|------|------------|
| SSE parsing is fragile | Use robust line scanner + explicit tests for malformed chunks. |
| Response shape drift | Record fixture from latest OpenAI API and pin test to it. |
| fasthttp pool misuse | Use `AcquireRequest`/`ReleaseRequest` consistently; run with `-race`. |
