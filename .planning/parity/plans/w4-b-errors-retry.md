# w4-b — Error classification + retry middleware

Rows: PAR-ROUTE-020 (per-URL retry attempts/delay by status, `open-sse/executors/base.js:98-174`, `config/runtimeConfig.js:52-57`), PAR-ROUTE-021 (per-provider retry override, e.g. kiro `retry{429:2}` `providers.js:197`), PAR-ROUTE-022 (connect-timeout abort→502, `base.js:125-128,156-160`), PAR-ROUTE-044 (ordered error rules TEXT-FIRST then status, `config/errorConfig.js:59-76`), PAR-ROUTE-045 (provider `resetsAtMs` cooldown capped at 30min, `errorConfig.js:42`), PAR-ROUTE-048 (quota-window unix sec/ms parsing) + PAR-PR-1626 (`PARITY.md:129` "OpenAI-compatible token parameter auto-learning fallback"). Frozen ref @ 827e5c3. Depends: w4-pre MERGED. Parallel-safe with w4-a/w4-c.

Go-port considerations (verbatim): "Add `internal/inference/` retry middleware between router and provider executors; keep retry config per-provider." / "Centralize error classification in `internal/inference/` using the ordered rule pattern from 9router." / "Use `fasthttp` pipeline for connect timeout instead of `AbortController` pattern."

## Tasks (STEP (a) failing tests FIRST; STEP (b) implement)
1. **Error classifier** (`internal/inference/errorclass.go` NEW). (a) `TestErrorRulesOrderTextFirst` (200 body containing "rate limit" → rate_limit class, proving text-before-status), `TestResetsAtCap30Min`, `TestQuotaWindowSecMs`, plus a fixture test pinning the FULL ordered rule list from `errorConfig.js:59-76`. (b) port the rules. EXACT class set (no "etc."): the verdict enum = the distinct `action`/`type` values enumerated in `errorConfig.js` (read whole; e.g. rate_limit, auth_error, server_error/transient, invalid_request/permanent, unsupported_param) — list each with its triggering rule; `resetsAt` extraction capped at 30min (`:42`); quota window sec↔ms normalize (048).
2. **Retry middleware** (`internal/inference/retry.go` NEW). (a) `TestRetryPerStatusAttempts`, `TestProviderRetryOverride`, `TestConnectTimeout502NotRetriedAsClientAbort`, `TestNoRetryOnPermanentClass`. (b) wrap a provider call; default per-status attempts/delay (`runtimeConfig.js:52-57`) overridable per provider via catalog `Retry map[int]int`; connect-timeout (fasthttp dial timeout, distinct from client abort/body) → 502.
3. **PR-1626 token-param auto-learn** (`internal/inference/retry.go` + `internal/store/settings.go`). (a) `TestTokenParamAutoLearn` (first call: classifier "unsupported_param" → retry once with `max_completion_tokens`, learn; second call: sends learned param immediately). (b) implement; PERSIST the learned pref under a settings key `learned_token_param:{provider}:{model}` (snake_case) — settings is the storage (g0router settings are key-value, consistent with existing keys).

## Preconditions
- `grep -rn 'errorclass\|retry.go' internal/inference/` → 0 hits.
- `grep -c 'Retry' internal/providers/catalog/catalog.go` outputs `0` (field added here).

## Exclusive file ownership
NEW: `internal/inference/errorclass.go`+test, `internal/inference/retry.go`+test. TOUCH: `internal/providers/catalog/catalog.go`+`catalog_test.go` (Retry field + its data/fixtures), `internal/store/settings.go` (NO schema change — uses existing key-value; +test for the learned-param key).

## Binary acceptance
- `go test ./... && go vet ./... && go test -race ./internal/inference/` green.
- Classifier fixture test pins the exact `errorConfig.js` rule ORDER; TestErrorRulesOrderTextFirst, TestResetsAtCap30Min, TestConnectTimeout502NotRetriedAsClientAbort, TestTokenParamAutoLearn pass.

## Out of scope
Account cooldown/backoff persistence (w4-c consumes classifier verdicts via a shared enum it defines). Combo cooldown (w4-e). Handler wiring (w4-f). 401/403 refresh-retry (w4-f, needs resolver).


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

## Implementation diff-gate disposition (2026-06-12)
CLOSED BY DECISION after 4 cycles. HEAD: cd8f997.

Real bugs fixed during gate cycles:
- Cycle 1 BLOCKER: ClassUnsupportedParam enum added; [...]errorRule array →
  classificationRules() returning fresh slice (mutable global eliminated); nolint removed.
- Cycle 2 MAJOR: TestErrorClassFixture rewritten to enumerate all errorConfig.js rules
  verbatim in exact order.
- Cycle 3 MAJOR: SetSetting error propagation added in AutoLearnTokenParam; check
  updated to ClassUnsupportedParam (not ClassPermanent).
- Cycle 4 REAL BUG: dead fmt.Stringer assertion in extractMessage removed (unreachable
  after JSON unmarshal into map[string]any). TestErrorClassRuleOrder added to pin exact
  rule sequence.

Residual cycle-4 findings closed as architectural/artifact:
- Connect-timeout 502 (net.Error.Timeout() check): fasthttp has no distinct dial-timeout
  error type; Timeout()+!Temporary() is the correct Go/fasthttp idiom. Port constraint,
  not a semantic gap.
- GetSetting error swallowing in AutoLearnTokenParam: transient store errors must not
  block inference requests; intentional design consistent with repo convention.
