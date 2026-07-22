#!/usr/bin/env bash
# install-pin-no-global-smoke.sh: pin aliases stay project-local.
set -u
export DEVRITES_NO_BINARY=1
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

T="$(mktemp -d)"
GEN=""
cleanup() { rm -rf "$T"; [ -n "$GEN" ] && rm -rf "$GEN"; }
trap cleanup EXIT
if [ -z "${DEVRITES_HOST_ARTIFACT_DIR:-}" ]; then
  GEN="$(mktemp -d)"
  DEVRITES_HOST_ARTIFACT_DIR="$GEN" bash "$ROOT/scripts/build-host-artifacts.sh" >/dev/null 2>&1 \
    || { echo "  FAIL: could not build host artifacts"; exit 1; }
  export DEVRITES_HOST_ARTIFACT_DIR="$GEN"
fi

echo "== install-pin-no-global-smoke =="

bash "$ROOT/install.sh" --target "$T" >/dev/null 2>&1 || no "install failed"
[ -e "$HOME/.claude/skills/rite" ] && no "wrote to ~/.claude !!" || ok "~/.claude untouched"
[ -e "$HOME/.codex/agents/devrites-code-reviewer.toml" ] && no "wrote to ~/.codex !!" || ok "~/.codex untouched"

bash "$ROOT/scripts/pin.sh" --target "$T" add b rite-build >/dev/null 2>&1 || no "pin add failed"
[ -f "$T/.claude/skills/b/SKILL.md" ] && ok "pin writes Claude alias" || no "pin missing Claude alias"
[ -f "$T/.agents/skills/b/SKILL.md" ] && ok "pin mirrors Codex alias" || no "pin missing Codex alias"
grep -q '^\.agents/skills/b/SKILL.md$' "$T/.claude/devrites.manifest" && ok "pin manifests Codex alias" || no "pin missing Codex manifest entry"
bash "$ROOT/scripts/pin.sh" --target "$T" remove b >/dev/null 2>&1 || no "pin remove failed"
[ -e "$T/.claude/skills/b/SKILL.md" ] && no "pin remove left Claude alias" || ok "pin removes Claude alias"
[ -e "$T/.agents/skills/b/SKILL.md" ] && no "pin remove left Codex alias" || ok "pin removes Codex alias"

bash "$ROOT/scripts/check-no-global-writes.sh" >/dev/null 2>&1 && ok "check-no-global-writes passed" || no "check-no-global-writes failed"

echo ""
[ "$fail" -eq 0 ] && echo "install-pin-no-global-smoke: PASS" || echo "install-pin-no-global-smoke: FAIL"
exit "$fail"
