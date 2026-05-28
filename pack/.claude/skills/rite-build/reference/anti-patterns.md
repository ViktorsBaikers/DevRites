# rite-build — anti-patterns

Load this when standing a non-trivial implementation decision, or when the
agent feels reluctance toward stopping after one slice, applying TDD, or
calling `devrites-doubt`.

Pack-wide rationalizations + red flags (incl. tests-later, defensive checks,
generic AI naming, drive-by refactors): see
[rules/anti-patterns.md](../../../rules/anti-patterns.md).

## Phase-specific rationalizations

| Excuse | Rebuttal |
|---|---|
| "Slices 1 and 2 are related; let me do both." | One slice, then stop. Full stop. The user asks for the next. |
| "It's faster to skip `devrites-doubt` for this small change." | Doubt is cheapest *before* the decision is standing. After is debugging. |

## Red Flags

- About to start slice N+1 without the user asking.
- Skipping `devrites-doubt` on a branching, boundary, data-model, auth, public-API, or migration decision.
- Adding a dependency or a second design system without recording rationale in `decisions.md`.
- Writing a `try/catch` block wider than what you actually handle.
- A comment that restates what the next line does.
- Touching files that aren't in `touched-files.md` for "tidiness".
