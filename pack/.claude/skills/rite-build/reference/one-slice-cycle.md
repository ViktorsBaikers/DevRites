# One-slice cycle

The discipline that makes large features manageable: build one thin slice, leave it
working and proven, stop. Never implement the whole feature in one pass.

## The cycle
```
SELECT  → restate slice goal + acceptance + scope boundary
LOAD    → only the files this slice touches
(SHAPE) → if UI: devrites-frontend-craft before code
(RED)   → if behavior change: write the failing test first
IMPLEMENT → smallest complete version; match conventions
(DOUBT) → before standing a non-trivial decision: devrites-doubt
PROVE   → targeted tests + browser proof for UI
RECORD  → state.md, evidence.md, touched-files.md
STOP    → report + recommend next; do not start slice N+1
```

## Why stop after one slice
- Keeps the diff reviewable (~one capability, roughly ≤100 lines of meaningful change).
- Surfaces integration/drift problems while they're cheap.
- Gives the user a decision point — they may reprioritize, reslice, or polish now.
- Prevents the "90% done" pile-up where nothing is actually proven.

## Restate the scope boundary
Before coding, write what this slice will and will **not** touch. This is the contract
you check yourself against — anything outside it is scope creep or a drift event.

## When the slice can't be completed cleanly
- Discovered the plan is wrong → **Spec Drift Guard** (stop, record, classify, maybe
  ask, `/rite-plan` repair).
- Slice is bigger than one cycle → stop and `/rite-plan reslice`.
- Blocked on a failing dependency → record in `state.md`, `/rite-plan unblock`.
Never power through by guessing.
