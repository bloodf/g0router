# w5-a fix micro-plan — diff-gate round 1 (Fable 5, 2026-06-12)

Source: `artifacts/w5-a-schema-pricing-diff-scoped-gpt.txt` (cycle 1, REJECT).
Per-finding triage with verbatim findings:

## Finding 1 (BLOCKER) — "internal/store/kv.go — Plan task 5 requires store-side
pricing read from kv scope='pricing' (JSON provider value) and store implementing
OverrideStore; diff only adds generic Set/Get/ListKV, so user pricing overrides
cannot be consumed."
TRIAGE: REAL. The concrete OverrideStore implementation is missing — only fakes
satisfy it. FIX: add to `internal/store/kv.go` (store stays a leaf — the return type
is plain maps, no usage import):
`func (s *Store) UserPricing() (map[string]map[string]map[string]float64, error)` —
`ListKV("pricing")` → for each provider key, json.Unmarshal the value into
`map[string]map[string]float64` (model → rate → value), skipping (with wrapped
error? NO — mirror ref: pricingRepo getUserPricing returns parsed map; a corrupt row
returns an error wrapped with provider context). Test FIRST:
`TestUserPricingReadsKV` (seed SetKV("pricing","gh",`{"gpt-5.3-codex":{"input":2.0}}`)
→ map contains gh/gpt-5.3-codex/input=2.0; empty scope → empty non-nil map; corrupt
JSON → error mentioning the provider key) in `internal/store/kv_test.go` — fails
(method missing) → implement. Add compile-time proof in
`internal/usage/pricing_test.go` is NOT possible without importing store (w5-a
layering); instead the binding check is the grep below + w5-f/w5-d wiring tests later.

## Finding 2 (MAJOR) — "internal/usage/pricing.go:105 — Merged returns
store.UserPricing errors unwrapped, violating repo convention to wrap errors with
context."
TRIAGE: REAL (AGENTS.md errors-are-values convention: wrap with fmt.Errorf
"context: %w"). FIX: wrap every OverrideStore error surface in
`internal/usage/pricing.go` (`Merged`, `PricingForModel`) as
`fmt.Errorf("user pricing: %w", err)`. Extend an existing test with a failing fake
asserting `errors.Is` still matches and the message carries context.

## Finding 3 (MAJOR) — "internal/usage/pricing.go:192 — PricingForModel matches user
overrides by stripped baseModel as well as full model; plan/ref specify user override
first by exact provider/model, with baseModel stripping in constants resolution only."
TRIAGE: REAL. Ref `pricingRepo.js:51-56` checks `userPricing[provider]?.[model]`
EXACTLY — no vendor-prefix stripping on the user-override step. FIX: remove the
baseModel lookup from the user-override branch (keep stripping ONLY inside
`ResolvePricing` constants chain per `pricing.js:235-238`). Test FIRST: extend
`TestResolvePricing`/add case — user override stored for "deepseek-chat", query
provider+model "deepseek/deepseek-chat" → user override NOT used (falls through to
constants); query exact "deepseek-chat" → user override used. Run failing → fix.

## Ownership
`internal/store/kv.go`(+test), `internal/usage/pricing.go`(+tests). No other files.

## Binary acceptance
- `go build ./... && go vet ./... && go test ./...` green; `go test -race ./internal/store/ ./internal/usage/` green.
- `grep -c 'func (s \*Store) UserPricing' internal/store/kv.go` = 1.
- `grep -c 'fmt.Errorf' internal/usage/pricing.go` ≥ 2.
- TestUserPricingReadsKV passes; the new exact-match override case passes.
