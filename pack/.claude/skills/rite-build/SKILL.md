---
name: rite-build
description: Implement exactly ONE vertical slice of the active feature, then stop with evidence — TDD, doubt loop, frontend craft, Spec Drift Guard. Use when the user says "build the next slice", "implement slice N", "continue", "code this slice". Not for bug fixes, prototypes, refactors outside scope, or two slices in a row.
argument-hint: "[slice number or name]"
user-invocable: true
---

# /rite-build — one verified slice

Build the next single slice, leave it working and proven, then **stop**. **Read the
active workspace first**; if none, tell the user to run `/rite-spec <feature>`.

## Rules consulted (read on demand from `.claude/rules/`)
DevRites skills Read `.claude/rules/core.md` as their first step (workflow step 0); the
other rule files load on demand. Pull these via the `Read` tool when the slice work needs
them:
- `coding-style.md` — naming, function shape, guard clauses, comments, reuse-first.
- `error-handling.md` — fail fast, no silent catches, fail closed.
- `testing.md` — pyramid, behaviour over implementation, see-it-fail-first.
- `patterns.md` — composition over inheritance, avoid premature abstraction.
- `security.md` — when the slice touches user input, auth, data, or external integrations.

## Operating rules
- **One slice at a time. DO NOT** start the next slice without the user asking.
- Evidence over confidence. Prefer existing conventions. Feature scope only — no
  drive-by refactors.
- Surface material assumptions; ask before adding dependencies or a second design
  system. The [Spec Drift Guard](reference/spec-drift-guard.md) is active throughout.
- **Avoid AI slop while writing** (canonical list in
  `rite-polish/reference/anti-ai-slop.md`). Prevent at the source: no over-defensive
  null/length checks, no blanket `catch`es that swallow errors, no useless wrappers, no
  over-engineered abstractions before two real callers, no generic AI naming, no
  tutorial-style comments, **don't go beyond the spec**, **reuse before you write**
  (`devrites-frontend-craft/reference/reuse-first.md`). Write the code the *project*
  would write, in its idiom. Polish catches what slips; build prevents.

## Workflow ([one-slice-cycle](reference/one-slice-cycle.md))
0. **Rules + AFK + readiness check.** Read `.claude/rules/core.md` first. Then read
   `state.md`. If `Status == awaiting_human` → **STOP**, tell the user to run
   `/rite-resolve <qid> "<answer>"`. If `state.md` has no `Plan approved: <iso>` field
   → **STOP**, tell the user the plan isn't approved yet (`/rite-define` writes it when
   the human confirms). If `.devrites/AFK` is present, re-derive the remaining AFK budget
   from `state.md`'s `AFK slices remaining: <n>` field (initialized from `.devrites/AFK`
   `max_slices` on the first AFK build); if it is `0` → **STOP** (forced HITL stop; raise
   the count in `state.md` or remove the sentinel to continue). See
   [`reference/afk-discipline.md`](reference/afk-discipline.md).
1. Read `spec.md`, `plan.md`, `tasks.md`, `state.md`, `assumptions.md`, `drift.md`,
   `questions.md`.
   If a **blocking `[NEEDS CLARIFICATION]`** remains or the spec/plan readiness gates
   don't pass, stop → `/rite-spec` (to resolve) or `/rite-plan` (to repair). Don't build
   on an unresolved spec.
2. Select the next pending slice (or the one in `$ARGUMENTS`). **Restate its goal,
   acceptance criteria, and scope boundary** in one short block. Confirm it's still the
   right next slice. Write the slice's `Mode` to `state.md` as `Slice mode: <HITL|AFK>` on
   **every** selection (not only on the HITL pause path); `/rite-resolve` clears or updates
   it on resume.
2a. **HITL gate (pre-action pause).** Read the slice's `Mode`. If `HITL` → render the
    checkpoint per [`reference/checkpoint-protocol.md`](reference/checkpoint-protocol.md):
    append a `questions.md` entry with the slice's `Checkpoint:` + `Gate:` + `SLA:`,
    write the `Awaiting human` block to `state.md`, set `Status: awaiting_human`, run
    the `notify:` hook if `.devrites/AFK` defines one, then **STOP**. Resume happens
    when the user runs `/rite-resolve <qid> "<answer>"`.
3. Load only the files this slice touches (use `touched-files.md` + codebase search). Prefer a code-intelligence index — `codegraph` (`.codegraph/` / `codegraph_*`) or `graphify` (`graphify-out/`) — for placement/callers/impact instead of broad file reads; fall back to file reads.
4. **Frontend?** If the slice touches UI ([frontend-trigger](reference/frontend-trigger.md)),
   apply `devrites-frontend-craft`, **building to `design-brief.md`** — the UX/UI contract
   `/rite-spec` shaped (`devrites-ux-shape`). Read it as the target (direction, the slice's
   key states, interaction model) and refine it for this surface; don't re-derive the design
   from scratch. If the slice is UI but no `design-brief.md` exists (e.g. a spec written
   before shaping), shape it now via `devrites-ux-shape` before coding.
5. **Uncertain framework/library fact?** Apply `devrites-source-driven` and record the
   source in `decisions.md` / `evidence.md`.
6. **Changing behavior?** Use TDD / the Prove-It pattern when feasible —
   [tdd](reference/tdd.md): write the failing test first.
7. Implement the **smallest complete** version of the slice. Match project style.
8. **Before standing any non-trivial decision** (branching, boundary crossing, data
   model, auth, public API, migration, user-flow change, "this is safe/scales") apply
   `devrites-doubt`. The doubt loop honours `.devrites/AFK` (see its AFK exception):
   findings below the slice's gate ceiling become advisory entries in `questions.md`;
   destructive / auth / public-API concerns always pause regardless.
9. Run the slice's **targeted tests** — a quick sanity check for *this* slice (the
   comprehensive proof is `/rite-prove`, once all slices are built). For UI, verify in the
   browser during craft (`devrites-frontend-craft` / `devrites-browser-proof`).
10. **Fail-on-red.** If targeted tests / types / lint failed: do **not** mark the slice
    `built`. AFK → append a blocking question to `questions.md` (gate=blocking, slice's
    SLA) + set `Status: awaiting_human`; HITL → pause as a blocking gate. Either way,
    `Next step: /rite-plan unblock` until resolved.
11. Update `state.md`, `evidence.md`, `touched-files.md` (and `browser-evidence.md` for
    UI). Capture per [evidence-standard](reference/evidence-standard.md). If
    `.devrites/AFK` is present, decrement the budget by running
    `bash .claude/skills/rite-build/scripts/tick-afk.sh <state.md path>` — it decrements
    `state.md`'s `AFK slices remaining` field, prints the new value, and exits `3` when it
    hits 0. **Exit 3 → STOP** (forced HITL stop; the cap is exhausted). Never rewrite
    `.devrites/AFK` `max_slices` in place — it is read-only initial budget.
12. **STOP.** Report and recommend the next step.

> **Mid-flight discipline.** When tempted to do two slices, skip TDD, skip `devrites-doubt`, add a defensive check, or wander outside `touched-files.md` — see [`anti-patterns`](reference/anti-patterns.md). Load it the moment you reach for the excuse.

## Output
```
Built slice <N — name>
Acceptance: <met/partial + evidence>
Tests: <command → pass/fail>
Browser proof: <summary | n/a>
Drift: <none | recorded + how handled>
Next: slices still pending → /rite-build (slice <N+1>);
      ALL slices built → /rite-prove (prove the completed feature)
↻ Hygiene: /clear between slices (state.md + touched-files.md + evidence.md carry forward); /rite-handoff if away > a few hours. See rules/context-hygiene.md.
```
**DO NOT continue to the next slice automatically.**
