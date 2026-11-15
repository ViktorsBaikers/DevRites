# The autocomplete loop: arm AFK, drive every phase

Autocomplete chains existing `/rite-*` workflows without routine-phase pauses. Acceptance-preserving Reslice authority is
`.dsh/skills/devrites-lib/reference/standards/acceptance-preserving-reslice.md`.

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

Read existing AFK first. Root serializes this short entry write with authority and
source-bookkeeping writes. On each explicit Autocomplete invocation, upsert only
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

Before each phase/fan-out/recovery/writer, check readiness/overlap and apply early
admission and the global pass below; local readiness is insufficient. Supply actual
per-invocation `DEVRITES_HOST_CAPACITY` under the
[all-phase host protocol](../../devrites-lib/reference/parallel-dispatch.md#host-reservations-and-subwaves);
count every retained/open/unknown actor, not only writers or preparation roles.
Explicit activation resets only activation-local counters, not durable recovery.
Pass resolved `--parallel N` to [Build](../../rite-build/reference/output.md#caller-owned-continuation).
Legacy or invented execution ceilings never stop this controller; human limits bind.
Resolve ranked picks via `devrites-engine state resolve`, not
a user handoff. Escalating, irreversible-risk, access, and unrecommended blocking
picks pause that source only. A Seal GO does not end the run.

## Early preparation admission

Follow the canonical [rolling preparation pool](../../devrites-lib/reference/parallel-workspaces.md#rolling-preparation-pool)
for Autocomplete + canonical authority + effective cap>1: bounded membership,
READY buffering, root-only creation from approved scope, durable parking/resume
and per-return refill. Ordinary Spec/Clarify and later native preparation share
controller-locked admission. Selected sources need no recursive successor start.
Standalone, cap 1 and legacy/headerless authority retain their prior boundary;
separate-session lookahead restrictions do not change.

Internal exclusive `context`/`dispatch open` flags:
- `--prepare-for <parent-source>`: permitted native non-Build preparation roles
  (only controlling-root context may be roleless), exact existing immediate
  successor/parent agreeing on canonical identity, binding, adjacency and chain.
  Retains exact parent identity in pool mode; outside it, exempts the child's
  successor gate. Neither form grants product-write authority.
- `--sequence-research`: only canonical `evidence-scout`, Spec/Clarify/Temper/Define/
  Plan/Vet, exact existing source and normal path/role/ledger constraints. Exempts
  authority/successor admission solely to investigate broken authority read-only:
  no acceptance, scope/approval/phase advance, successor creation, writers,
  readiness or preparation credit.

Context (`--skill rite-<phase>` too) checks before writes and reserves no capacity.
Existing admitted sources use context → packet → open → check-spawn → native
launch → `dispatch start` with actual handles, following complete host-reserved
subwaves. Openings alone prove no preparation. Missing pool targets instead use
the root [create action](../../devrites-lib/reference/parallel-workspaces.md#create-selected-identities),
then refresh frontier before context or dispatch; pending Spec is not Spec completion.

**Legacy/non-pool boundary:** derive preparatory-only B before any marked wave
from nonterminal parent A. Until ordinary unmarked B Temper/Define/Plan/Vet or
Build-wright admission (checking C first), A becoming terminal or ordered handoff
to B, C waits—even explicit C `next`/`check continuation`. No C creation/rerouting
before A's eligible phase. Completed B history retains this bound; Spec/Clarify/
research/read-only Build do not release it. Independent work stays visible; all
global-pass gates bind. This descendant wait does not apply to selected canonical
pool sources; actual contract dependencies still do.

Shared registry edits alone do not serialize disjoint modules. Isolate their
writer; refusal-only tests never replace positive requirements or defer their band.

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

At phase entry, before recovery/waits, execute the
[global pass](../../devrites-lib/reference/parallel-workspaces.md#continuous-global-frontier).
Observe → prioritize admitted writer picks → materialize selected missing identities
→ refresh → advance native preparation with remaining host slots. Consume each
return into its source and refill immediately; preserve native handles, return
cursors and protected proof windows. Run all admitted `frontier-concurrent:` routes
without finishing one source first. Resource deferral means queued, not started,
not a semantic hold; refresh after actor settlement. Creation, first actual writer
start, hold resolution, landing and artifact/capacity changes also trigger refresh.

Check [stop-conditions.md](stop-conditions.md) before advancing. Discard stale
pass evidence after source edits. Re-read the selected source before each phase per [phase target binding](../../devrites-lib/reference/phase-target-binding.md#phase-target-binding). On a cold resume or
new activation, run `devrites-engine check regression <slug>`: `BLOCKED` names a
durable fact lost since the last checkpoint — restore or route it through its
owning phase before continuing; `unproven` does not block. After each phase's
durable advance lands (green check loop, checkpoint commit), run
`devrites-engine check regression <slug> --update` to ratchet the progress
floor; `--update` never excuses an unreconciled regression.

## Sequence continuation

With a recorded successor, budget and no opt-out, Seal GO stays internal:
leave the owner sealed/unshipped with `/rite-ship`; resume the child's cursor/
controller or authorized native Spec. Use the
[shared grammar](../../devrites-lib/reference/workspace-artifact-schema.md#what-each-file-owns).
Charge each opening once. ACTIVE changes at ordered handoff, not early preparation,
creation or isolated independent Build. Canonical cap>1 preparation uses the
[rolling pool](../../devrites-lib/reference/parallel-workspaces.md#rolling-preparation-pool);
serial/legacy continuation pipelines only the immediate successor.

Before a new opening, require durable `state.md`
`sequence_workspaces_remaining > 0`; if absent, initialize the controller once
through its native workflow with `max_workspaces - 1` and `sequence_position: 1`.
Pool creation uses `state create-sequence <controller> <target> --brief <file>`:
the engine derives parent, position and budget; root must not supply counters or
decrement the parent. Resume matching identities byte-for-byte, even after native
authoring advances them. Only real brief plus pending Spec state is bootstrapped;
native Spec later owns normal artifacts and any approved release-role assignment.

Outside pool mode, `/rite-spec` seeds the child cursor under the
[continuation contract](../../rite-plan/reference/slicing.md#continuation-workspaces):
`sequence_parent` = current slug, `sequence_position` = parent position + 1,
`sequence_workspaces_remaining` = parent remaining − 1; an approved release
milestone gets `sequence_role: release`, requiring the union manifest. Serial/legacy
resume may re-seed missing fields from recorded brief parent/position and parent
counter without a second charge. No recorded parent/position means no membership.

Stop new sequence openings when: `continue_sequence: false`; no recorded next
entry; the `max_workspaces` cap is reached; or a genuine source human-owned, safety,
access or parked-recovery stop fires. These do not discard independent existing
work. `max_review_queue` never stops Autocomplete; agent-owned Seal `NO-GO` enters
recovery, not a handoff.
Intermediate sealed milestones continue without `--ship` and without Git mutation. On stop, report sequence position, sealed/unshipped workspaces, and
one release command (`/rite-autocomplete <release milestone> --ship`, or
`/rite-ship` for the current sealed workspace when the sequence is complete).

## Backtrack without handing off

Root owns earlier-phase recovery:

1. Save the originating phase/action unless a valid return cursor exists. On cold
   resume, reconcile durable fingerprints; restore `return_next_action` only from
   approved `test-plan.md`/evidence. Ambiguity routes to Vet, not execution.
2. Invoke Plan repair, affected Vet, remediation and proof inline. Nested `STOP`
   ends only that phase. Recovery Vet covers prior findings, changed paths/criteria
   and affected evidence, never unaffected axes.
3. Re-read state and `drift.md`/`evidence.md` after each return. Closed reproductions
   are progress; distinct Critical/Important invariants get separate budgets. Only
   unchanged fingerprints charge no-progress; Suggestion/Nit/FYI never prolong work.
4. Consume `repair-input:` and canonical blocked-input diagnostics through root
   [Plan preflight](../../rite-plan/reference/replan-and-repair.md#repair-after-spec-drift-guard),
   not another identical draft. Narrow no-findings closes only named findings, not
   the standing verdict. Record genuine holds as `frontier-blocked: <source> —
   <constraint; owner; re-evaluation trigger>` in source evidence; refresh. Continue
   independent work only within existing hold authority.
5. Spent consumptive authorization blocks re-execution, not offline diagnosis,
   repair or narrow Vet from retained evidence. After READY, request fresh authority.
6. Restore and consume the original cursor once prerequisites pass.

<!-- workflow-artifact-adapter: {"module":"devrites-lib/reference/standards/workflow-artifacts.md","entry":"loop tick sees Workflow Artifact trigger/state","action":"invoke classifier once under owner lock; no actor-history migration","return":"same loop cursor; no budget charge for verify/rerun"} -->
Ask only for human-owned decisions or mandatory safety/access actions. Resolve
blocking questions with `proposed`/Recommended options internally. Three unchanged
corrections spend that fingerprint strategy: escalate inline per the
[retry contract](../../devrites-lib/reference/standards/afk-hitl.md#retry-cap-no-progress-loops-and-self-resolve);
park after three escalations while independent work continues.

## Continuous caller obligation

Before reply, run `devrites-engine check continuation`; only `clear` permits reply.
Consume `invoke:`, `launch:`, `repair-input:` and `create: controller=X target=Y
parent=Z` internally. Creation requires an approved substantive brief,
`state create-sequence`, then refresh—not dispatch into a missing workspace.
Canonical writer picks precede preparation; legacy admission boundaries remain.
Local repair/picks/reconciliation/discharged deferrals keep their owning phase.
Route NEEDS_REPLAN, budgeted recovery and recommended answers to repair or
`state resolve`, subject to [resolver-failure stops](stop-conditions.md#other-stop-classes).
Cycles, context, compaction, duration and nested completion are not handoffs.
Persist until completion or a human/safety/access/parked-recovery stop. Report
the stop owner/action and resume condition—not the unchanged Autocomplete command.
