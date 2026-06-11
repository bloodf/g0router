# w4-pre — Audit wiring fixes + Wave-1 deferred pipeline helpers

Rows: audit items G1-G6 (`reviews/wave0-3-audit-2026-06-12.md`, file:line therein) + PAR-TRANS-006 (`stripContentTypes`, `open-sse/translator/index.js:58-72`), PAR-TRANS-051 (`injectReasoningContent`, `open-sse/utils/reasoningContentInjector.js:1-79`), PAR-TRANS-052 (deepseek-v4-pro-max/none alias expansion, `reasoningContentInjector.js:20-71`), PAR-TRANS-053 (`dedupeTools`, `open-sse/utils/toolDeduper.js:6-47`) — the Wave-1 deferrals recorded in WAVE-MAP/w1 closure. Frozen ref @ 827e5c3. Runs ALONE (first Wave-4 dispatch; touches server composition root).

## Tasks (each: STEP (a) named failing tests FIRST, run, show fail; STEP (b) implement)

1. **G1+G2 — wire credentials end-to-end** (`internal/server/server.go:35-46`):
   extend the flows map with `"gemini": auth.NewOAuthFlow(auth.GeminiOAuth(), st, nil)`
   and `"xai": auth.NewOAuthFlow(auth.XaiOAuth(), st, nil)` (constructors exist,
   `internal/auth/oauth.go`); construct `resolver := auth.NewCredentialResolver(st, flows)`
   (`credentials.go:29`) and call `infRouter.SetKeyResolver(resolver)`
   (`router.go:36`). Remove the stale "Phase 6+" comments at `internal/api/chat.go:63`
   and `internal/api/embeddings.go:37` (G6 partial).
   Tests FIRST: `TestServerWiresKeyResolver` (build the server with a store containing
   an api_key-kind connection for deepseek; a loopback /v1 chat request reaches an
   httptest upstream with `Authorization: Bearer <that key>` — proving store→resolver→
   router→adapter end-to-end), `TestServerFlowsIncludeGeminiXai` (OAuth start routes
   for gemini and xai return a redirect/auth URL, not not-found).
2. **G3 — `/v1/models/{id}` filters** (`internal/api/models.go:57-60`): return ONLY
   the matching model object (404 with the standard error envelope when unknown);
   drop the "Phase 4" comment. Tests FIRST: `TestModelsGetByID` (known id → single
   object), `TestModelsGetUnknown404`.
3. **G4 — randomUUID propagates errors** (`internal/auth/apikey.go:183-189`): change
   `randomUUID()` to return `(string, error)`; callers (the key generator path,
   already error-returning) wrap it; DELETE the `"0000000000000000"` placeholder.
   Tests FIRST: `TestKeyIDGenerationNoPlaceholder` (inject failing rand via the
   existing `randRead` seam → CreateAPIKey returns error, no key minted).
4. **G5 — stream loops select on client abort**: in `internal/api/chat.go`,
   `messages.go`, `responses.go` stream sections, wrap the channel range in a select
   with `ctx.Done()` so a client disconnect stops reading (and the deferred cleanup
   runs). Tests FIRST: `TestChatStreamStopsOnClientAbort` (cancel mid-stream → handler
   returns promptly; no further chunk writes).
5. **G6 — stale comments**: rewrite `internal/inference/router.go:17-18,43,45` to
   describe the actual w2-d catalog routing (delete TODO(phase-8) — Wave-4 plans own
   that work now). No behavior change; covered by existing tests staying green.
6. **PAR-TRANS-006/051/052/053 — request-pipeline helpers** (`internal/translation/`):
   port `stripContentTypes` (drop image/audio parts per provider capability flags,
   `translator/index.js:58-72`), `injectReasoningContent` + the deepseek-v4-pro-max/
   none expansion (`reasoningContentInjector.js:1-79`; ties into the w2-a catalog
   `UpstreamModelID` mapping), `dedupeTools` (`toolDeduper.js:6-47`); wire into
   `PreprocessChatRequest` (`internal/translation/preprocess.go`, the existing hook
   `chat.go:55` already calls). Tests FIRST per helper (table-driven, ref-shaped
   fixtures): `TestStripContentTypes`, `TestInjectReasoningContent`,
   `TestDeepseekProMaxExpansion`, `TestDedupeTools`.

## Preconditions
- `grep -rn 'SetKeyResolver' internal/server/ cmd/` → 0 hits (G1 unwired — the pass condition).
- `grep -c '"gemini"\|"xai"' internal/server/server.go` outputs 0 (G2).
- `grep -c 'func PreprocessChatRequest' internal/translation/preprocess.go` ≥ 1 (hook exists).

## Exclusive file ownership
TOUCH: `internal/server/server.go` (+`server_test.go`), `internal/api/models.go`,
`chat.go`, `messages.go`, `responses.go` (+ their tests), `internal/auth/apikey.go`
(+test), `internal/inference/router.go` (comments only), `internal/translation/
preprocess.go` + NEW helper files + tests.
NOT: guard.go, oidc, limiter, credentials.go logic, any provider adapter.

## Binary acceptance
- `go test ./... && go vet ./... && go test -race ./internal/server/ ./internal/auth/` green.
- `grep -rn 'SetKeyResolver' internal/server/server.go` ≥ 1; `grep -c '"gemini"\|"xai"' internal/server/server.go` ≥ 2.
- `grep -c '0000000000000000' internal/auth/apikey.go` → 0.
- `grep -c 'Phase 6+\|TODO(phase-8)\|Phase 5' internal/api/chat.go internal/api/embeddings.go internal/inference/router.go` → 0.
- `TestServerWiresKeyResolver`, `TestModelsGetByID`, `TestChatStreamStopsOnClientAbort`, `TestDedupeTools` pass.

## Out of scope
Retry/error classification (w4-b), aliases (w4-a), any routing feature. Resolver
internals (w3-f, merged). PAR-TRANS-046 usage clause (Wave 5, per w1 closure).
