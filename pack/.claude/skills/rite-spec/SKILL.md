---
name: rite-spec
description: Investigate a new feature and write `spec.md` (placement, gaps, design refs, measurable acceptance) under `.devrites/work/<slug>/`. Use when the user says "new feature", "spec this out", "start a project", "I have an idea". Not for planning approved specs (use `/rite-define`) or replanning (use `/rite-plan`).
argument-hint: "<feature or idea>"
user-invocable: true
---

# /rite-spec — investigate deeply, write the spec

The spec phase. Turn a request (even a vague one) into a **fully-covered, correctly-placed
`spec.md`** by investigating deeply and closing every material gap with the human — so
`/rite-define` can plan it and nothing is missed. **No plan, tasks, or code here** — those
are `/rite-define` and `/rite-build`.

## Rules consulted (read on demand from `.claude/rules/`)
**Step 0:** Read `.claude/rules/core.md` first. DevRites skills Read `.claude/rules/core.md`
as their first step; the other rule files load on demand. Pull `documentation.md` via `Read`
when capturing significant spec decisions (why-not-what, ADR-style notes in `decisions.md`).

## Operating rules (DevRites core)
- No silent assumptions · no guessing through confusion · prefer existing conventions ·
  ask the human when an answer changes scope, placement, data model, UX, security,
  migration risk, or acceptance.

## Workflow
0. **Read `.claude/rules/core.md`** — the always-on operating rules and anti-rationalizations.
   Then **run the shared orientation preamble** — it prints `state.md`, the artifacts present,
   the run mode (HITL/AFK), and the open-question tally by gate, so you orient deterministically
   instead of re-deriving state from raw Markdown:
   ```bash
   P=.claude/skills/devrites-lib/scripts/preamble.sh
   [ -f "$P" ] || P="${CLAUDE_SKILL_DIR:-}/../devrites-lib/scripts/preamble.sh"
   [ -f "$P" ] || P=pack/.claude/skills/devrites-lib/scripts/preamble.sh
   [ -f "$P" ] && bash "$P" || echo "(orientation preamble unavailable on this install — read state.md directly to orient)"
   ```
1. **Understand the request** (`$ARGUMENTS`). Restate the goal and the *real problem
   behind it* in a sentence or two.
2. **Investigate deeply** — [investigation](reference/investigation.md). Produce, and
   later write into the spec: **current behavior**; **placement** (which module/layer/
   file/component should own it, the seam, patterns to reuse, and integration points —
   callers, dependents, data, APIs/events); **what it resolves**; **issues**
   (conflicts/constraints); **gaps** (unknowns); **blast radius**. Use a code-intelligence
   index — `codegraph` (`.codegraph/` / `codegraph_*`) or `graphify` (`graphify-out/`) —
   for placement/callers/impact instead of broad file reads; fall back to reading files.
   Also discover the project's **test / build/typecheck/lint** commands and the
   frontend/backend systems; read `PRODUCT.md` / `DESIGN.md` / `CLAUDE.md` if present.
3. **Gather design references (optional)** — [references-intake](reference/references-intake.md).
   The human **may** attach screenshots, mockups, a Figma link, a video, or links — or
   **none at all** (perfectly normal; skip this step then). If any are given: **view/fetch**
   them, **save local files** into `.devrites/work/<slug>/references/`, and index them in
   `references.md`. They become the target later phases verify against.
3a. **Shape the UX/UI before code (if the feature touches UI)** — when this feature is
   frontend ([frontend-trigger](../rite-build/reference/frontend-trigger.md)), apply
   `devrites-ux-shape` **now**, woven into the spec — not as a separate phase. It turns the
   references + the spec into a feature-level **`design-brief.md`** (design direction, key
   states, interaction model, optional Figma/image visual-direction probe) that `/rite-build`
   targets so the UI is built to plan, not guessed. In HITL it pauses for the human to
   confirm the direction; in AFK it asserts the best guess and logs it. Pure
   backend/data/CLI features skip this.
4. **Close the gaps with the human.** Turn each material gap/issue into a question — one
   at a time, **best guess attached**, structured options + escape hatch. (Vague ask →
   `devrites-interview`; rough idea → `rite-pressure-test`; ladders in
   [interview-patterns](reference/interview-patterns.md) / [question-protocol](reference/question-protocol.md).)
   For a vague ask, **map the decision tree** and resolve each branch depth-first; **cover
   every dimension** (objective · scope · data model · UX · integration · non-functional ·
   acceptance) — resolved or explicitly deferred — and **stop when answers converge** or you
   can predict them (don't interrogate). Aim for **zero blocking gaps**. *If a gap is genuinely undecidable on paper (state
   machine that may deadlock, data shape ambiguity, "which UX wins") → suggest a
   scoped detour to `/rite-prototype` to answer that ONE question before
   continuing.*
5. **Create the workspace** + set `.devrites/ACTIVE`
   ([state-workspace](reference/state-workspace.md)). Write `spec.md`
   ([spec-template](reference/spec-template.md)) — WHAT/WHY, technology-agnostic, with
   **Placement & integration**, **Design references**, **Gaps/issues/decisions**, and
   measurable acceptance ([acceptance-criteria](reference/acceptance-criteria.md)). Also
   write `brief.md`, `references.md`, `questions.md`, `decisions.md`, `assumptions.md`,
   and an initial `state.md` (phase: spec). When the feature touches UI, `design-brief.md`
   is written here too (by `devrites-ux-shape`, step 3a).
6. **Run the spec readiness gate** (bottom of spec-template): no blocking
   `[NEEDS CLARIFICATION]`, placement decided, all material gaps resolved, any design
   references provided are saved, **UX/UI shaped into `design-brief.md` if the feature is
   UI**, requirements testable, success criteria measurable. When it passes, write
   `Spec gate: passed <iso>` to `state.md`. **Stop** when it passes.

> **Mid-flight discipline.** When tempted to skip investigation depth, gap-closing, or placement decisions — see [`anti-patterns`](reference/anti-patterns.md) (Common Rationalizations + Red Flags). Load it the moment you reach for the excuse.

## Output
```
Spec ready: <slug>
Objective: <one sentence>   Placement: <where it lives>
Resolves: <value>
References: <n saved | none provided>
Design brief: <design-brief.md shaped (compact|full) | n/a — not UI>
Gaps closed: <n>   Open (non-blocking): <n>
Next: big / risky feature (auth · data model · public API · migration · multi-slice · ambiguous scope)?
      → /rite-temper   (strategic review: scope mode + pre-mortem, hardens the spec) — then /rite-define.
      Small / reversible / unambiguous? → /rite-define directly.
↻ Hygiene: /clear before /rite-define (spec.md + references/ + decisions.md + assumptions.md + questions.md captured); /rite-handoff if away > a few hours. See rules/context-hygiene.md.
```
If a workspace with the slug already exists, update its spec rather than overwriting blindly.
