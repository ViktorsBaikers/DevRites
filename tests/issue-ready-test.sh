#!/usr/bin/env bash
# issue-ready.sh resolves "Blocked by:" within the blocked issue's own feature.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT

mkdir -p "$T/a" "$T/b"
printf '# A1\nStatus: open\n' >"$T/a/01-first.md"
printf '# A2\nStatus: ready-for-agent\nBlocked by: 01\n' >"$T/a/02-second.md"
printf '# B1\nStatus: done\n' >"$T/b/01-first.md"
printf '# B2\nStatus: open\nBlocked by: 01\n' >"$T/b/02-second.md"

out="$(bash "$ROOT/scripts/issue-ready.sh" "$T/b" "$T/a")"
fail=0
case "$out" in *"a/01-first.md"*) ;; *) echo "FAIL: unblocked a/01 missing"; fail=1 ;; esac
case "$out" in *"b/02-second.md"*) ;; *) echo "FAIL: b/02 blocked by done b/01 should be ready"; fail=1 ;; esac
case "$out" in *"a/02-second.md"*) echo "FAIL: a/02 resolved its blocker against another feature's 01"; fail=1 ;; esac

[ "$fail" -eq 0 ] && echo "PASS: issue-ready keys blockers per feature"
exit "$fail"
