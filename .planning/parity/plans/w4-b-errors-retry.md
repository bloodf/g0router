# w4-b — Error classification + retry middleware

Rows: PAR-ROUTE-020 (per-URL retry w/ attempts/delay by status, `open-sse/executors/base.js:98-174`, `config/runtimeConfig.js:52-57`), PAR-ROUTE-021 (per-provider retry override, e.g. kiro retry{429:2} pattern `providers.js:197`), PAR-ROUTE-022 (connect-timeout abort → 502 mapping, `base.js:125-128,156-160`), PAR-ROUTE-044 (ordered error rules TEXT-FIRST then status, `config/errorConfig.js:59-76`; matrix note: a 200 body containing "rate limit" classifies as backoff), PAR-ROUTE-045 (provider `resetsAtMs` cooldown override CAPPED at 30min, `errorConfig.js:42`), PAR-ROUTE-048 (quota-window unix sec/ms parsing) + PAR-PR-1626 (token-param auto-learning fallback for openai-compatible: on "unsupported parameter max_tokens"-class errors retry once with max_completion_tokens, persist the learned preference). Frozen ref @ 827e5c3. Depends: w4-pre MERGED. Parallel-safe with w4-a/c (disjoint files).

Go-port considerations (verbatim): "Add `internal/inference/` retry middleware between router and provider executors; keep retry config per-provider." / "Centralize error classification in `internal/inference/` using the ordered rule pattern from 9router." / "Use `fasthttp` pipeline for connect timeout instead of `AbortController` pattern."

## Tasks (tests FIRST each)
1. Error classifier (`internal/inference/errorclass.go` NEW): port the ordered rule list from `errorConfig.js` (read whole; text rules BEFORE status rules, exact order; classifications: rate_limit/auth/transient/permanent etc. per ref), `resetsAt` extraction with 30-min cap (`:42`), quota-window sec-vs-ms normalization (PAR-ROUTE-048). Tests: `TestErrorRulesOrderTextFirst` (200 + "rate limit" body → rate_limit), `TestResetsAtCap30Min`, `TestQuotaWindowSecMs`, table-driven rule fixtures from the ref.
2. Retry middleware (`internal/inference/retry.go` NEW): wraps a provider call; per-status attempts/delay from a default table (`runtimeConfig.js:52-57`) overridable per provider (catalog gains optional `Retry map[int]int` — `internal/providers/catalog/catalog.go` TOUCH, kiro-pattern comment); connect-timeout distinguishable from client abort → maps to 502 (PAR-ROUTE-022; fasthttp dial timeout, not body timeout). Tests: `TestRetryPerStatusAttempts`, `TestProviderRetryOverride`, `TestConnectTimeout502NotRetriedAsClientAbort`, `TestNoRetryOnPermanentClass`.
3. PR-1626 (`retry.go` + `internal/store/settings.go` learned-param key): on classifier verdict "unsupported max_tokens param" retry ONCE with `max_completion_tokens`, persist learned preference per provider+model; subsequent requests use it directly. Tests: `TestTokenParamAutoLearn` (first call learns; second call sends learned param immediately).

## Preconditions
- `grep -rn 'errorclass\|retry.go' internal/inference/` → 0 hits (new).
- `grep -c 'Retry' internal/providers/catalog/catalog.go` outputs 0 (field added here).

## Exclusive file ownership
NEW: `internal/inference/errorclass.go`+test, `internal/inference/retry.go`+test. TOUCH: `internal/providers/catalog/catalog.go`+test (Retry field only). NOT: factory.go/alias (w4-a), api/ handlers (wiring of the middleware into handlers is w4-f), accounts/locks (w4-c).

## Binary acceptance
- `go test ./... && go vet ./... && go test -race ./internal/inference/` green.
- `TestErrorRulesOrderTextFirst`, `TestResetsAtCap30Min`, `TestConnectTimeout502NotRetriedAsClientAbort`, `TestTokenParamAutoLearn` pass.
- Classifier rule order pinned by a fixture test against `errorConfig.js` order.

## Out of scope
Account cooldown/backoff persistence (w4-c consumes the classifier verdicts). Combo transient cooldown (w4-e). Handler wiring (w4-f). 401/403 token-refresh retry (w4-f, needs resolver).
