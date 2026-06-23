#!/usr/bin/env bash
# Unit test for the evidence-freshness gate (devrites-lib/scripts/evidence-fresh.sh).
#
# DevRites' core claim is "won't claim done without proof, and the proof must
# post-date the code it proves." That invariant is enforced live by
# evidence-fresh.sh but was previously exercised by no test. see-it-fail-first:
# assert STALE proof returns exit 3 AND fresh proof returns exit 0.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT="$ROOT/pack/.claude/skills/devrites-lib/scripts/evidence-fresh.sh"
[ -f "$SCRIPT" ] || { echo "missing $SCRIPT" >&2; exit 1; }

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
cd "$tmp"

slug="ftest"
ws=".devrites/work/$slug"
mkdir -p "$ws" src
echo "$slug" > .devrites/ACTIVE
echo "code" > src/app.js
printf '# touched\n- `src/app.js`\n' > "$ws/touched-files.md"
printf '# evidence\nproof\n'         > "$ws/evidence.md"

fail=0
run() { set +e; bash "$SCRIPT" "$slug" >/dev/null 2>&1; echo $?; set -e; }

# Case 1: STALE — a touched file is newer than the proof -> exit 3
touch -t 202601010000 "$ws/evidence.md"
touch -t 202601020000 src/app.js
rc=$(run)
if [ "$rc" -eq 3 ]; then echo "PASS: stale proof -> exit 3"; else echo "FAIL: stale expected 3, got $rc"; fail=1; fi

# Case 2: FRESH — proof post-dates the code -> exit 0
touch -t 202601030000 "$ws/evidence.md"
rc=$(run)
if [ "$rc" -eq 0 ]; then echo "PASS: fresh proof -> exit 0"; else echo "FAIL: fresh expected 0, got $rc"; fail=1; fi

[ "$fail" -eq 0 ] && echo "evidence-fresh-test: OK" || echo "evidence-fresh-test: FAILED"
exit "$fail"
