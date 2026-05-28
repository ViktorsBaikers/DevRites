# UI quality bar

The Phase-2 polish checklist. Go through it systematically — zoom in, squint, use the
thing. Little things add up. Don't polish before functionality is complete.

- [ ] **Aligned to the design system** — drift named and resolved by root cause (Phase 1)
- [ ] **IA & flow** match neighboring features
- [ ] **Visual hierarchy** — the primary action is obviously primary
- [ ] **Spacing** uses design tokens consistently; rhythm reads as deliberate
- [ ] **Alignment** perfect at every breakpoint
- [ ] **Typography** hierarchy consistent; line length sane (65–75ch prose)
- [ ] **Color & contrast** — semantic roles; text ≥ 4.5:1 (≥ 3:1 large/UI), WCAG 2.2 AA
- [ ] **All interaction states** — hover, active, focus, disabled, selected
- [ ] **Focus states visible** and **keyboard navigation** works
- [ ] **Loading** states clear; **empty** states welcoming with a next action
- [ ] **Error** states helpful (what happened + how to recover); **success** states clear
- [ ] **Forms** properly labeled, validated, with inline errors
- [ ] **Copy & terminology** consistent with the product's vocabulary
- [ ] **Icons** consistent set, properly sized and aligned
- [ ] **Responsive** behavior correct across target viewports
- [ ] **Target size** ≥ 24×24px (WCAG 2.2 AA); ≥ 44×44px for primary touch; **no drag-only** actions
- [ ] **Motion** smooth (~60fps), purposeful; **reduced-motion** respected
- [ ] **Performance basics** — no obvious jank, oversized images, or blocking work
- [ ] **No console errors or warnings**
- [ ] **No layout shift** on load
- [ ] Works in the project's **supported browsers**
- [ ] **Code quality** in touched UI files — no TODOs, console.logs, commented-out code

## Measurable bar (2026)
Polish verifies against the frontend quality standards
(`devrites-frontend-craft/reference/quality-standards.md`), measured not assumed:
- Core Web Vitals (p75): **LCP ≤ 2.5 s · INP ≤ 200 ms · CLS ≤ 0.1**.
- WCAG 2.2 AA: semantic HTML, keyboard-operable, visible focus ≥ 3:1, no drag-only.
- Responsive at **320 / 768 / 1024 / 1440**; survives 200% text zoom; no horizontal scroll.
- No console errors/warnings; no axe violations; `prefers-reduced-motion` honored.
Cite the evidence in `browser-evidence.md`.

## Triage
If it ships in 30 minutes, fix what users will actually notice (hierarchy, states,
broken layout, bad copy) before micro-spacing. Don't introduce bugs while polishing —
re-prove after changes.
