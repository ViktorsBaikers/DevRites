---
name: devrites-interview
description: Interview the user one question at a time to extract what they want. Use when the user says "interview me", "I'm not sure what I want", or `/rite-spec` / `/rite-define` flags the ask as underspecified. Not for casual clarification or ideation (use `/rite-pressure-test`).
user-invocable: false
---

# devrites-interview — extract intent

Close the gap between what the user said and what they want, at the cheapest moment —
before a plan, spec, or code exists.

## Protocol
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
  **option set** (`rules/afk-hitl.md` → "Option set"): recommended **first**, labelled
  `(Recommended)`, each with a dimension-tagged rationale (`logic · infra · business ·
  architecture`, + `security`/`UX`/`risk` when in scope), plus the escape hatch. Render via
  `AskUserQuestion` when the harness has it:
  ```
  1. <recommended> (Recommended) — logic: … · infra: … · business: … · architecture: …
  2. <alternative> — <rationale + the trade-off it accepts>
  3. Something else — I'll describe it
  ```

## Stop condition
Stop when **any** holds — don't interrogate past the point of value:
- **Confidence** — you can predict the next answer (~95%); remaining unknowns are
  reversible details that don't change the spec.
- **Convergence** — the last 2–3 answers only rubber-stamped your guesses and didn't
  move the spec.
- **Soft cap** — after ~8 material questions, proceed with your best-guess answers logged
  in `assumptions.md` rather than asking more (hard-stop sooner if the ask is small).

Depth on the few that matter, not breadth for its own sake. If answers stop converging —
you keep circling one area without progress — **reframe once** (below) instead of asking
another question.

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

Domain question ladders: `rite-spec/reference/interview-patterns.md`.
