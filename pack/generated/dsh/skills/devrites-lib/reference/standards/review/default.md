# Review: language-agnostic

> Applies when: reviewing any source file. Load alongside the file's
> language checklist.

## Defect probes

- **Dropped result.** A fallible call whose error/result value is discarded,
  swallowed by a catch-all, or logged-and-continued without a stated reason.
  Trace the failure path to its user-visible consequence before assigning
  severity.
- **Boundary values.** Off-by-one on ranges, empty collection, single element,
  max/min, zero, negative where only positive is expected, `nil`/`null`/`None`
  reaching a dereference or call.
- **Resource lifecycle.** Acquired resource (file, connection, lock, timer,
  subscription, temp dir) without a matching release on every exit path —
  especially early returns and error branches.
- **Concurrency.** Shared mutable state read/written without synchronization;
  a lock held across a blocking or fallible call; a goroutine/thread/task whose
  failure is unobserved; time-of-check/time-of-use gaps.
- **Copy-then-drift.** A new block that mirrors an existing one with only names
  changed — check whether the original receives fixes the copy will miss.
  Cross-check with `devrites-engine check dup` output when available.
- **Assertion-free test.** A test that exercises code without discriminating the
  promised behavior. A no-throw assertion is valid when non-throwing behavior is
  the actual contract and the targeted regression would make it fail; it cannot
  prove a returned value or side effect. Cite the missing proof at `file:line`.
- **Config/code mismatch.** A default, limit, path, or name in config that
  disagrees with the code that consumes it.
- **Unreachable branch.** A condition that can never hold given the types or
  the guarding checks above it.

## Do not flag

- Style choices a formatter or linter already enforces.
- "Could be more readable" without a concrete confusion a reader would hit.
- Hypothetical inputs the caller contract already excludes — verify the
  contract at call sites before claiming a missing guard.
- Performance concerns without a hot-path or data-size basis; cite the basis
  or mark it a question.
- Absence of a feature nobody asked for; scope expansion is a follow-up, not a
  finding.
