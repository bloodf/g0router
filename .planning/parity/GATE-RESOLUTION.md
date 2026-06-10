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

## Addendum — w0-e disposition (2026-06-09)
Implementer: Kimi (tasks 1-4) + Fable (tasks 5-6 under plan Amendment 1 after Kimi correctly blocked on the missing error-return mechanism). Reviewer: gpt-5.5. Verdict REJECT (1 BLOCKER, 2 MAJOR):
- BLOCKER (read-error branch untested): OVERRULED — Amendment 1 pre-documented that a non-EOF read error is unreachable while `resp.Body()` is fully buffered (`bytes.Reader` only EOFs); the branch is structurally identical (`ch <- streamError(...); return`) to the tested unmarshal path; Wave 1 PAR-TRANS-046 replaces all three per-provider loops with a central stream processor where the branch becomes reachable and tested. Extracting loops now is throwaway refactoring.
- MAJOR (AUD-036 missing non-stream assertion): FIXED — test now asserts `stream` is omitted from the Gemini body for both `Stream=true` and `Stream=false` (commit 54a32e09).
- MAJOR (consistency tests "hard-code field loops"): OVERRULED — misread. The hard-coded list is the AUD-row spec; coverage is checked against the `unsupported*Fields` variables, so list drift fails the test exactly as the critic demands.
w0-e APPROVED. Wave 0 complete: w0-a through w0-e all merged.

## Addendum — AUD-004 remediation deviation (2026-06-09)
AUD-004 remediation text says "rotate exposed ID". The ID (`9d1c250a-e61b-44d9-88ed-5944d1962f5e`) is Anthropic's public Claude Code OAuth client identifier — not our credential, not rotatable by us, and not a secret (RFC 8252 §8.4: native-app client IDs are public). 9router hardcodes the same value (`_refs/9router/src/lib/oauth/constants/oauth.js:21`). Authorized remediation: make it configurable via `G0ROUTER_ANTHROPIC_CLIENT_ID` with the public ID as default (preserves out-of-box parity). Plan w0-b implements this. Decision: orchestrator, surfaced to user in the Wave 0 plan summary.

## Addendum — matrix amendment: PAR-TRANS-056/057 (2026-06-09, w1-b planning)
The gated 9router-translation.md matrix omitted two items required for `/v1/messages` to function: the route itself and the claude→openai REQUEST translator (`request/claude-to-openai.js` — the matrix covered only that file's registration counterpart directions: openai→claude request rows 012-022 and claude→openai response rows 041/042). The w1-b plan gate correctly rejected citing these behaviors against pipeline-mechanics rows (PAR-TRANS-004) as insufficient traceability. Resolution: rows PAR-TRANS-056 and PAR-TRANS-057 appended to the matrix with frozen-ref evidence, verified by orchestrator against source. Bonus finding from verification: 9router returns OpenAI-shaped JSON (untranslated) to claude clients for NON-streaming `/v1/messages` requests (`nonStreamingHandler.js:15-16`); the w1-b draft had invented a Claude-JSON synthesis step that does not exist in the reference — removed for fidelity.

## w1-a diff gate (gpt-5.5, 2026-06-09) — dispositions
- BLOCKER (NormalizeThinkingConfig keeps thinking on empty message list): OVERRULED — matches reference. `isLastMessageFromUser` (`provider.js:335-340`) returns `true` for an empty list (`if (!messages?.length) return true;`), so 9router KEEPS thinking config on empty messages. The Go early-return reproduces this exactly. The plan's one-line summary ("last message not user → cleared") was a simplification; reference behavior governs parity.
- MAJOR (hasToolResults skips remaining IDs when one is answered): OVERRULED — matches reference. `toolCallHelper.js:128-143`: insertion is all-or-nothing per assistant message gated on `!hasToolResults(nextMsg, toolCallIds)`, and `hasToolResults` returns true if ANY one id matches (`:100`). Go reproduces the same skip-all semantics.
- MINOR (package-level toolIDPattern is global state): OVERRULED as written — immutable compiled regex at package level is idiomatic Go, not mutable global state. Related real smell fixed by orchestrator: `sanitizeToolID` recompiled its regex per call; hoisted to package level (`toolIDInvalidRun`).
w1-a APPROVED with orchestrator fix commit.

## Addendum — matrix correction: PAR-TRANS-011 constants (2026-06-09, w1-c planning)
Row text said adjustMaxTokens "boosts to min 4096 when tools present". Frozen ref disagrees: `open-sse/config/runtimeConfig.js:41-42` defines `DEFAULT_MAX_TOKENS = 64000` and `DEFAULT_MIN_TOKENS = 32000`; `maxTokensHelper.js:8-26` uses those. The 4096 figure is g0router's own `defaultMaxTokens` (gap column), which the analyzer conflated into the row text. Row corrected with source citation; verified by orchestrator against frozen ref.

## w1-b diff gate (gpt-5.5, 2026-06-09) — dispositions
- BLOCKER (tasks 1/2/5 "absent from diff"): OVERRULED — stale diff base. Tasks 1-5 commits (e7d7e7fd, 9169ceb1, 01a252c7, 05d2cfd4) were pushed to origin/main mid-implementation, so the `origin/main...HEAD` diff excluded them. Files exist with full test suites (`internal/translation/formats*.go`, `registry*.go`, `claude_request*.go`).
- BLOCKER (`chunk.Error` "silently swallowed"): OVERRULED — matches the w0-e contract exactly: abort without emitting `[DONE]`, never serialize the error (same as `writeSSEStream`, `internal/api/chat.go:30-44`). `TestMessagesHandlerStreamingAbortsOnErrorChunk` asserts both no-continuation and no-leak.
- BLOCKER (`[DONE]` emission out of scope): OVERRULED — reference behavior. 9router's flush emits `data: [DONE]\n\n` for claude-source streams unconditionally (`stream.js:407-411`); Anthropic's own API has no `[DONE]`, but parity targets 9router. w1-c's processor inherits this exact behavior.
- MAJOR (messageIDFromChunk fallback "resembles deferred fixInvalidId"): OVERRULED — in-row behavior. PAR-TRANS-044's source carries this fallback itself: `openai-to-claude.js:109-113` (`if (!state.messageId || state.messageId === "chat" || state.messageId.length < 8) { state.messageId = chunk.extend_fields?.requestId || ... }`). Distinct from streamHelpers' `fixInvalidId` (chunk id rewriting), which remains w1-c scope.
- MAJOR (nondeterministic multi-tool flush order): ACCEPTED — real defect. JS Map preserves insertion order; Go map iteration is randomized. Fixed by orchestrator: sorted ascending index iteration in the finish flush + `TestClaudeResponseMultiToolFlushOrder` (10-count run green).
w1-b APPROVED with orchestrator fix commit.
