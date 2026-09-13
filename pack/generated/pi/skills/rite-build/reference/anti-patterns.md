# rite-build: anti-patterns

Load when a non-trivial decision stands or stopping, TDD, or doubt discipline
feels inconvenient.

Pack-wide rationalizations and red flags live in
[standards/anti-patterns.md](../../devrites-lib/reference/standards/anti-patterns.md).

## Phase-specific rationalizations

| Excuse | Rebuttal |
|---|---|
| "Slices 1 and 2 are related; let me do both." | HITL needs a later user invocation. Explicit `.devrites/AFK` may let the root chain only after green proof and cap/pause gates; each wright still returns after one. |
| "It's faster to skip `devrites-doubt` for this small change." | Doubt is cheapest *before* the decision is standing. After is debugging. |

## Red Flags

- About to start slice N+1 outside explicit AFK or before its green-proof/cap/pause gates.
- Skipping `devrites-doubt` on a branching, boundary, data-model, auth, public-API, or migration decision.
- Adding a dependency or a second design system without recording rationale in `decisions.md`.
- Writing a `try/catch` block wider than what you handle.
- A comment that restates what the next line does.
- Touching files that aren't in `touched-files.md` for "tidiness".
