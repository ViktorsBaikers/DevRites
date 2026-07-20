# ADR-0005: Hooks are engine subcommands, not shell scripts

- **Status:** Accepted
- **Date:** 2026-07-08 (backfilled; see commits porting `reviewer-readonly`,
  `subagent-orient`, and the guard hooks into the Go engine)

## Context

Lifecycle hooks (orient on session start, readonly-guard for reviewers,
stop-gate, statusline, redwatch, …) began as standalone `pack/.claude/hooks/*.sh`
scripts. Shell hooks drift from the engine's own state logic, are hard to test
in isolation, behave differently across the two hosts, and duplicate parsing the
engine already does.

## Decision

Hooks are **subcommands of the one Go binary**: `devrites-engine hook <id>`.
The pack ships only `hooks.json` wiring plus host settings; every hook's logic
lives in the engine, sharing the `orient`/`gate`/`state` core. A control plane
selects which hooks fire: `DEVRITES_HOOK_PROFILE` (minimal / standard / strict)
and `DEVRITES_DISABLED_HOOKS`. Each hook has a golden parity test
(`tests/parity_*_test.go`) pinning its stdout + exit across both hosts.

## Alternatives considered

| Option | Why not |
|--------|---------|
| Keep hooks as `.sh` scripts | Drift from engine state logic, per-host divergence, poor testability, duplicated parsing. |
| One monolithic hook entry point | Loses per-hook enable/disable and per-hook golden tests. |
| Node hook runtime | Reintroduces a runtime dependency ADR-0001 deliberately avoids. |

## Consequences

- Hooks are unit-testable Go with golden parity snapshots, not shell fixtures.
- One implementation serves both hosts; the harness (ADR-0002) handles edge
  translation.
- Profiles make the hook surface tunable without editing wiring.
- The pack's `hooks/*.sh` files are removed; `hooks.json` is the only pack-side
  hook artifact.
