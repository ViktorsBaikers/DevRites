#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

mkdir -p "$TMP/skills/a" "$TMP/skills/b"
printf -- '---\nname: a\ndescription: A.\n---\nCall /b.\n' > "$TMP/skills/a/SKILL.md"
printf -- '---\nname: b\ndescription: B.\n---\n' > "$TMP/skills/b/SKILL.md"
printf '{"profiles":{"core":["a"]}}\n' > "$TMP/surface.json"
node "$ROOT/scripts/check-skill-deps.mjs" --skills-dir "$TMP/skills" --manifest "$TMP/surface.json" >/tmp/devrites_skill_deps.out

grep -q '2 skills checked' /tmp/devrites_skill_deps.out

node "$ROOT/scripts/check-instruction-size-baseline.mjs" >/tmp/devrites_size.out
grep -q 'instruction-size:' /tmp/devrites_size.out

echo "ok: skill surface and instruction size guards"
