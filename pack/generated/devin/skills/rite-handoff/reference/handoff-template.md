# Handoff doc template

Loaded on demand by `/rite-handoff`. Write the filled-in template to
`.devrites/work/<slug>/handoff.md` (or to a `$TMPDIR` file when no active feature).

```
# Handoff — <feature slug | "no active feature"> — <ISO date>

## Resume
Current objective: <one sentence>
Last completed slice: <SLICE-### | none>
Blockers: <none | q-YYYY-MM-DD-NNN / released Q-### / DRIFT-###>
Tree at write: HEAD <sha | no git>; dirty paths <n>
In flight: <none | wave <id> role <r> handle <h> | wright SLICE-### handle <h> | claim <id> <paths> | action <name>>; each `terminal: yes|unknown`

## Read next
1. `.devrites/work/<slug>/README.md` — workspace map.
2. `.devrites/work/<slug>/state.md` — current cursor.
3. `.devrites/work/<slug>/traceability.md` — acceptance coverage.
4. <phase-specific source artifact; link, do not duplicate>.

## Next action
<one command or action, no option set; any `terminal: unknown` row makes it reconcile:
`devrites-engine dispatch <slug> status`, `devrites-engine parallel status --root <repo> --slug <slug>`,
`devrites-engine claim list --all` — never a writer command>

## What changed this turn
- <short event; link to source artifact instead of copying full content>

## External references (chat-only)
- <URL / Figma / screenshot path / video timestamp>   (carried forward if still valid)

## Live assumptions (not yet in assumptions.md)
- ...   (this section should be near-empty after the sync step)

## Retired from the previous handoff
- <none | reference or assumption — why it no longer holds>

## Synced this turn
- decisions.md: <N entries appended>
- questions.md: <N appended>
- assumptions.md: <N appended>
- drift.md: <updated? yes/no>
- touched-files.md: <N appended>

## How to resume
1. Read this file.
2. Read `.devrites/work/<slug>/state.md` for the workspace cursor.
3. Tree check: compare `git rev-parse HEAD` and `git status --short` with `Tree at write`
   and `touched-files.md`. Other HEAD, other dirty count, or a path missing from
   `touched-files.md` → record it in `drift.md` or `questions.md` before any writer command.
4. Any `In flight` row `terminal: unknown` → run the reconcile commands first; never dispatch
   a second writer onto its paths.
5. Run `/rite-status` for the current phase / next action / open drift.
6. Continue with the `Next action` above.
```
