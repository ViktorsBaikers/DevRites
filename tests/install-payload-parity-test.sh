#!/usr/bin/env bash
# The payload completeness check in scripts/install-lib.sh (dr_payload_complete) must agree with
# the engine's required payload (hostpack.RequiredPayload), and install.sh/update.sh must rebuild
# the payload whenever a required host tree (including omp/ and pi/) is missing.
# PARITY_ROOT points the test at another source tree (e.g. a scratch copy of an older revision).
set -u
ROOT="${PARITY_ROOT:-$(cd "$(dirname "$0")/.." && pwd -P)}"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

echo "== install payload parity =="

# Required payload paths, read from the engine source (all hosts enabled).
sed -n '/^func RequiredPayload/,/^}/p' "$ROOT/engine/internal/hostpack/hostpack.go" \
  | grep -o '"[a-z][^"]*"' | tr -d '"' | sort -u > "$TMP/required"
for h in claude codex omp pi devin; do
  grep -q "^$h/" "$TMP/required" || no "engine required list has no $h/ entry"
done

mkpayload() { # dir: every engine-required path plus each host's standards file
  local h rel
  mkdir -p "$1"
  while IFS= read -r rel; do
    case "${rel##*/}" in *.*) mkdir -p "$1/$(dirname "$rel")"; : > "$1/$rel" ;; *) mkdir -p "$1/$rel" ;; esac
  done < "$TMP/required"
  for h in claude codex omp pi devin; do
    mkdir -p "$1/$h/skills/devrites-lib/reference/standards"
    : > "$1/$h/skills/devrites-lib/reference/standards/agents.md"
  done
}

mkpayload "$TMP/full"
if ( . "$ROOT/scripts/install-lib.sh"; dr_payload_complete "$TMP/full" ); then
  ok "full engine payload is complete"
else
  no "dr_payload_complete rejects a payload holding every engine-required path"
fi
while IFS= read -r rel; do
  mkpayload "$TMP/p"
  rm -rf "${TMP:?}/p/$rel"
  if ( . "$ROOT/scripts/install-lib.sh"; dr_payload_complete "$TMP/p" ); then
    no "dr_payload_complete accepts a payload missing $rel"
  fi
  rm -rf "$TMP/p"
done < "$TMP/required"
[ "$fail" -eq 0 ] && ok "every engine-required path is enforced by dr_payload_complete"

# install.sh and update.sh must rebuild the payload when a host tree is missing, and only then.
S="$TMP/src"; mkdir -p "$S/scripts" "$S/pack" "$TMP/bin"
cp "$ROOT/install.sh" "$ROOT/update.sh" "$S/"
cp "$ROOT/scripts/install-lib.sh" "$S/scripts/"
printf '{"version":"1.2.3"}\n' > "$S/package.json"
printf '#!/bin/sh\necho invoked >> "%s/marker"\nexit 0\n' "$TMP" > "$S/scripts/build-host-artifacts.sh"
printf '#!/bin/sh\ncase "$1" in version) echo v1.2.3 ;; esac\nexit 0\n' > "$TMP/bin/engine"
chmod +x "$TMP/bin/engine"
run() { # script payload
  rm -f "$TMP/marker"
  DEVRITES_ENGINE_CLI="$TMP/bin/engine" DEVRITES_HOST_ARTIFACT_DIR="$2" HOME="$TMP/home" \
    bash "$S/$1.sh" --target "$TMP/target" > "$TMP/out" 2>&1
}
for s in install update; do
  mkpayload "$TMP/fx"
  run "$s" "$TMP/fx"
  [ ! -f "$TMP/marker" ] && ok "$s: complete payload is not rebuilt" || no "$s: rebuilt a complete payload"
  for h in claude codex omp pi devin; do
    mkpayload "$TMP/fx"; rm -rf "${TMP:?}/fx/$h"
    run "$s" "$TMP/fx"
    [ -f "$TMP/marker" ] && ok "$s: rebuilds payload missing $h/" || no "$s: skipped rebuild for payload missing $h/ ($(tail -1 "$TMP/out"))"
  done
  rm -rf "$TMP/fx"
done
exit "$fail"
