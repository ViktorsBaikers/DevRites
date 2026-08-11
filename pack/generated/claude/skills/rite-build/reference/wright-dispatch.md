# Slice-wright dispatch

`/rite-build` gives one bounded task to exact `devrites-slice-wright`.
Root owns scope, artifacts, inspection, and routing but never writes source/tests; only
the wright writes them and invokes no agent.

## Host gate

Follow the canonical [source-writing boundary](../../devrites-lib/reference/standards/agents.md#source-writing-boundary):
Claude grants only the exact wright `acceptEdits`; Codex uses its workspace root,
the exact `:workspace` wright, and read-only specialists. Never bypass/substitute
the wright or recreate an engine bridge.

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
   distinguishable.

## Run

Ask the host for the exact writer in fresh context and wait. Never run two
writers in one worktree or substitute a generic agent.

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
