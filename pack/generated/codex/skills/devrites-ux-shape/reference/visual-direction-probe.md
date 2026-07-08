# Visual-direction probe — test the lane before the brief locks

Before finalizing `design-brief.md`, pressure-test the visual direction with something
concrete instead of words — *when it will actually clarify the brief*. This is the
DevRites-native answer to "build it according to the design (Figma, images)": show a
direction, get a reaction, then write the brief.

## When to run (all true)
- The work is **net-new** or directionally **ambiguous**. An existing surface to match →
  skip; the supplied references already ARE the target.
- Fidelity is **mid-fi or higher**. Sketch-only planning → skip.
- A probe tool is actually available (below). **Capability-gated** — never ask the user to
  install APIs or tooling. If none is available, **announce the skip in one line** and
  proceed to the brief. The one-line announcement is required; it forces a conscious choice
  instead of letting the step quietly evaporate.

## Pick the probe by what's available (first match)
1. **Figma reference supplied** → pull design context with the Figma integration (frames,
   tokens, components, a screenshot of the target). Record what it dictates. A Figma frame
   that conflicts with the project's design system is a **question for the user**, not a
   silent choice.
2. **Native image generation available** (Figma Make, an image-gen MCP, computer-use, or
   similar) → generate **2-4 distinct direction probes** from the discovery answers (color
   strategy, scene sentence, anchors). They must differ in **primary direction** —
   hierarchy, density, typographic voice, topology — not just palette tweaks. Treat them as
   *direction tests*, not final UI.
3. **A code-fidelity question** ("which of these layouts actually works in our stack") →
   route to **`$rite-prototype`** (2-4 real UI variations on one throwaway route). Prototype
   is DevRites' existing tool for this; don't rebuild it here.
4. **Reference sites / links only** → screenshot them (`devrites-browser-proof` tooling)
   and read the relevant intent (tone, density, interaction).
5. **Nothing available** → one-line skip, proceed.

## How to use the result
- Ask which direction feels closest, what's off, what carries forward. Don't treat
  generated imagery as final UX, copy, or accessibility behavior.
- If the probe reveals a mismatch, revise the **discovery inputs** (scene sentence, color
  strategy, anchors) *before* writing the brief — that's the whole point of probing.
- Save probe artifacts (generated images, Figma exports, screenshots) into
  `.devrites/work/<slug>/references/`, index them in `references.md` with what each shows,
  and note the winning direction in the brief's **Visual-direction probe** section.

## Limits
- Don't skip discovery because image generation is available.
- Don't run a probe for a minor refinement of existing work — it's for shaping a new
  surface or a big directional choice.
- Don't block the brief on a slow or failing tool — try once, then skip with the one-line
  note.
