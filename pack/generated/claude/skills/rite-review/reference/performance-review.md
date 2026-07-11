# Performance review (feature-scoped)

Apply only when a stated budget or visible regression risk makes performance relevant.
The canonical measurement, pitfall, optimization, and scope rules live in
[`performance.md`](../../devrites-lib/reference/standards/performance.md); apply every
one before filing a performance finding.

The agent runs **Source mode** (static scan, findings tagged `potential` + the verify
command) or **Measured mode** (real CWV numbers → a source-labeled scorecard). Baseline
checks + measurement commands: [`performance-checklist.md`](performance-checklist.md).

## Labels
A perf issue that breaches a stated budget is **Important** or **Critical**; a
speculative micro-optimization is a **Suggestion** at most.
