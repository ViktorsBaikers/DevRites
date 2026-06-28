#!/usr/bin/env bash
# install-smoke.sh — dry-run, temp install, idempotent reinstall, flag behavior,
# and the no-global-write guard. Exits non-zero on any failure.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok()   { printf '  ok: %s\n' "$*"; }
no()   { printf '  FAIL: %s\n' "$*"; fail=1; }
T="$(mktemp -d)"
cleanup() { rm -rf "$T"; }
trap cleanup EXIT

echo "== install-smoke (target: $T) =="

# 1) dry-run writes nothing
bash "$ROOT/install.sh" --target "$T" --dry-run >/dev/null 2>&1
[ -e "$T/.claude" ] && no "dry-run created .claude" || ok "dry-run changed nothing"

# 2) real install
bash "$ROOT/install.sh" --target "$T" >/dev/null 2>&1 || no "install exited non-zero"
[ -f "$T/.claude/devrites.manifest" ] && ok "manifest written" || no "no manifest"
for f in \
  ".claude/skills/rite/SKILL.md" \
  ".claude/skills/rite-define/SKILL.md" \
  ".claude/skills/rite-seal/SKILL.md" \
  ".claude/skills/rite-polish/SKILL.md" \
  ".claude/skills/rite-polish/reference/code.md" \
  ".claude/skills/rite-polish/reference/ui.md" \
  ".claude/skills/rite-pressure-test/SKILL.md" \
  ".claude/skills/devrites-doubt/SKILL.md" \
  ".claude/skills/devrites-lib/reference/parallel-dispatch.md" \
  ".claude/skills/devrites-lib/scripts/rites.sh" \
  ".claude/skills/devrites-frontend-craft/reference/shape.md" \
  ".claude/skills/devrites-debug-recovery/reference/build-the-loop.md" \
  ".claude/agents/devrites-code-reviewer.md" \
  ".claude/agents/devrites-spec-reviewer.md" \
  ".claude/rules/security.md" \
  ".claude/rules/anti-patterns.md" \
  ".claude/rules/README.md" \
  ".devrites/README.md" \
  ".devrites/ACTIVE" ; do
  [ -f "$T/$f" ] && ok "present: $f" || no "missing: $f"
done

# 2b) deleted/merged/renamed skills must NOT be present
for f in \
  ".claude/skills/rite-polish-code/SKILL.md" \
  ".claude/skills/rite-polish-ui/SKILL.md" \
  ".claude/skills/devrites-context-pack/SKILL.md" \
  ".claude/skills/devrites-selector/SKILL.md" \
  ".claude/skills/devrites-idea-refine/SKILL.md" \
  ".claude/skills/idea-pressure-test/SKILL.md" \
  ".claude/skills/devrites-prototype/SKILL.md" \
  ".claude/skills/devrites-handoff/SKILL.md" \
  ".claude/skills/devrites-zoom-out/SKILL.md" \
  ".claude/skills/devrites-code-simplifier/SKILL.md" \
  ".claude/skills/devrites-security-hardening/SKILL.md" \
  ".claude/skills/devrites-performance-check/SKILL.md" \
  ".claude/skills/devrites-parallel-review/SKILL.md" \
  ".claude/skills/devrites-rules/SKILL.md" ; do
  [ -f "$T/$f" ] && no "stale skill still installed: $f" || ok "removed: $f"
done

# /polish and /normalize aliases are removed in favor of /rite-polish modes.
[ -d "$T/.claude/skills/polish" ]    && no "stale /polish alias still installed"    || ok "no /polish alias (removed)"
[ -d "$T/.claude/skills/normalize" ] && no "stale /normalize alias still installed" || ok "no /normalize alias (removed)"

# 3) manifest count sanity (>= 80 files) and ACTIVE not manifest-managed
n="$(grep -vc '^#' "$T/.claude/devrites.manifest")"
[ "$n" -ge 80 ] && ok "manifest lists $n files" || no "manifest too small ($n)"
grep -q '\.devrites/ACTIVE' "$T/.claude/devrites.manifest" && no "ACTIVE should NOT be manifest-managed" || ok "ACTIVE excluded from manifest"

# 4) no global write
[ -e "$HOME/.claude/skills/rite" ] && no "wrote to ~/.claude !!" || ok "~/.claude untouched"

# 5) idempotent reinstall
out="$(bash "$ROOT/install.sh" --target "$T" 2>&1)"
echo "$out" | grep -q 'installed: 0' && ok "reinstall installs 0 new (idempotent)" || no "reinstall not idempotent"
echo "$out" | grep -q 'skipped(conflict): 0' && ok "reinstall skips 0 (all managed)" || no "reinstall reported conflicts"

# 6) --no-agents
T2="$(mktemp -d)"; bash "$ROOT/install.sh" --target "$T2" --no-agents >/dev/null 2>&1
[ -d "$T2/.claude/agents" ] && no "--no-agents still installed agents" || ok "--no-agents skipped agents"
[ -f "$T2/.claude/skills/rite-build/SKILL.md" ] && ok "--no-agents still installs skills" || no "--no-agents broke skills"
rm -rf "$T2"

# 7) --short-aliases=all installs /define /build /prove /seal wrappers
T3="$(mktemp -d)"; bash "$ROOT/install.sh" --target "$T3" --short-aliases=all >/dev/null 2>&1
[ -f "$T3/.claude/skills/define/SKILL.md" ] && ok "--short-aliases=all installs /define" || no "--short-aliases=all missing /define"
[ -f "$T3/.claude/skills/build/SKILL.md" ]  && ok "--short-aliases=all installs /build"  || no "--short-aliases=all missing /build"
rm -rf "$T3"

# 7b) --no-rules
T4="$(mktemp -d)"; bash "$ROOT/install.sh" --target "$T4" --no-rules >/dev/null 2>&1
[ -d "$T4/.claude/rules" ] && no "--no-rules still installed rules" || ok "--no-rules skipped rules"
[ -f "$T4/.claude/skills/rite-build/SKILL.md" ] && ok "--no-rules still installs skills" || no "--no-rules broke skills"
# default install DOES include rules
T5="$(mktemp -d)"; bash "$ROOT/install.sh" --target "$T5" >/dev/null 2>&1
[ -f "$T5/.claude/rules/security.md" ] && ok "default install includes DevRites rules" || no "default missing rules"
rm -rf "$T4" "$T5"

# 8) static guard script
bash "$ROOT/scripts/check-no-global-writes.sh" >/dev/null 2>&1 && ok "check-no-global-writes passed" || no "check-no-global-writes failed"

echo ""
[ "$fail" -eq 0 ] && echo "install-smoke: PASS" || echo "install-smoke: FAIL"
exit "$fail"
