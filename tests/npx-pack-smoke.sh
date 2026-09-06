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
quickstart_started=$SECONDS
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
candidate_head="$(git -C "$ROOT" rev-parse HEAD 2>/dev/null || printf 'no-head')"
if command -v shasum >/dev/null 2>&1; then
  candidate_sha="$(shasum -a 256 "$TGZ" | awk '{print $1}')"
else
  candidate_sha="$(sha256sum "$TGZ" | awk '{print $1}')"
fi

# 2) The tarball must carry everything the bundled installer needs and no
# repository-only scripts (files allowlist intact, runtime surface closed).
contents="$(tar -tzf "$TGZ")"
for need in \
  package/bin/devrites.mjs \
  package/install.sh \
  package/uninstall.sh \
  package/update.sh \
  package/scripts/ \
  package/scripts/codex-generate.sh \
  package/scripts/omp-generate.sh \
  package/scripts/build-host-artifacts.sh \
  package/pack/generated/claude/skills/rite-build/SKILL.md \
  package/pack/generated/codex/skills/rite-build/SKILL.md \
  package/pack/generated/codex/agents/devrites-code-reviewer.toml \
  package/pack/generated/codex/config.toml \
  package/pack/generated/omp/skills/rite-build/SKILL.md \
  package/pack/generated/omp/.omp-plugin/plugin.json \
  package/pack/.claude/ ; do
  echo "$contents" | grep -q "^$need" && ok "tarball ships $need" || no "tarball MISSING $need (files allowlist?)"
done
shipped_scripts="$(printf '%s\n' "$contents" | grep '^package/scripts/.' | grep -v '/$' | sort)"
runtime_scripts="$(printf '%s\n' \
  package/scripts/build-host-artifacts.sh \
  package/scripts/codex-generate.sh \
  package/scripts/install-lib.sh \
  package/scripts/omp-generate.sh | sort)"
[ "$shipped_scripts" = "$runtime_scripts" ] \
  && ok "tarball scripts are limited to the closed runtime set" \
  || { no "tarball ships repository-only or misses runtime scripts"; printf '    shipped:\n%s\n' "$shipped_scripts"; }
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
help="$($BIN --help 2>/dev/null)"
echo "$help" | grep -q 'Usage:' && ok "--help shows usage" || no "--help missing usage"
echo "$help" | grep -q 'hook-free native agents' && ok "--help describes native agents" || no "--help misses native-agent wording"
echo "$help" | grep -Eqi 'active hooks|and hooks' && no "--help claims hooks are installed" || ok "--help makes no active-hook claim"

# 5) a packaged install pins the binary lookup to this package's version.
FETCH_LOG="$FAKEBIN/fetch.log"
REDIRECT_CANCEL_LOG="$FAKEBIN/redirect-cancel.log"
FAKE_RELEASE_ENGINE="$FAKEBIN/release-devrites-engine"
FAKE_RELEASE_TAG="v$want"
export FETCH_LOG REDIRECT_CANCEL_LOG FAKE_RELEASE_ENGINE FAKE_RELEASE_TAG
: > "$REDIRECT_CANCEL_LOG"
SOURCE_ENGINE="${DEVRITES_ENGINE_CLI:-}"
if [ -x "$SOURCE_ENGINE" ] && [ "$("$SOURCE_ENGINE" version 2>/dev/null)" = "$FAKE_RELEASE_TAG" ]; then
  cp "$SOURCE_ENGINE" "$FAKE_RELEASE_ENGINE"
elif command -v go >/dev/null 2>&1; then
  (cd "$ROOT/engine" && CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X github.com/devrites/devrites/internal/version.Version=$FAKE_RELEASE_TAG" \
    -o "$FAKE_RELEASE_ENGINE" .) >/dev/null 2>&1 \
    || { no "could not build the version-pinned fake release engine"; exit 1; }
else
  no "no shared engine or Go toolchain for the fake release engine"
  exit 1
fi
chmod +x "$FAKE_RELEASE_ENGINE"
cat > "$FAKEBIN/fetch-mock.mjs" <<'JS'
import { createHash } from 'node:crypto';
import { readFileSync, appendFileSync } from 'node:fs';
const ok = (body) => new Response(body, { status: 200 });
const redirect = (location) => new Response(new ReadableStream({
  cancel() {
    appendFileSync(process.env.REDIRECT_CANCEL_LOG, `${location}\n`);
  },
}), { status: 302, headers: { location } });
globalThis.fetch = async (url, options) => {
  const u = String(url);
  appendFileSync(process.env.FETCH_LOG, `${u}\n`);
  if (options?.redirect !== 'manual') throw new Error('fetch must disable automatic redirects');
  if (u.includes('releases/latest')) return new Response('', { status: 404 });
  if (u.includes(`/releases/download/${process.env.FAKE_RELEASE_TAG}/devrites-`)) {
    const asset = u.split('/').pop();
    const location = process.env.FETCH_INSECURE_REDIRECT === '1'
      ? `http://downloads.invalid/${asset}`
      : `/mock-download/${asset}`;
    return redirect(location);
  }
  if (u.includes('/mock-download/devrites-') && u.endsWith('.sha256')) {
    const body = readFileSync(process.env.FAKE_RELEASE_ENGINE);
    const asset = u.split('/').pop().slice(0, -'.sha256'.length);
    return ok(`${createHash('sha256').update(body).digest('hex')}  ${asset}\n`);
  }
  if (u.includes('/mock-download/devrites-')) {
    if (process.env.FETCH_OVERSIZED === '1') {
      let chunks = 65;
      return ok(new ReadableStream({
        pull(controller) {
          if (!chunks--) return controller.close();
          controller.enqueue(new Uint8Array(1024 * 1024));
        },
      }));
    }
    return ok(readFileSync(process.env.FAKE_RELEASE_ENGINE));
  }
  return new Response('', { status: 404 });
};
JS
NODE_DIR="$(dirname "$(command -v node)")"
NODE_TMP="$FAKEBIN/node-tmp"
mkdir "$NODE_TMP"
env -u DEVRITES_HOST_ARTIFACT_DIR -u DEVRITES_NO_BINARY -u DEVRITES_ENGINE_CLI DEVRITES_BIN_DIR="$BIN_STAGE" TMPDIR="$NODE_TMP" NODE_OPTIONS="--import=$FAKEBIN/fetch-mock.mjs" PATH="$NODE_DIR:/usr/bin:/bin:/usr/sbin:/sbin" \
  "$BIN" --target "$TARGET_BINARY" --force >/dev/null 2>&1 \
  || no "packaged binary install exited non-zero"
fetch_calls="$(cat "$FETCH_LOG" 2>/dev/null || true)"
printf '%s' "$fetch_calls" | grep -q "releases/download/v$want/devrites-" \
  && ok "packaged binary lookup uses v$want" \
  || no "packaged binary lookup did not use v$want"
printf '%s' "$fetch_calls" | grep -q 'releases/latest' \
  && no "packaged binary lookup queried latest release" \
  || ok "packaged binary lookup avoids latest release"
printf '%s' "$fetch_calls" | grep -q 'https://github.com/mock-download/devrites-' \
  && ok "packaged binary lookup follows HTTPS redirects" \
  || no "packaged binary lookup did not follow HTTPS redirects"
[ "$(wc -l < "$REDIRECT_CANCEL_LOG")" -eq 2 ] \
  && ok "packaged binary lookup cancels redirect bodies" \
  || no "packaged binary lookup did not cancel each redirect body"
STAGED_ENGINE="$BIN_STAGE/devrites-engine"
[ -x "$STAGED_ENGINE" ] && ok "version-pinned engine staged" || no "version-pinned engine not staged"
find "$NODE_TMP" -mindepth 1 -print -quit | grep -q . \
  && no "successful Node acquisition leaked a temporary directory" \
  || ok "successful Node acquisition cleaned its temporary directory"

OVERSIZE_TMP="$FAKEBIN/oversize-tmp"
mkdir "$OVERSIZE_TMP"
NO_GO_BIN="$FAKEBIN/no-go"
mkdir "$NO_GO_BIN"
printf '#!/bin/sh\nexit 1\n' > "$NO_GO_BIN/go"
chmod +x "$NO_GO_BIN/go"
FAIL_PATH="$NO_GO_BIN:$NODE_DIR:/usr/bin:/bin:/usr/sbin:/sbin"
if env -u DEVRITES_HOST_ARTIFACT_DIR -u DEVRITES_ENGINE_CLI FETCH_OVERSIZED=1 TMPDIR="$OVERSIZE_TMP" \
  NODE_OPTIONS="--import=$FAKEBIN/fetch-mock.mjs" PATH="$FAIL_PATH" \
  "$BIN" --target "$TARGET_BINARY" --dry-run >"$FAKEBIN/oversize-output" 2>&1; then
  no "streamed binary larger than 64 MiB was accepted"
else
  ok "streamed binary larger than 64 MiB rejected"
fi
grep -Fq "release v$want asset devrites-" "$FAKEBIN/oversize-output" \
  && grep -Fq 'size limit failed' "$FAKEBIN/oversize-output" \
  && ok "Node acquisition reports release asset and size failure" \
  || no "Node acquisition did not retain the release asset size failure"
find "$OVERSIZE_TMP" -mindepth 1 -print -quit | grep -q . \
  && no "oversized Node acquisition leaked a temporary directory" \
  || ok "oversized Node acquisition cleaned its temporary directory"

INSECURE_LOG="$FAKEBIN/insecure-fetch.log"
INSECURE_CANCEL_LOG="$FAKEBIN/insecure-redirect-cancel.log"
INSECURE_TMP="$FAKEBIN/insecure-tmp"
mkdir "$INSECURE_TMP"
: > "$INSECURE_LOG"
: > "$INSECURE_CANCEL_LOG"
if env -u DEVRITES_HOST_ARTIFACT_DIR -u DEVRITES_ENGINE_CLI FETCH_LOG="$INSECURE_LOG" \
  REDIRECT_CANCEL_LOG="$INSECURE_CANCEL_LOG" FETCH_INSECURE_REDIRECT=1 TMPDIR="$INSECURE_TMP" \
  NODE_OPTIONS="--import=$FAKEBIN/fetch-mock.mjs" PATH="$FAIL_PATH" \
  "$BIN" --target "$TARGET_BINARY" --dry-run >/dev/null 2>&1; then
  no "HTTPS-to-HTTP redirect was accepted"
else
  ok "HTTPS-to-HTTP redirect rejected"
fi
grep -q '^http://' "$INSECURE_LOG" \
  && no "HTTPS-to-HTTP redirect reached an HTTP request" \
  || ok "HTTPS-to-HTTP redirect rejected before HTTP request"
[ "$(wc -l < "$INSECURE_CANCEL_LOG")" -eq 1 ] \
  && ok "rejected redirect body canceled" \
  || no "rejected redirect body was not canceled"
find "$INSECURE_TMP" -mindepth 1 -print -quit | grep -q . \
  && no "rejected redirect leaked a temporary directory" \
  || ok "rejected redirect cleaned its temporary directory"

# 6) dry-run writes nothing
env -u DEVRITES_HOST_ARTIFACT_DIR DEVRITES_ENGINE_CLI="$STAGED_ENGINE" "$BIN" --target "$TARGET" --dry-run >/dev/null 2>&1 || no "dry-run exited non-zero"
[ -e "$TARGET/.claude" ] && no "dry-run created .claude" || ok "dry-run changed nothing"
[ -e "$TARGET/.agents" ] && no "dry-run created .agents" || true
[ -e "$TARGET/.codex" ] && no "dry-run created .codex" || true

# 7) real install, driven entirely by the packaged artifact
env -u DEVRITES_HOST_ARTIFACT_DIR DEVRITES_ENGINE_CLI="$STAGED_ENGINE" "$BIN" --target "$TARGET" >/dev/null 2>&1 || no "install from the packaged artifact exited non-zero"
for f in \
  ".claude/devrites.manifest" \
  ".claude/skills/rite/SKILL.md" \
  ".agents/skills/rite/SKILL.md" \
  ".agents/skills/devrites-lib/reference/standards/security.md" \
  ".claude/agents/devrites-code-reviewer.md" \
  ".codex/agents/devrites-code-reviewer.toml" \
  ".codex/config.toml" \
  ".claude/skills/devrites-lib/reference/standards/security.md" \
  "AGENTS.md" \
  ".devrites/ACTIVE" ; do
  [ -f "$TARGET/$f" ] && ok "installed: $f" || no "missing after install: $f"
done
[ ! -e "$TARGET/.codex/hooks.json" ] && ok "packaged install creates no Codex root hooks" || no "packaged install created Codex root hooks"

# 8) the published package path reaches one real structural workspace check.
mkdir -p "$TARGET/.devrites/work/quickstart"
cat > "$TARGET/.devrites/work/quickstart/README.md" <<'EOF'
# Quickstart
phase: frame
status: running
next_action: write brief
last_updated: unknown

## Artifact map

- `state.md`

## Read next

- `state.md`

## Blocking gates

- None recorded.
EOF
cat > "$TARGET/.devrites/work/quickstart/state.md" <<'EOF'
## Cursor
| Key | Value |
| --- | --- |
| phase | frame |
| status | running |
EOF
printf 'quickstart\n' > "$TARGET/.devrites/ACTIVE"
readiness="$(DEVRITES_ROOT="$TARGET/.devrites" "$STAGED_ENGINE" check readiness quickstart 2>/dev/null)"
printf '%s' "$readiness" | grep -Fq 'reason: DRV-GATE-READINESS-PASSED' \
  && ok "published install path reaches structural workspace readiness" \
  || no "published install path did not reach structural workspace readiness"
printf '  evidence: candidate_head=%s package_sha256=%s observed_elapsed=%ss\n' \
  "$candidate_head" "$candidate_sha" "$((SECONDS - quickstart_started))"

# 9) project-local guarantee holds through the packaged path
[ -e "$HOME/.claude/skills/rite" ] && no "wrote to ~/.claude !!" || ok "~/.claude untouched"
[ -e "$HOME/.codex/agents/devrites-code-reviewer.toml" ] && no "wrote to ~/.codex !!" || ok "~/.codex untouched"

echo ""
[ "$fail" -eq 0 ] && echo "npx-pack-smoke: PASS" || echo "npx-pack-smoke: FAIL"
exit "$fail"
