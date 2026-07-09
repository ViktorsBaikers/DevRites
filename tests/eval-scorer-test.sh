#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
cat >"$TMP/out.txt" <<'TXT'
The reviewer found that processPayment 47 52 can double refund a new hire account when retries race.
TXT
cat >"$TMP/truth.json" <<'JSON'
{"findings":[{"id":"refund-race","keywords":["processPayment():47-52","double-refund","new hire","retries","race","account"]}]}
JSON
node "$ROOT/scripts/eval-scorer.mjs" "$TMP/out.txt" "$TMP/truth.json" >"$TMP/score.json"
grep -q '"found": 1' "$TMP/score.json"
grep -q '"matched": true' "$TMP/score.json"
echo "eval-scorer: PASS"
