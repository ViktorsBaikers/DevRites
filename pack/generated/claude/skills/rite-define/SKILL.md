---
name: rite-define
description: Define the first build plan from an approved spec: architecture, task slices, traceability, and state. Use when turning approved intent into its initial plan.
argument-hint: "[feature-slug]"
user-invocable: true
---

# /rite-define: plan from the spec

Turn the active feature's `spec.md` into a buildable workspace with architecture, an
implementation approach, dependency-ordered **vertical slices**, traceability, and a
state cursor. The spec defines what and why; this phase defines how. Keep spec,
architecture, plan, tasks, and traceability in their phase-owned files. **Do not write
code here.**

## Rules consulted (read on demand from `.claude/skills/devrites-lib/reference/standards/`)
Pull these via `Read` when shaping the plan:
- `development-workflow.md`: small batches, trunk-always-green, definition of done.
- `principles.md`: the project invariants (`.devrites/principles.md`) the chosen approach must conform to.
- `documentation.md`: record plan-time decisions and rationale.
- `../workspace-artifact-schema.md`: artifact purposes, budgets, IDs, and read triggers.

## Operating rules
- **Requires a readied spec.** Read the active workspace first; if `.devrites/ACTIVE` is empty,
  the workspace has no `spec.md`, its readiness gate hasn't passed, or any spec-quality
  `checklists/<domain>.md` has an open CRITICAL → **STOP** and tell the user to run
  `/rite-spec <feature>` first. A missing or non-`CLEAR` `decision-coverage.md` routes to
  `/rite-clarify`. **DO NOT plan from a missing, unreadied, or unclarified spec.**
- Apply `afk-hitl.md` decision ownership. Prefer conventions; source-check new dependencies or
  design systems, asking only about licensing/cost/security or explicit policy.
- **Author one section at a time.** Write `architecture.md` and `plan.md` section by
  section, pausing after each. For an open design choice or uncertain estimate, use a
  relevant technique from
  [`elicitation.md`](../devrites-lib/reference/standards/elicitation.md) (Tournament for two viable
  designs, Delphi for the estimate) before it hardens into slices.
- **Derive the slice count from the work.**
  One per independently-shippable increment, sized by `slicing.md`, every acceptance
  criterion mapped to ≥1 slice. A user-named count is a hint at most: slice logically and,
  if your honest count differs, present it and why. Never pad or compress to hit a figure.
  (`.devrites/AFK` `max_slices` is a separate AFK iteration budget, not the decomposition.)
- **Complexity does not shrink accepted scope.** Split/reorder for context, size,
  or dependencies; MUST NOT drop/defer approved REQ/AC. Reduction requires Spec
  Drift Guard + recorded human decision; hard/large is a decomposition signal.
- **Wide mechanical refactors slice expand → migrate → contract.** If one repeated change
  crosses many files, don't fake vertical slices. Add a compatibility/adapter slice, migrate
  small green batches, then remove the old path. If a batch cannot stay green, use an
  integration branch plus a final verify slice.
- **Root writes; drafter proposes.** Use the native bounded fresh-context contract in
  [`agents.md`](../devrites-lib/reference/standards/agents.md). The root owns architecture
  decisions, human questions, approval, and every canonical artifact write.

## Workflow
0. **Read `.claude/skills/devrites-lib/reference/standards/core.md`:** the always-on operating rules and anti-rationalizations.
   Resolve the active slug from `.devrites/ACTIVE`, require its `state.md`, and
   re-open `spec.md` and apply `spec-grammar.md`'s Native grammar re-read
   checklist. If there is no active workspace, no `spec.md`, the checklist
   fails, or its readiness gate hasn't passed →
   **STOP** and tell the user to run `/rite-spec <feature>` first.
   If `decision-coverage.md` is absent or does not say `Decision coverage: CLEAR`,
   **STOP** → `/rite-clarify`.
1. **Read the spec:** `spec.md` (objective, requirements, acceptance, **placement**,
   design references, gaps/decisions), plus `references.md`, `decisions.md`,
   `assumptions.md`, `decision-coverage.md`, **`strategy.md` if present** (the scope mode, deferred / out-of-scope
   register, and pre-mortem risks from `/rite-temper`: cut slices to mitigate the top risks
   and respect the IN/OUT line; map coverage against the **hardened** spec), and
   **`design-brief.md` if the feature touches UI** (the UX/UI contract `/rite-spec` shaped:
   its key states, interaction model, and proof targets drive how UI slices are cut). If a blocking
   `[NEEDS CLARIFICATION]` remains, stop → `/rite-clarify`.
1a. **Draft from fresh context.** Freeze the planning inputs and dispatch
   `devrites-plan-drafter` in `define` mode for one atomic candidate bundle:
   `architecture.md`, `plan.md`, `tasks.md`, and `traceability.md`, including proof mapping.
   Wait for and validate its bounded result. The drafter does not write or ask; any human-owned
   choice returns to this root context.
2. **Reconcile and decide the architecture + approach** (the HOW the spec deliberately
   omitted). Validate the candidate against live seams and the following rules; the root writes
   accepted content at step 6. Shape
   `architecture.md` for owning layer, boundaries, integration points, data/API/events,
   dependencies, risks, and affected areas; write only the build strategy in `plan.md`.
   Use a
   code-intelligence index if available (see
   `.claude/skills/devrites-lib/reference/standards/tooling.md`) for structure/impact; for the current API or behaviour of
   an external library/framework the architecture will rely on, consult context7 if available.
   Record significant options as `DEC-###`. For high-cost/hard-to-reverse boundaries, data
   models, public contracts, or dependencies, compare ≥2 viable approaches by drivers,
   trade-offs, and consequences. Specify cross-boundary interfaces for independent work:
   invariants, I/O, ordering/idempotency, errors, versioning, config, and relevant budgets.
   For any changed provider/consumer boundary, complete `plan.md`'s canonical
   `Shared contract proof` table with one reused contract artifact and provider- and
   consumer-side asserting tests that both consume it. Otherwise record the exact justified
   no-impact statement. Missing, one-sided, duplicated-contract, vague, or non-consuming proof blocks.
   **Deep-module check:** while sketching the major modules, look for opportunities
   to extract a **deep module**: a small, stable interface that hides a meaningful chunk
   of behavior and is independently testable. A *shallow* module whose interface is
   nearly as complex as its implementation adds no value; deepen it or delete
   it. Put independently testable deep-module behavior in the slice's `Tests/proof`;
   `/rite-vet` confirms the level.
2a. **Foreseeable-decision sweep.** Inspect questions, assumptions, architecture, dependencies,
   proof prerequisites, and proposed checkpoints. Search facts and decide reversible technical
   calls. New product/acceptance/policy/irreversible-risk gaps return to `/rite-clarify`; a build
   checkpoint survives only for unavailable pre-code evidence or mandatory action-time approval.
   **Completion:** no known implementation choice is postponed for build to ask later.
3. **Create vertical tasks:** each delivers one independently verifiable, observable
   capability end to end; first is the thinnest useful path, ordered by dependency then
   risk. Apply the slice-count and broad-refactor rules above plus
   `rite-plan/reference/slicing.md` and `rite-plan/reference/task-breakdown.md`.
   Mark per slice: **Frontend craft required**
   and **Browser proof required** (UI), and whether it's **fullstack** (FE+BE → contract
   first, see `devrites-frontend-craft/reference/fullstack.md`). **For UI slices, name which
   of `design-brief.md`'s key states + interaction the slice delivers, and give it a binary
   **Visual acceptance** target (state × viewport × input + target R-id/brief rule)**, so
   the design contract maps to slices as well as acceptance criteria.
   `Tests/proof` names exact command, cwd, expected signal, prerequisites, and mutable
   provenance inputs; `/rite-vet` preflights them. Write the portable repository command,
   never RTK/local wrappers, user-specific absolute paths, or temporary proof trees. When a
   shared contract changes, order its canonical artifact before both asserting tests and make
   the provider/consumer slice dependencies explicit.
4. **Map coverage and wiring:** every `AC-###` spec acceptance criterion maps to ≥1 `SLICE-###`
   (`rite-spec/reference/acceptance-criteria.md`); no orphaned criteria, no slice without a
   criterion. Lift covered/backstop `Edge Coverage` rows and resolved `Prohibitions (must-NOT)`
   rows into `traceability.md` and `test-plan.md`; unresolved rows get a gate/owner. Each
   cross-slice boundary names producer, consumer, invariant, integration step, and proof.
4a. **Persist traceability natively.** The drafter proposes and root writes
   `traceability.md` (`AC/REQ ID → slice → proof → evidence ID → files → status`).
   Re-read spec, tasks, and proof fields: every ID must exist and every mapping
   preserve meaning, not just labels. Reject orphans, inventions, and false mappings;
   `/rite-prove` and `/rite-seal` read this file directly. Reference the plan's
   `Shared contract proof` rows from existing slice/proof mappings; do not create a second
   traceability system.
5. **Complexity and deviations gate:** justify anything outside DevRites defaults (new dep,
   extra abstraction, second design system) in the plan; if you can't justify it, simplify.
   **Principles conformance:** read `.devrites/principles.md` (if present) and confirm the
   approach honors every declared invariant. A plan that conflicts with one is not "a deviation
   to justify away": either reshape the approach to conform, or, when the conflict is genuine and
   intended, route it through the Spec Drift Guard plus a recorded decision and a scoped principle
   exception a human approves. Never ready a plan that silently violates an invariant. (Re-scored
   as a blocking gate at `/rite-vet`; no file → none declared → nothing to check.)
6. **Write** `architecture.md`, `plan.md`, `tasks.md`, and `traceability.md`; update
   `state.md` (phase: plan → next `/rite-vet`).
6a. **Cross-artifact gate.** Read spec, tasks, and traceability together: every
   buildable AC/REQ maps to an existing slice/proof, every slice maps to real
   acceptance, and names/prose agree. Missing, duplicate, contradictory, or
   meaning-changing mappings block.
7. **Readiness gate** (plan-template): require CLEAR coverage, complete acceptance and
   cross-slice wiring/proof, complete `Shared contract proof`, risk-first acyclic order,
   justified deviations, rollback, and a
   closed decision sweep. **Stop and confirm** before code. Render the review-before-code
   digest first: `Intent` (one sentence from the spec), `Done means` (acceptance coverage x/y),
   `Plan sanity` (slice count + riskiest boundary/gate), `Expected build interruptions`
   (`none` or only justified action-time gates), and `Build exactly this?` (yes → approve;
   no → `/rite-plan revise`). When the human confirms the plan, write `Plan approved: <iso>` to
   `state.md` (see [state-workspace](../rite-spec/reference/state-workspace.md)); `/rite-build`
   checks this exists before building.

## tasks.md slice format

Use the canonical slice grammar in
[`workspace-artifact-schema.md`](../devrites-lib/reference/workspace-artifact-schema.md#canonical-slice-grammar).
Every slice must satisfy that complete field set; phase-specific gate details live in
[`reference/gates.md`](reference/gates.md).

> **Mid-flight discipline.** Do not skip vertical slicing, coverage mapping, or
> dependency ordering. See [`anti-patterns`](reference/anti-patterns.md).
