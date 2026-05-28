# Reference intake — design/style materials from the human

During the spec phase the human **may** hand you **how it should look or behave**:
screenshots, mockups, a Figma link, a video of a flow, a reference site, or a doc — **or
nothing at all** (no design, no screenshots, no explanation is perfectly normal). This is
entirely **optional**. When references *are* given, treat them as first-class inputs:
gather them, understand them, **save the local ones**, and index them so every later phase
(build, prove, polish, seal) can check the work against them. When there are none, skip
this and proceed with the spec.

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
| Ref | Type | Location | Shows / why it's a reference | Informs |
|-----|------|----------|------------------------------|---------|
| R1 | screenshot | references/login-mockup.png | target login layout + spacing | spec UI, slice 2, polish |
| R2 | figma | https://figma.com/… (+ export references/tokens.png) | design tokens, component set | all UI |
| R3 | video | references/checkout-flow.mp4 | expected step order + transitions | build, prove |
| R4 | link | https://example.com | tone + density to match | craft |
```

## Feed them into the spec
- Use references to sharpen `spec.md` (UI impact, success/acceptance — e.g. "matches R1")
  and the `design-brief.md` when UI is involved.
- A reference can *resolve a gap* ("which layout?") — record that in the gaps table.
- If a reference **conflicts** with the existing design system, that's an issue to raise
  with the human (match the system, or adopt the reference — their call).

## Later phases use them (wire-through)
`devrites-frontend-craft` builds **to** the references; `/rite-polish` compares the built
UI **against** them; `/rite-prove` / `browser-evidence.md` verify against them; `/rite-seal`
checks "matches the agreed design references" as acceptance. So save them once here and
everything downstream can connect to them.
