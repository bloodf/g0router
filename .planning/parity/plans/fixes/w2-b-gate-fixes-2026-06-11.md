# Fix micro-plan — w2-b diff-gate findings (2026-06-11)

Author: Fable 5. Implementer: kimi. Authorizing artifact:
`artifacts/w2-b-generic-openai-adapter-diff-scoped-gpt.txt`. One rebutted; one real (test).

## Rebuttal — BLOCKER (buffered, "not streaming") is FALSE POSITIVE / in-repo architecture

The finding says `ChatCompletionStream` calls blocking `p.client.Do` before
returning the channel, so it buffers rather than streams. The MERGED openai
provider does EXACTLY this: `internal/providers/openai/chat.go:104` `p.client.Do(req, resp)`
then `:135` `body := bytes.NewReader(resp.Body())` + `NewSSEScanner` — the accepted
g0router fasthttp `ClientPool` pattern (the pool reads the full body; the goroutine
then scans+emits chunks). w2-b mirrors it faithfully. Incremental wire-streaming
would require changing the shared `utils.ClientPool` for ALL providers (openai too),
which is a cross-cutting architecture change OUT OF w2-b's scope. The final emitted
chunks are identical; only arrival latency differs, and it matches the merged
openai provider. NO change — rebut with the openai/chat.go cites.

## Task 1 — add the AUD-047 post-hook failure test (MAJOR, real)

`chat.go` implements post-hook-failure → in-band streamError but has no test.
Add `TestGenericStreamPostHookError`: an httptest server streams one valid chunk;
a `postHookRunner` whose `Run` returns an error is passed; assert the stream emits
the chunk then an in-band `streamError` and closes (mirror the openai provider's
behavior at `openai/chat.go:158-164`).

## Acceptance (binary)

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -c 'TestGenericStreamPostHookError' internal/providers/generic/chat_test.go` ≥ 1 and it passes.
- Files touched ONLY: `internal/providers/generic/chat_test.go`. Do NOT git commit.

## Out of scope

Incremental wire-streaming (rebutted — matches merged openai/chat.go; cross-cutting ClientPool change). Any production change to chat.go (the buffered pattern is faithful).
