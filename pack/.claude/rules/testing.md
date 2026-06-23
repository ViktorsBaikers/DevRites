# Testing

Tests are evidence. They exist to prove behavior and to catch regressions — not to hit a
coverage number.

## Shape: a pyramid
- **Many** fast, isolated unit tests at the base.
- **Some** integration tests for how components actually fit together.
- **Few** end-to-end tests for critical user journeys (they're slow and the most
  flake-prone — reserve them for what matters).

## Completeness — every behavior, element, and flow has a test (at the right level)
"Tested" means *every* observable behavior, *every* interactive element (input field,
checkbox, radio, select, toggle, button, actionable link), and *every* user flow has at
least one **asserting** test — not that a coverage % is hit. Completeness counts behaviors
covered, not lines executed: 100% line coverage can still leave a button's click untested,
and a button with one asserting unit test is "covered" at far less than 100% lines. Chase the
behavior, not the number.

Put each test at the level that proves it cheapest and most reliably (the pyramid above):
- **Element / field behavior** — validation, required, format, min/max, toggle on→off, select
  options load + change fires, button enabled/disabled, handler runs → a **unit / component**
  test. The bulk of element coverage lives here.
- **Wiring across a few components** — form submit → store → re-render → an **integration** test.
- **A critical end-to-end journey** — login, checkout, a destructive / data-loss path → **one
  E2E** test. E2E is slow + flake-prone; reserve it for journeys, **never one-per-field**.

An interactive element or a user flow with **zero asserting test is a defect** — surfaced as an
unproven gap at `/rite-prove` and a NO-GO at `/rite-seal`, the same standing as an unproven
acceptance criterion. Assert what the element *does*, not that the markup exists.

## Assertion strength — a test that can't fail proves nothing
Completeness counts tests; strength makes them mean something. A test that passes for *any*
implementation is theatre — and it's the shape AI reaches for by default. Reject the weak forms:

- **No tautological assertions.** `expect(result).toBeDefined()` / `.not.toBeNull()` /
  `assert x is not None` pass for almost any return value. Assert the **actual value or
  observable effect** — `expect(total).toBe(42)`, the specific error thrown, the state changed,
  the row written, the event emitted.
- **Don't assert the mock.** A test that stubs a dependency to return `X` then asserts `X` came
  back tests the stub, not your code. Assert the real effect on real (or realistic) data.
- **Cover the unhappy edges, not just the happy path.** AI is strong on "valid input → success"
  and weak on empty / boundary / invalid-state / long-or-weird input — write those explicitly.
- **Prove it can fail (fault injection).** For a critical or regression path, after green,
  **break the code on purpose** (flip a comparison, drop a guard, return a constant) and confirm
  the test goes **red**. A test never seen failing against broken code is unproven — this is
  "see it fail first" extended past the happy path. Use the project's **mutation-testing** runner
  to automate it where one exists; otherwise spot-check the criticals by hand.
- **Don't mirror the implementation.** A test whose assertions restate the code under test
  (same constant, same formula, same branch) stays green even when the logic is wrong. Assert
  an **independently-derived** expected value — reasoned from the spec, not copied from the code.
- **Coverage says "ran"; mutation says "checked".** Line coverage proves a line executed, not
  that a test would catch it breaking. Where the project has a mutation runner, the changed-files
  mutation gate (`mutation-gate.sh`) certifies the suite would fail on a wrong implementation; a
  surviving mutant is a behaviour no test actually checks.

## Never weaken a failing test (test integrity)
A failing test is a signal, not an obstacle. Never delete it, skip it (`it.skip`, `xit`,
`@pytest.mark.skip`, `t.Skip`, `#[ignore]`), mark it `xfail`, narrow the run with `.only`, or
loosen its assertions to turn the suite green. A red test means one of two things: the code is
wrong (fix the code) or the test is wrong (surface it as a blocking question and get the change
agreed) — never quietly make the red go away. A test weakened to clear a gate is a **Critical**
finding; `test-integrity.sh` diffs the test files against the slice base and exits non-zero when
one is deleted, skipped, or loses assertions.

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
