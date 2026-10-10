#!/usr/bin/env bash
# The curl-bootstrap acquisition path in scripts/install-lib.sh must refuse an oversize payload,
# an HTTP failure, a multi-record sidecar and a digest mismatch. curl is mocked; no network.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
LIB="${INSTALL_LIB_SOURCE:-$ROOT/scripts/install-lib.sh}"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
MOCKBIN="$TMP/mockbin"; mkdir -p "$MOCKBIN"
cat > "$MOCKBIN/curl" <<'MOCK'
#!/bin/sh
for url do :; done
case "$url" in
  *.sha256) cat "$MOCK_SIDECAR" ;;
  *) [ -z "${MOCK_HTTP_FAIL:-}" ] || exit 22; cat "$MOCK_ASSET" ;;
esac
MOCK
printf '#!/bin/sh\nexit 1\n' > "$MOCKBIN/gh"
chmod +x "$MOCKBIN/curl" "$MOCKBIN/gh"
export PATH="$MOCKBIN:$PATH" DEVRITES_REF=v1.2.3
SRC="$TMP/src"; mkdir -p "$SRC"; printf '{"version":"1.2.3"}\n' > "$SRC/package.json"
# shellcheck source=/dev/null
source "$LIB"
asset="devrites-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')"
printf 'payload\n' > "$TMP/asset"
good="$(shasum -a 256 "$TMP/asset" | awk '{print $1}')"
bad="$(printf '%064d' 0)"

refuse() { # label, expected failure substring
  out="$TMP/engine"; rm -f "$out" "$out.sha256"; DR_ACQUIRE_FAILURE=""
  if dr_download_engine "$SRC" owner/repo "$out"; then echo "FAIL $1: acquisition succeeded" >&2; exit 1; fi
  case "$DR_ACQUIRE_FAILURE" in *"$2"*) ;; *) echo "FAIL $1: wanted '$2', got '$DR_ACQUIRE_FAILURE'" >&2; exit 1 ;; esac
  [ ! -e "$out" ] || { echo "FAIL $1: artifact left on disk" >&2; exit 1; }
}

export MOCK_ASSET="$TMP/asset" MOCK_SIDECAR="$TMP/sidecar"

printf '%s  %s\n' "$good" "$asset" > "$MOCK_SIDECAR"
MOCK_HTTP_FAIL=1 refuse http "HTTP status"

# one byte over the 64 MiB limit
head -c 67108865 /dev/zero > "$TMP/big"
MOCK_ASSET="$TMP/big" refuse oversize "size limit"

# a second record is refused even though the first matches
printf '%s  %s\n%s  other\n' "$good" "$asset" "$good" > "$MOCK_SIDECAR"
refuse multirecord "checksum failed"

printf '%s  %s\n' "$bad" "$asset" > "$MOCK_SIDECAR"
refuse mismatch "checksum failed"

echo 'install-lib acquisition refusals OK'
