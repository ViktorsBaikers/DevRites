# Deprecation & migration

Code is a liability, not an asset. Every line carries ongoing cost: bugs to fix, dependencies
to patch, security to maintain, the next engineer to onboard. Most teams build well and remove
badly, so dead and superseded code accumulates. Removing code that no longer earns its keep, and
moving users safely off the old path, is its own discipline, and a riskier one than writing new
code, because the failure mode is breaking something you forgot depended on it.

DevRites already **gates** the dangerous moves: destructive migration, auth/authz change,
public-API break, and data-loss paths always pause ([`afk-hitl.md`](afk-hitl.md) irreversible-risk
list). The gate stops you; this rule is the safe path it stops you to take.

## Code is a liability
Unused code is pure cost with no return: deleting earned-out code is a feature, not housekeeping.
But "I think this is unused" is a guess; prove it (below) before you act on it.

**Design for removal.** The cheapest deprecation is the one you planned when you built the thing.
When adding a new system, ask "how would we remove this in three years?": the answer forces a seam
(a flag, an adapter, a single call site) that makes the eventual exit a contraction instead of a
surgery. Code built with no exit in mind is what becomes a zombie (below).

## Hyrum's law: observable behavior is the contract
With enough consumers, *every* observable behavior is depended on by someone, not just the
documented API, but the quirks: timing, error shapes, ordering, the off-by-one nobody noticed.
Removing or changing a behavior you think is incidental can break invisible dependents. Assume a
consumer relies on it; verify against real usage rather than the spec's intent.

## Prove it's unused before you remove it
- Find the dependents: callers, subscribers, stored data, external clients, scheduled jobs. Use a
  code-intelligence index for blast radius ([`tooling.md`](tooling.md)), then confirm against
  **runtime** usage: a log/metric on the path proves zero traffic in a way static search can't
  ([`observability.md`](observability.md)). No-usage-confirmed beats no-usage-assumed.
- If you can't prove zero usage, you're not removing. You're deprecating (below).

## Expand → contract (parallel change)
Never big-bang a breaking change. Three independently-shippable, independently-reversible steps:
1. **Expand:** add the new path alongside the old; both work.
2. **Migrate:** move consumers and data to the new path; watch the old path's usage fall to zero.
3. **Contract:** remove the old path *only once telemetry confirms it's unused*.

The same shape applies to data: add column → backfill → switch reads → drop the old column. Every
destructive step has a rollback, or it doesn't ship.

## Deprecate before delete
- Mark the old path deprecated with a pointer to the replacement and a **removal trigger**: a
  date or a condition ("when v1 traffic hits zero"), not a vague "soon". A deprecation with no
  trigger is a permanent TODO.
- Emit a usage signal on the deprecated path so the removal trigger is a measured fact, not a
  hope ([`observability.md`](observability.md)).

## The Churn Rule: you own the deprecation, you own the migration
If you own the thing being deprecated, moving the consumers off it is *your* job, not theirs.
"They'll migrate on their own" is how a deprecation stalls advisory for years. Either do the
migration for them (a codemod, a backfill, a config flip they don't have to think about) or ship
the change as a backward-compatible update that needs no migration at all. Announcing a deadline
without providing the tooling, docs, and support to hit it isn't a migration; it's a break with a
countdown.

## Zombie code: nobody owns it, everybody depends on it
The dangerous middle state is not dead code (safe to remove) but **zombie** code: no one maintains
it, yet live consumers still call it. Signs: no commits in 6+ months, an active caller graph, no
named owner, failing tests nobody fixes. A zombie can't stay in limbo: it either gets **invested
in** (an owner, green tests, a real place in the architecture) or gets **removed** on the
prove-unused → expand→contract path above. Leaving it untouched is choosing the worst option: an
unmaintained dependency that breaks under someone who never signed up to fix it.

## Scope
A removal or migration is a feature with its own spec, slices, and evidence, not a drive-by
delete while you're in another change. A surprise deletion in an unrelated diff is a red flag
([`anti-patterns.md`](anti-patterns.md)), and a destructive step trips the seal's risk-and-rollback
check regardless.
