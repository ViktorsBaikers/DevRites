#!/usr/bin/env bash
# Guard the single-sourced published version against squash-merge reverts.
# package.json, README status, CHANGELOG, and package-lock must agree.
# When git tags are available, that version must be >= the highest vX.Y.Z tag.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
cd "$ROOT"

fail=0
ok() { printf 'ok: %s\n' "$*"; }
no() { printf 'FAIL: %s\n' "$*" >&2; fail=1; }

pkg="$(node -e 'const {version} = require("./package.json"); process.stdout.write(version)')"
if [[ ! "$pkg" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  no "package.json version is not a dotted triple: $pkg"
  exit 1
fi
ok "package.json version $pkg"

lock="$(node -e 'const lock = require("./package-lock.json"); process.stdout.write(lock.version + "\n" + (lock.packages[""] && lock.packages[""].version || ""))')"
lock_root="$(printf '%s\n' "$lock" | sed -n '1p')"
lock_pkg="$(printf '%s\n' "$lock" | sed -n '2p')"
[[ "$lock_root" == "$pkg" ]] && ok "package-lock.json version $lock_root" || no "package-lock.json version $lock_root != package.json $pkg"
[[ "$lock_pkg" == "$pkg" ]] && ok "package-lock packages[''] version $lock_pkg" || no "package-lock packages[''] version $lock_pkg != package.json $pkg"

readme_ver="$(sed -n 's/^\*\*Status:\*\* \[`v\([0-9][0-9.]*\)`\].*/\1/p' README.md | head -n1)"
[[ "$readme_ver" == "$pkg" ]] && ok "README status v$readme_ver" || no "README status v${readme_ver:-<missing>} != package.json $pkg"

changelog_ver="$(sed -n 's/^## \[\([0-9][0-9.]*\)\].*/\1/p' CHANGELOG.md | head -n1)"
[[ "$changelog_ver" == "$pkg" ]] && ok "CHANGELOG latest heading $changelog_ver" || no "CHANGELOG latest heading ${changelog_ver:-<missing>} != package.json $pkg"

highest_tag=""
if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  while IFS= read -r tag; do
    [[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]] || continue
    ver="${tag#v}"
    if [[ -z "$highest_tag" ]]; then
      highest_tag="$ver"
      continue
    fi
    if node -e '
      const a = process.argv[1].split(".").map(Number);
      const b = process.argv[2].split(".").map(Number);
      for (let i = 0; i < 3; i++) {
        if (a[i] > b[i]) process.exit(0);
        if (a[i] < b[i]) process.exit(1);
      }
      process.exit(1);
    ' "$ver" "$highest_tag"; then
      highest_tag="$ver"
    fi
  done < <(git tag --list 'v*' 2>/dev/null || true)
fi

if [[ -z "$highest_tag" ]]; then
  echo "skip: no vX.Y.Z git tags in this clone"
else
  ok "highest git tag v$highest_tag"
  if node -e '
    const pkg = process.argv[1].split(".").map(Number);
    const tag = process.argv[2].split(".").map(Number);
    for (let i = 0; i < 3; i++) {
      if (pkg[i] > tag[i]) process.exit(0);
      if (pkg[i] < tag[i]) process.exit(1);
    }
    process.exit(0);
  ' "$pkg" "$highest_tag"; then
    ok "package.json $pkg >= git tag v$highest_tag"
  else
    no "package.json $pkg is behind git tag v$highest_tag (likely a squash-merge revert of the release identity)"
  fi
fi

if [[ "$fail" -ne 0 ]]; then
  echo "published-version-identity-test: FAIL" >&2
  exit 1
fi
echo "published-version-identity-test: PASS"
