# Depth calibration: always vet, scale the rigor (never skip)

`$rite-vet` runs on **every** plan. Simple reversible work gets a **light pass**;
big/risky work gets a **full pass**, never a skip. Standalone Vet and Autocomplete
use these same depth rules.

A four-section deep-dive on a one-file reversible plan is wasteful; rubber-stamping a
migration-touching plan is dangerous. Match the effort to the stakes, but **always** leave a
recorded engineering verdict and a coverage plan.

Authority: `.agents/skills/devrites-lib/reference/standards/acceptance-preserving-reslice.md`.

<!-- BEGIN RESLICE ROUTE-TO-ACTION -->
- `FOLD` → fold technical topology; invalidate Vet/readiness; affected Vet before Build.
- `GUARD_AND_REPAIR` → no planning writes; Spec Drift Guard → Clarify → Plan repair → affected Vet.
- `BLOCKED_INPUT` → no planning writes; exact diagnostic; recover input; reclassify.
<!-- END RESLICE ROUTE-TO-ACTION -->

## Full pass when ANY trips
- **Irreversible-risk contact:** the plan touches anything on the [`afk-hitl.md`](../../devrites-lib/reference/standards/afk-hitl.md) irreversible-risk
  list: destructive data migration, auth/authz boundary, public-API break, external-service
  contract, filesystem destruction outside the workspace.
- **Data model:** new/changed entities, relationships, or persistence shape.
- **Cross-module blast radius:** the impact (from a code-intelligence index if available (see
  `../../devrites-lib/reference/standards/tooling.md`) or an honest estimate) crosses
  module/service boundaries or has many dependents.
- **Complexity:** `plan.md` touches **>8 files** or adds **>2 new services/modules/classes**.
- **Multi-slice / multi-day work**, new dependency/pattern/design system. Slice count alone never creates a human pause.
- **The user asked**, or `--full`.

Full pass = §0 scope challenge + all four axes (each walked finding-by-finding) + the failure-mode
table + dependency-safety review + the reviewer loop. See [`review-axes.md`](review-axes.md).

## Light pass otherwise (the default for simple plans: still never a skip)
When none of the full-pass triggers fire, run the **light pass**: a fast but real engineering
check that still leaves a verdict and a coverage plan. In a handful of lines:
- **§0 scope challenge, in brief:** confirm minimum diff + reuse (one line each); if a complexity
  smell or missed reuse appears, escalate that to the full treatment.
- **One-line scan per axis** (architecture / plan code-quality / test-coverage / performance):
  "No issues" is a valid result, but you must look and say so per axis, not assume.
- **Always** produce the acceptance→test map + any regression-Criticals in `test-plan.md`. This is
  the part that makes "a correct engineering plan for *everything*" real. It never downgrades,
  light or full.
- Write the short `eng-review.md` (`Depth: light`) and harden `tasks.md` with the test requirements.

A light-pass finding that turns out to be real **escalates that axis to full treatment** (walk it
with the human, confidence-banded). Light means *less ceremony*, never *less honesty*.

## In `$rite-autocomplete`
Autocomplete runs `$rite-vet` after `$rite-define` on **every** feature: light or full per the
triggers above, **never skipped**. Under the AFK gate ceiling:
- **Auto-apply (no pause):** hardening findings: added test requirements, error-handling /
  failure-mode coverage, tightened scope, reuse-over-rebuild, dependency-order fixes
  (these never grow acceptance). Record the rationale in `decisions.md`.
- **Topology findings:** marked action; slice count alone never pauses.
- **Stops:** human-owned irreversible-risk findings pause. Technical recovery follows
  [the canonical retry contract](../../devrites-lib/reference/standards/afk-hitl.md#retry-cap-no-progress-loops-and-self-resolve),
  with no depth-specific round limit; no-progress exhaustion blocks, never grants readiness.
- **Cross-model** is off by default; it runs only if `--cross-model` was explicitly armed.
