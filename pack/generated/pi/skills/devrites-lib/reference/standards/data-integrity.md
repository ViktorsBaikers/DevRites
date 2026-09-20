# Data integrity

> Applies when: a change writes durable state, alters schema, migrates or backfills data.

Load this when a change writes durable state, changes a schema, migrates or backfills
records, changes retention, can expose one tenant's data to another, or feeds a
model/feature pipeline (training data, features, labels, embeddings). Data work is
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

Timestamps normalize before they persist: one storage scale (UTC instants), explicit
conversion only at input/display boundaries; two writers storing different scales for
the same fact is an invariant violation caught in review. TZ/DST behavioral coverage
lives in [`testing.md`](testing.md).

An invariant enforced only by prose is not a control. Prefer a database constraint or
atomic storage primitive, then add behavioral proof at the public surface.

## Model and feature pipelines

Durable-state rules apply to ML-adjacent writes with their own failure modes:

- **Data contract:** producers and consumers bind to a versioned schema plus quality rules
  owned beside the pipeline. Two stages reading one field under different implicit schemas
  is an invariant violation caught like any schema drift.
- **Point-in-time correctness:** a feature may use only data available at prediction time.
  Label leakage through later-arriving data invalidates the artifact silently. **Failing
  case:** a feature computed from post-prediction-time data passes review; a backfilled
  label join reads as valid history.
- **Training/serving skew:** the same feature computed two ways is a behavioral difference
  to test, not a deployment note; drift monitoring rides the observability owner
  ([`observability.md`](observability.md)).

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

**Green types ≠ applied schema.** When the change touches ORM models, schema files, or
migration directories, `compiles`/`typechecks` prove nothing about the live store —
types can generate from config while the database is still old. The plan carries the
apply step (migrate/push) as an explicit blocking task; proof records the post-apply
state, not just the passing build.

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
- Prove cross-tenant denial with two distinct principals/tenants and existing data.
  An allowed control must reach the resource; the other principal must get no
  forbidden disclosure or effect. Two 404s against a missing fixture prove nothing.
- Minimize collected and returned fields. Define deletion/retention behavior for
  primary data and derived copies, and do not claim deletion while recoverable copies
  remain without a documented policy basis.
- Never place secrets or sensitive records in migration logs, rejected-row dumps, or
  evidence artifacts.

## Restore and replay preserve current policy

Before restored or replayed data becomes visible, reconcile it with current
retention/deletion records and authorization policy. Old backups, events and
indexes must not resurrect deleted records or revoked grants. Name the durable
deletion/revocation source and how recovery catches up with concurrent changes;
if reconciliation cannot be established, keep recovered data inaccessible.

Rehearse a snapshot/event from before deletion and revocation, then restore or
replay it under current policy. Assert the deleted record stays absent, a
revoked principal is denied, and an authorized control still reaches retained
data. Counts or successful restore commands alone do not prove these invariants.

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
  **Failing case:** GO with "revert if needed" and no rehearsal command or before/after
  counts → NO-GO.
- No data-loss or cross-tenant risk may be dismissed as "pre-existing" without baseline
  evidence from before the candidate.
