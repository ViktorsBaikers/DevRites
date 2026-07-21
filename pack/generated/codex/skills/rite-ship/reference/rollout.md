# Staged rollout & rollback — when the ship includes a live deploy

Opt-in. `$rite-ship`'s job ends at the git ladder (commit → push → tag / PR) and archiving the
workspace; most projects deploy from CI on merge, and their pipeline owns the rollout. Reach for
this reference only when the ship *itself* drives a live, staged production rollout the agent is
responsible for. When it doesn't apply, skip it — the same no-op discipline as the rest of the pack.

The governing idea is the one already in [`git-workflow.md`](../../devrites-lib/reference/standards/git-workflow.md) and
[`deprecation.md`](../../devrites-lib/reference/standards/deprecation.md): **a launch is only done when it's reversible,
observable, and incremental.** Write the rollback plan *before* you deploy, not after it breaks.

## The rollback plan is a pre-condition
Before the first byte ships, the plan exists, with a measured **time-to-rollback** per mechanism —
feature flag < 1 min, redeploy < 5 min, DB rollback < 15 min. If the fastest reversal is slow, that
is a launch risk to fix (add a flag) before shipping, not after. A destructive/migration step ships
only with its rollback proven (expand→contract, [`deprecation.md`](../../devrites-lib/reference/standards/deprecation.md)).

## Advance on evidence — the rollout decision thresholds
Stage the exposure and, at each stage, read the signals ([`observability.md`](../../devrites-lib/reference/standards/observability.md))
against a fixed table — advance on green, hold on yellow, roll back on red. Don't eyeball it.

| Signal | Green (advance) | Yellow (hold, investigate) | Red (roll back now) |
|---|---|---|---|
| Error rate | within ~10% of baseline | 10–100% over baseline | > 2× baseline |
| p95 latency | within ~20% of baseline | 20–50% over | > 50% over |
| Client JS errors | none new | < 0.1% of sessions | > 0.1% of sessions |
| Business metric on the path | neutral or up | < 5% decline | > 5% decline |

## Stage the exposure, with a monitoring window at each step
Internal/team → canary ~5% (hold 24–48h) → 25% → 50% → 100%, advancing only when **all** thresholds
are green and you can still roll back to the previous percentage at any point. Data integrity or a
security regression is an **immediate** rollback regardless of the table.

## Feature-flag hygiene
Every flag has an owner and an expiry; test both states in CI; don't nest flags (state explodes);
remove the flag and its dead branch within ~2 weeks of full rollout — a lingering flag is the
deprecation debt [`deprecation.md`](../../devrites-lib/reference/standards/deprecation.md) exists to prevent.

## Verify in the first hour (a runbook, not a vibe)
Health endpoint returns 200 · the error dashboard is flat · latency within budget · one critical
user flow driven by hand · logs/metrics/traces actually flowing ([`observability.md`](../../devrites-lib/reference/standards/observability.md)
"verify the telemetry fires") · a rollback dry-run confirmed reversible. Record the observations in
`ship.md` / `evidence.md` — an un-watched launch is an unproven one.
