#!/usr/bin/env bash
# DevRites subagent PreToolUse hook (wired from reviewer-agent frontmatter) — keep the
# read-only reviewers read-only: deny a Bash command that could mutate the tree or exfiltrate.
# A fresh-context reviewer reads untrusted source; a silent write path is a prompt-injection
# surface (rules/security.md). FAIL-OPEN, OBSERVE by default; DEVRITES_REVIEWER_RO=enforce.
set -u

input="$(cat)"
case "$input" in *'"tool_name"'*) : ;; *) exit 0 ;; esac
command -v node >/dev/null 2>&1 || exit 0
parsed="$(printf '%s' "$input" | node -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>{try{const j=JSON.parse(s);const ti=j.tool_input||{};process.stdout.write((j.tool_name||"")+""+(ti.command||""))}catch(e){}})' 2>/dev/null)"
[ -n "$parsed" ] || exit 0
tool="${parsed%%$'\001'*}"; cmd="${parsed#*$'\001'}"
[ "$tool" = "Bash" ] || exit 0
[ -n "$cmd" ] || exit 0

# Mutating / exfiltration tokens — reviewers only ever need read-only inspection.
printf '%s' "$cmd" | grep -qE '>>|[^0-9 ]>[^>&]|[[:space:]]-i([[:space:]]|$)|sed[[:space:]]+-i|\brm\b|\bmv\b|\btee\b|\btruncate\b|\bdd\b|chmod|chown|git[[:space:]]+(add|commit|push|reset|checkout|rm|mv|stash|tag|apply)|npm[[:space:]]+(install|i|publish|run)|pnpm[[:space:]]+(install|add)|yarn[[:space:]]+add|pip[[:space:]]+install|curl[[:space:]].*[[:space:]]-o\b|\bwget\b|\bscp\b|\bssh\b|\bnc\b' || exit 0

mode="${DEVRITES_REVIEWER_RO:-observe}"
if [ "$mode" = "enforce" ]; then
  printf '%s\n' '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"DevRites: reviewers are read-only. This Bash command can mutate or exfiltrate; inspect with Read/Grep/Glob and return findings — do not modify the tree or reach the network. (devrites-reviewer-readonly)"}}'
  exit 0
fi
root="${CLAUDE_PROJECT_DIR:-$PWD}"
slug="$(tr -d '[:space:]' < "$root/.devrites/ACTIVE" 2>/dev/null || true)"
[ -n "$slug" ] && printf '%s\tWOULD-BLOCK\t%s\n' "$(date '+%Y-%m-%dT%H:%M:%S' 2>/dev/null || echo '?')" "$(printf '%s' "$cmd" | head -c 80)" >> "$root/.devrites/work/$slug/.reviewer-ro.log" 2>/dev/null || true
exit 0
