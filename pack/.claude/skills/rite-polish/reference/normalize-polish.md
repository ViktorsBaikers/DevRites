# Normalize → Polish

Two phases, strict order. Normalize removes drift and aligns to the system; polish
sweats the details. Skipping normalize means polishing drift.

## Why this order
A beautiful component that ignores the project's tokens, components, and flow shape is
*worse* than a plain one that fits — it adds a second system the team must maintain.
Alignment is the precondition for polish, not an optional first step.

## Phase 1 — Normalize
Goal: this feature looks and behaves like it belongs in *this* product.
1. Discover the system (see `design-system-discovery.md`).
2. Diff your feature against it; classify each drift into one of **three
   buckets** — the bucket dictates the fix:

   | Bucket | What it is | Fix |
   |---|---|---|
   | **Token gap** | A value is hard-coded where a token should carry it (colour, spacing, type size, radius, shadow). | Use the existing token; if a needed slot doesn't exist, propose adding it to the system — don't invent one inline. |
   | **Component miss** | A one-off implementation exists where a shared component already covers the case (button, input, dialog, menu, table row). | Swap to the shared component. Don't fork it into the feature folder. |
   | **Flow / IA misalignment** | The feature's flow shape, naming, or mental model doesn't match neighbouring features (modal where the project goes inline; save-on-blur where the project uses explicit submit; new vocabulary for an existing concept). | Reshape the flow / rename to the project's vocabulary. The most expensive bucket — and the most damaging if skipped. |

   Sub-cases that map back to the buckets above: generic-AI-template
   styling → **Token gap** (hard-coded styling that should ride a token)
   or **Component miss** (rebuilt instead of reused); orphaned styles or
   components this feature created → **Component miss** (consolidate or
   remove); inconsistent IA / naming → **Flow / IA misalignment**.

3. Fix root causes, not symptoms. Ask the user when a design-system
   principle is genuinely ambiguous — don't guess the system's intent.

> The bucket discipline matters because **fixing the symptom without
> naming the cause is how drift compounds.** Patching a colour without
> adding the token bakes the next drift in; swapping a class without
> moving to the shared component leaves the next caller to redo the work.

## Phase 2 — Polish
Only after functionality is complete **and** Phase 1 is done. Walk `ui-quality-bar.md`
systematically. Zoom in: alignment, spacing rhythm, state coverage, copy. Verify in the
browser. Triage if it ships in 30 minutes — don't spend hours gilding.

## NEVER
- Polish before it's functionally complete.
- Polish without aligning first (decoration on drift).
- Guess at design-system principles instead of asking.
- Introduce bugs while polishing (re-prove after changes).
- Invent a new design system or add a component/icon library without asking.
