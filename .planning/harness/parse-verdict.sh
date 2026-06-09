#!/usr/bin/env bash
# Extract verdict from a critic artifact. Exit: 0 PASS, 1 REJECT/FAIL, 2 no verdict found.
set -uo pipefail
ART="$1"
[ -f "$ART" ] || { echo "VERDICT: MISSING (no artifact)"; exit 2; }
LINE="$(grep -E '^(VERDICT|GATES):' "$ART" | tail -1)"
if [ -z "$LINE" ]; then
  echo "VERDICT: MISSING artifact:$ART"
  exit 2
fi
echo "$LINE artifact:$ART"
case "$LINE" in
  *PASS*) exit 0 ;;
  *) exit 1 ;;
esac
