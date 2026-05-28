# Normalize + UI quality (Phase 3 + Phase 4)

Loaded from `/rite-polish` only when the diff touches UI. Two sub-phases:
normalize to the design system (Phase 3), then ship-quality detail pass
(Phase 4).

## Operating rules

- **NEVER polish without normalizing first** — decoration on drift is banned.
- Browser evidence required for UI polish when a browser can run
  ([browser-polish-evidence.md](browser-polish-evidence.md)).
- Don't invent a new design system. Ask when design-system principles are
  ambiguous. Spec Drift Guard applies (a design-system contradiction is drift).

## Phase 3 — Normalize

See [normalize-polish.md](normalize-polish.md).

1. **Discover the design system**
   ([design-system-discovery.md](design-system-discovery.md)): docs, tokens,
   shared components, spacing/type scales, color roles, icon set, interaction
   & form patterns, neighboring screens.
2. **Identify drift by root cause** — missing token, hard-coded value where a
   token exists, one-off where a shared component exists, conceptual
   misalignment with existing flows, inconsistent IA/naming, generic-AI-template
   styling, orphaned styles/components.
3. **Fix the root cause, not the symptom** — use existing tokens, swap to
   shared components, align flow shape with neighbors, consolidate
   duplication, remove obsolete styles created by this feature.

## Phase 4 — UI polish

See [ui-quality-bar.md](ui-quality-bar.md).

Systematic detail pass against the UI quality bar: IA/flow, visual hierarchy,
spacing/alignment, typography, color & contrast, all interaction states, focus
& keyboard nav, loading/empty/error/success/disabled states, forms &
validation, copy & terminology, icons & imagery, responsive behavior, touch
targets, motion (+ reduced motion), performance basics, console
errors/warnings, layout shift, and code quality in touched UI files.

Meet the **2026 quality bar** — CWV (LCP ≤ 2.5s / INP ≤ 200ms / CLS ≤ 0.1) and
WCAG 2.2 AA. Avoid [anti-ai-slop.md](anti-ai-slop.md). If the spec saved
design references in `references/`, **match them**.

Run the production-hardening sweep before declaring polish done — extreme
inputs, errors, network, i18n, permission states — per
[harden-checklist.md](harden-checklist.md). Polish that only works on the
happy path isn't polish.

### Refinement modes (optional dials within Phase 4)

| Mode | When | What it means |
|---|---|---|
| **bolder** | Surface reads safe / generic / forgettable. | Stronger hierarchy + weight contrast, one decisive accent, more committed scale. Reject generic-bold reflexes — gradients, glassmorphism, neon. |
| **quieter** | Surface reads loud / overstimulating. | Reduce chroma, increase whitespace, flatten elevation, soften motion. Personality stays; intensity drops. |
| **distill** | Too many elements / colours / sizes competing. | Remove anything that doesn't earn its place: redundant copy, decorative shadows/borders. One primary action per screen. |
| **harden** | Surface only works on the happy path. | Walk [harden-checklist.md](harden-checklist.md) explicitly. Design every error / empty / partial / offline state. |

Modes are advisory. Default Phase 4 covers all four already; the mode just
shifts the emphasis of the same pass when the brief asks for it.

## Output → appends to `polish-report.md`

```
Phase 3 (normalize): drift found → root-cause fixes
Phase 4 (UI polish): quality-bar deltas
Browser evidence: <summary>
```
