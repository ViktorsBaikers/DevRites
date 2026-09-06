# Browser proof checklist

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
- Interactive slices capture each relevant canonical state and the canonical
  viewport set. Omission needs a one-line `not-needed` reason; states must exist
  in source — an unreachable state's capture proves nothing.
- Browser-default chrome is judged by § Browser chrome; reflow, zoom and bounded
  two-dimensional exceptions by § Responsive. Compare to `design-brief.md` → Visual Verdict.
- Tooling unavailable ⇒ record fallback + limitation. Backend-only changes record
  that disposition instead of capturing quietly; UI copy follows
  `devrites-frontend-craft`, long-form prose follows `devrites-prose-craft`
  ([`prose-style.md`](prose-style.md)).

Detailed skill: `devrites-browser-proof`.
