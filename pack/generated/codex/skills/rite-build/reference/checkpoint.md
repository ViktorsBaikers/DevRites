# Checkpoint: crash-survivable WIP commits (opt-in)

`.devrites/` markdown survives compaction, but a slice's *source* stays uncommitted until
`$rite-ship`. A crash or killed session mid-build: the AFK / `$rite-autocomplete` case:
loses that source **and** the reasoning behind it. Checkpoint mode commits each proven
slice as a local `WIP`, so the work and its context outlive the session.

## When it's on
Opt-in via the `.devrites/CHECKPOINT` sentinel: the mirror of `.devrites/AFK`. Absent →
no checkpoints; `$rite-build` behaves exactly as before. A long unattended run
(`$rite-autocomplete`) is the case it earns its keep, so that path sets the sentinel for
the run. **Local-only by rule:** a checkpoint is never pushed: scratch work must not
trigger CI.

## The checkpoint commit: orchestrator, at RECORD, after gates are green
A checkpoint records a **proven** slice; it fires only after PROVE is green, never on a red
gate. Stage only the slice's `touched-files.md`, then commit locally with a
`[devrites-context]` body a cold reader can restore from:

```bash
[ -f .devrites/CHECKPOINT ] || exit 0   # sentinel absent → no-op, silent
git commit -m "WIP(<slug>): <slice>" -m "$(cat <<'BODY'
[devrites-context]
decisions: <one-line delta this slice added to decisions.md>
remaining: <pending slices — count or names>
dead-ends: <approaches ruled out this slice, if any>
BODY
)"
```

## Restore
The commit body is a durable record, not just a marker. After a crash, a fresh session:
or `$rite-status`: reads the last `WIP(<slug>)` body to reconstruct the decisions,
remaining slices, and dead-ends that in-session `.devrites/` state would have held. No
special tooling: it's a git log entry any agent can read.

## Collapse at ship
WIP commits are scratch and never reach shared history. `$rite-ship` folds them into the
one atomic feature commit before the Conventional-Commit ladder: see the collapse step in
[git-ship.md](../../rite-ship/reference/git-ship.md). Result: one clean commit, bisect
stays green.

## Autocomplete clean-baseline use
`$rite-autocomplete` may arm checkpoint mode after it verifies a clean or explicitly accepted baseline. Checkpoints are local-only crash recovery, not authorization to continue across red gates or to ship. `$rite-build` remains one-slice-at-a-time; autocomplete is the only opt-in loop that may invoke multiple builds.
