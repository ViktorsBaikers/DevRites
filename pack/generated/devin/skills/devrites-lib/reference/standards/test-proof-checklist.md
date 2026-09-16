# Test proof checklist

- New behavior has an asserting test, preferably written before implementation.
- The test was seen fail for the right reason.
- Verification commands and relevant output are recorded.
- Regression, edge, and error paths match the acceptance criteria.
- Applicable data, integration, topology, compatibility, concurrency, retry, interruption,
  and time-zone risks have discriminating cases or a recorded dismissal.
- Mocks do not remove the risk being claimed; wiring proof follows real data to the promised surface.
- A claimed pre-existing/environment-only failure has a same-command baseline.
- Passing existing tests alone is not proof of the change.

Detailed standard: `testing.md`.
