# One-slice cycle

The discipline that makes large features manageable: build one thin slice, leave it
working and proven, stop. Never implement the whole feature in one pass.

## The cycle
The orchestrator (`$rite-build`) gates and records; the **wright** writes. See
[`wright-dispatch.md`](wright-dispatch.md).
```
SELECT    → orchestrator: restate slice goal + acceptance + scope boundary; HITL gate (pause pre-code)
(SHAPE)   → orchestrator: if UI and no design-brief.md, shape it (devrites-ux-shape) before dispatch
(FORGE)   → orchestrator: if slice is Forge: yes, engine plan → snapshot → bound isolated
            candidates → extract → judge → record + merge one winner, then continue at DOUBT (forge.md)
DISPATCH  → hand the slice contract to devrites-slice-wright (fresh context). Inside the wright:
   ORIENT    → load only the files this slice touches; learn the project's idiom; reuse-first
   (RED)     → if behavior change: write the failing test first
   IMPLEMENT → smallest complete version; match conventions; devrites-frontend-craft for UI
   VERIFY    → targeted tests (+ types/lint/build); fail-on-red; return artifact — code + tests only
(DOUBT)   → orchestrator, on return: devrites-doubt each non-trivial decision the wright stood up
PROVE     → orchestrator: fail-on-red check on the wright's gates; browser proof for UI
RECORD    → orchestrator: after green verification, Forge cleanup/report if used; then
            state.md, evidence.md, touched-files.md (the canonical writer)
(CHECKPOINT) → orchestrator: if .devrites/CHECKPOINT is set, commit the proven slice
            local-only as WIP(<slug>) with a [devrites-context] body → survives a crash (checkpoint.md)
STOP      → report + recommend next; do not start slice N+1
```

## Why stop after one slice
- Keeps the diff reviewable (~one capability, roughly ≤100 lines of meaningful change).
- Surfaces integration/drift problems while they're cheap.
- Gives the user a decision point. They may reprioritize, reslice, or polish now.
- Prevents the "90% done" pile-up where nothing is proven.

## Restate the scope boundary
Before coding, write what this slice will and will **not** touch. This is the contract
you check yourself against: anything outside it is scope creep or a drift event.

## When the slice can't be completed cleanly
- Discovered the plan is wrong → **Spec Drift Guard** (stop, record, classify, maybe
  ask, `$rite-plan` repair).
- Slice is bigger than one cycle → stop and `$rite-plan reslice`.
- Blocked on a failing dependency → record in `state.md`, `$rite-plan unblock`.
Never power through by guessing.
