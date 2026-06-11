# Fix micro-plan — w2-a diff-gate findings (2026-06-11)

Author: Fable 5. Implementer: kimi. Authorizing artifact:
`artifacts/w2-a-provider-catalog-diff-scoped-gpt.txt`. One real (test-only); two rebutted.

## Rebuttal — BLOCKER #1 & MAJOR #2 (openrouter tts Params) are FALSE POSITIVES

The finding claims openrouter TTS entries must carry `Params`. Verified against the
frozen ref `providerModels.js:302-320`: ONLY the 4 `type:"image"` entries have a
`params` field; ALL `type:"tts"` and `type:"embedding"` entries have NO `params`.
So the Go catalog correctly omits Params for tts/embedding — that is byte-faithful,
not data loss. `TestOpenRouterCatalogTypes` checking Params only on image entries is
likewise correct (image is the only type with params in the ref). NO change.

## Task 1 — assert exact BaseURLs in TestLookupKnownProviders (MAJOR #3, real, test-only)

`catalog_test.go` `TestLookupKnownProviders` only checks non-empty BaseURL. Strengthen
to assert the EXACT BaseURL for each of the 11 entries against the ref `providers.js`
values (e.g. deepseek `https://api.deepseek.com/chat/completions`, groq
`https://api.groq.com/openai/v1/chat/completions`, openrouter
`https://openrouter.ai/api/v1/chat/completions`, ollama `https://ollama.com/api/chat`,
ollama-local `http://localhost:11434/api/chat`, xai `https://api.x.ai/v1/chat/completions`,
mistral `https://api.mistral.ai/v1/chat/completions`, cohere `https://api.cohere.ai/v1/chat/completions`,
together `https://api.together.xyz/v1/chat/completions`, fireworks
`https://api.fireworks.ai/inference/v1/chat/completions`, perplexity
`https://api.perplexity.ai/chat/completions`) — byte-exact config is the catalog's purpose.

## Acceptance (binary)

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `TestLookupKnownProviders` asserts exact BaseURL strings for all 11 entries (covered by go test).
- Files touched ONLY: `internal/providers/catalog/catalog_test.go`. Do NOT git commit.

## Out of scope

openrouter tts/embedding Params (rebutted — ref has none). Any production change
(the catalog data is correct; this is test-strengthening only).
