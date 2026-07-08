# Spec-quality checklists — "unit tests for the English"

Before the spec hands off to `/rite-define`, test the **requirement prose itself** the way you'd
unit-test code: is each requirement complete, unambiguous, and measurable? These checklists
interrogate the *writing*, not the implementation — there is no code to run yet. A criterion like
"the banner is prominent" passes a human skim and fails here ("prominent" is unquantified).
Catching it now is a one-line spec edit; catching it at `/rite-prove` is a reslice.

**Not implementation tests.** A checklist item asks "is *export* defined for an empty dataset?" —
it does **not** ask "does `exportCsv()` handle `[]`?". The first tests the spec; the second is
`/rite-vet`'s test-plan and `/rite-prove`'s job. These files never name a function, file, or library.

## Output — one file per requirement domain

Emit `.devrites/work/<slug>/checklists/<domain>.md`, one per domain the spec actually covers (skip
a domain the spec marks "none"). The domains mirror the interview taxonomy, so a gap here maps to a
`devrites-interview` dimension:

| Domain file | Tests the prose of |
|---|---|
| `functional.md` | Functional requirements + scenarios — is each capability stated, bounded, testable? |
| `data-model.md` | Key entities / data model — shapes, fields, lifecycle, relationships (skip if "none"). |
| `interaction.md` | API / UI impact + UX states — every screen state and contract named (skip if no UI/API). |
| `non-functional.md` | Constraints, auth / data sensitivity, latency / scale / compatibility budgets. |
| `edge-cases.md` | Empty / boundary / invalid / concurrent / failure paths the requirements imply. |

## Each item — a question, a verdict, the line it interrogates

```markdown
# Spec checklist: <domain> — <slug>
Scored: <iso>   Verdict: pass | gaps (<n CRITICAL / n minor>)

| # | Question (tests the requirement prose) | Verdict | Spec line | Severity |
|---|---|---|---|---|
| 1 | Is every quantitative qualifier ("prominent", "fast", "large") given a number or reference? | fail | "banner is prominent" | **CRITICAL** |
| 2 | Does each requirement name an observable outcome a test could check? | pass | FR-001..004 | — |
| 3 | Are all enumerations closed — no "etc." / "and so on" / open "…"? | pass | — | — |
```

Verdict is `pass` / `fail` / `n/a`. A `fail` carries a severity:
- **CRITICAL** — the ambiguity would change the build or its acceptance: an unquantified
  acceptance/success criterion, an incomplete enumeration in a requirement, an ambiguous data
  shape, an undefined edge case on a stated flow, a contradictory pair of requirements.
- **minor** — vague but non-load-bearing prose. Record it; it does not block.

## The question bank (seed — extend per domain)

Each question tests one of the failure modes of requirement prose:
- **Measurability** — every "good / fast / prominent / simple / secure" carries a number, a budget,
  or a named reference. No adjective stands in for a threshold.
- **Completeness** — every enumeration is closed (no "etc."); every requirement with a precondition
  states the failure branch; every entity names its required fields + lifecycle; every stated flow
  names its empty / error / boundary behaviour.
- **Clarity** — one entity, one name (no `user`/`customer`/`account` drift); no requirement two
  readers would implement differently; no "should" where "MUST" is meant.
- **Testability** — each acceptance criterion is binary and names (or clearly implies) its evidence.
  A criterion only provable by reading code is a fail.
- **Consistency** — no requirement contradicts another, the data model, or a non-goal.

## How it feeds the readiness gate

The spec **Readiness gate** (bottom of [`spec-template.md`](spec-template.md)) folds these in: every
emitted `checklists/<domain>.md` must reach `Verdict: pass` (zero CRITICAL fails) before the gate
passes. Minor fails are logged, not blocking. A single open CRITICAL keeps the spec `Status: Draft`.
`/rite-define` reads the checklists at its step 0 and **hard-blocks while any CRITICAL is unchecked**;
a spec that skips the checklists is treated as not-yet-scored — define stops and routes back here.

## Discipline
- Score honestly. A checklist you pass by softening the *question* instead of the *spec* is the
  spec-quality version of weakening a test to go green — it defeats the gate.
- Don't pad. Five real questions that find one CRITICAL beat thirty rubber-stamped rows.
- These are *English* tests. The moment a question needs a function name to answer, it belongs in
  `/rite-vet`'s `test-plan.md`, not here.
