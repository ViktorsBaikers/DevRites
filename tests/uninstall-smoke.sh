#!/usr/bin/env bash
# uninstall-smoke.sh — install into a temp project, then uninstall, asserting that
# DevRites files are removed, empty dirs pruned, and runtime state preserved.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }
T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT

echo "== uninstall-smoke (target: $T) =="
bash "$ROOT/install.sh" --target "$T" >/dev/null 2>&1 || no "install failed"
# simulate in-progress feature data that must be preserved
mkdir -p "$T/.devrites/work/demo"; echo "phase: build" > "$T/.devrites/work/demo/state.md"
printf 'demo\n' > "$T/.devrites/ACTIVE"

# dry-run uninstall changes nothing
bash "$ROOT/uninstall.sh" --target "$T" --dry-run >/dev/null 2>&1
[ -f "$T/.claude/devrites.manifest" ] && ok "dry-run uninstall kept manifest" || no "dry-run removed files"

# real uninstall
bash "$ROOT/uninstall.sh" --target "$T" >/dev/null 2>&1 || no "uninstall exited non-zero"
[ -e "$T/.claude/devrites.manifest" ] && no "manifest not removed" || ok "manifest removed"
# .claude is pruned of all manifest-managed DevRites content. The seeded
# settings.json is intentionally NOT manifested (install.sh seeds it once and
# preserves it on uninstall/update so the user's hooks survive), so tolerate it
# as the only legitimate survivor — anything else left is a prune leak.
if [ -d "$T/.claude" ]; then
  leftover="$(find "$T/.claude" -mindepth 1 ! -path "$T/.claude/settings.json")"
  [ -z "$leftover" ] && ok ".claude pruned (seeded settings.json preserved)" \
    || no ".claude has leftover DevRites content: $leftover"
else
  ok ".claude fully pruned"
fi
# runtime state preserved
[ -f "$T/.devrites/ACTIVE" ] && ok "ACTIVE preserved" || no "ACTIVE wrongly removed"
[ -f "$T/.devrites/work/demo/state.md" ] && ok "work/ feature data preserved" || no "work/ wrongly removed"
# scaffolding README removed (manifest-managed)
[ -f "$T/.devrites/README.md" ] && no ".devrites/README not removed" || ok ".devrites/README removed"

# uninstall with no manifest must error cleanly
T2="$(mktemp -d)"
bash "$ROOT/uninstall.sh" --target "$T2" >/dev/null 2>&1 && no "uninstall succeeded with no manifest" || ok "uninstall errors without manifest"
rm -rf "$T2"

# foreign file is never removed
T3="$(mktemp -d)"; bash "$ROOT/install.sh" --target "$T3" >/dev/null 2>&1
echo "mine" > "$T3/.claude/skills/rite/USER_NOTE.txt"
bash "$ROOT/uninstall.sh" --target "$T3" >/dev/null 2>&1
[ -f "$T3/.claude/skills/rite/USER_NOTE.txt" ] && ok "foreign file preserved (dir not nuked)" || no "foreign file removed!"
rm -rf "$T3"

echo ""
[ "$fail" -eq 0 ] && echo "uninstall-smoke: PASS" || echo "uninstall-smoke: FAIL"
exit "$fail"
