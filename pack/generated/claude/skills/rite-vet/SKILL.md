---
name: rite-vet
description: Review a defined engineering plan before code. Use for plan vetting or lock-in; not for implementation review or final readiness sealing.
argument-hint: "[slug] [--cross-model] [--full]"
user-invocable: true
---

# /rite-vet: review the plan before build

Vet every plan before code for scope, architecture, quality, proof, performance,
failure modes, and writer safety. Cite findings; fold accepted technical
hardening into planning artifacts; design Build tests. Temper owns product
scope, Vet owns implementation; current `$ARGUMENTS` (`--full`) feeds the depth triggers in
`reference/depth.md`; profiles never remove the exact plan-reviewer gate
([orchestration-profiles.md](../devrites-lib/reference/orchestration-profiles.md)).

## Rules

Read the active standard from: `principles.md`, `patterns.md`, `coding-style.md`,
`testing.md`, `spec-grammar.md`, `performance.md`, `error-handling.md`,
`development-workflow.md`, `afk-hitl.md`, `one-shot-actions.md`,
`developer-experience.md`, `elicitation.md`, and `definition-of-done.md`. Load
repository topology, data integrity, and integration reliability only when
triggered. Before classifying any Reslice, read `.claude/skills/devrites-lib/reference/standards/acceptance-preserving-reslice.md`.
When a
plan declares a root-authored executable workflow file, read
`workflow-artifacts.md`.

<!-- BEGIN RESLICE ROUTE-TO-ACTION -->
- `FOLD` → fold technical topology; invalidate Vet/readiness; affected Vet before Build.
- `GUARD_AND_REPAIR` → no planning writes; Spec Drift Guard → Clarify → Plan repair → affected Vet.
- `BLOCKED_INPUT` → no planning writes; exact diagnostic; recover input; reclassify.
<!-- END RESLICE ROUTE-TO-ACTION -->

## Invariants

- Review implementation, not ambition. Challenge creep, complexity, proof, and
  risk without changing accepted product scope.
- Root alone asks, decides, folds, writes, and sets readiness. Reviewers judge;
  they add no route policy. Cite every finding and confidence; suppress
  unverified or confidence ≤4 findings under `review-axes.md`.
  Lens arrows in `eng-lenses.md` are heuristics; band findings only under the
  four `review-axes.md` names.
- Auth, migration, public API, and data-model changes use maximum caution and the
  irreversible-risk stop. Project principles never become trade-offs.
- **Governance-protected paths** (`.devrites/**`, pack skill/agent trees,
  `NOTICE.md` generator regions, CI/hook config named in repo docs) require explicit
  human approval before plan slices may edit them. A slice touching a protected path
  without approval → Vet **NEEDS CLARIFICATION**.
  **Failing case:** plan edits another feature's `state.md` without recorded approval →
  fail closed.
- Use the lowest axis band; never average or round thin to ready. Search before
  asking and resolve reversible technical choices. Ask only human-owned choices.
- Preserve a valid technical return cursor. Agent-owned `NEEDS REPLAN` returns
  internally to its caller, not to the human.
- **Recovery recheck is bounded.** A valid cursor plus open fingerprint enters
  Recovery recheck; it does not start another Full Vet or repeat unaffected
  axes/reviewers.
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
1a. **Independent initial pass.** Freeze candidate and dispatch exact fresh
   read-only plan reviewer. Add developer-experience reviewer for developer
   surfaces and current strategy reviewer after significant Temper. Missing
   required account blocks.
1b. **Recovery recheck.** Require valid return cursor, accepted prior finding,
   exact fingerprint/reproduction, repaired candidate identity, changed
   paths/criteria, and affected drift/evidence. Freeze that packet and dispatch
   each exact owning reviewer once, fresh/read-only, limited to it. Do not rerun
   broad inventory or unaffected reviewers. Close the prior fingerprint only
   with discriminating evidence. Otherwise record one no-progress outcome. A
   different Critical/Important invariant needs exact evidence and a new
   fingerprint; a Suggestion, Nit, or FYI cannot keep recovery open. Reconcile
   shared artifact/readiness gates and return to caller or next repair.
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

   Technical gaps are `NEEDS REPLAN` and Plan repair. Product/risk gaps are
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
7. **Narrow recheck after edits.** Dispatch exact plan reviewer once per
   correction/fingerprint (`per correction/fingerprint`) with accepted findings,
   changed paths/criteria, and new
   identity. Within one correction, no broad third loop. If it changes plan,
   fold again. A closed input plus a distinct Critical/Important invariant returns
   that new fingerprint as progress. Then close matrix and rerun ID/meaning audit.
8. **Build readback and readiness.** Add a cited five-line readback to
   `eng-review.md` (artifacts.md §7 rows 1–5): outcome/ACs; IN/OUT/must-NOT; UI direction and architecture/
   critical flow; slice order/first slice; decisive proof/action-time gates. A
   fresh implementer must need no product, architecture, or proof invention.
   Contradiction, ownerlessness, or material ambiguity blocks via Clarify or Plan.

   Write exactly one `Implementation readiness: READY`, `NEEDS CLARIFICATION`,
   or `NEEDS REPLAN`. Root alone sets READY after every account, checklist,
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

## Phase exit (observable)

**Complete when:** `eng-review.md` records exactly one readiness verdict, readiness
binding SHA-256 passes, and every required reviewer account is admitted.

**Failing case:** READY written while a required reviewer returned `Outcome: gap` →
not complete; restore NEEDS REPLAN or dispatch missing reviewer.
