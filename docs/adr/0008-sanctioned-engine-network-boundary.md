# ADR-0008: Sanctioned engine network boundary

- **Status:** Accepted
- **Date:** 2026-07-20

## Context

ADR-0001 described the Go control plane as network-free. The binary now also
owns explicit installation/update and source-cache operations that require HTTP,
while deterministic workspace bookkeeping still must not depend on network or
model responses. The implemented boundary already isolates HTTP in one package.

## Decision

Keep deterministic workspace state, gate, hook, migration, and derivation logic
network- and model-free. Permit network access only in `internal/iohooks` for
explicit updater and source-cache operations. No other first-party package may
import network clients, and no engine package may call a model API.

## Alternatives considered

| Option | Why not |
|--------|---------|
| Forbid all network access in the binary | Would remove the verified updater and source-cache refresh behavior users already rely on. |
| Allow each consumer its own HTTP client | Expands the audit surface and weakens the executable package boundary. |
| Ship a second updater binary | Adds packaging, release, and cross-platform complexity without improving the existing package isolation. |

## Consequences

- Workspace verdicts remain deterministic and offline-capable.
- Network behavior has one auditable package and one guard test.
- The binary as a whole is not network-free; documentation must distinguish the
  control-plane core from explicit I/O commands.
