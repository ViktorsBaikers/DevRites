---
name: rite-prototype
description: User-invoked throwaway prototype for one logic or UI design question; delete or absorb after the answer.
user-invocable: true
disable-model-invocation: true
argument-hint: "[the question the prototype is answering]"
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


# $rite-prototype: answer one question with throwaway code

A prototype is throwaway code for exactly one question. Choose its shape from that
question.

## 0. Read core rules

Read `.agents/skills/devrites-lib/reference/standards/core.md` first: the operating rules and the "capture the answer"
persistence discipline apply even to throwaway code. The other rule files load on demand.

## 1. Name the question

If the user did not give the question explicitly, ask **one** short question to pin it
down. Examples that count as "the question":

- "Does this reducer correctly model add / undo / re-add of the same item?"
- "Which of these three settings layouts feels right?"
- "Will the proposed state machine deadlock when X overlaps Y?"

A vague request such as "explore this feature" is not enough. Ask once for a specific
question.

## 2. Pick the branch

Read the user's prompt and surrounding code, then choose one branch:

| Question shape | Branch | Artifact |
|---|---|---|
| "Does this **logic / state / data model** behave right?" | **Logic** | Smallest runnable harness (terminal script / REPL / mini node or python file) that pushes the model through the hard cases. |
| "What should this **look like / which UX wins**?" | **UI** | 2-4 **radically different** UI variations on a single route, switchable via a query param + a small floating selector bar. |

If genuinely ambiguous and the user is AFK: pick by the surrounding code (backend module
→ Logic; page / component → UI) and state the assumption in a comment at the top.

**Variations must differ in structure.** Colour or padding changes alone do not answer
a design question. Give each variation a different information architecture or
interaction model.

## 3. Rules that apply to both branches

1. **Mark it as throwaway.** Place the prototype near its eventual use, but use a name
   that clearly identifies it: `prototype/`, `__prototype__`, `.scratch/`, etc. Obey the project's
   existing routing / module convention; don't invent a new top-level structure.
2. **Provide one run command.** Use the project's existing task runner: `pnpm
   <name>`, `python <path>`, or `bun <path>`.
3. **No persistence by default.** State lives in memory. Persistence is usually the
   thing the prototype is *checking*, not a dependency. If the question explicitly
   involves a DB, hit a scratch DB or a local file with a name like
   `PROTOTYPE — wipe me`.
4. **Skip polish.** Add no tests, abstractions, or error handling beyond what is needed
   to run and answer the question.
5. **Surface state.** After every action (Logic) or on every variant switch (UI), print
   or render the full relevant state so the user can see what changed.
6. **Delete or absorb when done.** After answering the question, delete the prototype
   or fold the validated decision into production code.

## 4. Capture the answer

The answer is the only durable output. Before declaring the prototype done, write
one of:

- An entry in the active feature's `decisions.md` (`.devrites/work/<slug>/decisions.md`)
  with the question and the answer.
- A `NOTES.md` next to the prototype if no active feature exists.

If the user is present, ask for the verdict directly. In AFK, the prototype remains
incomplete until the verdict is filled in. A blank `VERDICT: ___` is not an answer.
Mark it **not yet complete**. Queue the question and do not delete the prototype until
a later pass records the verdict.

## Where this slots in

`$rite-prototype` is a scoped detour, not a separate phase. Typical uses:

- **During `$rite-clarify` (between spec and define):** a product/constraint question is
  undecidable on paper; prototype it, capture the answer, return to `$rite-clarify`.
- **Mid-build:** `$rite-build` hit a state-model ambiguity that the spec doesn't
  resolve. Drop into a Logic prototype, capture the answer, return.

After the prototype answers the question and the answer is recorded, return to the
calling phase. The prototype itself does **not** ship.

## Output

Reply-contract exception: scoped prototype detour. Run `devrites-engine progress` only when
an active workspace exists; otherwise skip it. Use the compact labels from the shared
completion reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md)).

```
Done: prototype answered <question>.
Changed: <prototype path | deleted after absorption>; decisions.md or NOTES.md
Evidence: run command <cmd -> result>; verdict <one line>
Open: <none | non-blocking delete/absorb follow-up>
Next: <single command returning to the calling phase>
Record: <.devrites/work/<slug>/decisions.md | NOTES.md path>
↻ Hygiene: /clear after the answer is recorded; delete or absorb throwaway code before shipping
```

If the verdict is missing, use the shared `Awaiting human` form. The prototype is
not `Done`, and `Resume:` points to the queued verdict question.
