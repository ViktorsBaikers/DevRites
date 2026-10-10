#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$(mktemp -d)"
trap 'rm -rf "$OUT"' EXIT

node "$ROOT/scripts/check-instruction-size-baseline.mjs" >"$OUT/devrites_size.out"
grep -q 'instruction-size:' "$OUT/devrites_size.out"

echo "ok: instruction size guard"
