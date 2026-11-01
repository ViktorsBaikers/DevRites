# Dependency graph

Order slices by what must exist before what. Keep it in `plan.md` as plain text.

## Build the graph
1. For each slice, list its hard prerequisites (a model it queries, an endpoint it
   calls, a component it composes, a contract/schema/config it consumes, or a deployable
   version that must already tolerate the change).
2. Draw edges prerequisite → dependent.
3. Topologically sort local and explicit `Workspace dependencies` producer edges.
   Validate each qualified producer in its live/safely archived tasks graph.
   Inspect assumptions, consumed interfaces, shared runtime resources and proof
   independence before declaring `none`; disjoint paths alone prove neither.
   Only durably BUILT/integrated producer checkpoints unblock consumers. Current
   native Plan/Vet approves early future admission; absent legacy declarations
   require narrow repair, not invented independence. Use the
   [canonical field](skill://devrites-lib/reference/workspace-artifact-schema.md#canonical-slice-grammar),
   not a second lane-policy artifact.

```
SLICE-001 (schema)  ->  SLICE-002 (write API)  ->  SLICE-003 (UI form)
                   \->  SLICE-004 (read API)   ->  SLICE-005 (UI list)
SLICE-006 (config)  -> independent
```

## Within a tier: risk-first
Among slices that are equally unblocked, do the **riskiest** first: the one most
likely to invalidate the plan (new integration, uncertain library behavior, migration).
Finding drift early is cheap; finding it at seal time is not.

## Smells
- A slice with many prerequisites → probably too big; reslice for a thinner cut that
  stands alone.
- A cycle (A needs B needs A) → the boundary is wrong; split the contract
  (`devrites-api-interface`) so one side can land against one canonical contract.
  Do not hide the cycle with duplicated types, a service locator, or an unowned stub.
- Everything depends on slice 1 → slice 1 is doing too much.
- File-disjoint slices share a database migration chain, generated contract, lockfile,
  queue/port, environment, or deployable → serialize that resource or isolate it explicitly.

## Cross-boundary edges
Mark edges that cross a frontend/backend or service boundary. Those slices should
define the contract first (so both sides can proceed) and trigger `devrites-doubt`
before standing the interface.

After editing `tasks.md`, run `devrites-engine check task-graph <slug>` before Vet.
`check readiness` and `check seal` also reject cycles, unknown dependencies,
malformed tokens, duplicate slice IDs, a missing `Dependencies`/`depends_on`
line, and a `depends_on` set that disagrees with `Dependencies`. Cycles or
unknown dependencies block readiness.

For monorepos/multiple repositories, annotate the proven root and deployable on each node.
For data/integration changes, include recovery ordering: expand before new writers,
backfill before contract, consumer compatibility before provider exposure, and monitoring
before rollout. The graph is incomplete if code order is safe but deploy order is not.
