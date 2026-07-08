#!/usr/bin/env bash
# cli-smoke.sh — exercise the npx entry point (bin/devrites.mjs): version/help,
# subcommand routing, flag passthrough, and that it drives the bundled bash
# installers (install / uninstall) correctly. Exits non-zero on any failure.
set -u
export DEVRITES_NO_BINARY=1   # pack smoke: the engine binary has its own lifecycle test (binary-lifecycle-test.sh)
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
CLI="$ROOT/bin/devrites.mjs"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }
T="$(mktemp -d)"
GEN="$(mktemp -d)"
cleanup() { rm -rf "$T" "$GEN"; }
trap cleanup EXIT
if [ -n "${DEVRITES_HOST_ARTIFACT_DIR:-}" ]; then
  cp -R "$DEVRITES_HOST_ARTIFACT_DIR"/. "$GEN"/ \
    || { echo "  FAIL: could not copy host artifacts"; exit 1; }
else
  DEVRITES_HOST_ARTIFACT_DIR="$GEN" bash "$ROOT/scripts/build-host-artifacts.sh" >/dev/null 2>&1 \
    || { echo "  FAIL: could not build host artifacts"; exit 1; }
fi
export DEVRITES_HOST_ARTIFACT_DIR="$GEN"
printf '\n<!-- cli-smoke-generated-sentinel -->\n' >> "$GEN/codex/skills/rite/SKILL.md"

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
echo "$help" | grep -q -- '--no-binary.*devrites-engine' && ok "--help lists --no-binary engine skip" || no "--help missing --no-binary"
echo "$help" | grep -q -- '--no-rules.*Deprecated no-op' && ok "--help marks --no-rules deprecated" || no "--help does not mark --no-rules deprecated"
echo "$help" | grep -q -- '--rules-only.*Deprecated no-op' && ok "--help marks --rules-only deprecated" || no "--help does not mark --rules-only deprecated"
echo "$help" | grep -q -- '--no-rules.*Skip the engineering rules' && no "--help still claims --no-rules skips rules" || ok "--help does not claim --no-rules skips rules"
echo "$help" | grep -q -- '--rules-only.*Install only the engineering rules' && no "--help still claims --rules-only installs only rules" || ok "--help does not claim --rules-only is selective"

FAKE_ENGINE="$T/devrites-engine"
cat > "$FAKE_ENGINE" <<'SH'
#!/usr/bin/env bash
printf 'engine:%s\n' "$*"
SH
chmod +x "$FAKE_ENGINE"
proxy_out="$(DEVRITES_ENGINE_CLI="$FAKE_ENGINE" node "$CLI" preamble alpha 2>/dev/null)"
[ "$proxy_out" = "engine:preamble alpha" ] && ok "engine subcommands proxy to devrites-engine" || no "engine proxy failed: $proxy_out"
archive_out="$(DEVRITES_ENGINE_CLI="$FAKE_ENGINE" node "$CLI" archive-search alpha 2>/dev/null)"
[ "$archive_out" = "engine:archive-search alpha" ] && ok "archive-search proxies to devrites-engine" || no "archive-search proxy failed: $archive_out"

grep -q 'ENGINE_COMMANDS' "$CLI" && no "npm shim still carries command list" || ok "npm shim forwards arbitrary engine commands"

# 3) default (bare) dry-run writes nothing
node "$CLI" --target "$T" --dry-run >/dev/null 2>&1 || no "dry-run exited non-zero"
[ -e "$T/.claude" ] && no "dry-run created .claude" || ok "dry-run changed nothing"
[ -e "$T/.agents" ] && no "dry-run created .agents" || true
[ -e "$T/.codex" ] && no "dry-run created .codex" || true

# 4) `add` alias routes to install (dry-run)
out="$(node "$CLI" add --target "$T" --dry-run 2>&1)"
echo "$out" | grep -q 'dry run' && ok "add alias routes to installer" || no "add alias did not run installer"
[ -e "$T/.claude" ] && no "add dry-run created .claude" || ok "add dry-run changed nothing"
[ -e "$T/.agents" ] && no "add dry-run created .agents" || true
[ -e "$T/.codex" ] && no "add dry-run created .codex" || true

# 5) real install via the CLI
node "$CLI" --target "$T" >/dev/null 2>&1 || no "install exited non-zero"
[ -f "$T/.claude/devrites.manifest" ] && ok "manifest written" || no "no manifest"
for f in \
  ".claude/skills/rite/SKILL.md" \
  ".agents/skills/rite/SKILL.md" \
  ".claude/skills/rite-define/SKILL.md" \
  ".agents/skills/rite-define/SKILL.md" \
  ".agents/skills/devrites-lib/reference/standards/security.md" \
  ".claude/agents/devrites-code-reviewer.md" \
  ".codex/agents/devrites-code-reviewer.toml" \
  ".codex/hooks.json" \
  ".claude/skills/devrites-lib/reference/standards/security.md" \
  "AGENTS.md" \
  ".devrites/README.md" \
  ".devrites/ACTIVE" ; do
  [ -f "$T/$f" ] && ok "present: $f" || no "missing: $f"
done
grep -q 'cli-smoke-generated-sentinel' "$T/.agents/skills/rite/SKILL.md" && ok "CLI install consumes generated Codex skill payload" || no "CLI install did not use generated Codex skill payload"

# 6) no global write
[ -e "$HOME/.claude/skills/rite" ] && no "wrote to ~/.claude !!" || ok "~/.claude untouched"
[ -e "$HOME/.codex/agents/devrites-code-reviewer.toml" ] && no "wrote to ~/.codex !!" || ok "~/.codex untouched"

# 7) uninstall via the CLI removes manifest files, preserves runtime state
node "$CLI" uninstall --target "$T" >/dev/null 2>&1 || no "uninstall exited non-zero"
[ -f "$T/.claude/devrites.manifest" ] && no "manifest survived uninstall" || ok "manifest removed"
[ -f "$T/.claude/skills/rite/SKILL.md" ] && no "skill survived uninstall" || ok "skills removed"
[ -f "$T/.agents/skills/rite/SKILL.md" ] && no "Codex skill survived uninstall" || ok "Codex skills removed"
[ -f "$T/.codex/agents/devrites-code-reviewer.toml" ] && no "Codex agent survived uninstall" || ok "Codex agents removed"
[ -f "$T/.codex/hooks.json" ] && no "Codex hooks survived uninstall" || ok "Codex hooks removed"
[ -f "$T/AGENTS.md" ] && no "AGENTS bridge survived uninstall" || ok "AGENTS bridge removed"
[ -f "$T/.devrites/ACTIVE" ] && ok "uninstall preserved .devrites/ACTIVE" || no "uninstall dropped runtime state"

# 8) unknown flag is passed through and rejected by the installer (non-zero)
node "$CLI" --target "$T" --bogus-flag >/dev/null 2>&1 && no "unknown flag did not fail" || ok "unknown flag passed through + rejected"

echo ""
[ "$fail" -eq 0 ] && echo "cli-smoke: PASS" || echo "cli-smoke: FAIL"
exit "$fail"
