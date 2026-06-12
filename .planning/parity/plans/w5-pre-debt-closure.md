# w5-pre — W4 carry-forward debt closure: production refresher + production runner

Authorizing artifacts (verbatim, with file:line):
- `w4-f-pipeline-glue.md:77` (cycle-3 disposition): "Tracked: wire `SetCredentialRefresher`
  in OAuth wave."
- `w4-e-combos.md:50-56` (cycle-3 disposition): "The production runner that wraps real
  HTTP 502/503/504 responses with ErrModelTransient is a pipeline glue concern deferred
  to w4-f."
- `w4-f-pipeline-glue.md:22` (§Exclusive file ownership): "Combo dispatch into the chat
  path: if w4-e merged, wire it here; else a noted follow-up task." (It was not wired —
  precondition grep below proves the gap.)
- `WAVE-5-MAP.md` w5-pre row + §Ownership tracks (combo dispatch glue explicitly in
  w5-pre; w5-pre runs ALONE so its `internal/api/chat.go` touch precedes every
  concurrent W5 job — the "ONLY internal/api editor" rule binds the concurrent phase).
Frozen ref @ 827e5c3. Runs ALONE, first in Wave 5. No PAR rows flip here; this plan
makes already-flipped rows (PAR-ROUTE-023, 001-004/011/024) real in production.

## Tasks

1. **Production `RefreshCredentials(connectionID)` + wiring (w4-f debt)** — evidence:
   `internal/api/chat.go:96-116` defines `CredentialRefresher` + `SetCredentialRefresher`
   with ZERO production callers; `internal/auth/credentials.go:116-165` already has
   `doRefresh`/`refreshAndPersist` (in-flight dedup, token rotation, persistence) keyed
   by providerType+conn — only the by-connection-ID entry point is missing.
   STEP (a): write `TestRefreshCredentialsByConnectionID` (resolver over a store holding
   an OAuth connection whose flow token endpoint is an httptest server returning a rotated
   access+refresh token → method returns new access token AND `GetConnection` shows
   persisted rotation) and `TestRefreshCredentialsUnknownConnection` (unknown ID → error,
   no panic); run — both fail (method does not exist).
   STEP (b): add `func (r *CredentialResolver) RefreshCredentials(connectionID string)
   (string, error)`: `store.GetConnection(id)` → resolve the flow key from the
   connection's provider (same providers-table type lookup `ResolveKey` performs at
   `credentials.go:39+`) → `doRefresh(providerType, conn)` (force; not gated on
   `shouldRefresh` — the caller has already seen a 401/403) → return merged AccessToken.
   Wire: in `internal/server/server.go` build `flows`+resolver BEFORE
   `RegisterOpenAIRoutes` (keep inside `st != nil` branch; pass nil otherwise) and extend
   `RegisterOpenAIRoutes(r, infRouter, st, refresher api.CredentialRefresher)`
   (`internal/server/routes_openai.go:11`) to call `chat.SetCredentialRefresher(refresher)`
   when non-nil. ChatHandler only — `internal/api/messages.go` has no refresher seam
   (verified: zero `refresh` matches) and adding one is out of scope.

2. **Production ModelRunner: transient wrap (w4-e debt)** — evidence:
   `internal/inference/combo.go:21-32` declares `ErrModelTransient` + the `ModelRunner`
   interface; grep shows NO production implementer; `internal/inference/selection.go:226`
   `WithAccountFallback` is the account-fallback engine awaiting a caller;
   `internal/inference/factory.go:37` `providerForModel` maps model→providerID.
   STEP (a): write `TestAccountRunnerWrapsTransient` (table-driven: fn returns error
   carrying `*schemas.ProviderError` StatusCode 502/503/504 → `errors.Is(err,
   ErrModelTransient)` true; 400/401/429 → false, original error preserved via
   `errors.As`) and `TestAccountRunnerDelegatesToSelection` (fake ConnStore with two
   connections; first fn call returns quota verdict → second connection tried); run — fail.
   STEP (b): NEW `internal/inference/runner.go`: `AccountRunner` holding a
   `*SelectionEngine`; `RunModel(model, fn)` = `providerForModel(model)` →
   `WithAccountFallback(providerID, model, fn)`; on final error, if the chain carries a
   `*schemas.ProviderError` with StatusCode ∈ {502,503,504}, return
   `errors.Join(ErrModelTransient, err)` (both sentinel and original remain matchable).

3. **Combo dispatch glue in chat path (w4-f noted follow-up)** — evidence:
   `internal/api/chat.go:199` resolves via `ResolveForModel` with no combo branch;
   `internal/store/combos.go:35` `GetCombo`; `internal/inference/combo.go:143`
   `ExecuteCombo(name, fn)`; ref behavior `open-sse/services/combo.js:108-198` (combo
   chain executes per-model with account fallback; client errors surface from the last
   attempt).
   STEP (a): write `TestChatComboDispatchFallsBack` (store combo `best=[m1,m2]`; fake
   provider for m1 always returns ProviderError 503, m2 succeeds → POST
   /v1/chat/completions with model=best returns 200 with m2's response),
   `TestChatComboAllFailReturnsError` (both fail → upstream error envelope, not panic),
   and `TestChatComboStreamFallsBackPreStream` (model=best with stream=true; m1's
   ChatCompletionStream errors before any chunk, m2 streams → SSE body carries m2
   chunks); run — all fail (no combo branch).
   STEP (b): construct the production chain in `server.New` (`st != nil` branch):
   `NewCooldownEngine(st, time.Now)` → `NewSelectionEngine(st, st, cd, time.Now)` →
   `AccountRunner` → `NewComboEngine(st, st, runner, time.Now, time.Sleep)`; inject into
   ChatHandler via a new setter (`SetComboEngine`, interface-typed in api to preserve
   layering — api must NOT import store; follow the w4-e `ComboLister` precedent at
   `internal/api/models.go`). In `Handle`: after bypass + unmarshal, if
   `GetCombo(model)` hits, run `ExecuteCombo(name, fn)` where fn(model, conn) builds
   the per-model request (req copy with Model=model), resolves the provider for that
   model via the router, dispatches with `schemas.Key{ID: conn.ID, Value: conn's
   access-token-or-secret}`, and maps the outcome to a `Verdict` via w4-b's classifier
   (`internal/inference/errorclass.go`). Streaming combos: fallback applies only to
   errors surfaced BEFORE the stream channel opens (ref parity: combo.js falls back on
   executor error, never mid-stream); once a channel is open, stream it and return.

## Preconditions (each states its own pass condition)
- `grep -rc 'SetCredentialRefresher' internal/server/routes_openai.go` outputs `0` (THIS is the w4-f gap; acceptance flips it to ≥1).
- `grep -rn 'func (.*) RunModel' internal/inference/ --include='*.go' | grep -v _test | wc -l` outputs `0` (w4-e gap — interface has no production implementer).
- `grep -c 'ExecuteCombo' internal/api/chat.go` outputs `0` (combo dispatch unwired).
- `grep -c 'func (r \*CredentialResolver) doRefresh' internal/auth/credentials.go` ≥ 1 (refresh machinery exists; only the entry point is new).

## Exclusive file ownership
NEW: `internal/inference/runner.go`(+test). TOUCH: `internal/auth/credentials.go`(+test),
`internal/server/server.go`(+test), `internal/server/routes_openai.go`(+test),
`internal/api/chat.go`(+test). NO other plan runs concurrently (w5-pre is ALONE).

## Binary acceptance
- `go build ./... && go vet ./...` clean; `go test ./...` green; `go test -race ./internal/inference/ ./internal/api/ ./internal/server/ ./internal/auth/` green.
- `grep -c 'SetCredentialRefresher' internal/server/routes_openai.go` ≥ 1.
- `grep -c 'func (r \*CredentialResolver) RefreshCredentials' internal/auth/credentials.go` = 1.
- `grep -c 'ErrModelTransient' internal/inference/runner.go` ≥ 1.
- `grep -c 'ExecuteCombo' internal/api/chat.go` ≥ 1 (structural; the BINDING combo checks
  are the behavioral tests below, which drive the full HTTP path through the wired engine
  — dead wiring cannot pass them).
- TestRefreshCredentialsByConnectionID, TestRefreshCredentialsUnknownConnection,
  TestAccountRunnerWrapsTransient, TestAccountRunnerDelegatesToSelection,
  TestChatComboDispatchFallsBack, TestChatComboAllFailReturnsError,
  TestChatComboStreamFallsBackPreStream all pass.

## Out of scope
All PAR-USAGE rows (w5-a..f). Virtual keys (w5-g). request_log/usage glue (w5-f).
Messages/responses/embeddings refresher seams (not in the debt record). Mid-stream
combo fallback (ref has none). Per-model SelectionEngine routing for NON-combo single
models (Router.Resolve path unchanged). Live model pinging for GetTestByKind (W6).
