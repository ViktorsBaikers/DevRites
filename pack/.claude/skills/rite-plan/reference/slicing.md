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

## First slice
Make slice 1 the **thinnest useful end-to-end path** — it flushes out integration and
convention surprises early, while changes are cheap.

## Slice independence
Order by dependency, but minimize coupling. A slice that needs three other slices first
is a smell — look for a thinner cut that stands alone.
