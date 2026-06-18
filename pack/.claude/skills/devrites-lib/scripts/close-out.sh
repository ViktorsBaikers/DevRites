#!/usr/bin/env bash
# Close a shipped feature: archive its workspace and clear the ACTIVE cursor.
#
# Moves .devrites/work/<slug>/ -> .devrites/archive/<slug>/ (preserving every .md
# audit file) and clears .devrites/ACTIVE so the next /rite-spec starts clean. The
# audit trail is never deleted — only relocated.
#
# Usage: close-out.sh <slug> [devrites-dir]
#   devrites-dir defaults to .devrites
#
# Exit codes:
#   0  archived (+ ACTIVE cleared if it pointed at <slug>)
#   4  bad args / workspace missing
#   5  archive destination already exists (refuse to clobber)
set -euo pipefail

slug="${1:-}"
dv="${2:-.devrites}"

if [ -z "$slug" ]; then
  echo "close-out: usage: close-out.sh <slug> [devrites-dir]" >&2
  exit 4
fi

work="$dv/work/$slug"
arch="$dv/archive/$slug"

if [ ! -d "$work" ]; then
  echo "close-out: no workspace at $work" >&2
  exit 4
fi
if [ -e "$arch" ]; then
  echo "close-out: archive already exists at $arch — refusing to clobber" >&2
  exit 5
fi

mkdir -p "$dv/archive"
mv "$work" "$arch"

# Clear ACTIVE only if it still points at the slug we just archived.
if [ -f "$dv/ACTIVE" ] && [ "$(tr -d '[:space:]' < "$dv/ACTIVE")" = "$slug" ]; then
  : > "$dv/ACTIVE"
  echo "close-out: archived $slug -> $arch and cleared ACTIVE"
else
  echo "close-out: archived $slug -> $arch (ACTIVE pointed elsewhere — left as-is)"
fi
exit 0
