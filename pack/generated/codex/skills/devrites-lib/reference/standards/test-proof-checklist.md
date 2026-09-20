# Test proof checklist

> Applies when: compact sweep before claiming test proof.

- New behavior has an asserting test, preferably written before implementation.
- The test was seen fail for the right reason.
- Verification commands and relevant output are recorded.
- Regression, edge, and error paths match the acceptance criteria.
- Applicable data, integration, topology, compatibility, concurrency, retry, interruption,
  and time-zone risks have discriminating cases or a recorded dismissal.
- Mocks do not remove the risk being claimed; wiring proof follows real data to the promised surface.
- A claimed pre-existing/environment-only failure has a same-command baseline.
- A runnable check that failed, timed out, was skipped, or could not start is a named blocker —
  fix the environment and rerun, or record the concrete blocker; never folded into "mostly green".
- Facts come from executed runs; anything reasoned but unexecuted is labeled an assumption.
- Passing existing tests alone is not proof of the change.

Detailed standard: `testing.md`.
