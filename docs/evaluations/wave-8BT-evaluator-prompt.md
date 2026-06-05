# g0router Wave 8.BT Evaluation

Evaluate completed wave `8.BT` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `internal/providers/gitlabduo/gitlabduo.go`
- `internal/providers/gitlabduo/gitlabduo_test.go`
- Diff/commits for Wave 8.BT

Check:
- The Wave 8.BS evaluator's non-blocking mutable-map finding is resolved.
- GitLab Duo model aliases are no longer stored in a package-level mutable map.
- `mappedRequest` still maps supported Duo aliases to the same upstream OpenAI/Anthropic models.
- Unsupported Duo aliases still fail closed.
- `ListModels` still returns deterministic Duo aliases.
- No GitLab Duo direct-access, header-forwarding, or provider-registration behavior changed.
- No unrelated files or providers changed.

Required gates:
- `go test ./internal/providers/gitlabduo -run 'Test(MappedRequestUsesFixedAliasTable|ListModelsReturnsDuoAliasesDeterministically)' -count=1`
- `go test ./... -count=1`
- `go vet ./...`
- `go build ./cmd/g0router`
- `npm --prefix ui test -- --run`
- `npm --prefix ui run build`
- `npm --prefix ui run e2e`
- `make build`
- `git diff --check`

Return:

## Verdict

PASS or FAIL

## Blocking Findings

Issues that must be fixed before advancing.

## Non-Blocking Findings

Risks or cleanup notes.

## Gate Results

Command results with exact failures if any.

## Workflow Status Review

Whether `docs/WORKFLOW.md` is accurate.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
