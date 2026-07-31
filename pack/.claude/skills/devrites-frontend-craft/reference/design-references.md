# Design references: discover, don't impose

What to read in the project before designing, and how register changes the rules.

## Discover these
| Thing | Where |
|---|---|
| Design docs | `PRODUCT.md`, `DESIGN.md`, `docs/design/*`, Storybook, READMEs |
| Tokens | `tailwind.config.*`, CSS custom properties, `theme.*`, token files |
| Shared components | components/ui dir, design-system package, what neighbors import |
| Type | families, weights, the size scale in use |
| Spacing | the spacing scale / tokens in use |
| Color roles | semantic names (primary/surface/muted/danger), not raw hex |
| Icons | the one icon library already imported |
| Patterns | how existing buttons/menus/dialogs/forms/toasts behave |
| Neighbors | the closest existing feature: match its flow shape |

## Register-specific rules
**Product surface** (dashboard/admin/app):
- System fonts are legitimate (`-apple-system, BlinkMacSystemFont, "Segoe UI",
  system-ui, sans-serif`); Inter is a fine default for a reason.
- One family usually carries headings, body, labels, data.
- Fixed rem scale (not fluid `clamp()` headings); tighter ratio (1.125-1.2).
- Density is a feature; tables can run dense; prose still ~65-75ch.

**Brand surface** (landing/marketing/campaign):
- More expressive type and larger scale contrast are appropriate.
- Display/body pairing can be worth it; motion is welcome (still purposeful).

## The rule
Match what exists. A new token, font, or component library is a decision the user makes,
not a default you reach for. When the system is ambiguous, ask: don't invent the
project's intent. When a project `DESIGN.md` is present it is the **rolled-up design
memory** earlier features proved and closed during Polish
(`../../rite-polish/reference/design-memory.md`): the
inherited system to build *to*, ahead of re-deriving direction from scratch.

## Scene-sentence: commit before choosing theme / direction
Dark vs light is not a default, and neither is "playful", "serious", or "editorial".
Before picking any of those, write **one sentence** that fixes the physical scene:

> *who* uses this, *where*, under *what ambient light / device*, in *what mood*.

If that sentence doesn't make the answer feel inevitable, it isn't concrete enough:
add detail until it does. Then design *for that scene*, not for the category.

Examples:
- ❌ "An observability dashboard" → forces nothing; "dark blue tech" is the reflex.
- ✅ "An on-call SRE glancing at incident severity at 2am on a 27-inch monitor in a
  dim room" → forces dark, high-contrast severity colours, scannable rows.
- ❌ "A finance app" → forces nothing; "navy + gold" is the reflex.
- ✅ "A self-employed designer reconciling last month's invoices at a kitchen table
  in afternoon light, on a 13-inch laptop" → forces a calm light theme with one
  committed accent for the running total.

Run the sentence, not the category. When the user supplies the brief, paraphrase the
sentence back to them as confirmation before designing.

## Named anchor references: steer with specifics, not adjectives
After the scene sentence, name **2-3 specific anchors** the surface should feel like:
real products, brands, or objects ("Linear's command bar", "a Teenage Engineering device",
"the Stripe dashboard"), **not adjectives** ("modern", "clean", "premium"). Adjectives are
unfalsifiable; a named anchor is checkable. You can hold the built UI next to it and ask
"does it read like that?" Anchors steer *direction*, not pixel-copying: take the relevant
trait (density, type voice, restraint), not the literal layout, and never the parts that
clash with this project's register or design system. The supplied `references/` files are
themselves anchors: name what trait each contributes. Record the anchors in
`design-brief.md`'s **Design direction** so build, polish, and seal share the same target.

## Building to a supplied reference (Figma / screenshot / image)
When the spec gathered a Figma frame, screenshot, or image and the brief names it the
**build target** (not just inspiration), the reference *is* the art direction and the code
is the implementation layer. This is the build-time counterpart to the shape-time
visual-direction probe (`../../devrites-ux-shape/reference/visual-direction-probe.md`):
the probe *chose* a lane; here you *match* the chosen target.

**Extract before you write**: read the reference deliberately, don't eyeball it:
- **Type:** family, the size steps used, weights, line-height, letter-spacing,
  the heading→body ratio. Map each to the project's type scale (or flag a missing step).
- **Spacing & rhythm:** padding, gaps, section spacing; infer the underlying step and
  round to the project's 4 pt scale rather than hard-coding the measured pixel.
- **Color:** the roles in play (surface / text / accent / border), not raw hex; bind to
  existing tokens, propose a token only where one is genuinely missing.
- **Layout & hierarchy:** grid, what's seen 1st / 2nd / 3rd, the density and motion the
  reference implies (cross-check against the brief's Calibration).
- **Component behavior:** the states the reference shows (and the ones it can't: hover,
  focus, loading, empty, error: design those from the brief, the reference won't have them).

Then implement to match, in the project's system:
- **Match the target, fill the gaps from the brief.** A static reference shows one state;
  ship the full state set (see `quality-standards.md`, "Focus & states").
- **A reference that conflicts with the design system is a question for the user**, not a
  silent override: name the conflict (token, font, spacing, a second system) and ask.
- **Don't crop a multi-section reference into pieces** to "extract" a section: work from
  the cleanest whole frame; if a region is unreadable, ask for a clearer asset rather than
  guessing.
- **Fidelity is faithfulness, not pixel-tracing:** carry the reference's hierarchy, rhythm,
  and voice; never copy a layout that clashes with this project's register or a11y floor.
- Record what the reference dictated (and any deviation + why) in the brief's
  **Build-time refinements**, so polish and seal check the build against the same target.

## NEVER (design references)

- Never default to **Inter / DM Sans / Plus Jakarta / Fraunces / Newsreader**
  because they're the "tasteful 2024 default". Use what the project uses.
- Never default to a **purple/blue gradient** when the project has a brand
  palette.
- Never pick a tone (playful / serious / corporate / hand-drawn) that
  contradicts the surface's **register**: brand vs product
  (`devrites-frontend-craft/SKILL.md` §2).
- Never add a second design system to "modernize" without explicit user
  approval. One design system per project; consistency beats local taste.
- Never invent a token. If a needed color/spacing/type slot doesn't exist,
  ask whether to add it to the system.
- Never use a screenshot of another product as a target without checking
  the *register* matches. A landing page reference is not a product UI
  reference.
