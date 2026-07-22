# Feature-scoped review

The review boundary is the **active feature**: the files in `touched-files.md` and the
current diff. This is a hard rule, not a guideline.

## In scope
- Code added/changed by this feature.
- Tests for this feature.
- Files this feature intentionally inspected and depends on (read for context, but
  don't refactor them).

## Out of scope (do NOT)
- Refactor unrelated modules because they're "nearby" or "while we're here".
- Delete suspected dead code outside this feature without asking the user.
- Restyle/upgrade dependencies or change project-wide config to suit this feature.
- Expand the review into a project audit.

## When you spot a real problem outside scope
Record it as an **[FYI] follow-up** in `review.md` (and suggest a separate feature/issue).
Don't fix it inline. Drive-by changes balloon the diff, dodge their own review, and
mix concerns the seal can't cleanly evaluate.

## Why scope discipline matters
A tight diff gets a real review; a sprawling one gets a rubber stamp. Scope creep is how
"a small fix" becomes an unreviewable, unprovable change. Keep the feature shippable and
the review honest.
