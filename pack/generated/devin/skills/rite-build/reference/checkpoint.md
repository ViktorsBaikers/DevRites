# Checkpoint: land local WIP only after the check loop is green

The wright does not commit onto the control/primary branch. Independent review
and fail-on-red proof run first; a red or gap finding dispatches another wright.
Only when that loop is green does the orchestrator land the local unpushed
`WIP(<slug>):` commit. Same rule for `/rite-build`, `/rite-prove`, `/rite-polish`,
`/rite-review`, and `/rite-autocomplete`. Never push. `.devrites/CHECKPOINT` is
not a gate.

## When (and when not)

- **Do:** after Independent Build review (or the phase's independent validators)
  and fail-on-red proof are green, with no open Critical/Important and no repair
  wright still in flight. Serial: git recipe below (skip empty index) or isolated
  FF of a matching `transfer_commit`. Parallel: `devrites-engine parallel
  integrate --apply-to-control` (no second control `git commit`).
- **Do not:** when a wright returns; before review/proof; with red/gap findings;
  fast-forward an isolated transfer onto control before those checks.

A native-worktree `transfer_commit` is transport on the worker branch only.

## The commit
Stage the exact candidate paths from the manifest, each as its own argv.
Never stage `touched-files.md`, a directory, glob, unrelated path, or user change.
Verify the staged set, then commit locally.

Subject: `WIP(<slug>):` plus an imperative summary of the change (from the
goal). Never a slice id, `SLICE-###`, a slice number, or "this slice"
(`WIP(admin-report): add CSV export for usage rows`, not `SLICE-003` or
`complete slice 3`). Product source, tests, comments, and filenames omit the
same marks; they stay in `.devrites/` only.

```bash
git commit -m "WIP(<slug>): <imperative summary>" -m "$(cat <<'BODY'
[devrites-context]
decisions: <one-line delta this work added to decisions.md>
remaining: <pending count>
dead-ends: <approaches ruled out, if any>
BODY
)"
```

**Local-only:** never push or let scratch work trigger CI.

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
