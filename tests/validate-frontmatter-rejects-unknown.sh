#!/usr/bin/env bash
# validate-frontmatter-rejects-unknown.sh: assert validate-frontmatter.py
# fails (non-zero) on duplicate or non-canonical fields, on an agent description
# over 45 words, on a description > 1024 chars, and on a multi-line description.
# validate-frontmatter-fails-closed.sh covers the PyYAML refusal and unparsable YAML.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
PY="$ROOT/scripts/validate-frontmatter.py"

if ! command -v python3 >/dev/null 2>&1; then
  echo "skip: python3 not found"
  exit 0
fi

if ! python3 -c 'import yaml' 2>/dev/null; then
  echo "FAIL: PyYAML not installed; run: pip install -r scripts/requirements-ci.txt"
  exit 1
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
(cd "$bare" && printf '%s\n' '---' 'name: bare-skill' 'description: A skill validated from a directory-less path.' 'user-invocable: true' '---' body > SKILL.md && python3 "$PY" SKILL.md >"$TMP/out" 2>&1)
code=$?
if [ "$code" -eq 0 ]; then
  printf 'ok: bare-path handled cleanly\n'
else
  printf 'FAIL: bare-path: validator exited %s (expected 0)\n' "$code"
  cat "$TMP/out"
  fail=1
fi

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


# 15) shipped descriptions containing ': ' must be quoted so they parse as scalars
assert_ok "shipped-overhaul-description" "$ROOT/pack/.claude/skills/overhaul/SKILL.md"
assert_ok "shipped-rite-fast-description" "$ROOT/pack/.claude/skills/rite-fast/SKILL.md"

# 16) agent tools must be a comma-separated scalar; list forms are misread by the host generators
cat > "$TMP/agents/devrites-fx-block.md" <<'EOF2'
---
name: devrites-fx-block
description: A read-only agent that declares tools as a block list.
tools:
  - Read
  - Grep
  - Glob
---
body
EOF2
assert_fail "agent-tools-block-list" "$TMP/agents/devrites-fx-block.md"

cat > "$TMP/agents/devrites-fx-flow.md" <<'EOF2'
---
name: devrites-fx-flow
description: A read-only agent that declares tools as a flow list.
tools: [Read, Grep, Glob]
---
body
EOF2
assert_fail "agent-tools-flow-list" "$TMP/agents/devrites-fx-flow.md"

cat > "$TMP/agents/devrites-fx-tools-folded.md" <<'EOF2'
---
name: devrites-fx-tools-folded
description: A read-only agent that declares tools as a folded scalar.
tools: >-
  Read, Grep, Glob
---
body
EOF2
assert_fail "agent-tools-folded" "$TMP/agents/devrites-fx-tools-folded.md"

cat > "$TMP/agents/devrites-fx-tools-quoted.md" <<'EOF2'
---
name: devrites-fx-tools-quoted
description: A read-only agent that declares tools as a quoted scalar.
tools: 'Read, Glob'
---
body
EOF2
assert_fail "agent-tools-single-quoted" "$TMP/agents/devrites-fx-tools-quoted.md"

cat > "$TMP/agents/devrites-fx-tools-dquoted.md" <<'EOF2'
---
name: devrites-fx-tools-dquoted
description: A read-only agent that declares tools as a double quoted scalar.
tools: "Read, Grep, Glob"
---
body
EOF2
assert_fail "agent-tools-double-quoted" "$TMP/agents/devrites-fx-tools-dquoted.md"

cat > "$TMP/agents/devrites-fx-tools-comment.md" <<'EOF2'
---
name: devrites-fx-tools-comment
description: A read-only agent that declares tools with a trailing comment.
tools: Read # read only
---
body
EOF2
assert_fail "agent-tools-trailing-comment" "$TMP/agents/devrites-fx-tools-comment.md"

# 17) agent description must be a plain scalar; folded, literal and quoted forms are misread
cat > "$TMP/agents/devrites-fx-folded.md" <<'EOF2'
---
name: devrites-fx-folded
description: >-
  A read-only agent whose description is a folded scalar.
tools: Read, Grep, Glob
---
body
EOF2
assert_fail "agent-description-folded" "$TMP/agents/devrites-fx-folded.md"

cat > "$TMP/agents/devrites-fx-quoted.md" <<'EOF2'
---
name: devrites-fx-quoted
description: "A read-only agent whose description is double quoted."
tools: Read, Grep, Glob
---
body
EOF2
assert_fail "agent-description-quoted" "$TMP/agents/devrites-fx-quoted.md"

cat > "$TMP/agents/devrites-fx-plain.md" <<'EOF2'
---
name: devrites-fx-plain
description: A read-only agent with plain scalar fields.
tools: Read, Grep, Glob
---
body
EOF2
assert_ok "agent-plain-scalars" "$TMP/agents/devrites-fx-plain.md"

# 18) agent frontmatter is an allowlist: only the exact line grammar the host generators parse
agent_fx() {  # name, then printf-style frontmatter lines (written verbatim, LF unless the line carries \r)
  _n="$1"; shift
  printf -- "$@" >"$TMP/agents/devrites-fx-$_n.md"
}
G='description: A read-only agent fixture.\n'
agent_fx crlf '---\r\nname: devrites-fx-crlf\r\n'"$G"'tools: Read, Grep, Glob\r\n---\r\nbody\r\n'
assert_fail "agent-crlf" "$TMP/agents/devrites-fx-crlf.md"
agent_fx crlf-fence '---\r\nname: devrites-fx-crlf-fence\n'"$G"'tools: Read, Grep, Glob\n---\nbody\n'
assert_fail "agent-crlf-fence-only" "$TMP/agents/devrites-fx-crlf-fence.md"
agent_fx fence-space '--- \nname: devrites-fx-fence-space\n'"$G"'tools: Read, Grep, Glob\n---\nbody\n'
assert_fail "agent-fence-trailing-space" "$TMP/agents/devrites-fx-fence-space.md"
agent_fx close-space '---\nname: devrites-fx-close-space\n'"$G"'tools: Read\n--- \nbody\n'
assert_fail "agent-closing-fence-trailing-space" "$TMP/agents/devrites-fx-close-space.md"
agent_fx dup-plain '---\nname: devrites-fx-dup-plain\n'"$G"'tools: Read, Grep, Glob\ntools: Bash\n---\nbody\n'
assert_fail "agent-duplicate-plain-key" "$TMP/agents/devrites-fx-dup-plain.md"
agent_fx dup-quoted '---\nname: devrites-fx-dup-quoted\n'"$G"'tools: Bash\n"tools": Read\n---\nbody\n'
assert_fail "agent-duplicate-quoted-key" "$TMP/agents/devrites-fx-dup-quoted.md"
agent_fx dup-space '---\nname: devrites-fx-dup-space\n'"$G"'tools: Bash\ntools : Read\n---\nbody\n'
assert_fail "agent-duplicate-space-colon-key" "$TMP/agents/devrites-fx-dup-space.md"
agent_fx dup-explicit '---\nname: devrites-fx-dup-explicit\n'"$G"'tools: Bash\n? tools\n: Read, Grep, Glob\n---\nbody\n'
assert_fail "agent-explicit-complex-key" "$TMP/agents/devrites-fx-dup-explicit.md"
agent_fx shadow '---\nname: devrites-fx-shadow\n'"$G"'initialPrompt: "start\ntools: Bash, Read"\n? tools\n: Read, Grep, Glob\n---\nbody\n'
assert_fail "agent-quoted-multiline-shadow" "$TMP/agents/devrites-fx-shadow.md"
agent_fx tools-cont '---\nname: devrites-fx-tools-cont\n'"$G"'tools: Read\n  , Grep, Glob, Bash\n---\nbody\n'
assert_fail "agent-tools-continuation" "$TMP/agents/devrites-fx-tools-cont.md"
agent_fx tools-cont2 '---\nname: devrites-fx-tools-cont2\n'"$G"'tools: TodoWrite\n  , Read, Grep, Glob\n---\nbody\n'
assert_fail "agent-tools-continuation-unmapped" "$TMP/agents/devrites-fx-tools-cont2.md"
agent_fx desc-cont '---\nname: devrites-fx-desc-cont\ndescription: A read-only agent\n  whose description continues.\ntools: Read, Grep, Glob\n---\nbody\n'
assert_fail "agent-description-continuation" "$TMP/agents/devrites-fx-desc-cont.md"
agent_fx desc-comment '---\nname: devrites-fx-desc-comment\ndescription: A read-only agent # hidden\ntools: Read, Grep, Glob\n---\nbody\n'
assert_fail "agent-description-comment" "$TMP/agents/devrites-fx-desc-comment.md"
agent_fx desc-flow '---\nname: devrites-fx-desc-flow\ndescription: [A read-only agent, fixture]\ntools: Read, Grep, Glob\n---\nbody\n'
assert_fail "agent-description-flow-list" "$TMP/agents/devrites-fx-desc-flow.md"
agent_fx desc-hash '---\nname: devrites-fx-desc-hash\ndescription: # not a description\ntools: Read, Grep, Glob\n---\nbody\n'
assert_fail "agent-description-hash-start" "$TMP/agents/devrites-fx-desc-hash.md"
agent_fx desc-alias '---\nname: devrites-fx-desc-alias\ndescription: *anchor\ntools: Read, Grep, Glob\n---\nbody\n'
assert_fail "agent-description-alias" "$TMP/agents/devrites-fx-desc-alias.md"
agent_fx tools-empty '---\nname: devrites-fx-tools-empty\n'"$G"'tools:\n---\nbody\n'
assert_fail "agent-tools-empty" "$TMP/agents/devrites-fx-tools-empty.md"
agent_fx tab-colon '---\nname: devrites-fx-tab-colon\n'"$G"'tools:\tRead, Grep, Glob\n---\nbody\n'
assert_fail "agent-tab-after-colon" "$TMP/agents/devrites-fx-tab-colon.md"
agent_fx tab-trail '---\nname: devrites-fx-tab-trail\n'"$G"'tools: Read, Grep, Glob\t\n---\nbody\n'
assert_fail "agent-tools-trailing-tab" "$TMP/agents/devrites-fx-tab-trail.md"
agent_fx nbsp '---\nname: devrites-fx-nbsp\n'"$G"'tools:\302\240Read, Grep\n---\nbody\n'
assert_fail "agent-nbsp-after-colon" "$TMP/agents/devrites-fx-nbsp.md"
agent_fx fullwidth '---\nname: devrites-fx-fullwidth\n'"$G"'tools: Read\357\274\214 Bash\n---\nbody\n'
assert_fail "agent-fullwidth-comma" "$TMP/agents/devrites-fx-fullwidth.md"
agent_fx comment-line '---\nname: devrites-fx-comment-line\n# a comment\n'"$G"'tools: Read, Grep, Glob\n---\nbody\n'
assert_fail "agent-comment-line" "$TMP/agents/devrites-fx-comment-line.md"
agent_fx blank-line '---\nname: devrites-fx-blank-line\n\n'"$G"'tools: Read, Grep, Glob\n---\nbody\n'
assert_fail "agent-blank-line" "$TMP/agents/devrites-fx-blank-line.md"
agent_fx no-close '---\nname: devrites-fx-no-close\n'"$G"'tools: Read, Grep, Glob\nbody\n'
assert_fail "agent-missing-closing-fence" "$TMP/agents/devrites-fx-no-close.md"

# 19) shipped forms: camelCase keys, '*' in tool tokens, and a flat block list under a list key
agent_fx camel '---\nname: devrites-fx-camel\n'"$G"'tools: Read, Grep\npermissionMode: plan\n---\nbody\n'
assert_ok "agent-camelcase-key" "$TMP/agents/devrites-fx-camel.md"
agent_fx wild '---\nname: devrites-fx-wild\n'"$G"'tools: Read, mcp__codegraph__*, mcp__graphify__*\n---\nbody\n'
assert_ok "agent-tools-wildcard" "$TMP/agents/devrites-fx-wild.md"
agent_fx skills-block '---\nname: devrites-fx-skills-block\n'"$G"'tools: Read\nskills:\n  - devrites-frontend-craft\n  - devrites-ux-shape\npermissionMode: plan\n---\nbody\n'
assert_ok "agent-skills-block-list" "$TMP/agents/devrites-fx-skills-block.md"
agent_fx unknown-key '---\nname: devrites-fx-unknown-key\n'"$G"'tools: Read\nmadeUpField: yes\n---\nbody\n'
assert_fail "agent-unknown-camelcase-key" "$TMP/agents/devrites-fx-unknown-key.md"
agent_fx nested-list '---\nname: devrites-fx-nested-list\n'"$G"'tools: Read\nskills:\n  - devrites-a\n    - devrites-b\n---\nbody\n'
assert_fail "agent-nested-block-list" "$TMP/agents/devrites-fx-nested-list.md"
agent_fx map-item '---\nname: devrites-fx-map-item\n'"$G"'tools: Read\nskills:\n  - name: devrites-a\n---\nbody\n'
assert_fail "agent-block-list-mapping-item" "$TMP/agents/devrites-fx-map-item.md"
agent_fx list-nonlist '---\nname: devrites-fx-list-nonlist\n'"$G"'tools: Read\npermissionMode:\n  - plan\n---\nbody\n'
assert_fail "agent-block-list-under-scalar-key" "$TMP/agents/devrites-fx-list-nonlist.md"
agent_fx skills-empty '---\nname: devrites-fx-skills-empty\n'"$G"'tools: Read\nskills:\npermissionMode: plan\n---\nbody\n'
assert_fail "agent-list-key-without-items" "$TMP/agents/devrites-fx-skills-empty.md"

# 20) the validator must see the same lines as the awk readers (split on LF only): any
#     control or Unicode line separator inside the frontmatter can hide a line from them
agent_fx nel-tools '---\nname: devrites-fx-nel-tools\ndescription: Reviews code read-only.\302\205tools: Read, Grep\n---\nbody\n'
assert_fail "agent-nel-hides-tools" "$TMP/agents/devrites-fx-nel-tools.md"
agent_fx ls-tools '---\nname: devrites-fx-ls-tools\ndescription: Reviews code read-only.\342\200\250tools: Read, Grep\n---\nbody\n'
assert_fail "agent-line-separator-hides-tools" "$TMP/agents/devrites-fx-ls-tools.md"
agent_fx ps-tools '---\nname: devrites-fx-ps-tools\ndescription: Reviews code read-only.\342\200\251tools: Read, Grep\n---\nbody\n'
assert_fail "agent-paragraph-separator" "$TMP/agents/devrites-fx-ps-tools.md"
agent_fx ff-fence '---\nname: devrites-fx-ff-fence\ndescription: Reviews code.\nmodel: sonnet\014---\nevilKey: yes\ntools: Read\n---\nbody\n'
assert_fail "agent-form-feed-hidden-fence" "$TMP/agents/devrites-fx-ff-fence.md"
agent_fx nel-fence '---\nname: devrites-fx-nel-fence\ndescription: Reviews code.\nmodel: sonnet\302\205---\nevilKey: yes\ntools: Read\n---\nbody\n'
assert_fail "agent-nel-hidden-fence" "$TMP/agents/devrites-fx-nel-fence.md"
agent_fx vt-fence '---\nname: devrites-fx-vt-fence\ndescription: Reviews code.\nmodel: sonnet\013---\nevilKey: yes\n---\nbody\n'
assert_fail "agent-vertical-tab-hidden-fence" "$TMP/agents/devrites-fx-vt-fence.md"
agent_fx ff-open '\014---\nname: devrites-fx-ff-open\n'"$G"'tools: Read\n---\nbody\n'
assert_fail "agent-form-feed-before-opening-fence" "$TMP/agents/devrites-fx-ff-open.md"
agent_fx nel-open '\302\205---\nname: devrites-fx-nel-open\n'"$G"'tools: Read\n---\nbody\n'
assert_fail "agent-nel-before-opening-fence" "$TMP/agents/devrites-fx-nel-open.md"
agent_fx c0 '---\nname: devrites-fx-c0\n'"$G"'tools: Read\037\n---\nbody\n'
assert_fail "agent-c0-control" "$TMP/agents/devrites-fx-c0.md"
agent_fx del '---\nname: devrites-fx-del\n'"$G"'tools: Read\177\n---\nbody\n'
assert_fail "agent-del-control" "$TMP/agents/devrites-fx-del.md"
agent_fx c1 '---\nname: devrites-fx-c1\ndescription: Reviews\302\230 code.\ntools: Read\n---\nbody\n'
assert_fail "agent-c1-control" "$TMP/agents/devrites-fx-c1.md"
agent_fx zwnbsp '---\nname: devrites-fx-zwnbsp\ndescription: Reviews\357\273\277 code.\ntools: Read\n---\nbody\n'
assert_fail "agent-zero-width-no-break-space" "$TMP/agents/devrites-fx-zwnbsp.md"
agent_fx bare-cr '---\nname: devrites-fx-bare-cr\ndescription: Reviews code.\rtools: Read\n---\nbody\n'
assert_fail "agent-bare-carriage-return" "$TMP/agents/devrites-fx-bare-cr.md"

# 21) every agent scalar must be complete on its own line: a value that opens a quote closes it
#     on that line, block scalars are rejected, and no HTML comment marker (include) is allowed
agent_fx dq-multiline '---\nname: devrites-fx-dq-multiline\n'"$G"'model: "sonnet\ntools: Read, Grep\neffort: high"\n---\nbody\n'
assert_fail "agent-double-quote-spans-lines" "$TMP/agents/devrites-fx-dq-multiline.md"
agent_fx sq-multiline '---\nname: devrites-fx-sq-multiline\n'"$G"'model: '"'"'sonnet\ntools: Read, Grep\neffort: high'"'"'\n---\nbody\n'
assert_fail "agent-single-quote-spans-lines" "$TMP/agents/devrites-fx-sq-multiline.md"
agent_fx dq-color '---\nname: devrites-fx-dq-color\n'"$G"'color: "blue\ntools: Read\nmaxTurns: 3"\n---\nbody\n'
assert_fail "agent-quote-spans-lines-other-key" "$TMP/agents/devrites-fx-dq-color.md"
agent_fx dq-name '---\nname: devrites-fx-dq-name\nmodel: "x\nname: overhaul-evil\ny"\n'"$G"'---\nbody\n'
assert_fail "agent-quote-hides-name" "$TMP/agents/devrites-fx-dq-name.md"
agent_fx dq-trailing '---\nname: devrites-fx-dq-trailing\n'"$G"'model: "sonnet" x\ntools: Read\n---\nbody\n'
assert_fail "agent-quote-closed-then-text" "$TMP/agents/devrites-fx-dq-trailing.md"
agent_fx flowseq-multiline '---\nname: devrites-fx-flowseq-multiline\n'"$G"'model: [sonnet,\ntools: Read, Grep]\n---\nbody\n'
assert_fail "agent-flow-seq-spans-lines" "$TMP/agents/devrites-fx-flowseq-multiline.md"
agent_fx flowmap-multiline '---\nname: devrites-fx-flowmap-multiline\n'"$G"'model: {a: b,\ntools: Read, Grep}\n---\nbody\n'
assert_fail "agent-flow-map-spans-lines" "$TMP/agents/devrites-fx-flowmap-multiline.md"
agent_fx block-literal '---\nname: devrites-fx-block-literal\n'"$G"'model: |\ntools: Read\n---\nbody\n'
assert_fail "agent-block-literal-scalar" "$TMP/agents/devrites-fx-block-literal.md"
agent_fx block-folded '---\nname: devrites-fx-block-folded\n'"$G"'model: >-\ntools: Read\n---\nbody\n'
assert_fail "agent-block-folded-scalar" "$TMP/agents/devrites-fx-block-folded.md"
agent_fx include-desc '---\nname: devrites-fx-include-desc\ndescription: Reviews code <!-- include:_shared/t.md -->\ntools: Read, Grep\n---\nbody\n'
assert_fail "agent-include-marker-in-value" "$TMP/agents/devrites-fx-include-desc.md"
agent_fx html-comment '---\nname: devrites-fx-html-comment\n'"$G"'model: sonnet <!-- x -->\ntools: Read\n---\nbody\n'
assert_fail "agent-html-comment-in-value" "$TMP/agents/devrites-fx-html-comment.md"
agent_fx tools-null '---\nname: devrites-fx-tools-null\n'"$G"'tools: null\n---\nbody\n'
assert_fail "agent-tools-null" "$TMP/agents/devrites-fx-tools-null.md"
agent_fx tools-false '---\nname: devrites-fx-tools-false\n'"$G"'tools: false\n---\nbody\n'
assert_fail "agent-tools-false" "$TMP/agents/devrites-fx-tools-false.md"
agent_fx tools-true '---\nname: devrites-fx-tools-true\n'"$G"'tools: true\n---\nbody\n'
assert_fail "agent-tools-true" "$TMP/agents/devrites-fx-tools-true.md"
agent_fx tools-no '---\nname: devrites-fx-tools-no\n'"$G"'tools: No\n---\nbody\n'
assert_fail "agent-tools-no" "$TMP/agents/devrites-fx-tools-no.md"
agent_fx tools-tilde '---\nname: devrites-fx-tools-tilde\n'"$G"'tools: ~\n---\nbody\n'
assert_fail "agent-tools-tilde" "$TMP/agents/devrites-fx-tools-tilde.md"
agent_fx name-comment '---\nname: devrites-fx-name-comment #x\n'"$G"'tools: Read\n---\nbody\n'
assert_fail "agent-name-trailing-comment" "$TMP/agents/devrites-fx-name-comment.md"
agent_fx quoted-ok '---\nname: devrites-fx-quoted-ok\n'"$G"'model: "sonnet"\ntools: Read, Grep\n---\nbody\n'
assert_ok "agent-closed-quoted-value" "$TMP/agents/devrites-fx-quoted-ok.md"

# 22) a scalar may not open with a YAML indicator (anchor, tag, alias, ...): YAML then reads the
#     value differently from the line readers, so lines the readers see can vanish from YAML
agent_fx anchor-dq '---\nname: devrites-fx-anchor-dq\n'"$G"'model: &m "sonnet\ntools: Read, Grep\neffort: high"\n---\nbody\n'
assert_fail "agent-anchor-double-quote-hides-tools" "$TMP/agents/devrites-fx-anchor-dq.md"
agent_fx anchor-sq '---\nname: devrites-fx-anchor-sq\n'"$G"'model: &m '"'"'sonnet\ntools: Read, Grep\neffort: high'"'"'\n---\nbody\n'
assert_fail "agent-anchor-single-quote-hides-tools" "$TMP/agents/devrites-fx-anchor-sq.md"
agent_fx tag-dq '---\nname: devrites-fx-tag-dq\n'"$G"'model: !!str "sonnet\ntools: Read, Grep\neffort: high"\n---\nbody\n'
assert_fail "agent-tag-double-quote-hides-tools" "$TMP/agents/devrites-fx-tag-dq.md"
agent_fx anchor-plain '---\nname: devrites-fx-anchor-plain\n'"$G"'model: &m sonnet\ntools: Read\n---\nbody\n'
assert_fail "agent-anchor-plain-value" "$TMP/agents/devrites-fx-anchor-plain.md"
agent_fx alias-value '---\nname: devrites-fx-alias-value\n'"$G"'model: *m\ntools: Read\n---\nbody\n'
assert_fail "agent-alias-value" "$TMP/agents/devrites-fx-alias-value.md"
agent_fx tag-plain '---\nname: devrites-fx-tag-plain\n'"$G"'model: !!str sonnet\ntools: Read\n---\nbody\n'
assert_fail "agent-tag-plain-value" "$TMP/agents/devrites-fx-tag-plain.md"
agent_fx directive-value '---\nname: devrites-fx-directive-value\n'"$G"'model: %%sonnet\ntools: Read\n---\nbody\n'
assert_fail "agent-percent-value" "$TMP/agents/devrites-fx-directive-value.md"
agent_fx at-value '---\nname: devrites-fx-at-value\n'"$G"'model: @sonnet\ntools: Read\n---\nbody\n'
assert_fail "agent-at-value" "$TMP/agents/devrites-fx-at-value.md"
agent_fx tick-value '---\nname: devrites-fx-tick-value\n'"$G"'model: `sonnet\ntools: Read\n---\nbody\n'
assert_fail "agent-backtick-value" "$TMP/agents/devrites-fx-tick-value.md"
agent_fx flowmap-value '---\nname: devrites-fx-flowmap-value\n'"$G"'model: {a: b}\ntools: Read\n---\nbody\n'
assert_fail "agent-flow-map-value" "$TMP/agents/devrites-fx-flowmap-value.md"
agent_fx anchor-skills '---\nname: devrites-fx-anchor-skills\n'"$G"'tools: Read\nmodel: &m "sonnet\nskills:\n  - a\neffort: high"\n---\nbody\n'
assert_fail "agent-anchor-hides-skills" "$TMP/agents/devrites-fx-anchor-skills.md"
agent_fx sq-ok '---\nname: devrites-fx-sq-ok\n'"$G"'model: '"'"'sonnet'"'"'\ntools: Read, Grep\n---\nbody\n'
assert_ok "agent-closed-single-quoted-value" "$TMP/agents/devrites-fx-sq-ok.md"
agent_fx plain-num '---\nname: devrites-fx-plain-num\n'"$G"'maxTurns: 3\nbackground: true\ntools: Read\n---\nbody\n'
assert_ok "agent-plain-number-and-bool" "$TMP/agents/devrites-fx-plain-num.md"

rm -rf "$TMP"

if [ "$fail" -ne 0 ]; then
  echo "VALIDATE-FRONTMATTER NEGATIVE TESTS: FAIL"
  exit 1
fi
echo "VALIDATE-FRONTMATTER NEGATIVE TESTS: PASS"
exit 0
