# `spec.md` template

Contract WHAT users get, WHY, success, and scope. HOW belongs in `plan.md`,
topology in `architecture.md`/`flows.md`, coverage in `traceability.md`.

Use `[NEEDS CLARIFICATION: <question>]` (blocking stops `$rite-clarify`) and
stable `REQ-001`/`AC-001` IDs. Link, never duplicate, source artifacts. Over
schema budget requires `Budget override: <reason>`.

```markdown
# Spec: <Feature>
Slug: <kebab-case>
Status: Draft | Ready
Created: <date>

## Problem
<What is broken, missing, or costly.>

## Goal
<User-visible outcome and why it matters.>

Capability impact: <affected capability/ies and change | none — specific justification>

## Non-goals
- <Explicitly out of scope.>

## Existing behavior to preserve
List affected observable public/security/data/operational outcomes that MUST
survive; map preserving REQ/AC plus current test/runtime/contract/source evidence.
No implementation detail. True greenfield: `none — no existing behavior in the affected scope`.

| Existing outcome | Preserved by | Current evidence |
| --- | --- | --- |
| <outcome that must not regress> | REQ-001 / AC-001 | <current evidence> |

## Users / actors
| Actor | Need |
| --- | --- |
| <actor> | <goal> |

## Requirements
- REQ-001: The system MUST <observable product behavior>.
- REQ-002: The system MUST NOT <prohibited behavior>.

## Acceptance criteria
Binary/evidence-backed; each maps to a requirement and later a traceability slice.

- [ ] AC-001: Given <state>, when <action>, then <outcome>. (REQ-001)
- [ ] AC-002: Given <state>, when <action>, then <outcome>. (REQ-002)

Behavioral/high-risk work uses this grammar, with the AC ID inside the scenario:

### Requirement: <name>
The system SHALL <core observable behavior>.

#### Scenario: <name>
- [ ] AC-003: **WHEN** <trigger> **THEN** <observable outcome>. (REQ-001)

## Edge Coverage
Use `covered | backstop | dismissed | unresolved`; target an existing REQ/AC
unless dismissed. `backstop` requires a named independent held-out,
property/metamorphic, or direct behavioral check plus the wrong behavior it
discriminates. If unavailable, use `unresolved`; presence/prose/self-judgment cannot pass.

| Edge ID | Requirement/AC | Class | Status | Reason/backstop |
| --- | --- | --- | --- | --- |
| EDGE-001 | AC-001 | empty/error/permission/race/migration | covered | <evidence/rationale> |

## Prohibitions (must-NOT)
Bespoke only; generic security/privacy stays in standards. Status:
`resolved/test | resolved/judgment | dismissed | unresolved`.

| Prohibition ID | Requirement/AC | Status | Test/evidence |
| --- | --- | --- | --- |
| PROH-001 | REQ-002 | resolved/test | <test/evidence link> |

## Edge cases
- <Boundary note not captured above.>

## AI-SPEC annex
- Model/RAG/agent/eval/LLM-output scope: `ai-spec.md` from `ai-spec-template.md`.
- Otherwise: not applicable.

## Measurable success
- <Metric or observable proof.>

## Scope boundaries
- Owns: <surface/behavior>.
- Does not own: <adjacent area>.
- Placement summary: <module/layer>; full map in `architecture.md`.

## Coverage seed
- Actors/journeys/components: <material surfaces>.
- States/data/contracts/integrations: <material boundaries>.
- Operations/proof: <config, observability, rollout/rollback, evidence constraints>.

## References
- `brief.md`: request/outcome/scope; `architecture.md`: placement/integration;
  `flows.md`: diagrams; `decisions.md`: decisions; `decision-coverage.md`: Clarify
  topology/verdict; `traceability.md`: Define coverage; `design-brief.md`: UI direction.

## Open questions
| Question ID | Gate | Question | Impact |
| --- | --- | --- | --- |
| Q-001 | blocking | [NEEDS CLARIFICATION: <question>] | AC-001 |

## Readiness gate
- [ ] No blocking clarification; REQ/AC IDs are valid and ACs independently provable.
- [ ] Existing affected behavior maps to preserving REQ/AC + current evidence, or uses the exact justified greenfield `none`.
- [ ] Edge rows target REQ/AC or justify dismissal; every backstop names independent discriminating evidence, else `unresolved`.
- [ ] Prohibitions resolve/dismiss; `resolved/test` links evidence.
- [ ] AI has `ai-spec.md` and UI has `design-brief.md`; out-of-scope work states not applicable.
- [ ] Non-goals/scope are explicit; capability impact is singular/specific and matches ledger deltas.
- [ ] Architecture/flows/decisions are linked, not duplicated; Coverage seed names Clarify surfaces.
```
