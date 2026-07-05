#!/usr/bin/env bash
# update-smoke.sh — exercise update.sh without hitting the network:
#   - default AND --rules-only installs survive an update --force,
#   - .devrites/ feature state is preserved across the upgrade,
#   - the manifest's recorded flags replay cleanly through install.sh's parser
#     (the C2 regression: a --rules-only manifest records --no-skills, which
#     install.sh must accept).
# Uses DEVRITES_UPDATE_BUNDLE to feed update.sh a locally-built release tarball.
set -u
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
[ -f "$D/.claude/rules/security.md" ]          && ok "default: rules survive update"  || no "default: rules missing after update"
[ -f "$D/.agents/devrites/rules/security.md" ] && ok "default: Codex rules mirror survives update" || no "default: Codex rules mirror missing after update"
[ -d "$D/.claude/agents" ]                      && ok "default: agents survive update" || no "default: agents missing after update"
[ -d "$D/.codex/agents" ]                       && ok "default: Codex agents survive update" || no "default: Codex agents missing after update"
[ -f "$D/.codex/config.toml" ]                   && ok "default: Codex config survives update" || no "default: Codex config missing after update"
[ -f "$D/.codex/hooks.json" ]                   && ok "default: Codex hooks survive update" || no "default: Codex hooks missing after update"
[ -f "$D/.codex/mcp/devrites-mcp.mjs" ]          && ok "default: Codex MCP server survives update" || no "default: Codex MCP server missing after update"
[ -f "$D/AGENTS.md" ]                           && ok "default: AGENTS bridge survives update" || no "default: AGENTS bridge missing after update"
[ -f "$D/.devrites/work/demo/state.md" ]        && ok "default: feature state preserved" || no "default: feature state lost"
grep -q '^phase: build$' "$D/.devrites/work/demo/state.md" && ok "default: state.md contents intact" || no "default: state.md content clobbered"
[ "$(cat "$D/.devrites/ACTIVE")" = "demo" ]     && ok "default: ACTIVE cursor preserved" || no "default: ACTIVE clobbered"

# ---- case 2: --rules-only install (the C2 round-trip) -------------------
R="$T/rules-only"; mkdir -p "$R"
bash "$ROOT/install.sh" --target "$R" --rules-only >/dev/null 2>&1 || no "rules-only install failed"
mkdir -p "$R/.devrites/work/demo"; echo "phase: spec" > "$R/.devrites/work/demo/state.md"
# the recorded flags must contain --no-skills for a rules-only install
RFLAGS="$(sed -n 's/^# devrites-flags:[[:space:]]*//p' "$R/.claude/devrites.manifest" | head -n1)"
case "$RFLAGS" in
  *--no-skills*) ok "rules-only manifest records --no-skills ($RFLAGS)" ;;
  *) no "rules-only manifest missing --no-skills (got: $RFLAGS)" ;;
esac
run_update "$R" || no "update --force (rules-only) exited non-zero — flag replay rejected?"
[ -f "$R/.claude/rules/security.md" ] && ok "rules-only: rules survive update" || no "rules-only: rules missing after update"
[ -d "$R/.claude/skills" ] && no "rules-only: update wrongly installed skills" || ok "rules-only: still no skills (flags honored)"
[ -d "$R/.claude/agents" ] && no "rules-only: update wrongly installed agents" || ok "rules-only: still no agents (flags honored)"
[ -d "$R/.agents" ] && no "rules-only: update wrongly installed Codex skills" || ok "rules-only: still no Codex skills (flags honored)"
[ -d "$R/.codex" ] && no "rules-only: update wrongly installed Codex agents" || ok "rules-only: still no Codex agents (flags honored)"
[ -f "$R/AGENTS.md" ] && no "rules-only: update wrongly installed AGENTS bridge" || ok "rules-only: still no AGENTS bridge (flags honored)"
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
