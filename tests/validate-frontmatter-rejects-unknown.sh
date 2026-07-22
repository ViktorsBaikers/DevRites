#!/usr/bin/env bash
# validate-frontmatter-rejects-unknown.sh: assert validate-frontmatter.py
# fails (non-zero) on any non-canonical SKILL.md field, on a description > 1024
# chars, and on a multi-line description.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
PY="$ROOT/scripts/validate-frontmatter.py"

if ! command -v python3 >/dev/null 2>&1; then
  echo "skip: python3 not found"
  exit 0
fi

TMP="$(mktemp -d 2>/dev/null || echo "${TMPDIR:-/tmp}/devrites-fm-test.$$")"
mkdir -p "$TMP"
fail=0

assert_fail() {
  _label="$1"; _file="$2"
  if python3 "$PY" "$_file" >"$TMP/out" 2>&1; then
    printf 'FAIL: %s: validator should have failed but exited 0\n' "$_label"
    cat "$TMP/out"
    fail=1
  else
    printf 'ok: %s: validator rejected as expected\n' "$_label"
  fi
}

assert_ok() {
  _label="$1"; _file="$2"
  if python3 "$PY" "$_file" >"$TMP/out" 2>&1; then
    printf 'ok: %s: validator accepted canonical skill\n' "$_label"
  else
    printf 'FAIL: %s: validator rejected canonical skill\n' "$_label"
    cat "$TMP/out"
    fail=1
  fi
}

# 1) unknown field should be rejected
cat > "$TMP/unknown-field.md" <<'EOF'
---
name: bogus
description: A bogus skill that has a non-canonical field.
made-up-field: yes
---
body
EOF
assert_fail "unknown-field" "$TMP/unknown-field.md"

# 2) allowed-tools is no longer canonical and must be rejected
cat > "$TMP/allowed-tools.md" <<'EOF'
---
name: bogus
description: A bogus skill that still has allowed-tools.
allowed-tools: Read Grep
---
body
EOF
assert_fail "allowed-tools" "$TMP/allowed-tools.md"

# 3) description > 1024 chars should be rejected
LONG="$(python3 -c 'print("x" * 1100)')"
cat > "$TMP/too-long.md" <<EOF
---
name: bogus
description: $LONG
---
body
EOF
assert_fail "description-too-long" "$TMP/too-long.md"

# 4) multi-line description should be rejected
cat > "$TMP/multiline.md" <<'EOF'
---
name: bogus
description: |
  line one of the description
  line two of the description
---
body
EOF
assert_fail "multiline-description" "$TMP/multiline.md"

# 5) a canonical skill should pass
cat > "$TMP/ok.md" <<'EOF'
---
name: ok-skill
description: A canonical skill with only name and description.
user-invocable: true
---
body
EOF
assert_ok "canonical-skill" "$TMP/ok.md"

rm -rf "$TMP"

if [ "$fail" -ne 0 ]; then
  echo "VALIDATE-FRONTMATTER NEGATIVE TESTS: FAIL"
  exit 1
fi
echo "VALIDATE-FRONTMATTER NEGATIVE TESTS: PASS"
exit 0
