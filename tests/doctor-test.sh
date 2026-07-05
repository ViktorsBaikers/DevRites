#!/usr/bin/env bash
# doctor-test.sh — fixture tests for the DevRites health diagnose (doctor.sh).
# Healthy project → clean (exit 0); each corruption → detected (exit 1 + the named issue).
set -u

HERE="$(cd "$(dirname "$0")" && pwd)"
DOCTOR="$HERE/../pack/.claude/skills/devrites-lib/scripts/doctor.sh"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
fail=0

# Build a HEALTHY DevRites project at $1 (install scripts present, empty ACTIVE).
mkproj() {
  local d="$1"
  mkdir -p "$d/.devrites/work" "$d/.claude/skills/devrites-lib/scripts" "$d/.claude/hooks"
  for s in preamble.sh progress.sh readiness.sh conventions.py; do : > "$d/.claude/skills/devrites-lib/scripts/$s"; done
  : > "$d/.devrites/ACTIVE"
}

write_manifest() {
  local d="$1"; shift
  mkdir -p "$d/.claude"
  {
    printf '# devrites-version: test\n'
    printf '# devrites-flags: %s\n' "${1:-}"
    shift || true
    printf '%s\n' "$@"
  } > "$d/.claude/devrites.manifest"
}

add_codex() {
  local d="$1"
  mkdir -p "$d/.agents/skills/rite" "$d/.agents/devrites/rules" "$d/.codex/agents" "$d/.codex/hooks" "$d/.codex/mcp"
  printf '# rite\n' > "$d/.agents/skills/rite/SKILL.md"
  printf '# core\n' > "$d/.agents/devrites/rules/core.md"
  printf 'name = "devrites-code-reviewer"\n' > "$d/.codex/agents/devrites-code-reviewer.toml"
  printf '{ "hooks": { "Stop": [ { "command": "bash .codex/hooks/devrites-stop-gate.sh" } ] } }\n' > "$d/.codex/hooks.json"
  : > "$d/.codex/hooks/devrites-stop-gate.sh"
  : > "$d/.codex/mcp/devrites-mcp.mjs"
  printf '[mcp_servers.devrites]\ncommand = "node"\nargs = [".codex/mcp/devrites-mcp.mjs"]\n' > "$d/.codex/config.toml"
  printf '# AGENTS.md — DevRites for Codex\n' > "$d/AGENTS.md"
  write_manifest "$d" "" \
    ".agents/skills/rite/SKILL.md" \
    ".agents/devrites/rules/core.md" \
    ".codex/agents/devrites-code-reviewer.toml" \
    ".codex/hooks.json" \
    ".codex/hooks/devrites-stop-gate.sh" \
    ".codex/mcp/devrites-mcp.mjs" \
    ".codex/config.toml" \
    "AGENTS.md"
}

add_codex_no_agents() {
  local d="$1"
  mkdir -p "$d/.agents/skills/rite" "$d/.agents/devrites/rules" "$d/.codex/hooks" "$d/.codex/mcp"
  printf '# rite\n' > "$d/.agents/skills/rite/SKILL.md"
  printf '# core\n' > "$d/.agents/devrites/rules/core.md"
  printf '{ "hooks": { "Stop": [ { "command": "bash .codex/hooks/devrites-stop-gate.sh" } ] } }\n' > "$d/.codex/hooks.json"
  : > "$d/.codex/hooks/devrites-stop-gate.sh"
  : > "$d/.codex/mcp/devrites-mcp.mjs"
  printf '[mcp_servers.devrites]\ncommand = "node"\nargs = [".codex/mcp/devrites-mcp.mjs"]\n' > "$d/.codex/config.toml"
  printf '# AGENTS.md — DevRites for Codex\n' > "$d/AGENTS.md"
  write_manifest "$d" "--no-agents" \
    ".agents/skills/rite/SKILL.md" \
    ".agents/devrites/rules/core.md" \
    ".codex/hooks.json" \
    ".codex/hooks/devrites-stop-gate.sh" \
    ".codex/mcp/devrites-mcp.mjs" \
    ".codex/config.toml" \
    "AGENTS.md"
}

# run doctor at root $1; expect exit code $2; if $3 given, the output must contain it.
expect() { # label root want_rc [grep]
  local out rc
  out="$(bash "$DOCTOR" --root "$2" 2>&1)"; rc=$?
  if [ "$rc" -ne "$3" ]; then echo "FAIL [$1]: rc $rc != $3"; echo "$out"; fail=1; return; fi
  if [ -n "${4:-}" ] && ! printf '%s' "$out" | grep -q "$4"; then
    echo "FAIL [$1]: output missing /$4/"; echo "$out"; fail=1; return
  fi
  echo "ok   [$1]"
}

# healthy → exit 0
H="$TMP/healthy"; mkproj "$H"
expect "healthy → clean" "$H" 0

# healthy Codex support → exit 0
CX="$TMP/codex"; mkproj "$CX"; add_codex "$CX"
expect "healthy Codex support → clean" "$CX" 0

# --no-agents Codex installs should not require .codex/agents
NA="$TMP/noagents"; mkproj "$NA"; add_codex_no_agents "$NA"
expect "--no-agents Codex support → clean" "$NA" 0

# not a DevRites project → silent exit 0
N="$TMP/plain"; mkdir -p "$N"
expect "non-devrites → silent" "$N" 0

# --no-codex installs may coexist with unrelated project AGENTS.md/.codex files.
NC="$TMP/nocodex"; mkproj "$NC"
mkdir -p "$NC/.codex"
printf '# project guidance\n' > "$NC/AGENTS.md"
printf 'model = "gpt-5-codex"\n' > "$NC/.codex/config.toml"
write_manifest "$NC" "--no-codex" ".claude/skills/devrites-lib/scripts/preamble.sh"
expect "--no-codex ignores unrelated Codex files" "$NC" 0

# stale ACTIVE → exit 1
S="$TMP/stale"; mkproj "$S"; printf 'ghost' > "$S/.devrites/ACTIVE"
expect "stale ACTIVE detected" "$S" 1 "stale ACTIVE"

# corrupt workspace (no state.md) → exit 1
C="$TMP/corrupt"; mkproj "$C"; printf 'feat' > "$C/.devrites/ACTIVE"; mkdir -p "$C/.devrites/work/feat"
expect "corrupt workspace detected" "$C" 1 "corrupt workspace"

# orphaned gate (open validating + state done) → exit 1
O="$TMP/orphan"; mkproj "$O"; printf 'feat' > "$O/.devrites/ACTIVE"; mkdir -p "$O/.devrites/work/feat"
printf 'phase: done\n' > "$O/.devrites/work/feat/state.md"
printf 'status: open\ngate: validating\n' > "$O/.devrites/work/feat/questions.md"
expect "orphaned gate detected" "$O" 1 "orphaned gate"

# broken hook wiring (settings references a missing hook) → exit 1
B="$TMP/brokenhook"; mkproj "$B"
printf '{ "hooks": { "SessionStart": [ { "command": "bash .claude/hooks/devrites-orient.sh" } ] } }\n' > "$B/.claude/settings.json"
expect "broken hook wiring detected" "$B" 1 "missing hook"

# broken Codex hook wiring → exit 1
CB="$TMP/brokencodexhook"; mkproj "$CB"; add_codex "$CB"; rm "$CB/.codex/hooks/devrites-stop-gate.sh"
expect "broken Codex hook wiring detected" "$CB" 1 "Codex hooks.json wires missing hook"

# partial Codex support → exit 1
CP="$TMP/partialcodex"; mkproj "$CP"; mkdir -p "$CP/.agents/skills"
write_manifest "$CP" "" ".agents/skills/rite/SKILL.md"
expect "partial Codex support detected" "$CP" 1 "Codex skill mirror incomplete"

# missing install script → exit 1
M="$TMP/incomplete"; mkproj "$M"; rm "$M/.claude/skills/devrites-lib/scripts/conventions.py"
expect "incomplete install detected" "$M" 1 "install incomplete"

# verbose prints ok: lines on a healthy project
verbose_out="$(bash "$DOCTOR" --root "$H" --verbose 2>&1)"
if printf '%s\n' "$verbose_out" | grep -q '^ok:'; then echo "ok   [verbose prints ok lines]"; else echo "FAIL [verbose prints ok lines]"; fail=1; fi

if [ "$fail" -ne 0 ]; then echo "DOCTOR TESTS: FAIL"; exit 1; fi
echo "DOCTOR TESTS: PASS"
exit 0
