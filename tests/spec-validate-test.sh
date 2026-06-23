#!/usr/bin/env bash
# spec-validate-test.sh — unit-test the spec-grammar validator (spec-validate.sh) in isolation.
# Asserts it PASSES a well-formed structured spec and a flat-bullet spec (no-op), and FAILS
# (exit 1) on each grammar violation: missing SHALL/MUST, no scenario, scenario missing
# WHEN/THEN, duplicate headers. Plus the usage (2) and missing-spec (5) exit codes.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
SV="$ROOT/pack/.claude/skills/devrites-lib/scripts/spec-validate.sh"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT
echo "== spec-validate-test (target: $T) =="

# rc <args...> → runs the validator, captures combined output in $OUT, returns its exit code.
OUT=""
rc() { OUT="$(bash "$SV" "$@" 2>&1)"; return $?; }

mk() { mkdir -p "$T/$1"; cat > "$T/$1/spec.md"; }

# 1) VALID structured spec → exit 0.
mk valid <<'EOF'
# Spec
## Acceptance criteria
### Requirement: Tokens expire after inactivity
The system SHALL reject a session token older than 15 minutes.
#### Scenario: expired token
- [ ] [AC1] **WHEN** a token aged > 15m is presented **THEN** respond 401
#### Scenario: fresh token
- [ ] [AC2] **WHEN** a token aged < 15m is presented **THEN** allow
### Requirement: Logout revokes the token
Logout MUST invalidate the token server-side.
#### Scenario: replay after logout
- [ ] [AC3] **WHEN** a logged-out token is replayed **THEN** respond 401
EOF
rc "$T/valid" && ok "valid structured spec → exit 0" || no "valid spec rejected (rc=$?)"

# 2) FLAT-bullet spec (no structured blocks) → exit 0 (no-op, never a failure).
mk flat <<'EOF'
# Spec
## Acceptance criteria
- [ ] [AC1] export returns a CSV with a header row
- [ ] [AC2] an empty dataset returns 204
EOF
rc "$T/flat" && ok "flat acceptance → exit 0 (no-op)" || no "flat spec rejected (rc=$?)"

# 3) Requirement with no SHALL/MUST → exit 1.
mk noshall <<'EOF'
### Requirement: Export behaves nicely
The export produces a file and emails the user.
#### Scenario: happy path
- **WHEN** a user requests an export **THEN** a file is produced
EOF
rc "$T/noshall" && no "missing SHALL/MUST accepted (should be exit 1)" || { [ "$?" -eq 1 ] && printf '%s' "$OUT" | grep -q 'no SHALL/MUST' && ok "missing SHALL/MUST → exit 1" || no "wrong failure for missing SHALL (rc=$?)"; }

# 4) Requirement with no scenario → exit 1.
mk noscenario <<'EOF'
### Requirement: It works
The system SHALL do the thing.
EOF
rc "$T/noscenario" && no "no-scenario requirement accepted (should be exit 1)" || { [ "$?" -eq 1 ] && printf '%s' "$OUT" | grep -q 'no "#### Scenario:"' && ok "no scenario → exit 1" || no "wrong failure for no scenario (rc=$?)"; }

# 5) Scenario missing THEN → exit 1.
mk nothen <<'EOF'
### Requirement: It works
The system SHALL do the thing.
#### Scenario: half a scenario
- **WHEN** something happens
EOF
rc "$T/nothen" && no "scenario without THEN accepted (should be exit 1)" || { [ "$?" -eq 1 ] && printf '%s' "$OUT" | grep -q 'no THEN line' && ok "scenario missing THEN → exit 1" || no "wrong failure for missing THEN (rc=$?)"; }

# 6) Duplicate Requirement headers → exit 1.
mk dup <<'EOF'
### Requirement: Same name
The system SHALL alpha.
#### Scenario: a
- **WHEN** a **THEN** b
### Requirement: Same name
The system SHALL beta.
#### Scenario: c
- **WHEN** c **THEN** d
EOF
rc "$T/dup" && no "duplicate headers accepted (should be exit 1)" || { [ "$?" -eq 1 ] && printf '%s' "$OUT" | grep -q 'duplicate Requirement header' && ok "duplicate headers → exit 1" || no "wrong failure for duplicate headers (rc=$?)"; }

# 7) Accepts a direct file path, not just a dir → exit 0.
rc "$T/valid/spec.md" && ok "direct spec.md file path → exit 0" || no "direct file path rejected (rc=$?)"

# 8) Missing spec.md in an existing dir → exit 5.
mkdir -p "$T/empty"
rc "$T/empty"; [ "$?" -eq 5 ] && ok "dir without spec.md → exit 5" || no "missing spec.md wrong exit (rc=$?)"

# 9) No argument → usage, exit 2.
rc; [ "$?" -eq 2 ] && ok "no argument → exit 2 (usage)" || no "missing arg wrong exit (rc=$?)"

echo ""
[ "$fail" -eq 0 ] && echo "spec-validate-test: PASS" || echo "spec-validate-test: FAIL"
exit "$fail"
