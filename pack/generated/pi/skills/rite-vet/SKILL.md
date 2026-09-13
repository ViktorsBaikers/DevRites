---
name: rite-vet
description: Review a defined engineering plan before code. Use for plan vetting or lock-in; not for implementation review or final readiness sealing.
argument-hint: "[slug] [--cross-model] [--full]"
user-invocable: true
---

# /rite-vet: review the plan before build

Vet every plan's scope, architecture, quality, proof, performance, failures and writer
safety. Fold cited technical findings; design Build tests. Temper owns product;
Vet owns implementation; current `$ARGUMENTS` (`--full`) feeds the depth triggers in
`reference/depth.md`; profiles never remove the exact plan-reviewer gate
([orchestration-profiles.md](../devrites-lib/reference/orchestration-profiles.md)).

## Rules

Read the active standard from: [`principles.md`](../devrites-lib/reference/standards/principles.md), [`patterns.md`](../devrites-lib/reference/standards/patterns.md), [`coding-style.md`](../devrites-lib/reference/standards/coding-style.md),
[`testing.md`](../devrites-lib/reference/standards/testing.md), [`spec-grammar.md`](../devrites-lib/reference/standards/spec-grammar.md), [`performance.md`](../devrites-lib/reference/standards/performance.md), [`error-handling.md`](../devrites-lib/reference/standards/error-handling.md),
[`development-workflow.md`](../devrites-lib/reference/standards/development-workflow.md), [`afk-hitl.md`](../devrites-lib/reference/standards/afk-hitl.md), [`one-shot-actions.md`](../devrites-lib/reference/standards/one-shot-actions.md),
[`developer-experience.md`](../devrites-lib/reference/standards/developer-experience.md), [`elicitation.md`](../devrites-lib/reference/standards/elicitation.md), and [`definition-of-done.md`](../devrites-lib/reference/standards/definition-of-done.md). Load
repository topology, data integrity, and integration reliability only when
triggered. Before classifying any Reslice, read `.pi/skills/devrites-lib/reference/standards/acceptance-preserving-reslice.md`.
When a
plan declares a root-authored executable workflow file, read
[`workflow-artifacts.md`](../devrites-lib/reference/standards/workflow-artifacts.md).

<!-- BEGIN RESLICE ROUTE-TO-ACTION -->
- `FOLD` → fold technical topology; invalidate Vet/readiness; affected Vet before Build.
- `GUARD_AND_REPAIR` → no planning writes; Spec Drift Guard → Clarify → Plan repair → affected Vet.
- `BLOCKED_INPUT` → no planning writes; exact diagnostic; recover input; reclassify.
<!-- END RESLICE ROUTE-TO-ACTION -->

## Invariants

- Review implementation, not ambition. Challenge creep, complexity, proof, and
  risk without changing accepted product scope.
- Root owns decisions/writes/readiness; reviewers judge without route policy.
  Cite findings/confidence; suppress
  unverified or confidence ≤4 findings under `review-axes.md`.
  Lens arrows in `eng-lenses.md` are heuristics; band findings only under the
  four `review-axes.md` names.
- Auth, migration, public API, and data-model changes use maximum caution and the
  irreversible-risk stop. Project principles never become trade-offs.
- **Governance-protected paths** (`.devrites/**`, pack skill/agent trees,
  `NOTICE.md` generator regions, CI/hook config named in repo docs) require explicit
  human approval before plan slices may edit them. A slice touching a protected path
  without approval → Vet **NEEDS CLARIFICATION**.
- Use the lowest axis band; never average or round thin to ready. Search before
  asking and resolve reversible technical choices. Ask only human-owned choices.
- Preserve a valid technical return cursor. Agent-owned `NEEDS REPLAN` returns
  internally to its caller, not to the human; on a direct invocation there is
  no caller, so Vet becomes it — save a return cursor
  (`return_phase`/`return_next_action`) naming this Vet pass as the caller and
  invoke `/rite-plan` repair inline under the same fingerprint caps. An
  agent-owned verdict with budget remaining is never a user-facing command.
- **Recovery recheck is bounded and batched.** A valid cursor plus open
  fingerprints enters Recovery recheck; it does not start another Full Vet or
  repeat unaffected axes/reviewers. One dispatch covers every open fingerprint
  of the fold — each checked individually — with a mandatory correction-created
  regression audit.
<!-- workflow-artifact-adapter: {"module":"devrites-lib/reference/standards/workflow-artifacts.md","entry":"plan declares root-authored executable workflow file","action":"emit exact admission; stale/missing authority uses PLAN_VET_REPAIR","return":"Vet READY cursor or exact technical replan"} -->
## Workflow

0. **Orient.** Read core. Resolve active slug, require state, and read plan,
   tasks, spec, decision coverage, optional strategy/design brief, decisions,
   assumptions, and state. Require approved Plan and `Decision coverage: CLEAR`;
   otherwise stop for Define or Clarify. Use code intelligence for placement,
   blast radius, and reuse.
1. **Select depth.** Apply `reference/depth.md` exactly; never skip. Every initial
   pass records an engineering verdict and test-plan coverage. A valid Recovery
   recheck retains prior depth and enters 1b.
1a. **Independent initial pass.** Freeze candidate and dispatch the exact fresh
   read-only plan reviewer — plus developer-experience reviewer for developer
   surfaces and the current strategy reviewer after significant Temper — in
   parallel under `../devrites-lib/reference/parallel-dispatch.md`; they are
   independent read-only passes on one frozen candidate. Missing
   required account blocks.
1b. **Recovery recheck.** Require valid return cursor, accepted prior finding,
   exact fingerprint/reproduction, repaired candidate identity, changed
   paths/criteria, affected drift/evidence, and the recorded delta self-check.
   Freeze that packet and dispatch each exact owning reviewer once,
   fresh/read-only, limited to it — all its open fingerprints in that dispatch.
   Do not rerun broad inventory or unaffected reviewers. Close the prior
   fingerprint only with discriminating evidence plus an explicit
   correction-created regression verdict on the touched invariants (acyclic
   deps, traceability, contract proof, sizing) and every region depending on a
   changed clause — enumerate dependents via the dep graph and traceability
   refs, not only patched sites. Otherwise record one no-progress outcome. A
   different Critical/Important invariant needs exact evidence and a new
   fingerprint; a Suggestion, Nit, or FYI cannot keep recovery open. Reviewer
   `Late:` rows on unchanged clauses follow the canonical
   [late-finding rule](../devrites-lib/reference/standards/afk-hitl.md#retry-cap-no-progress-loops-and-self-resolve):
   Critical with a failure path → new fingerprint; otherwise `eng-review.md`
   `## Deferred findings` (mechanism rows become one proof row at step 6). A packet
   marked `convergence_pressure` accepts an explicit, bounded declared
   residual/limitation folded into the contract text as closure for a finding
   marked as a refinement of an already-pinned clause; the recheck may re-raise
   it only by naming a concrete failure path the declaration omits. A
   `new_failure_mode` finding is never residual-eligible. A fold whose
   findings are all Suggestion-or-lower rechecks mechanically at the
   materialization gate — anchors applied, mirrors consistent, invariants
   pass — without a dispatched reviewer; sub-Important findings already
   cannot keep recovery open. Reconcile
   shared artifact/readiness gates, then enter step 8 or the next repair.
   Do not fall through to steps 2–7. Retain evidenced unchanged coverage; changed
   global contract/uncertain impact requires full applicable review. Reopen historical
   findings only for affected invariants/dependencies.
2. **Challenge scope.** Apply review-axes §0 and search accepted decisions.
   Harden to the smallest contract-complete plan, using marked topology action.
   Then verify bidirectional ID-and-meaning traceability across spec/plan/
   tasks/test-plan/traceability, acceptance, terms, principles, anti-slop,
   and conventions; every slice and test-plan row maps to a live requirement
   and vice versa. Critical gaps and
   unexcepted principle breaches block; record the challenge result in `eng-review.md`
   §2 (Scope challenge) after recheck.
3. **Preflight Build entry.** Under `reference/artifacts.md`, verify exact
   command/cwd/tool/version/prerequisite; output filters must preserve upstream
   failure. Verify dependencies from authoritative source plus nearest manifest.
   Run parser-sensitive syntax only in isolated fixtures. Remeasure mutable facts;
   live evidence wins, conflict marks stale, and unmeasurable conflict is a gap.
   Record complete SHA-256 provenance. Every behavioral mapping names a positive
   discriminating assertion and decisive signal, never only exit zero.

   For each consumptive action, bind every `reference/artifacts.md` Consumptive
   action gates column; every fingerprint identifies one actionable seam; aliasing multiple
   emit sites is a gap. Preflight observes but need not make future behavior pass.
4. **Audit readiness.** Goal-backward map every requirement, criterion, NFR,
   interaction, edge/prohibition, and decision row to one slice and executable
   proof. Verify UX/spec/architecture alignment, contracts, dependency order,
   slice independence/wiring, prerequisites, failure/observability/rollback, and
   ownership. The plan's `Shared contract proof` names one reused boundary
   artifact plus two consuming tests for every changed API/event/schema/provider-
   consumer seam, or an explicit no-impact statement. Missing, one-sided,
   duplicated-contract, vague, or non-consuming proof fails closed.

   Technical gaps are `NEEDS REPLAN` and Plan repair — each finding marks
   itself `new_failure_mode` or names the already-pinned clause/fingerprint
   it refines, making alias-detection and `convergence_pressure` eligibility
   mechanical — and carries `kind: contract | mechanism`
   (`review-axes.md` § Finding kind). Only `contract` findings open a Plan
   repair; a `mechanism` finding is closed by one `test-plan.md` proof row
   at step 6 and leaves readiness intact when nothing else is open.
   Product/risk gaps are
   `NEEDS CLARIFICATION` and Clarify. Neither becomes a Build qid.
5. **Review axes.** Apply `review-axes.md` through `eng-lenses.md`. Fold verified
   behavior-preserving technical findings; walk only human-owned decisions.
   Profile gate ceiling and Reslice marked action remain authoritative.
6. **Write outputs.** Produce every artifact in `reference/artifacts.md`. After
   editing intent/decision/assumption/question owners, re-scan affected coverage,
   assumptions, uncertainty, and gates. Keep state non-READY. Every scenario and
   criterion needs positive, discriminating proof; every slice must be one-pass
   implementable; developer plans need a predicted scorecard. Durable commands
   are portable repository commands, not host wrappers.
7. **Narrow recheck after edits.** Dispatch the exact plan reviewer once per
   fold — one dispatch covering every open fingerprint, preserving
   per correction/fingerprint accounting (each checked individually with its
   accepted findings, changed paths/criteria, new identity, and recorded delta
   self-check); it must return an explicit correction-created regression
   verdict over touched invariants plus every dependent of each changed clause
   (dep-graph and traceability lookup), not only fingerprint closure. Within one correction, no broad
   third loop. If it changes plan, fold again. A closed input plus a distinct Critical/Important invariant returns
   that new fingerprint as progress. Then close matrix and rerun ID/meaning audit.
8. **Build readback and readiness.** Add a cited five-line readback to
   `eng-review.md` (artifacts.md §7 rows 1–5): outcome/ACs; IN/OUT/must-NOT; UI direction and architecture/
   critical flow; slice order/first slice; decisive proof/action-time gates. No implementer should need to invent product, architecture, or proof.
   Contradiction, ownerlessness, or material ambiguity blocks via Clarify or Plan.

   Write exactly one `Implementation readiness: READY`, `NEEDS CLARIFICATION`,
   or `NEEDS REPLAN`. Root sets READY after every account, checklist,
   preflight, and sweep is green. Write phase/next step and emit one
   `Readiness inputs SHA-256` with
   `devrites-engine check readiness --emit-binding <slug>`; normal readiness check
   must pass. Technical failure records reproduction, not qid. Human gap awaits
   Clarify. Optional cross-model follows `reference/cross-model.md`.

   With READY, no pending remediation, and a valid technical return cursor,
   restore and consume the return cursor instead of defaulting to Build. Preserve
   it through admitted remediation. Only a real stop reaches the human.
9. **Stop at the Vet boundary.** Show Build readback, scope verdict, lowest axis,
   closed gaps, preflight, action checkpoints, and critical failures. Recommend
   Build only when READY.

> Do not replace interactive review with artifacts, change acceptance through
> hardening, score without source evidence, or ignore unexplained complexity.

## Phase exit

**Complete when:** `eng-review.md` records exactly one readiness verdict, readiness
binding SHA-256 passes, and every required reviewer account is admitted.

**Failing case:** READY written while a required reviewer returned `Outcome: gap` →
not complete; restore NEEDS REPLAN or dispatch missing reviewer.
