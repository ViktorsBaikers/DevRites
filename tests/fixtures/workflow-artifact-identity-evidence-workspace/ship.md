# Ship: workflow-artifact-identity

- Shipped at: 2026-08-22T22:55:00Z
- Verdict: GO (see seal.md)
- Branch: feat/workflow-artifact-identity
- Commit(s): 40fb5f2673ac2a1c23b9d661b68759e91bf4917b (includes prior local `1d9639f` on branch)
- Tag / PR: https://github.com/ViktorsBaikers/DevRites/pull/36

## What shipped
Workflow Artifact identity and sole-wright delivery (38-file candidate), plus the
already-staged acceptance-preserving reslice policy surface and workspace
observation engine support, pushed on the feature branch without Co-authored-by.

## Acceptance
- Criteria proven: 7 / 7   (full walk in evidence.md / seal.md)
- Outstanding (shipped with known follow-ups): intermediate dest plant under a
  real head (`tests/workflow-artifact-identity-test.sh:8835-8845`, Suggestion)

## Evidence pointers
- seal.md — final GO/NO-GO verdict + reviewer reconciliation
- evidence.md — acceptance walk; browser-evidence.md (if UI)
- review.md — multi-axis review findings

## Follow-ups (FYI, not blocking)
- Suggestion: reject `-L` on every `$_dest` path component and/or add a plant
  fixture under a real head
- design-memory: skipped (no UI)
- ledger sync: skipped (engine lacks `ledger`; `.devrites/specs/` gitignored)
- ADR promotion: skipped (decisions retained in archived workspace)
