# rite-build: anti-patterns

Load this when standing a non-trivial implementation decision, or when the
agent feels reluctance toward stopping after one slice, applying TDD, or
calling `devrites-doubt`.

Pack-wide rationalizations + red flags (incl. tests-later, defensive checks,
generic AI naming, drive-by refactors): see
[standards/anti-patterns.md](../../devrites-lib/reference/standards/anti-patterns.md).

## Phase-specific rationalizations

| Excuse | Rebuttal |
|---|---|
| "Slices 1 and 2 are related; let me do both." | One slice, then stop. Full stop. The user asks for the next. |
| "It's faster to skip `devrites-doubt` for this small change." | Doubt is cheapest *before* the decision is standing. After is debugging. |

## Forge ([forge.md](forge.md))

| Excuse | Rebuttal |
|---|---|
| "This slice is hard. I'll forge it." | Hard ≠ forkable. Forge competes ≥2 *genuinely-different* approaches with no clear winner. One hard approach is single-path; building it K times wastes K× for one answer. |
| "I'm not sure which approach is right, so let me compete them and decide." | Forge is for a real fork the plan couldn't settle, not for dodging a decision you can make now. Decide from the code/docs first; forge only when the alternatives are genuinely undecidable on paper. |
| "Candidate A's data layer plus B's UI would be the best of both." | Don't hand-merge candidates: two authors in one tree is the incoherence the single-writer rule exists to prevent. Land one winner whole; graft a *specific* runner-up idea only by continuing the winning wright once. |
| "The judge already scored these, so I can skip the post-return doubt." | The judge picks *between* candidates; it doesn't replace doubting the winner's standing decisions. Steps 4-7 run on the winner unchanged. |
| "A fourth candidate might be even better." | K is capped at 3. A fourth rarely moves the winner and multiplies cost; if none of three is right, the plan is wrong: `$rite-plan repair`, don't add candidates. |

## Red Flags

- About to start slice N+1 without the user asking.
- Skipping `devrites-doubt` on a branching, boundary, data-model, auth, public-API, or migration decision.
- Adding a dependency or a second design system without recording rationale in `decisions.md`.
- Writing a `try/catch` block wider than what you handle.
- A comment that restates what the next line does.
- Touching files that aren't in `touched-files.md` for "tidiness".
- Forging a slice with no acceptance / `test-plan.md` coverage for the judge to score against.
- Landing more than one candidate's code, or keeping a losing worktree/branch around after F7.
