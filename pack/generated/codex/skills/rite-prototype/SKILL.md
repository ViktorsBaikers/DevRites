---
name: rite-prototype
description: User-invoked throwaway prototype for one logic or UI design question; delete or absorb after the answer.
user-invocable: true
disable-model-invocation: true
argument-hint: "[the question the prototype is answering]"
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- When this skill asks for a DevRites specialist or writer agent, **explicitly** spawn the matching Codex custom agent from `.codex/agents/devrites-*.toml` through Codex subagents (`spawn_agent`), then wait for its result and reconcile it as the skill instructs. Do not do the review inline just because the instruction to spawn is embedded here: Codex under-fires embedded spawn/skill instructions (openai/codex #23496), so treat the spawn as required, not optional.
- The independence of a fresh-context subagent is the point. If Codex genuinely cannot spawn subagents in the current surface, run the documented inline fallback and **label the result an inline fallback, not an independent review**: an inline pass shares the calling context and is weaker evidence.
- Codex project hooks are installed in `.codex/hooks.json`. Review and trust them with `/hooks` before relying on hook enforcement.
- When this skill asks a HITL question via `AskUserQuestion`: Codex's equivalent (`request_user_input`) exists only in Plan mode. Outside Plan mode, render the option set as a plain numbered list in chat and **end the turn** so the human answers: NEVER silently pick an option yourself; auto-picking is AFK's contract, gated by the `.devrites/AFK` sentinel.


# $rite-prototype: throwaway code that answers ONE question

A prototype is **throwaway code that answers exactly one question**. The question
chooses the shape: get the shape wrong and the whole prototype wastes the user's time.

## 0. Read core rules

Read `.agents/skills/devrites-lib/reference/standards/core.md` first: the operating rules and the "capture the answer"
persistence discipline apply even to throwaway code. The other rule files load on demand.

## 1. Name the question

If the user did not give the question explicitly, ask **one** short question to pin it
down. Examples that count as "the question":

- "Does this reducer correctly model add / undo / re-add of the same item?"
- "Which of these three settings layouts feels right?"
- "Will the proposed state machine deadlock when X overlaps Y?"

A vague question ("explore this feature") is not enough: push back once to sharpen it.

## 2. Pick the branch

Read the user's prompt + the surrounding code. Two branches:

| Question shape | Branch | Artifact |
|---|---|---|
| "Does this **logic / state / data model** behave right?" | **Logic** | Smallest runnable harness (terminal script / REPL / mini node or python file) that pushes the model through the hard cases. |
| "What should this **look like / which UX wins**?" | **UI** | 2-4 **radically different** UI variations on a single route, switchable via a query param + a small floating selector bar. |

If genuinely ambiguous and the user is AFK: pick by the surrounding code (backend module
→ Logic; page / component → UI) and state the assumption in a comment at the top.

**Variations must differ in shape.** UI prototypes where the variations are just colour
or padding tweaks teach nothing: push each variation to a different *information
architecture* / *interaction model* so the user can pick a direction, not a polish.

## 3. Rules that apply to both branches

1. **Throwaway from day one, and visibly so.** Place the prototype next to where it
   will eventually be used (so context is obvious) but name it so a casual reader sees
   it's a prototype: `prototype/`, `__prototype__`, `.scratch/`, etc. Obey the project's
   existing routing / module convention; don't invent a new top-level structure.
2. **One command to run.** Use whatever task runner the project already has: `pnpm
   <name>`, `python <path>`, `bun <path>`. The user must start it without thinking.
3. **No persistence by default.** State lives in memory. Persistence is usually the
   thing the prototype is *checking*, not a dependency. If the question explicitly
   involves a DB, hit a scratch DB or a local file with a name like
   `PROTOTYPE — wipe me`.
4. **Skip the polish.** No tests. No error handling beyond what makes it run. No
   abstractions. The point is learn fast, delete fast.
5. **Surface state.** After every action (Logic) or on every variant switch (UI), print
   or render the full relevant state so the user can see what changed.
6. **Delete or absorb when done.** When the question has an answer, either delete the
   prototype or fold the validated decision into the real code: don't leave it
   rotting in the repo.

## 4. Capture the answer

The *answer* is the only durable output. Before declaring the prototype done, write
one of:

- An entry in the active feature's `decisions.md` (`.devrites/work/<slug>/decisions.md`)
  with the question and the answer.
- A `NOTES.md` next to the prototype if no active feature exists.

If the user is around, this is a quick conversation. If they're AFK, the prototype is
complete only once the verdict is filled in: a blank `VERDICT: ___` is not an
answer. Leave it marked **not yet complete** and queue the open question so a later pass
fills the verdict before the prototype is deleted.

## Where this slots in

`$rite-prototype` is a **scoped detour**, not a phase of its own. Typical placements:

- **Between `$rite-spec` and `$rite-define`:** a design question is undecidable on
  paper; prototype it, capture the answer, return to `$rite-define`.
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
