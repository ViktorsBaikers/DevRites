# Final evidence check

The seal verifies that every promise in `spec.md` is backed by evidence in the workspace.

## Acceptance-criteria pass
Walk `spec.md` → "Acceptance criteria" one by one. For each:
- Find the proving evidence in `evidence.md` / `browser-evidence.md` (command output,
  observation, screenshot described).
- Mark `[x]` + the evidence, or `[ ] unproven` + why.
- A criterion "proven" only by reading the code is **not proven**: require a runtime/
  test artifact, or downgrade it.
- Hold the claim→evidence line: **"tests pass" is not "the feature works"** (the acceptance
  walk is); **"bug fixed"** requires the original-symptom test now passing, not "code changed,
  assumed fixed"; **"the agent reported success"** is never evidence: the command output is.

Tag each new-workspace criterion `AC-###` in `spec.md` and carry the same id onto its checked line in
`seal.md`, then grade coverage deterministically: every spec criterion id must be checked:
```bash
devrites-engine check-acceptance .devrites/work/<slug>   # exit 1 = a criterion is uncovered/unproven → NO-GO
```

## Cross-check the artifacts
- `tasks.md`: are all slices in scope `built` AND is feature acceptance proven
  (`evidence.md`)? Acceptance proof lives at the feature level, not per slice: a slice
  is `built`, the feature is proven. Any pending slice that's part of this feature's
  definition of done is a gap.
- `review.md`: are all **Critical** and **Important** findings resolved (or accepted)?
- `drift.md`: every entry marked resolved? Open drift blocks.
- `questions.md`: any open question that changes behavior? Blocks. **Any entry with
  `gate: validating` and `status: open` is a NO-GO regardless of behavior impact**: an
  open validating gate is merge-blocking by definition; a slice marked
  `built (pending review)` is not done.
- `polish-report.md` (if UI): normalize+polish done, browser evidence present.
- `references/` (if the spec gathered design references): does the built UI **match the
  agreed references**? A mismatch on a referenced screen is a finding.

## Evidence freshness: a GATE, not a courtesy
Evidence must post-date the code it proves. **If `evidence.md` or `browser-evidence.md`
predates the latest edit to any file in `touched-files.md`** (e.g. a polish- or
review-phase edit landed after `$rite-prove`), a fresh `$rite-prove` run is **REQUIRED
before GO**: stale evidence is not proof. Compare the evidence timestamps against the
touched-files mtimes; if any code edit is newer, the proof is stale and this is a NO-GO
until re-proven. Record the fresh result.

Run the deterministic check rather than eyeballing mtimes:
```bash
devrites-engine evidence-fresh   # exit 3 = STALE proof → NO-GO until $rite-prove re-runs
```

## Test integrity & mutation: the suite must be real
Stale proof isn't the only path to a false GO; a *weakened* test suite is. Run the deterministic
gates:
```bash
devrites-engine test-integrity; echo "test-integrity rc=$?"   # exit 3 = a test deleted/skipped/loosened → Critical NO-GO
devrites-engine mutation-gate   # changed-files mutation score — band the verdict; survivors are unproven behaviours
devrites-engine package-existence; echo "package-existence rc=$?"   # exit 3 = a new import isn't declared in any manifest (hallucinated/typo-squatted dep) → Critical NO-GO
```
A test weakened since the slice base is a **Critical NO-GO**: the suite went green by lowering the
bar, not by the code being right. Record the mutation score in `seal.md`; under
`DEVRITES_MUTATION=enforce` a sub-threshold score blocks GO.

## Output into seal.md
The "Acceptance Criteria" and "Verification Evidence" sections of `seal.md` come
straight from this check. Unproven critical criteria → list them as **blockers**.
