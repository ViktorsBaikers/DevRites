#!/usr/bin/env bash
# conventions-ledger-test.sh — fixture tests for the B1 conventions ledger:
# M1 deterministic corroboration scorer + M2 ledger store (promote/contradict/read).
set -u

HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$HERE/.." && pwd -P)"
# The conventions ledger now ships in the Go engine (`devrites-engine conventions …`); this
# test exercises that runtime path. Build the binary hermetically, mirroring
# binary-lifecycle-test.sh, and self-skip where the Go toolchain / engine checkout is absent.
command -v go >/dev/null 2>&1 || { echo "  SKIP: go toolchain not available"; exit 0; }
[ -d "$ROOT/engine" ] || { echo "  SKIP: no engine/ (tarball install)"; exit 0; }
BIN="$(mktemp -d)"
( cd "$ROOT/engine" && go build -o "$BIN/devrites-engine" . ) || { echo "FAIL: could not build devrites-engine"; exit 1; }
CONV="$BIN/devrites-engine conventions"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP" "$BIN"' EXIT
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

# --- B2: orient digest (read-at-orient) ------------------------------------
B2="$TMP/b2"; mkdir -p "$B2"
# empty ledger → empty digest
eq "orient empty → nothing" "" "$($CONV orient --root "$B2")"
# a high-band convention (5 distinct slices → 0.90) and a low-band one (0.50)
for s in s1 s2 s3 s4 s5; do
  $CONV promote --root "$B2" --key error-model --kind error-model \
    --statement "Result<T,E>, no exceptions across boundaries" \
    --slug "$s" --evidence "proved in $s" --date 2026-06-20 >/dev/null
done
$CONV promote --root "$B2" --key http-client --kind layering \
  --statement "all outbound via lib/http with retry" --slug s1 \
  --evidence "proved in s1" --date 2026-06-20 >/dev/null
digest="$($CONV orient --root "$B2")"
case "$digest" in
  *"[0.90] error-model"*"[0.50] http-client"*) echo "ok   [orient sorts by band desc]" ;;
  *) echo "FAIL [orient sorts by band desc]: got '$digest'"; fail=1 ;;
esac
# --min-band filters the low one out
mb="$($CONV orient --root "$B2" --min-band 0.7)"
case "$mb" in
  *error-model*) case "$mb" in *http-client*) echo "FAIL [orient --min-band filters]"; fail=1 ;; *) echo "ok   [orient --min-band filters]" ;; esac ;;
  *) echo "FAIL [orient --min-band keeps high band]"; fail=1 ;;
esac
# a retired convention is omitted from the digest
$CONV promote  --root "$B2" --key legacy-flag --kind gotcha --statement "x" --slug s1 --evidence e --date 2026-06-20 >/dev/null
$CONV contradict --root "$B2" --key legacy-flag --slug s2 --evidence "removed" --date 2026-06-21 >/dev/null
case "$($CONV orient --root "$B2")" in
  *legacy-flag*) echo "FAIL [orient omits retired]"; fail=1 ;;
  *) echo "ok   [orient omits retired]" ;;
esac

# --- B2: contradiction logged to drift.md (--drift-file) -------------------
DRIFT="$B2/.devrites/work/featureX/drift.md"
$CONV contradict --root "$B2" --key error-model --slug s6 \
  --evidence "this slice throws across the boundary" --date 2026-06-22 \
  --drift-file "$DRIFT" >/dev/null
grep_in "drift entry written"      "## convention-drift: error-model" "$DRIFT"
grep_in "drift names the slice"    "slice: s6"                        "$DRIFT"

if [ "$fail" -ne 0 ]; then
  echo "CONVENTIONS-LEDGER TESTS: FAIL"; exit 1
fi
echo "CONVENTIONS-LEDGER TESTS: PASS"
exit 0
