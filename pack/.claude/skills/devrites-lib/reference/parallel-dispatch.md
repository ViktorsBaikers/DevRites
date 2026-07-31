# Native review dispatch

Host dispatches; DevRites governs.

## Roster

Review: Spec/Code; Seal: all applicable roles.

| Reviewer | Fires | Trigger |
|---|---|---|
| `devrites-spec-reviewer` | always | acceptance/scope |
| `devrites-code-reviewer` | always | implementation correctness |
| `devrites-test-analyst` | seal | completed feature |
| `devrites-frontend-reviewer` | conditional | UI/route/screen/style/design-token change |
| `devrites-security-auditor` | conditional | input/auth/data/permission/dependency/integration/secret |
| `devrites-performance-reviewer` | conditional | performance budget or hot path/query/growing set |
| `devrites-devex-reviewer` | conditional | public API/CLI/SDK/webhook/config/error/docs/onboarding |

Also: `devrites-simplifier-reviewer`, `devrites-doubt-reviewer`,
`devrites-strategy-reviewer`, `devrites-plan-reviewer`, or `devrites-retrospector`.

## Dispatch contract

1. Freeze one candidate; give each reviewer the same spec, paths, and diff.
   Parallelize only when writes/state/locks/tool sessions/scarce resources
   (ports/processes/quotas/FDs/browsers/MCP) are compatible. Frozen read-only
   usually is; pressure requires batches or serial work.
2. Ask for each exact named agent in fresh context; omit native syntax and
   `agent_type`/`task_name`/`fork_turns` from the instruction.
3. Wait for all applicable results before reconciling.
4. Apply [`agents.md` § Result admission](standards/agents.md#result-admission)
   and dedupe by evidence/root cause. Account for all seven in
   `review.md`/`seal.md`: `Outcome:` plus admitted account, or
   `Not-applicable: <inspected-scope reason>`. Spec/Code may cite unchanged
   `review.md`; persist no dispatch telemetry.
5. Missing profile, null/failure, or malformed output is a `gap`; never
   infer approval or substitute inline review.

On exhaustion/contention: stop spawning; collect running results; batch/serialize.
Never restart/orphan the cohort or infer approval.

Reviewers are read-only; accounts store evidence, never telemetry.
