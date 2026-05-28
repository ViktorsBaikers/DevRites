# Design references — discover, don't impose

What to read in the project before designing, and how register changes the rules.

## Discover these
| Thing | Where |
|---|---|
| Design docs | `PRODUCT.md`, `DESIGN.md`, `docs/design/*`, Storybook, READMEs |
| Tokens | `tailwind.config.*`, CSS custom properties, `theme.*`, token files |
| Shared components | components/ui dir, design-system package, what neighbors import |
| Type | families, weights, the size scale actually in use |
| Spacing | the spacing scale / tokens in use |
| Color roles | semantic names (primary/surface/muted/danger), not raw hex |
| Icons | the one icon library already imported |
| Patterns | how existing buttons/menus/dialogs/forms/toasts behave |
| Neighbors | the closest existing feature — match its flow shape |

## Register-specific rules
**Product surface** (dashboard/admin/app):
- System fonts are legitimate (`-apple-system, BlinkMacSystemFont, "Segoe UI",
  system-ui, sans-serif`); Inter is a fine default for a reason.
- One family usually carries headings, body, labels, data.
- Fixed rem scale (not fluid `clamp()` headings); tighter ratio (1.125–1.2).
- Density is a feature; tables can run dense; prose still ~65–75ch.

**Brand surface** (landing/marketing/campaign):
- More expressive type and larger scale contrast are appropriate.
- Display/body pairing can be worth it; motion is welcome (still purposeful).

## The rule
Match what exists. A new token, font, or component library is a decision the user makes,
not a default you reach for. When the system is ambiguous, ask — don't invent the
project's intent.

## Scene-sentence — commit before choosing theme / direction
Dark vs light is not a default, and neither is "playful", "serious", or "editorial".
Before picking any of those, write **one sentence** that fixes the physical scene:

> *who* uses this, *where*, under *what ambient light / device*, in *what mood*.

If that sentence doesn't make the answer feel inevitable, it isn't concrete enough —
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

## NEVER (design references)

- Never default to **Inter / DM Sans / Plus Jakarta / Fraunces / Newsreader**
  because they're the "tasteful 2024 default". Use what the project uses.
- Never default to a **purple/blue gradient** when the project has a brand
  palette.
- Never pick a tone (playful / serious / corporate / hand-drawn) that
  contradicts the surface's **register** — brand vs product
  (`devrites-frontend-craft/SKILL.md` §2).
- Never add a second design system to "modernize" without explicit user
  approval. One design system per project; consistency beats local taste.
- Never invent a token. If a needed color/spacing/type slot doesn't exist,
  ask whether to add it to the system.
- Never use a screenshot of another product as a target without checking
  the *register* matches. A landing page reference is not a product UI
  reference.
