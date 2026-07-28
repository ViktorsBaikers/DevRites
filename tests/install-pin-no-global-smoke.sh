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

printf 'user-owned skill path\n' > "$T/.claude/skills/foreign"
cp "$T/.claude/devrites.manifest" "$T/manifest-before-failed-add"
if bash "$ROOT/scripts/pin.sh" --target "$T" add foreign rite-build >/dev/null 2>&1; then
  no "pin add reported success for a foreign regular-file destination"
else
  ok "pin add rejects a foreign regular-file destination"
fi
grep -qx 'user-owned skill path' "$T/.claude/skills/foreign" \
  && ok "failed pin add preserves the foreign file" \
  || no "failed pin add changed the foreign file"
cmp -s "$T/manifest-before-failed-add" "$T/.claude/devrites.manifest" \
  && ok "failed pin add leaves the manifest unchanged" \
  || no "failed pin add corrupted the manifest"

bash "$ROOT/scripts/pin.sh" --target "$T" add keep rite-build >/dev/null 2>&1 || no "setup pin for remove safety failed"
printf 'user-owned Codex content\n' > "$T/.agents/skills/keep/SKILL.md"
cp "$T/.claude/devrites.manifest" "$T/manifest-before-failed-remove"
if bash "$ROOT/scripts/pin.sh" --target "$T" remove keep >/dev/null 2>&1; then
  no "pin remove accepted a foreign Codex alias"
else
  ok "pin remove rejects a foreign Codex alias"
fi
[ -f "$T/.claude/skills/keep/SKILL.md" ] \
  && ok "failed pin remove preserves the Claude alias" \
  || no "failed pin remove deleted the Claude alias"
grep -qx 'user-owned Codex content' "$T/.agents/skills/keep/SKILL.md" \
  && ok "failed pin remove preserves foreign Codex content" \
  || no "failed pin remove changed foreign Codex content"
cmp -s "$T/manifest-before-failed-remove" "$T/.claude/devrites.manifest" \
  && ok "failed pin remove leaves the manifest unchanged" \
  || no "failed pin remove corrupted the manifest"

bash "$ROOT/scripts/check-no-global-writes.sh" >/dev/null 2>&1 && ok "check-no-global-writes passed" || no "check-no-global-writes failed"

echo ""
[ "$fail" -eq 0 ] && echo "install-pin-no-global-smoke: PASS" || echo "install-pin-no-global-smoke: FAIL"
exit "$fail"
