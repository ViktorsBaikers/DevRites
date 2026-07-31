# Issue-driven rites

Local issue files are the source of truth; external trackers are references only.

## Flow

1. Create `.scratch/<slug>/PRD.md` when product detail is needed and one or more numbered implementation issues under `.scratch/<slug>/issues/<NN>-<issue-slug>.md`, starting at `01`.
2. Start `/rite-spec <slug>` from the relevant PRD/issues; copy lifecycle decisions into `.devrites/work/<slug>/`, while tracker status/comments remain in the local issue files.
3. Run the normal lifecycle: `/rite-clarify` → optional `/rite-temper` → `/rite-define` → `/rite-vet` → `/rite-build` → `/rite-prove` → `/rite-polish` → `/rite-review` → `/rite-seal` → `/rite-ship`.
4. After a recorded type-GO, mark completed local issues `Status: resolved` and append the ship evidence; include `Closes #NNN` only when an external issue reference exists.
5. Follow-ups become the next numbered local issue under `.scratch/<slug>/issues/` (or a new feature directory when scope differs); external auto-posting is out of scope.
