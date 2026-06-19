#!/usr/bin/env bash
# DevRites PreToolUse hook — auto-approve the pack's READ-ONLY orientation/gate scripts so
# they stop triggering a permission prompt on every skill run.
#
# Safety model: FAIL-OPEN. This hook never denies and never blocks. It only ever EMITS an
# `allow` for a command that (a) runs one of five known read-only DevRites scripts AND
# (b) contains no dangerous/exfiltration tokens. On any doubt it prints nothing and exits 0,
# which leaves Claude Code's normal permission flow untouched. Mutating DevRites scripts
# (resolve.sh / tick-afk.sh / close-out.sh) are intentionally NOT covered — they still prompt.
set -u

input="$(cat)"

# Fast path: 99% of Bash commands have nothing to do with DevRites — skip without spawning node.
case "$input" in *devrites-lib/scripts/*) : ;; *) exit 0 ;; esac

# Extract the actual command string. Use node (ships with Claude Code) for correct JSON;
# if it's unavailable or parsing fails, fall through to the normal prompt.
command -v node >/dev/null 2>&1 || exit 0
cmd="$(printf '%s' "$input" | node -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>{try{const j=JSON.parse(s);process.stdout.write((j.tool_input&&j.tool_input.command)||"")}catch(e){}})' 2>/dev/null)"
[ -n "$cmd" ] || exit 0

# (a) Must actually invoke one of the five read-only helpers.
case "$cmd" in
  *devrites-lib/scripts/preamble.sh*|\
  *devrites-lib/scripts/progress.sh*|\
  *devrites-lib/scripts/readiness.sh*|\
  *devrites-lib/scripts/evidence-fresh.sh*|\
  *devrites-lib/scripts/check-acceptance.sh*) : ;;
  *) exit 0 ;;
esac

# (b) Defense in depth — refuse to auto-approve a compound command that could do harm or
# exfiltrate. The legit resolver snippets only use `[ -f ]`, var assignment, `bash "$VAR"`,
# `echo`, and && / ||. Anything below -> fall through to a normal prompt.
case "$cmd" in
  *" rm "*|*"rm -"*|*" mv "*|*" dd "*|*mkfs*|*" > /"*|*">/"*|*" >> "*|*">> "*|\
  *chmod*|*chown*|*sudo*|*curl*|*wget*|*" scp "*|*" ssh "*|*'$('*|*'`'*|\
  *eval*|*base64*|*xargs*|*"git push"*|*"git commit"*|*"git reset"*|*"git checkout"*|\
  *" npm "*|*npx\ *|*pnpm*|*" yarn "*|*" pip "*|*python\ -c*|*node\ -e*|*perl\ *|*ruby\ -e*) exit 0 ;;
esac

printf '%s\n' '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow","permissionDecisionReason":"DevRites read-only orientation/gate script — auto-approved by the devrites-allow hook"}}'
exit 0
