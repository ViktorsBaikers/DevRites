#!/usr/bin/env bash
# validate-pack.sh: run the static pack validation. Thin wrapper around
# scripts/validate.sh so the test harness has a single entry point.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
exec bash "$ROOT/scripts/validate.sh"
