# Candidate integrity

Manifest and binding grammar remain in
[`workspace-artifact-schema.md`](workspace-artifact-schema.md).
`devrites-engine check candidate <slug>` alone hashes candidates.
Missing/duplicate/malformed/mismatched bindings block.

## Lifecycle

- Build maintains actual scoped `present`/`deleted` manifest rows.
- Prove binds approved real proof in evidence/browser artifacts; candidate checks
  before/after commands match; no source mutation.
- Polish closes only after code/UI corrections, capability folds, design-memory
  updates and ADR promotion; update manifest/affected proof for every path.
- Review binds reconciled coverage to that closed digest; corrections return
  through affected Prove and fresh affected reviewers.
- Seal binds the reviewed digest, rechecks bindings, runs `devrites-engine check seal <slug>`.
- Ship is candidate-read-only; bookkeeping only; source/manifest changes return through Prove,
  Review, Seal. No alternate hash implementation.

## Evidence validity

Verify evidence, scope/criterion, source, dependencies, configuration, environment, toolchain, command/cwd,
prerequisites, and decisive result as applicable. Unknown provenance blocks; source
hash alone is insufficient. External or time-sensitive observations rerun under
current authority. Required feature proof cannot substitute slice results.

Same-candidate accounts may be cited when permitted. Corrections require
new candidate checks, affected real proof and fresh affected exact reviewers for
all open findings and dependency/regression closure, including newly triggered roles.
After an all-met acceptance ledger has been executed, any candidate digest change
requires `devrites-engine gates reverify <slug>` against the corrected frozen
candidate before refreshed evidence or review bindings are written; ordinary
`gates run` skips already-met rows and cannot refresh them.
Changed contracts or uncertain closure require a full applicable pass.
Unchanged sub-scope needs explicit unchanged-input/dependency justification and a
link to original evidence. Never relabel an old candidate-bound account.
Keep one current aggregate binding; never copy a historical binding line, even quoted. Root reconciles complete coverage and binds
it anew. Missing coverage blocks; required roles, criteria, proof and authority remain.

## Resume validation

On resume — compaction, new session, reopened workspace — recorded evidence is
a claim, not a fact. Before reusing any of it, compare the ledger's candidate
digest, contract version, environment, gate/fingerprint counters, and coverage
inventory against the live workspace. Rows whose dependencies changed or cannot
be bounded invalidate; rows that provably still apply are cited as reuse with
their original observation. Counters survive session restarts and version
bumps — a renamed file or a fresh context never resets a spent attempt. See
[`standards/verification-methods.md`](standards/verification-methods.md) for the
status (`fresh`/`reused`/`failed`/`missing`/`blocked`) and result
(`pass`/`fail`/`unverified`) vocabulary coverage rows carry.

## Existing workspaces

Upgrade routes a released-workspace candidate defect to Prove with a cited
current-contract defect and unambiguous legacy scope/bytes: refresh the manifest and rerun real proof.
Never synthesize a historical pass or bytes; no legacy fallback. Unknown bindings
require full refresh. Apply Polish/Review/Seal; Upgrade re-audits. Failed gates block.
