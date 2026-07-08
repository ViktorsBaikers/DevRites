#!/usr/bin/env bash
# update-smoke.sh — exercise update.sh without hitting the network:
#   - default installs survive an update --force,
#   - .devrites/ feature state is preserved across the upgrade,
#   - the manifest's recorded flags replay cleanly through install.sh's parser,
#   - the retired --no-rules/--rules-only flags still parse (accepted no-ops).
# Uses DEVRITES_UPDATE_BUNDLE to feed update.sh a locally-built release tarball.
set -u
export DEVRITES_NO_BINARY=1   # pack smoke: the engine binary has its own lifecycle test (binary-lifecycle-test.sh)
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

T="$(mktemp -d)"
DIST=""
cleanup() { rm -rf "$T"; [ -n "$DIST" ] && rm -rf "$DIST"; }
trap cleanup EXIT

echo "== update-smoke =="

# ---- version from package.json (manifests are kept in lockstep) ----------
VERSION="$(sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$ROOT/package.json" | head -n1)"
[ -n "$VERSION" ] && ok "package.json version: $VERSION" || no "could not read package.json version"
TAG="v$VERSION"

# ---- build a local release bundle update.sh can consume ------------------
# build-release-tarball.sh writes dist/devrites-v<version>.tar.gz, which extracts
# to devrites-v<version>/ — exactly the layout update.sh's re-exec expects.
bash "$ROOT/scripts/build-release-tarball.sh" "$VERSION" >/dev/null 2>&1 || no "build-release-tarball failed"
BUNDLE="$ROOT/dist/devrites-${TAG}.tar.gz"
DIST="$ROOT/dist"
[ -f "$BUNDLE" ] && ok "built local bundle: ${BUNDLE#$ROOT/}" || no "no local bundle at $BUNDLE"

# update_one TARGET FLAGS… — run update.sh --force against TARGET using the
# local bundle (no network). $@ after TARGET are extra install.sh flags.
run_update() {
  _t="$1"
  DEVRITES_UPDATE_BUNDLE="$BUNDLE" bash "$ROOT/update.sh" --target "$_t" --to "$TAG" --force >/dev/null 2>&1
}

# ---- case 1: default install -------------------------------------------
D="$T/default"; mkdir -p "$D"
bash "$ROOT/install.sh" --target "$D" >/dev/null 2>&1 || no "default install failed"
# seed feature state that must survive the upgrade
mkdir -p "$D/.devrites/work/demo"; echo "phase: build" > "$D/.devrites/work/demo/state.md"
printf 'demo\n' > "$D/.devrites/ACTIVE"
run_update "$D" || no "update --force (default) exited non-zero"
[ -f "$D/.claude/skills/rite-build/SKILL.md" ] && ok "default: skills survive update" || no "default: skills missing after update"
[ -f "$D/.agents/skills/rite-build/SKILL.md" ] && ok "default: Codex skills survive update" || no "default: Codex skills missing after update"
[ -f "$D/.claude/skills/devrites-lib/reference/standards/security.md" ]          && ok "default: rules survive update"  || no "default: rules missing after update"
[ -f "$D/.agents/skills/devrites-lib/reference/standards/security.md" ] && ok "default: Codex rules mirror survives update" || no "default: Codex rules mirror missing after update"
[ -d "$D/.claude/agents" ]                      && ok "default: agents survive update" || no "default: agents missing after update"
[ -d "$D/.codex/agents" ]                       && ok "default: Codex agents survive update" || no "default: Codex agents missing after update"
[ -f "$D/.codex/config.toml" ]                   && ok "default: Codex config survives update" || no "default: Codex config missing after update"
[ -f "$D/.codex/hooks.json" ]                   && ok "default: Codex hooks survive update" || no "default: Codex hooks missing after update"
[ -f "$D/.codex/mcp/devrites-mcp.mjs" ]          && ok "default: Codex MCP server survives update" || no "default: Codex MCP server missing after update"
[ -f "$D/AGENTS.md" ]                           && ok "default: AGENTS bridge survives update" || no "default: AGENTS bridge missing after update"
[ -f "$D/.devrites/work/demo/state.md" ]        && ok "default: feature state preserved" || no "default: feature state lost"
grep -q '^phase: build$' "$D/.devrites/work/demo/state.md" && ok "default: state.md contents intact" || no "default: state.md content clobbered"
[ "$(cat "$D/.devrites/ACTIVE")" = "demo" ]     && ok "default: ACTIVE cursor preserved" || no "default: ACTIVE clobbered"

# ---- case 2: --rules-only is now a deprecated no-op (installs normally) --
# Retired: --rules-only used to record --no-skills for a minimal footprint. The
# engineering standards now ship inside the devrites-lib skill, so --rules-only
# just warns and performs a normal install; old manifests still replay cleanly.
R="$T/rules-only"; mkdir -p "$R"
bash "$ROOT/install.sh" --target "$R" --rules-only >/dev/null 2>&1 || no "rules-only install failed"
mkdir -p "$R/.devrites/work/demo"; echo "phase: spec" > "$R/.devrites/work/demo/state.md"
# the flag is retired: it must NOT record --no-skills anymore
RFLAGS="$(sed -n 's/^# devrites-flags:[[:space:]]*//p' "$R/.claude/devrites.manifest" | head -n1)"
case "$RFLAGS" in
  *--no-skills*) no "rules-only wrongly recorded --no-skills (flag retired; got: $RFLAGS)" ;;
  *) ok "rules-only is a no-op; manifest records a normal install ($RFLAGS)" ;;
esac
[ -f "$R/.claude/skills/devrites-lib/reference/standards/security.md" ] && ok "rules-only: standards present (ship with skills)" || no "rules-only: standards missing"
[ -f "$R/.claude/skills/rite-build/SKILL.md" ] && ok "rules-only: skills installed (no-op = normal install)" || no "rules-only: skills missing"
run_update "$R" || no "update --force (rules-only) exited non-zero — flag replay rejected?"
[ -f "$R/.claude/skills/devrites-lib/reference/standards/security.md" ] && ok "rules-only: standards survive update" || no "rules-only: standards missing after update"
[ -f "$R/.devrites/work/demo/state.md" ] && ok "rules-only: feature state preserved" || no "rules-only: feature state lost"

# ---- case 3: direct flag-replay unit test through install.sh's parser ---
# This is the heart of C2: every flag combination install.sh can record must
# parse back without "unknown option". Drive install.sh --dry-run so it only
# parses + plans, writing nothing.
P="$T/parser"; mkdir -p "$P"
for flags in \
  "--no-skills --no-agents --no-short-aliases" \
  "--no-agents" \
  "--no-codex" \
  "--no-rules" \
  "--rules-only" \
  "--no-short-aliases" \
  "--short-aliases=all" ; do
  # shellcheck disable=SC2086
  if bash "$ROOT/install.sh" --target "$P" --dry-run $flags >/dev/null 2>&1; then
    ok "install.sh parses recorded flags: $flags"
  else
    no "install.sh rejected recorded flags: $flags"
  fi
done

# bare --short-aliases is a no-op but must not be an error (Nit4)
if bash "$ROOT/install.sh" --target "$P" --dry-run --short-aliases >/dev/null 2>&1; then
  ok "install.sh accepts bare --short-aliases (warns, no-op)"
else
  no "install.sh errored on bare --short-aliases"
fi
# and it should warn that it's a no-op
bash "$ROOT/install.sh" --target "$P" --dry-run --short-aliases 2>&1 \
  | grep -qi 'no-op' && ok "bare --short-aliases warns it's a no-op" || no "bare --short-aliases did not warn"

# genuinely unknown options still fail
if bash "$ROOT/install.sh" --target "$P" --dry-run --totally-bogus >/dev/null 2>&1; then
  no "install.sh accepted a bogus flag"
else
  ok "install.sh still rejects unknown flags"
fi

echo ""
[ "$fail" -eq 0 ] && echo "update-smoke: PASS" || echo "update-smoke: FAIL"
exit "$fail"
