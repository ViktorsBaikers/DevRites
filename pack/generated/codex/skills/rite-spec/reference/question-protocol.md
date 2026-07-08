# Question protocol

The discipline behind `$rite-spec` and `devrites-interview`.

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

## Ranked option set — you recommend, the human decides
A material decision is **put to the human**, never settled for them. Render it as a ranked
option set — recommended option **first** and marked `(Recommended)`:
```
1. <recommended option> (Recommended) — <rationale + trade-off, tagged by dimension>
2. <alternative> — <implication>
3. <alternative> — <implication>
4. Something else — I'll describe it
```
2–4 real options, always the escape hatch (#4). In HITL render it via `AskUserQuestion`; full
render contract + AFK auto-pick behaviour: the **Option set** section of
[`afk-hitl.md`](../../devrites-lib/reference/standards/afk-hitl.md). Fold any web-search / docs finding into the
rationale as a cited source (`.agents/skills/devrites-lib/reference/standards/tooling.md`).

## Confidence changes the question's cost, not its owner
Being near-certain of the answer does **not** convert a material decision into one you make
silently. You still present the set — high confidence just means the human confirms your
`(Recommended)` option in a single pick instead of deliberating. The stop conditions below
govern when to stop **opening new lines of questioning**; they never license deciding a
material gap (scope · placement · data model · UX · security · migration · acceptance)
yourself. When unsure whether a gap is material, ask.

## Stop conditions — when to stop opening NEW questions (any one)
- **Convergence** — the last 2–3 picks only rubber-stamped your recommended option; the spec
  stopped moving. Remaining material gaps still get their option set (a fast one-pick confirm),
  but stop *hunting* for new ones.
- **Soft cap** — after ~8 material questions, put the remaining low-stakes gaps as
  best-guess assumptions logged to `assumptions.md` (hard-stop sooner if the ask is small).
  A genuinely material, irreversible gap is **not** capped away — it stays a blocking question.

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
`questions.md` and doesn't block. This feeds the `$rite-spec` readiness gate.

## What NOT to ask
- Things the codebase answers (read it first).
- Reversible implementation details (decide and note as an assumption).
- Everything at once "to be thorough." Thoroughness is depth on the few that matter.

## Record
Confirmed answers → `decisions.md` (with rationale). Standing assumptions →
`assumptions.md`. Open non-blocking items → `questions.md`.
