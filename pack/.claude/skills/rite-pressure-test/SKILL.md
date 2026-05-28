---
name: rite-pressure-test
description: Pressure-test a rough idea — diverge into 3-5 genuinely different options, then converge on one with the trade-off and decision hinge. Use when the user says "ideate", "refine this idea", "stress-test my plan", "I have a vague idea", or `/rite-spec` flags the concept as rough. Not for writing the spec (use `/rite-spec`).
user-invocable: true
---

# rite-pressure-test — diverge then converge

Use when the *idea* (not just the requirements) is rough. Generate options, then commit
to one — so `/rite-spec` has a real direction to specify.

## Diverge (widen)
- Generate 3–5 genuinely different approaches to the underlying goal, not variations of
  one. Cover at least: the obvious approach, a simpler/smaller approach, and a
  different-shape approach (different data model, flow, or boundary).
- For each: one-line description, what it optimizes for, rough cost, main risk.
- Stay concrete — name real entities, flows, and surfaces, not abstractions.

## Converge (commit)
- Weigh options against the goal, constraints, and existing codebase conventions.
- Recommend one, with the reason and the key trade-off accepted.
- Note what would change the recommendation (the decision's hinge).

## Boundaries
- This is exploration, not specification. Output a **direction**, not a finished spec —
  `/rite-spec` writes the spec.
- Don't over-explore: 3–5 options, one pass of convergence. If the user already knows
  the direction, skip this and go straight to `/rite-spec`.
- Ask the user to pick when two options are close and the choice changes the product.

## Output
```
Goal (restated): ...
Options:
  A. <approach> — optimizes <x>, cost <y>, risk <z>
  B. ...
Recommendation: <option> because <reason>; trade-off accepted: <...>
Hinge: <what would change this>
Next: /rite-spec <feature>  (using the chosen direction)
```
