# ADR-0002: Dual-host harness (Claude + Codex)

- **Status:** Accepted
- **Date:** 2026-07-08 (backfilled)

## Context

Each AI coding host (Claude Code, Codex, and — in adjacent tools — Cursor,
Cline, Copilot) speaks its own hook dialect: different stdin shapes, exit-code
conventions, and settings surfaces. The lifecycle logic (orient, gate, hook
decisions) is host-independent; only the edge translation differs.

## Decision

Support exactly **two** hosts today — `claude` and `codex` — behind a thin
`internal/harness` adapter that translates each host's hook stdin/exit
conventions into the shared `orient` + `gate` core. Host support is
enumerated in code, not open-ended. `harness-matrix --check` keeps
`docs/harness-compliance.md` in sync with the adapters (drift is a CI failure).

## Alternatives considered

| Option | Why not |
|--------|---------|
| A generic declarative capability/adapter registry now | Real ceiling-raiser for N hosts, but a large refactor unjustified at N=2. Recorded as a Proposed follow-up, not built. |
| Claude-only | Codex users are already real; single-host would strand them. |
| Per-host forks of the logic | Duplicates the lifecycle core across edges — the thing the adapter exists to prevent. |

## Consequences

- Adding a host today means editing `harness.go` — acceptable friction at N=2,
  the calcification risk rises with N (hence the Proposed registry follow-up).
- The shared core is tested once; only edge translation is host-specific.
- `harness-matrix --check` makes host-support drift visible in CI.
