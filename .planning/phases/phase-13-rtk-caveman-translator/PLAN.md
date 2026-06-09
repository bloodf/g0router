# Phase 13: RTK + Caveman + Translator

**Phase:** 13  
**Goal:** Port the 9router platform features: RTK compression, Caveman mode, and translator debug.  
**Requirements:** PLAT-01..09, MGMT-14, UI-15..16  
**Estimated duration:** 5–6 days  
**Wave:** 5 — 9router Features

---

## Why

These are signature 9router features that differentiate g0router from a plain proxy: RTK saves tokens, Caveman reduces output length, and translator debug makes routing transparent.

---

## Scope

### In scope
- `internal/platform/rtk.go` — detect and compress tool outputs (git diff, grep, find, ls, tree, logs).
- `internal/platform/caveman.go` — inject caveman-speak system prompt.
- `internal/platform/translator.go` — OpenAI ↔ Anthropic ↔ Gemini ↔ Cursor ↔ Kiro ↔ Vertex transformations.
- `internal/admin/translator.go` — `/api/translator/debug`, `/api/translator/test`.
- Thought separation: reasoning content stored separately from final content.
- Dashboard pages:
  - `routes/_app.translator.tsx`
  - Settings toggles for RTK and Caveman.

### Out of scope
- New translation targets beyond those listed (future).

---

## Verification

### Tests
1. RTK compresses a sample git diff to fewer tokens.
2. RTK fallback keeps original text when compression fails.
3. Caveman mode produces shorter output in fixture test.
4. Translator correctly maps OpenAI messages to Anthropic Messages format and back.
5. Thought content is preserved separately in responses.

### Manual verification
1. Enable RTK and send a request with a large tool result.
2. Use translator debug to inspect a request transformation.

---

## Tasks

1. Implement RTK filters.
2. Implement Caveman prompt injection.
3. Implement translator helpers.
4. Implement translator debug endpoints.
5. Add dashboard translator page and settings toggles.
6. Write tests and E2E coverage.
7. Verify gates.

---

## Risks

| Risk | Mitigation |
|------|------------|
| RTK over-compresses critical context | Make filters lossless; test that decompressed meaning is preserved. |
| Translator becomes unmaintainable | Keep provider-specific converters in separate files with clear mapping tables. |
