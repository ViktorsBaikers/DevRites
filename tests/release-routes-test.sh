#!/usr/bin/env bash
set -euo pipefail
# Acquisition arms must not short-circuit on an engine override inherited from a runner.
unset DEVRITES_ENGINE_CLI DEVRITES_CLI
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
MOCKBIN="$TMP/mockbin"
mkdir -p "$MOCKBIN"
cat > "$MOCKBIN/curl" <<'EOF'
#!/bin/sh
for url do :; done
case "$url" in *.sha256) cat "$MOCK_SIDECAR" ;; *) cat "$MOCK_ASSET" ;; esac
EOF
cat > "$MOCKBIN/devrites-engine" <<'EOF'
#!/bin/sh
exit 22
EOF
chmod +x "$MOCKBIN/curl" "$MOCKBIN/devrites-engine"
make_asset() { printf '#!/bin/sh
printf %s > "$MARKER"
exit 23
' "$2" > "$1"; chmod +x "$1"; }
make_asset "$TMP/trusted" success
make_asset "$TMP/attacker" attacker
sha256() { shasum -a 256 "$1" | awk '{print $1}'; }

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
  export MOCK_ASSET="$payload" MOCK_SIDECAR="$sidecar" PATH="$MOCKBIN:$PATH" MARKER="$TMP/$label.marker" DEVRITES_REF=v1.2.3 DEVRITES_REPO=owner/repo
  output="$TMP/$label.engine"
  DR_ACQUIRE_FAILURE=
  if dr_download_engine "$SHELL_SOURCE" owner/repo "$output"; then set +e; "$output"; status=$?; set -e; else status=1; fi
  if [ "$expected" = pass ]; then
    [ "$status" -eq 23 ] && [ "$(cat "$MARKER" 2>/dev/null)" = success ] || { echo "FAIL install-lib $label positive arm" >&2; exit 1; }
  else
    case "$DR_ACQUIRE_FAILURE" in *'checksum failed'*) ;; *) echo "FAIL install-lib $label was not refused by the sidecar checksum: $DR_ACQUIRE_FAILURE" >&2; exit 1 ;; esac
    [ "$status" -ne 23 ] && [ ! -e "$MARKER" ] && [ ! -e "$output" ] || { echo "FAIL install-lib $label ran or staged a sidecar-mismatched asset" >&2; exit 1; }
  fi
}
run_shell_case trusted "$TMP/trusted" pass
run_shell_case mismatch "$TMP/trusted" mismatch
echo 'install-lib ARM1 OK: marker=success, exit 23; sidecar mismatch refused'

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
  marker="$TMP/$label.npm.marker"
  set +e
  (cd "$NPMROOT" && MOCK_ASSET="$payload" MOCK_SIDECAR="$sidecar" MARKER="$marker" PATH="$NPMBIN:$NODE_DIR:/usr/bin:/bin" NODE_OPTIONS="--import=$NPMROOT/mock-fetch.mjs" DEVRITES_REF=1.2.3 DEVRITES_REPO=owner/repo node bin/devrites.mjs install) > "$TMP/$label.npm.out" 2>&1
  status=$?
  set -e
  if [ "$expected" = pass ]; then
    [ "$status" -eq 23 ] && [ "$(cat "$marker" 2>/dev/null)" = success ] || { cat "$TMP/$label.npm.out" >&2; echo "FAIL npm $label positive arm ($status)" >&2; exit 1; }
  else
    [ "$status" -ne 23 ] && [ ! -e "$marker" ] || { echo "FAIL npm $label ran an asset with a bad sidecar" >&2; exit 1; }
    grep -q 'checksum failed' "$TMP/$label.npm.out" || { cat "$TMP/$label.npm.out" >&2; echo "FAIL npm $label was not refused by the sidecar checksum" >&2; exit 1; }
  fi
}
run_npm_case trusted "$TMP/trusted" pass
run_npm_case mismatch "$TMP/trusted" mismatch
run_npm_case malformed "$TMP/trusted" malformed
run_npm_case empty "$TMP/trusted" empty
echo 'npm ARM1 OK: marker=success, exit 23; sidecar mismatch, malformed and empty sidecars refused'
