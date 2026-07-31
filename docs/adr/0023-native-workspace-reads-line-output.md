# ADR-0023: Native workspace reads and line-oriented engine output

- **Status:** Accepted
- **Date:** 2026-07-31

## Context

The installed skills and exact host agents already read the canonical workspace
Markdown. The optional `snapshot` command rebuilt the same state into a
`devrites.workspace.v1` document, added summary heuristics, and became a
mandatory-looking preamble across workflows despite having no external consumer
beyond repository smoke tests.

The `devrites-command/v1` wrapper similarly converted ordinary check and doctor
output into an agent-facing protocol. Its only repository consumer was the
outcome harness. Stable gate reason IDs remain useful, but they do not require a
versioned envelope. `doctor --verbose` produced exactly the normal doctor
report, and recovery's ignored `--class` option preserved an obsolete command
shape.

Other remaining engine responsibilities have different properties. AFK slice
and recovery-attempt limits are hard autonomy bounds, atomic state mutations
protect multi-file consistency, and no established secret scanner is installed
to replace the bounded scanner. The doctor is already a small deterministic
root/install diagnostic. Moving those responsibilities into model judgment
would weaken safety rather than simplify implementation.

## Decision

- Remove `snapshot`, its Go summary readers, `devrites.workspace.v1`, and its
  JSON schema. Skills resolve an explicit slug or `.devrites/ACTIVE`, require
  `.devrites/work/<slug>/state.md`, and read only the current phase artifacts.
- Remove engine `--json` handling and the `devrites-command/v1` envelope.
  Lifecycle checks emit line-oriented output with the existing stable
  `reason: DRV-...` identifier; repository evals assert exit status and that
  reason directly.
- `doctor` accepts no inert `--verbose` alias. Recovery rejects the obsolete
  `--class` option while continuing to tolerate unknown fields in already
  persisted JSONL entries.
- Retain `state tick-afk`, `state recovery`, the root/lock/atomic-write
  boundary, structural checks, evidence freshness, secret scanning,
  install/update/uninstall, doctor, and version.
- Retain profile-selected and conditionally applicable native reviewers. No Go
  broker, dispatch receipt, or agent protocol is introduced.
- Removed commands and flags fail visibly; there are no compatibility aliases
  or tombstones.

This ADR supersedes ADR-0022's snapshot and JSON-output clauses. Its native
orchestration, deterministic state/safety, released cursor compatibility, and
fresh irreversible-action approval rules remain in force.

## Alternatives considered

| Option | Why not |
|---|---|
| Keep snapshot as an optional convenience | Workflow references made it functionally mandatory, while direct ledger reads are simpler and authoritative. |
| Keep JSON for possible automation | No external consumer was established; a speculative protocol is not worth permanent compatibility cost. |
| Move AFK/recovery limits into prose | That turns hard autonomy stops into best-effort model compliance. |
| Remove the scanner and ask the model to inspect secrets | Secret handling is a trust-boundary control and no established deterministic replacement exists in the repository. |
| Replace the retained Go helpers with skill scripts | It duplicates root containment, locking, and cross-platform install behavior without reducing concepts. |

## Consequences

The public CLI and installed allowlist are smaller. Skills and status reporting
read the single authoritative ledger rather than a derived summary. Repository
outcome tests no longer need an engine JSON decoder.

Removing the two output schemas is intentionally breaking, but requires no data
migration because neither schema is persisted workspace state. Released
`state.md` cursor encodings remain readable. Deterministic autonomy and secret
safety remain testable outside model judgment.
