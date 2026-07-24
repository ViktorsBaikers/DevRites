# ADR-0010: Agent-first fresh-context orchestration

- **Status:** Accepted
- **Date:** 2026-07-23

## Context

Lifecycle skills were still doing large searches, planning, proof, and repair work in the
orchestrator's chat context. That increased context pressure and made delegation
host-dependent: Codex could treat an unavailable custom role as if no spawn primitive
existed, while one skill described an unnamed nested writer. Reviewer hooks also guarded
only selected Bash commands, and the slice writer claimed its own file scope.

The design review for
[ADR-0009](0009-prebuild-decision-coverage-and-readiness.md) also retained
file-backed fresh task packets, separated planner/checker/executor responsibilities,
bounded research fan-out, explicit concurrency limits, typed reconciliation, and
restartable artifact boundaries. We reject persistent swarms and parallel shared-tree
writers.

## Decision

1. Public `rite-*` skills remain the root orchestrators. They alone own human interaction,
   product/risk decisions, canonical `.devrites/` writes, phase routing, reconciliation,
   gates, and irreversible actions.
2. Heavy bounded work uses ephemeral fresh-context leaf agents at depth one. Normally no
   more than three read-only leaves run concurrently; file overlap, dependencies, or
   process/file-descriptor pressure reduce the wave to serial execution.
3. Add three read-only roles:
   `devrites-evidence-scout`, `devrites-plan-drafter`, and
   `devrites-proof-runner`. Existing independent reviewers remain critics, not authors.
4. `devrites-slice-wright` remains the only source/test writer. Writers never share a
   working tree; accepted proof, polish, or review corrections become one bounded wright
   contract.
5. Dispatch uses file-backed `agent-packet/v1` and `agent-result/v1` envelopes with a run
   ID, authoritative artifact paths, exact scope, budgets, immutable base/diff/touched-file
   hashes, side effects, and typed terminal status. The orchestrator rejects malformed,
   stale, or out-of-scope returns before persistence.
6. Host fallback is ordered: named project role; generic fresh `explorer`/`worker`
   reading the same role contract only when the host preserves its enforced read/write
   boundary (or isolates the writer); inline discipline when no safe fresh-context rung
   exists. Inline output is labelled non-independent and cannot silently satisfy an
   independence gate.
7. Agent security is fail-closed for declared leaf runs. Every role except the exact wright
   is read-only; the wright consumes an orchestrator-created exact path allowlist. Nested
   dispatch, self-claimed scope, installs, git publication, live migrations, and
   `.devrites/` writes are forbidden to leaves.
8. Objective failures use bounded technical recovery without asking the human to authorize
   another attempt. Only product/scope/policy choices, irreversible risk, and human-only
   access/actions create a human gate.

## Alternatives considered

| Option | Why not |
|---|---|
| Keep heavy work in the main chat | Reintroduces context accumulation and makes fresh independent judgment optional. |
| One agent per lifecycle phase with write access | Duplicates orchestration authority and creates competing canonical writers. |
| Persistent team/swarm with nested workers | Adds mailbox/watchdog state, amplifies resource pressure, and violates depth-one ownership. |
| Parallel source writers on one tree | Makes authorship, implicit decisions, rollback, and reconciliation ambiguous. |
| Inline fallback whenever a custom role is unavailable | Confuses “role unavailable” with “spawn unavailable” and falsely labels self-review independent. |

## Consequences

- Main-chat context holds routing and decisions, while agents reread canonical artifacts
  from scratch and return compact typed results.
- Delegation has deterministic authority, fallback, retry, and stale-result behavior on
  both Claude and Codex.
- Read-only fan-out can improve throughput, but resource limits are an upper bound rather
  than a target.
- Agent profiles, host generation, hooks, packet schemas, and composition tests must evolve
  together; generated host artifacts remain derived from `pack/.claude`.
