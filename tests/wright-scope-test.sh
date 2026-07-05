#!/usr/bin/env bash
# wright-scope-test.sh — unit-test the DevRites slice-wright scope hook.
set -u
ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"
HOOK="$ROOT/pack/.claude/hooks/devrites-wright-scope.sh"
fail=0
ok() { printf '  ok: %s\n' "$*"; }
no() { printf '  FAIL: %s\n' "$*"; fail=1; }

T="$(mktemp -d)"
trap 'rm -rf "$T"' EXIT
echo "== wright-scope-test (target: $T) =="

mkdir -p "$T/.devrites/work/demo" "$T/src"
printf 'demo\n' > "$T/.devrites/ACTIVE"
D="$T/.devrites/work/demo"
printf 'src/allowed.ts\n' > "$D/touched-files.md"

run() {
  local payload="$1" enforce="${2:-}"
  if [ "$enforce" = "enforce" ]; then
    CLAUDE_PROJECT_DIR="$T" DEVRITES_WRIGHT_SCOPE=enforce bash "$HOOK" <<<"$payload"
  else
    CLAUDE_PROJECT_DIR="$T" bash "$HOOK" <<<"$payload"
  fi
}
denies() { printf '%s' "$1" | grep -q '"permissionDecision":"deny"'; }

allowed='{"tool_name":"Edit","tool_input":{"file_path":"'"$T"'/src/allowed.ts"}}'
blocked='{"tool_name":"Write","tool_input":{"file_path":"'"$T"'/src/other.ts"}}'
patch_allowed='{"tool_name":"apply_patch","tool_input":{"command":"*** Begin Patch\n*** Update File: src/allowed.ts\n@@\n-old\n+new\n*** End Patch\n"}}'
patch_blocked='{"tool_name":"apply_patch","tool_input":{"command":"*** Begin Patch\n*** Update File: src/other.ts\n@@\n-old\n+new\n*** End Patch\n"}}'
patch_move_blocked='{"tool_name":"apply_patch","tool_input":{"command":"*** Begin Patch\n*** Update File: .devrites/work/demo/state.md\n*** Move to: src/moved.ts\n@@\n-old\n+new\n*** End Patch\n"}}'
book='{"tool_name":"Write","tool_input":{"file_path":"'"$T"'/.devrites/work/demo/state.md"}}'

out="$(run "$allowed" enforce)"; denies "$out" && no "blocked touched file" || ok "touched file → allow"
out="$(run "$blocked" enforce)"; denies "$out" && ok "enforce → deny file outside touched-files" || no "did not deny file outside touched-files"
out="$(run "$patch_allowed" enforce)"; denies "$out" && no "blocked touched apply_patch" || ok "touched apply_patch → allow"
out="$(run "$patch_blocked" enforce)"; denies "$out" && ok "enforce → deny apply_patch outside touched-files" || no "did not deny apply_patch outside touched-files"
out="$(run "$patch_move_blocked" enforce)"; denies "$out" && ok "enforce → deny apply_patch move outside touched-files" || no "did not deny apply_patch move outside touched-files"
out="$(run "$book" enforce)"; denies "$out" && no "blocked .devrites bookkeeping" || ok ".devrites bookkeeping → allow"

out="$(run "$blocked")"
denies "$out" && no "observe mode denied" || ok "observe → allow"
[ -f "$D/.wright-scope.log" ] && grep -q "WOULD-BLOCK" "$D/.wright-scope.log" && ok "observe logged WOULD-BLOCK" || no "observe did not log"

out="$(CLAUDE_PROJECT_DIR="$T" DEVRITES_WRIGHT_AGENT_REQUIRED=1 DEVRITES_WRIGHT_SCOPE=enforce bash "$HOOK" <<<"$blocked")"
denies "$out" && no "agent-required mode blocked payload without agent_type" || ok "agent-required mode ignores non-wright payload"
out="$(CLAUDE_PROJECT_DIR="$T" DEVRITES_WRIGHT_AGENT_REQUIRED=1 DEVRITES_WRIGHT_SCOPE=enforce bash "$HOOK" <<<"{\"tool_name\":\"Write\",\"agent_type\":\"devrites-code-reviewer\",\"tool_input\":{\"file_path\":\"$T/src/other.ts\"}}")"
denies "$out" && no "agent-required mode blocked non-wright agent_type" || ok "agent-required mode ignores non-wright agent_type"

wright='{"tool_name":"Write","agent_type":"devrites-slice-wright","tool_input":{"file_path":"'"$T"'/src/other.ts"}}'
out="$(CLAUDE_PROJECT_DIR="$T" DEVRITES_WRIGHT_AGENT_REQUIRED=1 DEVRITES_WRIGHT_SCOPE=enforce bash "$HOOK" <<<"$wright")"
denies "$out" && ok "agent-required mode applies to slice-wright" || no "agent-required mode did not apply to slice-wright"

echo ""
[ "$fail" -eq 0 ] && echo "wright-scope-test: PASS" || echo "wright-scope-test: FAIL"
exit "$fail"
