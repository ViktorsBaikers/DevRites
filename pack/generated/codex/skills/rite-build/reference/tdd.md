# TDD / Prove-It pattern

When a slice changes behavior, prove the change with a test that fails first, so you
know the test can fail, and that your code is what makes it pass.

## Red → Green → Refactor
1. **Red:** write the smallest test that expresses the new behavior. Run it; watch it
   fail for the *expected* reason (not a typo/import error).
2. **Green:** write the minimum code to pass. Resist gold-plating.
3. **Refactor:** clean up with the test as a safety net. Re-run; stay green.

## Prove-It (when strict TDD doesn't fit)
For code where a test-first flow is awkward (exploratory glue, config, some UI), still
produce evidence:
- Capture the current (wrong/absent) behavior: a failing assertion, a screenshot, a
  log line.
- Make the change.
- Capture the new behavior the same way. The before/after pair *is* the proof.

## Rules
- Use the project's existing test framework and commands (discovered in `spec.md` →
  "Commands discovered"). Don't introduce a new test runner to prove a slice.
- One behavior per test; name it for the behavior, not the function.
- A green test you never saw fail proves nothing: always see red first.
- Don't delete or weaken a failing test to "pass". If it's wrong, that may be drift.
- Record the test command + result in `evidence.md`.
