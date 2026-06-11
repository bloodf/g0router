# w3-e — Outbound proxy support (SSRF row, Stage-1 half)

Rows: PAR-AUTH-020 PARTIAL — STAGE-1 HALF (outbound proxy; evidence `open-sse/utils/proxyFetch.js:314-334`, `src/lib/network/outboundProxy.js`). The row's OTHER half — MITM DNS bypass (`proxyFetch.js:317-334` `shouldBypassMitmDns`/`resolveRealIP`) — exists ONLY for MITM-intercepted hosts of the antigravity provider (Stage-2 per `matrix/9router-providers.md` ranking) and the Wave-7 MITM proxy platform feature; it is DEFERRED with them (recorded in WAVE-3-MAP §Deferred). PAR-AUTH-017 (header sanitization) and PAR-AUTH-018 (debug-log prod gate) are NOT here: their substrates (request_log persistence; a debug-log utility) do not exist until Wave 5 — both deferred there (WAVE-3-MAP §Deferred; verified: `internal/logging/` contains only doc.go; no request_log table exists).

Frozen ref @ 827e5c3: `proxyFetch.js:310-316` — per-connection proxy URL wins, else env proxy for the target URL (standard HTTP(S)_PROXY/NO_PROXY semantics), else direct; `strictProxy === true` → proxy failure is a hard error, no direct fallback (`:323-326`).

In-repo integration: `internal/providers/utils/client.go:8-16` (`ClientPool` wraps `*fasthttp.Client` — NO proxy support today, verified), `internal/auth/oauth.go:68-74` (`NewOAuthFlow` takes `*http.Client` — net/http honors `HTTP(S)_PROXY` via `http.ProxyFromEnvironment` when configured).

## Scope decision

Stage-1 ports ENV-proxy support only: the per-connection `proxyUrl` field arrives
with connection-level proxy config (no Stage-1 row provides it — the connections
table has no proxy column today). Env-proxy is the half with existing substrate:
every Wave-2 adapter dials upstream through `ClientPool`.

## Preconditions (a "0 hits" grep exits 1 = pass)

- `grep -c 'Proxy\|proxy' internal/providers/utils/client.go` → 0 hits (none today)
- `grep -c 'fasthttpproxy' go.mod go.sum` → 0 hits OR present (fasthttp's own proxy helper subpackage — preferred dependency, already within the fasthttp module)

## Exclusive file ownership

TOUCH: `internal/providers/utils/client.go` + a NEW `internal/providers/utils/proxy_test.go`;
`internal/auth/oauth.go` (ONLY the default-client construction line `oauth.go:70-72`:
use a Transport with `http.ProxyFromEnvironment`) + existing auth test file.
NOT touched: adapters (they receive proxy behavior transparently via ClientPool),
guard/login/OIDC/API-key files (other w3 plans).

## Tasks (each: STEP (a) named failing tests FIRST, run, show fail; STEP (b) implement)

1. **ClientPool env-proxy** (`client.go`). Tests FIRST (`proxy_test.go`):
   `TestClientPoolUsesEnvProxy` (with `HTTP_PROXY` set to an httptest proxy stub →
   an outbound request arrives AT THE PROXY, not the target; uses `t.Setenv`),
   `TestClientPoolNoProxyDirect` (unset → direct; `NO_PROXY` host → direct).
   STEP (b): in `NewClientPool`, when the standard env vars are set, configure the
   fasthttp client `Dial` via `fasthttpproxy.FasthttpHTTPDialer`-family helpers
   (the fasthttp module's own proxy subpackage), honoring NO_PROXY; else unchanged.
2. **OAuth client env-proxy** (`oauth.go:70-72`). Tests FIRST:
   `TestOAuthDefaultClientHonorsEnvProxy` (flow's token request reaches the proxy
   stub when HTTP_PROXY set). STEP (b): default client gains
   `Transport: &http.Transport{Proxy: http.ProxyFromEnvironment}`.

## Binary acceptance criteria

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -c 'ProxyFromEnvironment' internal/auth/oauth.go` ≥ 1.
- `grep -rn 'func init(\|panic(' internal/providers/utils/client.go` → 0 hits.
- `TestClientPoolUsesEnvProxy`, `TestClientPoolNoProxyDirect`, `TestOAuthDefaultClientHonorsEnvProxy` pass.
- No adapter file changed (`git diff --stat` shows only the owned files).

## Out of scope

MITM DNS bypass + `strictProxy` per-connection semantics (deferred with antigravity/
Wave-7 MITM platform — `strictProxy` is a per-connection field that does not exist in
Stage-1). PAR-AUTH-017/018 (Wave 5 — substrates land there). Per-connection proxy
columns (future wave with connection-proxy UI).
