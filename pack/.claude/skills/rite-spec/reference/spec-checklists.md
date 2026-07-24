# Spec-quality checklists

Before `/rite-define`, check that each requirement is complete, unambiguous, and
measurable. These checklists evaluate the prose, not the implementation. For example,
"the banner is prominent" fails because "prominent" has no threshold. Fixing that in
the spec avoids a later reslice.

These are not implementation tests. Ask "is *export* defined for an empty dataset?", not
"does `exportCsv()` handle `[]`?". Implementation checks belong to `/rite-vet` and
`/rite-prove`. These files never name a function, file, or library.

## Output: one file per requirement domain

Emit `.devrites/work/<slug>/checklists/<domain>.md`, one per domain the spec covers.
Skip a domain marked "none". The domains match the interview taxonomy, so each gap maps
to a `devrites-interview` dimension:

| Domain file | Tests the prose of |
|---|---|
| `functional.md` | Functional requirements + scenarios: is each capability stated, bounded, testable? |
| `data-model.md` | Key entities / data model: shapes, fields, lifecycle, relationships (skip if "none"). |
| `interaction.md` | API / UI impact + UX states: every screen state and contract named (skip if no UI/API). |
| `non-functional.md` | Constraints, auth / data sensitivity, latency / scale / compatibility budgets, human-only proof prerequisites. |
| `edge-cases.md` | Empty / boundary / invalid / concurrent / failure paths the requirements imply. |

## Each item: a question, a verdict, the line it interrogates

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
- **CRITICAL:** the ambiguity would change the build or its acceptance: an unquantified
  acceptance/success criterion, an incomplete enumeration in a requirement, an ambiguous data
  shape, an undefined edge case on a stated flow, a contradictory pair of requirements.
- **minor:** vague but nonessential prose. Record it; it does not block.

## Question bank

Each question checks one requirement-prose failure mode:
- **Measurability:** every "good / fast / prominent / simple / secure" carries a number, a budget,
  or a named reference. No adjective stands in for a threshold.
- **Completeness:** every enumeration is closed (no "etc."); every requirement with a precondition
  states the failure branch; every entity names its required fields + lifecycle; every stated flow
  names its empty / error / boundary behaviour.
- **Clarity:** one entity, one name (no `user`/`customer`/`account` drift); no requirement two
  readers would implement differently; no "should" where "MUST" is meant.
- **Testability:** each acceptance criterion is binary and names (or clearly implies) its evidence.
  A criterion only provable by reading code is a fail.
- **Consistency:** no requirement contradicts another, the data model, or a non-goal.
- **Non-functional:** each NFR names affected REQ/AC IDs or a bounded `global` scope;
  human-only proof prerequisites name their owner.

## Readiness gate

The spec **Readiness gate** at the bottom of
[`spec-template.md`](spec-template.md) requires every
emitted `checklists/<domain>.md` must reach `Verdict: pass` (zero CRITICAL fails) before the gate
passes. Minor fails are logged, not blocking. A single open CRITICAL keeps the spec `Status: Draft`.
`/rite-define` reads the checklists at step 0 and **hard-blocks while any CRITICAL is
unchecked**. A spec without checklists is not yet checked, so define stops and routes
back here.

## Discipline
- Score honestly. Do not soften a checklist question to pass a weak spec.
- Don't pad. Five real questions that find one CRITICAL beat thirty rubber-stamped rows.
- If a question needs a function name, it belongs in `/rite-vet`'s `test-plan.md`.
