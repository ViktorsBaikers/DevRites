# rite-define — anti-patterns

Load this when standing a non-trivial planning decision in `/rite-define`,
or when the agent feels reluctance toward vertical slicing or coverage
mapping.

Pack-wide rationalizations + red flags: see [rules/anti-patterns.md](../../../rules/anti-patterns.md).

## Phase-specific rationalizations

| Excuse | Rebuttal |
|---|---|
| "Spec is good enough; just start coding." | Plan separates HOW from WHAT for a reason — missed HOW shows up as drift at slice 3. |
| "One big slice is fine, the work is all related." | If slice 1 isn't shippable on its own, you're not slicing vertically — you're staging waterfall in disguise. |
| "Tests can come at build time, not in tasks." | Every slice's `Tests to write/run` line is the contract that proves its acceptance — leave it blank, lose the contract. |
| "Backend + frontend belong in one slice." | Fullstack goes contract-first: split the contract, then build a thin vertical slice that crosses both layers. |
| "I can skip mapping every spec criterion." | An unmapped criterion is one nobody will build. Coverage isn't bureaucracy. |

## Red Flags

- A slice with no acceptance criterion link back to `spec.md`.
- No first slice that's end-to-end thin (every slice depends on infra that doesn't exist yet).
- A new dependency added without rationale in `decisions.md`.
- A slice that doesn't list "Existing to reuse / extend" (you didn't search).
- Plan readiness gate failing but you're about to call `/rite-build` anyway.
