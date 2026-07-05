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
tool="$(printf '%s' "$input" | node -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>{try{process.stdout.write(JSON.parse(s).tool_name||"")}catch(e){}})' 2>/dev/null)"
fpath="$(printf '%s' "$input" | node -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>{try{const ti=JSON.parse(s).tool_input||{};process.stdout.write(ti.file_path||ti.path||"")}catch(e){}})' 2>/dev/null)"
agent_type="$(printf '%s' "$input" | node -e 'let s="";process.stdin.on("data",d=>s+=d).on("end",()=>{try{process.stdout.write(JSON.parse(s).agent_type||"")}catch(e){}})' 2>/dev/null)"
[ -n "$tool" ] || exit 0
if [ "${DEVRITES_WRIGHT_AGENT_REQUIRED:-0}" = "1" ] && [ "$agent_type" != "devrites-slice-wright" ]; then
  exit 0
fi
case "$tool" in Edit|Write|MultiEdit|NotebookEdit|apply_patch) : ;; *) exit 0 ;; esac

bad_paths=""
if [ "$tool" = "apply_patch" ]; then
  bad_paths="$(printf '%s' "$input" | TF="$tf" ROOT="$root" node -e '
    const fs = require("fs");
    let s = ""; process.stdin.on("data", d => s += d).on("end", () => {
      try {
        const root = process.env.ROOT || "";
        const tf = process.env.TF || "";
        const touched = fs.existsSync(tf) ? fs.readFileSync(tf, "utf8") : "";
        const cmd = (JSON.parse(s).tool_input || {}).command || "";
        const paths = [...cmd.matchAll(/^\*\*\* (?:(?:Add|Update|Delete) File|Move to): (.+)$/gm)].map(m => m[1]);
        const bad = paths.filter(p => {
          if (p.startsWith(".devrites/") || p.includes("/.devrites/")) return false;
          const rel = p.startsWith(`${root}/`) ? p.slice(root.length + 1) : p;
          return !touched.includes(rel) && !touched.includes(p);
        });
        process.stdout.write(bad.join(", "));
      } catch {}
    });
  ' 2>/dev/null)"
else
  [ -n "$fpath" ] || exit 0
  case "$fpath" in *.devrites/*|.devrites/*) exit 0 ;; esac
  rel="${fpath#"$root"/}"
  if ! grep -qF "$rel" "$tf" 2>/dev/null && ! grep -qF "$fpath" "$tf" 2>/dev/null; then
    bad_paths="$fpath"
  fi
fi
[ -n "$bad_paths" ] || exit 0

mode="${DEVRITES_WRIGHT_SCOPE:-observe}"
if [ "$mode" = "enforce" ]; then
  printf '%s\n' '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"deny","permissionDecisionReason":"DevRites scope: this path is not in touched-files.md. Build only the slice contract; if the slice genuinely needs this file, return an Escalation so the orchestrator routes it through the Spec Drift Guard — do not widen scope yourself. (devrites-wright-scope)"}}'
  exit 0
fi
printf '%s\tWOULD-BLOCK\t%s\n' "$(date '+%Y-%m-%dT%H:%M:%S' 2>/dev/null || echo '?')" "$bad_paths" >> "$d/.wright-scope.log" 2>/dev/null || true
exit 0
