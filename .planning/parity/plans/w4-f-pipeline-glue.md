# w4-f — Request pipeline glue: detection, bypass, passthrough, refresh-retry, kind routes

Rows: PAR-ROUTE-033 (request format auto-detection, `open-sse/services/provider.js:49-126`), PAR-ROUTE-034 CANONICAL (bypass patterns for Claude CLI — the w1-ported `internal/translation/bypass_handler.go` exists but is UNWIRED (verified: zero references from internal/api); wire it into the chat/messages path per `utils/bypassHandler.js:11-91`), PAR-ROUTE-041 (native passthrough detection — same-ecosystem skip-translation, `handlers/chatCore.js:86-103`), PAR-ROUTE-042 (provider thinking-config override injection), PAR-ROUTE-043 (streaming vs non-streaming decision, `chatCore.js`), PAR-ROUTE-023 (401/403 token-refresh-and-retry, `chatCore.js:216-235` — uses the w4-pre-wired CredentialResolver), PAR-ROUTE-052 (refresh-before-dispatch — resolver shouldRefresh already does the check; this row WIRES the dispatch path assertion + flips), PAR-ROUTE-037/038 (/v1/models/{kind} + model-test-by-kind; Go-port consideration #7) + VERIFY-FLIPS: 028/029 (w3-d guard key validation/extraction — verify tests exist, flip), 035/036 Stage-1 halves (w2-b URL/header building — verify, flip with deferral note for multi-URL/spoof halves). Frozen ref @ 827e5c3. Depends: w4-a, w4-b MERGED (∥ c/d acceptable; combo dispatch via w4-e is wired HERE last if e merged, else a follow-up task noted).

## Tasks (tests FIRST each)
1. Format auto-detection (`internal/api/detect.go` NEW): port `provider.js:49-126` (OpenAI vs Claude vs Gemini request-shape detection rules, exact precedence). Tests: table-driven `TestFormatAutoDetect` with ref-shaped bodies.
2. Bypass wiring (`internal/api/chat.go`/`messages.go` TOUCH): call the EXISTING `translation` bypass handler (w1) for Claude-CLI warmup/count/title/skip patterns BEFORE provider dispatch (`bypassHandler.js:11-91` semantics — short-circuit responses). Tests: `TestBypassWarmupShortCircuits`, `TestBypassTitleSkip` (no provider call — assert via fake provider).
3. Passthrough + thinking + stream decision (`internal/api/` + `internal/inference/`): native passthrough when client format == provider format (041, `chatCore.js:86-103` — skip translate, body passthrough; the w1-g2 PAR-TRANS-050b deferral lands here for responses-passthrough); thinking override injection per provider config (042); streaming decision logic (043). Tests: `TestNativePassthroughSkipsTranslation`, `TestThinkingOverrideInjected`, `TestStreamDecision`.
4. Refresh-retry (023/052): on 401/403 classifier verdict, force-refresh via CredentialResolver and retry ONCE (`chatCore.js:216-235`); assert refresh-before-dispatch happens via the resolver path (052). Tests: `TestRefreshRetryOn401Once`, `TestNoRefreshLoopOn403Twice`.
5. Kind routes (037/038): `/v1/models/{kind}` filtering by catalog `Type` (llm/embedding/stt/image/tts — the w2-a Type field becomes consumed) + model-test endpoint routing by kind. Tests: `TestModelsByKind`, `TestModelTestRoutesByKind`.
6. VERIFY-FLIPS: run/cite the existing w3-d guard tests (028/029) and w2-b header/URL tests (035/036 Stage-1 halves) in the report; no new code unless a gap is found (if found → IMPL-BLOCKED, report).

## Preconditions
- `grep -rn 'bypass' internal/api/` → 0 hits (unwired — the pass condition).
- `grep -c 'func.*Classify' internal/inference/errorclass.go` ≥ 1 (w4-b merged).
- `grep -c 'ResolveModelAlias' internal/inference/alias.go` ≥ 1 (w4-a merged).

## Exclusive file ownership
NEW: `internal/api/detect.go`+test. TOUCH: `internal/api/chat.go`, `messages.go`, `responses.go`, `models.go` + tests (the ONLY Wave-4 plan editing internal/api at this point — runs after w4-pre/c/e models.go touches per the serial order), `internal/server/routes_openai.go` (kind route). NOT: translation internals (bypass handler reused as-is), guard, store.

## Binary acceptance
- `go test ./... && go vet ./... && go test -race ./internal/api/ ./internal/inference/` green.
- `grep -c 'bypass' internal/api/chat.go` ≥ 1 (wired).
- `TestBypassWarmupShortCircuits`, `TestNativePassthroughSkipsTranslation`, `TestRefreshRetryOn401Once`, `TestModelsByKind` pass.
- Verify-flip evidence section in the impl report for 028/029/035/036.

## Out of scope
Combo dispatch wiring if w4-e unmerged at dispatch (noted follow-up). VK routing (W5). Request logging (W5). Free-tier/Stage-2 provider rows.
