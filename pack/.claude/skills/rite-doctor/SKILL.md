---
name: rite-doctor
description: Diagnose DevRites install + workspace health on demand and print a full report — install integrity, a stale `.devrites/ACTIVE` pointer, a corrupt workspace, orphaned gates, and broken hook wiring. Use when the user says "rite doctor", "check my DevRites install", "is DevRites healthy", "diagnose devrites", or "why isn't the workflow picking up my feature". Not for debugging the user's own code (use `devrites-debug-recovery`) or for feature progress / next-action (use `/rite-status`).
argument-hint: ""
user-invocable: true
---

# /rite-doctor — health check

The on-demand deep report. The same checks run **silently at session start** (the orient
hook surfaces issues only when there are any); `/rite-doctor` runs them **verbosely** —
printing every check, pass or fail — so you can inspect health even when nothing is broken.

It is **read-only**: it never edits the workspace, never advances a phase, never blocks.

## Workflow
1. Run the diagnose core verbosely (resolve across install layouts):
   ```bash
   D=.claude/skills/devrites-lib/scripts/doctor.sh
   [ -f "$D" ] || D="${CLAUDE_SKILL_DIR:-}/../devrites-lib/scripts/doctor.sh"
   [ -f "$D" ] || D=pack/.claude/skills/devrites-lib/scripts/doctor.sh
   [ -f "$D" ] && bash "$D" --verbose; echo "doctor rc=$?"
   ```
2. Report the result. **rc=0** → "DevRites healthy" + the `ok:` checks. **rc=1** → list each
   `issue:` line with the fix it names, then the single command that resolves the most urgent
   one (a stale ACTIVE → `rite use <slug>` or `/rite-status`; an orphaned gate →
   `/rite-resolve <qid>`; an incomplete install → reinstall).
3. **Do not fix anything yourself** — doctor is diagnostic. Hand the user the fix command.

## Gotchas
- Read-only — never write the workspace or advance a phase (that's the lifecycle skills' job).
- It diagnoses **DevRites** health, not the user's application — code bugs go to
  `devrites-debug-recovery`; feature progress goes to `/rite-status`.
- Healthy is the common case; say so plainly and stop. Don't invent issues.

## Output
```
DevRites health: OK | N issue(s)
<for each issue: the problem + its fix>
Next: <single command for the most urgent issue, or "nothing to do">
```
