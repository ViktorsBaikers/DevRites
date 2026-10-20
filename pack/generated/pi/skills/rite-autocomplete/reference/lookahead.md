# Sequence lookahead (`--lookahead`)

> Root-only: not inlined in --role bundles.

A second session plans the next sequence continuation while the active
workspace builds, so the child's Spec → Vet overlaps its predecessor's
Build → Seal. The main loop later resumes the planned child instead of
specifying it again.

## Start and target

The operator starts the second host session with `/rite-autocomplete
--lookahead` once the `.devrites/ACTIVE` workspace is in Build or later. The
agent never launches a session itself. The target is the next recorded
continuation after the active workspace: the entry
[sequence continuation](loop.md#sequence-continuation) would open. Stop with
nothing to do when the active workspace is before Build, no entry remains,
`continue_sequence: false`, or the sequence budget is spent.

## Lease

Before any write, take the workspace lease:
`devrites-engine claim add --session lookahead-<target> --ttl 240 --workspace <target> --reason lookahead`.
Exit 3 means another session holds the target: stop. Re-add the lease with the
same session at each phase start, and release it with `devrites-engine claim
release` on every stop. A crashed session's lease reads `dead` and frees itself.

## Rules

- Never read or write `.devrites/ACTIVE`. Pass the target slug explicitly to every
  skill and engine command, including `/rite-spec <target> "<objective>"` and
  `devrites-engine handoff <target> --resume`.
- After the workspace exists, run `devrites-engine state set <target> sequence_lookahead=yes`.
- Run Spec → Clarify → Temper → Define → Plan → Vet as the main loop does.
  Define and Plan read the predecessor's current tree and plan. Record an
  assumption for each predecessor interface the plan relies on.
- At Vet, beside the readiness binding, record exactly one line
  `Predecessor readiness: <active-slug> <digest>` in `eng-review.md`. `<digest>`
  is the hex that `devrites-engine check readiness --emit-binding <active-slug>`
  prints. When the predecessor later changes its planning inputs, the child's
  readiness check blocks with `DRV-GATE-READINESS-STALE` and the main loop
  re-vets the child before Build.
- Never Build, never write product source or tests, never commit, and never
  arm or rewrite `.devrites/AFK`. Owner questions go to the target's
  `questions.md` and stop the lookahead.
- Stop at Vet `READY` with `next_action: build after predecessor Seal GO`, at a
  human gate, or when the target's lease is lost.

**Failing case:** the lookahead passes Vet and starts Build while the
predecessor is still sealing, so two root sessions write one tree.
