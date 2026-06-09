# Phase 12: Responses API + Batch + Files

**Phase:** 12  
**Goal:** Implement OpenAI Responses API, file management, and batch operations.  
**Requirements:** OPENAI-06, OPENAI-09..10  
**Estimated duration:** 5–6 days  
**Wave:** 4 — Advanced API Surface

---

## Why

These endpoints complete the OpenAI-compatible surface for agentic and high-throughput workloads.

---

## Scope

### In scope
- `internal/api/responses.go` — `POST /v1/responses` with event-based SSE.
- `internal/api/files.go` — file upload/list/retrieve/delete/content.
- `internal/api/batch.go` — batch create/list/retrieve/cancel.
- `internal/providers/openai/responses.go` — OpenAI Responses API converter.
- `internal/providers/openai/files.go` — file/batch converters.
- Local file storage for uploaded files.

### Out of scope
- Persistent job queue for batch execution (use simple SQLite polling).

---

## Verification

### Tests
1. Responses API returns correct shape.
2. Responses streaming uses `event:` types without `[DONE]`.
3. File upload stores file and returns OpenAI file object.
4. Batch create stores job and returns batch object.
5. Batch cancel updates status.

### Manual verification
1. Run Responses API streaming request.
2. Upload a file and retrieve it.

---

## Tasks

1. Extend provider interface for responses, files, batch.
2. Implement OpenAI responses converter.
3. Implement file storage and handlers.
4. Implement batch storage and handlers.
5. Write integration tests.
6. Verify gates.

---

## Risks

| Risk | Mitigation |
|------|------------|
| File storage path issues | Store files under configured `DATA_DIR`; validate paths. |
| Batch job polling overhead | Use SQLite index on status + scheduled time. |
