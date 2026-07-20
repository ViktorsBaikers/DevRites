# Architecture Decision Records

An ADR captures **one** load-bearing decision: the context that forced it, the
decision itself, the alternatives that were rejected and *why*, and the
consequences accepted. The rejected-alternatives table is the point — it keeps
the "why not X" from being re-litigated every quarter.

`CLAUDE.md` points every agent here (`CONTEXT.md` + `docs/adr/` are the
single-context domain record). Keep this directory the source of truth for
architecture; per-feature `decisions.md` files stay scoped to that feature.

## Convention

- Filename: `NNNN-kebab-title.md`, zero-padded, monotonic.
- Frontmatter fields: **Status** (Proposed / Accepted / Superseded by NNNN),
  **Date** (YYYY-MM-DD), **Deciders** (optional).
- Sections: **Context** → **Decision** → **Alternatives considered** (a table:
  option · why rejected) → **Consequences**.
- **Guard test:** an accepted ADR that asserts a runtime invariant SHOULD have a
  named regression test — `engine/tests/adr_NNNN_*_test.go` — so the decision
  has an executable proof it still holds. Reference the ADR number in the test's
  doc comment.

## Template

```markdown
# ADR-NNNN: <title>

- **Status:** Accepted
- **Date:** YYYY-MM-DD

## Context
<the forces: what problem, what constraints, what was true at the time>

## Decision
<what we chose, stated as a present-tense rule>

## Alternatives considered
| Option | Why not |
|--------|---------|
| ... | ... |

## Consequences
<what this makes easy, what it costs, what follow-up it implies>
```

## Index

| ADR | Title | Status | Guard test |
|-----|-------|--------|-----------|
| [0001](0001-go-engine-as-control-plane.md) | Go engine as deterministic control plane | Accepted | — |
| [0002](0002-dual-host-harness.md) | Dual-host harness (Claude + Codex) | Accepted | `tests/parity_*_test.go` |
| [0003](0003-gate-model-hitl-pause.md) | Gates block as HITL pause, never crash | Accepted | `tests/adr_0003_gate_exit_code_test.go` |
| [0004](0004-state-schema-phases-sections.md) | Phase-relative section completeness | Accepted | `tests/adr_0004_required_by_phase_test.go` |
| [0005](0005-hooks-as-engine-subcommands.md) | Hooks are engine subcommands, not shell scripts | Accepted | `tests/parity_*_test.go` |
| [0006](0006-clock-seam-and-engine-ci-gates.md) | Clock seam + Go static-analysis CI gates | Accepted | `tests/adr_0006_clock_seam_test.go` |
| [0007](0007-canonical-live-workspace-filenames.md) | Canonical live workspace filenames | Accepted | `internal/migrate/migrate_test.go`, `tests/migrate_cli_test.go` |
| [0008](0008-sanctioned-engine-network-boundary.md) | Sanctioned engine network boundary | Accepted | `tests/meta_test.go` |
