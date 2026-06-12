# w5-g fix micro-plan — diff-gate round 1 (Fable 5, 2026-06-12)

Source: `artifacts/w5-g-virtual-keys-diff-scoped-gpt.txt` (cycle 1, REJECT) + the
APIKey-attribution item TRANSFERRED from w5-f's cycle-3 disposition.

## REBUTTED — no change
- BLOCKER "handlers call AllowVK without nil guard; panics on x-g0-vk": DISPROVEN —
  `AllowVK` guards `if g == nil || g.resolver == nil` (`internal/api/vk.go:43-45`);
  a nil-receiver method call is legal Go and the guard returns allow. No panic path.
- BLOCKER/MAJOR "KeyIDs never used → not routing": PARTIAL DEFERRAL RECORDED —
  ProviderConfig.KeyIDs pins a VK to specific upstream connections; enforcing it
  requires connection-pinned dispatch through the selection engine
  (`SelectConnection(..., preferredConnID)` exists, but threading VK context into
  the dispatch path is the settings/catalog-driven work deferred with
  PAR-ROUTE-057/058 to W6). The Provider + AllowedModels constraints ARE enforced
  after Fix 2 below; quota + attribution bind the key. Recorded as the
  PAR-ROUTE-030 W6 note (same partial mechanism as PAR-USAGE-032).

## REAL → FIX

### Fix 1 (BLOCKER) — unknown virtual key bypasses enforcement
`vkResolverAdapter.ResolveVK` maps ErrNotFound to (nil, nil) and `AllowVK` returns
allow on nil vk (`vk.go:49-51`). A request ADDRESSING a key must be denied when the
key does not exist. FIX: AllowVK returns `(false, 401, "unknown virtual key")` when
the resolver returns nil for a NON-EMPTY key. Test FIRST:
`TestVKGateUnknownKeyDenied` (bogus header → 401, provider never called).

### Fix 2 (BLOCKER/MAJOR) — provider constraints flattened away
`storeVKToAPI` drops ProviderConfigs structure; only a flat model list survives.
FIX: `api.VKInfo` carries `Configs []VKProviderConfig{Provider string,
AllowedModels []string}`; extend the gate signature to
`AllowVK(key, model, providerID string)` (every handler already has the resolved
provider at the call site); allowed iff SOME config matches the resolved provider
AND (its AllowedModels is empty OR contains the model). Update the four handler
call sites + adapter. Tests: provider-mismatch denied; empty-AllowedModels config
allows any model of that provider.

### Fix 3 (TRANSFERRED from w5-f) — APIKey attribution
`UsageEntry.APIKey` exists but is never populated, so (a) byApiKey stats attribute
everything to local-no-key and (b) w5-g's OWN QuotaEngine.SumCostByAPIKey reads
ZERO production spend — budget enforcement is inert. FIX: when the x-g0-vk gate
admits a request, the handler passes the vk key into the record glue
(recordError/recordNonStream/recordStream gain an apiKey param or the glue gains a
WithAPIKey context); entry.APIKey = vk key. Test FIRST:
`TestVKSpendAttribution` — admitted vk request → request_log row api_key = vk key;
then a seeded-cost budget test proves SumCostByAPIKey sees it end-to-end.

### Fix 4 (MAJOR) — vk.go has no direct tests
ADD direct table-driven `TestVKGateAllow` in vk_test.go: absent header allow;
unknown key deny (Fix 1); inactive deny 403; resolver error deny 500; model/provider
mismatch deny 403 (Fix 2); quota deny 429; nil gate allow.

## Ownership
`internal/api/vk.go`(+test), `internal/api/{chat,messages,responses,embeddings}.go`
(+tests — gate call-site signature), `internal/api/usage_glue.go`(+test),
`internal/server/routes_openai.go` (adapter). ABSOLUTE PROHIBITION on
checkout/restore/stash of unowned paths; index.lock retry 5×10s.

## Binary acceptance
- `go build ./... && go vet ./... && go test ./...` green; `go test -race ./internal/api/ ./internal/governance/ ./internal/server/` green.
- TestVKGateUnknownKeyDenied, TestVKGateAllow, TestVKSpendAttribution pass.
- `grep -c 'Configs' internal/api/vk.go` ≥ 1; `grep -c 'APIKey' internal/api/usage_glue.go` ≥ 2 (field + population).
