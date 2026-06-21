#!/usr/bin/env bash
# DevRites subagent PreToolUse hook (wired from devrites-slice-wright frontmatter) — fence the
# wright to its slice: deny Edit/Write to a path not listed in touched-files.md. Feature-scope
# enforced at the source, the in-subagent companion to the post-hoc reconcile.sh gate. FAIL-OPEN,
# OBSERVE by default (logs to .wright-scope.log); enable with DEVRITES_WRIGHT_SCOPE=enforce.
set -u

input="$(cat)"
case "$input" in *'"tool_name"'*) : ;; *) exit 0 ;; esac
root="${CLAUDE_PROJECT_DIR:-$PWD}"
[ -f "$root/.devrites/ACTIVE" ] || exit 0
slug="$(tr -d '[:space:]' < "$root/.devrites/ACTIVE" 2>/dev/null)"
[ -n "$slug" ] || exit 0
d="$root/.devrites/work/$slug"
tf="$d/touched-files.md"
[ -f "$tf" ] || exit 0

command -v node >/dev/null 2>&1 || exit 0
parsed="$(printf '%s' "$input" | node -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>{try{const j=JSON.parse(s);const ti=j.tool_input||{};process.stdout.write((j.tool_name||"")+""+(ti.file_path||ti.path||""))}catch(e){}})' 2>/dev/null)"
[ -n "$parsed" ] || exit 0
tool="${parsed%%$'\001'*}"; fpath="${parsed#*$'\001'}"
case "$tool" in Edit|Write|MultiEdit|NotebookEdit) : ;; *) exit 0 ;; esac
[ -n "$fpath" ] || exit 0
case "$fpath" in *.devrites/*) exit 0 ;; esac

rel="${fpath#"$root"/}"
if grep -qF "$rel" "$tf" 2>/dev/null || grep -qF "$fpath" "$tf" 2>/dev/null; then exit 0; fi

mode="${DEVRITES_WRIGHT_SCOPE:-observe}"
if [ "$mode" = "enforce" ]; then
  printf '%s\n' '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"DevRites scope: this path is not in touched-files.md. Build only the slice contract; if the slice genuinely needs this file, return an Escalation so the orchestrator routes it through the Spec Drift Guard — do not widen scope yourself. (devrites-wright-scope)"}}'
  exit 0
fi
printf '%s\tWOULD-BLOCK\t%s\n' "$(date '+%Y-%m-%dT%H:%M:%S' 2>/dev/null || echo '?')" "$fpath" >> "$d/.wright-scope.log" 2>/dev/null || true
exit 0
