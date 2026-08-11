# Data integrity

Load this when a change writes durable state, changes a schema, migrates or backfills
records, changes retention, or can expose one tenant's data to another. Data work is
complete only when normal operation, interruption, retry, and rollback preserve the
declared invariants.

## Start with invariants and ownership

Name before planning implementation:

- the authoritative owner of each fact and every writer;
- uniqueness, referential, ordering, range, and lifecycle invariants;
- tenant/subject partition keys and authorization boundary;
- transaction boundary and externally visible commit point;
- retention/deletion obligation, including backups, replicas, caches, indexes, and
  derived stores;
- old and new readers/writers that coexist during rollout.

An invariant enforced only by prose is not a control. Prefer a database constraint or
atomic storage primitive, then add behavioral proof at the public surface.

## Migration and backfill path

Use **expand → migrate → contract** for compatibility across deployment units:

1. **Expand:** add backward-compatible storage and make old behavior continue to work.
2. **Migrate:** backfill in bounded, resumable batches with a stable cursor, rate limit,
   progress signal, and reject/error accounting. Re-running a completed batch MUST NOT
   duplicate or corrupt data.
3. **Verify:** reconcile source and target counts plus invariant-specific checks; sample
   records cannot replace whole-population checks for a destructive decision.
4. **Contract:** remove the old path only after all readers/writers have moved and
   runtime evidence shows no remaining consumer.

For a large table, plan lock duration, write amplification, replica lag, disk headroom,
and pause/resume behavior. A migration that is safe on an empty fixture may still be
unsafe at production volume.
Treat a partial migration as an explicit mixed-version state: identify migrated/unmigrated
rows, compatible readers/writers, resume cursor, rejected records, and reconciliation before
contracting the old path.

## Writes, retries, and concurrency

- Make duplicate requests/jobs/events converge on one effect with a durable idempotency
  key or uniqueness rule. A process-local set is not durable deduplication.
- Prevent lost updates with the storage system's atomic operation, transaction, version
  check, or explicit conflict response. "Last write wins" is a product decision, not a
  default.
- When locks are necessary, acquire them in one documented order, bound the wait, and
  keep the locked transaction minimal. A deadlock aborts and rolls back the whole unit;
  retry only the complete idempotent unit, never the half-finished statements. Prove the
  path with two contending actors and an invariant check after one is aborted/retried.
- Keep the transaction as small as correctness permits. If an external side effect
  cannot share the transaction, use an established outbox/inbox or reconciliation
  pattern and define the window where one side has committed.
- On partial failure, record enough durable state to distinguish `not started`,
  `applied`, and `needs reconciliation`. Never blindly retry an unknown outcome.
- Duplicate records need both prevention and repair: name the canonical survivor,
  references to re-point, and an auditable dry-run count.

## Tenant, privacy, and retention boundaries

- Derive tenant/subject scope from authenticated server-side context, not a caller's
  free-form id. Apply it to reads, writes, indexes, caches, jobs, exports, logs, and RAG
  retrieval.
- Prove cross-tenant denial with two distinct tenants and data; a single-tenant happy
  path cannot detect leakage.
- Minimize collected and returned fields. Define deletion/retention behavior for
  primary data and derived copies, and do not claim deletion while recoverable copies
  remain without a documented policy basis.
- Never place secrets or sensitive records in migration logs, rejected-row dumps, or
  evidence artifacts.

## Required plan and proof

For each applicable change, `plan.md` records:

| Invariant/risk | Expand/migrate/contract or write path | Interruption/retry behavior | Rollback/recovery | Proof |
| --- | --- | --- | --- | --- |
| `<what must remain true>` | `<ordered steps>` | `<resume/dedupe/conflict>` | `<restore/reconcile>` | `<test/query/rehearsal>` |

Proof covers the happy write plus invalid input, duplicate/retry, concurrent update,
mid-operation interruption, compatibility with the other deployed version, tenant
isolation when relevant, and rollback or forward-recovery rehearsal. Capture commands,
data scale, before/after counts, rejected rows, invariant results, and observed recovery.

## Fail-closed gates

- No destructive or contract step without verified backup/restore or a documented
  forward-only recovery accepted by the human owner.
- No migration GO with unknown old readers/writers, unresolved invariant violations,
  unbounded backfill, missing interruption state, or no production-scale risk estimate.
- No data-loss or cross-tenant risk may be dismissed as "pre-existing" without baseline
  evidence from before the candidate.
