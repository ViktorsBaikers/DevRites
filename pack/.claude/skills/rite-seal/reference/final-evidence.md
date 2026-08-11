# Final evidence check

Map every `spec.md` promise to fresh observed proof.

## Acceptance

For each AC/REQ/scenario/key link, exact `devrites-proof-runner` maps immutable
results and exact `devrites-spec-reviewer` checks implementation and proof
against its meaning. Bug fixes require original-symptom proof; self-report,
inspection alone, and generic "tests pass" do not count.

Behavioral claims require positive, discriminating evidence under `testing.md`.
Skipped/focused/filtered/pending, zero-test, assertion-free, tautological, unexecuted, or
exit-only results are unproven. Static gates prove only their named static criterion;
discriminating shell/golden/text assertions remain valid for textual or CLI acceptance.

## Cross-check

Require built slices, resolved or accepted findings, no drift/open validating
question, required UI/boundary proof, and an exact doubt verdict per stood decision.
Reconcile the final diff with the spec applicability map. Every triggered topology,
data, integration, security, and delivery owner needs its relevant observed failure/
recovery evidence or an evidence-backed dismissal; a generic green suite cannot close it.

## Candidate binding and test integrity

Run `devrites-engine check candidate <slug>` and require its digest to match the
single evidence/review bindings and browser binding when present. Diff tests,
dispatch exact `devrites-test-analyst`, run approved repository commands, then
require the same candidate digest and no source mutation. Deleted/skipped/focused/
weakened/tautological tests are Critical.

After the native semantic verdicts are recorded in `seal.md`, run:

```bash
devrites-engine check seal "<slug>"
```

This checks structure and exact candidate bindings, never acceptance, test
quality, doubt, or review. Mutation stays optional unless project policy or
`test-plan.md` requires it.
