# ADR-0003: Gates block as a HITL pause, never a crash

- **Status:** Accepted
- **Date:** 2026-07-08 (backfilled)

## Context

A gate is a deterministic completeness check — e.g. "does this feature have the
sections required to leave the current phase?" When a gate is not satisfied the
engine must stop the agent, but *how* it stops matters. A crash (non-zero,
stderr stack) reads as a tool failure and invites the agent to retry blindly or
route around it. The intended meaning is the opposite: **the work is
incomplete, pause and involve the human.**

## Decision

A blocked gate is a structured **human-in-the-loop pause**: a specific
"missing X" message on stdout and a distinct, reserved **exit code 3** — never a
crash, never a generic non-zero. Gates are **transition-fired** (they run only
when explicitly invoked at a phase boundary, not on every tool call). Two
kinds: `Readiness` (sections needed to leave the current phase) and `Seal` (the
full seal set). `StopGate` enforces the rest-point invariant: a feature claiming
completion with empty proof, or with `.red` set, cannot end the turn.

## Alternatives considered

| Option | Why not |
|--------|---------|
| Exit 1 / crash on an unmet gate | Indistinguishable from a real error; agents retry or route around it instead of pausing for the human. |
| Run gates on every tool call | Turns a transition check into per-action overhead and noise; gates are boundary events. |
| Auto-advance past a soft-missing section | Defeats the completeness guarantee the gate exists to provide. |

## Consequences

- Exit 3 is a reserved, load-bearing contract — the harness and hooks branch on
  it. Its guard test (`tests/adr_0003_gate_exit_code_test.go`) locks it.
- Blocks are legible: the agent (and human) see exactly which section is missing.
- Gate authors must classify a stop as "incomplete" (exit 3) vs a true error.
