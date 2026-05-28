# Dependency graph

Order slices by what must exist before what. Keep it in `plan.md` as plain text.

## Build the graph
1. For each slice, list its hard prerequisites (a model it queries, an endpoint it
   calls, a component it composes).
2. Draw edges prerequisite → dependent.
3. Topologically sort. Independent slices can be built in any order (or noted as
   parallelizable for the user).

```
Slice 1 (schema)  ──►  Slice 2 (write API)  ──►  Slice 3 (UI form)
                  └─►  Slice 4 (read API)   ──►  Slice 5 (UI list)
Slice 6 (config)  ── independent ──
```

## Within a tier: risk-first
Among slices that are equally unblocked, do the **riskiest** first — the one most
likely to invalidate the plan (new integration, uncertain library behavior, migration).
Finding drift early is cheap; finding it at seal time is not.

## Smells
- A slice with many prerequisites → probably too big; reslice for a thinner cut that
  stands alone.
- A cycle (A needs B needs A) → the boundary is wrong; split the contract
  (`devrites-api-interface`) so one side can land with a stub.
- Everything depends on slice 1 → slice 1 is doing too much.

## Cross-boundary edges
Mark edges that cross a frontend/backend or service boundary. Those slices should
define the contract first (so both sides can proceed) and trigger `devrites-doubt`
before standing the interface.
