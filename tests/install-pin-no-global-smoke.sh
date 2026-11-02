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

export HOME="$T/home"
mkdir -p "$HOME"

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

# Existing managed ancestors must never redirect pin writes outside the project.
for ancestor in .agents .agents/skills .claude .claude/skills; do
  for operation in add remove; do
    project="$T/alpha"
    outside="$HOME/beta"
    rm -rf "$project" "$outside" "$T/outside-before"
    mkdir -p "$project/.claude/skills/rite-build" "$project/.agents/skills"
    printf 'build skill\n' > "$project/.claude/skills/rite-build/SKILL.md"
    printf '# fixture manifest\n' > "$project/.claude/devrites.manifest"
    bash "$ROOT/scripts/pin.sh" --target "$project" add alpha rite-build >/dev/null 2>&1 \
      || no "ancestor setup pin failed"
    cp "$project/.claude/devrites.manifest" "$T/manifest-before"
    mv "$project/$ancestor" "$outside"
    ln -s "$outside" "$project/$ancestor"
    cp -R "$outside" "$T/outside-before"
    pin_args=(add beta rite-build)
    [ "$operation" = remove ] && pin_args=(remove alpha)
    if bash "$ROOT/scripts/pin.sh" --target "$project" "${pin_args[@]}" >/dev/null 2>&1; then
      no "pin $operation accepted linked $ancestor"
    else
      ok "pin $operation rejects linked $ancestor"
    fi
    diff -r "$T/outside-before" "$outside" >/dev/null 2>&1 \
      && ok "pin $operation preserves outside $ancestor bytes" \
      || no "pin $operation changed outside $ancestor bytes"
    cmp -s "$T/manifest-before" "$project/.claude/devrites.manifest" \
      && ok "pin $operation preserves manifest with linked $ancestor" \
      || no "pin $operation changed manifest with linked $ancestor"
  done
done

for leaf in claude-directory claude-file codex-directory codex-file; do
  project="$T/alpha"
  outside="$HOME/beta"
  rm -rf "$project" "$outside" "$T/outside-before"
  mkdir -p "$project/.claude/skills/rite-build" "$project/.agents/skills" "$outside"
  printf 'build skill\n' > "$project/.claude/skills/rite-build/SKILL.md"
  printf '# fixture manifest\n' > "$project/.claude/devrites.manifest"
  bash "$ROOT/scripts/pin.sh" --target "$project" add alpha rite-build >/dev/null 2>&1 \
    || no "leaf setup pin failed"
  cp "$project/.claude/devrites.manifest" "$T/manifest-before"
  case "$leaf" in
    claude-*) destination="$project/.claude/skills/alpha" ;;
    codex-*) destination="$project/.agents/skills/alpha" ;;
  esac
  case "$leaf" in
    *-file) destination="$destination/SKILL.md" ;;
  esac
  mv "$destination" "$outside/alpha"
  ln -s "$outside/alpha" "$destination"
  cp -R "$outside" "$T/outside-before"
  if bash "$ROOT/scripts/pin.sh" --target "$project" remove alpha >/dev/null 2>&1; then
    no "pin remove accepted linked $leaf"
  else
    ok "pin remove rejects linked $leaf"
  fi
  diff -r "$T/outside-before" "$outside" >/dev/null 2>&1 \
    && ok "pin remove preserves outside $leaf bytes" \
    || no "pin remove changed outside $leaf bytes"
  [ -L "$destination" ] && ok "pin remove preserves $leaf link" || no "pin remove deleted $leaf link"
  cmp -s "$T/manifest-before" "$project/.claude/devrites.manifest" \
    && ok "pin remove preserves manifest with linked $leaf" \
    || no "pin remove changed manifest with linked $leaf"
done

project="$T/alpha"
rm -rf "$project"
mkdir -p "$project/.claude/skills/rite-build"
printf 'build skill\n' > "$project/.claude/skills/rite-build/SKILL.md"
printf '# fixture manifest\n' > "$project/.claude/devrites.manifest"
ln -s "$project" "$T/project-alias"
bash "$ROOT/scripts/pin.sh" --target "$T/project-alias" add alpha rite-build >/dev/null 2>&1 \
  && ok "Claude-only pin permits canonical project alias" || no "Claude-only pin through project alias failed"
bash "$ROOT/scripts/pin.sh" --target "$T/project-alias" remove alpha >/dev/null 2>&1 \
  && ok "Claude-only unpin permits canonical project alias" || no "Claude-only unpin through project alias failed"
[ ! -e "$project/.agents" ] && ok "Claude-only pin leaves Codex absent" || no "Claude-only pin created Codex directory"

bash "$ROOT/scripts/check-no-global-writes.sh" >/dev/null 2>&1 && ok "check-no-global-writes passed" || no "check-no-global-writes failed"

echo ""
[ "$fail" -eq 0 ] && echo "install-pin-no-global-smoke: PASS" || echo "install-pin-no-global-smoke: FAIL"
exit "$fail"
