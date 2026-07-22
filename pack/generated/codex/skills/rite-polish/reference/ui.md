# Normalize + UI quality (Phase 3 + Phase 4)

Loaded from `$rite-polish` only when the diff touches UI. Two phases, strict
order: normalize to the design system (Phase 3), then the ship-quality detail
pass (Phase 4). Normalize removes drift and aligns to the system; polish sweats
the details. Skipping normalize means polishing drift.

## Operating rules

- **NEVER polish without normalizing first:** decoration on drift is banned.
  Phase 3 runs before Phase 4, always.
- Browser evidence required for UI polish when a browser can run
  ([browser-polish-evidence.md](browser-polish-evidence.md)).
- Don't invent a new design system. Ask when design-system principles are
  ambiguous. Spec Drift Guard applies (a design-system contradiction is drift).

## Why this order

A beautiful component that ignores the project's tokens, components, and flow
shape is *worse* than a plain one that fits. It adds a second system the team
must maintain. Alignment is the precondition for polish, not an optional first
step. **Fixing the symptom without naming the cause is how drift compounds:**
patching a colour without adding the token bakes the next drift in; swapping a
class without moving to the shared component leaves the next caller to redo the
work.

## Phase 3: Normalize

Goal: this feature looks and behaves like it belongs in *this* product.

1. **Discover the design system**
   ([design-system-discovery.md](design-system-discovery.md)): docs, tokens,
   shared components, spacing/type scales, color roles, icon set, interaction
   & form patterns, neighboring screens.
2. **Identify drift by root cause**, classifying each into one of **three
   buckets**: the bucket dictates the fix:

   | Bucket | What it is | Fix |
   |---|---|---|
   | **Token gap** | A value is hard-coded where a token should carry it (colour, spacing, type size, radius, shadow). | Use the existing token; if a needed slot doesn't exist, propose adding it to the system: don't invent one inline. |
   | **Component miss** | A one-off implementation exists where a shared component already covers the case (button, input, dialog, menu, table row). | Swap to the shared component. Don't fork it into the feature folder. |
   | **Flow / IA misalignment** | The feature's flow shape, naming, or mental model doesn't match neighbouring features (modal where the project goes inline; save-on-blur where the project uses explicit submit; new vocabulary for an existing concept). | Reshape the flow / rename to the project's vocabulary. The most expensive bucket, and the most damaging if skipped. |

   Sub-cases that map back to the buckets above: generic-AI-template styling →
   **Token gap** (hard-coded styling that should ride a token) or **Component
   miss** (rebuilt instead of reused); orphaned styles or components this
   feature created → **Component miss** (consolidate or remove); inconsistent
   IA / naming → **Flow / IA misalignment**.
3. **Fix root causes, not symptoms:** use existing tokens, swap to shared
   components, align flow shape with neighbors, consolidate duplication, remove
   obsolete styles created by this feature. Ask the user when a design-system
   principle is genuinely ambiguous: don't guess the system's intent.

## Phase 4: UI polish

Only after functionality is complete **and** Phase 3 is done. Systematic detail
pass against the UI quality bar below: zoom in, squint, use the thing. Little
things add up.

Meet the **2026 quality bar**. CWV (LCP ≤ 2.5s / INP ≤ 200ms / CLS ≤ 0.1) and
WCAG 2.2 AA. Avoid [anti-ai-slop.md](anti-ai-slop.md). If the spec saved design
references in `references/`, **match them**.

Run the production-hardening sweep before declaring polish done (extreme
inputs, errors, network, i18n, permission states) per
[harden-checklist.md](harden-checklist.md). Polish that only works on the happy
path isn't polish.

### UI quality bar (checklist)

- [ ] **Aligned to the design system:** drift named and resolved by root cause (Phase 3)
- [ ] **IA & flow** match neighboring features
- [ ] **Visual hierarchy:** the primary action is obviously primary
- [ ] **Spacing** uses design tokens consistently; rhythm reads as deliberate
- [ ] **Alignment** perfect at every breakpoint
- [ ] **Typography** hierarchy consistent; line length sane (65-75ch prose)
- [ ] **Color & contrast**: semantic roles; text ≥ 4.5:1 (≥ 3:1 large/UI), WCAG 2.2 AA
- [ ] **All interaction states**: hover, active, focus, disabled, selected
- [ ] **Focus states visible** and **keyboard navigation** works
- [ ] **Loading** states clear; **empty** states welcoming with a next action
- [ ] **Error** states helpful (what happened + how to recover); **success** states clear
- [ ] **Forms** properly labeled, validated, with inline errors
- [ ] **Copy & terminology** consistent with the product's vocabulary
- [ ] **Icons** consistent set, properly sized and aligned
- [ ] **Responsive** behavior correct across target viewports
- [ ] **Target size** ≥ 24×24px (WCAG 2.2 AA); ≥ 44×44px for primary touch; **no drag-only** actions
- [ ] **Motion** smooth (~60fps), purposeful; **reduced-motion** respected
- [ ] **Performance basics:** no obvious jank, oversized images, or blocking work
- [ ] **No console errors or warnings**
- [ ] **No layout shift** on load
- [ ] Works in the project's **supported browsers**
- [ ] **Code quality** in touched UI files: no TODOs, console.logs, commented-out code

### Measurable bar (2026)

Polish verifies against the frontend quality standards
(`devrites-frontend-craft/reference/quality-standards.md`), measured not assumed:
- Core Web Vitals (p75): **LCP ≤ 2.5 s · INP ≤ 200 ms · CLS ≤ 0.1**.
- WCAG 2.2 AA: semantic HTML, keyboard-operable, visible focus ≥ 3:1, no drag-only.
- Responsive at **320 / 768 / 1024 / 1440**; survives 200% text zoom; no horizontal scroll.
- No console errors/warnings; no axe violations; `prefers-reduced-motion` honored.
Cite the evidence in `browser-evidence.md`.

### Triage

If it ships in 30 minutes, fix what users will notice (hierarchy,
states, broken layout, bad copy) before micro-spacing. Don't introduce bugs
while polishing: re-prove after changes.

### Refinement modes (optional dials within Phase 4)

| Mode | When | What it means |
|---|---|---|
| **bolder** | Surface reads safe / generic / forgettable. | Stronger hierarchy + weight contrast, one decisive accent, more committed scale. Reject generic-bold reflexes: gradients, glassmorphism, neon. |
| **quieter** | Surface reads loud / overstimulating. | Reduce chroma, increase whitespace, flatten elevation, soften motion. Personality stays; intensity drops. |
| **distill** | Too many elements / colours / sizes competing. | Remove anything that doesn't earn its place: redundant copy, decorative shadows/borders. One primary action per screen. |
| **harden** | Surface only works on the happy path. | Walk [harden-checklist.md](harden-checklist.md) explicitly. Design every error / empty / partial / offline state. |

Modes are advisory. Default Phase 4 covers all four already; the mode just
shifts the emphasis of the same pass when the brief asks for it.

## NEVER

- Polish before it's functionally complete.
- Polish without aligning first (decoration on drift).
- Guess at design-system principles instead of asking.
- Introduce bugs while polishing (re-prove after changes).
- Invent a new design system or add a component/icon library without asking.

## Output → appends to `polish-report.md`

```
Phase 3 (normalize): drift found → root-cause fixes
Phase 4 (UI polish): quality-bar deltas
Browser evidence: <summary>
```
