# Convergence assessment: built / partial / absent

The rubric `/rite-converge` applies to every unit of intent. Load when running step 2-5.

## The unit set

Assess three kinds of unit against the live code:

1. **Acceptance criteria / scenarios:** each buildable `AC-###` and each `#### Scenario:`
   (WHEN/THEN) in `spec.md`. `## Success metrics` lines are **not** units. They carry no
   `AC-###` id, no slice can satisfy them, and this pass never enqueues one (`spec-grammar.md`).
2. **Plan touch-points:** the files/components/modules `plan.md` says get built or edited, and
   each existing slice's stated **Produces** (the interface it was meant to expose).
3. **Principles:** each invariant in `.devrites/principles.md`, checked against the code as it
   stands.

## The three verdicts

For each unit, decide **built / partial / absent**, and only *partial* and *absent* enqueue a
slice.

| Verdict | Test | Action |
|---|---|---|
| **built** | The behavior exists in the code **and** a test covers it (a criterion with code but no covering test is not built: `testing.md`). | Nothing: do not append. |
| **partial** | Some of it exists: happy path only, un-wired, untested, a TODO stub, or an edge case from the scenario's WHEN/THEN missing. | Append a slice for the **remainder**, its `Known-Gotchas` naming what's already there so `/rite-build` extends rather than rebuilds. |
| **absent** | No code implements it. | Append a full slice. |

Read the code to decide: do not infer from the plan. Apply `standards/tooling.md`: use the
primary available code index, cross-check only a named unresolved predicate, then use LSP/file
search so the verdict reflects live code. Read `traceability.md` to see what is mapped; only
code and tests tell you what is built.

Treat accepted invariants, failure/recovery rows, and triggered applicability obligations as
buildable units alongside REQ/AC/scenarios. A happy path with missing applicable
duplicate/retry/concurrency/tenant/timeout/order/interruption/rollback behavior is `partial`.

## Principle violations

A live violation of a declared invariant (secret in logs, money as float, a datastore added
with no ADR) is the **top-severity** gap regardless of coverage. It sorts **first** in the
appended batch as a remediation slice, and (because reshaping to honor an invariant can be an
irreversible-risk change) it pauses for a human even under AFK. Absent/empty principles file →
none declared → skip, never block.

## Spec drift vs code gap

Before appending a slice, separate the two failure directions:

- **Code gap:** the spec is right, the code falls short. → append a slice. (This skill's job.)
- **Spec drift:** the code is right, the requirement is stale or wrong. → **do not** append a
  slice that bends correct code to a wrong spec. Stop, route through the Spec Drift Guard
  (`rite-build/reference/spec-drift-guard.md`) + a recorded decision, and let `/rite-plan
  repair` reshape intent. Convergence enqueues work; it never launders a spec bug into a code
  task.

## Numbering + ordering the appended batch

- **Continue the ids:** the highest existing `SLICE-###` + 1, ascending. Never reuse or
  renumber an existing id.
- **Order after the existing slices**, dependency-first among the new ones (`depends_on`);
  a principle-remediation slice sorts before feature gaps.
- **Mark each** with `Convergence: <iso>` so a later reader (and `/rite-status`) can tell an
  original slice from a converged-in one.
- **Nothing unmet → append nothing.** No batch header, no marker, `tasks.md` byte-for-byte
  unchanged.
