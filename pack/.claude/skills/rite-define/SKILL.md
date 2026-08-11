---
name: rite-define
description: Define the first build plan from an approved spec: architecture, task slices, traceability, and state. Use when turning approved intent into its initial plan.
argument-hint: "[feature-slug]"
user-invocable: true
---

# /rite-define: plan from the spec

Turn `spec.md` into architecture, approach, dependency-ordered **vertical slices**,
traceability, and a state cursor. Spec owns what/why; Define owns how in phase-owned
artifacts. **Do not write code.**

## Rules consulted (read on demand from `.claude/skills/devrites-lib/reference/standards/`)
Pull these via `Read` when shaping the plan:
- `development-workflow.md`: small batches, trunk-always-green, definition of done.
- `principles.md`: the project invariants (`.devrites/principles.md`) the chosen approach must conform to.
- `documentation.md`: record plan-time decisions and rationale.
- `repository-topology.md`, `data-integrity.md`, and `integration-reliability.md`:
  load only for matching `spec.md` applicability rows; each applicable owner is mandatory.
- `../workspace-artifact-schema.md`: artifact purposes, budgets, IDs, and read triggers.

## Operating rules
- **Requires a readied spec.** Missing active workspace/spec/readiness, or an open CRITICAL
  spec checklist, stops to `/rite-spec <feature>`. Missing/non-`CLEAR`
  `decision-coverage.md` routes to `/rite-clarify`. Never plan unready/unclarified work.
- Apply `afk-hitl.md` ownership. Prefer conventions; source-check new dependencies/design
  systems, asking only about licensing, cost, security, or policy.
- **Author one section at a time.** Pause after each `architecture.md`/`plan.md` section.
  For an open design choice or uncertain estimate, use
  [`elicitation.md`](../devrites-lib/reference/standards/elicitation.md) (Tournament for two viable
  designs, Delphi for estimates) before slicing.
- **Derive the slice count from the work.**
  One per independently-shippable increment, sized by `slicing.md`; map every acceptance
  criterion. User counts are hints: explain honest differences, never pad/compress.
  `.devrites/AFK` `max_slices` is an iteration budget, not decomposition.
- **Complexity does not shrink scope.** Split/reorder; never drop/defer approved REQ/AC.
  Reduction needs Drift Guard + human decision; hard/large means decompose.
- **Wide refactors use expand → migrate → contract.** Add compatibility, migrate green
  batches, then remove the old path; if batches cannot stay green, use an integration branch
  plus final verify.
- **Root writes; drafter proposes.** Use the native bounded fresh-context contract in
  [`agents.md`](../devrites-lib/reference/standards/agents.md). Root owns decisions,
  questions, approval, and canonical writes.

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
   Reconcile every `Applicability map` row against live repository evidence. Load each
   applicable standard; a false `not applicable` is a blocking spec gap, not a planning
   shortcut.
1a. **Draft from fresh context.** Freeze the planning inputs and dispatch
   `devrites-plan-drafter` in `define` mode for one atomic candidate bundle:
   `architecture.md`, `plan.md`, `tasks.md`, and `traceability.md` with proof mapping.
   Validate its bounded result; it does not write/ask, and returns human choices to root.
2. **Reconcile architecture + approach** against live seams; root writes accepted content
   at step 6. Shape
   `architecture.md` for owning layer, boundaries, integration points, data/API/events,
   dependencies, risks, and affected areas; write only the build strategy in `plan.md`.
   Use a code index for structure/impact (see
   [`tooling.md`](../devrites-lib/reference/standards/tooling.md)); source-check current external
   library/framework behavior, using context7 when available.
   Record significant options as `DEC-###`. For high-cost/hard-to-reverse boundaries, data
   models, public contracts, or dependencies, compare ≥2 viable approaches by drivers,
   trade-offs, and consequences. Specify cross-boundary interfaces for independent work:
   invariants, I/O, ordering/idempotency, errors, versioning, config, and relevant budgets.
   Establish repository/deployable roots, state and contract ownership, shared mutable
   resources, and deployment order under `repository-topology.md`. For applicable durable
   data or integration rows, include the exact required plan table from `data-integrity.md`
   or `integration-reliability.md`; do not replace it with "handle retries/migration" prose.
   For any changed provider/consumer boundary, complete `plan.md`'s canonical
   `Shared contract proof` table with one reused contract artifact and provider- and
   consumer-side asserting tests that both consume it. Otherwise record the exact justified
   no-impact statement. Missing, one-sided, duplicated-contract, vague, or non-consuming proof blocks.
   **Deep-module check:** prefer small, stable interfaces hiding meaningful, testable
   behavior. Deepen/delete shallow modules whose interface matches implementation
   complexity. Put their independent behavior in `Tests/proof`; `/rite-vet` confirms.
2a. **Decision-horizon sweep.** Apply `reference/plan-template.md`'s classification to every
   question, assumption, architecture/dependency/proof choice, and checkpoint. Resolve
   planning items from source; only necessary executable evidence warrants a risk spike with
   discriminating criteria/fallback branches. Human blockers route to `/rite-clarify`.
   Implementation-local deferral needs owner slice, observable trigger, bounds/fallback,
   and resolution proof; action checkpoints name their signal. Preserve entries until
   evidence resolves/supersedes them. **Completion:** no blocker or unowned/unsupported
   deferral; deferral is not “ask later.”
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
   Map each applicable topology/data/integration risk to a slice, failure/recovery path,
   and discriminating proof; a risk cannot live only in the architecture narrative.
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
7. **Readiness gate** (`plan-template.md`): require CLEAR coverage; complete acceptance,
   wiring, shared-contract proof, applicable outputs, and rollback; risk-first acyclic order;
   justified deviations; and every horizon item present, with blockers resolved, planning
   items resolved/validly spiked, and local/action entries bounded and owned.
   **Stop and confirm** before code. Render the review-before-code
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
