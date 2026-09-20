# ADR-0029: v5 workspace schema and native migration

- **Status:** Accepted
- **Date:** 2026-08-28

## Context

The v5 engine changes the workspace contract: `state.md` carries a `schema`
cursor row declaring the workspace schema version, and the current contract is
schema 3. Workspaces written before v5 carry no row and resolve to schema 2,
the last pre-v5 contract.

The repository's preservation-first stance (ADR-0022, as applied by ADR-0025)
forbade an engine migrator: judgment work belongs to skills, and
`/rite-upgrade` routes repairs through phase owners. That stance served
read-compatible releases where no normalization was needed. A schema bump is
different: normalization — cursor encoding, missing-artifact reconciliation —
is deterministic, content-preserving work that needs per-feature locking,
atomic writes, and fail-closed refusal on ambiguity. Those primitives are
engine-owned.

## Decision

- The engine declares the current workspace schema (`state.SchemaVersion`), and
  each workspace declares its own through a `schema` row in its `state.md`
  cursor. An absent row means schema 2.
- `state resolve` and `state close` refuse workspaces whose declared schema
  differs from the engine's, naming the recovery path:
  `devrites-engine migrate <slug>` for pre-v5 workspaces, `upgrade devrites`
  for newer ones. Diagnostic checks (readiness, candidate, task-graph, observe)
  keep reading pre-v5 workspaces and reporting their established named gaps.
- `devrites-engine migrate` performs fail-closed normalization: cursor-form
  conversion through the existing dual-form primitives, missing required
  artifacts as empty stubs with no synthesized content, and byte-exact
  preservation of bound proof files. The v2→v3 normalization cannot invalidate
  a binding — it only creates missing files and rewrites the unbound ledger —
  and any future schema delta that would modify a bound file requires explicit
  per-finding confirmation before it applies.
- This supersedes the no-migrator invariant of ADR-0022 as applied by ADR-0025
  for deterministic schema normalization only. `/rite-upgrade` keeps
  preservation-first routing for judgment work and may hand mechanical
  normalization to the engine command.

## Alternatives considered

| Option | Why not |
|--------|---------|
| Keep the no-migrator stance | Every pre-v5 workspace would strand mid-feature or silently diverge from the declared contract; the manifest version alone cannot normalize. |
| Refuse all reads of pre-v5 workspaces | Diagnostics are the user's fastest evidence of what migration will change; refusing them removes actionable output without adding safety. |
| Migrate as a skill checklist | Cursor rewrites and artifact stubs need the feature lock, atomic writes, and deterministic refusal on ambiguity — engine-owned primitives. |
| New marker file per workspace | Duplicates the ledger; the cursor row travels inside the file every workspace already has. |

## Consequences

Pre-v5 workspaces remain readable; mutating or closing them requires
migration. Migration is deterministic and byte-preserving by default, so
recorded proof stays valid, and invalidation is never silent. The schema row
travels inside `state.md`, so no new file format exists. New workspaces created
by the v5 pack declare the row at creation.

Guard tests: `engine/internal/state/workspaceschema_test.go`,
`engine/tests/parity_resolve_test.go`, `engine/tests/parity_closeout_test.go`.
