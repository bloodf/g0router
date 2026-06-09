# Phase 5: Anthropic + Gemini Providers

**Phase:** 05  
**Goal:** Add converter-based providers for Anthropic and Gemini with format translation.  
**Requirements:** PROV-04..05, PLAT-07..09  
**Estimated duration:** 5–6 days  
**Wave:** 2 — Core Providers + Admin

---

## Why

Anthropic and Gemini are the two most important non-OpenAI providers. Getting their translators right early validates the converter pattern.

---

## Scope

### In scope
- `internal/providers/anthropic/` — provider, chat converter, error converter.
- `internal/providers/gemini/` — provider, chat converter, embedding converter, error converter.
- Format translation in `internal/platform/translator.go`:
  - OpenAI messages → Anthropic Messages API format.
  - OpenAI messages → Gemini format.
  - Tool schema normalization (strip `enumDescriptions`, empty `pages`).
  - Reasoning/thought content separation.
- Update `/v1/chat/completions` to route to Anthropic/Gemini when model resolves there.

### Out of scope
- Audio/image support for Gemini (Phase 11).
- Advanced translator debug UI (Phase 13).

---

## Verification

### Tests
1. Anthropic chat converter maps system prompt correctly.
2. Anthropic tool use response maps back to OpenAI tool_calls.
3. Gemini chat converter handles text + image content blocks.
4. Gemini embedding converter forwards `dimensions`.
5. Streaming works for both providers with correct SSE output.

### Manual verification
1. Send chat request with `model=anthropic/claude-3-5-sonnet` and verify response.
2. Send chat request with `model=gemini/gemini-1.5-pro` and verify response.

---

## Tasks

1. Implement Anthropic provider package.
2. Implement Gemini provider package.
3. Implement translator helpers.
4. Add fixture tests for both providers.
5. Update router to support these providers.
6. Verify gates.

---

## Risks

| Risk | Mitigation |
|------|------------|
| Anthropic tool_call indexing differs | Map index explicitly in converter; test parallel tool calls. |
| Gemini content block format is complex | Support text, image, and file parts incrementally. |
| Streaming deltas differ | Normalize deltas in provider-specific stream handlers. |
