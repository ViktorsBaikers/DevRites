# ADR-0025: Evidence-gated semantic workspace upgrades

- **Status:** Accepted
- **Date:** 2026-08-01

## Context

ADR-0012 separated semantic workspace upgrades from structural migration, but
its implementation depended on a readiness-artifact version, an engine
`upgrade-required` result, and a structural migrator. ADR-0022 and ADR-0024
later removed those duplicate semantic and migration surfaces. Official v1/v2
bullet and v3 table cursors are now read directly without rewriting them.

The public `/rite-upgrade` route still has value for an unfinished workspace
that cannot resume after the installed workflow changes. However, age, cursor
encoding, or an absent version marker cannot prove semantic staleness. A generic
"normalize old planning" instruction risks deleting valid history and duplicates
Clarify, Plan, Converge, and Vet.

## Decision

- Keep `/rite-upgrade [slug]` as an explicit compatibility audit and bounded
  orchestrator for active, unfinished workspaces produced by released packs.
- A repair requires a cited current rule, exact contradictory or missing
  workspace evidence, the affected gate, one owning rite, exact writable paths,
  and the smallest behavior-neutral delta. Provenance alone never qualifies.
- The read-only `devrites-upgrade-planner` returns exactly one fail-closed outcome:
  `current`, `repairable`, `unsupported`, or `gap`. `current` requires evidence
  for every applicable axis; missing inputs or unverifiable rules are `gap`.
- `/rite-upgrade` does not reproduce semantic repair logic. It sequences the
  existing owners—Clarify, Plan repair, Converge, and Vet—only when admitted
  findings require them. Their normal gates and write boundaries remain active.
- Before repair, freeze Git status and protected workspace identities. Afterward,
  re-audit once, compare the baseline, and run structural readiness. Never edit
  source/tests, completed work, historical evidence, existing decisions, or cursor format;
  never start Build.
- Installed-pack diagnosis remains `/rite-doctor`; pack replacement remains the
  installer/update path; legacy customization import remains the literal-only
  `/rite-customize --import-legacy` mode. None is inferred by Upgrade.

This supersedes ADR-0012's readiness-artifact version, engine return code,
structural migrator, and direct public-skill rewrite mechanics. It retains the
decision to keep package update, workspace storage compatibility, and semantic
workspace reconciliation separate.

## Alternatives considered

| Option | Why not |
|---|---|
| Retire `/rite-upgrade` | Removes a useful explicit entry point for diagnosing an older active workspace and forces users to guess among recovery rites. |
| Restore semantic versions or an engine staleness parser | Recreates a second semantic authority and can certify content from metadata rather than evidence. |
| Let Upgrade rewrite planning directly | Duplicates phase owners and creates a broad mutation path outside their gates. |
| Replay per-release migrations | Grows a permanent chain and preserves obsolete intermediate mechanics. |

## Consequences

Older official workspaces remain readable without migration. Upgrade can now
return `current` without touching them, repair only demonstrated defects through
their canonical owner, or stop explicitly when compatibility cannot be proved.
The command costs one fresh assessment and at most one re-audit, but it no longer
guesses from age or silently normalizes history.

Regression coverage lives in `tests/phase-gate-routing-test.sh` and generated
host-artifact parity tests.
