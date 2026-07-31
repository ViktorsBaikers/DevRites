# ADR-0018: Native sandbox and instruction writer boundary

- **Status:** Accepted
- **Date:** 2026-07-30

## Context

DevRites kept a Claude `wright-scope` hook, a Git-backed `reconcile` window,
and a snapshot-dependent `test-integrity` command to enforce one writer and an
exact per-slice path list. Native Claude and Codex profiles now enforce the
broader writer/read-only split directly. The remaining Go machinery duplicated
host policy, coupled builds to hook payloads and private Git snapshots, and
made ordinary corrections depend on an engine-owned source window.

Neither host provides a universal dynamic exact-file sandbox for a child. The
choice is therefore between retaining the second enforcement system or
accepting native sandboxing plus explicit task and diff-review instructions.

## Decision

DevRites accepts native host sandboxing plus instructions:

- the root never edits source or tests;
- only `devrites-slice-wright` is writable: Claude grants it `acceptEdits` and
  Codex grants it `:workspace`; the other 16 specialists remain read-only;
- the dispatch task states the smallest exact project-relative source/test
  paths, and the wright must not widen them;
- the root waits, compares the returned file list and `git diff --name-only`
  with the task contract, rejects any extra path, runs repository proof, reviews
  the test diff, and records exact touched files;
- test deletion, skipping, focusing, and weakened assertions remain prohibited
  behavior and are checked through normal tests, diff review, and dedicated
  test analysis;
- the engine no longer exposes `hook`, `reconcile`, or `test-integrity`;
- installation retains recognition and removal of legacy DevRites hooks so an
  upgrade cleans stale host configuration, but installs no active writer hook.

There is no compatibility command or replacement enforcement layer.

This ADR supersedes only the active engine-hook clause of ADR-0005, the
`wright-scope`/exact-allowlist/reconciliation clauses of ADR-0015, and the
exact-path reconciliation clauses of ADR-0017. Their other decisions and
historical context remain in force.

## Alternatives considered

| Option | Why not |
|---|---|
| Keep `wright-scope` and `reconcile` as defense in depth | They preserve a second host-policy system, private Git snapshot state, and hook-format coupling. |
| Rebuild test-integrity without reconciliation | It would recreate baseline machinery solely to inspect tests; normal proof, diff review, and the test analyst already own that behavior. |
| Let the root write small changes | It collapses the sole-writer role and makes the native permission split misleading. |
| Add another compatibility command | It retains the obsolete public surface without an operational owner. |

## Consequences

- Exact-path scope is honestly instruction-backed on both hosts, not hard
  technical enforcement.
- Native permissions still ensure only the wright is broadly writable and all
  other specialists remain read-only.
- Root diff inspection and test analysis are required acceptance steps rather
  than optional commentary.
- The engine becomes smaller and no longer owns source-window Git snapshots or
  hook payload parsing.
- `tests/phase-gate-routing-test.sh`,
  `tests/codex-agent-generation-test.sh`, `tests/hooks-parity-test.sh`, and
  focused installer tests guard the retained workflow and permission boundary.
