# ADR-0007: Canonical live workspace filenames

- **Status:** Accepted
- **Date:** 2026-07-20

## Context

ADR-0004 established phase-relative `proof` and `status` sections while the
live workspace later standardized its human-facing artifacts on `README.md`,
`state.md`, and `evidence.md`. Shipped skills, validators, the typed workflow
registry, and user documentation already write those names, but migration and
some engine comments still normalized in the opposite direction.

## Decision

Use `README.md`, `state.md`, and `evidence.md` as the canonical live workspace
map, cursor, and proof log. Continue reading `feature.md`/`index.md`, `status.md`,
and `proof.md` as compatibility aliases. `devrites-engine migrate` adds missing
canonical files from aliases without deleting or overwriting either form.
Internal `proof` and `status` section identifiers remain conceptual completeness
names, not canonical filename declarations.

## Alternatives considered

| Option | Why not |
|--------|---------|
| Restore `feature.md`, `status.md`, and `proof.md` as canonical | Would reverse the shipped workspace contract and require coordinated changes across skills, validators, docs, fixtures, and generated hosts. |
| Treat both directions as equally canonical | Makes migration non-idempotent and leaves writers without one preferred target. |
| Delete aliases after migration | Risks data loss and breaks existing workspaces and older installed packs. |

## Consequences

- Migration direction matches the files produced and required by the live pack.
- Existing aliases remain losslessly readable and are preserved on disk.
- Schema version remains `1`; this is an additive filename normalization, not a
  content-schema break.
