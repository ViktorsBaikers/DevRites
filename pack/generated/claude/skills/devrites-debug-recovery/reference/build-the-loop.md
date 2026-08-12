# Build the feedback loop

Start with a fast, deterministic, agent-runnable pass/fail signal. Bisection,
hypothesis testing, and instrumentation depend on it, so spend most debugging
effort on a reliable reproduction loop.

## Build the loop: try these in roughly this order

1. **Failing test** at whatever seam reaches the failure (unit / integration / e2e).
2. **Direct CLI / curl invocation** against the running dev server or process.
3. **Replay:** build a non-sensitive behaviorally equivalent fixture with safe credentials/data; verify the decisive signal matches. Never replay redaction markers. Unknown equivalence is `cannot_verify` plus safe manual steps.
4. **Throwaway harness:** spin up a minimal subset (one service, mocked deps) that triggers the failure with a single function call.
5. **Headless browser script** (Chrome DevTools MCP / Playwright): drives the UI, asserts on DOM/console/network.
6. **Bisection harness:** if the failure appeared between two known states (commit, dataset, version), automate "boot at state X, check, repeat" so `git bisect run` can find it.
7. **Differential harness:** same input through old-version vs new-version (or two configs), diff outputs.
8. **Property / fuzz loop:** if the failure is "sometimes wrong", run 1000 random inputs and look for the failure shape.
9. **Human-in-the-loop, structured:** last resort. If a human must click, drive *them* with a checklist so the loop stays structured. Captured output feeds back.

## Iterate on the loop itself

Once it works, improve it:

- Make it faster: cache setup, skip unrelated initialization, and narrow scope.
- Sharpen its signal: assert on the specific symptom, not only that the process did
  not crash.
- Make it deterministic: pin time, seed the RNG, isolate the filesystem, and freeze
  the network.

Prefer the shortest deterministic loop. A slow or flaky one makes each later
diagnostic step less reliable.

## Wait on a condition

Poll one named observable from fresh state with a bound; timeout reports predicate, bound, and
last value. Fixed delay is only for timing behavior or race reproduction—never readiness proof.

## Non-deterministic failures

Increase reproduction rate instead of waiting for perfection: repeat/parallelize, add stress,
or widen timing until the failure is practical to investigate.

Classify the non-determinism before choosing a tactic:
- **Timing** (race, ordering, async interleave): widen the window. Inject artificial delays at
  the suspect `await`, run under load/parallelism, and pin the scheduler. Use deliberate delays
  when they increase the reproduction rate.
- **Environment** (green here, red in CI/prod): diff the environments: dependency versions, env
  vars, locale, timezone, filesystem case-sensitivity, resource limits.
- **State** (fails only after certain prior runs): hunt a leaked global, singleton, cache, or DB
  row; run the trigger in isolation, then again after the suspect predecessor, and compare.
- **Truly random** (no pattern survives): add defensive logging keyed on the failure signature
  and alert on it in the wild. Gather reproductions before attempting a fix; do not guess
  without evidence.

## When you genuinely cannot build a loop

If no reliable loop exists, stop, list attempts, and ask for reproducing-environment access,
a sanitized HAR/log/dump/timestamped recording, or temporary instrumentation permission. Do
not proceed without a trusted reproduction.
