#!/usr/bin/env bash
# a1-guard-test.sh — unit-test the A1 pre-block hook (devrites-a1-guard.sh) in isolation.
# Asserts it blocks ONLY a main-thread source edit inside an open build window under enforce,
# and fails open in every other case (no workspace, no window, subagent, .devrites path,
# inline fallback, observe mode).
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
HOOK="$ROOT/pack/.claude/hooks/devrites-a1-guard.sh"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT
echo "== a1-guard-test (target: $T) =="

mkdir -p "$T/.devrites/work/demo" "$T/src"
printf 'demo\n' > "$T/.devrites/ACTIVE"
D="$T/.devrites/work/demo"

# run <payload-json> [enforce]  → echoes hook stdout; sets RC
run() {
  local payload="$1" enforce="${2:-}"
  if [ "$enforce" = "enforce" ]; then
    CLAUDE_PROJECT_DIR="$T" DEVRITES_A1_HOOK=enforce bash "$HOOK" <<<"$payload"
  else
    CLAUDE_PROJECT_DIR="$T" bash "$HOOK" <<<"$payload"
  fi
}
denies() { printf '%s' "$1" | grep -q '"permissionDecision":"deny"'; }

main_src='{"tool_name":"Edit","tool_input":{"file_path":"'"$T"'/src/a.ts"},"cwd":"'"$T"'"}'
sub_src='{"tool_name":"Edit","tool_input":{"file_path":"'"$T"'/src/a.ts"},"agent_id":"wright-123","agent_type":"devrites-slice-wright"}'
book='{"tool_name":"Write","tool_input":{"file_path":"'"$T"'/.devrites/work/demo/state.md"}}'

# 1) No window open yet → allow even a main-thread source edit.
out="$(run "$main_src")"; denies "$out" && no "blocked with no window open" || ok "no window → allow"

# Open the build window.
: > "$D/.reconcile-base"

# 2) Window + main-thread source edit, OBSERVE (default) → allow but log.
out="$(run "$main_src")"
denies "$out" && no "observe mode denied (should only log)" || ok "observe → allow"
[ -f "$D/.a1-guard.log" ] && grep -q "WOULD-BLOCK" "$D/.a1-guard.log" && ok "observe logged WOULD-BLOCK" || no "observe did not log"

# 3) Window + main-thread source edit, ENFORCE → deny.
out="$(run "$main_src" enforce)"; denies "$out" && ok "enforce → deny main-thread source edit" || no "enforce did not deny"

# 4) Window + SUBAGENT (wright) source edit, ENFORCE → allow (agent_id present).
out="$(run "$sub_src" enforce)"; denies "$out" && no "blocked the wright subagent!" || ok "subagent (agent_id) → allow"

# 5) Window + .devrites bookkeeping write, ENFORCE → allow.
out="$(run "$book" enforce)"; denies "$out" && no "blocked a .devrites bookkeeping write" || ok ".devrites path → allow"

# 6) Window + inline fallback sentinel + main-thread source edit, ENFORCE → allow.
: > "$D/.reconcile-inline"
out="$(run "$main_src" enforce)"; denies "$out" && no "blocked inline-fallback writer" || ok "inline fallback → allow"
rm -f "$D/.reconcile-inline"

# 7) Per-feature .a1-enforce flips enforce without the env var.
: > "$D/.a1-enforce"
out="$(run "$main_src")"; denies "$out" && ok ".a1-enforce file → enforce" || no ".a1-enforce did not enforce"
rm -f "$D/.a1-enforce"

# 8) Non-DevRites project (no ACTIVE) → allow.
T2="$(mktemp -d)"; mkdir -p "$T2/src"
out="$(CLAUDE_PROJECT_DIR="$T2" DEVRITES_A1_HOOK=enforce bash "$HOOK" <<<"{\"tool_name\":\"Edit\",\"tool_input\":{\"file_path\":\"$T2/src/a.ts\"}}")"
denies "$out" && no "blocked an edit in a non-DevRites project" || ok "no ACTIVE → allow"
rm -rf "$T2"

echo ""
[ "$fail" -eq 0 ] && echo "a1-guard-test: PASS" || echo "a1-guard-test: FAIL"
exit "$fail"
