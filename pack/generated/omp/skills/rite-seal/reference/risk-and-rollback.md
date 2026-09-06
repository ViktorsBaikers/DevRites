# Risk & rollback

Before GO, know how to undo the change and what could go wrong in production.

## Risk scan
Rank risks; for each, note likelihood × impact and any mitigation:
- **Topology/ownership**: affected roots/deployables, shared contracts/state/resources,
  dependency cycles, and intermediate deployment combinations.
- **Data**: schema changes, migration/backfill interruption, concurrent/duplicate writes,
  tenant/retention boundaries, destructive operations, and invariant reconciliation.
- **Security**: new trust boundaries, auth/authz changes, secret handling, new deps.
- **Compatibility**: API contract changes, breaking changes for existing clients,
  feature interactions.
- **Integration/operational**: timeouts/unknown outcomes, retry/idempotency, duplicate/order,
  partial/invalid responses, provider/queue/cache outage, config/env, and rate limits.
- **Delivery**: schema/config/application/worker/flag order, environment mismatch,
  monitoring gap, exposure stages, and time to detect/recover.
- **UX**: changed flows users rely on; unverified UI.

## Rollback plan (required for risky changes)
For each risky step, state how to back it out:
- **Migrations**: is there a reversible `down` / a documented manual revert? Is data
  loss possible on rollback? If rollback is unsafe, is bounded forward recovery rehearsed?
- **Feature flag**: can it be disabled without a deploy?
- **Revert boundary**: can the change be `git revert`-ed cleanly, or does it entangle
  with other work?
- **Data**: is there a backup / a way to restore prior state?
- **External/async effects**: can an unknown outcome be reconciled without duplicating it;
  who drains/replays/quarantines work and repairs partial state?
- **Configuration/deployables**: what order restores compatible versions and values at
  every root/environment?

## Blocking rules
- A **destructive or data-migration change with no rollback plan** is a **NO-GO**.
- A data change with unknown invariant/tenant impact, no interruption/retry proof, or no
  rollback/forward-recovery rehearsal is **NO-GO**.
- A new/changed external or asynchronous boundary with blind retry, silent partial success,
  or no outage/backlog recovery and monitoring is **NO-GO**.
- Unsafe intermediate deployment order, unvalidated required configuration, or no watched
  rollout signal/owner is **NO-GO** for live exposure.
- Record the chosen rollback path in `seal.md` → "Risks / Rollback".

## Fresh-context availability

A required review must run in a fresh spawned agent:

- If the named role is unavailable, record the roster skip reason and keep seal
  **NO-GO**.
- Never run the reviewer in the root context or log root work as a dispatched review.
- AFK never auto-accepts a missing reviewer.
- Security or irreversible-risk scope remains NO-GO while independent review is
  unavailable.

## Destructive operations
Confirm any destructive step with the user before shipping. Verify backups exist where
relevant. Never treat an irreversible action as routine.
