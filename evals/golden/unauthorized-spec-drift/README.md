# Adversarial: unauthorized spec drift

Canonical shippable workspace plus `AC-004` in `spec.md` only. `tasks.md`,
`test-plan.md`, and `seal.md` do not reference it. Outcome evals and
`check readiness` must treat this as a failure: a requirement appeared
without a mapped slice or test.

