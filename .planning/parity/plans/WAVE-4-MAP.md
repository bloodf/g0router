# Wave 4 — Routing: micro-plan index (Stage-1 scope)

Author: Fable 5. Orchestrator: Sonnet. Implementers: kimi. Gates: gpt-5.5.
**Non-authorizing INDEX** (like WAVE-2/3-MAP). Frozen ref @ 827e5c3. Depends on
Waves 1–3 — COMPLETE. Matrix: `matrix/9router-routing.md` (60 rows: 1 HAVE, 1
PARTIAL, 58 MISSING). WAVE-MAP row 4: "combo chains, fallback, rate-limit rotation,
bypass patterns (PAR-ROUTE-034 canonical), model aliases".

## Audit precondition

`reviews/wave0-3-audit-2026-06-12.md` found 2 HIGH wiring gaps (credential resolver
never wired: `SetKeyResolver` zero production callers; flows map = anthropic-only) +
3 real bugs (models/{id} unfiltered, randomUUID silent placeholder, stream loop no
abort) + stale comments. **w4-pre fixes all of these first** — routing work assumes
real credentials flow.

## Stage-1 scope decisions

- IN (40 rows): combos 001-004,011,024,046,047; aliases 005-010,040; locks/cooldown
  012-015,025,026,049; selection 016-019,027,050,051; retry/errors 020-022,044,045,
  048; pipeline 023,033,034,041,042,043,052; models-kind 037,038 (consideration #7).
- VERIFY-FLIP (already implemented by earlier waves — w4-f confirms + flips): 028/029
  (w3-d guard API-key validation/extraction), 036 Stage-1 half (w2-b config-driven
  headers), 035 Stage-1 half (single baseUrl building — multi-URL fallback exists only
  for Stage-2 antigravity, defers with it).
- DEFERRED: 030/031 virtual-key routing → **Wave 5** (governance/usage adjacency; 032
  schema already HAVE); 054 request-log attribution → **Wave 5** (request_log lands
  there, with PAR-AUTH-017/018); 039 free-tier injection, 053 project-id cold-miss,
  056 Kiro/Qoder live catalogs, 059 search/fetch models, 060 upstream UUID → **Stage
  2** (their providers are Stage-2); 055 proxy pools → **Stage-2/W7** (with w3-e MITM
  half); 057 custom models + 058 sub-config exposure → **Wave 6** (settings/UI-driven
  catalogs); PAR-PR-339 (combo list UI) → Wave 6; PAR-PR-1402/645 → Stage 2 (codex/
  cursor providers).
- PR ports IN: PAR-PR-485 (passthrough alias lookup by providerId → w4-a), PAR-PR-640
  (prevent infinite retry when all accounts error → w4-d), PAR-PR-648 (reset combo
  state on prop change → w4-e), PAR-PR-1626 (token-param auto-learning fallback →
  w4-b, openai-compatible).

## Micro-plan index (7 plans)

| Plan | Scope | Rows | Key ref evidence | Depends |
|---|---|---|---|---|
| **w4-pre** | Audit fixes: wire CredentialResolver+SetKeyResolver+gemini/xai flows in server.go; models/{id} filter; randomUUID error; stream abort select; stale comments. PLUS Wave-1 deferrals PAR-TRANS-006/051/052/053 (stripContentTypes/dedupeTools/injectReasoningContent pipeline helpers) | audit G1-G6 + PAR-TRANS-006/051/052/053 | audit doc; `server.go:35-46`, `models.go:57-60`, `apikey.go:183-189`; ref `chatCore.js` preprocess | — (FIRST) |
| **w4-a** | Aliases & resolution: alias map + cycle detection (DFS at write, consideration #6), ~140 provider aliases, `provider/model` prefix parsing (complete 007), name-prefix inference, provider-node prefixes, PR-485 | 005,006,007,008,009,010,040 + PR-485 | `open-sse/services/model.js:1-208` | w4-pre |
| **w4-b** | Error classification + retry: errorConfig port (text-first rules `errorConfig.js:59-76`, 30-min resetsAt cap `:42`), per-URL/provider retry, connect-timeout→502, quota-window parsing, retry middleware (consideration #3,4), PR-1626 | 020,021,022,044,045,048 + PR-1626 | `executors/base.js:98-174`, `config/errorConfig.js`, `config/runtimeConfig.js:52-57` | w4-pre (∥ w4-a) |
| **w4-c** | Connection/account state: `connection_model_locks` table (consideration #2), per-model + account locks, exponential backoff cooldown, success reset, disabled-model tracking + /v1/models exclusion, group-lock earliest-retry | 012,013,014,015,025,026,049 | `services/accountFallback.js`, `src/sse/services/auth.js:203-241` | w4-pre (∥ a,b) |
| **w4-d** | Selection & fallback: fill-first/round-robin/sticky, weighted, per-provider strategy override, pinned preference, excludeConnectionIds fallback loop, selection mutex (Go mutex, consideration), PR-640 | 016,017,018,019,027,050,051 + PR-640 | `src/sse/services/auth.js:9-157`, `src/sse/handlers/chat.js:162-245` | w4-c |
| **w4-e** | Combos: fallback + round-robin strategies, sticky limit (default-1 normalization), per-combo override, name validation, recursion protection, transient cooldown, earliest retry-after, /v1/models promotion, PR-648 | 001,002,003,004,011,024,046,047 + PR-648 | `open-sse/services/combo.js` whole | w4-a,b,c,d |
| **w4-f** | Pipeline glue: format auto-detect (033), WIRE the w1 bypass handler (034 canonical), native-passthrough detection (041), thinking override (042), stream decision (043), 401/403 refresh-retry + refresh-before-dispatch via the (now-wired) resolver (023, 052), /v1/models/{kind} + model-test-by-kind (037,038); VERIFY-FLIP 028/029/035/036 | 023,033,034,037,038,041,042,043,052 (+flips) | `services/provider.js:49-126`, `utils/bypassHandler.js:11-91`, `handlers/chatCore.js:86-103,216-235` | w4-a,b (∥ c,d) |

## Ownership tracks (concurrency rules learned in W3: NO shared files across live jobs)

- w4-a: `internal/inference/alias*.go`, `internal/providers/catalog/aliases.go`.
- w4-b: `internal/inference/errorclass*.go`, `retry*.go`; touches NO files of a/c.
- w4-c: `internal/store/connlocks*.go` + migrate.go, `internal/inference/accounts*.go`.
- w4-d: `internal/inference/selection*.go` (+accounts from c — AFTER c merges).
- w4-e: `internal/inference/combo*.go` + store combos table.
- w4-f: `internal/api/*` + router glue — serialized LAST among its deps; the ONLY
  plan touching internal/api.
- w4-pre: server.go/admin wiring + api fixes + translation helpers — runs ALONE first.

## Protocol (unchanged)

Plan → gpt-5.5 plan gate (≤3 cycles → decide) → kimi TDD impl → go test/vet/-race →
scoped diff gate (cumulative base..HEAD diffs; live-tree verification before any
closure) → merge → flip rows → WORKFLOW.md.

## Out of Wave-4 scope (explicit)

Virtual keys (W5), request-log/usage (W5), UI (W6), proxy pools + MITM (S2/W7),
Stage-2 providers' routing rows (with their providers), tunnels (W7).
