# `spec.md` template

Write the product contract: WHAT users get, WHY it matters, how success is
measured, and what is out of scope. Keep HOW in `plan.md`; put topology and
diagrams in `architecture.md` / `flows.md`; put coverage in `traceability.md`.

Rules:

1. Mark unknowns with `[NEEDS CLARIFICATION: <question>]`; blocking unknowns
   stop `/rite-define`.
2. Use stable IDs: `REQ-001` for requirements and `AC-001` for acceptance.
3. Link to source artifacts instead of duplicating them.
4. Keep the file compact. If it exceeds the schema budget, add
   `Budget override: <reason>`.

```markdown
# Spec: <Feature>
Slug: <kebab-case>
Status: Draft | Ready
Created: <date>

## Problem
What is broken, missing, or costly today.

## Goal
One paragraph describing the user-visible outcome and why it matters.

## Non-goals
- <Explicitly out of scope.>

## Users / actors
| Actor | Need |
| --- | --- |
| <actor> | <goal> |

## Requirements
- REQ-001: The system MUST <observable product behavior>.
- REQ-002: The system MUST NOT <prohibited behavior>.

## Acceptance criteria
Each criterion is binary and evidence-backed; every item maps to at least one
requirement and later to at least one slice in `traceability.md`.

- [ ] AC-001: Given <state>, when <action>, then <observable outcome>. (REQ-001)
- [ ] AC-002: Given <state>, when <action>, then <observable outcome>. (REQ-002)

For behavioral or high-risk requirements, use the structured form below. Keep the
`AC-###` ID inside the scenario so traceability remains machine-checkable.

### Requirement: <name>
The system SHALL <core observable behavior>.

#### Scenario: <name>
- [ ] AC-003: **WHEN** <trigger> **THEN** <observable outcome>. (REQ-001)

## Edge Coverage
Deterministic boundary checklist for the requirements. Use `covered`, `backstop`,
`dismissed`, or `unresolved`; every row targets an existing REQ/AC unless dismissed.

| Edge ID | Requirement/AC | Class | Status | Reason/backstop |
| --- | --- | --- | --- | --- |
| EDGE-001 | AC-001 | empty/error/permission/race/migration | covered | <test/evidence or rationale> |

## Prohibitions (must-NOT)
Only bespoke constraints; generic security/privacy canon stays in project standards.
Use `resolved/test`, `resolved/judgment`, `dismissed`, or `unresolved`.

| Prohibition ID | Requirement/AC | Status | Test/evidence |
| --- | --- | --- | --- |
| PROH-001 | REQ-002 | resolved/test | <test/evidence link> |

## Edge cases
- <Narrative notes for boundary cases not captured in the table.>

## AI-SPEC annex
- Required when the feature touches model calls, RAG, agents, evals, or LLM output: `ai-spec.md` from `ai-spec-template.md`.
- Otherwise: not applicable.

## Measurable success
- <Metric or observable proof that the feature worked.>

## Scope boundaries
- Owns: <product surface or behavior>.
- Does not own: <adjacent area>.
- Placement summary: <one-line module/layer summary>; full technical map lives in `architecture.md`.

## References
- `brief.md` - request, objective, non-goals, success definition.
- `architecture.md` - technical placement and integration points.
- `flows.md` - diagrams when useful.
- `decisions.md` - ADR-style product/technical decisions.
- `traceability.md` - AC/REQ coverage once `/rite-define` runs.
- `design-brief.md` - UI direction when UI is in scope.

## Open questions
| Question ID | Gate | Question | Impact |
| --- | --- | --- | --- |
| Q-001 | blocking | [NEEDS CLARIFICATION: <question>] | AC-001 |

## Readiness gate
- [ ] No blocking `[NEEDS CLARIFICATION]` markers remain.
- [ ] Requirements use `REQ-###` IDs.
- [ ] Acceptance criteria use `AC-###` IDs and are independently provable.
- [ ] Edge Coverage rows target existing REQ/AC IDs or carry a dismissal reason.
- [ ] Prohibitions have resolved/dismissed status; `resolved/test` rows link test/evidence.
- [ ] AI features have `ai-spec.md`; non-AI work states the annex is not applicable.
- [ ] Non-goals and scope boundaries are explicit.
- [ ] Architecture/flows/decisions are linked out instead of duplicated here.
- [ ] UI work has `design-brief.md`; non-UI work states UI is out of scope.
```
