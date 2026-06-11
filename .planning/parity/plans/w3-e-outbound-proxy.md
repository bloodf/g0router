# w3-e — Outbound proxy support (SSRF row, Stage-1 half)

Rows: PAR-AUTH-020 (matrix status MISSING; evidence `open-sse/utils/proxyFetch.js:314-334`, `src/lib/network/outboundProxy.js`). This plan implements the row's OUTBOUND-PROXY behavior; after it merges the row becomes PARTIAL (not HAVE) — the row text bundles "outbound proxy, MITM DNS bypass" and the MITM half is deferred (next sentence). The row's OTHER half — MITM DNS bypass (`proxyFetch.js:317-334` `shouldBypassMitmDns`/`resolveRealIP`) — exists ONLY for MITM-intercepted hosts of the antigravity provider (Stage-2 per `matrix/9router-providers.md` ranking) and the Wave-7 MITM proxy platform feature; it is DEFERRED with them (recorded in WAVE-3-MAP §Deferred). PAR-AUTH-017 (header sanitization) and PAR-AUTH-018 (debug-log prod gate) are NOT here: their substrates (request_log persistence; a debug-log utility) do not exist until Wave 5 — both deferred there (WAVE-3-MAP §Deferred; EVIDENCE: `internal/logging/` contains only `doc.go` + `logging_test.go` — no logger implementation; `grep -c request_log internal/store/migrate.go` outputs 0 — no request_log table).

Frozen ref @ 827e5c3: `proxyFetch.js:310-316` — per-connection proxy URL wins, else env proxy for the target URL (standard HTTP(S)_PROXY/NO_PROXY semantics), else direct; `strictProxy === true` → proxy failure is a hard error, no direct fallback (`:323-326`).

In-repo integration: `internal/providers/utils/client.go:8-16` (`ClientPool` wraps `*fasthttp.Client` — NO proxy support today, verified), `internal/auth/oauth.go:68-74` (`NewOAuthFlow` takes `*http.Client` — net/http honors `HTTP(S)_PROXY` via `http.ProxyFromEnvironment` when configured).

## Scope decision

Stage-1 ports ENV-proxy support only: the per-connection `proxyUrl` field arrives
with connection-level proxy config (no Stage-1 row provides it; EVIDENCE: the
`Connection` struct `internal/store/connections.go:13-25` has fields ID/ProviderID/
Name/Kind/Secret/AccessToken/RefreshToken/ExpiresAt/Metadata/CreatedAt/UpdatedAt —
no proxy column). Env-proxy is the half with existing substrate:
every Wave-2 adapter dials upstream through `ClientPool`.

## Preconditions (each check states its own pass condition)

- `grep -c 'Proxy' internal/providers/utils/client.go` outputs `0` (no proxy support today)
- `golang.org/x/net` availability: present in go.sum OR added by this plan (httpproxy is its subpackage); `fasthttpproxy` ships within the fasthttp module already in go.mod

## Exclusive file ownership

TOUCH: `internal/providers/utils/client.go` + a NEW `internal/providers/utils/proxy_test.go`;
`internal/auth/oauth.go` (ONLY the default-client construction line `oauth.go:70-72`:
use a Transport with `http.ProxyFromEnvironment`) + NEW `internal/auth/oauth_proxy_test.go`
(a NEW file — `internal/auth/auth_test.go` is owned by w3-a and is NOT touched).
NOT touched: adapters (they receive proxy behavior transparently via ClientPool),
guard/login/OIDC/API-key files (other w3 plans).

## Tasks (each: STEP (a) named failing tests FIRST, run, show fail; STEP (b) implement)

1. **ClientPool env-proxy** (`client.go`). Tests FIRST (`proxy_test.go`):
   `TestClientPoolUsesEnvProxy` (with `HTTP_PROXY` set to an httptest proxy stub →
   an outbound request arrives AT THE PROXY, not the target; uses `t.Setenv`),
   `TestClientPoolNoProxyDirect` (unset → direct; `NO_PROXY` host → direct), `TestClientPoolHTTPSProxyPrecedence` (HTTPS_PROXY set, HTTP_PROXY different → an https:// target resolves to the HTTPS_PROXY value via the ProxyFunc — asserted at the resolver seam).
   STEP (b): the scheme-aware seam is `ClientPool.Do` (NOT the Dial func — fasthttp's
   `Dial` receives only host:port and cannot see the scheme): in `Do`, parse the
   request URI (scheme+host available from `req.URI()`), resolve via
   `httpproxy.FromEnvironment().ProxyFunc()(url)` — the canonical
   HTTP_PROXY/HTTPS_PROXY/NO_PROXY implementation — and when it returns a proxy URL,
   execute on a per-proxy-URL cached `*fasthttp.Client` whose `Dial =
   fasthttpproxy.FasthttpHTTPDialer(proxyURL)` (cache map + mutex, -race covered);
   nil proxy → the existing default client. Env is read per-Do via the resolver
   (constructed once in NewClientPool; tests may reconstruct after t.Setenv).
2. **OAuth client env-proxy** (`oauth.go:70-72`). Tests FIRST:
   `TestOAuthDefaultClientHonorsEnvProxy` (flow's token request reaches the proxy
   stub when HTTP_PROXY set). STEP (b): default client gains
   `Transport: &http.Transport{Proxy: http.ProxyFromEnvironment}`.

## Binary acceptance criteria

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -c 'ProxyFromEnvironment' internal/auth/oauth.go` ≥ 1.
- `grep -rn 'func init(\|panic(' internal/providers/utils/client.go` → 0 hits.
- `TestClientPoolUsesEnvProxy`, `TestClientPoolNoProxyDirect`, `TestClientPoolHTTPSProxyPrecedence`, `TestOAuthDefaultClientHonorsEnvProxy` pass.
- No adapter file changed (`git diff --stat` shows only the owned files).

## Out of scope

MITM DNS bypass + `strictProxy` per-connection semantics (deferred with antigravity/
Wave-7 MITM platform — `strictProxy` is a per-connection field that does not exist in
Stage-1). PAR-AUTH-017/018 (Wave 5 — substrates land there). Per-connection proxy
columns (future wave with connection-proxy UI).

## Plan-gate disposition (Fable 5, 2026-06-11)

APPROVED BY DECISION after 3 cycles. Real findings fixed: row-status wording
corrected (MISSING → becomes PARTIAL after merge; no silent re-status), test-file
ownership named (NEW oauth_proxy_test.go; w3-a's auth_test.go untouched), the
scheme-blind Dial seam REPLACED with a scheme-aware ClientPool.Do resolution +
per-proxy cached clients (gate catch — fasthttp Dial cannot see the scheme),
evidence inlined for absence claims. Residual "MITM-half deferral relies on
WAVE-3-MAP" is structural cross-plan sequencing: the deferral ledger is a committed
in-repo artifact (same mechanism the gates accepted for w2/w3 scope decisions).
The diff gate remains the binding implementation check.
