# `design-brief.md` template — the UX/UI contract

The feature-level design decision, produced **before** code by `devrites-ux-shape` (at
`/rite-spec` when UI is detected) and the **target** the build, polish, and seal check
against. `devrites-frontend-craft` refines it per slice and appends build-time decisions;
it is never re-derived from scratch.

Pick the shape by how clear the answers are. **Compact** is the default for a crisp prompt;
**full** when the surface is ambiguous, multi-screen, or the user asked to shape it as a
step. Don't pad a clear brief into a long one to look thorough.

## Compact form (default)
```markdown
# Design brief: <feature>
Slug: <kebab>   Shaped: <iso>   Register: brand | product   Fidelity: <sketch|mid-fi|high-fi|production>

- **Building**: <one line — what + who it's for + the primary action>
- **Direction**: <color strategy> · "<scene sentence>" · anchors: <ref A, ref B>  (+ references/<file> R-ids)
- **States**: <the states this surface needs, comma-listed>
- **Interaction**: <inline / navigated / modal · feedback · entry→completion in a phrase>
- **Confirm or override?** <the one or two things you still want the user to confirm>
```

## Full form
```markdown
# Design brief: <feature>
Slug: <kebab>   Shaped: <iso>   Register: brand | product

## 1. Summary
What this is, who it's for, what it must accomplish (2-3 sentences).

## 2. Primary action
The single most important thing the user does or understands here. Everything else is secondary.

## 3. Design direction
- Color strategy: Restrained | Committed | Multi-role | Saturated — why (from the scene, not the category).
- Scene sentence: "<who uses it, where, under what light, in what mood>".
- Named anchors: <2-3 specific products / brands / objects> + saved references (R-ids → references/<file>).
- Departure: default (preserve identity) | departing because <explicit signal>.

## 4. Scope  *(task-scoped — not persisted to PRODUCT.md / DESIGN.md)*
Fidelity · breadth · interactivity · time intent.

## 5. Layout & hierarchy
What gets emphasis; what's seen 1st / 2nd / 3rd; how information flows. Hierarchy, not CSS.

## 6. Key states
Every state + what the user needs to see and feel:
| State | User sees / feels | Notes |
|---|---|---|
| default / populated | | |
| loading | | initial + subsequent |
| empty | | welcoming + the next action |
| error | | what happened + how to recover |
| success | | |
| disabled / no-permission | | |
| long-content / overflow / many items | | |

## 7. Interaction model
Inline vs navigated vs (rarely) modal; optimistic vs pending; focus & keyboard; feedback on
every action; the flow from entry to completion.

## 8. Content & copy
Labels, empty/error messages, microcopy in the product's voice. Realistic dynamic ranges.
For image-led surfaces: required media roles + likely source (project asset / generated /
SVG-CSS / icon library / accepted omission).

## 9. Responsive & a11y
Reflow 320→1440 (what stacks / collapses / scrolls); focus order, labels, contrast,
keyboard, target size. Floor: `../devrites-frontend-craft/reference/quality-standards.md`.

## 10. Visual-direction probe
Which probe ran (figma / images / prototype / skipped — no tool), which direction won, what
changed in the brief because of it. Artifacts saved under `references/`.

## 11. Open questions
Genuinely unresolved only. If you'd write "Recommend: X", decide X instead.

## Build-time refinements  *(appended by devrites-frontend-craft per slice)*
- Slice <N>: <decision made while building this surface + why>.
```

## How downstream phases use it
- `/rite-define` — UI slices map to the **Key states** + **Interaction model**; each UI
  slice names which states it covers.
- `/rite-build` + `devrites-frontend-craft` — build **to** the brief; refine per slice;
  append refinements above instead of re-deriving.
- `/rite-prove` + `browser-evidence.md` — verify the built UI against the brief's states +
  references.
- `/rite-polish` — Phase 4 reads the brief so polish honors the agreed direction.
- `/rite-seal` + `devrites-frontend-reviewer` — check the UI matches the brief (states
  covered, direction held, references matched).
