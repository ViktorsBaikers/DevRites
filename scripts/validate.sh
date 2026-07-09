#!/usr/bin/env bash
# validate.sh — static validation of the DevRites pack (no install required).
# Run from anywhere. Exits non-zero if any hard check fails.

set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
PACK="$ROOT/pack/.claude"
SKILLS="$PACK/skills"
AGENTS="$PACK/agents"
fail=0
section() { printf '\n=== %s ===\n' "$1"; }
bad() { printf 'FAIL: %s\n' "$*"; fail=1; }
good() { printf 'ok: %s\n' "$*"; }


# ---- 1. bash -n on every shell script ------------------------------------
section "bash syntax (bash -n)"
SH_LIST="$ROOT/install.sh $ROOT/uninstall.sh $ROOT/update.sh"
for f in "$ROOT"/scripts/*.sh "$ROOT"/tests/*.sh "$ROOT"/pack/.claude/hooks/*.sh "$ROOT"/pack/.claude/skills/*/scripts/*.sh; do [ -f "$f" ] && SH_LIST="$SH_LIST $f"; done
for f in $SH_LIST; do
  if bash -n "$f" 2>/tmp/dr_synerr; then good "syntax ${f#$ROOT/}"; else bad "syntax ${f#$ROOT/}: $(cat /tmp/dr_synerr)"; fi
done

# ---- 2. python syntax ----------------------------------------------------
section "python syntax"
if command -v python3 >/dev/null 2>&1; then
  for f in "$ROOT"/scripts/*.py; do
    [ -f "$f" ] || continue
    if python3 -c "import py_compile,sys; py_compile.compile('$f', doraise=True)" 2>/tmp/dr_pyerr; then
      good "compiles ${f#$ROOT/}"; else bad "py ${f#$ROOT/}: $(cat /tmp/dr_pyerr)"; fi
  done
else
  echo "skip: python3 not found"
fi


# ---- 2d. shell install helper ownership ----------------------------------
section "shell install helper ownership"
INSTALL_LIB_HITS="$(grep -nE 'dr_(write_manifest|ver_gt|packaged_release_tag|strip_marker_block|merge_marker_block|strip_codex_|merge_codex_|codex_hooks_all_devrites)' "$ROOT/scripts/install-lib.sh" 2>/dev/null || true)"
if [ -n "$INSTALL_LIB_HITS" ]; then
  bad "scripts/install-lib.sh contains install/update/uninstall-era helpers:"
  printf '%s\n' "$INSTALL_LIB_HITS" | sed "s|$ROOT/||"
else
  good "scripts/install-lib.sh has no install semantics helpers; install semantics stay in devrites-engine"
fi

# ---- 3. required skills exist, each with SKILL.md ------------------------
section "skills present + SKILL.md"
skill_count=0
for d in "$SKILLS"/*/; do
  [ -d "$d" ] || continue
  skill_count=$((skill_count + 1))
  [ -f "${d}SKILL.md" ] && good "$(basename "$d")" || bad "skill dir without SKILL.md: ${d#$PACK/}"
done
[ "$skill_count" -gt 0 ] || bad "no skills found in ${SKILLS#$ROOT/}"

# ---- 4. agents present ---------------------------------------------------
section "agents present"
agent_count=0
for agent_file in "$AGENTS"/*.md; do
  [ -f "$agent_file" ] || continue
  agent_count=$((agent_count + 1))
  good "$(basename "$agent_file" .md)"
done
[ "$agent_count" -gt 0 ] || bad "no agents found in ${AGENTS#$ROOT/}"

# ---- 5. frontmatter validation ------------------------------------------
section "frontmatter"
if command -v python3 >/dev/null 2>&1; then
  FM_FILES=""
  for d in "$SKILLS"/*/; do FM_FILES="$FM_FILES ${d}SKILL.md"; done
  for a in "$AGENTS"/*.md; do FM_FILES="$FM_FILES $a"; done
  if python3 "$ROOT/scripts/validate-frontmatter.py" $FM_FILES; then good "frontmatter parses"; else bad "frontmatter validation failed"; fi
else
  echo "skip: python3 not found"
fi

# ---- 6. /rite-polish orchestrator references its phase reference files ---
section "rite-polish orchestrator → reference files"
for s in reference/code.md reference/ui.md; do
  if grep -q "$s" "$SKILLS/rite-polish/SKILL.md" 2>/dev/null; then good "rite-polish references $s"; else bad "rite-polish does not reference $s"; fi
  if [ -f "$SKILLS/rite-polish/$s" ]; then good "$s present"; else bad "$s missing"; fi
done

# ---- 6b. skill inventory / documentation counts --------------------------
section "skills inventory"
if command -v node >/dev/null 2>&1; then
  if node "$ROOT/scripts/skills-inventory.mjs" >/tmp/dr_skills_inventory 2>&1; then
    cat /tmp/dr_skills_inventory
    good "skills inventory matches docs"
  else
    cat /tmp/dr_skills_inventory
    bad "skills inventory drifted"
  fi
else
  echo "skip: node not found"
fi

# ---- 7. broken reference links -------------------------------------------
section "reference links resolve"
if command -v python3 >/dev/null 2>&1; then
  python3 - "$SKILLS" <<'PY' || fail=1
import os, re, sys
skills = sys.argv[1]
link = re.compile(r"\]\(([^)]+?\.md)\)")
broken = 0; checked = 0
for name in os.listdir(skills):
    sd = os.path.join(skills, name)
    sm = os.path.join(sd, "SKILL.md")
    if not os.path.isfile(sm):
        continue
    text = open(sm, encoding="utf-8").read()
    for m in link.finditer(text):
        tgt = m.group(1)
        if tgt.startswith("http"):
            continue
        checked += 1
        cands = [
            os.path.normpath(os.path.join(sd, tgt)),      # skill-dir relative
            os.path.normpath(os.path.join(skills, tgt)),  # skills-root relative
        ]
        if not any(os.path.isfile(c) for c in cands):
            print("FAIL: broken link in %s/SKILL.md -> %s" % (name, tgt)); broken += 1
print("checked %d links, %d broken" % (checked, broken))
sys.exit(1 if broken else 0)
PY
  [ "$fail" -eq 0 ] && good "all reference links resolve" || true
else
  echo "skip: python3 not found"
fi

# ---- 8. size advisory (not a hard failure) -------------------------------
section "SKILL.md size advisory"
for d in "$SKILLS"/*/; do
  n=$(wc -l < "${d}SKILL.md" | tr -d ' ')
  [ "$n" -gt 200 ] && printf 'warn: %s is %s lines (recommended <=180 public / <=160 internal)\n' "$(basename "$d")" "$n"
done
echo "ok: size advisory complete"

# ---- 8b. skill pruning audit (advisory) ----------------------------------
section "skill pruning audit (advisory)"
if command -v node >/dev/null 2>&1; then
  node "$ROOT/scripts/skill-pruning-audit.mjs" || true
  good "skill pruning audit complete"
else
  echo "skip: node not found"
fi

# ---- 9. DevRites engineering rules present -------------------------------
section "DevRites rules present"
if [ -f "$ROOT/pack/.claude/skills/devrites-lib/reference/standards/README.md" ] && [ -f "$ROOT/pack/.claude/skills/devrites-lib/reference/standards/security.md" ]; then
  good "pack/.claude/skills/devrites-lib/reference/standards present ($(find "$ROOT/pack/.claude/skills/devrites-lib/reference/standards" -name '*.md' | wc -l | tr -d ' ') markdown files)"
else
  bad "DevRites rules missing (need pack/.claude/skills/devrites-lib/reference/standards/*.md)"
fi

# ---- 10. no global writes ------------------------------------------------
section "no personal paths in shipped artifacts"
if command -v python3 >/dev/null 2>&1; then
  if python3 "$ROOT/scripts/check-no-personal-paths.py" >/tmp/dr_personal_paths 2>&1; then cat /tmp/dr_personal_paths; good "no personal paths check passed"; else cat /tmp/dr_personal_paths; bad "personal path check failed"; fi
else
  echo "skip: python3 not found"
fi

# ---- 10b. no global writes ------------------------------------------------
section "no global ~/.claude writes"
if bash "$ROOT/scripts/check-no-global-writes.sh" >/tmp/dr_glob 2>&1; then good "no-global-writes check passed"; else bad "no-global-writes check failed"; cat /tmp/dr_glob; fi

# ---- 11. principle uniqueness — each canonical heading appears exactly once
section "principle uniqueness"
if bash "$ROOT/scripts/check-rule-uniqueness.sh" >/tmp/dr_uniq 2>&1; then
  cat /tmp/dr_uniq
  good "rule-uniqueness check passed"
else
  cat /tmp/dr_uniq
  bad "rule-uniqueness check failed (see scripts/check-rule-uniqueness.sh)"
fi

# ---- 11b. generated workspace schema fixtures ----------------------------
section "workspace artifact schema"
if command -v python3 >/dev/null 2>&1; then
  if python3 "$ROOT/scripts/validate-workspace-schema.py" "$ROOT/tests/fixtures/workspace-schema" >/tmp/dr_workspace_schema 2>&1; then
    cat /tmp/dr_workspace_schema
    good "workspace artifact schema fixtures valid"
  else
    cat /tmp/dr_workspace_schema
    bad "workspace artifact schema fixtures failed"
  fi
else
  echo "skip: python3 not found"
fi

# ---- 11c. user-facing completion reply contract --------------------------
section "rite completion reply contract"
if bash "$ROOT/scripts/check-reply-contract.sh" >/tmp/dr_reply_contract 2>&1; then
  cat /tmp/dr_reply_contract
  good "reply-contract check passed"
else
  cat /tmp/dr_reply_contract
  bad "reply-contract check failed (see scripts/check-reply-contract.sh)"
fi

# ---- 12. no runtime-broken pack/.claude/ path in installed prose ---------
# After install the leading pack/ is stripped, so any literal pack/.claude/skills/devrites-lib/reference/standards/
# or pack/.claude/skills/ in shipped SKILL.md / reference prose is a dead path
# at runtime. (Repo README/docs links are out of scope — they're GitHub links.)
section "no literal pack/.claude/ paths in shipped skill prose"
# Exclude the intentional resolution-snippet fallback (`... || P=pack/.claude/...`): the
# preamble snippet tries the installed `.claude/` path first, then `${CLAUDE_SKILL_DIR}`
# (plugin, best-effort), then the repo `pack/.claude/...` for DevRites self-development.
PACKPATH_HITS="$(grep -rnI -e 'pack/\.claude/skills/devrites-lib/reference/standards/' -e 'pack/\.claude/skills/' "$SKILLS" 2>/dev/null | grep -vE '\|\| [A-Z]+=pack/\.claude/skills/' || true)"
if [ -n "$PACKPATH_HITS" ]; then
  bad "literal pack/.claude/ path in shipped skill prose (strips to .claude/ on install):"
  printf '%s\n' "$PACKPATH_HITS" | sed "s|$ROOT/||"
else
  good "no literal pack/.claude/skills/devrites-lib/reference/standards/ or pack/.claude/skills/ in shipped skill prose"
fi

# ---- 14. no false session-start autoload claim ---------------------------
# DevRites ships no autoload wiring; skills Read .claude/skills/devrites-lib/reference/standards/core.md at step 0.
# Fail if any shipped skill or doc asserts native/session-start autoload.
section "no false session-start autoload claim"
AUTOLOAD_HITS="$(grep -rl 'autoloaded by Claude Code' "$ROOT/pack" "$ROOT/docs" "$ROOT/README.md" 2>/dev/null || true)"
if [ -n "$AUTOLOAD_HITS" ]; then
  bad "false 'autoloaded by Claude Code' claim — the pack ships no autoload wiring:"
  printf '%s\n' "$AUTOLOAD_HITS" | sed "s|$ROOT/||"
else
  good "no false session-start autoload claim in pack/ docs/ README.md"
fi

# ---- 14b. no deleted shell-helper guidance -------------------------------
# The workflow control plane moved from devrites-lib/*.sh helpers to the
# installed `devrites-engine` binary. Public docs and generated installer
# guidance must not direct users or agents back to deleted helper files.
section "no deleted shell-helper guidance"
DELETED_HELPER_HITS="$(grep -rnI \
  -e 'analyze\.sh' \
  -e 'preamble\.sh' \
  -e 'readiness\.sh' \
  -e 'evidence-fresh\.sh' \
  -e 'check-acceptance\.sh' \
  -e 'coverage\.sh' \
  -e 'doubt-coverage\.sh' \
  -e 'footprint\.sh' \
  -e 'learnings\.sh' \
  -e 'mutation-gate\.sh' \
  -e 'package-existence\.sh' \
  -e 'progress\.sh' \
  -e 'reconcile\.sh' \
  -e 'tick-afk\.sh' \
  -e 'test-integrity\.sh' \
  -e 'resolve\.sh' \
  -e 'close-out\.sh' \
  -e 'stuck\.sh' \
  -e 'spec-validate\.sh' \
  -e 'devrites\.sh' \
  -e 'devrites-a1-guard\.sh' \
  -e 'devrites-allow\.sh' \
  -e 'devrites-cursor\.sh' \
  -e 'devrites-orient\.sh' \
  -e 'devrites-redwatch\.sh' \
  -e 'devrites-refresh-indexes\.sh' \
  -e 'devrites-reviewer-readonly\.sh' \
  -e 'devrites-statusline\.sh' \
  -e 'devrites-stop-gate\.sh' \
  -e 'devrites-wright-scope\.sh' \
  -e '\.codex/hooks/' \
  "$ROOT/README.md" "$ROOT/SECURITY.md" "$ROOT/docs" "$ROOT/evals" "$ROOT/install.sh" "$ROOT/b"*"in" 2>/dev/null || true)"
if [ -n "$DELETED_HELPER_HITS" ]; then
  bad "deleted helper/control-plane guidance found:"
  printf '%s\n' "$DELETED_HELPER_HITS" | sed "s|$ROOT/||"
else
  good "public/generated guidance points at the devrites-engine binary, not deleted shell helpers"
fi

# ---- 15. shellcheck (error = blocking, warning = advisory) ---------------
# CI runners ship shellcheck, so the error-level gate is enforced on every PR.
# Locally it self-skips when shellcheck is absent (the gate is non-blocking only
# where the tool isn't installed — never silently downgraded where it is).
section "shellcheck (-S error blocking · -S warning advisory)"
if command -v shellcheck >/dev/null 2>&1; then
  for f in $SH_LIST; do
    if shellcheck -S error "$f"; then good "shellcheck ${f#"$ROOT"/}"; else bad "shellcheck (error) ${f#"$ROOT"/}"; fi
  done
  # warning-level is informational — surfaced per file, never fails the build.
  for f in $SH_LIST; do
    shellcheck -S warning "$f" >/dev/null 2>&1 || echo "  advisory (warning-level): ${f#"$ROOT"/}"
  done
else
  echo "skip: shellcheck not installed locally (optional — CI enforces the error-level gate)"
fi

# ---- summary -------------------------------------------------------------
printf '\n========================================\n'
if [ "$fail" -eq 0 ]; then printf 'VALIDATION PASSED\n'; else printf 'VALIDATION FAILED\n'; fi
exit "$fail"
