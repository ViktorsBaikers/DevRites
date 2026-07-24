# Deprecation & migration

Removing behavior is riskier than adding it because hidden consumers may depend on observable quirks as well as documented contracts (Hyrum's law). Destructive migration, public-API breakage, auth changes, and data-loss paths always use the irreversible-risk pause in [`afk-hitl.md`](afk-hitl.md).

## Prove it is unused

Find callers, subscribers, stored data, external clients, and scheduled jobs with code intelligence, then confirm runtime usage through [`observability.md`](observability.md). Static absence alone does not prove zero consumers. If zero usage cannot be proven, deprecate instead of deleting.

## Expand → migrate → contract

Use three independently shippable and reversible steps:

1. **Expand:** add the new path beside the old one.
2. **Migrate:** move consumers and data; observe old-path usage falling to zero.
3. **Contract:** remove the old path only after telemetry confirms zero use.

For data, this may mean add column → backfill → switch reads → drop the old column. Every destructive step needs a rollback.

## Deprecate before delete

- Mark the old path with its replacement and a measurable removal trigger: a date or a condition such as zero v1 traffic.
- Emit a usage signal so the trigger is evidence, not hope.
- The owner of the deprecated surface also owns migration. Provide the codemod, backfill, compatible update, documentation, and support consumers need; a deadline alone is a break with a countdown.

## Design for removal

When introducing a system, keep its exit bounded behind a flag, adapter, or small call surface. An unowned but still-used zombie must either regain an owner and green proof or follow the prove-unused and expand/contract path; do not leave it as an unmaintained dependency.

## Scope

Treat removal or migration as a feature with its own spec, slices, evidence, and risk review. Do not hide deletion inside an unrelated change.
