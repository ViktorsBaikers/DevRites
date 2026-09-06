# Golden outcome fixtures

`shippable-feature/` is the only canonical base. It contains every non-empty
artifact required by the current workflow manifest plus the final-only seal and
review files.

`run-outcome-evals.sh` is a deterministic integrated fixture, not a live model
evaluation. It builds one temporary engine, stages the manifest paths, obtains
their content digest from the engine, binds the final artifacts, copies that
base for each negative row, and proves that exactly one causal path changed. The
shell grader owns committed final-artifact checks: GO verdict, canonical checked
acceptance IDs, blockers, review, open questions, and final phase/status.
`check seal` owns deterministic structure, open human gates, and exact candidate
bindings; the runner proves harmless touches pass while candidate byte drift
blocks even when the original mtime is restored. The same runner proves that
every retired top-level semantic command is unknown.

`blocked-feature/` and `near-miss-unproven-ac/` remain compatibility examples.
They must stay NO-GO, but their overlapping failures are not matrix proof.

`out-of-scope-writer-diff/` adds `src/utils/format.ts` to the candidate without
naming it in `tasks.md`. `unauthorized-spec-drift/` adds `AC-004` to `spec.md`
without mapping it into `tasks.md`, `test-plan.md`, or `seal.md`. The runner
must fail both.
