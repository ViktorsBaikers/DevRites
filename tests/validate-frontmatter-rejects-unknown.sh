#!/usr/bin/env bash
# validate-frontmatter-rejects-unknown.sh: assert validate-frontmatter.py
# fails (non-zero) on duplicate or non-canonical fields, on an agent description
# over 45 words, on a description > 1024 chars, and on a multi-line description.
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

# 5) agent descriptions over 45 words should be rejected
mkdir -p "$TMP/agents"
AGENT_WORDS="$(python3 -c 'print("word " * 46)')"
cat > "$TMP/agents/too-many-words.md" <<EOF
---
name: overlong-agent
description: $AGENT_WORDS
---
body
EOF
assert_fail "agent-description-word-budget" "$TMP/agents/too-many-words.md"

# 6) duplicate fields must fail instead of silently taking the last value
cat > "$TMP/duplicate-field.md" <<'EOF'
---
name: duplicate-field
description: First description.
description: Second description.
---
body
EOF
assert_fail "duplicate-field" "$TMP/duplicate-field.md"

# 7) a canonical skill should pass
cat > "$TMP/ok.md" <<'EOF'
---
name: ok-skill
description: A canonical skill with only name and description.
user-invocable: true
---
body
EOF
assert_ok "canonical-skill" "$TMP/ok.md"

# 8) name missing should be rejected
cat > "$TMP/no-name.md" <<'EOF'
---
description: A skill without a name field.
user-invocable: true
---
body
EOF
assert_fail "missing-name" "$TMP/no-name.md"

# 9) reserved name substrings (agentskills.io/Anthropic rules) must be rejected
cat > "$TMP/reserved-name.md" <<'EOF'
---
name: claude-helper
description: A bogus skill whose name uses a reserved substring.
user-invocable: true
---
body
EOF
assert_fail "reserved-name-substring" "$TMP/reserved-name.md"

# 10) non-kebab names must be rejected
cat > "$TMP/Bad_Name.md" <<'EOF'
---
name: Bad_Name
description: A bogus skill whose name breaks kebab-case.
user-invocable: true
---
body
EOF
assert_fail "non-kebab-name" "$TMP/Bad_Name.md"

# 13) missing/empty description must still be rejected
cat > "$TMP/no-desc.md" <<'EOF'
---
name: no-desc
user-invocable: true
---
body
EOF
assert_fail "missing-description" "$TMP/no-desc.md"

# 14) a bare SKILL.md path (no directory component) must not crash the harness
bare="$TMP/bare-check"
mkdir -p "$bare"
(cd "$bare" && printf '%s\n' '---' 'name: bare-skill' 'description: A skill validated from a directory-less path.' 'user-invocable: true' '---' body > SKILL.md && python3 "$PY" SKILL.md >/dev/null 2>&1; code=$?; [ "$code" -ne 2 ] || { printf 'FAIL: bare-path crashed validator\n'; fail=1; }; [ "$code" -eq 0 ] || { printf 'ok: bare-path handled cleanly (exit %s)\n' "$code"; })

# 11) a real <dir>/SKILL.md whose dir does not match its name must be rejected
mkdir -p "$TMP/wrong-dir/other-name"
cat > "$TMP/wrong-dir/other-name/SKILL.md" <<'EOF'
---
name: wrong-dir
description: A canonical-layout skill whose directory disagrees with its name.
user-invocable: true
---
body
EOF
assert_fail "skill-name-dir-mismatch" "$TMP/wrong-dir/other-name/SKILL.md"

# 12) a matching <dir>/SKILL.md layout should pass
mkdir -p "$TMP/matching-name"
cat > "$TMP/matching-name/SKILL.md" <<'EOF'
---
name: matching-name
description: A canonical-layout skill whose directory matches its name.
user-invocable: true
---
body
EOF
assert_ok "skill-name-dir-match" "$TMP/matching-name/SKILL.md"


rm -rf "$TMP"

if [ "$fail" -ne 0 ]; then
  echo "VALIDATE-FRONTMATTER NEGATIVE TESTS: FAIL"
  exit 1
fi
echo "VALIDATE-FRONTMATTER NEGATIVE TESTS: PASS"
exit 0
