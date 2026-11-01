# The autocomplete loop: arm AFK, drive every phase

Autocomplete chains existing `/rite-*` workflows without routine-phase pauses. Acceptance-preserving Reslice authority is
`.claude/skills/devrites-lib/reference/standards/acceptance-preserving-reslice.md`.

<!-- BEGIN RESLICE ROUTE-TO-ACTION -->
<!-- include:../../devrites-lib/reference/_shared/reslice-autocomplete.md -->
<!-- END RESLICE ROUTE-TO-ACTION -->

## Arm AFK once

```yaml
allow_gates: [advisory, validating]
# max_slices / max_agents / max_minutes / max_review_queue omitted: unlimited
# continue_sequence: false         # opt out; default chains after Seal GO
# max_workspaces: 5                # sequence budget when chaining
# max_parallel: 10                 # fresh-run default; preserve retained lease capacity on cold resume; explicit `--parallel N` rewrites only this field
# max_tokens: <N>
# max_cost_usd: <amount>
# notify: "<cmd>"
```

Read existing sentinel first. Preserve valid bytes, except
when this invocation contains `--parallel N`: then write or replace only
`max_parallel: N` and leave every other key unchanged. Stop if the sentinel is
malformed. Do not stop because it names `max_slices` / `max_agents` /
`max_minutes` / `max_review_queue` or only `[advisory]`. If absent, write it
once after clarity, and include `max_parallel: N` when that flag is present.
Without the flag, seed a newly created sentinel from retained global lease capacity
when present; otherwise use fresh-run default 10. Never replace a valid existing
sentinel's capacity from lease/default. Never write `max_slices` from `--max-slices`.
It is otherwise read-only: never rewrite it after Vet or reset it on resume. Leftover `expires_at` is ignored
and never rewritten. **Failing case:** leftover `max_parallel: 1` keeps the run
serial after `--parallel 5`.

### Derive the mutable post-vet budget

After Vet, seed `afk_slices_remaining` from the pending count so every pending
slice can run. Ignore `--max-slices`, sentinel `max_slices`, and leftover
remaining. AFK configuration itself stays unchanged.

### Admit each unattended cycle

Before each phase, review fan-out, recovery, or writer dispatch: cheapest
readiness check first; reject overlap. Do not stop on `max_agents`,
`max_minutes`, or `max_review_queue`. Count every leaf result. New activations reset activation-local counters, retaining durable slice/recovery state.

[Build return](../../rite-build/reference/output.md#caller-owned-continuation).
Pass `--parallel N`; ignore leftover cap/headroom, remaining-0,
and `--max-slices` stops. Agent-invented dispatch, wait, tool-call, or turn budgets are not stop conditions.
An explicit human request to pause or run only a bounded subset still wins.
Path collisions serialize work; they do not end the run.
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

At every phase entry, before cold/local recovery, on reviewer/planning returns and
before native waits, Autocomplete executes the bounded
[global frontier pass](../../devrites-lib/reference/parallel-workspaces.md#continuous-global-frontier).
Keep source-local return cursors and pending handles; apply its protected
validation window before accepting accounts or landing work. This scheduling
precedes owner Seal and does not change ordered handoff below.

Check [stop-conditions.md](stop-conditions.md) before advancing. Discard stale
pass evidence after source edits. Re-read the selected source before each phase per [phase target binding](../../devrites-lib/reference/parallel-workspaces.md#phase-target-binding). On a cold resume or
new activation, run `devrites-engine check regression <slug>`: `BLOCKED` names a
durable fact lost since the last checkpoint — restore or route it through its
owning phase before continuing; `unproven` does not block. After each phase's
durable advance lands (green check loop, checkpoint commit), run
`devrites-engine check regression <slug> --update` to ratchet the progress
floor; `--update` never excuses an unreconciled regression.

## Sequence continuation

Unless `continue_sequence: false`, while sequence budget remains
([sentinel](../../devrites-lib/reference/standards/afk-hitl.md#sequence-continuation-continue_sequence-max_workspaces)),
step 10 does not end the run: leave the sealed workspace unshipped with its
`Next step: /rite-ship`, then invoke the next recorded continuation
(`/rite-spec <parent>-<n> "<objective>"` from the parent's `decisions.md` sequence)
and re-enter step 1. Spend one workspace per opened continuation.

For an already prepared child, do not re-specify. Follow
[`parallel-workspaces.md`](../../devrites-lib/reference/parallel-workspaces.md)
while the owner is still running: honor claims, repair/re-Vet stale readiness,
and admit only explicitly independent/dependency-satisfied isolated work.
ACTIVE changes to the child only at normal ordered handoff after the owner Seal;
resume its durable cursor without a second controller or another opening charge.

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

Green slice/phase checkpoints consume their recorded `next_action` internally;
only the caller replies finally at completion or a genuine stop.
Before a reply, `devrites-engine check continuation` must print `continuation: clear`; `required` means invoke `invoke:` now.
No user-facing reply is permitted while durable state contains agent-owned
`NEEDS_REPLAN`, an intermediate Plan/Vet action, a distinct retained
Critical/Important fingerprint below its cap or with an escalation left, runnable
work independent of a parked fingerprint, or an open blocking question that
already names a ranked recommended option. Invoke the next internal repair or
`devrites-engine state resolve` immediately, unless that writer already hit the
[resolver-failure stop](stop-conditions.md#other-stop-classes). **Failing case:**
`state resolve` exits 1 on every call and the loop re-invokes it without end. A narrow reviewer closing one finding and exposing another
Critical/Important invariant is progress, not exhaustion.

The number of completed repair/Vet cycles, context pressure, compaction, session
duration, and nested completion never turn checkpoints into handoffs. Persist and
resume to the requested rest point or shared human/safety/access/parked-recovery stop.
