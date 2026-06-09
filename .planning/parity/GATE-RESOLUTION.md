# PARITY.md critic-gate resolution (user-authorized close-out)

5 critic cycles run (1 false REJECT from harness bug, 4 substantive). User authorized Fable surgical close-out after cycle 4 (2026-06-09).

## Cycle 4 findings — disposition

| Finding | Disposition | Action |
|---|---|---|
| BLOCKER: 101 PAR-PR rows vs "129/129 mapped" contradiction | FIXED | Added explicit accounting: 129 PRs = 101 new PAR-PR rows + 28 amendments to existing rows; both §1 and §3 now state row-vs-PR distinction |
| MAJOR: issue count 139 entries vs unique total; #1123 dup | FIXED | #1123 deduped (was in ROUTE and UI); UI section now 0 issues with cross-ref; explicit total: 138 unique (74+32+10+1+21) |
| MAJOR: Wave 0 bundles lack TDD ordering | OVERRULED | Repeat finding (raised cycles 3 and 4). §2 explicitly declares bundles are plan-factory INPUT; each bundle becomes a micro-plan with TDD ordering in A6. TDD ordering belongs in micro-plans, not the synthesis index. Critic prompt cannot express this layering; human judgment applied |
| MAJOR: counting model ambiguous (same root as BLOCKER) | FIXED | Same fix as BLOCKER row above |
| MINOR: PAR-BF-OAI-303 exceeds 91 rows | FALSE | Verified: bifrost-openai.md uses sectioned non-contiguous numbering; PAR-BF-OAI-301..304 exist at matrix lines 107-110 |

## Verdict
PARITY.md APPROVED for Stage 0 user checkpoint per user authorization. All arithmetic re-verified by Fable against source artifacts.

## Addendum — Wave 0 plan gates close-out (2026-06-09)
w0-a/b/c PASSED the gpt-5.5 gate. w0-d/w0-e closed out after 4 cycles with fixes + logged overrules:
- FIXED (w0-d): `ui/src/routeTree.gen.ts` added to ownership (real catch — generated artifact).
- FIXED (w0-e): whole-package ownership (removes conditional test-file ambiguity); reproducible grep command for Anthropic negative evidence; AUD-046 dependency on w0-a's error return stated as hard, not conditional.
- OVERRULED (w0-d, route breadth, raised 3×): AUD-076's finding text — "zero page routes for 30+ e2e specs" — is the evidence; Bundle D's "at least one route file" is a floor. Creating spec-visited shells now avoids re-planning the same files in Wave 6.
- OVERRULED (w0-d, TDD ordering of task 2/3): `npm run build` red between router mount and shell creation IS the failing state; the critic's objection is sequencing pedantry on a two-step refactor whose end state has binary checks.
- OVERRULED (w0-d, recon step 3 DTO reading): reading JSON tags is reconnaissance for correct type shapes, not scope; no backend file is modified.
Authorized by user decision ("Fable fixes findings directly, one final gate run") + escalation pattern from PARITY close-out.

## Addendum — AUD-004 remediation deviation (2026-06-09)
AUD-004 remediation text says "rotate exposed ID". The ID (`9d1c250a-e61b-44d9-88ed-5944d1962f5e`) is Anthropic's public Claude Code OAuth client identifier — not our credential, not rotatable by us, and not a secret (RFC 8252 §8.4: native-app client IDs are public). 9router hardcodes the same value (`_refs/9router/src/lib/oauth/constants/oauth.js:21`). Authorized remediation: make it configurable via `G0ROUTER_ANTHROPIC_CLIENT_ID` with the public ID as default (preserves out-of-box parity). Plan w0-b implements this. Decision: orchestrator, surfaced to user in the Wave 0 plan summary.
