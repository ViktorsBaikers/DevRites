# ADR-0034: Workflow Artifact byte corrections get code review, not a plan loop

- **Status:** Accepted
- **Date:** 2026-10-15

## Context

A Workflow Artifact is a root-authored executable file under `.devrites/` that
proves a slice (for example a native UI test launcher). Vet admits its design:
paths, modes, behavior/interface/fixture rows, and the proof command. The root
authors the bytes in a disposable preflight after Vet `READY`. The source handle
derives from the readiness binding (ADR-0027), so promoting corrected bytes
needs a new binding, and only Vet may record one.

In practice every bug fix in such a script ran `/rite-plan repair` (plan-drafter)
plus an affected `/rite-vet` (plan-reviewer). The plan reviewer then reviewed
Python cleanup and rollback code. One measured feature spent 36% of agent time
after Build started in Plan and Vet. All 144 of its Plan/Vet waves served one
slice's proof harness, for example four consecutive archive-policy repair/Vet
pairs for close and rollback bugs.

## Decision

- A correction that leaves every admission row and `WA-*` reference row
  unchanged is a technical fix. It follows `OFFLINE_RECOVERY` and never
  `/rite-plan repair`.
- The root records the corrected bytes' identity digest as one exact
  `test-plan.md` line below the admission block:
  `Workflow Artifact reviewed identity: <64 lowercase hex>`.
- `devrites-engine check drift` keeps a second digest of `test-plan.md` with
  only that exact line shape removed. When it is the only change across all
  readiness inputs, the drift is reported as `reviewed-identity-only`.
- In that case narrow `/rite-vet` dispatches one fresh `devrites-code-reviewer`
  on the prior-source→re-preflight diff against the unchanged admission rows,
  instead of a plan drafter and plan reviewer. Fold and recheck follow the
  canonical retry contract. Green review lets Vet record the new binding as
  usual (ADR-0027 still applies; the engine never refreshes it).
- Any other drift, including a malformed identity line, keeps the existing
  narrow recheck by the plan reviewer.

This narrows ADR-0027's "semantic review" for this one case to an independent
code review of the exact bytes. The Workflow Artifact module, source-handle
derivation, routes, and golden values are unchanged.

## Consequences

- The reviewed bytes are now bound into a readiness input. Before this, Vet
  approved the design but never the exact corrected bytes.
- A contract change to a Workflow Artifact still goes through Plan and Vet.
- Slice-bound Vet rechecks also count toward the per-slice round cap
  (`dispatch open --slice`), so a harness that keeps failing forces a recorded
  reslice or simpler-proof review.

## Alternatives considered

| Option | Why not |
|---|---|
| Derive the source handle from the identity digest | Changes the protocol's crypto, golden values, and the reference model test, for the same saving. |
| Let the engine refresh the binding after review | ADR-0027 keeps binding renewal a Vet act; the engine stays thin. |
| Keep plan-reviewer for byte fixes | Measured cost above; a plan reviewer is the wrong specialist for code defects. |
