# ADR-0011: Separate Define authoring from the Plan checkpoint

- **Status:** Accepted
- **Date:** 2026-07-23

## Context

The typed lifecycle gave both `define` and `plan` the same requirements and
`/rite-define` resume command. Current skills already use `/rite-define` for
first-pass plan authoring and `/rite-plan` only to repair or reslice an existing
plan, so the duplicate state behavior made resume routing ambiguous.

## Decision

- `define` is in-progress plan authoring and resumes `/rite-define`.
- `plan` is the approved or repaired plan checkpoint and resumes `/rite-vet`.
- `/rite-plan` remains an explicit repair/reslice operation; it is not the
  checkpoint's normal resume command.
- Keep the `plan` phase ID and all phase aliases. Existing schema-v2 and legacy
  workspaces therefore remain readable without migration.
- The typed phase registry owns a unique transition-right sentence for every
  state; the generated manifest and current lifecycle docs derive from it.

## Consequences

An interrupted approved plan continues into engineering review instead of
re-entering plan authoring. Repair still uses `/rite-plan`, and no compatible
workspace is rewritten merely because resume routing became unambiguous.
