# rite-build phase contract

Apply [`one-slice-cycle.md`](one-slice-cycle.md) and
[`candidate-integrity.md`](../../devrites-lib/reference/candidate-integrity.md).
Opt-in `--parallel N` replaces steps 2–8 with [`parallel-batch.md`](parallel-batch.md);
Spec Drift Guard still applies.

1. **Gate.** Read core, `.devrites/ACTIVE`, then `devrites-engine orient <slug>`
   and the Build read-next in
   [`workspace-artifact-schema.md`](../../devrites-lib/reference/workspace-artifact-schema.md#read-next-by-phase):
   the selected slice via `devrites-engine observe slice <slug> <SLICE-ID>`, not whole
   `tasks.md`; over-budget or `bulk_files` artifacts are read by anchor only. Require
   `Implementation readiness: READY`, current `Readiness inputs SHA-256`, exact
   required accounts, ID/meaning traceability and green proof preflight.
   Run `devrites-engine check readiness <slug>`; any miss/nonzero blocks.
2. **Select.** Pick the lowest pending slice whose `depends_on` are all built, using
   `orient` `task_graph.slices` plus the `state.md` built list — never by scanning
   `tasks.md`; then `observe slice` it. Restate its goal, ACs, exclusions and exact paths. Include
   applicable topology/data/integration invariants and failure/recovery/proof cases.
   Verify relevant assumptions live; falsified acceptance/architecture/scope/proof
   routes through Spec Drift Guard. Resolve human-owned/irreversible gates before
   source work; AFK follows its ceiling. Vet-ready Workflow Artifacts use
   [`workflow-artifacts.md`](../../devrites-lib/reference/standards/workflow-artifacts.md).
3. **Dispatch.** Apply [`wright-dispatch.md`](wright-dispatch.md). Put the smallest
   exact project-relative source/test path list directly in the task;
   dispatch the exact `devrites-slice-wright` fresh. Root never writes product
   paths; wright cannot widen scope; missing profile blocks.
<!-- workflow-artifact-adapter: {"module":"devrites-lib/reference/standards/workflow-artifacts.md","entry":"Build gate enters or resumes transaction","action":"invoke canonical operation table; reconcile exact result","return":"same slice/checkpoint cursor or Plan/Vet route"} -->
4. **Inspect.** Wait; compare returned paths and `git diff --name-only` with the
   contract. Reject stale/partial/malformed/out-of-scope work; preserve user work.
   Restoration uses the bounded wright.
5. **Review.** Apply [Independent Build review](#independent-build-review).
   Missing verdicts, principle breaches and open Critical/Important block acceptance.
6. **Prove.** Against the frozen pre-slice candidate inspect
   test hunks for deletion, skipping/focus, tautology, or weaker expectations.
   Use step 5's exact test-analyst account; missing/adverse criterion verdict is
   Critical. Run only `test-plan.md` repository/CI commands, record output, recheck
   `git diff --name-only`. Missing/weakened/red/source-mutating proof cannot pass.
   Pre-existing/environment-specific claims need a predating same-command,
   same-environment baseline; otherwise unresolved.

   Technical failure uses `devrites-debug-recovery` and
   [canonical recovery](../../devrites-lib/reference/standards/afk-hitl.md#retry-cap-no-progress-loops-and-self-resolve).
   Reapply host admission before correction; affected proof/evidence follows candidate
   integrity. Human gates remain product/irreversible/access decisions.
7. **Record green.** Upsert actual scoped `present`/`deleted` rows in `touched-files.md`;
   update state, evidence and applicable UI/browser proof. Manifest stays mutable
   until Polish. For a release milestone, after the last slice's rows run
   `devrites-engine state merge-manifest <slug>` to fold the recorded
   predecessor chain into the union manifest before Prove. Record stood
   decisions/dead ends; refresh verified/falsified
   assumptions. Propose reviewed durable project rules in the nearest guidance.
8. **Account/reply.** Apply `afk-discipline.md`: charge
   exactly once after each green built slice; never negative, stop at zero,
   malformed budget fails closed. Reply names next pending slice or `$rite-prove`
   only when all built; no decorative renderer/dispatch telemetry.

One contract builds one slice: preserve TDD, exact paths and independent judgment.

## Independent Build review

After inspection, freeze full slice scope and cumulative baseline diff. Initial
inventory: one full-diff `devrites-code-reviewer`, one `devrites-test-analyst`,
and one exact `devrites-doubt-reviewer` per stood decision that meets
`devrites-doubt` § When to use (boundary, data model, auth, public API, migration,
user flow, assumption tests cannot prove); other stood decisions are recorded in
`decisions.md` without dispatch. Dispatch fresh,
read-only under [`parallel-dispatch.md`](../../devrites-lib/reference/parallel-dispatch.md).
Wait for every required account to become terminal; missing/gap needs a valid result.
Fold supported in-scope Critical/Important findings by cause into one inventory.
Dispatch one repair-all wright instructed to fix every finding, then attack its own
correction for fix-introduced defects before returning; durable contract defects use Spec
Drift Guard.
Suggestion/Nit/FYI cannot prolong recovery.

Rechecks use Vet's bounded pattern: each exact responsible reviewer checks
all open findings owned by that role and affected dependency/regression closure
in one fresh dispatch. Supply new candidate, inventory, correction diff, tests
and unchanged-coverage evidence under candidate integrity; include newly triggered
roles. A changed contract or uncertain impact requires the full inventory.
Collect all required rechecks before another folded correction. Canonical recovery
owns fingerprints/budget and the
[late-finding rule](../../devrites-lib/reference/standards/afk-hitl.md#retry-cap-no-progress-loops-and-self-resolve):
a recheck's `Late:` rows on hunks unchanged since that role's previous pass go to
`touched-files.md` `## Review trail` for `$rite-review`, not into another slice
repair, unless Critical with a failure path. Evidenced scope coverage never
guarantees no later defect; the feature-level Review and Seal gates still run.
