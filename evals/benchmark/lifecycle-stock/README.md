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

The JSON report gives `score` per kind and `cost` from the run's `metrics.jsonl`,
`dispatch.json`, and `questions.md`:

| Field | Meaning |
| --- | --- |
| `wall_hours` | first to last metrics event |
| `agent_minutes` | sum of every returned role's `started` to `returned_at` in `dispatch.json` |
| `active_minutes` | wall time outside open blocking or escalating questions (`raised_at` to `answered_at`) |
| `root_only_minutes`, `root_only_fraction` | active time with no dispatched agent running, and its share of active time |
| `context_bytes` | bytes of every context bundle (`context` metrics events) |
| `dispatches`, `wright_dispatches`, `questions`, `return_outcomes`, `abandoned_waves` | counts; outcomes are the ones recorded at `dispatch return --outcome` |

To add tokens, pass the host's transcript directory for the run's project and
the host: `--transcripts <dir> --host omp|claude`. The grader then adds
`cost.tokens` from
[`scripts/transcript-usage.py`](../../../scripts/transcript-usage.py), which
reads offline and never writes. The block has model, turns, input, cache read,
cache write, output, and cost per role and per wave, plus root-session totals.
Cost is `null` on Claude Code, which records none. The directory is never
guessed. On omp it is the project's folder under the omp sessions directory.
On Claude Code it is the project's folder under `~/.claude/projects`. A layout
the script does not recognize gives `"usage": "unavailable"`.

## Use it

Grade the release before a speed change and the candidate after it, on the same
host and model. One run per side is noisy, so compare mode needs at least three
distinct runs per side:

```bash
python3 -I scripts/grade-lifecycle-benchmark.py --benchmark evals/benchmark/lifecycle-stock \
  --baseline "$BASE1" "$BASE2" "$BASE3" --candidate "$CAND1" "$CAND2" "$CAND3" \
  --baseline-label omp/<model> --candidate-label omp/<model>
```

The labels are optional, but when you give one you must give both, and they
must match. The report gives probe passes per side and the median, min, and max
of each numeric `cost` field. The verdict is `accept` only when every probe
passed in every candidate run, every run on both sides has a metrics ledger, and
the median `context_bytes`, `agent_minutes`, or `root_only_minutes` dropped. Keep the change only on `accept`. `reasons`
says why a run was rejected.

`reference/fixed` is a correct solution and `reference/naive` meets the acceptance
text while keeping every seeded defect. `tests/lifecycle-benchmark-grader-test.sh`
proves each probe fails alone on its own defect and passes on the fix. Never copy
`probes/` into a run: agents that can see the probes are no longer being measured.
