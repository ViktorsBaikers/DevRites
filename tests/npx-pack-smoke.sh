#!/usr/bin/env bash
# npx-pack-smoke.sh: verify the PACKAGED npm artifact installs and runs, the way
# `npx devrites` resolves it.
#
# cli-smoke.sh runs bin/devrites.mjs straight from the repo tree, so it never sees
# what npm publish would ship. This test closes that gap: it packs the package
# (npm pack → the same files/.npmignore path publish uses), installs the
# tarball into an isolated global prefix, and drives the resolved `devrites` bin.
# It catches the regressions the in-tree smoke can't: a dropped entry in
# package.json "files", an over-broad .npmignore, or a bad
# bin/shebang: each of which silently breaks real `npx devrites`.
#
# Packs a throwaway tree so lifecycle hooks never mutate your working copy. By
# default it copies the current tracked working tree so local tracked and
# untracked changes are tested before commit; set DEVRITES_NPX_PACK_FROM_HEAD=1
# for a deterministic committed-HEAD release check. Fully offline (no runtime deps).
set -u
quickstart_started=$SECONDS
export DEVRITES_NO_BINARY=1 # pack smoke: the engine binary has its own lifecycle test (binary-lifecycle-test.sh)
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() {
  printf '  FAIL: %s\n' "$*"
  fail=1
}

command -v node >/dev/null 2>&1 || {
  echo "  FAIL: node not on PATH"
  exit 1
}
command -v npm >/dev/null 2>&1 || {
  echo "  FAIL: npm not on PATH"
  exit 1
}

WORK="$(mktemp -d)"
PACKDIR="$(mktemp -d)"
PREFIX="$(mktemp -d)"
TARGET="$(mktemp -d)"
BIN_STAGE="$(mktemp -d)"
TARGET_BINARY="$(mktemp -d)"
FAKEBIN="$(mktemp -d)"
UNPACKED="$(mktemp -d)"
cleanup() { rm -rf "$WORK" "$PACKDIR" "$PREFIX" "$TARGET" "$BIN_STAGE" "$TARGET_BINARY" "$FAKEBIN" "$UNPACKED"; }
trap cleanup EXIT
export npm_config_cache="$WORK/.npm-cache"

echo "== npx-pack-smoke =="

# 1) Export to a clean tree (non-destructive), pack it.
if [ "${DEVRITES_NPX_PACK_FROM_HEAD:-0}" = "1" ] && git -C "$ROOT" rev-parse HEAD >/dev/null 2>&1; then
  git -C "$ROOT" archive --format=tar HEAD | tar -x -C "$WORK" && ok "exported HEAD to a clean tree" ||
    {
      no "git archive failed"
      echo "npx-pack-smoke: FAIL"
      exit 1
    }
elif git -C "$ROOT" rev-parse --show-toplevel >/dev/null 2>&1; then
  (cd "$ROOT" && git ls-files -z --cached --others --exclude-standard |
    while IFS= read -r -d '' path; do [ -e "$path" ] && printf '%s\0' "$path"; done |
    tar --null -T - -cf -) | tar -x -C "$WORK" &&
    ok "exported current working tree to a clean tree" ||
    {
      no "working-tree export failed"
      echo "npx-pack-smoke: FAIL"
      exit 1
    }
else
  cp -R "$ROOT"/. "$WORK"/ 2>/dev/null && ok "no git HEAD: copied working tree"
fi

mkdir -p "$WORK/docs/internal"
printf 'private work must survive npm pack\n' >"$WORK/docs/internal/private-work.txt"
(cd "$WORK" && env -u DEVRITES_HOST_ARTIFACT_DIR npm pack --pack-destination "$PACKDIR") >/dev/null 2>&1 ||
  {
    no "npm pack failed"
    echo "npx-pack-smoke: FAIL"
    exit 1
  }
[ -f "$WORK/docs/internal/private-work.txt" ] &&
  ok "npm pack preserves ignored developer files" ||
  no "npm pack deleted docs/internal developer data"
TGZ="$(ls "$PACKDIR"/devrites-*.tgz 2>/dev/null | head -1)"
{ [ -n "$TGZ" ] && [ -f "$TGZ" ]; } && ok "packed $(basename "$TGZ")" ||
  {
    no "no tarball produced"
    echo "npx-pack-smoke: FAIL"
    exit 1
  }
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
  package/scripts/devin-generate.sh \
  package/scripts/omp-generate.sh \
  package/scripts/pi-generate.sh \
  package/scripts/build-host-artifacts.sh \
  package/scripts/expand-includes.py \
  package/pack/generated/claude/skills/rite-build/SKILL.md \
  package/pack/generated/codex/skills/rite-build/SKILL.md \
  package/pack/generated/codex/agents/devrites-code-reviewer.toml \
  package/pack/generated/codex/config.toml \
  package/pack/generated/omp/skills/rite-build/SKILL.md \
  package/pack/generated/omp/.omp-plugin/plugin.json \
  package/pack/generated/pi/skills/rite-build/SKILL.md \
  package/pack/generated/pi/agents/devrites-code-reviewer.md \
  package/pack/generated/pi/prompts/rite-build.md \
  package/pack/generated/pi/AGENTS.md \
  package/pack/generated/devin/skills/rite-build/SKILL.md \
  package/pack/generated/devin/agents/devrites-code-reviewer.md \
  package/pack/generated/devin/AGENTS.md \
  package/pack/.claude/; do
  echo "$contents" | grep -q "^$need" && ok "tarball ships $need" || no "tarball MISSING $need (files allowlist?)"
done
shipped_scripts="$(printf '%s\n' "$contents" | grep '^package/scripts/.' | grep -v '/$' | sort)"
runtime_scripts="$(printf '%s\n' \
  package/scripts/build-host-artifacts.sh \
  package/scripts/codex-generate.sh \
  package/scripts/devin-generate.sh \
  package/scripts/expand-includes.py \
  package/scripts/install-lib.sh \
  package/scripts/omp-generate.sh \
  package/scripts/pi-generate.sh | sort)"
[ "$shipped_scripts" = "$runtime_scripts" ] &&
  ok "tarball scripts are limited to the closed runtime set" ||
  {
    no "tarball ships repository-only or misses runtime scripts"
    printf '    shipped:\n%s\n' "$shipped_scripts"
  }
# The fallback builder must run from the packed files alone: every script it
# references ships, and a build in a tree holding only packed files succeeds.
tar -xzf "$TGZ" -C "$UNPACKED"
for ref in $(grep -o '\$ROOT/scripts/[A-Za-z0-9_.-]*' "$UNPACKED/package/scripts/build-host-artifacts.sh" | sort -u); do
  [ -f "$UNPACKED/package/${ref#\$ROOT/}" ] && ok "builder dependency packed: ${ref#\$ROOT/}" ||
    no "builder dependency NOT packed: ${ref#\$ROOT/}"
done
if DEVRITES_HOST_ARTIFACT_DIR="$UNPACKED/rebuilt" bash "$UNPACKED/package/scripts/build-host-artifacts.sh" >"$UNPACKED/build.log" 2>&1 &&
  [ -f "$UNPACKED/rebuilt/claude/skills/rite-build/SKILL.md" ]; then
  ok "builder runs from the packed tree and populates its output"
else
  no "builder failed from the packed tree"
  tail -n 5 "$UNPACKED/build.log" | sed 's/^/    /'
fi
echo "$contents" | grep -q '__pycache__' && no "tarball ships __pycache__ (package exclusions failed)" ||
  ok "no __pycache__ in tarball"
echo "$contents" | grep -q 'docs/internal' && no "tarball ships docs/internal (package exclusions failed)" ||
  ok "no docs/internal in tarball"
echo "$contents" | grep -Eq '^package/engine/(tests|testdata)/' && no "tarball ships engine/tests or engine/testdata (package exclusions failed)" ||
  ok "no engine/tests or engine/testdata in tarball"

# 3) Install the tarball into an isolated global prefix: offline, real ~/.npm global untouched.
npm install -g --prefix "$PREFIX" --no-audit --no-fund "$TGZ" >/dev/null 2>&1 ||
  {
    no "npm install -g of the tarball failed"
    echo "npx-pack-smoke: FAIL"
    exit 1
  }
BIN="$PREFIX/bin/devrites"
[ -x "$BIN" ] && ok "bin shim installed + executable (\$PREFIX/bin/devrites)" ||
  no "bin shim missing/not executable (bin field? shebang? exec bit?)"

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
: >"$REDIRECT_CANCEL_LOG"
SOURCE_ENGINE="${DEVRITES_ENGINE_CLI:-}"
if [ -x "$SOURCE_ENGINE" ] && [ "$("$SOURCE_ENGINE" version 2>/dev/null)" = "$FAKE_RELEASE_TAG" ]; then
  cp "$SOURCE_ENGINE" "$FAKE_RELEASE_ENGINE"
elif command -v go >/dev/null 2>&1; then
  (cd "$ROOT/engine" && CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X github.com/devrites/devrites/internal/version.Version=$FAKE_RELEASE_TAG" \
    -o "$FAKE_RELEASE_ENGINE" .) >/dev/null 2>&1 ||
    {
      no "could not build the version-pinned fake release engine"
      exit 1
    }
else
  no "no shared engine or Go toolchain for the fake release engine"
  exit 1
fi
chmod +x "$FAKE_RELEASE_ENGINE"
cat >"$FAKEBIN/fetch-mock.mjs" <<'JS'
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
  if (process.env.FETCH_HANG === '1') return new Promise(() => {});
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
# gh stand-in: attests only the fixture engine digest, signed by ATTESTED_SIGNER, and
# accepts only a request pinned to the default repository and that exact signer workflow.
# The wrong-digest, wrong-repo and wrong-workflow arms below prove it is strict.
GH_BIN="$FAKEBIN/gh-bin"
mkdir "$GH_BIN"
ATTESTED_ENGINE="$FAKE_RELEASE_ENGINE"
ATTESTED_SIGNER=ViktorsBaikers/DevRites/.github/workflows/ci.yml@refs/heads/main
export ATTESTED_ENGINE ATTESTED_SIGNER
cat >"$GH_BIN/gh" <<'SH'
#!/bin/sh
[ "$1" = attestation ] && [ "$2" = verify ] && [ "$4" = --repo ] && [ "$6" = --signer-workflow ] &&
  [ "$5" = ViktorsBaikers/DevRites ] && [ "$7" = "$ATTESTED_SIGNER" ] || exit 64
got=$(shasum -a 256 "$3" | awk '{print $1}') || exit 65
want=$(shasum -a 256 "$ATTESTED_ENGINE" | awk '{print $1}')
[ "$got" = "$want" ]
SH
chmod +x "$GH_BIN/gh"
NODE_TMP="$FAKEBIN/node-tmp"
mkdir "$NODE_TMP"
NOGH_STAGE="$FAKEBIN/nogh-stage"
mkdir "$NOGH_STAGE"
if env -u DEVRITES_HOST_ARTIFACT_DIR -u DEVRITES_NO_BINARY -u DEVRITES_ENGINE_CLI DEVRITES_BIN_DIR="$NOGH_STAGE" TMPDIR="$NODE_TMP" NODE_OPTIONS="--import=$FAKEBIN/fetch-mock.mjs" PATH="$NODE_DIR:/usr/bin:/bin:/usr/sbin:/sbin" \
  "$BIN" --target "$TARGET_BINARY" --force >/dev/null 2>&1; then
  no "engine acquisition succeeded without attestation verification"
else
  ok "engine acquisition refuses without attestation verification"
fi
[ -e "$NOGH_STAGE/devrites-engine" ] && no "unattested engine was staged" || ok "unattested engine not staged"

# Each arm serves a self-consistent sidecar, so only the attestation can refuse it.
refused_by_attestation() {
  arm="$1"
  shift
  stage="$FAKEBIN/$arm-stage"
  mkdir "$stage"
  if env -u DEVRITES_HOST_ARTIFACT_DIR -u DEVRITES_NO_BINARY -u DEVRITES_ENGINE_CLI "$@" DEVRITES_BIN_DIR="$stage" TMPDIR="$NODE_TMP" NODE_OPTIONS="--import=$FAKEBIN/fetch-mock.mjs" PATH="$GH_BIN:$NODE_DIR:/usr/bin:/bin:/usr/sbin:/sbin" \
    "$BIN" --target "$TARGET_BINARY" --force >"$FAKEBIN/$arm.out" 2>&1; then
    no "$arm engine install succeeded"
  elif grep -Fq 'attestation verification failed' "$FAKEBIN/$arm.out"; then
    ok "$arm engine install refused by attestation"
  else
    no "$arm engine install failed for a reason other than attestation"
  fi
  find "$stage" -mindepth 1 -print -quit | grep -q . && no "$arm engine was staged" || ok "$arm engine not staged"
}
printf '#!/bin/sh\nexit 0\n' >"$FAKEBIN/attacker-engine"
chmod +x "$FAKEBIN/attacker-engine"
refused_by_attestation wrong-digest FAKE_RELEASE_ENGINE="$FAKEBIN/attacker-engine"
refused_by_attestation wrong-repo DEVRITES_REPO=attacker/fork ATTESTED_SIGNER=attacker/fork/.github/workflows/ci.yml@refs/heads/main
refused_by_attestation wrong-workflow ATTESTED_SIGNER=ViktorsBaikers/DevRites/.github/workflows/release.yml@refs/heads/main
: >"$FETCH_LOG"
: >"$REDIRECT_CANCEL_LOG"
env -u DEVRITES_HOST_ARTIFACT_DIR -u DEVRITES_NO_BINARY -u DEVRITES_ENGINE_CLI DEVRITES_BIN_DIR="$BIN_STAGE" TMPDIR="$NODE_TMP" NODE_OPTIONS="--import=$FAKEBIN/fetch-mock.mjs" PATH="$GH_BIN:$NODE_DIR:/usr/bin:/bin:/usr/sbin:/sbin" \
  "$BIN" --target "$TARGET_BINARY" --force >/dev/null 2>&1 ||
  no "packaged binary install exited non-zero"
fetch_calls="$(cat "$FETCH_LOG" 2>/dev/null || true)"
printf '%s' "$fetch_calls" | grep -q "releases/download/v$want/devrites-" &&
  ok "packaged binary lookup uses v$want" ||
  no "packaged binary lookup did not use v$want"
printf '%s' "$fetch_calls" | grep -q 'releases/latest' &&
  no "packaged binary lookup queried latest release" ||
  ok "packaged binary lookup avoids latest release"
printf '%s' "$fetch_calls" | grep -q 'https://github.com/mock-download/devrites-' &&
  ok "packaged binary lookup follows HTTPS redirects" ||
  no "packaged binary lookup did not follow HTTPS redirects"
[ "$(wc -l <"$REDIRECT_CANCEL_LOG")" -eq 2 ] &&
  ok "packaged binary lookup cancels redirect bodies" ||
  no "packaged binary lookup did not cancel each redirect body"
STAGED_ENGINE="$BIN_STAGE/devrites-engine"
[ -x "$STAGED_ENGINE" ] && ok "version-pinned engine staged" || no "version-pinned engine not staged"
find "$NODE_TMP" -mindepth 1 -print -quit | grep -q . &&
  no "successful Node acquisition leaked a temporary directory" ||
  ok "successful Node acquisition cleaned its temporary directory"

OVERSIZE_TMP="$FAKEBIN/oversize-tmp"
mkdir "$OVERSIZE_TMP"
NO_GO_BIN="$FAKEBIN/no-go"
mkdir "$NO_GO_BIN"
printf '#!/bin/sh\nexit 1\n' >"$NO_GO_BIN/go"
chmod +x "$NO_GO_BIN/go"
FAIL_PATH="$NO_GO_BIN:$NODE_DIR:/usr/bin:/bin:/usr/sbin:/sbin"
if env -u DEVRITES_HOST_ARTIFACT_DIR -u DEVRITES_ENGINE_CLI FETCH_OVERSIZED=1 TMPDIR="$OVERSIZE_TMP" \
  NODE_OPTIONS="--import=$FAKEBIN/fetch-mock.mjs" PATH="$FAIL_PATH" \
  "$BIN" --target "$TARGET_BINARY" --dry-run >"$FAKEBIN/oversize-output" 2>&1; then
  no "streamed binary larger than 64 MiB was accepted"
else
  ok "streamed binary larger than 64 MiB rejected"
fi
grep -Fq "release v$want asset devrites-" "$FAKEBIN/oversize-output" &&
  grep -Fq 'size limit failed' "$FAKEBIN/oversize-output" &&
  ok "Node acquisition reports release asset and size failure" ||
  no "Node acquisition did not retain the release asset size failure"
find "$OVERSIZE_TMP" -mindepth 1 -print -quit | grep -q . &&
  no "oversized Node acquisition leaked a temporary directory" ||
  ok "oversized Node acquisition cleaned its temporary directory"

INSECURE_LOG="$FAKEBIN/insecure-fetch.log"
INSECURE_CANCEL_LOG="$FAKEBIN/insecure-redirect-cancel.log"
INSECURE_TMP="$FAKEBIN/insecure-tmp"
mkdir "$INSECURE_TMP"
: >"$INSECURE_LOG"
: >"$INSECURE_CANCEL_LOG"
if env -u DEVRITES_HOST_ARTIFACT_DIR -u DEVRITES_ENGINE_CLI FETCH_LOG="$INSECURE_LOG" \
  REDIRECT_CANCEL_LOG="$INSECURE_CANCEL_LOG" FETCH_INSECURE_REDIRECT=1 TMPDIR="$INSECURE_TMP" \
  NODE_OPTIONS="--import=$FAKEBIN/fetch-mock.mjs" PATH="$FAIL_PATH" \
  "$BIN" --target "$TARGET_BINARY" --dry-run >/dev/null 2>&1; then
  no "HTTPS-to-HTTP redirect was accepted"
else
  ok "HTTPS-to-HTTP redirect rejected"
fi
grep -q '^http://' "$INSECURE_LOG" &&
  no "HTTPS-to-HTTP redirect reached an HTTP request" ||
  ok "HTTPS-to-HTTP redirect rejected before HTTP request"
[ "$(wc -l <"$INSECURE_CANCEL_LOG")" -eq 1 ] &&
  ok "rejected redirect body canceled" ||
  no "rejected redirect body was not canceled"
find "$INSECURE_TMP" -mindepth 1 -print -quit | grep -q . &&
  no "rejected redirect leaked a temporary directory" ||
  ok "rejected redirect cleaned its temporary directory"

# An interrupted acquisition removes its private temporary directory.
for sig_case in INT:130 TERM:143; do
  sig="${sig_case%%:*}"
  want_status="${sig_case##*:}"
  SIGNAL_TMP="$FAKEBIN/signal-$sig-tmp"
  mkdir "$SIGNAL_TMP"
  env -u DEVRITES_HOST_ARTIFACT_DIR -u DEVRITES_ENGINE_CLI FETCH_HANG=1 TMPDIR="$SIGNAL_TMP" \
    NODE_OPTIONS="--import=$FAKEBIN/fetch-mock.mjs" PATH="$FAIL_PATH" \
    "$BIN" --target "$TARGET_BINARY" --dry-run >/dev/null 2>&1 &
  sig_pid=$!
  for _ in $(seq 1 100); do
    find "$SIGNAL_TMP" -mindepth 1 -print -quit | grep -q . && break
    sleep 0.1
  done
  kill -s "$sig" "$sig_pid" 2>/dev/null
  for _ in $(seq 1 50); do
    kill -0 "$sig_pid" 2>/dev/null || break
    sleep 0.1
  done
  kill -0 "$sig_pid" 2>/dev/null && kill -s KILL "$sig_pid" 2>/dev/null
  wait "$sig_pid"
  sig_status=$?
  [ "$sig_status" -eq "$want_status" ] &&
    ok "SIG$sig ends acquisition with status $want_status" ||
    no "SIG$sig ended acquisition with status $sig_status, want $want_status"
  find "$SIGNAL_TMP" -mindepth 1 -print -quit | grep -q . &&
    no "SIG$sig leaked a temporary directory" ||
    ok "SIG$sig cleaned the temporary directory"
done

# A running engine keeps its exit status, receives signals aimed at the wrapper, and
# leaves no orphan or temporary directory behind.
SLEEP_ENGINE="$FAKEBIN/sleep-engine"
cat >"$SLEEP_ENGINE" <<'SH'
#!/bin/sh
[ -n "${ENGINE_EXIT:-}" ] && exit "$ENGINE_EXIT"
echo $$ >"$ENGINE_PID_FILE"
exec sleep 30
SH
chmod +x "$SLEEP_ENGINE"
cat >"$FAKEBIN/signal-driver.mjs" <<'JS'
import { spawn } from 'node:child_process';
import { existsSync, readFileSync } from 'node:fs';
const [mode, sig, pidFile, bin, ...args] = process.argv.slice(2);
const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
const child = spawn(bin, args, { detached: true, stdio: 'ignore' });
const exited = new Promise((resolve) => child.on('exit', (code, signal) => resolve(`${code} ${signal}`)));
let enginePid = 0;
for (let i = 0; i < 150 && !enginePid; i += 1) {
  if (existsSync(pidFile)) enginePid = Number(readFileSync(pidFile, 'utf8').trim()) || 0;
  if (!enginePid) await sleep(100);
}
const alive = (pid) => { try { process.kill(pid, 0); return true; } catch { return false; } };
process.kill(mode === 'group' ? -child.pid : child.pid, sig);
const result = await Promise.race([exited, sleep(5000).then(() => 'hung hung')]);
console.log(`${result} ${enginePid && alive(enginePid) ? 'orphan' : 'clean'}`);
for (const target of [-child.pid, enginePid]) if (target) try { process.kill(target, 'SIGKILL'); } catch {}
JS
engine_env() {
  env -u DEVRITES_HOST_ARTIFACT_DIR -u DEVRITES_ENGINE_CLI FAKE_RELEASE_ENGINE="$SLEEP_ENGINE" ATTESTED_ENGINE="$SLEEP_ENGINE" \
    NODE_OPTIONS="--import=$FAKEBIN/fetch-mock.mjs" PATH="$GH_BIN:$NODE_DIR:/usr/bin:/bin:/usr/sbin:/sbin" "$@"
}
for engine_case in group:INT:130 group:TERM:143 group:HUP:129 wrapper:INT:130 wrapper:TERM:143 wrapper:HUP:129; do
  IFS=: read -r mode sig want_status <<<"$engine_case"
  case_tmp="$FAKEBIN/engine-$mode-$sig-tmp"
  mkdir "$case_tmp"
  set -- $(engine_env ENGINE_PID_FILE="$FAKEBIN/engine-$mode-$sig.pid" TMPDIR="$case_tmp" \
    node "$FAKEBIN/signal-driver.mjs" "$mode" "SIG$sig" "$FAKEBIN/engine-$mode-$sig.pid" "$BIN" --target "$TARGET_BINARY" --dry-run)
  [ "${1:-}" = "$want_status" ] && [ "${2:-}" = null ] &&
    ok "SIG$sig to the $mode ends a running engine with status $want_status" ||
    no "SIG$sig to the $mode ended a running engine with '${*:-nothing}', want status $want_status"
  [ "${3:-}" = clean ] && ok "SIG$sig to the $mode leaves no orphan engine" ||
    no "SIG$sig to the $mode left the engine running"
  find "$case_tmp" -mindepth 1 -print -quit | grep -q . &&
    no "SIG$sig to the $mode leaked a temporary directory" ||
    ok "SIG$sig to the $mode cleaned the temporary directory"
done
set --
exit_tmp="$FAKEBIN/engine-exit-tmp"
mkdir "$exit_tmp"
engine_env ENGINE_EXIT=7 TMPDIR="$exit_tmp" "$BIN" --target "$TARGET_BINARY" --dry-run >/dev/null 2>&1
exit_status=$?
[ "$exit_status" -eq 7 ] && ok "a running engine's exit status passes through" ||
  no "engine exit status 7 became $exit_status"
find "$exit_tmp" -mindepth 1 -print -quit | grep -q . &&
  no "a finished engine run leaked a temporary directory" ||
  ok "a finished engine run cleaned the temporary directory"

# An engine that cannot start is reported with status 127 and an explanation.
LAUNCH_TMP="$FAKEBIN/launch-tmp"
mkdir "$LAUNCH_TMP"
printf '#!/nonexistent/interpreter\n' >"$FAKEBIN/missing-interpreter-engine"
chmod +x "$FAKEBIN/missing-interpreter-engine"
printf '#!/bin/sh\nexit 0\n' >"$FAKEBIN/not-executable-engine"
chmod 644 "$FAKEBIN/not-executable-engine"
engine_env DEVRITES_ENGINE_CLI="$FAKEBIN/missing-interpreter-engine" TMPDIR="$LAUNCH_TMP" \
  "$BIN" --target "$TARGET_BINARY" --dry-run >"$FAKEBIN/enoent-output" 2>&1
launch_status=$?
[ "$launch_status" -eq 127 ] && ok "an engine that cannot be found exits 127" ||
  no "an engine that cannot be found exited $launch_status, want 127"
grep -Fq 'devrites-engine was not found and could not be acquired' "$FAKEBIN/enoent-output" &&
  ok "an engine that cannot be found is explained" ||
  no "an engine that cannot be found was not explained"
engine_env DEVRITES_ENGINE_CLI="$FAKEBIN/not-executable-engine" TMPDIR="$LAUNCH_TMP" \
  "$BIN" --target "$TARGET_BINARY" --dry-run >"$FAKEBIN/eacces-output" 2>&1
launch_status=$?
[ "$launch_status" -eq 127 ] && ok "an engine that cannot be executed exits 127" ||
  no "an engine that cannot be executed exited $launch_status, want 127"
grep -Fq 'devrites: failed to launch devrites-engine:' "$FAKEBIN/eacces-output" &&
  grep -Fq EACCES "$FAKEBIN/eacces-output" &&
  ok "an engine that cannot be executed is explained" ||
  no "an engine that cannot be executed was not explained"
printf '\001\002\003 not an executable image\n' >"$FAKEBIN/invalid-format-engine"
chmod +x "$FAKEBIN/invalid-format-engine"
engine_env DEVRITES_ENGINE_CLI="$FAKEBIN/invalid-format-engine" TMPDIR="$LAUNCH_TMP" \
  "$BIN" --target "$TARGET_BINARY" --dry-run >"$FAKEBIN/enoexec-output" 2>&1
launch_status=$?
[ "$launch_status" -eq 127 ] && ok "an engine in an invalid executable format exits 127" ||
  no "an engine in an invalid executable format exited $launch_status, want 127"
# Only platforms whose spawn reports ENOEXEC can explain it; glibc execvp reruns the file under /bin/sh instead.
if [ "$(uname -s)" = Darwin ]; then
  grep -Fq 'devrites: failed to launch devrites-engine:' "$FAKEBIN/enoexec-output" &&
    grep -Fq "$FAKEBIN/invalid-format-engine" "$FAKEBIN/enoexec-output" &&
    ok "an engine in an invalid executable format is explained" ||
    no "an engine in an invalid executable format was not explained"
fi

# A PATH engine of another version is not a substitute for a failed release acquisition.
STALE_BIN="$FAKEBIN/stale-engine"
STALE_LOG="$FAKEBIN/stale-engine.log"
STALE_TMP="$FAKEBIN/stale-tmp"
mkdir "$STALE_BIN" "$STALE_TMP"
: >"$STALE_LOG"
cat >"$STALE_BIN/devrites-engine" <<'SH'
#!/bin/sh
[ "$1" = version ] && { echo v0.0.1; exit 0; }
echo "$*" >>"$STALE_LOG"
SH
chmod +x "$STALE_BIN/devrites-engine"
export STALE_LOG
if env -u DEVRITES_HOST_ARTIFACT_DIR -u DEVRITES_ENGINE_CLI FETCH_LOG="$FAKEBIN/stale-fetch.log" FAKE_RELEASE_TAG=v0.0.0-none TMPDIR="$STALE_TMP" \
  NODE_OPTIONS="--import=$FAKEBIN/fetch-mock.mjs" PATH="$STALE_BIN:$FAIL_PATH" \
  "$BIN" update --target "$TARGET_BINARY" --dry-run >"$FAKEBIN/stale-output" 2>&1; then
  no "update ran a PATH engine of another version after release acquisition failed"
else
  ok "update refuses a PATH engine of another version"
fi
grep -Fq "release v$want" "$FAKEBIN/stale-output" &&
  ok "stale PATH engine refusal names the release" ||
  no "stale PATH engine refusal did not name the release"
grep -Fq 'update --source-dir' "$STALE_LOG" &&
  no "stale PATH engine was invoked for update" ||
  ok "stale PATH engine not invoked for update"

# uninstall prefers a PATH engine that supports it over a release download or a Go build.
UNINSTALL_BIN="$FAKEBIN/uninstall-engine"
UNINSTALL_LOG="$FAKEBIN/uninstall-engine.log"
UNINSTALL_FETCH_LOG="$FAKEBIN/uninstall-fetch.log"
UNINSTALL_TMP="$FAKEBIN/uninstall-tmp"
mkdir "$UNINSTALL_BIN" "$UNINSTALL_TMP"
cat >"$UNINSTALL_BIN/devrites-engine" <<'SH'
#!/bin/sh
[ "$1" = uninstall ] && [ "$2" = -h ] && exit "${UNINSTALL_HELP_EXIT:-0}"
[ "$1" = version ] && { echo v0.0.1; exit 0; }
echo "$*" >>"$UNINSTALL_LOG"
SH
chmod +x "$UNINSTALL_BIN/devrites-engine"
export UNINSTALL_LOG
: >"$UNINSTALL_LOG"
: >"$UNINSTALL_FETCH_LOG"
env -u DEVRITES_HOST_ARTIFACT_DIR -u DEVRITES_ENGINE_CLI FETCH_LOG="$UNINSTALL_FETCH_LOG" TMPDIR="$UNINSTALL_TMP" \
  NODE_OPTIONS="--import=$FAKEBIN/fetch-mock.mjs" PATH="$UNINSTALL_BIN:$FAIL_PATH" \
  "$BIN" uninstall --target "$TARGET_BINARY" --dry-run >/dev/null 2>&1
grep -Fq 'uninstall --source-dir' "$UNINSTALL_LOG" &&
  ok "uninstall runs the PATH engine" || no "uninstall did not run the PATH engine"
[ -s "$UNINSTALL_FETCH_LOG" ] &&
  no "uninstall fetched a release before using the PATH engine" ||
  ok "uninstall with a PATH engine fetches nothing"
: >"$UNINSTALL_FETCH_LOG"
env -u DEVRITES_HOST_ARTIFACT_DIR -u DEVRITES_ENGINE_CLI UNINSTALL_HELP_EXIT=2 FETCH_LOG="$UNINSTALL_FETCH_LOG" TMPDIR="$UNINSTALL_TMP" \
  NODE_OPTIONS="--import=$FAKEBIN/fetch-mock.mjs" PATH="$UNINSTALL_BIN:$FAIL_PATH" \
  "$BIN" uninstall --target "$TARGET_BINARY" --dry-run >/dev/null 2>&1
[ -s "$UNINSTALL_FETCH_LOG" ] &&
  ok "uninstall acquires a release when the PATH engine lacks uninstall" ||
  no "uninstall trusted a PATH engine that lacks uninstall"

# The attestation check is bounded and a timeout is a refusal.
cat >"$FAKEBIN/gh-spy.mjs" <<'JS'
import cp from 'node:child_process';
import { syncBuiltinESMExports } from 'node:module';
import { appendFileSync } from 'node:fs';
const real = cp.spawnSync;
cp.spawnSync = (command, args, options) => {
  if (command !== 'gh') return real(command, args, options);
  appendFileSync(process.env.GH_SPY_LOG, `${options?.timeout}\n`);
  if (process.env.GH_SIM_TIMEOUT === '1') return { error: Object.assign(new Error('spawnSync gh ETIMEDOUT'), { code: 'ETIMEDOUT' }), status: null, signal: 'SIGTERM' };
  return real(command, args, options);
};
syncBuiltinESMExports();
JS
GH_SPY_LOG="$FAKEBIN/gh-spy.log"
ATTEST_TMP="$FAKEBIN/attest-tmp"
mkdir "$ATTEST_TMP"
export GH_SPY_LOG
: >"$GH_SPY_LOG"
env -u DEVRITES_HOST_ARTIFACT_DIR -u DEVRITES_ENGINE_CLI TMPDIR="$ATTEST_TMP" \
  NODE_OPTIONS="--import=$FAKEBIN/fetch-mock.mjs --import=$FAKEBIN/gh-spy.mjs" PATH="$GH_BIN:$FAIL_PATH" \
  "$BIN" --target "$TARGET_BINARY" --dry-run >/dev/null 2>&1
grep -Fxq 120000 "$GH_SPY_LOG" && ok "Node attestation verification is bounded" ||
  no "Node attestation verification has no timeout"
env -u DEVRITES_HOST_ARTIFACT_DIR -u DEVRITES_ENGINE_CLI GH_SIM_TIMEOUT=1 TMPDIR="$ATTEST_TMP" \
  NODE_OPTIONS="--import=$FAKEBIN/fetch-mock.mjs --import=$FAKEBIN/gh-spy.mjs" PATH="$GH_BIN:$FAIL_PATH" \
  "$BIN" --target "$TARGET_BINARY" --dry-run >"$FAKEBIN/attest-timeout.out" 2>&1 &&
  no "a timed-out attestation was accepted" ||
  { grep -Fq 'attestation verification failed' "$FAKEBIN/attest-timeout.out" &&
    ok "a timed-out Node attestation is refused" || no "a timed-out Node attestation failed for another reason"; }

# The shell bootstrap bounds the same check: a gh that outlives the limit is a refusal.
SHELL_ATTEST="$FAKEBIN/shell-attest"
mkdir "$SHELL_ATTEST" "$SHELL_ATTEST/bin" "$SHELL_ATTEST/src"
printf '{"version":"1.2.3"}\n' >"$SHELL_ATTEST/src/package.json"
printf 'payload\n' >"$SHELL_ATTEST/asset"
cat >"$SHELL_ATTEST/bin/curl" <<'SH'
#!/bin/sh
for url do :; done
case "$url" in
  *.sha256) printf '%s  %s\n' "$(shasum -a 256 "$SHELL_ATTEST/asset" | awk '{print $1}')" "${url##*/}" | sed 's/\.sha256$//' ;;
  *) cat "$SHELL_ATTEST/asset" ;;
esac
SH
printf '#!/bin/sh\nsleep 5\nexit 0\n' >"$SHELL_ATTEST/bin/gh"
chmod +x "$SHELL_ATTEST/bin/curl" "$SHELL_ATTEST/bin/gh"
export SHELL_ATTEST
attest_started=$SECONDS
PATH="$SHELL_ATTEST/bin:$PATH" DEVRITES_REF=v1.2.3 DR_ATTEST_TIMEOUT=1 bash -c '
  source "$1"
  dr_download_engine "$2" owner/repo "$3" || { printf "%s\n" "$DR_ACQUIRE_FAILURE"; exit 3; }
' _ "$ROOT/scripts/install-lib.sh" "$SHELL_ATTEST/src" "$SHELL_ATTEST/engine" >"$SHELL_ATTEST/out" 2>&1
attest_status=$?
[ "$attest_status" -eq 3 ] && grep -Fq 'attestation verification failed' "$SHELL_ATTEST/out" &&
  [ "$((SECONDS - attest_started))" -lt 5 ] && [ ! -e "$SHELL_ATTEST/engine" ] &&
  ok "a shell attestation that outlives its limit is refused" ||
  no "a shell attestation that outlives its limit was not refused promptly (status $attest_status)"

# A gh that finishes in time leaves no watchdog or sleep behind.
printf '#!/bin/sh\nexit 0\n' >"$SHELL_ATTEST/bin/gh"
attest_marker=$((40000 + $$ % 20000))
rm -f "$SHELL_ATTEST/engine"
PATH="$SHELL_ATTEST/bin:$PATH" DEVRITES_REF=v1.2.3 DR_ATTEST_TIMEOUT="$attest_marker" bash -c '
  source "$1"
  dr_download_engine "$2" owner/repo "$3"
' _ "$ROOT/scripts/install-lib.sh" "$SHELL_ATTEST/src" "$SHELL_ATTEST/engine" >/dev/null 2>&1
attest_status=$?
sleep 0.3
if [ "$attest_status" -eq 0 ] && ! pgrep -f "sleep $attest_marker" >/dev/null; then
  ok "a shell attestation that finishes in time leaves no watchdog behind"
else
  pkill -f "sleep $attest_marker" >/dev/null 2>&1
  no "a shell attestation that finishes in time left a watchdog behind (status $attest_status)"
fi

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
  ".pi/skills/rite/SKILL.md" \
  ".pi/agents/devrites-code-reviewer.md" \
  ".pi/prompts/rite-build.md" \
  ".claude/skills/devrites-lib/reference/standards/security.md" \
  "AGENTS.md" \
  ".devrites/ACTIVE"; do
  [ -f "$TARGET/$f" ] && ok "installed: $f" || no "missing after install: $f"
done
[ ! -e "$TARGET/.codex/hooks.json" ] && ok "packaged install creates no Codex root hooks" || no "packaged install created Codex root hooks"

# 8) the published package path reaches one real structural workspace check.
mkdir -p "$TARGET/.devrites/work/quickstart"
cat >"$TARGET/.devrites/work/quickstart/README.md" <<'EOF'
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
cat >"$TARGET/.devrites/work/quickstart/state.md" <<'EOF'
## Cursor
| Key | Value |
| --- | --- |
| phase | frame |
| status | running |
EOF
printf 'quickstart\n' >"$TARGET/.devrites/ACTIVE"
readiness="$(DEVRITES_ROOT="$TARGET/.devrites" "$STAGED_ENGINE" check readiness quickstart 2>/dev/null)"
printf '%s' "$readiness" | grep -Fq 'reason: DRV-GATE-READINESS-PASSED' &&
  ok "published install path reaches structural workspace readiness" ||
  no "published install path did not reach structural workspace readiness"
printf '  evidence: candidate_head=%s package_sha256=%s observed_elapsed=%ss\n' \
  "$candidate_head" "$candidate_sha" "$((SECONDS - quickstart_started))"

# 9) project-local guarantee holds through the packaged path
[ -e "$HOME/.claude/skills/rite" ] && no "wrote to ~/.claude !!" || ok "~/.claude untouched"
[ -e "$HOME/.codex/agents/devrites-code-reviewer.toml" ] && no "wrote to ~/.codex !!" || ok "~/.codex untouched"
[ -e "$HOME/.pi/agents/devrites-code-reviewer.md" ] && no "wrote to ~/.pi !!" || ok "~/.pi untouched"

echo ""
[ "$fail" -eq 0 ] && echo "npx-pack-smoke: PASS" || echo "npx-pack-smoke: FAIL"
exit "$fail"
