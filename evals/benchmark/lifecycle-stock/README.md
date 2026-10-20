# Lifecycle benchmark: stock release

One fixed feature request against a small Go module with three seeded defects in
the code the feature touches. It measures what a speed change to the lifecycle
costs or saves, and whether quality held while it did.

| ID | Kind | What the hidden probe checks |
| --- | --- | --- |
| A1, A2 | acceptance | `Release` returns reserved stock; an over-release fails with `ErrOverRelease` and changes nothing |
| D1 | seeded | reserving exactly the remaining stock succeeds (the fixture uses `>`) |
| D2 | seeded | a zero or negative quantity is refused (the fixture accepts it) |
| D3 | seeded | concurrent callers never race or corrupt counts (the fixture has no lock; run with `-race` when a C compiler exists) |
| S1, S2 | scope | `stock/loader.go` and `go.mod` stay byte-identical (no drive-by edit, no new dependency) |

The task in [`task.md`](task.md) names none of the defects. A lifecycle that reads
the code it changes, takes "shared by concurrent request handlers" seriously, and
validates its inputs finds them; one that only satisfies the literal request does not.
`stock/loader.go` swallows a decode error on purpose. Fixing it is out of scope, and
S1 fails a run that does.

## Run it

The run starts a paid native session, so a human runs it; CI never does.

1. Copy the fixture to a fresh directory and commit it as the baseline:
   `cp -R evals/benchmark/lifecycle-stock/fixture "$RUN" && git -C "$RUN" init -q && git -C "$RUN" add -A && git -C "$RUN" commit -qm baseline`.
2. Install DevRites into `$RUN` for the host under test, then start the host there.
3. Run `/rite-autocomplete` with the text of `task.md` as the idea. Answer the
   clarification window and nothing else.
4. Grade it:
   `python3 -I scripts/grade-lifecycle-benchmark.py --benchmark evals/benchmark/lifecycle-stock --run "$RUN"`.

The JSON report gives `score` per kind and `cost` from the run's `metrics.jsonl`
and `questions.md`: wall hours, agent minutes, dispatches, wright dispatches,
questions asked, reviewer outcomes recorded at `dispatch return --outcome`, and
abandoned waves.

## Use it

Grade the release before a speed change and the candidate after it, on the same
host and model. Keep the change only when `acceptance`, `seeded`, and `scope` hold
and `cost` drops. One run per side is noisy; compare at least three.

`reference/fixed` is a correct solution and `reference/naive` meets the acceptance
text while keeping every seeded defect. `tests/lifecycle-benchmark-grader-test.sh`
proves each probe fails alone on its own defect and passes on the fix. Never copy
`probes/` into a run: agents that can see the probes are no longer being measured.
