---
name: rite-polish
description: Polish the active feature before review — code polish always; UI normalize + ship-quality pass when UI is touched. Use when the user says "polish this", "finish before review", "normalize the UI", or "make it bolder/quieter". Not for repo-wide refactors or pre-`$rite-prove` polish.
argument-hint: "[target | bolder | quieter | distill | harden | normalize-only]"
user-invocable: true
---

## Codex compatibility

This is the Codex mirror of a DevRites skill. In Codex:

- Load DevRites engineering standards from `.agents/skills/devrites-lib/reference/standards/`. Read `.agents/skills/devrites-lib/reference/standards/core.md` before workflow work, then load the other `.agents/skills/devrites-lib/reference/standards/*.md` files exactly when this skill asks for them.
- Use the installed `devrites-engine` binary as the canonical runtime helper surface for orientation, gates, and state mutation.
- When this skill asks for a DevRites specialist or writer agent, **explicitly** spawn the matching Codex custom agent from `.codex/agents/devrites-*.toml` through Codex subagents (`spawn_agent`), then wait for its result and reconcile it as the skill instructs. Do not do the review inline just because the instruction to spawn is embedded here — Codex under-fires embedded spawn/skill instructions (openai/codex #23496), so treat the spawn as required, not optional.
- The independence of a fresh-context subagent is the point. If Codex genuinely cannot spawn subagents in the current surface, run the documented inline fallback and **label the result an inline fallback, not an independent review** — an inline pass shares the calling context and is weaker evidence.
- Codex project hooks are installed in `.codex/hooks.json`. Review and trust them with `/hooks` before relying on hook enforcement.


# $rite-polish — finish before review

The "finish" phase. **Always** code-polishes; **and** runs UI normalize +
design polish when the feature touches UI. Self-review the work to ship
quality *before* handing it to `$rite-review`. The two-phase split (code,
UI) lives in `reference/code.md` and `reference/ui.md` — read each on demand,
don't load both up front.

## Operating rules

- **Functionality complete first.** Polish runs after `$rite-prove` (full
  feature proven).
- Feature scope only.
- For UI: **NEVER polish without normalizing first** — decoration on drift is
  banned.

## Orchestration

0. **Read** `.agents/skills/devrites-lib/reference/standards/core.md` first (the always-on operating rules). The
   per-phase rule files (`coding-style.md`, `error-handling.md`, …) load on demand
   from `reference/code.md` / `reference/ui.md` when their phase runs.
   Then **run the shared orientation preamble** — it prints `state.md`, the artifacts present,
   the run mode (HITL/AFK), and the open-question tally by gate, so you orient deterministically
   instead of re-deriving state from raw Markdown:
   ```bash
   devrites-engine preamble
   ```
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
4. **If UI scope detected** read [`reference/ui.md`](reference/ui.md), and read
   `design-brief.md` if present (the UX/UI contract `devrites-ux-shape` shaped at spec and
   `devrites-frontend-craft` refined while building) so the polish honors the agreed
   direction + states. **Read the `## Visual Verdict` table in `browser-evidence.md` if present:
   its `FAIL` and `PARTIAL` rows are the normalize/quality-bar worklist** — fix the root cause of
   each (a missing state, an off-token CTA, an anti-slop hit), don't decorate around it. Then run
   **Phase 3 (normalize)** → **Phase 4 (UI polish)**. Honor argument modes:
   - `bolder | quieter | distill | harden` — passed to Phase 4 as the
     emphasis dial.
   - `normalize-only` — run Phase 3 and stop (no Phase 4).
5. **Re-verify after any code edit** — polish edits code, so the proof from
   `$rite-prove` no longer post-dates it. Run the relevant fast checks (the
   targeted tests for the touched files + typecheck/lint; browser re-check if UI
   changed) and record a **`Re-verification:`** line in `polish-report.md`. A
   polish that changed code without a green re-verification isn't finished.
6. **Aggregate output** — both phases append to the single `polish-report.md`.

## Refinement modes

When the user (or your own assessment) names a direction the UI should move,
pass it through to Phase 4. Modes don't bypass normalize + quality-bar work —
they shape the polish *after* the system is aligned. See `reference/ui.md` for
the mode table.

> **Mid-flight discipline.** When tempted to polish UI without normalize, cite
> clean lint as proof of quality, skip Phase 2 on a backend diff, or delete a
> Chesterton's Fence — see [anti-patterns](reference/anti-patterns.md).

## Output → `polish-report.md`

Write the detailed report to `polish-report.md`. In chat, run `devrites-engine progress` first,
then use the shared completion reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md)).
Default success shape:
```
Done: polished <slug | target>; code/backend/UI phases <ran|n/a>.
Changed: polish-report.md, <files touched | workspace only>, browser-evidence.md <updated|n/a>
Evidence: re-verification <cmd -> pass | n/a>; browser <summary | n/a>
Open: <none | design questions | re-prove needed before review>
Next: $rite-review
Record: .devrites/work/<slug>/polish-report.md
↻ Hygiene: /clear before $rite-review
```
