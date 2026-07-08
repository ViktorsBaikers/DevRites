---
name: rite-doctor
description: Diagnose DevRites install + workspace health on demand and print a full report — install integrity, Codex/Claude hook wiring, Codex support mirrors, a stale `.devrites/ACTIVE` pointer, a corrupt workspace, and orphaned gates. Use when the user says "rite doctor", "check my DevRites install", or "why isn't the workflow picking up my feature". Not for debugging the user's own code (use `devrites-debug-recovery`) or for feature progress / next-action (use `/rite-status`).
argument-hint: ""
user-invocable: true
---

# /rite-doctor — health check

The on-demand deep report. The same checks run **silently at session start** (the orient
hook surfaces issues only when there are any); `/rite-doctor` runs them **verbosely** —
printing every check, pass or fail — so you can inspect health even when nothing is broken.
It covers both Claude Code wiring and optional Codex mirrors/hooks when those files are present.

It is **read-only**: it never edits the workspace, never advances a phase, never blocks.

## Workflow
1. Run the diagnose core verbosely (resolve across install layouts):
   ```bash
   devrites-engine doctor --verbose; echo "doctor rc=$?"
   ```
1a. **Surface the learnings nudge** — point the user at `/rite-learn` when a pattern recurs across
   shipped features (read-only; silent when there's nothing to say):
   ```bash
   devrites-engine learnings nudge
   ```
1b. **Validate project extensions + overrides** (read-only — report, don't sync). A user rite/
   reviewer under `.devrites/extensions/` is held to the same schema as the shipped pack; a
   reviewer override under `.devrites/overrides/` may add emphasis but never relax a gate:
   ```bash
   devrites-engine extensions validate; echo "extensions rc=$?"
   devrites-engine overrides validate;  echo "overrides rc=$?"
   ```
   - **extensions rc=1** — an extension is malformed (missing frontmatter, empty, duplicate name).
     Fix the named file; once valid, the user mirrors it into the harness with
     `devrites-engine extensions sync`.
   - **overrides rc=1** — an override reads like it waives a gate. That is the one thing overrides
     must not do — hand the user the offending file to rewrite as added emphasis, not a waiver.
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
Reply-contract exception: workspace-less diagnostic. It does not render `devrites
progress`, but it follows the compact labels and one-next-action rule from
[`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md).

```
Done: DevRites health checked; <OK | n issues>.
Changed: workspace only
Evidence: devrites-engine doctor --verbose rc=<0|1>; learnings nudge <summary|none>; extensions/overrides <ok|n issues>
Open: <none | issue count and top issue>
Next: <single command for the most urgent issue>
Record: not applicable
↻ Hygiene: /clear if stopping; /compact (doctor issue) if fixing now
```
