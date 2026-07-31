# ADR-0020: Thin engine and native host orchestration

- **Status:** Accepted
- **Date:** 2026-07-31

## Context

DevRites already installs exact Claude and Codex specialist definitions. Adding
Go dispatch versions, tiers, receipts, polling, or fallbacks would mirror host
internals, drift when a host changes, and turn workflow depth into a second
protocol. The engine also accumulated repository-only and presentation commands
that did not need a cross-host runtime primitive.

## Decision

- Skills name the exact generated `devrites-*` role and a bounded task. Claude
  or Codex owns discovery, dispatch, scheduling, waiting, follow-up, result
  delivery, and compatibility with its own agent protocol.
- Quick, Standard, and Full select admission/evidence/review depth over one
  lifecycle and artifact schema. They are not engine or agent API versions.
- The Go engine is limited to deterministic cross-host runtime work: persisted
  workspace state, readiness/acceptance/integrity gates, secret scanning,
  bounded recovery, safe migration, and managed installation diagnostics.
- Repository JSON/schema/generated parity/build/release validation stays in
  repository scripts and CI. Installed skills use the engine and native host;
  they never assume DevRites source-tree scripts exist in a consumer project.
- `status`, `budget`, `mutation-gate`, standalone `review-integrity`, and
  `validate-pack` are removed at the next major boundary with ordinary
  unknown-command behavior and no tombstones. Snapshot replaces machine status;
  reviewer accounts remain seal-internal; real mutation runners own mutation
  proof; doctor owns installed-pack health.
- Explicit versions remain only on persisted documents that need stable readers,
  such as workspace snapshots and migration journals.

## Alternatives considered

| Option | Why not |
|---|---|
| Keep a Go broker for Codex/Claude agents | Duplicates native APIs and creates V1/V2 compatibility work with no product value. |
| Keep every legacy command as a wrapper | Preserves two authorities and makes help claim capabilities the engine does not own. |
| Move deterministic state and safety into prompts | Loses machine-enforced, cross-host behavior and testable failure contracts. |

## Consequences

Host releases require native conformance tests rather than engine protocol
changes. The engine surface is smaller and more stable. Repository maintainers
and installed users have deliberately different validation surfaces, which must
not be mixed in shipped skills.

This narrows ADR-0001's broad control-plane scope and consolidates the native
orchestration direction of ADR-0010, ADR-0013, and ADR-0019 without changing
their retained deterministic safety gates. It supersedes ADR-0019's
environment-driven opted-in mutation-gate clause; only evidence from a real
repository mutation runner may support a mutation-testing claim.
