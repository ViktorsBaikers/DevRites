#!/usr/bin/env bash
# install-flag-parser-invalid-smoke.sh: parser warnings and unknown flag rejection.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT

echo "== install-flag-parser-invalid-smoke =="

out="$(bash "$ROOT/install.sh" --target "$T" --dry-run --short-aliases 2>&1)"
status=$?
if [ "$status" -eq 0 ]; then
  ok "install.sh accepts bare --short-aliases (warns, no-op)"
else
  no "install.sh errored on bare --short-aliases"
fi
echo "$out" | grep -qi 'no-op' && ok "bare --short-aliases warns it's a no-op" || no "bare --short-aliases did not warn"

if bash "$ROOT/install.sh" --target "$T" --dry-run --totally-bogus >/dev/null 2>&1; then
  no "install.sh accepted a bogus flag"
else
  ok "install.sh still rejects unknown flags"
fi

echo ""
[ "$fail" -eq 0 ] && echo "install-flag-parser-invalid-smoke: PASS" || echo "install-flag-parser-invalid-smoke: FAIL"
exit "$fail"
