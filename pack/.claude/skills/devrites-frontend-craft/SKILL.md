---
name: devrites-frontend-craft
description: Implement frontend at senior designer-engineer quality — design-system discovery, shape-before-code, every state, a11y + WCAG 2.2 AA, anti-AI-slop, 2026 CWV bar, fullstack via contract-first slices. Use when the user says "build the UI", "ship-quality frontend", "design system", "a11y", or `/rite-build` detects UI. Not for polishing a built feature or design exploration.
user-invocable: false
---

# devrites-frontend-craft — UI like a senior designer-engineer

Build UI that belongs in *this* product, handles every state, and avoids generic-AI
tells. Integrated into the feature slice, not a separate design project.

## 1. Foundation discovery
Framework, routing, components, tokens, CSS methodology, icon set, existing UI patterns,
and any `PRODUCT.md` / `DESIGN.md` / design docs. Use the project's system — don't
import a new one. (Detail: `reference/design-references.md`.)
- **Load the design references the spec gathered** — `.devrites/work/<slug>/references.md`
  + the saved files in `references/` (screenshots, Figma, video, links). When present they
  are the **build target**: match the layout, spacing, states, and behavior they show
  (pull Figma context if a Figma integration is available). A reference that conflicts
  with the design system is a question for the user, not a silent choice.

## Reuse first — search before you build ([reference/reuse-first.md](reference/reuse-first.md))
Before creating any new **component, style, token, icon, hook, util, or helper**, **search
the project** for an existing one that fits — using the code-intelligence index
(`codegraph` / `graphify`) or grep over `components/`, design tokens, `hooks/`, `utils/`,
`lib/`. **Reuse → extend → build new** (in that order). Don't fork an existing component
into the feature folder. **AHA caveat:** don't force reuse on a wrong shape — duplication
is cheaper than the wrong abstraction. Record per slice what was reused / extended /
created new (so the seal can see the consistency story).

## 2. Register detection
- **Brand surface** — landing, marketing, portfolio, campaign: expressive type, larger
  scale contrast, motion welcome.
- **Product surface** — dashboard, admin, settings, app UI, tools: system fonts
  legitimate, one family often enough, fixed rem scale, tight ratio, density is fine.

## 3. Shape before code ([reference/shape.md](reference/shape.md))
User goal · primary action · information hierarchy · **all states** (default, loading,
empty, error, success, disabled, long-content) · responsive behavior · accessibility
requirements · interaction model. **Ask before coding if the visual direction or UX
flow is ambiguous.**

## 4. Build ([reference/craft.md](reference/craft.md))
- **Reuse first** — apply [reuse-first](reference/reuse-first.md) before reaching for new
  code. Compose from existing components/tokens; extend only if it fits without distortion.
- Build the **smallest UI** the current slice needs — don't pre-build screens.
- Don't add a second component library or icon set without asking.
- Cover the states you shaped, not just the happy path.
- **Reduce cognitive load** — no wall of options: group, mark the recommended choice, use
  progressive disclosure.
- **Copy in the product's voice**; shift tone by moment — success brief, error empathetic
  + actionable, loading reassuring, destructive serious. Never humor in errors. Empty
  states say *why* + the next action.
- **First-use**: get the user to first value fast; onboarding proves worth, it doesn't
  teach the whole product.

## 5. Verify & record (meet the bar)
- Hit the [2026 quality bar](reference/quality-standards.md): Core Web Vitals
  (LCP ≤2.5s / INP ≤200ms / CLS ≤0.1), WCAG 2.2 AA (keyboard, visible focus, contrast,
  ≥24px targets / 44px touch, no drag-only), responsive at 320/768/1024/1440 — and run
  its verification gate (no console errors, no axe violations, all states).
- Verify in the browser (`devrites-browser-proof`) — screenshots opened and described,
  console clean, responsive + keyboard checked.
- Record design decisions in `design-brief.md` and evidence in `browser-evidence.md`.

## Fullstack (frontend + backend in one feature)
When the feature needs both sides, follow [reference/fullstack.md](reference/fullstack.md):
define the **API/data contract first** (`devrites-api-interface`), slice **vertically**
through the layers (DB → service → API → UI) one capability at a time, apply the
engineering rules to the backend and this craft to the frontend, map every contract error
to a real UI state, and **prove both layers** (contract tests + browser proof).

## Anti-AI-slop (banned defaults unless the project's system uses them)
Purple/blue gradients · gradient text · glassmorphism by default · cards-in-cards ·
identical card grids everywhere · rounded-square icon tile above every heading ·
gray-on-color text · hero-metric cliché · decorative bounce/elastic easing · random
Inter-for-everything · modal-first thinking. Full list + rationale:
`rite-polish/reference/anti-ai-slop.md`.

## Default vs departure
Preserve the existing identity (default, ~90%). Reject it only on an explicit signal (a
design doc naming *this* surface as the failure, or the user asking to rebuild). If
unsure, you're in default mode — the cost of a wrong departure is unrecoverable.
