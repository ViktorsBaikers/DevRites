# Question protocol

The discipline behind `/rite-spec` and `devrites-interview`.

## One question at a time
Ask exactly one question per turn. Multiple questions at once get one answered and the
rest ignored, and they signal you haven't prioritized.

## Always attach your best guess
Every question carries your current assumption and *why*:
> "I'm assuming the export is CSV only (simplest, covers the stated use case). Right,
> or do you need XLSX/PDF too?"
This converts an open question into a cheap yes/no correction and reveals your model so
the user can fix the premise, not just the question.

## Highest-value question first
Order by how much the answer changes the build. A question that changes the data model
or acceptance criteria beats a cosmetic one. If two are equal, ask the one that unblocks
the most downstream work.

## Structured options when the space is enumerable
```
1. <option> — <what it implies for build/scope>
2. <option> — <implication>
3. <option> — <implication>
4. Something else — I'll describe it
```
Always include the escape hatch (#4). Mark your recommended option.

## Confidence stop
Stop when you can **predict the user's answer** to the next question — that's ~95%
confidence. Signs you should stop: you're asking about reversible details, the user is
rubber-stamping your guesses, or the remaining unknowns don't change the spec.

## What NOT to ask
- Things the codebase answers (read it first).
- Reversible implementation details (decide and note as an assumption).
- Everything at once "to be thorough." Thoroughness is depth on the few that matter.

## Record
Confirmed answers → `decisions.md` (with rationale). Standing assumptions →
`assumptions.md`. Open non-blocking items → `questions.md`.
