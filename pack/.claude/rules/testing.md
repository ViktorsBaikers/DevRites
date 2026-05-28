# Testing

Tests are evidence. They exist to prove behavior and to catch regressions — not to hit a
coverage number.

## Shape: a pyramid
- **Many** fast, isolated unit tests at the base.
- **Some** integration tests for how components actually fit together.
- **Few** end-to-end tests for critical user journeys (they're slow and the most
  flake-prone — reserve them for what matters).

## Test behavior, not implementation
- Assert on observable behavior and public interfaces, not private internals — so a
  refactor that preserves behavior keeps tests green.
- One behavior per test; name the test for the behavior. A failure should point straight
  at what broke.
- Cover the unhappy paths: empty, boundary, error, permission-denied, and concurrency
  cases — not just the happy path.

## See it fail first
For new behavior, watch the test fail for the *expected* reason before you make it pass.
A green test you never saw red proves nothing.

## Determinism — no flaky tests
- A flaky test is a broken test. Isolate and fix it immediately; don't paper over it with
  retries or `sleep`.
- Mock/stub external services so tests are predictable and fast. Use stable selectors in
  UI tests, not brittle positional ones.
- No hidden shared state or order-dependence between tests.

## Use the project's tooling
Run the project's existing test commands and framework. Don't introduce a new test runner
to prove one change. If a project has no tests, propose the minimal setup — ask before
adding a framework.
