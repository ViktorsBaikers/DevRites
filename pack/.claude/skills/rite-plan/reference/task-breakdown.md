# Task breakdown

How to turn a spec into ordered, vertical slices in `tasks.md`.

## Each slice is one task

Use the complete canonical grammar in
[`workspace-artifact-schema.md`](../../devrites-lib/reference/workspace-artifact-schema.md#canonical-slice-grammar).
This reference owns decomposition; the shared schema owns field names and meanings.

## Process
1. List the capabilities the spec promises.
2. Map each to a vertical slice (thinnest end-to-end cut).
3. Order by dependency; risk-first within a tier (do the scary slice early while it's
   cheap to change).
4. Mark which slices touch UI (→ frontend craft + browser proof), map each to the design
   brief states, and give each a binary visual acceptance target. Mark which slices cross
   a module/service boundary (→ `devrites-api-interface`, `devrites-doubt`).
5. Write or update `traceability.md` so every `AC-###` maps to ≥1 `SLICE-###`, a
   planned proof, evidence status, and likely touched files.
6. Sanity check: every acceptance criterion in `spec.md` maps to ≥1 slice; no slice has
   an unowned criterion.
7. Resolve every foreseeable human-owned checkpoint now. Keep a build-time checkpoint only
   when it needs evidence unavailable before code or mandatory action-time approval, and say why.
8. Make every `Tests/proof` entry executable: exact command, working directory, prerequisites,
   expected signal, and mutable inputs whose provenance evidence must bind.

## Keep it honest
- **Interfaces prevent cross-slice drift.** A slice-wright sees only its own contract; the
  `Interfaces` line is how it learns the names and types neighboring slices use. A function
  called `clearLayers()` in slice 3 and `clearFullLayers()` in slice 7 is a planning bug:
  name shared symbols once, here.
- Don't pre-write code in the task: describe the outcome, not the implementation.
- Don't bundle "while we're here" work into a slice. That's scope creep; log it as a
  follow-up instead.
- Don't slice to a target number: the count comes from the capabilities + the sizing
  rule (`slicing.md`), not a figure anyone named.
