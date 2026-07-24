# ADR-0009: Pre-build decision coverage and implementation readiness

- **Status:** Accepted
- **Date:** 2026-07-23

> This ADR amends the lifecycle order in
> [ADR-0004](0004-state-schema-phases-sections.md) without changing its
> phase-relative completeness model.

## Context

Real feature runs reached `/rite-build` with product/constraint gaps and incomplete
tooling/proof prerequisites. The build then repeatedly asked for `rite-resolve` or
`rite-plan repair`, including approval for bounded technical recovery that an agent should
own. Earlier phases had good local checklists, but no mandatory whole-feature decision
coverage artifact, and the engine's build-readiness gate checked only `state.md`.

DevRites already had a reusable clarification protocol in `devrites-interview`; the missing
piece was lifecycle ownership and an executable gate, not another interview implementation.

The design review retained multi-dimensional readiness checks, fact-first
clarification, behavior scenarios, cross-artifact consistency, goal-backward
plan checking, and an explicit implementation-readiness assessment.

We deliberately reject asking the human to approve every technical finding and unbounded
interview ceremony. Those patterns reproduce the interruption fatigue this change fixes.

## Decision

1. Add one public phase between spec and temper:
   `frame → spec → clarify → temper → define → plan → vet → build → converge → prove → polish → review → seal → ship → done`.
2. `/rite-clarify` is mandatory but adaptive. It enumerates the feature topology, searches
   facts before asking, reuses `devrites-interview`, and takes a zero-question fast path when
   the written contract is already complete.
3. Clarification writes `decision-coverage.md`. Buildable work must contain
   `Decision coverage: CLEAR`; Partial, Missing, unowned, or blocking rows cannot pass.
4. Keep `/rite-vet` as the single final pre-build gate rather than adding a duplicate
   readiness phase. It performs goal-backward implementation-readiness checks and writes
   exactly one typed verdict:
   `READY`, `NEEDS CLARIFICATION`, or `NEEDS REPLAN`.
5. The engine requires both `Decision coverage: CLEAR` and
   `Implementation readiness: READY` (plus `test-plan.md`) at build entry. Missing
   clarification routes to `/rite-clarify`; missing vet readiness routes to `/rite-vet`.
   Neither route manufactures a human question.
6. Later-phase workspaces may retrofit decision coverage without cursor regression when the
   contract is unchanged. A changed requirement or acceptance criterion still uses the Spec
   Drift Guard and replanning.

## Alternatives considered

| Option | Why not |
|---|---|
| Strengthen spec/define/vet prose only | Still leaves no auditable proof that the whole feature topology was scanned and no deterministic build-entry gate. |
| Add a new final `/rite-ready` phase | Duplicates `/rite-vet`, adds ceremony, and splits one readiness source of truth across two phases. |
| Make clarification optional for small work | Optional gates are exactly how omissions survive; an adaptive zero-question pass has negligible cost while retaining the invariant. |
| Ask every unresolved item during build | Defers cheap decisions to the most expensive point and turns routine technical recovery into human authorization. |

## Consequences

- Planning cannot begin without explicit decision coverage, and build cannot begin without
  both clarification and engineering-readiness verdicts.
- Small/well-specified work gains one cheap deterministic pass, not an interview tax.
- Existing active workspaces missing the new artifacts receive actionable `/rite-clarify`
  and `/rite-vet` routes rather than false `/rite-resolve` prompts.
- `engine/internal/state/schema.go` remains the typed lifecycle/workspace authority. This
  topology is schema v2; migration upgrades declarations without fabricating readiness
  evidence, and generated manifests/host artifacts change with the authority.
