---
name: rite-vet
description: Engineering review of a defined plan before any code — challenge implementation scope (reuse-vs-rebuild, minimum diff, complexity smell), review architecture / plan code-quality / test-coverage design / performance through senior-engineer lenses, confidence-band every finding behind a quote-the-source verification gate, then harden `plan.md` / `tasks.md` and write `eng-review.md` + the build-readable `test-plan.md` (spec-level gaps fold back via the Spec Drift Guard). Optional `--cross-model` adds a different-model second opinion. Use when the user says "vet the plan", "engineering review", "review the architecture", "lock in the plan", "check the implementation plan", or before building any feature. Runs on every plan — depth scales to stakes (light pass on simple plans, full rigor on big/risky), never skipped; always part of `/rite-autocomplete`. Not for the spec's strategy (`/rite-temper`), one mid-build decision (`devrites-doubt`), a code diff (`/rite-review`), or the final gate (`/rite-seal`).
argument-hint: "[slug] [--cross-model] [--full]"
user-invocable: true
---

# /rite-vet — vet the plan before you build

Take a defined plan and **vet** it the way a senior staff engineer would in a plan review:
challenge the scope, walk architecture / plan code-quality / test-coverage / performance,
calibrate every finding by confidence (and refuse to emit one you can't trace to a quoted
line), design the test coverage the build will target, and map the failure modes and
parallel lanes — *before* `/rite-build` writes a line. The one DevRites step that hardens
the **implementation plan** at the engineering level and folds the result into the canonical
contract, so the build follows a reviewed plan. **Read the active workspace first**; if
there's no `plan.md`, tell the user to run `/rite-define`.

This is the engineering counterpart to `/rite-temper` (which is strategic, on the *spec*).
Temper decides *the right thing*; vet decides *the right way to build it*.

## Rules consulted (read on demand from `.claude/rules/`)
**Step 0:** Read `.claude/rules/core.md` first. Pull on demand: `patterns.md` +
`coding-style.md` (the over-engineering / reuse-first / YAGNI rubric — reuse the pack's
standard), `testing.md` (the test-coverage axis), `performance.md` (the perf axis),
`error-handling.md` (failure-mode coverage), `development-workflow.md` (parallel lanes,
definition of done), `afk-hitl.md` (irreversible-risk list + gate ceiling).

## Operating rules
- **Review the plan, not the spec's ambition.** The spec's scope/ambition is `/rite-temper`'s
  job and is treated as settled here. Vet asks *given this scope, is this the right, simplest,
  best-tested, lowest-risk way to build it* — and challenges only implementation scope creep.
- **You harden the plan directly; the reviewer judges.** Vet *is* the plan-hardening phase, so
  behavior-preserving plan refinements (test requirements, tightened scope boundaries, ordering,
  parallel lanes, error-handling + failure-mode coverage) are written straight into
  `plan.md` / `tasks.md` / `test-plan.md` — you are the single canonical writer. A finding that
  changes **acceptance criteria or product behavior** is *not* a plan refinement: it routes
  through the **Spec Drift Guard** (record in `drift.md`, recorded decision, then `/rite-plan
  repair` for any structural reslice). Nothing that grows the build's scope lands without a
  recorded human decision.
- **Confidence over assertion.** Every finding carries a confidence band; a finding you cannot
  back by quoting the plan/spec line (or the code it references) is forced to low confidence and
  suppressed from the main report — see the verification gate in [`reference/review-axes.md`](reference/review-axes.md).
- **Bound rigor by reversibility.** Auth / migration / public-API / data-model touches get
  maximum conservatism and always pause (irreversible-risk list), regardless of run mode.
- **Honest verdict, gated on the floor.** Never round "thin" up to "ready"; the axis verdict is
  the weakest finding, not an average. Record every call's *why*.

## Workflow
0. **Read `.claude/rules/core.md`**, then the workspace: `plan.md`, `tasks.md`, `spec.md`
   (for intent + acceptance), `strategy.md` (if `/rite-temper` ran), `decisions.md`,
   `assumptions.md`, `design-brief.md` (if UI), `state.md`. Require a `plan.md` whose
   Readiness gate passes (or `Plan approved`) — else STOP → `/rite-define`. Prefer the
   code-intelligence index (codegraph / graphify) for placement / blast-radius / reuse checks.
1. **Calibrate depth — never skip** — [`reference/depth.md`](reference/depth.md). Every plan is
   vetted; what scales is the *depth*. A simple, single-module, reversible plan with no
   irreversible-risk / data-model / new-pattern trigger → **light pass** (brief scope check + a
   one-line scan per axis + the acceptance→test map). Any full-pass trigger (or `--full`) → the
   **full pass** below. There is no skip: every feature leaves a recorded engineering verdict and
   a `test-plan.md` coverage map.
2. **Scope Challenge (blocking gate)** — [`reference/review-axes.md`](reference/review-axes.md)
   §0. What already exists that solves a sub-problem (reuse vs rebuild)? The minimum diff for the
   stated acceptance? Complexity smell (the plan touches **>8 files** or adds **>2 new
   services/modules**) → **STOP and ask** before any axis. Verify each new pattern / infra choice
   against a built-in (dispatch `devrites-source-driven`); completeness check (with AI, full
   coverage is ~100× cheaper than the human-hours saved by a shortcut — prefer complete); and a
   distribution check for any new artifact.
3. **Four-axis review** — [`reference/review-axes.md`](reference/review-axes.md), through the
   senior-engineer lenses in [`reference/eng-lenses.md`](reference/eng-lenses.md): **Architecture
   → Plan code-quality → Test-coverage design → Performance**, ≤8 findings per axis, each
   `[severity] (confidence: N/10) <ref> — finding`. **Walk findings WITH the human, one at a
   time** via `AskUserQuestion` (best-guess + why + options with effort/risk/maintenance, mapped
   to a rule) — the artifact is the *output* of the review, not a substitute for it. (AFK ceiling
   single-sourced in [`reference/depth.md`](reference/depth.md): hardening /
   coverage-increasing findings auto-apply; **anything that grows scope or changes acceptance is a
   blocking pause**; irreversible-risk always pauses.)
4. **Required outputs** — the test-coverage diagram + per-gap test requirements (the **regression
   rule** is mandatory, no question), failure-mode table, "NOT in scope", "What already exists",
   and the worktree parallelization strategy. Shapes in [`reference/review-axes.md`](reference/review-axes.md).
5. **Write `eng-review.md` + `test-plan.md`, fold back** — [`reference/artifacts.md`](reference/artifacts.md).
   `eng-review.md` is the durable record; `test-plan.md` is the build-readable coverage target
   (`/rite-build` and `/rite-prove` read it). Harden `plan.md` / `tasks.md` directly for
   behavior-preserving refinements; route every acceptance/behavior-changing delta through the
   **Spec Drift Guard** (`drift.md` + recorded decision + `/rite-plan repair`). Append
   `decisions.md` (one ADR per material call) and `assumptions.md`. Update `state.md`:
   `Phase: vet`, `Next step: /rite-build`; on a blocking pause write the `Awaiting human` block +
   `Status: awaiting_human` before stopping.
6. **Adversarial verification loop (full pass)** — dispatch [`devrites-plan-reviewer`](../../agents/devrites-plan-reviewer.md)
   (fresh context, **only** `plan.md` + `tasks.md` + `spec.md` + the rubric — no authoring
   reasoning). Resolve actionable findings, re-dispatch; **cap ≤3 iterations**. An axis still
   below bar after 3 → blocking question (HITL) or AFK gate-ceiling entry. On a **light pass**
   the fresh-context loop is skipped — the per-axis scan + the `test-plan.md` coverage map are the
   verdict (escalate to this loop if the light scan surfaces a real finding). With `--cross-model`,
   add one genuinely different-model pass — [`reference/cross-model.md`](reference/cross-model.md)
   (informational until the human approves each finding). If sub-agents are unavailable, do the
   independent rubric pass yourself in a separate read, discarding the authoring reasoning (a
   flagged fallback, not an independent review).
7. **STOP.** Report the scope verdict, the per-axis floor, the coverage gaps closed, and the
   failure-mode criticals; recommend `/rite-build`.

> **Mid-flight discipline.** When tempted to batch-dump findings into `eng-review.md` and skip
> the walk-through, harden the plan past a finding that actually changes acceptance, score before
> quoting the source, or wave through a complexity smell "to keep moving" — see
> [`reference/anti-patterns.md`](reference/anti-patterns.md).

## Output
```
Vetted: <slug>
Depth: light | full (<trigger that escalated it>)
Scope: reuse <n found> / minimum-diff <ok|trimmed N> / complexity <ok|smell: N files, M new services → asked>
Axes (floor → verdict):  Architecture <band> · Code-quality <band> · Tests <band> · Performance <band>
Findings: <Critical n / Important n / Suggestion n>   (suppressed low-confidence: n)
Coverage: <x/y paths> planned · GAPS closed <n> · regressions flagged <n>  → test-plan.md
Failure modes: <n> mapped (<n critical: no test + no handling + silent>)
Parallelization: <n lanes — n parallel / n sequential> | sequential (no opportunity)
Reviewer loop: <n> iter · cross-model: ran (codex) | off
Plan: hardened in place | <n> deltas routed via Spec Drift Guard → /rite-plan repair
Next: /rite-build   (builds the vetted plan)
↻ Hygiene: /clear before /rite-build (eng-review.md + test-plan.md + plan edits + decisions.md captured). See rules/context-hygiene.md.
```
**DO NOT write code, slice, or run the build here** — that's `/rite-build`. Vet reviews and hardens the plan; it never implements.
