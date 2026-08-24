# One-slice cycle

One wright dispatch builds one thin, proven slice. HITL stops; only explicit
`.devrites/AFK` lets the controlling root chain another pending slice under the
green-proof, cap, and pause rules. Each wright returns after exactly one slice.

## The cycle

The orchestrator (`$rite-build`) gates and records; the **wright** writes. See
[`wright-dispatch.md`](wright-dispatch.md).

```
SELECT    → orchestrator: restate slice goal + acceptance + scope boundary; HITL gate (pause pre-code)
(SHAPE)   → orchestrator: missing UI design-brief.md → shape it, then $rite-vet before dispatch
DISPATCH  → hand the slice contract to devrites-slice-wright (fresh context). Inside the wright:
   ORIENT    → load only the files this slice touches; learn the project's idiom; reuse-first
   (RED)     → if behavior change: write the failing test first
   IMPLEMENT → smallest complete version; match conventions; devrites-frontend-craft for UI
   VERIFY    → writer-safe tests/types/lint; return code + tests
(DOUBT)   → orchestrator, on return: devrites-doubt each non-trivial decision the wright stood up
PROVE     → root: required build/browser/E2E, then returned-diff/path review; fail-on-red
RECORD    → orchestrator: after green verification, update state.md/evidence.md and
            upsert the authoritative candidate manifest from the actual scoped diff
(CHECKPOINT) → orchestrator: if .devrites/CHECKPOINT is set, commit the proven slice
            local-only as WIP(<slug>) with a [devrites-context] body → survives a crash (checkpoint.md)
NEXT      → HITL root reports and stops; AFK root may repeat only under afk-discipline.md
```

## Why the boundary matters

- Keeps diffs reviewable and reveals integration or drift early.
- Preserves a HITL decision point and prevents unproven pile-ups.

## Restate the scope boundary

Before coding, write what this slice will and will **not** touch. This is the contract
you check yourself against: anything outside it is scope creep or a drift event.

## When the slice can't be completed cleanly

- Discovered the plan is wrong → **Spec Drift Guard** (stop, record, classify, maybe
  ask, `$rite-plan` repair).
- Slice is bigger than one cycle → stop and `$rite-plan reslice`.
- Blocked on a failing dependency → record in `state.md`, `$rite-plan unblock`.
Never power through by guessing.
