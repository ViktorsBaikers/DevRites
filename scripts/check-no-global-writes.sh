#!/usr/bin/env bash
# check-no-global-writes.sh — static guard that the installer/uninstaller never
# writes to the user's global ~/.claude, and that the global guard is present.
# Exits non-zero on violation.

set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0

SCRIPTS="$ROOT/install.sh $ROOT/uninstall.sh"
for f in "$ROOT"/scripts/*.sh; do SCRIPTS="$SCRIPTS $f"; done

# 1) the no-global guard marker must exist in install.sh
if ! grep -q 'GUARD:no-global' "$ROOT/install.sh"; then
  echo "FAIL: install.sh is missing the GUARD:no-global guard"
  fail=1
else
  echo "ok: install.sh contains the no-global guard"
fi

# 2) no write operation may target ~/.claude or \$HOME/.claude.
#    Write verbs are actual file-writing commands / redirections. ('install' is
#    intentionally excluded: the installer uses cp/mkdir, and "install" appears in
#    prose like "install into a project", which is not a write.)
WRITE_RE='(\bmkdir\b|\bcp\b|\btee\b|\brsync\b|\bmv\b|\bln\b|>>?)'
GLOBAL_RE='(\$HOME/\.claude|~/\.claude)'
for f in $SCRIPTS; do
  [ -f "$f" ] || continue
  # lines that contain a global-claude path AND a write verb, excluding comments
  hits="$(grep -nE "$GLOBAL_RE" "$f" | grep -vE '^\s*[0-9]+:\s*#' | grep -E "$WRITE_RE" || true)"
  if [ -n "$hits" ]; then
    echo "FAIL: possible global write in ${f#$ROOT/}:"
    printf '%s\n' "$hits" | sed 's/^/    /'
    fail=1
  fi
done
[ "$fail" -eq 0 ] && echo "ok: no write operations target ~/.claude"

if [ "$fail" -eq 0 ]; then
  echo "PASS: no global-write risks detected"
else
  echo "FAILED: global-write check"
fi
exit "$fail"
