# Phase 2: Schemas + Catalog

**Phase:** 02  
**Goal:** Define all shared Go types and build the model catalog with pricing lookup.  
**Requirements:** CATALOG-01..08, OPENAI-11  
**Estimated duration:** 3–4 days  
**Wave:** 1 — Foundation

---

## Why

The provider interface, API handlers, and governance layer all depend on shared schemas and a model catalog. Building these first prevents cascading rewrites later.

---

## Scope

### In scope
- Define shared Go types in `internal/schemas/`:
  - `chat.go` — chat request/response types
  - `responses.go` — Responses API types
  - `embedding.go` — embedding types
  - `images.go` — image generation types
  - `audio.go` — speech/transcription types
  - `files.go`, `batch.go` — file and batch types
  - `errors.go` — uniform `ProviderError` + OpenAI `ErrorResponse`
  - `catalog.go` — model/pricing types
  - `governance.go` — virtual key/provider config types
  - `provider.go` — `Provider` capability interface
- Implement `internal/catalog/`:
  - Load built-in seed JSON at startup.
  - Thread-safe in-memory lookup table.
  - `Lookup(provider, model, mode)` with fallback chain.
  - `GetModelsForProvider`, `GetProvidersForModel`, `IsModelAllowedForProvider`.
  - `CalculateCost(provider, model, mode, usage)`.
  - Background sync from upstream pricing sheet.
- Seed catalog JSON with top providers and models (OpenAI, Anthropic, Gemini, Groq, Mistral, etc.).

### Out of scope
- Provider implementations.
- API handlers.
- Dashboard UI.

---

## Verification

### Tests
1. Catalog loads seed data and returns entries for known models.
2. Cross-provider resolution returns expected candidates.
3. `IsModelAllowedForProvider` correctly handles `["*"]`, explicit lists, and empty lists.
4. Cost calculation matches expected values from fixture usage.
5. Background sync updates pricing without crashing when upstream is unavailable.

### Manual verification
1. Run `go test ./internal/catalog/...` and see all green.
2. Print catalog lookup results from a small CLI program.

---

## Tasks

1. Define all schema types.
2. Define `Provider` interface.
3. Implement catalog data structures and lookup methods.
4. Write seed catalog JSON.
5. Implement background sync with fallback to seed.
6. Write table-driven tests.
7. Verify gates.

---

## Risks

| Risk | Mitigation |
|------|------------|
| Schema types are incomplete | Base them on OpenAI API spec + BiFrost schema references; iterate as providers are added. |
| Pricing seed becomes stale | Background sync + custom overrides; not a blocker for phase completion. |
| Cross-provider resolution is wrong | Use explicit fixture tests for `claude-3-5-sonnet` and `gpt-4o`. |
