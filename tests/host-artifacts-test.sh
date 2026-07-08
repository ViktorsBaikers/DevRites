#!/usr/bin/env bash
# host-artifacts-test.sh — validate prebuilt Claude/Codex host artifacts.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT
OUT="$T/generated"

echo "== host-artifacts-test =="

DEVRITES_HOST_ARTIFACT_DIR="$OUT" bash "$ROOT/scripts/build-host-artifacts.sh" >/dev/null 2>&1 \
  && ok "build-host-artifacts completed" \
  || no "build-host-artifacts failed"

for f in \
  "claude/skills/rite-build/SKILL.md" \
  "claude/agents/devrites-code-reviewer.md" \
  "claude/settings.json" \
  "codex/skills/rite-build/SKILL.md" \
  "codex/skills/rite-build/reference/wright-dispatch.md" \
  "codex/agents/devrites-code-reviewer.toml" \
  "codex/AGENTS.md" \
  "codex/hooks.json" \
  "README.md" ; do
  [ -f "$OUT/$f" ] && ok "artifact present: $f" || no "artifact missing: $f"
done

src_skills="$(find "$ROOT/pack/.claude/skills" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')"
claude_skills="$(find "$OUT/claude/skills" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')"
codex_skills="$(find "$OUT/codex/skills" -mindepth 1 -maxdepth 1 -type d | wc -l | tr -d ' ')"
[ "$claude_skills" = "$src_skills" ] && ok "Claude artifact skill count matches source" || no "Claude artifact skill count mismatch"
[ "$codex_skills" = "$src_skills" ] && ok "Codex artifact skill count matches source" || no "Codex artifact skill count mismatch"

grep -q '## Codex compatibility' "$OUT/codex/skills/rite-build/SKILL.md" \
  && ok "Codex skill artifact has compatibility block" \
  || no "Codex skill artifact missing compatibility block"
grep -q '.agents/skills/devrites-lib/reference/standards/core.md' "$OUT/codex/skills/rite-build/SKILL.md" \
  && ok "Codex skill artifact uses mirrored rules path" \
  || no "Codex skill artifact missing mirrored rules path"
grep -q '.codex/agents/devrites-slice-wright.toml' "$OUT/codex/skills/rite-build/SKILL.md" \
  && ok "Codex skill artifact references Codex agent TOML" \
  || no "Codex skill artifact missing Codex agent TOML"

if grep -R -nE 'pack/\.claude|\.claude/skills|\.claude/agents|(^|[^A-Za-z0-9_./-])/rite(-[a-z0-9-]+)?([^A-Za-z0-9_-]|$)' "$OUT/codex/skills" "$OUT/codex/agents" >/tmp/dr_host_artifacts_paths 2>/dev/null; then
  no "Codex artifacts contain stale Claude paths or slash invocations"
  sed -n '1,40p' /tmp/dr_host_artifacts_paths
else
  ok "Codex artifacts contain no stale Claude paths or slash invocations"
fi

if command -v python3 >/dev/null 2>&1; then
  python3 - "$OUT" <<'PY'
import json, pathlib, sys, tomllib
root = pathlib.Path(sys.argv[1])
tomllib.loads((root / "codex/agents/devrites-code-reviewer.toml").read_text())
json.loads((root / "codex/hooks.json").read_text())
PY
  rc="$?"
  if [ "$rc" -eq 0 ]; then
    ok "Codex agent TOML/hooks JSON artifacts parse"
  else
    if python3 - <<'PY' >/dev/null 2>&1
try:
    import tomllib  # noqa: F401
except ModuleNotFoundError:
    raise SystemExit(1)
PY
    then
      no "Codex agent TOML/hooks JSON artifacts invalid"
    else
      node -e "JSON.parse(require('fs').readFileSync(process.argv[1] + '/codex/hooks.json', 'utf8'))" "$OUT" \
        && ok "Codex hooks JSON parses; TOML parse skipped (python has no tomllib)" \
        || no "Codex hooks JSON artifact invalid"
    fi
  fi
else
  ok "Codex TOML/JSON parse skipped (python3 not found)"
fi

echo ""
[ "$fail" -eq 0 ] && echo "host-artifacts-test: PASS" || echo "host-artifacts-test: FAIL"
exit "$fail"
