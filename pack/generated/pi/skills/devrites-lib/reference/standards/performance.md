# Performance

Measure first. An optimization without a measurement is a guess that adds complexity.

## Measure before you optimize
- Establish a number: a timing, a query count, a payload/bundle size, a memory figure:
  against a budget or a baseline.
- No measurement → no performance claim, and usually no change. "Feels slow" is a
  hypothesis to test, not a reason to refactor.

## Optimize responsibly
- Fix the **measured** bottleneck, then **re-measure** to prove the win (before/after).
  An optimization that doesn't move the number is just added complexity: revert it.
- **Measurement not reproducible in CI** (noisy host, external dependency): label the claim
  `Lab (<named command/environment>)` — never an elapsed-time assertion in shared CI (a
  flaking wall-clock test is a flaky test, [`testing.md`](testing.md)). Budget regression:
  re-measure; fix to budget or record the accepted regression with reason and owner.

## Frontend: Core Web Vitals
For UI work, measure-first means LCP / INP / CLS judged against real numbers, each labeled
by source (`Field (CrUX)`, `Lab (Lighthouse)`, `Trace (DevTools)`): field and lab are not
interchangeable, and static source cannot measure a CWV. The reviewer captures these via the
browser-proof ladder when a budget exists, then judges them in Measured mode (a
source-labeled scorecard); with no artifact it runs in Source mode and names the command.
Baseline checks + measurement commands:
[`rite-review/reference/performance-checklist.md`](../../../rite-review/reference/performance-checklist.md).

## Scope
Optimize what the change touches or what a measurement flags. Project-wide performance
work is its own effort: record it as a follow-up, don't smuggle it into an unrelated
change.

## Unbounded work (failing cases)

These are performance defects even before a budget exists. Name the bound or
record `cannot_verify` with the missing measurement.

- **N+1 / fan-out:** a list or handler that issues one query/call per item
  with no cap, batch, or pagination. **Failing case:** a 10-row fixture is
  green; 10k rows time out in production.
- **Unbounded render:** a view that mounts the full collection with no
  windowing, pagination, or virtualization when the set can grow.
- **Cache without invalidation:** a cache write with no TTL, explicit
  invalidate-on-write, or stampeded-miss plan. **Failing case:** a stale
  read is the only proof the cache "works."
- **Environment skew:** a lab number from a local SSD or empty dataset
  labeled as field/production evidence ([`testing.md`](testing.md) elapsed-time
  rule still applies). Re-measure on the named environment or keep the `Lab`
  label.
- **Unmeasured hot path:** an optimization on a path with no before-number.
  Revert; it is complexity.
