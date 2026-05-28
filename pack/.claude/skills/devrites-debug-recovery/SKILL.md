---
name: devrites-debug-recovery
description: Debug systematically when tests, builds, or runtime/browser checks fail — deterministic feedback loop, reproduce, 3-5 ranked hypotheses, instrument, fix root cause, regression-test. Use when the user says "debug this", "why is it failing", "it broke", "the build is red", "the tests fail". Not for code review or flaky tests (fix the flake first).
user-invocable: false
---

# devrites-debug-recovery — fix the root cause, not the symptom

Disciplined recovery from failures. **NO shotgun edits, NO blanket retries.**

## When to invoke

Loaded by `/rite-prove` (and during `/rite-build`) when something fails. Use
when tests, builds, typecheck, runtime, or browser checks are red and the next
move is unclear.

## The six-phase cycle

1. **Build the feedback loop** — fast, deterministic, agent-runnable pass/fail
   signal. **This is the skill** — be aggressive here.
   See [build-the-loop.md](reference/build-the-loop.md).
2. **Reproduce** — run the loop. Confirm the failure matches the user's report
   (not a nearby failure); capture the **exact error text**; confirm
   reproducibility (or a high enough repro rate for flaky bugs). Do not proceed
   without reproduction.
3. **Ranked hypotheses (3-5, falsifiable)** — generate the list before testing
   any of them. Each must state a prediction.
   See [hypotheses.md](reference/hypotheses.md).
4. **Instrument** — debugger > logs > "log everything and grep". One variable
   at a time. Tagged debug-log prefixes.
   See [instrumentation.md](reference/instrumentation.md).
5. **Fix + regression test** — write the regression test before the fix, but
   only if a correct seam exists. If no correct seam: that IS the finding;
   record it.
   See [regression-test.md](reference/regression-test.md).
6. **Cleanup + classify** — repro gone, debug logs gone, throwaway harnesses
   gone, hypothesis recorded. Classify the failure.
   See [cleanup-and-classify.md](reference/cleanup-and-classify.md).

## Hard rules

- **Quote real error text;** never paraphrase it away.
- **Change one thing at a time** so you know what fixed it.
- **Do NOT loosen / delete a failing assertion** to get green — check whether
  it's drift first (route via `/rite-plan repair`).
- **Do NOT hide flakiness** with sleeps / retries — characterize it.
- **3 failed attempts on the same root cause → escalate**: re-hypothesize from
  scratch, invoke `devrites-doubt`, or ask the user. Don't keep trying
  variations of a wrong idea.
