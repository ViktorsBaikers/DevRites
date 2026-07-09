# Issue-driven rites

Local issue files are the source of truth; external trackers are references only.

## Flow

1. Create `.scratch/<slug>/issue.md` for the problem and optional `.scratch/<slug>/prd.md` for product detail.
2. Start `/rite-spec <slug>` from those files; copy decisions into `.devrites/work/<slug>/`, not back to the tracker.
3. Run the normal lifecycle: `/rite-define` → `/rite-vet` → `/rite-build` → `/rite-prove` → `/rite-review` → `/rite-seal`.
4. `/rite-ship` may include `Closes #NNN` only after a recorded type-GO.
5. Follow-ups go to a new local `.scratch/<next-slug>/issue.md` first; external auto-posting is out of scope.
