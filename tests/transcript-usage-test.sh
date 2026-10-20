#!/usr/bin/env bash
# transcript-usage.py must join each dispatched role to its own host transcript,
# count every assistant turn once, report no paths, and degrade to "unavailable"
# on a layout it does not know.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
SCRIPT="$ROOT/scripts/transcript-usage.py"
T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT

# The security reviewer has no transcript on either host, so it is reported missing.
W="$T/proj/.devrites/work/feat"
mkdir -p "$W"
cat >"$W/dispatch.json" <<'EOF'
{"waves": {
  "build-1": {"phase": "build", "state": "complete", "roles": {
    "devrites-slice-wright": {"handle": "Slice001Wright", "started": "2026-10-20T10:00:05Z", "returned": true, "returned_at": "2026-10-20T10:12:00Z"}}},
  "review-1": {"phase": "build", "state": "complete", "roles": {
    "devrites-code-reviewer": {"handle": "CodeReviewer", "started": "2026-10-20T10:13:05Z", "returned": true, "returned_at": "2026-10-20T10:18:00Z"},
    "devrites-security-reviewer": {"handle": "SecurityReviewer", "started": "2026-10-20T10:13:06Z", "returned": true, "returned_at": "2026-10-20T10:18:00Z"}}}
}}
EOF

# omp: <dir>/<session>.jsonl is the root, <dir>/<session>/<Handle>.jsonl a subagent.
# Slice001Wright also exists in session s2, 20 minutes off, and in s0 a day earlier:
# the nearest start wins, the far one is out of range, and neither root counts.
O="$T/omp"
mkdir -p "$O/s0" "$O/s1" "$O/s2"
omp_turn() { # <ts> <model> <input> <cacheRead> <cacheWrite> <output> <cost>
  printf '{"type":"message","timestamp":"%s","message":{"role":"assistant","model":"%s","usage":{"input":%s,"cacheRead":%s,"cacheWrite":%s,"output":%s,"cost":{"total":%s}}}}\n' "$@"
}
{
  echo '{"type":"session","timestamp":"2026-10-20T09:00:00Z"}'
  omp_turn 2026-10-20T09:01:00Z grok-4.7 10 100 20 5 0.5
} >"$O/s1.jsonl"
{
  echo '{"type":"title","updatedAt":"2026-10-20T10:00:00Z"}'
  echo '{"type":"session","timestamp":"2026-10-20T10:00:00Z"}'
  omp_turn 2026-10-20T10:00:01Z claude-sonnet-5-5 0 0 0 0 0
  echo '{"type":"message","timestamp":"2026-10-20T10:00:02Z","message":{"role":"user","content":"go"}}'
  omp_turn 2026-10-20T10:00:03Z claude-sonnet-5-5 2 1000 50 30 0.25
  echo '{"type":"message","timestamp":"2026-10-20T10:00:04Z","message":{"role":"toolResult","usage":{"input":999}}}'
  echo 'not json'
} >"$O/s1/Slice001Wright.jsonl"
{
  echo '{"type":"session","timestamp":"2026-10-20T10:13:00Z"}'
  omp_turn 2026-10-20T10:13:01Z gpt-6.1-sol 400 0 0 40 0.1
} >"$O/s1/CodeReviewer.jsonl"
for s in s0 s2; do
  ts=2026-10-20T10:20:00Z
  [ "$s" = s0 ] && ts=2026-10-19T10:00:00Z
  { echo "{\"type\":\"session\",\"timestamp\":\"$ts\"}"; omp_turn "$ts" other 9999 0 0 9999 9; } >"$O/$s/Slice001Wright.jsonl"
  { echo "{\"type\":\"session\",\"timestamp\":\"$ts\"}"; omp_turn "$ts" other 7777 0 0 7777 7; } >"$O/$s.jsonl"
done

# claude: subagents/agent-*.jsonl join by the sibling meta.json agentType; one
# message is written once per content block and counts once.
C="$T/claude"
mkdir -p "$C/s1/subagents"
claude_turn() { # <ts> <id> <model> <input> <cacheRead> <cacheWrite> <output>
  printf '{"type":"assistant","timestamp":"%s","message":{"id":"%s","role":"assistant","model":"%s","usage":{"input_tokens":%s,"cache_read_input_tokens":%s,"cache_creation_input_tokens":%s,"output_tokens":%s}}}\n' "$@"
}
{
  echo '{"type":"user","timestamp":"2026-10-20T09:00:00Z","message":{"role":"user","content":"go"}}'
  claude_turn 2026-10-20T09:01:00Z m1 claude-opus-5-5 3 300 30 7
  claude_turn 2026-10-20T09:01:00Z m1 claude-opus-5-5 3 300 30 7
} >"$C/s1.jsonl"
{
  echo '{"type":"user","timestamp":"2026-10-20T10:00:00Z","message":{"role":"user","content":"go"}}'
  claude_turn 2026-10-20T10:00:01Z w1 claude-sonnet-5-5 0 0 0 0
  claude_turn 2026-10-20T10:00:02Z w2 claude-sonnet-5-5 2 500 40 20
  claude_turn 2026-10-20T10:00:02Z w2 claude-sonnet-5-5 2 500 40 20
} >"$C/s1/subagents/agent-a1.jsonl"
echo '{"agentType":"devrites-slice-wright"}' >"$C/s1/subagents/agent-a1.meta.json"
{
  echo '{"type":"user","timestamp":"2026-10-20T10:13:00Z","message":{"role":"user","content":"go"}}'
  claude_turn 2026-10-20T10:13:01Z r1 claude-opus-5-5 100 0 0 10
} >"$C/s1/subagents/agent-b1.jsonl"
echo '{"agentType":"devrites-code-reviewer"}' >"$C/s1/subagents/agent-b1.meta.json"

mkdir -p "$T/unknown"
echo 'hello' >"$T/unknown/notes.txt"

# check <host> <dir> <golden-json> runs the script and compares its whole report.
check() {
  python3 -I "$SCRIPT" --host "$1" --transcripts "$2" "$W" >"$T/out.json"
  python3 -I - "$T/out.json" "$3" "$1" <<'PY'
import json, sys
text = open(sys.argv[1]).read()
if '"/' in text:
    sys.exit(f"FAIL: {sys.argv[3]}: report contains an absolute path")
got, want = json.loads(text), json.loads(sys.argv[2])
if got != want:
    sys.exit(f"FAIL: {sys.argv[3]}: got {json.dumps(got, sort_keys=True)}\nwant {json.dumps(want, sort_keys=True)}")
PY
}

OMP_WRIGHT='{"runs": 1, "turns": 2, "models": {"claude-sonnet-5-5": 2}, "input": 2, "cache_read": 1000, "cache_write": 50, "output": 30, "cost": 0.25}'
OMP_REVIEWER='{"runs": 1, "turns": 1, "models": {"gpt-6.1-sol": 1}, "input": 400, "cache_read": 0, "cache_write": 0, "output": 40, "cost": 0.1}'
check omp "$O" "{\"host\": \"omp\", \"usage\": \"available\", \"dispatches\": {\"joined\": 2, \"missing\": 1},
  \"roles\": {\"devrites-slice-wright\": $OMP_WRIGHT, \"devrites-code-reviewer\": $OMP_REVIEWER},
  \"waves\": {\"feat/build-1\": $OMP_WRIGHT, \"feat/review-1\": $OMP_REVIEWER},
  \"root\": {\"runs\": 1, \"turns\": 1, \"models\": {\"grok-4.7\": 1}, \"input\": 10, \"cache_read\": 100, \"cache_write\": 20, \"output\": 5, \"cost\": 0.5}}"

CL_WRIGHT='{"runs": 1, "turns": 2, "models": {"claude-sonnet-5-5": 2}, "input": 2, "cache_read": 500, "cache_write": 40, "output": 20, "cost": null}'
CL_REVIEWER='{"runs": 1, "turns": 1, "models": {"claude-opus-5-5": 1}, "input": 100, "cache_read": 0, "cache_write": 0, "output": 10, "cost": null}'
check claude "$C" "{\"host\": \"claude\", \"usage\": \"available\", \"dispatches\": {\"joined\": 2, \"missing\": 1},
  \"roles\": {\"devrites-slice-wright\": $CL_WRIGHT, \"devrites-code-reviewer\": $CL_REVIEWER},
  \"waves\": {\"feat/build-1\": $CL_WRIGHT, \"feat/review-1\": $CL_REVIEWER},
  \"root\": {\"runs\": 1, \"turns\": 1, \"models\": {\"claude-opus-5-5\": 1}, \"input\": 3, \"cache_read\": 300, \"cache_write\": 30, \"output\": 7, \"cost\": null}}"

# An omp layout read as claude, or an unknown directory, joins nothing: unavailable, exit 0.
UNAVAILABLE='"usage": "unavailable", "reason": "no dispatched role joined a transcript in this layout", "dispatches": {"joined": 0, "missing": 3}}'
check claude "$O" "{\"host\": \"claude\", $UNAVAILABLE"
check omp "$T/unknown" "{\"host\": \"omp\", $UNAVAILABLE"

for args in "--host omp --transcripts $T/missing $W" "--host codex --transcripts $O $W" "--host omp --transcripts $O $T/nowhere" "--host omp $W"; do
  code=0
  # shellcheck disable=SC2086 # each case is a deliberately split argument list
  python3 -I "$SCRIPT" $args >/dev/null 2>&1 || code=$?
  if [ "$code" -ne 2 ]; then
    echo "FAIL: transcript-usage accepted $args (exit $code)"
    exit 1
  fi
done

echo "transcript-usage: ok"
