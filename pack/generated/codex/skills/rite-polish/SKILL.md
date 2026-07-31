---
name: rite-polish
description: Polish the active feature's code and any touched UI before review. Use for finish or normalization requests; not for repository-wide refactors.
argument-hint: "[target | bolder | quieter | distill | harden | normalize-only]"
user-invocable: true
---

# $rite-polish: finish before review

Polish code for every feature. When the feature touches UI, normalize and polish the
UI as well. Complete this self-review before `$rite-review`. The code and UI phases
live in `reference/code.md` and `reference/ui.md`; read only the phase in scope.

## Operating rules

- **Functionality complete first.** Polish runs after `$rite-prove` (full
  feature proven).
- Follow the shared
  [`candidate-integrity.md`](../devrites-lib/reference/candidate-integrity.md).
  Polish owns every candidate-affecting correction and durable rollup before Review.
- Feature scope only.
- For UI, **normalize before polishing**. Do not add decoration on top of drift.
- **Root selects; wright edits.** The controlling chat assesses and reconciles, but every
  accepted source/test correction is dispatched to the sole writer,
  `devrites-slice-wright`, through
  [`agents.md`](../devrites-lib/reference/standards/agents.md). Never edit source inline or
  run two correction writers concurrently.

## Orchestration

0. **Read** `.agents/skills/devrites-lib/reference/standards/core.md` first (the always-on operating rules). The
   per-phase rule files (`coding-style.md`, `error-handling.md`, …) load on demand
   from `reference/code.md` / `reference/ui.md` when their phase runs; for UI scope also read
   `.agents/skills/devrites-lib/reference/standards/browser-proof-checklist.md`.
   Then read the explicit or active workspace's `state.md` directly.
1. **Read** `state.md`, `touched-files.md`, the current candidate digest, and the
   `git diff` for the active workspace (or `$ARGUMENTS` if a target was given).
2. **Detect UI scope:** UI is touched if the diff or `touched-files.md`
   contains any of: `.tsx`, `.jsx`, `.vue`, `.svelte`, `.html`, `.css`,
   `.scss`, `.sass`, `.less`, `.styl`, component dirs (`components/`,
   `pages/`, `routes/`, `app/`, `views/`, `screens/`), Storybook stories,
   or design-token files. When in doubt, look for visual changes that need
   verification.
3. **Always** read [`reference/code.md`](reference/code.md) and assess **Phase 1
   (code polish)**; if backend was touched, assess **Phase 2 (backend polish)** from
   the same file. Reconcile the findings, then send accepted corrections as one bounded
   wright contract.
4. **If UI scope detected** read [`reference/ui.md`](reference/ui.md), and read
   `design-brief.md` if present so the polish follows the direction and states established
   by `devrites-ux-shape` and refined by `devrites-frontend-craft`. **Read the
   `## Visual Verdict` table in `browser-evidence.md` if present:
   its `FAIL` and `PARTIAL` rows are the normalize/quality-bar worklist**: identify the root
   cause of each (a missing state, an off-token CTA, or an anti-slop hit) rather than
   hiding it with decoration. Assess **Phase 3 (normalize)** → **Phase 4 (UI polish)**,
   then send accepted UI
   corrections to the wright (which invokes the relevant craft skill). Honor argument modes:
   - `bolder | quieter | distill | harden`: passed to Phase 4 as the
     emphasis dial.
   - `normalize-only`: assess Phase 3 and stop (no Phase 4).
5. **Finish durable rollups before Review.** Apply the capability
   [`ledger`](reference/ledger.md) when requirements changed, the optional UI
   [`design memory`](reference/design-memory.md), and durable
   [`ADR promotion`](reference/adr-promotion.md). Add every changed project path
   to the candidate manifest; none of these writes waits for Ship.
6. **Re-prove and close.** After all accepted code/UI corrections and rollups,
   run `devrites-engine check candidate <slug>`. Any digest change requires
   affected real re-proof using the approved commands, fresh proof-runner
   validation, refreshed evidence/browser bindings, and an updated candidate
   manifest. Record a **`Re-verification:`** line in `polish-report.md`. Close
   the candidate for Review only after these checks are green.
7. **Aggregate output:** both phases append to the single `polish-report.md`.

## Refinement modes

Pass the requested UI direction to Phase 4. Modes do not bypass normalization or the
quality bar; they apply after the system is aligned. See `reference/ui.md`.

> **Mid-flight discipline.** When tempted to polish UI without normalize, cite
> clean lint as proof of quality, skip Phase 2 on a backend diff, or delete a
> Chesterton's Fence: see [anti-patterns](reference/anti-patterns.md).

## Output → `polish-report.md`

Write the detailed report to `polish-report.md`.
