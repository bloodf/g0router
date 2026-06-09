#!/usr/bin/env bash
# Gate runner via M2.7-HS: runs go test/vet (+ optional ui build), digests output to GATES: PASS|FAIL.
# Usage: run-gates.sh [label] [ui]
set -uo pipefail

LABEL="${1:-gates}"
WITH_UI="${2:-}"
HARNESS_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$HARNESS_DIR/../.." && pwd)"
ART="$HARNESS_DIR/artifacts/${LABEL}-gates.log"
cd "$REPO_ROOT"

CMDS="go test ./... && go vet ./..."
[ "$WITH_UI" = "ui" ] && CMDS="$CMDS && (cd ui && npm run build)"

PROMPT="You are a CI gate runner. Working directory: $REPO_ROOT
Run exactly: $CMDS
Then report. Output contract (MUST follow exactly):
- If everything passes: first line is exactly 'GATES: PASS'. Nothing else.
- If anything fails: first line is exactly 'GATES: FAIL', then one line per failure with package, test name, and the single most relevant error line. Max 20 lines total. NEVER paste full logs.
Do not modify any files. Do not run any other commands."

timeout -k 5 1800 pi -p --provider minimax --model MiniMax-M2.7-highspeed --no-session -t read,bash "$PROMPT" </dev/null >"$ART" 2>&1

if grep -q "GATES: PASS" "$ART"; then
  echo "GATES: PASS ($LABEL) artifact:$ART"
  exit 0
elif grep -q "GATES: FAIL" "$ART"; then
  echo "GATES: FAIL ($LABEL) artifact:$ART"
  sed -n '/GATES: FAIL/,$p' "$ART" | head -21
  exit 1
else
  echo "GATES: AMBIGUOUS ($LABEL) artifact:$ART — rerun failing package directly"
  exit 2
fi
