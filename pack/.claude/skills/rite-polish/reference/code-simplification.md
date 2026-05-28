# Code simplification (review-scoped)

Reduce complexity in the feature's touched code while **preserving exact behavior**. This
step runs in **every `/rite-polish` Phase 1**, delegating the audit to
`devrites-audit simplify`. Scope = active feature only.

## Measure first, target hotspots
Untargeted cleanup just moves decision points around. Aim at the genuinely complex spots
in the feature's diff — deep nesting, long branchy functions, high cyclomatic complexity,
sprawling conditionals — not code that's already clear.

## Named techniques (behavior-preserving)
- **Guard clauses** → flatten nested if/else; return early on unwanted cases.
- **Extract Method** → a coherent block into a named, single-responsibility helper.
- **Simplify conditionals** → switch / lookup table over a long if-else; decompose a
  complex boolean into well-named parts.
- **Dedupe**; inline single-use indirection; replace hand-rolled utils with the
  stdlib/existing helper.
- **Delete dead code** this feature added.

## Chesterton's Fence
Before removing anything, understand *why* it's there. If you can't explain a check,
branch, or wrapper, you may not remove it — many "useless" lines guard a real edge case.

## Behavior preservation
Observable behavior stays identical; tests stay green. If behavior would change, it's not
simplification — it needs its own acceptance + proof (and maybe drift handling). Prefer
transformations with obvious equivalence.

## Don't over-reduce / proportionality
Some logic is inherently branchy; forcing the complexity number down by **hiding** branches
elsewhere is worse, not better — readability is the goal, not a metric. And don't spend
disproportionate effort on small, stable, rarely-touched code; target central/often-read code.

## Guardrails
- Feature scope only — no project-wide refactor.
- Don't delete suspected dead code **outside** this feature without asking.
- **Re-prove after simplifying** — a simplification that breaks a test wasn't
  behavior-preserving.
- Cleverness that's shorter but harder to read is not simpler.
