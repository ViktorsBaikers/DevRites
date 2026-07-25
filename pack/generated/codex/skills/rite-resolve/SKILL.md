---
name: rite-resolve
description: User-invoked resume verb for answering, dropping, or batch-resolving open `questions.md` gates.
argument-hint: "<qid> \"<answer>\"  |  --drop <qid> [\"<reason>\"]  |  --batch <path-to-yaml>"
user-invocable: true
disable-model-invocation: true
required-agent-roles: none
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- **Invocation and dispatch are different:** invoke means run a skill in this context; dispatch means start a fresh agent with `spawn_agent`, await it, and reconcile its result. Never describe inline skill work as a dispatch.
- On MultiAgent V2, call `spawn_agent` with the exact named `agent_type=devrites-<role>`, a unique `task_name`, and `fork_turns="none"`. Codex loads that role TOML's `developer_instructions` natively. Because V2 collaboration lifecycle calls bypass hooks, DevRites verifies the current durable parent/child rollout for the exact role, wait, completion, and non-empty delivered result.
- On MultiAgent V1, when the named role is not exposed, use generic `explorer` for a read-only role with `fork_turns="none"` and name exactly one `.codex/agents/devrites-<role>.toml` contract in the message. Trusted `.codex/hooks.json` injects that contract's exact `developer_instructions` and binds the child to the fail-closed reviewer read-only guard.
- On MultiAgent V1, `devrites-slice-wright` uses generic `worker` with `fork_turns="none"` and the exact role TOML named in the message. Trusted `.codex/hooks.json` binds it to the active reconcile window and `.wright-allowlist`; do not substitute `worker` for an exposed V2 named role.
- The invoked skill's `required-agent-roles` frontmatter arms the fail-closed Stop receipt. Every listed role must have a confirmed start, wait, and non-empty result in this turn.
- If any required named or generic agent dispatch is unavailable or rejected, stop for HITL. Never execute a DevRites specialist role in the root context.
- Wait for every required fresh-context dispatch before reconciling or advancing. A backgrounded or lost result is incomplete.
- Codex project hooks are installed in `.codex/hooks.json`; declared-leaf hooks are scoped inside `.codex/agents/devrites-*.toml`. Review and trust them with `/hooks` before relying on hook enforcement.
- When this skill asks a HITL question via `AskUserQuestion`: Codex's equivalent (`request_user_input`) exists only in Plan mode. Outside Plan mode, render the option set as a plain numbered list in chat and **end the turn** so the human answers: NEVER silently pick an option yourself; auto-picking is AFK's contract, gated by the `.devrites/AFK` sentinel.


# $rite-resolve: answer the human gate

`$rite-resolve` resumes an **async** human gate: a checkpoint that already paused and
**stopped the session** (an AFK blocking/escalating/irreversible queue, or a HITL pause
left unanswered), plus `--batch`. When `$rite-build` asks a question **inline**
via `AskUserQuestion` and the human is present, that pick resolves the gate **in place** through
the same `devrites-engine resolve` writer. You don't type `$rite-resolve` for it. For the async case this
skill takes the human's answer (or `--drop` / `--batch`), writes it to `questions.md`, updates
`state.md` (clears `Awaiting human`, sets `Status: running`), and recommends the next command.

It has one verb, one source of truth (`questions.md`), and one cursor (`state.md`). The
full AFK / HITL contract lives in
[`afk-hitl.md`](../devrites-lib/reference/standards/afk-hitl.md).

## Rules consulted (read on demand from `.agents/skills/devrites-lib/reference/standards/`)

Pull these via `Read` when shaping the resolve:

- `afk-hitl.md`: gate taxonomy, `questions.md` schema, AFK exception rules.
- `documentation.md`: record decisions and rationale where the answer changes scope.

## Operating rules

- **Requires an active workspace.** Read `.devrites/ACTIVE` first; if empty or the slug
  has no `questions.md`, **STOP** and tell the user to run `$rite-spec <feature>` first.
- **One mutation per call.** A single qid (or `--batch` file) per invocation; never
  silently coalesce multiple human decisions into one log entry.
- **Never overwrite an answered question.** If the qid's `status` is `answered` or
  `dropped`, refuse with the existing answer; ask the user to open a new qid that
  references the old one (the file is the audit trail).
- **If the answer materially changes scope, architecture, or acceptance**, route it
  through the Spec Drift Guard (`$rite-plan repair`) **after** writing the answer: do
  not modify `spec.md` / `plan.md` inside this skill.
- **The script is the source of truth.** Always invoke
  `devrites-engine resolve`. It keeps `questions.md` + `state.md` consistent and emits the
  next-action recommendation. The one `state.md` field this skill may write by hand is the
  unblocked slice's `Slice mode` (step 4, the named exception); everything else goes through
  the script, never by hand.
- **Human gates are for human-only decisions, not the agent's work.** A `questions.md` entry the
  human must answer is a genuine decision (a scope / design / risk call only the human can make),
  not a task the agent can do. If a question is really agent-doable ("should I write the
  test?", "go implement X"), do not record a human answer that returns the agent's work to it:
  flag the mis-tag and route it to the right skill (`$rite-build`, `$rite-plan unblock`,
  `devrites-debug-recovery`). The human resolves decisions; the agent does the work.

## Workflow

0. **Read `.agents/skills/devrites-lib/reference/standards/core.md`** (operating rules + persistence discipline) before
   touching the workspace.
   Then run `devrites-engine preamble` for deterministic workspace orientation.
1. **Parse arguments.** `$ARGUMENTS` is one of:
   - `<qid> "<answer>"`: answer the single open question.
   - `--drop <qid>` (optional `"<reason>"`): mark the question `dropped`; record
     the reason inline.
   - `--batch <path-to-yaml>`: bulk resolve, one entry per qid (see
     [`reference/answer-protocol.md`](reference/answer-protocol.md) for the batch
     format).
2. **Load context.** Read `state.md`, `questions.md`, and the relevant slice from
   `tasks.md`. Confirm the qid is `status: open`. If `state.md` `Status` is not
   `awaiting_human` and the question's `gate` is `blocking`, surface the inconsistency
   before proceeding (don't auto-repair: flag it).
3. **Apply explicit consent.** Supplying `<qid> "<answer>"`, `--drop`, or `--batch` is the
   user's explicit consent for this local workspace mutation. Echo the qid, answer/drop,
   and slice being unblocked, then continue immediately; do not ask the user to confirm the
   command they just typed.
4. **Mutate.** Run `devrites-engine resolve` with the same
   arguments. The script:
   - flips the qid's `status` to `answered` / `dropped` and stamps `answered_at` + `answer`;
   - if the qid is in `state.md`'s `Awaiting human` block (single-question pause), clears
     that block and sets `Status: running`;
   - appends a `Log` line to `state.md`.

   On resume, also clear or update the unblocked slice's `Slice mode` in `state.md`: if
   the answer lets the slice proceed, drop the pause-time `Slice mode` so `$rite-build`
   re-derives it on the next selection; if the answer re-shapes how the slice should be
   built, set `Slice mode` to match.
5. **Post-resolve hand-off.** If the answer changes product behavior or acceptance →
   recommend `$rite-plan repair`. Otherwise → recommend the slice's natural next action
   (typically `$rite-build` for the slice that was awaiting).
   **Completion:** the resolved state contains exactly one next command.
6. **STOP.** This skill does not run `$rite-build` itself: the user re-enters the
   workflow explicitly.

> **Mid-flight discipline.** Don't edit `spec.md` / `plan.md` to "incorporate" the
> answer. That's `$rite-plan repair`. Don't silently retry a build after the answer
> lands: the user types the next command. Don't merge two open questions into one
> answered entry: each question is independently auditable.

## Output

**Progress first**: run `devrites-engine progress`, then use the shared completion reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md)).
Default success shape:
```
Done: resolved <qid> for <gate> at <slice/phase>.
Changed: questions.md, state.md, decisions.md <updated|n/a>
Evidence: not applicable; answer persisted before resume
Open: <none | remaining questions | plan repair needed>
Next: <single recommended command>
Record: .devrites/work/<slug>/questions.md
↻ Hygiene: no /clear needed; the answer is persisted
```
