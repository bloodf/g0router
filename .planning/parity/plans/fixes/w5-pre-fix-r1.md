# w5-pre fix micro-plan — diff-gate round 1 (Fable 5, 2026-06-12)

Source: `artifacts/w5-pre-debt-closure-diff-scoped-gpt.txt` (cycle 1, REJECT).
Per-finding triage with verbatim findings:

## Finding 1 (BLOCKER) — "internal/api/chat_test.go:612 — Required combo tests were
scoped to a stored combo/full production chain, but these use `fakeComboDispatcher`,
so the production `newComboDispatcher`/`ComboEngine` wiring can be dead or broken and
still pass."
TRIAGE: REAL (the api-level tests correctly test the handler against the seam; the
PRODUCTION ADAPTER `newComboDispatcher` in `internal/server/server.go` is itself
untested). FIX: add `TestProductionComboDispatcherBridges` in
`internal/server/server_test.go`: t.TempDir() store with a provider, one connection,
and combo `best=[m1,m2]`; build the EXACT production chain server.New builds
(`NewCooldownEngine(st, time.Now)` → `NewSelectionEngine(st, st, cd, time.Now)` →
`NewAccountRunner(...)` → `NewComboEngine(st, st, runner, time.Now, <no-op sleep>)`
→ `newComboDispatcher(...)`); assert (a) `IsCombo("best")` true / `IsCombo("m1")`
false, (b) `ExecuteCombo("best", fn)` invokes fn with the seeded connection's ID and
non-empty credential, (c) an fn returning a quota-verdict error for m1 falls through
to m2 (engine semantics through the bridge). Test-first: write, run, MUST PASS
against existing code (this is a coverage gap fix, not a behavior change) — if it
FAILS, the wiring defect it exposes is fixed as part of this plan.

## Finding 2 (MAJOR) — "internal/server/server.go:31 — Production refresher/combo
wiring in `server.New` has no meaningful test; existing route tests were only updated
to pass `nil`."
TRIAGE: ARCHITECTURAL CONSTRAINT, REBUTTED with recorded precedent + partial fix.
`server.New` returns only `*fasthttp.Server` with no introspection seam; provider
base URLs are catalog-hardcoded — the EXACT constraint recorded in w4-pre's
diff-gate disposition ("server.New creates infRouter locally with no injectable
seam … Binary grep checks are the binding acceptance criteria for server.go
wiring"). Binding structural checks here: `grep -c 'newComboDispatcher'
internal/server/server.go` ≥ 1, `grep -c 'SetCredentialRefresher\|refresher'
internal/server/routes_openai.go` ≥ 1 (verified live). The CONSTRUCTION path is
covered by Finding-1's new test (same chain server.New builds).

## Finding 3 (MAJOR) — "internal/server/routes_openai.go:11 — `RegisterOpenAIRoutes`
setter behavior for non-nil `refresher` and `comboDisp` is untested; no test proves
`ChatHandler` receives either dependency."
TRIAGE: REAL for comboDisp (behaviorally provable), REBUTTED-as-compile-check for
refresher. FIX: (a) `TestRegisterOpenAIRoutesPlumbsComboDispatcher` in
`internal/server/routes_openai_test.go`: register routes passing an api-typed fake
dispatcher (IsCombo("combomodel")=true; ExecuteCombo returns a canned success
response via fn) → serve POST /v1/chat/completions {"model":"combomodel"} through
`r.Handler` → 200 with the canned content proves the dispatcher reached the
ChatHandler through RegisterOpenAIRoutes (nil-dispatcher control request returns
the normal resolve error instead). (b) compile-time assertion in
`internal/server/server_test.go`: `var _ api.CredentialRefresher =
(*auth.CredentialResolver)(nil)` — proves the production resolver satisfies the
handler dependency; the refresher RETRY behavior is already covered by api-level
tests (chat_test.go refresh-retry suite, w4-f) and the resolver by
TestRefreshCredentialsByConnectionID (w5-pre Task 1).

## Ownership
TESTS ONLY: `internal/server/server_test.go`, `internal/server/routes_openai_test.go`.
Production code changes ONLY if Finding-1's test exposes a real wiring defect.

## Binary acceptance
- `go build ./... && go vet ./... && go test ./...` green; `go test -race ./internal/server/` green.
- TestProductionComboDispatcherBridges, TestRegisterOpenAIRoutesPlumbsComboDispatcher pass.
- `grep -c 'api.CredentialRefresher = (\*auth.CredentialResolver)(nil)' internal/server/server_test.go` = 1.
