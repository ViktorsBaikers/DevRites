#!/usr/bin/env bash
# Loads-manifest validator gate: the script must accept a clean fixture and
# reject each integrity violation class it exists to catch.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CHECK="$ROOT/scripts/check-loads-manifest.py"
T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT

SK="$T/pack/.claude/skills"
AG="$T/pack/.claude/agents"
mkdir -p "$SK/devrites-lib/reference" "$SK/rite-build" "$AG" "$T/engine/internal/lib"

# Engine signal source: one real signal regexp so signal-matched triggers pass.
cat > "$T/engine/internal/lib/triggers.go" <<'EOF'
package lib

var triggerSignals = []struct {
	nameRe *regexp.Regexp
}{
	{regexp.MustCompile(`security`)},
}

func suggestTriggers() {}
EOF

cat > "$SK/devrites-lib/reference/workspace-artifact-schema.md" <<'EOF'
# Schema
Artifacts: spec.md plan.md tasks.md state.md
EOF

cat > "$SK/devrites-lib/reference/core.md" <<'EOF'
# core
EOF
cat > "$SK/devrites-lib/reference/extra.md" <<'EOF'
# extra
EOF
cat > "$AG/devrites-slice-wright.md" <<'EOF'
# wright
EOF

expect_fail() { # name, expected-substring
  local name="$1" want="$2"
  if python3 "$CHECK" --root "$T" >"$T/out.txt" 2>&1; then
    echo "FAIL: $name accepted (expected rejection)"; exit 1
  fi
  grep -q "$want" "$T/out.txt" || { echo "FAIL: $name missing '$want':"; cat "$T/out.txt"; exit 1; }
}

expect_ok() { # name
  if ! python3 "$CHECK" --root "$T" >"$T/out.txt" 2>&1; then
    echo "FAIL: $1 rejected:"; cat "$T/out.txt"; exit 1
  fi
}

# --- baseline: valid manifest passes --------------------------------------
cat > "$SK/rite-build/SKILL.md" <<'EOF'
---
name: rite-build
---
<!-- loads: {"always":["devrites-lib/reference/core.md"],"triggers":{"security":["devrites-lib/reference/extra.md"],"agents":["devrites-lib/reference/extra.md"]},"workspace":["spec.md"],"workspaceByRole":{"slice-wright":["spec.md"]}} -->
# build
EOF
expect_ok "valid manifest"

# --- missing always path ---------------------------------------------------
cat > "$SK/rite-build/SKILL.md" <<'EOF'
<!-- loads: {"always":["devrites-lib/reference/ghost.md"]} -->
# build
EOF
expect_fail "missing always path" "always path missing"

# --- unknown top-level key -------------------------------------------------
cat > "$SK/rite-build/SKILL.md" <<'EOF'
<!-- loads: {"always":[],"bogus":[]} -->
# build
EOF
expect_fail "unknown key" "unknown loads keys"

# --- dead trigger: no signal, no annotation, no prose mention --------------
cat > "$SK/rite-build/SKILL.md" <<'EOF'
<!-- loads: {"triggers":{"zanzibar":["devrites-lib/reference/extra.md"]}} -->
# build
EOF
expect_fail "dead trigger" "trigger 'zanzibar' is dead"

# --- live trigger via prose mention ---------------------------------------
cat > "$SK/rite-build/SKILL.md" <<'EOF'
<!-- loads: {"triggers":{"zanzibar":["devrites-lib/reference/extra.md"]}} -->
# build
Apply zanzibar handling when the plan crosses services.
EOF
expect_ok "prose-mentioned trigger"

# --- live trigger via annotation elsewhere in the pack ---------------------
cat > "$SK/devrites-lib/reference/core.md" <<'EOF'
# core
Use the spare standard (trigger `zanzibar`).
EOF
cat > "$SK/rite-build/SKILL.md" <<'EOF'
<!-- loads: {"triggers":{"zanzibar":["devrites-lib/reference/extra.md"]}} -->
# build
EOF
expect_ok "annotated trigger"
cat > "$SK/devrites-lib/reference/core.md" <<'EOF'
# core
EOF

# --- trigger path missing --------------------------------------------------
cat > "$SK/rite-build/SKILL.md" <<'EOF'
<!-- loads: {"triggers":{"security":["devrites-lib/reference/ghost.md"]}} -->
# build
EOF
expect_fail "missing trigger path" "path missing"

# --- workspaceByRole role with no agent file --------------------------------
cat > "$SK/rite-build/SKILL.md" <<'EOF'
<!-- loads: {"workspaceByRole":{"ghost-role":["spec.md"]}} -->
# build
EOF
expect_fail "unresolvable role" "maps to no agent file"

# --- agents map override resolves ------------------------------------------
cat > "$AG/custom-runner.md" <<'EOF'
# custom
EOF
cat > "$SK/rite-build/SKILL.md" <<'EOF'
<!-- loads: {"agents":{"runner":"custom-runner.md"},"workspaceByRole":{"runner":[]}} -->
# build
EOF
expect_ok "agents-map override"

# --- workspace artifact not in schema ---------------------------------------
cat > "$SK/rite-build/SKILL.md" <<'EOF'
<!-- loads: {"workspace":["bogus.md"]} -->
# build
EOF
expect_fail "unknown artifact" "not declared in workspace-artifact-schema"

# --- invalid JSON -----------------------------------------------------------
cat > "$SK/rite-build/SKILL.md" <<'EOF'
<!-- loads: {not json} -->
# build
EOF
expect_fail "invalid JSON" "not valid JSON"

echo "ok: loads-manifest validator accepts clean manifests and rejects every violation class"
