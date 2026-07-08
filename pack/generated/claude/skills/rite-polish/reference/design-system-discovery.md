# Design-system discovery

Before touching UI, learn the system that already exists. Read it from the codebase —
don't impose a generic one.

## Register first
Decide the surface's **register** (drives every other rule):
- **Brand surface** — landing, marketing, portfolio, campaign. More expressive type,
  larger scale contrast, motion welcome.
- **Product surface** — dashboard, admin, settings, app UI, tools. System fonts are
  legitimate; one family often suffices; fixed rem scale; tighter ratio (1.125–1.2);
  density is a feature.

## What to discover
| Thing | Where to look |
|---|---|
| Design docs | `PRODUCT.md`, `DESIGN.md`, `docs/design/*`, Storybook |
| Tokens | `tailwind.config.*`, CSS custom properties, `theme.*`, tokens files |
| Shared components | components/ui dir, design-system package, imports neighbors use |
| Spacing scale | the spacing tokens / Tailwind scale actually in use |
| Type scale | heading/body sizes, font families, weights in use |
| Color roles | semantic names (primary/surface/muted/danger), not raw hexes |
| Icon set | the one icon library already imported |
| Interaction patterns | how existing buttons/menus/dialogs/toasts behave |
| Form patterns | label placement, validation, error display in existing forms |
| Neighboring screens | the closest existing feature — match its flow shape |

## Rules
- Use existing tokens/components **first**. A hard-coded value where a token exists is
  drift.
- Don't add a second component library or icon set without asking.
- Match the closest neighbor's IA and flow shape rather than inventing a new one.
- **Default vs departure**: preserve the existing identity (default, ~90% of cases).
  Reject it only on an explicit signal — a design doc that points at *this* surface as
  the failure, or the user asking to rebuild it. If unsure, you're in default mode.
