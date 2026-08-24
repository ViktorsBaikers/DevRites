# Slice-wright dispatch

`/rite-build` gives one bounded task to exact `devrites-slice-wright`.
Root owns scope, artifacts, inspection, and routing but never writes source/tests; only
the wright writes them and invokes no agent.

## Host gate

Follow the [source-writing boundary](../../devrites-lib/reference/standards/agents.md#source-writing-boundary):
Claude → exact wright `acceptEdits`; Codex → workspace root + `:workspace` wright +
read-only specialists. Never bypass the wright or recreate an engine bridge.

## Isolated writer-worktree pilot

Use native isolation only when the current host exposes an explicit named-writer
worktree plus result reconciliation; separate agent threads or inherited sandboxes do
not qualify. Codex CLI custom subagents therefore use `same-worktree` unless a future
supported interface passes this capability gate—never create a manual worktree from the
read-only root to imitate it. Also require `git rev-parse
--show-superproject-working-tree` to be empty; resolved git/common dirs; committed,
clean candidate/index; all inputs at `HEAD`; no other writer; and green cheap isolated
baseline proof. Otherwise use serial same-worktree dispatch—never stash, commit, or
discard user work to qualify. Ask host for an explicit current-`HEAD` base when
supported. Regardless of host defaults, the wright's first command must prove actual `git rev-parse HEAD` equals supplied
`worktree_base`. Mismatch returns a gap with no write before project reads or baseline proof.

The isolated wright returns one local unpushed `transfer_commit`, its `worktree_base`,
and exact files. Root proves descendant base, exact `git diff --name-only`
`` `<base>..<transfer>` ``, no `.devrites/**`/submodule/symlink/unrelated delta, unchanged
source base, and no user-work overwrite. Use only host-native explicit reconciliation;
never ad hoc copy, cherry-pick, or merge from read-only root. Compare transferred bytes,
run approved proof, record evidence, then let host remove worktree.

Conflict, extra/missing commit, moved base, or cleanup failure is `gap`/STOP:
preserve the worktree and commit. Without explicit reconciliation, use same-worktree serial.
Parallel writers only under `/rite-build --parallel N` with path-disjoint,
abort-batch, and control `parallel-lease.md` ([`parallel-batch.md`](parallel-batch.md)).
Same-worktree multi-writer / root-emulated worktrees stay forbidden.

## Prepare

1. Derive the smallest exact project-relative source/test path list; reject
   directories/globs, traversal, symlink escapes, duplicates, and `.devrites/**`.
   A target composed only of vetted executable workflow artifacts routes to the
   controlling root under
   [`workflow-artifacts.md`](../../devrites-lib/reference/standards/workflow-artifacts.md);
   its rejection here is not a blocker.
2. Put it in the task with goal, verbatim acceptance, exclusions, context,
   `test-plan.md` proof commands, and applicable standards. For each triggered
   topology/data/integration standard, include only the feature-specific owner/invariant,
   failure or partial-state case, recovery rule, and required proof from the vetted plan.
   Do not paste the whole standard or silently omit an applicable risk.
   A new path requires a new bounded contract; the wright cannot widen the task.
3. Record `git diff --name-only` before dispatch so unrelated work remains
   distinguishable. For an isolated pilot, also record committed base SHA and exact
   baseline status before asking the host for isolation.

## Run

Ask the host for the exact writer in fresh context and wait. Default: one writer.
Never two writers in one worktree, mixed modes, or generic substitutes. Opt-in
`/rite-build --parallel N` fans out only under [`parallel-batch.md`](parallel-batch.md).

## Inspect and prove

1. Compare the returned file list and `git diff --name-only` with task paths.
   Reject a result that omits any required key, including empty-or-filled
   bookkeeping arrays, or adds a path. Restore only through the same bounded
   wright; root never widens scope or edits source.
2. Inspect the test diff for deletion, skipping, focus markers, or loosened
   assertions. Dedicated test analysis treats weakening as Critical.
   Confirm a test for a data/integration/topology risk can actually exhibit that risk;
   mocks that erase it and one-root proof offered for another root are unproven.
3. Run only repository proof already approved by `test-plan.md`, then inspect
   `git diff --name-only` again in case a proof tool changed source.
4. Challenge decisions. Human choices return; native diagnosis routes failures.
   The root counts the wright plus recovery failures by causal fingerprint from
   the current context and recorded Dead ends/evidence. Three total failed
   attempts is the limit; persist each and never dispatch a fourth related try.
5. Reconcile returned reuse, conventions, principles, sources, assumptions,
   decisions, dead ends, follow-ups, gates, and touched files. Persist the
   relevant facts in canonical artifacts, not the wholesale report.
