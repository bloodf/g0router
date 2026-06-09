#!/usr/bin/env bash
# Dispatch a worker job from a job JSON spec. Usage: run-worker.sh <job.json>
set -uo pipefail

JOB="$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"
HARNESS_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$HARNESS_DIR/../.." && pwd)"

jqf() { python3 -c "import json,sys;j=json.load(open('$JOB'));v=j.get('$1','');print(v if v is not None else '')"; }

ID="$(jqf id)"
WORKER="$(jqf worker)"
TIMEOUT="$(jqf timeout)"; TIMEOUT="${TIMEOUT:-3600}"
PROMPT_FILE="$REPO_ROOT/$(jqf prompt_file)"
ART="$HARNESS_DIR/artifacts/${ID}.log"

[ -f "$PROMPT_FILE" ] || { echo "FATAL: prompt file missing: $PROMPT_FILE"; exit 2; }
PROMPT="$(cat "$PROMPT_FILE")"

cd "$REPO_ROOT"
case "$WORKER" in
  kimi)
    # kimi lingers after completion; timeout kill is expected (exit 124 != failure)
    timeout -k 5 "$TIMEOUT" kimi -p "$PROMPT" </dev/null >"$ART" 2>&1
    ;;
  m3)
    timeout -k 5 "$TIMEOUT" pi -p --provider minimax --model MiniMax-M3 --no-session "$PROMPT" </dev/null >"$ART" 2>&1
    ;;
  m27-hs)
    timeout -k 5 "$TIMEOUT" pi -p --provider minimax --model MiniMax-M2.7-highspeed --no-session -t read,bash "$PROMPT" </dev/null >"$ART" 2>&1
    ;;
  gpt-5.5)
    timeout -k 5 "$TIMEOUT" pi -p --provider openai-codex --model gpt-5.5 --no-session -nt "$PROMPT" </dev/null >"$ART" 2>&1
    ;;
  *)
    echo "FATAL: unknown worker '$WORKER'"; exit 2
    ;;
esac
RC=$?

# Success = all expected outputs exist (exit code unreliable for kimi)
MISSING=$(python3 - "$JOB" "$REPO_ROOT" <<'EOF'
import json,sys,os
j=json.load(open(sys.argv[1])); root=sys.argv[2]
missing=[p for p in j.get("expected_outputs",[]) if not os.path.exists(os.path.join(root,p))]
print("\n".join(missing))
EOF
)
if [ -n "$MISSING" ]; then
  echo "JOB:$ID STATUS:INCOMPLETE rc=$RC missing_outputs:"
  echo "$MISSING"
  echo "artifact:$ART"
  exit 1
fi
echo "JOB:$ID STATUS:DONE rc=$RC artifact:$ART"
