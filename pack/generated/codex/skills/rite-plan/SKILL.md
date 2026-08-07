---
name: rite-plan
description: Re-plan existing work when reality invalidates the plan: reslice a slice that is too big, repair drift, reorder dependencies, split boundaries, or unblock work. Not first-pass decomposition.
argument-hint: "[mode: decompose|reslice|repair|reorder|split|unblock|course-correct|revise]"
user-invocable: true
---

# $rite-plan: (re)plan an active feature

Update an existing plan when implementation evidence, drift, or a user decision makes it
wrong. **Read the active workspace first.** If `.devrites/ACTIVE` is empty or its workspace
is missing, stop and tell the
user to run `$rite-spec <feature>`. **Revise mode is artifact-only**: reconcile
`spec.md` / `architecture.md` / `plan.md` / `tasks.md` / `traceability.md` without
editing source code.

## Rules consulted (read on demand from `.agents/skills/devrites-lib/reference/standards/`)
Pull `development-workflow.md` via `Read` when reshaping slice cadence or DoD criteria.

## Operating rules
- Update the spec when needed, but never plan around a known-wrong assumption.
- If a change alters product behavior, scope, architecture, data model, UX, security,
  or migration risk → search facts first, then route the human-owned contract decision
  through `$rite-clarify` (using the Spec Drift Guard). Reversible technical repair is
  agent-owned and must not become a question.
- Keep each slice small enough for one focused build → prove cycle.
- **Slice count is derived, never dictated:** reslice when a slice fails the sizing rule
  (multiple "and"s, can't build+prove in one cycle), not to hit a user-named tally. A
  requested count is a hint at most; slice logically and explain if it differs. See
  [`reference/slicing.md`](reference/slicing.md) ("How many slices?").
- **Size by complexity, order by dependency.** A slice carries a `Complexity: N/5` score (from
  `$rite-define`); a slice scoring **>3** is a reslice trigger unless its inline reason justifies
  the irreducible complexity. Honor each slice's `depends_on:`: the next *buildable* slice is the
  lowest pending one whose dependencies are all built. This preserves one-slice-at-a-time execution.
- **Root writes; drafter proposes.** Follow
  [`agents.md`](../devrites-lib/reference/standards/agents.md). The controlling chat owns
  human questions, decisions, reconciliation, and all planning-artifact writes.
- **Nested repair preserves its caller.** When `state.md` contains a valid
  technical-backtracking return cursor, preserve any valid return cursor
  byte-for-byte. `$rite-vet` is the next internal prerequisite, not a command to
  hand back to the human.

## Workflow
0. Read `.agents/skills/devrites-lib/reference/standards/core.md` (operating rules) before reshaping anything.
   Then resolve the explicit or active slug, require its `state.md`, and read
   the cursor directly.
1. Read `spec.md`, `decision-coverage.md`, `plan.md`, `tasks.md`, `state.md`, `drift.md`,
   `eng-review.md`, and the current `git diff` (if a repo). Read `decisions.md` and
   `assumptions.md`. Require `Decision coverage: CLEAR`; otherwise STOP → `$rite-clarify`.
   If a code-intelligence
   index is available: `codebase-memory-mcp` first, cross-checked with `codegraph`
   (`.codegraph/` / `codegraph_*` tools) + `graphify` (`graphify-out/`), else standard methods
   (LSP / `Read`/`Grep`/`Glob`); see `.agents/skills/devrites-lib/reference/standards/tooling.md`:
   prefer it for structural questions (what calls X, what would
   changing Y break) over reading whole files, to keep planning context lean. For an external
   dependency's current API surface, consult context7 if available.
2. **Pick the mode** (`$ARGUMENTS` or infer):
   - **decompose:** first/again break the feature into vertical slices.
   - **reslice:** a slice is too large; split into thinner end-to-end slices.
   - **repair:** a Spec Drift Guard event; fold the resolution into plan + tasks.
   - **reorder:** fix the dependency order.
   - **split:** separate backend/frontend contracts (see `devrites-api-interface`).
   - **unblock:** a verification failed; re-route around the blocker.
   - **course-correct:** a deliberate mid-build *pivot* (the user changed their mind), distinct
     from accidental drift: classify the change, assess its impact across the remaining slices,
     decide rollback vs forward-fix, and update `spec.md` + `plan.md` + `tasks.md` + `decisions.md`
     atomically. An acceptance/behavior change still goes through the user first. When the plan
     names an `MVP cut`, offer it as the retreat option: falling back to the cut is a pre-agreed
     scope, not a new negotiation.
   - **revise:** apply a requested planning-artifact revision and reconcile existing artifacts in
     any direction; propose the file edit set first, confirm each file before writing, and **never
     edit source code**. The sole confirmation exception is explicit `$rite-upgrade` with an
     admitted `repairable` assessment naming current rule, evidence, gate, exact paths, and
     delta; it authorizes only that neutral workspace edit, never source or history.
     **Gate first: revise or new?** Same intent? More than 50% of existing scope
     survives? original *not* completable without this? Two "no"s → new work: recommend
     sealing/shipping the current workspace (MVP cut if named) then `$rite-spec` for the new
     intent, and stop. Revise preserves context; a new workspace separates the work.
   See [replan-and-repair](reference/replan-and-repair.md) for each mode's steps.
2a. **Draft the repair from fresh context.** Freeze the inputs and dispatch
   `devrites-plan-drafter` in `repair` mode with only the selected mode, affected artifact
   paths, settled contract, and observed failure/drift. Await one atomic `plan-candidate`;
   the drafter writes nothing and returns human-owned choices separately.
3. Reason about dependencies: [dependency-graph](reference/dependency-graph.md).
   Reconcile `plan.md`'s canonical `Shared contract proof`: changed provider/consumer
   boundaries keep one reused contract artifact ahead of both asserting tests, and unaffected
   plans retain the specific no-impact statement. Missing, one-sided, duplicated-contract, vague, or
   non-consuming proof routes to `$rite-vet` only after repair.
   **Completion:** the slice graph is cycle-free and every dependency names an existing slice.
4. Re-slice using vertical-slice rules: [slicing](reference/slicing.md) and
   [task-breakdown](reference/task-breakdown.md). Prefer thin, shippable, verifiable.
   **Completion:** every slice is independently shippable/provable or carries an irreducibility reason.
5. Reconcile the candidate against steps 3 and 4, then the root updates `plan.md`, `tasks.md`,
   `state.md`, and appends rationale to `decisions.md`.
   Any change to `architecture.md`, `plan.md`, `tasks.md`, or `traceability.md` invalidates
   the previous vet verdict: set `Phase: plan`, `Next step: $rite-vet`, and, when
   `eng-review.md` exists, set `Implementation readiness: NEEDS REPLAN`. Never retain READY
   across changed planning inputs. Preserve `Plan approved` only for behavior/acceptance-neutral
   technical repair; clear and reconfirm it when the contract changed. If you stopped for drift,
   mark the `drift.md` entry resolved. Never remove or overwrite a valid caller
   return cursor while writing the Plan checkpoint.
6. If product behavior/acceptance criteria change, confirm through `$rite-clarify` before
   writing, re-close `decision-coverage.md`, then reconcile the plan. After any edit to
   `brief.md`, `spec.md`, `decisions.md`, `assumptions.md`, or `questions.md`, including a
   behavior-neutral technical rationale appended to `decisions.md`, re-scan the affected coverage
   rows, assumption audit, residual uncertainty, and closed gates. Partial/Missing, an unowned
   material assumption, or an open blocking/escalating question routes `$rite-clarify`/HITL.
   Re-read every affected row and require current evidence before restoring `CLEAR`.
   **Completion:** the change is classified, and every behavior/acceptance change has explicit
   confirmation recorded before the artifacts are updated.
7. **Done when:** every slice is sized (builds + proves in one cycle; no slice scoring >3
   left unjustified), the dependency order is acyclic, every `drift.md` entry you stopped for
   is marked resolved, revised artifacts agree with each other, no source files changed in
   `revise` mode, behavior-change-vs-not is confirmed (`no`, or clarified), and every changed
   plan ends at `$rite-vet` rather than returning directly to build. The `Shared contract proof`
   table or justified no-impact statement must still match the revised boundary set.
   If any check fails, loop back: don't hand off a half-reshaped plan.

When invoked inline by a controlling rite, return the completed Plan checkpoint
to that caller so it can invoke Vet immediately. The phase boundary is not a
user-facing stop unless a genuine HITL, safety/access, or exhausted-recovery
condition was recorded.

> **Mid-flight discipline.** Do not change product behavior without confirmation or
> absorb drift silently. See [`anti-patterns`](reference/anti-patterns.md).
