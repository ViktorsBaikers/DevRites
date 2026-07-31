# Acceptance proof

Map each `spec.md` criterion to `test | command | browser | judgment` evidence
and one-line reason. Run `spec-grammar.md`'s Native grammar re-read on the saved
spec before mapping; reopen after corrections. No parser/script may replace it.

Every structured WHEN/THEN scenario needs a passing asserting test. When
`test-plan.md` exists, every acceptance/gap requirement, interactive element, and
user flow needs a passing result. Missing coverage blocks.

Behavioral proof MUST be positive, discriminating, and observed under `testing.md`:
assertion/output changes when behavior is absent or wrong. Skipped, focused,
filtered, pending, zero-test, assertion-free, tautological, or unexecuted results
do not count. Build/compile/typecheck/lint prove only their static criterion;
discriminating shell assertions and golden/text comparisons may prove text/CLI.

## Backstop disposition

A `backstop` passes only on its spec-named independent held-out,
property/metamorphic, or direct behavioral check plus discriminating observed
output. Presence, test names without results, author judgment, or an expectation
copied from the same implementation logic fails.

Undefined/unavailable signal: `cannot_verify`, evidence
`insufficient_spec: <missing fact or evidence surface>`, then Spec Drift; never
judgment or pass.

## Critical-path assertion strength

For regression-Critical, irreversible, and data-loss paths, inject a small break,
confirm the covering test turns red, then revert; use the existing mutation tool.
A green test on broken code is unproven. Pure transforms also get a round-trip or
metamorphic property check when applicable.

## Runtime observability branch

For endpoints, jobs, consumers, integrations, user flows, or new error paths,
trigger failure and observe the required log/metric/trace per `observability.md`;
record it. Skip pure internal, docs, config, and type-only changes.

## Developer-facing branch

For public API/CLI/SDK/webhook/config/error/onboarding, execute the real entry on
clean state, measure time-to-hello-world, and capture verbatim failure output.
Browser-capture docs/quickstarts when applicable. Write the scorecard to
`devex.md` and headline evidence to `evidence.md`.

## Wiring branch

Exercise/follow every key `plan.md` link in the assembled feature; record each as
`EVID-###`, or record none declared. An unwired link blocks.
