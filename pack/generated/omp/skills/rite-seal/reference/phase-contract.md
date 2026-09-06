# rite-seal phase contract

Seal's candidate action follows
[`candidate-integrity.md`](../../devrites-lib/reference/candidate-integrity.md).

1. **Read.** Resolve the explicit slug or `.devrites/ACTIVE`, require its
   `state.md`, then read required workspace artifacts, conditional
   browser/design/devex evidence, and final diff.
2. **Recheck identity and real proof.** Run
   `devrites-engine check candidate <slug>` and require its exact digest to match
   the single evidence/review bindings and the browser binding when present.
   Apply `spec-grammar.md`'s Native grammar re-read checklist, then run only
   `test-plan.md`'s repository/CI commands against that frozen digest. Rerun the
   candidate check afterward and require the identical digest and no source
   mutation. Capture command, cwd, exit, decisive output, and identity.
   Missing/red/mismatched blocks; unavailable is `cannot_verify`, never pass.
3. **Judge acceptance.** Dispatch exact `devrites-proof-runner` on immutable
   proof and exact `devrites-spec-reviewer` on the same candidate. Reconcile every
   AC/REQ/scenario/key link by ID and meaning; missing/stale/self-attested/mismatched is NO-GO.
4. **Judge tests.** Diff baseline/candidate tests for deletion, skip/focus,
   tautology, or weakening; dispatch exact `devrites-test-analyst`. Missing or
   adverse account, or green tests not asserting mapped behavior, is Critical.
5. **Resolve gates.** Open blocking/escalating/validating questions and drift
   block. Each stood decision needs a reconciled exact `devrites-doubt-reviewer` verdict.
6. **Risk and native review.** Apply [`risk-and-rollback.md`](risk-and-rollback.md)
   and the [`review roster`](../../devrites-lib/reference/parallel-dispatch.md) on
   the candidate. Admit each result through
   [`result admission`](../../devrites-lib/reference/standards/agents.md#result-admission).
   `## Reviewer Accounts` preserves all seven; gap/missing/malformed blocks;
   conditional `Not-applicable` names why.
7. **Route corrections.** Reconcile cited findings/disagreements, but do not
   edit the candidate or manifest in Seal. Route an accepted correction through
   one bounded wright, update the manifest, run affected Prove, complete a fresh
   Review, then restart Seal on the new digest.
8. **Draft, then check deterministic invariants.** Write `seal.md` from
   [`seal-template.md`](seal-template.md) with the GO/NO-GO rationale, acceptance map,
   test-analysis account, stood-decision verdicts, all seven reviewer accounts,
   and exactly one `Candidate SHA-256: <64 lowercase hex>` line. Then run:
   ```bash
   devrites-engine check seal "<slug>"
   ```
   This checks structure and exact candidate bindings, not prose or semantics. Blocks exit `3`;
   invalid state exits `2`. GO sets `Next step: /rite-ship`; NO-GO names the
   smallest fix. Never commit, push, tag, publish, deploy, or close here.

> When tempted to round NO-GO up to GO or average away reviewer disagreement,
> load [`anti-patterns.md`](anti-patterns.md).
