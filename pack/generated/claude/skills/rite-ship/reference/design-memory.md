# Design memory: record proven design rules in `DESIGN.md`

This optional `/rite-ship` step applies only to UI features. Feature design decisions
live in `design-brief.md` and archive with the workspace. With the user's approval,
copy proven, reusable design rules into the project-level `DESIGN.md` for later
features.

`devrites-ux-shape` §1 and
`devrites-frontend-craft/reference/design-references.md` read `DESIGN.md` when present.
This step writes that file from sealed feature evidence.

## When it runs
- **UI features only.** No UI in the diff → skip silently.
- **At ship, GO sealed, after the git plan, before type-GO** (step 2a), so the user sees
  the full staged change set (code + `DESIGN.md`) under the single type-GO that ships it.
- **Opt in and confirm.** A project-wide artifact is outside feature scope
  (`standards/core.md` rule 7), so never write it silently. Present a ranked option set
  (see [`afk-hitl.md`](../../devrites-lib/reference/standards/afk-hitl.md), "Option set");
  **default is skip**. The user opts to persist.
- **AFK:** first-time `DESIGN.md` creation is a `validating` gate because it adds a
  persistent project artifact. Propose and queue it; do not create it automatically. An append to an
  existing `DESIGN.md` is `advisory` and may auto-proceed when `allow_gates` permits, since
  it only adds evidence-proven entries.

## Inputs (proven only)
- `design-brief.md`: direction, color strategy, **calibration** (density / motion), key
  states, named anchors, and the per-slice **Build-time refinements**.
- The **final diff** and `touched-files.md`: the tokens, components, and states that
  shipped. Record only what the diff proves, not everything the brief intended.
- Existing `DESIGN.md` if present: the merge target.

## What to record
Record only reusable rules established by the shipped, proven feature. Exclude choices
that apply only to this feature.
- **Tokens introduced + used:** new color roles / spacing steps / type steps / elevation
  / radius that shipped and are reusable. Names and values, mirrored from the code.
- **Color strategy and calibration baseline:** the strategy (Restrained / Committed / …) and
  established density / motion; record a deviation as a deviation, not a new
  default.
- **Type & spacing scales** in actual use; **motion** classes + easing; **materiality**
  (elevation set, hairline usage, glass / texture policy).
- **Component behaviors:** for each shared component established or extended by the
  feature, record its shipped states and interaction model.
- **Named-anchor lineage:** the anchors used by shipped features.
- **Anti-slop exceptions:** any banned default the project *intentionally* uses (with the
  why), so build / polish don't "correct" a deliberate choice.

## Merge discipline
- **Append, don't overwrite.** Add proven entries; never silently rewrite an existing one.
- **Ask about conflicts.** If a new token, font, or strategy contradicts an existing
  `DESIGN.md` entry, present the option set to the user. Do not resolve it yourself.
  Keep one design system per project.
- **Never invent.** Include only rules proven by the feature.
- **Attribute and keep it short.** Tag each new entry with the feature slug. Future
  specs read this file, so use concise entries.

## How it commits
`DESIGN.md` is a tracked project artifact and ships **in the feature commit**:
1. Write / update `<project-root>/DESIGN.md` from the template below.
2. **Append `DESIGN.md` to `touched-files.md`** so the existing commit scope (SKILL
   steps 2 and 4) stages it under the same type-GO. Do not create a second commit.
3. Note the rollup in `ship.md` (what was added, or "design-memory: skipped by user").

## `DESIGN.md` template (project-level, stack-agnostic)
```markdown
# DESIGN.md — <project> design memory

> Rolled up from shipped features by `/rite-ship` (design memory). Read by
> `devrites-ux-shape` and `devrites-frontend-craft` as the inherited system. Evidence-only:
> every entry shipped and was proven. Edit through a feature's ship, not by hand.

## Register & direction
Default register per surface (brand / product) and the scene-derived direction the product
keeps converging on.

## Color
Strategy baseline (Restrained | Committed | Multi-role | Saturated). Token roles
(surface / text / accent / border / danger / …) with values or token names. Dark-mode approach.

## Calibration
Density baseline (Airy | Balanced | Dense) · Motion baseline (Minimal | Standard | Expressive).
Surfaces that deviate + why.

## Typography
Families; the scale in use (fixed rem / fluid clamp); heading↔body ratio; weights.

## Spacing
The scale (4 pt multiples) and section rhythm in use.

## Motion
Classes in use, easing, `prefers-reduced-motion` handling.

## Materiality
Elevation token set, hairline usage, glass policy, texture / asset approach.

## Components (proven behaviors)
| Component | States shipped | Interaction model | First proven by |
|---|---|---|---|
| <name> | default/hover/focus/loading/empty/error/… | inline / navigated / modal · feedback | <slug> |

## Named-anchor lineage
The anchors features steered toward (keeps new work coherent).

## Anti-slop — project exceptions
Banned defaults this project *intentionally* uses, with the why (so build/polish don't undo them).
```

## NEVER (design memory)
- Never roll up unattended on first creation: a new persistent project artifact is a
  confirmed step, never an AFK silent write.
- Never write an entry the feature didn't *prove* shipped: no aspirational tokens.
- Never overwrite or silently reconcile a conflicting existing entry: surface it.
- Never persist a feature-only one-off as if it were the project default.
- Never add `DESIGN.md` to the commit without appending it to `touched-files.md` first
  (it would fall outside the ship's commit scope).
