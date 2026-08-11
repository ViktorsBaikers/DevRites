# Testing

Tests are evidence. They exist to prove behavior and to catch regressions, not to hit a
coverage number.

## Shape: a pyramid
- **Many** fast, isolated unit tests at the base.
- **Some** integration tests for how components fit together.
- **Few** end-to-end tests for critical user journeys (they're slow and the most
  flake-prone: reserve them for what matters).

Size by **resource cost**, not by folder. A **small** test runs in one process, no I/O: the
base of the pyramid, and the only size allowed to be many. A **medium** test touches the local
machine (filesystem, a container, `localhost`). A **large** test crosses the network or a real
external service: the flaky, slow tier E2E lives in. Push every behavior to the smallest size
that can still prove it; a behavior provable small but written large is a slow test you'll
learn to skip.

## Completeness: every behavior, element, and flow has a test (at the right level)
"Tested" means *every* observable behavior, *every* interactive element (input field,
checkbox, radio, select, toggle, button, actionable link), and *every* user flow has at
least one **asserting** test, not that a coverage % is hit. Completeness counts behaviors
covered, not lines executed: 100% line coverage can still leave a button's click untested,
and a button with one asserting unit test is "covered" at far less than 100% lines. Chase the
behavior, not the number.

Poor test coverage or missing tests in a brownfield area is baseline risk, not permission to leave
the changed behavior unproven. Add the smallest surface-anchored regression test and separate
pre-existing failures with same-command evidence.

Acceptance and tests are **surface-anchored**: assert the outermost surface the intent names. If the feature promised an API response, assert the API response; a database row behind it is supporting evidence, not proof.

Put each test at the level that proves it cheapest and most reliably (the pyramid above):
- **Element / field behavior:** validation, required, format, min/max, toggle on→off, select
  options load + change fires, button enabled/disabled, handler runs → a **unit / component**
  test. The bulk of element coverage lives here.
- **Wiring across a few components:** form submit → store → re-render → an **integration** test.
- **A critical end-to-end journey:** login, checkout, a destructive / data-loss path → **one
  E2E** test. E2E is slow + flake-prone; reserve it for journeys, **never one-per-field**.

An interactive element or a user flow with **zero asserting test is a defect**: surfaced as an
unproven gap at `$rite-prove` and a NO-GO at `$rite-seal`, the same standing as an unproven
acceptance criterion. Assert what the element *does*, not that the markup exists.

## Assertion strength: a test that can't fail proves nothing
Completeness counts tests; strength makes them mean something. A test that passes for *any*
implementation is theatre, and it's the shape AI reaches for by default. Reject the weak forms:

### Positive, discriminating proof

A behavioral requirement is proved only by observed positive, discriminating evidence that
would fail if the behavior were absent or wrong. Skipped, focused, filtered, or pending tests,
zero-test runs, assertion-free tests, tautologies, unexecuted commands, and success inferred
only from exit status cannot prove behavior.

Build, compile, typecheck, and lint prove only their corresponding static criterion, never
runtime behavior. Explicit shell assertions and golden/text comparisons remain valid when the
criterion genuinely concerns a textual or command-line artifact and the assertion
discriminates the required result.

- **No tautological assertions.** `expect(result).toBeDefined()` / `.not.toBeNull()` /
  `assert x is not None` pass for almost any return value. Assert the **actual value or
  observable effect**: `expect(total).toBe(42)`, the specific error thrown, the state changed,
  the row written, the event emitted.
- **Don't assert the mock.** A test that stubs a dependency to return `X` then asserts `X` came
  back tests the stub, not your code. Assert the real effect on real (or realistic) data.
- **Cover the unhappy edges, not just the happy path.** AI is strong on "valid input → success"
  and weak on empty or missing input, omitted required fields, boundary values, invalid state,
  and long-or-weird input: write those explicitly and assert the promised rejection/default.
- **Prove it can fail.** For a critical or regression path, break the code deliberately and confirm the test goes red; use the project's mutation runner when one exists.
- **Don't mirror the implementation.** A test whose assertions restate the code under test
  (same constant, same formula, same branch) stays green even when the logic is wrong. Assert
  an **independently-derived** expected value: reasoned from the spec, not copied from the code.
- **Don't grep the source.** A test that reads a source file as text and asserts it
  `contains("foo")` proves a string exists, not that the behavior works. It passes on dead code
  and breaks on a harmless rename. **Execute** the code and assert its effect; reserve text
  scanning for genuinely textual artifacts (generated output, a committed manifest, a golden snapshot).
- **Coverage says "ran"; mutation says "checked".** Line coverage proves a line executed, not
  that a test would catch it breaking. Where the project has a mutation runner,
  use its documented command; a surviving mutant is a behaviour no test checks.

## Never weaken a failing test (test integrity)
A failing test is a signal, not an obstacle. Never delete it, skip it (`it.skip`, `xit`,
`@pytest.mark.skip`, `t.Skip`, `#[ignore]`), mark it `xfail`, narrow the run with `.only`, or
loosen its assertions to turn the suite green. A red test means one of two things: the code is
wrong (fix the code) or the test is wrong (surface it as a blocking question and get the change
agreed): never quietly make the red go away. A test weakened to clear a gate is a **Critical**
finding. The root's diff review and dedicated test analysis compare the
candidate with its base and reject deleted, skipped, focused, or weakened tests.

## The verification gap: green, but the test doesn't prove the change
Diff review catches reaching green by *weakening* a test. This catches the quieter failure: a
test that was never touched, is fully green, and still doesn't exercise the behavior that changed.
A passing suite is not proof the *change* is proven: the suite could pass identically with the
change reverted. Run this trace for each behavioral change in the diff:

1. **Screen for behavioral change.** What in this diff changes an observable output, not just its
   shape? Treat a dependency bump, a build/config edit, or a data change as behavioral too: the
   load-bearing change often isn't the line that looks important.
2. **Name what changed.** State the old behavior and the new one in one sentence each.
3. **Trace to the consumer.** Find where the changed code is called from.
4. **Inspect the consumer's test.** Does an *asserting*, surface-anchored test drive that consumer through the
   **new** behavior, not merely execute the path, and not assert the old expectation still?
5. **Confirm the gap is real.** A finding is: `<change at file:line>` has no test that would go red
   if it regressed: cite the test that *should* cover it and show what it misses. No general
   advice; a gap you can't point at is not a finding (the verification gate applies).

A changed behavior with no test that would fail on its regression is an **unproven gap**: the same
standing as an untested element or an unproven acceptance criterion. A source
change with no test-file delta is a pointer to run this trace, never a verdict
on its own.

## DAMP over DRY in tests
Test code optimizes for a different reader than production code: someone staring at a failure who
needs the whole scenario in front of them. A test should read like a spec: arrange, act, assert,
visible in one screen. Prefer a little repetition over a clever shared helper that hides what the
test exercises; **D**escriptive **A**nd **M**eaningful **P**hrases beat **D**on't **R**epeat **Y**ourself
here. (This trades against production `coding-style.md` reuse-first on purpose: a shared fixture
that makes the reader scroll away to understand the case has cost more than the duplication saved.)

## Test doubles: reach for the real thing first
Prefer, in order: **real > fake > stub > mock**. Use the real collaborator when it's fast and
deterministic; a **fake** (an in-memory implementation that honors the contract) when it isn't; a
**stub** for a canned return; a **mock** (asserting *how* it was called) only at a true boundary
you own. Over-mocking is the failure mode: a suite stitched from mocks passes while production
breaks, because it tested the stubs, not the code (see "Don't assert the mock" above).

| Double it | Leave it real |
|---|---|
| The database, network, filesystem, clock, randomness | Pure functions and business logic |
| A third-party API or paid/rate-limited service | Your own internal utilities and transforms |
| Anything non-deterministic or slow | Validation and mapping under test |

## Prove the risk the design actually introduces

Select cases from the accepted spec and applicable standards, not a generic count:

- Durable data changes apply [`data-integrity.md`](data-integrity.md): invalid write,
  duplicate/retry, concurrent update, interrupted migration/backfill, old/new version
  coexistence, tenant denial, and rollback/forward recovery as relevant.
- API/webhook/queue/cache work applies
  [`integration-reliability.md`](integration-reliability.md): invalid/partial response,
  auth failure, timeout/unknown outcome, rate limit, outage, duplicate, out-of-order,
  poison/backlog, and stale-cache/partition behavior as relevant.
- Multi-root/service work applies [`repository-topology.md`](repository-topology.md):
  provider and consumer both consume the canonical contract and run from their proven
  roots. One member's green suite cannot prove another member.
- Compatibility/delivery work drives both feature-flag states and old/new caller or
  schema combinations. Migration-before-code and code-before-migration order each need a
  declared expected result.

Dismiss an irrelevant case with a reason; silently omitting an applicable case is a gap.

## False-positive and coincidental-reliance checks

- **Trace cause to effect.** A test proves wiring only when real input reaches the new
  implementation and its distinct output reaches the promised surface. Registration,
  file existence, a spy call, or a fixture containing the expected text can pass while
  production still uses the old path.
- **Change the load-bearing input or implementation.** For a critical link, perturb the
  input or break the link and observe the surface assertion fail. If another path happens
  to produce the same output, the test relies on coincidence and needs a discriminating
  fixture/assertion.
- **Do not mock away the named risk.** A timeout test whose mock cannot time out, a
  transaction test without transaction boundaries, or a tenant test with one tenant is
  mislabeled coverage. Use a contract-capable fake, local integration surface, sandbox,
  or authorized real boundary appropriate to the risk.
- **Baseline environmental claims.** "Pre-existing", "only fails in CI", or "works in one
  region/time zone" requires a before-candidate run or other dated baseline on the same
  command and environment. Without it, classify the result as unresolved.

## Determinism: no flaky tests
- A flaky test is a broken test. Isolate and fix it immediately; don't paper over it with
  retries or `sleep`.
- Mock/stub external services so tests are predictable and fast. Use stable selectors in
  UI tests, not brittle positional ones.
- No hidden shared state or order-dependence between tests.
- **Seam the clock; never read it raw in a tested path.** Route wall-clock reads through one
  injectable seam (an env override like `DEVRITES_NOW`, or an injected clock) so time/date-derived
  output is pinned in tests. A raw `time.Now()` feeding output makes a golden snapshot rot at the
  next day boundary: green today, red tomorrow, for no code change. The test must control time so
  its result depends on behavior, not when the suite runs.
- Pin the time zone and locale independently of the instant. Cover offset/date rollover,
  daylight-saving gap/fold where the product supports it, and serialization round trips;
  a UTC-only unit test does not prove local-calendar behavior.
- **No elapsed-time assertions.** `assert elapsed < 200ms` / `took` under a threshold tests the
  CI runner's load, not your code: flaky by construction. Assert the *result*, not the duration;
  for ordering or concurrency use a deterministic signal (a fake clock, a channel), never a `sleep`.

## Use the project's tooling
Run the project's existing test commands and framework. Don't introduce a new test runner
to prove one change. If a project has no tests, propose the minimal setup: ask before
adding a framework.
