# Development workflow

Ship small, integrate often, keep the main branch releasable. DevRites runs this
lifecycle atop that loop: spec → define → vet → build → prove → polish → review → seal → ship.

## Work in small batches
- Break work into thin, independently shippable slices and integrate them frequently:
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
- Every change is reviewed before merge (see `code-review.md`). Keep reviews fast: slow
  reviews push people back toward big, risky batches.
- Set expectations on PR size and merge cadence: small, frequent, single-concern.

## Definition of done
The **Definition of Done** is the project's *standing bar*: the floor every change clears,
distinct from a slice's **acceptance criteria** (which change per slice and say what *this* work
must do). Acceptance criteria answer "did we build the right thing"; the DoD answers "is it
shippable at all". The bar does not move with deadline pressure: a Definition of Done that gets
renegotiated each time is not one. Apply it in tiers so the cost matches the scope:

- **Every slice:** Correctness + Quality: meets its acceptance criteria with **evidence**
  (tests/build/runtime: see `testing.md`), behaviour verified at runtime not just compiled; tests
  and build green in CI; reviewed and **within scope, no unrelated refactor snuck in**; edge and
  error paths handled, not only the happy path.
- **Every feature:** Integration + Documentation: works with the rest of the system, migrations/
  config/flags accounted for, backward compatibility considered for any public-interface change;
  docs/comments touched are updated (`documentation.md`) and describe the *current state in timeless
  language*, not the change history.
- **Every release:** Ship-readiness: security reviewed for any untrusted input/auth/data
  (`security.md`), observability on new critical paths (`observability.md`), a **rollback for
  anything risky**, and a **human has reviewed and approved before merge**.

Two invariants across all tiers: evidence post-dates the code it proves: edits made after
`/rite-prove` (polish/review) require re-proof before seal, stale evidence is not proof; and
"tests pass" is not a synonym for done. "Code written" is not done. "Proven and reviewed" is.

## Incremental delivery
Deliver a working end-to-end path each slice, then expand. Avoid the "90% done, nothing
proven" pile-up: one finished, verified slice beats five half-built ones.
