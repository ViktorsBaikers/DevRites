# Development workflow

Ship small, integrate often, keep the main branch releasable. This is the general
engineering loop the DevRites feature lifecycle (spec → define → build → prove → polish →
review → seal → ship) runs on top of.

## Work in small batches
- Break work into thin, independently shippable slices and integrate them frequently —
  ideally merging to the main branch at least once a day.
- Small batches shrink merge pain, surface integration problems early, and get real
  reviews. Large batches hide defects and stall.

## Short-lived branches, trunk always green
- Prefer short-lived branches off the main branch (trunk-based). Long-running branches
  drift and create painful merges.
- The main branch is **always in a releasable state**. Validate every change through a
  **fast, reliable CI pipeline** (tests + build) before it merges.
- Hide incomplete work behind a **feature flag / toggle** rather than a long branch, so
  partial work can land without blocking releases or breaking the trunk.

## Review gate
- Every change is reviewed before merge (see `code-review.md`). Keep reviews fast — slow
  reviews push people back toward big, risky batches.
- Set expectations on PR size and merge cadence: small, frequent, single-concern.

## Definition of done
A slice is done only when:
- it meets its acceptance criteria with **evidence** (tests/build/runtime — see
  `testing.md`);
- tests and build are green in CI;
- it's reviewed and within scope;
- docs/comments touched by the change are updated (`documentation.md`);
- any destructive/migration step has a rollback.
- evidence post-dates the code it proves; edits made after `/rite-prove` (polish/review)
  require re-proof before seal — stale evidence is not proof.
"Code written" is not done. "Proven and reviewed" is.

## Incremental delivery
Deliver a working end-to-end path each slice, then expand. Avoid the "90% done, nothing
proven" pile-up — one finished, verified slice beats five half-built ones.
