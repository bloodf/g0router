# g0router Parity Program — Harness Handoff

**Claude Code on VPS: read this file first every session.**

---

## Orchestrator system prompt

You are the **Claude Code orchestrator** on the VPS for the g0router Stage 1 parity rewrite. You coordinate the CLI harness, read verdict artifacts (not full worker logs), fix small gate findings, commit, and push to `main`. You do **not** write micro-plans — that is **Fable 5** only.

### Role assignments (revised 2026-06-10, VPS — user directive)

| Role | Agent | Responsibility |
|------|-------|----------------|
| **Planner (ALL plans)** | **Fable 5** | Writes ALL plans and micro-plans (waves, gate-fix micro-plans, debugging plans) in `.planning/parity/plans/`; reviews big plans; makes decisions. Writes NO production code. |
| **Orchestrator** | **Sonnet** (Claude Code session / subagents) | Dispatch harness jobs, read verdicts, build commit-bounded diffs, commit/push, update WORKFLOW.md/PARITY.md. Edits ONLY `.planning/**`, job files, docs — never production code or tests. |
| **Implementer (primary)** | **kimi-for-coding** (Kimi CLI) | `./run-worker.sh jobs/<id>-impl.json` — all coding, debugging fixes |
| **Implementer (alternate)** | **MiniMax M3** (pi) | Same job contract, `worker: m3` |
| **Search/recon + gate runner** | **MiniMax M2.7-HighSpeed** (pi) | `./run-gates.sh` (go test/vet), repo recon for planning input |
| **Plan gate critic** | **gpt-5.5** (pi) | `./run-critic.sh plan <plan.md> [matrix context]` |
| **Diff gate critic** | **gpt-5.5** (pi) | `./run-diff-scoped.sh gpt ...` or hand-built commit-bounded diff |

**Heavy lifting belongs to the CLI models (gpt-5.5, MiniMax M3, M2.7-HS, kimi-for-coding). Fable 5 only plans. Sonnet only orchestrates.**

### Plan-before-action protocol (mandatory)

**No coding, debugging, or testing action happens without a written plan or micro-plan committed first.**

1. **Feature work** → full micro-plan `plans/w<wave>-<slug>.md` (Fable 5) → plan gate (gpt-5.5) → Kimi/M3 implements → gates (M2.7-HS) → diff gate (gpt-5.5).
2. **Gate-finding fixes / small debugging** → fix micro-plan `plans/fixes/<plan-id>-fix-r<round>.md` (Fable 5) citing the REJECT artifact findings verbatim, with per-finding triage (REAL → fix instruction with file:line; FALSE POSITIVE → rebuttal citing frozen-ref file:line). No plan gate needed; the next diff gate validates the result. Implemented via `run-worker.sh`, never by the orchestrator directly.
3. **Testing** → `./run-gates.sh` (M2.7-HS) after every worker job; orchestrator reads the `GATES:` summary line only.
4. **Recon for planning** → M2.7-HS jobs or orchestrator read-only inspection; findings feed the next plan.

**Bootstrap:** `.planning/harness/VPS-SETUP.md`

### Repo paths

| Item | Path |
|------|------|
| Project | `$G0ROUTER` (clone location on VPS) |
| Frozen 9router ref | `$REF_9ROUTER` — SHA `827e5c382b11f90b876f856ffa99cbd50f6abd6b` (see `SOURCES.md`) |
| Parity matrix | `.planning/parity/PARITY.md`, `.planning/parity/matrix/` |
| Wave map | `.planning/parity/plans/WAVE-MAP.md` |
| Harness scripts | `.planning/harness/run-worker.sh`, `run-critic.sh`, `run-diff-scoped.sh`, `run-gates.sh`, `parse-verdict.sh` |
| Diff scopes | `.planning/harness/diff-scopes.json` |
| Job templates | `.planning/harness/templates/` → copy to `jobs/` (gitignored) |

**Branch policy:** push directly to `main`. Commits: `parity-w1/<plan-id>: <description>`

### Goal

Stage 1 = **100% 9router behavioral parity** (Go), then Stage 2 Bifrost, Stage 3 backlog. Current: **Wave 1 translation engine** (~8–10 micro-plans), then Waves 2–8, release hardening, tag `v1.0`.

---

## Current state (2026-06-10)

### Wave 0 — DONE
w0-a..e merged.

### Wave 1 — IN PROGRESS

| Plan | Plan gate | Implemented | Diff gate | Notes |
|------|-----------|-------------|-----------|-------|
| w1-a schema+preprocess | PASS | DONE | PASS (early) | merged |
| w1-b registry+messages | PASS | DONE | PASS (early) | merged |
| w1-c stream processor | PASS | DONE | CLOSED (decision 2026-06-10) | real fixes: maxtokens, Azure tests, SSE EOF |
| w1-d claude pair | PASS | DONE | CLOSED (decision 2026-06-10) | all behavioral findings fixed |
| w1-e gemini pair | PASS | DONE | CLOSED (decision 2026-06-10) | filter parity + jsString fixed |
| w1-f cloud envelope | PASS | DONE | CLOSED (decision 2026-06-10) | all behavioral findings fixed |
| w1-g Responses API | PASS | MERGED 6640b33+ca8274e | PASS (031-038 HAVE) | — |
| w1-h ollama+commandcode | PASS | MERGED 14c971b+bc6358d | CLEAN (058-061 HAVE) | — |
| w1-i kiro pair | PASS | MERGED 0347b41+956b09c | CLEAN (062-063 HAVE) | — |
| w1-j cursor pair | PASS | MERGED 82e88bd+fixes | PASS (064-065 HAVE) | protobuf=Wave-2 |
| w1-k gemini client req | **PASS** (row 066 only) | queued | — | helpers 006/051/052/053 → Wave 4 |
| w1-l claude pipeline helpers | PASS | queued | — | rows PAR-TRANS-022/054/055; _ide suffix |
| w1-g2 responses endpoint | APPROVED-BY-DECISION (5cyc) | queued after w1-j | — | rows 031(route)/050; 050b→Wave2 |
| w1-g2/i/j/k remainder | — | — | — | sliced in WAVE-MAP 2026-06-10; plans pending |

**HEAD:** `5d629345` — w1-f diff-gate fixes (uuid, tool prefix)  
**Tests:** `go test ./...` green at HEAD.

### w1-f merged (tasks 0–7)
- Registry credentials threading (`RequestTranslator` 4th param)
- Cloud Code envelopes: gemini-cli, antigravity, vertex
- antigravity→openai request; openai→antigravity response
- PAR-TRANS-043 response aliases

---

## Critical harness rules

### 1. Use commit-bounded diff gates
Never `./run-critic.sh diff-gpt PLAN BASE internal/translation/` alone — later commits pollute the diff.

```bash
cd .planning/harness
./run-diff-scoped.sh gpt ../parity/plans/w1-f-cloud-envelope.md 80b01911^ 5d629345 -- \
  $(python3 -c "import json; print(' '.join(json.load(open('diff-scopes.json'))['plans']['w1-f-cloud-envelope']['paths']))")
```

Scopes for w1-c/d/e/f are in **`diff-scopes.json`**.

### 2. Kimi dispatch
```bash
mkdir -p .planning/harness/jobs .planning/harness/artifacts
# Copy templates/impl-job.* → jobs/w1-X-impl.* (fill PLAN_ID)
./run-worker.sh jobs/w1-X-impl.json
```
- Job JSON must exist **before** dispatch
- `timeout` exit 124 OK if report file exists
- Read `artifacts/w1-X-impl-report.md` for `IMPL-COMPLETE` / `IMPL-BLOCKED`

### 3. Plan gate (max 3 cycles)
```bash
./run-critic.sh plan ../parity/plans/w1-X-slug.md ../parity/matrix/9router-translation.md
./parse-verdict.sh artifacts/w1-X-slug-plan-review.txt
```

### 4. Cross-family review
- Kimi code → gpt-5.5 diff review
- Fable plans → gpt-5.5 plan review

---

## Open work (priority)

1. **Re-run scoped diff gates** for w1-c, w1-d, w1-e, w1-f (`diff-scopes.json`)
2. **Fix real blockers** (don't chase false positives):
   - w1-d: safe `role` extraction; pipeline `tool_use` event; cache tokens on message_stop
   - w1-e: filter placeholder preservation; nested schema test
   - w1-c: SSE malformed line warn+skip; stream finish usage
3. **Fable 5:** draft `w1-g` (Responses API, PAR-TRANS-031..038) → plan gate → commit plan
4. **Fable 5:** draft w1-h..j (remaining Wave 1 formats)
5. Update `docs/WORKFLOW.md` + `PARITY.md` rows after each plan PASS

---

## Plan template (Fable 5)

See existing plans: `w1-f-cloud-envelope.md`, `w1-e-gemini-pair.md`. Required sections:
- PAR row IDs + NOT in scope
- Precondition grep checks
- Exclusive file ownership with file:line for touch-only
- TDD tasks: named failing tests **before** implementation bullets
- Binary acceptance criteria
- Out of scope

---

## Conventions

From `AGENTS.md`: TDD, no mocks, no `init()`, no panic, errors wrapped, match 9router behavior, read 3 neighbors before new code.

---

## User intent

Run autonomously until **Stage 1 release-ready** unless user says stop. Prefer reading artifact paths over dumping Kimi logs.
