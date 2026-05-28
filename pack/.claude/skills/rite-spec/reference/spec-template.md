# `spec.md` template

Write **what** to build and **why** — never **how** (implementation goes in `plan.md`,
written later by `/rite-define`). Ground every section in the codebase you investigated;
don't invent files, commands, or conventions.

**Two hard rules:**
1. **Mark every unknown** with `[NEEDS CLARIFICATION: the exact question]` instead of
   guessing. Open markers block `/rite-define` and `/rite-build` (no silent assumptions).
2. **No implementation details** in the spec — no tech/library/endpoint choices in the
   requirements or success criteria. Those are technology-agnostic.

```markdown
# Spec: <Feature>
Slug: <kebab-case>   Created: <date>   Status: Draft | Ready

## Objective
One paragraph: what this delivers and the value. WHAT + WHY, not HOW.

## Users / actors
Who uses it and the goal each has.

## Problem statement
What's broken/missing today; what users do instead.

## User scenarios  *(prioritized, each independently testable)*
Order by importance; each scenario should be shippable + verifiable on its own → it
becomes a build slice. Use Given/When/Then.
- **P1** — Given <state>, When <action>, Then <observable outcome>.
- **P2** — Given <state>, When <action>, Then <observable outcome>.
- **P3** — ...

## Functional requirements  *(testable, unambiguous)*
Number them so tasks and the seal can reference them.
- **FR-001**: The system MUST <capability>.
- **FR-002**: The system MUST <capability>.
- **FR-003**: The system MUST NOT <prohibited behavior>.
  (Mark gaps: "FR-004: [NEEDS CLARIFICATION: is export CSV-only or also XLSX?]")

## Key entities / data model   *(if data is involved, else "none")*
Entities, key fields, relationships, lifecycle (created/updated/soft-deleted). No DB
schema here — that's the plan.

## API / UI impact   *(else "none")*
Endpoints + contracts (shape, status, errors) at the WHAT level; screens + the states
each must handle (default/loading/empty/error/success/disabled).

## Design references   *(if the human supplied any, else "none")*
The screenshots / mockups / Figma / video / links that define the target look & behavior,
from references.md. Name what each shows and which scenarios it governs.
- R1 — references/<file> or <url> — <what it shows> → governs <scenario/FR>

## Success criteria  *(measurable AND technology-agnostic)*
Observable outcomes that mean "this worked". Numbers where possible. No tech names.
- Good: "A user exports 10k rows and receives a complete file in under 5s."
- Good (UI): "The list view matches reference R1 at 1280px and 375px."
- Bad: "The /export Sidekiq job uses streaming CSV." (that's HOW — belongs in plan)

## Acceptance criteria
Binary, evidence-backed checklist; each maps to a scenario/FR. See acceptance-criteria.md.
- [ ] <criterion> (FR-00x)

## Non-goals
Explicitly out of scope for this version.

## Constraints
Tech/time/compatibility/perf/security/compliance constraints that bound the solution.

## Placement & integration  *(where it lives — from investigation.md)*
So the work is **correctly placed to be used**, not bolted on:
- **Owns / lives in**: the module / layer / file / component that should hold this, + the
  right seam.
- **Reuse / extend**: existing patterns, components, utilities to build on — not duplicate.
- **Integration points**: callers & dependents; data read/written; APIs / events /
  contracts touched (how it interacts with the rest of the system).
- **Affected areas**: the real modules / routes / models / components this change touches.
- **Blast radius**: what could break (from the code-graph impact / callers).

## Commands discovered
- Tests: <cmd>   Build/typecheck: <cmd>   Lint: <cmd>   Run/dev: <cmd>
(From package scripts / Makefile / Gemfile / pyproject / go.mod / CI.)

## Test strategy
What proves each acceptance criterion (unit/integration/e2e/manual).

## Browser proof strategy   *(if UI, else "n/a")*
Which proof-ladder rung, which routes/viewports, and which design references to verify against.

## Risks
Ranked. Include migration / data / security / UX risks.

## Gaps, issues & decisions  *(drive the open count toward zero before /rite-define)*
Every material gap/issue found in investigation, the options offered to the user, and the
outcome. Resolved here = not rediscovered as drift later.
| Item | Type (gap / issue / conflict) | Options offered | Decision (owner) | Status |
|------|------------------------------|-----------------|------------------|--------|
| <e.g. token scope> | gap | per-user / per-session / follow-up | per-session (user) | resolved |

## Open questions  *( [NEEDS CLARIFICATION] register )*
List every open marker; blocking ones must be zero at the gate.

## Boundaries
- **Always do**: conventions to follow without asking.
- **Ask first**: new deps, second design system, schema/migration, auth changes, scope expansion.
- **Never do**: destructive ops, drive-by refactors, unrelated cleanup.

## Readiness gate  *(must pass before /rite-define plans it)*
- [ ] No blocking `[NEEDS CLARIFICATION]` markers remain (deferred ones are non-blocking)
- [ ] **Placement decided** — where it lives + integration points are known
- [ ] All material gaps/issues have a recorded decision
- [ ] Design references gathered + saved (if the human supplied any)
- [ ] Requirements are testable and unambiguous
- [ ] Success criteria are measurable and technology-agnostic
```
