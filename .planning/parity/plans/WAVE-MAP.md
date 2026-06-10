# Stage 1 Wave Map — micro-plan factory index

Author: Fable 5 (plans). Orchestrator: Claude Code (VPS). Source: PARITY.md (approved 2026-06-09) + 10 checkpoint decisions.
Every micro-plan in `.planning/parity/plans/` cites PARITY row IDs, passes the gpt-5.5 critic gate before dispatch, and is executed by Kimi with a gpt-5.5 scoped diff gate before merge.

## Wave structure

Waves run sequentially; plans inside a wave run in parallel when file ownership is disjoint. Row counts include PAR-PR ports mapped to that domain.

| Wave | Scope | Rows | Plans (est.) | Depends on | Why this order |
|---|---|---:|---:|---|---|
| 0 | Audit remediation — bundles A–E from PARITY §2 | 22 BROKEN (+10 co-located DEBT) | 5 | — | Broken crypto/error paths poison every later diff; UI scaffolding (bundle D) unblocks Wave 6 e2e |
| 1 | Translation engine core: translator registry, 12 wire formats, openai-intermediate normalization, helpers (tool-call IDs, max_tokens, thinking, cache_control), SSE stream processor | PAR-TRANS 55 + ~25 PR ports | 8–10 | 0 | Everything routes through translation; providers and routing consume it |
| 2 | Providers: all 43+ adapters, executors, model catalogs, full Bifrost-size interface (decision 9) with typed not-implemented stubs | PAR-PROV 66 + ~25 PR ports | 10–14 | 1 | Adapters consume translator; interface expansion lands here once |
| 3 | OAuth + auth: ~15 monolithic per-provider OAuth handlers ordered by popularity (decision 1), token refresh, session hardening (opaque tokens, decision 2) | PAR-AUTH 29 + ~8 PR ports | 6–8 | 2 | Handlers need provider adapters to validate against |
| 4 | Routing: combo chains, fallback, rate-limit rotation, bypass patterns (PAR-ROUTE-034 canonical), model aliases | PAR-ROUTE 60 + ~20 PR ports | 6–8 | 1, 2 | Routing composes providers + translation |
| 5 | Usage: request_log accounting, cost computation, token counting, Overview aggregations | PAR-USAGE 40 + ~4 PR ports | 4–5 | 4 | Logs real routed traffic |
| 6 | Dashboard UI: page-by-page parity (Vite + React 19 + Tailwind 4 + shadcn/ui), 39 locales via react-i18next (decision 3), e2e specs | PAR-UI 128 + ~12 PR ports | 10–14 | 3, 4, 5 | UI consumes management + usage APIs |
| 7 | Platform: Go-native equivalents (decision 10) — systemd/launchd service, go-selfupdate, crypto/tls CA MITM proxy, download-on-demand tunnels, CLI | PAR-PLAT 48 | 5–7 | 4 | Independent of UI; needs working gateway to wrap |
| 8 | Release hardening: 138 negative-test issues as regression specs, live smoke CI for reverse-engineered providers (decision 5), docs, v1.0 tag | — | 3–4 | all | Exit gate: 100% Stage 1 rows HAVE except exclusion list |

Excluded (user-approved, decision 4): 9router Cowork/MCP-bridge rows serving the disabled Cowork feature (`matrix/9router-mcp.md`); superseded by Stage 2 Bifrost MCP gateway. PAR-MCP rows that describe client-facing MCP tool injection used by live flows stay in Wave 1/4 as applicable.

## Plan factory protocol (per micro-plan)

1. Fable 5 writes `plans/w<wave>-<slug>.md`: cited PAR/AUD rows, TDD-ordered tasks (failing test first), exact file ownership, binary acceptance criteria, out-of-scope list.
2. gpt-5.5 critic gate (`run-critic.sh plan`); max 3 reject cycles then user escalation.
3. Kimi implements verbatim; deviations require plan amendment, not improvisation.
4. Run gates (`go test ./...`, `go vet ./...`, ui build when touched).
5. gpt-5.5 scoped diff gate (`run-diff-scoped.sh gpt` + `diff-scopes.json`); REJECT loops back to Kimi or orchestrator fix.
6. Merge to main, mark rows HAVE in PARITY.md, update docs/WORKFLOW.md.

## Sizing

Total Stage 1: 589 implementable rows, ~60-75 micro-plans, 9 waves. Plans are written wave-by-wave (not all upfront) so later plans absorb learnings; the critic gate applies to every plan regardless.

## Wave 1 remainder slicing (Fable 5, 2026-06-10)

After w1-a..f gate closure and w1-g (Responses API translators, in flight):

| Plan | Scope | Ref surface | Rows |
|------|-------|-------------|------|
| w1-g2 | `/v1/responses` route + handler over registry pipeline; `response.failed` flush synthesis | `formats.js:23-24`, index.js route flow, `stream.js` failed-flush | PAR-TRANS-050, 002 (openai-responses clause), 031 route note |
| w1-h | ollama + commandcode translator pairs (+ ollamaTransform util) | `openai-to-ollama.js` (192), `ollama-to-openai.js` (152), `ollamaTransform.js` (85), `openai-to-commandcode.js` (170), `commandcode-to-openai.js` (197) | 002 clauses, 047 NDJSON closure |
| w1-i | kiro translator pair (+ kiroConstants) | `openai-to-kiro.js` (583; `.old.js` is dead — NOT ported), `kiro-to-openai.js` (195), `config/kiroConstants.js` | 002 clause |
| w1-j | cursor translator pair + protobuf/checksum utils | `openai-to-cursor.js` (183), `cursor-to-openai.js` (30), `cursorProtobuf.js` (904), `cursorChecksum.js` (149) | 002 clause |
| w1-k | gemini-format client requests + claude pipeline helpers | `gemini-to-openai.js` request (147), `claudeCloaking.js` (155), `bypassHandler.js` (298), `toolDeduper.js` (49), `reasoningContentInjector.js` (79), deepseek alias | PAR-TRANS-006, 022, 051, 052, 053, 054, 055; 057 verification vs w1-b |

Order: g2 after w1-g lands; h → i → j → k by ascending integration risk; plans
written one at a time so each absorbs prior gate learnings.
