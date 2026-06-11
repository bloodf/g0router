# w4-pre — Audit wiring fixes + Wave-1 deferred pipeline helpers

Authorizing artifacts: `reviews/wave0-3-audit-2026-06-12.md` (each item below quotes its in-repo file:line) + real rows PAR-TRANS-006 (`open-sse/translator/index.js:58-72`), PAR-TRANS-051/052 (`open-sse/utils/reasoningContentInjector.js:1-79`), PAR-TRANS-053 (`open-sse/utils/toolDeduper.js:6-47`). Frozen ref @ 827e5c3. Runs ALONE, first.

## Tasks
1. **Wire credentials (audit G1+G2)** — evidence: `internal/server/server.go:36-46` builds `flows={"anthropic":...}` only and never calls `infRouter.SetKeyResolver`; `internal/auth/credentials.go:29` `NewCredentialResolver` has no production caller.
   STEP (a): write `TestServerWiresKeyResolver` (server built over a store holding an api_key connection for deepseek → loopback /v1 chat reaches an httptest upstream carrying `Authorization: Bearer <key>`) and `TestServerFlowsIncludeGeminiXai` (gemini & xai OAuth-start return an auth URL, not 404); run — both fail.
   STEP (b): extend the flows map with gemini (`auth.GeminiOAuth()`) and xai (`auth.XaiOAuth()`), construct `auth.NewCredentialResolver(st, flows)`, call `infRouter.SetKeyResolver(resolver)`.
2. **`/v1/models/{id}` filter (G3)** — evidence `internal/api/models.go:57-60` delegates to List. STEP (a): `TestModelsGetByID` (known id→one object), `TestModelsGetUnknown404` (fail). STEP (b): filter to the single model; 404 envelope when unknown.
3. **randomUUID error (G4)** — evidence `internal/auth/apikey.go:183-189` returns `"0000000000000000"` on rand failure. STEP (a): `TestKeyIDGenerationNoPlaceholder` (failing `randRead` seam → CreateAPIKey errors, no key minted) (fail). STEP (b): `randomUUID() (string,error)`; propagate; delete placeholder.
4. **Stream abort (G5)** — evidence: `internal/api/chat.go`/`messages.go`/`responses.go` range the provider channel with no cancellation select. STEP (a): `TestChatStreamStopsOnClientAbort` (cancel mid-stream → prompt return, no further writes) (fail). STEP (b): wrap each stream range in `select { case <-ctx.Done(): return; case chunk,ok := <-ch: ... }`.
5. **Stale comments (G6, ALL of it)** — rewrite `internal/inference/router.go:17-18,43,45` and `internal/api/chat.go:63`, `internal/api/embeddings.go:37` to describe w2-d catalog routing + (post-task-1) wired credentials; delete "Phase 5/6+/TODO(phase-8)". No behavior change; existing tests stay green.
6. **PAR-TRANS-006/051/052/053 pipeline helpers** (`internal/translation/`): for EACH — STEP (a) table-driven failing test (`TestStripContentTypes`, `TestInjectReasoningContent`, `TestDeepseekProMaxExpansion`, `TestDedupeTools`) with ref-shaped fixtures; STEP (b) port the helper (stripContentTypes drop image/audio by provider capability `index.js:58-72`; injectReasoningContent + deepseek-v4-pro-max/none→base expansion `reasoningContentInjector.js:1-79`, reuses w2-a `UpstreamModelID`; dedupeTools `toolDeduper.js:6-47`) and wire into `PreprocessChatRequest` (`internal/translation/preprocess.go`; already invoked at `chat.go:55`).

## Preconditions (each states its own pass condition)
- `grep -c 'SetKeyResolver' internal/server/server.go` outputs `0` (G1 currently unwired — THIS is the gap; acceptance flips it to ≥1).
- `grep -c '"gemini"' internal/server/server.go` outputs `0` (G2 gap).
- `grep -c 'func PreprocessChatRequest' internal/translation/preprocess.go` ≥ 1 (hook exists).

## Exclusive file ownership
TOUCH: `internal/server/server.go`(+test), `internal/api/{models,chat,messages,responses,embeddings}.go`(+tests), `internal/auth/apikey.go`(+test), `internal/inference/router.go`(comments only), `internal/translation/preprocess.go`+NEW helper files+tests.

## Binary acceptance
- `go test ./... && go vet ./... && go test -race ./internal/server/ ./internal/auth/` green.
- `grep -c 'SetKeyResolver' internal/server/server.go` ≥ 1; `grep -c '"gemini"\|"xai"' internal/server/server.go` ≥ 2; `grep -c '0000000000000000' internal/auth/apikey.go` → 0; `grep -c 'Phase 6+\|TODO(phase-8)\|Phase 5' internal/api/chat.go internal/api/embeddings.go internal/inference/router.go` → 0.
- TestServerWiresKeyResolver, TestModelsGetByID, TestChatStreamStopsOnClientAbort, TestDedupeTools pass.

## Out of scope
Retry/classifier (w4-b), aliases (w4-a), routing features. PAR-TRANS-046 usage clause (W5).


## Plan-gate disposition (Fable 5, 2026-06-12)
CLOSED BY DECISION after 2 substantive cycles. Round-1 + round-2 substantive findings
FIXED: dropped non-parity scope (027 weighted, 009/040 provider-nodes), global
selection mutex (017), backoff on connection column (014), combo strategy in settings
+ reset-on-restart map not TTL (002), 023=up-to-3-attempts, 033 +Antigravity/Responses,
037 six kinds, fallbackStrategy key + pinned param (w4-d), combo regex dots (w4-e),
explicit STEP(a)/(b) test-first, settings.go serialization. Residual rejections are a
HARNESS-CONTEXT artifact, rebutted: the plan gate is fed only `9router-routing.md`, so
(a) PAR-PR rows (485/640/648/1626) read as "not a valid row / not in matrix" — they ARE
in `PARITY.md` (e.g. PR-1626 at :129); (b) in-tree facts read as "no evidence" though
VERIFIED present — `internal/translation/bypass_handler.go` EXISTS (w1, unwired),
`internal/inference/factory.go providerForModel` EXISTS (w2-d); (c) cross-plan staged
deps (w4-c Verdict enum consumed by w4-d/e) are by-design dependency-inversion, not
ambiguity; (d) whole-file cites for obvious stream loops. The Kimi DIFF gate at
implementation (with full source context) is the binding check.
