#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
TAG="v1.2.3"
ASSET="devrites-$TAG.tar.gz"

command -v python3 >/dev/null 2>&1 || { echo "bootstrap-security-test: python3 is required for adversarial fixtures" >&2; exit 1; }

make_archive() {
  python3 - "$1" "$2" "$TAG" <<'PY'
import io
import sys
import tarfile

out, case, tag = sys.argv[1:]
prefix = f"devrites-{tag}"

class Zeros(io.RawIOBase):
    def __init__(self, remaining): self.remaining = remaining
    def readable(self): return True
    def readinto(self, buffer):
        if not self.remaining: return 0
        count = min(len(buffer), self.remaining)
        buffer[:count] = b"\0" * count
        self.remaining -= count
        return count

with tarfile.open(out, "w:gz", compresslevel=1) as archive:
    def directory(name):
        entry = tarfile.TarInfo(name)
        entry.type = tarfile.DIRTYPE
        entry.mode = 0o700
        archive.addfile(entry)
    def regular(name, body=b""):
        entry = tarfile.TarInfo(name)
        entry.size = len(body)
        entry.mode = 0o700
        archive.addfile(entry, io.BytesIO(body))

    root = "wrong-prefix" if case == "wrong-prefix" else prefix
    directory(root)
    # Every bundle except the origin-substituted one writes "success"; the
    # attacker bundle writes "attacker" so a refusal and an execution cannot be
    # confused for each other by the marker.
    marker = "attacker" if case == "attacker" else "success"
    regular(f"{root}/install.sh", f'#!/bin/sh\nprintf {marker} > "$MARKER"\nexit 23\n'.encode())
    if case == "multiple-prefix": directory("devrites-v9.9.9")
    elif case == "traversal": regular(f"{prefix}/../escape", b"bad")
    elif case == "dot-segment": regular(f"{prefix}/./escape", b"bad")
    elif case == "absolute": regular("/absolute", b"bad")
    elif case == "backslash": regular(f"{prefix}\\escape", b"bad")
    elif case == "control": regular(f"{prefix}/bad\tname", b"bad")
    elif case == "duplicate":
        regular(f"{prefix}/same", b"one")
        regular(f"{prefix}/same", b"two")
    elif case == "symlink":
        entry = tarfile.TarInfo(f"{prefix}/link")
        entry.type = tarfile.SYMTYPE
        entry.linkname = "install.sh"
        archive.addfile(entry)
    elif case == "special":
        entry = tarfile.TarInfo(f"{prefix}/pipe")
        entry.type = tarfile.FIFOTYPE
        archive.addfile(entry)
    elif case == "member-count":
        for number in range(10000): directory(f"{prefix}/d{number}")
    elif case == "long-path": regular(f"{prefix}/" + "x" * 4100, b"bad")
    elif case == "expanded-size":
        entry = tarfile.TarInfo(f"{prefix}/large")
        entry.size = 256 * 1024 * 1024 + 1
        archive.addfile(entry, io.BufferedReader(Zeros(entry.size), 1024 * 1024))
PY
}

sha256() {
  if command -v shasum >/dev/null 2>&1; then shasum -a 256 "$1" | awk '{print $1}'; else sha256sum "$1" | awk '{print $1}'; fi
}

for case in valid attacker wrong-prefix multiple-prefix traversal dot-segment absolute backslash control duplicate symlink special member-count long-path expanded-size; do
  make_archive "$TMP/$case.tar.gz" "$case"
done
printf '{"tag_name":"%s"}\n' "$TAG" > "$TMP/metadata.json"
truncate -s 1048577 "$TMP/oversized-metadata.json"
truncate -s 67108865 "$TMP/oversized-archive.tar.gz"
printf 'not a gzip stream\n' > "$TMP/invalid-gzip.tar.gz"

# The release job attests every bundle it publishes, so every fixture here is
# attested -- except the origin-substituted one, which the release job never
# signed and therefore has no attestation to verify. This list stands in for the
# signed provenance the release job publishes about its own artifacts. It is
# written here and read only by the verifier stand-in: the mock origin below
# never serves it and install.sh has no way to reach it.
ATTESTED_DIGESTS="$TMP/attested-digests.txt"
: > "$ATTESTED_DIGESTS"
for archive in "$TMP"/*.tar.gz; do
  [ "$archive" = "$TMP/attacker.tar.gz" ] && continue
  sha256 "$archive" >> "$ATTESTED_DIGESTS"
done

# The only repository the mock origin serves; any other repository gets a 22.
export MOCK_REPO=owner/repo

MOCKBIN="$TMP/mockbin"
mkdir "$MOCKBIN"
cat > "$MOCKBIN/curl" <<'EOF'
#!/bin/sh
out=""; url=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) out="$2"; shift 2 ;;
    --connect-timeout|--max-time|--max-filesize|--proto|--proto-redir) shift 2 ;;
    --tlsv1.2|-fL) shift ;;
    *) url="$1"; shift ;;
  esac
done
printf '%s\n' "$url" >> "$MOCK_LOG"
case "$url" in
  "https://api.github.com/repos/$MOCK_REPO/"*|"https://github.com/$MOCK_REPO/"*) ;;
  *) if [ -n "$out" ]; then printf partial > "$out"; else printf partial; fi; exit 22 ;;
esac
case "$url" in
  https://api.github.com/*) source="$MOCK_METADATA" ;;
  *.sha256) source="$MOCK_SIDECAR" ;;
  *) source="$MOCK_ARCHIVE" ;;
esac
[ -f "$source" ] || {
  if [ -n "$out" ]; then printf partial > "$out"; else printf partial; fi
  exit 22
}
if [ -n "$out" ]; then cp "$source" "$out"; else cat "$source"; fi
EOF
chmod +x "$MOCKBIN/curl"

# Stand-in for `gh attestation verify`.
#
# The real command checks three things: the Sigstore signature over the fetched
# provenance bundle, the workflow identity in the signing certificate, and the
# bundle's subject digest against the file on disk. Only the last is
# reproducible here, so this stand-in enforces the identity arguments install.sh
# passes and then the subject digest, against the digest list above. An artifact
# the release job never signed has no attestation to verify whatever the origin
# serves beside it. It exits 64 rather than guessing if install.sh ever calls it
# with a shape this stand-in does not understand, so a rewritten invocation
# cannot pass by accident.
cat > "$MOCKBIN/gh" <<'EOF'
#!/bin/sh
[ "${1:-}" = attestation ] && [ "${2:-}" = verify ] || {
  printf 'unexpected gh invocation: %s\n' "$*" >&2
  exit 64
}
shift 2
file=""; repo=""; signer=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --repo|-R) repo="$2"; shift 2 ;;
    --signer-workflow) signer="$2"; shift 2 ;;
    --owner|-o|--signer-repo|--signer-digest|--predicate-type|--cert-identity|-i|--cert-oidc-issuer|--source-digest|--source-ref|--bundle|-b|--format|--jq|--template|--limit|-L|--hostname|--digest-alg|-d)
      shift 2 ;;
    --*) shift ;;
    *) file="$1"; shift ;;
  esac
done
[ "$repo" = "$MOCK_EXPECTED_REPO" ] || {
  printf 'gh: provenance lookup is not pinned to %s (got %s)\n' "$MOCK_EXPECTED_REPO" "${repo:-none}" >&2
  exit 64
}
[ "$signer" = "$MOCK_EXPECTED_SIGNER" ] || {
  printf 'gh: signer workflow is not pinned to %s (got %s)\n' "$MOCK_EXPECTED_SIGNER" "${signer:-none}" >&2
  exit 64
}
[ -n "$file" ] && [ -f "$file" ] || { printf 'gh: no artifact to verify\n' >&2; exit 64; }
if command -v shasum >/dev/null 2>&1; then
  digest="$(shasum -a 256 "$file" | awk '{print $1}')"
else
  digest="$(sha256sum "$file" | awk '{print $1}')"
fi
grep -qxF "$digest" "$MOCK_ATTESTED_DIGESTS" 2>/dev/null || {
  printf 'no matching attestation found for %s\n' "$file" >&2
  exit 1
}
EOF
chmod +x "$MOCKBIN/gh"

# The stand-in must itself be strict: it accepts only the pinned repository,
# signer workflow and attested digest.
standin_gh() {
  MOCK_ATTESTED_DIGESTS="$ATTESTED_DIGESTS" MOCK_EXPECTED_REPO=owner/repo MOCK_EXPECTED_SIGNER=owner/repo/.github/workflows/ci.yml@refs/heads/main \
    "$MOCKBIN/gh" attestation verify "$@" >/dev/null 2>&1
}
SIGNER=owner/repo/.github/workflows/ci.yml@refs/heads/main
standin_gh "$TMP/valid.tar.gz" --repo owner/repo --signer-workflow "$SIGNER" || { echo "FAIL gh stand-in rejected the pinned request" >&2; exit 1; }
standin_gh "$TMP/valid.tar.gz" --repo other/repo --signer-workflow "$SIGNER" && { echo "FAIL gh stand-in accepted a wrong repo" >&2; exit 1; }
standin_gh "$TMP/valid.tar.gz" --repo owner/repo --signer-workflow owner/repo/.github/workflows/other.yml@refs/heads/main && { echo "FAIL gh stand-in accepted a wrong signer workflow" >&2; exit 1; }
standin_gh "$TMP/attacker.tar.gz" --repo owner/repo --signer-workflow "$SIGNER" && { echo "FAIL gh stand-in accepted an unattested digest" >&2; exit 1; }

run_case() {
  name="$1"; archive="$2"; sidecar_mode="$3"; expected="$4"; use_metadata="${5:-0}"; expected_signal="${6:-}"
  case_dir="$TMP/case-$name"
  runtime="$case_dir/runtime"
  mkdir -p "$case_dir/source" "$runtime"
  cp "$ROOT/install.sh" "$case_dir/source/install.sh"
  sidecar="$case_dir/sidecar"
  case "$sidecar_mode" in
    valid) printf '%s  %s\n' "$(sha256 "$archive")" "$ASSET" > "$sidecar" ;;
    missing) sidecar="$case_dir/missing" ;;
    malformed) printf 'not-a-checksum  %s\n' "$ASSET" > "$sidecar" ;;
    mismatch) printf '%064d  %s\n' 0 "$ASSET" > "$sidecar" ;;
    wrong-name) printf '%s  wrong.tar.gz\n' "$(sha256 "$archive")" > "$sidecar" ;;
    oversized) truncate -s 4097 "$sidecar" ;;
    multiple) printf '%s  %s\n%s  %s\n' "$(sha256 "$archive")" "$ASSET" "$(sha256 "$archive")" "$ASSET" > "$sidecar" ;;
  esac
  : > "$case_dir/urls"
  marker="$case_dir/executed"
  ref="$TAG"
  metadata="$TMP/metadata.json"
  [ "$use_metadata" = 0 ] || ref=""
  [ "$name" != oversized-metadata ] || metadata="$TMP/oversized-metadata.json"
  set +e
  MARKER="$marker" MOCK_LOG="$case_dir/urls" MOCK_ARCHIVE="$archive" MOCK_SIDECAR="$sidecar" MOCK_METADATA="$metadata" \
    MOCK_ATTESTED_DIGESTS="$ATTESTED_DIGESTS" MOCK_EXPECTED_REPO=owner/repo MOCK_EXPECTED_SIGNER=owner/repo/.github/workflows/ci.yml@refs/heads/main \
    TMPDIR="$runtime" PATH="$MOCKBIN:$PATH" DEVRITES_REF="$ref" DEVRITES_REPO="owner/repo" \
    bash "$case_dir/source/install.sh" >"$case_dir/output" 2>&1
  status=$?
  set -e
  if [ "$expected" = pass ]; then
    [ "$status" -eq 23 ] && [ "$(cat "$marker" 2>/dev/null)" = success ] || { echo "FAIL: $name did not execute verified bundle" >&2; exit 1; }
  else
    [ "$status" -ne 0 ] && [ ! -e "$marker" ] || { echo "FAIL: $name was not rejected before execution" >&2; exit 1; }
  fi
  if [ -n "$expected_signal" ] && ! grep -Fq "$expected_signal" "$case_dir/output"; then
    echo "FAIL: $name did not report $expected_signal" >&2
    cat "$case_dir/output" >&2
    exit 1
  fi
  if find "$runtime" -mindepth 1 -print -quit | grep -q .; then
    echo "FAIL: $name leaked bootstrap files" >&2
    exit 1
  fi
  if grep -Ev '^https://(api\.github\.com/repos|github\.com)/owner/repo/' "$case_dir/urls" | grep -q .; then
    echo "FAIL: $name requested a URL outside owner/repo" >&2
    cat "$case_dir/urls" >&2
    exit 1
  fi
  download_url="https://github.com/owner/repo/releases/download/$TAG/$ASSET"
  if [ "$use_metadata" = 1 ]; then
    grep -Fxq "https://api.github.com/repos/owner/repo/releases/latest" "$case_dir/urls" || {
      echo "FAIL: $name did not request latest-release metadata from owner/repo" >&2
      exit 1
    }
  fi
  if [ "$expected" = pass ]; then
    grep -Fxq "$download_url" "$case_dir/urls" && grep -Fxq "$download_url.sha256" "$case_dir/urls" || {
      echo "FAIL: $name did not fetch the owner/repo release tarball and checksum" >&2
      cat "$case_dir/urls" >&2
      exit 1
    }
  fi
  if grep -Eq 'raw\.githubusercontent|archive/refs/(tags|heads)|/main([./]|$)' "$case_dir/urls"; then
    echo "FAIL: $name attempted unchecked fallback URL" >&2
    exit 1
  fi
}

run_case valid "$TMP/valid.tar.gz" valid pass
run_case valid-latest "$TMP/valid.tar.gz" valid pass 1
run_case checksum-missing "$TMP/valid.tar.gz" missing fail
run_case checksum-malformed "$TMP/valid.tar.gz" malformed fail
run_case checksum-mismatch "$TMP/valid.tar.gz" mismatch fail 0 "release $TAG asset $ASSET: checksum failed"
run_case checksum-wrong-name "$TMP/valid.tar.gz" wrong-name fail
run_case checksum-oversized "$TMP/valid.tar.gz" oversized fail
run_case checksum-multiple "$TMP/valid.tar.gz" multiple fail
run_case oversized-metadata "$TMP/valid.tar.gz" valid fail 1
run_case oversized-archive "$TMP/oversized-archive.tar.gz" valid fail
run_case invalid-gzip "$TMP/invalid-gzip.tar.gz" valid fail 0 "gzip decompression failed"
run_case wrong-prefix "$TMP/wrong-prefix.tar.gz" valid fail
run_case multiple-prefix "$TMP/multiple-prefix.tar.gz" valid fail
run_case traversal "$TMP/traversal.tar.gz" valid fail
run_case dot-segment "$TMP/dot-segment.tar.gz" valid fail
run_case absolute "$TMP/absolute.tar.gz" valid fail
run_case backslash "$TMP/backslash.tar.gz" valid fail
run_case control "$TMP/control.tar.gz" valid fail
run_case duplicate "$TMP/duplicate.tar.gz" valid fail
run_case symlink "$TMP/symlink.tar.gz" valid fail
run_case special "$TMP/special.tar.gz" valid fail
run_case member-count "$TMP/member-count.tar.gz" valid fail
run_case long-path "$TMP/long-path.tar.gz" valid fail
run_case expanded-size "$TMP/expanded-size.tar.gz" valid fail

DECOMPRESS_BIN="$TMP/decompress-bin"
DECOMPRESS_LOG="$TMP/decompress-writes"
DECOMPRESS_MARKER="$TMP/decompress-executed"
DECOMPRESS_RUNTIME="$TMP/decompress-runtime"
DECOMPRESS_SOURCE="$TMP/decompress-source"
mkdir "$DECOMPRESS_BIN" "$DECOMPRESS_RUNTIME" "$DECOMPRESS_SOURCE"
cp "$MOCKBIN/curl" "$MOCKBIN/gh" "$DECOMPRESS_BIN/"
cp "$ROOT/install.sh" "$DECOMPRESS_SOURCE/install.sh"
cat > "$DECOMPRESS_BIN/gzip" <<'EOF'
#!/bin/sh
set -e
i=0
while [ "$i" -lt 128 ]; do
  i=$((i + 1))
  printf '%s\n' "$i" >> "$DECOMPRESS_LOG"
  dd if=/dev/zero bs=4194304 count=1 2>/dev/null
done
EOF
chmod +x "$DECOMPRESS_BIN/gzip"
printf '%s  %s\n' "$(sha256 "$TMP/valid.tar.gz")" "$ASSET" > "$TMP/decompress-sidecar"
set +e
DECOMPRESS_LOG="$DECOMPRESS_LOG" MARKER="$DECOMPRESS_MARKER" MOCK_LOG="$TMP/decompress-urls" MOCK_ARCHIVE="$TMP/valid.tar.gz" \
  MOCK_SIDECAR="$TMP/decompress-sidecar" MOCK_METADATA="$TMP/metadata.json" TMPDIR="$DECOMPRESS_RUNTIME" \
  MOCK_ATTESTED_DIGESTS="$ATTESTED_DIGESTS" MOCK_EXPECTED_REPO=owner/repo MOCK_EXPECTED_SIGNER=owner/repo/.github/workflows/ci.yml@refs/heads/main \
  PATH="$DECOMPRESS_BIN:$PATH" DEVRITES_REF="$TAG" DEVRITES_REPO=owner/repo \
  bash "$DECOMPRESS_SOURCE/install.sh" >"$TMP/decompress-output" 2>&1
decompress_status=$?
set -e
decompress_writes="$(wc -l < "$DECOMPRESS_LOG" | tr -d ' ')"
[[ "$decompress_status" -ne 0 && ! -e "$DECOMPRESS_MARKER" && "$decompress_writes" -ge 80 && "$decompress_writes" -le 84 ]] || {
  echo "FAIL: bounded decompression consumed $decompress_writes producer blocks" >&2
  exit 1
}
grep -Fq 'decompressed archive exceeds 320 MiB limit' "$TMP/decompress-output" || {
  echo "FAIL: oversized decompressed stream did not report its cause" >&2
  cat "$TMP/decompress-output" >&2
  exit 1
}
if find "$DECOMPRESS_RUNTIME" -mindepth 1 -print -quit | grep -q .; then
  echo "FAIL: oversized decompressed stream leaked bootstrap files" >&2
  exit 1
fi

HOSTILE_CWD="$TMP/hostile-cwd"
HOSTILE_MARKER="$TMP/hostile-cwd-executed"
mkdir -p "$HOSTILE_CWD/pack" "$HOSTILE_CWD/scripts"
cat > "$HOSTILE_CWD/scripts/install-lib.sh" <<'EOF'
printf 'executed\n' > "$HOSTILE_MARKER"
exit 91
EOF
for shim in install update uninstall; do
  rm -f "$HOSTILE_MARKER"
  set +e
  (
    cd "$HOSTILE_CWD"
    cat "$ROOT/$shim.sh" | HOSTILE_MARKER="$HOSTILE_MARKER" DEVRITES_REF=main bash
  ) >/dev/null 2>&1
  hostile_status=$?
  set -e
  [[ "$hostile_status" -ne 0 && ! -e "$HOSTILE_MARKER" ]] || {
    echo "FAIL: piped $shim shim trusted a hostile current-directory bundle" >&2
    exit 1
  }
done

CANARY_BIN="$TMP/canary-bin"
CANARY_LOG="$TMP/canary-writes"
mkdir "$CANARY_BIN" "$TMP/canary-source" "$TMP/canary-runtime"
cp "$ROOT/install.sh" "$TMP/canary-source/install.sh"
cat > "$CANARY_BIN/curl" <<'EOF'
#!/bin/sh
set -e
out=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) out="$2"; shift 2 ;;
    --connect-timeout|--max-time|--max-filesize|--proto|--proto-redir) shift 2 ;;
    --tlsv1.2|-fL) shift ;;
    *) shift ;;
  esac
done
if [ -n "$out" ]; then exec 3>"$out"; else exec 3>&1; fi
i=0
while [ "$i" -lt 64 ]; do
  i=$((i + 1))
  printf '%s\n' "$i" >> "$CANARY_LOG"
  dd if=/dev/zero bs=65536 count=1 >&3 2>/dev/null
done
EOF
chmod +x "$CANARY_BIN/curl"
set +e
CANARY_LOG="$CANARY_LOG" TMPDIR="$TMP/canary-runtime" PATH="$CANARY_BIN:$PATH" \
  DEVRITES_REPO=owner/repo bash "$TMP/canary-source/install.sh" >/dev/null 2>&1
canary_status=$?
set -e
canary_writes="$(wc -l < "$CANARY_LOG" | tr -d ' ')"
[[ "$canary_status" -ne 0 && "$canary_writes" -le 20 ]] || {
  echo "FAIL: bounded download consumed $canary_writes producer blocks" >&2
  exit 1
}

PREFLIGHT_BIN="$TMP/preflight-bin"
PREFLIGHT_LOG="$TMP/preflight-writes"
mkdir "$PREFLIGHT_BIN" "$TMP/preflight-source" "$TMP/preflight-runtime"
cp "$MOCKBIN/curl" "$MOCKBIN/gh" "$PREFLIGHT_BIN/"
cp "$ROOT/install.sh" "$TMP/preflight-source/install.sh"
cat > "$PREFLIGHT_BIN/tar" <<'EOF'
#!/bin/sh
set -e
case "$1" in
  -tf)
    printf 'devrites-v1.2.3/\ndevrites-v1.2.3/install.sh\n'
    ;;
  -tvf)
    i=0
    while [ "$i" -lt 128 ]; do
      i=$((i + 1))
      printf '%s\n' "$i" >> "$PREFLIGHT_LOG"
      if [ "$i" -eq 1 ]; then
        printf 'lrwxr-xr-x  0 owner group 0 Jan 1 00:00 devrites-v1.2.3/link\n'
      else
        printf '%s' '-rw-r--r--  0 owner group 1 Jan 1 00:00 devrites-v1.2.3/file-'
        dd if=/dev/zero bs=65536 count=1 2>/dev/null | tr '\000' x
        printf '\n'
      fi
    done
    ;;
  *) exit 99 ;;
esac
EOF
chmod +x "$PREFLIGHT_BIN/tar"
printf '%s  %s\n' "$(sha256 "$TMP/valid.tar.gz")" "$ASSET" > "$TMP/preflight-sidecar"
set +e
PREFLIGHT_LOG="$PREFLIGHT_LOG" MOCK_LOG="$TMP/preflight-urls" MOCK_ARCHIVE="$TMP/valid.tar.gz" \
  MOCK_SIDECAR="$TMP/preflight-sidecar" MOCK_METADATA="$TMP/metadata.json" TMPDIR="$TMP/preflight-runtime" \
  MOCK_ATTESTED_DIGESTS="$ATTESTED_DIGESTS" MOCK_EXPECTED_REPO=owner/repo MOCK_EXPECTED_SIGNER=owner/repo/.github/workflows/ci.yml@refs/heads/main \
  PATH="$PREFLIGHT_BIN:$PATH" DEVRITES_REF="$TAG" DEVRITES_REPO=owner/repo \
  bash "$TMP/preflight-source/install.sh" >/dev/null 2>&1
preflight_status=$?
set -e
preflight_writes="$(wc -l < "$PREFLIGHT_LOG" | tr -d ' ')"
[[ "$preflight_status" -ne 0 && "$preflight_writes" -le 10 ]] || {
  echo "FAIL: archive metadata preflight consumed $preflight_writes producer records" >&2
  exit 1
}

mkdir "$TMP/mktemp-bin"
cp "$MOCKBIN/curl" "$TMP/mktemp-bin/curl"
printf '#!/bin/sh\nexit 1\n' > "$TMP/mktemp-bin/mktemp"
chmod +x "$TMP/mktemp-bin/mktemp"
mkdir "$TMP/mktemp-source" "$TMP/mktemp-runtime"
cp "$ROOT/install.sh" "$TMP/mktemp-source/install.sh"
if MOCK_LOG="$TMP/mktemp-urls" MOCK_ARCHIVE="$TMP/valid.tar.gz" MOCK_SIDECAR="$TMP/no-sidecar" MOCK_METADATA="$TMP/metadata.json" \
  TMPDIR="$TMP/mktemp-runtime" PATH="$TMP/mktemp-bin:$PATH" DEVRITES_REF="$TAG" bash "$TMP/mktemp-source/install.sh" >/dev/null 2>&1; then
  echo "FAIL: mktemp failure was accepted" >&2
  exit 1
fi
[ ! -s "$TMP/mktemp-urls" ] || { echo "FAIL: mktemp failure still reached the network" >&2; exit 1; }

for boundary in invalid-tag invalid-repo; do
  mkdir "$TMP/$boundary-source" "$TMP/$boundary-runtime"
  cp "$ROOT/install.sh" "$TMP/$boundary-source/install.sh"
  : > "$TMP/$boundary-urls"
  ref="$TAG"; repo="owner/repo"
  [ "$boundary" != invalid-tag ] || ref="main"
  [ "$boundary" != invalid-repo ] || repo="owner/repo/extra"
  if MOCK_LOG="$TMP/$boundary-urls" MOCK_ARCHIVE="$TMP/valid.tar.gz" MOCK_SIDECAR="$TMP/no-sidecar" MOCK_METADATA="$TMP/metadata.json" \
    TMPDIR="$TMP/$boundary-runtime" PATH="$MOCKBIN:$PATH" DEVRITES_REF="$ref" DEVRITES_REPO="$repo" \
    bash "$TMP/$boundary-source/install.sh" >/dev/null 2>&1; then
    echo "FAIL: $boundary was accepted" >&2
    exit 1
  fi
  [ ! -s "$TMP/$boundary-urls" ] || { echo "FAIL: $boundary reached the network" >&2; exit 1; }
done

if grep -Eq 'raw\.githubusercontent|archive/refs/(tags|heads)|devrites-(bootstrap|install)\.\$\$' "$ROOT/install.sh" "$ROOT/update.sh" "$ROOT/uninstall.sh"; then
  echo "FAIL: unchecked or predictable fallback remains" >&2
  exit 1
fi

# Origin substitution.
#
# The release checksum sidecar is fetched from the same origin as the tarball it
# checks, so an attacker who substitutes the origin supplies both the artifact
# and a digest that matches it. A digest cannot distinguish that from a genuine
# release. Only a proof of origin that does not travel with the artifact can,
# and the release job already mints one: a build-provenance attestation bound to
# the artifact digest.
#
# Two arms, and the order is load-bearing. ARM 1 runs first and must reach
# execution on a trusted bundle; if it does not, this harness never got far
# enough to observe anything and ARM 2 would pass for the wrong reason.
#
# Both arms run through the same mock origin and the same release URL, and their
# request logs are compared below, so the only difference between them is the
# bytes the origin serves. The attacker sidecar digest is computed over the
# attacker tarball, so it is internally consistent and the sidecar check passes.
echo "== ARM 1 positive control: trusted origin, attested bundle =="
run_case attested-trusted "$TMP/valid.tar.gz" valid pass
echo "ARM1 OK: harness reaches execution on an attested bundle (marker=success, exit 23)"

echo
echo "== ARM 2 negative control: origin substituted, digest computed over the attacker tarball =="
run_case attested-attacker "$TMP/attacker.tar.gz" valid fail 0 "attestation verification failed"
echo "ARM2 OK: install refused the origin-substituted bundle before execution"

cmp -s "$TMP/case-attested-trusted/urls" "$TMP/case-attested-attacker/urls" || {
  echo "FAIL: the two arms did not request the same URLs, so the refusal is not attributable to the origin substitution" >&2
  diff -u "$TMP/case-attested-trusted/urls" "$TMP/case-attested-attacker/urls" >&2 || true
  exit 1
}
if grep -Fq 'attacker' "$TMP/case-attested-attacker/output"; then
  echo "FAIL: attacker bundle reported itself as executed" >&2
  cat "$TMP/case-attested-attacker/output" >&2
  exit 1
fi

# The check must fail closed, not degrade: an install with no verifier available
# must refuse rather than fall back to the sidecar the origin controls.
echo
echo "== fail-closed: no attestation verifier available =="
NOGH_BIN="$TMP/nogh-bin"
NOGH_SOURCE="$TMP/nogh-source"
NOGH_RUNTIME="$TMP/nogh-runtime"
NOGH_MARKER="$TMP/nogh-executed"
mkdir -p "$NOGH_BIN" "$NOGH_SOURCE" "$NOGH_RUNTIME"
cp "$MOCKBIN/curl" "$NOGH_BIN/curl"
cp "$ROOT/install.sh" "$NOGH_SOURCE/install.sh"
printf '%s  %s\n' "$(sha256 "$TMP/valid.tar.gz")" "$ASSET" > "$TMP/nogh-sidecar"
set +e
MARKER="$NOGH_MARKER" MOCK_LOG="$TMP/nogh-urls" MOCK_ARCHIVE="$TMP/valid.tar.gz" MOCK_SIDECAR="$TMP/nogh-sidecar" \
  MOCK_METADATA="$TMP/metadata.json" TMPDIR="$NOGH_RUNTIME" PATH="$NOGH_BIN:/usr/bin:/bin" DEVRITES_REF="$TAG" \
  DEVRITES_REPO=owner/repo bash "$NOGH_SOURCE/install.sh" >"$TMP/nogh-output" 2>&1
nogh_status=$?
set -e
[[ "$nogh_status" -ne 0 && ! -e "$NOGH_MARKER" ]] || {
  echo "FAIL: an attested bundle was installed without a provenance verifier" >&2
  cat "$TMP/nogh-output" >&2
  exit 1
}
grep -Fq 'attestation verification failed' "$TMP/nogh-output" || {
  echo "FAIL: a missing provenance verifier was not reported as the cause" >&2
  cat "$TMP/nogh-output" >&2
  exit 1
}
echo "FAIL-CLOSED OK: no verifier means refuse, not install"

# A verifier that never returns must not hang the bootstrap: the attestation
# lookup is bounded and a timeout is a refusal with its own message. perl
# supplies the outer alarm because macOS has no timeout(1).
echo
echo "== fail-closed: attestation verifier hangs =="
HANG_BIN="$TMP/hang-bin"
HANG_SOURCE="$TMP/hang-source"
HANG_RUNTIME="$TMP/hang-runtime"
HANG_MARKER="$TMP/hang-executed"
mkdir -p "$HANG_BIN" "$HANG_SOURCE" "$HANG_RUNTIME"
cp "$MOCKBIN/curl" "$HANG_BIN/curl"
printf '#!/bin/sh\nexec sleep 3600\n' > "$HANG_BIN/gh"
chmod +x "$HANG_BIN/gh"
cp "$ROOT/install.sh" "$HANG_SOURCE/install.sh"
printf '%s  %s\n' "$(sha256 "$TMP/valid.tar.gz")" "$ASSET" > "$TMP/hang-sidecar"
set +e
MARKER="$HANG_MARKER" MOCK_LOG="$TMP/hang-urls" MOCK_ARCHIVE="$TMP/valid.tar.gz" MOCK_SIDECAR="$TMP/hang-sidecar" \
  MOCK_METADATA="$TMP/metadata.json" TMPDIR="$HANG_RUNTIME" PATH="$HANG_BIN:$PATH" DEVRITES_REF="$TAG" \
  DEVRITES_REPO=owner/repo DEVRITES_ATTEST_TIMEOUT=2 \
  perl -e 'alarm 20; exec @ARGV' bash "$HANG_SOURCE/install.sh" >"$TMP/hang-output" 2>&1
hang_status=$?
set -e
[[ "$hang_status" -eq 1 && ! -e "$HANG_MARKER" ]] || {
  echo "FAIL: a hung attestation verifier was not refused promptly (exit $hang_status)" >&2
  cat "$TMP/hang-output" >&2
  exit 1
}
grep -Fq 'attestation verification timed out' "$TMP/hang-output" || {
  echo "FAIL: a hung attestation verifier was not reported as a timeout" >&2
  cat "$TMP/hang-output" >&2
  exit 1
}
echo "HANG OK: a verifier that never returns means refuse, not wait"

# Runs the whole bootstrap against a stub gh read from stdin, under an outer
# alarm, with the output piped so a leaked child holding the pipe would show up.
# Sets ATT_DIR, ATT_STATUS and ATT_SECS. Every stub sleeps for a bounded time so
# a regression cannot leave a process running forever.
attest_run() {
  name="$1"; limit="$2"
  ATT_DIR="$TMP/att-$name"
  mkdir -p "$ATT_DIR/bin" "$ATT_DIR/src" "$ATT_DIR/rt"
  cat > "$ATT_DIR/bin/gh"
  chmod +x "$ATT_DIR/bin/gh"
  cp "$MOCKBIN/curl" "$ATT_DIR/bin/curl"
  cp "$ROOT/install.sh" "$ATT_DIR/src/install.sh"
  printf '%s  %s\n' "$(sha256 "$TMP/valid.tar.gz")" "$ASSET" > "$ATT_DIR/sidecar"
  set +e
  att_start=$SECONDS
  MARKER="$ATT_DIR/executed" MOCK_LOG="$ATT_DIR/urls" MOCK_ARCHIVE="$TMP/valid.tar.gz" MOCK_SIDECAR="$ATT_DIR/sidecar" \
    MOCK_METADATA="$TMP/metadata.json" TMPDIR="$ATT_DIR/rt" PATH="$ATT_DIR/bin:$PATH" DEVRITES_REF="$TAG" \
    DEVRITES_REPO=owner/repo DEVRITES_ATTEST_TIMEOUT="$limit" \
    perl -e 'alarm 20; exec @ARGV' bash "$ATT_DIR/src/install.sh" 2>&1 | cat > "$ATT_DIR/output"
  ATT_STATUS="${PIPESTATUS[0]}"
  ATT_SECS=$((SECONDS - att_start))
  set -e
  [[ "$ATT_STATUS" -eq 1 && ! -e "$ATT_DIR/executed" && "$ATT_SECS" -lt 15 ]] || {
    echo "FAIL: $name: expected a prompt refusal (exit 1, nothing executed), got exit $ATT_STATUS after ${ATT_SECS}s" >&2
    cat "$ATT_DIR/output" >&2
    exit 1
  }
}

attest_expect() {
  pattern="$1"; label="$2"
  grep -Fq -- "$pattern" "$ATT_DIR/output" || {
    echo "FAIL: $label" >&2
    cat "$ATT_DIR/output" >&2
    exit 1
  }
}

attest_reject() {
  pattern="$1"; label="$2"
  ! grep -Fq -- "$pattern" "$ATT_DIR/output" || {
    echo "FAIL: $label" >&2
    cat "$ATT_DIR/output" >&2
    exit 1
  }
}

echo
echo "== fail-closed: verifier ignores SIGTERM =="
attest_run ignterm 2 <<'EOF'
#!/bin/sh
trap '' TERM
i=0
while [ "$i" -lt 25 ]; do sleep 1; i=$((i + 1)); done
EOF
attest_expect 'attestation verification timed out after 2s' "a verifier that ignores SIGTERM was not stopped and reported as a timeout"
echo "SIGKILL OK: a verifier that ignores SIGTERM is killed and refused"

echo
echo "== fail-closed: verifier forks a long-lived child and hangs =="
attest_run forkhang 2 <<'EOF'
#!/bin/sh
sleep 12 &
wait
EOF
attest_expect 'attestation verification timed out after 2s' "a hung verifier with a forked child was not reported as a timeout"
[[ "$ATT_SECS" -lt 10 ]] || { echo "FAIL: a forked child held the pipeline open for ${ATT_SECS}s" >&2; exit 1; }
echo "FORK OK: a forked child does not hold the bootstrap open"

echo
echo "== fail-closed: verifier crashes instantly =="
attest_run segv 30 <<'EOF'
#!/bin/sh
kill -SEGV $$
EOF
attest_expect 'attestation verification failed' "a crashing verifier was not reported as a failure"
attest_expect 'status 139' "a crashing verifier's exit status was not reported"
attest_reject 'timed out' "an instant crash was misreported as a timeout"
attest_run exit255 30 <<'EOF'
#!/bin/sh
exit 255
EOF
attest_expect 'attestation verification failed' "an exit-255 verifier was not reported as a failure"
attest_expect 'status 255' "an exit-255 verifier's status was not reported"
attest_reject 'timed out' "an exit-255 verifier was misreported as a timeout"
echo "CRASH OK: crashes are failures with their exit status, not timeouts"

echo
echo "== fail-closed: auth status hangs =="
attest_run authhang 2 <<'EOF'
#!/bin/sh
[ "$1" = auth ] || exit 1
exec sleep 25
EOF
attest_expect 'timed out' "a hung gh auth status was not reported as a timeout"
attest_reject 'not authenticated' "a hung gh auth status was reported as not authenticated"
echo "AUTH OK: a hung auth status is a timeout, not a login problem"

# A signal during a hung lookup must stop the lookup and its watchdog and leave
# nothing behind: no timeout marker, no stub or watchdog process, a non-zero exit.
# Distinct sleep durations make the stub and the watchdog countable by command line.
echo
echo "== signals during a hung lookup leave nothing behind =="
sig_left() { ps -axo command | grep -Ec "sleep ($1|$2)\$" || true; }
for sig in TERM HUP INT; do
  SIG_DIR="$TMP/sig-$sig"
  SIG_STUB=$((70000 + RANDOM % 9000))
  SIG_LIMIT=$((60000 + RANDOM % 9000))
  mkdir -p "$SIG_DIR/bin" "$SIG_DIR/src" "$SIG_DIR/rt"
  printf '#!/bin/sh\nexec sleep %s\n' "$SIG_STUB" > "$SIG_DIR/bin/gh"
  chmod +x "$SIG_DIR/bin/gh"
  cp "$MOCKBIN/curl" "$SIG_DIR/bin/curl"
  cp "$ROOT/install.sh" "$SIG_DIR/src/install.sh"
  printf '%s  %s\n' "$(sha256 "$TMP/valid.tar.gz")" "$ASSET" > "$SIG_DIR/sidecar"
  MARKER="$SIG_DIR/executed" MOCK_LOG="$SIG_DIR/urls" MOCK_ARCHIVE="$TMP/valid.tar.gz" MOCK_SIDECAR="$SIG_DIR/sidecar" \
    MOCK_METADATA="$TMP/metadata.json" TMPDIR="$SIG_DIR/rt" PATH="$SIG_DIR/bin:$PATH" DEVRITES_REF="$TAG" \
    DEVRITES_REPO=owner/repo DEVRITES_ATTEST_TIMEOUT="$SIG_LIMIT" \
    perl -e '$SIG{$_} = "DEFAULT" for qw(INT HUP TERM); alarm 30; exec @ARGV' bash "$SIG_DIR/src/install.sh" >"$SIG_DIR/output" 2>&1 &
  sig_pid=$!
  for _ in $(seq 100); do
    [[ "$(sig_left "$SIG_STUB" 0)" -gt 0 ]] && break
    sleep 0.1
  done
  [[ "$(sig_left "$SIG_STUB" 0)" -gt 0 ]] || { echo "FAIL: $sig: the stub verifier never started" >&2; kill -9 "$sig_pid" 2>/dev/null || true; exit 1; }
  kill -s "$sig" "$sig_pid"
  set +e
  wait "$sig_pid"
  sig_status=$?
  set -e
  for _ in $(seq 20); do
    [[ "$(sig_left "$SIG_STUB" "$SIG_LIMIT")" -eq 0 ]] && break
    sleep 0.1
  done
  sig_procs="$(sig_left "$SIG_STUB" "$SIG_LIMIT")"
  sig_files="$(ls -A "$SIG_DIR/rt")"
  pkill -f "sleep ($SIG_STUB|$SIG_LIMIT)\$" 2>/dev/null || true
  [[ "$sig_status" -ne 0 && "$sig_status" -ne 142 && "$sig_procs" -eq 0 && -z "$sig_files" && ! -e "$SIG_DIR/executed" ]] || {
    echo "FAIL: $sig during a hung lookup: exit $sig_status, $sig_procs processes left, files left: ${sig_files:-none}" >&2
    cat "$SIG_DIR/output" >&2
    exit 1
  }
done
echo "SIGNAL OK: TERM, HUP and INT stop the lookup and clean up"

# A signal that lands after the command is forked but before its PID is recorded
# must still stop it. A DEBUG trap raises the signal at exactly that point, so the
# window is hit on every run instead of by chance.
echo
echo "== a signal between the fork and the PID capture still stops the command =="
RACE_DIR="$TMP/race"
RACE_STUB=$((70000 + RANDOM % 9000))
mkdir -p "$RACE_DIR/bin" "$RACE_DIR/rt"
printf '#!/bin/sh\nexec sleep %s\n' "$RACE_STUB" > "$RACE_DIR/bin/gh"
chmod +x "$RACE_DIR/bin/gh"
sed -n '/^run_bounded()/,/^}/p;/^stop_bounded()/,/^}/p' "$ROOT/install.sh" > "$RACE_DIR/fns.sh"
cat > "$RACE_DIR/run.sh" <<EOF
. "$RACE_DIR/fns.sh"
RUN_JOB=""; RUN_WATCH=""
BOOTSTRAP_DIR="$RACE_DIR/rt"
set -o functrace
trap 'stop_bounded; exit 1' TERM
race() { [ "\$BASH_COMMAND" != 'job=\$!' ] || kill -TERM \$\$; }
trap race DEBUG
run_bounded 60000 gh attestation verify x
EOF
set +e
PATH="$RACE_DIR/bin:$PATH" perl -e '$SIG{TERM} = "DEFAULT"; alarm 20; exec @ARGV' bash "$RACE_DIR/run.sh" >"$RACE_DIR/output" 2>&1
race_status=$?
set -e
sleep 0.3
race_procs="$(ps -axo command | grep -Ec "sleep $RACE_STUB\$" || true)"
pkill -f "sleep $RACE_STUB\$" 2>/dev/null || true
[[ "$race_status" -eq 1 && "$race_procs" -eq 0 ]] || {
  echo "FAIL: a signal between the fork and the PID capture: exit $race_status, $race_procs stub processes left" >&2
  cat "$RACE_DIR/output" >&2
  exit 1
}
echo "RACE OK: a signal before the PID is recorded still stops the command"

# An unusable DEVRITES_ATTEST_TIMEOUT must never remove the bound: it falls back
# to the default. The function is run with the watchdog replaced by a recorder
# so the limit it would have enforced is observable without waiting for it.
echo
echo "== attestation timeout value is validated =="
extract_verify="$(sed -n '/^verify_attestation()/,/^}/p' "$ROOT/install.sh")"
attest_limit_for() {
  (
    DEVRITES_REPO=owner/repo
    DEVRITES_ATTEST_TIMEOUT="$1"
    [ "$1" = UNSET ] && unset DEVRITES_ATTEST_TIMEOUT
    gh() { :; }
    run_bounded() { printf '%s\n' "$1"; return 0; }
    eval "$extract_verify"
    verify_attestation /dev/null 2>/dev/null | head -n 1
  )
}
for bad in abc 0 -1 2s 1.5 00 007 123456 "1 2"; do
  got="$(attest_limit_for "$bad")"
  [[ "$got" == 120 ]] || { echo "FAIL: DEVRITES_ATTEST_TIMEOUT='$bad' gave limit '$got', expected the default 120" >&2; exit 1; }
done
for good in "" UNSET; do
  got="$(attest_limit_for "$good")"
  [[ "$got" == 120 ]] || { echo "FAIL: unset/empty DEVRITES_ATTEST_TIMEOUT gave limit '$got', expected 120" >&2; exit 1; }
done
got="$(attest_limit_for 7)"
[[ "$got" == 7 ]] || { echo "FAIL: DEVRITES_ATTEST_TIMEOUT=7 gave limit '$got'" >&2; exit 1; }
warned="$( (DEVRITES_REPO=owner/repo DEVRITES_ATTEST_TIMEOUT=abc; gh() { :; }; run_bounded() { return 0; }; eval "$extract_verify"; verify_attestation /dev/null 2>&1 >/dev/null) )"
case "$warned" in *DEVRITES_ATTEST_TIMEOUT*120*) ;; *) echo "FAIL: an invalid DEVRITES_ATTEST_TIMEOUT was clamped silently" >&2; exit 1 ;; esac
echo "TIMEOUT VALUE OK: invalid values fall back to 120 with a warning"

echo "bootstrap-security-test: PASS"
