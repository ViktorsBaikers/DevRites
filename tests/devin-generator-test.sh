#!/usr/bin/env bash
# Focused checks for the Claude-to-Devin generator used by host packaging.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

TMP_GEN_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_GEN_DIR"' EXIT

# shellcheck source=../scripts/devin-generate.sh
. "$ROOT/scripts/devin-generate.sh"

echo "== devin-generator-test =="

expect_tools() {
  local got="$1" want="$2" label="$3"
  [ "$got" = "$want" ] && ok "$label" || {
    no "$label"
    printf '    got:  %s\n    want: %s\n' "$got" "$want"
  }
}

# Claude tool names map onto Devin builtins; unknown names drop.
expect_tools \
  "$(_devin_map_tools "Read, Grep, Glob, Bash" "devrites-code-reviewer")" \
  "read, grep, glob, exec" \
  "reviewer tools map to Devin names"

expect_tools \
  "$(_devin_map_tools "Read, Edit, Write, Bash, Glob, Grep, Skill" "devrites-slice-wright")" \
  "read, edit, write, exec, glob, grep, skill" \
  "wright keeps write, exec, and skill"

expect_tools \
  "$(_devin_map_tools "Read, Grep, Glob, Bash, WebFetch, WebSearch" "devrites-evidence-scout")" \
  "read, grep, glob, exec, webfetch, web_search" \
  "evidence-scout keeps Devin web tools"

expect_tools \
  "$(_devin_map_tools "Read, Grep, Glob" "devrites-proof-runner")" \
  "read, grep, glob" \
  "proof-runner map stays read-only without exec"

expect_tools \
  "$(_devin_map_tools "" "devrites-slice-wright")" \
  "read, edit, write, exec, grep, glob, skill" \
  "empty tools falls back to the wright set"

expect_tools \
  "$(_devin_map_tools "" "devrites-code-reviewer")" \
  "read, grep, glob, exec" \
  "empty tools falls back to the reviewer set"

expect_tools \
  "$(_devin_map_tools "Notebook, Task, Agent" "devrites-code-reviewer")" \
  "read, grep, glob, exec" \
  "unmapped tools drop and fall back when nothing maps"

wright_src="$TMP_GEN_DIR/devrites-slice-wright.md"
reviewer_src="$TMP_GEN_DIR/devrites-code-reviewer.md"
proof_src="$TMP_GEN_DIR/devrites-proof-runner.md"
frontend_src="$TMP_GEN_DIR/devrites-frontend-reviewer.md"
cat >"$wright_src" <<'EOF'
---
name: devrites-slice-wright
description: Write-capable executor.
tools: Read, Edit, Write, Bash, Glob, Grep, Skill
permissionMode: acceptEdits
---

Body.
EOF
cat >"$reviewer_src" <<'EOF'
---
name: devrites-code-reviewer
description: Reviews a diff.
tools: Read, Grep, Glob, Bash
permissionMode: plan
---

Body.
EOF
cat >"$proof_src" <<'EOF'
---
name: devrites-proof-runner
description: Validates proof.
tools: Read, Grep, Glob
permissionMode: plan
---

Body.
EOF
cat >"$frontend_src" <<'EOF'
---
name: devrites-frontend-reviewer
description: Reviews UI.
tools: Read, Grep, Glob, Bash
skills:
  - devrites-frontend-craft
permissionMode: plan
---

Body.
EOF

gen_devin_agent "$wright_src" "$TMP_GEN_DIR/out-wright.md"
gen_devin_agent "$reviewer_src" "$TMP_GEN_DIR/out-reviewer.md"
gen_devin_agent "$proof_src" "$TMP_GEN_DIR/out-proof.md"
gen_devin_agent "$frontend_src" "$TMP_GEN_DIR/out-frontend.md"

for tool in read edit write exec grep glob skill; do
  grep -qx "  - $tool" "$TMP_GEN_DIR/out-wright.md" &&
    ok "wright allowed-tools includes $tool" ||
    no "wright allowed-tools missing $tool"
done
grep -q '^allowed-tools:' "$TMP_GEN_DIR/out-wright.md" &&
  ok "wright uses Devin allowed-tools field" ||
  no "wright missing allowed-tools field"

for f in "$TMP_GEN_DIR/out-wright.md" "$TMP_GEN_DIR/out-reviewer.md" "$TMP_GEN_DIR/out-proof.md" "$TMP_GEN_DIR/out-frontend.md"; do
  if grep -qE 'permissionMode|^tools:|^  - (Read|Edit|Write|Bash|Glob|Grep|Skill|WebFetch|WebSearch)$' <(awk 'NR==1&&$0=="---"{fm=1;next} fm&&$0=="---"{exit} fm{print}' "$f"); then
    no "$(basename "$f") leaks Claude frontmatter"
  else
    ok "$(basename "$f") frontmatter is Devin-native"
  fi
done

readonly_ok=1
for f in "$TMP_GEN_DIR/out-reviewer.md" "$TMP_GEN_DIR/out-proof.md"; do
  for bad in edit write; do
    grep -qx "  - $bad" "$f" && no "$(basename "$f") gained $bad" && readonly_ok=0
  done
done
[ "$readonly_ok" -eq 1 ] && ok "read-only generated agents stay read-only"

grep -qx "  - skill" "$TMP_GEN_DIR/out-frontend.md" &&
  ok "canonical skills: block grants the skill tool" ||
  no "canonical skills: block did not grant the skill tool"

# Skill trigger translation: Claude invocation flags become Devin triggers.
mk_skill() {
  local _dir="$1" _ui="$2" _dmi="$3"
  mkdir -p "$_dir"
  {
    printf '%s\n' "---"
    printf 'name: %s\n' "$(basename "$_dir")"
    printf 'description: test\n'
    [ -n "$_ui" ] && printf 'user-invocable: %s\n' "$_ui"
    [ -n "$_dmi" ] && printf 'disable-model-invocation: %s\n' "$_dmi"
    printf '%s\n' "---"
    printf 'Body.\n'
  } >"$_dir/SKILL.md"
}

mk_skill "$TMP_GEN_DIR/sk-both" true ""
gen_devin_skill_file "$TMP_GEN_DIR/sk-both/SKILL.md" "$TMP_GEN_DIR/out-sk-both.md"
if grep -q '^triggers:' "$TMP_GEN_DIR/out-sk-both.md"; then
  no "default skill gained a triggers restriction"
else
  ok "default skill keeps Devin default triggers"
fi

mk_skill "$TMP_GEN_DIR/sk-user" true true
gen_devin_skill_file "$TMP_GEN_DIR/sk-user/SKILL.md" "$TMP_GEN_DIR/out-sk-user.md"
grep -qx '  - user' "$TMP_GEN_DIR/out-sk-user.md" &&
  ! grep -qx '  - model' "$TMP_GEN_DIR/out-sk-user.md" &&
  ok "explicit-only skill becomes triggers: [user]" ||
  no "explicit-only skill triggers wrong"

mk_skill "$TMP_GEN_DIR/sk-model" false ""
gen_devin_skill_file "$TMP_GEN_DIR/sk-model/SKILL.md" "$TMP_GEN_DIR/out-sk-model.md"
grep -qx '  - model' "$TMP_GEN_DIR/out-sk-model.md" &&
  ! grep -qx '  - user' "$TMP_GEN_DIR/out-sk-model.md" &&
  ok "model-only skill becomes triggers: [model]" ||
  no "model-only skill triggers wrong"

mk_skill "$TMP_GEN_DIR/sk-none" false true
gen_devin_skill_file "$TMP_GEN_DIR/sk-none/SKILL.md" "$TMP_GEN_DIR/out-sk-none.md"
grep -qx 'triggers: \[\]' "$TMP_GEN_DIR/out-sk-none.md" &&
  ok "library skill becomes triggers: []" ||
  no "library skill triggers wrong"

for f in "$TMP_GEN_DIR/out-sk-both.md" "$TMP_GEN_DIR/out-sk-user.md" "$TMP_GEN_DIR/out-sk-model.md" "$TMP_GEN_DIR/out-sk-none.md"; do
  if grep -qE 'user-invocable|disable-model-invocation' "$f"; then
    no "$(basename "$f") leaks Claude invocation fields"
  else
    ok "$(basename "$f") drops Claude invocation fields"
  fi
done

# The generated artifacts must be loadable by devin CLI when it is installed.
if command -v devin >/dev/null 2>&1; then
  proj="$TMP_GEN_DIR/devin-proj"
  mkdir -p "$proj/.devin/skills" "$proj/.devin/agents"
  cp -R "$ROOT/pack/generated/devin/skills/." "$proj/.devin/skills/" 2>/dev/null || true
  cp "$ROOT"/pack/generated/devin/agents/*.md "$proj/.devin/agents/" 2>/dev/null || true
  if [ -d "$proj/.devin/skills/rite" ] && [ -f "$proj/.devin/agents/devrites-slice-wright.md" ]; then
    if (cd "$proj" && devin doctor 2>&1 | grep -q "17 profile(s) loaded"); then
      ok "devin doctor loads all 17 generated profiles"
    else
      no "devin doctor did not load all 17 generated profiles"
    fi
    if (cd "$proj" && devin skills list --json 2>/dev/null | grep -q '"name": "rite-build"'); then
      ok "devin skills list discovers generated skills"
    else
      no "devin skills list missing generated skills"
    fi
  else
    ok "devin probe skipped (generated payload not built)"
  fi
else
  ok "devin probe skipped (devin CLI not installed)"
fi

[ "$fail" -eq 0 ] && echo "devin-generator-test: PASS" || echo "devin-generator-test: FAIL"
exit "$fail"
