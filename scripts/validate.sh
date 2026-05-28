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

PUBLIC="rite rite-spec rite-define rite-plan rite-build rite-prove rite-polish rite-review rite-seal rite-status rite-resolve rite-pressure-test rite-zoom-out rite-prototype rite-handoff"
INTERNAL="devrites-interview devrites-source-driven devrites-doubt devrites-frontend-craft devrites-browser-proof devrites-debug-recovery devrites-api-interface devrites-audit"
AGENT_FILES="devrites-spec-reviewer devrites-code-reviewer devrites-test-analyst devrites-frontend-reviewer devrites-security-auditor devrites-performance-reviewer devrites-doubt-reviewer devrites-simplifier-reviewer"

# ---- 1. bash -n on every shell script ------------------------------------
section "bash syntax (bash -n)"
SH_LIST="$ROOT/install.sh $ROOT/uninstall.sh"
for f in "$ROOT"/scripts/*.sh "$ROOT"/tests/*.sh; do [ -f "$f" ] && SH_LIST="$SH_LIST $f"; done
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

# ---- 3. required skills exist, each with SKILL.md ------------------------
section "skills present + SKILL.md"
for s in $PUBLIC $INTERNAL; do
  if [ -f "$SKILLS/$s/SKILL.md" ]; then good "$s"; else bad "missing skill or SKILL.md: $s"; fi
done
# any stray skill dir without SKILL.md?
for d in "$SKILLS"/*/; do
  [ -f "${d}SKILL.md" ] || bad "skill dir without SKILL.md: ${d#$PACK/}"
done

# ---- 4. agents present ---------------------------------------------------
section "agents present"
for a in $AGENT_FILES; do
  if [ -f "$AGENTS/$a.md" ]; then good "$a"; else bad "missing agent: $a"; fi
done

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

# ---- 9. DevRites engineering rules present -------------------------------
section "DevRites rules present"
if [ -f "$ROOT/pack/.claude/rules/README.md" ] && [ -f "$ROOT/pack/.claude/rules/security.md" ]; then
  good "pack/.claude/rules present ($(find "$ROOT/pack/.claude/rules" -name '*.md' | wc -l | tr -d ' ') rule files)"
else
  bad "DevRites rules missing (need pack/.claude/rules/*.md)"
fi

# ---- 10. no global writes ------------------------------------------------
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

# ---- 12. shellcheck (advisory) -------------------------------------------
section "shellcheck (advisory)"
if command -v shellcheck >/dev/null 2>&1; then
  for f in $SH_LIST; do shellcheck -S warning "$f" || echo "  (shellcheck advisory only — not failing the build)"; done
else
  echo "skip: shellcheck not installed (optional)"
fi

# ---- summary -------------------------------------------------------------
printf '\n========================================\n'
if [ "$fail" -eq 0 ]; then printf 'VALIDATION PASSED\n'; else printf 'VALIDATION FAILED\n'; fi
exit "$fail"
