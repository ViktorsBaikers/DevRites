---
name: devrites-interview
description: One-question-at-a-time interview to extract what the user wants — each question carries a best-guess + structured options; stop at ~95% confidence. Use when the user says "interview me", "grill me", "I'm not sure what I want", or `/rite-spec` / `/rite-define` flags the ask as underspecified. Not for casual clarification or ideation (use `rite-pressure-test`).
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
- **Structured options** when the space is enumerable:
  ```
  1. <option> — <implication>
  2. <option> — <implication>
  3. Something else — I'll describe it
  ```
  Always include the escape hatch; mark your recommendation.

## Stop condition
Stop when you can **predict the user's answer** (~95% confidence) — when remaining
unknowns are reversible details that don't change the spec. Don't interrogate; depth on
the few that matter, not breadth for its own sake.

## Don't ask
- Things the codebase answers (read it first).
- Reversible implementation details (decide, log as an assumption).
- Everything at once "to be thorough."

## Output
A short summary the caller can use: objective in one sentence, confirmed decisions,
still-open (non-blocking) items, recommended next step. If a workspace is active, write
Q&A to `questions.md`, confirmed calls to `decisions.md`, standing guesses to
`assumptions.md`. If not, just return the summary — don't create a workspace.

Domain question ladders: `rite-spec/reference/interview-patterns.md`.
