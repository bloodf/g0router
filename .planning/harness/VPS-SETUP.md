# VPS bootstrap — g0router parity harness

Run once on a fresh VPS before Claude Code takes over orchestration.

## 1. Clone repos

```bash
export G0ROUTER="$HOME/g0router"          # or your path
export REF_9ROUTER="$HOME/_refs/9router"  # frozen reference (required)

git clone git@github.com:bloodf/g0router.git "$G0ROUTER"
cd "$G0ROUTER"

mkdir -p "$(dirname "$REF_9ROUTER")"
git clone https://github.com/decolua/9router.git "$REF_9ROUTER"
cd "$REF_9ROUTER" && git checkout 827e5c382b11f90b876f856ffa99cbd50f6abd6b
```

Verify SHA matches `.planning/parity/SOURCES.md`.

## 2. Toolchain

```bash
# Go (match go.mod version)
go version

# Harness CLIs (must be on PATH)
which kimi    # Moonshot Kimi CLI
which pi      # pi coding agent (openai-codex + minimax providers)
which timeout # coreutils

cd "$G0ROUTER"
go test ./...
go vet ./...
```

Configure `pi` providers per `.planning/harness/README.md` (openai-codex for gpt-5.5 critic, minimax if using M3).

## 3. Secrets / auth

- **Kimi:** API key in Kimi CLI config
- **gpt-5.5 critic:** `pi --provider openai-codex` auth
- **Git push:** deploy key or `gh auth login`
- **gh:** use `env -u GH_TOKEN -u GITHUB_TOKEN gh ...` if stale token env breaks auth

## 4. Harness runtime dirs (gitignored, create on VPS)

```bash
cd "$G0ROUTER"
mkdir -p .planning/harness/artifacts .planning/harness/jobs
chmod +x .planning/harness/*.sh
```

Copy job templates when dispatching:

```bash
PLAN_ID=w1-g   # example
cp .planning/harness/templates/impl-job.json .planning/harness/jobs/${PLAN_ID}-impl.json
# Edit PLAN_ID placeholders in JSON + write prompt from templates/impl-job.prompt.md
```

## 5. Start Claude Code

```bash
cd "$G0ROUTER"
claude   # or your Claude Code entrypoint
```

**First messages to Claude Code:**

1. Read `.planning/harness/HANDOFF.md` (orchestrator playbook)
2. Read `AGENTS.md` and `.planning/parity/plans/WAVE-MAP.md`
3. Continue Wave 1 from "Next actions" in HANDOFF

## 6. Scoped diff gate (preferred)

```bash
cd .planning/harness
# Paths from diff-scopes.json for plan slug w1-f-cloud-envelope:
./run-diff-scoped.sh gpt ../parity/plans/w1-f-cloud-envelope.md 80b01911^ 5d629345 -- \
  internal/translation/cloud_code.go ...
```

Or use `diff-scopes.json` + a small wrapper script Claude Code can generate per plan.

## 7. Environment variables (optional)

| Var | Default | Purpose |
|-----|---------|---------|
| `REF_9ROUTER` | `~/Developer/github.com/bloodf/_refs/9router` | Override ref path in prompts |
| `G0ROUTER` | repo cwd | Repo root in job prompts |
