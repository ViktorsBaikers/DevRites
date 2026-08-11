# Observability

Observability is proof the feature works in **production**: the evidence ladder extended
past your machine. `$rite-prove` shows it works on localhost; observability is how you know
it still works, and why it broke, once real traffic hits it. Un-instrumented code is a claim
you can't verify after deploy.

## Scope: when this applies
Only when the change has a runtime surface worth debugging in prod: a new endpoint/route, a
background job, a queue consumer, an external integration, a user-facing flow, or a new error
path. Skip it for pure-internal refactors, docs, config-only, or type-only changes: the same
scope discipline as [`performance.md`](performance.md). Don't instrument a typo fix.

## The on-call test
The litmus for "is this observable": **if this breaks at 3am, can you tell *what* broke and
*why* from the signals alone, without shipping a new build just to add logging?** If the
answer is no, it isn't done. Instrument the failure path you just wrote, not only the happy
path.

**Write the questions first.** Before instrumenting, name the 2-4 questions an on-call engineer
will ask ("which dependency is down?", "which tenant is affected?"); every signal you add
answers one of them. A signal that maps to no question is noise you pay to store and never read.

## Structured logs
- Log the events you'd need to reconstruct a failure: request boundaries, state transitions,
  external-call outcomes, validation rejections, and authz denials.
- Structured (key/value or JSON), not string soup: a log you can't query is a log you won't
  read. Carry a correlation id (request / trace / job id) so one incident's lines join up.
- **Never log secrets, tokens, or PII** ([`security.md`](security.md),
  [`error-handling.md`](error-handling.md)). Levels mean something: `error` is a page-worthy
  claim, not routine flow.

## Metrics & SLIs
- Cover the signals that page someone: request rate, error rate, latency/duration, and
  saturation of any bounded resource the change adds (a pool, a queue, a cache).
- Emit a counter on the **failure** branch, not just success: an error you don't count is an
  error you can't alert on.
- Name the one Service Level Indicator for the feature's critical path; pin a target (SLO)
  when the project tracks them.
- **Cardinality is the metric failure mode.** A label's values come from a small, fixed set.
  Never label a metric with `user_id`, `email`, `request_id`, a full URL, or raw error text:
  unbounded label values explode the series count and the bill. Group by class (`5xx`, not
  `503`); put the high-cardinality id in a log line or a trace, where it belongs.
- **Percentiles always, averages never.** Read latency as a histogram at p50 / p95 / p99. An
  average hides the tail, and the tail is where the pain (and the SLO breach) lives.

For integrations and asynchronous work, include outcome class and recovery state: timeout,
rate limit, invalid response, retry exhausted, duplicate suppressed, oldest-message age,
backlog depth, poison/quarantine count, and reconciliation lag as applicable. Keep labels
bounded; put provider/request/job identifiers in protected logs or traces.

## Traces (across a boundary)
When a request crosses a service, queue, or async boundary, propagate a trace/correlation id
so the end-to-end path is reconstructable, and span the external call and the slow operation.
A latency regression you can't attribute to a span is a guess.

## Alerts: symptom, not cause
Alert on user-visible symptoms (error-rate spike, SLO burn), not on every internal gauge: a
noisy alert gets muted, and a muted alert is no alert. Every alert names an owner and a first
action, or it's noise.

Two severities, and only two: **page** (a human must act now) and **ticket** (handle it within
hours). A third tier is the noise everyone learns to mute: collapse it into one of the two.

## Verify the telemetry fires (evidence, not assumption)
Instrumentation you added but never watched emit is unproven: the same standing as a test you
never saw fail ([`testing.md`](testing.md) "See it fail first"). Trigger the path, confirm the
log line / metric / span appears, and record the observation in `evidence.md`. "I
added logging" with no observed emission is not done.

Also prove the **monitoring gap is closed**: the signal reaches the dashboard/query or alert
the declared owner actually watches, its threshold is tied to a project baseline or SLO, and
the first recovery action is executable. An emitted metric with no consumer is orphaned
telemetry, not rollout evidence.

## Confirm-before-remove
Telemetry is also how you prove a removal is safe: query real usage before deleting code or a
feature, rather than assuming it's dead ([`deprecation.md`](deprecation.md)). No-usage-confirmed
beats no-usage-assumed.

## Scope discipline
Instrument what the change touches. Retrofitting observability across a whole service is its
own effort: record it as a follow-up, don't smuggle it into an unrelated change.
