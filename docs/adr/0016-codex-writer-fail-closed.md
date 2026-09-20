# ADR-0016: Codex source writing fails closed

- **Status:** Accepted
- **Date:** 2026-07-30

## Context

ADR-0013 and ADR-0015 assumed a Codex custom child could become the sole
write-capable slice-wright under a source-read-only root and that a child
`PreToolUse` hook could enforce the dynamic exact file allowlist.

Live Codex 0.146.0 probes disproved both assumptions. A child inherits the
parent turn's active sandbox and permission overrides; agent defaults such as
`:workspace` or `workspace-write` cannot elevate it. Project and agent-local
hooks also did not observe the tested child shell calls. Keeping the old profile
would therefore either block all source work or claim an enforcement boundary
that did not run.

## Decision

The root remains source-read-only on every host.

- Claude may run the exact `devrites-slice-wright` with native edit permission
  and the retained `wright-scope` exact-allowlist hook.
- Every generated Codex specialist, including `devrites-slice-wright`, is
  hook-free and `:read-only`. A Codex workflow that needs source changes stops
  for HITL or an approved writing-host handoff.
- DevRites never relaxes the Codex root, accepts a self-declared writer role,
  substitutes a generic child, or restores an engine dispatch bridge.
- Because Claude is the only shipped hook consumer, the command is simply
  `devrites-engine hook wright-scope`; there is no runtime harness selector.

This decision supersedes only the Codex writer-permission and writer-hook
clauses of ADR-0013 and ADR-0015. Their native orchestration, read-only root,
and engine-boundary decisions remain accepted.

## Alternatives considered

| Option | Why not |
|---|---|
| Relax the Codex root to workspace-write | It removes the hard root boundary and lets orchestration write source inline. |
| Keep the Codex writer hook as best effort | Live child calls bypassed it, so the claimed exact scope was false. |
| Restore engine-owned dispatch | Codex already owns native role selection, waiting, and result delivery; a bridge duplicates the host protocol. |
| Let the model follow an allowlist instruction without enforcement | A security boundary cannot depend on voluntary compliance. |

## Consequences

- Codex review/research orchestration remains native and fully read-only.
- Codex build/correction phases pause instead of silently weakening isolation.
- Claude keeps the only engine hook until a supported native isolated writer
  can express the same dynamic exact-file boundary.
- Generation and runtime tests assert observable permissions and fail-closed
  behavior, not tool-call spelling or dispatch receipts.
