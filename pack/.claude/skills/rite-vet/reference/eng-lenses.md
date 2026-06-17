# Senior-engineer lenses — how to *see* the findings

These are not extra checklist items. They are the instincts an experienced staff engineer
applies while reading a plan — the pattern recognition that separates "read the plan" from
"caught the landmine". Apply them across all four axes in [`review-axes.md`](review-axes.md):
each lens turns a vague unease into a concrete, quotable finding.

## The lenses

1. **Boring by default.** A team gets a few innovation tokens; everything else should be proven
   technology. When the plan introduces something novel (a new datastore, a hand-rolled queue, a
   bespoke pattern), ask whether this is a wise token to spend or whether a boring built-in does
   it. → Architecture / Scope axes.
2. **Blast radius.** For every decision: what's the worst case, and how many systems / callers /
   users does it touch? A change with wide blast radius and no rollback is a different risk class
   than a leaf change. → Architecture / Reversibility.
3. **Reversibility preference.** Prefer the choice that's cheap to undo — feature flag, additive
   migration, incremental rollout — over the one that's right only if you're right the first time.
   Make the cost of being wrong low. → Architecture / Reversibility.
4. **Incremental over revolutionary.** Strangler-fig, not big-bang rewrite; refactor first, then
   change behavior — never both in one slice. If the plan does a structural rewrite and a behavior
   change together, split them. → Plan code-quality / slice shape.
5. **Systems over heroes.** Design for a tired engineer at 3am, not your sharpest engineer on
   their best day. A plan that only works if every step is done perfectly is fragile. Favor
   guard rails, explicit errors, and obvious failure paths. → Code-quality / Failure modes.
6. **Essential vs accidental complexity.** Before anything the plan adds, ask Brooks's question:
   is this solving a real problem, or one we created? Accidental complexity is the cheapest thing
   to cut. → Scope / Code-quality.
7. **Make the change easy, then make the easy change.** If a slice is hard because the surrounding
   structure fights it, the first move is a (separate, behavior-preserving) refactor — call it out
   so the plan sequences it before the feature work, not tangled into it. → Plan code-quality.
8. **Failure is information.** Every new codepath fails *somehow* in production — timeout, nil,
   race, stale data, partial write. The plan should name the failure and decide: test it, handle
   it, or accept it loudly (never silently). → Failure-mode coverage.
9. **DX is product quality.** Slow CI, painful local dev, untestable seams → worse software over
   time. If the plan makes the code harder to test or run locally, that's a real finding, not a nit.
   → Test-coverage / Code-quality.
10. **Trust boundary discipline.** Every untrusted input crosses an explicit validation + authz
    boundary before it reaches trusted core logic. A plan that lets a value skip the boundary is a
    security finding even pre-code. → Architecture / Security seam.

## Using them
- A lens is a *prompt*, not a verdict. It turns "something's off here" into a finding you can
  quote and band. If a lens fires, follow it to the specific plan line, then run it through the
  confidence + verification gate.
- Don't recite the lenses in the output — recite the *findings* they produced.
- When a plan is genuinely clean against a lens, that's worth one line of evidence ("Reversibility:
  additive migration + flag — strong"), not silence; it shows the lens was applied.
