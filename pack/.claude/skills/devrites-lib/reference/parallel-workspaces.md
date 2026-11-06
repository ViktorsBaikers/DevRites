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

Carry SOURCE through every phase, nested resolution and cold handoff. Use existing
slug arguments (`context <source>`, `handoff <source> --resume`). Qid-only
`state resolve` retains its argv: bind only that invocation's `DEVRITES_ROOT` and
`DEVRITES_WORKSPACE` as above, never a fabricated resolver slug or process-global
change. Early future work preserves ACTIVE and the retained controller.
Standalone Spec still sets ACTIVE; `--lookahead` remains planning/Vet-only.
Target binding changes neither phase approval nor Ship authority.

## Authority and admission

Keep ACTIVE on the ordered owner until normal handoff. One repository-global
lease owns control-tip mutations; recover its controller, source identities and
handles before more admission, including after handoff. No second Build root.
`--parallel N` caps active writers globally: explicit N, else AFK `max_parallel`,
else retained capacity on resume, else 10. One means one writer total. Review
queues retain host/reviewer caps; capacity or Reslice never waives fanout or proof.

Only checked source-related sequence membership qualifies; user-mentioned or
unrelated directories are not enrollment. Current canonical `roadmap.md` follows
[the grammar](workspace-artifact-schema.md#what-each-file-owns): one related live/
archived owner, actual chain root, same-owner unique decision and contiguous
position table. It supersedes historical tables for scheduling without rewriting
them. Invalid canonical authority fails closed; absence retains conservative
legacy resolution. Honor opt-out, opening budgets and human bounds; charge each
opening once. Lookahead claims retain artifacts through terminal reconciliation.
Root alone writes workspace artifacts; leaves never write `.devrites/**`.

Native Plan/Vet must bind CURRENT readiness, contract, producer assumptions and
explicit future `Workspace dependencies` per [slice grammar](workspace-artifact-schema.md#canonical-slice-grammar).
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

## Recover avoidable serialization

At native Plan/Vet, and when a full global scan leaves writer capacity unused,
audit the remaining frontier before treating serialization as irreducible:

1. Name what each edge consumes: product/interface, integration evidence, shared
   writable path, runtime resource or explicit authority. Sequence/release order
   alone is not a product dependency; current recorded prohibitions still bind.
2. Recut unstarted work only where an accepted vertical capability is functional,
   independently provable and rollbackable against an already real contract.
   Keep producer edges on actual consumers and dependent integration/proof.
   Consolidate hub edits in their real producer; worktrees alone do not isolate
   shared files, databases, ports, credentials or measurement windows.
3. Root routes a concrete opportunity through native Plan and the canonical
   [Reslice classifier](standards/acceptance-preserving-reslice.md), preserving
   every acceptance meaning, prohibition, semantic link and proof obligation.
   Carry the blocked edge/resource and proposed independent cut into the drafter
   packet. Invalidate readiness, obtain affected Vet, then reselect; a proposed
   split or green branch is not admission authority.
4. Freeze admissions to affected sources during repair. Preserve built and leased
   slice identities/contracts; drain/reconcile affected workers before publishing
   a repair that changes their inputs. Unrelated source planning/writers continue.
5. Preserve human/safety/access/resource gates and explicit planning-only bounds.
   Classify any authority change through its owning decision; never replace an
   explicit prohibition with inferred permission or create placeholder slices.

Completion: either admit a currently approved independent cut, or retain
`Serial rationale: <consumed contract/resource/authority; next unblock event>`
in affected `tasks.md` slice blocks only when normal Plan write authority permits; otherwise
report the existing blocker without planning writes. Revisit after relevant
inputs change, not merely because capacity remains unused. Target width never
justifies extra architecture, acceptance loss, premature proof or repeatedly
reslicing unchanged work.

## Continuous global frontier

Run this pass at phase entry, before local recovery/waits/reply, and after landing
or planning changes. A persisted `dispatch return` emits fresh scheduling after
locks release, even when later claim release fails: retain the nonzero failure
and claim exclusion. Exact successful claim reconciliation also refreshes;
rejected/unpersisted returns do not. Ordinary status stays read-only. Keep local
return cursors and native handles: one controller, not another phase or queue.

1. Read `devrites-engine frontier <source>`, then its selected `frontier-authority`
   paths before preparation. Follow the actual owner/decision, including archives;
   local phase context is not a reason to create a second owner. Execute native actions.
   Normalize unresolved legacy sequence authority through its owning decision/
   phase. Positive budget, ambiguous prose or path disjointness grants nothing.
   A current invalid canonical map is an error, never a fallback to history.
   At controller cap>1, resolve sequence authority and start actual immediate-
   successor preparation before ordinary local Temper/Define/Plan/Vet context or
   dispatch open, and retain the full Build gate before claims/rounds/history.
   Follow [early preparation](parallel-dispatch.md#early-preparation-admission)
   for native launch/start, internal markers and bounded preparatory-only routing.
   Missing targets need native Spec → Clarify → Temper → Define/Plan → affected
   Vet; existing targets need current completed/readiness state, registered pending
   native preparation, or an existing human/dependency hold. Completed live/
   archived successors need no reopening; valid finite tails discharge the
   obligation. Placeholders, reports and unopened waves prove nothing.
   Opt-out, zero opening budget or source human/question holds prevent new
   openings, not existing child preparation or independent work.
   If preparation cannot start, retain `frontier-blocked: <successor> — <constraint>`
   in source `evidence.md`. `frontier-blocked: sequence — <constraint>` may defer
   unresolved historical normalization for valid independent local work. Each row
   names one concrete native-reviewed constraint; keep it visible and revisit on
   change. It discharges only that scheduling obligation, not readiness, malformed
   canonical/AFK data, paths, claims, isolation, proof or human gates. It cannot
   waive already-ready foreign picks or certify preparation/normalization.
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
   Discharged deferrals are not repeatedly actionable. Foreign work/preparation
   and authority normalization precede local Build; local-only reconciliation,
   repair and picks retain their owning phase, not recursive Autocomplete.
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
4. Start every selected native writer before waiting; continue independent native
   planning/review while they run. Protected validation defers unsafe landings,
   not independent preparation. Outside that window, land individually green
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
