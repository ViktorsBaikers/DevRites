# The autocomplete loop: arm AFK, drive every phase

Autocomplete chains existing `/rite-*` workflows without routine-phase pauses. Acceptance-preserving Reslice authority is
`.pi/skills/devrites-lib/reference/standards/acceptance-preserving-reslice.md`.

<!-- BEGIN RESLICE ROUTE-TO-ACTION -->
- `FOLD` → keep Plan repair/affected Vet internal; no stop solely for topology/count.
- `GUARD_AND_REPAIR` → enter Spec Drift Guard/Clarify; pause only at an existing human-owned gate; resume Plan/Vet internally.
- `BLOCKED_INPUT` → no planning writes; stop internal branch; exact diagnostic; recover authority; reclassify.
<!-- END RESLICE ROUTE-TO-ACTION -->

## Arm AFK once

```yaml
mode: autocomplete
allow_gates: [advisory, validating]
# max_slices / max_agents / max_minutes / max_review_queue omitted: unlimited
# continue_sequence: false         # opt out; default chains after Seal GO
# max_workspaces: 5                # sequence budget when chaining
# max_parallel: 10                 # fresh-run default; preserve retained lease capacity on cold resume; explicit `--parallel N` rewrites only this field
# max_tokens: <N>
# max_cost_usd: <amount>
# notify: "<cmd>"
```

Read existing AFK first. On each explicit Autocomplete invocation, upsert only
`mode: autocomplete` and, if supplied, `max_parallel: N`; preserve other fields.
Missing mode retains standalone AFK behavior until that explicit entry. Invalid
or duplicate mode fails closed; mode grants neither phase nor action authority.
If absent, create AFK after clarity with the mode above; capacity is explicit N,
else retained global lease capacity, else 10. Preserve existing capacity without
the flag. Nested phases/wakes leave AFK unchanged: no `max_slices` flag seeding,
expiry rewrite or reset. Legacy slice/agent/time/review limits do not stop this
controller. **Failing case:** `--parallel 5` retains `max_parallel: 1`.

### Derive the mutable post-vet budget

After Vet, seed `afk_slices_remaining` from the pending count so every pending
slice can run. Ignore `--max-slices`, sentinel `max_slices`, and leftover
remaining. AFK configuration itself stays unchanged.

### Admit each unattended cycle

Before each phase, fan-out, recovery or writer: run cheap readiness and reject
overlap. Before Build admission, execute the shared immediate-successor
preparation/deferral and global cohort rules; local readiness alone is insufficient.
Count every leaf result. Explicit activations reset activation-local counters,
not durable slice/recovery state. Pass resolved `--parallel N` to
[Build](../../rite-build/reference/output.md#caller-owned-continuation).
Ignored legacy budgets and local serial constraints are not whole-run stops.
Invented ceilings on dispatches, waits, tool calls or turns cannot stop execution;
human instructions to pause or limit scope still bind.
Resolve ranked validating/blocking picks via `devrites-engine state resolve`;
never hand `/rite-resolve` to the user. Escalating, irreversible-risk, access, blocking
without recommendations, and Seal judgment (`slice: seal`) still pause.

## Phase arc

Workspace files, not chat, carry state. Read/execute each phase skill:

| Step | Phase | Completion / edge |
| --- | --- | --- |
| 1 | `/rite-spec` | investigate and write testable intent |
| 2 | `/rite-clarify` | topology-first scan; require `Decision coverage: CLEAR`, then arm AFK |
| 3 | `/rite-temper` | harden or reduce; `expand` and extra acceptance stop for a human; irreversible risk still pauses |
| 4 | `/rite-define` | approved plan/tasks/traceability |
| 5 | `/rite-vet` | every plan; derive mutable budget after READY |
| 6 | `/rite-build` | execute shared controller selection/admission; charge each source once per landed slice |
| 7 | `/rite-prove` | all slices built except deferred (`Satisfies: none`); approved proof; recovery on red |
| 8 | `/rite-polish` | re-prove after code edits |
| 9 | `/rite-review` | in-scope correction then fresh proof |
| 10 | `/rite-seal` | GO/NO-GO; no Git |
| 11 | `/rite-ship` | only after GO; `--ship` never authorizes Git; stop at literal-GO/native approval |

At phase entry, before local recovery or waits, execute the
[global pass](../../devrites-lib/reference/parallel-workspaces.md#continuous-global-frontier).
Consume the fresh frontier emitted after a recorded `dispatch return`; after
landing, planning/artifact changes or capacity updates, rerun frontier. Execute
authorized preparation/selection, preserving pending handles, local cursors and
protected validation windows. Reporting or creating a placeholder is not progress.

Check [stop-conditions.md](stop-conditions.md) before advancing. Discard stale
pass evidence after source edits. Re-read the selected source before each phase per [phase target binding](../../devrites-lib/reference/parallel-workspaces.md#phase-target-binding). On a cold resume or
new activation, run `devrites-engine check regression <slug>`: `BLOCKED` names a
durable fact lost since the last checkpoint — restore or route it through its
owning phase before continuing; `unproven` does not block. After each phase's
durable advance lands (green check loop, checkpoint commit), run
`devrites-engine check regression <slug> --update` to ratchet the progress
floor; `--update` never excuses an unreconciled regression.

## Sequence continuation

With a recorded successor, opening budget and no opt-out, Seal GO stays internal:
leave the owner sealed/unshipped with `/rite-ship`, then resume the prepared child
or invoke its authorized native Spec. Sequence authority follows the
[shared grammar](../../devrites-lib/reference/workspace-artifact-schema.md#what-each-file-owns),
not invented slugs. Charge an opening once; preparation may pipeline before Seal,
one immediate successor per admitted source, not the whole roadmap at once.
Reuse the child's durable cursor and controller. ACTIVE changes only at normal
ordered handoff; early planning or isolated independent Build does not advance it.

Before invoking a continuation, require durable `state.md`
`sequence_workspaces_remaining > 0`; if absent, first write
`max_workspaces - 1` and `sequence_position: 1`. `/rite-spec` seeds the child's sequence cursor at creation per the
[continuation contract](../../rite-plan/reference/slicing.md#continuation-workspaces);
verify the created child's `state.md`: `sequence_parent` = current slug,
`sequence_position` = parent position + 1,
`sequence_workspaces_remaining` = parent remaining − 1, and — when the recorded
entry is the sequence's release milestone — `sequence_role: release` (gates then
require the union manifest). On resume, re-seed missing child fields from its `brief.md` parent/position and
parent counter (same values, no second charge). No recorded parent/position
means no sequence membership: stop the chain.

Stop sequence continuation when: `continue_sequence: false`; no recorded next
entry (the final milestone); the `max_workspaces` cap is reached; or a genuine
human-owned, safety, access, or parked-recovery stop fires. `max_review_queue`
never stops Autocomplete; agent-owned Seal `NO-GO` enters recovery, not a handoff.
Intermediate sealed milestones continue without `--ship` and without Git mutation. On stop, report sequence position, sealed/unshipped workspaces, and
one release command (`/rite-autocomplete <release milestone> --ship`, or
`/rite-ship` for the current sealed workspace when the sequence is complete).

## Backtrack without handing off

Autocomplete remains caller for agent-owned earlier-phase gaps:

1. Save the originating phase/action unless a valid native return cursor exists.
2. On cold resume, reconcile the terminal cursor against durable fingerprint
   accounting. Restore `return_next_action` only from the approved
   `test-plan.md`/evidence action; ambiguity returns to Vet and never licenses
   execution.
3. Invoke required Plan repair, affected Vet, remediation, and proof inline.
   Recovery Vet is a narrow Vet recheck of prior findings, changed paths/criteria,
   and affected evidence; it never restarts unaffected axes. A nested `STOP`
   ends that phase only; do not hand the intermediate command to the user.
4. Re-read state after each nested phase. Reconcile exact causal fingerprints
   from `drift.md`/`evidence.md`. A closed reproduction is progress. A different
   Critical/Important invariant gets a separate budget; Suggestion/Nit/FYI does
   not prolong the chain. Only a no-progress result charges the same fingerprint.
5. For failed consumptive action, spent authorization blocks only another real
   execution. Use retained evidence for offline diagnosis, repair, and narrow
   Vet; after READY, pause for fresh authorization.
6. Restore and consume the original cursor when prerequisites are green.

<!-- workflow-artifact-adapter: {"module":"devrites-lib/reference/standards/workflow-artifacts.md","entry":"loop tick sees Workflow Artifact trigger/state","action":"invoke classifier once under owner lock; no actor-history migration","return":"same loop cursor; no budget charge for verify/rerun"} -->
Ask only for a human-owned decision or mandatory safety/access action. An open
blocking question with `proposed` or option 1 labelled Recommended is not that
decision: resolve it and continue. Three no-progress corrections of one exact
fingerprint spend only its strategy: escalate inline per the
[retry contract](../../devrites-lib/reference/standards/afk-hitl.md#retry-cap-no-progress-loops-and-self-resolve),
and park it after three escalations while independent work continues.

## Continuous caller obligation

After a green slice or phase, execute its recorded `next_action` internally;
reply only at completion or a genuine stop.

Before replying, run `devrites-engine check continuation`: `required` means
consume `invoke:` now; only `clear` permits a final reply. Under controller mode,
actionable normalization, successor preparation or foreign work precedes local
Build. Local-only reconciliation, repair, picks or discharged scheduling deferrals
retain the owning phase edge, not recursive `/rite-autocomplete`.
Agent-owned NEEDS_REPLAN, budgeted Critical/Important recovery, independent work
or a recommended blocking answer stays internal. Invoke repair or `state resolve`
unless the [resolver-failure stop](stop-conditions.md#other-stop-classes) applies.
A distinct invariant is progress, not exhaustion; repeated failed resolver calls
without correction are not progress.

The number of completed repair/Vet cycles, context pressure, compaction, session
duration, and nested completion never turn checkpoints into handoffs. Persist and
resume to the requested rest point or shared human/safety/access/parked-recovery stop.
