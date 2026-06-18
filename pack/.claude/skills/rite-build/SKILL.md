---
name: rite-build
description: Implement exactly ONE vertical slice of the active feature, then stop with evidence. A fresh-context `devrites-slice-wright` writes the slice (orient → TDD → verify, anti-slop, project idiom); this skill gates it (readiness, HITL/AFK, doubt loop, Spec Drift Guard) and records the evidence. Use when the user says "build the next slice", "implement slice N", "continue", "code this slice". Not for bug fixes, prototypes, refactors outside scope, or two slices in a row.
argument-hint: "[slice number or name]"
user-invocable: true
---

# /rite-build — one verified slice

Build the next single slice, leave it working and proven, then **stop**. **Read the
active workspace first**; if none, tell the user to run `/rite-spec <feature>`.

This skill is the **orchestrator**: it owns the gates and the workspace; a fresh-context
[`devrites-slice-wright`](../../agents/devrites-slice-wright.md) owns the **writing**. You run
pre-flight (readiness, slice select, HITL pause), dispatch the wright for the build core, then
run the post-return gates (doubt, fail-on-red, record, stop). See
[`reference/wright-dispatch.md`](reference/wright-dispatch.md).

## Rules consulted (read on demand from `.claude/rules/`)
DevRites skills Read `.claude/rules/core.md` as their first step (workflow step 0). The
following load on demand — **the wright reads them** (they are named in its contract) while it
writes; read them yourself for the doubt/record gates or in the inline fallback:
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
  `rite-polish/reference/anti-ai-slop.md`). `devrites-slice-wright` enforces this **at the
  source** — its anti-slop charter is the same list: no over-defensive null/length checks, no
  blanket `catch`es that swallow errors, no useless wrappers, no over-engineered abstractions
  before two real callers, no generic AI naming, no tutorial-style comments, **don't go beyond
  the spec**, **reuse before you write**
  (`devrites-frontend-craft/reference/reuse-first.md`). It writes the code the *project* would
  write, in its idiom; you verify the charter held on return. Polish catches what slips; build
  prevents.

## Workflow ([one-slice-cycle](reference/one-slice-cycle.md))
0. **Rules + AFK + readiness check.** Read `.claude/rules/core.md` first. Then **run the
   shared orientation preamble** — it prints `state.md`, the artifacts present, the run
   mode (HITL/AFK), and the open-question tally by gate, deterministically:
   ```bash
   P=.claude/skills/devrites-lib/scripts/preamble.sh
   [ -f "$P" ] || P="${CLAUDE_SKILL_DIR:-}/../devrites-lib/scripts/preamble.sh"
   [ -f "$P" ] || P=pack/.claude/skills/devrites-lib/scripts/preamble.sh
   [ -f "$P" ] && bash "$P" || echo "(orientation preamble unavailable on this install — read state.md directly to orient)"
   ```
   Orient from its digest. If `Status == awaiting_human` → **STOP**, tell the user to run
   `/rite-resolve <qid> "<answer>"`. If `state.md` has no `Plan approved: <iso>` field
   → **STOP**, tell the user the plan isn't approved yet (`/rite-define` writes it when
   the human confirms). If `.devrites/AFK` is present, re-derive the remaining AFK budget
   from `state.md`'s `AFK slices remaining: <n>` field (initialized from `.devrites/AFK`
   `max_slices` on the first AFK build); if it is `0` → **STOP** (forced HITL stop; raise
   the count in `state.md` or remove the sentinel to continue). See
   [`reference/afk-discipline.md`](reference/afk-discipline.md).
1. Read `spec.md`, `plan.md`, `tasks.md`, `assumptions.md`, `drift.md`, and `test-plan.md`
   if present (the vetted coverage target from `/rite-vet` — the slice's tests come from
   here when it exists). `state.md` and the open-`questions.md` tally are already in the
   preamble digest from step 0 — re-read `questions.md` only for the full text of a flagged
   blocking question.
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
3. **Dispatch the build core to `devrites-slice-wright`** — one `Task` call, fresh context.
   Assemble the slice contract and send it per
   [`reference/wright-dispatch.md`](reference/wright-dispatch.md): the slice goal, acceptance
   criteria, and **scope boundary**; the paths it may touch (`touched-files.md`); the context
   paths to read (`spec.md`, `plan.md`, `decisions.md`, `assumptions.md`, plus `test-plan.md`
   when present — its per-gap test requirements + regression-criticals for this slice are the
   coverage the wright must write — and `design-brief.md`
   when the slice touches UI per [frontend-trigger](reference/frontend-trigger.md)); and the
   `.claude/rules/` files in scope. The wright **orients** on the project's idiom (preferring a
   code-intelligence index — `codegraph` (`.codegraph/` / `codegraph_*`) or `graphify`
   (`graphify-out/`) — for placement/callers/impact), writes the **failing test first** when
   behaviour changes ([tdd](reference/tdd.md)), implements the **smallest complete** version in
   the project's style (applying `devrites-frontend-craft` to `design-brief.md` for UI, and
   `devrites-source-driven` for uncertain framework facts), runs the slice's **targeted tests**
   (plus typecheck / lint / build where the project has them), and returns a structured artifact
   — **code + tests only; it does not write the workspace files.** If the slice is UI but no `design-brief.md` exists (e.g. a spec written before
   shaping), shape it via `devrites-ux-shape` before the wright codes. If the `Task` tool is
   unavailable, run the wright's discipline **inline** as a flagged fallback (see the reference)
   — same one-slice cycle, no isolation.
4. **Doubt the decisions it stood up.** For each entry in the wright's `Decisions stood`
   (branching, boundary crossing, data model, auth, public API, migration, user-flow change,
   "this is safe/scales") apply `devrites-doubt` **before accepting the slice** — the writer
   doesn't grade its own decisions. The doubt loop honours `.devrites/AFK` (see its AFK
   exception): findings below the slice's gate ceiling become advisory entries in `questions.md`;
   destructive / auth / public-API concerns always pause regardless. A non-empty `Escalation` in
   the artifact is handled here too: irreversible-risk / blockers → blocking question + set
   `Status: awaiting_human`; a scope-changing answer → `/rite-plan repair` (Spec Drift Guard),
   never silently into the slice. **If an irreversible-risk item shows up under the wright's
   `Decisions stood` rather than `Escalation`**, treat that misclassification as itself a
   blocking protocol violation — pause and re-dispatch with the item flagged out-of-bounds, do
   **not** doubt-and-accept it. (The wright's return is the not-yet-load-bearing moment — the
   slice isn't `built` or merged yet — so this post-return doubt is still pre-commit.)
5. **Fail-on-red.** If the wright's `Gates` were red (targeted tests / types / lint) or it
   couldn't verify: do **not** mark the slice `built`. AFK → append a blocking question to
   `questions.md` (gate=blocking, slice's SLA) + set `Status: awaiting_human`; HITL → pause as a
   blocking gate. Either way, `Next step: /rite-plan unblock` until resolved.
6. **Record — you are the canonical writer.** From the wright's artifact, update `state.md`,
   `evidence.md`, `touched-files.md` (and `browser-evidence.md` for UI). **Evidence is the
   wright's real command output, not its say-so.** Capture per
   [evidence-standard](reference/evidence-standard.md). If `.devrites/AFK` is present, decrement
   the budget by running `bash .claude/skills/devrites-lib/scripts/tick-afk.sh <state.md path>` —
   it decrements `state.md`'s `AFK slices remaining` field, prints the new value, and exits `3`
   when it hits 0. **Exit 3 → STOP** (forced HITL stop; the cap is exhausted). Never rewrite
   `.devrites/AFK` `max_slices` in place — it is read-only initial budget.
7. **STOP.** Report and recommend the next step.

> **Mid-flight discipline.** The wright (or you, in the inline fallback) must resist doing two slices, skipping TDD, adding a defensive check, or wandering outside `touched-files.md`; you must resist skipping the post-return `devrites-doubt` because the wright "seems confident" — see [`anti-patterns`](reference/anti-patterns.md). Load it the moment you reach for the excuse.

## Output
```
Built slice <N — name>
Built by: devrites-slice-wright (fresh context) | inline fallback
Acceptance: <met/partial + evidence>
Tests: <command → pass/fail>
Browser proof: <summary | n/a>
Drift: <none | recorded + how handled>
Next: slices still pending → /rite-build (slice <N+1>);
      ALL slices built → /rite-prove (prove the completed feature)
↻ Hygiene: /clear between slices (state.md + touched-files.md + evidence.md carry forward); /rite-handoff if away > a few hours. See rules/context-hygiene.md.
```
**DO NOT continue to the next slice automatically.**
