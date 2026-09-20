# ADR-0012: Separate semantic workspace upgrades from structural migration

- **Status:** Accepted
- **Date:** 2026-07-24

## Context

Updating DevRites replaces the installed engine, skills, agents, and hooks. It
does not prove that an active workspace was planned under the rules shipped by
that update.

The existing `devrites-engine migrate` command has a narrower job: it repairs
workspace layout, aliases, and state schema. Extending it to rewrite plans would
mix structural storage changes with product decisions and could damage completed
work.

This distinction matters most for long-running workspaces. A plan may still be
internally consistent while carrying an obsolete proof recipe, a local command
wrapper, or readiness evidence produced by an older workflow. Byte hashes alone
cannot tell us that those instructions remain valid.

## Decision

DevRites will keep three upgrade operations separate:

| Operation | Responsibility |
| --- | --- |
| `devrites-engine update` | Update the installed engine and workflow pack. |
| `devrites-engine migrate` | Normalize workspace layout and structural state schema. |
| `/rite-upgrade [slug]` | Reconcile an active workspace with the current semantic planning contract. |

Semantic compatibility is versioned by the readiness-artifact contract, not by
the DevRites release number or workspace state schema. The current contract is
`devrites.readiness-artifacts.v2`.

`decision-coverage.md`, `eng-review.md`, and `test-plan.md` declare the semantic
contract that produced them. Build readiness returns code `8`
(`upgrade-required`) when existing readiness artifacts use an older, missing,
or incompatible contract, and routes the workspace to `/rite-upgrade`.

`/rite-upgrade` is desired-state reconciliation, not a replay of historical
migrations. It:

- works only on the active unfinished planning surface;
- preserves product source, completed slice bodies, proof, evidence, and
  accepted decisions;
- removes obsolete engine-repair recipes and machine-local command wrappers
  from active canonical plans;
- reruns current coverage, planning, and engineering-readiness checks;
- asks the user only when the current contract exposes a real product or risk
  decision; and
- makes no changes when the workspace is already current and ready, completed,
  or archived.

A fresh read-only `devrites-upgrade-planner` classifies the gap before any
mutation. The public skill remains the sole writer and verifies the resulting
workspace before build resumes.

## Alternatives considered

| Option | Why not |
| --- | --- |
| Treat the installed DevRites release as the workspace contract | Release numbers describe the whole product and do not say which planning semantics produced an artifact. |
| Extend `devrites-engine migrate` to rewrite planning artifacts | Structural migration cannot safely make product judgments or distinguish completed evidence from unfinished instructions. |
| Replay one migration for every historical release | It would preserve obsolete intermediate mechanics and make long-lived workspaces depend on an ever-growing migration chain. Desired-state reconciliation is smaller and safer. |
| Force users to restart old workspaces | It discards trustworthy completed work and evidence without improving the unfinished plan. |

## Consequences

- Updating DevRites cannot silently certify an old plan as current.
- Structural migration stays deterministic and safe to run independently.
- Active work can move forward without discarding trustworthy completed
  evidence or recreating an old engine.
- New semantic rules require a readiness-contract revision and a bounded
  desired-state upgrade path.
- Archived workspaces remain historical records rather than being rewritten to
  match current conventions.
