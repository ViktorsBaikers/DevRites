---
name: rite-vet
description: Vet a defined plan before code with senior-engineer engineering review. Use when the user says "vet the plan", "engineering review", "lock in the plan", or before building. Not for code review or final seal.
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
contract, so the build follows a reviewed plan. Runs on **every** plan (depth scales to
stakes; never skipped) and is always part of `/rite-autocomplete`; `--cross-model` adds a
different-model second opinion. **Read the active workspace first**; if there's no
`plan.md`, tell the user to run `/rite-define`.

This is the engineering counterpart to `/rite-temper` (which is strategic, on the *spec*).
Temper decides *the right thing*; vet decides *the right way to build it*.

## Rules consulted (read on demand from `.claude/skills/devrites-lib/reference/standards/`)
**Step 0:** Read `.claude/skills/devrites-lib/reference/standards/core.md` first. Pull on demand: `principles.md` (the project
invariants gate — how `.devrites/principles.md` is scored pass/fail), `patterns.md` +
`coding-style.md` (the over-engineering / reuse-first / YAGNI rubric — reuse the pack's
standard), `testing.md` (the test-coverage axis) + `spec-grammar.md` (when the spec uses
structured Requirement/Scenario blocks, each scenario is a coverage unit `test-plan.md` must
map), `performance.md` (the perf axis),
`error-handling.md` (failure-mode coverage), `development-workflow.md` (parallel lanes,
definition of done), `afk-hitl.md` (irreversible-risk list + gate ceiling),
`developer-experience.md` (when the plan ships a developer-facing surface — API / CLI / SDK /
webhook / config / error messages / getting-started — predict the DX scorecard here),
`elicitation.md` (the move-set to deepen an axis finding or a risky design choice — selected by
the section's risk: Tournament for two viable designs, Delphi for a shaky estimate, Assumption
Audit for a plan resting on unstated beliefs).
- `definition-of-done.md` — standing Done bar: acceptance mapped, fresh proof, no open hard gates, scoped edits, rollback/docs where needed.


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
0. **Read `.claude/skills/devrites-lib/reference/standards/core.md`** first.
   Then **run the shared orientation preamble** — it prints `state.md`, the artifacts present,
   the run mode (HITL/AFK), and the open-question tally by gate, so you orient deterministically
   instead of re-deriving state from raw Markdown:
   ```bash
   devrites-engine preamble
   devrites-engine snapshot
   ```
   Then the workspace: `plan.md`, `tasks.md`, `spec.md`
   (for intent + acceptance), `strategy.md` (if `/rite-temper` ran), `decisions.md`,
   `assumptions.md`, `design-brief.md` (if UI), `state.md`. Require a `plan.md` whose
   Readiness gate passes (or `Plan approved`) — else STOP → `/rite-define`. Prefer a
   code-intelligence index if available — codebase-memory-mcp first, cross-checked with codegraph + graphify, else standard methods (LSP / Read/Grep/Glob)
   (see `.claude/skills/devrites-lib/reference/standards/tooling.md`) — for placement / blast-radius / reuse checks.
1. **Calibrate depth — never skip** — [`reference/depth.md`](reference/depth.md). Every plan is
   vetted; what scales is the *depth*. A simple, single-module, reversible plan with no
   irreversible-risk / data-model / new-pattern trigger → **light pass** (brief scope check + a
   one-line scan per axis + the acceptance→test map). Any full-pass trigger (or `--full`) → the
   **full pass** below. There is no skip: every feature leaves a recorded engineering verdict and
   a `test-plan.md` coverage map.
2. **Scope Challenge (blocking gate)** — [`reference/review-axes.md`](reference/review-axes.md)
   §0. Search prior archived decisions for the plan's main nouns before asking the human to re-decide:
   `devrites-engine decisions search "<2-4 plan nouns>"` (run `decisions index` first if needed).
   What already exists that solves a sub-problem (reuse vs rebuild)? The minimum diff for the
   stated acceptance? Complexity smell (the plan touches **>8 files** or adds **>2 new
   services/modules**) → **STOP and ask** before any axis. Verify each new pattern / infra choice
   against a built-in (dispatch `devrites-source-driven`); completeness check (with AI, full
   coverage is ~100× cheaper than the human-hours saved by a shortcut — prefer complete); and a
   distribution check for any new artifact.
2a. **Cross-artifact analyze gate + principles / charter / conventions gate.** Before the axes, run
   one read-only consistency+coverage pass over `spec.md` + `plan.md` + `tasks.md` (+ `traceability.md`
   if present); any **CRITICAL** — an acceptance criterion with no slice, a slice satisfying no
   criterion, a contradiction across artifacts — **blocks `/rite-build`** until resolved. Then score
   the three project gates as explicit **pass/fail** on the planned approach:
   - **Principles** (`.devrites/principles.md`, rubric in [`principles.md`](../devrites-lib/reference/standards/principles.md))
     — the authored invariants the project will not break. A plan that bakes in a violation of a
     declared principle with **no recorded, human-approved exception** is a **top-severity** finding,
     walked **first**, and **blocks `/rite-build`**. Absent or empty file → none declared → passes;
     **never block for the absence of principles**. A genuine need to break one routes to a scoped,
     dated exception in the principles register — never a silent work-around (adding the exception is
     an irreversible-risk decision: it always pauses for a human, even in AFK).
   - **The anti-slop charter** (`coding-style.md` + `prose-style.md`) and **the conventions ledger**
     (`.devrites/conventions.md`) — a plan that bakes in a god-module, a speculative abstraction with
     no second caller, or a dependency where an in-repo option exists is a **top-severity** violation.
   **Re-check all three after the axes harden the plan** (post-design). Write the result to `analysis.md`.
   ```bash
   devrites-engine analyze; echo "analyze rc=$?"
   ```
3. **Four-axis review** — [`reference/review-axes.md`](reference/review-axes.md), through the
   senior-engineer lenses in [`reference/eng-lenses.md`](reference/eng-lenses.md): **Architecture
   → Plan code-quality → Test-coverage design → Performance**, ≤8 findings per axis, each
   `[severity] (confidence: N/10) <ref> — finding`. **Walk findings WITH the human, one at a
   time** via `AskUserQuestion` (best-guess + why + options with effort/risk/maintenance, mapped
   to a rule) — the artifact is the *output* of the review, not a substitute for it.
   When an axis finding hinges on a genuinely open design choice or a shaky estimate, deepen that
   one finding with a fitting technique from
   [`elicitation.md`](../devrites-lib/reference/standards/elicitation.md) (Tournament for two viable
   designs, Delphi for the estimate, Assumption Audit for unstated beliefs) before you band it. (AFK ceiling
   single-sourced in [`reference/depth.md`](reference/depth.md): hardening /
   coverage-increasing findings auto-apply; **anything that grows scope or changes acceptance is a
   blocking pause**; irreversible-risk always pauses.)
4. **Required outputs** — the test-coverage diagram + per-gap test requirements (the **regression
   rule** is mandatory, no question). When the spec uses the structured grammar
   (`spec-grammar.md`), the diagram maps **each `#### Scenario:` (WHEN/THEN) to ≥1 planned
   test** — an unmapped scenario is a coverage gap the build must close. Then the failure-mode
   table, "NOT in scope", "What already exists",
   and the worktree parallelization strategy. Ground the parallelization section in the
   engine's advisory planner:
   ```bash
   devrites-engine lanes plan "$(cat .devrites/ACTIVE 2>/dev/null)"
   ```
   Shapes in [`reference/review-axes.md`](reference/review-axes.md).
   **Developer-facing surface?** If the plan ships one (`developer-experience.md` — API / CLI / SDK /
   webhook / config / error messages / getting-started), the DX scorecard is **predicted by a
   fresh-context `devrites-devex-reviewer` in predict mode** (dispatched in step 6 alongside the
   plan-reviewer): it scores the *planned* surface — time-to-hello-world estimate plus the
   getting-started, error-message, and ergonomics friction the plan bakes in — and writes `devex.md`,
   the **prediction the boomerang measures against** at `/rite-prove` / `/rite-seal`. If subagents are
   unavailable, score it inline as a flagged fallback. Absent surface → skip, no `devex.md`
   (greenfield no-op, like the principles gate).
   Also a **PRP one-pass-implementable check** per slice brief (the build's pre-flight): confirm each
   slice's Consumes/Produces, Known-Gotchas, validation commands, and reuse targets are present and
   concrete. A UI slice also needs `Design brief states` and binary `Visual acceptance`
   (state × viewport × input + target R-id/brief rule). A brief that can't be built and
   visually judged in one pass is a finding; harden the slice until it clears before `/rite-build`.
4a. **Forge gate (rare — confirm or clear).** For each slice carrying `Forge: yes` (proposed by
   `/rite-define`), and any slice the architecture axis showed has **≥2 genuinely-viable approaches
   with no clear winner at Complexity ≥4**, confirm the flag: name the 2–3 candidate strategies that
   actually differ (different data shape, different seam, reuse-vs-build — not variations of one), and
   confirm the slice's acceptance + `test-plan.md` give the judge an objective scorecard. **Clear**
   `Forge: yes` back to `no` when the review settled on one approach, the slice is below the complexity
   bar, or you can't name two real strategies — competing a decided or trivial slice burns K× the build
   for nothing. Forge is a **build-cost** decision, not an irreversible one: it never bypasses a gate,
   and under AFK its K candidates count against the slice budget. Record the confirmed strategies in
   the slice brief so `/rite-build` competes them
   ([`rite-build/reference/forge.md`](../rite-build/reference/forge.md)). No flagged slice → nothing to do.
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
   reasoning). **When the plan ships a developer-facing surface** (`developer-experience.md`),
   dispatch [`devrites-devex-reviewer`](../../agents/devrites-devex-reviewer.md) in **predict mode**
   in the same parallel pass — it scores the *planned* surface and writes the predicted `devex.md`
   (Source mode); a single predict pass suffices pre-build, so it sits outside the plan-rubric
   iteration. Resolve actionable findings, re-dispatch the plan-reviewer; **cap ≤3 iterations**. An axis still
   below bar after 3 → blocking question (HITL) or AFK gate-ceiling entry. On a **light pass**
   the fresh-context loop is skipped — the per-axis scan + the `test-plan.md` coverage map are the
   verdict (escalate to this loop if the light scan surfaces a real finding). On a **full pass**, run
   `devrites-engine outside-voice`; when it prints `available`, add one genuinely different-model
   Codex pass over the same artifacts/diff. `--cross-model` forces this check even outside full-pass
   defaults. Findings are informational until the human approves each one with line quotes —
   [`reference/cross-model.md`](reference/cross-model.md). If sub-agents are unavailable, do the
   independent rubric pass yourself in a separate read, discarding the authoring reasoning (a
   flagged fallback, not an independent review).
7. **STOP.** Report the scope verdict, the per-axis floor, the coverage gaps closed, and the
   failure-mode criticals; recommend `/rite-build`.

> **Mid-flight discipline.** When tempted to batch-dump findings into `eng-review.md` and skip
> the walk-through, harden the plan past a finding that actually changes acceptance, score before
> quoting the source, or wave through a complexity smell "to keep moving" — see
> [`reference/anti-patterns.md`](reference/anti-patterns.md).

## Output

**Progress first** — run `devrites-engine progress`, then use the shared completion reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md)).
Default success shape:
```
Done: plan vetted for <slug>; depth <light|full> with axis floor <band>.
Changed: eng-review.md, test-plan.md, plan.md, decisions.md
Evidence: coverage <x/y> planned; findings Critical <n> / Important <n> / Suggestion <n>; reviewer loop <n>; outside-voice <ran|skipped-unavailable|disabled>
Open: <none | blockers | plan deltas routed via Spec Drift Guard>
Next: /rite-build
Record: .devrites/work/<slug>/eng-review.md
↻ Hygiene: /clear before /rite-build
```
**DO NOT write code, slice, or run the build here** — that's `/rite-build`. Vet reviews and hardens the plan; it never implements.
