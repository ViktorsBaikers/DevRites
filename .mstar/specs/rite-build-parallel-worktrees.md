# Spec stub: rite-build parallel worktrees

**plan_id:** `001-rite-build-parallel-worktrees`  
**SSOT plan:** `.mstar/plans/001-rite-build-parallel-worktrees.md`  
**Status:** Prepare — specify complete (product); plan drafted, not user-locked

## One-line product contract

Default `/rite-build` stays **one verified slice then stop**. Opt-in `/rite-build --parallel N` (**2≤N≤3**; N=1/omitted ≡ serial) builds only **path-disjoint** slices in **git worktrees**, then **serial-integrates** into the feature line; **one red/gap aborts the whole batch**. Control checkout owns shared `.devrites/work/<slug>` bookkeeping. AFK charges only after successful integrate.

## Locked clarify pointers

See plan § Locked clarify (2026-08-23): merge model, UX flag, isolation, eligibility, failure, bookkeeping.

## Acceptance (product)

Falsifiable criteria live in the plan § Specify → Acceptance language (A1–A8) and Definition of Done (D1–D8). This stub is a durable pointer only; do not implement from the stub alone.
