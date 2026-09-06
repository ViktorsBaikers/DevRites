# Candidate integrity

This file owns the Build-to-Ship lifecycle for one content-bound candidate. The
exact `touched-files.md` and binding grammar remain in
[`workspace-artifact-schema.md`](workspace-artifact-schema.md); phase skills link
here and state only their phase action.

`devrites-engine check candidate <slug>` is the only candidate hash authority.
A missing, duplicate, malformed, or mismatched binding stops the lifecycle.

## Lifecycle

- **Build maintains** the authoritative manifest from each green slice's actual
  scoped diff. It may change until Polish closes the candidate.
- **Prove binds** approved real proof to one digest. Check the candidate before
  commands and again after them; the digest must be identical and proof must not
  mutate candidate sources. Record it in evidence and browser evidence when present.
- **Polish closes** the candidate only after every code/UI correction, capability
  fold, optional design-memory update, and durable ADR promotion. Update the
  manifest for every such path; a digest change requires affected real re-proof
  and refreshed evidence/browser bindings before Review.
- **Review binds** all immutable reviewers and `review.md` to the same closed
  digest. A candidate correction updates the manifest and returns through affected
  Prove, then a fresh Review.
- **Seal binds** `seal.md` to that reviewed digest, rechecks every binding, then
  runs `devrites-engine check seal <slug>`. A correction returns through Prove and
  Review before Seal runs again.
- **Ship is candidate-read-only.** It may update workspace ship/state/archive
  bookkeeping, but any project or manifest mutation returns through Prove,
  Review, and Seal. Git integrity uses the canonical digest; it never implements
  another candidate hash.

## Existing workspaces

An older unfinished workspace must refresh the manifest and rerun real proof before
continuing. Upgrade routes a released-workspace candidate defect to Prove only after its
read-only assessment cites the current contract and exact defect. Prove may refresh from
unambiguous legacy scope and currently observed project bytes, then writes fresh bindings.
Never synthesize a historical pass or infer old bytes. There is no legacy fallback.

Normal Polish rollups and re-proof, Review, and Seal follow when their current gates
apply. Upgrade re-audits afterward; a failed candidate, seal, or readiness check remains
a gap. Exact manifest and binding grammar stays in the workspace schema above.
