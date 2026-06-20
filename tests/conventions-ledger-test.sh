#!/usr/bin/env bash
# conventions-ledger-test.sh — fixture tests for the B1 conventions ledger:
# M1 deterministic corroboration scorer + M2 ledger store (promote/contradict/read).
set -u

HERE="$(cd "$(dirname "$0")" && pwd)"
CONV="python3 $HERE/../pack/.claude/skills/devrites-lib/scripts/conventions.py"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
fail=0

eq() { # label expected actual
  if [ "$2" = "$3" ]; then echo "ok   [$1]"; else echo "FAIL [$1]: expected '$2', got '$3'"; fail=1; fi
}
grep_in() { # label pattern file
  if grep -q "$2" "$3"; then echo "ok   [$1]"; else echo "FAIL [$1]: /$2/ not in $3"; fail=1; fi
}

LEDGER="$TMP/.devrites/conventions.md"

# --- M1: scorer (pure, deterministic, monotonic) ---------------------------
eq "band c1/k0 seeds mid"   "0.50" "$($CONV band --corroborations 1)"
eq "band c2/k0 rises"       "0.60" "$($CONV band --corroborations 2)"
eq "band c5/k0 hits cap"    "0.90" "$($CONV band --corroborations 5)"
eq "band c9/k0 stays capped" "0.90" "$($CONV band --corroborations 9)"
eq "band c2/k1 lowered"     "0.40" "$($CONV band --corroborations 2 --contradictions 1)"
eq "band net<=0 retires"    "retired" "$($CONV band --corroborations 1 --contradictions 1)"
eq "band determinism"       "$($CONV band --corroborations 3)" "$($CONV band --corroborations 3)"

# --- M2: store -------------------------------------------------------------
# read on a missing ledger is clean + empty
eq "read missing → empty" "" "$($CONV read --root "$TMP")"

# promote a new convention → created mid-band with provenance
$CONV promote --root "$TMP" --key "test-runner" \
  --statement "tests run with vitest, co-located *.test.ts" --kind test \
  --slug 01-list --evidence "evidence.md: 12 passing" --date 2026-06-20 >/dev/null
grep_in "promote creates entry"  "^## test-runner"      "$LEDGER"
grep_in "promote band 0.50"      "band: 0.50"           "$LEDGER"
grep_in "promote provenance"     "01-list 2026-06-20"   "$LEDGER"

# corroborate from a DISTINCT slice → band rises to 0.60, 2 corroborations
$CONV promote --root "$TMP" --key "test-runner" \
  --statement "" --kind test --slug 02-detail \
  --evidence "evidence.md: 8 passing" --date 2026-06-21 >/dev/null
grep_in "corroborate band 0.60"  "band: 0.60"           "$LEDGER"
grep_in "corroborate count 2"    "corroborations: 2"    "$LEDGER"

# idempotent: re-promote SAME slug must not double-count
$CONV promote --root "$TMP" --key "test-runner" \
  --statement "" --kind test --slug 02-detail \
  --evidence "re-seal" --date 2026-06-21 >/dev/null
grep_in "idempotent same-slug"   "corroborations: 2"    "$LEDGER"

# contradict → emits a DRIFT line and lowers the band
drift="$($CONV contradict --root "$TMP" --key "test-runner" \
  --slug 03-api --evidence "now uses jest" --date 2026-06-22)"
case "$drift" in
  DRIFT:*test-runner*) echo "ok   [contradict emits drift]" ;;
  *) echo "FAIL [contradict emits drift]: got '$drift'"; fail=1 ;;
esac
grep_in "contradict count 1"     "contradictions: 1"    "$LEDGER"
grep_in "contradict band 0.40"   "band: 0.40"           "$LEDGER"

# contradict an unknown key → non-zero, no crash
if $CONV contradict --root "$TMP" --key "nope" --slug x --evidence y >/dev/null 2>&1; then
  echo "FAIL [contradict unknown → error]"; fail=1
else
  echo "ok   [contradict unknown → error]"
fi

if [ "$fail" -ne 0 ]; then
  echo "CONVENTIONS-LEDGER TESTS: FAIL"; exit 1
fi
echo "CONVENTIONS-LEDGER TESTS: PASS"
exit 0
