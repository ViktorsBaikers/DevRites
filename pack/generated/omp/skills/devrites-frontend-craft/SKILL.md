---
name: devrites-frontend-craft
description: Build accessible production UI and all states during /rite-build. Use for frontend or design-system implementation; not for polish or design exploration.
user-invocable: false
---

# devrites-frontend-craft: UI like a senior designer-engineer

Build UI that belongs in *this* product, handles every state, and avoids generic-AI
tells. Integrated into the feature slice, not a separate design project.

## 1. Foundation discovery
Framework, routing, components, tokens, CSS methodology, icon set, existing UI patterns,
and any `PRODUCT.md` / `DESIGN.md` / design docs. Use the project's system: don't
import a new one. (Detail: `reference/design-references.md`.)
- Load the design brief + references the spec gathered (roles: target / constraint / inspiration — detail: `reference/design-references.md`).

## Reuse first: search before you build
Before creating any new component, style, token, icon, hook, util, or helper, search the
project for an existing one and **reuse → extend → build new**: see
[reference/reuse-first.md](reference/reuse-first.md) for the search targets, the AHA
caveat, and the per-slice reuse record.

## 2. Register detection
Read the discovery and register contract in
[`reference/design-references.md`](reference/design-references.md#discover-these).
Completion: the surface is classified as brand or product and its existing tokens,
components, patterns, and nearest neighbor are named before design choices begin.

## 3. Shape before code: build to the brief ([reference/shape.md](reference/shape.md))
The feature's **`design-brief.md`** is your target: `/rite-spec` shaped it up front
(`devrites-ux-shape`): design direction, key states, interaction model, the visual-direction
probe. **Read it first and refine it for this slice's surface; don't re-derive the design
from scratch.** Confirm the slice covers the brief's states for this surface (default,
loading, empty, error, success, disabled, long-content), its information hierarchy +
primary action, responsive behavior, a11y, interaction model, and proof targets. **If a UI slice has no
`design-brief.md`** (a spec written before shaping), shape it now via `devrites-ux-shape`
before coding. **Ask before coding if the visual direction or UX flow is still ambiguous.**

## 4. Build ([reference/craft.md](reference/craft.md))
- Compose from existing components/tokens (reuse-first, above) before reaching for new code.
- Build the **smallest UI** the current slice needs: don't pre-build screens.
- Don't add a second component library or icon set without asking.
- Cover the states you shaped, not just the happy path.
- **Reduce cognitive load:** no wall of options: group, mark the recommended choice, use
  progressive disclosure.
- **Copy in the product's voice**; shift tone by moment: success brief, error empathetic
  + actionable, loading reassuring, destructive serious. Never humor in errors. Empty
  states say *why* + the next action.
- **First-use**: get the user to first value fast; onboarding proves worth, it doesn't
  teach the whole product.

## 5. Verify & record (meet the bar)
- Hit the [2026 quality bar](reference/quality-standards.md): Core Web Vitals
  (LCP ≤2.5s / INP ≤200ms / CLS ≤0.1), WCAG 2.2 AA (keyboard, visible focus, contrast,
  ≥24px targets / 44px touch, no drag-only), responsive at 320/768/1024/1440, Browser
  chrome (themed selection/caret/scrollbar/focus — not UA defaults), and run
  its verification gate (no console errors, no axe violations, all states).
- Run the **visual convergence loop** in the browser (using `devrites-browser-proof`; deltas land in its Visual Verdict scorecard): render the
  slice's named states/viewports/input modes, open the screenshots, compare them with the
  brief + target R-ids, record material deltas, fix, and re-render until none remain. A
  detector/checklist is a floor, not the visual verdict.
- **Every interactive element has an asserting test at the right level:** each field,
  checkbox, radio, select, toggle, button, and actionable link gets a unit/component test for
  what it *does* (validation, toggle, options, enabled/disabled, handler); critical journeys
  get one E2E. Browser proof shows it renders; the asserting test proves it works. No element
  ships unverified: `standards/testing.md` "Completeness"; inventory in `test-plan.md`.
- **Append build-time refinements** to `design-brief.md` (§3); `/rite-polish` Phase 4 and
  `/rite-seal`'s frontend reviewer read it. Record runtime evidence in `browser-evidence.md`.

## Fullstack (frontend + backend in one feature)
When the feature needs both sides, follow [reference/fullstack.md](reference/fullstack.md):
define the **API/data contract first** (`devrites-api-interface`), slice **vertically**
through the layers (DB → service → API → UI) one capability at a time, apply the
engineering rules to the backend and this craft to the frontend, map every contract error
to a real UI state, and **prove both layers** (contract tests + browser proof).

## Anti-AI-slop
The banned-defaults list and the countable mechanical pre-flight live in
`rite-polish/reference/anti-ai-slop.md` (canonical owner): run both at build and polish
time — a slop pattern in the slice is a polish finding.

## Default vs departure
Preserve the existing identity (default, ~90%). Reject it only on an explicit signal (a
design doc naming *this* surface as the failure, or the user asking to rebuild). If
unsure, you're in default mode: the cost of a wrong departure is unrecoverable.
