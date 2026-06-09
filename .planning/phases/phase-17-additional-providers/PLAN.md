# Phase 17: Additional Providers

**Phase:** 17  
**Goal:** Add the remaining providers: Groq, Mistral, Cohere, DeepSeek, MiniMax, Fireworks, Together, Ollama, Bedrock, Vertex.  
**Requirements:** PROV-06..07  
**Estimated duration:** 5–7 days  
**Wave:** 5 — 9router Features

---

## Why

20+ provider support is a core promise. These providers round out the catalog.

---

## Scope

### In scope
- `internal/providers/groq/`, `mistral/`, `cohere/`, `deepseek/`, `minimax/`, `fireworks/`, `together/`, `ollama/` — chat + models.
- `internal/providers/bedrock/`, `vertex/` — chat + embeddings with cloud auth.
- OpenAI-compatible providers use minimal converter passthrough.
- Catalog seed data updated for all new providers.

### Out of scope
- Provider-specific features not covered by OpenAI-compatible surface.

---

## Verification

### Tests
1. Each provider package has fixture-based converter tests.
2. Each provider implements chat completion and list models.
3. Bedrock and Vertex implement embeddings.
4. Catalog lookup returns entries for all new providers.

### Manual verification
1. Configure one new provider and run a chat completion.

---

## Tasks

1. Implement each provider package following the OpenAI provider pattern.
2. Add seed catalog entries.
3. Write fixture tests.
4. Verify gates.

---

## Risks

| Risk | Mitigation |
|------|------------|
| Too many providers at once | Implement in priority order: Groq, Mistral, Bedrock, Vertex, then the rest. |
| AWS/GCP auth complexity | Use standard SDK patterns; test with fixture data first. |
