# Slicing

DevRites builds in **thin vertical slices**: each slice cuts through every layer it
needs (data → logic → API → UI) and leaves the system in a working, testable state.

## Vertical, not horizontal
Horizontal layering delays integration feedback. Vertical slicing delivers a usable path:
```
SLICE-001: Create a task   (DB + API + minimal UI)   -> user can create + test passes
SLICE-002: List tasks      (query + API + UI)         -> user can see them
SLICE-003: Edit a task     (update + API + UI)        -> user can modify
SLICE-004: Delete a task   (delete + API + UI + confirm) -> full CRUD
```

## Sizing a slice
A slice is the right size when it:
- delivers one observable capability end-to-end;
- can finish its writer, independent review, and slice proof within `$rite-build`;
- has acceptance criteria you can verify with evidence;
- can be rolled back on its own;
- is worth its own review gate: a reviewer could meaningfully reject *this* slice while
  approving its neighbor. If two slices can only pass or fail together, they're one slice.

Size by independent invariants, failure modes, and proof seams, not file count or
the word "and". Split independently acceptable/provable responsibilities; several
files may serve one inseparable capability. Full feature `$rite-prove` follows all
slices; slice proof never replaces final integration or acceptance proof.

For `Complexity` and irreducible boundaries, use the canonical
[`tasks.md` slice grammar](../../devrites-lib/reference/workspace-artifact-schema.md#canonical-slice-grammar).
Keep the smallest cut that stands alone end-to-end; do not add slices or gates for
fragments that can only pass together. No slice size guarantees first-pass correctness.

## How many slices?: derive, don't dictate
The slice **count is an output, not an input.** It falls out of the work: one slice per
independently-shippable vertical increment that satisfies one (or a tight group of)
acceptance criteria and passes the sizing test above. A 2-criterion feature is 1-2 slices;
a 12-criterion feature is however many thin end-to-end cuts those 12 need.

- **Never slice to a target number.** Don't pad a small feature into "5 slices" or
  compress a large one into "3" because a figure was named.
- **A user-supplied count is a hint at most.** If the user says "do it in 4", slice the
  work logically first; if your honest decomposition differs, present the logical count
  and *why*, then let them redirect: don't silently force their number.
- **Re-size by the rule, not the tally** (`$rite-plan reslice`): split a slice because it
  failed the sizing test, not to hit a count.

`.devrites/AFK` `max_slices` limits unattended execution, never plan decomposition.

## Feature ceiling: split an epic, never override the budget

The derived count is also a size signal. A decomposition that cannot fit the
`tasks.md` / `plan.md` / `traceability.md` budgets in
[`workspace-artifact-schema.md`](../../devrites-lib/reference/workspace-artifact-schema.md)
(about ten canonical slices) is an epic: every Vet/Doubt/Build review re-reads the
whole surface, each round finds new material, and the loop never converges.
`Budget override` never covers slice or acceptance-criteria count.

- Keep the thinnest independently shippable subset here; the rest becomes spec
  `Non-goals` (via the Spec Drift Guard), each continuation a named future `$rite-spec`
  workspace. Record the sequence once in `decisions.md`.
- Acceptance-preserving: every REQ/AC keeps its ID, owned by exactly one workspace.
- Earliest owner wins: Temper's one-sentence test → Define/Plan decomposition → Vet §0
  scope finding → `$rite-plan course-correct` (`MVP cut`). Mid-build, unbuilt slices
  beyond the ceiling move; built slices stay.
- The ceiling bounds one **workspace**, never the product: the sequence delivers every
  slice, in order, each with its own Vet/Build/Review loop. A human may still accept a
  single larger workspace by recording the decision in `decisions.md` and a structural
  `Budget override:` in the affected artifacts; the extra review rounds on the whole
  surface are then the accepted, named cost. The deterministic gate
  (`check readiness`/`check seal`) blocks only a material overshoot without such an
  override, so the choice stays explicit rather than accidental.

### Continuation workspaces

Granularity is a real cost choice, not a formality: a workspace costs ~8–12 fixed
dispatches (drafter, plan reviewer, proof runner, spec/code reviewers, seal roster;
Temper is skippable when the parent's `strategy.md` already covers the scope) plus
~3–5 per slice. The split does not remove per-slice cost; it bounds each round's
context, stops the non-convergent whole-surface loop, isolates a plan defect's blast
radius to its own workspace, and lets independent continuations run in parallel.

Make each continuation cheap and drift-free:

- `brief.md` names the parent slug and the sequence position; `decisions.md` holds the
  sequence once (in the parent).
- The child's `state.md` seeds the durable cursor at creation — `sequence_parent` =
  parent slug, `sequence_position` = parent position + 1,
  `sequence_workspaces_remaining` = parent remaining − 1, and `sequence_role:
  release` only when the recorded entry is the sequence's release milestone.
  Values derive from the parent's `state.md` and its `decisions.md` record,
  exactly once; resume re-derives the same values rather than charging again.
  A workspace that is not a recorded continuation leaves all four rows absent.
  This is what `state merge-manifest` walks and what the release-union gate reads,
  so a manually opened continuation needs the same seeding as an armed run.
- Read the parent's `architecture.md`, `plan.md`, `decisions.md` (or its archived copy)
  as **design input** — never re-derive settled architecture, dependency choices, or
  interfaces. Proof is always fresh for the new candidate; parent evidence is cited as
  input, never as proof for this workspace.
- Shared interfaces, data shapes, and contracts are pinned **once**, in the workspace
  that introduces them, and referenced afterwards. A continuation that must change a
  pinned contract does so through its own `plan.md` shared-contract proof and an ADR
  that names the superseded owner — never by silently re-deciding it.
- A request that names a recorded continuation is a new workspace, not
  `$rite-plan revise` on the parent.
- Phases chain automatically only **inside** one workspace (`$rite-autocomplete`
  drives spec → clarify → temper → define → vet → build × slices → prove → polish →
  review → seal; plain HITL chains plan↔vet inline and stops at each other boundary).
  Next continuation is a new invocation by default. Same-run chaining is
  `continue_sequence: true` (Deferred-ship).

### Release milestone (full-picture gate)

Milestone gates are feature-scoped: `prove` proves *that* workspace's acceptance,
`review`/`seal` judge *that* candidate. A product release therefore needs one more
workspace — the **release milestone** — planned last, whose subject is the assembled
product:

- Its ACs are product-level: cross-milestone integration, end-to-end journeys,
  performance/release budgets, docs/rollout. Milestone ACs stay owned by their
  milestones and are **referenced** here, never re-owned.
- Its candidate manifest lists the union of shipped surfaces, so one `prove` → `review`
  → `seal` binds the whole product, and one `ship` commits/pushes/tags the release.
  Its `state.md` cursor carries `sequence_role: release` (seeded when the recorded
  continuation is the sequence's release entry); `check candidate`/`check seal`
  then require the manifest to cover the whole recorded chain. Build that union
  deterministically: `devrites-engine state merge-manifest <release-slug>` walks
  the recorded `sequence_parent` chain (live `work/` first, then `archive/`) and
  folds every predecessor's manifest into the release `## Candidate manifest`;
  on a path collision the later sequence position's row wins. An explicit
  ordered `<pred>...` argument exists for chains recorded before the sequence
  cursor fields.
- Earlier milestones keep their own slice-level proof and their own prove/review/seal;
  that is what stops a late defect from invalidating 170 slices of work. Skipping a
  milestone's proof is allowed only by recording the decision — the release milestone
  then carries the full acceptance burden, and that cost is the named trade-off.
- The capability ledger (`.devrites/specs/<capability>/spec.md`) accumulates the
  product picture across milestones; Polish folds each milestone's deltas.
- **Deferred-ship sequence** (`.devrites/AFK` `continue_sequence: true`, `max_workspaces: N`):
  milestones are sealed but not shipped, and the armed run opens the next recorded
  continuation automatically; local `WIP(<slug>):` checkpoints still land with
  human summaries (never a slice id),
  so the tree stays clean. One release ship at the end collapses the sequence's
  checkpoints into the single release commit, pushes/tags it, and may archive the
  sealed predecessors. Because no milestone boundary was shipped, later work can touch
  an earlier milestone's surfaces without an affected-re-proof trigger — so the release
  milestone re-proves the union acceptance map, and that full re-proof is this mode's
  named cost. Trade-off against per-milestone ship: no shared-history rollback point
  per increment, one Git approval at the end.
- What is full-surface here is the **review**: reviewers see the whole product diff.
  Re-proof stays targeted — this milestone's product ACs, everything a later milestone
  touched, and any uncertain closure. Milestone evidence is cited only under
  [evidence validity](../../devrites-lib/reference/candidate-integrity.md#evidence-validity)
  with explicit unchanged-input/dependency justification and a link to the original
  account; a blind re-proof of every milestone AC is a recorded decision, not a default.

## First slice
Make slice 1 the **thinnest useful end-to-end path**. It flushes out integration and
convention surprises early, while changes are cheap.

**Risk-first exception.** When the biggest risk is a technical *unknown*: "will this library
even do X?", "can we hit the latency target?": let slice 1 (or a throwaway spike ahead of it)
prove that unknown, even if it isn't the thinnest user-facing path. Fail the risky bet first,
while pivoting is still cheap.

## Slice independence
Order by dependency, but minimize coupling. A slice that needs three other slices first
is a smell: look for a thinner cut that stands alone.
