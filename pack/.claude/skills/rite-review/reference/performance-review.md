# Performance review (feature-scoped)

Apply only when performance is relevant (a stated budget exists) or a regression risk is
visible. Delegates to `devrites-audit perf`. **Measure first** — never optimize on
a hunch.

## Measure before claiming
- Establish a number: a timing, a query count, a payload/bundle size, a render metric.
- Compare against the budget (from `spec.md`) or the pre-change baseline.
- No measurement → no performance claim, and usually no optimization.

## What to look for (feature scope)
- **Backend**: N+1 queries, missing indexes on new queries, unbounded result sets,
  work done per-request that could be cached/batched, sync work that blocks.
- **Frontend (Core Web Vitals)**: LCP (largest content paint), CLS (layout shift), INP
  (interaction latency). Oversized images, render-blocking work, large bundles added,
  unnecessary re-renders.
- **General**: accidental quadratic loops, repeated work in hot paths, large allocations.

## Optimize responsibly
- Fix the measured bottleneck, then **re-measure** to prove the win (before/after in
  `evidence.md`). An optimization with no measured improvement is just added complexity.
- Don't trade correctness or readability for a micro-win that doesn't move the metric.
- Keep it in feature scope; record project-wide perf issues as follow-ups.

## Labels
A perf issue that breaches a stated budget is **Important** or **Critical**; a
speculative micro-optimization is a **Suggestion** at most.
