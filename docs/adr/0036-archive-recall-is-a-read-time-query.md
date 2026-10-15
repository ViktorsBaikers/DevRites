# ADR-0036: Archive recall is a read-time query

- **Status:** Accepted
- **Date:** 2026-10-15

## Context

Shipped features are archived with their decisions, review and seal findings,
drift, and candidate manifest. Only `/rite-spec` consulted the archive, by
keyword. Define, Review, and debugging never saw what an earlier feature decided
or found about the same files, so a constraint learned once had to be learned
again. ADR-0014 and ADR-0022 removed engine index workers, ingestion, and
telemetry, and `/rite-learn` forbids a learning ledger or index.

## Decision

- `devrites-engine recall --paths <paths>` answers one question on read: which
  archived features touched these paths, and which of their decisions,
  carry-forward constraints, and review/seal/drift lines name them. The links
  are derived from `touched-files.md` and the `Files:` fields already in the
  archive. Nothing is written, cached, or indexed.
- Define folds relevant pointers into `Prior-slice learnings`, Review hands
  them to lane reviewers as context (never as findings), and debug recovery
  reads prior dead ends before ranking hypotheses.
- Build records confirmed causes as `decisions.md` carry-forward constraints,
  which later slices quote and recall surfaces to later features.
  `/rite-learn` stays the only route from a recurring lesson to project guidance.

## Consequences

- Recall cost grows with the archive and is paid only when called.
- A pointer is prior context: live source and the current spec still win.
