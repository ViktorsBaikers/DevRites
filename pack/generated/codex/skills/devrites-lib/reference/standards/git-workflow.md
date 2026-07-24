# Git workflow

History is documentation. Keep it clean, atomic, and readable.

## Conventional Commits
Format: `type(scope): subject`
- **type**: `feat fix docs style refactor perf test build ci chore revert` (lower-case).
- **scope**: the area changed (lower-case).
- **subject**: imperative mood, no leading capital, no trailing period
  ("add export endpoint", not "Added export endpoint.").
- Keep the header short (~50, hard cap ~72). Put the *why* in the body, wrapped, after a
  blank line. Reference issues in the footer.

## Atomic commits
- One logical change per commit; it should build and pass tests on its own.
- Don't mix refactor + behavior change in one commit: split them so each is reviewable
  and revertible.

## Small, focused branches/PRs
- One concern per branch. Short-lived; rebase/merge per the project's convention.
- Keep PRs small (see `code-review.md`) so they get a real review.

## Change summary: prove the scope
When you hand back a change, state what you deliberately left alone as well as what you touched.
A **"Things I didn't touch (intentionally)"** line (the adjacent smells you noticed and declined
to fix) proves the diff is feature-scoped ([`core.md`](core.md) rule 7) and matches
`touched-files.md`, rather than an unsolicited renovation the reviewer has to untangle from the
real change. Noticed-but-not-fixed becomes an FYI follow-up, never a silent addition to this diff.

## Merge conflict recovery
When git is mid-merge or mid-rebase, recover before other workflow work:

1. Inspect state: `git status --short --branch`, then note whether this is a merge or rebase.
2. For each conflicted file, identify **our intent** and **their intent** before editing.
3. Resolve each hunk by preserving both intents when possible; if intents conflict, pick one and
   record the trade-off in the commit/PR body.
4. `git add` the resolved files, then run the smallest relevant checks.
5. Finish with `git merge --continue` or `git rebase --continue`; abort only when the chosen
   branch direction is wrong, not because a hunk is annoying.

## Versioning & changelog
- Semver describes what consumers may rely on. Treat an observable breaking change as major.
- Derive the version from the tag; do not maintain competing manual copies.
- Curate changelog entries by user impact in the change that earns them; a changelog is not a git log.

## Never commit
- Secrets, credentials, tokens, or `.env` files. If one lands in history, rotate it and
  scrub it: deleting the file in a later commit is not enough.
- Generated artifacts, dependencies, or large binaries that belong in ignore rules.
