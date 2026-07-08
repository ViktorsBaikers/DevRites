---
name: rite-status
description: Read-only report on the active feature — phase, active slice, next action, evidence, open questions, drift, risks, and handoff readiness. Not for invoking phases or for the menu (use `$rite`).
argument-hint: "[feature-slug]"
user-invocable: true
disable-model-invocation: true
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- When this skill asks for a DevRites specialist or writer agent, **explicitly** spawn the matching Codex custom agent from `.codex/agents/devrites-*.toml` through Codex subagents (`spawn_agent`), then wait for its result and reconcile it as the skill instructs. Do not do the review inline just because the instruction to spawn is embedded here — Codex under-fires embedded spawn/skill instructions (openai/codex #23496), so treat the spawn as required, not optional.
- The independence of a fresh-context subagent is the point. If Codex genuinely cannot spawn subagents in the current surface, run the documented inline fallback and **label the result an inline fallback, not an independent review** — an inline pass shares the calling context and is weaker evidence.
- Codex project hooks are installed in `.codex/hooks.json`. Review and trust them with `/hooks` before relying on hook enforcement.


# $rite-status — active feature status

Read-only. Report where the active feature stands. **Do not run any phase.**

## Load state

Run the shared DevRites preamble (default slug = `.devrites/ACTIVE`):

```bash
devrites-engine preamble [feature-slug]
```

The preamble prints the active workspace's `state.md` and the list of artifacts
present. The `!`-prefix dynamic-context-injection idiom is **not** used here so
the skill stays portable across harnesses; the `devrites` binary is the cross-harness
mechanism.

## What to output

If no workspace: tell the user to run `$rite-spec <feature>` to start. Stop.

Otherwise summarize from the loaded state, concisely:

1. **Feature** + one-line objective (from `brief.md`).
2. **Phase**, **active slice** + its slice mode (from `state.md`). **Run mode**
   (`afk` / `hitl`) is derived from `.devrites/AFK` presence (the preamble
   already does this), not from a `state.md` field — present = `afk`, absent = `hitl`.
3. **Status** — `running` / `awaiting_human` / `blocked` / `done`. If
   `awaiting_human`, render the `Awaiting human` block from `state.md` (qid, gate,
   question, proposed, raised_at, blocking_slices) and instruct
   `$rite-resolve <qid> "<answer>"`.
4. **Next action** — the single recommended next command.
5. **Evidence** — what's proven vs unproven (from `evidence.md` /
   `browser-evidence.md`).
6. **Open questions** — count by gate (from `questions.md`:
   `n open: x blocking · y validating · z advisory`) + the blocking qids by id and one-line question.
7. **Unresolved drift** (`drift.md`).
8. **Risks** (from `state.md` / `spec.md`); if `.devrites/principles.md` exists, note the count
   of declared invariants and flag any exception in its register that is past its review/removal
   trigger (a stale exception is a standing risk). Omit if there's no principles file.
9. **Ready for handoff?** — see section below.

Flag anything blocking: unresolved drift, failing evidence, `Status: awaiting_human`,
or open questions that change product behavior. End with the recommended next command
and nothing else.

## Ready for handoff?

After the summary items (1–8), render the handoff readiness section (item 9 above) last —
the canonical output order is: progress footer → summary items 1–8 → handoff readiness.
A fresh agent
or a future session should be able to pick the work up from the workspace
**alone**, with zero chat context. Check:

- [ ] `state.md` has a **single** next-action command (not a list of options).
- [ ] `questions.md` lists every unresolved question raised in this or any
      prior session.
- [ ] `decisions.md` records the rationale for any non-obvious choice
      ("why this, not the obvious alternative").
- [ ] `assumptions.md` lists every load-bearing assumption that wasn't
      confirmed by code or by the user.
- [ ] `drift.md` shows current resolution status for every drift event
      (open / asked-user / repaired).
- [ ] `evidence.md` is current for the active phase (no claims without
      recorded commands + output).

If any item above fails, suggest `$rite-handoff` to compact chat-only
context into the workspace before the session ends.

## Gotchas
- Read-only — never advance a phase or write to the workspace; that's the phase skills' job.
- Report from the `.devrites/` files, not chat memory (it's gone after `/clear`).
- Give the single next command, not a menu of options — an ambiguous next-action fails the handoff check.

## Output format

**Progress first** — `$rite-status` *is* the status meter: render it by running
`devrites-engine progress`. Then use the shared completion reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md))
as a read-only status variant:
```
Done: status read for <slug>; phase <phase> and active slice <slice|n/a>.
Changed: workspace only
Evidence: <current proof summary | not applicable yet>
Open: <none | questions n | drift n | blockers n | handoff gaps n>
Next: <single recommended command>
Record: .devrites/work/<slug>/state.md
↻ Hygiene: <handoff ready | $rite-handoff before /clear>
```

If gaps exist, the recommended next command is to **persist them first** (see
`.agents/skills/devrites-lib/reference/standards/core.md` — "Persistence before stopping"). Only then
move to the phase's next action.
