# One controller, multiple workspace sources

> Root-only: not inlined in --role bundles. Every phase uses target binding below;
> Autocomplete owns the global workflow. Standalone Build stays source-local.

## Phase target binding

For every lifecycle phase and cold handoff, resolve the existing explicit target
or caller-supplied SOURCE slug first; use `.devrites/ACTIVE` only when neither is
supplied. All workspace context, cursor, questions, artifact reads/writes, approval,
candidate bindings and proof belong to that source. A missing source artifact
routes its prerequisite; it never falls back to another workspace's artifacts.
An explicit valid workspace does not require ACTIVE to name it or to exist.

Carry that source through Spec → Clarify → Temper → Define/Plan → Vet and later
Build/Converge/Prove/Polish/Review/Seal/Ship entrypoints. Use existing slug arguments
where supported, including `context <source>` and `handoff <source> --resume`.
For qid-only `state resolve` (including `--drop`/`--batch`), keep its existing argv
and bind the call with `DEVRITES_ROOT=<absolute-control>/.devrites` and
`DEVRITES_WORKSPACE=<absolute-control>/.devrites/work/<source>`. The override is a
workspace path, not a slug. Apply this environment only to that invocation;
neither a fabricated resolver slug argument nor process-global environment
mutation is routing. Use the same source binding for nested recommended answers.

Preparing or running an admitted future source preserves the ordered owner's
ACTIVE and retained writer/controller. Ordinary standalone Spec still sets ACTIVE;
explicit `--lookahead` still stops at planning/Vet and never becomes a Build root.
Target binding changes neither phase approval nor Ship authority.

## Authority and admission

Resolve one controlling root and ordered lifecycle owner. `.devrites/ACTIVE` stays
on that owner until normal sequence handoff; scheduling a future slice does not
advance ACTIVE or create a second Build root. One repository-global lease owns
control-tip mutations. Resume its controller, source identities and retained
handles before admitting more work; status queried via ACTIVE/source reports the
actual retained controller even after ordered handoff. Another retained
controller is not bypassable; no extra cursor pointer is needed.

The existing `--parallel N` is one global active-writer cap across all sources:
explicit invocation value, else AFK `max_parallel`, else retained lease capacity
on resume, else 10. Explicit 1 means one writer, not one writer per workspace.
Review queues obey host capacity and existing reviewer caps separately; they never
erase required specialist accounts.

Discover only recorded forward sequence continuations through `sequence_parent`
and the parent's recorded objectives. Honor `continue_sequence: false`, opening
budget and human bounded requests. Charge each opening once; reuse prepared
children and their cursor rather than re-specifying or charging them again.
An unrelated directory or user-mentioned slug is not automatic enrollment.
Respect workspace claims: a live lookahead owner retains its artifacts until
release/terminal reconciliation. Root alone writes workspace artifacts; leaves
never write `.devrites/**`.

For each future slice, native Plan/Vet must have CURRENT approved readiness,
a complete bound contract and explicit `Workspace dependencies` under
[canonical slice grammar](workspace-artifact-schema.md#canonical-slice-grammar).
Plan/Vet checks assumptions, consumed interfaces, shared runtime resources and
independent slice proof. `none` is a native declaration, never an inference from
path disjointness, local `Dependencies: none`, planning digests or a green branch.
Only durably BUILT/integrated producer checkpoints satisfy qualified dependencies;
validate each producer reference and graph, including safely archived predecessors.
Cycles, unknown/malformed references and shared unisolated runtime withhold unsafe
candidates. A waiting producer withholds only its consumers, not unrelated work.

Missing declarations in legacy future plans need narrow Plan repair + affected
Vet before early admission; existing local/standalone plans remain valid. Absent,
incomplete, stale or human-blocked future work is a planning/repair/blocked
opportunity, not a fatal pool failure. Retain every readiness reason: fixing stale
bindings does not waive a coexisting human/access/safety gate. Re-Vet affected
predecessor assumptions, then record current bindings; never auto-refresh a digest.

## Continuous global frontier

At entry, each landing, planning/review return and before any wait or final reply:

1. Re-read controller lease, per-source state/questions/readiness and authorized
   sequence. Scan all stages, not only the owner's next local slice. While writers
   run, root advances authorized Spec → Clarify → Temper → Define/Plan → Vet,
   including narrow repair/re-Vet of prepared targets, under unchanged approval
   and safety rules. Missing human authority blocks that target, not safe siblings.
   If an explicit/AFK capacity change differs from retained capacity, reconcile it
   before selection using `parallel add --root <repo> --slug <controller> --cap N --json -`
   with stdin `[]`. This changes capacity only, not workers or a complete lease's
   status. A decrease below active count refuses: pause admissions and drain to
   the requested count, then retry; never cancel workers to meet a lower cap.
2. Query the mixed pool in stable sequence order, controller first (the ordered
   ACTIVE owner may now differ):
   `devrites-engine parallel select --root <repo> --slug <controller> --from-plan --workspaces <controller,authorized-source,...> --cap N`.
   `--workspaces` is an internal data-source parameter, not a user feature flag.
   Read all notices and route repair/waits to their source. Existing running slots
   and paths seed selection globally. Zero new picks with live writers means keep
   planning/reviewing or wait, not Prove for those unfinished sources.
3. Preflight selected slices against their SOURCE workspace with Build's normal
   slice/readiness/precondition, exact-path and runtime-isolation gates. Early
   foreign Build always uses isolated workers. For cap>1 on a capable host,
   isolate even the first lone local writer, so later sources can refill. Persist
   `--cap N` on create/add. With a retained lease, add even one pick; a complete
   retained lease can reopen only at unchanged control tip. After cleanup,
   create a new batch from current control. Never start another controller.
   Unsupported isolation uses truthful serial owner dispatch and ordinary ordered
   handoff, never early same-tree foreign Build or concurrent same-worktree writers.
4. Launch every selected native writer before waiting on any; keep scheduling
   planning and independent review while they run. Land each individually green
   sibling through Build review/proof and `parallel integrate --slice <key>`;
   refill immediately from the whole authorized pool. A red sibling repairs in
   place and holds its slot/paths; it does not delay independent green landings.
   Never satisfy a producer edge on writer return or green-but-unlanded work.

## Identity, dispatch and proof

Mixed JSON uses `{"slices":[{"id":"SLICE-001","paths":["src/child.go"],"workspace":"child"}]}`.
Omitted workspace means controller; explicit controller normalizes to that lane.
Lease targeting uses bare `SLICE-001` only for controller, `child/SLICE-001` for
foreign sources. Duplicate local IDs across sources remain distinct typed keys.
Use the lease's worktree/branch paths, not hand-built namespaces.

Controller slug owns `parallel create/add/status/record-green/integrate/cleanup`.
SOURCE slug owns `context`, `packet`, `dispatch open/start/return`, slice checks,
candidate freeze, proof, repair, evidence and bookkeeping. Open source-local waves
with local roles `slice-wright.<SLICE-ID>` and local `--slice <SLICE-ID>`; no slash
in a dispatch role. Start every source wave before first wait. Root explicitly
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
Before any non-lease product correction, shared durable Polish write or control
checkpoint, freeze new writer admissions, drain/reconcile retained writers, land
safe green work and cleanup the lease. Preserve blocked work; never force-clean
it merely to advance a phase. Perform phase-owned correction/proof/checkpoint,
then resume the pool from new HEAD. Never accept unrelated commits as expected
ControlTip or invent phase slices. Read-only frozen proof/review and planning-only
`.devrites` repair may overlap writers; none may mutate the leased product/control tip.
Normal ordered handoff reuses the same controller and prepared child cursor.
Never archive controller/source while retained lease owns it; clean up owned
workers first, preserving salvage and unknown outcomes.

Early child WIPs can interleave with parent WIPs. Parent-only Ship must not absorb
those commits as if parent alone were sealed. Deferred release waits for every
participating sequence member's final proof/Review/Seal, uses engine-generated
release union manifest and re-proves affected cross-workspace acceptance. Disclose
all participating checkpoints in the exact Git plan, verify unchanged candidate
and staged bytes, then require fresh literal `GO` plus native approval. Parallelism
grants no push/tag/deploy authority and never broadens an earlier Ship approval.
