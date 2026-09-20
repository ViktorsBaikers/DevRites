# ADR-0019: Native boundary with deterministic gates

- **Status:** Accepted
- **Date:** 2026-07-30

## Context

ADR-0018 replaced engine hook and reconcile enforcement with native host
permissions plus exact-path instructions. That simplification must not remove
independent quality gates, lifecycle rest points, or action-time approval. Its
statement that `test-integrity` was removed became obsolete when a
reconcile-free, committed-HEAD comparison was restored.

## Decision

- Native permissions retain the writer boundary: the root and reviewers do not
  edit source or tests; only `devrites-slice-wright` is writable, under an exact
  path contract checked in the returned diff.
- Deterministic gates remain independent of host hooks. Every successful
  lifecycle advance runs `readiness <slug>` immediately before completion;
  `/rite-seal` uses the final `seal <slug>` aggregate instead.
- `test-integrity` compares the working tree with committed `HEAD` without
  reconcile snapshots. The final seal retains acceptance, evidence freshness,
  test integrity, review integrity, doubt coverage, and opted-in mutation gates.
- Irreversible Git requires the exact commit/push/tag/PR plan and fresh literal
  user approval for one attempt. Seal GO, AFK, and autocomplete flags are not
  authority; changed or retried plans require fresh approval, and native host
  permission/sandbox prompts remain authoritative.

This supersedes only ADR-0018's statement that `test-integrity` is removed and
its rejection of a reconcile-free test-integrity gate. ADR-0018's native
permission, instruction-backed exact-path, hook-removal, and no-reconcile
decisions remain in force.

## Alternatives considered

| Option | Why not |
|---|---|
| Restore engine hooks or reconcile snapshots | Recreates host-policy duplication and snapshot machinery. |
| Rely on instructions alone for quality and completion | Removes deterministic evidence independent of model behavior. |
| Treat seal GO or `--ship` as Git authority | Conflates quality approval with fresh irreversible-action consent. |

## Consequences

The engine remains free of hook dispatch, writer-path classifiers, and
reconcile state. Native profiles and exact instructions own execution scope;
readiness, final seal, test integrity, and fresh action-time approval preserve
the unrelated safety and quality guarantees.
