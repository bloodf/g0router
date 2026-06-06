# Risk Register — Phase 12B

- Silent behavior change during code move → snapshot route table before task-1; tests re-pointed, never rewritten.
- task-4 (inference pipeline) is highest blast radius → do it last; lean on 48 KB integration suite as the net.
- Coverage drops below 95.0% as logic moves between packages → run coverage gate per task, not just at checkpoint.
- Race exposure shifts when state moves packages → `go test -race ./...` per task.
- Repository-interface churn breaks a consumer's compile → interfaces defined in consumer pkg; `*store.Store` satisfies implicitly, no store edits.
- Task balloons past ~1 day → stop, commit green state, split the task.
- `git mv` skipped → history lost; prefer `git mv` for all file moves.
- Scope creep ("while I'm here") → enforce explicit non-goals; no renames/features/test deletion.
