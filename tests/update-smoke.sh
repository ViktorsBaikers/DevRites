#!/usr/bin/env bash
# Exercise update.sh without using the network:
#   - default installs survive an update --force,
#   - .devrites/ feature state is preserved across the upgrade,
#   - the retired --rules-only install shape still updates cleanly.
set -u
export DEVRITES_NO_BINARY=1   # pack smoke: the engine binary has its own lifecycle test (binary-lifecycle-test.sh)
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

echo "== update-smoke =="

run_update() {
  local target="$1"
  bash "$ROOT/update.sh" --target "$target" --force >/dev/null 2>&1
}

case_default() {
  local d="$T/default"; mkdir -p "$d"; fail=0
  bash "$ROOT/install.sh" --target "$d" >/dev/null 2>&1 || no "default install failed"
  mkdir -p "$d/.devrites/work/demo"; echo "phase: build" > "$d/.devrites/work/demo/state.md"
  printf 'demo\n' > "$d/.devrites/ACTIVE"
  run_update "$d" || no "update --force (default) exited non-zero"
  [ -f "$d/.claude/skills/rite-build/SKILL.md" ] && ok "default: skills survive update" || no "default: skills missing after update"
  [ -f "$d/.agents/skills/rite-build/SKILL.md" ] && ok "default: Codex skills survive update" || no "default: Codex skills missing after update"
  [ -f "$d/.claude/skills/devrites-lib/reference/standards/security.md" ] && ok "default: rules survive update" || no "default: rules missing after update"
  [ -f "$d/.agents/skills/devrites-lib/reference/standards/security.md" ] && ok "default: Codex rules mirror survives update" || no "default: Codex rules mirror missing after update"
  [ -d "$d/.claude/agents" ] && ok "default: agents survive update" || no "default: agents missing after update"
  [ -d "$d/.codex/agents" ] && ok "default: Codex agents survive update" || no "default: Codex agents missing after update"
  [ -e "$d/.codex/hooks.json" ] && no "default: Codex root hooks present after update" || ok "default: Codex root hooks absent after update"
  [ -f "$d/.codex/config.toml" ] && ok "default: Codex permission config survives update" || no "default: Codex permission config missing after update"
  [ -e "$d/.codex/mcp" ] && no "default: DevRites MCP directory present after update" || ok "default: DevRites MCP directory absent after update"
  [ -f "$d/AGENTS.md" ] && ok "default: AGENTS bridge survives update" || no "default: AGENTS bridge missing after update"
  [ -f "$d/.devrites/work/demo/state.md" ] && ok "default: feature state preserved" || no "default: feature state lost"
  grep -q '^phase: build$' "$d/.devrites/work/demo/state.md" && ok "default: state.md contents intact" || no "default: state.md content clobbered"
  [ "$(cat "$d/.devrites/ACTIVE")" = "demo" ] && ok "default: ACTIVE cursor preserved" || no "default: ACTIVE clobbered"
  exit "$fail"
}

case_rules_only() {
  local r="$T/rules-only"; mkdir -p "$r"; fail=0
  bash "$ROOT/install.sh" --target "$r" --rules-only >/dev/null 2>&1 || no "rules-only install failed"
  mkdir -p "$r/.devrites/work/demo"; echo "phase: spec" > "$r/.devrites/work/demo/state.md"
  local rflags
  rflags="$(sed -n 's/^# devrites-flags:[[:space:]]*//p' "$r/.claude/devrites.manifest" | head -n1)"
  case "$rflags" in
    *--no-skills*) no "rules-only wrongly recorded --no-skills (flag retired; got: $rflags)" ;;
    *) ok "rules-only is a no-op; manifest records a normal install ($rflags)" ;;
  esac
  [ -f "$r/.claude/skills/devrites-lib/reference/standards/security.md" ] && ok "rules-only: standards present (ship with skills)" || no "rules-only: standards missing"
  [ -f "$r/.claude/skills/rite-build/SKILL.md" ] && ok "rules-only: skills installed (no-op = normal install)" || no "rules-only: skills missing"
  run_update "$r" || no "update --force (rules-only) exited non-zero - flag replay rejected?"
  [ -f "$r/.claude/skills/devrites-lib/reference/standards/security.md" ] && ok "rules-only: standards survive update" || no "rules-only: standards missing after update"
  [ -f "$r/.devrites/work/demo/state.md" ] && ok "rules-only: feature state preserved" || no "rules-only: feature state lost"
  exit "$fail"
}

case_customized() {
  local d="$T/customized" manifest managed tmp out
  mkdir -p "$d"; fail=0
  bash "$ROOT/install.sh" --target "$d" >/dev/null 2>&1 || no "customized: install failed"
  manifest="$d/.claude/devrites.manifest"
  tmp="$manifest.tmp"
  sed 's/^# devrites-version:.*/# devrites-version: 0.0.0/' "$manifest" > "$tmp" && mv "$tmp" "$manifest"
  managed="$d/.claude/skills/rite/SKILL.md"
  printf 'local customization\n' > "$managed"
  out="$(bash "$ROOT/update.sh" --target "$d" 2>&1)" \
    && no "customized: default update silently succeeded" \
    || ok "customized: default update aborts"
  [ "$(cat "$managed")" = "local customization" ] && ok "customized: default update preserved file" || no "customized: default update changed file"
  printf '%s' "$out" | grep -q -- 'rerun with --force' && ok "customized: update gives force remediation" || no "customized: update missing force remediation"
  run_update "$d" || no "customized: forced update failed"
  [ "$(cat "$managed")" != "local customization" ] && ok "customized: forced update replaced file" || no "customized: forced update kept file"
  exit "$fail"
}

pids=()
case_default & pids+=("$!")
case_rules_only & pids+=("$!")
case_customized & pids+=("$!")
for pid in "${pids[@]}"; do
  wait "$pid" || fail=1
done

echo ""
[ "$fail" -eq 0 ] && echo "update-smoke: PASS" || echo "update-smoke: FAIL"
exit "$fail"
