# ADR-0033: `/rite-fast` ships its own agents and its own writer

- **Status:** Accepted
- **Date:** 2026-09-26

## Context

Mid-size work had two routes. `/rite-quick` covers one small reversible change. The
full lifecycle adds strategy review, a vetted plan, and review plus proof after every
slice. Users asked for a lane in between: one spec-and-plan step with clarifying
questions, up to ten slices built with no gate between them, one completeness check
with gap fixes, then review, polish, and proof.

The lifecycle agents are shaped for per-slice gates and engine workspace artifacts
(`state.md`, `tasks.md`, `touched-files.md`, readiness bindings). `devrites-slice-wright`
runs writer-safe proof on every slice, which is the step the fast lane removes.
ADR-0010, ADR-0015, ADR-0017, ADR-0018 and ADR-0019 keep one writer among `devrites-*`
specialists; ADR-0031 already scoped that rule for `/overhaul`.

## Decision

- `/rite-fast` is an explicit-only skill (`disable-model-invocation: true`). It stands
  outside the engine lifecycle: its record is `spec.md`, `plan.md`, and `evidence.md`
  under `.devrites/work/<slug>/`, with no `state.md`, no `.devrites/ACTIVE`, and no
  seal. No engine change is needed.
- The pack ships four agents that only `/rite-fast` dispatches: `fast-planner`,
  `fast-checker`, and `fast-critic` (read-only, `permissionMode: plan`), and
  `fast-builder` (write-capable in the exact paths of its contract,
  `permissionMode: acceptEdits`). `fast-builder` runs no test suite; the skill runs one
  check after all slices and proof last.
- `validate-agent-composition.py` accepts `fast-builder` as write-capable when it states
  that mode. Codex gives it `default_permissions = ":workspace"`; omp, pi, and Devin give
  it the write tool set. The other `fast-*` agents stay read-only on every host.

## Consequences

- The one-writer clauses of the lifecycle ADRs govern `devrites-*` specialists only;
  `/overhaul` (ADR-0031) and `/rite-fast` each own one more writer.
- A `/rite-fast` run is not visible to `/rite-status`, `orient`, `next`, or `handoff`.
  It resumes from its own `plan.md` ledger.
- Four more agents are installed, updated, and removed with the pack. `/rite-fast`
  stops when they are missing and no fresh context is available.

Verified by `scripts/validate-agent-composition.py`, `tests/rite-fast-contract-test.sh`,
and the host generators' `fast-builder` branches.
