---
name: rite-doctor
description: Doctor DevRites setup or index health. Use when workflow wiring is broken, the user asks to check DevRites, or says "reindex". Not for application bugs.
argument-hint: "[--code | --reindex]"
user-invocable: true
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- **Invocation and dispatch are different:** invoke means run a skill in this context; dispatch means start a fresh agent with `spawn_agent`, await it, and reconcile its result. Never describe inline skill work as a dispatch.
- Inspect the current `spawn_agent` role list. When the named `devrites-<role>` is exposed, dispatch it with `fork_turns="none"`; full-history forks inherit the parent type. The matching project contract is `.codex/agents/devrites-<role>.toml`.
- If a named role is not exposed, use generic `explorer` for every read-only role with `fork_turns="none"`. Tell it to read `.codex/agents/devrites-<role>.toml`, follow its `developer_instructions`, and execute the unchanged packet. Trusted `.codex/hooks.json` binds `agent_type=explorer` to the fail-closed reviewer read-only guard.
- For `devrites-slice-wright`, trusted `.codex/hooks.json` binds generic `worker` (`agent_type=worker`) to the active reconcile window and exact `.wright-allowlist`. Dispatch that worker with `fork_turns="none"`, tell it to read `.codex/agents/devrites-slice-wright.toml`, and execute the unchanged packet. Never create `.reconcile-inline` when this safe rung is available.
- A missing custom role is not evidence that spawning is unavailable. Only when the project hooks are unavailable or untrusted, no spawn primitive exists, or higher-priority policy rejects a safe spawn may the root run the documented discipline inline. Label it `independence: fallback`, never call it independent, create `.reconcile-inline` only for that path, and apply every fallback risk gate.
- Wait for every required fresh-context dispatch before reconciling or advancing. A backgrounded or lost result is incomplete.
- Codex project hooks are installed in `.codex/hooks.json`; declared-leaf hooks are scoped inside `.codex/agents/devrites-*.toml`. Review and trust them with `/hooks` before relying on hook enforcement.
- When this skill asks a HITL question via `AskUserQuestion`: Codex's equivalent (`request_user_input`) exists only in Plan mode. Outside Plan mode, render the option set as a plain numbered list in chat and **end the turn** so the human answers: NEVER silently pick an option yourself; auto-picking is AFK's contract, gated by the `.devrites/AFK` sentinel.


# $rite-doctor: health check

The on-demand deep report. The same checks run **silently at session start** (the orient
hook surfaces issues only when there are any); `$rite-doctor` runs them **verbosely**:
printing every check, pass or fail, so you can inspect health even when nothing is broken.
It covers version drift, Claude Code wiring, optional Codex mirrors/hooks, stale host artifacts, and missing install markers when those files are present. With `--code`, it also runs the read-only project code-health dashboard (`devrites-engine health`).
It also reports an in-progress git merge/rebase and points to `git-workflow.md`'s conflict
recovery playbook.

It is **read-only**: it never edits the workspace, never advances a phase, never blocks.

## Workflow
1. Run the diagnose core verbosely (resolve across install layouts):
   ```bash
   devrites-engine doctor; echo "doctor rc=$?"
   ```
1a. **Surface the learnings nudge**: point the user at `$rite-learn` when a pattern recurs across
   shipped features (read-only; silent when there's nothing to say):
   ```bash
   devrites-engine learnings nudge
   ```
1b. **Code-health dashboard (only when `$ARGUMENTS` includes `--code`).** Run the read-only check and surface the PASS/WARN/FAIL table; it appends `.devrites/health.jsonl` for trends and never blocks doctor:
   ```bash
   devrites-engine health; echo "health rc=$?"
   ```
1c. **Validate project extensions + overrides** (read-only: report, don't sync). A user rite/
   reviewer under `.devrites/extensions/` is held to the same schema as the shipped pack; a
   reviewer override under `.devrites/overrides/` may add emphasis but never relax a gate:
   ```bash
   devrites-engine extensions validate; echo "extensions rc=$?"
   devrites-engine overrides validate;  echo "overrides rc=$?"
   ```
   - **extensions rc=1:** an extension is malformed (missing frontmatter, empty, duplicate name).
     Fix the named file; once valid, the user mirrors it into the harness with
     `devrites-engine extensions sync`.
   - **overrides rc=1:** an override reads like it waives a gate. That is the one thing overrides
     must not do: hand the user the offending file to rewrite as added emphasis, not a waiver.
1d. **Refresh indexes only when `$ARGUMENTS` includes `--reindex`.** Load and execute
   `devrites-refresh-indexes`; report its synchronous refresh result, then continue the
   diagnostic. This changes optional indexes, never project source or DevRites state.
2. Report the result. **rc=0** → "DevRites healthy" + the `ok:` checks. **rc=1** → list each
   `issue:` line with the fix it names, then the single command that resolves the most urgent
   one (a stale ACTIVE → `rite use <slug>` or `$rite-status`; an orphaned gate →
   `$rite-resolve <qid>`; an incomplete install → reinstall). If the output includes
   `git-state: merge in progress` or `git-state: rebase in progress`, make the next action the
   `git-workflow.md` merge-conflict recovery playbook.
3. **Do not fix anything yourself:** doctor is diagnostic. Hand the user the fix command.
   **Completion:** exactly one highest-priority fix command is reported and no source/workspace file changed.

## Gotchas
- Read-only: never write the workspace or advance a phase (that's the lifecycle skills' job).
- It diagnoses **DevRites** health, not the user's application: code bugs go to
  `devrites-debug-recovery`; feature progress goes to `$rite-status`.
- Healthy is the common case; say so plainly and stop. Don't invent issues.

## Output
Reply-contract exception: workspace-less diagnostic. It does not render `devrites
progress`, but it follows the compact labels and one-next-action rule from
[`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md).

```
Done: DevRites health checked; <OK | n issues>.
Changed: workspace only
Evidence: devrites-engine doctor rc=<0|1>; health <skipped|PASS|WARN|FAIL>; reindex <skipped|result>; learnings nudge <summary|none>; extensions/overrides <ok|n issues>
Open: <none | issue count and top issue>
Next: <single command for the most urgent issue>
Record: not applicable
↻ Hygiene: /clear if stopping; /compact (doctor issue) if fixing now
```
