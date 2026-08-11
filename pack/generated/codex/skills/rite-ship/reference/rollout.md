# Staged rollout and recovery: live deploys only

Load this only when `$rite-ship` itself is explicitly authorized to drive a live
deployment. Git-only Ship stops before rollout; CI-owned deployment follows the pipeline's
runbook. A Seal GO or AFK setting never authorizes production action.

## Preconditions before exposure

Record one rollout sheet in `ship.md`/`evidence.md`:

| Item | Required decision/evidence |
| --- | --- |
| Units and order | Repository/deployable, schema, config, application, worker, contract, and flag order; safe old/new combinations. |
| Exposure stages | Project-native internal/canary/cohort/percentage/region stages and hold window. |
| Advance/hold/abort signals | Project baseline or SLO, measurement window, minimum sample, exact threshold, and owner. |
| Recovery | Fastest safe mechanism, steps, owner, measured/rehearsed time, and data/external-effect reconciliation. |
| Observability | Watched dashboard/query/alert, failure signal, and executable first action. |
| Authorization | Exact target/action approved for this attempt; no inferred retry permission. |

Do not import generic percentage, latency, error-rate, or time-to-rollback numbers. Use
accepted product risk, current baseline/SLO, traffic volume, and platform capability. If the
project has no defensible threshold or signal, the monitoring gap blocks live exposure.

## Choose the smallest reversible mechanism

- A feature flag is useful only when the off path preserves current behavior, both states
  are tested, disabling it stops the risky effect, and it has an owner/removal trigger.
  Do not add a flag to a change already reversible by a safe atomic deploy.
- A flag cannot reverse destructive data/schema effects. Apply
  [`data-integrity.md`](../../devrites-lib/reference/standards/data-integrity.md) and prove
  restore or forward recovery separately.
- External APIs, webhooks, queues, jobs, and caches apply
  [`integration-reliability.md`](../../devrites-lib/reference/standards/integration-reliability.md):
  reconcile unknown outcomes, drain/quarantine/replay safely, and protect downstream capacity.
- Multi-root/service rollout follows
  [`repository-topology.md`](../../devrites-lib/reference/standards/repository-topology.md);
  references to another repository never grant write/deploy authority there.

## Stage, observe, decide

At each authorized stage:

1. Verify the intended versions/config/schema and candidate identity on the exact target.
2. Exercise one critical success path and the declared degradation/recovery signal.
3. Observe for the recorded window and sample; bind results to the stage and baseline.
4. **Advance** only when every advance condition holds. **Hold** on ambiguous or
   insufficient evidence. **Abort/recover** immediately on an abort condition, security or
   tenant breach, data-integrity violation, unreconciled duplicate/unknown effect, or loss of
   observability.
5. Re-verify after recovery; record partial effects and reconciliation. A rollback command
   exiting zero is not proof the prior state or data was restored.

Never compress stages because an early sample "looks fine," continue through a monitoring
gap, or retry a failed live action without fresh authorization when the attempt/target changes.

## Completion

Rollout is complete only when full intended exposure meets the recorded window/signals,
telemetry is still watched, migrations/backfills and queues/reconciliation are settled,
documentation matches deployed behavior, and temporary flags/compatibility paths have a
dated removal owner. Otherwise report the exact current stage and remaining risk; do not call
the launch done.
