# ADR-0035: The Claude root no longer runs in plan mode

- **Status:** Accepted
- **Date:** 2026-10-15

## Context

ADR-0015 and ADR-0017 kept the Claude root source-read-only with project
`permissions.defaultMode: plan`. Claude Code plan mode forbids every edit except
the plan file, so the root could not write the `.devrites/work/<slug>/` artifacts
every lifecycle skill requires. No skill told the root to leave plan mode, and
approving a plan switches the session out of plan mode for good. Once the parent
runs in `acceptEdits`, auto, or bypass mode, Claude also ignores a subagent's own
`permissionMode`, so the read-only reviewer modes stopped being enforced.

## Decision

- DevRites settings no longer set `defaultMode`. The installer stops writing
  `plan` and removes a `plan` it wrote earlier; it never overrides a mode the
  user chose.
- The root stays source- and test-read-only as workflow policy, as on Codex.
  Allow rules cover the engine commands and `/.devrites/**` edits; source edits
  go through `devrites-slice-wright` (and the `fast-builder` / `overhaul-*`
  writers of ADR-0031 and ADR-0033).
- Read-only roles keep `permissionMode: plan` as a best-effort guard. Their
  guarantee is a tool list without Edit/Write plus their instructions.
  `overhaul-implementer` gets `permissionMode: acceptEdits` like the other writers.

## Consequences

- Supersedes the Claude root plan-mode clauses of ADR-0015 and ADR-0017.
- `/rite-doctor` checks writer tool grants and modes, not root plan mode.
- Claude root source safety is now instruction-backed on every host.
