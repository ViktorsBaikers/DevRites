# Browser proof checklist

> Applies when: UI proof: compact sweep before claiming a rendered/browser pass.

Compact sweep only. Numbers and failing cases live in
[`quality-standards.md`](../../../devrites-frontend-craft/reference/quality-standards.md)
§ Focus & states, § Responsive, and § Browser chrome — do not restate a shorter
viewport set here.

- Open the real UI (never screenshot-only); check console and network for errors.
- Before reading the builder's verdict, inventory the raw captures against the
  required routes/states/viewports and candidate identity. Open each capture: a
  wrong route, blank render, unexpected login wall, inconsistent dimensions, or
  stale candidate is missing evidence. Recapture; until resolved, record
  `cannot_verify` and block visual approval. A convincing narrative cannot repair it.
- **Evidence matrix.** Layout-bearing states (default, long content, and every
  role-applicable surface state from [`quality-standards.md`](../../../devrites-frontend-craft/reference/quality-standards.md) § Focus & states) × every
  canonical viewport; interaction states at ≥1 small and ≥1 large viewport (hover
  only with a hover-capable pointer); one keyboard traversal; a `reduce` capture when
  motion changed; every shipped theme. Each cell holds a capture, `cannot_verify`, or
  `not-needed (<reason>)`; an empty cell is `cannot_verify`. States must exist in source
  — an unreachable state's capture proves nothing.
- A capture counts only with fonts loaded, animations settled, the real route (a harness
  or story capture is labelled and never fills a route cell) and matching candidate
  identity. Hover, touch, iOS zoom or safe-area behaviour seen only in emulation is
  `unverified (device)`, never a pass.
- A finding closes only with a capture of the same route, state and viewport class on the
  corrected candidate; a failed or incomplete capture run never counts as gone.
  **Failing case:** a 320 px overflow closed by a clean 1440 px screenshot.
- Browser-default chrome is judged by § Browser chrome; reflow, zoom and bounded
  two-dimensional exceptions by § Responsive. Compare to `design-brief.md` → Visual Verdict.
- Tooling unavailable ⇒ record fallback + limitation. Backend-only changes record
  that disposition instead of capturing quietly; UI copy follows
  `devrites-frontend-craft`, long-form prose follows `devrites-prose-craft`
  ([`prose-style.md`](prose-style.md)).

Detailed skill: `devrites-browser-proof`.
