# ADR-0037: Autocomplete lands local checkpoints and self-resolves own scratch

- **Status:** Accepted
- **Date:** 2026-10-17
- **Supersedes:** the ADR-0022 clause that put local checkpoint commits under fresh approval

## Context

ADR-0022 kept the rule that a seal GO, AFK mode, or autocomplete flag never
authorizes an irreversible action, and listed `commit` with push, tag, and PR. The
generated `AGENTS.md` block repeated it. The checkpoint protocol and
`afk-hitl.md` already treated a local, unpushed `WIP(<slug>):` checkpoint as
landing automatically and folded at `/rite-ship`. Hosts read the generated block as
a host ask-first rule, so `/rite-autocomplete` stopped for a human at every slice
checkpoint. On Boxset, 3 of 5 human stops in one day were local checkpoint approvals.
The other 2 were safety gates for fixed `/tmp` paths that the writer had itself
created, which filesystem birth times proved afterwards.

## Decision

- A local, unpushed `WIP(<slug>):` checkpoint commit made under the checkpoint
  protocol (exact manifest staging, scope checks, no amend, reset, rebase, push, or
  force) is not irreversible. It lands without a prompt in AFK and HITL runs.
- Push, tag, PR, the ship commit, and any history rewrite remain irreversible and
  keep fresh, explicit, per-attempt user approval.
- A writer's overwrite or removal of a path outside the workspace that it created
  itself in the same wave is not filesystem destruction. When birth times and
  ownership prove it, the root records the evidence and continues. An unprovable
  path still pauses.
- A host repository's own ask-first commit rule still wins.

## Consequences

- `/rite-autocomplete` runs from Frame to a sealed, locally checkpointed feature
  without per-slice human stops; `/rite-ship` remains the single human Git gate.
- Each landed checkpoint stays reversible by the owner (`git reset`) until ship.
