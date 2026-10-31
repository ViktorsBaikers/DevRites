# Sequence lookahead (`--lookahead`)

> Root-only: not inlined in --role bundles.

Plans the next continuation while the active workspace builds. Spec → Vet
overlaps the predecessor's Build → Seal.

## Start and target

Start `$rite-autocomplete --lookahead` once `.devrites/ACTIVE` is in Build or later. The agent never launches a session. Target: the next continuation
([sequence continuation](loop.md#sequence-continuation)). Stop when that workspace
is before Build, no entry remains, `continue_sequence: false`, or the budget is spent.

## Lease

Before any write:
`devrites-engine claim add --session lookahead-<target> --ttl 240 --workspace <target> --reason lookahead`.
Exit 3: held, stop. Re-add the same session each phase. Release with `devrites-engine claim release` on stop. A `dead` lease frees itself.

## Rules

- Never read or write `.devrites/ACTIVE`. Pass the slug to `$rite-spec <target> "<objective>"` and `devrites-engine handoff <target> --resume`.
- After the workspace exists, run `devrites-engine state set <target> sequence_lookahead=yes`.
- Run Spec → Clarify → Temper → Define → Plan → Vet. Define and Plan read the predecessor tree and plan. Record one assumption per predecessor interface used.
- At Vet record `Predecessor readiness: <active-slug> <digest>` in `eng-review.md` (the `--emit-binding` hex). A later planning change blocks `DRV-GATE-READINESS-STALE`; re-vet before Build.
- Predecessor still in Build or later forbids only Build, product source, tests, commits, and `.devrites/AFK`. It does not skip Define, Plan, or Vet, and is not `awaiting_human`. Acceptance, scope, and irreversible risk still stop.
- No spawn tool is not gate `access` or `awaiting_human`. Root writes Define and Plan. No Vet `READY` until those roles are admitted; else stop `running`, `next_action: $rite-vet`.
- On every entry, re-read `state.md` and `questions.md`. Do not report a stop from chat. `continuation: clear` with `awaiting_human` is not done: fold resolved decisions into the spec, then continue. An except-principle or change-the-spec question complies: change the spec and resolve it. Ask only when the approved objective cannot comply.
- Stop at Vet `READY` with `next_action: build after predecessor Seal GO`, at a real human gate, or when the lease is lost.

**Failing case:** Vet then Build while the predecessor is still sealing, so two sessions write one tree. Or Define, Plan, and Vet skipped because the predecessor is still Build, or a missing strategy-reviewer spawn recorded as gate `access`.

**Failing case:** stops to except a principle the spec can avoid.
