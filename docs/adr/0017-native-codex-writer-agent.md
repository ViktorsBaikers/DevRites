# ADR-0017: Native Codex writer-agent execution

- **Status:** Accepted
- **Date:** 2026-07-30

## Context

ADR-0016 made every Codex role read-only after live Codex 0.146.0 probes showed
that a child cannot elevate above its parent's active permission ceiling and
that child-local hooks did not reliably observe writer tool calls. That choice
preserved a hard read-only root, but it also prevented
`devrites-slice-wright` from executing its defined implementation role.

DevRites requires every catalog agent to remain installed and to execute when
its workflow step names it. Restoring an engine-owned dispatch bridge would
duplicate Codex's native spawn, wait, follow-up, and result-delivery lifecycle.

## Decision

Codex uses native custom-agent execution for every DevRites role:

- the root `devrites-orchestrator` permission profile extends `:workspace` so a
  write-capable child can run;
- only `devrites-slice-wright` uses
  `default_permissions = ":workspace"`;
- every other generated specialist uses
  `default_permissions = ":read-only"`;
- all generated Codex agents remain hook-free;
- the root never edits source or tests; it dispatches the exact wright in a
  fresh context, waits, and reconciles the result;
- the root writes the exact `.wright-allowlist`, runs
  `devrites-engine reconcile snapshot` before dispatch, and runs
  `reconcile check` and `reconcile close` afterward;
- missing required roles, unauthorized drift, or an unavailable permission
  profile stop the workflow. They never permit role substitution, inline root
  implementation, allowlist widening, or an engine dispatch bridge.

Claude's boundary is unchanged: its root remains in plan mode and the exact
slice-wright retains `permissionMode: acceptEdits` plus the `wright-scope`
pre-tool hook.

This ADR supersedes ADR-0016 and the Codex read-only-root clauses of ADR-0015.
Their live-probe findings and the native-host orchestration boundary remain
valid.

## Alternatives considered

| Option | Why not |
|---|---|
| Keep Codex source-writing disabled | It leaves `devrites-slice-wright` installed but unable to execute its required role. |
| Let the Codex root implement directly | It bypasses the exact writer role and collapses workflow separation. |
| Restore engine-owned dispatch | Codex already owns agent creation, scheduling, waiting, and result delivery. |
| Re-enable the Codex child hook | Live probes showed that it was not a dependable enforcement surface. |

## Consequences

- Every DevRites role can execute natively on Codex.
- The Codex root's no-source-writing boundary is instruction-enforced rather
  than enforced by a narrower sandbox.
- Codex exact-path enforcement is detect-and-stop through reconciliation, not
  a pre-write hook. Claude retains pre-write exact-path enforcement.
- The engine remains responsible for deterministic state, evidence, integrity,
  and reconciliation, not agent dispatch.
