# Native review dispatch

Host dispatch; DevRites governs.

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

If the candidate diff has no matching surfaces, record
`Not-applicable: no relevant paths in diff` for `frontend-reviewer`,
`security-auditor`, and `performance-reviewer`; do not dispatch them.

## Dispatch contract

1. Freeze one candidate; give each reviewer the same spec, paths, and diff.
   Parallelize only when writes/state/locks/tool sessions/scarce resources
   (ports/processes/quotas/FDs/browsers/MCP) are compatible. Frozen read-only
   usually is; pressure requires batches or serial work.
   One fresh-context preflight before fan-out checks host configuration/evidence for
   inherited history/memory/sibling summaries. Use supported clean-context settings;
   unverified independence is a gap. Do not launch a probe agent per task;
   still validate each result's context and candidate.
2. Build each reviewer's read-set once with
   `devrites-engine context <slug> --phase <p> --role <role> --trigger <set>` and
   hand the bundle path as the packet's source list instead of a file roster.
   Evaluate the skill's trigger conditions and pass each that fires — a role's
   Trigger column maps to manifest names: `security-auditor` → `security`,
   `frontend-reviewer` → `ui`/`frontend`, `performance-reviewer` → `performance`.
   The `agents` dispatch contract needs no flag: `--role` auto-fires it
   (`auto=[agents]` in the output). The command prints `unselected=[...]`; an
   omitted applicable trigger is a gap, not a shortcut.
3. Enforce the launch barrier through the engine, not prose. Open the wave
   (`devrites-engine dispatch <slug> open --phase <p> --wave <w> --role <r>...`),
   launch every native agent nonblocking, record each returned host handle
   (`dispatch ... start --wave <w> --role <r> --handle <id>`), then `seal` —
   seal fails until every declared role has a distinct handle, which makes a
   serial launch-then-wait pattern impossible to record. Record each completion
   with `dispatch ... return`; `return` refuses before seal. `dispatch status`
   exits nonzero while the wave is open, sealed, or abandoned. `start`/`return`
   auto-record the dispatch/return metrics events — no separate ledger call. If
   an opened wave cannot obtain every handle or reach `seal`, run
   `devrites-engine dispatch <slug> abandon --wave <w> --reason <reason>` before
   recording the gap; never leave an open wave to block a retry.
4. Ask for each exact named agent in fresh context; omit native syntax and
   `agent_type`/`task_name`/`fork_turns` from the instruction.
5. Every dispatch reaches a terminal classification before reconciliation:
   `Outcome: findings`, `Outcome: no-findings`, `Outcome: gap`, or
   `unavailable: <reason>` when the step could not run at all. Reconciling over a
   partial roster is forbidden; an unavailable conditional step is recorded with its
   marker in `review.md`/`seal.md` — an omitted sub-step reads as a clean run.
   Findings later withdrawn at reconciliation stay visible: the account records
   `withdrawn: <n> (<reason>)` rather than silently dropping them.
6. Apply [[`agents.md`](standards/agents.md) § Result admission](standards/agents.md#result-admission)
   and dedupe by evidence/root cause. Account for all seven in
   `review.md`/`seal.md`: `Outcome:` plus admitted account, or
   `Not-applicable: <inspected-scope reason>`. Spec/Code may cite valid unchanged
   `review.md` under [evidence validity](candidate-integrity.md#evidence-validity); persist no dispatch telemetry.
7. Missing profile, null/failure, or malformed output is a `gap`; never
   infer approval or substitute inline review.

On exhaustion/contention: stop spawning; collect running results; batch/serialize.
Never restart/orphan the cohort or infer approval. Reviewers are read-only;
accounts store evidence, never telemetry. Same-file writers stay serial.
Capacity rejection is backpressure: retry batches; never shrink a roster.

## Cancellation and terminal reconciliation

Cancellation acknowledgement means the host accepted a request, not that the
child stopped. A transport timeout likewise leaves execution unknown. Keep the
child's handle, path/resource ownership and capacity reservation until the
native host confirms a terminal state and returned work has been reconciled.
Query or wait on that handle; do not replace the child, release its slot, or
reuse its resources while it may still run. If terminal state cannot be
established, record a gap and retain ownership. A result classification alone
does not prove host termination.

## Scale limits

The role table bounds rosters. Use roughly four concurrent read-only reviewers;
eight only with disjoint paths/resources. Refuse unbounded swarms and role
substitution: sharpen scope or use a bounded second pass when coverage is missing.
Overlapping reviewers increase duplicate findings and reconciliation work.

## Children are tools, not peers

A dispatched agent is a tool invocation: the parent owns the prompt, the
expected artifact, and admission. Reviewers do not negotiate with each other
or with the implementer. A mesh without a parent is not a DevRites profile.
**Failing case:** two reviewers agree a compromise in a shared channel and the
parent records that compromise as independent review.

Arbitration/independence → [agents.md § Independence](standards/agents.md#independence); writer batches → [`parallel-batch.md`](../../rite-build/reference/parallel-batch.md).
