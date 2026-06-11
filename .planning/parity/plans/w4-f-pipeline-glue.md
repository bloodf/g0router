# w4-f — Request pipeline glue: detection, bypass, passthrough, refresh-retry, kind routes

Rows: PAR-ROUTE-033 (format auto-detection — OpenAI, Claude, Gemini, **Antigravity, Responses** — `open-sse/services/provider.js:49-126`), PAR-ROUTE-034 CANONICAL (Claude-CLI bypass patterns — the w1-ported `internal/translation/bypass_handler.go` exists but is UNWIRED, verified zero refs from internal/api; wire it, `utils/bypassHandler.js:11-91`), PAR-ROUTE-041 (native passthrough, `handlers/chatCore.js:86-103`), PAR-ROUTE-042 (thinking-config override), PAR-ROUTE-043 (stream vs non-stream decision), PAR-ROUTE-023 (token refresh on 401/403 — `refreshWithRetry` UP TO 3 ATTEMPTS, `handlers/chatCore.js:216-235`), PAR-ROUTE-052 (refresh-before-dispatch via the w4-pre-wired resolver), PAR-ROUTE-037 (`/v1/models/{kind}` — kinds **image, tts, stt, embedding, image-to-text, web**, `src/app/api/v1/models/[kind]/route.js:1-55`), PAR-ROUTE-038 (model-test routing by kind). Frozen ref @ 827e5c3. Depends: w4-a, w4-b MERGED.

VERIFY-FLIPS (separate task, evidence-cited): 028 (API-key validation `requireApiKey`) + 029 (key extraction Bearer/x-api-key) — w3-d delivered these; evidence `internal/auth/apikey.go NewAPIKeyValidator`, `internal/server/guard_test.go TestGuardV1RemoteValidKey`. 035 Stage-1 half (single-URL building) + 036 Stage-1 half (header building) — w2-b; evidence `internal/providers/generic/chat.go`, `generic/chat_test.go TestGenericChatCustomHeaders`.

## Tasks (STEP (a) failing tests FIRST; STEP (b) implement) — each task atomic
1. **Format auto-detect** (`internal/api/detect.go` NEW). (a) `TestFormatAutoDetect` table incl. OpenAI/Claude/Gemini/Antigravity/Responses bodies (all 5 per `provider.js:49-126`). (b) port the full detection precedence.
2. **Bypass wiring** (`internal/api/chat.go`,`messages.go` TOUCH). (a) `TestBypassWarmupShortCircuits`, `TestBypassTitleSkip` (no provider call — fake provider). (b) call the existing translation bypass handler before dispatch (`bypassHandler.js:11-91`).
3a. **Native passthrough** (PAR-ROUTE-041 + the w1-g2 PAR-TRANS-050b responses-passthrough deferral). (a) `TestNativePassthroughSkipsTranslation`. (b) when client format == provider format, skip translate.
3b. **Thinking override** (042). (a) `TestThinkingOverrideInjected`. (b) inject per provider config.
3c. **Stream decision** (043). (a) `TestStreamDecision`. (b) stream vs non-stream branch logic.
4. **Refresh-retry** (023/052). (a) `TestRefreshRetryUpTo3On401`, `TestNoRefreshLoopBeyond3` (matrix: refreshWithRetry UP TO 3 ATTEMPTS — not once). (b) on 401/403 verdict force-refresh via resolver + retry up to 3 (`chatCore.js:216-235`); refresh-before-dispatch already via resolver shouldRefresh (052 assert).
5. **Kind routes** (037/038). (a) `TestModelsByKind` (each of image/tts/stt/embedding/image-to-text/web filters catalog `Type`; note: g0router catalog Type currently ∈ llm/embedding/stt/image/tts — map image-to-text/web per ref; if a kind has no catalog members it returns empty list, not 404), `TestModelTestRoutesByKind`. (b) `/v1/models/{kind}` filter + model-test-by-kind route.
6. **VERIFY-FLIPS** (no new code unless gap). Cite & run the existing tests above for 028/029/035/036; if any behavior is actually missing → IMPL-BLOCKED with the gap. Otherwise the impl report records the citations and these rows flip at merge.

## Preconditions
- `grep -rn 'bypass' internal/api/` → 0 hits (unwired — the gap).
- `grep -c 'func.*Classify' internal/inference/errorclass.go` ≥ 1 (w4-b); `grep -c 'ResolveModelAlias' internal/inference/alias.go` ≥ 1 (w4-a).

## Exclusive file ownership
NEW: `internal/api/detect.go`+test. TOUCH: `internal/api/{chat,messages,responses,models}.go`+tests, `internal/server/routes_openai.go` (kind routes). This is the ONLY remaining Wave-4 plan editing internal/api; it dispatches AFTER w4-pre/w4-c/w4-e have made their models.go touches (serial). Combo dispatch into the chat path: if w4-e merged, wire it here; else a noted follow-up task.

## Binary acceptance
- `go test ./... && go vet ./... && go test -race ./internal/api/ ./internal/inference/` green.
- `grep -c 'bypass' internal/api/chat.go` ≥ 1; TestFormatAutoDetect covers all 5 formats; TestRefreshRetryUpTo3On401, TestNativePassthroughSkipsTranslation, TestModelsByKind pass.
- Verify-flip section cites file:line for 028/029/035/036 existing tests.

## Out of scope
Provider-node detection (Stage-2). VK routing (W5). Request logging (W5). Free-tier/Stage-2 provider rows.


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
