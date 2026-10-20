# ADR-0038: The engine flags unattributed root edits

- **Status:** Accepted
- **Date:** 2026-10-20
- **Narrows:** the ADR-0018 clause "There is no compatibility command or replacement enforcement layer", for root-edit attribution only

## Context

ADR-0018 made the root's read-only rule and the wright's exact-path scope
instruction-backed and removed the engine's Git-backed reconcile window. ADR-0035
then took the Claude root out of plan mode, so no host now has a technical guard on
root edits. `check diff-scope` runs only when a wright returns and checks only that
wright's allowlist: it blames the wright for a root edit, and it never sees an edit
made after a slice's last wright.

In four `/rite-autocomplete` sessions on a downstream project, the root made 34
successful product edits outside `.devrites/`, and 33 of them had no open claim.
One deleted file was first taken for owner work. Fixing that cost an owner
decision, a restore wright, and a dropped workaround.

## Decision

- The engine keeps `<feature>/tree-baseline.json`: dirty product paths (git
  status, outside `.devrites/` and `.scratch/`; gitignored files never appear)
  mapped to the `check diff-scope` fingerprint. The first `dispatch open` writes
  it, and that dirt counts as accepted user work. The return that completes a
  wright wave, in any phase, takes in the wave's slice write sets and claim paths. Paths
  that are no longer dirty keep their recorded fingerprint, so collapsing a
  checkpoint commit back into the work tree flags nothing.
- `dispatch open`, `next` and `check candidate` exit `3` when a dirty product
  path has a fingerprint that differs from the baseline and no live claim or
  in-flight wright wave covers it. The message names the paths and both
  remedies. Each finding writes an `unattributed-edit` row to `metrics.jsonl`.
- Remedies need no human stop. The root routes the path through a bounded
  wright: a wright wave (Build or Polish) whose `--claim` or slice write set
  names the path may open. When the evidence shows owner work, gate output, a
  returned wright outside a dispatch wave, or a protocol-sanctioned root write
  (Polish ADR promotion, `DESIGN.md`), the root records
  it with `dispatch <slug> attribute --reason <evidence> --path <path>`, which
  logs the reason and takes the path into the baseline.
- The engine never reverts, restores, or deletes a file.
- A workspace without `tree-baseline.json`, a root that is not a project's
  `.devrites/`, or a project outside git is never blocked.
- No hook and no private Git ref; the tripwire takes no Git snapshot (the
  ref-free tree object that `check windows --since-dispatch` records per build
  wright wave is a separate read aid). Exact-path wright scope stays
  instruction-backed plus `check diff-scope`, as ADR-0018 decided.

## Alternatives considered

| Option | Why not |
|---|---|
| PreToolUse hook on root edits | The measured failures ran on a host the hook would not cover. Hooks bring back the payload coupling that ADR-0015 and ADR-0018 removed, and ADR-0016 found child hooks bypassed. |
| Git-backed snapshot refs | ADR-0018 removed private Git snapshot state. Workspace JSON plus `git status` reads are enough. |
| Engine restores flagged paths | It cannot tell owner work from root edits, so a restore could destroy owner work. |
| Keep instructions only | The downstream evidence shows the instruction alone does not hold on an autocomplete root. |

## Consequences

- A root edit is flagged at the root's next `dispatch open`, `next` or `check
  candidate`, not first at seal or by a later session.
- Known limit: while a wright runs, the engine cannot tell a root edit from a
  wright edit on a path that wright owns. It also cannot tell owner edits from
  root edits. Hence the attribution remedy, never a revert.
- A root commit that bypasses the checkpoint protocol hides an edit from the
  tripwire. Review and seal still read the full diff.
- `engine/internal/lib/treebaseline_test.go` guards the decision.
