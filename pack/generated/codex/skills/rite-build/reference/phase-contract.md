# rite-build phase contract

See [`one-slice-cycle.md`](one-slice-cycle.md); candidate lifecycle is
[`candidate-integrity.md`](../../devrites-lib/reference/candidate-integrity.md).

1. **Orient and gate.** Read core, `.devrites/ACTIVE`, `state.md`, and required
   slice artifacts. Require `Implementation readiness: READY`, its current
   `Readiness inputs SHA-256`, all applicable exact reviewer accounts,
   ID-and-meaning traceability, and green proof preflight; run
   `devrites-engine check readiness <slug>`. Any miss/nonzero stops.
2. **Select one target.** Restate goal, acceptance criteria, exclusions, and exact
   source/test paths. Before dispatch, validate relevant `assumptions.md` rows
   against live evidence. If one is disproved and changes acceptance, architecture,
   scope, or proof, use the Spec Drift Guard/plan recovery; a human-owned unknown
   stops. Resolve HITL before source work. AFK answers only permitted gates;
   human-owned or irreversible risk persists a question and stops for `$rite-resolve`.
   If the target is only an exact Vet-ready executable workflow-artifact set under
   the active feature workspace, apply
   [`workflow-artifacts.md`](../../devrites-lib/reference/standards/workflow-artifacts.md)
   instead of treating it as a product slice.
3. **Dispatch or materialize.** Product source/tests apply [`wright-dispatch.md`](wright-dispatch.md).
   Put the smallest exact project-relative source/test path list directly in the task;
   dispatch the exact `devrites-slice-wright` fresh. Root never writes those product
   paths, wright never widens, and a missing profile stops.
   For the workflow-artifact branch, the controlling root writes only the admitted
   `.devrites/work/<slug>/` paths. Before an active journal or target write, it
   compiles the materializer and runs the exact transaction implementation in a
   disposable same-layout fixture covering success, replacement failure, rollback,
   retained-temporary cleanup, and rerun. It then proves the complete atomic set,
   rechecks an
   identical product candidate, records hashes/evidence, and runs narrow Vet. Do not
   dispatch any agent as a substitute writer and do not charge a product slice.

### Executable workflow-artifact branch

The controlling root materializes the exact admitted workflow-artifact set and
**does not dispatch the wright**. This branch never changes product slice state or
the candidate manifest; it returns through narrow Vet before any consumptive action.
4. **Inspect the return.** Wait; compare its file list and `git diff --name-only`
   with the contract. Reject stale, partial, malformed, or out-of-scope work.
   Preserve user work; source restoration uses the same bounded wright.
5. **Challenge stood decisions.** For each stood decision, run exact
   `devrites-doubt-reviewer` fresh/read-only and record accepted or resolved-rejected
   in `decisions.md`. A missing verdict, principle breach, scope change, or
   irreversible risk blocks.
6. **Prove without guessing.** Against the frozen pre-slice candidate, inspect
   test hunks for deletion, skipping/focus, tautology, or weaker expectations. Dispatch exact
   `devrites-test-analyst` on that immutable diff; missing account or adverse criterion
   verdict is Critical. Run only `test-plan.md` repository/CI commands, capture output,
   and recheck `git diff --name-only`. Missing, weakened, red, or source-mutating proof
   is not success.

   Technical failure uses bounded `devrites-debug-recovery`: reapply the host gate
   before an accepted in-slice correction, keep one exact fingerprint/three
   no-progress attempts, and never rerun an unchanged check. A closed reproduction
   is progress; a different Critical/Important invariant gets its own fingerprint.
   Ask humans only for product decisions,
   irreversible risk, or genuinely human-only access.
7. **Record.** After green proof, upsert `touched-files.md`'s authoritative candidate
   manifest from the actual scoped diff with explicit `present`/`deleted` rows;
   update `state.md`, `evidence.md`, and applicable UI/browser evidence. The manifest
   stays mutable until Polish closes it.
   Record stood decisions/dead ends in `decisions.md`; update checked assumptions'
   status/evidence, never leaving a disproved row live. If code reveals a durable
   project rule, propose a reviewed `AGENTS.md`/nearest-doc update, not a scored ledger.
8. **AFK and reply.** Under `afk-discipline.md`, root charges
   exactly once after each green built slice, never below zero; stop before another dispatch at zero, and fail closed
   on malformed budget. Use the reply contract; name the next pending slice or, only
   when all slices are built, `$rite-prove`. Emit no decorative progress renderer or
   dispatch telemetry.

> **Mid-flight discipline.** One writer contract builds one slice. Do not skip
> TDD, widen paths, fold in a second slice, or accept the writer's judgment as
> independent review.
