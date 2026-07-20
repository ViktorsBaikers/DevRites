# ADR-0004: Phase-relative section completeness

- **Status:** Accepted
- **Date:** 2026-07-08 (backfilled)

## Context

A feature's working state must be legible to both humans and the engine, and
"is it complete?" must be answerable deterministically at each step of the
lifecycle. A single long document makes completeness opaque and burns context;
a rigid "all fields required always" model blocks early phases on artifacts
that don't exist yet (there's no proof during framing).

## Decision

Model feature state as **six single-concern section files** — `spec`, `plan`,
`decisions`, `tasks`, `proof`, `status` (with transitional aliases
`evidence→proof`, `state→status` that `migrate` normalizes). Drive the lifecycle
through the ordered rite-\* arc:
`frame → spec → temper → define → plan → vet → build → converge → prove → polish → review → seal → ship → done`.
Completeness is **phase-relative and additive**: the typed phase registry maps
each phase to the sections that must have real content to leave it; a section
not yet required never blocks. `SchemaVersion = 1`; a feature may declare its
own and the engine refuses anything newer than it understands. Files evolve
additively.

## Alternatives considered

| Option | Why not |
|--------|---------|
| One `feature.md` mega-document | Completeness isn't self-evident; every read pays for the whole file. |
| All sections required in every phase | Blocks framing/spec phases on proof/tasks that legitimately don't exist yet. |
| Status inferred from git / external tracker | Couples state to a tool outside the workspace; breaks the git-diffable, self-contained record. |

## Consequences

- Completeness is a table lookup, not a judgment call — cheap and reproducible.
- Small files stay context-cheap and make "what's missing" self-evident.
- `SchemaVersion = 1` is young; migration is single-version. Hardening the
  writer into one pure transition function is a recorded follow-up
  (`docs/research/gsd-core-adoption.md` §3.2).
- The typed registry in `engine/internal/state/schema.go` is the lifecycle
  authority. Its invariant tests lock order, aliases, commands, requirements,
  and cross-format manifest freshness.
