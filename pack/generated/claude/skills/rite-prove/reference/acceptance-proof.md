# Acceptance proof

Map every `spec.md` acceptance criterion to evidence and label its proof class:
`test`, `command`, `browser`, or `judgment` with a one-line reason. Re-run the
structured grammar gate before mapping scenarios:

```bash
devrites-engine spec-validate ".devrites/work/<slug>" --against .devrites/specs
```

Each structured WHEN/THEN scenario needs a passing asserting test. If `test-plan.md`
exists, every acceptance mapping, per-gap requirement, interactive element, and user
flow needs a passing result. Missing coverage is an unproven blocker.

## Critical-path assertion strength

For regression-Critical, irreversible, and data-loss paths, fault-inject a small
behavior break and confirm the covering test goes red, then revert. Use the existing
mutation tool when present. Run:

```bash
devrites-engine test-integrity
devrites-engine mutation-gate
```

A test that stays green on broken code is unproven. Pure transforms also get a
round-trip or metamorphic property check when applicable.

## Runtime observability branch

For endpoints, jobs, consumers, integrations, user-facing flows, or new error paths,
apply the on-call test from `observability.md`: trigger the failure path and observe the
required log, metric, or trace emit. Record the observation. Skip pure internal, docs,
config, and type-only changes.

## Developer-facing branch

For a public API, CLI, SDK, webhook, config contract, error path, or getting-started
flow, execute the real entry point on clean state. Measure time-to-hello-world and
capture verbatim failure output. Browser-capture docs/quickstarts when applicable.
Write the measured scorecard to `devex.md` and headline evidence to `evidence.md`.

## Wiring branch

Walk every key link declared by `plan.md` and exercise or follow it in the assembled
feature. Record each as `EVID-###`; if none are declared, record that fact. An unwired
link is a blocker.
