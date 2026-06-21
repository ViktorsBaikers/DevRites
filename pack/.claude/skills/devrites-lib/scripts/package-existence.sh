#!/usr/bin/env bash
# package-existence.sh — slopsquatting / hallucinated-dependency gate. Before a slice's
# new third-party imports are trusted, assert each one is actually DECLARED in the project
# manifest (not just imported) — the entry point for AI-hallucinated package names that
# don't exist or typo-squat a real one. Distinct from a known-bad-version (IOC) scanner:
# this catches the fabricated/undeclared dep. Manifest check is deterministic + offline;
# a registry-existence probe is best-effort and skipped without network.
#
# Usage: package-existence.sh [slug]   (slug defaults to .devrites/ACTIVE)
# Exit:  0 clean / nothing to check · 2 no workspace · 3 undeclared import found
set -u

slug="${1:-}"
[ -n "$slug" ] || slug="$(cat .devrites/ACTIVE 2>/dev/null || true)"
[ -n "$slug" ] || { echo "package-existence: no active workspace." >&2; exit 2; }
d=".devrites/work/$slug"
[ -d "$d" ] || { echo "package-existence: no workspace at $d." >&2; exit 2; }

root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
[ -n "$root" ] || { echo "package-existence: not a git repo — skipped." >&2; exit 0; }
base_tree="$(cat "$d/.reconcile-base" 2>/dev/null || true)"
ref="${base_tree:-HEAD}"

# Manifest text to grep declared deps against (JS/TS, Python, Go, Rust).
manifests=""
for m in package.json requirements.txt pyproject.toml Pipfile go.mod Cargo.toml; do
  [ -f "$root/$m" ] && manifests="$manifests $root/$m"
done
[ -n "$manifests" ] || { echo "package-existence: no recognized manifest — skipped." >&2; exit 0; }

declared() { grep -qiE "(^|[^[:alnum:]_-])$(printf '%s' "$1" | sed 's/[.[\*^$()+?{|]/\\&/g')([^[:alnum:]_-]|\$)" $manifests 2>/dev/null; }

# Newly-added import lines in the diff (lines starting with '+', not the +++ header).
added="$(git -C "$root" diff "$ref" 2>/dev/null | grep -E '^\+' | grep -vE '^\+\+\+')"

# Extract bare top-level package names from common import forms.
pkgs="$(printf '%s\n' "$added" | grep -oE \
  "from +['\"][^'\".]+|require\(['\"][^'\".]+|import +[A-Za-z0-9_]+|^\+[[:space:]]*import +[a-zA-Z0-9_]+" \
  2>/dev/null \
  | sed -E "s/.*['\"]//; s/^\+?[[:space:]]*(from|import|require\()?[[:space:]]*//" \
  | grep -oE '^[@A-Za-z0-9_][A-Za-z0-9_./-]*' \
  | sed -E 's#^(@[^/]+/[^/]+).*#\1#; s#^([^@][^/]+)/.*#\1#' \
  | grep -vE '^(\.|/|[A-Z]:)' \
  | sort -u)"

# Standard-library / local prefixes that never need a manifest entry.
STD='^(os|sys|re|json|math|time|datetime|typing|collections|itertools|functools|pathlib|subprocess|logging|fmt|errors|context|strings|strconv|net|http|io|bufio|std|core|alloc|crate|self|super|react|node:|fs|path|util|crypto|events|stream|child_process)$'

bad=0; report=""
while IFS= read -r p; do
  [ -n "$p" ] || continue
  printf '%s' "$p" | grep -qE "$STD" && continue
  if ! declared "$p"; then
    bad=$((bad+1)); report="${report}  - ${p}: imported but not declared in any manifest
"
  fi
done <<EOF
$pkgs
EOF

if [ "$bad" -gt 0 ]; then
  echo "package-existence: $bad imported package(s) are NOT declared in a manifest — verify each exists and add it via the package manager:" >&2
  printf '%s' "$report" >&2
  echo "An undeclared import is how hallucinated/typo-squatted packages slip in. Confirm the name on the registry before trusting it." >&2
  exit 3
fi
echo "package-existence: OK — every new third-party import is declared in a manifest."
exit 0
