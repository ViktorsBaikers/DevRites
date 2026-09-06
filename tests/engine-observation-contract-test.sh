#!/usr/bin/env bash
# Golden CLI transcript contract for workspace observation commands.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
ENGINE="${DEVRITES_ENGINE_CLI:-}"
ENGINE_TMP=""
cleanup() {
  rm -rf "${PROJECT:-}"
  if [ -n "$ENGINE_TMP" ] && [ -f "$ENGINE_TMP" ]; then
    rm -f "$ENGINE_TMP"
  fi
}
trap cleanup EXIT

if [ -z "$ENGINE" ] || [ ! -x "$ENGINE" ]; then
  if command -v go >/dev/null 2>&1 && [ -d "$ROOT/engine" ]; then
    ENGINE_TMP="$(mktemp)"
    ENGINE="$ENGINE_TMP"
    (cd "$ROOT/engine" && CGO_ENABLED=0 go build -trimpath -o "$ENGINE" .)
  else
    echo "  SKIP: no DEVRITES_ENGINE_CLI and cannot build engine"
    exit 0
  fi
fi

PROJECT="$(mktemp -d)"
SLUG=add-csv-export
WS="$PROJECT/.devrites/work/$SLUG"
mkdir -p "$WS"
cp -R "$ROOT/evals/golden/shippable-feature/." "$WS/"
printf '%s\n' "$SLUG" >"$PROJECT/.devrites/ACTIVE"

orient="$(
  DEVRITES_ROOT="$PROJECT" "$ENGINE" orient "$SLUG" 2>/dev/null
)"
observe="$(
  DEVRITES_ROOT="$PROJECT" "$ENGINE" observe summary "$SLUG" 2>/dev/null
)"
if [ "$orient" != "$observe" ]; then
  echo "FAIL: orient and observe summary differ"
  exit 1
fi

python3 - "$orient" <<'PY'
import json
import sys

summary = json.loads(sys.argv[1])
for key in ("slug", "phase", "status", "next_action"):
    if key not in summary:
        raise SystemExit(f"missing summary key: {key}")
if summary["slug"] != "add-csv-export":
    raise SystemExit(f"slug={summary['slug']!r}")
if summary["phase"] != "done":
    raise SystemExit(f"phase={summary['phase']!r}")
PY

mkdir -p "$PROJECT/.claude" "$PROJECT/.codegraph"
printf 'devrites\n' >"$PROJECT/.claude/devrites.manifest"
indexes="$(
  DEVRITES_ROOT="$PROJECT" "$ENGINE" check indexes --root "$PROJECT" 2>/dev/null
)"
python3 - "$indexes" <<'PY'
import json
import sys

report = json.loads(sys.argv[1])
if not report.get("engine_version"):
    raise SystemExit("missing engine_version")
manifest = report.get("manifest") or {}
if not manifest.get("present"):
    raise SystemExit("manifest not present")
indexes = report.get("indexes") or []
paths = [row.get("path") for row in indexes]
want = [".codegraph", ".code-review-graph", ".codebase-memory"]
if paths != want:
    raise SystemExit(f"indexes paths={paths!r}, want {want!r}")
if not indexes[0].get("present"):
    raise SystemExit(".codegraph should be present")
PY

echo "engine-observation-contract: PASS"
