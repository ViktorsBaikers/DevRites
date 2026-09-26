# Repair and measurement

## Contents

- [One approved repair slice](#one-approved-repair-slice)
- [Simplification repairs](#simplification-repairs)
- [Oracles by finding kind](#oracles-by-finding-kind)
- [Comments in repairs](#comments-in-repairs)
- [Oracles by lane](#oracles-by-lane)
- [Never do this to get green](#never-do-this-to-get-green)
- [False-green defenses](#false-green-defenses)
- [The assembled candidate](#the-assembled-candidate)
- [Performance measurement](#performance-measurement)
- [Measurement windows](#measurement-windows)
- [Benchmark tool](#benchmark-tool)

## One approved repair slice

Only an `implementer` context writes, only inside one approved task's exact paths,
and only after the coordinator recorded the approval.

1. Recheck the task's input file hashes, ownership, requirement, baseline failures
   and approval; re-read the code to edit.
2. Write the smallest meaningful failing regression test or other falsifiable oracle
   for the confirmed defect and run it against the unfixed code. Classify the first
   run: **good red** (fails at the named assertion for the expected reason — quote the
   failing line), **broken test** (setup, import or compile error — not a valid red
   for a logic defect), **false pass** (passes on the unfixed code — the oracle does
   not reach the defect), or **not run** (not discovered, filtered or skipped). Only a
   good red continues; a repair other than a defect fix gets its red from
   [Oracles by finding kind](#oracles-by-finding-kind).
3. Implement the smallest coherent fix. Keep supported contracts and UX unchanged
   except for the exact approved change. No formatting sweep, no speculative
   abstraction. Add comments only as [Comments in repairs](#comments-in-repairs)
   allows.
4. Run the same oracle to green, then the relevant unit, integration, contract,
   end-to-end, static and security checks plus the impact cascade on dependents.
   Capture commands, exit codes and output as evidence.
5. Refactor only while the meaningful tests stay green. Update affected docs,
   schemas, contracts, migrations and lockfiles inside the approved scope through the
   project's canonical generators, never by hand-editing generated output.
6. A fresh `verifier` context verifies the fix and its neighboring failure modes and
   runs the new test against the old implementation in a disposable baseline copy
   (apply the recorded baseline patches) to prove it still catches the original
   defect.
7. Remeasure approved performance targets; compare guardrails and side effects. If
   the slice fails, revert only the agent-owned hunks (preserving user work), record
   the failed attempt, and revise or re-approve material plan changes.

Workers never run git write commands (stage, commit, stash, checkout, reset) and
never leave unaccounted changes; the coordinator compares observed changed paths with
the contract before admitting a patch.

A pure refactor starts with characterization or contract tests and strengthens weak
oracles; never damage correct code just to see red. A simplification follows
[Simplification repairs](#simplification-repairs). Performance work starts with a
reproducible benchmark or query-budget oracle; ordinary unit tests do not assert
flaky wall-clock times — use controlled benchmarks plus deterministic query counts or
complexity bounds. When a bug cannot be reproduced safely, record the exact missing
fact, propose a safe alternative and obtain a documented exception where needed; a
waived verification stays a visible gap and may block readiness.

## Simplification repairs

A confirmed [over-engineering](audit-domains.md#over-engineering-and-dead-weight)
finding — delete, inline, reuse, replace with standard library, platform or installed
dependency, remove a dependency, shrink — preserves behavior, so its oracle proves
preservation instead of a defect:

1. **Population evidence**, rerun on the current fingerprint: the search that shows
   the removed element is unused, single-implementation or never varied, with the
   same scope rules as the finding.
2. **Reach:** characterization or contract tests over the behavior that stays. Show
   they reach it: a seeded mutant in the affected code turns them red on a disposable
   baseline copy (this is the good red; quote it). Code with zero references has no
   behavior to reach; the population evidence stands in for the red.
3. **Green on the candidate:** the same tests, the full relevant suite, build,
   typecheck and lint, with discovered and executed counts; for a dependency removal
   also a clean install from the lockfile.
4. **Result:** lines, files and dependencies removed, recorded as evidence. A
   simplification that changes a public, persisted or wire contract is not a
   simplification: it needs its own approved contract task.

## Oracles by finding kind

Every approved task needs a check that fails before the repair and passes after
it. When correct code has no natural red, seed a known fault on a disposable
baseline copy and show the new artifact catches it; when neither exists, the
finding stays audit-only. A zero-mutant, zero-test or zero-link run is never a
pass.

| Repair | Red on the baseline | Green on the candidate |
| --- | --- | --- |
| Defect | Regression test fails at the named assertion | Same test passes |
| Missing or weak test | A seeded mutant (or reverted behavior) in the covered lines survives the old suite; name the mutation tool and version | The new test kills it; diff-scoped mutation score recorded |
| Simplification | [Simplification repairs](#simplification-repairs) | |
| Documentation | The documented command, example, generated reference or link check fails, or `git diff --exit-code` shows drift after regeneration | It runs, compiles or regenerates without a diff |
| Observability | A test with an in-memory log, metric or trace sink finds no record with the required keys (trace ID, stable semantic-convention names) | The record is present; deleting the new line turns it red again |
| CI, build, IaC or container config | The linter or policy tool reports the named rule ID (actionlint, zizmor, hadolint, Checkov, conftest) | That ID is gone and nothing new appears |
| Dependency | The advisory scanner reports the advisory against a pinned database snapshot | Same scanner and snapshot report nothing new; lockfile diff limited to the approved package and its required closure |
| Contract or public API | The compatibility or schema diff (oasdiff, buf breaking, cargo-semver-checks, gorelease, griffe, api-extractor) or a consumer contract test fails | Zero breaking findings, or the approved version bump |
| Operations | A fault test fails: a hanging dependency, SIGTERM during a request, a stopped database, a missing required setting | The call fails within its timeout, the request drains, readiness goes unready while liveness stays up, startup names the setting |
| Flaky test | At least one failure in N ≥ 10 shuffled runs of the unchanged code | Zero failures in the same N runs |
| Performance | [Benchmark tool](#benchmark-tool) verdict or a deterministic budget | |

## Comments in repairs

Default to no new comments: names, types and the regression test carry the meaning.
Add one only when the code cannot show why it is written this way: an invariant, an
ordering, locking or security trap, a workaround for a named external bug (link it),
a measured performance trade-off, or a compatibility constraint. Doc comments on a
public API are the exception when the project's convention requires them (for
example Go exported identifiers); match the neighboring style.

Never write in target code:

- finding, task, run or plan IDs, or the word "overhaul": they belong in the receipt;
- comments that restate the code, or test comments that restate the test name or
  label Arrange/Act/Assert;
- edit narration ("Fixed", "Added to handle", "Now correctly") or what the code did
  before the change; history belongs in the commit, not the source;
- chat residue, guesses stated as fact, stacked hedges, or sales words ("robust",
  "seamless", "comprehensive").

Put the constraint first, in literal words, in one sentence where possible. Keep a
hedge only when it names a real, specific uncertainty ("unverified on Windows").
Leave existing comments alone unless the change makes one false; then correct it.

## Oracles by lane

- **Frontend:** reproduce the behavior — stale or out-of-order responses,
  cancellation, state transitions, optimistic failure and rollback, focus loss, the
  broken keyboard path — not "the component renders". A DOM-only unit test does not
  prove real browser or native-engine behavior.
- **Backend:** the native service, data or concurrency oracle: deterministic
  interleavings, transaction outcomes, idempotent replays, query counts.
- **Boundary:** producer and consumer evidence plus the integrated journey; a mock
  client never proves server compatibility, and separately green sides with
  incompatible mocks prove nothing about the real contract.
- **UI or contract-changing repairs:** compare against baseline journeys, the
  relevant UI states and every compatible consumer; backend unit tests alone never
  approve them.

## Never do this to get green

Delete or skip failing tests; weaken assertions; update snapshots blindly; suppress
diagnostics; reduce workloads; loosen quality gates, retries, timeouts, tolerances or
statistical thresholds; mock away the failure; change expected outcomes to match the
patch. Quarantine an unrelated flaky test only with baseline evidence and explicit
user authorization. A justified correction to a test or harness needs independent
evidence that the old oracle was wrong, a versioned explanation, baseline and
candidate re-evaluation, and approval when the acceptance meaning changes.

## False-green defenses

Record test discovery and selection as well as the exit status. None of these can
satisfy a required oracle:

- zero tests discovered or executed; an unintended filter, shard or "only changed"
  mode that silences "no tests found";
- skipped, pending or expected-failure cases counted as passes; a runner summary that
  counts flaky or skipped tests as success;
- cached output from different inputs, or reused output where fresh execution is
  required;
- a subprocess failure swallowed by a wrapper, or a pipeline that masks the exit
  status (`cmd | tail` without `pipefail`);
- a snapshot run that only created missing baselines; a replay command that exits 0
  with nothing to replay;
- an unsupported platform or a test that matches implementation details without
  exercising the contract.

Confirm that a seeded failing example actually reaches the assertion, and keep both
the observed failure and the candidate's observed pass. For flaky checks, record
every attempt, its environment and reset conditions, and the first-run outcome: a
pass only after retry is a flaky observation, not proof that the defect is gone.

## The assembled candidate

Test the integrated candidate, never only each worker's isolated result. Before
combining patches, check they apply cleanly together (for example `git apply --check`
or `git merge-tree` on copies). When individually green patches fail together, bisect
the combination to find the interacting pair; fix the interaction under existing task
authority when covered, otherwise propose a plan amendment. Keep the last
independently verified candidate reference and a per-slice patch and evidence history
in the run area — never as commits in the user's repository. Roll back only
attributable changes whose ownership is verified; never jump back to an earlier high
score or erase a failed experiment. An edit after final verification invalidates the
affected final evidence.

## Performance measurement

The goal is substantial wins on genuine bottlenecks — preferably at least 2×, ideally
3× or more — never a promise for every path. First establish the baseline and the
bottleneck's share of the user-visible outcome. If profiling does not support the
requested gain, or it would break correctness or UX, or exceed the approved risk, show
the evidence and ask the user to approve an alternative target, a broader intervention
or no change; never lower the target quietly. A correctness or security fix needs no
speedup to be valuable, and a trivial absolute gain is never sold as transformational
through a flattering percentage.

A versioned benchmark contract (`revisions/bench-<id>-r<N>.json`) fixes, before any
comparison: baseline and candidate fingerprints; one harness revision for both; build
mode, flags and compiler/runtime/database versions; hardware, OS, resources,
background load, network, browser and device; datasets with cardinality, skew, seeds
and correctness checksums; scenario, concurrency and arrival model (open or closed),
warm/cold cache policy and reset procedure; primary metric and direction, percentiles
or throughput criterion, absolute user or SLO targets, the 2× minimum and 3× stretch
where justified, materiality threshold and regression tolerances; warmup, repetitions
and the statistical method; raw-sample location; error and timeout treatment; stop
conditions.

Run arms interleaved (ABAB or randomized blocks) on the same host, never all
baseline runs under one system state and all candidate runs under another. Warm JIT
runtimes fairly and measure cold start separately when it matters. Use production
builds for production-facing claims. Profile during benchmarking to confirm what is
measured. Include realistic service, database and network work; never drop failures
or slow requests to improve an average. Choose the load model on purpose, account for
coordinated omission and dropped work, and report offered versus completed load; a
result at lower load or a higher error rate is not an equivalent speedup.

```text
lower is better:   speedup = baseline / candidate    reduction% = 100 × (baseline − candidate) / baseline
higher is better:  gain    = candidate / baseline    increase%  = 100 × (candidate − baseline) / baseline
```

A 2× latency speedup is a 50% reduction, not 200%. Report absolute values and
uncertainty with every ratio. Zero or near-zero baselines, or values at the timer's
resolution, get no ratio. Never average unlike ratios into an "overall speedup"; a
workload-weighted aggregate needs predeclared weights. The resampling unit is one
independent invocation — never inner iterations. Use at least 10 invocations per
arm and more when the interval is wide. The benchmark tool below reports the ratio of
medians with a seeded 95% bootstrap interval: claim "at least k×" only when the
interval's lower bound is at least k; an interval spanning 1 or k is inconclusive
and is never rounded up. Confirm a selected win on fresh runs; do not try variants
until one lucky sample passes. An unmeasured change earns no credit, and an
optimization with no measurable benefit is reverted (agent-owned hunks only) and
logged so it is not retried.

For databases keep query counts, plans where safe, rows scanned and returned, buffer
and IO behavior, and write, storage and lock costs. For UI keep critical journeys,
render and interaction traces and metrics under matched fixtures and devices, and
label every number:

| Signal | Measures | Does not establish |
| --- | --- | --- |
| Field p75 (CrUX or first-party RUM) | Real users over a window, by route and device | Effect of an undeployed change; routes without data (unavailable ≠ pass) |
| Lab trace | One session under stated conditions; LCP parts, INP phases, long tasks | How common the problem is in the field |
| Lighthouse score | One synthetic navigation, version-weighted | User experience, INP, cross-version comparison |
| TBT | Lab main-thread blocking during load | Field INP or post-load responsiveness |
| CLS | Unexpected layout-shift clusters | Latency; the shift initiator |
| Bundle size | Bytes per route | Runtime cost unless runtime metrics move |
| Automated a11y scan | A subset of criteria (`incomplete` is not a pass) | Conformance, keyboard or screen-reader usability |
| Static inspection | A hypothesis | Any metric value |

Never deploy to obtain measurements. Keep frontend, backend and boundary benchmark
views tied to one workload graph: a 3× faster query is not a 3× faster page until the
page is measured; a faster server never excuses a slower client or a broken keyboard
journey; CLS never gets a latency ratio. Track guardrails for correctness, security,
stale-data behavior, memory, CPU, write latency, tail latency, errors, accessibility
and visual or interaction stability; surface any trade-off and get approval. Explain
a smaller end-to-end gain with the bottleneck analysis.

## Measurement windows

Review agents may run concurrently; measurements may not share CPU, browser,
database, network or caches with other tests or agents. Record the isolation evidence
(what else was running, load average, governor). When isolation is unavailable,
disclose the contamination, mark the comparison inconclusive and rerun in a
controlled window; a contaminated ratio never passes a gate.

## Benchmark tool

`devrites-engine overhaul bench <result.json>` prints the comparison. Input: `direction` (exactly
`lower_is_better` or `higher_is_better`; anything else is rejected, never guessed),
raw per-invocation `baseline` and `candidate` samples, the recorded run `order`, the
`target` ratio, `materiality`, optional `resolution`, `resamples` and `seed`.
Verdicts: `SUPPORTS_TARGET`, `IMPROVEMENT_BELOW_TARGET`, `IMPROVEMENT`,
`INCONCLUSIVE`, `REGRESSION`, `NO_RATIO`. Fewer than 10 samples per arm, arms not
recorded as interleaved, or a missing interval cap any improvement at
`INCONCLUSIVE`.

The ratio carries two 95% intervals: the seeded bootstrap of the median ratio
(`ci95`) and the distribution-free Hodges–Lehmann ratio bounds (`hodges_lehmann`),
exact from the Mann–Whitney distribution and needing no seed. A claim uses the
smaller lower bound; either upper bound below 1 is `REGRESSION`. A zero or negative
sample, or more than four million baseline × candidate pairs, leaves no
Hodges–Lehmann bounds and caps the verdict. Optional inputs:

- `quantile` (for example `0.99`) compares a tail. Give each invocation as its raw
  request latencies (the tool takes their quantile) or as its own quantile with
  `requests_per_invocation`, never both in one arm. Fewer than `10 / (1 − quantile)`
  requests per invocation (1000 for p99) caps the verdict and any budget PASS. Never pool requests across invocations: the
  invocation stays the resampling unit.
- `tolerance` (for example `0.05`) adds a `guardrail` for metrics that must not get
  worse: `NON_INFERIOR` when the lower bound is at least `1 / (1 + tolerance)`,
  `REGRESSION` when the upper bound is below it, otherwise `INCONCLUSIVE`. "Not
  significant" is never "no regression".
- `budget` (`value`, `quantile` default `0.5`, `deterministic`) tests the candidate
  against an absolute target with one-sided 95% order-statistic bounds: `PASS` only
  when the bound clears the target, `FAIL` when the opposite bound misses it,
  `INSUFFICIENT_DATA` when there are too few invocations for that quantile (p75 needs
  11, p95 59, p99 299). `deterministic: true` (bytes, query or request counts) needs
  every invocation to agree, else `NONDETERMINISTIC`.
- `error_rates` (`baseline`, `candidate` per-invocation rates, absolute `tolerance`,
  optional `requests` totals) guards failures: the candidate-minus-baseline interval
  is the wider of an invocation bootstrap and the Newcombe score interval; `PASS`
  only when its upper bound is within tolerance. Pair every latency claim under load
  with it: fast failures flatter latency.
- `load` (`offered` and `completed` per arm) marks a run where either arm completed
  under 98% of offered load as saturated and caps every verdict.
