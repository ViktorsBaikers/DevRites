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

## Stop conditions (any one)
- **Confidence** — you can predict the user's answer to the next question (~95%).
- **Convergence** — the last 2–3 answers only rubber-stamped your guesses; the spec
  stopped moving.
- **Soft cap** — after ~8 material questions, proceed with best-guess answers logged as
  assumptions (hard-stop sooner if the ask is small).

If you keep circling one area without progress, **reframe once** — challenge the premise
rather than asking again.

## Coverage gate — am I done asking?
Before declaring the interview complete, each dimension is **resolved** or **explicitly
deferred** (logged, non-blocking) — never silently skipped:
- [ ] **Objective** — one-sentence success + the real problem behind it.
- [ ] **Scope** — what's in vs explicitly out for v1.
- [ ] **Data model** — core entities + relationships (or "none").
- [ ] **UX / flow** — happy path + empty / loading / error / permission states (or "no UI").
- [ ] **Integration** — external systems / APIs / contracts (or "none").
- [ ] **Non-functional** — auth, sensitive data, latency / scale (or "n/a").
- [ ] **Acceptance** — how each requirement is *proven* (test / observation).

A blocking gap in any dimension keeps the interview open; a deferred one goes to
`questions.md` and doesn't block. This feeds the `/rite-spec` readiness gate.

## What NOT to ask
- Things the codebase answers (read it first).
- Reversible implementation details (decide and note as an assumption).
- Everything at once "to be thorough." Thoroughness is depth on the few that matter.

## Record
Confirmed answers → `decisions.md` (with rationale). Standing assumptions →
`assumptions.md`. Open non-blocking items → `questions.md`.
