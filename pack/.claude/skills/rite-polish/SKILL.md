---
name: rite-polish
description: Polish the active feature before review — code polish always; UI normalize + ship-quality pass when UI is touched. Use when the user says "polish this", "finish before review", "ship-quality", "normalize the UI", "bolder", "quieter", "distill", "harden". Not for repo-wide refactors or pre-`/rite-prove` polish.
argument-hint: "[target | bolder | quieter | distill | harden | normalize-only]"
user-invocable: true
---

# /rite-polish — finish before review

The "finish" phase. **Always** code-polishes; **and** runs UI normalize +
design polish when the feature touches UI. Self-review the work to ship
quality *before* handing it to `/rite-review`. The two-phase split (code,
UI) lives in `reference/code.md` and `reference/ui.md` — read each on demand,
don't load both up front.

## Operating rules

- **Functionality complete first.** Polish runs after `/rite-prove` (full
  feature proven).
- Feature scope only.
- For UI: **NEVER polish without normalizing first** — decoration on drift is
  banned.

## Orchestration

1. **Read** `state.md`, `touched-files.md`, and the `git diff` for the active
   workspace (or `$ARGUMENTS` if a target was given).
2. **Detect UI scope** — UI is touched if the diff or `touched-files.md`
   contains any of: `.tsx`, `.jsx`, `.vue`, `.svelte`, `.html`, `.css`,
   `.scss`, `.sass`, `.less`, `.styl`, component dirs (`components/`,
   `pages/`, `routes/`, `app/`, `views/`, `screens/`), Storybook stories,
   or design-token files. When in doubt, look for visual changes that need
   verification.
3. **Always** read [`reference/code.md`](reference/code.md) and run **Phase 1
   (code polish)**; if backend was touched, continue into **Phase 2 (backend
   polish)** from the same file.
4. **If UI scope detected** read [`reference/ui.md`](reference/ui.md) and run
   **Phase 3 (normalize)** → **Phase 4 (UI polish)**. Honor argument modes:
   - `bolder | quieter | distill | harden` — passed to Phase 4 as the
     emphasis dial.
   - `normalize-only` — run Phase 3 and stop (no Phase 4).
5. **Aggregate output** — both phases append to the single `polish-report.md`.

## Refinement modes

When the user (or your own assessment) names a direction the UI should move,
pass it through to Phase 4. Modes don't bypass normalize + quality-bar work —
they shape the polish *after* the system is aligned. See `reference/ui.md` for
the mode table.

> **Mid-flight discipline.** When tempted to polish UI without normalize, cite
> clean lint as proof of quality, skip Phase 2 on a backend diff, or delete a
> Chesterton's Fence — see [anti-patterns](reference/anti-patterns.md).

## Output → `polish-report.md`

```
Target: <slug | path/route/component>
Phase 1 (code polish): findings → fixes (technique + why behavior preserved)
Phase 2 (backend polish): error/log/data/API/cleanup fixes | n/a (no backend)
Phase 3 (normalize):    drift found → root-cause fixes     | n/a (no UI)
Phase 4 (UI polish):    quality-bar deltas                 | n/a (no UI)
Browser evidence: <summary | n/a>
Open design questions asked: <none | list>
Next: /rite-review
↻ Hygiene: /clear between polish targets and before /rite-review (polish-report.md + browser-evidence.md captured). See rules/context-hygiene.md.
```
