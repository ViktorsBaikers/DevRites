# Development workflow

Ship small, integrate often, and keep the main branch releasable. Workflow phase order comes from the engine manifest, not this standard.

## Work in small batches
- Break work into thin, independently shippable slices and integrate them frequently:
  ideally merging to the main branch at least once a day.

## Short-lived branches, trunk always green
- Prefer short-lived branches off the main branch.
- The main branch is **always in a releasable state**. Validate every change through a
  **fast, reliable CI pipeline** (tests + build) before it merges.
- Hide incomplete work behind a **feature flag / toggle** rather than a long branch, so
  partial work can land without blocking releases or breaking the trunk.

## Review gate
- A human reviews every change before merge; `code-review.md` owns review scope and evidence.

## Definition of done

[`definition-of-done.md`](definition-of-done.md) owns the standing bar. Acceptance criteria prove the slice-specific behavior; the Definition of Done proves the change is shippable.
