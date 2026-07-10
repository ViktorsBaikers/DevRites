# Reference intake — design/style materials from the human

During the spec phase the human **may** hand you **how it should look or behave**:
screenshots, mockups, a Figma link, a video of a flow, a reference site, or a doc — **or
nothing at all** (no design, no screenshots, no explanation is perfectly normal). This is
entirely **optional**. When references *are* given, treat them as first-class inputs:
gather them, understand them, **save the local ones**, and classify each as a fidelity
**target**, a hard **constraint**, or **inspiration**. That role prevents a mood reference
from silently becoming a pixel-match contract. When there are none, skip this and proceed
with the spec.

## Gather + understand each reference
- **Images / screenshots / mockups** — open and **view** them (the Read tool renders
  images). Describe what they show: layout, components, spacing, states, the target look.
- **Figma link** — if a Figma integration is available, pull its design context (frames,
  tokens, components); otherwise record the link and ask the human for an export/screenshot
  so there's something concrete to match offline.
- **Links / reference sites / docs** — fetch and read for the relevant intent (tone,
  layout, interaction, quality bar). Capture *why* it's a reference, not just the URL.
- **Video** — note what it demonstrates (the expected interaction/flow). Save it; later
  phases can step through it as tooling allows.

## Save local assets into the workspace
Copy any local file (screenshot, mockup, video, export) into
`.devrites/work/<slug>/references/` so it's durable and connectable later:
```
mkdir -p .devrites/work/<slug>/references
cp "<the file the human gave>" .devrites/work/<slug>/references/<clear-name>.<ext>
```
Use clear names (`login-mockup.png`, `checkout-flow.mp4`). Don't move the user's
original; copy it. For remote-only refs (a live Figma/URL), record the link in the index.

## Index in `references.md`
```markdown
# References: <slug>
| Ref | Role | Type | Location | Shows / why it's a reference | Informs |
|-----|------|------|----------|------------------------------|---------|
| R1 | target | screenshot | references/login-mockup.png | approved login composition + spacing | spec UI, slice 2, proof |
| R2 | constraint | figma | https://figma.com/… (+ export references/tokens.png) | required tokens + component set | all UI |
| R3 | target | video | references/checkout-flow.mp4 | approved step order + transitions | build, proof |
| R4 | inspiration | link | https://example.com | useful tone + density, not identity/layout | shape |
```

Roles are normative: **target** means compare fidelity, **constraint** means satisfy the
named rule, and **inspiration** means extract the cited principle without copying identity,
composition, copy, or distinctive assets.

## Feed them into the spec + the design brief
- Use references to sharpen `spec.md` (UI impact, success/acceptance — e.g. "matches R1").
- When the feature touches UI, these references are the primary input to **`devrites-ux-shape`**
  (spec step 3a): they anchor the design direction and can seed the visual-direction probe
  (a Figma link → pulled design context; reference sites → screenshots). The resulting
  `design-brief.md` cites them by R-id **and role**.
- A reference can *resolve a gap* ("which layout?") — record that in the gaps table.
- If a reference **conflicts** with the existing design system, that's an issue to raise
  with the human (match the system, or adopt the reference — their call).

## Later phases use them (wire-through)
`devrites-frontend-craft` builds to the approved brief and its **target** references,
honors **constraints**, and uses **inspiration** only for the named principle. `$rite-prove`
records rendered comparisons in `browser-evidence.md`; `$rite-polish` and `$rite-seal`
reuse that same contract. Save and classify once here so every downstream phase makes the
same fidelity decision.
