#!/usr/bin/env bash
set -euo pipefail
# Acquisition arms must not short-circuit on an engine override inherited from a runner.
unset DEVRITES_ENGINE_CLI DEVRITES_CLI
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
MOCKBIN="$TMP/mockbin"
mkdir -p "$MOCKBIN"
cat > "$MOCKBIN/gh" <<'EOF'
#!/bin/sh
printf "%s\n" "$*" >> "$GH_LOG"
[ "$1" = attestation ] && [ "$2" = verify ] && [ "$4" = --repo ] && [ "$5" = owner/repo ] && [ "$6" = --signer-workflow ] && [ "$7" = owner/repo/.github/workflows/ci.yml@refs/heads/main ] || exit 64
got=$(shasum -a 256 "$3" | awk '{print $1}') || exit 65
grep -Fxq "$got" "$ATTESTED_DIGESTS"
EOF
cat > "$MOCKBIN/curl" <<'EOF'
#!/bin/sh
for url do :; done
case "$url" in *.sha256) cat "$MOCK_SIDECAR" ;; *) cat "$MOCK_ASSET" ;; esac
EOF
cat > "$MOCKBIN/devrites-engine" <<'EOF'
#!/bin/sh
exit 22
EOF
chmod +x "$MOCKBIN/gh" "$MOCKBIN/curl" "$MOCKBIN/devrites-engine"
make_asset() { printf '#!/bin/sh
printf %s > "$MARKER"
exit 23
' "$2" > "$1"; chmod +x "$1"; }
make_asset "$TMP/trusted" success
make_asset "$TMP/attacker" attacker
sha256() { shasum -a 256 "$1" | awk '{print $1}'; }

# The gh stand-in accepts only the pinned repository, signer workflow and attested digest.
sha256 "$TMP/trusted" > "$TMP/standin.allowed"
standin_gh() { GH_LOG=/dev/null ATTESTED_DIGESTS="$TMP/standin.allowed" "$MOCKBIN/gh" attestation verify "$TMP/$1" --repo "$2" --signer-workflow "$3" >/dev/null 2>&1; }
SIGNER=owner/repo/.github/workflows/ci.yml@refs/heads/main
standin_gh trusted owner/repo "$SIGNER" || { echo "FAIL gh stand-in rejected the pinned request" >&2; exit 1; }
standin_gh trusted other/repo "$SIGNER" && { echo "FAIL gh stand-in accepted a wrong repo" >&2; exit 1; }
standin_gh trusted owner/repo owner/repo/.github/workflows/other.yml@refs/heads/main && { echo "FAIL gh stand-in accepted a wrong signer workflow" >&2; exit 1; }
standin_gh attacker owner/repo "$SIGNER" && { echo "FAIL gh stand-in accepted an unattested digest" >&2; exit 1; }

# Exercise dr_download_engine and its execution handoff against local command stand-ins.
SHELL_SOURCE="$TMP/shell-source"
mkdir -p "$SHELL_SOURCE"
printf '{"version":"1.2.3"}
' > "$SHELL_SOURCE/package.json"
source "${INSTALL_LIB_SOURCE:-$ROOT/scripts/install-lib.sh}"
asset="devrites-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')"
run_shell_case() {
  label="$1"; payload="$2"; expected="$3"
  sidecar="$TMP/$label.sha256"
  sidecar_src="$payload"; [ "$expected" != mismatch ] || sidecar_src="$TMP/attacker"
  printf '%s  %s
' "$(sha256 "$sidecar_src")" "$asset" > "$sidecar"
  allowed="$TMP/$label.allowed"
  if [ "$expected" = fail ]; then sha256 "$TMP/trusted" > "$allowed"; else sha256 "$payload" > "$allowed"; fi
  export MOCK_ASSET="$payload" MOCK_SIDECAR="$sidecar" ATTESTED_DIGESTS="$allowed" GH_LOG="$TMP/$label.gh.log" PATH="$MOCKBIN:$PATH" MARKER="$TMP/$label.marker" DEVRITES_REF=v1.2.3 DEVRITES_REPO=owner/repo
  output="$TMP/$label.engine"
  DR_ACQUIRE_FAILURE=
  if dr_download_engine "$SHELL_SOURCE" owner/repo "$output"; then set +e; "$output"; status=$?; set -e; else status=1; fi
  if [ "$expected" = pass ]; then
    [ "$status" -eq 23 ] && [ "$(cat "$MARKER" 2>/dev/null)" = success ] || { echo "FAIL install-lib $label positive arm" >&2; exit 1; }
  elif [ "$expected" = mismatch ]; then
    case "$DR_ACQUIRE_FAILURE" in *'checksum failed'*) ;; *) echo "FAIL install-lib $label was not refused by the sidecar checksum: $DR_ACQUIRE_FAILURE" >&2; exit 1 ;; esac
    [ "$status" -ne 23 ] && [ ! -e "$MARKER" ] && [ ! -e "$output" ] || { echo "FAIL install-lib $label ran or staged a sidecar-mismatched asset" >&2; exit 1; }
  else
    [ "$status" -ne 23 ] && [ ! -e "$MARKER" ] && grep -q 'attestation verify' "$GH_LOG" || { echo "FAIL install-lib $label negative arm" >&2; exit 1; }
  fi
}
run_shell_case trusted "$TMP/trusted" pass
run_shell_case attacker "$TMP/attacker" fail
run_shell_case mismatch "$TMP/trusted" mismatch
echo 'install-lib ARM1 OK: marker=success, exit 23; ARM2 OK: refused before execution; sidecar mismatch refused'

# Fail closed: a digest-correct asset must still be refused when no verifier is installed.
NOGH_BIN="$TMP/nogh-bin"
mkdir -p "$NOGH_BIN"
cp "$MOCKBIN/curl" "$NOGH_BIN/curl"
# Every system tool except gh, so a runner that ships gh in /usr/bin still has none on PATH.
for f in /usr/bin/* /bin/*; do n="${f##*/}"; [ "$n" = gh ] || [ -e "$NOGH_BIN/$n" ] || ln -s "$f" "$NOGH_BIN/$n"; done
printf '%s  %s\n' "$(sha256 "$TMP/trusted")" "$asset" > "$TMP/nogh.sha256"
if PATH="$NOGH_BIN" command -v gh >/dev/null 2>&1; then echo "FAIL install-lib missing-gh harness: gh still resolvable" >&2; exit 1; fi
nogh_msg="$(PATH="$NOGH_BIN" MOCK_ASSET="$TMP/trusted" MOCK_SIDECAR="$TMP/nogh.sha256"; if dr_download_engine "$SHELL_SOURCE" owner/repo "$TMP/nogh.engine"; then echo ACCEPTED; else echo "$DR_ACQUIRE_FAILURE"; fi)"
case "$nogh_msg" in
  *'attestation verification failed'*) [ ! -e "$TMP/nogh.engine" ] || { echo "FAIL install-lib missing gh left the engine staged" >&2; exit 1; } ;;
  *) echo "FAIL install-lib accepted or mis-reported a missing gh: $nogh_msg" >&2; exit 1 ;;
esac
echo 'install-lib missing gh OK: refused'

# npm route: temporary package root plus in-process fetch mock, never the network.
NPMROOT="$TMP/npm-root"
mkdir -p "$NPMROOT/bin" "$NPMROOT/pack"
cp -R "$ROOT/pack/generated" "$NPMROOT/pack/generated"
cp "${DEVRITES_MJS_SOURCE:-$ROOT/bin/devrites.mjs}" "$NPMROOT/bin/devrites.mjs"
cp "$ROOT/package.json" "$NPMROOT/package.json"
cat > "$NPMROOT/mock-fetch.mjs" <<'EOF'
import { readFile } from 'node:fs/promises';
globalThis.fetch = async (url) => {
  const path = String(url);
  const file = path.endsWith('.sha256') ? process.env.MOCK_SIDECAR : process.env.MOCK_ASSET;
  return new Response(await readFile(file), { status: 200 });
};
EOF
# A PATH without devrites-engine, so a refused acquisition surfaces its reason.
NPMBIN="$TMP/npm-bin"
mkdir -p "$NPMBIN"
cp "$MOCKBIN/gh" "$NPMBIN/gh"
NODE_DIR="$(dirname "$(command -v node)")"
if PATH="$NPMBIN:$NODE_DIR:/usr/bin:/bin" command -v devrites-engine >/dev/null 2>&1; then echo "FAIL npm harness: devrites-engine still resolvable" >&2; exit 1; fi
run_npm_case() {
  label="$1"; payload="$2"; expected="$3"
  sidecar="$TMP/$label.npm.sha256"
  sidecar_src="$payload"; [ "$expected" != mismatch ] || sidecar_src="$TMP/attacker"
  case "$expected" in
    malformed) printf 'not-a-digest  %s\n' "$asset" > "$sidecar" ;;
    empty) : > "$sidecar" ;;
    *) printf '%s  %s
' "$(sha256 "$sidecar_src")" "$asset" > "$sidecar" ;;
  esac
  allowed="$TMP/$label.npm.allowed"
  if [ "$expected" = fail ]; then sha256 "$TMP/trusted" > "$allowed"; else sha256 "$payload" > "$allowed"; fi
  marker="$TMP/$label.npm.marker"
  set +e
  (cd "$NPMROOT" && MOCK_ASSET="$payload" MOCK_SIDECAR="$sidecar" ATTESTED_DIGESTS="$allowed" GH_LOG="$TMP/$label.npm.gh.log" MARKER="$marker" PATH="$NPMBIN:$NODE_DIR:/usr/bin:/bin" NODE_OPTIONS="--import=$NPMROOT/mock-fetch.mjs" DEVRITES_REF=1.2.3 DEVRITES_REPO=owner/repo node bin/devrites.mjs install) > "$TMP/$label.npm.out" 2>&1
  status=$?
  set -e
  if [ "$expected" = pass ]; then
    [ "$status" -eq 23 ] && [ "$(cat "$marker" 2>/dev/null)" = success ] || { cat "$TMP/$label.npm.out" >&2; echo "FAIL npm $label positive arm ($status)" >&2; exit 1; }
  elif [ "$expected" = mismatch ] || [ "$expected" = malformed ] || [ "$expected" = empty ]; then
    [ "$status" -ne 23 ] && [ ! -e "$marker" ] || { echo "FAIL npm $label ran an asset with a bad sidecar" >&2; exit 1; }
    grep -q 'checksum failed' "$TMP/$label.npm.out" || { cat "$TMP/$label.npm.out" >&2; echo "FAIL npm $label was not refused by the sidecar checksum" >&2; exit 1; }
  else
    [ "$status" -ne 23 ] && [ ! -e "$marker" ] || { echo "FAIL npm $label negative arm" >&2; exit 1; }
    grep -q 'attestation verify' "$TMP/$label.npm.gh.log" || { echo "FAIL npm $label did not invoke attestation verification" >&2; exit 1; }
  fi
}
run_npm_case trusted "$TMP/trusted" pass
run_npm_case attacker "$TMP/attacker" fail
run_npm_case mismatch "$TMP/trusted" mismatch
run_npm_case malformed "$TMP/trusted" malformed
run_npm_case empty "$TMP/trusted" empty
echo 'npm ARM1 OK: marker=success, exit 23; ARM2 OK: refused before execution; sidecar mismatch, malformed and empty sidecars refused'

# install.sh bootstrap: a gh that is installed but unauthenticated must be reported as an
# authentication failure naming `gh auth login`; an authenticated gh that rejects the
# attestation keeps the provenance message. Both stay fatal.
BOOT_DIR="$TMP/bootstrap"
mkdir -p "$BOOT_DIR/src"
cp "${INSTALL_SH_SOURCE:-$ROOT/install.sh}" "$BOOT_DIR/src/install.sh"
printf 'payload' | gzip -c > "$BOOT_DIR/devrites-v1.2.3.tar.gz"
printf '%s  devrites-v1.2.3.tar.gz\n' "$(sha256 "$BOOT_DIR/devrites-v1.2.3.tar.gz")" > "$BOOT_DIR/devrites-v1.2.3.tar.gz.sha256"
cat > "$BOOT_DIR/curl" <<'EOF2'
#!/bin/sh
for url do :; done
case "$url" in *.sha256) cat "$BOOT_SIDECAR" ;; *) cat "$BOOT_ASSET" ;; esac
EOF2
run_bootstrap_case() {
  label="$1"; auth_exit="$2"
  bin="$BOOT_DIR/$label-bin"
  mkdir -p "$bin"
  cp "$BOOT_DIR/curl" "$bin/curl"
  printf '#!/bin/sh\n[ "$1" = auth ] && exit %s\nexit 1\n' "$auth_exit" > "$bin/gh"
  chmod +x "$bin/curl" "$bin/gh"
  set +e
  PATH="$bin:/usr/bin:/bin" BOOT_ASSET="$BOOT_DIR/devrites-v1.2.3.tar.gz" BOOT_SIDECAR="$BOOT_DIR/devrites-v1.2.3.tar.gz.sha256" \
    DEVRITES_REF=1.2.3 DEVRITES_REPO=owner/repo TMPDIR="$TMP" bash "$BOOT_DIR/src/install.sh" > "$BOOT_DIR/$label.out" 2>&1
  status=$?
  set -e
  [ "$status" -ne 0 ] || { echo "FAIL bootstrap $label accepted an unverified release" >&2; exit 1; }
  grep -q 'attestation verification failed' "$BOOT_DIR/$label.out" || { cat "$BOOT_DIR/$label.out" >&2; echo "FAIL bootstrap $label did not report an attestation failure" >&2; exit 1; }
}
run_bootstrap_case unauth 1
grep -q 'not authenticated' "$BOOT_DIR/unauth.out" && grep -q 'gh auth login' "$BOOT_DIR/unauth.out" || { cat "$BOOT_DIR/unauth.out" >&2; echo "FAIL bootstrap with unauthenticated gh did not name authentication and gh auth login" >&2; exit 1; }
run_bootstrap_case authed 0
grep -q 'no build provenance' "$BOOT_DIR/authed.out" && ! grep -q 'gh auth login' "$BOOT_DIR/authed.out" || { cat "$BOOT_DIR/authed.out" >&2; echo "FAIL bootstrap with authenticated gh lost the provenance message or blamed authentication" >&2; exit 1; }
echo 'bootstrap unauthenticated gh OK: fatal, names authentication and gh auth login; authenticated gh keeps provenance message'
