# $rite-converge anti-patterns

Load the moment you reach for one of these excuses.

## Common rationalizations

- *"I'll just fix this existing slice instead of appending a new one."*. No. The append-only
  rule is what makes convergence **re-runnable** and its output **auditable**: a reader diffs
  `tasks.md` and sees exactly what this pass added. Rewriting an existing slice destroys that
  and can silently drop coverage. Append; never edit an existing `SLICE-###`.
- *"The gap is tiny. I'll just patch the code while I'm here."*. No. Convergence **assesses
  and enqueues**; `$rite-build` implements. The moment you edit application code you've skipped
  the build's slice discipline, tests, and reviewer. Enqueue it, however small.
- *"It's basically built. I'll call it done."*. Happy-path-only, untested, or half-wired is
  **partial**, not built (`convergence-assessment.md`). Rounding partial up to built is how a
  feature ships with the edge cases that were the whole reason for the scenario.
- *"The code does X, the spec says Y. I'll enqueue a slice to make the code match the spec."*:
  Only if you're sure the **spec** is right. If the code is right and the requirement is stale,
  that's **Spec Drift**: route it through the guard, don't launder a spec bug into a code task.
- *"The spec has no `AC-###` ids, so there's nothing to converge."*. Assess plan touch-points
  and existing-slice Produces directly against the code; a coverage id is convenient, not
  required. (And flag the untagged spec so `$rite-spec` can add ids.)
- *"I'll rewrite the whole tasks.md to be clean while I'm at it."*, That is a **reslice**, and
  it belongs to `$rite-plan`, not here. Convergence only ever *appends*.

## Red flags in your own output

- An edit to `tasks.md` that touches a line **above** the appended batch → you rewrote history.
  Revert; append only.
- Any change to `spec.md`, `plan.md`, or a source file in this pass → out of scope.
- An appended slice with no `Satisfies:` line → it traces to nothing; either tie it to a
  criterion/touch-point or it isn't convergence work.
- A "clean" report that still left a `## Convergence` header or marker in `tasks.md` → clean
  must be **byte-for-byte unchanged**.
- Reusing or renumbering an existing `SLICE-###` id → ids are append-only and stable.
