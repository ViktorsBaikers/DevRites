---
name: devrites-frontend-craft
description: Implement frontend at senior designer-engineer quality — every state covered, anti-AI-slop, WCAG 2.2 AA. Use when the user says "build the UI", "design system", "a11y", or `/rite-build` detects UI. Not for polishing a built feature or design exploration.
user-invocable: false
---

# devrites-frontend-craft — UI like a senior designer-engineer

Build UI that belongs in *this* product, handles every state, and avoids generic-AI
tells. Integrated into the feature slice, not a separate design project.

## 1. Foundation discovery
Framework, routing, components, tokens, CSS methodology, icon set, existing UI patterns,
and any `PRODUCT.md` / `DESIGN.md` / design docs. Use the project's system — don't
import a new one. (Detail: `reference/design-references.md`.)
- **Load the design brief + references the spec gathered** —
  `.devrites/work/<slug>/design-brief.md` (the UX/UI contract from `devrites-ux-shape` — the
  primary build target) plus `references.md` + the saved files in `references/`
  (screenshots, Figma, video, links). When present they are the **build target**: match the
  direction, layout, spacing, states, and behavior they show (pull Figma context if a Figma
  integration is available) — extract type / spacing / color-roles / layout / component
  behavior to it deliberately, don't eyeball it
  (`reference/design-references.md` — *Building to a supplied reference*). A reference that
  conflicts with the design system is a question for the user, not a silent choice.

## Reuse first — search before you build
Before creating any new component, style, token, icon, hook, util, or helper, search the
project for an existing one and **reuse → extend → build new** — see
[reference/reuse-first.md](reference/reuse-first.md) for the search targets, the AHA
caveat, and the per-slice reuse record.

## 2. Register detection
- **Brand surface** — landing, marketing, portfolio, campaign: expressive type, larger
  scale contrast, motion welcome.
- **Product surface** — dashboard, admin, settings, app UI, tools: system fonts
  legitimate, one family often enough, fixed rem scale, tight ratio, density is fine.

## 3. Shape before code — build to the brief ([reference/shape.md](reference/shape.md))
The feature's **`design-brief.md`** is your target — `/rite-spec` shaped it up front
(`devrites-ux-shape`): design direction, key states, interaction model, the visual-direction
probe. **Read it first and refine it for this slice's surface; don't re-derive the design
from scratch.** Confirm the slice covers the brief's states for this surface (default,
loading, empty, error, success, disabled, long-content), its information hierarchy +
primary action, responsive behavior, a11y, and interaction model. **If a UI slice has no
`design-brief.md`** (a spec written before shaping), shape it now via `devrites-ux-shape`
before coding. **Ask before coding if the visual direction or UX flow is still ambiguous.**

## 4. Build ([reference/craft.md](reference/craft.md))
- Compose from existing components/tokens (reuse-first, above) before reaching for new code.
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
- **Every interactive element has an asserting test at the right level** — each field,
  checkbox, radio, select, toggle, button, and actionable link gets a unit/component test for
  what it *does* (validation, toggle, options, enabled/disabled, handler); critical journeys
  get one E2E. Browser proof shows it renders; the asserting test proves it works. No element
  ships unverified — `standards/testing.md` "Completeness"; inventory in `test-plan.md`.
- **Append build-time refinements** to `.devrites/work/<slug>/design-brief.md` — the
  design-contract artifact `devrites-ux-shape` produced at spec; this skill refines it per
  slice (the "Build-time refinements" section), it does not own or recreate it. `/rite-polish`
  reads it in its UI phase and `/rite-seal` includes it in its artifact read list. Record
  runtime evidence in `browser-evidence.md`.

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
