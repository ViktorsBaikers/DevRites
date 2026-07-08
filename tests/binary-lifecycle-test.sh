#!/usr/bin/env bash
# binary-lifecycle-test.sh — the global devrites-engine binary's install / update / downgrade
# / uninstall lifecycle (issue 10), exercised hermetically. DEVRITES_BIN_DIR points
# the installer at a throwaway bin dir (never the real /usr/local/bin or ~/.local/bin)
# and DEVRITES_REF pins the stamped version; the release download 404s and falls back
# to `go build` from engine/, so the version is deterministic and offline-safe.
# Requires the Go toolchain + an engine/ checkout; self-skips otherwise.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

command -v go >/dev/null 2>&1 || { echo "  SKIP: go toolchain not available"; exit 0; }
[ -d "$ROOT/engine" ] || { echo "  SKIP: no engine/ (tarball install)"; exit 0; }

BIN="$(mktemp -d)"; TGT="$(mktemp -d)"
trap 'rm -rf "$BIN" "$TGT"' EXIT
export DEVRITES_BIN_DIR="$BIN"
export PATH="$BIN:$PATH"   # so the downgrade guard's `command -v devrites-engine` resolves it

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

# 8) an unstamped source build (version "dev") must be UPGRADABLE to a release —
#    the downgrade guard only compares real semver, so "dev" is never "newest".
( cd "$ROOT/engine" && CGO_ENABLED=0 go build -o "$BIN/devrites-engine" . ) >/dev/null 2>&1
[ "$("$BIN/devrites-engine" version 2>/dev/null)" = "dev" ] && ok "planted a dev build" || no "could not plant a dev build"
DEVRITES_REF="v9.9.9" bash "$ROOT/install.sh" --target "$TGT" --force >/dev/null 2>&1
v="$("$BIN/devrites-engine" version 2>/dev/null || true)"
[ "$v" = "v9.9.9" ] && ok "dev build upgraded to a release ($v)" || no "dev build NOT upgraded: '$v' (downgrade guard false-positive)"

[ "$fail" -eq 0 ] && echo "PASS: binary lifecycle" || echo "FAILED: binary lifecycle"
exit "$fail"
