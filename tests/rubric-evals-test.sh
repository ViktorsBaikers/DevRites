#!/usr/bin/env bash
# Rubric grader vote rules against a localhost mock judge: D2 == 0 or a stated
# total that disagrees with the dimension sum fails the row; bad --samples is a
# usage error; the summary names the real output directory.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp="$(mktemp -d)"
server_pid=""
trap '[ -n "$server_pid" ] && kill "$server_pid" 2>/dev/null; rm -rf "$tmp"' EXIT

mkdir -p "$tmp/transcripts"
printf '{"corpus":"rite-build-parallel","scenario_id":"BP1-default-or-n1-does-not-fan-out","transcript":"specimen"}\n' \
  >"$tmp/transcripts/t.json"

# The judge replies with whatever reply.txt holds at request time.
cat >"$tmp/judge.mjs" <<'JS'
import http from 'node:http';
import { readFileSync, writeFileSync } from 'node:fs';
const [replyFile, portFile] = process.argv.slice(2);
const server = http.createServer((req, res) => {
  req.resume();
  req.on('end', () => {
    res.setHeader('content-type', 'application/json');
    res.end(JSON.stringify({ choices: [{ message: { content: readFileSync(replyFile, 'utf8') } }] }));
  });
});
server.listen(0, '127.0.0.1', () => writeFileSync(portFile, String(server.address().port)));
JS
node "$tmp/judge.mjs" "$tmp/reply.txt" "$tmp/port" &
server_pid=$!
for _ in $(seq 1 100); do [ -s "$tmp/port" ] && break; sleep 0.1; done
[ -s "$tmp/port" ] || { echo "FAIL: mock judge did not start" >&2; exit 1; }

run_rubric() {
  set +e
  output="$(DEVRITES_RUBRIC_JUDGE=mock/judge DEVRITES_RUBRIC_JUDGE_API_KEY=test \
    DEVRITES_RUBRIC_JUDGE_BASE_URL="http://127.0.0.1:$(cat "$tmp/port")" \
    node "$ROOT/scripts/run-rubric-evals.mjs" --transcripts "$tmp/transcripts" --out "$tmp/out" --strict "$@" 2>&1)"
  code=$?
  set -e
}

case_reply() {
  local id="$1" want_code="$2" reply="$3"
  printf 'VERDICT: pass\nSCORE: %s\nNOTES: mock\n' "$reply" >"$tmp/reply.txt"
  run_rubric --samples 1
  [ "$code" -eq "$want_code" ] || { printf 'FAIL %s: exit=%s want %s\n%s\n' "$id" "$code" "$want_code" "$output" >&2; exit 1; }
  printf '  PASS %-24s exit=%s\n' "$id" "$code"
}

case_reply clean-pass 0 '2/2/2 total 6/6'
case_reply capitulation-d2-zero 1 '2/0/2 total 6/6'
case_reply capitulation-d2-zero-sum 1 '2/0/2 total 4/6'
case_reply total-mismatch 1 '1/2/1 total 6/6'

grep -Fq "results in $tmp/out/rubric-latest.json" <<<"$output" \
  || { printf 'FAIL: summary does not name the output dir\n%s\n' "$output" >&2; exit 1; }
printf '  PASS %-24s\n' summary-names-out-dir

for bad in abc 0 1.5 ''; do
  run_rubric --samples "$bad"
  [ "$code" -eq 1 ] && grep -Fq -- '--samples must be a positive integer' <<<"$output" \
    || { printf 'FAIL samples=%q: exit=%s\n%s\n' "$bad" "$code" "$output" >&2; exit 1; }
done
printf '  PASS %-24s\n' bad-samples

echo "rubric eval regression: PASS"
