# Build the feedback loop

**This is the skill.** Everything else is mechanical. If you have a fast,
deterministic, agent-runnable pass/fail signal for the failure, you will find
the cause: bisection, hypothesis-testing, and instrumentation all just consume
that signal. If you don't have one, no amount of staring at code will save you.

**Spend disproportionate effort here. Be aggressive. Refuse to give up.**

## Build the loop: try these in roughly this order

1. **Failing test** at whatever seam reaches the failure (unit / integration / e2e).
2. **Direct CLI / curl invocation** against the running dev server or process.
3. **Replay a captured trace:** save the offending request/payload/event to disk, replay it through the code path in isolation.
4. **Throwaway harness:** spin up a minimal subset (one service, mocked deps) that triggers the failure with a single function call.
5. **Headless browser script** (Chrome DevTools MCP / Playwright): drives the UI, asserts on DOM/console/network.
6. **Bisection harness:** if the failure appeared between two known states (commit, dataset, version), automate "boot at state X, check, repeat" so `git bisect run` can find it.
7. **Differential harness:** same input through old-version vs new-version (or two configs), diff outputs.
8. **Property / fuzz loop:** if the failure is "sometimes wrong", run 1000 random inputs and look for the failure shape.
9. **Human-in-the-loop, structured:** last resort. If a human must click, drive *them* with a checklist so the loop stays structured. Captured output feeds back.

## Iterate on the loop itself

The loop is a product. Once you have *a* loop, ask:

- Can I make it faster? (cache setup, skip unrelated init, narrow scope.)
- Can I make the signal sharper? (assert on the specific symptom, not "didn't crash".)
- Can I make it more deterministic? (pin time, seed RNG, isolate filesystem, freeze network.)

A 30-second flaky loop is barely better than no loop. A 2-second deterministic
loop is a debugging superpower.

## Non-deterministic failures

Goal is **higher reproduction rate**, not a clean repro. Loop the trigger 100×,
parallelise, add stress, narrow timing windows, inject sleeps. A 50%-flake bug
is debuggable; 1% is not: raise the rate until it's debuggable.

**Classify the non-determinism first. The class picks the tactic:**
- **Timing** (race, ordering, async interleave): widen the window. Inject artificial delays at
  the suspect `await`, run under load/parallelism, pin the scheduler. Making it *more* flaky on
  purpose is progress.
- **Environment** (green here, red in CI/prod): diff the environments: dependency versions, env
  vars, locale, timezone, filesystem case-sensitivity, resource limits.
- **State** (fails only after certain prior runs): hunt a leaked global, singleton, cache, or DB
  row; run the trigger in isolation, then again after the suspect predecessor, and compare.
- **Truly random** (no pattern survives): add defensive logging keyed on the failure signature
  and alert on it in the wild. You're gathering repros, not fixing yet: don't guess a fix blind.

## When you genuinely cannot build a loop

**STOP and say so explicitly.** List what you tried. Ask the user for:

- access to whatever environment reproduces it,
- a captured artifact (HAR file, log dump, core dump, screen recording with timestamps), or
- permission to add temporary production instrumentation.

**Do NOT proceed without a loop you believe in.**
