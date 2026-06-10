#!/usr/bin/env bash
# Commit-bounded diff gate (fixes BASE...HEAD pollution from later plans).
#
# Usage:
#   run-diff-scoped.sh gpt <plan.md> <base-commit> <end-commit> [--] [path filters...]
#   run-diff-scoped.sh kimi <plan.md> <base-commit> <end-commit> [--] [path filters...]
#
# Example:
#   ./run-diff-scoped.sh gpt ../parity/plans/w1-f-cloud-envelope.md 80b01911^ 5d629345 -- \
#     internal/translation/cloud_code.go internal/translation/registry.go
set -uo pipefail

REVIEWER="${1:?reviewer: gpt|kimi}"; shift
PLAN_REL="${1:?plan file}"; shift
BASE="${1:?base commit}"; shift
END="${1:?end commit}"; shift
[ "${1:-}" = "--" ] && shift

HARNESS_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$HARNESS_DIR/../.." && pwd)"
PLAN="$(cd "$(dirname "$PLAN_REL")" && pwd)/$(basename "$PLAN_REL")"

cd "$REPO_ROOT"
[ -s "$PLAN" ] || { echo "FATAL: plan missing: $PLAN"; exit 2; }

ID="$(basename "$PLAN" .md)-diff-scoped-${REVIEWER}"
ART="$HARNESS_DIR/artifacts/${ID}.txt"
DIFF_FILE="$HARNESS_DIR/artifacts/${ID}.diff"

if [ "$#" -gt 0 ]; then
  git diff "$BASE".."$END" -- "$@" >"$DIFF_FILE"
else
  git diff "$BASE".."$END" >"$DIFF_FILE"
fi
[ -s "$DIFF_FILE" ] || { echo "FATAL: empty diff ($BASE..$END)"; exit 2; }

PROMPT="$(cat "$HARNESS_DIR/prompts/critic-diff.md")
--- MICRO-PLAN: $PLAN ---
$(cat "$PLAN")
--- DIFF ($BASE..$END) ---
$(cat "$DIFF_FILE")"

case "$REVIEWER" in
  gpt)
    timeout -k 5 1800 pi -p --provider openai-codex --model gpt-5.5 --no-session -nt "$PROMPT" </dev/null >"$ART" 2>&1
    ;;
  kimi)
    timeout -k 5 1800 kimi -p "$PROMPT" </dev/null >"$ART" 2>&1
    ;;
  *)
    echo "FATAL: reviewer must be gpt or kimi"; exit 2
    ;;
esac

"$HARNESS_DIR/parse-verdict.sh" "$ART"
