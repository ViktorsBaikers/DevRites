# Design memory — roll proven design language into project `DESIGN.md`

Optional, UI-only step at `/rite-ship`. A feature's design decisions live in its
feature-scoped `design-brief.md` and are thrown away when the workspace archives. **Design
memory** is the deliberate exception: when a UI feature ships, roll the design language it
*proved* up into a project-level `DESIGN.md`, so the next feature's `devrites-ux-shape`
inherits the system instead of re-discovering it.

The consumers already exist — `devrites-ux-shape` §1 and
`devrites-frontend-craft/reference/design-references.md` both read `DESIGN.md` when present.
This step is the missing **producer**. It closes the loop: feature N seals its design
language → feature N+1 builds to it.

## When it runs
- **UI features only.** No UI in the diff → skip silently.
- **At ship, GO sealed, after the git plan, before type-GO** (step 2a) — so the user sees
  the full staged change set (code + `DESIGN.md`) under the single type-GO that ships it.
- **Opt-in and confirmed.** Persisting to a project-wide artifact is outside feature scope
  (`rules/core.md` rule 7) — so it is *never silent*. Present a ranked option set
  (`rules/afk-hitl.md` — Option set); **default is skip**. The user opts to persist.
- **AFK**: first-time `DESIGN.md` *creation* is treated as a `validating` gate (a new
  persistent project artifact) — propose + queue, don't auto-create. An *append* to an
  existing `DESIGN.md` is `advisory` and may auto-proceed when `allow_gates` permits, since
  it only adds evidence-proven entries.

## Inputs (proven only)
- `design-brief.md` — direction, color strategy, **calibration** (density / motion), key
  states, named anchors, and the per-slice **Build-time refinements**.
- The **final diff** + `touched-files.md` — what tokens / components / states *actually
  shipped* (the evidence). Roll up only what the diff proves, not what the brief intended.
- Existing `DESIGN.md` if present — the merge target.

## What to roll up — the project's *converged* system, evidence-gated
Only entries the shipped, proven feature establishes. Each is durable, cross-feature design
language — not this feature's one-off choices.
- **Tokens introduced + used** — new color roles / spacing steps / type steps / elevation
  / radius that shipped and are reusable. Names and values, mirrored from the code.
- **Color strategy + calibration baseline** — the strategy (Restrained / Committed / …) and
  density / motion the project keeps landing on; note a deviation as a deviation, not a new
  default.
- **Type & spacing scales** in actual use; **motion** classes + easing; **materiality**
  (elevation set, hairline usage, glass / texture policy).
- **Component behaviors** — for each shared component this feature established or extended:
  the states it ships and its interaction model. Grows one feature at a time.
- **Named-anchor lineage** — the anchors features steered toward, so later work stays
  coherent with what's already built.
- **Anti-slop exceptions** — any banned default the project *intentionally* uses (with the
  why), so build / polish don't "correct" a deliberate choice.

## Merge discipline
- **Append, don't overwrite.** Add proven entries; never silently rewrite an existing one.
- **Conflict is a question, not an edit.** A new token / font / strategy that contradicts an
  existing `DESIGN.md` entry → surface it to the user (the option set), don't resolve it
  yourself. One design system per project.
- **Never invent.** If the feature didn't prove it, it doesn't go in. `DESIGN.md` is design
  *memory*, not design *aspiration*.
- **Attribute + keep lean.** Tag each new entry with the feature slug so the lineage is
  legible; one good line beats a paragraph. The file is read every future spec — keep it
  scannable.

## How it commits
`DESIGN.md` is a tracked project artifact, so it ships **in the feature commit**:
1. Write / update `<project-root>/DESIGN.md` from the template below.
2. **Append `DESIGN.md` to `touched-files.md`** so the existing ship commit-scoping (SKILL
   steps 2 + 4) stages it — the design-memory write rides the same commit and the same
   type-GO, no second commit.
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
- Never roll up unattended on first creation — a new persistent project artifact is a
  confirmed step, never an AFK silent write.
- Never write an entry the feature didn't *prove* shipped — no aspirational tokens.
- Never overwrite or silently reconcile a conflicting existing entry — surface it.
- Never persist a feature-only one-off as if it were the project default.
- Never add `DESIGN.md` to the commit without appending it to `touched-files.md` first
  (it would fall outside the ship's commit scope).
