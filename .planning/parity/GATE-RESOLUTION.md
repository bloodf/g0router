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

## Addendum — w0-a execution deviations (2026-06-09)
- **Implementer switch**: M3 (pi/MiniMax) died silently 3× mid-plan (tasks 4-6, ~30-40 min sessions, exit 1, no log). Tasks 1-4 salvaged from M3 commits. Tasks 5-6 implemented by Fable (orchestrator) under the user's "continue, don't stop" directive. Kimi diff gate still applies (cross-family for both authors).
- **Task 5 seam**: fasthttp's in-memory `RequestCtx` response buffer never returns write errors, so the plan's "fake fasthttp writer" is not constructible against `*fasthttp.RequestCtx` directly. The SSE loop was extracted verbatim into `writeSSEStream(w streamWriter, ch)` where `streamWriter` is the 2-method subset of `RequestCtx` the loop uses. No behavior change; enables AUD-008 write-failure injection.
- **Task 6 mechanism**: provider stream loops run inside goroutines feeding `chan *schemas.StreamChunk`; a literal `return fmt.Errorf(...)` has no receiver. Implemented Bundle A's acceptance ("provider stream aborts on JSON unmarshal error") as goroutine return + channel close. Caller-visible error propagation is w0-e scope (AUD-046) and will ride the mechanism w0-e introduces.
- **Acceptance grep 2 over-broad**: `grep -rn "rand.Read" internal/store internal/auth` still matches `secret.go:33`, `crypto.go:36`, `password.go:24`. All three already check and wrap the error (`if _, err := rand.Read(...)`) and two are outside w0-a file ownership. AUD-001/002/003 named only `newID`/`newToken`/`randomURLSafe` (unchecked sites). Refined check: same grep filtered to unchecked calls → empty. No code change needed.

## Addendum — w0-a diff gate disposition (2026-06-09)
Kimi verdict: PASS, 1 MAJOR + 4 MINOR.
- MAJOR (bare `return` vs `return fmt.Errorf` in provider goroutines): OVERRULED — pre-logged deviation above; the chunk channel cannot carry an error; caller-visible propagation is w0-e/AUD-046 scope.
- MINOR double-wrapped "generate token" context: FIXED (newToken now wraps as "read random bytes").
- MINOR unchecked `[DONE]` write: FIXED.
- MINOR separate test funcs vs table-driven: OVERRULED — style preference; tests are binary and per-handler failure isolation is clearer.
- MINOR acceptance grep non-empty: pre-logged above (grep over-broad; remaining sites already check errors).

## Addendum — w0-d diff gate disposition (2026-06-09)
Kimi verdict: REJECT, single MAJOR: commits use `parity-w0/w0-d:` instead of AGENTS.md `phase-N/task-M:`. OVERRULED — false positive. `prompts/implementer-base.md` (binding for the parity program) mandates `parity-w0/<plan-id>: <task summary>`; the parity-wave prefix supersedes phase numbering for Stage 1 work. The critic was not given implementer-base.md as context. Zero functional findings ("everything looks correct functionally"). w0-d APPROVED. Follow-up: critic-diff prompt amended to include the commit-format rule.

## Addendum — w0-b diff gate disposition (2026-06-09)
Implementer: Kimi (M3 retired after 3 silent deaths). Reviewer: gpt-5.5 (`diff-gpt` mode added to run-critic.sh; cross-family preserved). Two runs.
Run 1 (REJECT, 7 BLOCKER + 2 MAJOR): 5 blockers were orchestrator working-tree contamination — Fable left harness/docs files uncommitted and Kimi's first commit swept them in (.DS_Store, parse-verdict.sh, implementer-base.md, GATE-RESOLUTION.md, WORKFLOW.md, ui/dist). Process fix: orchestrator commits its own files before dispatching implementers. Remaining 2 blockers (admin/oauth.go, admin/providers.go outside ownership): OVERRULED — plan-authoring defect, not implementation defect; AUD-013 changes `pathID` to `(string, bool)` and ALL callers must handle the new signature; those callers live in oauth.go/providers.go. Changes are minimal call-site updates.
Run 2 (path-filtered to cmd/ internal/; REJECT, 1 BLOCKER + 2 MAJOR):
- BLOCKER "FK rebuild misses schema relationships": FACTUALLY WRONG. Schema has exactly two FK relationships (sessions.user_id→users.id RESTRICT, connections.provider_id→providers.id CASCADE) — both implemented. `oauth_sessions.provider` is a provider-name string, not a foreign key. users/providers/settings have no FK columns.
- MAJOR "unversioned rebuild": OVERRULED — repo has no migration-version table; idempotency via `PRAGMA foreign_key_list` state inspection matches the existing `ensureColumn` state-inspection pattern. Introducing version infrastructure would exceed plan scope.
- MAJOR "env read every call": OVERRULED — `AnthropicOAuth()` has exactly one production call site, in server route construction at startup; env is read once at construction as the task requires.
w0-b APPROVED.

## Addendum — w0-c diff gate disposition (2026-06-09)
Implementer: Kimi. Reviewer: gpt-5.5. Verdict REJECT (1 BLOCKER, 1 MAJOR, 1 MINOR).
- BLOCKER (exported `ConvertChatRequest` discarded the error from the new error-returning variant): VALID — real catch. FIXED by Fable: the wrapper is gone; `ConvertChatRequest` itself returns `(*GenerateContentRequest, error)`; all callers (chat.go ×2, tests) updated. Commit 8b8c7f6.
- MAJOR (tests target unexported variant): FIXED by the same rename — tests now exercise the exported API.
- MINOR (single-case vs table-driven test style): OVERRULED — style preference; assertions are binary.
w0-c APPROVED after fix.

## Addendum — AUD-004 remediation deviation (2026-06-09)
AUD-004 remediation text says "rotate exposed ID". The ID (`9d1c250a-e61b-44d9-88ed-5944d1962f5e`) is Anthropic's public Claude Code OAuth client identifier — not our credential, not rotatable by us, and not a secret (RFC 8252 §8.4: native-app client IDs are public). 9router hardcodes the same value (`_refs/9router/src/lib/oauth/constants/oauth.js:21`). Authorized remediation: make it configurable via `G0ROUTER_ANTHROPIC_CLIENT_ID` with the public ID as default (preserves out-of-box parity). Plan w0-b implements this. Decision: orchestrator, surfaced to user in the Wave 0 plan summary.
