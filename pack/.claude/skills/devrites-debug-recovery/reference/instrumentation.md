# Instrumentation

Each probe maps to a specific prediction from the hypothesis phase.
**Change one variable at a time.**

## Tool preference

1. **Debugger / REPL inspection** if the env supports it. One breakpoint beats ten logs.
2. **Targeted logs** at the boundaries that distinguish hypotheses.
3. **NEVER** "log everything and grep".

## Tagged prefixes

Tag every debug log with a unique prefix, such as `[DEBUG-a4f2]`, so one grep
locates every temporary log for removal during cleanup.

## Perf branch

For performance regressions, logs are usually wrong. Establish a baseline
measurement (timing harness, `performance.now()`, profiler, query plan), then
bisect. **Measure first, fix second.**
