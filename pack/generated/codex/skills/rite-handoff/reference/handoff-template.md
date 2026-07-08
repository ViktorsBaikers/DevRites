# Handoff doc template

Loaded on demand by `$rite-handoff`. Write the filled-in template to
`.devrites/work/<slug>/handoff.md` (or to a `$TMPDIR` file when no active feature).

```
# Handoff — <feature slug | "no active feature"> — <ISO date>

## Resume
Current objective: <one sentence>
Last completed slice: <SLICE-### | none>
Blockers: <none | Q-### / DRIFT-###>

## Read next
1. `.devrites/work/<slug>/README.md` — workspace map.
2. `.devrites/work/<slug>/state.md` — current cursor.
3. `.devrites/work/<slug>/traceability.md` — acceptance coverage.
4. <phase-specific source artifact; link, do not duplicate>.

## Next action
<one command or action, no option set>

## What changed this turn
- <short event; link to source artifact instead of copying full content>

## External references (chat-only)
- <URL / Figma / screenshot path / video timestamp>

## Live assumptions (not yet in assumptions.md)
- ...   (this section should be near-empty after the sync step)

## Synced this turn
- decisions.md: <N entries appended>
- questions.md: <N appended>
- assumptions.md: <N appended>
- drift.md: <updated? yes/no>
- touched-files.md: <N appended>

## How to resume
1. Read this file.
2. Read `.devrites/work/<slug>/state.md` for the workspace cursor.
3. Run `$rite-status` for the current phase / next action / open drift.
4. Continue with the `Next action` above.
```
