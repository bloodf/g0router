# Fix micro-plan — w1-c stream processor diff-gate findings (2026-06-10)

Author: Fable 5 (planner). Plan-before-action: this triage precedes any action.

HANDOFF.md carried two w1-c blockers from the original (broad-diff, polluted)
REJECT: "SSE malformed line warn+skip" and "stream finish usage".

## Triage — both findings are already satisfied at HEAD

| Finding | State at HEAD | Evidence |
|---------|---------------|----------|
| SSE malformed line warn+skip | IMPLEMENTED + TESTED | `internal/providers/utils/sse.go:94-103` logs `[WARN] failed to parse SSE line` and returns not-ok (line skipped, stream continues); test asserts both skip and warn at `internal/providers/utils/sse_test.go:73-86` |
| Stream finish usage | IMPLEMENTED per plan scope | `internal/translation/stream.go:63-66` attaches `state.Usage` on finish chunks. The w1-c plan (line 47) explicitly ports `stream.js:288-299` WITHOUT `estimateUsage`/`addBufferToUsage`/`filterUsageForFormat` — those are Wave 5 (PAR-USAGE-001..012); acceptance check (plan line 61) requires `estimateUsage` absent from stream.go |

## Action

1. NO code changes.
2. Re-run the scoped diff gate over the commit-bounded range from
   `diff-scopes.json` (`32108f04..6317eb4b`, w1-c owned paths). None of the
   w1-d/e/f gate-fix commits touch w1-c paths, so the base range is clean.
3. If the gate returns REAL findings (not artifacts/false positives vs the
   frozen ref), a follow-up fix micro-plan will be written and dispatched to
   kimi/M3 via `run-worker.sh` per the plan-before-action protocol.

## Out of scope

Usage estimation/buffering/filtering (Wave 5). Thinking accumulation (w1-e,
landed there). Any file outside the w1-c ownership list.
