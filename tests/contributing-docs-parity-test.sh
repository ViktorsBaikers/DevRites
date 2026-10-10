#!/usr/bin/env bash
# README <-> CONTRIBUTING anchors resolve, and the commit types/scopes documented in
# CONTRIBUTING.md and the commitlint workflow comment match commitlint.config.js.
# usage: contributing-docs-parity-test.sh [repo-root]
set -euo pipefail

ROOT="${1:-$(cd "$(dirname "$0")/.." && pwd -P)}"
R="$ROOT/README.md"
C="$ROOT/CONTRIBUTING.md"
L="$ROOT/commitlint.config.js"
W="$ROOT/.github/workflows/commitlint.yml"
fail=0

# README must link CONTRIBUTING.md; every README.md#anchor CONTRIBUTING links must be a README heading.
if grep -qF '](CONTRIBUTING.md)' "$R"; then echo "ok   README links CONTRIBUTING.md"; else echo "MISS README link to CONTRIBUTING.md"; fail=1; fi
slugs=$(grep -E '^#{1,6} ' "$R" | sed -E 's/^#+ +//' | tr 'A-Z' 'a-z' | sed -E 's/[^a-z0-9 _-]//g; s/ /-/g')
for a in $(grep -oE 'README\.md#[A-Za-z0-9_-]+' "$C" | sed 's/.*#//' | sort -u); do
  if grep -qxF -- "$a" <<<"$slugs"; then echo "ok   README anchor $a"; else echo "MISS README anchor $a"; fail=1; fi
done

# Sorted, one-per-line list of a config array (key = type-enum|scope-enum).
cfg() { awk -v k="'$1'" 'index($0,k){f=1} f&&/\[.*\]/&&!/^ *\[ *$/ && !/always/{print; exit}' "$L" \
  | grep -oE "'[a-z-]+'" | tr -d "'" | sort; }
# Items after a "<label>" marker, split on | or , (inline code spans / bullets).
docs() { sed -E "s/[\`|,']/ /g" | tr -s ' ' '\n' | grep -E '^[a-z][a-z-]*$' | sort -u; }

want_t=$(cfg type-enum); want_s=$(cfg scope-enum)
[ -n "$want_t" ] && [ -n "$want_s" ] || { echo "FAIL could not parse commitlint.config.js"; exit 1; }

got_ct=$(grep -A1 -F '**type**' "$C" | tail -1 | docs)
got_cs=$(grep -A1 -F '**scope**' "$C" | tail -1 | docs)
got_wt=$(grep -F -- '- `type` ∈' "$W" | sed 's/.*∈//' | docs)
got_ws=$(grep -F -- '- `scope` ∈' "$W" | sed 's/.*∈//' | docs)

cmp_list() { # name want got
  if [ "$2" = "$(sort -u <<<"$3")" ]; then echo "ok   $1"; else
    echo "FAIL $1"; echo "  config: $(tr '\n' ' ' <<<"$2")"; echo "  doc:    $(tr '\n' ' ' <<<"$3")"; fail=1; fi
}
cmp_list "CONTRIBUTING types" "$want_t" "$got_ct"
cmp_list "CONTRIBUTING scopes" "$want_s" "$got_cs"
cmp_list "commitlint.yml types" "$want_t" "$got_wt"
cmp_list "commitlint.yml scopes" "$want_s" "$got_ws"

exit "$fail"
