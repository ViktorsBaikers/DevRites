---
name: rite-status
description: Read-only report on the active feature — phase, active slice, next action, evidence, open questions, drift, risks, handoff readiness. Use when the user says "where am I", "what's next", "status", "is this ready to ship", "show the active feature". Not for invoking phases or for the menu (use `/rite`).
argument-hint: "[feature-slug]"
user-invocable: true
disable-model-invocation: true
---

# /rite-status — active feature status

Read-only. Report where the active feature stands. **Do not run any phase.**

## Load state

Run this snippet (the shared DevRites preamble — one canonical command that
resolves the install layouts: installed `.claude/` → plugin via `${CLAUDE_SKILL_DIR}`
→ repo `pack/`, with a graceful fallback to reading `state.md`; default slug =
`.devrites/ACTIVE`):

```bash
P=.claude/skills/devrites-lib/scripts/preamble.sh
[ -f "$P" ] || P="${CLAUDE_SKILL_DIR:-}/../devrites-lib/scripts/preamble.sh"
[ -f "$P" ] || P=pack/.claude/skills/devrites-lib/scripts/preamble.sh
[ -f "$P" ] && bash "$P" [feature-slug] || echo "(orientation preamble unavailable on this install — read state.md directly to orient)"
```

The preamble prints the active workspace's `state.md` and the list of artifacts
present. The `!`-prefix dynamic-context-injection idiom is **not** used here so
the skill stays portable across harnesses; the script is the cross-harness
mechanism.

## What to output

If no workspace: tell the user to run `/rite-spec <feature>` to start. Stop.

Otherwise summarize from the loaded state, concisely:

1. **Feature** + one-line objective (from `brief.md`).
2. **Phase**, **active slice** + its slice mode (from `state.md`). **Run mode**
   (`afk` / `hitl`) is derived from `.devrites/AFK` presence (the preamble
   already does this), not from a `state.md` field — present = `afk`, absent = `hitl`.
3. **Status** — `running` / `awaiting_human` / `blocked` / `done`. If
   `awaiting_human`, render the `Awaiting human` block from `state.md` (qid, gate,
   question, proposed, raised_at, blocking_slices) and instruct
   `/rite-resolve <qid> "<answer>"`.
4. **Next action** — the single recommended next command.
5. **Evidence** — what's proven vs unproven (from `evidence.md` /
   `browser-evidence.md`).
6. **Open questions** — count by gate (from `questions.md`:
   `n open: x blocking · y validating · z advisory`) + the blocking qids by id and one-line question.
7. **Unresolved drift** (`drift.md`).
8. **Risks** (from `state.md` / `spec.md`).
9. **Ready for handoff?** — see section below.

Flag anything blocking: unresolved drift, failing evidence, `Status: awaiting_human`,
or open questions that change product behavior. End with the recommended next command
and nothing else.

## Ready for handoff?

After items 1–6, render an explicit handoff readiness section. A fresh agent
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

If any item above fails, suggest `/rite-handoff` to compact chat-only
context into the workspace before the session ends.

## Gotchas
- Read-only — never advance a phase or write to the workspace; that's the phase skills' job.
- Report from the `.devrites/` files, not chat memory (it's gone after `/clear`).
- Give the single next command, not a menu of options — an ambiguous next-action fails the handoff check.

## Output format

**Footer first** — `/rite-status` *is* the status meter: render it by running the progress footer (`progress.sh`, resolved like the step-0 preamble — canonical snippet in `devrites-lib/SKILL.md`). The slice meter + flow ribbon answer "where am I / how much is left". Then the handoff-readiness check:
```
Ready for handoff: yes / partial (n gaps) / no

Gaps (if any):
- state.md "next action" lists 2 options, not 1
- 3 open questions raised in conversation not yet in questions.md
- ...
```

If gaps exist, the recommended next command is to **persist them first** (see
`.claude/rules/core.md` — "Persistence before stopping"). Only then
move to the phase's next action.
