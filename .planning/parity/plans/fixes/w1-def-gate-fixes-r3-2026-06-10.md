# Fix micro-plan — w1-d / w1-e / w1-f diff-gate findings, round 3 (2026-06-10)

Author: Fable 5 (planner). Status: RETROACTIVE — documents the in-flight fix batch
authored before the plan-before-action protocol was adopted mid-session; committed
together with the changes it governs. All future fix batches get their plan
committed BEFORE implementation, and implementation goes to kimi/M3 via
`run-worker.sh` — never the orchestrator.

Verdict artifacts triaged (gpt-5.5, commit-bounded diffs):
- `artifacts/w1-d-claude-pair-diff-scoped-gpt.txt` (round 4)
- `artifacts/w1-e-gemini-pair-diff-scoped-gpt.txt` (round 2)
- `artifacts/w1-f-cloud-envelope-diff-scoped-gpt.txt` (round 5)

## w1-d findings

| # | Finding | Triage | Action |
|---|---------|--------|--------|
| 1 | BLOCKER: tests call `openaiToClaudeRequest` with 3 args; "will not compile" | **FALSE POSITIVE** — artifact of the split commit-bounded diff: the base range predates the w1-f credentials param on shared files; `go test ./...` is green at HEAD | Rebut in gate prompt; no code change |
| 2 | MAJOR: JSON-schema response-format test only checks substrings | REAL (test strength) | `openai_claude_request_test.go`: assert the exact verbatim prompt text incl. fenced 2-space pretty-printed schema, per ref `openai-to-claude.js:112-117` |
| 3 | MINOR: `getContentBlocksFromMessage` never uses `toolNameMap` | **FALSE POSITIVE vs parity** — frozen ref has the identical unused param (`openai-to-claude.js:210`); ref's `CLAUDE_OAUTH_TOOL_PREFIX` is `""` so names pass uncloaked | Add ref-citing comment above the function; no behavior change |

## w1-e findings

| # | Finding | Triage | Action |
|---|---------|--------|--------|
| 1 | BLOCKER: keyword list has 10 UI styling keys, plan requires 11 | **FALSE POSITIVE / plan typo** — normative source `geminiHelper.js:21-22` lists exactly 10; Go matches byte-for-byte | Plan amendment in `w1-e-gemini-pair.md` correcting "11" → "10" with the enumerated list (Fable 5 amendment) |
| 2 | MAJOR: all-filtered content placeholder inserted then dropped; "not preserved or tested" | **MATCHES REF** — ref inserts `{type:text,text:""}` (openaiHelper.js:49-51) and its second pass drops the message (60-74); end state is the message disappearing. Not a defect; was untested | Add `TestFilterAllThinkingMessageDropped` documenting the ref-exact end state |

## w1-f findings

| # | Finding | Triage | Action |
|---|---------|--------|--------|
| 1 | BLOCKER: accumulated tool call `id` not emitted in `functionCall` | **FALSE POSITIVE vs parity** — ref emits only `name`+`args` (`openai-to-antigravity.js:64-69`) | Add ref-citing comment at the emission site; no behavior change |
| 2 | MAJOR: alias test only checks non-nil for antigravity→openai | REAL (test strength) | Add `FormatAntigravity` to the identity loop in `TestResponseAliasesUseGeminiTranslator` (reflect pointer equality vs `geminiToOpenAIResponse`) |
| 3 | MINOR: padded obvious comment `// Generation config` | REAL (style) | Remove the comment in `antigravity_openai_request.go` |

## Acceptance

- `go test ./...` and `go vet ./...` green (verified before commit).
- Each plan's next scoped diff gate (gpt-5.5) runs over base range + own fix commits only,
  with cross-scope fix commits excluded and the false-positive rebuttals stated in the
  gate prompt NOTE.

## Out of scope

- w1-c findings (separate fix micro-plan, implemented by worker per new protocol).
- Any change to `internal/translation` behavior beyond the items listed above.
