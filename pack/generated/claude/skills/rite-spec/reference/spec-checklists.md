# Spec-quality checklists

Before `/rite-define`, test requirement prose for completeness, clarity, and
measurability—not implementation. “Banner is prominent” lacks a threshold; ask
whether export defines empty data, never whether `exportCsv()` handles `[]`.
Function/file/library checks belong to `/rite-vet` and `/rite-prove`.

## Output: one file per requirement domain

Emit `.devrites/work/<slug>/checklists/<domain>.md` per covered domain; skip
`none`. Each domain maps gaps to a `devrites-interview` dimension:

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
| 2 | Does each requirement name an observable outcome a test could check? | pass | REQ-001..004 | — |
| 3 | Are all enumerations closed — no "etc." / "and so on" / open "…"? | pass | — | — |
```

Verdict: `pass | fail | n/a`. A fail is **CRITICAL** when ambiguity changes build
or acceptance: unquantified success, open enumeration, ambiguous data shape,
undefined stated-flow edge, or contradictory requirements. Other vague prose is
**minor**: record it; do not block.

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
- **Preservation:** each material brownfield outcome appears in `Existing behavior
  to preserve` with preserving REQ/AC and current evidence. Missing/vague “no
  regressions” or unjustified `none` is CRITICAL.
- **Backstops:** each row names an independent held-out, property/metamorphic, or
  direct behavioral check and the failure it discriminates; confidence/presence/self-review fail.
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
