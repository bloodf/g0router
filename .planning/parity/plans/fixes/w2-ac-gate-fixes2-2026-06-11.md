# Fix micro-plan — w2-a + w2-c residual diff-gate findings (2026-06-11)

Author: Fable 5. Implementer: kimi. Disjoint packages (catalog vs ollama).
Artifacts: w2-a-/w2-c-provider...diff-scoped-gpt.txt (latest rounds).

## w2-a Task — exact Format + ollama-ID assertions (test-only, real)

`internal/providers/catalog/catalog_test.go` `TestLookupKnownProviders`: also assert
the EXACT `Format` per entry (openai for the 9; "ollama" for ollama+ollama-local).
`internal/providers/catalog/models_test.go` `TestModelsForOllama`: assert the EXACT
6 ollama model IDs (gpt-oss:120b, kimi-k2.5, glm-5, minimax-m2.5, glm-4.7-flash,
qwen3.5) — not just length.

## w2-c Task 1 — dedicated post-hook test (real)

The corrected malformed test (skip-semantics) removed post-hook coverage. Add
`TestOllamaStreamPostHookError` to `internal/providers/ollama/chat_test.go`: httptest
streams a valid NDJSON line; a `postHookRunner` returning an error → assert the chunk
then an in-band `streamError` then close (mirror `openai/chat.go:158-164`).

## w2-c Task 2 — restrict New to explicit IDs (real)

`internal/providers/ollama/provider.go` `New`: restrict to `id == "ollama" || id == "ollama-local"`
explicitly (in addition to the Format check), per the plan's Task-1 contract.
Test `TestNewOllamaRejectsNonOllama` already covers rejection; add a case that a
hypothetical non-ollama-id with Format "ollama" is also rejected (defensive).

## w2-c Task 3 — fix stale comment (real, MINOR)

`internal/providers/ollama/chat.go` ~:278: the comment says malformed chunks abort
with an in-band error; the scanner SKIPS malformed NDJSON before this path. Update
the comment to state malformed NDJSON is skipped by the scanner (sse.go:71-78); the
in-band error path here covers post-hook/read failures only.

## Acceptance (binary)

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `TestLookupKnownProviders` asserts exact Format; `TestModelsForOllama` asserts the 6 exact IDs.
- `TestOllamaStreamPostHookError` exists and passes; `New` rejects non-ollama ids.
- Files touched ONLY: `catalog_test.go`, `models_test.go`, `ollama/chat_test.go`, `ollama/provider.go`, `ollama/chat.go` (comment). Do NOT git commit.

## Out of scope

w2-b (closed by decision). Production catalog/adapter logic beyond the ID check + comment.
