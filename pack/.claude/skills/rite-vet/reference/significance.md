# Significance test — fire the full plan review, or skip it

`/rite-vet` is **significance-gated**: full engineering rigor only when the plan is big or
risky enough to earn it, a one-line skip otherwise. This is the shared definition used by both
the optional `/rite-vet` path and the always-run (significance-gated) `/rite-autocomplete` step,
so they agree on when it fires.

A four-section engineering review of a one-file, reversible plan is theatre; the same review of
a plan that touches a migration and three services is the cheapest insurance you'll buy. Gate on
stakes and shape, not uniformly. `--full` overrides a skip when the user wants the full pass anyway.

## Fire the FULL review when ANY trips
- **Irreversible-risk contact** — a slice touches anything on the `afk-hitl.md` irreversible-risk
  list: destructive data migration, auth/authz boundary, public-API break, external-service
  contract, filesystem destruction outside the workspace.
- **Data model** — new/changed entities, relationships, or persistence shape.
- **Cross-module blast radius** — the codegraph/graphify impact (or honest estimate) crosses
  module/service boundaries or has many dependents.
- **Complexity smell** — `plan.md` touches **>8 files** or adds **>2 new services/modules/classes**.
- **Multi-slice / multi-day** — `tasks.md` decomposes into several slices (a wrong architecture or
  test-coverage call is expensive to unwind once code lands).
- **New dependency or pattern** — the plan adds a dependency, a second design system, or an
  architectural/concurrency pattern not already in the codebase.
- **The user asked** — "vet the plan", "engineering review", "review the architecture", "lock in
  the plan", "test coverage check", an explicit `/rite-vet` invocation, or `--full`. An explicit
  ask always fires.

## Skip (low stakes) when ALL hold
- Single-module, reversible, behavior-shaped-not-structural plan.
- No irreversible-risk contact; blast radius contained to the touched files.
- ≤8 files, no new service/dependency/pattern; the dependency order is trivial or single-slice.
- Acceptance criteria already map cleanly to the slices (nothing obviously untested).

On skip, **don't run the passes**. Write only the one-line verdict to `eng-review.md`:

```markdown
# Eng review: <slug>
Significance: skipped — low stakes (<the trigger that was NOT met, e.g. "single-module, reversible, ≤8 files, no new pattern">)
Next: /rite-build
```

Then recommend `/rite-build`. A skip is a *recorded* decision, not a silent omission — the seal
can see the call was made deliberately.

## In `/rite-autocomplete`
Autocomplete runs the significance test on every feature after `/rite-define`. On skip → straight
to the build loop (no pause). On fire → run the full review under the AFK gate ceiling:

- **Auto-apply (no pause):** findings that *harden* the plan without growing it — added test
  requirements, added error-handling / failure-mode coverage, tightened scope boundaries, reuse
  of an existing component over a rebuild, dependency-order fixes, parallel-lane annotations.
  These never grow acceptance, so AFK applies them and records the rationale in `decisions.md`.
- **Blocking pause:** any finding that **grows the build's scope, adds a slice, changes an
  acceptance criterion, or alters product behavior** — autocomplete hardens and may *prune*, it
  never *expands* unattended.
- **Always pause:** irreversible-risk findings (migration / auth / public-API / data-model), and
  any axis still below bar after the ≤3-iteration reviewer loop.
- **Cross-model is off by default** in AFK; it runs only if `--cross-model` was explicitly armed.
