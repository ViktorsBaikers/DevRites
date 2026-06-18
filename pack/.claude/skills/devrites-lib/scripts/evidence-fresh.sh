#!/usr/bin/env bash
# Evidence-freshness gate for DevRites — enforces the rule in
# rite-seal/reference/final-evidence.md ("evidence must post-date the code it
# proves; if any touched file is newer than the proof it is a NO-GO until
# re-proven") as a deterministic exit code, instead of a prose mtime comparison
# the model must remember to perform. Used by /rite-seal; usable by /rite-prove.
# Read-only.
#
# Usage: evidence-fresh.sh [slug]
#   slug — optional; defaults to .devrites/ACTIVE
#
# Compares the newest mtime among files referenced in touched-files.md
# (backtick-quoted paths that still exist) against evidence.md /
# browser-evidence.md. If any touched file is newer, the proof is stale.
#
# Exit codes:
#   0  fresh (evidence post-dates every touched file) — or nothing to compare
#   3  STALE — a touched file is newer than the proof → re-run /rite-prove
#   5  no workspace / no evidence to check

set -euo pipefail

slug="${1:-$(cat .devrites/ACTIVE 2>/dev/null || true)}"
d=".devrites/work/$slug"
if [ -z "$slug" ] || [ ! -d "$d" ]; then
  printf 'evidence-fresh: no active workspace (slug=%s)\n' "${slug:-<unset>}" >&2
  exit 5
fi

ev="$d/evidence.md"; bev="$d/browser-evidence.md"; tf="$d/touched-files.md"
if [ ! -f "$ev" ] && [ ! -f "$bev" ]; then
  printf 'evidence-fresh: no evidence.md / browser-evidence.md in %s — run /rite-prove\n' "$d" >&2
  exit 5
fi

# Portable mtime in epoch seconds: GNU coreutils `stat -c`, BSD/macOS `stat -f`.
mtime() { stat -c %Y "$1" 2>/dev/null || stat -f %m "$1" 2>/dev/null || echo 0; }

newest_ev=0
for f in "$ev" "$bev"; do
  [ -f "$f" ] || continue
  m=$(mtime "$f"); [ "$m" -gt "$newest_ev" ] && newest_ev=$m
done

if [ ! -f "$tf" ]; then
  printf 'evidence-fresh: no touched-files.md — nothing to compare (treating as fresh).\n'
  exit 0
fi

newest_code=0; newest_path=""
while IFS= read -r p; do
  [ -n "$p" ] && [ -f "$p" ] || continue
  m=$(mtime "$p")
  if [ "$m" -gt "$newest_code" ]; then newest_code=$m; newest_path="$p"; fi
done < <(grep -oE '`[^`]+`' "$tf" 2>/dev/null | tr -d '`')

if [ "$newest_code" -gt "$newest_ev" ]; then
  printf 'evidence-fresh: STALE — %s is newer than the proof. Re-run /rite-prove before GO.\n' "$newest_path" >&2
  exit 3
fi

printf 'evidence-fresh: OK — evidence post-dates every touched file.\n'
exit 0
