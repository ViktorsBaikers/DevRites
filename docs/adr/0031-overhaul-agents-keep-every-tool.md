# ADR-0031: `/overhaul` ships its own agents that keep every tool

- **Status:** Accepted
- **Date:** 2026-09-25

## Context

The explicit-only `/overhaul` skill dispatched its eight roles (recon, frontend
engineer, backend engineer, boundary, craft, specialist, implementer, verifier) as
generic subagents told to read `skills/overhaul/roles/<role>.md`. Registered agents
give each role a named dispatch target on every host, carry the role contract as the
agent's own instructions, and let a host pick a model per role.

The DevRites lifecycle keeps one writer: ADR-0010, ADR-0015, ADR-0017, ADR-0018 and
ADR-0019 make `devrites-slice-wright` the only writable specialist. Every `/overhaul`
role also writes, because each one records a receipt and evidence in the run area,
and the implementer edits approved target paths. `/overhaul` has its own approval
gate, path contracts, snapshot fingerprints and admission tool
(`devrites-engine overhaul`), independent of the lifecycle.

## Decision

- The pack ships `pack/.claude/agents/overhaul-{recon,frontend-engineer,
  backend-engineer,boundary,craft,specialist,implementer,verifier}.md`. They are
  dispatched only by `/overhaul`, and their composition line names `/overhaul` as the
  caller.
- These agents keep every tool: no `tools:` allowlist and no `permissionMode:`
  override, so Claude Code applies the user's own permission mode. Codex gives them
  `default_permissions = ":workspace"`; omp, pi and Devin give them the full write
  tool set. Host permission prompts stay authoritative and are never bypassed.
- Read-only behavior for reviewer roles stays an instruction in the agent body. The
  coordinator detects writes by comparing snapshot fingerprints around every wave and
  admits patches only when the observed changed paths equal the approved contract.
- The role files are removed; each agent file is the single source of its role.
- `validate-agent-composition.py` accepts `overhaul-*` agents as write-capable when
  they state their mode. The lifecycle's one-writer rule is unchanged for every
  `devrites-*` agent.

## Consequences

- The ADR-0010/0015/0017/0018/0019 writer clauses now read "one writer among
  `devrites-*` specialists"; they do not govern `/overhaul`.
- Eight more agents are installed, updated and removed with the pack. The installer
  keeps an existing unmanaged `.claude/agents/overhaul-*.md`, and `/overhaul` stops
  with `BLOCKED_ENVIRONMENT` when a required agent is missing (for example with
  `--no-agents`).
- Instruction-level read-only modes give weaker isolation than host allowlists. The
  user chose full tool access; fingerprints and path-contract admission remain the
  enforcement.

Verified by `scripts/validate-agent-composition.py`, `tests/codex-agent-generation-test.sh`
and the host generators' `overhaul-*` branches.
