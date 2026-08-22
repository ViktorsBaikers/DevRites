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
allow_gates: [advisory]
max_slices: 10
max_agents: 32
max_minutes: 120
max_review_queue: 8
expires_at: "<now + 4 hours>"
# max_tokens: <N>
# max_cost_usd: <amount>
# notify: "<cmd>"
```

Read an existing sentinel first. Preserve it byte-for-byte when valid; stop if
its gate ceiling exceeds advisory, required envelope is malformed/expired, or
`max_slices` conflicts with the invocation. If absent, write it once after
clarity, replacing the safe slice default only for explicit `--max-slices N`
and resolving one absolute UTC expiry. It is read-only: never rewrite it after
Vet or reset it on resume.

### Derive the mutable post-vet budget

After Vet, count pending slices and take the minimum of that count, explicit
flag, and sentinel cap. Before first Build, seed absent state-owned
`afk_slices_remaining`; retain the lower valid existing value and never increase
or reinitialize it. AFK configuration itself stays unchanged.

### Admit each unattended cycle

Before every phase, review fan-out, recovery, or writer dispatch, apply
`afk-hitl.md#unattended-resource-envelope`: cheapest current-state/readiness
check first; reject overlap; count unresolved review; check expiry and native
agent/token/cost/time headroom. Count every leaf result. At the review cap, only
reconciliation that reduces the queue may run. Any reached or unobservable
bound stops before more work and persists the winning reason. A new activation
gets fresh activation-local counters but retains durable slice/recovery state,
expiry, and recomputed queue.

`allow_gates: [advisory]` means validating gates pause now rather than becoming
Seal NO-GO. Only explicit human configuration outside Autocomplete may widen it.

## Phase arc

Workspace files carry state; chat does not. Read and execute each phase skill:

| Step | Phase | Completion / edge |
| --- | --- | --- |
| 1 | `$rite-spec` | investigate and write testable intent |
| 2 | `$rite-clarify` | topology-first scan; require `Decision coverage: CLEAR`, then arm AFK |
| 3 | `$rite-temper` | harden or reduce; expansion and irreversible risk pause |
| 4 | `$rite-define` | approved plan/tasks/traceability |
| 5 | `$rite-vet` | every plan; derive mutable budget after READY |
| 6 | `$rite-build` ×N | one pending slice per wright; charge once only after green built state |
| 7 | `$rite-prove` | all slices built; approved proof; recovery on red |
| 8 | `$rite-polish` | re-prove after code edits |
| 9 | `$rite-review` | in-scope correction then fresh proof |
| 10 | `$rite-seal` | GO/NO-GO; no Git |
| 11 | `$rite-ship` | only after GO; `--ship` never authorizes Git; stop at literal-GO/native approval |

Before advancing, check resource admission and
[stop-conditions.md](stop-conditions.md). After source edits, discard stale pass
evidence. Re-read the active workspace before each phase.

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
