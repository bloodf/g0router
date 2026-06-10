Read and obey {{REPO_ROOT}}/.planning/harness/prompts/implementer-base.md

Plan to implement: {{REPO_ROOT}}/.planning/parity/plans/{{PLAN_FILE}}
Repo root: {{REPO_ROOT}}
Report file: {{REPO_ROOT}}/.planning/harness/artifacts/{{JOB_ID}}-report.md

Commit message prefix for this plan: `parity-w1/{{PLAN_ID}}: <task summary>`.
The 9router reference source lives at {{REF_9ROUTER}} — read the cited files before porting behavior.
Run the plan's precondition check first; if it fails, write IMPL-BLOCKED to the report file and stop.
