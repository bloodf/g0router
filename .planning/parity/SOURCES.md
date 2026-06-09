# Parity Reference Sources (frozen)

## North star
g0router is a **total replacement** for both references. End state: a user running 9router or Bifrost can switch to g0router and lose nothing.
- v1.0 = drop-in 9router replacement (behavior + page-by-page dashboard parity, Go-native equivalents for platform-bound features).
- v1.1 = full Bifrost OpenAI surface + MCP gateway.
- Stage 3 = remaining Bifrost capabilities (adaptive LB, semantic cache, guardrails, hierarchical governance, observability, cluster mode) — planned, user-approved, then built.
PARITY.md rows and micro-plans are scoped against this bar: a gap in either reference is a gap in g0router.

Cloned 2026-06-09. No re-sync during Stage 1. Final upstream delta review only at v1.0 tag.

| Repo | Path | Frozen SHA | Commit date | License |
|------|------|-----------|-------------|---------|
| decolua/9router | `~/Developer/github.com/bloodf/_refs/9router` | `827e5c382b11f90b876f856ffa99cbd50f6abd6b` | 2026-06-06 (v0.4.71) | MIT |
| maximhq/bifrost | `~/Developer/github.com/bloodf/_refs/bifrost` | `ca212988f50a836954d4b454a9fec7af05affcf4` | 2026-06-10 | Apache-2.0 |

## License decisions

- **9router (MIT):** permits derived reimplementation and near-literal UI component porting. Preserve copyright notice attribution: add 9router MIT attribution to `docs/REFERENCES.md` and release notes for ported UI structures.
- **Bifrost (Apache-2.0):** permits derived reimplementation. We port designs/behaviors, not code verbatim; if any code is adapted verbatim, retain the Apache-2.0 notice for those portions.

## Repo shapes (top level)

- 9router: Next.js app — `src/`, `open-sse/` (SSE translation core), `cli/`, `scripts/`, `tests/`, `docs/`, `i18n/`, `skills/`
- bifrost: Go — `core/`, `framework/`, `transports/`, `plugins/`, `ui/`, `docs/`, `tests/`, `npx/`
