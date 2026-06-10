# CLI Harness — job contract

**VPS orchestrator:** Claude Code — read `HANDOFF.md` and `VPS-SETUP.md` first.  
**Planner:** Fable 5 only (plans in `.planning/parity/plans/`). Claude Code does not write plans.

Cursor/local dev may also orchestrate; same harness rules apply.

## Pinned models (verified 2026-06-09, all smoke-tested)

| Alias | Invocation | Role |
|-------|-----------|------|
| `kimi` | `timeout -k 5 <T> kimi -p "<prompt>" </dev/null` | Implementer (all code) |
| `m3` | `pi -p --provider minimax --model MiniMax-M3 --no-session "<prompt>"` | Implementer (legacy; Kimi preferred) |
| `m27-hs` | `pi -p --provider minimax --model MiniMax-M2.7-highspeed --no-session -t read,bash "<prompt>"` | Search/recon + gate runner |
| `gpt-5.5` | `pi -p --provider openai-codex --model gpt-5.5 --no-session -nt "<prompt>"` | Plan-gate + diff-gate critic (no tools) |
| `fable-5` | User/Fable session → commit plan `.md` files | **Sole plan author** |

### Known quirks (do not re-discover these)

- `kimi`: `-y`/`--auto` CANNOT combine with `-p`. Prompt mode runs tools without approval anyway (file writes verified). Process **lingers after completing** — always wrap in `timeout`; exit 124 is NOT failure. Success = expected output files exist + completion text in log.
- `pi`: default model config is stale (`zai/glm-5.1` warning is noise). ALWAYS pass `--provider` + `--model` explicitly. Always pass `--no-session`. Output ends with terminal control sequences — strip or grep around them.
- `gh`: stale `GH_TOKEN` env breaks auth. Use `env -u GH_TOKEN -u GITHUB_TOKEN gh ...`.

## Scripts

| Script | Purpose |
|--------|---------|
| `run-worker.sh <job.json>` | Dispatch worker job (kimi or pi+model per job spec), tee to artifacts/ |
| `run-critic.sh plan\|diff\|diff-gpt ...` | Plan gate (gpt-5.5) or diff gate; emits `VERDICT: PASS\|REJECT` |
| `run-diff-scoped.sh gpt\|kimi <plan> <base> <end> [-- paths]` | **Preferred** commit-bounded diff gate |
| `run-gates.sh [label]` | M2.7-HS runs go test/vet (+ ui build if `ui` arg); emits `GATES: PASS\|FAIL` summary |
| `parse-verdict.sh <artifact>` | Extract verdict line; exit 0 PASS, 1 REJECT/FAIL, 2 missing |
| `diff-scopes.json` | BASE/END/paths for w1-c..f scoped diff gates |
| `templates/` | Job JSON/prompt templates → copy to gitignored `jobs/` |

## Job JSON schema

```json
{
  "id": "a1-translation",
  "worker": "kimi",
  "model": null,
  "timeout": 3600,
  "prompt_file": ".planning/harness/jobs/a1-translation.prompt.md",
  "expected_outputs": [".planning/parity/matrix/9router-translation.md"]
}
```

`worker` ∈ {`kimi`,`m3`,`m27-hs`,`gpt-5.5`}. `model` resolved from alias table above.

## Verdict contract (critics)

```
VERDICT: PASS|REJECT
FINDINGS:
- [BLOCKER|MAJOR|MINOR] <finding>
COUNTERARGUMENT: <strongest case against this plan/diff>
```

Gate runner contract: first line `GATES: PASS` or `GATES: FAIL` + failing package/test + key error line, max 20 lines.

## Rules

- Plans authored ONLY by **Fable 5** (committed to `.planning/parity/plans/`). Claude Code orchestrates; external CLIs implement/review.
- Cross-family review: Kimi code → gpt-5.5 diff (`run-diff-scoped.sh gpt`); Fable plans → gpt-5.5 plan gate.
- Use **commit-bounded** diffs for diff gates (`run-diff-scoped.sh` + `diff-scopes.json`), not `BASE...HEAD` on broad paths.
- Reject loop max 3 cycles, then escalate to user.
- Every implementation task cites a PARITY row ID.
