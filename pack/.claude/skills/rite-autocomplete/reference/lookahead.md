# Sequence lookahead (`--lookahead`)

> Root-only: not inlined in --role bundles.

Plans the next continuation while the active workspace builds. Spec → Vet
overlaps the predecessor's Build → Seal.

## Start and target

Start `/rite-autocomplete --lookahead` once `.devrites/ACTIVE` is in Build or later. The agent never launches a session. Target: the next continuation
([sequence continuation](loop.md#sequence-continuation)). Stop when that workspace
is before Build, no entry remains, `continue_sequence: false`, or the budget is spent.

## Lease

Before any write:
`devrites-engine claim add --session lookahead-<target> --ttl 240 --workspace <target> --reason lookahead`.
Exit 3: held, stop. Re-add the same session each phase. Release with `devrites-engine claim release` on stop. A `dead` lease frees itself.

## Rules

- Preserve the ordered owner's `.devrites/ACTIVE`; bind every phase and handoff to the target per shared [phase target binding](../../devrites-lib/reference/parallel-workspaces.md#phase-target-binding).
- After the workspace exists, run `devrites-engine state set <target> sequence_lookahead=yes`.
- Run Spec → Clarify → Temper → Define → Plan → Vet. Define and Plan read the predecessor tree and plan. Record one assumption per predecessor interface used.
- At Vet record `Predecessor readiness: <active-slug> <digest>` in `eng-review.md` (the `--emit-binding` hex). A later planning change blocks `DRV-GATE-READINESS-STALE`; re-vet before Build.
- This explicit second session stays planning-only: no Build, product source/tests, commits or `.devrites/AFK` writes, even after predecessor Seal. Main controller may later consume the prepared target under [`parallel-workspaces.md`](../../devrites-lib/reference/parallel-workspaces.md); its sanctioned isolated Build is not a second same-tree writer. Acceptance, scope and irreversible risk still stop.
- No spawn tool is not gate `access` or `awaiting_human`. Root writes Define and Plan. No Vet `READY` until those roles are admitted; else stop `running`, `next_action: /rite-vet`.
- On every entry, re-read `state.md` and `questions.md`. Do not report a stop from chat. `continuation: clear` with `awaiting_human` is not done: fold resolved decisions into the spec, then continue. An except-principle or change-the-spec question complies: change the spec and resolve it. Ask only when the approved objective cannot comply.
- Stop at Vet `READY` with `next_action: /rite-build <target>` as a cursor for the main controller (not permission for this lookahead session to Build), at a real human gate, or when the lease is lost. Release the planning claim so the main controller can recheck/admit it.

**Failing case:** lookahead session starts Build; main controller is blanket-blocked until predecessor Seal despite current native independent admission; or Define/Plan/Vet are skipped because predecessor is still Build. Missing strategy-reviewer spawn is not gate `access`.

**Failing case:** stops to except a principle the spec can avoid.
