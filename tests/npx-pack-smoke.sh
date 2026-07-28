#!/usr/bin/env bash
# npx-pack-smoke.sh: verify the PACKAGED npm artifact installs and runs, the way
# `npx devrites` resolves it.
#
# cli-smoke.sh runs bin/devrites.mjs straight from the repo tree, so it never sees
# what npm publish would ship. This test closes that gap: it packs the package
# (npm pack → the same files/.npmignore/prepack path publish uses), installs the
# tarball into an isolated global prefix, and drives the resolved `devrites` bin.
# It catches the regressions the in-tree smoke can't: a dropped entry in
# package.json "files", an over-broad .npmignore, a broken prepack, or a bad
# bin/shebang: each of which silently breaks real `npx devrites`.
#
# Packs a throwaway tree so lifecycle hooks never mutate your working copy. By
# default it copies the current tracked working tree so local tracked and
# untracked changes are tested before commit; set DEVRITES_NPX_PACK_FROM_HEAD=1
# for a deterministic committed-HEAD release check. Fully offline (no runtime deps).
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
  cp -R "$ROOT"/. "$WORK"/ 2>/dev/null && ok "no git HEAD: copied working tree"
fi

mkdir -p "$WORK/docs/internal"
printf 'private work must survive npm pack\n' > "$WORK/docs/internal/private-work.txt"
( cd "$WORK" && env -u DEVRITES_HOST_ARTIFACT_DIR npm pack --pack-destination "$PACKDIR" ) >/dev/null 2>&1 \
  || { no "npm pack failed"; echo "npx-pack-smoke: FAIL"; exit 1; }
[ -f "$WORK/docs/internal/private-work.txt" ] \
  && ok "npm pack preserves ignored developer files" \
  || no "npm pack deleted docs/internal developer data"
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
  package/scripts/codex-generate.sh \
  package/scripts/build-host-artifacts.sh \
  package/pack/generated/claude/skills/rite-build/SKILL.md \
  package/pack/generated/codex/skills/rite-build/SKILL.md \
  package/pack/generated/codex/agents/devrites-code-reviewer.toml \
  package/pack/generated/codex/hooks.json \
  package/pack/.claude/ ; do
  echo "$contents" | grep -q "^$need" && ok "tarball ships $need" || no "tarball MISSING $need (files allowlist?)"
done
echo "$contents" | grep -q '__pycache__' && no "tarball ships __pycache__ (package exclusions failed)" \
  || ok "no __pycache__ in tarball"
echo "$contents" | grep -q 'docs/internal' && no "tarball ships docs/internal (package exclusions failed)" \
  || ok "no docs/internal in tarball"

# 3) Install the tarball into an isolated global prefix: offline, real ~/.npm global untouched.
npm install -g --prefix "$PREFIX" --no-audit --no-fund "$TGZ" >/dev/null 2>&1 \
  || { no "npm install -g of the tarball failed"; echo "npx-pack-smoke: FAIL"; exit 1; }
BIN="$PREFIX/bin/devrites"
[ -x "$BIN" ] && ok "bin shim installed + executable (\$PREFIX/bin/devrites)" \
  || no "bin shim missing/not executable (bin field? shebang? exec bit?)"

# 4) Drive the resolved bin as npx would: no `node` prefix, so this exercises the shebang.
want="$(node -e "process.stdout.write(require('$ROOT/package.json').version)")"
got="$("$BIN" --version 2>/dev/null)"
[ "$got" = "$want" ] && ok "--version reports $got" || no "--version ($got) != package.json ($want)"
"$BIN" --help 2>/dev/null | grep -q 'Usage:' && ok "--help shows usage" || no "--help missing usage"

# 5) a packaged install pins the binary lookup to this package's version.
FETCH_LOG="$FAKEBIN/fetch.log"
FAKE_RELEASE_ENGINE="$FAKEBIN/release-devrites-engine"
FAKE_RELEASE_SHA="$FAKEBIN/release-devrites-engine.sha256"
FAKE_RELEASE_TAG="v$want"
export FETCH_LOG FAKE_RELEASE_ENGINE FAKE_RELEASE_SHA FAKE_RELEASE_TAG
cat > "$FAKE_RELEASE_ENGINE" <<'SH'
#!/usr/bin/env sh
case "$1" in
  install) exit 0 ;;
  version) printf '%s\n' "${FAKE_RELEASE_TAG:-v0.0.0}"; exit 0 ;;
  *) exit 0 ;;
esac
SH
chmod +x "$FAKE_RELEASE_ENGINE"
if command -v shasum >/dev/null 2>&1; then
  shasum -a 256 "$FAKE_RELEASE_ENGINE" > "$FAKE_RELEASE_SHA"
elif command -v sha256sum >/dev/null 2>&1; then
  sha256sum "$FAKE_RELEASE_ENGINE" > "$FAKE_RELEASE_SHA"
else
  no "no sha256 tool for fake release binary"
fi
cat > "$FAKEBIN/fetch-mock.mjs" <<'JS'
import { readFileSync, appendFileSync } from 'node:fs';
const ok = (body) => new Response(body, { status: 200 });
globalThis.fetch = async (url) => {
  const u = String(url);
  appendFileSync(process.env.FETCH_LOG, `${u}\n`);
  if (u.includes('releases/latest')) return new Response('', { status: 404 });
  if (u.includes(`/releases/download/${process.env.FAKE_RELEASE_TAG}/devrites-`) && u.endsWith('.sha256')) return ok(readFileSync(process.env.FAKE_RELEASE_SHA));
  if (u.includes(`/releases/download/${process.env.FAKE_RELEASE_TAG}/devrites-`)) return ok(readFileSync(process.env.FAKE_RELEASE_ENGINE));
  return new Response('', { status: 404 });
};
JS
NODE_DIR="$(dirname "$(command -v node)")"
env -u DEVRITES_HOST_ARTIFACT_DIR -u DEVRITES_NO_BINARY DEVRITES_BIN_DIR="$BIN_STAGE" NODE_OPTIONS="--import=$FAKEBIN/fetch-mock.mjs" PATH="$NODE_DIR:/usr/bin:/bin:/usr/sbin:/sbin" \
  "$BIN" --target "$TARGET_BINARY" --force >/dev/null 2>&1 \
  || no "packaged binary install exited non-zero"
fetch_calls="$(cat "$FETCH_LOG" 2>/dev/null || true)"
printf '%s' "$fetch_calls" | grep -q "releases/download/v$want/devrites-" \
  && ok "packaged binary lookup uses v$want" \
  || no "packaged binary lookup did not use v$want"
printf '%s' "$fetch_calls" | grep -q 'releases/latest' \
  && no "packaged binary lookup queried latest release" \
  || ok "packaged binary lookup avoids latest release"

# 6) dry-run writes nothing
env -u DEVRITES_HOST_ARTIFACT_DIR "$BIN" --target "$TARGET" --dry-run >/dev/null 2>&1 || no "dry-run exited non-zero"
[ -e "$TARGET/.claude" ] && no "dry-run created .claude" || ok "dry-run changed nothing"
[ -e "$TARGET/.agents" ] && no "dry-run created .agents" || true
[ -e "$TARGET/.codex" ] && no "dry-run created .codex" || true

# 7) real install, driven entirely by the packaged artifact
env -u DEVRITES_HOST_ARTIFACT_DIR "$BIN" --target "$TARGET" >/dev/null 2>&1 || no "install from the packaged artifact exited non-zero"
for f in \
  ".claude/devrites.manifest" \
  ".claude/skills/rite/SKILL.md" \
  ".agents/skills/rite/SKILL.md" \
  ".agents/skills/devrites-lib/reference/standards/security.md" \
  ".claude/agents/devrites-code-reviewer.md" \
  ".codex/agents/devrites-code-reviewer.toml" \
  ".codex/hooks.json" \
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
