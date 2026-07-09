# `plan.md` template

Write **how** to build what `spec.md` defines. The plan is **living, not sacred** —
`/rite-plan` repairs it when reality disagrees. This is where technology choices live
(they're banned from the spec).

```markdown
# Plan: <Feature>
Spec: ./spec.md   Date: <date>

## Summary
1–2 sentences: the primary requirement + the chosen approach.

## Technical context
- Language / runtime + version: <e.g. TypeScript 5, Node 20>
- Frameworks / libraries in play: <...>
- Storage / data: <...>
- Testing tools + commands: <from spec "Commands discovered">
- Target / platform / constraints: <...>
- `[NEEDS CLARIFICATION: ...]` for any unknown that affects the approach.

## Approach
The strategy in a few sentences. Why this over the alternatives considered.

## Slice strategy
How the feature is split into vertical `SLICE-###` increments, why the order is
risk-first within dependency tiers, and which acceptance criteria each slice covers. For a
wide mechanical refactor, use expand → migrate batches → contract; keep every migrate batch
green, or name the integration branch + final verify slice.

## Architecture decisions
Key decisions + rationale (mirror into decisions.md). New pattern vs reuse — prefer
reuse of existing project conventions.

## Dependency graph
What must exist before what (text is fine):
- SLICE-001 (no deps) → SLICE-002 (needs SLICE-001) → SLICE-004 (needs SLICE-002, SLICE-003)
- SLICE-003 (independent / parallelizable)

## Implementation order
Ordered slice list + the reason for the order (risk-first within a dependency tier).

## Validation strategy
After which slices to run tests / build / browser proof.

## Complexity & deviations gate
List anything that deviates from DevRites defaults (prefer existing conventions, the
simplest approach, feature scope only, no new deps/design system) and **justify it**.
If you can't justify a deviation, simplify instead of recording it.
| Deviation | Why needed | Simpler option rejected because |
|-----------|-----------|---------------------------------|
| <e.g. new dependency X> | <reason> | <why the in-repo option won't work> |

## Rollback
How to back out each risky step (migration down, feature flag, revert boundary, backup).

## Scope boundaries
What this plan will NOT touch. Restate "Ask first" / "Never do" from the spec.

## Source docs needed
Framework/library docs to consult (triggers devrites-source-driven). Record URLs in
decisions.md / evidence.md when used.

## Readiness gate  *(must pass before /rite-build)*
- [ ] Every spec acceptance criterion is covered by a slice
- [ ] Dependency order is acyclic and risk-first
- [ ] No unjustified deviation remains in the complexity gate
- [ ] Rollback exists for every destructive / migration step
- [ ] Every `Mode: HITL` slice has `Gate`, `SLA`, and `Checkpoint` populated
- [ ] Any wide mechanical refactor is sliced expand → migrate batches → contract, with green migrate batches or an integration branch + final verify slice
- [ ] No `Gate: blocking` slice is implicitly chained behind an AFK slice without surfacing the dependency
```
