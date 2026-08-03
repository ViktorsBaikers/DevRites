# Architecture decision records

An ADR records one architectural decision, the context behind it, the options
that were rejected, and the consequences the team accepted. Recording why an
option lost prevents the same debate from starting again later.

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
  named regression test such as `engine/tests/adr_NNNN_*_test.go`. Reference the
  ADR number in the test's doc comment so the decision has executable proof.

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
| [0001](0001-go-engine-as-control-plane.md) | Go engine as deterministic control plane | Accepted in part; all-state-transition/derivation scope superseded by 0024 | none |
| [0002](0002-dual-host-harness.md) | Dual-host harness (Claude + Codex) | Accepted | `tests/parity_*_test.go` |
| [0003](0003-gate-model-hitl-pause.md) | Gates block as HITL pause, never crash | Accepted | `engine/tests/adr_0003_gate_exit_code_test.go` |
| [0004](0004-state-schema-phases-sections.md) | Phase-relative section completeness | Accepted | `engine/tests/adr_0004_required_by_phase_test.go`, `engine/internal/state/state_test.go` |
| [0005](0005-hooks-as-engine-subcommands.md) | Hooks are engine subcommands, not shell scripts | Accepted; active hook clause superseded by 0018 | historical |
| [0006](0006-clock-seam-and-engine-ci-gates.md) | Clock seam + Go static-analysis CI gates | Accepted in part; next-qid seam/guard superseded by 0024, CI clauses retained | historical |
| [0007](0007-canonical-live-workspace-filenames.md) | Canonical live workspace filenames | Accepted; unsupported alias/migration clauses superseded by 0022 | `engine/internal/state/state_test.go`, `engine/internal/lib/cursor_compat_test.go` |
| [0008](0008-sanctioned-engine-network-boundary.md) | Sanctioned engine network boundary | Accepted in part; engine network allowance superseded by 0024 | `engine/tests/meta_test.go` |
| [0009](0009-prebuild-decision-coverage-and-readiness.md) | Pre-build decision coverage and implementation readiness | Accepted; semantic engine-gate implementation superseded by 0022 | historical |
| [0010](0010-agent-first-fresh-context-orchestration.md) | Agent-first fresh-context orchestration | Accepted | `tests/native-orchestration-contract-test.sh`, `tests/codex-agent-generation-test.sh` |
| [0011](0011-define-plan-transition-rights.md) | Separate Define authoring from the Plan checkpoint | Accepted | `engine/internal/state/state_test.go`, `engine/tests/adr_0011_define_plan_test.go` |
| [0012](0012-semantic-workspace-upgrades.md) | Separate semantic workspace upgrades from structural migration | Accepted in intent; detection and repair mechanics superseded by 0025 | historical |
| [0013](0013-native-codex-agent-orchestration.md) | Native Codex agent orchestration | Accepted; Codex writer clause superseded by 0016 | `tests/codex-agent-generation-test.sh`, `tests/codex-generator-test.sh` |
| [0014](0014-native-host-hook-boundary.md) | Native host capability boundary | Superseded by 0015 | historical |
| [0015](0015-read-only-root-native-orchestration.md) | Read-only root with native host orchestration | Accepted; Codex root/writer clauses superseded by 0017; exact-path/hook/reconcile clauses superseded by 0018 | `engine/internal/install/install_test.go`, `tests/codex-agent-generation-test.sh` |
| [0016](0016-codex-writer-fail-closed.md) | Codex source writing fails closed | Superseded by 0017 | historical |
| [0017](0017-native-codex-writer-agent.md) | Native Codex writer-agent execution | Accepted; exact-path/reconcile clauses superseded by 0018 | `tests/codex-agent-generation-test.sh`, `tests/codex-generator-test.sh`, `tests/codex-runtime-smoke.sh` |
| [0018](0018-native-sandbox-instruction-writer-boundary.md) | Native sandbox and instruction writer boundary | Accepted in part; native sandbox/writer boundary retained, superseded clauses resolved by 0022 | `engine/internal/install/install_test.go`, `tests/codex-agent-generation-test.sh` |
| [0019](0019-native-boundary-with-deterministic-gates.md) | Native boundary with deterministic gates | Accepted in part; semantic gates superseded by 0022, native writer and fresh approval retained | `engine/internal/gate/gate_test.go` |
| [0020](0020-thin-engine-native-orchestration-boundary.md) | Thin engine and native host orchestration | Accepted in part; engine-surface/migration scope superseded by 0022, native orchestration retained | `tests/native-orchestration-contract-test.sh`, `engine/root_routing_test.go` |
| [0021](0021-observable-workspace-compatibility.md) | Observable and reversible workspace compatibility | Superseded by 0022 | historical |
| [0022](0022-native-orchestration-thin-engine.md) | Native orchestration with a thin deterministic engine | Accepted in part; snapshot/JSON-output clauses superseded by 0023; retained policy-command/install clauses narrowed by 0024 | `engine/root_routing_test.go`, `engine/internal/gate/gate_test.go`, `engine/internal/lib/cursor_compat_test.go`, `tests/native-orchestration-contract-test.sh` |
| [0023](0023-native-workspace-reads-line-output.md) | Native workspace reads and line-oriented engine output | Accepted in part; AFK/recovery/doctor retention superseded by 0024 | `engine/root_routing_test.go`, `engine/internal/gate/gate_test.go`, `tests/native-orchestration-contract-test.sh`, `tests/npx-pack-smoke.sh` |
| [0024](0024-native-policy-offline-installer-boundary.md) | Native policy and offline installer boundary | Accepted | `tests/native-orchestration-contract-test.sh`, `tests/phase-gate-routing-test.sh`, `tests/install-smoke.sh`, `tests/host-artifacts-test.sh` |
| [0025](0025-evidence-gated-workspace-upgrades.md) | Evidence-gated semantic workspace upgrades | Accepted | `tests/phase-gate-routing-test.sh`, `tests/host-artifacts-test.sh` |
| [0026](0026-content-bound-proof-and-bounded-inputs.md) | Content-bound proof and bounded inputs | Accepted | `engine/internal/lib/candidate_test.go`, `engine/internal/lib/evidencefresh_test.go`, `engine/internal/gate/gate_test.go`, `engine/root_routing_test.go`, `engine/tests/parity_githelpers_test.go`, `tests/native-orchestration-contract-test.sh`, `tests/phase-gate-routing-test.sh`, `tests/install-smoke.sh`, `tests/update-smoke.sh`, `tests/npx-pack-smoke.sh`, `tests/release-tarball-test.sh` |
| [0027](0027-content-bound-build-readiness.md) | Content-bound build readiness | Accepted | `engine/tests/adr_0027_readiness_binding_test.go` |
