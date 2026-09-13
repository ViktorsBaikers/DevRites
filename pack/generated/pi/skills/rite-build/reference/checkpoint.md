# Checkpoint: proven slices land as local WIP commits

A slice's source never stays uncommitted after its gates are green: at RECORD
the orchestrator commits the proven slice as a local `WIP`, so the working
tree stays clean and the work survives a crash or compaction.

## The checkpoint commit: orchestrator, at RECORD, after gates are green
A checkpoint records a **proven** slice only after its gates are green. Stage the exact
slice-owned candidate paths from the manifest, passing each literal path separately.
Never stage `touched-files.md`, a directory, glob, unrelated path, or user change.
Verify the staged set, then commit locally with a `[devrites-context]` body:

```bash
git commit -m "WIP(<slug>): <slice>" -m "$(cat <<'BODY'
[devrites-context]
decisions: <one-line delta this slice added to decisions.md>
remaining: <pending slices — count or names>
dead-ends: <approaches ruled out this slice, if any>
BODY
)"
```

When host reconciliation already landed the isolated transfer commit, nothing
is staged and the step no-ops. **Local-only:** never push a checkpoint or let
scratch work trigger CI.

## Restore
After a crash, a fresh session may read the last `WIP(<slug>)` body as crash context.
Authoritative state remains in `.devrites/`: validate the body against that workspace;
never reconstruct or advance state from the commit body. Reload per
[`context-hygiene.md`](../../devrites-lib/reference/standards/context-hygiene.md)
(`.devrites/ACTIVE`, `state.md`, `questions.md`, `decisions.md`, `test-plan.md`/`evidence.md`).

## Collapse at ship
WIP commits are scratch and never reach shared history. `/rite-ship` folds them into the
one atomic feature commit before the Conventional-Commit ladder: see the collapse step in
[git-ship.md](../../rite-ship/reference/git-ship.md). Result: one clean commit, bisect
stays green.

## Autocomplete
Checkpoints are unconditional; `/rite-autocomplete` needs no sentinel arming.
Each wright returns after one slice; HITL stops, while explicit `.devrites/AFK`
lets the controlling Build root chain another green slice only under its cap
and pause rules.
