---
name: rite-vet
description: Review a defined engineering plan before code. Use for plan vetting or lock-in; not for implementation review or final readiness sealing.
argument-hint: "[slug] [--cross-model] [--full]"
user-invocable: true
---

<!-- loads: {"always":["devrites-lib/reference/phase-target-binding.md","devrites-lib/reference/standards/core.md","devrites-lib/reference/standards/code-navigation.md","devrites-lib/reference/standards/code-index.md","devrites-lib/reference/orchestration-profiles.md","devrites-lib/reference/parallel-dispatch.md","devrites-lib/reference/standards/gates.md","rite-vet/reference/review-axes.md","rite-vet/reference/artifacts.md","rite-vet/reference/depth.md","rite-vet/reference/eng-lenses.md","rite-vet/reference/anti-patterns.md","devrites-lib/reference/serialization-audit.md"],"triggers":{"afk":["devrites-lib/reference/standards/afk-hitl.md"],"applicability":["devrites-lib/reference/standards/repository-topology.md","devrites-lib/reference/standards/data-integrity.md","devrites-lib/reference/standards/integration-reliability.md"],"assumption-delta":["devrites-lib/reference/standards/assumption-checkpoints.md"],"cross-model":["rite-vet/reference/cross-model.md"],"devex":["devrites-lib/reference/standards/developer-experience.md"],"dod":["devrites-lib/reference/standards/definition-of-done.md"],"elicitation":["devrites-lib/reference/standards/elicitation.md"],"errors":["devrites-lib/reference/standards/error-handling.md"],"grammar":["devrites-lib/reference/standards/spec-grammar.md"],"one-shot":["devrites-lib/reference/standards/one-shot-actions.md"],"performance":["devrites-lib/reference/standards/performance.md"],"principles":["devrites-lib/reference/standards/principles.md"],"reslice":["devrites-lib/reference/standards/acceptance-preserving-reslice.md"],"testing":["devrites-lib/reference/standards/testing.md"],"workflow":["devrites-lib/reference/standards/development-workflow.md"],"workflow-artifacts":["devrites-lib/reference/standards/workflow-artifacts.md"],"yagni":["devrites-lib/reference/standards/patterns.md","devrites-lib/reference/standards/coding-style.md"],"parallel-workspaces":["devrites-lib/reference/parallel-workspaces.md","devrites-lib/reference/workspace-artifact-schema.md"]},"workspace":["brief.md","spec.md","state.md","decisions.md","assumptions.md","questions.md","decision-coverage.md","strategy.md","architecture.md","plan.md","tasks.md","traceability.md","eng-review.md","test-plan.md","gates.md"],"rootOnly":["devrites-lib/reference/parallel-workspaces.md","devrites-lib/reference/phase-target-binding.md","devrites-lib/reference/standards/code-index.md","devrites-lib/reference/standards/afk-hitl.md","devrites-lib/reference/standards/elicitation.md","devrites-lib/reference/orchestration-profiles.md","devrites-lib/reference/parallel-dispatch.md","devrites-lib/reference/standards/workflow-artifacts.md"],"workspaceByRole":{"doubt-reviewer":["spec.md","architecture.md","plan.md","tasks.md","test-plan.md","decisions.md","state.md"],"devex-reviewer":["spec.md","plan.md","tasks.md","decisions.md","state.md"],"plan-reviewer":["spec.md","plan.md","tasks.md","test-plan.md","traceability.md","decision-coverage.md","architecture.md","decisions.md","state.md"],"strategy-reviewer":["brief.md","spec.md","plan.md","decisions.md","decision-coverage.md","strategy.md","state.md"]}} -->
> Read-set manifest: `devrites-engine context --skills-root .omp/skills <slug> --phase vet` bundles every file named below into one deduplicated bundle. Trigger names map to the conditional rules in the sections that follow. Apply [controller load conditions](skill://devrites-lib/reference/phase-target-binding.md#controller-guidance-load) before controller work (trigger `parallel-workspaces`). Read index-first under [core Context hygiene](skill://devrites-lib/reference/standards/core.md#context-hygiene-end-of-every-phase).


# /rite-vet: review the plan before build

Vet each plan: scope, architecture, quality, proof, performance, failures, writer
safety. Fold cited technical findings; design Build tests. Temper owns product,
Vet implementation. Current `$ARGUMENTS` (`--full`) feeds `reference/depth.md`
triggers; profiles never remove the exact plan-reviewer gate
([orchestration-profiles.md](skill://devrites-lib/reference/orchestration-profiles.md)).

## Rules

Read the active standard from: [`principles.md`](skill://devrites-lib/reference/standards/principles.md), [`patterns.md`](skill://devrites-lib/reference/standards/patterns.md) + [`coding-style.md`](skill://devrites-lib/reference/standards/coding-style.md) (over-engineering / YAGNI rubric, trigger `yagni`),
[`testing.md`](skill://devrites-lib/reference/standards/testing.md), [`spec-grammar.md`](skill://devrites-lib/reference/standards/spec-grammar.md), [`performance.md`](skill://devrites-lib/reference/standards/performance.md), [`error-handling.md`](skill://devrites-lib/reference/standards/error-handling.md) (trigger `errors`),
[`development-workflow.md`](skill://devrites-lib/reference/standards/development-workflow.md), [`afk-hitl.md`](skill://devrites-lib/reference/standards/afk-hitl.md), [`one-shot-actions.md`](skill://devrites-lib/reference/standards/one-shot-actions.md),
[`developer-experience.md`](skill://devrites-lib/reference/standards/developer-experience.md), [`elicitation.md`](skill://devrites-lib/reference/standards/elicitation.md), and [`definition-of-done.md`](skill://devrites-lib/reference/standards/definition-of-done.md) (trigger `dod`). Load
repository topology, data integrity, and integration reliability only when
triggered (trigger `applicability`). When the plan's scope generalizes an existing boundary — a second
platform/tenant/source of truth, a newly optional field, a constant turned parameter —
confirm the [`assumption-checkpoints.md`](skill://devrites-lib/reference/standards/assumption-checkpoints.md) question (trigger `assumption-delta`)
was answered and recorded, not skipped-as-negative.
Before classifying any Reslice, read `.omp/skills/devrites-lib/reference/standards/acceptance-preserving-reslice.md`.
When a
plan declares a root-authored executable workflow file, read
[`workflow-artifacts.md`](skill://devrites-lib/reference/standards/workflow-artifacts.md).

<!-- BEGIN RESLICE ROUTE-TO-ACTION -->
- `FOLD` → fold technical topology; invalidate Vet/readiness; affected Vet before Build.
- `GUARD_AND_REPAIR` → no planning writes; Spec Drift Guard → Clarify → Plan repair → affected Vet.
- `BLOCKED_INPUT` → no planning writes; exact diagnostic; recover input; reclassify.
<!-- END RESLICE ROUTE-TO-ACTION -->

## Invariants

- Review implementation, not ambition. Challenge creep/complexity/proof/risk
  without changing accepted product scope.
- Root owns decisions/writes/readiness; reviewers judge, not route. Cite
  findings/confidence; suppress unverified or confidence ≤4 findings per
  `review-axes.md`. `eng-lenses.md` arrows are heuristics; use only the four
  `review-axes.md` band names.
- Auth, migration, public API, and data-model changes need maximum caution,
  full review, and explicit rollback. Pause only for
  [canonical irreversible-risk items](skill://devrites-lib/reference/standards/afk-hitl.md#irreversible-risk-list-always-pause)
  or unresolved human-owned decisions, not an already-approved reversible internal
  data-model change. Project principles never become trade-offs.
- **Governance-protected paths** (`.devrites/**`, pack skill/agent trees,
  `NOTICE.md` generator regions, documented CI/hook config) require explicit
  human approval before plan slices edit them; absent approval → Vet
  **NEEDS CLARIFICATION**.
- Use lowest axis band; never average/round thin to ready. Search; resolve
  reversible technical choices; ask only human-owned.
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
- Refuse the rationalizations in [`anti-patterns`](reference/anti-patterns.md).
<!-- workflow-artifact-adapter: {"module":"devrites-lib/reference/standards/workflow-artifacts.md","entry":"plan declares root-authored executable workflow file","action":"emit exact admission; stale/missing authority uses PLAN_VET_REPAIR","return":"Vet READY cursor or exact technical replan"} -->
## Workflow

0. **Orient.** Read core. Resolve source per [phase target binding](skill://devrites-lib/reference/phase-target-binding.md#phase-target-binding), require state, and read plan,
   tasks, spec, decision coverage, optional strategy/design brief, decisions,
   assumptions, and state. Require approved Plan and `Decision coverage: CLEAR`;
   otherwise stop for Define or Clarify. Check placement (`get_architecture`),
   blast radius (`trace_path`), and reuse (`search_graph`) per [`code-navigation.md`](skill://devrites-lib/reference/standards/code-navigation.md).
   Under Autocomplete, return scheduling for its
   [global frontier pass](skill://devrites-lib/reference/parallel-workspaces.md#continuous-global-frontier) (trigger `parallel-workspaces`)
   before cold/local recovery; preserve Vet's source-local return cursor.
1. **Select depth.** Apply `reference/depth.md` exactly; never skip. Initial passes
   record engineering verdict and test-plan coverage. Valid Recovery rechecks
   retain prior depth; enter 1b.
1a. **Independent initial pass.** Freeze candidate; dispatch exact fresh read-only
   plan reviewer, plus developer-experience reviewer for developer surfaces
   (trigger `devex`) and current strategy reviewer after significant Temper, in
   parallel under `skill://devrites-lib/reference/parallel-dispatch.md`: independent
   read-only passes on one frozen candidate. Missing required account blocks.
   Meanwhile run root steps 2–4, including step 3 preflight commands, on that
   candidate. Under Autocomplete, before native wait return scheduling to its
   global frontier pass with reviewer handles retained; standalone Vet stays
   source-local. Apply the shared protected validation window and identity
   recheck before reconciling returned accounts.
1b. **Recovery recheck.** Root validates the existing return cursor, accepted prior
   findings and each exact owning reviewer, fingerprint/reproduction, repaired
   candidate identity, changed clauses/paths/criteria, affected drift/evidence,
   and recorded delta self-check. Keep prior admitted wave/role/handle/return
   provenance in the root-owned `eng-review.md` recovery record only. Missing
   provenance uses step 1a's initial applicable review, never a forged recovery claim.
   For folds requiring independent recheck, freeze the repaired candidate and
   dispatch each exact owning reviewer once, fresh/read-only, covering all its
   open fingerprints. For each owner, run in this order:

   ```text
   devrites-engine context --skills-root .omp/skills <slug> --phase vet --role <owner> --recheck packets/<fold>.<owner>.inventory.md [--slice <id> ...] [--trigger <applicable-set>]
   devrites-engine packet <slug> --wave <fresh-wave> --phase vet --role <owner> --recheck [--slice <id> ...] --focus packets/<fold>.<owner>.focus.md
   devrites-engine dispatch <slug> open --phase vet --wave <fresh-wave> --role <owner> [--slice <id> ...] --packet packets/<fresh-wave>.<owner>.md
   ```

   Repeat `--slice <id>` once per affected slice, identically in all three
   commands. No slice means plan-level affected recovery, not initial Vet. The
   inventory contains neutral reproduction, each owner fingerprint, correction
   scope, exact source/rule citations, and dependency-plus-traceability closure.
   Focus rows keep `- <ID> <path:line> <text>`. Keep prior verdicts, scores, and
   `*.return.md` links out of packets, inventories, and focus files. The engine
   checks flag shapes, readability, and integrity; root checks semantic completeness.
   Under Autocomplete, before native wait use the same controller pass and retained
   handles as step 1a; standalone Vet stays source-local. Apply the same
   protected-window account admission.
   Do not rerun broad inventory or unaffected reviewers. Close the prior
   fingerprint only with discriminating evidence plus an explicit
   correction-created regression verdict on the touched invariants (acyclic
   deps, traceability, contract proof, sizing) and every region depending on a
   changed clause — enumerate dependents via the dep graph and traceability
   refs, not only patched sites. Otherwise record one no-progress outcome. A
   different Critical/Important invariant needs exact evidence and a new
   fingerprint; a Suggestion, Nit, or FYI cannot keep recovery open. Reviewer
   `Late:` rows on unchanged clauses follow the canonical
   [late-finding rule](skill://devrites-lib/reference/standards/afk-hitl.md#retry-cap-no-progress-loops-and-self-resolve):
   Critical with a failure path → new fingerprint; otherwise `eng-review.md`
   `## Deferred findings` (mechanism rows become one proof row at step 6). A packet
   marked `convergence_pressure` accepts an explicit, bounded declared
   residual/limitation folded into the contract text as closure for a finding
   marked as a refinement of an already-pinned clause; the recheck may re-raise
   it only by naming a concrete failure path the declaration omits. A
   `new_failure_mode` finding is never residual-eligible. A fold whose
   findings are all `mechanism` kind or Suggestion-or-lower rechecks mechanically at the
   materialization gate — anchors applied (a clean `state apply-bundle`), mirrors consistent, invariants
   pass — without a dispatched reviewer; sub-Important findings already
   cannot keep recovery open. Reconcile
   shared artifact/readiness gates, then enter step 8 or the next repair.
   Do not fall through to steps 2–7. Retain evidenced unchanged coverage; changed
   global contract/uncertain impact requires full applicable review. Reopen historical
   findings only for affected invariants/dependencies.
   The recheck still closes every unbuilt slice's write set under step 8. The plan
   reviewer returns affected dimension bands plus the explicit `Regression:` verdict;
   unchanged dimensions are outside this account, not newly passed. Root retains
   valid prior coverage and uses the lowest applicable band across current accounts
   without sending prior scores to seed any reviewer.
2. **Challenge scope.** Apply review-axes §0; search accepted decisions.
   Harden to the smallest contract-complete plan, using marked topology action.
   Steps 2 and 4 form one complete root-owned cross-artifact audit on the frozen
   candidate: verify bidirectional ID-and-meaning traceability across spec/plan/
   tasks/test-plan/traceability, acceptance, terms, principles, anti-slop,
   and conventions; every slice and test-plan row maps to a live requirement
   and vice versa. Critical gaps/unexcepted principle breaches block. Record
   inputs, artifact presence, dependency closure, and scope result in
   `eng-review.md` §2 (Scope challenge); step 4 records its readiness result in §2b.
   Consuming checklists may cite these results only within this root-owned frozen
   pass, supported by current conversation evidence. Any input edit or unknown
   dependency invalidates affected coverage; unknown impact requires the full
   applicable audit. No persistent check cache or new verdict.
3. **Preflight Build entry.** Under `reference/artifacts.md`, verify exact
   command/cwd/tool/version/prerequisite; output filters must preserve upstream
   failure. Verify dependencies from authoritative source plus nearest manifest.
   Run parser-sensitive syntax only in isolated fixtures. Remeasure mutable facts;
   live evidence wins, conflict marks stale, and unmeasurable conflict is a gap.
   Record complete SHA-256 provenance. Every behavioral mapping names a positive
   discriminating assertion and decisive signal, never only exit zero.

   For each consumptive action, bind every `reference/artifacts.md` Consumptive
   action gates column; every fingerprint identifies one actionable seam; aliasing multiple
   emit sites is a gap. Preflight observes but need not make future behavior pass,
   except that a manifest/lockfile edit is dry-run now: before READY run
   `devrites-engine check preconditions <slug>` (rows with `Run at` `vet`, including each
   `dependency-dry-run` and preserve-baseline row). A failure folds into Plan repair; only an
   owner-owned decision becomes a question, batched under the afk-hitl decision-chain rule.
4. **Audit readiness.** Complete the same root audit from step 2 and record its
   readiness mapping/result in `eng-review.md` §2b. For explicit workflow-only
   tasks, follow
   [`workflow-artifacts.md`](skill://devrites-lib/reference/standards/workflow-artifacts.md#workflow-only-planning):
   check AC/REQ → admitted WA behavior and executable proof, all admission fields,
   fixtures, bounds and root ownership, not product slice writer/closure checks.
   Keep empty product candidate and zero built slices; no Build/wright dispatch.
   Missing/malformed admission routes Plan; stale admission requires fresh Vet and
   ordinary readiness SHA binding. Never switch ACTIVE for a named workspace.
   Otherwise goal-backward map every requirement, criterion, NFR,
   interaction, edge/prohibition, and decision row to one slice and executable
   proof. Verify UX/spec/architecture alignment, contracts, dependency order,
   slice independence/wiring, prerequisites, failure/observability/rollback, and
   ownership. The plan's `Shared contract proof` names one reused boundary
   artifact plus two consuming tests for every changed API/event/schema/provider-
   consumer seam, or an explicit no-impact statement. Missing, one-sided,
   duplicated-contract, vague, or non-consuming proof fails closed. Each slice
   names a fast writer-safe check its wright can run; proof reachable only through
   root-owned full gates is a `mechanism` finding.
   For canonical future-pool plans, apply the
   [consumed-input contract](skill://devrites-lib/reference/workspace-artifact-schema.md#consumed-planning-inputs):
   root and native reviewers verify `Consumed planning inputs` against actual design
   use, not roadmap scope ownership. Missing real input parks only this source
   under the shared controller policy; inventing design to keep the queue full is not repair.

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
   implementable; developer plans (trigger `devex`) write the devex reviewer's
   predicted scorecard to `devex.md`. Durable commands
   are portable repository commands, not host wrappers — resolve them with
   `devrites-engine detect commands` (Makefile/package.json/manifests, no
   execution) and cite the resolved text verbatim; an `unresolved` slot means
   name the concrete command in the plan, never guess one.
   Once the `## Build-entry preflight` table is final, run
   `devrites-engine gates scaffold <slug>` to seed `gates.md` (one pending gate
   per `AC-###`; never clobber an existing ledger), then author each gate's
   `CHECK`/`EXPECT`/`CWD` per artifacts.md §`gates.md` so every runnable oracle
   is an approved preflight row. Run
   `devrites-engine gates lint <slug> --strict` and resolve every finding before
   READY; the readiness check refuses a missing or malformed ledger but does not
   replace this oracle-quality gate.
7. **Narrow recheck after edits.** When the rules below require dispatch, validate
   step 1b's prerequisites and root-only recovery record, then use its exact
   context → packet → dispatch sequence for each owning reviewer on the frozen
   repair candidate. Keep inventory/focus neutral and use a fresh wave. Missing
   provenance requires step 1a's initial applicable review. A slice-bound recheck
   repeats `--slice <id>` per affected slice in all three commands; no slice means
   plan-level affected recovery (every 5 rechecks
   per slice need a reslice review, as in Build). **Workflow Artifact bytes-only:**
   after `OFFLINE_RECOVERY`, a correction leaving every admission/`WA-*` row
   unchanged records `Workflow Artifact reviewed identity: <re-preflight identity
   digest>` below the admission block; when `check drift` reports only
   `reviewed-identity-only`, the reviewer is one fresh `devrites-code-reviewer` on
   the prior-source→re-preflight diff against those rows (no drafter or plan
   reviewer); green proceeds to step 8. Otherwise dispatch each exact owning reviewer once per
   fold — one dispatch covering every open fingerprint that reviewer owns, preserving
   per correction/fingerprint accounting (each checked individually with its
   neutral reproduction, changed clauses/paths/criteria, new identity, and recorded delta
   self-check); it must return an explicit correction-created regression verdict
   over touched invariants plus every dependent of each changed clause (dep-graph
   and traceability lookup), not only fingerprint closure. The plan reviewer also
   returns affected dimension bands. Unchanged
   dimensions are not newly passed; root retains valid prior coverage and takes the
   lowest applicable band across current accounts. Every unbuilt slice still needs
   write-set closure; changed global contracts or unknown impact require full
   applicable review. Within one correction, no broad
   third loop. If it changes plan, fold again. A closed input plus a distinct Critical/Important invariant returns
   that new fingerprint as progress. Then close matrix and recheck affected
   ID/meaning mappings and their dependency/traceability closure under step 8.
8. **Build readback and readiness.** Add a cited five-line readback to
   `eng-review.md` (artifacts.md §7 rows 1–5): outcome/ACs; IN/OUT/must-NOT; UI direction and architecture/
   critical flow; slice order/first slice; decisive proof/action-time gates. No implementer should need to invent product, architecture, or proof.
   Contradiction, ownerlessness, or material ambiguity blocks via Clarify or Plan.
   Cite the recorded steps 2+4 audit for unchanged portions; re-read corrected
   inputs and recheck everything affected by step-6 or later edits, including
   dependent mappings outside edited slices. Unknown dependency/impact requires
   the full applicable audit. Record refreshed results in §§2/2b before consuming
   them here. This reuse never replaces native grammar/post-correction rereads,
   mandatory phase-entry/exit checks, fresh required reviewer passes, preflight,
   or proof. Neither successful `state apply-bundle` nor an unchanged readiness
   digest grants readiness.

   Write exactly one `Implementation readiness: READY`, `NEEDS CLARIFICATION`,
   or `NEEDS REPLAN`. Root sets READY after every account, checklist,
   preflight, and sweep is green, including the
   [`spec-grammar.md`](skill://devrites-lib/reference/standards/spec-grammar.md) marker sweep whose
   `marker sweep:` line is recorded in `eng-review.md`; sweep current artifact
   presence as well as content, not only unchanged spec bytes. A kept hit blocks READY. Write phase/next step and emit one
   `Readiness inputs SHA-256` with
   `devrites-engine check readiness --emit-binding <slug>`; normal readiness check
   must pass; workflow-only checks admission instead. For product work it replays Build-entry `check slice` on every unbuilt slice, so
   READY needs each to pass (fix failures in `/rite-plan`). READY also needs every
   unbuilt slice's write set closed against the live tree (plan-reviewer
   dimension 6), so frontier gaps fold in one Plan repair, not one per slice; an
   affected recheck also closes every unbuilt slice and never defers a closure row.
   For canonical future-pool preparation, require `Consumed planning inputs: none`
   or unique comma-separated related slugs in `plan.md`. Record exactly one
   `Predecessor readiness: <slug> <digest>` per declared producer in `eng-review.md`
   using its current `--emit-binding` hex, including nonadjacent producers; for an
   empty set write only `Predecessor readiness: none`. Duplicate, missing, extra, or
   drifted bindings refuse READY. An unrelated adjacent predecessor needs no binding;
   legacy absent declarations remain valid only outside canonical future admission.
   Explicit separate-session lookahead still records its active predecessor binding.
   Apply native dependency-safety review to explicit `Workspace dependencies`; only
   durably BUILT/integrated producer checkpoints satisfy them. Planning digests never
   substitute for those gates; stale inputs require affected re-Vet, never auto-rebinding.
   Record the per-input drift baseline alongside it:
   `devrites-engine check drift <slug> --record` — a later stale binding then
   names the exact artifact that moved instead of "something changed". An `ADVISORY
   proportion:` line names a slice whose proof outgrew its ACs: move its rows without a
   `Proves` spec line to `Deferred hardening` before READY, never deepen them. Technical failure records reproduction, not qid. Human gap awaits
   Clarify. Optional cross-model follows `reference/cross-model.md` (trigger `cross-model`).

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
