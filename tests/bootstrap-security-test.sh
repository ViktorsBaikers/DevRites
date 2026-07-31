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
    regular(f"{root}/install.sh", b"#!/bin/sh\nprintf success > \"$MARKER\"\nexit 23\n")
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

for case in valid wrong-prefix multiple-prefix traversal dot-segment absolute backslash control duplicate symlink special member-count long-path expanded-size; do
  make_archive "$TMP/$case.tar.gz" "$case"
done
printf '{"tag_name":"%s"}\n' "$TAG" > "$TMP/metadata.json"
truncate -s 1048577 "$TMP/oversized-metadata.json"
truncate -s 67108865 "$TMP/oversized-archive.tar.gz"
printf 'not a gzip stream\n' > "$TMP/invalid-gzip.tar.gz"

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
  if grep -Eq 'raw\.githubusercontent|archive/refs/(tags|heads)|/main([./]|$)' "$case_dir/urls"; then
    echo "FAIL: $name attempted unchecked fallback URL" >&2
    exit 1
  fi
}

run_case valid "$TMP/valid.tar.gz" valid pass
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
cp "$MOCKBIN/curl" "$DECOMPRESS_BIN/curl"
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
cp "$MOCKBIN/curl" "$PREFLIGHT_BIN/curl"
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

echo "bootstrap-security-test: PASS"
