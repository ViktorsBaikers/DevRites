# Acceptance proof

Map each `spec.md` criterion to `test | command | browser | judgment` evidence
and one-line reason. Run `spec-grammar.md`'s Native grammar re-read on the saved
spec before mapping; reopen after corrections. No parser/script may replace it.

Every structured WHEN/THEN scenario needs a passing asserting test. When
`test-plan.md` exists, every acceptance/gap requirement, interactive element, and
user flow needs a passing result. Missing coverage blocks.

Apply [`testing.md`'s positive, discriminating proof](../../devrites-lib/reference/standards/testing.md#positive-discriminating-proof),
including invalid-result exclusions and static/textual criterion boundaries.
Unsupported behavioral claims are `cannot_verify` and block proof.

## Silent-failure probe

When tests pass but error paths, dropped results, or partial success could hide
failure, require at least one **discriminating** check that would fail if the silent
path regressed (assert the failure surface, not only the happy path).

**Failing case:** handler returns success while logging or swallowing internally; the
suite stays green → map `cannot_verify` unless a test asserts the user-visible or
contract failure outcome.

## Backstop disposition

A `backstop` passes only on its spec-named independent held-out,
property/metamorphic, or direct behavioral check plus discriminating observed
output. Presence, test names without results, author judgment, or an expectation
copied from the same implementation logic fails.

Undefined/unavailable signal: `cannot_verify`, evidence
`insufficient_spec: <missing fact or evidence surface>`, then Spec Drift; never
judgment or pass.

## Applicable system-risk proof

Reconcile `spec.md`'s applicability map with the final diff. For each triggered owner:

- `repository-topology.md`: prove commands ran at every affected root and both provider
  and consumer used the same canonical contract; identify generated/vendor sources.
- `data-integrity.md`: prove declared invariants plus relevant duplicate/retry,
  concurrency, interruption/resume, old/new coexistence, tenant isolation, and
  rollback/forward-recovery cases with recorded scale/counts.
- `integration-reliability.md`: prove relevant invalid/partial response, auth failure,
  timeout/unknown outcome, rate limit/outage, duplicate/order, poison/backlog, cache/partition,
  degradation, and recovery behavior.

An evidence-backed dismissal is valid for an irrelevant case. An applicable case with no
executable proof surface is `cannot_verify`; do not infer it from an implementation review
or a mock that cannot produce the risk.

## Critical-path assertion strength

For regression-Critical, irreversible, and data-loss paths, require
[`testing.md`'s safe perturbation](../../devrites-lib/reference/standards/testing.md#safe-perturbation).
Missing evidence is `cannot_verify`; a green test on broken code is unproven.
Pure transforms also get a round-trip or
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

For every critical link, apply safe perturbation to the load-bearing input/link and observe the
promised surface fail. Existence/registration/spy-call evidence is insufficient when a
different path can coincidentally produce the expected result.

## Baseline and environment claims

Classify a failure as pre-existing or environment-specific only with a dated
before-candidate result for the same command, working directory, prerequisites, and
material environment. Otherwise record it as an unresolved candidate result. For time/local
behavior, bind the instant, time zone, locale, and relevant old/new configuration.
