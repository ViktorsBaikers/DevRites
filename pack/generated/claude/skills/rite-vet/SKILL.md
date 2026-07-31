---
name: rite-vet
description: Review a defined engineering plan before code. Use for plan vetting or lock-in; not for implementation review or final readiness sealing.
argument-hint: "[slug] [--cross-model] [--full]"
user-invocable: true
---

# /rite-vet: review the plan before build

Vet every plan before code for scope, architecture, quality, proof, performance,
failure modes, and parallel safety. Cite findings, fold accepted technical hardening
into the plan, and design Build tests. Risk sets depth; `/rite-autocomplete` has
Vet and `--cross-model` adds one opinion. Missing `plan.md` routes to `/rite-define`.

`/rite-temper` owns product scope/strategy; Vet owns implementation. Quick/Standard/Full
comes from [`orchestration-profiles.md`](../devrites-lib/reference/orchestration-profiles.md),
but every profile keeps the exact plan-reviewer gate.

## Rules consulted (read on demand from `.claude/skills/devrites-lib/reference/standards/`)
Pull the standard named by the active axis: `principles.md`, `patterns.md`,
`coding-style.md`, `testing.md`, `spec-grammar.md`, `performance.md`,
`error-handling.md`, `development-workflow.md`, `afk-hitl.md`,
`developer-experience.md`, `elicitation.md`, and `definition-of-done.md`.


## Operating rules
- **Review implementation, not ambition.** Temper's product scope is settled; challenge
  only implementation creep, complexity, proof, and risk.
- **Root hardens; reviewers judge.** Root alone asks, decides, folds back, writes, and sets
  readiness. Write behavior-preserving test, boundary, ordering, error, and failure-mode
  refinements into `plan.md`/`tasks.md`/`test-plan.md`. Acceptance/product changes require
  the Spec Drift Guard: `drift.md`, recorded decision, then `/rite-plan repair` for reslicing.
  Build scope never grows without a recorded human decision.
- **Support every finding.** Cite its plan/spec/code source and confidence. Unverified or
  confidence ≤4 stays suppressed per [`review-axes.md`](reference/review-axes.md).
- **Apply maximum caution to hard-to-reverse changes.** Auth, migration, public API, and
  data-model changes always pause under the irreversible-risk list.
- **Use the lowest axis band.** Never round `thin` up to `ready`; do not average the
  axes. Record the reason for every decision.
- **Search before asking.** Verify facts and fold reversible technical hardening into the
  plan; ask only human-owned choices under `afk-hitl.md`. Dispatch uses the bounded
  [`agents.md`](../devrites-lib/reference/standards/agents.md) contract.

## Workflow
0. **Read `.claude/skills/devrites-lib/reference/standards/core.md`** first.
   Then resolve the active slug from `.devrites/ACTIVE`, require its
   `state.md`, and read the workspace: `plan.md`, `tasks.md`, `spec.md`,
   `decision-coverage.md`
   (for intent + acceptance), `strategy.md` (if `/rite-temper` ran), `decisions.md`,
   `assumptions.md`, `design-brief.md` (if UI), `state.md`. Require a `plan.md` whose
   Readiness gate passes (or `Plan approved`): else STOP → `/rite-define`. Require
   `Decision coverage: CLEAR`: else STOP → `/rite-clarify`. Prefer a
   code-intelligence index if available (see
   `.claude/skills/devrites-lib/reference/standards/tooling.md`) for placement / blast-radius / reuse checks.
1. **Set review depth. Never skip this step.** Apply
   [`reference/depth.md`](reference/depth.md) exactly. Every plan leaves a recorded
   engineering verdict and `test-plan.md` coverage map.
1a. **Independent pass at every depth.** Freeze the candidate; dispatch exact
   `devrites-plan-reviewer` fresh/read-only and validate its report. Add exact
   `devrites-devex-reviewer` for developer surfaces and require the current exact
   `devrites-strategy-reviewer` verdict after significant Temper. Missing accounts
   block; never substitute inline work.
2. **Scope challenge (blocking gate):** apply §0 of
   [`reference/review-axes.md`](reference/review-axes.md). Search accepted ADRs and
   relevant workspace `decisions.md` files directly. Harden to the smallest
   behavior-preserving plan; ask only when that changes acceptance or explicit architecture policy.
2a. **Cross-artifact/project gates.** Apply explicit checklists to spec, tasks,
   and traceability. Every AC/REQ maps by ID and meaning to a real slice/proof;
   every slice maps back. Check terminology/conflicts, principles, anti-slop, and
   conventions. Critical blocks; principle exceptions are human-owned. Recheck
   after hardening and write `analysis.md`.
2b. **Build-entry preflight.** Using [`reference/artifacts.md`](reference/artifacts.md), verify
   every exact proof command/cwd/tool/version/prerequisite; package names against authoritative
   source and nearest manifest/lockfile; parser-sensitive syntax in an isolated fixture;
   applicable UI/browser harnesses. Remeasure decision-bearing counts/versions/state
   claims read-only: live facts win; conflicts mark stale artifacts;
   unmeasurable conflict = gap. Record complete SHA-256 provenance inputs. Require
   every behavioral mapping to name a positive,
   discriminating assertion and decisive signal, not merely a command or expected exit zero.
   Preflight observes; it need not make future behavior pass.
2c. **Implementation-readiness audit.** Goal-backward map every REQ/AC/NFR, interaction,
   edge/prohibition, and decision-coverage row to a slice and executable proof. Verify
   UX/spec/architecture alignment, producer-consumer contracts, slice independence/order and
   wiring, exact prerequisites, failure paths, operations, observability, rollout, and rollback.
   Check `plan.md`'s canonical `Shared contract proof`: every changed API/event/schema or other
   provider/consumer boundary needs one reused artifact and two asserting tests that consume it;
   no boundary change needs the specific no-impact statement. Missing, one-sided, duplicated-contract,
   vague, or non-consuming proof fails closed.
   Fold technical fixes into the plan. Product/risk gaps are `NEEDS CLARIFICATION` →
   `/rite-clarify`; technical plan/preflight gaps are `NEEDS REPLAN` → `/rite-plan repair`.
   Neither becomes a build qid.
3. **Review four axes:** apply [`reference/review-axes.md`](reference/review-axes.md)
   through [`reference/eng-lenses.md`](reference/eng-lenses.md). Fold verified,
   behavior-preserving technical findings into the plan. Walk only human-owned decisions with
   the human, one coherent option set at a time. The AFK ceiling remains owned by
   [`reference/depth.md`](reference/depth.md): scope/acceptance changes and irreversible risk pause.
4. **Required outputs:** prepare every shape and fold-back in
   [`reference/artifacts.md`](reference/artifacts.md), using the review rules in
   [`reference/review-axes.md`](reference/review-axes.md). Derive dependency order
   from the plan; let the native host schedule independent read-only work while
   keeping source writers serial.
   Completion: every scenario and acceptance criterion maps to planned positive,
   discriminating proof, every slice is
   one-pass implementable, the Build-entry preflight is green or names an owned prerequisite,
   and developer-facing plans have a predicted `devex.md` scorecard. Durable proof commands
   are portable repository commands; host-local wrappers belong only in observed
   recorded execution evidence.
5. **Write and fold back every artifact required by
   [`reference/artifacts.md`](reference/artifacts.md).** Route every
   acceptance/behavior-changing delta through the **Spec Drift Guard** (`drift.md` +
   recorded decision + `/rite-plan repair`). After any edit to
   `brief.md`, `spec.md`, `decisions.md`, `assumptions.md`, or `questions.md`, re-scan the
   affected coverage rows, assumption audit, residual uncertainty, and closed gates.
   Partial/Missing, an unowned material assumption, or an open blocking/escalating question is
   `NEEDS CLARIFICATION` → `/rite-clarify`/HITL; never refresh past it.
   After each fold-back rerun step 2a; missing/meaning-changing mappings block.
   Keep `state.md` non-READY for step 6.
6. **One narrow recheck after edits.** If the candidate changed, dispatch exact
   `devrites-plan-reviewer` once with accepted findings, changed paths/criteria,
   and new identity. No full or third loop; if it changes the plan, repeat step 5.
   Then close the matrix and rerun step 2a's ID-and-meaning audit.

6a. **Build readback.** From the final owning artifacts, add a cited five-line
   readback to `eng-review.md` §7: outcome/ACs; IN/OUT and must-NOT boundaries;
   UI direction when applicable plus chosen architecture/rationale and
   critical happy/error flow; slice order and first slice; decisive proof and
   justified action-time gates. Ask whether a fresh implementer could explain and
   build it without inventing a product, design, architecture, or proof decision.
   Contradictory, ownerless, or materially ambiguous statements block READY:
   human-owned behavior, policy, or acceptance returns to `/rite-clarify`;
   technical architecture,
   slicing, or proof returns to `/rite-plan repair`.
   The readback is a derived view and MUST NOT override its cited owners.

   Write one typed field to `eng-review.md`: `Implementation readiness: READY`,
   `NEEDS CLARIFICATION`, or `NEEDS REPLAN`. Root alone sets READY after every
   exact account, checklist, proof preflight, and final sweep passes with no
   foreseeable human choice except a justified action-time checkpoint. Write
   `Phase: vet`, `Next step: /rite-build`, then emit the one exact
   `Readiness inputs SHA-256` line with `devrites-engine check readiness
   --emit-binding <slug>` only after this recheck. Normal `devrites-engine check
   readiness <slug>` must then pass; otherwise record the blocker. Technical failure
   records repro and `/rite-plan repair`, no qid; a human-owned gap routes
   `/rite-clarify` and awaiting-human.
   [`reference/cross-model.md`](reference/cross-model.md) owns the optional explicit
   cross-model integration.
   Completion: the final axis floor clears, an objective technical blocker is recorded, or a
   genuine human-owned gate is recorded.
7. **STOP.** Show the Build readback, scope verdict, lowest axis band, closed coverage
   gaps, preflight, action-time checkpoints, and failure-mode criticals; recommend
   `/rite-build` only when the entry contract is ready.

> **Mid-flight discipline.** Do not replace the interactive review with
> `eng-review.md`, change acceptance through plan hardening, score without source
> evidence, or ignore unexplained complexity. See
> [`reference/anti-patterns.md`](reference/anti-patterns.md).
