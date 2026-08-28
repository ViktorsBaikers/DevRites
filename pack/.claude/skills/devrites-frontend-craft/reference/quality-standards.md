# Frontend quality standards (2026)

The measurable bar UI work is held to. Project conventions win where they're stricter;
these are the floor, not the ceiling.

## Performance: Core Web Vitals (field/p75 targets)
- **LCP** (Largest Contentful Paint) ≤ **2.5 s**
- **INP** (Interaction to Next Paint: the current responsiveness metric, replaced FID)
  ≤ **200 ms**
- **CLS** (Cumulative Layout Shift) ≤ **0.1**
- Keep an interactive page's shipped JS lean (a budget around **≤400 KB gzipped** is a
  good default); lazy-load below-the-fold and heavy/optional code; minimize hydration;
  size and reserve space for images/media to avoid layout shift.

## Accessibility: WCAG 2.2 AA
- **Semantic HTML first:** real `<button>`/`<nav>`/`<label>`/headings; ARIA only to fill
  gaps semantics can't.
- **Keyboard**: every interactive element operable by keyboard; logical tab order; **focus
  visible** with ≥ **3:1** contrast against its background (never remove the outline
  without an equal replacement).
- **Contrast**: text ≥ **4.5:1** (≥ 3:1 for large text and UI/graphics).
- **Target size**: ≥ **24×24** CSS px (WCAG 2.2 AA, SC 2.5.8); prefer **44×44** for primary
  touch targets.
- **No drag-only** interactions: provide a single-pointer alternative (SC 2.5.7).
- Labels/names on all controls; errors announced; respects `prefers-reduced-motion`.
- **Test** with keyboard, a screen reader, and an automated checker (e.g. axe): early.

## Motion
- Purposeful only; UI feedback ~≤200 ms, transitions ~≤500 ms (class table below). Never
  animate to mask slow loading; honor `prefers-reduced-motion`.

## Responsive
- Fluid layouts; no fixed widths that break. Verify at **320 / 768 / 1024 / 1440** px; no
  horizontal scroll; survives **200% text zoom**.
- **`dvh`, not `vh`**, for full-height surfaces (`min-height: 100dvh`, never `h-screen` /
  `100vh`: mobile address bars break them). A hero fits the *initial* viewport: headline
  ≤2 lines, primary CTA above the fold.

## Design system
- Use the project's **tokens** (spacing/type/color roles): never hard-code a value a
  token covers. Compose from existing components; one icon set.

### Consistency locks (one per surface)
Lock these in `design-brief.md` and hold them page-wide: mid-scroll drift ("a different
website by section 7") is a tell: **one accent** (no new accent appearing later on the
page), **one radius scale**, **one theme** (light or dark, not both).

## Numerical bar (enforceable specifics)

These are the hard numbers that move "good UI" from opinion to checkable.
Project tokens win when stricter; these are the floor.

### Color
- **OKLCH-only** for new tokens: perceptually uniform, predictable lightness
  steps across hues. Stop reaching for HSL; the lightness lies.
  - Example: `oklch(0.62 0.18 264)` for a saturated primary.
- **Never `#000` / `#fff`** as raw values. Pure black/white are too harsh and
  too clinical; use a near-black (`oklch(0.18 0 0)`) and near-white
  (`oklch(0.98 0 0)`) or the project's surface tokens.
- Contrast follows WCAG 2.2: text ≥ 4.5:1, UI/graphics ≥ 3:1. Verify
  against the *rendered* background, not the theoretical one.
- **Tint neutrals toward the brand hue:** pure-grey neutrals look sterile.
  Nudge every grey 0.005-0.01 chroma toward the brand colour: invisible at
  a glance, but the surface stops feeling generic.
- **Alpha is a smell:** heavy translucency usually means an incomplete palette.
  Define explicit overlay/hover tokens; transparency is for focus rings and scrims.
- **Every colored surface token ships its `on-` pair** (`primary`/`on-primary`,
  `destructive`/`on-destructive`, …) so contrast is designed in, not patched after.

#### Color commitment: pick the strategy before the palette
Decide *how committed* the surface is to colour before opening the picker.
Four positions on a single axis; pick one, then design within it:

| Strategy | Rough coverage | Use for |
|---|---|---|
| **Restrained** | tinted neutrals + 1 accent ≤ ~10% of the surface | Most product UI; brand surfaces that want to look quiet. |
| **Committed** | one saturated colour carries 30-60% of the surface | Brand pages with a strong identity; product feature surfaces that need a hero colour. |
| **Multi-role** | 3-4 named colour roles, each used deliberately | Brand campaigns; product data-viz with distinct meanings. |
| **Saturated** | the surface *is* the colour: full-bleed colour ground | Brand heroes, campaign pages, splash moments. |

Two corollaries:
- The "one accent ≤ ~10%" rule belongs to **Restrained only.** Committed /
  Multi-role / Saturated exceed it deliberately. Don't collapse every
  surface back to Restrained by reflex.
- *Register* (brand vs product) doesn't pick the strategy by itself: brand
  surfaces aren't always Saturated, product surfaces aren't always
  Restrained. Pick from the scene, not the category.

### Calibration: density & motion
Colour commitment fixes *how much colour*; two more axes fix *how much space* and
*how much movement*. Set one position on each (from the **scene sentence**, the same
way colour is picked) and carry both in `design-brief.md` so the build targets a
calibration instead of re-deciding it per slice.

**Density**: information per viewport; drives which spacing steps dominate:

| Position | Feels like | Spacing steps that dominate | Use for |
|---|---|---|---|
| **Airy** | gallery / calm / room to breathe | `32 / 48 / 64 / 80 / 96` | brand pages, onboarding, focus moments, low-data surfaces |
| **Balanced** | most product UI | `16 / 20 / 24 / 32` | dashboards, forms, settings: the default |
| **Dense** | cockpit / data-rich / power tool | `4 / 8 / 12 / 16` | tables, monitors, terminals, pro tools lived in all day |

**Motion**: how much the surface moves; drives which motion classes (table below) are in play:

| Position | Feels like | Motion classes in play | Use for |
|---|---|---|---|
| **Minimal** | crisp, almost still | Instant + State | dense tools, regulated/trust surfaces, reduced-motion-leaning |
| **Standard** | responsive, purposeful | Instant + State + Layout | most product + brand UI: the default |
| **Expressive** | choreographed, scroll-aware | + Entrance, deliberate sequencing | brand heroes, launch/campaign moments, story scroll |

Corollaries (same shape as colour commitment):
- Pick from the **scene sentence**, not the category: "an on-call SRE at 2am" → Dense +
  Minimal; "a launch hero in afternoon light" → Airy + Expressive. Register doesn't decide
  it by itself.
- `prefers-reduced-motion` overrides Motion **downward at runtime** regardless of position:
  an Expressive surface still ships a Minimal path.
- The position is a target, not a straitjacket: a surface may break density locally for
  emphasis: name the exception in the brief, don't let the whole page drift off it.

### Spacing
- **4pt base scale** (`4 / 8 / 12 / 16 / 20 / 24 / 32 / 40 / 48 / 64 / 80 /
  96`). Project tokens may use a multiplier; never hardcode in-between
  values.
- Inline spacing rhythms in multiples of 4 px. Optical alignments may need a
  1 - 2 px nudge: record the *why* in a comment.

### Typography scale
- **Brand surfaces**: fluid `clamp()` headings (e.g.,
  `clamp(2rem, 1rem + 3vw, 3.5rem)`). Display/body pairing OK.
- **Product surfaces**: **fixed rem scale** (1.125 - 1.2 ratio). One family
  usually carries headings, body, labels, data.
- Body line-height ~1.5; headings ~1.1 - 1.25. Prose width ~ 65-75 ch.
- Display ceiling: `clamp()` max ≤ ~6rem: larger is shouting. Display
  letter-spacing floor **≥ −0.04em**; tighter and the letters touch.
- `text-wrap: balance` on h1-h3; `text-wrap: pretty` on long prose.

### Motion
| Class | Duration | Use |
|---|---|---|
| Instant | 100 - 150 ms | Hover/press, tooltip enter, color shifts |
| State | 200 - 300 ms | Toggle, focus ring, dropdown |
| Layout | 300 - 500 ms | Modal/drawer enter, route transitions |
| Entrance | 500 - 800 ms | Page entrance, hero animation |

- **Exit at ~75 % of enter.** A 300 ms enter pairs with a 225 ms exit.
- **Bounce / elastic easing banned** unless the project's design system
  explicitly uses it.
- Honor `prefers-reduced-motion`: reduce or remove non-essential motion
  entirely.

### Dark mode (three-axis compensation)
A token-flip dark mode is sterile. When the project ships dark, compensate
three axes from light:
- **Line-height** `+0.05 – 0.10` (text breathes more on dark surfaces).
- **Letter-spacing** `+0.01 – 0.02 em` (perceived spacing tightens on dark).
- **Weight** `+1 step` for very small or low-contrast text.

If the project has dark tokens already, follow them. If not and dark is in
scope, propose the compensation rather than ship a flat invert.

### Focus & states (8 required, 3 conditional)
Every interactive element ships **8 visual/interaction states**:
`default`, `hover`, `active`, `focus-visible`, `disabled`, `loading`,
`selected`, and an error/invalid surface when relevant. Data surfaces add the
conditional three whenever the data can produce them: **partial** (a missing field
renders an explicit em-dash/placeholder — never `null` or `0`), **conflict** (a
concurrent-edit/version-mismatch surface), and **offline/unreachable** (stale-data
banner with retry, not a silently cached render). **Failing case:** a row with a
missing value renders `0` or blank and the review reads it as real data.
- `:focus-visible` ring: **2 - 3 px**, **≥ 3:1** contrast against the
  background, **offset 2 px** so the focus is unambiguous on dense layouts.

### Container queries vs viewport queries
- **Component breakpoints:** use **container queries** (`@container`). The
  same card adapts to its column whether the viewport is 320 or 1920.
- **Page-level layout:** use viewport queries
  (`@media (min-width: ...)`) for sidebar collapses, header reflows.
- Mixing the two is normal; using only one is usually a code smell.

### Semantic z-index scale
Pick a scale per surface role; never use raw `z-index: 9999`. Suggested
floor:

| Role | z |
|---|---|
| Base content | `auto` / `0` |
| Sticky header | `50` |
| Dropdown / popover | `100` |
| Drawer / sheet | `200` |
| Modal | `300` |
| Toast | `400` |
| System dialog / debug overlay | `500` |

### Materiality (elevation & surface)
Depth is earned, not defaulted. Reach **down** this ladder in order; stop at the first
rung that carries the structure.
- **Hairline first.** A **1 px border / divider** (a hairline in a near-neutral token)
  defines structure before any shadow does. Borders separate; shadows lift. Most "cards"
  want a hairline, not elevation.
- **Elevation is a token scale, not an ad-hoc value.** When a surface genuinely lifts
  (menu, popover, dragged item), use a **small shadow set from the design system** with one
  consistent light source: never a one-off `box-shadow`. The elevation step and the
  semantic z-index role move together.
- **Texture needs contrast.** Grain / noise / pattern earns its place only when it reads
  against the surface. Invisible texture is bytes with no signal: drop it.
- **Asset-led material.** Rich material (photographic grain, real product imagery,
  generated art) comes from a **real raster / generated asset**, not an SVG/CSS
  approximation of one. Don't fake a photo with gradients; ship the asset or use a flat
  token.
- **No decorative glass.** `backdrop-filter: blur(...)` is for a fixed/sticky surface over
  moving content (a sticky header, a sheet over scroll), never a default panel look
  (`rite-polish/reference/anti-ai-slop.md`).

### NEVER (UI numerical bar)
- Never reintroduce anything [`rite-polish` anti-ai-slop](../../rite-polish/reference/anti-ai-slop.md) bans.
- Never hard-code a spacing value the 4 pt scale or project tokens cover.
- Never animate an exit at 100 % of enter duration (feels uncontrolled).
- Never use raw `z-index` numbers outside the semantic scale.
- Never use viewport queries for component-internal reflows when the
  component is reused at different widths.
- Never ship dark mode as a flat token invert (compensate three axes).
- Never reach for a shadow where a 1 px hairline carries the structure.
- Never invent a one-off `box-shadow` outside the elevation token set.
- Never ship texture / grain that doesn't read against its surface.

## Verification gate (a UI slice isn't done until all pass)
- [ ] Renders with **no console errors/warnings**
- [ ] **Keyboard**: tab through reaches everything; focus visible; Esc/Enter behave
- [ ] **Screen reader** conveys content + structure
- [ ] **All states**: default / loading / empty / error / success / disabled
- [ ] **Responsive** at 320 / 768 / 1024 / 1440; no overflow; zoom-safe
- [ ] **No accessibility violations** (axe or equivalent)
- [ ] Meets the **CWV budget** above (measure, don't assume)
- [ ] Aligned to the **design system** (tokens, components, type, spacing)
- [ ] Clickables show `cursor: pointer` + a hover state; icons are SVG from the one set
      (no emoji); nothing trapped under fixed/sticky bars
- [ ] **Consistency locks hold** (one accent / one radius scale / one theme) and the
      mechanical pre-flight passes (`rite-polish/reference/anti-ai-slop.md`)
