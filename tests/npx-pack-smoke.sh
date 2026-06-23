#!/usr/bin/env bash
# npx-pack-smoke.sh — verify the PACKAGED npm artifact installs and runs, the way
# `npx devrites` actually resolves it.
#
# cli-smoke.sh runs bin/devrites.mjs straight from the repo tree, so it never sees
# what npm publish would ship. This test closes that gap: it packs the package
# (npm pack → the same files/.npmignore/prepack path publish uses), installs the
# tarball into an isolated global prefix, and drives the resolved `devrites` bin.
# It catches the regressions the in-tree smoke can't — a dropped entry in
# package.json "files", an over-broad .npmignore, a broken prepack, or a bad
# bin/shebang — each of which silently breaks real `npx devrites`.
#
# Packs committed HEAD into a throwaway tree, so prepack's `rm -rf` runs there and
# never mutates your working copy. Fully offline (the package has no runtime deps).
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

command -v node >/dev/null 2>&1 || { echo "  FAIL: node not on PATH"; exit 1; }
command -v npm  >/dev/null 2>&1 || { echo "  FAIL: npm not on PATH";  exit 1; }

WORK="$(mktemp -d)"; PACKDIR="$(mktemp -d)"; PREFIX="$(mktemp -d)"; TARGET="$(mktemp -d)"
cleanup() { rm -rf "$WORK" "$PACKDIR" "$PREFIX" "$TARGET"; }
trap cleanup EXIT

echo "== npx-pack-smoke =="

# 1) Export committed HEAD to a clean tree (deterministic + non-destructive), pack it.
if git -C "$ROOT" rev-parse HEAD >/dev/null 2>&1; then
  git -C "$ROOT" archive --format=tar HEAD | tar -x -C "$WORK" && ok "exported HEAD to a clean tree" \
    || { no "git archive failed"; echo "npx-pack-smoke: FAIL"; exit 1; }
else
  cp -R "$ROOT"/. "$WORK"/ 2>/dev/null && ok "no git HEAD — copied working tree"
fi

( cd "$WORK" && npm pack --pack-destination "$PACKDIR" ) >/dev/null 2>&1 \
  || { no "npm pack failed"; echo "npx-pack-smoke: FAIL"; exit 1; }
TGZ="$(ls "$PACKDIR"/devrites-*.tgz 2>/dev/null | head -1)"
{ [ -n "$TGZ" ] && [ -f "$TGZ" ]; } && ok "packed $(basename "$TGZ")" \
  || { no "no tarball produced"; echo "npx-pack-smoke: FAIL"; exit 1; }

# 2) The tarball must carry everything the bundled installer needs (files allowlist intact).
contents="$(tar -tzf "$TGZ")"
for need in \
  package/bin/devrites.mjs \
  package/install.sh \
  package/uninstall.sh \
  package/update.sh \
  package/scripts/ \
  package/pack/.claude/ ; do
  echo "$contents" | grep -q "^$need" && ok "tarball ships $need" || no "tarball MISSING $need (files allowlist?)"
done
echo "$contents" | grep -q '__pycache__' && no "tarball ships __pycache__ (prepack didn't clean)" \
  || ok "no __pycache__ in tarball"

# 3) Install the tarball into an isolated global prefix — offline, real ~/.npm global untouched.
npm install -g --prefix "$PREFIX" --no-audit --no-fund "$TGZ" >/dev/null 2>&1 \
  || { no "npm install -g of the tarball failed"; echo "npx-pack-smoke: FAIL"; exit 1; }
BIN="$PREFIX/bin/devrites"
[ -x "$BIN" ] && ok "bin shim installed + executable (\$PREFIX/bin/devrites)" \
  || no "bin shim missing/not executable (bin field? shebang? exec bit?)"

# 4) Drive the resolved bin as npx would — no `node` prefix, so this exercises the shebang.
want="$(node -e "process.stdout.write(require('$ROOT/package.json').version)")"
got="$("$BIN" --version 2>/dev/null)"
[ "$got" = "$want" ] && ok "--version reports $got" || no "--version ($got) != package.json ($want)"
"$BIN" --help 2>/dev/null | grep -q 'Usage:' && ok "--help shows usage" || no "--help missing usage"

# 5) dry-run writes nothing
"$BIN" --target "$TARGET" --dry-run >/dev/null 2>&1 || no "dry-run exited non-zero"
[ -e "$TARGET/.claude" ] && no "dry-run created .claude" || ok "dry-run changed nothing"

# 6) real install, driven entirely by the packaged artifact
"$BIN" --target "$TARGET" >/dev/null 2>&1 || no "install from the packaged artifact exited non-zero"
for f in \
  ".claude/devrites.manifest" \
  ".claude/skills/rite/SKILL.md" \
  ".claude/agents/devrites-code-reviewer.md" \
  ".claude/rules/security.md" \
  ".devrites/ACTIVE" ; do
  [ -f "$TARGET/$f" ] && ok "installed: $f" || no "missing after install: $f"
done

# 7) project-local guarantee holds through the packaged path
[ -e "$HOME/.claude/skills/rite" ] && no "wrote to ~/.claude !!" || ok "~/.claude untouched"

echo ""
[ "$fail" -eq 0 ] && echo "npx-pack-smoke: PASS" || echo "npx-pack-smoke: FAIL"
exit "$fail"
