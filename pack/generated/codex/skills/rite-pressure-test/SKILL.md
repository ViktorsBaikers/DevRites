---
name: rite-pressure-test
description: Pressure-test a rough/vague idea: ideate, explore 3-5 approaches, radically different shapes, diverge then converge on direction before spec. Not for writing spec.
argument-hint: "[rough idea or plan to stress-test]"
user-invocable: true
---

# $rite-pressure-test: diverge then converge

Use when the idea—not only requirements—is rough. Compare directions before
`$rite-spec`; never implement. Read [`core.md`](../devrites-lib/reference/standards/core.md)
first and write only workspace/ticket artifacts.

## Diverge (widen)

- Search `decisions.md`, accepted ADRs, and relevant archives for rejected directions;
  revive one only with evidence that answers its recorded reason, citing both.
- Generate 3-5 different shapes, including obvious, smaller, and different-boundary/data/
  flow options. Use named lenses— inversion, constraint removal, audience shift, 10×, or
  expert practice—without padding. Borrow analogous mechanisms, not branding.
- For each, name concrete entities/flows/surfaces, optimization, rough cost, and main risk.

## Converge (commit)

- Weigh options against the goal, constraints, and existing codebase conventions.
- Prefer felt pain over nice-to-have. Rank differentiation: new capability > 10× gain >
  new audience > new context > better UX > cheaper; do not dress a vitamin as a painkiller.
- Recommend one, with the reason and the key trade-off accepted.
- Note what would change the recommendation (the decision's hinge).

## Premise evidence floor

If the hinge is an uncertain material fact, freeze at most three questions for
`devrites-evidence-scout`. Validate every citation; missing/malformed/unverifiable output is
`unavailable`, never support. Record claim, support, strongest contrary evidence, and
`supported | assumption | refuted`. Refutation changes the recommendation; an assumption
advances only when non-decisive with bounded downside. Weak/conflicting decisive evidence
returns **Hold** with the resolving evidence, never `$rite-spec`. Tie-breaker:
see [`intent-map.md`](../devrites-lib/reference/intent-map.md) (`rite-pressure-test`
vs `rite-spec`). Do not research preferences
or reversible implementation choices.

## Boundaries

- This is exploration, not specification. Output a **direction**, not a finished spec:
  `$rite-spec` writes the spec.
- Don't over-explore: 3-5 options, one pass of convergence. If the user already knows
  the direction, skip this and go straight to `$rite-spec`. If the effort is too foggy for
  one pass, start an investigation map at `.devrites/work/<slug>/investigation-map.md`
  with `Destination`, `Decisions so far`, `Not yet specified`, `Out of scope`, plus one
  frontier question per session.
- Ask the user to pick when two options are close and the choice changes the product.
- Name a **"Not doing" list**: the good options you deliberately cut. It's the highest-value
  output of convergence: it hands `$rite-spec` its scope boundary. When a cut is
  durable, offer to record the reason in the active feature's `decisions.md` or
  a durable ADR rather than a parallel rejection index.

```
Done: pressure test complete for <goal>.
Changed: workspace only
Evidence: options compared <n>; recommendation <option>; hinge <condition>; premises <supported/assumption/refuted + sources>
Open: <none | unresolved premise and resolving evidence>
Next: <when supported: $rite-spec <feature>; when Hold: none — requires <resolving evidence>>
Record: not applicable
↻ Hygiene: /clear before starting the lifecycle
```
