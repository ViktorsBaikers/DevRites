# Shape before code

Decide what the UI *is* before how it looks. Aesthetics follow structure; structure
follows the user's goal. Skipping this is how you get pretty UI that doesn't work.

## Answer these before writing markup
1. **User goal** — what is the user here to accomplish? One sentence.
2. **Primary action** — the single most important thing on this surface. It must be
   visually obvious. Everything else is secondary.
3. **Information hierarchy** — what must be seen first, second, third. Rank content;
   the layout expresses the ranking.
4. **States** — design all of them, not just the happy path:
   - default / populated
   - loading (initial + subsequent)
   - empty (with a clear next action — welcoming, not a dead end)
   - error (what happened + how to recover)
   - success
   - disabled / read-only / no-permission
   - long content / overflow / many items
5. **Responsive** — how it reflows from small (375) to large (1280). What collapses,
   stacks, hides, or scrolls.
6. **Accessibility** — focus order, labels, contrast, keyboard operability, semantics.
7. **Interaction model** — what's inline vs. navigated vs. (rarely) a modal; optimistic
   vs. pending; how feedback is given.

## Ask when ambiguous
If the visual direction or the UX flow isn't determined by the existing system + the
spec, ask the user (show 2–3 concrete options) before building. Guessing a flow is
expensive to undo once coded.

## Output
A short shape note in `design-brief.md`: goal, primary action, hierarchy, the state
list, responsive plan, a11y notes. This drives the build and the polish checklist.

## NEVER (shape)

- Never start coding markup without naming the **primary action** for the
  surface.
- Never design only the populated/happy state. Empty / loading / error /
  disabled / long-content are not optional.
- Never decide the visual direction by personal taste when the project has
  an existing system. Discover first (`design-references.md`).
- Never invent a new flow when a neighboring feature already has one — match
  the neighbor, then deviate only with a recorded reason.
- Never paper over an ambiguous spec with a "reasonable guess". Ask the user
  with 2 – 3 concrete options.
- Never name a state for the *technical* condition (`isLoadingTrue`) when
  the *user-facing* name (`Loading your invoices…`) tells the user what's
  happening.
