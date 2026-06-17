# Depth calibration — always vet, scale the rigor (never skip)

`/rite-vet` runs on **every** plan — every feature deserves a correct engineering plan, not
just the big ones. What changes with stakes is the *depth*, never *whether* it runs: a simple,
reversible plan gets a fast **light pass**; a big or risky one gets the **full pass**. There is
no skip. This shared definition is used by both the standalone `/rite-vet` and the always-run
`/rite-autocomplete` step, so they agree on how deep to go.

A four-section deep-dive on a one-file reversible plan is wasteful; rubber-stamping a
migration-touching plan is dangerous. Match the effort to the stakes — but **always** leave a
recorded engineering verdict and a coverage plan.

## Full pass when ANY trips
- **Irreversible-risk contact** — the plan touches anything on the `afk-hitl.md` irreversible-risk
  list: destructive data migration, auth/authz boundary, public-API break, external-service
  contract, filesystem destruction outside the workspace.
- **Data model** — new/changed entities, relationships, or persistence shape.
- **Cross-module blast radius** — the codegraph/graphify impact (or honest estimate) crosses
  module/service boundaries or has many dependents.
- **Complexity** — `plan.md` touches **>8 files** or adds **>2 new services/modules/classes**.
- **Multi-slice / multi-day**, or a new dependency / pattern / second design system.
- **The user asked**, or `--full`.

Full pass = §0 scope challenge + all four axes (each walked finding-by-finding) + the failure-mode
table + parallelization + the reviewer loop. See [`review-axes.md`](review-axes.md).

## Light pass otherwise (the default for simple plans — still never a skip)
When none of the full-pass triggers fire, run the **light pass**: a fast but real engineering
check that still leaves a verdict and a coverage plan. In a handful of lines:
- **§0 scope challenge, in brief** — confirm minimum diff + reuse (one line each); if a complexity
  smell or missed reuse appears, escalate that to the full treatment.
- **One-line scan per axis** (architecture / plan code-quality / test-coverage / performance) —
  "No issues" is a valid result, but you must look and say so per axis, not assume.
- **Always** produce the acceptance→test map + any regression-Criticals in `test-plan.md`. This is
  the part that makes "a correct engineering plan for *everything*" real — it never downgrades,
  light or full.
- Write the short `eng-review.md` (`Depth: light`) and harden `tasks.md` with the test requirements.

A light-pass finding that turns out to be real **escalates that axis to full treatment** (walk it
with the human, confidence-banded). Light means *less ceremony*, never *less honesty*.

## In `/rite-autocomplete`
Autocomplete runs `/rite-vet` after `/rite-define` on **every** feature — light or full per the
triggers above, **never skipped**. Under the AFK gate ceiling:
- **Auto-apply (no pause):** hardening findings — added test requirements, error-handling /
  failure-mode coverage, tightened scope, reuse-over-rebuild, dependency-order / parallel-lane fixes
  (these never grow acceptance). Record the rationale in `decisions.md`.
- **Blocking pause:** any finding that **grows scope, adds a slice, changes an acceptance
  criterion, or alters product behavior**.
- **Always pause:** irreversible-risk findings, and any axis still below bar after the
  ≤3-iteration reviewer loop.
- **Cross-model** is off by default; it runs only if `--cross-model` was explicitly armed.
