#!/usr/bin/env bash
# CONTRIBUTING.md must name every command the ci.yml validate job runs.
# usage: contributing-ci-parity-test.sh [repo-root]
set -euo pipefail

ROOT="${1:-$(cd "$(dirname "$0")/.." && pwd -P)}"
C="$ROOT/CONTRIBUTING.md"
Y="$ROOT/.github/workflows/ci.yml"
fail=0

# Commands between "  validate:" and "  tests:" in ci.yml, minus setup steps.
cmds=$(awk '/^  validate:$/{f=1} /^  tests:$/{f=0} f && /run:/{sub(/.*run: */,""); print}' "$Y" \
  | grep -vE 'pip install|npm ci|ci-install-validate-tools|devrites-detect' || true)
[ -n "$cmds" ] || { echo "FAIL no validate commands found in ci.yml"; exit 1; }

while IFS= read -r cmd; do
  key=$(grep -oE 'scripts/[a-z0-9_.-]+|npm run audit|osv-scanner' <<<"$cmd" | head -1 || true)
  if [ -z "$key" ]; then echo "SKIP $cmd"; continue; fi
  if grep -qF -- "$key" "$C"; then echo "ok   $key"; else echo "MISS $key"; fail=1; fi
done <<<"$cmds"

for k in 'golangci-lint' 'go test'; do
  if grep -qF -- "$k" "$C"; then echo "ok   $k"; else echo "MISS $k"; fail=1; fi
done

# Checklist item 7 must name the engine and security jobs.
item7=$(awk '/^7\. \*\*CI must be green/{f=1} f&&/^Draft PRs/{f=0} f' "$C")
if grep -q 'engine (go test' <<<"$item7"; then echo "ok   item7 engine"; else echo "MISS item7 engine"; fail=1; fi
if grep -qi 'security' <<<"$item7"; then echo "ok   item7 security"; else echo "MISS item7 security"; fail=1; fi

exit "$fail"
