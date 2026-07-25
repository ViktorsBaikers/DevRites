#!/usr/bin/env bash
# validate.sh: static validation of the DevRites pack (no install required).
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

section "generated skill payload budget"
if command -v node >/dev/null 2>&1; then
  if node "$ROOT/scripts/check-generated-skill-budget.mjs" "$SKILLS"; then good "generated skill payload budget passed"; else bad "generated skill payload budget failed"; fi
else
  echo "skip: node not found"
fi

section "instruction size ratchet"
if command -v node >/dev/null 2>&1; then
  if node "$ROOT/scripts/check-instruction-size-baseline.mjs"; then good "instruction size baseline passed"; else bad "instruction size baseline failed"; fi
else
  echo "skip: node not found"
fi

section "reference reachability governance"
if command -v node >/dev/null 2>&1; then
  if node "$ROOT/scripts/check-reference-governance.mjs"; then good "reference reachability governance passed"; else bad "reference reachability governance failed"; fi
else
  echo "skip: node not found"
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

# ---- 6c. skill anatomy + routing + host parity ----------------------------
section "skill anatomy"
if command -v python3 >/dev/null 2>&1; then
  if python3 "$ROOT/scripts/validate-skill-anatomy.py" >/tmp/dr_skill_anatomy 2>&1; then cat /tmp/dr_skill_anatomy; good "skill anatomy contracts passed"; else cat /tmp/dr_skill_anatomy; bad "skill anatomy validation failed"; fi
else
  echo "skip: python3 not found"
fi

section "deterministic routing/collision evals"
if command -v python3 >/dev/null 2>&1; then
  if python3 "$ROOT/scripts/run-routing-evals.py" >/tmp/dr_routing_evals 2>&1; then cat /tmp/dr_routing_evals; good "routing/collision evals passed"; else cat /tmp/dr_routing_evals; bad "routing/collision evals failed"; fi
else
  echo "skip: python3 not found"
fi

section "command host parity"
if command -v python3 >/dev/null 2>&1; then
  if python3 "$ROOT/scripts/validate-command-parity.py" >/tmp/dr_command_parity 2>&1; then cat /tmp/dr_command_parity; good "command host parity passed"; else cat /tmp/dr_command_parity; bad "command host parity failed"; fi
else
  echo "skip: python3 not found"
fi

section "agent composition"
if command -v python3 >/dev/null 2>&1; then
  if python3 "$ROOT/scripts/validate-agent-composition.py" >/tmp/dr_agent_composition 2>&1; then cat /tmp/dr_agent_composition; good "agent composition contracts passed"; else cat /tmp/dr_agent_composition; bad "agent composition validation failed"; fi
else
  echo "skip: python3 not found"
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

# ---- 8. skill pruning + step contracts ----------------------------------
section "skill pruning + step contracts"
if command -v node >/dev/null 2>&1 && [ -f "$ROOT/scripts/skill-pruning-audit.mjs" ]; then
  if node "$ROOT/scripts/skill-pruning-audit.mjs"; then good "skill pruning and step contracts passed"; else bad "skill pruning step contracts failed"; fi
else
  echo "skip: node or skill-pruning-audit.mjs not found"
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

# ---- 11. principle uniqueness: each canonical heading appears exactly once
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
if command -v go >/dev/null 2>&1; then
  if (cd "$ROOT/engine" && go run ./internal/state/cmd/workflowmanifest -check -out internal/state/workflow_manifest.json) >/tmp/dr_workflow_manifest 2>&1; then
    good "workflow manifest is fresh"
  else
    cat /tmp/dr_workflow_manifest
    bad "workflow manifest drifted from the typed state registry"
  fi
else
  echo "skip: go not found; workflow manifest freshness not checked"
fi
if command -v python3 >/dev/null 2>&1; then
  if python3 "$ROOT/scripts/check-authority-drift.py" >/tmp/dr_authority_drift 2>&1; then
    cat /tmp/dr_authority_drift
    good "authority-derived docs are current"
  else
    cat /tmp/dr_authority_drift
    bad "authority-derived docs drifted"
  fi
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
# Installed paths omit the leading pack/. A literal pack/.claude/skills/ path
# in shipped skill or reference prose will not resolve. Repository README and
# documentation links are GitHub links, so this check ignores them.
section "no literal pack/.claude/ paths in shipped skill prose"
# Keep the resolution-snippet fallback (`... || P=pack/.claude/...`). It checks
# the installed `.claude/` path first, then `${CLAUDE_SKILL_DIR}` as a
# best-effort plugin path, and finally the repository path during development.
PACKPATH_HITS="$(grep -rnI -e 'pack/\.claude/skills/devrites-lib/reference/standards/' -e 'pack/\.claude/skills/' "$SKILLS" 2>/dev/null | grep -vE '\|\| [A-Z]+=pack/\.claude/skills/' || true)"
if [ -n "$PACKPATH_HITS" ]; then
  bad "literal pack/.claude/ path in shipped skill prose (strips to .claude/ on install):"
  printf '%s\n' "$PACKPATH_HITS" | sed "s|$ROOT/||"
else
  good "no literal pack/.claude/skills/devrites-lib/reference/standards/ or pack/.claude/skills/ in shipped skill prose"
fi

# ---- 14. no false session-start autoload claim ---------------------------
# DevRites has no autoload wiring. Skills read
# .claude/skills/devrites-lib/reference/standards/core.md at step 0. Reject any
# shipped skill or document that claims native session-start autoloading.
section "no false session-start autoload claim"
AUTOLOAD_HITS="$(grep -rl 'autoloaded by Claude Code' "$ROOT/pack" "$ROOT/docs" "$ROOT/README.md" 2>/dev/null || true)"
if [ -n "$AUTOLOAD_HITS" ]; then
  bad "false 'autoloaded by Claude Code' claim: the pack ships no autoload wiring:"
  printf '%s\n' "$AUTOLOAD_HITS" | sed "s|$ROOT/||"
else
  good "no false session-start autoload claim in pack/ docs/ README.md"
fi

# ---- 14b. no deleted shell-helper guidance -------------------------------
# Public docs and generated installer guidance must use the installed
# `devrites-engine` binary, not the retired devrites-lib/*.sh helpers.
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
# CI runners include shellcheck and enforce the error-level gate on every PR.
# Local validation skips this gate only when shellcheck is not installed.
section "shellcheck (-S error blocking · -S warning advisory)"
if command -v shellcheck >/dev/null 2>&1; then
  for f in $SH_LIST; do
    if shellcheck -S error "$f"; then good "shellcheck ${f#"$ROOT"/}"; else bad "shellcheck (error) ${f#"$ROOT"/}"; fi
  done
  # Warnings are advisory. Print them per file without failing the build.
  for f in $SH_LIST; do
    shellcheck -S warning "$f" >/dev/null 2>&1 || echo "  advisory (warning-level): ${f#"$ROOT"/}"
  done
else
  echo "skip: shellcheck not installed locally (optional: CI enforces the error-level gate)"
fi

# ---- summary -------------------------------------------------------------
printf '\n========================================\n'
if [ "$fail" -eq 0 ]; then printf 'VALIDATION PASSED\n'; else printf 'VALIDATION FAILED\n'; fi
exit "$fail"
