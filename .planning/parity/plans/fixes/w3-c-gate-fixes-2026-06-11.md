# Fix — w3-c diff-gate findings (2026-06-11). All REAL (one security).

Author: Fable 5. Implementer: kimi. Dispatch AFTER w3-d merges (shared internal/admin package).

## Task 1 — SECURITY: public probe must NOT read stored secrets (BLOCKER)
`internal/admin/oidc.go` `OIDCTest`/probe: REMOVE the fallback to stored settings
(`oidc_client_secret`, issuer, etc.). The public `/api/auth/oidc/test` endpoint
operates EXCLUSIVELY on caller-provided values (tokenEndpoint, clientId, clientSecret,
redirectUri from the request body), per the plan and ref `probeOidcClientSecret`
(`oidc.js:144-210`, which takes all inputs as parameters). An unauthenticated caller
must never be able to trigger validation using the server's stored secret. Test:
`TestProbeDoesNotUseStoredSecret` — stored oidc_client_secret set in settings; probe
called with NO body secret → returns the "no client secret provided, skipped" result
(tested:false, valid:null), NOT a validation using the stored secret.

## Task 2 — probe accepts caller tokenEndpoint (MAJOR, PAR-AUTH-028 shape)
Probe request body includes `token_endpoint` (snake_case) used directly; do NOT force
issuer discovery. Matches `probeOidcClientSecret({tokenEndpoint,...})` (`oidc.js:144-153`).

## Task 3 — snake_case JSON response (MAJOR, repo convention)
Rename probe response fields to snake_case: `discoveryOk`→`discovery_ok`,
`clientSecretTested`→`client_secret_tested`, etc. (AGENTS.md snake_case envelope).
Update the probe tests accordingly.

## Task 4 — prove public reachability through the guard (MAJOR)
`TestProbeEndpointPublic`: exercise the endpoint through the routed server (guard-wrapped
handler), NOT a direct `h.OIDCTest` call — assert a no-session request reaches it (200/
result), proving it is in the public set. Use the server test harness like
`TestServerGuardWired`.

## Acceptance (binary)
- `go test ./... && go vet ./...` green.
- `grep -rn 'oidc_client_secret\|settings\[' internal/admin/oidc.go` shows the PROBE path does NOT read stored secret (only start/callback may read config).
- `grep -c 'discoveryOk\|clientSecretTested\|camelCase' internal/admin/oidc.go` → 0 (snake_case).
- `TestProbeDoesNotUseStoredSecret`, `TestProbeEndpointPublic` (routed) pass.
- Files ONLY: internal/admin/oidc.go, internal/admin/oidc_test.go, internal/auth/oidc.go (if probe signature needs the tokenEndpoint param), internal/auth/oidc_test.go. Do NOT git commit.
