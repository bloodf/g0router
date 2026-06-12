# CLI Orchestrator Protocol

A generic, reusable protocol for running Claude Code, oh-my-claudecode (OMC),
Headroom, claude-mem, and Pi Code as one autonomous development pipeline. The
protocol is harness-agnostic at its core: any orchestrator that can (a) write
plans, (b) dispatch CLI workers, (c) run review gates, and (d) persist memory
can implement it. Tool-specific sections name the concrete commands for this
stack.

Precedence in this repository: the 9router parity program (Waves 0–5) ran under
`.planning/harness/HANDOFF.md`; that document remains the historical record for
those waves. From Wave 6 onward, and for all non-parity work, THIS protocol
governs. Where both apply, this protocol wins.

---

## 1. Roles and model routing

| Role | Model / tool | Responsibility |
|---|---|---|
| Runtime orchestrator | **Sonnet** (Claude Code session) | Default future orchestrator: dispatches workers, reads verdict artifacts, reconciles gates, commits/pushes, updates docs. Writes no production code, no plans. |
| Planning lead | **Fable 5** | Writes ALL plans and micro-plans; owns autonomous decision-making (see §2); writes acceptance criteria, fix plans, dispositions. Writes no production code. |
| Antagonist reviewer | **GPT-5.5** (`pi --provider openai-codex --model gpt-5.5`) | Adversarial review of plans and diffs: gaps, bugs, weak assumptions, missing criteria, risky shortcuts. |
| Primary implementer | **MiniMax M3 high-thinking** (`pi --provider minimax --model MiniMax-M3`) via Pi Code | Implements exactly one approved micro-plan at a time. |
| Deep code reviewer | **Kimi-for-coding** (`kimi` CLI) | Reviews M3 implementation output with full source context. |
| Fast CLI operator | **M2.7 Highspeed** (`pi --provider minimax --model MiniMax-M2.7-highspeed`) | Validation commands, commit messages, commits, pushes, main/origin integration. |
| Investigation workers | **Claude Code subagents** (Anthropic models) | Focused research, planning support, verification, in-session orchestration. |
| Context layer | **Headroom** | Compression, reversible retrieval, SharedContext handoffs, failure learning. |
| Memory layer | **claude-mem** | Persistent cross-session memory, timeline reconstruction, durable observations. |

Routing is per-task and explicit in every plan. Substituting a model for a role
is a plan-level decision by Fable 5, recorded with rationale.

## 2. Autonomy rule

Work autonomously. Do not require human babysitting.

When a decision is needed:

1. Fable 5 first derives the best decision from project docs, existing code
   patterns, prior memory (claude-mem), `AGENTS.md`, `CLAUDE.md`, and the
   safest low-blast-radius default.
2. If a reasonable default exists, choose it, document why (in the plan or its
   disposition), and continue.
3. Ask the human ONLY when no safe default exists AND the decision would cause
   irreversible state, destructive action, credential use, schema change,
   production deploy, force push, or a major architectural commitment.
4. If asking is required, ask ONE specific bounded question with a recommended
   default.

Never pause for routine naming, style, file placement, implementation approach,
review-loop, validation, or CLI execution decisions. Decide and continue.

## 3. No-leftovers rule

Agents and CLIs never leave unfinished work behind. Forbidden in code, tests,
docs, and configs:

- TODO comments, FIXME comments
- placeholder implementations, stubs, no-op fallbacks
- skipped tests, "implement later" notes
- temporary hacks, dead files created for later use
- partial wiring presented as complete

(Wave-5 evidence for why this rule exists: `NewWithShutdown` shipped with zero
production callers and `UsageEntry.APIKey` shipped unpopulated — both correctly
built, both inert. Dead wiring passes tests; only gates and this rule catch it.)

When an agent discovers required-but-unplanned work:

1. STOP that worker.
2. Return an escalation to the orchestrator containing: the missing work, why
   it is required, affected files, risk level, proposed acceptance criteria.
3. The orchestrator routes the escalation to Fable 5 for a full plan or
   micro-plan.
4. No Pi Code dispatch resumes until the new plan exists and passes review.

If external information is truly unavailable, state the exact blocker. Never
paper over it with a placeholder.

## 4. Behavioral rules

Apply to every plan, micro-plan, dispatch, review, validation, and shipping
step.

**Think before coding.** No silent assumptions — surface them explicitly
before implementation. If multiple interpretations exist, pick the safest and
note it, unless the Autonomy rule's ask-condition triggers. If a simpler
approach exists, choose it. Push back on risky or overcomplicated requests. If
ambiguity affects an irreversible action, stop and ask.

**Simplicity first.** Minimum code or documentation that solves the task. No
features beyond the request, no abstractions for single-use code, no
speculative flexibility, no error handling for impossible scenarios. If the
solution grows large, simplify before continuing.

**Surgical changes.** Touch only files the task requires. No improving adjacent
code, comments, or formatting. No unrelated refactors. Match existing style.
Mention unrelated dead code or problems; do not delete them. Remove only
imports/variables/functions/docs/config made obsolete by your own change.
Every changed line traces directly to the task.

**Goal-driven execution.** Define success criteria and verification BEFORE
execution. For multi-step work, write the plan as:

```
1. [Step] -> verify: [check]
2. [Step] -> verify: [check]
3. [Step] -> verify: [check]
```

Loop until verified. Never claim success without evidence (command output, test
results, grep proofs).

## 5. Non-negotiable Pi Code rule

Before ANY Pi Code dispatch, a written plan must exist:

- **Full plan** for multi-step or cross-file work.
- **Micro-plan** for a single bounded task.

No Pi Code worker may code, research, review, validate, commit, push, or run
CLI work without one. Every plan includes: scope, target files, model, allowed
commands, acceptance criteria, validation, and stop conditions. See §8 and §9
for the required plan contents.

## 6. Required preflight

Before writing docs or dispatching workers, verify (do not assume) each item.
If any required tool is missing, STOP and output exact setup steps. Never fake
availability.

| Item | Verification | Status in this repo (2026-06-12) |
|---|---|---|
| Claude Code | running session | OK |
| oh-my-claudecode | `omc --version` | OK (4.14.6) |
| Headroom CLI | `headroom --version` | OK (0.25.0) |
| Headroom MCP tools | `headroom_compress/retrieve/stats` resolvable in-session | **NOT CONNECTED** — register the Headroom MCP server in Claude Code (`claude mcp add headroom -- headroom mcp` or per Headroom docs) before relying on MCP-path retrieval; until then use the CLI path (§7) |
| claude-mem | plugin active; `mcp-search` tools resolvable; UI on :37700 | OK |
| Pi Code / Pi CLI | `pi --version`; models listed | OK (0.79.1) |
| Kimi CLI | `kimi --version` | OK (0.11.0) |
| stop-slop skill (global) | present in session skills list | OK |
| prompt-master skill (global) | present in session skills list | OK |
| Model aliases | `pi models` shows `gpt-5.5`, `MiniMax-M3`, `MiniMax-M2.7-highspeed`; `kimi` on PATH; Fable 5/Sonnet native to Claude Code | OK |

stop-slop and prompt-master must be present before orchestration starts:
prompt-master is used when authoring worker prompts for non-Claude CLIs;
stop-slop applies to any human-facing prose the pipeline produces.

## 7. Headroom usage

Claude Code runs under Headroom. Use it as the context compression and
retrieval layer:

- Prefer `headroom wrap claude` to launch Claude Code sessions.
- Use Headroom MCP tools when available: `headroom_compress`,
  `headroom_retrieve`, `headroom_stats`. (See §6 for the current MCP gap and
  enablement step.)
- Treat compressed context as a POINTER, not proof. Retrieve originals before
  making exact claims about code, docs, commands, or review findings.
  (Wave-5 evidence: a shell-level output compressor made a 117KB diff read as
  21KB, masking a real argv-limit failure — compressed views lie about exact
  sizes and content.)
- Use Headroom memory and SharedContext for cross-agent handoffs (orchestrator
  → worker context briefs).
- Run `headroom learn` after failed or corrected sessions to write durable
  corrections into `CLAUDE.md` or `AGENTS.md`.
- Telemetry: this project has no telemetry restriction on record; if one is
  adopted, disable or document Headroom telemetry at preflight.

## 8. claude-mem usage

claude-mem is the persistent memory layer. Verify it is installed and active
at preflight (§6).

Retrieval flow (in order):

1. Search the compact memory index (`memory_search` / `smart_search`).
2. Inspect the timeline around relevant hits (`timeline`).
3. Fetch full observations ONLY for selected IDs (`get_observations`).

Rules:

- Search claude-mem BEFORE planning whenever prior work may exist.
- Store stable cross-session findings: final decisions, validation evidence,
  recurring failures and their fixes, lessons learned.
- Never store secrets. Use private exclusion tags for sensitive content.

## 9. Required pipeline

### 9.1 Memory and context preflight

- Query claude-mem for prior decisions, failures, project conventions, known
  pitfalls (flow in §8).
- Use Headroom to compress large retrieved context; retrieve exact originals
  before acting on quoted files, commands, or review findings.
- Produce a short context brief with citations or observation IDs when
  available. The brief is the dispatch context for downstream workers.

### 9.2 Planning loop

1. Fable 5 creates the full plan, resolving decisions autonomously where safe
   (§2).
2. GPT-5.5 reviews the plan antagonistically.
3. Fable 5 revises.
4. The Claude Code orchestrator checks the plan is actionable.
5. Repeat until no blocking plan findings remain. (Convergence guard: after 3
   substantive review cycles, Fable 5 may close by decision with per-finding
   triage — REAL findings fixed, false positives rebutted with file:line
   evidence — appended to the plan. Review gates can be non-convergent;
   ground truth, not the reviewer, is binding.)

Every full plan includes: Objective; Assumptions; Decisions made by Fable 5
and why; Ambiguities requiring human input (only if no safe default exists);
Scope; Non-goals; Task graph; Per-task model routing; Required micro-plans
before Pi Code dispatch; Acceptance criteria; Validation commands; Review
gates; Stop conditions; Rollback notes; No-leftovers enforcement.

### 9.3 Micro-plan gate before Pi Code

Before EVERY Pi Code dispatch, a micro-plan exists containing:

- Task ID
- Assumptions
- Fable 5 decision and rationale
- Ambiguities, or confirmation that none block execution
- Model to run
- Exact prompt to send
- Files/directories in scope
- Files/directories forbidden (workers NEVER `git checkout`/`restore`/`stash`
  any unowned path, even "temporarily for verification" — a Wave-5 worker did
  exactly that and destroyed a concurrent worker's in-progress files)
- Allowed commands
- Expected output artifact
- Acceptance criteria
- Validation command
- Stop conditions
- Context handoff source (Headroom SharedContext or claude-mem observation IDs)
- Confirmation that no TODOs, placeholders, skipped tests, or deferred work
  are allowed

Pi Code dispatch is forbidden until this micro-plan exists.

### 9.4 Implementation loop

1. MiniMax M3 high-thinking implements exactly ONE approved micro-plan through
   Pi Code.
2. If M3 finds required unplanned work, M3 stops and escalates (§3).
3. Kimi-for-coding reviews the implementation.
4. GPT-5.5 performs adversarial code review (commit-bounded diff, exact file
   scope — path globs leak concurrent work into the diff and produce false
   findings).
5. The Claude Code orchestrator verifies the implementation matches the
   approved plan and contains no leftovers.
6. Any blocking issue → Fable 5 writes a fix micro-plan → M3 executes it
   through Pi Code.
7. Repeat until Kimi, GPT-5.5, and Claude Code all pass (or the convergence
   guard in §9.2 step 5 applies, with live-tree verification before closure).

### 9.5 Validation loop

- M2.7 Highspeed runs validation through Pi Code: available lint, typecheck,
  unit tests, integration tests, build, and smoke checks.
- Use Headroom for long logs; retrieve exact failing sections before
  diagnosing.
- Store recurring failures and their fixes in claude-mem.
- Validation failure → fix micro-plan BEFORE any further Pi Code dispatch.
- Validation revealing unplanned required work → escalate (§3), route through
  Fable 5.

### 9.6 Shipping loop

- M2.7 Highspeed drafts the commit message through Pi Code.
- Claude Code reviews the final diff before commit.
- M2.7 Highspeed commits and pushes ONLY after validation passes.
- STOP before: destructive git operations, force pushes, schema migrations,
  production deploys, credential changes.
- Record final decisions, validation evidence, and lessons in claude-mem.
- Run Headroom failure learning (`headroom learn`) if the session had
  corrections or failed loops.

## 10. oh-my-claudecode usage rules

- `/team` — in-session Claude Code staged orchestration.
- `/ralplan` — high-risk planning consensus.
- `/ask` or `omc ask` — advisor artifacts.
- `/ccg` — synthesis of multiple model perspectives.
- `/ultraqa` — repeated quality-gate loops.
- `/ralph` — persistent verified completion.
- `/skill` — inspect installed skills.
- `/skillify` — extract reusable workflows from a session.
- Do NOT use `omc team` when background-process-only workers are required:
  `omc team` uses tmux CLI panes. Prefer Claude Code background agents, Pi
  Code background jobs, or OMC in-session skills when tmux is disallowed.

## 11. Claude Code usage rules

- Durable shared instructions live in `AGENTS.md`.
- Keep `CLAUDE.md` a thin import where possible: `@AGENTS.md`, with
  Claude-specific rules below the import only when necessary. (This repo
  already follows that shape.)
- `.claude/agents/` — project subagents. `~/.claude/agents/` — user-wide
  subagents.
- `.claude/rules/` — path-scoped rules (this repo does not currently use it;
  add it only when a path-scoped rule is actually needed).
- Skills for repeatable workflows; subagents for focused research, planning,
  review, and validation; background agents for independent full-session work.
- Agent teams only when enabled and worker communication is needed; clean up
  teams through the lead session only.
- Require plan approval before risky implementation.

## 12. Stop conditions

Stop and ask before:

- deleting any file
- adding a dependency
- changing package manager config
- modifying database schema
- force pushing
- deploying
- touching files outside the task's declared scope
- embedding credentials
- using tmux when background-process-only execution is required
- dispatching Pi Code without a plan or micro-plan
- proceeding when no safe autonomous decision exists and the outcome is
  irreversible
