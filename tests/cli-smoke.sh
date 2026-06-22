#!/usr/bin/env bash
# cli-smoke.sh — exercise the npx entry point (bin/devrites.mjs): version/help,
# subcommand routing, flag passthrough, and that it drives the bundled bash
# installers (install / uninstall) correctly. Exits non-zero on any failure.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
CLI="$ROOT/bin/devrites.mjs"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }
T="$(mktemp -d)"
cleanup() { rm -rf "$T"; }
trap cleanup EXIT

echo "== cli-smoke (target: $T) =="

command -v node >/dev/null 2>&1 || { echo "  FAIL: node not on PATH"; exit 1; }
[ -f "$CLI" ] || { echo "  FAIL: missing $CLI"; exit 1; }

# 1) --version matches package.json
want="$(node -e "process.stdout.write(require('$ROOT/package.json').version)")"
got="$(node "$CLI" --version 2>/dev/null)"
[ "$got" = "$want" ] && ok "--version reports $got" || no "--version ($got) != package.json ($want)"

# 2) --help shows usage + subcommands
help="$(node "$CLI" --help 2>/dev/null)"
echo "$help" | grep -q 'Usage:' && ok "--help shows usage" || no "--help missing usage"
echo "$help" | grep -q 'uninstall' && ok "--help lists uninstall" || no "--help missing uninstall"

# 3) default (bare) dry-run writes nothing
node "$CLI" --target "$T" --dry-run >/dev/null 2>&1 || no "dry-run exited non-zero"
[ -e "$T/.claude" ] && no "dry-run created .claude" || ok "dry-run changed nothing"

# 4) `add` alias routes to install (dry-run)
out="$(node "$CLI" add --target "$T" --dry-run 2>&1)"
echo "$out" | grep -q 'dry run' && ok "add alias routes to installer" || no "add alias did not run installer"
[ -e "$T/.claude" ] && no "add dry-run created .claude" || ok "add dry-run changed nothing"

# 5) real install via the CLI
node "$CLI" --target "$T" >/dev/null 2>&1 || no "install exited non-zero"
[ -f "$T/.claude/devrites.manifest" ] && ok "manifest written" || no "no manifest"
for f in \
  ".claude/skills/rite/SKILL.md" \
  ".claude/skills/rite-define/SKILL.md" \
  ".claude/agents/devrites-code-reviewer.md" \
  ".claude/rules/security.md" \
  ".devrites/README.md" \
  ".devrites/ACTIVE" ; do
  [ -f "$T/$f" ] && ok "present: $f" || no "missing: $f"
done

# 6) no global write
[ -e "$HOME/.claude/skills/rite" ] && no "wrote to ~/.claude !!" || ok "~/.claude untouched"

# 7) uninstall via the CLI removes manifest files, preserves runtime state
node "$CLI" uninstall --target "$T" >/dev/null 2>&1 || no "uninstall exited non-zero"
[ -f "$T/.claude/devrites.manifest" ] && no "manifest survived uninstall" || ok "manifest removed"
[ -f "$T/.claude/skills/rite/SKILL.md" ] && no "skill survived uninstall" || ok "skills removed"
[ -f "$T/.devrites/ACTIVE" ] && ok "uninstall preserved .devrites/ACTIVE" || no "uninstall dropped runtime state"

# 8) unknown flag is passed through and rejected by the installer (non-zero)
node "$CLI" --target "$T" --bogus-flag >/dev/null 2>&1 && no "unknown flag did not fail" || ok "unknown flag passed through + rejected"

echo ""
[ "$fail" -eq 0 ] && echo "cli-smoke: PASS" || echo "cli-smoke: FAIL"
exit "$fail"
