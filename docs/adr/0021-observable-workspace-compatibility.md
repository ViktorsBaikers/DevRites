# ADR-0021: Observable and reversible workspace compatibility

- **Status:** Accepted
- **Date:** 2026-07-31

## Context

Legacy workspace layouts, filenames, cursors, phase aliases, and older schemas
must remain readable during the current support window. Silent compatibility
cannot prove when removal is safe, while an automatic in-place rewrite can
fabricate state, overwrite conflicting evidence, or leave a partial migration.

## Decision

- Runtime, validator, and migration share one active-workspace predicate:
  `state.md` or a workspace map in active `work/` or compatible `features/`.
  Archives, spec-only/empty directories, and operational remnants are ignored.
- Normal successful legacy reads append a bounded mode-`0600` JSONL record with
  only release, timestamp, category, slug, and canonical replacement. Pure
  validation and migration reads emit no telemetry.
- Compatibility is removed only by a later decision using records inside its
  declared release/time window; this change does not guess a removal date or
  build a future importer.
- Bare `migrate` and `migrate preview` are read-only. The deterministic manifest
  contains project-relative pre/post metadata and hashes, conflicts, ACTIVE
  selection, and byte/path counts, never artifact bodies or absolute paths.
- Apply requires an exact saved manifest, re-previews under one project lock,
  and verifies a private journal plus backup/post recovery material before the
  first target mutation. Rollback validates all material and touched path states
  before restoration. Both operations are idempotent and resumable; recovery
  material remains available.
- Preview, apply, and rollback each have a separate fresh approval boundary.
  Workflow mode, seal, AFK, or prior approval cannot imply mutation authority.

## Alternatives considered

| Option | Why not |
|---|---|
| Rewrite every legacy directory automatically | Cannot distinguish remnants or conflicts and creates unreviewed destructive behavior. |
| Keep compatibility forever without telemetry | Makes removal evidence impossible and retains permanent complexity. |
| Record full legacy content for analysis | Leaks project data and is unnecessary for usage counts. |
| Delete backups after success | Removes the verified recovery path before an operator accepts the result. |

## Consequences

Compatibility has a measurable cost and a deliberate later removal trigger.
Migration is slower than an unchecked rename because it hashes, journals, and
locks the whole transaction, but it preserves evidence and can recover from
every recorded interruption boundary without guessing.
