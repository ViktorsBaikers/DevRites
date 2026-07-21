#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

node "$ROOT/scripts/check-instruction-size-baseline.mjs" >/tmp/devrites_size.out
grep -q 'instruction-size:' /tmp/devrites_size.out

echo "ok: instruction size guard"
