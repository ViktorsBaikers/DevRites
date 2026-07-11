---
name: devrites-interview
description: Internal DevRites skill; DevRites agents invoke it explicitly, not by prompt match.
user-invocable: false
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- When this skill asks for a DevRites specialist or writer agent, **explicitly** spawn the matching Codex custom agent from `.codex/agents/devrites-*.toml` through Codex subagents (`spawn_agent`), then wait for its result and reconcile it as the skill instructs. Do not do the review inline just because the instruction to spawn is embedded here — Codex under-fires embedded spawn/skill instructions (openai/codex #23496), so treat the spawn as required, not optional.
- The independence of a fresh-context subagent is the point. If Codex genuinely cannot spawn subagents in the current surface, run the documented inline fallback and **label the result an inline fallback, not an independent review** — an inline pass shares the calling context and is weaker evidence.
- Codex project hooks are installed in `.codex/hooks.json`. Review and trust them with `/hooks` before relying on hook enforcement.
- When this skill asks a HITL question via `AskUserQuestion`: Codex's equivalent (`request_user_input`) exists only in Plan mode. Outside Plan mode, render the option set as a plain numbered list in chat and **end the turn** so the human answers — NEVER silently pick an option yourself; auto-picking is AFK's contract, gated by the `.devrites/AFK` sentinel.


# devrites-interview — extract intent

Close the gap between what the user said and what they want, at the cheapest moment —
before a plan, spec, or code exists.

## Protocol
- **State a confidence number.** Open with your one-line hypothesis of what the user wants and an
  explicit **0–100%** confidence in it. The number forces honesty — if you wrote 85% but can't
  predict how the user reacts to your next three questions, the number is wrong. Below **~70%**,
  append the single thing still unresolved so the user can close the gap directly.
- **One question per turn.** Multiple questions get one answered and the rest ignored.
- **Attach your best guess** to every question, with the reason:
  > "I'm assuming export is CSV only (covers the stated use case). Right, or also XLSX?"
  This turns an open question into a cheap correction and exposes your model so the user
  can fix the premise.
- **Highest-value question first** — order by how much the answer changes the build. A
  question that moves the data model or acceptance criteria beats a cosmetic one.
- **Impact-priority + bounded blocking.** Order unknowns **scope > security/privacy > UX >
  technical**, and cap **blocking** questions at **≤3** per pass — ask the few that actually gate
  the spec; **default-and-record** the rest in `assumptions.md` (best-guess + why). A reversible
  detail never earns a blocking question.
- **Structured options** when the space is enumerable — present them as the standard ranked
  **option set** (`standards/afk-hitl.md` → "Option set"): recommended **first**, labelled
  `(Recommended)`, each with a dimension-tagged rationale (`logic · infra · business ·
  architecture`, + `security`/`UX`/`risk` when in scope), plus the escape hatch. Render via
  `AskUserQuestion` when the harness has it:
  ```
  1. <recommended> (Recommended) — logic: … · infra: … · business: … · architecture: …
  2. <alternative> — <rationale + the trade-off it accepts>
  3. Something else — I'll describe it
  ```

## Stop condition
Stop **opening new questions** when **any** holds — don't interrogate past the point of value.
(Stopping the interview ≠ deciding for the user: a material call still goes back as a ranked
option set — confidence lowers the question's *cost* to a one-pick confirm, not its *owner*.)
- **Confidence — the predict-three test.** The 95% bar is checkable, not a feeling: *can you
  predict the user's reaction to the next three questions you would ask?* If yes, you're done
  extracting. If several rounds pass and you still can't predict, something foundational is
  missing — step back and name it, don't grind out more questions.
- **Convergence** — the last 2–3 answers only rubber-stamped your guesses and didn't
  move the spec.
- **Soft cap** — after ~8 material questions, proceed with your best-guess answers logged
  in `assumptions.md` rather than asking more (hard-stop sooner if the ask is small).

Depth on the few that matter, not breadth for its own sake. If answers stop converging —
you keep circling one area without progress — **reframe once** (below) instead of asking
another question.

## Want vs. should-want
People answer with what they think they *should* want — the buzzword, the best practice, the
convention — not what they actually want. Watch for tells: abstract virtues ("scalable", "clean",
"modern"), deferral to what's conventional, an answer that could be pasted into any project. When
you hear one, ask the unlock question — *"if you didn't have to justify this to anyone, what would
you actually want?"* — it often does more than the previous five questions. Extract the real want,
then record it; don't design to the performed one.

## What counts as a yes
Approval is an **explicit** yes, and several common replies aren't one:
- *"Whatever you think is best"* / *"you decide"* — **delegation, not approval.** Re-ask with two
  concrete options so the user chooses on the substance, not the deferral.
- *"Sounds good"* / *"sure, let's go"* — a polite exit; probe once that it's real agreement, not
  fatigue, on anything material.
- **Silence** — not consent. Half of misalignment is silent disagreement about what *won't* be
  built. Make the "out of scope" line explicit and get a yes on it too.

## Don't ask
- Things the codebase answers (read it first).
- Reversible implementation details (decide, log as an assumption).
- Everything at once "to be thorough."

## When the ask is vague — map the decision tree first
For a one-line or fuzzy ask (`"design a contact page"`), don't fire isolated questions.
First sketch the **decision tree** — the branches the answer splits into — and resolve
each branch **depth-first** with the protocol above. Domain branches per area:
`rite-spec/reference/interview-patterns.md`.

## Reframe (once, when stuck)
If the interview isn't converging, spend **one** turn challenging the premise rather than
refining it — *"is a form even the right answer here, or a mailto / booking link?"* A good
reframe collapses several open branches. Use it sparingly, then resume the protocol.

## /clarify mode — coverage scan of an existing spec
When invoked to clarify a written spec (not extract intent from scratch), scan it against the fixed
taxonomy and mark each **Clear / Partial / Missing**: Functional scope · Data model · Interaction
(API / UI states) · Non-functional (auth / latency / scale / compliance) · Edge cases (empty /
boundary / invalid / concurrent / failure). Then ask **≤5 prioritized questions** (impact order
above), targeting Missing before Partial, one per turn with a best-guess attached. **Integrate each
answer into the right spec section** as it lands — not just a Q&A log — and append a dated
**`## Clarifications`** block to `spec.md` (Q + resolution) for durable provenance. Re-run the scan
after answers; stop when every area is Clear or explicitly deferred, then re-score the affected
`checklists/<domain>.md`.

## Output
A short summary the caller can use: objective in one sentence, confirmed decisions,
still-open (non-blocking) items, recommended next step. If a workspace is active, write
Q&A to `questions.md`, confirmed calls to `decisions.md`, standing guesses to
`assumptions.md`. If not, just return the summary — don't create a workspace.
