# Final evidence check

The seal verifies that every promise in `spec.md` is backed by evidence in the workspace.

## Acceptance-criteria pass
Walk `spec.md` → "Acceptance criteria" one by one. For each:
- Find the proving evidence in `evidence.md` / `browser-evidence.md` (command output,
  observation, screenshot described).
- Mark `[x]` + the evidence, or `[ ] unproven` + why.
- A criterion "proven" only by reading the code is **not proven** — require a runtime/
  test artifact, or downgrade it.

## Cross-check the artifacts
- `tasks.md`: are all slices in scope built and proven? Any pending slice that's part of
  this feature's definition of done is a gap.
- `review.md`: are all **Critical** and **Important** findings resolved (or accepted)?
- `drift.md`: every entry marked resolved? Open drift blocks.
- `questions.md`: any open question that changes behavior? Blocks.
- `polish-report.md` (if UI): normalize+polish done, browser evidence present.
- `references/` (if the spec gathered design references): does the built UI **match the
  agreed references**? A mismatch on a referenced screen is a finding.

## Re-run when cheap
If a key test or build is fast, re-run it now rather than trusting a stale log —
especially after review-phase edits. Record the fresh result.

## Output into seal.md
The "Acceptance Criteria" and "Verification Evidence" sections of `seal.md` come
straight from this check. Unproven critical criteria → list them as **blockers**.
