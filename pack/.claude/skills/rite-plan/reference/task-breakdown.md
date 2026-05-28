# Task breakdown

How to turn a spec into ordered, vertical slices in `tasks.md`.

## Each slice is one task, in this format
```markdown
## Slice N: <name>
Goal:                       # single observable capability
Acceptance criteria:        # binary, evidence-backed (see acceptance-criteria.md)
Files likely touched:       # real paths from codebase inspection
Tests to write/run:         # the command(s) that prove it
Browser proof required:     # yes/no (yes if UI — see frontend-trigger.md)
Frontend craft required:    # yes/no
Dependencies:               # slice numbers that must land first
Rollback notes:             # how to back this slice out
Evidence required:          # what /rite-prove must capture
```

## Process
1. List the capabilities the spec promises.
2. Map each to a vertical slice (thinnest end-to-end cut).
3. Order by dependency; risk-first within a tier (do the scary slice early while it's
   cheap to change).
4. Mark which slices touch UI (→ frontend craft + browser proof) and which cross a
   module/service boundary (→ `devrites-api-interface`, `devrites-doubt`).
5. Sanity check: every acceptance criterion in `spec.md` maps to ≥1 slice; no slice has
   an unowned criterion.

## Keep it honest
- Don't pre-write code in the task — describe the outcome, not the implementation.
- Don't bundle "while we're here" work into a slice. That's scope creep; log it as a
  follow-up instead.
