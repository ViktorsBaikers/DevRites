# Deprecation & migration

Code is a liability, not an asset. Every line carries ongoing cost — bugs to fix, dependencies
to patch, security to maintain, the next engineer to onboard. Most teams build well and remove
badly, so dead and superseded code accumulates. Removing code that no longer earns its keep, and
moving users safely off the old path, is its own discipline — and a riskier one than writing new
code, because the failure mode is breaking something you forgot depended on it.

DevRites already **gates** the dangerous moves: destructive migration, auth/authz change,
public-API break, and data-loss paths always pause ([`afk-hitl.md`](afk-hitl.md) irreversible-risk
list). The gate stops you; this rule is the safe path it stops you to take.

## Code is a liability
Unused code is pure cost with no return — deleting earned-out code is a feature, not housekeeping.
But "I think this is unused" is a guess; prove it (below) before you act on it.

## Hyrum's law — observable behavior is the contract
With enough consumers, *every* observable behavior is depended on by someone — not just the
documented API, but the quirks: timing, error shapes, ordering, the off-by-one nobody noticed.
Removing or changing a behavior you think is incidental can break invisible dependents. Assume a
consumer relies on it; verify against real usage rather than the spec's intent.

## Prove it's unused before you remove it
- Find the dependents: callers, subscribers, stored data, external clients, scheduled jobs. Use a
  code-intelligence index for blast radius ([`tooling.md`](tooling.md)), then confirm against
  **runtime** usage — a log/metric on the path proves zero traffic in a way static search can't
  ([`observability.md`](observability.md)). No-usage-confirmed beats no-usage-assumed.
- If you can't prove zero usage, you're not removing — you're deprecating (below).

## Expand → contract (parallel change)
Never big-bang a breaking change. Three independently-shippable, independently-reversible steps:
1. **Expand** — add the new path alongside the old; both work.
2. **Migrate** — move consumers and data to the new path; watch the old path's usage fall to zero.
3. **Contract** — remove the old path *only once telemetry confirms it's unused*.

The same shape applies to data: add column → backfill → switch reads → drop the old column. Every
destructive step has a rollback, or it doesn't ship.

## Deprecate before delete
- Mark the old path deprecated with a pointer to the replacement and a **removal trigger** — a
  date or a condition ("when v1 traffic hits zero"), not a vague "soon". A deprecation with no
  trigger is a permanent TODO.
- Emit a usage signal on the deprecated path so the removal trigger is a measured fact, not a
  hope ([`observability.md`](observability.md)).

## Scope
A removal or migration is a feature with its own spec, slices, and evidence — not a drive-by
delete while you're in another change. A surprise deletion in an unrelated diff is a red flag
([`anti-patterns.md`](anti-patterns.md)), and a destructive step trips the seal's risk-and-rollback
check regardless.
