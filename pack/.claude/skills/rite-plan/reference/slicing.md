# Slicing

DevRites builds in **thin vertical slices**: each slice cuts through every layer it
needs (data → logic → API → UI) and leaves the system in a working, testable state.

## Vertical, not horizontal
Horizontal ("build all models, then all controllers, then all views") delays working
software and hides integration risk until the end. Vertical delivers a usable path each
slice:
```
Slice 1: Create a task   (DB + API + minimal UI)   → user can create + test passes
Slice 2: List tasks      (query + API + UI)         → user can see them
Slice 3: Edit a task     (update + API + UI)        → user can modify
Slice 4: Delete a task   (delete + API + UI + confirm) → full CRUD
```

## Sizing a slice
A slice is the right size when it:
- delivers one observable capability end-to-end;
- can be built and proven in a single `/rite-build` → `/rite-prove` cycle;
- has acceptance criteria you can verify with evidence;
- can be rolled back on its own.

Too big if: it touches many unrelated files, has multiple "and"s in its goal, or you
can't name its single observable outcome. → reslice.

## How many slices? — derive, don't dictate
The slice **count is an output, not an input.** It falls out of the work: one slice per
independently-shippable vertical increment that satisfies one (or a tight group of)
acceptance criteria and passes the sizing test above. A 2-criterion feature is 1–2 slices;
a 12-criterion feature is however many thin end-to-end cuts those 12 need.

- **Never slice to a target number.** Don't pad a small feature into "5 slices" or
  compress a large one into "3" because a figure was named.
- **A user-supplied count is a hint at most.** If the user says "do it in 4", slice the
  work logically first; if your honest decomposition differs, present the logical count
  and *why*, then let them redirect — don't silently force their number.
- **Re-size by the rule, not the tally** (`/rite-plan reslice`): split a slice because it
  failed the sizing test, not to hit a count.

Not to be confused with `.devrites/AFK` `max_slices` — that's an AFK *iteration budget*
capping how many slices the unattended loop builds before it stops, **not** the plan's
decomposition. The plan always holds exactly the slices the work needs; `max_slices` only
limits how many run unattended.

## First slice
Make slice 1 the **thinnest useful end-to-end path** — it flushes out integration and
convention surprises early, while changes are cheap.

## Slice independence
Order by dependency, but minimize coupling. A slice that needs three other slices first
is a smell — look for a thinner cut that stands alone.
