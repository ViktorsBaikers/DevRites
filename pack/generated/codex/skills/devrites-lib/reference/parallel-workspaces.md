# One controller, multiple workspace sources

> Root-only: not inlined in --role bundles. Every phase uses
> [phase target binding](phase-target-binding.md#phase-target-binding);
> Autocomplete owns the global workflow. Standalone Build stays source-local.

## Authority and admission

Keep ACTIVE on the ordered owner until normal handoff. One repository-global
lease owns control-tip mutations; recover its controller, source identities and
handles before more admission, including after handoff. No second Build root.
`--parallel N` caps active writers globally: explicit N, else AFK `max_parallel`,
else retained capacity on resume, else 10. One means one writer total. The canonical
preparation pool uses the same effective N, not additional writer authority.
All phases share the actual [host actor budget](parallel-dispatch.md#host-reservations-and-subwaves);
capacity or Reslice never waives required reviews or proof.

Only checked source-related sequence membership qualifies; user-mentioned or
unrelated directories are not enrollment. Current canonical `roadmap.md` follows
[the grammar](workspace-artifact-schema.md#what-each-file-owns): one related live/
archived owner, actual chain root, same-owner unique decision and contiguous
position table. It supersedes historical tables for scheduling without rewriting
them. Invalid canonical authority fails closed; absence retains conservative
legacy resolution. Honor opt-out, opening budgets and human bounds; charge each
opening once. Lookahead claims retain artifacts through terminal reconciliation.
Root alone writes workspace artifacts; leaves never write `.devrites/**`. Serialize
short AFK, roadmap/authority, brief and source-bookkeeping writes at the root. Use
controller-locked engine mutations for creation and dispatch admission/parking;
never run simultaneous direct edits to shared authority or workflow state. Native
drafting/review can overlap; root consumes each result into its own source.

Native Plan/Vet must bind CURRENT readiness, contract, producer assumptions and
explicit future `Workspace dependencies` per [slice grammar](workspace-artifact-schema.md#canonical-slice-grammar).
Canonical pool plans also declare `Consumed planning inputs: none` or a unique
comma-separated producer-slug list. Vet records `Predecessor readiness: none` or
one `Predecessor readiness: <slug> <digest>` for each declared producer, including
nonadjacent producers. Review actual design use, not just the declaration. Exact
set/digest matching is required: missing, duplicate, extra or drifted bindings
withhold readiness. Adjacency and roadmap scope ownership alone are not consumed
planning inputs; unrelated adjacent drift does not invalidate explicit `none`.
`none` is a native declaration, not an inference from paths, local dependencies,
digests or green branches. Only durable BUILT/integrated checkpoints satisfy
validated producer references/graphs, including safely archived predecessors.
Unknown references, cycles and shared unisolated runtime withhold unsafe slices;
waiting producers withhold consumers, not independent work. Legacy future plans
need narrow Plan repair/affected Vet; standalone/local plans retain their rules.
Per-slice withheld diagnostics do not block other returned eligible IDs;
workspace-wide readiness failures still withhold the source.
Preserve coexisting stale-readiness and human/access/safety reasons. Re-Vet changed
assumptions; never auto-refresh digests or copy archives into live `work/`.

## Rolling preparation pool

This is the canonical owner of rolling preparation policy. It activates only for
Autocomplete with valid canonical roadmap authority and effective capacity N>1.
Standalone Build, explicit cap 1 and legacy/headerless authority keep their existing
serial or immediate-successor boundary. Two-column canonical roadmaps remain valid;
sequence order identifies immediate parents, not semantic prerequisites or approval.
Do not add a preparation dependency graph or enroll unrelated existing directories.

Reconstruct membership on every resume from canonical identities, current source
state and retained dispatch accounts; do not maintain another queue or pool counter.
Count workspaces, not roles: retained preparation, unheld pre-Build work and READY
work awaiting its first writer all occupy slots. The pre-Build controller counts
too. An open preparation wave reserves membership but is not started-work credit;
context reads reserve nothing. A real pending Spec cursor is queued work, not
completed Spec. Only actual native handles establish overlap.

The first actual Build writer start permanently removes that source from future
preparation buffering; a read-only Build actor does not. Native repair of a source
with a prior writer start stays outside this pool. Terminal settlement or durable
parking also frees a slot. READY alone does not, so ready buffering remains bounded.
Refill in roadmap order after creation, each actor return/settlement, first writer
start, hold resolution and landing; never wait for an entire preparation wave.

### Create selected identities

Consume `frontier-create: controller=X target=Y parent=Z` (continuation emits
`create: controller=X target=Y parent=Z`) at the root, after eligible writer picks.
Read recorded approved scope and its provenance; author a substantive real brief
for Y. An identity table is not scope approval. Missing or contradictory scope
routes native Spec/Plan clarification, not invented acceptance. Initialize the
controller's existing sequence cursor once through its native workflow, then run:

```text
devrites-engine state create-sequence <controller> <target> --brief <file>
```

Materialize selected missing identities in roadmap order, refreshing frontier after
each result. The command derives immediate parent, position and remaining budget
under controller/target locks. It atomically creates only real `brief.md` and
schema-4 `state.md`, pending `$rite-spec <target>`; native Spec owns normal artifacts
afterward. It sets no release role, moves no ACTIVE and decrements no parent budget.
Matching live identities reuse evolved bytes before new-opening policy, budget or
capacity checks: do not recreate, overwrite or charge again on resume. Refusals
retain their exact recovery route. Never request target context or dispatch against
a missing directory; refresh after creation, then invoke its admitted native phase.

### Park and resume

A real human/dependency hold parks only its source. Retain the concrete native hold
or `frontier-blocked: <target> — <constraint>` evidence with its re-evaluation
trigger; `blocked`/`awaiting_human` alone may still have runnable recovery cursors.
After every source actor settles, run `devrites-engine dispatch <source> park`
**before** clearing that hold or evidence row. The durable marker survives hold
removal and restart; unresolved or unknown handles still consume capacity. Once
the hold resolves, run `devrites-engine dispatch <source> resume`; controller-locked
admission requires unchanged canonical identity and free preparation capacity. If
full, leave it parked and refresh after a slot frees. Resume never silently erases
another hold. Independent sources continue while opening budgets remain available.

## Recover avoidable serialization

Apply the complete [serialization audit](serialization-audit.md#recover-avoidable-serialization)
at native Plan/Vet and when a full global scan leaves writer capacity unused.

## Continuous global frontier

Run this pass at phase entry, before local recovery/waits/reply, and after creation,
actor settlement, first writer start, hold resolution, landing or planning changes.
A persisted `dispatch return` emits fresh scheduling after locks release, even when
later claim release fails: retain the nonzero failure and claim exclusion. Exact
successful claim reconciliation also refreshes; rejected/unpersisted returns do not.
Ordinary status stays read-only. Keep local return cursors and native handles: one
controller, not another phase or queue. Supply actual per-invocation host capacity
through the [dispatch protocol](parallel-dispatch.md#host-reservations-and-subwaves).

1. Read `devrites-engine frontier <source>`, then its selected `frontier-authority`
   paths before preparation. Follow the actual owner/decision, including archives;
   local phase context is not a reason to create a second owner. Normalize unresolved
   legacy authority through its owning decision/phase. Positive budget, ambiguous
   prose or path disjointness grants nothing; invalid canonical authority fails closed.
   Under the [rolling pool](#rolling-preparation-pool), prioritize admitted writer
   picks, execute selected creation actions, refresh, then advance admitted native
   Spec → Clarify → Temper → Define/Plan → affected Vet with remaining host slots.
   Launch each admitted `frontier-concurrent:` command on its source without waiting
   for another source to finish. Selected sources do not recursively prepare their
   own successor before working, nor wait for numeric predecessors to finish. Native
   phases must resolve actual consumed contracts or park only the affected source.
   Queued preparation is never started-work credit and never a prerequisite that
   deadlocks otherwise eligible writers. `frontier-resource-deferred:` is honest
   host backpressure, not a semantic hold, READY verdict or preparation evidence;
   refresh it after writer launch or actor settlement.
   Outside canonical pool mode, preserve the existing immediate-successor obligation
   and preparatory-only descendant boundary in
   [early preparation](../../rite-autocomplete/reference/loop.md#early-preparation-admission).
   Actual native preparation, completed live/archived successors, a valid finite
   tail or a concrete disclosed deferral discharges that legacy obligation;
   placeholders, reports and unopened waves do not.
   Opt-out, zero opening budget and source human/question holds prevent new openings,
   not independent existing work. In Autocomplete, `blocked`/`awaiting_human` with
   no question cursor or open blocking/escalating question may still run its Spec,
   Clarify, Temper, Define, Plan or Vet recovery cursor.
   Keep concrete `frontier-blocked: <target> — <constraint>` records visible;
   `frontier-blocked: sequence — <constraint>` can defer unresolved historical
   normalization for valid independent local work. Native-reviewed deferrals waive
   only that scheduling obligation, never readiness, malformed canonical/AFK data,
   paths, claims, isolation, proof, human gates or already-ready foreign picks.
   Use durable park/resume above before removing a pool source's hold.
2. Reconcile explicit/AFK capacity via `parallel add --root <repo> --slug
   <controller> --cap N --json -`, stdin `[]`. Below occupied count, freeze every
   new launch, even reserved waves; reconcile/drain and retry without cancelling.
   Recover recorded landing intent; unrelated HEAD/ref movement
   stays blocked. Rerun frontier after changes and execute its exact authorized
   `frontier-select-command`. Local select notices do not broaden picks. Existing
   global slots and paths seed selection; reserve the mixed cohort in create/add
   before separate source-local wave opens. Reservations assemble width, never
   prove actor checkout/start. Ready foreign picks cannot disappear behind local
   `serial:` records. A foreign deferral is exactly
   `serial: child/SLICE-001 — <reason>` in source evidence; bare `SLICE-001` is local.
   Substantive-reason rules apply; native workflow judges the constraint.
   Re-evaluate owning serial rows when constraints change; engine checks exact
   scope, not the truth of a host/resource claim.
   Discharged deferrals are not repeatedly actionable. Canonical authority and
   writer gates precede admission; queued pool preparation does not delay eligible
   writers. Legacy successor admission retains its boundary. Local-only
   reconciliation, repair and picks keep their owning phase, not recursive Autocomplete.
   Zero picks with live writers means planning/wait, not premature Prove.
3. Apply SOURCE Build readiness/preconditions, exact paths and runtime isolation.
   Early foreign Build always uses isolated workers. On a capable host at cap>1,
   isolate even the first local writer for later refill; persist `--cap N`.
   Add one pick to a retained lease; reopen a complete lease only at unchanged
   control tip, or create from current control after cleanup. Unsupported
   isolation uses qualified serial deferral, truthful local-owner dispatch and
   ordinary ordered handoff, never early same-tree foreign/concurrent writers.
   New dispatch accounts retain opaque `controller_batch` and immutable per-slice
   normalized `admitted_paths`. Retain only each pending writer's lane; returned/
   integrated siblings may release slots for rolling refill. Missing snapshots or
   changed lease paths require actor reconciliation, never backfill from new tasks.
   Exact claim identity/session still matters; release/expiry does not stop an actor.
   Same-session claims allocate space, not extra write authority. None of these
   records proves actual cwd/profile: verify through the native host.
4. Start every admitted native writer before waiting, using complete host-reserved
   subwaves; continue independent native preparation/review with remaining host
   slots. Refresh after each recorded first writer start to refill preparation.
   Protected validation defers unsafe landings, not independent preparation.
   Outside that window, land individually green
   siblings through Build review/proof and `parallel integrate --slice <key>`;
   refill globally. Red/integrate-failed siblings keep slots/paths for repair.
   Durable integration alone satisfies producer edges. After worker-bound
   accounts and source bookkeeping settle, `parallel prune --root <repo> --slug
   <controller>` removes only verified clean integrated checkouts, retaining
   transfer refs/lease identities. Dirty/ignored data keeps its recovery path.

## Identity, dispatch and proof

Mixed JSON uses `{"slices":[{"id":"SLICE-001","paths":["src/child.go"],"workspace":"child"}]}`.
Omitted workspace means controller; explicit controller normalizes to that lane.
Lease targeting uses bare `SLICE-001` only for controller, `child/SLICE-001` for
foreign sources. Duplicate local IDs across sources remain distinct typed keys.
Use the lease's worktree/branch paths, not hand-built namespaces.

Controller slug owns `parallel create/add/status/record-green/integrate/prune/cleanup`.
SOURCE slug owns `context`, `packet`, `dispatch open/start/return`, slice checks,
candidate freeze, proof, repair, evidence and bookkeeping. Open source-local waves
with local roles `slice-wright.<SLICE-ID>` and local `--slice <SLICE-ID>`; no slash
in a dispatch role. Start every role in each host-reserved subwave before its first wait. Root explicitly
supplies source slug, absolute worktree cwd, base and exact native profile to every
writer/reviewer. Writer profile is `devrites-slice-wright`, never a suffixed or
generic profile; reviewers use exact workflow-named `devrites-*` profiles in fresh
read-only context. Labels/handles are not proof of native profile selection.
Linked worktrees use `DEVRITES_ROOT=<control>/.devrites` for metadata, not source cwd.

Run controller/global commands in a per-invocation environment without inherited
worker `DEVRITES_WORKSPACE`; source calls use explicit source slug and control
`DEVRITES_ROOT` in a source-bound environment, never another source's override.
Never mutate process-global environment or ACTIVE to route a call.

Freeze/review/prove each source slice in its own worktree/candidate window under
Build's full required inventory, affected rechecks and repair rules. After landing,
write `WIP(<source>):` with human summary, union only that source's paths into its
manifest, persist checked BUILT/evidence and charge once there. Run source-local
regression checks on resume and ratchet only after durable green advance. Neither
controller bookkeeping nor sibling proof closes a child's acceptance.

## Lifecycle closeout and Ship

Prove/Polish/Review/Seal remain per workspace with fresh candidate bindings. Shared
durable Polish outputs serialize; source changes invalidate affected evidence.
From candidate freeze/validator dispatch through account admission, preserve the
exact candidate, control tip and all consumed metadata/validator inputs. Independent
authorized planning and isolated writers may proceed only without changing those
inputs or shared runtime resources. Defer integration, checkpoints, cross-source
corrections and consumed-artifact edits until the window closes; retained green
workers keep their identities/slots. Recheck candidate, control-tip identity and
consumed inputs before accepting each account; drift invalidates affected accounts
and requires native fresh validation, never acceptance against a moving tip.
After any protected validation window closes, before a non-lease product correction,
shared durable Polish write or control checkpoint, freeze new writer admissions,
drain/reconcile retained writers, land
safe green work and cleanup the lease. Preserve blocked work; never force-clean
it merely to advance a phase. Perform phase-owned correction/proof/checkpoint,
then resume the pool from new HEAD. Never accept unrelated commits as expected
ControlTip or invent phase slices. Read-only frozen proof/review and planning-only
`.devrites` repair may overlap writers; none may mutate the leased product/control tip.
Normal ordered handoff reuses the same controller and prepared child cursor.
Never archive controller/source while retained lease owns it; clean up owned
workers first, preserving salvage and unknown outcomes.
Global scheduling is not global gating or ordered handoff. Unmet or abandoned
owner gates remain non-green: an owner request to defer obligations routes Spec
Drift Guard → Plan → affected Vet under existing authority, or retains the blocked
source. It never becomes Prove green or Seal GO merely to reach another source.

Early child WIPs can interleave with parent WIPs. Parent-only Ship must not absorb
those commits as if parent alone were sealed. Deferred release waits for every
participating sequence member's final proof/Review/Seal, uses engine-generated
release union manifest and re-proves affected cross-workspace acceptance. Disclose
all participating checkpoints in the exact Git plan, verify unchanged candidate
and staged bytes, then require fresh literal `GO` plus native approval. Parallelism
grants no push/tag/deploy authority and never broadens an earlier Ship approval.
