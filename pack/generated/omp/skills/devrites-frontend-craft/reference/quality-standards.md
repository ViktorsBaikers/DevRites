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
- Purposeful only (class table below). Never animate to mask slow loading; honor
  `prefers-reduced-motion`.

## Responsive
- Fluid layouts; no fixed widths that break. Verify at **320 / 768 / 1024 / 1440** px;
  preserve content and controls at **200% text zoom** and **400% browser zoom from
  1280 px** (320 CSS px). This is the canonical viewport set for UI evidence: `rite-polish/reference/ui.md`, `browser-proof-checklist.md`, and
  `browser-polish-evidence.md` cite it rather than restating it.
- **No page-level horizontal scroll anywhere in 320–1920.** Sweep between checkpoints:
  a `1fr` track given a 1024 px image can overflow at 900 px. Fix intrinsic sizing
  (for example `minmax(0, 1fr)`), wrapping and media constraints; root clipping is
  not a reflow fix. Clip decoration only, never text, controls or focus indicators.
  A necessary two-dimensional table/map may scroll inside a named, bounded,
  keyboard-accessible region; surrounding headings, text and pagination must reflow.
  Record and test that exception. **Failing case:** a localized CTA disappears behind
  `overflow-x: clip`; the missing scrollbar does not prove accessible reflow.
- **`dvh`, not `vh`**, for full-height surfaces (`min-height: 100dvh`, never `h-screen` /
  `100vh`: mobile address bars break them). A hero fits the *initial* viewport: headline
  ≤2 lines, primary CTA above the fold.

## Design system
- Use the project's **tokens** (spacing/type/color roles): never hard-code a value a
  token covers. Compose from existing components; one icon set.

### Consistency locks (one per surface)
Lock these in `design-brief.md` and hold them page-wide: mid-scroll drift ("a different
website by section 7") is a tell: **one declared color strategy and role budget**
(§ Color commitment; no unbriefed accent appearing later), **one radius scale**, **one theme** (light or dark, not both).

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
  against the *rendered* background, not the theoretical one. Procedure: pair every
  `color` declaration with its computed background; a button whose text and fill sit
  within 5% lightness and 0.05 chroma of each other fails regardless of the palette.
- **Restrained accent, capped:** one accent covers **≤5% of the viewport** at rest
  (3–5 placements above the fold). Accent-filled panels fail this strategy; another strategy requires the explicit brief and role budgets below.
- **Tint neutrals toward the brand hue:** pure-grey neutrals look sterile.
  Nudge every grey 0.005-0.01 chroma toward the brand colour: invisible at
  a glance, but the surface stops feeling generic.
- **Alpha is a smell:** heavy translucency usually means an incomplete palette.
  Define explicit overlay/hover tokens; transparency is for focus rings and scrims.
- **Every colored surface token ships its `on-` pair** (`primary`/`on-primary`,
  `destructive`/`on-destructive`, …) so contrast is designed in, not patched after.

#### Color commitment: pick the strategy before the palette
Decide *how committed* the surface is to colour before opening the picker. Four positions on a single axis; pick one, then design within it:

| Strategy | Rough coverage | Use for |
|---|---|---|
| **Restrained** | tinted neutrals + 1 accent ≤5% of the viewport at rest | Most product UI; brand surfaces that want to look quiet. |
| **Committed** | one saturated colour carries 30-60% of the surface | Brand pages with a strong identity; product feature surfaces that need a hero colour. |
| **Multi-role** | 3-4 named colour roles, each used deliberately | Brand campaigns; product data-viz with distinct meanings. |
| **Saturated** | the surface *is* the colour: full-bleed colour ground | Brand heroes, campaign pages, splash moments. |

Two corollaries:
- The ≤5% accent cap belongs to **Restrained only**. For Committed / Multi-role /
  Saturated, the brief must name the strategy, purpose, tokens and measured coverage
  budget per role and viewport. No blanket waiver: an unbudgeted colored panel fails.
  Preserve distinct success/warning/error meanings; semantic states do not authorize decorative accents. Check rendered captures against the declared budgets.
- *Register* (brand vs product) doesn't pick the strategy by itself: brand
  surfaces aren't always Saturated, product surfaces aren't always Restrained. Pick from the scene, not the category.

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
- Adjacent hierarchy levels differ by **≥1.5× in at least one signal** (size, weight,
  or color); verified under a blur(8px) squint — a flat field means the hierarchy is
  decorative. **Failing case:** label and helper text differ only by 0.05rem.
- All-caps display text sets **line-height ≥1.0**; tighter collides cap heights.
- **Hero font-scale attribution:** a hero headline that wraps past its planned lines at
  the target viewport is a font-size error first — fix the scale, don't shorten the
  copy. **Failing case:** a four-line hero headline "fixed" by cutting the sentence.
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
  entirely. Reduced means fewer and gentler, never zero feedback.
- **Loading feedback appears by ~200 ms**: skeletons/spinners replace stillness
  after that threshold; below it, feedback flicker reads as jank. Hover intents
  may delay 150–300 ms; focus indication never fades in — the ring is instant.

### Forms
- Inputs match button height (44 px floor); visual targets under 24 px expand
  their hit area with a pseudo-element to the 44 px minimum.
- Helper/error text reserves **`min-height: 1lh`** so messages never shift layout.
- Border width is pinned across states (state changes recolor, never resize).
- Disabled is visible through **three channels**: muted color, `not-allowed`
  cursor, and blocked interaction — color alone fails.
- Mobile inputs set **≥16 px** font (smaller triggers iOS zoom-on-focus).
- Numeric/data columns set **`font-variant-numeric: tabular-nums`**; columns of
  figures align or the surface reads broken.

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
`selected`, and an error/invalid surface (required wherever the element can be invalid). Data surfaces add the
conditional three whenever the data can produce them: **partial** (a missing field
renders an explicit em-dash/placeholder — never `null` or `0`), **conflict** (a
concurrent-edit/version-mismatch surface), and **offline/unreachable** (stale-data
banner with retry, not a silently cached render). **Failing case:** a row with a
missing value renders `0` or blank and the review reads it as real data.
- `:focus-visible` ring: **2 - 3 px**, **≥ 3:1** contrast against the
  background, **offset 2 px** so the focus is unambiguous on dense layouts.

### Browser chrome
User-agent defaults are unfinished craft. Per UI slice, theme or explicitly
decline (briefed) each of: `::selection` (token colors, not UA blue),
`caret-color` on editable fields, scrollbar styling or a recorded
`scrollbar-width` decision, and the project focus-ring token in place of the
unstyled UA outline. **Failing case:** layout, type, and the 8+3 states pass
while the UA blue focus ring and default selection remain.

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
- [ ] **All states**: the canonical 8 + 3 lattice above (§ Focus & states), not a
      shorter local list
- [ ] **Responsive** at the canonical viewport set (§ Responsive); no lost content or
      controls, no page overflow; bounded exceptions tested; zoom-safe
- [ ] **No accessibility violations** (axe or equivalent)
- [ ] Meets the **CWV budget** above (measure, don't assume)
- [ ] Aligned to the **design system** (tokens, components, type, spacing)
- [ ] **Browser chrome** themed or declined in the brief (§ Browser chrome)
- [ ] Clickables show `cursor: pointer` + a hover state; icons are SVG from the one set
      (no emoji); nothing trapped under fixed/sticky bars
- [ ] **Consistency locks hold** (declared color strategy/roles / one radius scale / one theme) and the
      mechanical pre-flight passes (`rite-polish/reference/anti-ai-slop.md`)

## Craft convergence bar

Craft work terminates; it does not loop. Findings classify as **Critical /
Major / Minor**: done means zero Critical, zero Major, and every remaining Minor
accepted **in writing** (owner + reason) in `polish-report.md` — an unwritten
minor is an ignored finding, not an accepted one. Re-running the full bar after a
fix pass happens only on an explicit opt-in ("re-run the full bar"); "keep going"
is not opt-in. Each pass captures every supported viewport, not only the one that
failed. **Failing case:** the 375 px overflow is fixed, 320 px still overflows,
and the report reads clean.
