#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
if [[ -z "${DEVRITES_ENGINE_CLI:-}" ]]; then
  echo "shared-engine-version-test: SKIP (run through scripts/run-tests.mjs)"
  exit 0
fi

expected="$(node -e 'const fs = require("node:fs"); const {version} = JSON.parse(fs.readFileSync(process.argv[1], "utf8")); process.stdout.write(`v${version}`)' "$ROOT/package.json")"
actual="$("$DEVRITES_ENGINE_CLI" --version)"
if [[ "$actual" != "$expected" ]]; then
  echo "shared-engine-version-test: FAIL (got $actual, want $expected)" >&2
  exit 1
fi

echo "shared-engine-version-test: PASS ($actual)"
