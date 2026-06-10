#!/usr/bin/env bash
# Critic gates. Usage:
#   run-critic.sh plan <plan-file> [extra-context-file...]   -> gpt-5.5 reviews a plan/matrix doc
#   run-critic.sh diff <plan-file> [git_base]                -> kimi reviews implementation diff vs plan
set -uo pipefail

MODE="$1"; shift
HARNESS_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$HARNESS_DIR/../.." && pwd)"

abspath() { cd "$(dirname "$1")" && printf '%s/%s' "$(pwd)" "$(basename "$1")"; }

case "$MODE" in
  plan)
    TARGET="$(abspath "$1")"; shift || true
    CTXFILES=()
    for f in "$@"; do CTXFILES+=("$(abspath "$f")"); done
    cd "$REPO_ROOT"
    [ -s "$TARGET" ] || { echo "FATAL: target empty/missing: $TARGET"; exit 2; }
    ID="$(basename "$TARGET" .md)-plan-review"
    ART="$HARNESS_DIR/artifacts/${ID}.txt"
    CTX=""
    for f in "${CTXFILES[@]:-}"; do [ -n "$f" ] && CTX+=$'\n\n--- CONTEXT FILE: '"$f"$' ---\n'"$(cat "$f")"; done
    PROMPT="$(cat "$HARNESS_DIR/prompts/critic-plan.md")
--- DOCUMENT UNDER REVIEW: $TARGET ---
$(cat "$TARGET")$CTX"
    timeout -k 5 900 pi -p --provider openai-codex --model gpt-5.5 --no-session -nt "$PROMPT" </dev/null >"$ART" 2>&1
    ;;
  diff)
    TARGET="$(abspath "$1")"
    BASE="${2:-main}"
    shift 2 || shift $#   # remaining args = optional path filters
    cd "$REPO_ROOT"
    [ -s "$TARGET" ] || { echo "FATAL: target empty/missing: $TARGET"; exit 2; }
    ID="$(basename "$TARGET" .md)-diff-review"
    ART="$HARNESS_DIR/artifacts/${ID}.txt"
    DIFF_FILE="$HARNESS_DIR/artifacts/${ID}.diff"
    if [ "$#" -gt 0 ]; then
      git diff "$BASE"...HEAD -- "$@" >"$DIFF_FILE"
    else
      git diff "$BASE"...HEAD >"$DIFF_FILE"
    fi
    [ -s "$DIFF_FILE" ] || { echo "FATAL: empty diff"; exit 2; }
    PROMPT="$(cat "$HARNESS_DIR/prompts/critic-diff.md")
--- MICRO-PLAN: $TARGET ---
$(cat "$TARGET")
--- DIFF (vs $BASE) ---
$(cat "$DIFF_FILE")"
    timeout -k 5 1800 kimi -p "$PROMPT" </dev/null >"$ART" 2>&1
    ;;
  *)
    echo "FATAL: mode must be plan|diff"; exit 2
    ;;
esac

"$HARNESS_DIR/parse-verdict.sh" "$ART"
