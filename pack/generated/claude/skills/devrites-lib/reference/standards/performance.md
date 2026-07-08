# Performance

Measure first. An optimization without a measurement is a guess that adds complexity.

## Measure before you optimize
- Establish a number: a timing, a query count, a payload/bundle size, a memory figure —
  against a budget or a baseline.
- No measurement → no performance claim, and usually no change. "Feels slow" is a
  hypothesis to test, not a reason to refactor.

## Common pitfalls to look for
- **N+1 queries** and unbounded result sets; fetch what you need, batch, paginate.
- Repeated work in hot paths; cache or hoist computation that doesn't change per call.
- Accidental quadratic loops over growing collections.
- Oversized payloads/assets; blocking work on the critical path; chatty round-trips.
- Synchronous work that blocks the request/UI when it could be deferred.

## Optimize responsibly
- Fix the **measured** bottleneck, then **re-measure** to prove the win (before/after).
  An optimization that doesn't move the number is just added complexity — revert it.
- Don't trade correctness or readability for a micro-win that doesn't matter.
- Prefer a better algorithm or query over micro-tuning; the big wins are structural.

## Frontend — Core Web Vitals
For UI work, measure-first means LCP / INP / CLS judged against real numbers, each labeled
by source (`Field (CrUX)`, `Lab (Lighthouse)`, `Trace (DevTools)`) — field and lab are not
interchangeable, and static source cannot measure a CWV. The reviewer captures these via the
browser-proof ladder when a budget exists, then judges them in Measured mode (a
source-labeled scorecard); with no artifact it runs in Source mode and names the command.
Baseline checks + measurement commands:
[`rite-review/reference/performance-checklist.md`](../../../rite-review/reference/performance-checklist.md).

## Scope
Optimize what the change touches or what a measurement flags. Project-wide performance
work is its own effort — record it as a follow-up, don't smuggle it into an unrelated
change.
