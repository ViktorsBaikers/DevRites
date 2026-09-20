# ADR-0013: Native Codex agent orchestration

- **Status:** Accepted
- **Date:** 2026-07-29

## Context

ADR-0010 established fresh, depth-one specialist agents with one write-capable
slice wright. At the time, Codex custom-agent routing varied across MultiAgent
versions, so DevRites added a Go state machine that parsed prompts and rollout
files, rewrote generic `explorer`/`worker` spawns, and tracked spawn, start,
wait, stop, and result receipts.

Current Codex releases load project agents from `.codex/agents/*.toml`, apply
their `developer_instructions` and sandbox settings, and own the
spawn/wait/result lifecycle. Keeping a second lifecycle in DevRites caused
false failures when the native child started correctly but the synthetic
receipt did not match.

Codex still cannot infer DevRites workflow state or the slice wright's
per-window exact file allowlist.

## Decision

For Codex, DevRites uses the host's named custom-agent orchestration directly:

- skills request the exact `devrites-<role>` agent with a fresh context and
  wait for its result;
- Codex loads the role's generated `.codex/agents/*.toml` profile;
- read-only roles use native `sandbox_mode = "read-only"`; and
- an unavailable named role stops for human intervention instead of switching
  to a version-specific or generic compatibility lane.

Generated Codex skills and agents translate the canonical Claude-authored
paths and file formats, but do not receive a repeated compatibility preamble.
Codex-wide operating guidance lives once in the installed project
`AGENTS.md`, which Codex loads natively.

The engine does not parse Codex rollout files or track parallel
spawn/start/wait/stop receipts. `required-agent-roles` and `dispatch-waive`
metadata exist only for that deleted receipt system and are removed. The skill
body is the source of truth for when a specialist is required.

DevRites hooks remain only where they enforce DevRites-owned state or policy.
In particular, the slice wright keeps its engine-backed exact allowlist and
reconcile boundary. A wright-only start hook captures canonical state before
writer work begins. Typed `agent-packet/v1` and `agent-result/v1` envelopes,
root-owned reconciliation, lifecycle gates, and destructive-action policy also
remain.

ADR-0010 still governs fresh-context ownership, depth, typed results, and the
single-writer model. This record replaces its generic Codex fallback and
receipt-based identity implementation.

## Alternatives considered

| Option | Why not |
|---|---|
| Keep the V1/V2 receipt state machine | It duplicates the host lifecycle, depends on unstable rollout internals, and can reject a correctly started native agent. |
| Keep generic `explorer`/`worker` fallback | It weakens exact role identity and preserves version-specific branches after named custom agents became the supported interface. |
| Remove all agent-related hooks | Codex cannot derive the active reconcile snapshot or enforce the wright's dynamic exact file allowlist. |
| Author separate Claude and Codex agent rosters | It creates two mutable sources of truth; generated Codex TOML remains the smaller dual-host adapter. |

## Consequences

- Codex agent dispatch has one path: the exact named custom agent.
- DevRites no longer blocks completion on synthetic collaboration receipts.
- Reviewer safety relies on Codex's native sandbox rather than a duplicative
  Codex engine hook.
- Codex host guidance has one project-level copy instead of being injected
  into every generated skill and agent.
- The engine surface and its tests shrink substantially.
- A future Codex API change is handled in generated profiles and guidance, not
  by adding another runtime-version state machine.
