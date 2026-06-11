# Wave 3 — OAuth + auth hardening: micro-plan index (Stage-1 scope)

Author: Fable 5 (planner). Orchestrator: Sonnet. Implementers: kimi/M3. Gates: gpt-5.5.
**Non-authorizing INDEX** (like WAVE-2-MAP): each `w3-<slug>.md` micro-plan carries its
own rows, evidence, TDD tasks, and binary acceptance, and goes through the plan gate
before dispatch. Frozen ref @ 827e5c3. Depends on Wave 2 — COMPLETE.

## Stage-1 scope decision (mirrors the Wave-2 rescope)

WAVE-MAP row 3 says "~15 monolithic per-provider OAuth handlers ordered by popularity
(decision 1)" — but its own "why this order" states **handlers need provider adapters
to validate against**, and the providers-matrix Stage-1 ranking
(`matrix/9router-providers.md:216+`) deferred the OAuth providers (codex, gemini-cli,
qwen, iflow, antigravity, github, kiro, cursor, kimi-coding, cline, kilocode) to
Stage 2+ WITH their adapters. Therefore Wave 3 Stage-1 implements OAuth handlers ONLY
for providers whose adapters exist today: **anthropic/claude** (PAR-AUTH-019 PARTIAL —
complete it), **gemini** (adapter HAVE; OAuth fields `providers.js:58-62`), **xai**
(generic-adapter HAVE; OAuth fields `providers.js:273-280`, deferred from w2-b).
The other ~11 handlers land in Stage 2 with their providers (decision 1's popularity
order applies within each stage). Dashboard/gateway auth hardening (the bulk of
PAR-AUTH) is fully in-scope — it has no Stage-2 dependency.

Decision 2 governs sessions: **opaque SQLite tokens, no JWT** — PAR-AUTH-003 (JWT)
is closed by decision (opaque equivalent already HAVE per PAR-AUTH-030); PAR-PR-1711's
cookie semantics are adapted to the opaque token (unified parser yes; 30d TTL no —
decision 2 fixes 7-day).

## Row coverage (30 PAR-AUTH rows)

- Already HAVE (6): 001, 004, 016, 024, 025, 030.
- Closed by decision 2 (1): 003 (JWT → opaque tokens).
- In-scope MISSING/PARTIAL (23): 002, 005-015, 017-023, 026-029 → the 6 plans below.
- PR ports in-scope: PAR-PR-1249 (OAuth redirect URI for remote deployments → w3-f),
  PAR-PR-1711 (unified cookie parser, opaque-adapted → w3-b). Deferred with their
  Stage-2 providers: PAR-PR-717/641/1388/1458/1004/665.

## Micro-plan index (6 plans, two tracks)

| Plan | Scope | PAR-AUTH rows | Key ref evidence | Depends |
|---|---|---|---|---|
| **w3-a** | Login hardening: default-password fallback, auth-mode switch (password/oidc/both), login rate limiter + progressive lockout + 1h auto-reset + client-IP extraction, password-reset CLI | 002, 006, 014, 015, 023, 026 | `src/app/api/auth/login/route.js:40-50`, `src/lib/auth/loginLimiter.js:3-51`, `cli/src/cli/menus/settings.js:177-204`, `src/app/api/auth/status/route.js:13` | W2 |
| **w3-b** | Centralized dashboard guard middleware (path lists), local-only route gate (loopback host+origin), tunnel dashboard toggle + tunnel login block, unified cookie parser (PR-1711, opaque-adapted) | 007, 011, 013, 027 + PR-1711 | `src/dashboardGuard.js:22-65,69-100,165-241,197-214`, `src/app/api/auth/login/route.js:11-16,33-35` | w3-a |
| **w3-c** | OIDC dashboard login with PKCE, logout clears OIDC cookies, OIDC cookie TTL (10 min), client-secret probe endpoint | 005, 021, 022, 028 | `src/lib/auth/oidc.js:74-78,144-210`, `src/app/api/auth/oidc/start/route.js:24-46,42`, `src/app/api/auth/logout/route.js:8-10` | w3-a, w3-b |
| **w3-d** | API key system: key table with machineId, key format machineId+CRC8, remote API-key validation in guard, loopback no-key access, CLI token auth | 008, 009, 010, 012, 029 | `src/shared/utils/apiKey.js:34-38`, `src/lib/db/schema.js:74-84`, `src/dashboardGuard.js:6-19,35,102-122,177` | w3-b |
| **w3-e** | Security hardening: request-log header sanitization, debug-log production gate, SSRF outbound-proxy protections | 017, 018, 020 | `src/lib/db/repos/requestDetailsRepo.js:46-54`, `open-sse/utils/debugLog.js:3`, `open-sse/utils/proxyFetch.js:314-334` | W2 (independent of a-d) |
| **w3-f** | Provider OAuth (decision 1, monolithic per-provider): complete anthropic (019 PARTIAL — token persistence + refresh-on-expiry + key resolution into adapters), add gemini + xai handlers, PR-1249 redirect URI, credentials plumbing (providerSpecificData → ollama host override from w2-c, generic-adapter refresh hook from w2-b) | 019 + PR-1249 (+ w2 deferrals) | `open-sse/services/oauthCredentialManager.js`, `src/lib/oauth/services/*`, `providers.js:58-62,273-280,442-445`, `executors/default.js:186-312`; in-repo `internal/auth/oauth.go:34-184` | W2 (parallel to a-e track) |

## Tracks & ownership

- **Dashboard track (serial):** w3-a → w3-b → w3-c and w3-d (c∥d after b; both touch
  guard-adjacent files — c owns OIDC files, d owns apikey files; the guard file itself
  is owned by b, c/d only ADD hook call-sites in their own files or via b-provided
  extension points). w3-e independent (logging/proxy files).
- **Provider track (parallel):** w3-f touches `internal/auth/oauth.go`,
  `internal/providers/*` credential plumbing — disjoint from the dashboard track.

## Per-micro-plan protocol (unchanged)

Fable 5 plan → gpt-5.5 plan gate (max 3 cycles → decide) → kimi impl (TDD) →
`go test ./... && go vet ./...` (+`-race` where concurrency) → scoped diff gate →
merge → flip rows HAVE → WORKFLOW.md.

## Out of Wave-3 scope (explicit)

The ~11 Stage-2 provider OAuth handlers + their PR ports. JWT (decision 2). Tunnel
implementation itself (Wave 7 platform — w3-b only gates on its config flag). UI
login pages (Wave 6). Combo/fallback routing (Wave 4). Usage (Wave 5).
