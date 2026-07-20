---
name: rite-define
description: Define the first build plan from an approved spec: architecture, task slices, traceability, and state. Use when turning approved intent into its initial plan.
argument-hint: "[feature-slug]"
user-invocable: true
---

# /rite-define — plan from the spec

Read the active feature's `spec.md` and turn it into a buildable workspace: feature
architecture, approach, a dependency-ordered set of **vertical slices**, traceability, and
the state cursor. The spec is the WHAT/WHY (from `/rite-spec`); this is the HOW. Splitting
spec, architecture, plan, tasks, and traceability keeps each file small and phase-owned.
**No code here.**

## Rules consulted (read on demand from `.claude/skills/devrites-lib/reference/standards/`)
Pull these via `Read` when shaping the plan:
- `development-workflow.md` — small batches, trunk-always-green, definition of done.
- `principles.md` — the project invariants (`.devrites/principles.md`) the chosen approach must conform to.
- `documentation.md` — record plan-time decisions and rationale.
- `../workspace-artifact-schema.md` — artifact purposes, budgets, IDs, and read triggers.

## Operating rules
- **Requires a readied spec.** Read the active workspace first; if `.devrites/ACTIVE` is empty,
  the workspace has no `spec.md`, its readiness gate hasn't passed, or any spec-quality
  `checklists/<domain>.md` has an open CRITICAL → **STOP** and tell the user to run
  `/rite-spec <feature>` first. **DO NOT plan from a missing or unreadied spec.**
- Prefer existing conventions; ask before adding a dependency or a second design system.
- **Author section by section, not in one dump.** Write `architecture.md` / `plan.md` one section
  at a time and pause after each; a section resting on an open design choice or a shaky estimate can
  be deepened right there with a technique from
  [`elicitation.md`](../devrites-lib/reference/standards/elicitation.md) (Tournament for two viable
  designs, Delphi for the estimate) before it hardens into slices.
- **Slice count is derived, never dictated.** The number of slices falls out of the work
  — one per independently-shippable increment, sized by `slicing.md`, every acceptance
  criterion mapped to ≥1 slice. A user-named count is a hint at most: slice logically and,
  if your honest count differs, present it and why. Never pad or compress to hit a figure.
  (`.devrites/AFK` `max_slices` is a separate AFK iteration budget, not the decomposition.)
- **Wide mechanical refactors slice expand → migrate → contract.** If one repeated change
  crosses many files, don't fake vertical slices. Add a compatibility/adapter slice, migrate
  small green batches, then remove the old path. If a batch cannot stay green, use an
  integration branch plus a final verify slice.

## Workflow
0. **Read `.claude/skills/devrites-lib/reference/standards/core.md`** — the always-on operating rules and anti-rationalizations.
   Then **run the shared orientation preamble** — it confirms the active feature and which
   artifacts exist (it prints `state.md`, the artifacts present, the run mode, and the
   open-question tally):
   ```bash
   devrites-engine preamble
   devrites-engine snapshot
   devrites-engine spec-skeleton ".devrites/work/$(cat .devrites/ACTIVE 2>/dev/null)"
   ```
   If there is no active workspace, no `spec.md`, `spec-skeleton` blocks, or its readiness gate hasn't passed →
   **STOP** and tell the user to run `/rite-spec <feature>` first.
1. **Read the spec** — `spec.md` (objective, requirements, acceptance, **placement**,
   design references, gaps/decisions), plus `references.md`, `decisions.md`,
   `assumptions.md`, **`strategy.md` if present** (the scope mode, deferred / out-of-scope
   register, and pre-mortem risks from `/rite-temper` — cut slices to mitigate the top risks
   and respect the IN/OUT line; map coverage against the **hardened** spec), and
   **`design-brief.md` if the feature touches UI** (the UX/UI contract `/rite-spec` shaped —
   its key states, interaction model, and proof targets drive how UI slices are cut). If a blocking
   `[NEEDS CLARIFICATION]` remains, stop → `/rite-spec`.
2. **Decide the architecture + approach** (the HOW the spec deliberately omitted): write
   `architecture.md` for owning layer, boundaries, integration points, data/API/events,
   dependencies, risks, and affected areas; write only the build strategy in `plan.md`.
   Use a
   code-intelligence index if available (see
   `.claude/skills/devrites-lib/reference/standards/tooling.md`) for structure/impact; for the current API or behaviour of
   an external library/framework the architecture will rely on, consult context7 if available.
   Record significant options in `decisions.md` as `DEC-###` ADR entries.
   **Deep-modules check** — while sketching the major modules, look for opportunities
   to extract a **deep module**: a small, stable interface that hides a meaningful chunk
   of behavior, and is therefore independently testable. A *shallow* module — interface
   nearly as complex as its implementation — earns nothing; either deepen it or delete
   it. Where a slice will produce a deep module, confirm with the user which deep
   modules they want unit-tested in isolation (this informs the slice's "Tests to
   write/run" field).
3. **Slice into vertical tasks** — each delivers one observable capability end-to-end and
   is verifiable on its own; the **count emerges from the work, not a target number**;
   first slice = thinnest useful end-to-end path; order by dependency (risk-first within a
   tier). For a broad mechanical refactor, use expand → migrate batches → contract instead
   of pretending each touched file is a product slice; every migrate batch must stay green,
   or route through an integration branch + final verify slice. Use `rite-plan/reference/slicing.md` and
   `rite-plan/reference/task-breakdown.md`. Mark per slice: **Frontend craft required**
   and **Browser proof required** (UI), and whether it's **fullstack** (FE+BE → contract
   first, see `devrites-frontend-craft/reference/fullstack.md`). **For UI slices, name which
   of `design-brief.md`'s key states + interaction the slice delivers, and give it a binary
   **Visual acceptance** target (state × viewport × input + target R-id/brief rule)** — so
   the design contract maps onto slices, not just acceptance criteria.
4. **Map coverage** — every `AC-###` spec acceptance criterion maps to ≥1 `SLICE-###`
   (`rite-spec/reference/acceptance-criteria.md`); no orphaned criteria, no slice without a
   criterion. Lift covered/backstop `Edge Coverage` rows and resolved `Prohibitions (must-NOT)`
   rows into `traceability.md` and `test-plan.md`; unresolved rows go to `assumptions.md` with
   their gate/owner.
4a. **Parallel-lane sanity check** — after drafting `tasks.md` but before asking for plan
   approval, run the advisory lane planner:
   ```bash
   devrites-engine lanes plan "$(cat .devrites/ACTIVE 2>/dev/null)"
   ```
   Use it to spot independent read-only/review lanes and dependency mistakes, but do not
   weaken DevRites' default of one production-write slice at a time.
4b. **Persist the traceability matrix** — write `traceability.md` (`AC/REQ ID → slice(s) →
   test/proof → evidence ID → touched files → status`), the living map `/rite-prove` and
   `/rite-seal` walk. Generate it with `devrites-engine coverage` when available, then save/rename
   the output as `traceability.md`, or write the table by hand from the same inputs if the
   script is absent:
   ```bash
   S="$(cat .devrites/ACTIVE 2>/dev/null)"
   devrites-engine coverage "$S" > ".devrites/work/$S/traceability.md"
   ```
5. **Complexity & deviations gate** — justify anything off DevRites defaults (new dep,
   extra abstraction, second design system) in the plan; if you can't justify it, simplify.
   **Principles conformance:** read `.devrites/principles.md` (if present) and confirm the
   approach honors every declared invariant. A plan that conflicts with one is not "a deviation
   to justify away" — either reshape the approach to conform, or, when the conflict is genuine and
   intended, route it through the Spec Drift Guard plus a recorded decision and a scoped principle
   exception a human approves. Never ready a plan that silently violates an invariant. (Re-scored
   as a blocking gate at `/rite-vet`; no file → none declared → nothing to check.)
6. **Write** `architecture.md`, `plan.md`, `tasks.md`, and `traceability.md`; update
   `state.md` (phase: plan → next `/rite-vet`).
6a. **Cross-artifact gate — now that `tasks.md` exists.** Run the deterministic
   spec↔tasks coverage/consistency check; any non-zero result blocks plan readiness:
   ```bash
   S="$(cat .devrites/ACTIVE 2>/dev/null)"
   devrites-engine analyze "$S"
   ```
7. **Readiness gate** (bottom of plan-template): every acceptance criterion covered by a
   slice, dependency order acyclic + risk-first, no unjustified deviation, rollback for
   every destructive/migration step. **Stop and confirm** before code. Render the review-before-code
   digest first: `Intent` (one sentence from the spec), `Done means` (acceptance coverage x/y),
   `Plan sanity` (slice count + riskiest boundary/gate), and `Build exactly this?` (yes → approve;
   no → `/rite-plan revise`). When the human confirms the plan, write `Plan approved: <iso>` to
   `state.md` (see [state-workspace](../rite-spec/reference/state-workspace.md)); `/rite-build`
   checks this exists before building.

## tasks.md slice format

Use the canonical slice grammar in
[`workspace-artifact-schema.md`](../devrites-lib/reference/workspace-artifact-schema.md#canonical-slice-grammar).
Every slice must satisfy that complete field set; phase-specific gate details live in
[`reference/gates.md`](reference/gates.md).

> **Mid-flight discipline.** When tempted to skip vertical slicing, coverage mapping, or dependency-order discipline — see [`anti-patterns`](reference/anti-patterns.md) (Common Rationalizations + Red Flags). Load it the moment you reach for the excuse.

## Output

**Progress first** — run `devrites-engine progress`, then use the shared completion reply contract
([`devrites-lib/reference/reply-contract.md`](../devrites-lib/reference/reply-contract.md)).
Default success shape:
```
Done: plan written for <slug>; <n> vertical slices defined.
Changed: architecture.md, plan.md, tasks.md, traceability.md, decisions.md, state.md
Evidence: not applicable; acceptance coverage <x/y> mapped in traceability.md
Open: <none | plan questions | Alternative: /rite-plan revise to reshape artifacts>; review digest: intent + coverage + plan sanity rendered
Next: /rite-vet
Record: .devrites/work/<slug>/plan.md
↻ Hygiene: /clear after user confirms the plan
```
