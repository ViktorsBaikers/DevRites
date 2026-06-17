# Craft — build the UI

With the shape decided and the system discovered, build the smallest correct UI for the
slice. Quality is in the details and the states, not in novelty.

## Build order
1. **Structure** — semantic markup that expresses the hierarchy from the shape note.
2. **Compose from the system** — use existing shared components and tokens. If a needed
   component doesn't exist, build it in the project's style (don't import a new library
   without asking).
3. **State coverage** — wire up every state you shaped (loading/empty/error/success/
   disabled/long-content), not just the populated happy path.
4. **Interaction** — feedback on every action; focus management; keyboard support;
   sensible defaults (Enter submits, Esc closes, etc.).
5. **Responsive** — verify the reflow at the target viewports.

## Use the system, don't fight it
- Tokens for spacing/color/type — never hard-code a value a token covers.
- Match the nearest neighbor's patterns for forms, buttons, menus, toasts.
- One icon set (the project's). Consistent sizing/alignment.

## Smallest UI for the slice
Build what *this* slice needs. Don't scaffold future screens, settings, or variations
"while you're here" — that's scope creep and it dodges its own review.

## Quality tells (the difference between fine and crafted)
- The primary action is unmistakable.
- Empty states teach the next step; error states offer recovery.
- Spacing reads as deliberate; alignment holds at every breakpoint.
- Copy is in the product's voice, specific, and short.
- No console noise; no layout shift.

## Avoid the slop
Don't reach for the banned defaults (`rite-polish/reference/anti-ai-slop.md`). If the
project genuinely uses one, follow the project — consistency wins.

## Record
Append slice build-time refinements → `design-brief.md` (the brief `devrites-ux-shape`
produced at spec; refine it, don't recreate it). Then verify in the browser
(`devrites-browser-proof`) and record evidence before claiming the UI works.

## NEVER (craft)

- Never hard-code a value a token covers (color, spacing, type, radius,
  shadow). If the token is missing, ask before adding one.
- Never import a new component library or icon set without asking — the
  project has one already.
- Never ship only the populated state. The full state set (loading / empty
  / error / success / disabled / long-content) is the minimum.
- Never animate as decoration. Motion is feedback or a focus shift, not
  jewellery.
- Never use raw `#000` / `#fff`, viewport queries for component-internal
  reflows, or raw `z-index` numbers outside the semantic scale
  (`quality-standards.md` — Numerical bar).
- Never write `console.log` or leave commented-out code in shipped UI.
- Never scaffold "future screens" — out of slice scope.
