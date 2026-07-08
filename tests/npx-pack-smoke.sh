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
# Packs a throwaway tree, so prepack's `rm -rf` runs there and never mutates your
# working copy. By default it copies the current tracked working tree so local
# tracked and untracked changes are tested before commit; set DEVRITES_NPX_PACK_FROM_HEAD=1 for a
# deterministic committed-HEAD release check. Fully offline (no runtime deps).
set -u
export DEVRITES_NO_BINARY=1   # pack smoke: the engine binary has its own lifecycle test (binary-lifecycle-test.sh)
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

command -v node >/dev/null 2>&1 || { echo "  FAIL: node not on PATH"; exit 1; }
command -v npm  >/dev/null 2>&1 || { echo "  FAIL: npm not on PATH";  exit 1; }

WORK="$(mktemp -d)"; PACKDIR="$(mktemp -d)"; PREFIX="$(mktemp -d)"; TARGET="$(mktemp -d)"
BIN_STAGE="$(mktemp -d)"; TARGET_BINARY="$(mktemp -d)"; FAKEBIN="$(mktemp -d)"
cleanup() { rm -rf "$WORK" "$PACKDIR" "$PREFIX" "$TARGET" "$BIN_STAGE" "$TARGET_BINARY" "$FAKEBIN"; }
trap cleanup EXIT
export npm_config_cache="$WORK/.npm-cache"

echo "== npx-pack-smoke =="

# 1) Export to a clean tree (non-destructive), pack it.
if [ "${DEVRITES_NPX_PACK_FROM_HEAD:-0}" = "1" ] && git -C "$ROOT" rev-parse HEAD >/dev/null 2>&1; then
  git -C "$ROOT" archive --format=tar HEAD | tar -x -C "$WORK" && ok "exported HEAD to a clean tree" \
    || { no "git archive failed"; echo "npx-pack-smoke: FAIL"; exit 1; }
elif git -C "$ROOT" rev-parse --show-toplevel >/dev/null 2>&1; then
  (cd "$ROOT" && git ls-files -z --cached --others --exclude-standard \
    | while IFS= read -r -d '' path; do [ -e "$path" ] && printf '%s\0' "$path"; done \
    | tar --null -T - -cf -) | tar -x -C "$WORK" \
    && ok "exported current working tree to a clean tree" \
    || { no "working-tree export failed"; echo "npx-pack-smoke: FAIL"; exit 1; }
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

# 5) a packaged install pins the binary lookup to this package's version.
CURL_LOG="$FAKEBIN/curl.log"
export CURL_LOG
cat > "$FAKEBIN/curl" <<'SH'
#!/usr/bin/env sh
printf '%s\n' "$*" >> "$CURL_LOG"
case "$*" in
  *releases/latest*) exit 42 ;;
  *) exit 22 ;;
esac
SH
cat > "$FAKEBIN/devrites" <<'SH'
#!/usr/bin/env sh
exit 127
SH
chmod +x "$FAKEBIN/curl" "$FAKEBIN/devrites"
NODE_DIR="$(dirname "$(command -v node)")"
env -u DEVRITES_NO_BINARY DEVRITES_BIN_DIR="$BIN_STAGE" PATH="$FAKEBIN:$NODE_DIR:/usr/bin:/bin:/usr/sbin:/sbin" \
  "$BIN" --target "$TARGET_BINARY" --force >/dev/null 2>&1 \
  || no "packaged binary install exited non-zero"
curl_calls="$(cat "$CURL_LOG" 2>/dev/null || true)"
printf '%s' "$curl_calls" | grep -q "releases/download/v$want/devrites-" \
  && ok "packaged binary lookup uses v$want" \
  || no "packaged binary lookup did not use v$want"
printf '%s' "$curl_calls" | grep -q 'releases/latest' \
  && no "packaged binary lookup queried latest release" \
  || ok "packaged binary lookup avoids latest release"

# 6) dry-run writes nothing
"$BIN" --target "$TARGET" --dry-run >/dev/null 2>&1 || no "dry-run exited non-zero"
[ -e "$TARGET/.claude" ] && no "dry-run created .claude" || ok "dry-run changed nothing"
[ -e "$TARGET/.agents" ] && no "dry-run created .agents" || true
[ -e "$TARGET/.codex" ] && no "dry-run created .codex" || true

# 7) real install, driven entirely by the packaged artifact
"$BIN" --target "$TARGET" >/dev/null 2>&1 || no "install from the packaged artifact exited non-zero"
for f in \
  ".claude/devrites.manifest" \
  ".claude/skills/rite/SKILL.md" \
  ".agents/skills/rite/SKILL.md" \
  ".agents/skills/devrites-lib/reference/standards/security.md" \
  ".claude/agents/devrites-code-reviewer.md" \
  ".codex/agents/devrites-code-reviewer.toml" \
  ".codex/config.toml" \
  ".codex/hooks.json" \
  ".codex/mcp/devrites-mcp.mjs" \
  ".claude/skills/devrites-lib/reference/standards/security.md" \
  "AGENTS.md" \
  ".devrites/ACTIVE" ; do
  [ -f "$TARGET/$f" ] && ok "installed: $f" || no "missing after install: $f"
done

# 8) project-local guarantee holds through the packaged path
[ -e "$HOME/.claude/skills/rite" ] && no "wrote to ~/.claude !!" || ok "~/.claude untouched"
[ -e "$HOME/.codex/agents/devrites-code-reviewer.toml" ] && no "wrote to ~/.codex !!" || ok "~/.codex untouched"

echo ""
[ "$fail" -eq 0 ] && echo "npx-pack-smoke: PASS" || echo "npx-pack-smoke: FAIL"
exit "$fail"
