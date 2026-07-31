# ADR-0015: Read-only root with native host orchestration

- **Status:** Accepted
- **Date:** 2026-07-30

## Context

ADR-0014 retained four engine hooks after an initial native-host cleanup. Three
of them still duplicated boundaries the hosts can enforce directly:

- `a1-guard` inferred root-versus-writer ownership from hook payloads;
- `git-guard` implemented a second destructive-action permission system; and
- `stop-gate` rechecked lifecycle state when the host stopped.

The engine also retained orchestration and convenience surfaces that Codex,
Claude, native plugins, repository search, or CI already provide. This made
DevRites implement the same work twice and coupled it to host event formats.

Current hosts can load hierarchical instructions and exact named agent
profiles, apply native permission modes per root/agent, schedule and await
subagents, retain session history, and expose skills/plugins/search directly.

## Decision

DevRites uses this boundary:

```text
AGENTS.md / skills
        ↓
Codex or Claude interprets the workflow
        ↓
the host runs the exact native custom-agent profile
        ↓
the host waits and delivers results
        ↓
the root reconciles; the engine validates state/evidence/invariants
```

The root orchestration context is source-read-only. It may invoke the explicit
DevRites commands that update `.devrites/**` and `.scratch/**`. Every specialist
is read-only except `devrites-slice-wright`, which receives native source-write
permission for one bounded task.

- Claude uses project `permissions.defaultMode: plan`; the slice-wright profile
  uses `permissionMode: acceptEdits`.
- Codex uses a read-only project profile with state/artifact path overrides;
  the slice-wright profile uses `sandbox_mode = "workspace-write"`.
- User-authorized commit, push, tag, publish, deploy, and migration actions use
  the host's normal explicit approval boundary. The engine does not implement a
  second Git authority system.

`wright-scope` is the only engine hook. It enforces the root-owned exact
`.wright-allowlist`; `reconcile` separately validates the resulting source
window. Both remain until universal per-agent worktree isolation can replace
them.

The engine owns deterministic workflow state, evidence freshness/integrity,
artifact completeness, exact write reconciliation, and secret scanning. It
does not own:

- agent dispatch, waiting, result collection, or scheduling;
- tool-call fields, file-backed agent envelopes, rollout parsing, or dispatch
  receipts;
- profile/context caches, archive/decision indexes, lane planners, or runbooks;
- progress rendering, preambles, session telemetry, health/reviewer scores, or
  fingerprints;
- guessed repository commands, index refresh wrappers, learning/convention
  scoring, or extension registries.

Lifecycle rest points call explicit readiness/seal checks instead of relying on
a Stop hook. Repository scripts and CI remain the authority for build, test,
lint, and release commands.

This ADR supersedes ADR-0014, the hook-runtime portions of ADR-0005, the custom
packet/result portions of ADR-0010 and ADR-0013, and ADR-0003's historical
StopGate clause. Their historical context remains unchanged.

## Alternatives considered

| Option | Why not |
|---|---|
| Keep `git-guard`, `a1-guard`, and `stop-gate` as defense in depth | They duplicate native permission/lifecycle boundaries and preserve hook-format coupling. |
| Keep agent packets and receipts without engine dispatch | They add a second identity/lifecycle protocol without an enforcing consumer. |
| Remove `wright-scope` now | Native per-agent sandboxes do not yet express the dynamic per-slice exact file allowlist in every supported host/worktree mode. |
| Let the root write source for small fixes | It collapses the single-writer boundary and makes A1-style inference necessary again. |

## Consequences

- Host configuration and agent profiles become the permission source of truth.
- The engine hook surface shrinks to `wright-scope`.
- Skills state semantic intent once and let the host construct native calls.
- Engine and shell tests assert observable permissions/invariants rather than
  host-internal event spelling.
- Shipping and other irreversible actions require normal host/user approval.
- Universal isolated agent worktrees are the deletion condition for
  `wright-scope`.
