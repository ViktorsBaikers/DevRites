# The autocomplete loop: arm AFK, drive every phase

Autocomplete sequences existing `/rite-*` workflows without pausing between
routine phases. Acceptance-preserving Reslice authority is
`.agents/skills/devrites-lib/reference/standards/acceptance-preserving-reslice.md`.

<!-- BEGIN RESLICE ROUTE-TO-ACTION -->
- `FOLD` → keep Plan repair/affected Vet internal; no stop solely for topology/count.
- `GUARD_AND_REPAIR` → enter Spec Drift Guard/Clarify; pause only at an existing human-owned gate; resume Plan/Vet internally.
- `BLOCKED_INPUT` → no planning writes; stop internal branch; exact diagnostic; recover authority; reclassify.
<!-- END RESLICE ROUTE-TO-ACTION -->

## Arm AFK once

```yaml
allow_gates: [advisory, validating]
# max_slices / max_agents / max_minutes / max_review_queue omitted: unlimited
# continue_sequence: true          # opt-in: chain recorded continuations after Seal GO
# max_workspaces: 5                # sequence budget when chaining
# max_parallel: 10                 # Build batch cap; 1 = serial (default: eligible, cap 10)
# max_tokens: <N>
# max_cost_usd: <amount>
# notify: "<cmd>"
```

Read an existing sentinel first. Preserve it byte-for-byte when valid; stop if
malformed. Do not stop because it names `max_slices` / `max_agents` /
`max_minutes` / `max_review_queue` or only `[advisory]`. If absent, write it
once after clarity. Never write `max_slices` from `--max-slices`.
It is read-only: never rewrite it after Vet or reset it on resume. Leftover
`expires_at` is ignored and never rewritten.

### Derive the mutable post-vet budget

After Vet, seed `afk_slices_remaining` from the pending count so every pending
slice can run. Ignore `--max-slices`, sentinel `max_slices`, and leftover
remaining. AFK configuration itself stays unchanged.

### Admit each unattended cycle

Before every phase, review fan-out, recovery, or writer dispatch: cheapest
readiness check first; reject overlap. Do not stop on `max_agents`,
`max_minutes`, or `max_review_queue`. Count every leaf result. A new activation
gets fresh activation-local counters but retains durable slice/recovery state.

When driving `$rite-build`, ignore afk-discipline remaining-0, `--max-slices`,
and parallel AFK headroom. Close validating questions
(recommended pick) instead of queuing them.

## Phase arc

Workspace files carry state; chat does not. Read and execute each phase skill:

| Step | Phase | Completion / edge |
| --- | --- | --- |
| 1 | `$rite-spec` | investigate and write testable intent |
| 2 | `$rite-clarify` | topology-first scan; require `Decision coverage: CLEAR`, then arm AFK |
| 3 | `$rite-temper` | harden, reduce, or expand; irreversible risk still pauses |
| 4 | `$rite-define` | approved plan/tasks/traceability |
| 5 | `$rite-vet` | every plan; derive mutable budget after READY |
| 6 | `$rite-build` batch loop | largest eligible path-disjoint set (cap 10; one-slice round when <2 eligible); recompute after each round; charge once per green built slice |
| 7 | `$rite-prove` | all slices built; approved proof; recovery on red |
| 8 | `$rite-polish` | re-prove after code edits |
| 9 | `$rite-review` | in-scope correction then fresh proof |
| 10 | `$rite-seal` | GO/NO-GO; no Git |
| 11 | `$rite-ship` | only after GO; `--ship` never authorizes Git; stop at literal-GO/native approval |

Before advancing, check
[stop-conditions.md](stop-conditions.md). After source edits, discard stale pass
evidence. Re-read the active workspace before each phase.

## Sequence continuation

With `continue_sequence: true` and sequence budget remaining
([sentinel](../../devrites-lib/reference/standards/afk-hitl.md#sequence-continuation-continue_sequence-max_workspaces)),
step 10 does not end the run: leave the sealed workspace unshipped with its
`Next step: $rite-ship`, then invoke the next recorded continuation
(`$rite-spec <parent>-<n> "<objective>"` from the parent's `decisions.md` sequence)
and re-enter step 1. Spend one workspace per opened continuation.

Spend accounting is durable in `state.md` cursors, not chat. Before invoking the
continuation, ensure the current workspace's `sequence_workspaces_remaining` is
set — absent, write `max_workspaces - 1` and `sequence_position: 1` first — and
`> 0`. `$rite-spec` seeds the child's sequence cursor at creation per the
[continuation contract](../../rite-plan/reference/slicing.md#continuation-workspaces);
after the child's `state.md` exists, verify it — `sequence_parent` = the current
slug, `sequence_position` = parent position + 1,
`sequence_workspaces_remaining` = parent remaining − 1, and — when the recorded
entry is the sequence's release milestone — `sequence_role: release` (gates then
require the union manifest). On resume, a child whose
fields are missing is re-seeded from its `brief.md` parent/position and the
parent's counter (same values, never a second charge); a child with no recorded
parent/position is not a sequence member and the chain stops.

Stop instead when: no recorded next entry; the cap or review-queue bound is
reached; a human-owned, safety, access, or exhaustion condition fires; or Seal returns
`NO-GO`. On stop, report the sequence position, the sealed-but-unshipped workspaces, and
the single release command (`$rite-autocomplete <release milestone> --ship`, or
`$rite-ship` for the current sealed workspace when the sequence is complete).

## Backtrack without handing off

The active Autocomplete root remains caller whenever a later phase exposes an
agent-owned earlier-phase gap:

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
Ask only for a human-owned decision or mandatory safety/access action. Three
no-progress corrections of one exact fingerprint exhaust; preserve its
reproduction and dead ends without another Plan/Vet command.

## Continuous caller obligation

No user-facing reply is permitted while durable state contains agent-owned
`NEEDS_REPLAN`, an intermediate Plan/Vet action, or a distinct retained
Critical/Important fingerprint below its cap. Invoke the next internal repair
immediately. A narrow reviewer closing one finding and exposing another
Critical/Important invariant is progress, not exhaustion.

The number of completed repair/Vet cycles is not a stop condition. Context
pressure, compaction, session duration, and nested completion do not convert an
internal checkpoint into a handoff. Persist it and resume until the requested
rest point or a shared human/safety/access/exhaustion condition.
