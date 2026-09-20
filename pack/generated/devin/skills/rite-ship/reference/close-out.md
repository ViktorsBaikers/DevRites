# Close-out: archive the workspace, free the active slot

Closing a feature means it stops being the *active* work, not that its record is
deleted. DevRites keeps the audit trail; it just moves out of the live path.

## What close-out does

1. **Mark done.** Set `state.md` → `Phase: done`, `Status: done`, and a `Next step`.
   When this workspace's `decisions.md` records an unfinished
   [continuation sequence](../../rite-plan/reference/slicing.md#continuation-workspaces),
   that step is the **exact** next command — `/rite-spec <parent>-<n> "<objective>"` —
   and the ship reply prints the same line. Otherwise it stays
   `/rite-spec <next feature>`. Nothing spawns the continuation automatically:
   the recorded command is the handoff, so the human starts each increment knowingly.
2. **Archive.** Run the deterministic script:
   ```bash
   devrites-engine state close <slug>
   ```
   It moves `.devrites/work/<slug>/` → `.devrites/archive/<slug>/` (every `.md`
   intact) and clears `.devrites/ACTIVE` **only if** ACTIVE still points at `<slug>`.
   It refuses to clobber an existing `.devrites/archive/<slug>/` (exit 5).
3. **Confirm the observed post-state.** The archive exists with its audit records
   intact and the live workspace is absent. If ACTIVE named this slug, verify it is
   empty; if it named another feature, verify that value remains unchanged. Only an
   empty cursor permits the claim "no active feature".

Any nonzero exit stops close-out. Archive collision (exit 5) never authorizes overwrite.
If clearing ACTIVE fails, the engine attempts to move the archive back; that rollback
can also fail. Inspect the actual live path, archive and cursor before reporting or
retrying. Record the command/error and observed locations; never claim restored state
from an attempted rollback or repeat a move against an unverified destination.

## Sequence predecessors

A deferred-ship sequence leaves earlier milestones sealed and unarchived in
`.devrites/work/`. The release ship's disclosed plan may close them
(`devrites-engine state close '<slug>'` per predecessor) after the release commit
lands; each close is safe because `ACTIVE` names the release workspace. Closing is
bookkeeping only: the release commit already carries their bytes, and their audit
records move intact.

## Why archive, not delete

- The `.md` files (`spec`, `decisions`, `assumptions`, `evidence`, `seal`, `ship`, …)
  are the project's record of *why* the feature is the way it is. A future
  `/rite-zoom-out` or incident review reads them.
- Deleting them would make the workflow's own "evidence over confidence" rule a lie.

## Re-opening an archived feature

Before restoring an archive, verify its identity, released cursor form and declared
workspace schema, that the live destination is absent, and that ACTIVE is empty or
already names this slug. A collision or another active feature stops restoration;
do not replace its workspace or cursor. Supported released cursor forms remain valid:
age or encoding alone never requires normalization (see
[workspace-artifact-schema.md](../../devrites-lib/reference/workspace-artifact-schema.md)).
An unsupported declared schema stops restoration: record the observed version,
installed engine's recovery diagnostic and pending normalization owner. Do not guess
a repair or claim `/rite-upgrade` migrates archives; archived/done work is its no-op.

After an authorized restoration, verify the actual paths and cursor and preserve the
audit records. Reconcile the resumed scope through `/rite-converge`; changes return to
the applicable earlier lifecycle stage. Historical proof/GO remains historical: a new
candidate requires fresh Prove → Polish → Review → Seal bindings before Ship. Restoring files
does not reactivate an old GO.

Cross-feature review is user-invoked through `/rite-learn`; close-out does not
mine, score, nudge, or write a retrospective ledger.
