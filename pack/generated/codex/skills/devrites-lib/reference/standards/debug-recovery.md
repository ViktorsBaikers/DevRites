# Debug recovery (async wait discipline)

Triggered standard for polling async readiness without blind sleep. Skill owner:
[`devrites-debug-recovery`](../../../devrites-debug-recovery/SKILL.md).

## Condition-based wait (bounded)

When waiting for async readiness (server start, job completion, browser signal):

1. Set `max_wait_ms` (default 30_000 unless artifact specifies otherwise).
2. Poll with **condition check** — never fixed sleep as the primary strategy.
3. Capture **last signal** (last log line, HTTP status, DOM state) on timeout.
4. Record artifact: `{ condition, max_wait_ms, last_signal, outcome }`.

**Failing case:** `sleep(5)` loop with no captured last signal → recovery incomplete;
treat as flaky/unproven.

## Relationship to debug-recovery skill

The seven-step recovery cycle owns reproduction and fix. This standard owns the
**wait recipe** only; do not duplicate the full cycle here.
