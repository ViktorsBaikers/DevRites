#!/usr/bin/env bash
# footprint-test.sh — the fan-out footprint renders deterministically from the dispatch
# log, counts only what DevRites tracks, and NEVER emits a token/dollar figure.
set -u

HERE="$(cd "$(dirname "$0")" && pwd)"
FP="$HERE/../pack/.claude/skills/devrites-lib/scripts/footprint.sh"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
fail=0
mkdir -p "$TMP/.devrites/work/feat"

# 1. no log → n/a
out="$(cd "$TMP" && bash "$FP" render feat)"
[ "$out" = "Footprint: n/a (no dispatch records)" ] && echo "ok   [no log → n/a]" || { echo "FAIL [no log → n/a]: '$out'"; fail=1; }

# 2. deterministic render from a fixed-epoch fixture
cat > "$TMP/.devrites/work/feat/footprint.log" <<'EOF'
1000 wright 01-list
1100 wright 02-detail
1200 reviewer devrites-code-reviewer
1200 reviewer devrites-security-auditor
1300 reviewer devrites-test-analyst
EOF
out="$(cd "$TMP" && bash "$FP" render feat)"
want="Footprint: 5 subagents (2 wright · 3 reviewers) · 2 slices · ~5m active"
[ "$out" = "$want" ] && echo "ok   [deterministic render]" || { echo "FAIL [deterministic render]"; echo " got: $out"; echo "want: $want"; fail=1; }

# 3. never emits a token/dollar/cost figure
if printf '%s' "$out" | grep -qiE 'token|\$|cost|usd'; then echo "FAIL [no token/cost figure]"; fail=1; else echo "ok   [no token/cost figure]"; fi

# 4. log appends a dispatch record
before="$(wc -l < "$TMP/.devrites/work/feat/footprint.log")"
( cd "$TMP" && bash "$FP" log feat audit security )
after="$(wc -l < "$TMP/.devrites/work/feat/footprint.log")"
[ "$after" -eq "$((before + 1))" ] && echo "ok   [log appends a record]" || { echo "FAIL [log appends a record]: $before -> $after"; fail=1; }

# 5. log on a missing workspace never fails the caller
( cd "$TMP" && bash "$FP" log ghost wright x ); rc=$?
[ "$rc" -eq 0 ] && echo "ok   [log missing workspace → rc 0]" || { echo "FAIL [log missing workspace]: rc=$rc"; fail=1; }

if [ "$fail" -ne 0 ]; then echo "FOOTPRINT TESTS: FAIL"; exit 1; fi
echo "FOOTPRINT TESTS: PASS"
exit 0
