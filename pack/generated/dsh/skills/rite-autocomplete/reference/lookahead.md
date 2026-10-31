# Sequence lookahead (`--lookahead`)

> Root-only: not inlined in --role bundles.

A second session plans the next continuation while the active workspace
builds. Spec → Vet overlaps the predecessor's Build → Seal. The main loop
resumes that child.

## Start and target

Start `/rite-autocomplete --lookahead` once `.devrites/ACTIVE` is in Build or later. The agent never launches a session. The target is the next recorded continuation: the entry
[sequence continuation](loop.md#sequence-continuation) would open. Stop with
nothing to do when that workspace is before Build, no entry remains,
`continue_sequence: false`, or the sequence budget is spent.

## Lease

Before any write:
`devrites-engine claim add --session lookahead-<target> --ttl 240 --workspace <target> --reason lookahead`.
Exit 3: the target is held, stop. Re-add the same session each phase. Release with `devrites-engine claim release` on every stop. A `dead` lease frees itself.

## Rules

- Never read or write `.devrites/ACTIVE`. Pass the target slug on every command, including `/rite-spec <target> "<objective>"` and `devrites-engine handoff <target> --resume`.
- After the workspace exists, run `devrites-engine state set <target> sequence_lookahead=yes`.
- Run Spec → Clarify → Temper → Define → Plan → Vet. Define and Plan read the predecessor tree and plan. Record an assumption per predecessor interface the plan uses.
- At Vet record one `Predecessor readiness: <active-slug> <digest>` line in `eng-review.md`. `<digest>` is the hex from `devrites-engine check readiness --emit-binding <active-slug>`. A later predecessor planning change blocks with `DRV-GATE-READINESS-STALE`; the main loop re-vets before Build.
- Predecessor still in Build or later forbids only Build, product source, tests, commits, and `.devrites/AFK`. It does not skip Define, Plan, or Vet, and is not `awaiting_human`. Acceptance, scope, or irreversible-risk questions still stop via `questions.md`.
- No spawn tool is not gate `access` or `awaiting_human`. Record owed roles pending in `state.md`. Root writes Define and Plan. No Vet `READY` until those roles are admitted. Reviewers still owed: stop `running`, `next_action: /rite-vet`.
- Stop at Vet `READY` with `next_action: build after predecessor Seal GO`, at a
  real human gate, or when the target's lease is lost.

**Failing case:** lookahead passes Vet and starts Build while the predecessor is still sealing, so two sessions write one tree.

**Failing case:** Define, Plan, and Vet skipped because the predecessor is still Build, or a missing strategy-reviewer spawn recorded as gate `access`.
