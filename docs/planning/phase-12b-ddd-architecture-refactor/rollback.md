# Rollback — Phase 12B

- Pushes go directly to main; each task is its own commit (`phase-12b/task-M: ...`).
- Undo a single task: `git revert <task-commit-sha>` then `go test ./... -count=1 && go vet ./... && go build ./cmd/g0router`.
- Undo the whole phase: `git revert --no-commit <first>^..<last>` over the phase commit range, then run the per-commit gate before pushing.
- No DB migrations are introduced or altered in this phase — nothing to reverse at the schema layer; `internal/store` is untouched (task-2 only adds consumer-side interfaces).
- Reverts are pure code moves; reverting restores the prior package layout with no data-state implications.
- In-progress red state > 30 min → reset to last green commit rather than reverting individual edits.
