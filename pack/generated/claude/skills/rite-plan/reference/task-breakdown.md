# Task breakdown

How to turn a spec into ordered, vertical slices in `tasks.md`.

## Each slice is one task, in this format
```markdown
## SLICE-001 <name>
Goal:                       # single observable capability
Satisfies: AC-001[, AC-002] # reverse traceability to spec.md
Acceptance criteria:        # binary, evidence-backed (see ../../rite-spec/reference/acceptance-criteria.md)
Files likely touched:       # real paths from codebase inspection
Tests/proof:                # command(s) or EVID-### that prove it
Browser proof required:     # yes/no (yes if UI — see ../../rite-build/reference/frontend-trigger.md)
Frontend craft required:    # yes/no
Mode: AFK | HITL
Gate: advisory | validating | blocking | escalating
Dependencies:               # SLICE-### IDs that must land first
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
5. Write or update `traceability.md` so every `AC-###` maps to ≥1 `SLICE-###`, a
   planned proof, evidence status, and likely touched files.
6. Sanity check: every acceptance criterion in `spec.md` maps to ≥1 slice; no slice has
   an unowned criterion.

## Keep it honest
- Don't pre-write code in the task — describe the outcome, not the implementation.
- Don't bundle "while we're here" work into a slice. That's scope creep; log it as a
  follow-up instead.
- Don't slice to a target number — the count comes from the capabilities + the sizing
  rule (`slicing.md`), not a figure anyone named.
