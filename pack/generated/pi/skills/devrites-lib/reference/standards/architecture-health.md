# Architecture health

> Applies when: an audit, vet, or seal needs a structural read on the whole
> codebase — not the diff — and a dependency index is available.

A health score compresses six structural signals into one 0–100 number with a
letter grade. It is an *audit instrument*, not a gate: it ranks surfaces for
attention and tracks drift between milestones. It never proves a change safe
and never substitutes for review.

## When to measure

Compute a score only when a code index covers the repo (see
[`code-navigation.md`](code-navigation.md) § Index freshness). With no index,
report `unmeasured` — a placeholder number presented as a measurement is a
false pass. **Failing case:** an audit reports "health: 100/100" on a repo the
instrument could not parse.

## Dimensions

Weight the dimensions exactly once; do not invent extra dimensions ad hoc —
record repo-specific signals as findings instead.

| Dimension | Weight | What it measures | Healthy | Warning |
| --- | --- | --- | --- | --- |
| Coupling | 25% | Cross-file edges per file; penalize god-file outliers (max ≫ 3× mean) and >70% cross-directory share | avg ≤3 connections | avg >10 |
| Cohesion | 20% | Share of a directory's edges that stay inside it | ≥70% internal | <30% |
| Circular deps | 20% | File-level dependency cycles per 100 files | 0 cycles | >5 per 100 |
| God files | 15% | Files with >3× mean connections | none | >6% of files |
| Orphans & dead exports | 10% | Unreferenced files; exported symbols with zero dependents (60/40 blend) | 0% orphans | >10% orphans or >5% dead exports |
| Depth | 10% | Longest dependency chain, cycles collapsed to one hop each (SCC-condensed DAG) | ≤4 levels | >8 |

Grade bands: ≥90 A · ≥80 B · ≥70 C · ≥60 D · below F.

## Rules

- **Measure before and after.** A score's value is the delta: record
  before/after for structural work in `evidence.md`; a drop >3 points needs a
  named cause, not a shrug.
- **Count runtime edges, not type-only edges.** `import type`-style
  relationships inform dead-code analysis but do not execute; including them
  overstates coupling.
- **Exclude fixtures and generated files** from orphan/dead counts — a
  directory of test fixtures is not dead code.
- **Depth is computed on the condensed graph.** Collapse each cycle (strongly
  connected component) to one node first; an exact longest-path search on a
  cyclic graph is unbounded and returns no honest number.
- **Every recommendation names the dimension it moves.** "Improve the
  architecture" is not an action; "extract `x/` → `y/` edge to cut cross-dir
  coupling (Coupling, est. +8)" is.
- **Trend over snapshots.** Keep the score per milestone in `evidence.md` or
  the feature's audit artifact; a single reading without a baseline is a
  weather report, not a climate signal.

## Orphan and dead-export semantics

An orphan file has zero cross-file runtime edges. A dead export is an exported
top-level symbol (function/class/interface/type/enum/constant) with zero
incoming edges — member-level symbols (methods, properties) are excluded:
they are not meaningful architecture-level candidates and produce false
positives. Both counts feed only this dimension; the *removal decision* for a
dead export follows [`deprecation.md`](deprecation.md), including its
confidence rubric — a low-confidence dead lead is a watch item, never an
automatic deletion.
