---
name: rite-status
description: User-invoked read-only active-feature report: phase, active slice, next action, evidence, open questions, drift, risks, and handoff readiness.
argument-hint: "[feature-slug]"
user-invocable: true
disable-model-invocation: true
required-agent-roles: none
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Installed `.agents/` mirrors may be Git-ignored. If a repository-aware file tool refuses an ignored path, read it with a native filesystem command instead; a tool refusal is not a completed task.
- For automatic Engram calls, omit optional `project` and `session_id` unless an exact value came from Engram or repository configuration. Never derive either from `task_name`, a run ID, directory name, or normalized slug. Call `mem_session_summary` without them by default; on `unknown_session` or `unknown_project`, retry once with both optional fields omitted. If auto-detection is ambiguous, ask the user instead of guessing.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- **Invocation and dispatch are different:** invoke means run a skill in this context; dispatch means start a fresh agent with `spawn_agent`, await it, and reconcile its result. Never describe inline skill work as a dispatch.
- On MultiAgent V2, call `spawn_agent` with the exact named `agent_type=devrites-<role>`, a unique `task_name`, and `fork_turns="none"`. A missing visible `agent_type` field is still V2—not capability loss, V1, or HITL—so send it anyway. If the named call rejects it, stop before any generic/default spawn. Codex loads the role TOML's `developer_instructions` natively; DevRites verifies the durable rollout, wait, completion, and delivered result.
- Only after the runtime explicitly identifies MultiAgent V1, use generic `explorer` for a read-only role with `fork_turns="none"` and name exactly one `.codex/agents/devrites-<role>.toml` contract in the message. Trusted `.codex/hooks.json` injects that contract's exact `developer_instructions` and binds the child to the fail-closed reviewer read-only guard.
- On explicitly identified MultiAgent V1, `devrites-slice-wright` uses generic `worker` with `fork_turns="none"` and the exact role TOML named in the message. Trusted `.codex/hooks.json` binds it to the active reconcile window and `.wright-allowlist`.
- The invoked skill's `required-agent-roles` frontmatter arms the fail-closed Stop receipt. Every listed role must have a confirmed start, wait, and non-empty result in this turn.
- If the required dispatch for the explicitly identified runtime is unavailable or rejected, stop for HITL. Never switch runtime lanes. Never execute a DevRites specialist role in the root context.
- Wait for every required fresh-context dispatch before reconciling or advancing. A backgrounded or lost result is incomplete.
- Codex project hooks are installed in `.codex/hooks.json`; declared-leaf hooks are scoped inside `.codex/agents/devrites-*.toml`. Review and trust them with `/hooks` before relying on hook enforcement.
- When this skill asks a HITL question via `AskUserQuestion`: Codex's equivalent (`request_user_input`) exists only in Plan mode. Outside Plan mode, render the option set as a plain numbered list in chat and **end the turn** so the human answers: NEVER silently pick an option yourself; auto-picking is AFK's contract, gated by the `.devrites/AFK` sentinel.


# $rite-status: active feature status

Read-only. Report where the active feature stands. **Do not run any phase.**

## Load state

Run the shared DevRites preamble plus the machine snapshot (default slug = `.devrites/ACTIVE`):

```bash
devrites-engine preamble [feature-slug]
devrites-engine snapshot [feature-slug]
```

The preamble is the human-readable orientation digest: it prints `state.md`, artifacts
present, run mode, and open-question counts. The snapshot is the canonical
cross-harness machine contract: phase, section completeness, current slice,
evidence/drift/review freshness, capability readiness, and **both** host-specific
next-command forms under `nextCommands` (`claude` = `/rite-*`, `codex` = `$rite-*`).
Use the command form for the current host; do not invent or hardcode one.
The `!`-prefix dynamic-context-injection idiom is **not** used here so the skill
stays portable across harnesses; the `devrites` binary is the cross-harness mechanism.

## What to output

If no workspace: tell the user to run `$rite-spec <feature>` to start. Stop.

Otherwise summarize from the loaded state, concisely:

1. **Feature** + one-line objective (from `brief.md`).
2. **Phase**, **active slice** + its slice mode (from `state.md`). **Run mode**
   (`afk` / `hitl`) is derived from `.devrites/AFK` presence (the preamble
   already does this), not from a `state.md` field: present = `afk`, absent = `hitl`.
3. **Status:** `running` / `awaiting_human` / `blocked` / `done`. If
   `awaiting_human`, render the `Awaiting human` block from `state.md` (qid, gate,
   question, proposed, raised_at, blocking_slices) and instruct
   `$rite-resolve <qid> "<answer>"`.
4. **Next action:** the single recommended next command from `snapshot.nextCommands`
   for the current host.
5. **Evidence:** what's proven vs unproven (from `evidence.md` /
   `browser-evidence.md`).
6. **Open questions:** count by gate (from `questions.md`:
   `n open: x blocking · y validating · z advisory`) + the blocking qids by id and one-line question.
7. **Unresolved drift** (`drift.md`).
8. **Risks** (from `state.md` / `spec.md`); if `.devrites/principles.md` exists, note the count
   of declared invariants and flag any exception in its register that is past its review/removal
   trigger (a stale exception is a standing risk). Omit if there's no principles file.
9. **Ready for handoff?:** see section below.

Flag anything blocking: unresolved drift, failing evidence, `Status: awaiting_human`,
or open questions that change product behavior. End with the recommended next command
and nothing else.

## Ready for handoff?

After the summary items (1-8), render the handoff readiness section (item 9 above) last:
the canonical output order is: progress footer → summary items 1-8 → handoff readiness.
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
- Read-only: never advance a phase or write to the workspace; that's the phase skills' job.
- Report from the `.devrites/` files, not chat memory (it's gone after `/clear`).
- Give the single next command, not a menu of options: an ambiguous next-action fails the handoff check.

## Output format

**Progress first**: `$rite-status` *is* the status meter: render it by running
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
`.agents/skills/devrites-lib/reference/standards/core.md`. "Persistence before stopping"). Only then
move to the phase's next action.
