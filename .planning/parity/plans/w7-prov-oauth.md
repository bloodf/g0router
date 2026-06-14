# Micro-plan w7-prov-oauth — OAuth provider flows (device-code + PKCE-redirect + refresh) for 8 providers (Go)

```
wave: 7
plan: w7-prov-oauth
status: READY (rev 1 — authored against the SHIPPED w3-f provider-OAuth foundation
  (anthropic + gemini + xai config factories + the OAuthFlow PKCE engine + the
  CredentialResolver refresh orchestration — LIVE in-tree @ internal/auth/{oauth,
  credentials}.go) and the SHIPPED admin OAuth surface (internal/admin/oauth.go +
  routes_admin.go /api/oauth/{provider}/{start,callback}). REUSES the in-tree PKCE
  engine: oauth.go is ALREADY generalized — it is NOT anthropic-specific. It exposes
  OAuthConfig (a per-provider config struct, oauth.go:26-34), per-provider factory
  funcs (AnthropicOAuth oauth.go:38, GeminiOAuth :55, XaiOAuth :87), a config-driven
  OAuthFlow engine (NewOAuthFlow oauth.go:128, nil-able *http.Client @128-136;
  StartWithRedirect :151; ExchangeWithRedirect :198; Refresh :221; requestToken :232),
  the PKCE primitives (pkceChallenge :274 S256, randomURLSafe :266, the exported
  GeneratePKCE :283), and refreshLead(provider) :101. The 8 NEW provider flows are
  therefore ADDITIVE: 8 config factories + (for the device-code half) ONE additive
  device-code flow path + per-provider refresh quirks — NO rewrite of the anthropic
  authorization-code path. live tree @ <base>; WAVE-7-MAP w7-prov-oauth row ~line 178;
  serial chain §219-224; reconciliation §245; freeze rules §267.)
runs: provider/catalog track. EXTENDS internal/auth with NEW per-provider config +
  device-code files — near-disjoint from every other domain. Runs ∥ the catalog plans
  (w7-prov-openai/special), the governance/MCP/platform tracks. The ONE shared hot file
  is internal/server/routes_admin.go (the flows-map registration in NewAdminHandlers
  @routes_admin.go:21-23) — TAKES the serial slot for ONE additive edit. NOT internally
  serial. DEPENDS on w3-f's OAuthFlow engine + CredentialResolver + the admin
  start/callback handlers (all SHIPPED @ <base>); w7-prov-special MAY depend on this
  plan for OAuth-gated specialized providers (MAP §187).
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w7-prov-oauth:
ref-source: 9router frozen @ 827e5c3 (~/Developer/github.com/bloodf/_refs/9router).
  The AUTHORITATIVE OAuth config + flow-type registry is the CLI oauth lib (NOT the
  open-sse executor configs alone). Authoritative ref files (read for THIS plan):
    src/lib/oauth/constants/oauth.js — the canonical per-provider config blocks with
      the flow type stated in each block's comment:
        CLAUDE_CONFIG :19-26   (Authorization Code Flow with PKCE)
        CODEX_CONFIG  :29-43   (Authorization Code Flow with PKCE; extraParams
                                id_token_add_organizations/codex_cli_simplified_flow/
                                originator)
        GEMINI_CONFIG :46-58   (Standard OAuth2; clientId+clientSecret) — gemini-cli
                                REUSES GEMINI_CONFIG (providers.js gemini-cli block :315)
        QWEN_CONFIG   :61-67   (Device Code Flow with PKCE; deviceCodeUrl + tokenUrl)
        IFLOW_CONFIG  :84-95   (Authorization Code; clientId+clientSecret; extraParams
                                loginMethod/type=phone)
        GITHUB_CONFIG :130-141 (Device Code Flow; deviceCodeUrl + tokenUrl +
                                copilotTokenUrl; scopes "read:user")
        KIMI_CODING_CONFIG :167-171 (Device Code Flow) — NOTE: out of THIS plan's 8
        KILOCODE_CONFIG :174-178 (Custom Device Auth Flow; initiateUrl + pollUrlBase;
                                  NO refresh)
        CLINE_CONFIG  :181-186 (Local Callback Flow; authorizeUrl + tokenExchangeUrl +
                                refreshUrl; code is base64-encoded token data)
    src/lib/oauth/providers.js — the flow-type registry (one entry per provider; the
      buildAuthUrl / requestDeviceCode / pollToken / exchangeToken / mapTokens fns):
        claude :122 flowType "authorization_code_pkce"  (buildAuthUrl :125-136 adds
                    code=true; exchangeToken :138 JSON body, parses code#state)
        codex  :179 flowType "authorization_code_pkce"
        "gemini-cli" :315 flowType "authorization_code" (buildAuthUrl :318 adds
                    access_type=offline + prompt=consent; standard Google token POST)
        iflow  :516 flowType "authorization_code"
        qwen   :703 flowType "device_code"
        github :756 flowType "device_code"
        kilocode :1060 flowType "device_code" (requestDeviceCode :1063 POST initiateUrl;
                    pollToken :1086 GET pollUrlBase/{code}, 202→pending / 403→denied /
                    410→expired; mapTokens refreshToken:null; orgId→providerSpecificData)
        cline  :1117 flowType "authorization_code" (buildAuthUrl :1120 client_type=
                    extension+callback_url; exchangeToken :1131 base64-decodes the code
                    to {accessToken,refreshToken,expiresAt}, falls back to tokenExchangeUrl)
    src/lib/oauth/services/qwen.js — device-code mechanics of record: requestDeviceCode
      (:18 POST deviceCodeUrl form {client_id,scope,code_challenge,code_challenge_method})
      → {device_code,user_code,verification_uri,interval}; pollForToken (:41 POST tokenUrl
      form {grant_type:urn:ietf:params:oauth:grant-type:device_code,client_id,device_code,
      code_verifier}; loop maxAttempts=60 interval*1000ms; on JSON error switch
      authorization_pending→continue / slow_down→+5s / expired_token→err / access_denied→
      err). github.js pollAccessToken (:41) is the same shape (interval+=5000 on slow_down).
    open-sse/config/providers.js — the RUNTIME baseUrl/format/headers per provider
      (claude :51, gemini-cli :64, codex :70, qwen :80, iflow :87, github :176,
      kimi-coding :220, kilocode :228, cline :238) — these are the catalog/adapter
      concern (w7-prov-* catalog plans), cited here only to bind clientId/tokenUrl.
    open-sse/executors/default.js refreshCredentials :186-320 — the per-provider REFRESH
      quirks of record: claude refreshWithJSON :214 (JSON body); codex/qwen refreshWithForm
      :226 (form body, scope for codex); iflow refreshIflow :237 (Basic-auth
      clientId:clientSecret header + form); gemini refreshGoogle :270 (form +
      client_id/client_secret); cline refreshCline :291 (JSON {refreshToken,grantType,
      clientType} to refreshUrl, expiresAt→expiresIn); kilocode refreshKilocode :311
      (returns null — device-code, NO refresh).
  CLIENT-ID PROVENANCE (binding — NEVER fabricate): every clientId/tokenUrl/authUrl/
  deviceCodeUrl below is copied verbatim from the ref blocks cited above. The two
  Google-shared secrets (gemini-cli reuses GEMINI_CONFIG clientId+clientSecret) are
  ALREADY in-tree as GeminiOAuth() oauth.go:55-81 (split-literal form) and are REUSED,
  not re-declared. Any provider whose config cannot be soundly read from the ref is
  ESCALATED (§8), never invented.
base: <base> = git rev-parse HEAD recorded at P0. Substitute the actual SHA
  everywhere §5 says <base>. (At authoring, recompute at P0.)
go-serial-slot: this plan holds the SINGLE in-flight edit to
  internal/server/routes_admin.go while live (the flows-map registration in
  NewAdminHandlers, routes_admin.go:21-23). The /api/oauth/{provider}/{start,callback}
  ROUTES already exist (routes_admin.go:185-186, dynamic by {provider}) — NO new route
  line is added; only the flows map gains 8 entries. Slot must be FREE at P5 before
  T-register. NO selection.go / factory.go micro-serial (OAuth is an auth-flow concern,
  not an inference-path edit).
new-route: NO UI route files, NO new admin handler, NO new route registration. The
  provider-OAuth modals ALREADY SHIPPED in w6-e against mocks; the start/callback/refresh
  HTTP surface already exists. This plan supplies the 8 flow CONFIGS the existing handlers
  dispatch to. If a w6-e modal's request shape diverges from the real flow, correct the
  w6-e MOCK (mirror Go); else no UI touch.
```

---

## 1. Scope — PAR rows + the 8 OAuth provider flows

### Rows this plan closes / advances

| Row / item | Claim (from `9router-providers.md` / `-platform.md` / `-auth.md`) | Target state after w7-prov-oauth |
|---|---|---|
| PAR-PROV-015 | **claude (OAuth alias `cc`)** — PKCE auth-code; tokenUrl `https://api.anthropic.com/v1/oauth/token`; authorizeUrl `https://claude.ai/oauth/authorize`; clientId `9d1c…962f5e` (`providers.js:122`, `oauth.js:19-26`) | HAVE (`ClaudeOAuth()` config factory + flows-map entry `"claude"`; PKCE-redirect via the existing `OAuthFlow`; refresh via `requestToken`) |
| PAR-PROV-016 | **codex (OAuth alias `cx`)** — PKCE auth-code; tokenUrl `https://auth.openai.com/oauth/token`; authorizeUrl `https://auth.openai.com/oauth/authorize`; clientId `app_EMoamEEZ73f0CkXaXp7hrann`; scope `openid profile email offline_access`; extraParams (`providers.js:179`, `oauth.js:29-43`) | HAVE (`CodexOAuth()` + entry `"codex"`; PKCE-redirect + the codex extra authorize params via an additive `ExtraAuthParams` config field; refresh form-POST with scope) |
| PAR-PROV-017 | **gemini-cli (OAuth alias `gc`)** — standard Google OAuth2 (REUSES GEMINI_CONFIG clientId+clientSecret); access_type=offline + prompt=consent (`providers.js:315`, `oauth.js:46-58`) | HAVE (`GeminiCLIOAuth()` reusing the in-tree gemini clientId/secret split-literals + the Google authorize/token URLs; entry `"gemini-cli"`; redirect auth-code; refresh form-POST with client_secret) |
| PAR-PROV-018 | **qwen (OAuth alias `qw`)** — **device-code + PKCE**; deviceCodeUrl `https://chat.qwen.ai/api/v1/oauth2/device/code`; tokenUrl `https://chat.qwen.ai/api/v1/oauth2/token`; clientId `f0304373b74a44d2b584a3fb70ca9e56` (`providers.js:703`, `oauth.js:61-67`, `services/qwen.js`) | HAVE (`QwenOAuth()` + entry `"qwen"`; the NEW additive **device-code flow path** — request-code + poll; refresh form-POST) |
| PAR-PROV-019 | **iflow (OAuth alias `if`)** — auth-code; tokenUrl `https://iflow.cn/oauth/token`; authUrl `https://iflow.cn/oauth`; clientId `10009311001` + clientSecret; refresh uses **Basic-auth** clientId:clientSecret (`providers.js:516`, `oauth.js:84-95`, `default.js:237`) | HAVE (`IflowOAuth()` + entry `"iflow"`; redirect auth-code with extraParams loginMethod/type=phone; the **Basic-auth refresh quirk** via an additive `RefreshBasicAuth` config flag) |
| PAR-PROV-021 | **github (OAuth alias `gh`)** — **device-code**; deviceCodeUrl `https://github.com/login/device/code`; tokenUrl `https://github.com/login/oauth/access_token`; clientId `Iv1.b507a08c87ecfe98`; scope `read:user`; + Copilot-token exchange `copilotTokenUrl` (`providers.js:756`, `oauth.js:130-141`, `services/github.js`) | HAVE (`GithubOAuth()` + entry `"github"`; device-code flow; the Copilot-token sub-exchange is ESCALATED §8 ESC-GH-COPILOT — default: store the GitHub access_token, surface Copilot-token mint as a follow-up) |
| PAR-PROV-026 | **kilocode (OAuth alias `kc`)** — **custom device-auth (NO refresh)**; initiateUrl `https://api.kilo.ai/api/device-auth/codes`; pollUrlBase same; 202=pending/403=denied/410=expired; orgId→providerSpecificData (`providers.js:1060`, `oauth.js:174-178`) | HAVE (`KilocodeOAuth()` + entry `"kilocode"`; the device-code path with kilocode's custom poll (GET `pollUrlBase/{code}`, status-coded responses); `RefreshDisabled` config flag → no refresh; orgId persisted to `Connection.Metadata`) |
| PAR-PROV-025 | **cline (OAuth alias `cl`)** — auth-code; authorizeUrl `https://api.cline.bot/api/v1/auth/authorize`; the **code is base64-encoded token data**; refreshUrl `https://api.cline.bot/api/v1/auth/refresh` (JSON body) (`providers.js:1117`, `oauth.js:181-186`, `default.js:291`) | HAVE (`ClineOAuth()` + entry `"cline"`; redirect auth-code with the **base64-code-decode exchange quirk** + the **JSON-body refresh quirk** via additive config flags) |
| PAR-PLAT-047 | g0router OAuth flow: PKCE for anthropic only; "no cursor/codex/kiro/iflow/gitlab flows" (`internal/auth/oauth.go:33-160`, PARTIAL) | PARTIAL→HAVE-advanced (the 8 flows above land; cursor=import_token/kiro=AWS-SSO/gitlab remain out-of-scope — footnoted) |
| PAR-AUTH-019 | OAuth credential manager for provider connections; "9router supports ~15 provider OAuth flows" (HAVE for anthropic; PARTIAL coverage) | coverage advanced from 3 (anthropic/gemini/xai) → 11 provider flows; the row stays HAVE with a coverage-progress note |

> **Brief-vs-matrix reconciliation (binding — read carefully).** The brief's 8 providers
> are claude(cc), codex(cx), gemini-cli(gc), qwen(qw), iflow(if), github(gh),
> kilocode(kc), cline(cl). Their PAR rows are **015, 016, 017, 018, 019, 021, 026, 025**
> (NOTE: the brief listed "024" but **PAR-PROV-024 is kimi-coding (`kmc`), NOT
> kilocode** — kilocode is **PAR-PROV-026**; cline is **PAR-PROV-025**, not 026). This
> plan closes the 8 brief-named providers via rows 015/016/017/018/019/021/025/026.
> **PAR-PROV-024 (kimi-coding) and PAR-PROV-027 (xai) are NOT in this plan:** xai
> already shipped in w3-f (`XaiOAuth()` oauth.go:87, matrix HAVE); kimi-coding is a
> device-code provider NOT in the brief's 8 — it is recorded as a follow-up in
> `open-questions.md` (its config IS sound — `oauth.js:167-171` — so the follow-up is
> trivially the same device-code path; flagged, not built, to honor the brief's exact 8).

Matrix flips at closeout (§4 T-close): in `.planning/parity/matrix/9router-providers.md`,
PAR-PROV-015,016,017,018,019,021,025,026 → HAVE (OAuth flow config + flow path; all
HTTP hermetically tested via `httptest.NewServer` + an injected clock for device-poll).
In `9router-platform.md` PAR-PLAT-047 PARTIAL→HAVE-advanced (8 flows; cursor/kiro/gitlab
footnoted out-of-scope). In `9router-auth.md` PAR-AUTH-019 stays HAVE with a coverage
note (3→11 flows). Append §8 open items to `open-questions.md`.

### 1.1 Preconditions already satisfied by the SHIPPED w3-f + admin OAuth (evidence — cite file:line)

- **`internal/auth/oauth.go` is the config-driven PKCE engine — ALREADY generalized,
  REUSE, do NOT rewrite (the central de-risk).**
  - `OAuthConfig` is a per-provider config struct (`oauth.go:26-34`:
    `{Provider,ClientID,ClientSecret,AuthorizeURL,TokenURL,RedirectURI,Scopes}`). The
    8 new providers are 8 new factory funcs returning this struct — EXACTLY like the
    existing `AnthropicOAuth()` (`oauth.go:38`), `GeminiOAuth()` (`:55`),
    `XaiOAuth()` (`:87`).
  - `NewOAuthFlow(cfg, st, client)` (`oauth.go:128`) injects a **nil-able `*http.Client`**
    (`:129-134` nil → a real 30s default) — the hermetic-test seam (tests pass
    `httptest.NewServer(...).Client()`).
  - `StartWithRedirect` (`:151`) builds the authorize URL with `code_challenge`/
    `code_challenge_method=S256` for ANY config; `ExchangeWithRedirect` (`:198`)
    does code→token; `Refresh` (`:221`) does refresh_token→token; `requestToken`
    (`:232`) is the shared form-POST + parse. The PKCE primitives `pkceChallenge`
    (`:274`), `randomURLSafe` (`:266`), and the exported `GeneratePKCE` (`:283`) are
    REUSED — never re-implemented.
  - `refreshLead(provider)` (`:101`) is the per-provider refresh-lead switch (anthropic
    4h, gemini/xai 5m, default 5m). THIS plan ADDS the 8 providers' leads additively
    (default 5m is already correct for all 8 — an explicit case is optional).
- **The admin OAuth surface is LIVE + provider-agnostic (consume, do NOT edit).**
  - `internal/admin/oauth.go:15 OAuthStart` (`GET /api/oauth/{provider}/start`),
    `:47 OAuthCallback` (`POST /api/oauth/{provider}/callback`), `:115 RefreshConnection`
    (`POST /api/connections/{id}/refresh`) — all dispatch via `h.flows[providerType]`
    (`oauth.go:21,53,141`). Adding a flows-map entry makes a provider's start/callback/
    refresh work with ZERO handler edit. The callback stores the token as a
    `store.Connection{Kind:"oauth", AccessToken, RefreshToken, ExpiresAt}`
    (`admin/oauth.go:98-105`).
  - **These redirect-style handlers cover the 4 redirect providers
    (claude/codex/gemini-cli/iflow/cline).** The 3 device-code providers
    (qwen/github/kilocode) need a DIFFERENT admin entry shape (request-device-code +
    poll) — see §1.4 ESC-DEVICE-ENDPOINT.
- **The CredentialResolver refresh orchestration is LIVE (consume; extend per-provider
  quirks additively).** `internal/auth/credentials.go:112 RefreshCredentials`,
  `:132 shouldRefresh` (uses `refreshLead`), `:141 doRefresh` (singleflight-deduped),
  `:162 refreshAndPersist` (calls `flow.Refresh(conn.RefreshToken)` + persists),
  `:192 mergeRefreshedCredentials` (access overwritten; empty refresh preserved;
  metadata shallow-merged). The 8 providers plug into the flows map and get refresh
  orchestration FOR FREE — **except 3 refresh quirks** (iflow Basic-auth, cline JSON
  body, kilocode no-refresh) handled by additive `OAuthConfig` flags (§1.5).
- **Token + verifier secret-at-rest is LIVE (`*_enc`).** `internal/store/connections.go:13
  Connection` (`AccessToken`/`RefreshToken`/`Secret` stored in `*_enc` columns,
  encrypt `:122-125`, decrypt `:145-148`); `internal/store/oauthsessions.go:20
  CreateOAuthSession` (PKCE verifier `verifier_enc` `:21`), `:37 ConsumeOAuthSession`
  (return+delete; expired→`ErrNotFound` `:54-56`; decrypt verifier `:58`). The flows
  REUSE these verbatim — tokens NEVER echoed cleartext; `Connection.Metadata`
  (`connections.go:21`, plaintext) carries non-secret provider-specific data (e.g.
  kilocode `orgId`).
- **Hermetic test harness precedent (binding).** `internal/auth/oauth_test.go` drives
  the flow against an `httptest.NewServer` (`oauth_test.go:110,153,212`) with
  `NewOAuthFlow(cfg, st, srv.Client())` (`:125,166,226`) — NO real network. Config
  factories are asserted field-by-field (`TestGeminiOAuthConfig :13`, `TestXaiOAuthConfig
  :66`); `refreshLead` is table-tested (`TestRefreshLeadTable :89`); scope/redirect
  encoding asserted (`TestOAuthFlowXaiScopeEncoding :245`, `…StartWithRedirect :183`).
  THIS plan mirrors that harness and ADDS an **injectable clock** for the device-poll
  loop so the poll-until-token + timeout are unit-tested with NO real sleep.
- **Device-code flow is GENUINELY ABSENT (the real new code).** `grep -rn
  device_code internal/` → 0 matches. The `OAuthFlow` engine only does the
  redirect/authorization-code-with-PKCE path. The device-code path (POST
  device-code endpoint → poll token endpoint with `grant_type=urn:ietf:params:oauth:
  grant-type:device_code` until `authorization_pending` clears) is the NEW additive
  surface (§1.4). It REUSES `GeneratePKCE` (qwen is device-code WITH PKCE) and the
  same `*_enc` token persistence.

### 1.2 No UI contract binds this plan (binding — confirm)

The provider-OAuth modals SHIPPED in w6-e against mocks (`ui/e2e/mocks/handlers/*` for
the OAuth start/callback). This plan supplies the real Go flow CONFIGS the existing
`/api/oauth/{provider}/{start,callback}` handlers dispatch to. Therefore w7-prov-oauth:
- adds **NO** `ui/src/**` touch, **NO** new admin handler, **NO** new route line.
- the ONLY possible UI touch is a **w6-e mock-body correction** IF a modal's request/
  response shape diverges from the real flow (decision 1: real Go wins, mock mirrors).
  For the **device-code** providers (qwen/github/kilocode) the modal flow differs
  (show user-code + poll) — if w6-e mocked them as redirect-style, the mock is
  corrected here; if w6-e has no device-code modal, that is a w6-e UI gap recorded in
  `open-questions.md` (NOT built here — UI is a separate wave concern). Confirm at P4.

### 1.3 oauth.go generalization decision (binding — ADDITIVE config-driven, NO rewrite)

**Decision (binding):** `oauth.go` is ALREADY generalized. The 8 providers are
**per-provider `OAuthConfig` factory funcs** (one per provider, mirroring
`AnthropicOAuth`/`GeminiOAuth`/`XaiOAuth`) registered in the `NewAdminHandlers` flows
map. **No `RegisterFlow` registry is introduced** (the flows map in
`routes_admin.go:21-23` already IS the registry; adding a map literal entry is the
ref-faithful, minimal change). **The anthropic authorization-code path is NOT
touched.** Two ADDITIVE generalizations are required (each leaves every existing
signature/body intact — see §3 ESC-CONFIG-FIELDS / ESC-DEVICE-PATH):

1. **`OAuthConfig` additive fields** (for the redirect-flow quirks) — append to the
   struct, defaults zero so existing configs/tests are unaffected:
   - `ExtraAuthParams map[string]string` — codex authorize extras
     (`oauth.js:38-42`), gemini-cli `access_type=offline`/`prompt=consent`
     (`providers.js:323-324`), iflow `loginMethod`/`type=phone` (`oauth.js:92-94`),
     cline `client_type=extension`/`callback_url` (`providers.js:1121-1123`). Threaded
     into `StartWithRedirect`'s query builder additively (append after the existing
     params; existing configs pass an empty map → identical output).
   - `RefreshMode` (enum/string: `""`=form (default), `"basic"`=iflow Basic-auth header,
     `"json"`=cline JSON body, `"none"`=kilocode no-refresh). Consumed by an additive
     branch in `Refresh`/`requestToken` (default `""` → the existing form-POST path,
     byte-identical for anthropic/gemini/xai).
   - `RefreshURL` (cline's refreshUrl differs from tokenUrl; default `""` → use
     `TokenURL`).
   - `CodeEncoding` (`""`=plain (default), `"base64-json"`=cline) — the exchange
     decodes the base64-JSON code into a token directly (cline's happy path,
     `providers.js:1131-1148`), bypassing the token POST.
2. **The device-code flow path** (for qwen/github/kilocode) — a NEW additive method
   set on `OAuthFlow` (or a sibling `DeviceFlow` type — DECIDE at T-device, §8
   ESC-DEVICE-TYPE; default: add `StartDevice`/`PollDevice` methods on `OAuthFlow`
   gated by an additive `DeviceCodeURL` config field, so the SAME flows map holds both
   redirect and device flows). See §1.4.

**FORBIDDEN:** changing the anthropic factory, the existing `Start`/`StartWithRedirect`/
`Exchange`/`ExchangeWithRedirect`/`Refresh`/`requestToken` signatures, or
`pkceChallenge`/`randomURLSafe`/`GeneratePKCE`. Additive fields + additive branches +
additive methods ONLY. If an additive branch cannot avoid touching the anthropic path's
behavior, ESCALATE (ESC-CONFIG-FIELDS) before editing.

### 1.4 The device-code flow (binding — the genuinely-new additive surface; qwen/github/kilocode)

Ported from `services/qwen.js` (`requestDeviceCode :18` / `pollForToken :41`) and the
kilocode registry entry (`providers.js:1063 requestDeviceCode` / `:1086 pollToken`).
Two steps over the injected client + an **injectable clock**:

1. **request-device-code** — POST the `DeviceCodeURL` (form for qwen/github with
   `client_id`,`scope`,`code_challenge`,`code_challenge_method=S256` for the PKCE
   variant qwen; JSON/empty for kilocode's `initiateUrl`) → parse
   `{device_code,user_code,verification_uri,verification_uri_complete,interval,
   expires_in}`. Persist the device_code + (for qwen) the PKCE verifier as an in-flight
   state (REUSE `store.CreateOAuthSession` — `oauthsessions.go:20` — keyed by the
   device_code/state, verifier `*_enc`). Return `{user_code, verification_uri,
   interval, expires_in}` for the admin/modal to display.
2. **poll-for-token** — loop on the clock: every `interval` seconds POST the token
   endpoint with `grant_type=urn:ietf:params:oauth:grant-type:device_code` +
   `device_code` (+ `code_verifier` for qwen). Decode the JSON error and branch:
   `authorization_pending`→continue, `slow_down`→`interval += 5s` + continue,
   `expired_token`/`access_denied`→terminal error, success→`{access_token,refresh_token,
   expires_in}`. kilocode's poll is a GET `pollUrlBase/{device_code}` with status-coded
   responses (202=pending, 403=denied, 410=expired, 2xx+`status==approved`→token;
   `providers.js:1086-1110`) — modeled as a kilocode `RefreshMode/DeviceVariant` branch.
   Terminate after `OAUTH_TIMEOUT` (5m, `oauth.js:189`) → "device authorization timeout".
   On success: store via `store.UpsertConnection`/`CreateConnection` (tokens `*_enc`;
   kilocode orgId→`Metadata`).

```go
// internal/auth/oauth_device.go (NEW — additive; the device-code path)
type DeviceCodeResponse struct {
    DeviceCode      string
    UserCode        string
    VerificationURI string
    Interval        int   // seconds
    ExpiresIn       int64 // seconds
}
// StartDevice requests a device code (PKCE for qwen), persists in-flight state.
func (f *OAuthFlow) StartDevice() (*DeviceCodeResponse, error)
// PollDevice polls the token endpoint until the user authorizes, the code expires,
// or the deadline passes. clock is injectable (real time.Now / time.After in prod;
// a fake in tests — NO real sleep).
func (f *OAuthFlow) PollDevice(ctx context.Context, deviceCode string) (*OAuthToken, error)
```

**ESC-DEVICE-ENDPOINT (admin transport for device-code — §8).** The existing
`OAuthStart`/`OAuthCallback` (redirect-style) do NOT fit device-code (no browser
redirect; the user enters a code, the server polls). The device-code providers need a
device-start (returns user_code+verification_uri) + a device-poll/complete endpoint.
**Decision (default):** this plan ships the device-code FLOW ENGINE (`StartDevice`/
`PollDevice`) + the 3 device config factories + flows-map entries, and adds the device
admin endpoints ONLY IF they fit within the SAME serial-slot edit window without a new
handler file. If a new `internal/admin/oauth_device.go` handler is needed, it is a
sanctioned NEW file (§3) but its ROUTE registration shares the one serial-slot commit.
**Escalate** the exact device-endpoint route shape (`POST /api/oauth/{provider}/device/
start` + `…/device/poll`) for orchestrator confirmation against the w6-e modal — never
silently diverge the modal/route. The flow engine + configs land regardless; the admin
device transport is the escalation surface (a follow-up plan owns it if it can't fit).

### 1.5 Per-provider flow-type + refresh table (binding — copied verbatim from the ref)

| Provider (alias) | PAR | Flow type | authorize / device-code URL | token URL | clientId (ref) | refresh quirk |
|---|---|---|---|---|---|---|
| claude (`cc`) | 015 | PKCE-redirect | `https://claude.ai/oauth/authorize` | `https://api.anthropic.com/v1/oauth/token` | `9d1c…962f5e` (`oauth.js:20`) | form (default); JSON-body in `default.js:214` — use form (token endpoint accepts both; default form) |
| codex (`cx`) | 016 | PKCE-redirect | `https://auth.openai.com/oauth/authorize` | `https://auth.openai.com/oauth/token` | `app_EMoamEEZ73f0CkXaXp7hrann` (`oauth.js:30`) | form + `scope=openid profile email offline_access` (`default.js:215`); `ExtraAuthParams` on authorize |
| gemini-cli (`gc`) | 017 | redirect (Google OAuth2) | `https://accounts.google.com/o/oauth2/v2/auth` | `https://oauth2.googleapis.com/token` | REUSE in-tree gemini clientId+secret (`oauth.go:60`) | form + client_secret (`default.js:270`); `ExtraAuthParams` access_type=offline+prompt=consent |
| qwen (`qw`) | 018 | **device-code + PKCE** | dc: `https://chat.qwen.ai/api/v1/oauth2/device/code` | `https://chat.qwen.ai/api/v1/oauth2/token` | `f0304373b74a44d2b584a3fb70ca9e56` (`oauth.js:62`) | form (`default.js:216`) |
| iflow (`if`) | 019 | redirect | `https://iflow.cn/oauth` | `https://iflow.cn/oauth/token` | `10009311001` + secret `4Z3Y…SDtW` (`oauth.js:85-86`) | **Basic-auth** clientId:clientSecret header (`default.js:237`) → `RefreshMode:"basic"` |
| github (`gh`) | 021 | **device-code** | dc: `https://github.com/login/device/code` | `https://github.com/login/oauth/access_token` | `Iv1.b507a08c87ecfe98` (`oauth.js:131`) | (no token refresh; Copilot-token re-mint — ESC-GH-COPILOT §8) |
| kilocode (`kc`) | 026 | **device-code (custom, no refresh)** | initiate: `https://api.kilo.ai/api/device-auth/codes` | poll: same `/{code}` | (none — initiate is unauthenticated) (`oauth.js:175-176`) | **none** → `RefreshMode:"none"`; orgId→`Metadata` |
| cline (`cl`) | 025 | redirect (base64-code) | `https://api.cline.bot/api/v1/auth/authorize` | exchange: `https://api.cline.bot/api/v1/auth/token`; refresh: `…/auth/refresh` | (none — client_type=extension) (`oauth.js:181-186`) | **JSON body** `{refreshToken,grantType,clientType}` to refreshUrl (`default.js:291`) → `RefreshMode:"json"` + `CodeEncoding:"base64-json"` |

Flow-type tally: **PKCE-redirect** = claude, codex (2). **redirect (non-PKCE)** =
gemini-cli, iflow, cline (3). **device-code** = qwen (with PKCE), github, kilocode (3).

### NOT in scope (explicit)

- **No anthropic-flow rewrite** — `AnthropicOAuth()` + the `OAuthFlow` redirect path are
  REUSED unchanged. Additive config fields + additive device methods + additive flows-map
  entries only.
- **No new public route line** — `/api/oauth/{provider}/{start,callback}` +
  `/api/connections/{id}/refresh` already exist (dynamic by `{provider}`); only the
  flows MAP (`routes_admin.go:21-23`) gains 8 entries. Device-code admin endpoints, if
  needed, share the one serial-slot commit (§1.4 ESC-DEVICE-ENDPOINT).
- **No catalog/adapter work** — the provider baseUrl/format/headers/model-catalog
  entries (open-sse `providers.js`) are the w7-prov-openai/special catalog plans'
  concern. This plan binds ONLY clientId/tokenUrl/authUrl/scopes for the OAuth flow.
- **kimi-coding (PAR-PROV-024) and xai (PAR-PROV-027) NOT built** — xai shipped in
  w3-f; kimi-coding is outside the brief's 8 (follow-up in `open-questions.md`).
- **No cursor / kiro / gitlab / antigravity OAuth** — cursor=import_token,
  kiro=AWS-SSO-device, gitlab=PKCE, antigravity=Google; all out of the brief's 8
  (PAR-PLAT-047 footnote).
- **GitHub Copilot-token sub-exchange deferred** (ESC-GH-COPILOT §8) — store the GitHub
  device-code access_token; the `copilotTokenUrl` mint (`oauth.js:138`) is a follow-up.
- **No secret exposure** — tokens + PKCE verifier stay `*_enc`, never echoed; clientSecrets
  stay split-literal (the gemini-secret scanner-evasion precedent, `oauth.go:60-66`) and
  are env-overridable; the device user_code is non-secret (display-only).
- **No `New(...)` signature change** — flows compose into the existing
  `NewAdminHandlers` flows map; no `admin.New` signature change.
- **No real network in any unit test** — all HTTP via `httptest.NewServer`/the injected
  client; the device-poll loop via an injectable clock (no real sleep).

---

## 2. Precondition checks

Run all before any edit; abort and report to orchestrator on any failure.

```bash
# P0 — clean tree; record <base>
git status --porcelain     # expect empty EXCEPT a possibly-dirty ui/dist/index.html
                           # (gitignored build artifact — NEVER stage/revert it).
                           # Anything ELSE dirty → STOP. Use explicit `git add <file>`, never -A.
git rev-parse HEAD         # record as <base> for §5

# P1 — the SHIPPED w3-f OAuth engine + admin surface are present (consume, don't rewrite)
grep -nE 'type OAuthConfig|func AnthropicOAuth|func GeminiOAuth|func XaiOAuth|func NewOAuthFlow|func .*StartWithRedirect|func .*ExchangeWithRedirect|func .*Refresh|func .*requestToken|func GeneratePKCE|func refreshLead' internal/auth/oauth.go
grep -nE 'func .*OAuthStart|func .*OAuthCallback|func .*RefreshConnection|h.flows\[' internal/admin/oauth.go
grep -nE 'func .*RefreshCredentials|func shouldRefresh|func .*refreshAndPersist|func mergeRefreshedCredentials' internal/auth/credentials.go
grep -nE 'flows := map\[string\]\*auth.OAuthFlow|"anthropic": auth.NewOAuthFlow' internal/server/routes_admin.go
grep -nE '/api/oauth/\{provider\}/start|/api/oauth/\{provider\}/callback' internal/server/routes_admin.go

# P2 — the secret-at-rest precedent is present (consume)
grep -nE 'AccessToken|RefreshToken|Metadata|s.cipher.Encrypt|s.cipher.Decrypt' internal/store/connections.go | head
grep -nE 'func .*CreateOAuthSession|func .*ConsumeOAuthSession|verifier_enc' internal/store/oauthsessions.go

# P3 — the device-code gap is REAL (no device-code path yet)
grep -rn 'device_code\|deviceCode\|grant-type:device' internal/ ; echo "^ expect EMPTY"
test ! -e internal/auth/oauth_device.go && echo "device-code gap OK"

# P4 — the w6-e OAuth modals/mocks (consume-only; correct ONLY on divergence)
grep -rln 'oauth' ui/e2e/mocks/handlers/ 2>/dev/null ; echo "^ the w6-e OAuth mocks (correct only if a flow shape diverges)"
# Record whether device-code modals exist for qwen/github/kilocode (open-question if absent).

# P5 — routes_admin.go serial slot is FREE (chain holder released)
git log --oneline -5 -- internal/server/routes_admin.go
# Orchestrator MUST confirm no concurrent W7 plan holds an unmerged routes_admin.go edit
# before T-register. This plan TAKES the slot for ONE additive flows-map edit, then RELEASES.

# P6 — green at base (HERMETIC)
go test ./... && go vet ./... && go build ./...     # exit 0 (no net)
```

---

## 3. Exclusive file ownership

After w7-prov-oauth merges, all CREATE files are owned by w7-prov-oauth; later plans
consume, never edit (MAP decision 7).

**CREATE — auth (NEW per-provider config + device-code path):**

| File | Contract |
|---|---|
| `internal/auth/oauth_providers.go` | The 8 `OAuthConfig` factory funcs: `ClaudeOAuth`, `CodexOAuth`, `GeminiCLIOAuth`, `QwenOAuth`, `IflowOAuth`, `GithubOAuth`, `KilocodeOAuth`, `ClineOAuth`. Each returns the verbatim ref config (clientId/tokenUrl/authUrl/deviceCodeUrl/scopes + the additive quirk fields). clientSecrets split-literal + env-overridable (mirror `GeminiOAuth` oauth.go:55-67). No `init()`; pure constructors. |
| `internal/auth/oauth_providers_test.go` | Field-by-field config assertions per provider (mirror `TestGeminiOAuthConfig`/`TestXaiOAuthConfig`): clientId/tokenUrl/authUrl/scopes/quirk-flags; env-override determinism; clientSecret prefix/suffix/length where applicable. RED first. |
| `internal/auth/oauth_device.go` | The device-code path: `DeviceCodeResponse`, `(f *OAuthFlow) StartDevice()`, `(f *OAuthFlow) PollDevice(ctx, deviceCode)`; the qwen/github form variant + the kilocode GET-status variant; injectable clock (a `now func()`/`after func(d) <-chan` flow field, real default, fake in tests); REUSE `GeneratePKCE` + `store.CreateOAuthSession`/`ConsumeOAuthSession`. No `init()`; errors-as-values. |
| `internal/auth/oauth_device_test.go` | Via `httptest.NewServer` + fake clock: request-device-code parse; poll loop `authorization_pending`→success (no real sleep); `slow_down`→interval bump; `expired_token`/`access_denied`→terminal err; deadline→timeout err; kilocode 202/403/410 status branches; qwen PKCE `code_verifier` sent on poll; tokens never echoed cleartext. RED first. NO real network/sleep. |

**EXTEND — auth (additive ONLY; no existing signature/body change to the anthropic path):**

| File | Change (additive ONLY) |
|---|---|
| `internal/auth/oauth.go` | ADD to `OAuthConfig`: `ExtraAuthParams map[string]string`, `RefreshMode string`, `RefreshURL string`, `CodeEncoding string`, `DeviceCodeURL string` (all zero-default → existing configs byte-identical). ADD additive branches: in `StartWithRedirect` append `ExtraAuthParams` after the existing query (empty map → no-op); in `Refresh`/`requestToken` a `RefreshMode` switch (`""`→existing form path UNCHANGED; `basic`→Basic-auth header; `json`→JSON body to `RefreshURL||TokenURL`; `none`→return a sentinel "refresh not supported"); in `ExchangeWithRedirect` a `CodeEncoding=="base64-json"` early-decode branch. ADD the 8 providers to `refreshLead` ONLY if a non-default lead is needed (default 5m is correct — likely no change). NO existing signature changes; NO anthropic-path behavior change (asserted by the existing oauth_test.go staying green + a byte-identical golden for anthropic authorize URL). ESC-CONFIG-FIELDS if any branch can't avoid touching the default path. |
| `internal/auth/oauth_test.go` (EXTEND additively) | If the additive branches need coverage co-located with the engine (e.g. a golden asserting anthropic authorize URL is byte-identical with an empty `ExtraAuthParams`), ADD there; per-provider config tests live in `oauth_providers_test.go`. |

**CREATE — admin (CONDITIONAL — only if device-code admin transport fits the slot, §1.4):**

| File | Contract |
|---|---|
| `internal/admin/oauth_device.go` (CONDITIONAL — ESC-DEVICE-ENDPOINT) | `OAuthDeviceStart` (returns user_code+verification_uri+interval) + `OAuthDevicePoll`/`OAuthDeviceComplete` (drives `PollDevice`, stores the connection). ONLY if the device admin transport lands in this plan; else NOT created (follow-up plan owns it; the flow engine still ships). Reuses `writeData`/`writeError`. |
| `internal/admin/oauth_device_test.go` (CONDITIONAL) | Via `newTestEnv` + httptest: device-start returns the display fields; complete stores a connection with `*_enc` tokens; no token echoed. RED first. |

**MODIFY — serial-slot flows-map registration (additive only):**

| File | Change |
|---|---|
| `internal/server/routes_admin.go` | ADD the 8 flows-map entries to `NewAdminHandlers` (`:21-23`): `"claude": auth.NewOAuthFlow(auth.ClaudeOAuth(), st, nil)`, … through `"cline"`. (If device admin endpoints land — ESC-DEVICE-ENDPOINT — ADD `r.POST("/api/oauth/{provider}/device/start", …)` + `…/device/poll` in the SAME commit.) NOTHING else changes. ONE commit. SERIAL SLOT — only holder while live; RELEASE on close. |

**MODIFY — w6-e mock corrections (CONDITIONAL — mirror real Go, decision 1):**

| File | Change |
|---|---|
| `ui/e2e/mocks/handlers/<oauth>.ts` (BODY — CONDITIONAL) | Correct an OAuth modal mock body ONLY if a flow's request/response shape diverges from the real Go (ESC-MOCK §8). Device-code modals (qwen/github/kilocode) differ from redirect — if w6-e mocked them redirect-style, correct; if absent, record a UI gap in `open-questions.md` (do NOT build UI). |

**FORBIDDEN:** everything else. Explicitly: the `AnthropicOAuth` factory + the existing
`Start`/`StartWithRedirect`/`Exchange`/`ExchangeWithRedirect`/`Refresh`/`requestToken`/
`pkceChallenge`/`randomURLSafe`/`GeneratePKCE` signatures + bodies (REUSE; additive
fields/branches only); `internal/admin/oauth.go` (the redirect handlers — CONSUME, do
NOT edit); `internal/auth/credentials.go` (the refresh orchestration — CONSUME; the
per-provider refresh quirks live in the `OAuthConfig.RefreshMode` branch in oauth.go,
NOT in credentials.go); all pre-existing `internal/store/*.go` (REUSE connections +
oauthsessions; NO migration change expected — tokens already `*_enc`, orgId→existing
`Metadata`); all `internal/providers/*` + `internal/inference/*` (catalog/adapter is
the w7-prov-* catalog plans); all other `internal/admin/*`; all `ui/src/**` (FROZEN);
all mocks/seeds/specs except the sanctioned OAuth mock-body correction. Touching any of
these is an automatic REJECT.

---

## 4. TDD tasks

Cadence (strict, AGENTS.md "TDD always: write test first, see it fail, write minimum
code to pass"): **no Go impl may exist before its `_test.go` is committed RED.**
`go test ./... && go vet ./... && go build ./...` green at EVERY commit, FULLY
HERMETIC (no real network via `httptest.NewServer`; no real sleep via the injected
clock). The existing `internal/auth/oauth_test.go` stays green throughout (additive
fields default-zero → byte-identical anthropic/gemini/xai behavior). Order: config
factories → additive engine fields/branches (redirect quirks) → device-code path →
flows-map registration → (conditional device admin) → closeout.

### T-configs — STEP(a) RED, STEP(b) impl (the 8 config factories)
STEP(a): write `internal/auth/oauth_providers_test.go` (field-by-field per provider,
env-override determinism, secret prefix/suffix/length where applicable). `go test
./internal/auth/ -run OAuthConfig` → FAIL. Commit RED:
`phase-1/w7-prov-oauth: failing 8-provider OAuth config tests (TDD red)`.
STEP(b): implement `internal/auth/oauth_providers.go` (the 8 factories; verbatim ref
configs; split-literal env-overridable secrets). Gates green. Commit:
`phase-1/w7-prov-oauth: 8 provider OAuth config factories (claude/codex/gemini-cli/qwen/iflow/github/kilocode/cline)`.

### T-engine — STEP(a) RED, STEP(b) impl (additive redirect-flow quirk fields)
STEP(a): EXTEND `oauth_test.go` (or a new `oauth_quirks_test.go`) with: anthropic
authorize URL byte-identical with empty `ExtraAuthParams` (regression guard); codex/
gemini-cli/iflow/cline authorize includes `ExtraAuthParams`; iflow refresh sends
Basic-auth header (`RefreshMode:"basic"`); cline refresh JSON body to `RefreshURL`
(`RefreshMode:"json"`); cline `CodeEncoding:"base64-json"` exchange decodes the code;
kilocode `RefreshMode:"none"` returns the sentinel. All via `httptest.NewServer`. →
FAIL. Commit RED: `phase-1/w7-prov-oauth: failing additive OAuth config-quirk tests (TDD red)`.
STEP(b): ADD the `OAuthConfig` fields + the additive branches in `StartWithRedirect`/
`Refresh`/`requestToken`/`ExchangeWithRedirect` (anthropic path UNCHANGED). Gates green
(existing oauth_test.go still green). Commit:
`phase-1/w7-prov-oauth: additive OAuthConfig quirks (extra-auth-params, refresh-mode, base64-code)`.

### T-device — STEP(a) RED, STEP(b) impl (the device-code path; qwen/github/kilocode)
STEP(a): write `internal/auth/oauth_device_test.go` (request-device-code parse; poll
pending→success via fake clock; slow_down interval bump; expired/denied terminal; deadline
timeout; kilocode 202/403/410; qwen PKCE code_verifier on poll; no token echo). → FAIL.
Commit RED: `phase-1/w7-prov-oauth: failing device-code flow tests (TDD red)`.
STEP(b): implement `internal/auth/oauth_device.go` (`StartDevice`/`PollDevice` +
injectable clock; the qwen/github form variant + kilocode GET-status variant; REUSE
GeneratePKCE + oauthsessions). Gates green (no real sleep). Commit:
`phase-1/w7-prov-oauth: device-code OAuth flow (request-code + poll, injectable clock)`.

### T-register — serial-slot flows-map registration
TAKE the serial slot (orchestrator confirms FREE at P5). ADD the 8 flows-map entries to
`NewAdminHandlers` (§3). (If device admin endpoints land — ESC-DEVICE-ENDPOINT — write
their RED test + impl `internal/admin/oauth_device.go` FIRST, then add their route lines
in this same commit.) Gates: `go test ./... && go vet ./... && go build ./...` green.
Commit (ONE commit touches the serial file):
`phase-1/w7-prov-oauth: register 8 provider OAuth flows (serial slot)`.

### T-close — full gates + closeout
```bash
go test ./internal/auth/... ./internal/admin/ -run 'OAuth|Flow|Device' -v
go test ./... && go vet ./... && go build ./...                       # HERMETIC — no net/sleep
```
Flip `.planning/parity/matrix/9router-providers.md`: PAR-PROV-015,016,017,018,019,021,
025,026 → HAVE (OAuth flow config + path; HTTP hermetically tested). Advance
`9router-platform.md` PAR-PLAT-047 PARTIAL→HAVE-advanced (8 flows; cursor/kiro/gitlab
footnote). Note `9router-auth.md` PAR-AUTH-019 coverage 3→11. Append §8 open items to
`open-questions.md` (kimi-coding follow-up; GitHub-Copilot-token deferral;
ESC-DEVICE-ENDPOINT outcome; any w6-e device-modal UI gap). Update `docs/WORKFLOW.md`
(P0 base; the generalization/device/refresh decisions; the flows mcp-3 consumes; the
serial-slot take/release). Final commit:
`phase-1/w7-prov-oauth: close — 8 provider OAuth flows; matrix flip`.
**On the close commit, RELEASE the routes_admin.go serial slot.**

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0. Diff gate is
**w7-prov-oauth commit-range-scoped** (§7).

**Test gates (HERMETIC — no real network, no real sleep)**
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/auth/... -run 'OAuth|Flow|Device' -v` → exit 0, all pass
  (8 config factories field-by-field; additive quirks: extra-auth-params on
  codex/gemini-cli/iflow/cline, Basic-auth refresh iflow, JSON refresh cline,
  base64-code cline, no-refresh kilocode; device-code: request+poll+pending+slow_down+
  expired+denied+timeout+kilocode-status+qwen-PKCE; anthropic byte-identical regression).
- `go test ./internal/admin/ -run 'OAuth|Device' -v` → exit 0 (existing redirect handler
  tests stay green; conditional device-handler tests pass if shipped).
- The pre-existing `internal/auth/oauth_test.go` suite stays GREEN (additive-only proof).

**TDD-order proof** — each impl file's covering test is in an earlier-or-equal commit:
```bash
for pair in \
  "internal/auth/oauth_providers_test.go:internal/auth/oauth_providers.go" \
  "internal/auth/oauth_device_test.go:internal/auth/oauth_device.go" ; do
  tf=${pair%%:*}; im=${pair##*:}
  ct=$(git log --format=%ct --diff-filter=A -1 -- "$tf")
  cf=$(git log --format=%ct --diff-filter=A -1 -- "$im")
  [ "$ct" -le "$cf" ] || echo "TDD VIOLATION: $im"     # prints nothing
done
```

**Grep proofs (per provider + per decision)**
```bash
# 8 config factories present
grep -nE 'func ClaudeOAuth|func CodexOAuth|func GeminiCLIOAuth|func QwenOAuth|func IflowOAuth|func GithubOAuth|func KilocodeOAuth|func ClineOAuth' internal/auth/oauth_providers.go   # 8 hits
# each provider's tokenUrl/clientId bound from the ref (verbatim)
grep -nE 'api.anthropic.com/v1/oauth/token|auth.openai.com/oauth/token|chat.qwen.ai/api/v1/oauth2|iflow.cn/oauth|github.com/login|api.kilo.ai/api/device-auth|api.cline.bot/api/v1/auth' internal/auth/oauth_providers.go
# device-code path present
grep -nE 'func .*StartDevice|func .*PollDevice|grant-type:device_code|urn:ietf:params:oauth:grant-type:device_code' internal/auth/oauth_device.go
grep -nE 'authorization_pending|slow_down|expired_token|access_denied' internal/auth/oauth_device.go   # poll branches
# additive quirk fields present, anthropic path untouched
grep -nE 'ExtraAuthParams|RefreshMode|RefreshURL|CodeEncoding|DeviceCodeURL' internal/auth/oauth.go
# injected-clock-in-device-test (no real sleep)
grep -nE 'httptest.NewServer|func .* clock|now func\(\)|after func\(' internal/auth/oauth_device_test.go
! grep -nE 'time.Sleep' internal/auth/oauth_device.go && echo "no real sleep in device path OK"
# flows-map registration (serial slot)
grep -nE '"claude": auth.NewOAuthFlow|"codex":|"gemini-cli":|"qwen":|"iflow":|"github":|"kilocode":|"cline":' internal/server/routes_admin.go   # 8 entries
# no init(); errors-as-values
! grep -rn 'func init(' internal/auth/oauth_providers.go internal/auth/oauth_device.go && echo "no init() OK"
```

**No-secret-exposure proofs (binding)**
```bash
# tokens stored *_enc (via the existing connections/oauthsessions store — no plaintext
# token column added by this plan):
! grep -nE 'access_token TEXT|refresh_token TEXT' internal/store/*.go | grep -v _enc && echo "no plaintext token column OK"
# no token echoed in any device-handler DTO (if shipped):
test ! -e internal/admin/oauth_device.go || ! grep -nE 'AccessToken|RefreshToken' internal/admin/oauth_device.go | grep -iE 'json:"' && echo "no token json field OK"
# clientSecrets stay split-literal + env-overridable (scanner-evasion precedent):
grep -nE 'os.Getenv\(' internal/auth/oauth_providers.go   # secrets env-overridable
# additive migrations only — none expected:
git diff <base>..HEAD -- internal/store/migrate.go | wc -l   # = 0 (no store change)
```

**Anthropic-regression proof (binding — additive-only)**
```bash
# the existing oauth_test.go suite must stay green (no anthropic/gemini/xai behavior change):
go test ./internal/auth/ -run 'OAuth|Refresh|Gemini|Xai' -v   # exit 0, all pre-existing pass
# golden: anthropic authorize URL byte-identical with empty ExtraAuthParams (in T-engine test)
```

**Negative / freeze proofs (w7-prov-oauth commit-range — §7)**
```bash
R="<first-w7-prov-oauth>^..<last-w7-prov-oauth>"
# Only the sanctioned files changed:
git diff $R --name-only -- internal/ | grep -vE \
 'internal/auth/(oauth_providers|oauth_device)(_test)?\.go|internal/auth/oauth(_test|_quirks_test)?\.go|internal/admin/oauth_device(_test)?\.go|internal/server/routes_admin\.go' \
 | wc -l                                                                  # = 0
# Frozen reused surfaces untouched:
git diff $R --name-only -- internal/admin/oauth.go internal/auth/credentials.go internal/store/connections.go internal/store/oauthsessions.go | wc -l   # = 0
# anthropic factory body untouched (additive fields only, no factory edit):
git diff $R -- internal/auth/oauth.go | grep -E '^-' | grep -v '^---' | grep -iE 'AnthropicOAuth|func \(f \*OAuthFlow\) (Start|Exchange|Refresh|requestToken)\(' | wc -l   # = 0 (no deletions in those)
# UI frozen except a sanctioned OAuth mock-body correction:
git diff $R --name-only -- ui/src/ | wc -l                               # = 0
# routes_admin.go = exactly ONE commit, additive:
git log --oneline $R -- internal/server/routes_admin.go | wc -l          # = 1
```

---

## 6. Out of scope (restated, binding)

No anthropic-flow rewrite (REUSE the config-driven engine; additive fields/branches/
methods only). No new public route line (flows-map entry only; device endpoints share
the one serial commit or defer). No catalog/adapter/model work (w7-prov-* catalog plans).
No kimi-coding (024) / xai (027 — shipped w3-f) / cursor / kiro / gitlab / antigravity.
No GitHub Copilot-token mint (deferred). No `credentials.go` body edit (refresh quirks
live in the `OAuthConfig.RefreshMode` branch). No migration change (tokens already
`*_enc`; orgId→existing `Metadata`). No `New(...)` signature change. No real network/
sleep in tests. No secret exposure. Any provider whose config can't be soundly read
from the ref → escalate (§8) with a recommended default; flip ONLY soundly-built rows.

## 7. Diff-gate scope

W7 provider/catalog plans commit to main concurrently, so a broad `<base>..HEAD` range
sweeps in sibling commits. The diff gate MUST be scoped to w7-prov-oauth's own commits:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w7-prov-oauth:" | awk '{print $1}'`
then `git diff <first>^..<last> -- [file list]`.

`git diff <first>^..<last> --name-only` must be exactly a subset of:
```
internal/auth/oauth_providers.go
internal/auth/oauth_providers_test.go
internal/auth/oauth_device.go
internal/auth/oauth_device_test.go
internal/auth/oauth.go                  (additive OAuthConfig fields + additive branches ONLY)
internal/auth/oauth_test.go             (additive regression/quirk tests; or oauth_quirks_test.go)
internal/admin/oauth_device.go          (CONDITIONAL — ESC-DEVICE-ENDPOINT)
internal/admin/oauth_device_test.go     (CONDITIONAL)
internal/server/routes_admin.go         (serial-slot additive flows-map; ONE commit)
ui/e2e/mocks/handlers/<oauth>.ts        (CONDITIONAL body-only — mirror Go on divergence)
.planning/parity/matrix/9router-providers.md
.planning/parity/matrix/9router-platform.md
.planning/parity/matrix/9router-auth.md
.planning/parity/plans/open-questions.md
docs/WORKFLOW.md
```
Any file outside this list in the scoped diff is an automatic review REJECT.
`internal/admin/oauth.go`, `internal/auth/credentials.go`, the store files, all
`internal/providers/*`/`internal/inference/*`, and all `ui/src/**` are deliberately
ABSENT — touching them is an automatic REJECT. The `routes_admin.go` edit must appear
in exactly ONE commit and the serial slot is released on close.

## 8. Escalations / decisions (explicit — recommended defaults, do not fabricate)

- **ESC-CONFIG-FIELDS (RESOLVED at authoring — additive generalization, binding default).**
  `oauth.go` is already config-driven; the 8 providers are config factories + 5 additive
  `OAuthConfig` fields (`ExtraAuthParams`, `RefreshMode`, `RefreshURL`, `CodeEncoding`,
  `DeviceCodeURL`) consumed by additive branches that default to the EXISTING anthropic
  path byte-for-byte. **Decision: additive fields + additive branches; NO anthropic
  rewrite, NO `RegisterFlow` registry** (the flows map IS the registry). If any additive
  branch cannot avoid changing the default-path behavior, STOP and escalate before
  editing. Proven by the anthropic-regression golden + the green pre-existing oauth_test.go.
- **ESC-DEVICE-TYPE (RESOLVED — methods on OAuthFlow, binding default).** The device-code
  path is added as `StartDevice`/`PollDevice` METHODS on the existing `OAuthFlow`
  (gated by the additive `DeviceCodeURL` field), so the SAME flows map holds both
  redirect and device flows and the admin handlers can branch on `cfg.DeviceCodeURL != ""`.
  Alternative (a sibling `DeviceFlow` type) is rejected as more surface for no gain.
- **ESC-DEVICE-ENDPOINT (OPEN — admin transport for device-code; recommended default).**
  The existing `OAuthStart`/`OAuthCallback` are redirect-shaped and do NOT fit device-code
  (user enters a code; server polls). **Recommended default:** ship the device FLOW
  ENGINE + the 3 device config factories + flows-map entries regardless; add a NEW
  `internal/admin/oauth_device.go` (`POST /api/oauth/{provider}/device/start` +
  `…/device/poll`) ONLY if it fits the one serial-slot commit window AND mirrors the w6-e
  modal. If the route shape can't be confirmed against the w6-e modal, DEFER the device
  admin transport to a follow-up plan (record in `open-questions.md`) — the engine still
  ships and is consumable. Flag for orchestrator confirmation; never silently diverge
  the modal/route.
- **ESC-GH-COPILOT (RESOLVED — defer Copilot-token mint, binding default).** GitHub's
  device-code yields a GitHub access_token; using Copilot requires a SECOND exchange to
  `copilotTokenUrl` (`oauth.js:138`, `services/github.js:106`). **Decision:** store the
  GitHub access_token from the device flow; the Copilot-token mint (short-lived, re-minted
  per request) is a follow-up recorded in `open-questions.md`. The OAuth flow row
  (PAR-PROV-021) flips on the device-code half; the Copilot-token integration is the
  adapter/runtime concern (w7-prov-* / w7-usage-quota).
- **ESC-KMC-XAI (RESOLVED — out of the brief's 8, follow-up).** PAR-PROV-024
  (kimi-coding, device-code, `oauth.js:167-171`) and PAR-PROV-027 (xai — already shipped
  w3-f) are NOT in this plan. kimi-coding's config IS sound (trivially the same
  device-code path + the `buildKimiHeaders` quirk) — recorded as a low-effort follow-up
  in `open-questions.md`, not built (honoring the brief's exact 8). The brief's "024"
  was a row-number slip (kilocode is 026).
- **ESC-MOCK (CONDITIONAL — w6-e OAuth modal ripple).** The w6-e OAuth modals/mocks are
  consumed-unchanged unless a flow's request/response shape diverges from the real Go
  (decision 1: real Go wins, mock mirrors). Device-code modals differ from redirect — if
  w6-e mocked qwen/github/kilocode redirect-style, correct the mock body; if w6-e has no
  device-code modal, record a UI gap in `open-questions.md` (do NOT build UI here). If a
  mock correction reds a non-w7-prov-oauth spec, STOP + ESCALATE.
- **Serial-slot dependency (§1.3 / P5).** w7-prov-oauth TAKES the routes_admin.go slot
  for ONE additive flows-map edit and RELEASES on close. Orchestrator confirms exactly
  one unmerged holder (MAP decision 3) before T-register.
- **No fabricated credentials (binding).** Every clientId/tokenUrl/authUrl/deviceCodeUrl
  is copied verbatim from the cited ref blocks. Any provider whose config cannot be
  soundly read is ESCALATED with a recommended default and NOT flipped — never invent a
  client_id or endpoint.
```
