#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
node "$ROOT/tests/readonly-workflow-test.mjs"
