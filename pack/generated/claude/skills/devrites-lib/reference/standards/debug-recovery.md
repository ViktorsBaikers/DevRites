# Debug recovery (async wait discipline)

Triggered standard for polling async readiness without blind sleep. Skill owner:
[`devrites-debug-recovery`](../../../devrites-debug-recovery/SKILL.md).

## Condition-based wait (bounded)

When waiting for async readiness (server start, job completion, browser signal):

1. Set `max_wait_ms` (default 30_000 unless the calling task or test-plan artifact specifies otherwise).
2. Poll with **condition check** — never fixed sleep as the primary strategy.
3. Capture **last signal** (last log line, HTTP status, DOM state) on timeout.
4. Record artifact: `{ condition, max_wait_ms, last_signal, outcome }`.

**Failing case:** `sleep(5)` loop with no captured last signal → recovery incomplete;
treat as flaky/unproven.

## Diagnosis write freeze

During diagnosis, writable paths are the reproduction harness plus the files
named by the current ranked hypothesis. Other product paths are frozen until
the hypothesis is confirmed. **Failing case:** while debugging one test, the
agent "fixes" lint in an unrelated package.

## Relationship to debug-recovery skill

The seven-step recovery cycle owns reproduction and fix. This standard owns the
**wait recipe** only; do not duplicate the full cycle here.
