#!/usr/bin/env bash
# Exercise the global devrites-engine lifecycle from issue 10 in a temporary
# directory, including install, update, downgrade, and uninstall.
# DEVRITES_BIN_DIR keeps the real system paths untouched, and DEVRITES_REF pins
# the stamped version. A failed release download falls back to `go build` from
# engine/, which keeps the test offline
# and deterministic. The test skips itself without Go or an engine checkout.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

command -v go >/dev/null 2>&1 || { echo "  SKIP: go toolchain not available"; exit 0; }
[ -d "$ROOT/engine" ] || { echo "  SKIP: no engine/ (tarball install)"; exit 0; }

BIN="$(mktemp -d)"; TGT="$(mktemp -d)"; ENGINE_TMP_ROOT="$(mktemp -d)"; NETBIN="$(mktemp -d)"; GEN=""
GOCACHE="${GOCACHE:-$BIN/go-cache}"
mkdir -p "$GOCACHE"
export GOCACHE
trap 'rm -rf "$BIN" "$TGT" "$ENGINE_TMP_ROOT" "$NETBIN"; [ -n "$GEN" ] && rm -rf "$GEN"' EXIT
printf '#!/bin/sh\nexit 22\n' > "$NETBIN/curl"
chmod +x "$NETBIN/curl"
if [ -z "${DEVRITES_HOST_ARTIFACT_DIR:-}" ]; then
  GEN="$(mktemp -d)"
  DEVRITES_HOST_ARTIFACT_DIR="$GEN" bash "$ROOT/scripts/build-host-artifacts.sh" >/dev/null 2>&1 \
    || { echo "  FAIL: could not build host artifacts"; exit 1; }
  export DEVRITES_HOST_ARTIFACT_DIR="$GEN"
fi
export DEVRITES_BIN_DIR="$BIN"
export PATH="$NETBIN:$BIN:$PATH"   # keep acquisition offline and let the downgrade guard resolve the staged engine

echo "== binary-lifecycle (bin: $BIN) =="

# 1) DEVRITES_NO_BINARY / --no-binary skips the binary entirely.
DEVRITES_NO_BINARY=1 bash "$ROOT/install.sh" --target "$TGT" --force >/dev/null 2>&1
[ ! -e "$BIN/devrites-engine" ] && ok "DEVRITES_NO_BINARY skips the binary" || no "binary installed despite DEVRITES_NO_BINARY"

# 2) dry-run plans the binary but installs nothing.
out="$(bash "$ROOT/install.sh" --target "$TGT" --dry-run 2>&1)"
printf '%s' "$out" | grep -q 'would install the global devrites-engine control-plane binary' && ok "dry-run plans the binary" || no "dry-run missing the binary plan line"
[ ! -e "$BIN/devrites-engine" ] && ok "dry-run writes no binary" || no "dry-run wrote a binary"

# 3) real install builds from source and stamps the pinned version.
DEVRITES_REF="v9.9.9" bash "$ROOT/install.sh" --target "$TGT" --force >/dev/null 2>&1
[ -x "$BIN/devrites-engine" ] && ok "binary installed to DEVRITES_BIN_DIR" || no "binary not installed"
[ ! -e "$BIN/devrites" ] && ok "installer does not overwrite the npm devrites shim name" || no "installer wrote colliding devrites shim name"
v="$("$BIN/devrites-engine" version 2>/dev/null || true)"
[ "$v" = "v9.9.9" ] && ok "version stamped ($v)" || no "version wrong: '$v' (want v9.9.9)"

# 4) update replaces a newer release in place.
DEVRITES_REF="v9.9.10" bash "$ROOT/install.sh" --target "$TGT" --force >/dev/null 2>&1
v="$("$BIN/devrites-engine" version 2>/dev/null || true)"
[ "$v" = "v9.9.10" ] && ok "update replaced in place ($v)" || no "update did not replace: '$v'"

# 5) downgrade guard: an older release is refused; the newer binary stays.
DEVRITES_REF="v0.0.1" bash "$ROOT/install.sh" --target "$TGT" --force >/dev/null 2>&1
v="$("$BIN/devrites-engine" version 2>/dev/null || true)"
[ "$v" = "v9.9.10" ] && ok "downgrade refused (kept $v)" || no "downgrade not refused: now '$v'"

# 6) uninstall removes the shared binary.
bash "$ROOT/uninstall.sh" --target "$TGT" >/dev/null 2>&1
[ ! -e "$BIN/devrites-engine" ] && ok "uninstall removed the binary" || no "uninstall left the binary"

# 7) uninstall --keep-binary leaves it in place.
DEVRITES_REF="v9.9.9" bash "$ROOT/install.sh" --target "$TGT" --force >/dev/null 2>&1
bash "$ROOT/uninstall.sh" --target "$TGT" --keep-binary >/dev/null 2>&1
[ -x "$BIN/devrites-engine" ] && ok "uninstall --keep-binary keeps it" || no "--keep-binary removed the binary"

# 8) An unstamped source build ("dev") can be upgraded to a release. The
#    downgrade guard compares only semantic versions, so "dev" is never newer.
( cd "$ROOT/engine" && CGO_ENABLED=0 go build -o "$BIN/devrites-engine" . ) >/dev/null 2>&1
[ "$("$BIN/devrites-engine" version 2>/dev/null)" = "dev" ] && ok "planted a dev build" || no "could not plant a dev build"
DEVRITES_REF="v9.9.9" bash "$ROOT/install.sh" --target "$TGT" --force >/dev/null 2>&1
v="$("$BIN/devrites-engine" version 2>/dev/null || true)"
[ "$v" = "v9.9.9" ] && ok "dev build upgraded to a release ($v)" || no "dev build was not upgraded: '$v' (downgrade guard false positive)"

# 9) a wrong staged version is rejected before replacement.
STAGED_WRONG="$(mktemp)"
cat > "$STAGED_WRONG" <<'EOF'
#!/bin/sh
[ "$1" = version ] && echo 1.0.0
EOF
chmod +x "$STAGED_WRONG"
DEVRITES_ENGINE_CLI="$STAGED_WRONG" DEVRITES_REF="v9.9.11" "$BIN/devrites-engine" install \
  --source-dir "$ROOT" --payload-dir "$DEVRITES_HOST_ARTIFACT_DIR" --target "$TGT" --force >/dev/null 2>&1 \
  && no "wrong staged version was accepted" || ok "wrong staged version rejected before replacement"
v="$("$BIN/devrites-engine" version 2>/dev/null || true)"
[ "$v" = "v9.9.9" ] && ok "wrong staged version left old binary intact" || no "wrong staged version changed binary: '$v'"
rm -f "$STAGED_WRONG"

# 10) post-install verification failure restores the same-directory backup.
STAGED_POST="$(mktemp)"
cat > "$STAGED_POST" <<EOF
#!/bin/sh
if [ "\$1" = version ]; then
  case "\$0" in
    "$BIN/devrites-engine") echo 9.9.12 ;;
    *) echo 9.9.11 ;;
  esac
fi
EOF
chmod +x "$STAGED_POST"
DEVRITES_ENGINE_CLI="$STAGED_POST" DEVRITES_REF="v9.9.11" "$BIN/devrites-engine" install \
  --source-dir "$ROOT" --payload-dir "$DEVRITES_HOST_ARTIFACT_DIR" --target "$TGT" --force >/dev/null 2>&1 \
  && no "post-install wrong version was accepted" || ok "post-install wrong version rejected"
v="$("$BIN/devrites-engine" version 2>/dev/null || true)"
[ "$v" = "v9.9.9" ] && ok "post-install failure restored old binary" || no "rollback did not restore old binary: '$v'"
rm -f "$STAGED_POST"

# 11) The shim owns and removes a source-built engine directory after exit.
TMPDIR="$ENGINE_TMP_ROOT" DEVRITES_REF="v9.9.13" bash "$ROOT/install.sh" --target "$TGT" --dry-run >/dev/null 2>&1
if find "$ENGINE_TMP_ROOT" -mindepth 1 -print -quit | grep -q .; then
  no "shell shim leaked its temporary engine directory"
else
  ok "shell shim removes its temporary engine directory"
fi

# 12) Non-exact versions fail closed before a build or download can be selected.
TMPDIR="$ENGINE_TMP_ROOT" DEVRITES_REF="main" bash "$ROOT/install.sh" --target "$TGT" --dry-run >/dev/null 2>&1 \
  && no "non-semver engine reference was accepted" \
  || ok "non-semver engine reference rejected"

[ "$fail" -eq 0 ] && echo "PASS: binary lifecycle" || echo "FAILED: binary lifecycle"
exit "$fail"
